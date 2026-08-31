package logger_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/b45/tenet-commerce/backend/pkg/logger"
)

func TestContextLogger(t *testing.T) {
	// 1. Fallback to global logger when context is empty
	ctx := context.Background()
	l := logger.FromContext(ctx)
	assert.NotNil(t, l)

	// 2. Retrieve embedded logger from context
	customLogger := logger.NewLogger().With(slog.String("custom_field", "test_value"))
	ctxWithLogger := logger.NewContext(ctx, customLogger)
	retrieved := logger.FromContext(ctxWithLogger)
	assert.Equal(t, customLogger, retrieved)
}

func TestTraceMiddleware_GeneratesTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.RealIPMiddleware())
	router.Use(logger.TraceMiddleware())
	router.GET("/test-trace", func(c *gin.Context) {
		traceID, _ := c.Get("trace_id")
		spanID, _ := c.Get("span_id")
		c.JSON(http.StatusOK, gin.H{
			"trace_id": traceID,
			"span_id":  spanID,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-trace", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Headers must be present
	respTraceID := w.Header().Get("X-Trace-ID")
	respSpanID := w.Header().Get("X-Span-ID")
	assert.NotEmpty(t, respTraceID)
	assert.NotEmpty(t, respSpanID)
	assert.Contains(t, w.Body.String(), respTraceID)
}

func TestTraceMiddleware_PropagatesExistingTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.RealIPMiddleware())
	router.Use(logger.TraceMiddleware())
	router.GET("/test-propagate", func(c *gin.Context) {
		traceID, _ := c.Get("trace_id")
		c.JSON(http.StatusOK, gin.H{"trace_id": traceID})
	})

	customTrace := "incoming-distributed-trace-uuid-12345"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test-propagate", nil)
	req.Header.Set("X-Trace-ID", customTrace)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, customTrace, w.Header().Get("X-Trace-ID"))
}

func TestRealIPMiddleware_ResolvesProxyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.RealIPMiddleware())
	router.GET("/test-ip", func(c *gin.Context) {
		clientIP, _ := c.Get("client_ip")
		c.JSON(http.StatusOK, gin.H{"ip": clientIP})
	})

	// Case 1: Cloudflare IP takes highest priority
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/test-ip", nil)
	req1.Header.Set("CF-Connecting-IP", "198.51.100.1")
	req1.Header.Set("X-Real-IP", "198.51.100.2")
	req1.Header.Set("X-Forwarded-For", "198.51.100.3")
	router.ServeHTTP(w1, req1)
	assert.Contains(t, w1.Body.String(), "198.51.100.1")

	// Case 2: X-Real-IP takes second priority
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/test-ip", nil)
	req2.Header.Set("X-Real-IP", "198.51.100.2")
	req2.Header.Set("X-Forwarded-For", "198.51.100.3")
	router.ServeHTTP(w2, req2)
	assert.Contains(t, w2.Body.String(), "198.51.100.2")

	// Case 3: X-Forwarded-For first client in chain
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("GET", "/test-ip", nil)
	req3.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1, 192.168.1.1")
	router.ServeHTTP(w3, req3)
	assert.Contains(t, w3.Body.String(), "203.0.113.50")
}

func TestRecoveryMiddleware_CatchesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(logger.RealIPMiddleware())
	router.Use(logger.TraceMiddleware())
	router.Use(logger.RecoveryMiddleware())

	router.GET("/panic-route", func(c *gin.Context) {
		panic("database connection suddenly severed!")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic-route", nil)
	router.ServeHTTP(w, req)

	// Must return 500 without crashing the test runner
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "PANIC_RECOVERED")
	assert.NotEmpty(t, w.Header().Get("X-Trace-ID"))
}
