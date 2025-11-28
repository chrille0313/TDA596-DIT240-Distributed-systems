package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"time"
)

type Key = string
type Value = string

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   Key
	Value Value
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(Key, []Value) Value) {
	for {
		reply := askForTask()

		if reply == nil {
			log.Printf("worker: no task received, exiting")
			return
		}

		err := executeTask(reply, mapf, reducef)
		if err != nil {
			log.Printf("worker: %v", err)
		}
	}
}

func askForTask() *TaskReply {
	args := NoArgs{}
	reply := &TaskReply{}

	for {
		ok := call("Coordinator.RequestTask", &args, &reply)
		if !ok {
			log.Printf("worker: coordinator unreachable, assuming job is done and exiting")
			return nil
		}

		if reply.Task == nil {
			log.Printf("worker: received reply without task, retrying")
			time.Sleep(200 * time.Millisecond)
			continue
		}

		return reply
	}
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
		time.Sleep(200 * time.Millisecond)
		return nil
	default:
		return fmt.Errorf("worker: unsupported task type %v", reply.Task.Type)
	}

	call("Coordinator.ReportTaskCompletion", reply.Task, &NoArgs{})
	return nil
}

func runMapTask(task *Task, data *MapTaskData, mapf func(string, string) []KeyValue) error {
	log.Printf("worker: processing map task %d (%s)", task.ID, data.File)

	mappedContents, err := mapContents(data.File, mapf)
	if err != nil {
		return err
	}

	buckets := bucketContents(mappedContents, data.Buckets)
	return writeBucketsToFiles(task.ID, buckets)
}

func mapContents(filePath string, mapf func(string, string) []KeyValue) ([]KeyValue, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("worker: cannot open %s: %w", filePath, err)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("worker: cannot read %s: %w", filePath, err)
	}

	err = file.Close()
	if err != nil {
		return nil, fmt.Errorf("worker: cannot close %s: %w", filePath, err)
	}

	return mapf(filePath, string(content)), nil
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
	filename := intermediateFileName(taskID, bucket)

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
	log.Printf("worker: processing reduce task %d (bucket %d)", task.ID, data.Bucket)

	mappedContents := make([]KeyValue, 0)
	for _, mapTaskID := range data.MapTasks {
		bucketContents, err := readBucketFile(mapTaskID, data.Bucket)
		if err != nil {
			return err
		}

		mappedContents = append(mappedContents, bucketContents...)
	}

	groupedContents := groupByKey(mappedContents)
	reducedContents := reduceContents(groupedContents, reducef)
	return writeReducedContentsToFile(data.Bucket, reducedContents)
}

func readBucketFile(taskID TaskID, bucket int) ([]KeyValue, error) {
	filename := intermediateFileName(taskID, bucket)
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("worker: cannot open %s: %w", filename, err)
	}
	defer file.Close()

	keyValues := make([]KeyValue, 0)
	decoder := json.NewDecoder(file)
	for {
		var kv KeyValue
		err := decoder.Decode(&kv)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("worker: cannot decode %s: %w", filename, err)
		}
		keyValues = append(keyValues, kv)
	}
	return keyValues, nil
}

func groupByKey(keyValues []KeyValue) map[Key][]Value {
	grouped := make(map[Key][]Value)
	for _, kv := range keyValues {
		grouped[kv.Key] = append(grouped[kv.Key], kv.Value)
	}
	return grouped
}

func reduceContents(groupedContents map[Key][]Value, reducef func(Key, []Value) Value) []KeyValue {
	reducedContents := make([]KeyValue, 0, len(groupedContents))
	for key, values := range groupedContents {
		reducedContents = append(reducedContents, KeyValue{Key: key, Value: reducef(key, values)})
	}
	return reducedContents
}

func writeReducedContentsToFile(bucket int, reducedContents []KeyValue) error {
	filename := reduceOutputFileName(bucket)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("worker: cannot create %s: %w", filename, err)
	}
	defer file.Close()

	for _, kv := range reducedContents {
		fmt.Fprintf(file, "%v %v\n", kv.Key, kv.Value)
	}

	return nil
}

func intermediateFileName(mapID TaskID, bucket int) string {
	return fmt.Sprintf("mr-%d-%d", mapID, bucket)
}

func reduceOutputFileName(bucket int) string {
	return fmt.Sprintf("mr-out-%d", bucket)
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
