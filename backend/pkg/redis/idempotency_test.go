package redis_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/b45/tenet-commerce/backend/pkg/redis"
)

func TestIdempotencyMiddleware(t *testing.T) {
	ctx := context.Background()
	rdb, err := redis.NewRedisClient(ctx)
	if err != nil {
		t.Skipf("Skipping Redis test: Redis server not available: %v", err)
	}
	defer rdb.Close()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "test-tenant")
		c.Next()
	})
	router.Use(redis.IdempotencyMiddleware(rdb, 10*time.Second))

	counter := 0
	router.POST("/test-mutation", func(c *gin.Context) {
		counter++
		c.JSON(http.StatusCreated, gin.H{
			"execution_count": counter,
			"message":         "Transaction successful",
		})
	})

	idempotencyKey := "test-key-" + time.Now().Format("20060102150405.000000")

	// 1. First execution -> Should process and return 201 Created (counter = 1)
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/test-mutation", nil)
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusCreated, w1.Code)
	assert.Contains(t, w1.Body.String(), `"execution_count":1`)

	// 2. Duplicate execution -> Should serve cached response (counter STILL 1)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/test-mutation", nil)
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusCreated, w2.Code)
	assert.Equal(t, "HIT", w2.Header().Get("X-Cache-Lookup"))
	assert.Contains(t, w2.Body.String(), `"execution_count":1`)
	assert.Equal(t, 1, counter, "Handler must NOT be executed a second time")

	// 3. Missing Idempotency-Key -> Should return 400 Bad Request
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/test-mutation", nil)
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusBadRequest, w3.Code)
	assert.Contains(t, w3.Body.String(), "MISSING_IDEMPOTENCY_KEY")
}
