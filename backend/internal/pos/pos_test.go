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

	"github.com/b45/tenet-commerce/backend/internal/ledger"
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

	// Ensure test products have sufficient stock for repeatable checkout tests
	_, _ = db.Pool.Exec(ctx, `UPDATE tenant_al_barakah_mart.inventory SET stock_quantity = 50 WHERE stock_quantity < 10`)

	return db, tenantRepo
}

func TestPOS_GetCatalog(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
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
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
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
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
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

func TestPOS_Checkout_WithCashChangeAndBakeryNotes(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
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

	customerName := "Ibu Kartika"
	notes := "Tulisan: Selamat Ulang Tahun ke-7 Salsa, Lilin angka 7"
	cashTendered := 200000.00
	idempotencyKey := fmt.Sprintf("pos-cash-test-%d", time.Now().UnixNano())

	checkoutPayload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-BEEF-01", Quantity: 2}, // 75000 * 2 = 150000
		},
		PaymentMethod:  "CASH",
		DiscountAmount: 5000.00, // Total = 145000
		CashTendered:   &cashTendered,
		CustomerName:   &customerName,
		Notes:          &notes,
	}

	bodyBytes, _ := json.Marshal(checkoutPayload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pos/checkout", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"success":true`)
	assert.Contains(t, w.Body.String(), `"cash_tendered":200000`)
	assert.Contains(t, w.Body.String(), `"change_amount":55000`)
	assert.Contains(t, w.Body.String(), "Ibu Kartika")
	assert.Contains(t, w.Body.String(), "Selamat Ulang Tahun ke-7 Salsa")
}

func TestPOS_Checkout_QRIS_WithPaymentReference(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
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

	rrn := "QRIS-RRN-99887766"
	idempotencyKey := fmt.Sprintf("pos-qris-test-%d", time.Now().UnixNano())

	checkoutPayload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-CHICKEN-01", Quantity: 1}, // 35000
		},
		PaymentMethod:    "QRIS",
		PaymentReference: &rrn,
	}

	bodyBytes, _ := json.Marshal(checkoutPayload)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/pos/checkout", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"payment_method":"QRIS"`)
	assert.Contains(t, w.Body.String(), "QRIS-RRN-99887766")
}

func TestPOS_GetOrders_And_OrderDetail(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/orders", posHandler.GetOrders)
	router.GET("/pos/orders/:id", posHandler.GetOrderDetail)

	// 1. Query Orders list
	wList := httptest.NewRecorder()
	reqList, _ := http.NewRequest("GET", "/pos/orders?limit=10&offset=0", nil)
	router.ServeHTTP(wList, reqList)

	assert.Equal(t, http.StatusOK, wList.Code)
	assert.Contains(t, wList.Body.String(), `"success":true`)
	assert.Contains(t, wList.Body.String(), `"meta"`)

	var listRes struct {
		Data []pos.OrderSummary `json:"data"`
	}
	err := json.Unmarshal(wList.Body.Bytes(), &listRes)
	require.NoError(t, err)
	require.NotEmpty(t, listRes.Data)

	firstOrder := listRes.Data[0]

	// 2. Query Single Order detail by ID
	wDetail := httptest.NewRecorder()
	reqDetail, _ := http.NewRequest("GET", "/pos/orders/"+firstOrder.ID, nil)
	router.ServeHTTP(wDetail, reqDetail)

	assert.Equal(t, http.StatusOK, wDetail.Code)
	assert.Contains(t, wDetail.Body.String(), firstOrder.TransactionNumber)
	assert.Contains(t, wDetail.Body.String(), `"items"`)
}

