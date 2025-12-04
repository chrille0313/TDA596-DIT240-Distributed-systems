package mr

import (
	"log"
	"net/http"
	"net/rpc"
	"os"
)

// Start a coordinator server that listens for RPCs over HTTP.
func (c *Coordinator) server() string {
	if err := rpc.Register(c); err != nil {
		log.Fatalf("coordinator: cannot register RPC server: %v", err)
	}

	rpc.HandleHTTP()
	address := getCoordinatorAddress()

	go func() {
		debugf("coordinator: listening on %s", address)
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Fatalf("coordinator: cannot start server: %v", err)
		}
	}()

	return address
}

// Send an RPC request to the coordinator, wait for the response.
// returns false if something goes wrong.
func CallCoordinator(rpcname string, args interface{}, reply interface{}) bool {
	address := getCoordinatorAddress()
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		debugf("worker: failed to dial coordinator at %s: %v", address, err)
		return false
	}
	defer client.Close()

	if err := client.Call("Coordinator." + rpcname, args, reply); err != nil {
		debugf("worker: failed to call coordinator at %s: %v", address, err)
		return false
	}

	return true
}

func getCoordinatorAddress() string {
	if addr := os.Getenv("MR_COORDINATOR_ADDRESS"); addr != "" {
		return addr
	}
	return "0.0.0.0:7777"
}