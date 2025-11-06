package http

import (
	"fmt"
	"net/http"
	"strings"
)

type Response struct {
	Request    *http.Request
	Protocol   string
	StatusCode StatusCode
	Headers    Headers
	Body       string
}

func NewResponse(request *http.Request) *Response {
	protocol := request.Proto  // Mirror the protocol of the request
	if protocol == "" {
		protocol = "HTTP/1.1"  // Otherwise use HTTP/1.1 by default
	}
	return &Response{
		Request:    request,
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

func (response *Response) String() string {
	s := response.formatStartLine()
	s += response.formatHeaders()

	if response.Body != "" {
		s += "\r\n"
		s += response.Body
	}

	return s
}

func (response *Response) Bytes() []byte {
	return []byte(response.String())
}