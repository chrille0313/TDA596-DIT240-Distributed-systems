package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type ID int

type TaskState int

const (
	TaskStateNone TaskState = iota
	TaskStateUnassigned
	TaskStateRunning
	TaskStateDone
)

type MapTask struct {
	id         ID
	filePath   string
	state      TaskState
	assignedAt time.Time
}

func (t *MapTask) isTimedOut() bool {
	return t.state == TaskStateRunning && time.Now().After(t.assignedAt.Add(10*time.Second))
}

type Coordinator struct {
	nReduce int
	mu      sync.Mutex

	mapTasks map[ID]*MapTask
	// reduceTasks  [ID]Task
}

func (c *Coordinator) GetTask(args *NoArgs, reply *TaskReply) error {
	reply.File = ""
	c.mu.Lock()

	mapTasksDone := true
	for i, task := range c.mapTasks {
		if task.state == TaskStateUnassigned || task.isTimedOut() {
			taskID := ID(i)

			reply.Id = taskID
			reply.Type = TaskMap
			reply.File = task.filePath
			reply.Buckets = c.nReduce

			task.state = TaskStateRunning
			task.assignedAt = time.Now()
			mapTasksDone = false
			break
		}
	}
	c.mu.Unlock()

	if mapTasksDone {
		// All map tasks are done, assign reduce tasks here
	}

	return nil
}

func (c *Coordinator) SignalDone() {

	c.mapTasks[0].state = TaskStateDone
}

// start a thread that listens for RPCs from worker.go
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

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false

	// Your code here.

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		mapTasks: make(map[ID]*MapTask, len(files)),
		nReduce:  nReduce,
	}

	for i, filePath := range files {
		taskID := ID(i)
		c.mapTasks[taskID] = &MapTask{id: taskID, filePath: filePath, state: TaskStateUnassigned}
	}

	c.server()
	return &c
}
