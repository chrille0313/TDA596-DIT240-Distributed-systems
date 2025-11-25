package mr

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"strconv"
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
	task := askForTask()

	if task.Type == TaskMap {
		fmt.Println("Worker received file:", task.File)
		mappedContents := mapContents(task.File, mapf)
		groupContents := groupMappedContents(mappedContents)
		outputMappedContents(task, groupContents)
	}
}

func askForTask() TaskReply {
	args := NoArgs{}
	reply := TaskReply{}

	for reply.Type == TaskNone || reply.Type == TaskWait {
		ok := call("Coordinator.GetTask", &args, &reply)
		if !ok {
			fmt.Printf("call failed!\n")
		}
	}

	return reply
}

func mapContents(filePath string, mapf func(string, string) []KeyValue) []KeyValue {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("cannot open %v", filePath)
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filePath)
	}
	file.Close()
	return mapf(filePath, string(content))
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

func outputMappedContents(task TaskReply, mappedContents []KeyGroup) {
	for _, item := range mappedContents {
		hash := ihash(item.Key)
		bucket := hash % task.Buckets
		outputFileName := getOutputFileName(task, bucket)

		f, err := os.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("cannot open %v", outputFileName)
		}
		defer f.Close()

		_, err = fmt.Fprintf(f, "%v %v\n", item.Key, item.Values)
		if err != nil {
			log.Fatalf("cannot write to %v", outputFileName)
		}
	}
}

func getOutputFileName(task TaskReply, bucket int) string {
	return "mr-out-" + strconv.Itoa(task.Id) + "-" + strconv.Itoa(bucket)
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
