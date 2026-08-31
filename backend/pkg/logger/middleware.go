package logger

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// RealIPMiddleware resolves the real client IP behind proxies (Cloudflare, Nginx, ALB)
func RealIPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := ""

		// 1. Cloudflare header
		if cfIP := strings.TrimSpace(c.GetHeader("CF-Connecting-IP")); cfIP != "" {
			clientIP = cfIP
		} else if xRealIP := strings.TrimSpace(c.GetHeader("X-Real-IP")); xRealIP != "" {
			// 2. Nginx / reverse proxy header
			clientIP = xRealIP
		} else if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
			// 3. X-Forwarded-For header (first client IP in comma-separated chain)
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				clientIP = strings.TrimSpace(ips[0])
			}
		}

		// 4. Gin fallback
		if clientIP == "" || net.ParseIP(clientIP) == nil {
			clientIP = c.ClientIP()
		}

		c.Set("client_ip", clientIP)
		c.Next()
	}
}

// generateSpanID creates a random 8-byte hex string (OpenTelemetry-compatible)
func generateSpanID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(bytes)
}

// TraceMiddleware generates or propagates distributed trace correlation IDs (trace_id, span_id)
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Resolve or generate trace_id
		traceID := strings.TrimSpace(c.GetHeader("X-Trace-ID"))
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 2. Generate span_id for this request operation
		spanID := generateSpanID()

		// 3. Inject trace identifiers into Gin context and response headers
		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)
		c.Header("X-Trace-ID", traceID)
		c.Header("X-Span-ID", spanID)

		// 4. Create request-scoped child logger pre-populated with trace metadata
		reqLogger := globalLogger.With(
			slog.String("trace_id", traceID),
			slog.String("span_id", spanID),
		)

		// 5. Inject into standard Go context
		c.Request = c.Request.WithContext(NewContext(c.Request.Context(), reqLogger))

		c.Next()
	}
}

// AccessLogMiddleware captures structured HTTP access metrics for Loki/Promtail/CloudWatch
func AccessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Post-request execution metrics
		latency := time.Since(start)
		latencyMs := float64(latency.Microseconds()) / 1000.0
		status := c.Writer.Status()
		clientIP, _ := c.Get("client_ip")
		traceID, _ := c.Get("trace_id")
		spanID, _ := c.Get("span_id")
		tenantSlug := c.GetString("tenant_slug")
		userID := c.GetString("user_id")
		role := c.GetString("role")

		fullPath := path
		if rawQuery != "" {
			fullPath = path + "?" + rawQuery
		}

		// Collect structured attributes
		attrs := []any{
			slog.String("trace_id", traceID.(string)),
			slog.String("span_id", spanID.(string)),
			slog.String("method", c.Request.Method),
			slog.String("path", fullPath),
			slog.Int("status", status),
			slog.Float64("latency_ms", latencyMs),
			slog.String("client_ip", clientIP.(string)),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.Int("body_bytes_sent", c.Writer.Size()),
		}

		if tenantSlug != "" {
			attrs = append(attrs, slog.String("tenant_slug", tenantSlug))
		}
		if userID != "" {
			attrs = append(attrs, slog.String("user_id", userID))
		}
		if role != "" {
			attrs = append(attrs, slog.String("role", role))
		}

		// Log with appropriate severity level
		reqLogger := FromContext(c.Request.Context())
		if status >= http.StatusInternalServerError {
			reqLogger.Error("HTTP Request Failed", attrs...)
		} else if status >= http.StatusBadRequest {
			reqLogger.Warn("HTTP Request Client Warning", attrs...)
		} else {
			reqLogger.Info("HTTP Request Completed", attrs...)
		}
	}
}

// RecoveryMiddleware catches panics, logs structured stack traces, and returns clean 500 JSON
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				traceID, _ := c.Get("trace_id")
				spanID, _ := c.Get("span_id")
				stack := string(debug.Stack())

				reqLogger := FromContext(c.Request.Context())
				reqLogger.Error("Panic Recovered",
					slog.Any("error", err),
					slog.String("trace_id", traceID.(string)),
					slog.String("span_id", spanID.(string)),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
					slog.String("stack_trace", stack),
				)

				response.AbortInternalServerError(c, "PANIC_RECOVERED",
					"An unexpected internal server error occurred. Please reference your Trace ID.")
			}
		}()

		c.Next()
	}
}

// AuditBodyMiddleware captures request payload for sensitive mutation endpoints (login, financial state, etc.)
func AuditBodyMiddleware() gin.HandlerFunc {
	const maxBodyBytes = 10 * 1024 // 10 KB limit to prevent memory exhaustion

	return func(c *gin.Context) {
		if c.Request.Body != nil && (c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch) {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyBytes))
			if err == nil && len(bodyBytes) > 0 {
				// Restore body so downstream JSON binding still works
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				// Log audit event
				reqLogger := FromContext(c.Request.Context())
				reqLogger.Info("Audit Event Payload Captured",
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("payload", string(bodyBytes)),
				)
			}
		}

		c.Next()
	}
}
