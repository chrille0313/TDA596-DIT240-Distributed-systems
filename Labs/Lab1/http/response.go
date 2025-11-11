package http

import (
	"fmt"
	"net/http"
	"strings"
)

type Response struct {
	Protocol   string
	StatusCode StatusCode
	Headers    Headers
	Body       string
}

func NewResponse(request *http.Request) *Response {
	protocol := "HTTP/1.1"
	if request != nil && request.Proto != "" {
		protocol = request.Proto
	}

	return &Response{
		Protocol:   protocol,
		StatusCode: Ok,
		Headers:    make(Headers),
		Body:       "",
	}
}

func (response *Response) formatStartLine() string {
	return fmt.Sprintf("%s %d %s\r\n", response.Protocol, response.StatusCode, response.StatusCode.Phrase())
}

func (response *Response) formatHeaders() string {
	headers := ""
	for key, values := range response.Headers {
		header := key + ": " + strings.Join(values, ", ") + "\r\n"
		headers += header
	}

	return headers
}

func (response *Response) hasBody() bool {
	_, ok := response.Headers["Content-Length"]
	return ok
}

func (response *Response) String() string {
	s := response.formatStartLine()
	s += response.formatHeaders()
	s += "\r\n"

	if response.hasBody() {	
		s += response.Body
	}

	return s
}

func (response *Response) Bytes() []byte {
	return []byte(response.String())
}