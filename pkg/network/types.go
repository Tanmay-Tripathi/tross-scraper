// Package network provides the shared outbound HTTP caller, so timeouts, trace
// headers, cookies and redacted logging stay consistent in one place.
package network

import "net/http"

// Method is an HTTP verb.
type Method string

const (
	GET  Method = "GET"
	POST Method = "POST"
)

// Response is a completed exchange; Data holds the raw body, error or success.
type Response struct {
	Data           []byte
	HttpStatusCode int
	Headers        http.Header
}

// IsSuccess reports whether the response carried a 2xx status.
func (r *Response) IsSuccess() bool {
	return r != nil && r.HttpStatusCode >= 200 && r.HttpStatusCode < 300
}

// ApiPayload describes a single outbound request.
type ApiPayload struct {
	Method      Method
	Url         string
	QueryParams map[string]string
	Headers     http.Header
	// Body is JSON-encoded when non-nil.
	Body any
}
