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

type Coordinator struct {
	nReduce int
	mu      sync.Mutex

	mapTasks map[TaskID]*MapTask
	// reduceTasks map[ID]*Task
}

func (c *Coordinator) RequestTask(args *NoArgs, reply *TaskReply) error {
	*reply = TaskReply{}
	now := time.Now()

	c.mu.Lock()
	mapTask := c.pickMapTaskLocked(now)
	c.mu.Unlock()

	if mapTask != nil {
		reply.Task = mapTask.Task
		reply.MapData = &MapTaskData{File: mapTask.File, Buckets: mapTask.Buckets}
	} else {
		reply.Task = &Task{Type: TaskWait, State: TaskStateUnassigned, AssignedAt: now}
	}

	return nil
}

func (c *Coordinator) pickMapTaskLocked(now time.Time) *MapTask {
	for _, mapTask := range c.mapTasks {
		if c.shouldAssignTask(mapTask.Task) {
			mapTask.Task.MarkRunning(now)
			return mapTask
		}
	}
	return nil
}

func (c *Coordinator) shouldAssignTask(task *Task) bool {
	return task.State == TaskStateUnassigned || (task.State == TaskStateRunning && !task.IsTimedOut())
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
		mapTasks: make(map[TaskID]*MapTask, len(files)),
		nReduce:  nReduce,
	}

	for i, filePath := range files {
		taskID := TaskID(i)
		c.mapTasks[taskID] = &MapTask{Task: &Task{ID: taskID, Type: TaskMap, State: TaskStateUnassigned}, File: filePath, Buckets: nReduce}
	}

	c.server()
	return &c
}
