package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

func setupPOSTestRouterWithIdempotency(t *testing.T, db *database.PostgresDB) (*gin.Engine, *pos.Service) {
	gin.SetMode(gin.TestMode)

	tenantRepo := tenant.NewRepository(db)
	ledgerService := ledger.NewService(ledger.NewRepository())
	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			Permissions: []string{"pos:checkout", "pos:read", "pos:void"},
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))

	rdb := newTestRedisClient(t)
	posHandler.RegisterRoutes(router.Group("/api/v1/pos"), rdb)

	return router, posService
}

func TestIdempotency_ReplayReturnsIdenticalResponseWithoutRepeatingEffects(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router, _ := setupPOSTestRouterWithIdempotency(t, db)

	// SKU-CHICKEN-01 starts with initial stock 100 in tenant_al_barakah_mart
	initialStock := stockForSKU(t, db, "SKU-CHICKEN-01")

	idempotencyKey := fmt.Sprintf("idem-replay-%d", time.Now().UnixNano())
	payload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{
				SKU:      "SKU-CHICKEN-01",
				Quantity: 2,
			},
		},
		PaymentMethod: "CASH",
		CashTendered:  func() *float64 { val := float64(150000); return &val }(),
	}
	bodyBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	// 1. First execution
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	require.Equal(t, http.StatusCreated, w1.Code)
	type checkoutResp struct {
		Success bool `json:"success"`
		Data    struct {
			TransactionID     string `json:"transaction_id"`
			TransactionNumber string `json:"transaction_number"`
		} `json:"data"`
	}

	var resp1 checkoutResp
	err = json.Unmarshal(w1.Body.Bytes(), &resp1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp1.Data.TransactionID)

	stockAfterFirst := stockForSKU(t, db, "SKU-CHICKEN-01")
	assert.Equal(t, initialStock-2, stockAfterFirst, "stock must decrement by 2 on first execution")

	// 2. Second execution with identical payload and key
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusCreated, w2.Code)
	assert.Equal(t, "true", w2.Header().Get("Idempotent-Replayed"), "replay header must be present")

	var resp2 checkoutResp
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	require.NoError(t, err)
	assert.Equal(t, resp1.Data.TransactionID, resp2.Data.TransactionID, "transaction IDs must match on replay")
	assert.Equal(t, resp1.Data.TransactionNumber, resp2.Data.TransactionNumber, "transaction numbers must match")

	stockAfterSecond := stockForSKU(t, db, "SKU-CHICKEN-01")
	assert.Equal(t, stockAfterFirst, stockAfterSecond, "stock must NOT decrement on replay")
}

func TestIdempotency_PayloadConflictRejection(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router, _ := setupPOSTestRouterWithIdempotency(t, db)

	idempotencyKey := fmt.Sprintf("idem-conflict-%d", time.Now().UnixNano())

	// Payload 1: 1 item
	payload1 := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{
				SKU:      "SKU-CHICKEN-01",
				Quantity: 1,
			},
		},
		PaymentMethod: "CASH",
		CashTendered:  func() *float64 { val := float64(100000); return &val }(),
	}
	body1, _ := json.Marshal(payload1)

	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusCreated, w1.Code)

	// Payload 2: Altered quantity with same idempotency key
	payload2 := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{
				SKU:      "SKU-CHICKEN-01",
				Quantity: 2, // Changed!
			},
		},
		PaymentMethod: "CASH",
		CashTendered:  func() *float64 { val := float64(200000); return &val }(),
	}
	body2, _ := json.Marshal(payload2)

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
	assert.Contains(t, w2.Body.String(), "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_PAYLOAD")
}

func TestIdempotency_PostCommitReplaySurvivesRedisFlush(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	rdb := newTestRedisClient(t)

	tenantRepo := tenant.NewRepository(db)
	ledgerService := ledger.NewService(ledger.NewRepository())
	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			Permissions: []string{"pos:checkout"},
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	posHandler.RegisterRoutes(router.Group("/api/v1/pos"), rdb)

	idempotencyKey := fmt.Sprintf("idem-flush-%d", time.Now().UnixNano())
	payload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{
				SKU:      "SKU-BEEF-01",
				Quantity: 1,
			},
		},
		PaymentMethod: "CASH",
		CashTendered:  func() *float64 { val := float64(150000); return &val }(),
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	// Step 1: Initial successful checkout
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusCreated, w1.Code)

	type flushResp struct {
		Success bool `json:"success"`
		Data    struct {
			TransactionID string `json:"transaction_id"`
		} `json:"data"`
	}

	var resp1 flushResp
	err = json.Unmarshal(w1.Body.Bytes(), &resp1)
	require.NoError(t, err)

	// Step 2: Flush Redis completely to simulate cache eviction or Redis restart
	require.NoError(t, rdb.RDB.FlushDB(context.Background()).Err())

	// Step 3: Send duplicate request with Redis completely empty
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusCreated, w2.Code)
	assert.Equal(t, "true", w2.Header().Get("Idempotent-Replayed"))

	var resp2 flushResp
	err = json.Unmarshal(w2.Body.Bytes(), &resp2)
	require.NoError(t, err)
	assert.Equal(t, resp1.Data.TransactionID, resp2.Data.TransactionID, "must replay identically from PostgreSQL")
}

