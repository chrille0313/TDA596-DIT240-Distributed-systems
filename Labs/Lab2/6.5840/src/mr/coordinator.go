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

type MapFile struct {
	path  string
	inUse bool
}

type Coordinator struct {
	nReduce      int
	files        []*MapFile
	mu           sync.Mutex
	timeoutEvent chan ID
	taskTimeouts TimeoutList
}

func (c *Coordinator) GetTask(args *NoArgs, reply *TaskReply) error {
	reply.File = ""
	c.mu.Lock()
	for i, file := range c.files {
		if !file.inUse {
			reply.Id = i
			reply.Type = TaskMap
			reply.File = file.path
			reply.Buckets = c.nReduce
			file.inUse = true

			// Start a timeout event for this task
			break
		}
	}
	c.mu.Unlock()
	return nil
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

func (c *Coordinator) handleTimeouts() {
	for {
		currentTime := time.Now()
		expired := c.taskTimeouts.PopExpired(currentTime)

		for _, taskID := range expired {
			
		}
	}
}


// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{
		files:   make([]*MapFile, len(files)),
		nReduce: nReduce,
	}

	for i, filePath := range files {
		c.files[i] = &MapFile{path: filePath}
	}

	c.server()
	go c.handleTimeouts()
	return &c
}
