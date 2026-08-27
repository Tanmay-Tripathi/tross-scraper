// Package network provides the shared outbound HTTP client. Every call the
// service makes to an external API should go through NetworkOps so that
// timeouts, trace-header propagation and response handling stay consistent.
package network

import "net/http"

// ContentType enumerates the request bodies NetworkOps knows how to encode.
type ContentType string

const (
	ApplicationJSON               ContentType = "application/json"
	TextPlain                     ContentType = "text/plain"
	ApplicationXWwwFormURLEncoded ContentType = "application/x-www-form-urlencoded"
)

// Method is an HTTP verb.
type Method string

const (
	GET    Method = "GET"
	POST   Method = "POST"
	PUT    Method = "PUT"
	PATCH  Method = "PATCH"
	DELETE Method = "DELETE"
)

// Response is the outcome of a request. Data always holds the raw body; when
// ApiPayload.JsonObject was set the body has also been unmarshalled into it.
type Response struct {
	Data           []byte
	HttpStatusCode int
	Headers        http.Header
	Cookies        []*http.Cookie
}

// IsSuccess reports whether the response carried a 2xx status.
func (r *Response) IsSuccess() bool {
	return r != nil && r.HttpStatusCode >= 200 && r.HttpStatusCode < 300
}

// ApiPayload describes a single outbound request.
type ApiPayload struct {
	Method Method
	Url    string

	QueryParams map[string]string
	Headers     http.Header
	Cookies     []*http.Cookie

	// ContentType selects how Body is encoded. Defaults to ApplicationJSON.
	ContentType ContentType
	// Body is encoded according to ContentType. A nil body sends no payload.
	// For ApplicationXWwwFormURLEncoded it must be a map[string]string.
	Body any

	// JsonObject, when non-nil, receives the JSON-decoded response body.
	JsonObject any
}
