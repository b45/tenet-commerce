package idempotency

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

type cachedResponse struct {
	Status int    `json:"status"`
	Hash   string `json:"hash"`
	Body   string `json:"body"`
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// DurableIdempotencyMiddleware enforces two-tier durable idempotency:
//  1. Fast-path in-flight lease via Redis SETNX.
//  2. Durable source-of-truth table `idempotency_requests` in the PostgreSQL tenant schema.
//
// Identical re-submissions return the stored response with `Idempotent-Replayed: true`.
// Reusing an idempotency key with a different request payload is hard-rejected with 409 Conflict.
func DurableIdempotencyMiddleware(redisClient *pkgRedis.Client, defaultTTL time.Duration) gin.HandlerFunc {
	if defaultTTL <= 0 {
		defaultTTL = DefaultTTL
	}

	return func(c *gin.Context) {
		// Only check on state-mutating methods
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}

		idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
		if idempotencyKey == "" {
			response.AbortBadRequest(c, "MISSING_IDEMPOTENCY_KEY",
				"Idempotency-Key header is required for transaction operations")
			return
		}

		ctx := c.Request.Context()
		reqLogger := logger.FromContext(ctx)

		// 1. Capture and buffer request body for hashing
		var rawBody []byte
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				reqLogger.Error("Failed to read request body for idempotency hashing", "error", err)
				response.AbortBadRequest(c, "INVALID_REQUEST_BODY", "Failed to read request payload")
				return
			}
			rawBody = bodyBytes
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		targetRoute := c.FullPath()
		if targetRoute == "" {
			targetRoute = c.Request.URL.Path
		}

		// 2. Calculate canonical request hash
		requestHash, err := ComputeFingerprint(c.Request.Method, targetRoute, rawBody)
		if err != nil {
			reqLogger.Error("Failed to compute request fingerprint", "error", err)
			response.AbortInternalServerError(c, "FINGERPRINT_ERROR", "Failed to process request fingerprint")
			return
		}

		tenantSlug := c.GetString("tenant_slug")
		if tenantSlug == "" {
			tenantSlug = "default"
		}

		leaseKey := fmt.Sprintf("idempotency:lease:%s:%s:%s", tenantSlug, targetRoute, idempotencyKey)
		cacheKey := fmt.Sprintf("idempotency:cache:%s:%s:%s", tenantSlug, targetRoute, idempotencyKey)

		// 3. Fast-path check: Check Redis cache for already completed responses
		if redisClient != nil {
			if cachedVal, err := redisClient.RDB.Get(ctx, cacheKey).Result(); err == nil {
				var cached cachedResponse
				if jsonErr := json.Unmarshal([]byte(cachedVal), &cached); jsonErr == nil {
					if cached.Hash != requestHash {
						reqLogger.Warn("Idempotency key reused with different payload (from Redis cache)",
							"idempotency_key", idempotencyKey,
							"expected_hash", cached.Hash,
							"actual_hash", requestHash,
						)
						response.AbortConflict(c, "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_PAYLOAD",
							"Idempotency-Key is already in use for a different request payload")
						return
					}

					reqLogger.Info("Serving idempotent response from Redis cache",
						"idempotency_key", idempotencyKey,
						"status", cached.Status,
					)
					c.Header("Idempotent-Replayed", "true")
					c.Header("X-Cache-Lookup", "HIT")
					c.Header("Content-Type", "application/json; charset=utf-8")
					c.String(cached.Status, cached.Body)
					c.Abort()
					return
				}
			}
		}

		// 4. Durable check: Query PostgreSQL `idempotency_requests` in the active tenant schema
		conn, exists := tenant.GetConn(c)
		if !exists {
			reqLogger.Error("Tenant database connection not found for idempotency coordination")
			response.AbortInternalServerError(c, "DATABASE_UNAVAILABLE", "Database connection context not found")
			return
		}

		var (
			existingID         string
			existingHash       string
			existingStatus     string
			existingStatusCode *int
			existingBody       []byte
			existingExpiresAt  time.Time
		)

		querySelect := `
			SELECT id, request_hash, status, response_status_code, response_body, expires_at
			FROM idempotency_requests
			WHERE idempotency_key = $1 AND target_route = $2
		`
		err = conn.QueryRow(ctx, querySelect, idempotencyKey, targetRoute).Scan(
			&existingID,
			&existingHash,
			&existingStatus,
			&existingStatusCode,
			&existingBody,
			&existingExpiresAt,
		)

		if err == nil {
			// Record already exists in PostgreSQL
			if existingHash != requestHash {
				reqLogger.Warn("Idempotency key reused with different payload (from PostgreSQL)",
					"idempotency_key", idempotencyKey,
					"expected_hash", existingHash,
					"actual_hash", requestHash,
				)
				response.AbortConflict(c, "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_PAYLOAD",
					"Idempotency-Key is already in use for a different request payload")
				return
			}

			if existingStatus == StatusCompleted {
				// Replay cached response
				statusCode := http.StatusOK
				if existingStatusCode != nil {
					statusCode = *existingStatusCode
				}

				// Populate Redis cache for future queries
				if redisClient != nil {
					cacheData, _ := json.Marshal(cachedResponse{
						Status: statusCode,
						Hash:   existingHash,
						Body:   string(existingBody),
					})
					_ = redisClient.RDB.Set(ctx, cacheKey, string(cacheData), defaultTTL).Err()
				}

				reqLogger.Info("Serving idempotent response from PostgreSQL durable record",
					"idempotency_key", idempotencyKey,
					"status", statusCode,
				)
				c.Header("Idempotent-Replayed", "true")
				c.Header("X-Cache-Lookup", "HIT")
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.Data(statusCode, "application/json; charset=utf-8", existingBody)
				c.Abort()
				return
			}

			if existingStatus == StatusProcessing {
				if time.Now().Before(existingExpiresAt) {
					reqLogger.Warn("Concurrent mutation with in-flight idempotency key",
						"idempotency_key", idempotencyKey,
					)
					response.AbortConflict(c, "CONCURRENT_MUTATION_IN_PROGRESS",
						"A transaction with this Idempotency-Key is currently being processed")
					return
				}

				// Lock expired; renew lease
				_, _ = conn.Exec(ctx, `
					UPDATE idempotency_requests
					SET locked_at = NOW(), expires_at = NOW() + $1, updated_at = NOW()
					WHERE id = $2
				`, InProgressLockTTL, existingID)
			}
		} else if err != pgx.ErrNoRows {
			reqLogger.Error("Failed to query durable idempotency record", "error", err)
			response.AbortInternalServerError(c, "IDEMPOTENCY_STORE_ERROR", "Failed to check transaction idempotency")
			return
		} else {
			// Record does not exist: insert as PROCESSING
			queryInsert := `
				INSERT INTO idempotency_requests (
					id, idempotency_key, target_route, request_hash, status, locked_at, expires_at
				) VALUES (
					gen_random_uuid(), $1, $2, $3, 'PROCESSING', NOW(), NOW() + $4
				)
				ON CONFLICT (idempotency_key, target_route) DO NOTHING
			`
			tag, insErr := conn.Exec(ctx, queryInsert, idempotencyKey, targetRoute, requestHash, InProgressLockTTL)
			if insErr != nil {
				reqLogger.Error("Failed to insert processing idempotency record", "error", insErr)
				response.AbortInternalServerError(c, "IDEMPOTENCY_STORE_ERROR", "Failed to reserve idempotency key")
				return
			}

			if tag.RowsAffected() == 0 {
				// Concurrent insert won the race
				response.AbortConflict(c, "CONCURRENT_MUTATION_IN_PROGRESS",
					"A transaction with this Idempotency-Key is currently being processed")
				return
			}
		}

		// 5. Acquire fast-path Redis in-flight lease
		if redisClient != nil {
			_, _ = redisClient.RDB.SetNX(ctx, leaseKey, "PROCESSING", InProgressLockTTL).Result()
		}

		// 6. Wrap ResponseWriter to capture downstream execution results
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		statusCode := c.Writer.Status()

		// 7. Post-execution reconciliation
		if statusCode >= 200 && statusCode < 300 {
			// Commit durable response into PostgreSQL
			respBytes := blw.body.Bytes()
			queryComplete := `
				UPDATE idempotency_requests
				SET status = 'COMPLETED',
				    response_status_code = $1,
				    response_body = $2,
				    expires_at = NOW() + $3,
				    updated_at = NOW()
				WHERE idempotency_key = $4 AND target_route = $5
			`
			if _, updErr := conn.Exec(ctx, queryComplete, statusCode, respBytes, defaultTTL, idempotencyKey, targetRoute); updErr != nil {
				reqLogger.Error("Failed to update durable idempotency record to COMPLETED", "error", updErr)
			}

			// Store in Redis cache for fast replays
			if redisClient != nil {
				cacheData, _ := json.Marshal(cachedResponse{
					Status: statusCode,
					Hash:   requestHash,
					Body:   blw.body.String(),
				})
				_ = redisClient.RDB.Set(ctx, cacheKey, string(cacheData), defaultTTL).Err()
				_ = redisClient.RDB.Del(ctx, leaseKey).Err()
			}
		} else {
			// On request failure (4xx or 5xx), clean up or mark FAILED so client can retry
			queryFail := `
				UPDATE idempotency_requests
				SET status = 'FAILED', updated_at = NOW()
				WHERE idempotency_key = $1 AND target_route = $2
			`
			_, _ = conn.Exec(ctx, queryFail, idempotencyKey, targetRoute)

			if redisClient != nil {
				_ = redisClient.RDB.Del(ctx, leaseKey).Err()
			}
		}
	}
}
