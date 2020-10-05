# Lab 1: MapReduce

The lab assignment you can find full description and useful hints on the [lab 1 website](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html).

## Part 1: Communication and Sending Task

The first step is to setup the communication between the master and workers.

You can see all code change for Part 1 in one [commit](https://github.com/alfmunny/MIT6.824-Distributed-Systems/commit/fd0677e3480d9a395286f6f613d811320422ec80)


### RPC and Task

Define the entity `TaskInfo`, with which we can send through RPC between master and workers.

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

Let's start with something simple. Only two queues, one for Map tasks, and one for Reduce tasks.
The Queue should have a lock, in case we run into concurrency issues if we run multiple workers.

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

Implement some basic operations for the queue, like Pop() and Push(), lock() and unlock()
Note that, we send a 

```go
import "sync"

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

Now we have Task entity contains our information to send and also two queues to remember them.

We can implement something to consume the queue.

Every time the worker asks for a task, we send a Map task, and move it to the Reduce queue(it is maybe not the case we eventually should have, but for now, let's only see if we can communicate).
If we have Reduce task in queue, we send it to worker immediately.

worker.go
```go
func CallAskTask() *TaskInfo {
	args := ExampleArgs{}
	reply := TaskInfo{}
	call("Master.AskTask", &args, &reply)
	return &reply
}
```

Note that we send a `TaskEnd` if we run out of tasks on both queues.
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

The worker asks for a task, and do the Map or Reduce, depends on the task type. When it receives a `TaskEnd`, it just stops.

We don't implement the actual Map and Reduce now. Let them be empty.

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

### Run it

```bash
cd src/main
mkdir mr-tmp
go build -buildmode=plugin ../mrapps/wc.go
go run mrmaster pg-*.txt
```
And in another shell
```bash
go run mrworker wc.go
```

If you can see the worker communicates with master about the MapReduce tasks, great.

Then we can build the hard part: Map and Reduce.

## Part 2: Map and Reduce 

Now we are going to implement the Map and Reduce.

See all changes in one [commit](https://github.com/alfmunny/MIT6.824-Distributed-Systems/commit/bf91c14fe423d1296c6d3b6141afb6063ee26742)

### Map
First we use Map to generate the Key-Value based intermediate files. It can separate a file in several pieces and let the reducer process them in parallel.

Map program will take the txt files as input to do followings:

1. Extract the Key-Value pair from each file.
```json
{
	"Key": "room",
	"Value": "1"
}
```
	
2. Separate the Key-Value pairs into different files based on hashing and nReduce

We tell the worker how many reduce tasks(nReduce) we want to have for each file. And name the files according to the this index.

For example: 

If nReduce := 10, and for the first file(fileindex 0), we will have 10 intermediate files for Reduce. The file names will be like: 

`mr-0-0, mr-0-1, mr-0-2, ... , mr-0-9`

We have 8 text files as input, so there will be 8x10 files after Map procedure.

```
mr-0-0, mr-0-1, mr-0-2, ... , mr-0-9
mr-1-0, mr-1-1, mr-1-2, ... , mr-1-9
mr-2-0, mr-2-1, mr-2-2, ... , mr-2-9
...
mr-9-0, mr-9-1, mr-9-2, ... , mr-9-9

```

In this lab, we will save them to folder `mr-tmp`.

3. Encode the Key-Value pair with Json Encoder, and save them into the files


```go
func workerMap(mapf func(string, string) []KeyValue, taskInfo *TaskInfo) {
	filename := taskInfo.FileName
	fmt.Printf("Mapping on %v\n", filename)

	// Read file
	intermediate := []KeyValue{}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()

	// Map generate the Key-Value pairs 
	kva := mapf(filename, string(content))
	intermediate = append(intermediate, kva...)

	nReduce := taskInfo.NReduce
	outFiles := make([]*os.File, nReduce)
	fileEncs := make([]*json.Encoder, nReduce)

	//Write to intermediate files

	// generate the file name prefix with the file index
	outprefix := "mr-tmp/mr-" + strconv.Itoa(taskInfo.FileIndex) + "-" //mr-tmp/mr-0-

	for outindex := 0; outindex < nReduce; outindex++ {
		outname := outprefix + strconv.Itoa(outindex) // generate the whole file name mr-tmp/mr-0-0
		outFiles[outindex], _ = os.Create(outname) // create the file
		fileEncs[outindex] = json.NewEncoder(outFiles[outindex]) // create the encoder
	}

	// Write each Key-Value pair into these different files generated above
	for _, kv := range intermediate {
		outindex := ihash(kv.Key) % nReduce
		file = outFiles[outindex]
		enc := fileEncs[outindex]
		err := enc.Encode(&kv)
		if err != nil {
			fmt.Printf("File %v Key %v Value %v Error: %v\n", filename, kv.Key, kv.Value, err)
			panic("Json encode failed")
		}

	}
}
```

### Reduce

After all Map tasks are done, we begin to start Reduce task.

1. Load all files for one specific part.
2. Extract the Key-Value pairs
3. Sort the Key-Value pairs
4. Call reduce function on each distinct Key
5. Export the result to file

There are a lot of code you can use from `mrsequential.go`, such as read and write function, and also how to extract the Key-Value pairs, and how to sort and count them.

Remember, we have to load all intermediate files for one PartIndex, since they are supposed to contain same Keys based on the hash function.

For example for part 1, we have to load `mr-0-1, mr-1-1, to ... mr-9-1`.

```go
func workerReduce(reducef func(string, []string) string, taskInfo *TaskInfo) {

	fmt.Printf("Reducing on part %v\n", taskInfo.PartIndex)
	//// Read all files with the same PartIndex

	intermediate := []KeyValue{}
	for i := 0; i < taskInfo.NFiles; i++ {
		filename := "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(taskInfo.PartIndex)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
		file.Close()
	}

	//
	// a big difference from real MapReduce is that all the
	// intermediate data is in one place, intermediate[],
	// rather than being partitioned into NxM buckets.
	//

	sort.Sort(ByKey(intermediate))

	outname := "mr-out-" + strconv.Itoa(taskInfo.PartIndex)
	ofile, _ := os.Create(outname)

	//
	// call Reduce on each distinct key in intermediate[],
	// and print the result to mr-out-0.
	//
	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}

	ofile.Close()
	CallTaskDone(taskInfo)
}

