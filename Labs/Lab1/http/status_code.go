package http

type StatusCode int

const (
	Ok                  StatusCode = 200
	Created             StatusCode = 201
	BadRequest          StatusCode = 400
	NotFound            StatusCode = 404
	InternalServerError StatusCode = 500
	NotImplemented      StatusCode = 501
)

func (statusCode StatusCode) Phrase() string {
	switch statusCode {
	case Ok:
		return "OK"
	case Created:
		return "Created"
	case BadRequest:
		return "Bad Request"
	case NotFound:
		return "Not Found"
	case InternalServerError:
		return "Internal Server Error"
	case NotImplemented:
		return "Not Implemented"
	default:
		return "Unknown"
	}
}