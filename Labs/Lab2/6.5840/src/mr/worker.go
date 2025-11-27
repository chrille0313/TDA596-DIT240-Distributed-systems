package mr

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

type KeyGroup struct {
	Key    string
	Values []string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
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
			log.Printf("worker: task request failed")
			return nil
		}

		if reply.Task == nil {
			log.Printf("worker: received reply without task, retrying")
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if reply.Task.Type == TaskWait {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		return reply
	}
}

func executeTask(reply *TaskReply, mapf func(string, string) []KeyValue, reducef func(string, []string) string) error {
	if reply.Task == nil {
		return fmt.Errorf("worker: missing task in reply")
	}

	switch reply.Task.Type {
	case TaskMap:
		if reply.MapData == nil {
			return fmt.Errorf("worker: map task %d missing data", reply.Task.ID)
		}
		return runMapTask(reply.Task, reply.MapData, mapf)
	case TaskReduce:
		if reply.ReduceData == nil {
			return fmt.Errorf("worker: reduce task %d missing data", reply.Task.ID)
		}
		return runReduceTask(reply.Task, reply.ReduceData, reducef)
	default:
		return fmt.Errorf("worker: unsupported task type %v", reply.Task.Type)
	}
}

func runMapTask(task *Task, data *MapTaskData, mapf func(string, string) []KeyValue) error {
	log.Printf("worker: processing map task %d (%s)", task.ID, data.File)

	mappedContents, err := mapContents(data.File, mapf)
	if err != nil {
		return err
	}

	groupedContents := groupMappedContents(mappedContents)
	return outputMappedContents(task.ID, data.Buckets, groupedContents)
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

func groupMappedContents(mappedContents []KeyValue) []KeyGroup {
	grouped := make(map[string][]string)
	for _, kv := range mappedContents {
		grouped[kv.Key] = append(grouped[kv.Key], kv.Value)
	}

	result := make([]KeyGroup, 0, len(grouped))
	for key, values := range grouped {
		result = append(result, KeyGroup{Key: key, Values: values})
	}

	return result
}

func outputMappedContents(taskID TaskID, buckets int, mappedContents []KeyGroup) error {
	for _, item := range mappedContents {
		hash := ihash(item.Key)
		bucket := hash % buckets
		outputFileName := getOutputFileName(taskID, bucket)

		f, err := os.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("worker: cannot open %s: %w", outputFileName, err)
		}

		_, err = fmt.Fprintf(f, "%v %v\n", item.Key, item.Values)
		if err != nil {
			f.Close()
			return fmt.Errorf("worker: cannot write to %s: %w", outputFileName, err)
		}

		err = f.Close()
		if err != nil {
			return fmt.Errorf("worker: cannot close %s: %w", outputFileName, err)
		}
	}
	return nil
}

func getOutputFileName(taskID TaskID, bucket int) string {
	return "mr-out-" + strconv.Itoa(int(taskID)) + "-" + strconv.Itoa(bucket)
}

func runReduceTask(task *Task, data *ReduceTaskData, reducef func(string, []string) string) error {
	// TODO: implement
	_, _, _ = task, data, reducef
	log.Fatal("worker: not implemented")
	return nil
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
