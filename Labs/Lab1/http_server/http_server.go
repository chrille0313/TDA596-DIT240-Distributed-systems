package main

import (
	"Lab1/http"
	"flag"
	"net"
	"os"
)

func main() {
	host := flag.String("host", "0.0.0.0", "Host to bind to")
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

	public := "./public"
	err := os.MkdirAll(public, os.ModePerm)
	if err != nil {
		return
	}

	address := net.JoinHostPort(*host, *port)
	server := http.NewServer()

	server.Get(FileGetHandler)
	server.Post(FilePostHandler)

	server.Listen(address, *maxConnections)
}
