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
	mapTaskQueuing    TaskQueue
	reduceTaskQueuing TaskQueue
	mapTaskRunning    TaskQueue
	reduceTaskRunning TaskQueue
	isDone            bool
	filenames         []string
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

func (tq *TaskQueue) Size() int {
	return len(tq.taskArray)

}

func (tq *TaskQueue) RemoveTask(fileIndex int, partIndex int) {
	tq.lock()
	for i := 0; i < tq.Size(); {
		taskInfo := tq.taskArray[i]
		if (taskInfo.FileIndex == fileIndex) && (taskInfo.PartIndex == partIndex) {
			tq.taskArray = append(tq.taskArray[:i], tq.taskArray[i+1:]...)
		} else {
			i++
		}
	}
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
	if m.reduceTaskQueuing.Size() > 0 {
		taskInfo := m.reduceTaskQueuing.Pop()
		m.reduceTaskRunning.Push(taskInfo)
		*reply = taskInfo
		fmt.Printf("%v sent to reducer\n", taskInfo.FileName)
		return nil
	}

	if m.mapTaskQueuing.Size() > 0 {
		taskInfo := m.mapTaskQueuing.Pop()
		m.mapTaskRunning.Push(taskInfo)
		*reply = taskInfo
		fmt.Printf("%v sent to mapper\n", taskInfo.FileName)
		return nil
	}

	if m.mapTaskRunning.Size() == 0 && m.reduceTaskRunning.Size() == 0 {
		reply.State = TaskEnd
		m.isDone = true
		return nil
	} else {
		reply.State = TaskWait
		return nil
	}
}

func (m *Master) TaskDone(args *TaskInfo, reply *ExampleReply) error {
	switch args.State {
	case TaskMap:
		m.mapTaskRunning.RemoveTask(args.FileIndex, args.PartIndex)
		fmt.Printf("Map task on %vth file %v complete\n", args.FileIndex, args.FileName)
		if m.mapTaskRunning.Size() == 0 && m.mapTaskQueuing.Size() == 0 {
			m.createReduceTask(args)
		}
		break
	case TaskReduce:
		m.reduceTaskRunning.RemoveTask(args.FileIndex, args.PartIndex)
		fmt.Printf("Reduce task on %vth part complete\n", args.PartIndex)
		break
	default:
		panic("Wrong Task Done")
	}
	return nil
}

func (m *Master) createReduceTask(taskInfo *TaskInfo) error {
	for i := 0; i < taskInfo.NReduce; i++ {
		newTaskInfo := TaskInfo{}
		newTaskInfo.State = TaskReduce
		newTaskInfo.PartIndex = i
		newTaskInfo.NFiles = len(m.filenames)
		m.reduceTaskQueuing.Push(newTaskInfo)
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
	m.filenames = files

	// Your code here.
	for fileindex, filename := range files {
		taskInfo := TaskInfo{TaskMap, filename, fileindex, 0, nReduce, len(files)}
		m.mapTaskQueuing.Push(taskInfo)
	}

	m.server()
	return &m
}
