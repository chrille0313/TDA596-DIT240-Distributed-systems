package mr

type NoArgs struct{}

type TaskReply struct {
	Task       *Task
	MapData    *MapTaskData
	ReduceData *ReduceTaskData
}

type MapTaskData struct {
	Key     string
	Content string
	Buckets int
}

type ReduceTaskData struct {
	Bucket         int
	MapTaskOutputs map[TaskID]string // TaskID -> Address of worker which produced the output
}

type ReportTaskCompletionArgs struct {
	Task          *Task
	WorkerAddress string
}

type ReportMissingMapOutputsArgs struct {
	MapTaskID TaskID
}

type FetchBucketArgs struct {
	MapTaskID TaskID
	Bucket    int
}

type FetchBucketReply struct {
	KeyValues []KeyValue
}
