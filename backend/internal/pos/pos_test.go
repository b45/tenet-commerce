package pos_test

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

	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

func setupTestDB(t *testing.T) (*database.PostgresDB, *tenant.Repository) {
	ctx := context.Background()
	db, err := database.NewPostgresDB(ctx)
	if err != nil {
		t.Skipf("Skipping POS test: Database not available: %v", err)
	}
	tenantRepo := tenant.NewRepository(db)
	return db, tenantRepo
}

func TestPOS_GetCatalog(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/products", posHandler.GetProducts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pos/products", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SKU-BEEF-01")
	assert.Contains(t, w.Body.String(), "SKU-CHICKEN-01")
}

func TestPOS_Checkout_SuccessAndStockDecrement(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.POST("/pos/checkout", posHandler.Checkout)

	// Fetch initial stock for SKU-BEEF-01
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, _ = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public;")

	var initialStock int
	err = conn.QueryRow(ctx, `
		SELECT i.stock_quantity 
		FROM products p 
		JOIN inventory i ON p.id = i.product_id 
		WHERE p.sku = 'SKU-BEEF-01'
	`).Scan(&initialStock)
	require.NoError(t, err)
	conn.Release()

	// Execute purchase of 2 units of SKU-BEEF-01
	idempotencyKey := fmt.Sprintf("pos-test-key-%d", time.Now().UnixNano())
	checkoutPayload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-BEEF-01", Quantity: 2},
		},
		PaymentMethod:  "CASH",
		DiscountAmount: 5000.00,
	}

	bodyBytes, _ := json.Marshal(checkoutPayload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pos/checkout", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"success":true`)
	assert.Contains(t, w.Body.String(), "TXN-")
	assert.Contains(t, w.Body.String(), `"total_amount":145000`) // (75000 * 2) - 5000 = 145000

	// Verify stock was decremented by exactly 2
	conn2, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, _ = conn2.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public;")

	var finalStock int
	err = conn2.QueryRow(ctx, `
		SELECT i.stock_quantity 
		FROM products p 
		JOIN inventory i ON p.id = i.product_id 
		WHERE p.sku = 'SKU-BEEF-01'
	`).Scan(&finalStock)
	require.NoError(t, err)
	conn2.Release()

	assert.Equal(t, initialStock-2, finalStock, "Stock must be decremented by exactly 2")
}

func TestPOS_Checkout_InsufficientStock(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	posService := pos.NewService(posRepo)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.POST("/pos/checkout", posHandler.Checkout)

	// Attempt to purchase 99999 units (far exceeds stock)
	idempotencyKey := fmt.Sprintf("pos-oversell-test-%d", time.Now().UnixNano())
	checkoutPayload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-BEEF-01", Quantity: 99999},
		},
		PaymentMethod: "CASH",
	}

	bodyBytes, _ := json.Marshal(checkoutPayload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pos/checkout", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(w, req)

	// Must fail with 409 Conflict
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "INSUFFICIENT_STOCK")
}
