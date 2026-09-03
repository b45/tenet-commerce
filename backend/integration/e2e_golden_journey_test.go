package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/manager"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/supplychain"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

// setupFullCommerceRouter mounts all domain handlers into a single coherent HTTP engine
func setupFullCommerceRouter(t *testing.T, db *database.PostgresDB, rdb *pkgRedis.Client) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tenantRepo := tenant.NewRepository(db)
	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(pos.NewRepository(), ledgerService)
	scService := supplychain.NewService(supplychain.NewRepository(), ledgerService)
	mgrService := manager.NewService(manager.NewRepository())

	posHandler := pos.NewHandler(posService)
	scHandler := supplychain.NewHandler(scService)
	ledgerHandler := ledger.NewHandler(ledgerService)
	mgrHandler := manager.NewHandler(mgrService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			UserID:   "11111111-1111-1111-1111-111111111111",
			TenantID: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
			Role:     "MANAGER",
			Permissions: []string{
				"pos:checkout", "pos:void", "inventory:read", "inventory:write",
				"supply_chain:manage", "ledger:read", "ledger:write", "analytics:read",
			},
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))

	posHandler.RegisterRoutes(router.Group("/api/v1/pos"), rdb)
	scHandler.RegisterRoutes(router.Group("/api/v1/supply-chain"), rdb)
	ledgerHandler.RegisterRoutes(router.Group("/api/v1/ledger"), rdb)
	mgrHandler.RegisterRoutes(router.Group("/api/v1/manager"))

	return router
}