```

Now we have to add mechanism to deal the running task.

1. Add two queues in the master `mapTaskRunning` and `reduceTaskRunning`
```go
type Master struct {
	mapTaskQueuing    TaskQueue
	reduceTaskQueuing TaskQueue
	mapTaskRunning    TaskQueue
	reduceTaskRunning TaskQueue
	isDone            bool
	filenames         []string
}

```
2. Rewrite AskTask action to cope with the two queues just created

```go
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
```
3. Add TaskDone action, so the worker call tell the master the task is done when it's done.

Note: We generate all the Reduce tasks only when all the Map tasks are done.

```go
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
```

Generate all the Reduce tasks.

```go
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
```

4. After the task is done, we have to remove the task from the queue. So implement the RemoveTask on TaskQueue

```go
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
```


### Run 

Now we can run the test `test-mr.sh`

This script creates the `mr-tmp` folder by itself and run the code inside the folder.

You have to adjust the path in your worker if the path was different.

```bash
cd src/main
sh test-mr.sh

```
You can see we pass almost all the tests, except the crash test. That's our next topic.

## Part 3: Handle Crash

To handle crash, the hints of the lab has mentioned two solutions

1. Use temporary files to prevent worker from reading the unfinished intermediate. Rename it to output file after the task is done. 
2. Use a timeout mechanism to re-start mapper or reducer. We can assume after 10 secs, if the task is still not done, we can move it back to the waiting queue.


See all Part 3 code change [here](https://github.com/alfmunny/MIT6.824-Distributed-Systems/commit/ab979016224c7e8f58b9e44c58bf7ff45e8a1251).

### Temporary files

Modify the `wokerMap` and `workerReduce`.


```go
func workerMap(mapf func(string, string) []KeyValue, taskInfo *TaskInfo) {
....
	for outindex := 0; outindex < nReduce; outindex++ {
		outFiles[outindex], _ = ioutil.TempFile("", "mr-tmp-*")
		fileEncs[outindex] = json.NewEncoder(outFiles[outindex])
	}

	...

	for outindex, file := range outFiles {
		outname := outprefix + strconv.Itoa(outindex)
		oldpath := filepath.Join(file.Name())
		os.Rename(oldpath, outname)
		file.Close()
	}
...
}
```



```go
func workerReduce(reducef func(string, []string) string, taskInfo *TaskInfo) {

...

	ofile, _ := ioutil.TempFile("", "mr-tmp-*")

	....

	os.Rename(filepath.Join(ofile.Name()), outname)
	ofile.Close()
	CallTaskDone(taskInfo)

}
```

### Timeout

We add a new field in the `TaskInfo` to record a timestamp.

`rpc.go`
```go
type TaskInfo {
	TimeStamp time.Time
}
```

Collect the timeout Tasks back to queue.

`master.go`

```go
func (this *TaskInfo) OutOfTime() bool {
	return time.Now().Sub(this.TimeStamp) > time.Duration(time.Second*10)
}

