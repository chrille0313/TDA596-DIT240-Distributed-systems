package main

import (
	"Lab1/http"
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	builtInHttp "net/http"
	"strings"
)

func main() {
	host := flag.String("host", "0.0.0.0", "Host to bind to")
	port := flag.String("port", "8080", "Port to listen on")
	origin := flag.String("origin", "localhost:8080", "The server to proxy incoming requests to")
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

	server.Get(func(request *builtInHttp.Request, rb *http.ResponseBuilder) error {
		connection, err := net.Dial("tcp", *origin)
		if err != nil {
			return err
		}
		defer connection.Close()

		req := request.Method + " " + request.URL.Path + " " + request.Proto + "\r\nHost: " + address + "\r\n\r\n"
		fmt.Println(req)

		connection.Write([]byte(req))

		response, err := builtInHttp.ReadResponse(bufio.NewReader(connection), nil)
		if err != nil {
			return err
		}

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}

		rb.Status(http.StatusCode(response.StatusCode)).Bytes(body)
		for headerKey, headerValue := range response.Header {
			rb.Header(headerKey, strings.Join(headerValue, ","))
		}

		return nil
	})

	server.Listen(address, *maxConnections)
}