// TestE2E_FullCommerceLifecycle_GoldenJourney executes a complete real-world enterprise commerce day:
// 1. Supplier Onboarding & Halal Cert Registration
// 2. PO Issuance for 20 units of Beef
// 3. Goods Receipt (+20 Stock, Auto-post AP Journal: Debit 1030 / Credit 2010)
// 4. POS Cashier Checkout (-5 Stock, Cash Tender & Change, Auto-post Sales Journal: Debit Cash/COGS, Credit Rev/Inv)
// 5. Product Traceability Audit (From shelf stock back to Halal cert)
// 6. Trial Balance Audit (Sum Debits == Sum Credits)
// 7. Manager Analytics & KPI Aggregation
func TestE2E_FullCommerceLifecycle_GoldenJourney(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router := setupFullCommerceRouter(t, db, rdb)
	now := time.Now()

	// -------------------------------------------------------------------------
	// Step 0: Identify Product (SKU-BEEF-01) and Record Initial Stock
	// -------------------------------------------------------------------------
	var productID uuid.UUID
	var initialStock int
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		err = conn.QueryRow(ctx, `
			SELECT p.id, COALESCE(i.stock_quantity, 0)
			FROM products p
			LEFT JOIN inventory i ON i.product_id = p.id
			WHERE p.sku = 'SKU-BEEF-01'
		`).Scan(&productID, &initialStock)
		require.NoError(t, err)
		conn.Release()
	}

	// -------------------------------------------------------------------------
	// Step 1: Onboard Halal Supplier with Valid BPJPH Certificate
	// -------------------------------------------------------------------------
	supplierPayload := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-GJ-%d", now.UnixNano()),
		CompanyName:   "PT Berkah Jagal Nusantara",
		ContactPerson: "Haji Sulaiman",
		ContactEmail:  "sulaiman@berkahjagal.co.id",
		ContactPhone:  "08123456789",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("BPJPH-GJ-%d", now.UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Ruminant Slaughtering & Distribution",
			ValidFrom:         now.AddDate(0, -1, 0).Format("2006-01-02"),
			ExpiryDate:        now.AddDate(2, 0, 0).Format("2006-01-02"),
		},
	}
	suppBody, _ := json.Marshal(supplierPayload)
	suppReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(suppBody))
	suppReq.Header.Set("Content-Type", "application/json")
	suppW := httptest.NewRecorder()
	router.ServeHTTP(suppW, suppReq)
	require.Equal(t, http.StatusCreated, suppW.Code)

	var suppResp struct {
		Data supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(suppW.Body.Bytes(), &suppResp))
	supplierID := suppResp.Data.ID
	require.NotNil(t, suppResp.Data.ComplianceCertificate)
	certID := suppResp.Data.ComplianceCertificate.ID.String()

	// -------------------------------------------------------------------------
	// Step 2: Issue Purchase Order for 20 units of Beef @ Rp 75.000 (Rp 1.500.000)
	// -------------------------------------------------------------------------
	poPayload := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       supplierID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{ProductID: productID.String(), Quantity: 20, UnitCost: 75000},
		},
	}
	poBody, _ := json.Marshal(poPayload)
	poReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	poReq.Header.Set("Content-Type", "application/json")
	poReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-po-gj-%d", now.UnixNano()))
	poW := httptest.NewRecorder()
	router.ServeHTTP(poW, poReq)
	require.Equal(t, http.StatusCreated, poW.Code)

	var poResp struct {
		Data supplychain.PurchaseOrder `json:"data"`
	}
	require.NoError(t, json.Unmarshal(poW.Body.Bytes(), &poResp))
	poID := poResp.Data.ID
	assert.Equal(t, "ISSUED", poResp.Data.Status)
	assert.Equal(t, float64(1500000), poResp.Data.TotalAmount)

	// -------------------------------------------------------------------------
	// Step 3: Receive Goods Receipt for 20 units (+20 Inventory Stock)
	// -------------------------------------------------------------------------
	grPayload := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: poID.String(),
		Notes:           "Fresh certified Halal beef delivered in temperature-controlled truck",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID.String(), ReceivedQuantity: 20},
		},
	}
	grBody, _ := json.Marshal(grPayload)
	grReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody))
	grReq.Header.Set("Content-Type", "application/json")
	grReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-gr-gj-%d", now.UnixNano()))
	grW := httptest.NewRecorder()
	router.ServeHTTP(grW, grReq)
	require.Equal(t, http.StatusCreated, grW.Code)

	var grResp struct {
		Data supplychain.GoodsReceipt `json:"data"`
	}
	require.NoError(t, json.Unmarshal(grW.Body.Bytes(), &grResp))
	grID := grResp.Data.ID

	// Verify PO is now RECEIVED and has 0 remaining balance
	poDetailReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/purchase-orders/%s", poID), nil)
	poDetailW := httptest.NewRecorder()
	router.ServeHTTP(poDetailW, poDetailReq)
	require.Equal(t, http.StatusOK, poDetailW.Code)

	var poDetailResp struct {
		Data supplychain.PurchaseOrderDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(poDetailW.Body.Bytes(), &poDetailResp))
	assert.Equal(t, "RECEIVED", poDetailResp.Data.Status)
	require.Len(t, poDetailResp.Data.Items, 1)
	assert.Equal(t, 20, poDetailResp.Data.Items[0].ReceivedQuantity)
	assert.Equal(t, 0, poDetailResp.Data.Items[0].RemainingQuantity)

	// Verify GR Detail includes Ledger cross-reference
	grDetailReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/goods-receipts/%s", grID), nil)
	grDetailW := httptest.NewRecorder()
	router.ServeHTTP(grDetailW, grDetailReq)
	require.Equal(t, http.StatusOK, grDetailW.Code)

	var grDetailResp struct {
		Data supplychain.GoodsReceiptDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(grDetailW.Body.Bytes(), &grDetailResp))
	assert.Equal(t, float64(1500000), grDetailResp.Data.TotalValuation)
	require.NotNil(t, grDetailResp.Data.LedgerEntryNumber, "ledger journal entry must exist for GR")

	// -------------------------------------------------------------------------
	// Step 4: POS Cashier Checkout: Customer Buys 5 units of Beef
	// 5 units @ SKU-BEEF-01, Tender Rp 500.000 Cash
	// -------------------------------------------------------------------------
	cashTender := 500000.0
	checkoutPayload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-BEEF-01", Quantity: 5},
		},
		PaymentMethod: "CASH",
		CashTendered:  &cashTender,
	}
	coBody, _ := json.Marshal(checkoutPayload)
	coReq := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(coBody))
	coReq.Header.Set("Content-Type", "application/json")
	coReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-pos-gj-%d", now.UnixNano()))
	coW := httptest.NewRecorder()
	router.ServeHTTP(coW, coReq)
	require.Equal(t, http.StatusCreated, coW.Code)

	var coResp struct {
		Data pos.CheckoutResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(coW.Body.Bytes(), &coResp))
	assert.Equal(t, float64(500000), coResp.Data.CashTendered)
	assert.Greater(t, coResp.Data.TotalAmount, float64(0))
	assert.GreaterOrEqual(t, coResp.Data.ChangeAmount, float64(0))
	assert.NotEmpty(t, coResp.Data.TransactionNumber)

	// -------------------------------------------------------------------------
	// Step 5: Verify Document-Level Traceability (Shelf Stock to Halal Supplier)
	// -------------------------------------------------------------------------
	traceReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/traceability/product/%s", productID), nil)
	traceW := httptest.NewRecorder()
	router.ServeHTTP(traceW, traceReq)
	require.Equal(t, http.StatusOK, traceW.Code)

	var traceResp struct {
		Data supplychain.ProductTraceabilityReport `json:"data"`
	}
	require.NoError(t, json.Unmarshal(traceW.Body.Bytes(), &traceResp))
	report := traceResp.Data
	assert.Equal(t, productID, report.ProductID)
	assert.Equal(t, "SKU-BEEF-01", report.SKU)
	// Stock: initial + 20 (received) - 5 (sold)
	assert.Equal(t, initialStock+15, report.CurrentStock)

	var foundSupplier bool
	for _, s := range report.Suppliers {
		if s.CompanyName == "PT Berkah Jagal Nusantara" {
			foundSupplier = true
			require.NotEmpty(t, s.Certificates)
			assert.Equal(t, "BPJPH", s.Certificates[0].IssuingAuthority)
			break
		}
	}
	assert.True(t, foundSupplier, "traceability report must link to Halal supplier")

	// -------------------------------------------------------------------------
	// Step 6: Verify Double-Entry Sharia Ledger Trial Balance
	// Total Debits MUST EQUAL Total Credits across all posted journal entries
	// -------------------------------------------------------------------------
	tbReq := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/trial-balance", nil)
	tbW := httptest.NewRecorder()
	router.ServeHTTP(tbW, tbReq)
	require.Equal(t, http.StatusOK, tbW.Code)

	var tbResp struct {
		Data struct {
			TotalDebit  float64 `json:"total_debit"`
			TotalCredit float64 `json:"total_credit"`
			IsBalanced  bool    `json:"is_balanced"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(tbW.Body.Bytes(), &tbResp))
	assert.True(t, tbResp.Data.IsBalanced, "double-entry trial balance must be strictly balanced")
	assert.InDelta(t, tbResp.Data.TotalDebit, tbResp.Data.TotalCredit, 0.001, "total debit must strictly equal total credit")

	// -------------------------------------------------------------------------
	// Step 7: Verify Store Manager Dashboard KPIs & Financial Summary
	// -------------------------------------------------------------------------
	mgrReq := httptest.NewRequest(http.MethodGet, "/api/v1/manager/dashboard", nil)
	mgrW := httptest.NewRecorder()
	router.ServeHTTP(mgrW, mgrReq)
	require.Equal(t, http.StatusOK, mgrW.Code)

	var mgrResp struct {
		Data manager.DashboardSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(mgrW.Body.Bytes(), &mgrResp))
	// Sales summary must capture today's revenue
	assert.Greater(t, mgrResp.Data.SalesSummary.TodayGrossSales, float64(0))
	assert.GreaterOrEqual(t, mgrResp.Data.SalesSummary.TodayOrdersCount, 1)
	// Active accounts count in general ledger
	assert.Greater(t, mgrResp.Data.FinancialSummary.ActiveAccountsCount, 0)
}

// TestE2E_ConcurrentFinalUnitOversellDefense proves that under concurrent checkout load
// targeting the final remaining inventory units, database row-level locking strictly
// defends against overselling:
// Stock = 3 units -> 12 concurrent checkouts of 1 unit each -> Exactly 3 succeed, 9 rejected.
func TestE2E_ConcurrentFinalUnitOversellDefense(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router := setupFullCommerceRouter(t, db, rdb)

	// Set a dedicated product stock (SKU-OIL-01) to exactly 3 units
	var productID uuid.UUID
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		err = conn.QueryRow(ctx, "SELECT id FROM products WHERE sku = 'SKU-OIL-01'").Scan(&productID)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "UPDATE inventory SET stock_quantity = 3 WHERE product_id = $1", productID)
		require.NoError(t, err)
		conn.Release()
	}

	const concurrentWorkers = 12
	var successCount int32
	var failureCount int32

	var wg sync.WaitGroup
	wg.Add(concurrentWorkers)

	for i := 0; i < concurrentWorkers; i++ {
		workerID := i
		go func() {
			defer wg.Done()
			cashTender := 100000.0
			payload := pos.CheckoutRequest{
				Items: []pos.CartItemRequest{
					{SKU: "SKU-OIL-01", Quantity: 1},
				},
				PaymentMethod: "CASH",
				CashTendered:  &cashTender,
			}
			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", fmt.Sprintf("idem-oversell-%d-%d", time.Now().UnixNano(), workerID))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusCreated {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&failureCount, 1)
			}
		}()
	}

	wg.Wait()

	// Assert: Exactly 3 checkouts succeeded, exactly 9 rejected
	assert.Equal(t, int32(3), successCount, "exactly 3 checkouts should succeed for 3 available units")
	assert.Equal(t, int32(9), failureCount, "remaining checkouts should be rejected to prevent overselling")

	// Verify final database stock is exactly 0
	var finalStock int
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		err = conn.QueryRow(ctx, "SELECT stock_quantity FROM inventory WHERE product_id = $1", productID).Scan(&finalStock)
		require.NoError(t, err)
		conn.Release()
	}
	assert.Equal(t, 0, finalStock, "final database stock must be exactly 0 (no negative/phantom stock)")
}

// TestE2E_ConcurrentIdempotentReplayDefense proves that firing identical concurrent requests
// with the same idempotency key results in exactly 1 mutation and consistent replay responses.
func TestE2E_ConcurrentIdempotentReplayDefense(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router := setupFullCommerceRouter(t, db, rdb)

	var productID uuid.UUID
	var initialStock int
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		err = conn.QueryRow(ctx, `
			SELECT p.id, i.stock_quantity
			FROM products p
			JOIN inventory i ON i.product_id = p.id
			WHERE p.sku = 'SKU-HONEY-01'
		`).Scan(&productID, &initialStock)
		require.NoError(t, err)
		conn.Release()
	}

	const concurrentReplays = 6
	sharedKey := fmt.Sprintf("idem-replay-shared-%d", time.Now().UnixNano())
	cashTender := 200000.0
	payload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-HONEY-01", Quantity: 1},
		},
		PaymentMethod: "CASH",
		CashTendered:  &cashTender,
	}
	bodyBytes, _ := json.Marshal(payload)

	var wg sync.WaitGroup
	wg.Add(concurrentReplays)
	responses := make([]*httptest.ResponseRecorder, concurrentReplays)

	for i := 0; i < concurrentReplays; i++ {
		idx := i
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", sharedKey)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			responses[idx] = w
		}()
	}

	wg.Wait()

	// Invariant: Exactly 1 concurrent request processes the mutation (201 Created),
	// while other in-flight concurrent attempts are protected against race conditions with 409 Conflict.
	var successCount int
	var conflictCount int
	var transactionNumber string

	for _, w := range responses {
		if w.Code == http.StatusCreated {
			successCount++
			var coResp struct {
				Data pos.CheckoutResponse `json:"data"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &coResp))
			if transactionNumber == "" {
				transactionNumber = coResp.Data.TransactionNumber
			} else {
				assert.Equal(t, transactionNumber, coResp.Data.TransactionNumber, "every 201 response must have identical transaction number")
			}
		} else if w.Code == http.StatusConflict {
			conflictCount++
		}
	}
	assert.GreaterOrEqual(t, successCount, 1, "at least 1 request should succeed")
	assert.Equal(t, concurrentReplays, successCount+conflictCount, "all requests must be either 201 (success/replay) or 409 (in-flight conflict)")
	assert.NotEmpty(t, transactionNumber)

	// Now send a post-completion idempotent replay request with the same key
	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(bodyBytes))
	replayReq.Header.Set("Content-Type", "application/json")
	replayReq.Header.Set("Idempotency-Key", sharedKey)
	replayW := httptest.NewRecorder()
	router.ServeHTTP(replayW, replayReq)

	assert.Equal(t, http.StatusCreated, replayW.Code, "post-completion request must return cached 201 Created")
	var replayResp struct {
		Data pos.CheckoutResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(replayW.Body.Bytes(), &replayResp))
	assert.Equal(t, transactionNumber, replayResp.Data.TransactionNumber, "replay must return identical transaction number")

	// Verify inventory decremented ONLY ONCE
	var finalStock int
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		err = conn.QueryRow(ctx, "SELECT stock_quantity FROM inventory WHERE product_id = $1", productID).Scan(&finalStock)
		require.NoError(t, err)
		conn.Release()
	}
	assert.Equal(t, initialStock-1, finalStock, "stock should decrement exactly once despite multiple concurrent replays")
}
