package http

type HTTPMethod string

const (
	Get     = "GET"
	Head    = "HEAD"
	Options = "OPTIONS"
	Trace   = "TRACE"
	Put     = "PUT"
	Delete  = "DELETE"
	Post    = "POST"
	Patch   = "PATCH"
	Connect = "CONNECT"
)