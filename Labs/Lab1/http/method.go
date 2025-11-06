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

func (method HTTPMethod) IsValid() bool {
	switch method {
	case Get, Head, Options, Trace, Put, Delete, Post, Patch, Connect:
		return true
	default:
		return false
	}
}