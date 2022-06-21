package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//steal from mrsequential.go

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// main/mrworker.go calls this function.
//
//不断循环向coordinator请求工作
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	//进入循环，向 Coordinator 申请 Task
	for {
		args := RPCArgs{}
		reply := RPCReply{}
		call("Coordinator.ApplyForTask", &args, &reply)
		switch reply.TaskInfo.TaskType {
		//map任务
		case 0:
			mapper(&reply.TaskInfo, mapf)
		case 1:
			reducer(&reply.TaskInfo, reducef)
		case 2:
			fmt.Println("Waiting")
			time.Sleep(time.Second)
			continue
		case 3:
			fmt.Println("All tasks done")
			return
		}
		//通过RPC，再将完成的task信息发给coordinator
		args.TaskInfo = reply.TaskInfo
		call("Coordinator.TaskCompleted", &args, &reply)
	}
	// Your worker implementation here.
	// uncomment to send the Example RPC to the coordinator.
	// CallExample()
}

//steal from mrsequential.go
func mapper(task *Task, mapf func(string, string) []KeyValue) {
	fmt.Printf("Map worker get task %d-%s\n", task.TaskId, task.Filename)

	intermediate := make([][]KeyValue, task.NReduce)
	for i := 0; i < task.NReduce; i++ {
		intermediate[i] = make([]KeyValue, 0)
	}
	file, err := os.Open(task.Filename)
	if err != nil {
		log.Fatalf("cannot open %v", task.Filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", task.Filename)
	}
	file.Close()
	kva := mapf(task.Filename, string(content))
	for _, kv := range kva {
		intermediate[ihash(kv.Key)%task.NReduce] = append(intermediate[ihash(kv.Key)%task.NReduce], kv)
	}

	for i := 0; i < task.NReduce; i++ {
		if len(intermediate[i]) == 0 {
			continue
		}
		oname := fmt.Sprintf("mr-%d-%d", task.TaskId, i)
		ofile, _ := ioutil.TempFile("./", "tmp_")
		enc := json.NewEncoder(ofile)
		for _, kv := range intermediate[i] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Fatalf("Json encode error: Key-%s, Value-%s", kv.Key, kv.Value)
			}
		}
		ofile.Close()
		os.Rename(ofile.Name(), oname)
	}
}

func reducer(task *Task, reducef func(string, []string) string) {
	fmt.Printf("Reduce worker get task %d\n", task.TaskId)

	intermediate := make([]KeyValue, 0)

	for i := 0; i < task.NMap; i++ {
		iname := fmt.Sprintf("mr-%d-%d", i, task.TaskId)
		ifile, _ := os.Open(iname)
		dec := json.NewDecoder(ifile)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		ifile.Close()
	}
	sort.Sort(ByKey(intermediate))

	oname := fmt.Sprintf("mr-out-%d", task.TaskId)
	ofile, _ := ioutil.TempFile("./", "tmp_")

	for i := 0; i < len(intermediate); {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}
	ofile.Close()
	os.Rename(ofile.Name(), oname)
}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
