package main

import (
	"Lab1/http"
	"errors"
	"flag"
	"net"
	nethttp "net/http"
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

	server.Get(func(r *nethttp.Request, rb *http.ResponseBuilder) error {
		contentType, ok := contentTypeForPath(r.URL.Path)
		if !ok {
			return &http.HTTPError{Status: http.BadRequest}
		}

		data, err := readFile(r.URL.Path)
		if err != nil {
			return &http.HTTPError{Status: statusFromFileError(err)}
		}

		rb.Bytes(data).Header("Content-Type", contentType)
		return nil
	})

	server.Post(func(r *nethttp.Request, rb *http.ResponseBuilder) error {
		defer r.Body.Close()

		err := writeFile(r.URL.Path, r.Body);
		if err != nil {
			return &http.HTTPError{Status: statusFromFileError(err)}
		}

		rb.Status(http.Created)
		return nil
	})

	server.Listen(address, *maxConnections)
}

func statusFromFileError(err error) http.StatusCode {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return http.NotFound
	case errors.Is(err, os.ErrPermission),
		errors.Is(err, os.ErrInvalid),
		errors.Is(err, errUnsupportedContentType):
		return http.BadRequest
	default:
		return http.InternalServerError
	}
}
