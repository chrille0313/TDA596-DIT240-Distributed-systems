package mr

type TaskReply struct {
	Task       *Task
	MapData    *MapTaskData
	ReduceData *ReduceTaskData
}

type MapTaskData struct {
	Key string
	Value string
	Buckets int
}

type ReduceTaskData struct {
	Bucket   int
	MapTaskOutputs map[TaskID]string
}