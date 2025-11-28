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
	mu sync.Mutex

	mapTasks    map[TaskID]*MapTask
	reduceTasks map[TaskID]*ReduceTask
}

func (c *Coordinator) RequestTask(args *NoArgs, reply *TaskReply) error {
	*reply = TaskReply{}
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	mapTask := c.pickMapTaskLocked(now);
	if mapTask != nil {
		reply.Task = mapTask.Task
		reply.MapData = &MapTaskData{File: mapTask.File, Buckets: mapTask.Buckets}
		return nil
	}

	if !c.allMapTasksDoneLocked() {
		reply.Task = &Task{Type: TaskWait, State: TaskStateUnassigned, AssignedAt: now}
		return nil
	}

	reduceTask := c.pickReduceTaskLocked(now);
	if reduceTask != nil {
		reply.Task = reduceTask.Task
		reply.ReduceData = &ReduceTaskData{Bucket: reduceTask.Bucket, MapTasks: reduceTask.MapTasks}
		return nil
	}

	reply.Task = &Task{Type: TaskWait, State: TaskStateUnassigned, AssignedAt: now}
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

func (c *Coordinator) pickReduceTaskLocked(now time.Time) *ReduceTask {
	for _, reduceTask := range c.reduceTasks {
		if c.shouldAssignTask(reduceTask.Task) {
			reduceTask.Task.MarkRunning(now)
			return reduceTask
		}
	}
	return nil
}

func (c *Coordinator) shouldAssignTask(task *Task) bool {
	return task.State == TaskStateUnassigned || (task.State == TaskStateRunning && task.IsTimedOut())
}

func (c *Coordinator) ReportTaskCompletion(args *Task, reply *NoArgs) error {
	log.Printf("coordinator: received task completion report for task %d", args.ID)

	c.mu.Lock()
	defer c.mu.Unlock()

	switch args.Type {
	case TaskMap:
		mapTask, exists := c.mapTasks[args.ID]
		if exists && mapTask.Task.State != TaskStateDone {
			mapTask.Task.State = TaskStateDone

			for _, reduceTask := range c.reduceTasks {
				reduceTask.MapTasks = append(reduceTask.MapTasks, mapTask.Task.ID)
			}
		}
	case TaskReduce:
		reduceTask, exists := c.reduceTasks[args.ID]
		if exists {
			reduceTask.Task.State = TaskStateDone
		}
	}
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

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduceTasks is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduceTasks int) *Coordinator {
	c := Coordinator{
		mapTasks:    make(map[TaskID]*MapTask, len(files)),
		reduceTasks: make(map[TaskID]*ReduceTask, nReduceTasks),
	}
	nMapTasks := len(files)

	log.Printf("coordinator: starting with %d map tasks and %d reduce tasks", nMapTasks, nReduceTasks)

	for i, filePath := range files {
		taskID := TaskID(i)
		c.mapTasks[taskID] = &MapTask{Task: &Task{ID: taskID, Type: TaskMap, State: TaskStateUnassigned}, File: filePath, Buckets: nReduceTasks}
	}

	for i := 0; i < nReduceTasks; i++ {
		taskID := TaskID(i)
		c.reduceTasks[taskID] = &ReduceTask{
			Task:     &Task{ID: taskID, Type: TaskReduce, State: TaskStateUnassigned},
			Bucket:   i,
			MapTasks: make([]TaskID, 0, nMapTasks),
		}
	}

	c.server()
	return &c
}

func (c *Coordinator) allMapTasksDoneLocked() bool {
	for _, mapTask := range c.mapTasks {
		if mapTask.Task.State != TaskStateDone {
			return false
		}
	}
	return true
}

func (c *Coordinator) allReduceTasksDoneLocked() bool {
	for _, reduceTask := range c.reduceTasks {
		if reduceTask.Task.State != TaskStateDone {
			return false
		}
	}
	return true
}
