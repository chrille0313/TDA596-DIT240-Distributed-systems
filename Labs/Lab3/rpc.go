package chord

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

// Start a coordinator server that listens for RPCs over HTTP.
func StartRPCServer(obj interface{}) string {
	if err := rpc.Register(obj); err != nil {
		log.Fatalf("cannot register RPC server: %v", err)
	}

	rpc.HandleHTTP()
	address := getLocalAddress()

	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Fatalf("cannot start server: %v", err)
		}
	}()

	return address
}

func CallRPC(address string, method string, args interface{}, reply interface{}) error {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Call(method, args, reply); err != nil {
		return err
	}

	return nil
}

func getLocalAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}
