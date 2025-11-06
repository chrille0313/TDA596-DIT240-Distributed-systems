package main

import (
	"flag"
	"fmt"
	"http_server/http"
	"net"
	f "net/http"
)

func main() {
	host := flag.String("host", "127.0.0.1", "Host to bind to")
	port := flag.String("port", "8080", "Port to listen on")
	portAlias := flag.String("p", "", "Alias for -port")
	maxConnections := flag.Uint("maxConnections", 10, "Maximum concurrent connections")
	
	flag.Parse()

	if *portAlias != "" {
		*port = *portAlias
	}

	if len(flag.Args()) > 0 {
		*port = flag.Arg(0)
	}

	address := net.JoinHostPort(*host, *port)
	server := http.NewServer()

	// FIXME: implement correctly
	server.Get(func(req *f.Request, res *f.Response) {
		fmt.Println("INSIDE GET")
	})

	// server.

	// END FIXME

	server.Listen(address, *maxConnections)
}
