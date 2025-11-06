package main

import (
	"flag"
	"http_server/http"
	"net"
	builtInHttp "net/http"
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

	// TODO: Read and return file from server
	server.Get(func(req *builtInHttp.Request, res *http.ResponseBuilder) {
		res.Body("Hello, World!")
	})

	server.Post(func(req *builtInHttp.Request, res *http.ResponseBuilder) {
		// TODO: Create file on server
		res.Status(http.Created)
	})

	server.Listen(address, *maxConnections)
}
