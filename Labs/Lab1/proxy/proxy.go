package main

import (
	"Lab1/http"
	"bufio"
	"flag"
	"io"
	"net"
	nethttp "net/http"
	"strings"
)

func main() {
	host := flag.String("host", "0.0.0.0", "Host to bind to")
	port := flag.String("port", "80", "Port to listen on")
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

	server.Get(func(request *nethttp.Request, rb *http.ResponseBuilder) error {
		return handleProxyRequest(request, rb)
	})

	server.Listen(address, *maxConnections)
}

func handleProxyRequest(request *nethttp.Request, rb *http.ResponseBuilder) error {
	targetHost := request.URL.Host
	if targetHost == "" {
		targetHost = request.Host
	}
	if targetHost == "" {
		return &http.HTTPError{Status: http.BadRequest}
	}
	if !strings.Contains(targetHost, ":") {
		targetHost += ":80"
	}

	connection, err := net.Dial("tcp", targetHost)
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}
	defer connection.Close()

	request.Header.Del("Proxy-Connection")
	request.RequestURI = ""

	err = request.Write(connection)
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}

	response, err := nethttp.ReadResponse(bufio.NewReader(connection), request)
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return &http.HTTPError{Status: http.InternalServerError}
	}

	rb.Status(http.StatusCode(response.StatusCode)).Bytes(body)
	for headerKey, headerValues := range response.Header {
		rb.Header(headerKey, strings.Join(headerValues, ", "))
	}

	return nil
}