func TestIdempotency_MissingKeyRejected(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router, _ := setupPOSTestRouterWithIdempotency(t, db)

	payload := map[string]any{"payment_method": "CASH"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Idempotency-Key is omitted

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_IDEMPOTENCY_KEY")
}

// TestIdempotency_SameKeyDifferentConcreteResourceDoesNotReplay verifies that sending the same
// idempotency key and identical body to two distinct resource IDs (e.g. /orders/orderA/void vs /orders/orderB/void)
// does NOT replay the first response for the second resource.
func TestIdempotency_SameKeyDifferentConcreteResourceDoesNotReplay(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	rdb := newTestRedisClient(t)

	tenantRepo := tenant.NewRepository(db)
	ledgerService := ledger.NewService(ledger.NewRepository())
	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			Permissions: []string{"pos:checkout", "pos:void"},
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	posHandler.RegisterRoutes(router.Group("/api/v1/pos"), rdb)

	// Step 1: Create Order 1
	idempotencyKey := fmt.Sprintf("idem-multi-%d", time.Now().UnixNano())
	payloadCheckout := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-CHICKEN-01", Quantity: 1},
		},
		PaymentMethod: "CASH",
		CashTendered:  func() *float64 { val := float64(100000); return &val }(),
	}
	bodyCheckout, _ := json.Marshal(payloadCheckout)

	reqCheckout := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(bodyCheckout))
	reqCheckout.Header.Set("Content-Type", "application/json")
	reqCheckout.Header.Set("Idempotency-Key", idempotencyKey+"-chk")
	wCheckout := httptest.NewRecorder()
	router.ServeHTTP(wCheckout, reqCheckout)
	require.Equal(t, http.StatusCreated, wCheckout.Code)

	var respCheckout struct {
		Data struct {
			TransactionID string `json:"transaction_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wCheckout.Body.Bytes(), &respCheckout))
	orderID1 := respCheckout.Data.TransactionID

	// Step 2: Void Order 1 with shared void key
	voidPayload := map[string]string{"reason": "customer cancelled"}
	voidBody, _ := json.Marshal(voidPayload)

	sharedVoidKey := fmt.Sprintf("idem-void-shared-%d", time.Now().UnixNano())
	reqVoid1 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/orders/"+orderID1+"/void", bytes.NewReader(voidBody))
	reqVoid1.Header.Set("Content-Type", "application/json")
	reqVoid1.Header.Set("Idempotency-Key", sharedVoidKey)
	wVoid1 := httptest.NewRecorder()
	router.ServeHTTP(wVoid1, reqVoid1)
	require.Equal(t, http.StatusOK, wVoid1.Code)
	assert.Contains(t, wVoid1.Body.String(), orderID1)

	// Step 3: Attempt void on a DIFFERENT order ID with the SAME sharedVoidKey and identical body
	differentOrderID := "99999999-9999-9999-9999-999999999999"
	reqVoid2 := httptest.NewRequest(http.MethodPost, "/api/v1/pos/orders/"+differentOrderID+"/void", bytes.NewReader(voidBody))
	reqVoid2.Header.Set("Content-Type", "application/json")
	reqVoid2.Header.Set("Idempotency-Key", sharedVoidKey)
	wVoid2 := httptest.NewRecorder()
	router.ServeHTTP(wVoid2, reqVoid2)

	// Must NOT replay Order 1's void response! It must fail with 404 (or 400) for the non-existent order.
	assert.NotEqual(t, http.StatusOK, wVoid2.Code, "second resource must NOT replay cached response from first resource")
	assert.Empty(t, wVoid2.Header().Get("Idempotent-Replayed"), "must not be flagged as replayed")
	assert.Contains(t, wVoid2.Body.String(), "NOT_FOUND")
}

// TestIdempotency_KeyLengthLimit verifies that oversized idempotency keys are rejected.
func TestIdempotency_KeyLengthLimit(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router, _ := setupPOSTestRouterWithIdempotency(t, db)

	payload := pos.CheckoutRequest{
		Items:         []pos.CartItemRequest{{SKU: "SKU-CHICKEN-01", Quantity: 1}},
		PaymentMethod: "CASH",
		CashTendered:  func() *float64 { val := float64(100000); return &val }(),
	}
	body, _ := json.Marshal(payload)

	// Key exceeding 128 characters
	oversizedKey := fmt.Sprintf("oversized-%0130d", 1)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", oversizedKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "IDEMPOTENCY_KEY_TOO_LONG")
}
