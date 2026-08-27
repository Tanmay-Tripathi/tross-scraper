package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

// NetworkOpsMethods is the outbound-HTTP surface clients depend on.
type NetworkOpsMethods interface {
	// SendRequest performs the call described by payload. It returns a
	// Response for any completed exchange — including 4xx/5xx — and an error
	// only when the request could not be completed or decoded.
	SendRequest(ctx context.Context, payload *ApiPayload) (*Response, error)
}

// Options tunes the underlying http.Client.
type Options struct {
	// Timeout bounds a single request. Defaults to 30s.
	Timeout time.Duration
	// EnableCookieJar keeps cookies across calls made by this instance, which
	// is what session-based scraping needs. Off by default.
	EnableCookieJar bool
}

// NetworkOps is a named, logged HTTP caller.
type NetworkOps struct {
	name   string
	logger log.Logger
	client *http.Client
}

// NewNetworkOps builds an HTTP caller. name appears in logs so several
// instances (one per upstream) stay distinguishable.
func NewNetworkOps(name string, logger log.Logger, opts Options) (NetworkOpsMethods, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	if opts.EnableCookieJar {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("create cookie jar for %q: %w", name, err)
		}
		client.Jar = jar
	}

	logger.Infof("created network ops %q with a %s timeout", name, timeout)
	return &NetworkOps{name: name, logger: logger, client: client}, nil
}

func (n *NetworkOps) SendRequest(ctx context.Context, payload *ApiPayload) (*Response, error) {
	logger := n.logger.With(ctx)

	if err := validate(payload); err != nil {
		return nil, err
	}

	body, headers, err := encodeBody(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: encode request body: %w", n.name, err)
	}

	req, err := http.NewRequestWithContext(ctx, string(payload.Method), payload.Url, body)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", n.name, err)
	}

	req.Header = global.PropagateTraceHeaders(ctx, headers)
	for _, cookie := range payload.Cookies {
		req.AddCookie(cookie)
	}
	req.URL.RawQuery = mergeQuery(req.URL.Query(), payload.QueryParams)

	start := time.Now()
	httpResp, err := n.client.Do(req)
	if err != nil {
		logger.Errorf("%s: %s %s failed after %s: %v", n.name, req.Method, redact(req.URL), time.Since(start), err)
		return nil, fmt.Errorf("%s: %w", n.name, err)
	}
	defer func() {
		if closeErr := httpResp.Body.Close(); closeErr != nil {
			logger.Warnf("%s: failed to close response body: %v", n.name, closeErr)
		}
	}()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: read response body: %w", n.name, err)
	}

	response := &Response{
		Data:           data,
		HttpStatusCode: httpResp.StatusCode,
		Headers:        httpResp.Header,
		Cookies:        httpResp.Cookies(),
	}

	logger.Infof("%s: %s %s -> %d in %s", n.name, req.Method, redact(req.URL), response.HttpStatusCode, time.Since(start))

	if payload.JsonObject != nil && len(data) > 0 {
		if err := json.Unmarshal(data, payload.JsonObject); err != nil {
			return response, fmt.Errorf("%s: decode response body: %w", n.name, err)
		}
	}

	return response, nil
}

func validate(payload *ApiPayload) error {
	switch {
	case payload == nil:
		return errors.New("api payload is nil")
	case strings.TrimSpace(payload.Url) == "":
		return errors.New("api payload url is empty")
	case strings.TrimSpace(string(payload.Method)) == "":
		return errors.New("api payload method is empty")
	default:
		return nil
	}
}

// encodeBody serialises payload.Body and returns the headers to send with it.
func encodeBody(payload *ApiPayload) (io.Reader, http.Header, error) {
	headers := cloneHeaders(payload.Headers)

	if payload.Body == nil {
		return nil, headers, nil
	}

	contentType := payload.ContentType
	if contentType == "" {
		contentType = ApplicationJSON
	}

	switch contentType {
	case ApplicationJSON:
		encoded, err := json.Marshal(payload.Body)
		if err != nil {
			return nil, nil, err
		}
		headers.Set(global.ContentType.String(), string(ApplicationJSON))
		return bytes.NewReader(encoded), headers, nil

	case ApplicationXWwwFormURLEncoded:
		fields, ok := payload.Body.(map[string]string)
		if !ok {
			return nil, nil, fmt.Errorf("form-encoded body must be a map[string]string, got %T", payload.Body)
		}
		values := url.Values{}
		for key, value := range fields {
			values.Set(key, value)
		}
		headers.Set(global.ContentType.String(), string(ApplicationXWwwFormURLEncoded))
		return strings.NewReader(values.Encode()), headers, nil

	case TextPlain:
		text, ok := payload.Body.(string)
		if !ok {
			return nil, nil, fmt.Errorf("text/plain body must be a string, got %T", payload.Body)
		}
		headers.Set(global.ContentType.String(), string(TextPlain))
		return strings.NewReader(text), headers, nil

	default:
		return nil, nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func cloneHeaders(headers http.Header) http.Header {
	if headers == nil {
		return make(http.Header)
	}
	return headers.Clone()
}

func mergeQuery(existing url.Values, params map[string]string) string {
	for key, value := range params {
		existing.Set(key, value)
	}
	return existing.Encode()
}

// redact strips the query string from a URL before it reaches the logs, since
// upstream query params routinely carry tokens and identifiers.
func redact(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + u.Host + u.Path
}
