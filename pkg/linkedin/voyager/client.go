package voyager

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/network"
)

const (
	// BaseURL is the root of LinkedIn's internal API.
	BaseURL = "https://www.linkedin.com/voyager/api"
	// cookieURL is the origin the session cookies are scoped to.
	cookieURL = "https://www.linkedin.com"

	// acceptNormalized asks for the flat data/included shape the resolver reads.
	acceptNormalized = "application/vnd.linkedin.normalized+json+2.1"
	// restliVersion selects LinkedIn's modern response protocol.
	restliVersion = "2.0.0"
)

// Credentials are the two cookies that constitute a logged-in session.
type Credentials struct {
	// LiAt is the li_at cookie. This is the login.
	LiAt string
	// JSessionID is the JSESSIONID value with no surrounding quotes.
	JSessionID string
	// UserAgent must look like a real browser.
	UserAgent string
}

// Valid reports whether both cookies are present.
func (c Credentials) Valid() bool {
	return strings.TrimSpace(c.LiAt) != "" && strings.TrimSpace(c.JSessionID) != ""
}

// Client calls Voyager endpoints with one long-lived session.
type Client struct {
	network network.NetworkOpsMethods
	creds   Credentials
	logger  log.Logger
}

// NewClient seeds the cookie jar and returns a ready client. One instance is
// reused process-wide, so it behaves like one browser tab rather than a new login per call.
func NewClient(netOps network.NetworkOpsMethods, creds Credentials, logger log.Logger) (*Client, error) {
	if !creds.Valid() {
		return nil, fmt.Errorf("linkedin credentials are incomplete: both li_at and jsessionid are required")
	}

	// JSESSIONID is sent with quotes, csrf-token without. Getting this pair wrong
	// is the usual cause of "CSRF check failed". Quoted is what puts the quotes on
	// the wire: a literal '"' inside Value is an invalid cookie byte, so net/http
	// strips it, logs "dropping invalid bytes", and sends the cookie unquoted —
	// which is not what a browser sends.
	cookies := []*http.Cookie{
		{Name: "li_at", Value: creds.LiAt, Domain: ".linkedin.com", Path: "/"},
		{Name: "JSESSIONID", Value: creds.JSessionID, Quoted: true, Domain: ".linkedin.com", Path: "/"},
	}
	if err := netOps.SetCookies(cookieURL, cookies); err != nil {
		return nil, fmt.Errorf("seed linkedin cookies: %w", err)
	}

	return &Client{network: netOps, creds: creds, logger: logger}, nil
}

// headers builds the header set every Voyager call needs.
func (c *Client) headers() http.Header {
	h := make(http.Header)
	h.Set("accept", acceptNormalized)
	h.Set("csrf-token", c.creds.JSessionID)
	h.Set("x-restli-protocol-version", restliVersion)
	h.Set("user-agent", c.creds.UserAgent)
	h.Set("accept-language", "en-US,en;q=0.9")
	h.Set("x-li-lang", "en_US")
	// The front end sends this; its absence is a cheap tell.
	h.Set("referer", cookieURL+"/feed/")
	return h
}

// Get calls a Voyager path relative to BaseURL. Non-2xx answers come back as a
// *StatusError, so callers map upstream codes without reading HTTP themselves.
func (c *Client) Get(ctx context.Context, path string, query map[string]string) ([]byte, error) {
	resp, err := c.network.SendRequest(ctx, &network.ApiPayload{
		Method:      network.GET,
		Url:         BaseURL + path,
		QueryParams: query,
		Headers:     c.headers(),
	})
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, &StatusError{
			StatusCode: resp.HttpStatusCode,
			Path:       path,
			Body:       snippet(resp.Data),
		}
	}

	return resp.Data, nil
}

// GetGraph calls a Voyager path and returns the response already indexed.
func (c *Client) GetGraph(ctx context.Context, path string, query map[string]string) (*Graph, error) {
	payload, err := c.Get(ctx, path, query)
	if err != nil {
		return nil, err
	}
	return NewGraph(payload)
}

// SessionValid makes one cheap authenticated call to check the cookies. Readiness
// uses it so an expired session shows on a dashboard, not to a caller.
func (c *Client) SessionValid(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := c.Get(ctx, "/me", nil)
	return err
}

// snippet trims an error body; Voyager can answer with a whole HTML page.
func snippet(data []byte) string {
	const limit = 300
	text := strings.TrimSpace(string(data))
	if len(text) > limit {
		return text[:limit] + "…"
	}
	return text
}
