package mr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(Key, []Value) Value) {
	server()

	for {
		reply := askCoordinatorForTask()

		if reply == nil {
			debugf("worker: no task received, exiting")
			return
		}

		err := executeTask(reply, mapf, reducef)
		if err != nil {
			debugf("worker: task %v failed: %v", reply.Task.Type, err)
		}
	}
}

func askCoordinatorForTask() *TaskReply {
	args := &NoArgs{}
	reply := &TaskReply{}

	ok := CallCoordinator("RequestTask", args, reply)
	if !ok {
		debugf("worker: coordinator unreachable, assuming job is done and exiting")
		return nil
	}

	return reply
}

func executeTask(reply *TaskReply, mapf func(string, string) []KeyValue, reducef func(Key, []Value) Value) error {
	if reply.Task == nil {
		return fmt.Errorf("worker: missing task in reply")
	}

	switch reply.Task.Type {
	case TaskMap:
		if reply.MapData == nil {
			return fmt.Errorf("worker: map task %d missing data", reply.Task.ID)
		} else if err := runMapTask(reply.Task, reply.MapData, mapf); err != nil {
			return err
		}
	case TaskReduce:
		if reply.ReduceData == nil {
			return fmt.Errorf("worker: reduce task %d missing data", reply.Task.ID)
		} else if err := runReduceTask(reply.Task, reply.ReduceData, reducef); err != nil {
			return err
		}
	case TaskWait:
		time.Sleep(time.Second)
		return nil
	default:
		return fmt.Errorf("worker: unsupported task type %v", reply.Task.Type)
	}

	reportTaskCompletion(reply.Task)
	return nil
}

func runMapTask(task *Task, data *MapTaskData, mapf func(string, string) []KeyValue) error {
	debugf("worker: processing map task (id: %d, key: %s)", task.ID, data.Key)

	mappedContents := mapf(data.Key, data.Content)
	buckets := bucketContents(mappedContents, data.Buckets)
	return writeBucketsToFiles(task.ID, buckets)
}

func bucketContents(keyValues []KeyValue, nBuckets int) [][]KeyValue {
	buckets := make([][]KeyValue, nBuckets)
	for _, kv := range keyValues {
		bucket := ihash(kv.Key) % nBuckets
		buckets[bucket] = append(buckets[bucket], kv)
	}
	return buckets
}

func writeBucketsToFiles(taskID TaskID, buckets [][]KeyValue) error {
	for bucket, keyValues := range buckets {
		err := writeBucketToFile(taskID, bucket, keyValues)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeBucketToFile(taskID TaskID, bucket int, keyValues []KeyValue) error {
	filename := taskIntermediateFileName(taskID, bucket)

	tmp, err := os.CreateTemp(filepath.Dir(filename), "mr-bucket-*")
	if err != nil {
		return fmt.Errorf("worker: cannot create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	encoder := json.NewEncoder(tmp)
	for _, kv := range keyValues {
		err := encoder.Encode(&kv)
		if err != nil {
			tmp.Close()
			return fmt.Errorf("worker: cannot write %s: %w", filename, err)
		}
	}

	err = tmp.Close()
	if err != nil {
		return fmt.Errorf("worker: cannot close temp file: %w", err)
	}

	err = os.Rename(tmp.Name(), filename)
	if err != nil {
		return fmt.Errorf("worker: cannot rename temp file to %s: %w", filename, err)
	}

	return nil
}

func runReduceTask(task *Task, data *ReduceTaskData, reducef func(Key, []Value) Value) error {
	debugf("worker: processing reduce task (id: %d, bucket: %d)", task.ID, data.Bucket)

	mappedContents := make([]KeyValue, 0)
	for mapTaskID, producerAddress := range data.MapTaskOutputs {
		var (
			bucketContents []KeyValue
			err            error
		)

		if producerAddress == workerAddress {
			bucketContents, err = readBucketFile(mapTaskID, data.Bucket)
		} else {
			bucketContents, err = fetchBucketFromWorker(producerAddress, mapTaskID, data.Bucket)
		}

		if err != nil {
			reportMissingMapOutputs(mapTaskID)
			return err
		}

		mappedContents = append(mappedContents, bucketContents...)
	}

	groupedContents := groupByKey(mappedContents)
	reducedContents := reduceContents(groupedContents, reducef)
	return writeReducedContentsToBucketFile(data.Bucket, reducedContents)
}

func groupByKey(keyValues []KeyValue) map[Key][]Value {
	grouped := make(map[Key][]Value)
	for _, kv := range keyValues {
		grouped[kv.Key] = append(grouped[kv.Key], kv.Value)
	}
	return grouped
}

func reduceContents(groupedContents map[Key][]Value, reducef func(Key, []Value) Value) []KeyValue {
	sortedKeys := make([]string, 0, len(groupedContents))
	for key := range groupedContents {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	reducedContents := make([]KeyValue, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		values := groupedContents[key]
		reducedContents = append(reducedContents, KeyValue{Key: key, Value: reducef(key, values)})
	}
	return reducedContents
}

func reportTaskCompletion(task *Task) {
	args := &ReportTaskCompletionArgs{
		Task:          task,
		WorkerAddress: workerAddress,
	}

	CallCoordinator("ReportTaskCompletion", args, &NoArgs{})
}

func reportMissingMapOutputs(mapTaskID TaskID) {
	args := &ReportMissingMapOutputsArgs{
		MapTaskID: mapTaskID,
	}

	CallCoordinator("ReportMissingMapOutputs", args, &NoArgs{})
}

func reduceOutputFileName(bucket int) string {
	return fmt.Sprintf("mr-out-%d", bucket)
}

func fetchBucketFromWorker(address string, mapTaskID TaskID, bucket int) ([]KeyValue, error) {
	args := &FetchBucketArgs{
		MapTaskID: mapTaskID,
		Bucket:    bucket,
	}
	reply := &FetchBucketReply{}

	if !CallWorker(address, "FetchBucket", args, reply) {
		return nil, fmt.Errorf("worker: failed to fetch map %d bucket %d from %s", mapTaskID, bucket, address)
	}

	return reply.KeyValues, nil
}
