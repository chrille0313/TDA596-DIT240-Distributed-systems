package main

import (
	"log"
	"net/http"
	"net/rpc"
)

func ListenRPC(address string, obj any) {
	if err := rpc.Register(obj); err != nil {
		log.Fatalf("cannot register RPC server: %v", err)
	}

	rpc.HandleHTTP()
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("cannot start server: %v", err)
	}
}

func IsNodeAliveRPC(address NodeAddress) bool {
	_, err := CallNodeRPC[IsAliveArgs, IsAliveReply](address, "Node.IsAlive", &IsAliveArgs{})
	return err == nil
}

func CallNodeRPC[Targs any, TReply any](address NodeAddress, method string, args *Targs) (*TReply, error) {
	return CallRPC[Targs, TReply](string(address), method, args)
}

func CallRPC[Targs any, TReply any](address string, method string, args *Targs) (*TReply, error) {
	reply := new(TReply)

	client, err := rpc.DialHTTP("tcp", string(address))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Call(method, args, reply); err != nil {
		return nil, err
	}

	return reply, nil
}
