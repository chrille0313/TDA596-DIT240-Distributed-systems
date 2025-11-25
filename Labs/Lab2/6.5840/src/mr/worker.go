package mr

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
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
		mappedContents := mapContents(task.File, mapf)
		outputMappedContents(mappedContents, task.Buckets)
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

func outputMappedContents(mappedContents []KeyValue, buckets int) {
	for _, item := range mappedContents {
		hash := ihash(item.Key)
		bucket := hash % buckets
		
	}

	fmt.Println(mappedContents)
}

func getOutputFileName(bucket int) string {
	return "mr-out-" + strconv.Itoa(bucket)
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