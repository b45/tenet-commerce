package redis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

const (
	// DefaultIdempotencyTTL is 24 hours
	DefaultIdempotencyTTL = 24 * time.Hour
	// InProgressLockTTL temporary hold while transaction is being executed
	InProgressLockTTL = 30 * time.Second
)

type cachedResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}

// bodyLogWriter captures the response body for caching
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// IdempotencyMiddleware ensures that mutating operations (POST, PUT, PATCH)
// with an Idempotency-Key header are executed exactly once per tenant.
func IdempotencyMiddleware(redisClient *Client, defaultTTL time.Duration) gin.HandlerFunc {
	if defaultTTL <= 0 {
		defaultTTL = DefaultIdempotencyTTL
	}

	return func(c *gin.Context) {
		// Only check on mutation methods
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if idempotencyKey == "" {
			// Idempotency-Key is strictly required for POST checkout
			response.AbortBadRequest(c, "MISSING_IDEMPOTENCY_KEY",
				"Idempotency-Key header is required for transaction operations")
			return
		}

		tenantSlug := c.GetString("tenant_slug")
		if tenantSlug == "" {
			tenantSlug = "default"
		}

		redisKey := fmt.Sprintf("idempotency:%s:%s", tenantSlug, idempotencyKey)
		ctx := c.Request.Context()
		reqLogger := logger.FromContext(ctx)

		// 1. Try to set the key in Redis (SETNX)
		// If key exists, it means this request was already seen
		isNew, err := redisClient.RDB.SetNX(ctx, redisKey, "PROCESSING", InProgressLockTTL).Result()
		if err != nil {
			reqLogger.Error("Redis idempotency SetNX failed", "key", redisKey, "error", err)
			// On Redis failure, fail open or abort depending on safety policy; for financial POS, abort safely
			response.AbortInternalServerError(c, "IDEMPOTENCY_STORE_ERROR", "Failed to verify transaction idempotency")
			return
		}

		if !isNew {
			// Key already exists! Fetch existing state
			val, err := redisClient.RDB.Get(ctx, redisKey).Result()
			if err != nil {
				reqLogger.Warn("Failed to read existing idempotency key", "key", redisKey, "error", err)
				response.AbortConflict(c, "TRANSACTION_IN_PROGRESS", "Transaction is currently being processed. Please retry shortly.")
				return
			}

			if val == "PROCESSING" {
				// Concurrent execution or currently ongoing
				c.Header("X-Cache-Lookup", "PROCESSING")
				response.AbortConflict(c, "TRANSACTION_IN_PROGRESS", "A transaction with this Idempotency-Key is currently being processed")
				return
			}

			// Cached completed response
			var cached cachedResponse
			if err := json.Unmarshal([]byte(val), &cached); err == nil {
				reqLogger.Info("Idempotent response served from cache", "idempotency_key", idempotencyKey, "status", cached.Status)
				c.Header("X-Cache-Lookup", "HIT")
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.String(cached.Status, cached.Body)
				c.Abort()
				return
			}

			response.AbortConflict(c, "DUPLICATE_IDEMPOTENCY_KEY", "This Idempotency-Key has already been used")
			return
		}

		// 2. New transaction - capture downstream response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		statusCode := c.Writer.Status()

		// 3. Post-execution: Cache on success (2xx), delete key on error (4xx/5xx) so client can retry
		if statusCode >= 200 && statusCode < 300 {
			cacheData, _ := json.Marshal(cachedResponse{
				Status: statusCode,
				Body:   blw.body.String(),
			})
			_ = redisClient.RDB.Set(ctx, redisKey, string(cacheData), defaultTTL).Err()
			reqLogger.Info("Idempotency key saved", "key", redisKey, "ttl", defaultTTL)
		} else {
			// On client error or server failure, remove the lock so the client can fix inputs and retry
			_ = redisClient.RDB.Del(ctx, redisKey).Err()
		}
	}
}
