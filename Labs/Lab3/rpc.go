package main

import (
	"log"
	"net/http"
	"net/rpc"
)

// Start a coordinator server that listens for RPCs over HTTP.
func StartRPCServer(address string, obj any) {
	if err := rpc.Register(obj); err != nil {
		log.Fatalf("cannot register RPC server: %v", err)
	}

	rpc.HandleHTTP()
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
}

func CallNodeRPC(address NodeAddress, method string, args, reply any) error {
	return CallRPC(string(address), method, args, reply)
}

func CallRPC(address string, method string, args, reply any) error {
	client, err := rpc.DialHTTP("tcp", string(address))
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Call(method, args, reply); err != nil {
		return err
	}

	return nil
}
