package http

import (
	"net/http"
	"strconv"
)

type ResponseBuilder struct {
	response *Response
}

func NewResponseBuilder(request *http.Request) *ResponseBuilder {
	return &ResponseBuilder{
		response: NewResponse(request),
	}
}

func (builder *ResponseBuilder) Status(statusCode StatusCode) *ResponseBuilder {
	builder.response.StatusCode = statusCode
	return builder
}

func (builder *ResponseBuilder) Ok() *ResponseBuilder {
	return builder.Status(Ok)
}

func (builder *ResponseBuilder) Header(key string, value string) *ResponseBuilder {
	builder.response.Headers[key] = []string{value}
	return builder
}

// TODO: adapt body + content-type to the actual type
func (builder *ResponseBuilder) Body(body string) *ResponseBuilder {
	if body == "" {
		return builder
	}

	builder.response.Body = body
	builder.response.Headers["Content-Length"] = []string{strconv.Itoa(len(body))}
	builder.response.Headers["Content-Type"] = []string{"text/plain"} 
	return builder
}

func (builder *ResponseBuilder) Build() *Response {
	return builder.response
}