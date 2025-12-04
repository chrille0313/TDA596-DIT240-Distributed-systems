package mr

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Coordinator struct {
	mu          sync.Mutex
	mapTasks    map[TaskID]*MapTask
	reduceTasks map[TaskID]*ReduceTask
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduceTasks is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduceTasks int) *Coordinator {
	c := &Coordinator{
		mapTasks:    make(map[TaskID]*MapTask, len(files)),
		reduceTasks: make(map[TaskID]*ReduceTask, nReduceTasks),
	}
	nMapTasks := len(files)

	debugf("coordinator: starting with %d map tasks and %d reduce tasks", nMapTasks, nReduceTasks)

	for _, filePath := range files {
		taskID := TaskID(getGlobalID())
		c.mapTasks[taskID] = &MapTask{Task: &Task{ID: taskID, Type: TaskMap, State: TaskStateUnassigned}, File: filePath, Buckets: nReduceTasks}
	}

	for bucket := 0; bucket < nReduceTasks; bucket++ {
		taskID := TaskID(getGlobalID())
		c.reduceTasks[taskID] = &ReduceTask{
			Task:           &Task{ID: taskID, Type: TaskReduce, State: TaskStateUnassigned},
			Bucket:         bucket,
			MapTaskOutputs: make(map[TaskID]string, nMapTasks),
		}
	}

	c.server()
	return c
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.allMapTasksDoneLocked() && c.allReduceTasksDoneLocked()
}

func (c *Coordinator) RequestTask(args *NoArgs, reply *TaskReply) error {
	*reply = TaskReply{}

	c.mu.Lock()
	defer c.mu.Unlock()

	mapTask := c.pickMapTaskLocked()
	if mapTask != nil {
		reply.Task = mapTask.Task
		reply.Task.MarkRunning(time.Now())

		fileContent, err := os.ReadFile(mapTask.File)
		if err != nil {
			return fmt.Errorf("coordinator: cannot read file %s: %w", mapTask.File, err)
		}

		reply.MapData = &MapTaskData{Key: mapTask.File, Content: string(fileContent), Buckets: mapTask.Buckets}
		return nil
	}

	if !c.allMapTasksDoneLocked() {
		reply.Task = &Task{Type: TaskWait}
		reply.Task.MarkRunning(time.Now())
		return nil
	}

	reduceTask := c.pickReduceTaskLocked()
	if reduceTask != nil {
		reply.Task = reduceTask.Task
		reply.Task.MarkRunning(time.Now())
		reply.ReduceData = &ReduceTaskData{
			Bucket:         reduceTask.Bucket,
			MapTaskOutputs: reduceTask.MapTaskOutputs,
		}
		return nil
	}

	reply.Task = &Task{Type: TaskWait}
	reply.Task.MarkRunning(time.Now())
	return nil
}

func (c *Coordinator) ReportTaskCompletion(args *ReportTaskCompletionArgs, reply *NoArgs) error {
	task := args.Task
	debugf("coordinator: received task completion report for task (id: %d)", task.ID)

	c.mu.Lock()
	defer c.mu.Unlock()

	switch task.Type {
	case TaskMap:
		mapTask, exists := c.mapTasks[task.ID]
		if exists && mapTask.Task.State != TaskStateDone {
			mapTask.Task.State = TaskStateDone

			for _, reduceTask := range c.reduceTasks {
				reduceTask.MapTaskOutputs[mapTask.Task.ID] = args.WorkerAddress
			}
		}
	case TaskReduce:
		reduceTask, exists := c.reduceTasks[task.ID]
		if exists {
			reduceTask.Task.State = TaskStateDone
		}
	}

	return nil
}

func (c *Coordinator) ReportMissingMapOutputs(args *ReportMissingMapOutputsArgs, reply *NoArgs) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	mapTask, exists := c.mapTasks[args.MapTaskID]
	if !exists {
		return nil
	}

	if mapTask.Task.State == TaskStateDone {
		debugf("coordinator: map task %d outputs unavailable; rescheduling", mapTask.Task.ID)
		mapTask.Task.State = TaskStateUnassigned
		mapTask.Task.AssignedAt = time.Time{}
		for _, reduceTask := range c.reduceTasks {
			delete(reduceTask.MapTaskOutputs, mapTask.Task.ID)
		}
	}

	return nil
}

func (c *Coordinator) pickMapTaskLocked() *MapTask {
	for _, mapTask := range c.mapTasks {
		if c.shouldAssignTask(mapTask.Task) {
			return mapTask
		}
	}
	return nil
}

func (c *Coordinator) pickReduceTaskLocked() *ReduceTask {
	for _, reduceTask := range c.reduceTasks {
		if c.shouldAssignTask(reduceTask.Task) {
			return reduceTask
		}
	}
	return nil
}

func (c *Coordinator) shouldAssignTask(task *Task) bool {
	return task.State == TaskStateUnassigned || (task.State == TaskStateRunning && task.IsTimedOut())
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
