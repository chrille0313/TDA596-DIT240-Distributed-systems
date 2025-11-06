package http

import "fmt"

type RequestMethod int

const (
	Get RequestMethod = iota
	Post
)

func (RequestMethod) FromString(method string) (RequestMethod, error) {
	switch method {
	case "GET":
		return Get, nil
	case "POST":
		return Post, nil
	default:
		return 0, fmt.Errorf("unsupported method: %s", method)
	}
}