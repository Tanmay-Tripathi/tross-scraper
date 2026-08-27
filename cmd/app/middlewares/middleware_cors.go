package middlewares

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
)

// preflightMaxAge is how long a browser may cache a preflight result.
const preflightMaxAge = 12 * time.Hour

// allowedHeaders are the request headers a browser may send cross-origin.
var allowedHeaders = strings.Join([]string{
	global.ContentType.String(),
	global.XIdempotencyKey.String(),
	global.XApiKey.String(),
	global.CorrelationID.String(),
	global.RequestID.String(),
	"Authorization",
}, ", ")

// allowedMethods are the verbs a browser may use cross-origin.
var allowedMethods = strings.Join([]string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}, ", ")

// MiddlewareCorsMethods guards cross-origin browser access.
type MiddlewareCorsMethods interface {
	// Handler answers preflights and adds CORS headers; a no-op when no origins
	// are configured.
	Handler() gin.HandlerFunc
}

type MiddlewareCors struct {
	Access *MiddlewareAccess
}

func NewMiddlewareCors(access *MiddlewareAccess) MiddlewareCorsMethods {
	return &MiddlewareCors{Access: access}
}

func (m *MiddlewareCors) Handler() gin.HandlerFunc {
	origins := m.Access.Cfg.Cors.AllowedOrigins

	if len(origins) == 0 {
		m.Access.Logger.Infof("no CORS origins configured, cross-origin browser requests will be rejected")
		return func(c *gin.Context) { c.Next() }
	}

	allowAll := slices.Contains(origins, "*")
	m.Access.Logger.Infof("CORS enabled for origins: %s", strings.Join(origins, ", "))

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// No Origin means it is not a cross-origin browser request.
		if origin == "" {
			c.Next()
			return
		}

		if !allowAll && !slices.Contains(origins, origin) {
			// Respond without the allow headers; the browser blocks it.
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		// Echo the origin rather than "*"; browsers reject "*" with credentials.
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", global.XRequestID.String())
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowedMethods)
			c.Header("Access-Control-Allow-Headers", allowedHeaders)
			c.Header("Access-Control-Max-Age", strconv.Itoa(int(preflightMaxAge.Seconds())))
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
