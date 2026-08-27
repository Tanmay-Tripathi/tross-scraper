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
	// SendRequest returns a Response for any completed exchange, 4xx and 5xx
	// included, and an error only when the request could not be made.
	SendRequest(ctx context.Context, payload *ApiPayload) (*Response, error)
	// SetCookies seeds the jar; later requests to that host replay them.
	SetCookies(rawURL string, cookies []*http.Cookie) error
}

// Options tunes the underlying http.Client.
type Options struct {
	// Timeout bounds a single request. Defaults to 30s.
	Timeout time.Duration
	// EnableCookieJar keeps cookies across calls, so one instance behaves like
	// one browser tab rather than a fresh login each time.
	EnableCookieJar bool
	// MaxResponseBytes caps the body we read so a runaway upstream cannot exhaust memory.
	MaxResponseBytes int64
	// NoRedirects returns the 3xx itself instead of following it. For a JSON API a
	// redirect is an answer — usually "you are not logged in" — and following it
	// turns one failed call into ten requests against an upstream that just
	// rejected us.
	NoRedirects bool
}

const defaultMaxResponseBytes = 16 << 20

// NetworkOps is a named, logged HTTP caller. One instance per upstream.
type NetworkOps struct {
	name     string
	logger   log.Logger
	client   *http.Client
	maxBytes int64
}

// NewNetworkOps builds an HTTP caller; name appears in logs.
func NewNetworkOps(name string, logger log.Logger, opts Options) (*NetworkOps, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	maxBytes := opts.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxResponseBytes
	}

	client := &http.Client{Timeout: timeout}
	if opts.NoRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	if opts.EnableCookieJar {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("create cookie jar for %q: %w", name, err)
		}
		client.Jar = jar
	}

	logger.Infof("created network ops %q (timeout %s, cookie jar %t)", name, timeout, opts.EnableCookieJar)
	return &NetworkOps{name: name, logger: logger, client: client, maxBytes: maxBytes}, nil
}

func (n *NetworkOps) SetCookies(rawURL string, cookies []*http.Cookie) error {
	if n.client.Jar == nil {
		return fmt.Errorf("%s: cookie jar is not enabled", n.name)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%s: parse cookie url: %w", n.name, err)
	}

	n.client.Jar.SetCookies(parsed, cookies)
	return nil
}

func (n *NetworkOps) SendRequest(ctx context.Context, payload *ApiPayload) (*Response, error) {
	logger := n.logger.With(ctx)

	if err := validate(payload); err != nil {
		return nil, fmt.Errorf("%s: %w", n.name, err)
	}

	body, err := encodeBody(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: encode request body: %w", n.name, err)
	}

	req, err := http.NewRequestWithContext(ctx, string(payload.Method), payload.Url, body)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", n.name, err)
	}

	req.Header = global.PropagateTraceHeaders(ctx, cloneHeaders(payload.Headers))
	if payload.Body != nil {
		req.Header.Set(global.ContentType.String(), "application/json")
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

	data, err := io.ReadAll(io.LimitReader(httpResp.Body, n.maxBytes))
	if err != nil {
		return nil, fmt.Errorf("%s: read response body: %w", n.name, err)
	}

	logger.Infof("%s: %s %s -> %d (%d bytes) in %s",
		n.name, req.Method, redact(req.URL), httpResp.StatusCode, len(data), time.Since(start))

	return &Response{
		Data:           data,
		HttpStatusCode: httpResp.StatusCode,
		Headers:        httpResp.Header,
	}, nil
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

func encodeBody(payload *ApiPayload) (io.Reader, error) {
	if payload.Body == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(payload.Body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(encoded), nil
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

// redact strips the query string before logging; those params carry tokens.
func redact(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.Scheme + "://" + u.Host + u.Path
}