func TestPOS_VoidOrder_SuccessAndReversal(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
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
	router.POST("/pos/orders/:id/void", posHandler.VoidOrder)

	// 1. Checkout an order with 2 units of chicken
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, _ = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public;")

	var stockBefore int
	_ = conn.QueryRow(ctx, "SELECT i.stock_quantity FROM products p JOIN inventory i ON p.id = i.product_id WHERE p.sku = 'SKU-CHICKEN-01'").Scan(&stockBefore)
	conn.Release()

	idempotencyKey := fmt.Sprintf("pos-void-test-%d", time.Now().UnixNano())
	checkoutPayload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-CHICKEN-01", Quantity: 2},
		},
		PaymentMethod: "CASH",
	}

	bodyBytes, _ := json.Marshal(checkoutPayload)
	wCheckout := httptest.NewRecorder()
	reqCheckout, _ := http.NewRequest("POST", "/pos/checkout", bytes.NewBuffer(bodyBytes))
	reqCheckout.Header.Set("Content-Type", "application/json")
	reqCheckout.Header.Set("Idempotency-Key", idempotencyKey)
	router.ServeHTTP(wCheckout, reqCheckout)
	require.Equal(t, http.StatusCreated, wCheckout.Code)

	var checkoutRes struct {
		Data pos.CheckoutResponse `json:"data"`
	}
	_ = json.Unmarshal(wCheckout.Body.Bytes(), &checkoutRes)
	orderID := checkoutRes.Data.TransactionID

	// 2. Void the order
	voidPayload := pos.VoidRequest{
		Reason: "Customer cancel - wrong cake size selected",
	}
	voidBytes, _ := json.Marshal(voidPayload)
	wVoid := httptest.NewRecorder()
	reqVoid, _ := http.NewRequest("POST", "/pos/orders/"+orderID+"/void", bytes.NewBuffer(voidBytes))
	reqVoid.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wVoid, reqVoid)

	assert.Equal(t, http.StatusOK, wVoid.Code)
	assert.Contains(t, wVoid.Body.String(), `"status":"VOIDED"`)
	assert.Contains(t, wVoid.Body.String(), "Customer cancel - wrong cake size selected")

	// 3. Verify stock has been restored back
	connVerify, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, _ = connVerify.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public;")

	var stockAfter int
	_ = connVerify.QueryRow(ctx, "SELECT i.stock_quantity FROM products p JOIN inventory i ON p.id = i.product_id WHERE p.sku = 'SKU-CHICKEN-01'").Scan(&stockAfter)
	connVerify.Release()

	assert.Equal(t, stockBefore, stockAfter, "Stock should be restored to initial quantity after void")

	// 4. Double void must be rejected with 409 Conflict
	wVoidAgain := httptest.NewRecorder()
	reqVoidAgain, _ := http.NewRequest("POST", "/pos/orders/"+orderID+"/void", bytes.NewBuffer(voidBytes))
	reqVoidAgain.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wVoidAgain, reqVoidAgain)

	assert.Equal(t, http.StatusConflict, wVoidAgain.Code)
	assert.Contains(t, wVoidAgain.Body.String(), "TRANSACTION_ALREADY_VOIDED")
}

func TestPOS_DailySummary(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/daily-summary", posHandler.GetDailySummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pos/daily-summary", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total_orders"`)
	assert.Contains(t, w.Body.String(), `"net_sales"`)
	assert.Contains(t, w.Body.String(), `"payment_breakdown"`)
}

func TestPOS_QRISConfig(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/qris", posHandler.GetQRISConfig)
	router.PUT("/pos/qris", posHandler.UpdateQRISConfig)

	// 1. Get QRIS Config
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest("GET", "/pos/qris", nil)
	router.ServeHTTP(wGet, reqGet)

	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Contains(t, wGet.Body.String(), `"merchant_name"`)
	assert.Contains(t, wGet.Body.String(), `"qr_string"`)

	// 2. Update QRIS Config
	updateCfg := pos.QRISConfig{
		MerchantName: "Toko Kue B45 Bakery QRIS",
		NMID:         "ID1987654321",
		QRString:     "00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID19876543210303UMI5204549953033605802ID5925TOKO KUE B45 BAKERY QRIS6010JAKARTA SE61051234062070703A0163041D2B",
		QRImageURL:   "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=sample-b45-bakery",
	}
	cfgBytes, _ := json.Marshal(updateCfg)
	wPut := httptest.NewRecorder()
	reqPut, _ := http.NewRequest("PUT", "/pos/qris", bytes.NewBuffer(cfgBytes))
	reqPut.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wPut, reqPut)

	assert.Equal(t, http.StatusOK, wPut.Code)
	assert.Contains(t, wPut.Body.String(), "Toko Kue B45 Bakery QRIS")
}

