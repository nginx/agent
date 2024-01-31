package http

type RequestError struct {
	StatusCode int
	Message    string
}

func (r *RequestError) Error() string {
	return r.Message
}
