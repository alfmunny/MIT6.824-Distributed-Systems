# Lab 1: MapReduce

## Part 1: Communication and Sending Task

The first step is to setup the communication between the master and workers.

### MapReduce Model

### RPC and Task

rpc.go
```go
const (
	TaskMap    = 0
	TaskReduce = 1
	TaskWait   = 2
	TaskEnd    = 3
)

type TaskInfo struct {
	State     int
	FileName  string
	FileIndex int
	NReduce   int
	Nfiles    int
}
```

master.go
```go
type Master struct {
	mapTask    TaskQueue
	reduceTask TaskQueue
	isDone     bool
}

type TaskQueue struct {
	taskArray []TaskInfo
	mutex     sync.Mutex
}
```

### Queue, Pop and Push, and Mutex

```go
import "sync"

type TaskQueue struct {
	taskArray []TaskInfo
	mutex     sync.Mutex
}

func (tq *TaskQueue) lock() {
	tq.mutex.Lock()
}

func (tq *TaskQueue) unlock() {
	tq.mutex.Unlock()
}

func (tq *TaskQueue) Pop() TaskInfo {
	tq.lock()

	if tq.taskArray == nil {
		tq.unlock()
		taskInfo := TaskInfo{}
		taskInfo.State = TaskEnd
		return taskInfo
	}

	length := len(tq.taskArray)
	ret := tq.taskArray[length-1]
	tq.taskArray = tq.taskArray[:length-1]
	tq.unlock()
	return ret
}

func (tq *TaskQueue) Push(taskInfo TaskInfo) {
	tq.lock()
	tq.taskArray = append(tq.taskArray, taskInfo)
	tq.unlock()
}
```


### Sending Task

worker.go
```go
func CallAskTask() *TaskInfo {
	args := ExampleArgs{}
	reply := TaskInfo{}
	call("Master.AskTask", &args, &reply)
	return &reply
}
```

master.go
```go
func (m *Master) AskTask(args *ExampleArgs, reply *TaskInfo) error {
	if len(m.reduceTask.taskArray) > 0 {
		taskInfo := m.reduceTask.Pop()
		*reply = taskInfo
		fmt.Printf("%v sent to reducer\n", taskInfo.FileName)
	} else if len(m.mapTask.taskArray) > 0 {
		taskInfo := m.mapTask.Pop()
		*reply = taskInfo
		taskInfo.State = TaskReduce
		m.reduceTask.Push(taskInfo)
		fmt.Printf("%v sent to mapper\n", taskInfo.FileName)
	} else {
		reply.State = TaskEnd
		m.isDone = true
	}

	return nil
}
```

### Receiving Task

worker.go
```go
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	for {
		taskInfo := CallAskTask()
		switch taskInfo.State {
		case TaskMap:
			workerMap(mapf, taskInfo)
		case TaskReduce:
			workerReduce(reducef, taskInfo)
		case TaskEnd:
			fmt.Println("All tasks completed on master.")
			return
		default:
			panic("Invalid task state.")
		}
	}
}

func workerMap(mapf func(string, string) []KeyValue, taskInfo *TaskInfo) {
	filename := taskInfo.FileName
	fmt.Printf("Mapping on %v\n", filename)
}

func workerReduce(reducef func(string, []string) string, taskInfo *TaskInfo) {
	filename := taskInfo.FileName
	fmt.Printf("Reducing on %v\n", filename)
}
```


See code change in one [commit](https://github.com/alfmunny/MIT6.824-Distributed-Systems/commit/fd0677e3480d9a395286f6f613d811320422ec80)

## Part 2: Map and Reduce 

## Part 3: Handle Crash