func TestPOS_BakeryCategories_CRUD(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/categories", posHandler.GetCategories)
	router.GET("/pos/categories/:id", posHandler.GetCategoryByID)
	router.POST("/pos/categories", posHandler.CreateCategory)
	router.PUT("/pos/categories/:id", posHandler.UpdateCategory)
	router.DELETE("/pos/categories/:id", posHandler.DeleteCategory)

	// 1. List seeded categories (should include CAT-CAKE, CAT-BREAD)
	wList := httptest.NewRecorder()
	reqList, _ := http.NewRequest("GET", "/pos/categories", nil)
	router.ServeHTTP(wList, reqList)
	assert.Equal(t, http.StatusOK, wList.Code)
	assert.Contains(t, wList.Body.String(), "CAT-CAKE")
	assert.Contains(t, wList.Body.String(), "CAT-BREAD")

	// 2. Create new category
	uniqueCode := fmt.Sprintf("CAT-%d", time.Now().UnixNano()%100000)
	createReq := pos.CreateCategoryRequest{
		Name: "Kue Kering & Hampers Lebaran",
		Code: uniqueCode,
	}
	bodyBytes, _ := json.Marshal(createReq)
	wCreate := httptest.NewRecorder()
	reqCreate, _ := http.NewRequest("POST", "/pos/categories", bytes.NewBuffer(bodyBytes))
	reqCreate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wCreate, reqCreate)
	assert.Equal(t, http.StatusCreated, wCreate.Code)

	var createdCat struct {
		Data pos.Category `json:"data"`
	}
	err := json.Unmarshal(wCreate.Body.Bytes(), &createdCat)
	require.NoError(t, err)
	assert.NotEmpty(t, createdCat.Data.ID)
	assert.Equal(t, "Kue Kering & Hampers Lebaran", createdCat.Data.Name)

	catID := createdCat.Data.ID

	// 3. Get Category by ID
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest("GET", "/pos/categories/"+catID, nil)
	router.ServeHTTP(wGet, reqGet)
	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Contains(t, wGet.Body.String(), uniqueCode)

	// 4. Update Category
	updateReq := pos.UpdateCategoryRequest{
		Name: "Kue Kering Premium & Hampers",
		Code: uniqueCode,
	}
	updateBytes, _ := json.Marshal(updateReq)
	wUpdate := httptest.NewRecorder()
	reqUpdate, _ := http.NewRequest("PUT", "/pos/categories/"+catID, bytes.NewBuffer(updateBytes))
	reqUpdate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUpdate, reqUpdate)
	assert.Equal(t, http.StatusOK, wUpdate.Code)
	var updatedCat struct {
		Data pos.Category `json:"data"`
	}
	_ = json.Unmarshal(wUpdate.Body.Bytes(), &updatedCat)
	assert.Equal(t, "Kue Kering Premium & Hampers", updatedCat.Data.Name)

	// 5. Delete Category
	wDelete := httptest.NewRecorder()
	reqDelete, _ := http.NewRequest("DELETE", "/pos/categories/"+catID, nil)
	router.ServeHTTP(wDelete, reqDelete)
	assert.Equal(t, http.StatusOK, wDelete.Code)
}

func TestPOS_BakeryProduct_CRUD(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/products/:id", posHandler.GetProductByID)
	router.POST("/pos/products", posHandler.CreateProduct)
	router.PUT("/pos/products/:id", posHandler.UpdateProduct)
	router.DELETE("/pos/products/:id", posHandler.DeleteProduct)

	// 1. Create Product
	uniqueSKU := fmt.Sprintf("SKU-BAKE-%d", time.Now().UnixNano()%1000000)
	catID := "c0000000-0000-0000-0000-000000000010" // Kue Tart & Custom Cake
	desc := "Bolu chiffon lembut pandan wangi dengan santan kelapa murni"
	barcode := fmt.Sprintf("899%d", time.Now().UnixNano()%1000000000)
	createReq := pos.CreateProductRequest{
		Name:              "Chiffon Pandan Special 20cm",
		SKU:               uniqueSKU,
		Barcode:           &barcode,
		Description:       &desc,
		CategoryID:        &catID,
		UnitPrice:         55000,
		CostPrice:         32000,
		InitialStock:      20,
		ReorderThreshold:  5,
		WarehouseLocation: "BAKERY_CHILLER_B",
		ComplianceTags:    []string{"HALAL_MUI"},
	}
	bodyBytes, _ := json.Marshal(createReq)
	wCreate := httptest.NewRecorder()
	reqCreate, _ := http.NewRequest("POST", "/pos/products", bytes.NewBuffer(bodyBytes))
	reqCreate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wCreate, reqCreate)
	assert.Equal(t, http.StatusCreated, wCreate.Code)

	var createdProd struct {
		Data pos.Product `json:"data"`
	}
	err := json.Unmarshal(wCreate.Body.Bytes(), &createdProd)
	require.NoError(t, err)
	prodID := createdProd.Data.ID
	assert.NotEmpty(t, prodID)
	assert.Equal(t, 20, createdProd.Data.StockQuantity)
	assert.True(t, createdProd.Data.IsHalalCertified)

	// 2. Get Product by ID
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest("GET", "/pos/products/"+prodID, nil)
	router.ServeHTTP(wGet, reqGet)
	assert.Equal(t, http.StatusOK, wGet.Code)
	assert.Contains(t, wGet.Body.String(), uniqueSKU)
	assert.Contains(t, wGet.Body.String(), "Chiffon Pandan Special 20cm")

	// 3. Update Product
	updateDesc := "Bolu chiffon ekstra pandan suji harum"
	updateReq := pos.UpdateProductRequest{
		Name:             "Chiffon Pandan Special 20cm (Premium)",
		Barcode:          &barcode,
		Description:      &updateDesc,
		CategoryID:       &catID,
		UnitPrice:        60000,
		CostPrice:        35000,
		ReorderThreshold: 6,
		ComplianceTags:   []string{"HALAL_MUI"},
	}
	updateBytes, _ := json.Marshal(updateReq)
	wUpdate := httptest.NewRecorder()
	reqUpdate, _ := http.NewRequest("PUT", "/pos/products/"+prodID, bytes.NewBuffer(updateBytes))
	reqUpdate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUpdate, reqUpdate)
	assert.Equal(t, http.StatusOK, wUpdate.Code)
	assert.Contains(t, wUpdate.Body.String(), "Chiffon Pandan Special 20cm (Premium)")

	// 4. Soft Delete Product
	wDelete := httptest.NewRecorder()
	reqDelete, _ := http.NewRequest("DELETE", "/pos/products/"+prodID, nil)
	router.ServeHTTP(wDelete, reqDelete)
	assert.Equal(t, http.StatusOK, wDelete.Code)
	assert.Contains(t, wDelete.Body.String(), "soft-deleted successfully")
}

