package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

//Task类型以及作业阶段
type State int

const (
	Map State = iota
	Reduce
	Wait
	Done
)

//创建 Coordinator 结构体
type Coordinator struct {
	mux                   sync.Mutex // 锁，避免并发冲突
	stage                 State      // 当前作业阶段：MAP/REDUCE/DONE
	nMap                  int        // MAP任务数量
	nReduce               int        // Reduce任务数量
	InputFiles            []string
	Intermediates         [][]string   // Map任务产生的R个中间文件的信息
	mapTasksReady         map[int]Task // map任务池 等待被分配的
	mapTasksInProgress    map[int]Task // map任务池 已经被分配的
	reduceTasksReady      map[int]Task // reduce任务池 等待被分配的
	reduceTasksInProgress map[int]Task // reduce任务池 已经被分配的

	// Your definitions here.
}

//创建 Task 结构体
type Task struct {
	TaskId        int   // Task的id
	TaskType      State // Task的类型
	Filename      string
	NMap          int
	NReduce       int // 传入的reducer的数量，用于后续hash
	Intermediates []string
	TimeStamp     time.Time // 用于之后的超时判定
	//FileSlice []string // 输入文件的切片
}

// Worker 向 Coordinator 请求任务
func (c *Coordinator) ApplyForTask(args *RPCArgs, reply *RPCReply) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	//判断是map阶段还是reduce阶段，进而针对不同池子进行判断
	switch c.stage {
	case 0:
		//map等待池里如果有任务就发出去
		if len(c.mapTasksReady) > 0 {
			for k, v := range c.mapTasksReady {
				v.TimeStamp = time.Now()    //记录开始时间
				c.mapTasksInProgress[k] = v //将其放入map运行池中
				reply.TaskInfo = v          //RPC
				delete(c.mapTasksReady, k)  //在map准备池中删除
			}
			//如果map等待池里空了，但是map运行池里还有任务，则让worker等待
		} else if len(c.mapTasksInProgress) > 0 {
			reply.TaskInfo = Task{TaskType: 2}
		}
		return nil

	case 1:
		//reduce等待池里如果有任务就发出去
		if len(c.reduceTasksReady) > 0 {
			for k, v := range c.reduceTasksReady {
				v.TimeStamp = time.Now()       //记录开始时间
				c.reduceTasksInProgress[k] = v //将其放入reduce运行池中
				reply.TaskInfo = v             //RPC
				delete(c.mapTasksReady, k)     //在reduce准备池中删除
			}
			//如果reduce等待池里空了，但是reduce运行池里还有任务，则让worker等待
		} else if len(c.reduceTasksInProgress) > 0 {
			reply.TaskInfo = Task{TaskType: 2}
			//如果reduce等待池和reduce运行池都空了，说明作业结束，worker可以退出了
		} else {
			reply.TaskInfo = Task{TaskType: 3}
		}
		return nil
	}
	return nil
}

//更新task，包括来自worker的汇报和判断是否应进入reduce阶段
func (c *Coordinator) TaskCompleted(args *RPCArgs, reply *RPCReply) error {
	c.mux.Lock()
	defer c.mux.Unlock()
	//判断完成的任务是什么类型
	switch args.TaskInfo.TaskType {
	case 0:
		delete(c.mapTasksInProgress, args.TaskInfo.TaskId)
		//如果此时map等待池和map运行池已经清空，则可以进入下一阶段
		if len(c.mapTasksReady) == 0 && len(c.mapTasksInProgress) == 0 {
			c.stage = 1
		}
	case 1:
		delete(c.reduceTasksInProgress, args.TaskInfo.TaskId)
	}
	return nil
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	ret := false

	if len(c.mapTasksReady) == 0 && len(c.mapTasksInProgress) == 0 &&
		len(c.reduceTasksReady) == 0 && len(c.reduceTasksInProgress) == 0 {
		ret = true
	}

	// Your code here.

	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//

//初始化 Coordinator ，并且创建map任务（同时注意处理超时）
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		stage:                 0,          //初始作业阶段：MAP
		nMap:                  len(files), //MAP任务数量，被分割成的文件数量
		nReduce:               nReduce,
		Intermediates:         make([][]string, nReduce),
		mapTasksReady:         make(map[int]Task),
		mapTasksInProgress:    make(map[int]Task),
		reduceTasksReady:      make(map[int]Task),
		reduceTasksInProgress: make(map[int]Task),
	}

	//创建map任务（将原始文件切成16MB-64MB的文件）
	c.createMapTask()

	// Your code here.

	c.server()

	go c.catchTimeOut()

	return &c
}

//根据传入的filename创建map，每个文件对应一个map task
func (c *Coordinator) createMapTask() {
	for i, file := range c.InputFiles {
		c.mapTasksReady[i] = Task{
			Filename:  file,
			TaskType:  0,
			TaskId:    i,
			NMap:      c.nMap,
			NReduce:   c.nReduce,
			TimeStamp: time.Now(),
		}
	}
}

//根据map得到的Intermediates创建reduce，每个文件对应一个reduce task，map阶段结束才可开始运行
func (c *Coordinator) createReduceTask() {
	for i, file := range c.Intermediates {
		c.reduceTasksReady[i] = Task{
			Intermediates: file,
			TaskType:      1,
			TaskId:        i,
			NMap:          c.nMap,
			NReduce:       c.nReduce,
			TimeStamp:     time.Now(),
		}
	}
}

//处理超时，如果超过10s无反应就交给另一个worker
func (c *Coordinator) catchTimeOut() {
	for {
		time.Sleep(5 * time.Second)
		c.mux.Lock()
		for k, v := range c.mapTasksInProgress {
			if time.Now().Sub(v.TimeStamp) > 10*time.Second {
				c.mapTasksReady[k] = v
				delete(c.mapTasksInProgress, k)
			}
		}
		for k, v := range c.reduceTasksInProgress {
			if time.Now().Sub(v.TimeStamp) > 10*time.Second {
				c.reduceTasksReady[k] = v
				delete(c.reduceTasksInProgress, k)
			}
		}
	}
	c.mux.Unlock()
}
