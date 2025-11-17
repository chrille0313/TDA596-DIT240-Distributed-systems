package http

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
)

type RequestHandler func(*http.Request, *ResponseBuilder) error

type Server struct {
	connections     chan net.Conn
	requestHandlers map[HTTPMethod]RequestHandler
}

func NewServer() *Server {
	return &Server{
		requestHandlers: make(map[HTTPMethod]RequestHandler),
	}
}

func (server *Server) handleError(err error) *Response {
	status := InternalServerError

	httpErr, ok := err.(*HTTPError)
	if ok {
		status = httpErr.Status
	}

	response := &Response{
		Protocol:   "HTTP/1.1",
		StatusCode: status,
		Headers:    make(Headers),
	}

	response.Headers["Connection"] = []string{"close"}

	return response
}

func (server *Server) handleRequest(request *http.Request) (*Response, error) {
	responseBuilder := NewResponseBuilder(request)

	method := HTTPMethod(request.Method)
	if !method.IsValid() {
		return nil, &HTTPError{Status: BadRequest}
	}

	handler, exists := server.requestHandlers[method]
	if !exists {
		return nil, &HTTPError{Status: NotImplemented}
	}

	err := handler(request, responseBuilder)

	if err != nil {
		return nil, err
	}

	return responseBuilder.Build(), nil
}

func (server *Server) handleConnection(connection net.Conn) {
	defer func() {
		connection.Close()
		<-server.connections
	}()

	var response *Response
	reader := bufio.NewReader(connection)
	request, err := http.ReadRequest(reader)

	if err != nil {
		err = &HTTPError{Status: BadRequest}
	} else {
		response, err = server.handleRequest(request)
	}

	if err != nil {
		response = server.handleError(err)
	}

	server.writeResponse(connection, response)
}

func (server *Server) Listen(address string, maxConnections uint) {
	server.connections = make(chan net.Conn, maxConnections)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()
	defer close(server.connections)

	log.Println("Listening on", address)

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		server.connections <- connection
		go server.handleConnection(connection)
	}
}

func (server *Server) registerHandler(method HTTPMethod, handler RequestHandler) {
	server.requestHandlers[method] = handler
}

func (server *Server) Get(handler RequestHandler) {
	server.registerHandler(Get, handler)
}

func (server *Server) Post(handler RequestHandler) {
	server.registerHandler(Post, handler)
}

func (server *Server) writeResponse(connection net.Conn, response *Response) error {
	if response == nil {
		return fmt.Errorf("response cannot be nil")
	}

	log.Println(connection.RemoteAddr(), "-", response.StatusCode)

	_, err := connection.Write(response.Bytes())
	return err
}
