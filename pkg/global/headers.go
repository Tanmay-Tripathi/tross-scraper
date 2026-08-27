package global

// Header is a well-known HTTP header name used across the service.
type Header string

const (
	CorrelationID   Header = "correlation_id"
	RequestID       Header = "request_id"
	XRequestID      Header = "x-request-id"
	XIdempotencyKey Header = "x-idempotency-key"
	XApiKey         Header = "x-api-key"
	XRealIP         Header = "x-real-ip"
	UserAgent       Header = "user-agent"
	ContentType     Header = "content-type"
)

func (h Header) String() string {
	return string(h)
}
