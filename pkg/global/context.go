package global

import (
	"context"
	"net/http"
	"strings"
)

// ContextKey is a dedicated type for context.WithValue keys so that values
// stored by this package can never collide with keys from other packages.
type ContextKey string

const (
	ContextRequestIDKey     ContextKey = "request_id"
	ContextCorrelationIDKey ContextKey = "correlation_id"
)

// WithRequestID returns a context carrying the given request ID. An empty
// requestID leaves the context untouched.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return ctx
	}
	return withValue(ctx, ContextRequestIDKey, requestID)
}

// RequestIDFromContext returns the request ID stored in ctx, or "" if absent.
func RequestIDFromContext(ctx context.Context) string {
	return stringValue(ctx, ContextRequestIDKey)
}

// WithCorrelationID returns a context carrying the given correlation ID. An
// empty correlationID leaves the context untouched.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return ctx
	}
	return withValue(ctx, ContextCorrelationIDKey, correlationID)
}

// CorrelationIDFromContext returns the correlation ID stored in ctx, or "" if absent.
func CorrelationIDFromContext(ctx context.Context) string {
	return stringValue(ctx, ContextCorrelationIDKey)
}

// PropagateTraceHeaders copies the request and correlation IDs from ctx onto
// headers so that downstream calls stay traceable. Existing values win.
func PropagateTraceHeaders(ctx context.Context, headers http.Header) http.Header {
	if headers == nil {
		headers = make(http.Header)
	}

	if strings.TrimSpace(headers.Get(RequestID.String())) == "" {
		if requestID := RequestIDFromContext(ctx); requestID != "" {
			headers.Set(RequestID.String(), requestID)
		}
	}

	if strings.TrimSpace(headers.Get(CorrelationID.String())) == "" {
		if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
			headers.Set(CorrelationID.String(), correlationID)
		}
	}

	return headers
}

func withValue(ctx context.Context, key ContextKey, value any) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, key, value)
}

func stringValue(ctx context.Context, key ContextKey) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(key).(string)
	return strings.TrimSpace(value)
}
