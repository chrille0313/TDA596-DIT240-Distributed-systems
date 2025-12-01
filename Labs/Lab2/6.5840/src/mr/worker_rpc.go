package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

var workerAddress string

// Start a worker server that listens for RPCs over HTTP.
func server() string {
	if workerAddress != "" {
		return workerAddress
	}

	listenAddr := getWorkerListenAddress()
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("worker: cannot listen on %s: %v", listenAddr, err)
	}

	workerAddress = listener.Addr().String()

	if err := rpc.RegisterName("WorkerData", &WorkerDataService{}); err != nil {
		log.Fatalf("worker: cannot register data service: %v", err)
	}

	rpc.HandleHTTP()

	go func() {
		debugf("worker: listening on %s", workerAddress)
		if err := http.Serve(listener, nil); err != nil {
			log.Fatalf("worker: cannot start server: %v", err)
		}
	}()

	return workerAddress
}

// CallWorker sends an RPC request to another worker and waits for the reply.
func CallWorker(address string, rpcname string, args interface{}, reply interface{}) bool {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		debugf("worker: failed to dial %s: %v", address, err)
		return false
	}
	defer client.Close()

	if err := client.Call("WorkerData."+rpcname, args, reply); err != nil {
		debugf("worker: RPC %s failed: %v", rpcname, err)
		return false
	}

	return true
}

func getWorkerListenAddress() string {
	if addr := os.Getenv("MR_WORKER_ADDRESS"); addr != "" {
		return addr
	}
	return "127.0.0.1:"
}

/*
 * RPC handlers
 */

type WorkerDataService struct{}

// Fetch content of a specific map/reduce bucket.
func (s *WorkerDataService) FetchBucket(args *FetchBucketArgs, reply *FetchBucketReply) error {
	keyValues, err := readBucketFile(args.MapTaskID, args.Bucket)
	if err != nil {
		return err
	}

	reply.KeyValues = keyValues
	return nil
}