func TestPOS_InventoryStockAdjustment_ShrinkageWriteOff(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.POST("/pos/inventory/adjust", posHandler.AdjustStock)
	router.GET("/pos/products/:id", posHandler.GetProductByID)

	// Black Forest Cake seeded product: 10000000-0000-0000-0000-000000000011
	productID := "10000000-0000-0000-0000-000000000011"

	// Fetch current stock
	wBefore := httptest.NewRecorder()
	reqBefore, _ := http.NewRequest("GET", "/pos/products/"+productID, nil)
	router.ServeHTTP(wBefore, reqBefore)
	require.Equal(t, http.StatusOK, wBefore.Code)
	var prodBefore struct {
		Data pos.Product `json:"data"`
	}
	_ = json.Unmarshal(wBefore.Body.Bytes(), &prodBefore)
	initialStock := prodBefore.Data.StockQuantity

	// Write off 1 cake due to DAMAGE (frosting collapsed in transit)
	adjustReq := pos.StockAdjustmentRequest{
		ProductID:      productID,
		AdjustmentType: "SUBTRACT",
		Quantity:       1,
		Reason:         "DAMAGE",
		Notes:          "Kue terbentur saat penataan etalase, krim rusak",
	}
	bodyBytes, _ := json.Marshal(adjustReq)
	wAdjust := httptest.NewRecorder()
	reqAdjust, _ := http.NewRequest("POST", "/pos/inventory/adjust", bytes.NewBuffer(bodyBytes))
	reqAdjust.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wAdjust, reqAdjust)

	assert.Equal(t, http.StatusOK, wAdjust.Code)
	assert.Contains(t, wAdjust.Body.String(), "DAMAGE")
	assert.Contains(t, wAdjust.Body.String(), "JE-ADJ-")

	var adjResp struct {
		Data pos.StockAdjustmentResponse `json:"data"`
	}
	err := json.Unmarshal(wAdjust.Body.Bytes(), &adjResp)
	require.NoError(t, err)
	assert.Equal(t, initialStock-1, adjResp.Data.NewQuantity)
	assert.Equal(t, -1, adjResp.Data.QuantityDelta)
	assert.NotNil(t, adjResp.Data.LedgerEntryNumber)
	assert.Contains(t, *adjResp.Data.LedgerEntryNumber, "JE-ADJ-")
}

func TestPOS_InventoryLowStockAlert(t *testing.T) {
	db, tenantRepo := setupTestDB(t)
	defer db.Close()

	posRepo := pos.NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(posRepo, ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/pos/inventory/low-stock", posHandler.GetLowStock)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/pos/inventory/low-stock", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data"`)
	assert.Contains(t, w.Body.String(), `"meta"`)
}

