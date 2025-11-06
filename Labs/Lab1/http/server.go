package http

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
)

type RequestHandler func(*http.Request, *http.Response)

type Server struct {
	connections     chan net.Conn
	requestHandlers map[RequestMethod]RequestHandler
}

func NewServer() *Server {
	return &Server{
		requestHandlers: make(map[RequestMethod]RequestHandler),
	}
}

func (server *Server) handleRequest(request *http.Request) {
	method, err := RequestMethod(0).FromString(request.Method)
	if err != nil {
		fmt.Println("Unsupported method:", request.Method)
		return
	}

	handler, exists := server.requestHandlers[method]
	if !exists {
		fmt.Println("No handler registered for method:", request.Method)
		return
	}

	handler(request, nil)  // FIXME: don't pass nil
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

	server.handleRequest(request)
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

func (server *Server) registerHandler(method RequestMethod, handler RequestHandler) {
	server.requestHandlers[method] = handler
}

func (server *Server) Get(handler RequestHandler) {
	server.registerHandler(Get, handler)
}

func (server *Server) Post(handler RequestHandler) {
	server.registerHandler(Post, handler)
}