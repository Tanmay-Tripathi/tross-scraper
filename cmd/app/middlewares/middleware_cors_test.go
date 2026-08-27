package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/log"
)

func init() { gin.SetMode(gin.TestMode) }

func newCorsRouter(origins ...string) *gin.Engine {
	access := &MiddlewareAccess{
		Cfg:    &config.Config{Cors: config.CorsConfig{AllowedOrigins: origins}},
		Logger: log.NewWithZerolog(zerolog.Nop()),
	}

	router := gin.New()
	router.Use(NewMiddlewareCors(access).Handler())
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	return router
}

func do(router *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/ping", nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestCorsAllowsConfiguredOrigin(t *testing.T) {
	recorder := do(newCorsRouter("https://app.example.com"), http.MethodGet, "https://app.example.com")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the caller's origin echoed back", got)
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestCorsRejectsUnknownOrigin(t *testing.T) {
	recorder := do(newCorsRouter("https://app.example.com"), http.MethodGet, "https://evil.example.com")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent for an unlisted origin", got)
	}
}

func TestCorsRejectsPreflightFromUnknownOrigin(t *testing.T) {
	recorder := do(newCorsRouter("https://app.example.com"), http.MethodOptions, "https://evil.example.com")

	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCorsAnswersPreflight(t *testing.T) {
	recorder := do(newCorsRouter("https://app.example.com"), http.MethodOptions, "https://app.example.com")

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	for _, header := range []string{
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Max-Age",
	} {
		if recorder.Header().Get(header) == "" {
			t.Errorf("%s is missing from the preflight response", header)
		}
	}
}

func TestCorsWildcardEchoesOrigin(t *testing.T) {
	recorder := do(newCorsRouter("*"), http.MethodGet, "https://anything.example.com")

	// "*" plus credentials is rejected by browsers, so the origin is echoed.
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://anything.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the caller's origin echoed back", got)
	}
}

func TestCorsDisabledWhenNoOriginsConfigured(t *testing.T) {
	recorder := do(newCorsRouter(), http.MethodGet, "https://app.example.com")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want no CORS headers when disabled", got)
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want the request to pass through", recorder.Code)
	}
}

// A same-origin or server-to-server call sends no Origin header and must be
// left completely untouched.
func TestCorsIgnoresRequestsWithoutOrigin(t *testing.T) {
	recorder := do(newCorsRouter("https://app.example.com"), http.MethodGet, "")

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent", got)
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
