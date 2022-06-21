# ComputerNetworking-Final
6.824 lab1

*本次期末设计是用的6.824这门公开课的lab1。*

*截至目前其实没有完成，但感觉收获颇多，之后也会再找时间完成。*

*故在此记录心路历程。*

*（修改补充的文件仅是mr文件夹下的三个文件：mr/coordinator.go , mr/rpc.go , mr/worker.go）*

### 挑战（兼背景介绍） ###

- **lab链接**：http://nil.csail.mit.edu/6.824/2021/labs/lab-mr.html
- **MapReduce paper**：https://pdos.csail.mit.edu/6.824/papers/mapreduce.pdf
- **实验目的**：要求实现一个简单的MapReduce系统。但其实我之前也并不了解mapreduce为何物（希望助教老师包容我的无知🥺），所以最开始理解起来也花了一番功夫。
- **与本专业（生物科学）的联系**：嗯...只能说任何学科的发展都离不开这种基础又本质的智慧吧！并且在人类基因组计划的推波助澜吓，现在生命科学的研究已经得到了非常非常多的不连续数据，而可以解决这一切，使生命科学进一步发展的便是现代计算机科学了。而mapreduce作为分布式计算发展中的重要存在，虽然似乎它现今的使用不如以前广泛，但它的核心思想却如此朴素和巧妙，对大规模数据集的并行运算具有很大的影响。
- **使用语言**：go。其实这是我第一次接触go...所以果然也是很大的挑战吧！
- **运行环境**：我的运行环境是MacOS Monterey 12.4 ; go 18.1.2


### 实现过程 ###

- 首先是阅读了MapReduce的paper，也看了一些讲解，b站的一位up主（似乎是一位Google工程师）讲得很不错！
- 然后又去阅读了lab的要求，试运行了一下，再结合之前看的MapReduce相关的东西，大概明白了是要写个啥。
- 接下来是我感觉很重要，也是花了最多时间的一步，就是想清楚到底需要实现什么以及如何去实现，为了赶进度，还是看了蛮多别人写好的笔记的。
- 嗯..然后是去大概了解了一下go..不过因为抱着一种在实践中成长的心理，也没花太多时间在理论学习上（
- 之后就是开始写啦，中途也参考了一些别人的代码。
- 最后是测试调试，其实是很失败的，不过在凌晨四点也不想继续改了，而且感觉对学习成果还是有点满意的。虽然最后分数不会好看，但是..辅修嘛！就不要看那么重要了！！

### 功能分析 ###

*其实这样的笔记真的好多了，但还是想自己再写一下（独自享受寂静的夜）*

- 首先，先说明需要修改的文件：mr/coordinator.go , mr/rpc.go , mr/worker.go
- 角色为1个coordinator和多个worker：coordinator自然是负责调度分配，worker就是执行具体的任务啦
- 任务分为Map和Reduce两种，Map是映射，Reduce是规约
- coordinator需要做的：
  -  将一开始输入的文件分割成很多小文件（按照paper的说法，是分成很多16-64MB的小文件）
  -  上一步得到的小文件就是做map任务的原材料，也就是说coordinator需要将这些文件分发给worker来进行map任务（其实文件并不是由coordinator发的，coordinator只负责告诉worker要做的是什么任务，对应的什么文件，worker会自己去找文件来处理，只是感觉这样讲起来直观一点（？
  -  coordinator需要清楚地去管理这些事务，比如哪些文件已经分发出去了，哪些文件还在这没被处理
  -  等所有文件都经过map任务处理好之后，coordinator正式宣布：我们进入下一阶段————也就是reduce任务阶段啦！
  -  经过map处理后得到的中间文件即为reduce任务的原材料，也就是说coordinator需要将这些文件分发给worker来进行reduce任务（同上，文件并不是由coordinator分发，coordinator只负责告诉worker要做的是什么任务，其对应的什么文件，接下来都是worker的事
  -  等所有文件都经过reduce任务处理好之后，coordinator正式宣布：我们完成工作啦！
  -  需要注意的是，coordinator还会监测worker有没有偷懒（宕机什么的..），lab里给出的是10s的timeout，也就是如果10s后还没有收到worker的完成汇报，它也不管你是咋滴，就会把对应的任务重新放回待办中，分配给下一个来要任务的worker（没错..从来不是coordinator主动分配的任务，都是worker们主动来要的..）
- worker需要做的：
  -  首先，就是一直一直向coordinator要任务..怎么会这么积极！
  -  根据coordinator的指示去执行map或者reduce任务
  -  完成之后向coordinator汇报，并且继续要任务..
  -  如果已经没有待办任务了，但是还有别的worker在执行任务，coordinator并不会让你退下，而是让你休息等待一小会，再接着请求任务..
  -  直到没有待办任务和正在办的任务了，也就是说所有任务都完成了！coordinator会在你要任务的时候告诉你，这时就可以正式下班了
 - RPC：承担coordinator和worker之间的通信功能

