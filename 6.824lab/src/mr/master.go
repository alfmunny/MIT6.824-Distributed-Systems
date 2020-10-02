package mr

import (
	"fmt"
	"sync"
)
import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"

type Master struct {
	// Your definitions here.
	mapTask    TaskQueue
	reduceTask TaskQueue
	isDone     bool
}

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

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

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

//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := false

	// Your code here.
	if m.isDone {
		ret = true
		fmt.Println("All work is done. Shut down master, Bye.")
	}

	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}

	// Your code here.
	for fileindex, filename := range files {
		taskInfo := TaskInfo{TaskMap, filename, fileindex, nReduce, len(files)}
		m.mapTask.Push(taskInfo)
	}

	m.server()
	return &m
}
