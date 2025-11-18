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
	Body       []byte
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
		Body:       nil,
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

func (response *Response) Bytes() []byte {
	startLine := response.formatStartLine()
	headers := response.formatHeaders()
	s := startLine + headers + "\r\n"
	b := []byte(s)

	if response.hasBody() {
		b = append(b, []byte(response.Body)...)
	}

	return b
}