### 代码（没有）实现 ###

*虽然最后出了问题，但还是记录一下大致思路吧*
- coordinator
  -  定义coordinator结构体
  -  定义task结构体
  -  初始化coordinator（感觉coordinator就是一组多个机器上启动用户程序众多副本中的lucky dog..摇身一变就成了调度者，别的program只能黯然离场（x（不过子非鱼，安知鱼之乐..
  -  在这里定义了两个map集合（其实是4个）分别来储存管理map任务和reduce任务，也就是mapready（待办的map任务）、mapinprogress（正在被办的map任务）、reduceready（待办的reduce任务）、reduceinprogress（正在被办的reduce任务）。为什么要把map和reduce任务分开管理呢..其实不分开处理也是可以的，因为map任务都被执行完了才会进行reduce任务，而且本身任务就具有phase/stage属性。可是这样的话，对不太聪明的我来说，就有一些困难，总是搅在一团。最后再三思考还是决定分开储存，这样会比较好理解和具像化一点（人傻就是这样😭）
  -  map任务创建，这一步应该在初始化里完成，但是在初始化时引用了单独写的函数，所以就分开讲了
  -  reduce任务创建，这一步同样也是在更新任务时才被引用
  -  响应worker的请求，也就是看待办池里还有没有任务，要是有就满足worker的要求，要是没有就让worker过一会再来问
  -  更新任务状态，包括接收worker的任务完成汇报，以及根据情况来判断是否应该进入ruduce阶段
  -  监督worker，也就是超时响应，超过10s就把任务放回待办池，交给之后前来的worker。但需要注意的是，此时并非完全放弃最初的worker，谁先交任务都可以，只是后交的会直接被抛弃
  -  判断完成状态，这里用四个map集合都空来判定
  -  其它，比如listens for RPCs from worker.go之类的，就是自带的代码了
- worker
  -  请求任务与交付任务（死循环），因为本身就是交了马上要任务嘛，就放一起好了
  -  mapper，也就是具体如何做map任务
  -  reducer，也即是具体如何做reduce任务
  -  其它，比如listens for RPCs from coordinator.go之类的，自带代码
- RPC
  -  task结构体里已定义所需信息，所以只需要将task结构体作为通信的参数和返回值即可

### 运行截图（部分） ###

<img width="682" alt="image" src="https://user-images.githubusercontent.com/104363753/174912268-2d972f8e-9c11-47a6-a4fd-1ac7081c7671.png">


### 困难重重 ###

- 首先就是go语言吧..好多都是现用现查，比如map集合用法之类的，最开始还想用channel，但最后不想搞就将就用了map（
- 很惭愧的一点，因为我大部分时间都是在考虑实现的逻辑结构，所以其实是忽略了map和reduce任务的具体实现（而且我认为这和计网没那么大关系嘛咳咳（x，加之我看到lab tips里说“You can steal some code from mrsequential.go for reading Map input files, for sorting intermedate key/value pairs between the Map and Reduce, and for storing Reduce output in files.”，就以为不用太在意，结果最后写到的时候发现真的挺懵的orz，所以mapper函数和reducer函数确实基本上是steal的..（不过似乎大部分人也是从mrsequential.go里改的
- 最开始因为task的状态想了蛮久的，正如我前面所说，最后的解决办法是把map任务和reduce任务分开管理了。有可能是因为我一开始一直先入为主地认为，map和reduce任务可以同时进行，所以还纠结了特别久coordinator是随机分配map或者reduce任务还是有什么机制，然后猛然发现map和reduce根本就是两个阶段分开的..但这无疑对我之后的理解也造成了蛮大影响的（这脑子，就是转不快啊！
- 赶ddl要命啊..

### （目前）失败的合理解释 ###

- 菜
- 菜
- 菜
- 说实在，要是我一下子就成功了才是奇怪的事情吧（
- ddl在这，确实没时间，当然这绝对是我自己的问题..
- 可能是steal的代码的问题..因为不了解，所以我都看不太懂它是在做什么（
- go的版本问题？lab要求写的是1.15 or later，我使用的是1.18
- 我有些怀疑是不是环境参数之类的问题，因为我在跑测试程序的时候，中间报了好一些诸如“connect: connection refused”的错
- 还是想说，要是一下子就成功了才奇怪吧（

### 收获 ###

- 当然是我前面已经提过的，这基础而又本质，朴素而又精巧的智慧
- ## “Don't start a lab the night before it is due; it's much more time efficient to do the labs in several sessions spread over multiple days. Tracking down bugs in distributed system code is difficult, because of concurrency, crashes, and an unreliable network. Problems often require thought and careful debugging to understand and fix.” ##






