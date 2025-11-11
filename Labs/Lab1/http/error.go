package http

import "fmt"

type HTTPError struct {
	Status StatusCode
}

func (err *HTTPError) Error() string {
	return fmt.Sprintf("%d %s", err.Status, err.Status.Phrase())
}