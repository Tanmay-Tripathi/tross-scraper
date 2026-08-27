package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Tanmay-Tripathi/tross-scraper/pkg/global"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/utils"
	"github.com/Tanmay-Tripathi/tross-scraper/pkg/validation"
)

// replayTTL is how long a completed response stays replayable.
const replayTTL = 6 * time.Hour

// MiddlewareIdempotencyMethods guards handlers that must not run twice for the
// same client-supplied key.
type MiddlewareIdempotencyMethods interface {
	// WithIdempotency replays the stored response for a repeated
	// x-idempotency-key instead of invoking the handler again. keyPrefix
	// namespaces the cache entries so two routes cannot collide on one key.
	WithIdempotency(keyPrefix string) gin.HandlerFunc
}

type MiddlewareIdempotency struct {
	Access *MiddlewareAccess
}

func NewMiddlewareIdempotency(access *MiddlewareAccess) MiddlewareIdempotencyMethods {
	return &MiddlewareIdempotency{Access: access}
}

// cachedResponse is what gets stored for a completed request.
type cachedResponse struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// responseRecorder tees the handler's response into a buffer so it can be
// cached once the status code is known.
type responseRecorder struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (m *MiddlewareIdempotency) WithIdempotency(keyPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		logger := m.Access.Logger.With(ctx)

		idempotencyKey, appErr := validation.RequireNonEmpty(c.GetHeader(global.XIdempotencyKey.String()))
		if appErr != nil {
			logger.Warnf("rejected request to %s: missing %s header", c.FullPath(), global.XIdempotencyKey)
			utils.SendApiResponseV2[any](c, nil, nil, appErr)
			return
		}

		cacheKey := fmt.Sprintf("idmp:%s:%s", keyPrefix, idempotencyKey)

		// A cache outage must not take the endpoint down: fall through to the
		// handler and accept that this one call is not deduplicated.
		var stored cachedResponse
		found, err := m.Access.Cache.GetJSON(ctx, cacheKey, &stored)
		if err != nil {
			logger.Warnf("idempotency cache read failed for %s, proceeding without replay: %v", cacheKey, err)
		}

		if found {
			logger.Infof("replaying cached response for idempotency key %s", idempotencyKey)
			c.Data(stored.Status, gin.MIMEJSON, stored.Body)
			c.Abort()
			return
		}

		recorder := &responseRecorder{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = recorder

		c.Next()

		// Only successful responses are replayable; a failure should be
		// retryable with the same key.
		status := c.Writer.Status()
		if status < 200 || status >= 300 {
			return
		}

		entry := cachedResponse{Status: status, Body: recorder.body.Bytes()}
		if err := m.Access.Cache.SetJSON(ctx, cacheKey, entry, replayTTL); err != nil {
			logger.Warnf("failed to cache response for idempotency key %s: %v", idempotencyKey, err)
		}
	}
}
