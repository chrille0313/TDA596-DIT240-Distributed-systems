package mr

import "time"

type TaskID int

type TaskState int

const (
	TaskStateNone TaskState = iota
	TaskStateUnassigned
	TaskStateRunning
	TaskStateDone
)

type TaskType int

const (
	TaskNone TaskType = iota
	TaskWait
	TaskMap
	TaskReduce
)

const taskTimeout = 10 * time.Second

type Task struct {
	ID         TaskID
	Type       TaskType
	State      TaskState
	AssignedAt time.Time
}

func (t *Task) IsTimedOut() bool {
	return time.Since(t.AssignedAt) >= taskTimeout
}

func (t *Task) MarkRunning(now time.Time) {
	t.State = TaskStateRunning
	t.AssignedAt = now
}

type MapTask struct {
	Task    *Task
	File    string
	Buckets int
}

type ReduceTask struct {
	Task           *Task
	Bucket         int
	MapTaskOutputs map[TaskID]string // TaskID -> Adress of worker which produced the output
}
