package http

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
)

type RequestHandler func(*http.Request, *ResponseBuilder)

type Server struct {
	connections     chan net.Conn
	requestHandlers map[HTTPMethod]RequestHandler
}

func NewServer() *Server {
	return &Server{
		requestHandlers: make(map[HTTPMethod]RequestHandler),
	}
}

func (server *Server) handleRequest(request *http.Request) *Response {
	responseBuilder := NewResponseBuilder(request)
	
	method := HTTPMethod(request.Method)
	if !method.IsValid() {
		return responseBuilder.Status(BadRequest).Build()
	}

	handler, exists := server.requestHandlers[method]
	if !exists {
		return responseBuilder.Status(NotImplemented).Build()
	}

	handler(request, responseBuilder)
	return responseBuilder.Build()
}

func (server *Server) handleConnection(connection net.Conn) {
	defer func() {
		connection.Close()
		<-server.connections
	}()

	reader := bufio.NewReader(connection)
	request, err := http.ReadRequest(reader)
	if err != nil {
		fmt.Println("Error reading request:", err)
		return
	}

	response := server.handleRequest(request)

	log.Println(connection.RemoteAddr(), "-", request.Method, request.URL.Path, request.Proto, response.StatusCode, response.Headers["Content-Length"])

	_, err = connection.Write(response.Bytes())
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}
}

func (server *Server) Listen(address string, maxConnections uint) {
	server.connections = make(chan net.Conn, maxConnections)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()
	defer close(server.connections)

	fmt.Println("Listening on " + address)

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