func (this *TaskInfo) SetNow() {
	this.TimeStamp = time.Now()
}

func (tq *TaskQueue) TimeOutQueue() []TaskInfo {
	ret := make([]TaskInfo, 0)
	tq.lock()
	for i := 0; i < tq.Size(); {
		t := tq.taskArray[i]
		if t.OutOfTime() {
			tq.taskArray = append(tq.taskArray[:i], tq.taskArray[i+1:]...)
			ret = append(ret, t)
		} else {
			i++
		}
	}
	tq.unlock()
	return ret
}

func (m *Master) CollectTimeOutQueue() {
	for {
		time.Sleep(time.Duration(5 * time.Second))
		timeoutQueue := m.reduceTaskRunning.TimeOutQueue()

		if len(timeoutQueue) > 0 {
			m.reduceTaskQueuing.lock()
			m.reduceTaskQueuing.taskArray = append(m.reduceTaskQueuing.taskArray, timeoutQueue...)

			m.reduceTaskQueuing.unlock()
		}

		timeoutQueue = m.mapTaskRunning.TimeOutQueue()

		if len(timeoutQueue) > 0 {
			fmt.Println(timeoutQueue)
			m.mapTaskQueuing.lock()
			m.mapTaskQueuing.taskArray = append(m.mapTaskQueuing.taskArray, timeoutQueue...)
			m.mapTaskQueuing.unlock()

		}
	}
}

```

Set the TimeStamp when a new Task is added to the queue.

```go
func (m *Master) AskTask(args *ExampleArgs, reply *TaskInfo) error {
	if m.reduceTaskQueuing.Size() > 0 {
		taskInfo := m.reduceTaskQueuing.Pop()
		taskInfo.SetNow()
		m.reduceTaskRunning.Push(taskInfo)
		*reply = taskInfo
		fmt.Printf("%v sent to reducer\n", taskInfo.FileName)
		return nil
	}

	if m.mapTaskQueuing.Size() > 0 {
		taskInfo := m.mapTaskQueuing.Pop()
		taskInfo.SetNow()
		m.mapTaskRunning.Push(taskInfo)
		*reply = taskInfo
		fmt.Printf("%v sent to mapper\n", taskInfo.FileName)
		return nil
	}
	...
}
```

Start the collect service. Don't forget add the `TimeStamp` to initializer.

```go
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{}
	m.filenames = files

	// Your code here.
	for fileindex, filename := range files {
		taskInfo := TaskInfo{time.Now(), TaskMap, filename, fileindex, 0, nReduce, len(files)}
		m.mapTaskQueuing.Push(taskInfo)
	}

	go m.CollectTimeOutQueue()

	m.server()
	return &m
}

```

Now if you run `sh test-mr.sh`, you should pass all tests.
