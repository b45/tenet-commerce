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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/supplychain"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

type apiResponseEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func setupSupplyChainTestRouter(t *testing.T, db *database.PostgresDB, services ...*supplychain.Service) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tenantRepo := tenant.NewRepository(db)
	ledgerService := ledger.NewService(ledger.NewRepository())
	scService := supplychain.NewService(supplychain.NewRepository(), ledgerService)
	if len(services) > 0 {
		scService = services[0]
	}
	scHandler := supplychain.NewHandler(scService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{Permissions: []string{"supply_chain:manage"}})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	rdb := newTestRedisClient(t)
	scHandler.RegisterRoutes(router.Group("/api/v1/supply-chain"), rdb)

	return router
}

func TestSupplyChain_GoodsReceiptAtomicityAndReconciliation(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create a supplier with a valid Halal certificate (strict mode ON for al-barakah-mart)
	supplierReq := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-TEST-%d", time.Now().UnixNano()),
		CompanyName:   "PT Berkah Pangan Mandiri",
		ContactPerson: "Ahmad",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("CERT-HALAL-%d", time.Now().UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Daging Sapi Halal Segar",
			ValidFrom:         time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, err := json.Marshal(supplierReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("sup-key-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var supplierResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &supplierResp))
	var createdSupplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(supplierResp.Data, &createdSupplier))
	require.NotNil(t, createdSupplier.ComplianceCertificate)

	certID := createdSupplier.ComplianceCertificate.ID.String()

	// Step 2: Create Purchase Order for 10 units of Beef (10000000-0000-0000-0000-000000000001)
	productID := "10000000-0000-0000-0000-000000000001"
	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       createdSupplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{
				ProductID: productID,
				Quantity:  10,
				UnitCost:  50000,
			},
		},
	}
	poBody, err := json.Marshal(poReq)
	require.NoError(t, err)

	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-key-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))
	assert.Equal(t, "ISSUED", createdPO.Status)

	initialStock := stockForSKU(t, db, "SKU-BEEF-01")

	// Step 3: Partial Receipt of 4 units
	grKey1 := fmt.Sprintf("gr-key-%d-1", time.Now().UnixNano())
	grReq1 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Batch 1 delivery: 4 units",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 4},
		},
	}
	grBody1, err := json.Marshal(grReq1)
	require.NoError(t, err)

	reqGR1 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
	reqGR1.Header.Set("Content-Type", "application/json")
	reqGR1.Header.Set("Idempotency-Key", grKey1)
	wGR1 := httptest.NewRecorder()
	router.ServeHTTP(wGR1, reqGR1)
	require.Equal(t, http.StatusCreated, wGR1.Code)

	var grResp1 apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wGR1.Body.Bytes(), &grResp1))
	var gr1 supplychain.GoodsReceipt
	require.NoError(t, json.Unmarshal(grResp1.Data, &gr1))
	assert.Equal(t, grKey1, gr1.IdempotencyKey)
	assert.Len(t, gr1.Items, 1)

	// Invariant Checks after partial receipt:
	// - PO status is PARTIALLY_RECEIVED
	// - Stock incremented by exactly 4
	// - Balanced general ledger journal entry posted
	currentStock := stockForSKU(t, db, "SKU-BEEF-01")
	assert.Equal(t, initialStock+4, currentStock)
	assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")
	assertLedgerJournalExists(t, db, gr1.ID, 4*50000.0)

	// Step 4: Replay with identical Idempotency-Key returns original receipt without duplicate side-effects
	reqGR1Replay := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
	reqGR1Replay.Header.Set("Content-Type", "application/json")
	reqGR1Replay.Header.Set("Idempotency-Key", grKey1)
	wGR1Replay := httptest.NewRecorder()
	router.ServeHTTP(wGR1Replay, reqGR1Replay)
	require.Equal(t, http.StatusCreated, wGR1Replay.Code)

	var grReplayResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wGR1Replay.Body.Bytes(), &grReplayResp))
	var grReplay supplychain.GoodsReceipt
	require.NoError(t, json.Unmarshal(grReplayResp.Data, &grReplay))
	assert.Equal(t, gr1.ID, grReplay.ID, "replay must return identical goods receipt")
	assert.Equal(t, initialStock+4, stockForSKU(t, db, "SKU-BEEF-01"), "stock must not double increment on replay")

	// Step 5: Idempotency Key Conflict (reusing grKey1 on another PO)
	// Create another PO first
	reqPO2 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO2.Header.Set("Content-Type", "application/json")
	reqPO2.Header.Set("Idempotency-Key", fmt.Sprintf("po2-key-%d", time.Now().UnixNano()))
	wPO2 := httptest.NewRecorder()
	router.ServeHTTP(wPO2, reqPO2)
	require.Equal(t, http.StatusCreated, wPO2.Code)
	var po2Resp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO2.Body.Bytes(), &po2Resp))
	var createdPO2 supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(po2Resp.Data, &createdPO2))

	conflictReq := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO2.ID.String(),
		Notes:           "Conflicting key delivery",
		Items:           []supplychain.CreateGRItemRequest{{ProductID: productID, ReceivedQuantity: 2}},
	}
	conflictBody, _ := json.Marshal(conflictReq)
	reqConflict := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(conflictBody))
	reqConflict.Header.Set("Content-Type", "application/json")
	reqConflict.Header.Set("Idempotency-Key", grKey1) // re-used on PO2
	wConflict := httptest.NewRecorder()
	router.ServeHTTP(wConflict, reqConflict)
	assert.Equal(t, http.StatusConflict, wConflict.Code)

	// Step 6: Over-receipt rejection (attempting to receive 7 units when only 6 remain)
	overReceiptReq := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Items:           []supplychain.CreateGRItemRequest{{ProductID: productID, ReceivedQuantity: 7}},
	}
	overBody, _ := json.Marshal(overReceiptReq)
	reqOver := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(overBody))
	reqOver.Header.Set("Content-Type", "application/json")
	reqOver.Header.Set("Idempotency-Key", fmt.Sprintf("gr-key-over-%d", time.Now().UnixNano()))
	wOver := httptest.NewRecorder()
	router.ServeHTTP(wOver, reqOver)
	assert.Equal(t, http.StatusUnprocessableEntity, wOver.Code)
	assert.Equal(t, initialStock+4, stockForSKU(t, db, "SKU-BEEF-01"), "failed over-receipt must not change stock")

	// Step 7: Final Partial Receipt of remaining 6 units -> PO completes to RECEIVED
	grKey2 := fmt.Sprintf("gr-key-%d-2", time.Now().UnixNano())
	grReq2 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Batch 2 delivery: remaining 6 units",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 6},
		},
	}
	grBody2, _ := json.Marshal(grReq2)
	reqGR2 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody2))
	reqGR2.Header.Set("Content-Type", "application/json")
	reqGR2.Header.Set("Idempotency-Key", grKey2)
	wGR2 := httptest.NewRecorder()
	router.ServeHTTP(wGR2, reqGR2)
	require.Equal(t, http.StatusCreated, wGR2.Code)

	assert.Equal(t, initialStock+10, stockForSKU(t, db, "SKU-BEEF-01"), "total stock should reflect full PO fulfillment")
	assertPOStatus(t, db, createdPO.ID, "RECEIVED")

	// Step 8: Subsequent receipt on already-received PO is rejected
	reqGRPostComplete := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody2))
	reqGRPostComplete.Header.Set("Content-Type", "application/json")
	reqGRPostComplete.Header.Set("Idempotency-Key", fmt.Sprintf("gr-key-late-%d", time.Now().UnixNano()))
	wLate := httptest.NewRecorder()
	router.ServeHTTP(wLate, reqGRPostComplete)
	assert.Equal(t, http.StatusConflict, wLate.Code)
}

func TestSupplyChain_HalalComplianceHardBlocksReceiptOnExpiredCert(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// 1. Create supplier with certificate valid today
	supplierReq := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-EXP-%d", time.Now().UnixNano()),
		CompanyName:   "PT Daging Nusantara Halal",
		ContactPerson: "Hassan",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("CERT-EXP-%d", time.Now().UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Daging Sapi",
			ValidFrom:         time.Now().AddDate(0, -1, 0).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(0, 1, 0).Format("2006-01-02"),
		},
	}
	body, _ := json.Marshal(supplierReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("sup-exp-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var supplierResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &supplierResp))
	var supplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(supplierResp.Data, &supplier))
	certID := supplier.ComplianceCertificate.ID.String()

	// 2. Issue PO
	productID := "10000000-0000-0000-0000-000000000001"
	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       supplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{ProductID: productID, Quantity: 5, UnitCost: 50000},
		},
	}
	poBody, _ := json.Marshal(poReq)
	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-key-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))

	// 3. Simulate certificate expiry before receipt
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	_, err = conn.Exec(context.Background(), `
		UPDATE tenant_al_barakah_mart.compliance_certificates
		SET expiry_date = CURRENT_DATE - INTERVAL '1 day'
		WHERE id = $1
	`, supplier.ComplianceCertificate.ID)
	conn.Release()
	require.NoError(t, err)

	initialStock := stockForSKU(t, db, "SKU-BEEF-01")

	// 4. Attempt Goods Receipt - should be hard blocked by Halal compliance engine
	grReq := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Attempt delivery with expired halal cert",
		Items:           []supplychain.CreateGRItemRequest{{ProductID: productID, ReceivedQuantity: 5}},
	}
	grBody, _ := json.Marshal(grReq)
	reqGR := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody))
	reqGR.Header.Set("Content-Type", "application/json")
	reqGR.Header.Set("Idempotency-Key", fmt.Sprintf("gr-expired-test-%d", time.Now().UnixNano()))
	wGR := httptest.NewRecorder()
	router.ServeHTTP(wGR, reqGR)

	assert.Equal(t, http.StatusUnprocessableEntity, wGR.Code)
	var errResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wGR.Body.Bytes(), &errResp))
	assert.Equal(t, "COMPLIANCE_ERROR", errResp.Error.Code)
	assert.Equal(t, initialStock, stockForSKU(t, db, "SKU-BEEF-01"), "stock must not be modified when compliance check fails")
	assertPOStatus(t, db, createdPO.ID, "ISSUED")
}

func TestSupplyChain_ExactMoney_ThreeUnitsAt10001Produces30003(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create Supplier
	supplierReq := supplychain.CreateSupplierRequest{
		Code:        fmt.Sprintf("SUPP-EXACT-%d", time.Now().UnixNano()%1000000),
		CompanyName: "Exact Meat Supplier",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("EXACT-CERT-%d", time.Now().UnixNano()%1000000),
			IssuingAuthority:  "BPJPH",
			Scope:             "Poultry",
			ValidFrom:         "2024-01-01",
			ExpiryDate:        "2030-01-01",
		},
	}
	sBody, _ := json.Marshal(supplierReq)
	reqS := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(sBody))
	reqS.Header.Set("Content-Type", "application/json")
	reqS.Header.Set("Idempotency-Key", fmt.Sprintf("exact-supp-%d", time.Now().UnixNano()))
	wS := httptest.NewRecorder()
	router.ServeHTTP(wS, reqS)
	require.Equal(t, http.StatusCreated, wS.Code)

	var sResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wS.Body.Bytes(), &sResp))
	var supplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(sResp.Data, &supplier))

	productID := "10000000-0000-0000-0000-000000000001"

	require.NotNil(t, supplier.ComplianceCertificate)
	certID := supplier.ComplianceCertificate.ID.String()

	// Step 2: Create Purchase Order with 3 units @ 10,001 IDR.
	// Invariant: 3 * 10001 = exactly 30003 IDR without floating point penny drift.
	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       supplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{
				ProductID: productID,
				Quantity:  3,
				UnitCost:  10001,
			},
		},
	}
	poBody, _ := json.Marshal(poReq)
	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-exact-key-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))
	assert.Equal(t, 30003.0, createdPO.TotalAmount)
	require.Len(t, createdPO.Items, 1)
	assert.Equal(t, 30003.0, createdPO.Items[0].Subtotal)

	// Step 3: Receive all 3 units via Goods Receipt
	grKey := fmt.Sprintf("gr-exact-key-%d", time.Now().UnixNano())
	grReq := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Exact receipt: 3 units at 10001 IDR",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 3},
		},
	}
	grBody, _ := json.Marshal(grReq)
	reqGR := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody))
	reqGR.Header.Set("Content-Type", "application/json")
	reqGR.Header.Set("Idempotency-Key", grKey)
	wGR := httptest.NewRecorder()
	router.ServeHTTP(wGR, reqGR)
	require.Equal(t, http.StatusCreated, wGR.Code)

	var grResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wGR.Body.Bytes(), &grResp))
	var gr supplychain.GoodsReceipt
	require.NoError(t, json.Unmarshal(grResp.Data, &gr))

	// Step 4: Verify ledger journal entry posted has debit and credit = exactly 30003.00
	assertLedgerJournalExists(t, db, gr.ID, 30003.0)
	assertPOStatus(t, db, createdPO.ID, "RECEIVED")
}

func TestSupplyChain_RejectsFractionalAndOverflowCosts(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	productID := "10000000-0000-0000-0000-000000000001"

	// 1. Fractional unit cost (e.g. 10001.50) must be rejected with 400 Bad Request
	poReqFractional := map[string]interface{}{
		"supplier_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		"items": []map[string]interface{}{
			{
				"product_id": productID,
				"quantity":   3,
				"unit_cost":  10001.50,
			},
		},
	}
	poBody1, _ := json.Marshal(poReqFractional)
	reqPO1 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody1))
	reqPO1.Header.Set("Content-Type", "application/json")
	reqPO1.Header.Set("Idempotency-Key", fmt.Sprintf("po-frac-key-%d", time.Now().UnixNano()))
	wPO1 := httptest.NewRecorder()
	router.ServeHTTP(wPO1, reqPO1)
	assert.Equal(t, http.StatusBadRequest, wPO1.Code)
	assert.Contains(t, wPO1.Body.String(), "INVALID_MONETARY_AMOUNT")

	// 2. Unit cost exceeding 1 Billion IDR must be rejected with 400 Bad Request
	poReqOverflow := map[string]interface{}{
		"supplier_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		"items": []map[string]interface{}{
			{
				"product_id": productID,
				"quantity":   1,
				"unit_cost":  1000000001,
			},
		},
	}
	poBody2, _ := json.Marshal(poReqOverflow)
	reqPO2 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody2))
	reqPO2.Header.Set("Content-Type", "application/json")
	reqPO2.Header.Set("Idempotency-Key", fmt.Sprintf("po-overflow-key-%d", time.Now().UnixNano()))
	wPO2 := httptest.NewRecorder()
	router.ServeHTTP(wPO2, reqPO2)
	assert.Equal(t, http.StatusBadRequest, wPO2.Code)
	assert.Contains(t, wPO2.Body.String(), "INVALID_MONETARY_AMOUNT")
}

func assertPOStatus(t *testing.T, db *database.PostgresDB, poID uuid.UUID, expectedStatus string) {
	t.Helper()
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()

	var status string
	err = conn.QueryRow(context.Background(), `
		SELECT status FROM tenant_al_barakah_mart.purchase_orders WHERE id = $1
	`, poID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, expectedStatus, status)
}

func assertLedgerJournalExists(t *testing.T, db *database.PostgresDB, grID uuid.UUID, expectedAmount float64) {
	t.Helper()
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()

	var entryID uuid.UUID
	var totalDebit, totalCredit float64
	err = conn.QueryRow(context.Background(), `
		SELECT le.id, COALESCE(SUM(lel.debit_amount), 0), COALESCE(SUM(lel.credit_amount), 0)
		FROM tenant_al_barakah_mart.ledger_entries le
		JOIN tenant_al_barakah_mart.ledger_entry_lines lel ON lel.ledger_entry_id = le.id
		WHERE le.source_document_id = $1 AND le.source_document_type = 'GOODS_RECEIPT'
		GROUP BY le.id
	`, grID).Scan(&entryID, &totalDebit, &totalCredit)
	require.NoError(t, err)
	assert.Equal(t, expectedAmount, totalDebit)
	assert.Equal(t, expectedAmount, totalCredit)
	assert.Equal(t, totalDebit, totalCredit, "general ledger journal MUST be strictly balanced")
}

func TestSupplyChain_ReceivingReconciliation_SequentialAndOverreceipt(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create active supplier with valid cert
	supplierReq := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-RECON-%d", time.Now().UnixNano()),
		CompanyName:   "PT Rekonsiliasi Distribusi",
		ContactPerson: "Bambang",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("CERT-RECON-%d", time.Now().UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Minyak Goreng Halal",
			ValidFrom:         time.Now().AddDate(0, 0, -10).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, err := json.Marshal(supplierReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("sup-recon-key-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var supplierResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &supplierResp))
	var createdSupplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(supplierResp.Data, &createdSupplier))
	require.NotNil(t, createdSupplier.ComplianceCertificate)

	certID := createdSupplier.ComplianceCertificate.ID.String()

	// Step 2: Create PO with quantity 100 of SKU-OIL-01 (10000000-0000-0000-0000-000000000004)
	productID := "10000000-0000-0000-0000-000000000004"
	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       createdSupplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{
				ProductID: productID,
				Quantity:  100,
				UnitCost:  25000,
			},
		},
	}
	poBody, err := json.Marshal(poReq)
	require.NoError(t, err)

	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-recon-key-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))
	assert.Equal(t, "ISSUED", createdPO.Status)

	initialStock := stockForSKU(t, db, "SKU-OIL-01")

	// Step 3: Receive 60 units (remaining = 40)
	grReq1 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Batch 1: 60 units",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 60},
		},
	}
	grBody1, _ := json.Marshal(grReq1)
	reqGR1 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
	reqGR1.Header.Set("Content-Type", "application/json")
	reqGR1.Header.Set("Idempotency-Key", fmt.Sprintf("gr-seq-1-%d", time.Now().UnixNano()))
	wGR1 := httptest.NewRecorder()
	router.ServeHTTP(wGR1, reqGR1)
	require.Equal(t, http.StatusCreated, wGR1.Code)

	assert.Equal(t, initialStock+60, stockForSKU(t, db, "SKU-OIL-01"))
	assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")

	// Step 4: Attempt to receive 41 units (remaining is only 40) -> Must fail with 422 RECEIPT_RECONCILIATION_FAILED
	grReqOver := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Over-receipt attempt: 41 units",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 41},
		},
	}
	grBodyOver, _ := json.Marshal(grReqOver)
	reqGROver := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBodyOver))
	reqGROver.Header.Set("Content-Type", "application/json")
	reqGROver.Header.Set("Idempotency-Key", fmt.Sprintf("gr-seq-over-%d", time.Now().UnixNano()))
	wGROver := httptest.NewRecorder()
	router.ServeHTTP(wGROver, reqGROver)
	assert.Equal(t, http.StatusUnprocessableEntity, wGROver.Code)
	assert.Contains(t, wGROver.Body.String(), "RECEIPT_RECONCILIATION_FAILED")
	// Verify stock unchanged
	assert.Equal(t, initialStock+60, stockForSKU(t, db, "SKU-OIL-01"))
	assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")

	// Step 5: Receive exactly remaining 40 units -> PO status must transition to RECEIVED
	grReq2 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Batch 2: remaining 40 units",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 40},
		},
	}
	grBody2, _ := json.Marshal(grReq2)
	reqGR2 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody2))
	reqGR2.Header.Set("Content-Type", "application/json")
	reqGR2.Header.Set("Idempotency-Key", fmt.Sprintf("gr-seq-2-%d", time.Now().UnixNano()))
	wGR2 := httptest.NewRecorder()
	router.ServeHTTP(wGR2, reqGR2)
	require.Equal(t, http.StatusCreated, wGR2.Code)

	assert.Equal(t, initialStock+100, stockForSKU(t, db, "SKU-OIL-01"))
	assertPOStatus(t, db, createdPO.ID, "RECEIVED")
}

func TestSupplyChain_ReceivingReconciliation_MultiLineOrder(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create supplier
	supplierReq := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-MULTI-%d", time.Now().UnixNano()),
		CompanyName:   "PT Multi Pangan Segar",
		ContactPerson: "Hendra",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("CERT-MULTI-%d", time.Now().UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Daging dan Ayam Segar",
			ValidFrom:         time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, _ := json.Marshal(supplierReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("sup-multi-key-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var supplierResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &supplierResp))
	var createdSupplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(supplierResp.Data, &createdSupplier))

	certID := createdSupplier.ComplianceCertificate.ID.String()

	// Step 2: PO with 2 lines: Beef (qty 10) and Chicken (qty 20)
	beefID := "10000000-0000-0000-0000-000000000001"
	chickenID := "10000000-0000-0000-0000-000000000002"

	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       createdSupplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{ProductID: beefID, Quantity: 10, UnitCost: 50000},
			{ProductID: chickenID, Quantity: 20, UnitCost: 25000},
		},
	}
	poBody, _ := json.Marshal(poReq)
	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-multi-key-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))

	beefInitial := stockForSKU(t, db, "SKU-BEEF-01")
	chickenInitial := stockForSKU(t, db, "SKU-CHICKEN-01")

	// Step 3: Receive 100% of beef line, but 0% of chicken line
	grReq1 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Beef fully delivered",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: beefID, ReceivedQuantity: 10},
		},
	}
	grBody1, _ := json.Marshal(grReq1)
	reqGR1 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
	reqGR1.Header.Set("Content-Type", "application/json")
	reqGR1.Header.Set("Idempotency-Key", fmt.Sprintf("gr-multi-1-%d", time.Now().UnixNano()))
	wGR1 := httptest.NewRecorder()
	router.ServeHTTP(wGR1, reqGR1)
	require.Equal(t, http.StatusCreated, wGR1.Code)

	assert.Equal(t, beefInitial+10, stockForSKU(t, db, "SKU-BEEF-01"))
	assert.Equal(t, chickenInitial, stockForSKU(t, db, "SKU-CHICKEN-01"))
	// PO MUST remain PARTIALLY_RECEIVED because Chicken is not yet fulfilled
	assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")

	// Step 4: Receive remaining chicken line (20 units) -> PO should now transition to RECEIVED
	grReq2 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Chicken fully delivered",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: chickenID, ReceivedQuantity: 20},
		},
	}
	grBody2, _ := json.Marshal(grReq2)
	reqGR2 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody2))
	reqGR2.Header.Set("Content-Type", "application/json")
	reqGR2.Header.Set("Idempotency-Key", fmt.Sprintf("gr-multi-2-%d", time.Now().UnixNano()))
	wGR2 := httptest.NewRecorder()
	router.ServeHTTP(wGR2, reqGR2)
	require.Equal(t, http.StatusCreated, wGR2.Code)

	assert.Equal(t, beefInitial+10, stockForSKU(t, db, "SKU-BEEF-01"))
	assert.Equal(t, chickenInitial+20, stockForSKU(t, db, "SKU-CHICKEN-01"))
	// Now all lines are 100% fulfilled
	assertPOStatus(t, db, createdPO.ID, "RECEIVED")
}

func TestSupplyChain_ReceivingReconciliation_MissingInventoryFailsAndRollsBack(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create a product WITHOUT an inventory row in tenant_al_barakah_mart
	productID := uuid.New()
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	_, err = conn.Exec(context.Background(), `
		INSERT INTO tenant_al_barakah_mart.products (id, sku, name, unit_price, cost_price, is_active)
		VALUES ($1, $2, 'Ghost Product Without Inventory', 10000, 8000, true)
	`, productID, fmt.Sprintf("SKU-GHOST-%d", time.Now().UnixNano()))
	require.NoError(t, err)
	conn.Release()

	// Step 2: Create supplier and PO referencing this ghost product
	supplierReq := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-GHOST-%d", time.Now().UnixNano()),
		CompanyName:   "PT Ghost Supply",
		ContactPerson: "Casper",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("CERT-GHOST-%d", time.Now().UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Ghost Supply Halal",
			ValidFrom:         time.Now().AddDate(0, 0, -1).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, _ := json.Marshal(supplierReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("sup-ghost-key-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var supplierResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &supplierResp))
	var createdSupplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(supplierResp.Data, &createdSupplier))

	certID := createdSupplier.ComplianceCertificate.ID.String()

	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       createdSupplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{ProductID: productID.String(), Quantity: 15, UnitCost: 8000},
		},
	}
	poBody, _ := json.Marshal(poReq)
	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-ghost-key-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))

	// Step 3: Attempt Goods Receipt on ghost product -> Must fail with 422 INVENTORY_RECORD_NOT_FOUND
	grReq := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Delivery of ghost item",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID.String(), ReceivedQuantity: 15},
		},
	}
	grBody, _ := json.Marshal(grReq)
	reqGR := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody))
	reqGR.Header.Set("Content-Type", "application/json")
	reqGR.Header.Set("Idempotency-Key", fmt.Sprintf("gr-ghost-key-%d", time.Now().UnixNano()))
	wGR := httptest.NewRecorder()
	router.ServeHTTP(wGR, reqGR)

	assert.Equal(t, http.StatusUnprocessableEntity, wGR.Code)
	assert.Contains(t, wGR.Body.String(), "INVENTORY_RECORD_NOT_FOUND")

	// Verify PO status remained ISSUED (clean atomic rollback)
	assertPOStatus(t, db, createdPO.ID, "ISSUED")

	// Verify no goods receipt was saved in database
	conn2, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn2.Release()
	var grCount int
	err = conn2.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM tenant_al_barakah_mart.goods_receipts WHERE purchase_order_id = $1
	`, createdPO.ID).Scan(&grCount)
	require.NoError(t, err)
	assert.Equal(t, 0, grCount, "goods receipt must not exist after rollback")
}

func TestSupplyChain_InlineQualityChecks(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create Supplier with active Halal cert
	supplierReq := supplychain.CreateSupplierRequest{
		Code:        fmt.Sprintf("SUP-QC-%d", time.Now().UnixNano()),
		CompanyName: "PT Agro Makmur QC Test",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("CERT-QC-%d", time.Now().UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Produk Pangan Halal",
			ValidFrom:         time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, err := json.Marshal(supplierReq)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("sup-qc-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var supplierResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &supplierResp))
	var createdSupplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(supplierResp.Data, &createdSupplier))
	certID := createdSupplier.ComplianceCertificate.ID.String()

	// Step 2: Create PO 100 units of Beef (10000000-0000-0000-0000-000000000001) at 50,000 IDR
	productID := "10000000-0000-0000-0000-000000000001"
	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       createdSupplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{
				ProductID: productID,
				Quantity:  100,
				UnitCost:  50000,
			},
		},
	}
	poBody, _ := json.Marshal(poReq)
	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-qc-%d", time.Now().UnixNano()))
	wPO := httptest.NewRecorder()
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))
	assert.Equal(t, "ISSUED", createdPO.Status)

	initialStock := stockForSKU(t, db, "SKU-BEEF-01")

	// Case 1: Inconsistent arithmetic (delivered 60, accepted 50, rejected 5 -> 50+5 != 60) -> 400 Bad Request
	{
		del := 60
		acc := 50
		rej := 5
		reason := "Some defect"
		badArithmeticReq := supplychain.CreateGoodsReceiptRequest{
			PurchaseOrderID: createdPO.ID.String(),
			Notes:           "Invalid arithmetic attempt",
			Items: []supplychain.CreateGRItemRequest{
				{
					ProductID:         productID,
					DeliveredQuantity: &del,
					AcceptedQuantity:  &acc,
					RejectedQuantity:  &rej,
					QCReason:          &reason,
				},
			},
		}
		badBody, _ := json.Marshal(badArithmeticReq)
		reqBad := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(badBody))
		reqBad.Header.Set("Content-Type", "application/json")
		reqBad.Header.Set("Idempotency-Key", fmt.Sprintf("gr-bad-arith-%d", time.Now().UnixNano()))
		wBad := httptest.NewRecorder()
		router.ServeHTTP(wBad, reqBad)
		assert.Equal(t, http.StatusBadRequest, wBad.Code)
		assert.Equal(t, initialStock, stockForSKU(t, db, "SKU-BEEF-01"), "stock must not change on rejected request")
	}

	// Case 2: Rejected > 0 without reason -> 400 Bad Request
	{
		del := 60
		acc := 55
		rej := 5
		badReasonReq := supplychain.CreateGoodsReceiptRequest{
			PurchaseOrderID: createdPO.ID.String(),
			Notes:           "Missing reason attempt",
			Items: []supplychain.CreateGRItemRequest{
				{
					ProductID:         productID,
					DeliveredQuantity: &del,
					AcceptedQuantity:  &acc,
					RejectedQuantity:  &rej,
				},
			},
		}
		badReasonBody, _ := json.Marshal(badReasonReq)
		reqBadReason := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(badReasonBody))
		reqBadReason.Header.Set("Content-Type", "application/json")
		reqBadReason.Header.Set("Idempotency-Key", fmt.Sprintf("gr-bad-reason-%d", time.Now().UnixNano()))
		wBadReason := httptest.NewRecorder()
		router.ServeHTTP(wBadReason, reqBadReason)
		assert.Equal(t, http.StatusBadRequest, wBadReason.Code)
	}

	// Case 3: Mixed acceptance on PO 100: delivered 60, accepted 55, rejected 5
	// Expected:
	// - stock +55
	// - remaining outstanding 45
	// - PO status = PARTIALLY_RECEIVED
	// - QC item: accepted=55, rejected=5, outcome=PARTIAL_ACCEPT, reason recorded
	// - Ledger journal posted for exactly 55 * 50,000 = 2,750,000 IDR
	var gr1ID string
	{
		del := 60
		acc := 55
		rej := 5
		reason := "Packaging torn on 5 boxes"
		grReq1 := supplychain.CreateGoodsReceiptRequest{
			PurchaseOrderID: createdPO.ID.String(),
			Notes:           "Batch 1 delivery with inline QC",
			Items: []supplychain.CreateGRItemRequest{
				{
					ProductID:         productID,
					DeliveredQuantity: &del,
					AcceptedQuantity:  &acc,
					RejectedQuantity:  &rej,
					QCReason:          &reason,
				},
			},
		}
		gr1Body, _ := json.Marshal(grReq1)
		reqGR1 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(gr1Body))
		reqGR1.Header.Set("Content-Type", "application/json")
		reqGR1.Header.Set("Idempotency-Key", fmt.Sprintf("gr-qc-1-%d", time.Now().UnixNano()))
		wGR1 := httptest.NewRecorder()
		router.ServeHTTP(wGR1, reqGR1)
		require.Equal(t, http.StatusCreated, wGR1.Code)

		var gr1Resp apiResponseEnvelope
		require.NoError(t, json.Unmarshal(wGR1.Body.Bytes(), &gr1Resp))
		var gr1 supplychain.GoodsReceipt
		require.NoError(t, json.Unmarshal(gr1Resp.Data, &gr1))
		gr1ID = gr1.ID.String()

		require.Len(t, gr1.Items, 1)
		assert.Equal(t, 60, gr1.Items[0].DeliveredQuantity)
		assert.Equal(t, 55, gr1.Items[0].AcceptedQuantity)
		assert.Equal(t, 5, gr1.Items[0].RejectedQuantity)
		assert.Equal(t, "PARTIAL_ACCEPT", gr1.Items[0].QCOutcome)
		require.NotNil(t, gr1.Items[0].QCReason)
		assert.Equal(t, reason, *gr1.Items[0].QCReason)
		require.NotNil(t, gr1.Items[0].InspectedBy)
		assert.Equal(t, "11111111-1111-1111-1111-111111111111", gr1.Items[0].InspectedBy.String())

		// Stock incremented by exactly 55
		assert.Equal(t, initialStock+55, stockForSKU(t, db, "SKU-BEEF-01"))
		assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")
		assertLedgerJournalExists(t, db, gr1.ID, 55*50000.0)

		// Check PO detail reflects remaining 45
		reqPODetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/purchase-orders/%s", createdPO.ID), nil)
		wPODetail := httptest.NewRecorder()
		router.ServeHTTP(wPODetail, reqPODetail)
		require.Equal(t, http.StatusOK, wPODetail.Code)
		var poDetailResp apiResponseEnvelope
		require.NoError(t, json.Unmarshal(wPODetail.Body.Bytes(), &poDetailResp))
		var poDetail supplychain.PurchaseOrderDetail
		require.NoError(t, json.Unmarshal(poDetailResp.Data, &poDetail))
		require.Len(t, poDetail.Items, 1)
		assert.Equal(t, 55, poDetail.Items[0].ReceivedQuantity)
		assert.Equal(t, 45, poDetail.Items[0].RemainingQuantity)
	}

	// Case 4: All-rejected inspection: delivered 20, accepted 0, rejected 20
	// Expected:
	// - zero stock increment
	// - zero ledger journal entry (skipped)
	// - PO remaining stays 45, status stays PARTIALLY_RECEIVED (never marked RECEIVED)
	// - inspection history immutable and queryable
	{
		del := 20
		acc := 0
		rej := 20
		reason := "Temperature abuse, thawed meat"
		grReqAllReject := supplychain.CreateGoodsReceiptRequest{
			PurchaseOrderID: createdPO.ID.String(),
			Notes:           "Batch 2: Completely spoiled delivery",
			Items: []supplychain.CreateGRItemRequest{
				{
					ProductID:         productID,
					DeliveredQuantity: &del,
					AcceptedQuantity:  &acc,
					RejectedQuantity:  &rej,
					QCReason:          &reason,
				},
			},
		}
		grAllRejectBody, _ := json.Marshal(grReqAllReject)
		reqGRReject := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grAllRejectBody))
		reqGRReject.Header.Set("Content-Type", "application/json")
		reqGRReject.Header.Set("Idempotency-Key", fmt.Sprintf("gr-qc-all-reject-%d", time.Now().UnixNano()))
		wGRReject := httptest.NewRecorder()
		router.ServeHTTP(wGRReject, reqGRReject)
		require.Equal(t, http.StatusCreated, wGRReject.Code)

		var grRejectResp apiResponseEnvelope
		require.NoError(t, json.Unmarshal(wGRReject.Body.Bytes(), &grRejectResp))
		var grReject supplychain.GoodsReceipt
		require.NoError(t, json.Unmarshal(grRejectResp.Data, &grReject))

		require.Len(t, grReject.Items, 1)
		assert.Equal(t, 20, grReject.Items[0].DeliveredQuantity)
		assert.Equal(t, 0, grReject.Items[0].AcceptedQuantity)
		assert.Equal(t, 20, grReject.Items[0].RejectedQuantity)
		assert.Equal(t, "REJECT", grReject.Items[0].QCOutcome)
		require.NotNil(t, grReject.Items[0].QCReason)
		assert.Equal(t, reason, *grReject.Items[0].QCReason)

		// Zero stock change!
		assert.Equal(t, initialStock+55, stockForSKU(t, db, "SKU-BEEF-01"))
		assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")

		// Verify no ledger journal was created for this all-rejected receipt
		func() {
			conn, err := db.Pool.Acquire(context.Background())
			require.NoError(t, err)
			defer conn.Release()
			var entryCount int
			err = conn.QueryRow(context.Background(), `
				SELECT COUNT(*) FROM tenant_al_barakah_mart.ledger_entries
				WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id = $1
			`, grReject.ID).Scan(&entryCount)
			require.NoError(t, err)
			assert.Equal(t, 0, entryCount, "all-rejected receipt must have zero ledger journals")
		}()

		// Verify GetGoodsReceiptDetail returns full QC data

		reqGRDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/goods-receipts/%s", grReject.ID), nil)
		wGRDetail := httptest.NewRecorder()
		router.ServeHTTP(wGRDetail, reqGRDetail)
		require.Equal(t, http.StatusOK, wGRDetail.Code)

		var grDetailResp apiResponseEnvelope
		require.NoError(t, json.Unmarshal(wGRDetail.Body.Bytes(), &grDetailResp))
		var grDetail supplychain.GoodsReceiptDetail
		require.NoError(t, json.Unmarshal(grDetailResp.Data, &grDetail))
		require.Len(t, grDetail.Items, 1)
		assert.Equal(t, 0, grDetail.Items[0].AcceptedQuantity)
		assert.Equal(t, 20, grDetail.Items[0].RejectedQuantity)
		assert.Equal(t, 20, grDetail.Items[0].DeliveredQuantity)
		assert.Equal(t, "REJECT", grDetail.Items[0].QCOutcome)
		assert.Equal(t, 0.0, grDetail.TotalValuation)
		assert.Nil(t, grDetail.LedgerEntryNumber)
		assert.Nil(t, grDetail.Items[0].StockMovementID)
		var movementCount int
		err = db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM tenant_al_barakah_mart.stock_movements
			WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id = $1`, grReject.ID).Scan(&movementCount)
		require.NoError(t, err)
		assert.Zero(t, movementCount, "rejected goods must not create any stock movement")
	}

	// Verify Detail of first mixed receipt
	{
		reqGR1Detail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/goods-receipts/%s", gr1ID), nil)
		wGR1Detail := httptest.NewRecorder()
		router.ServeHTTP(wGR1Detail, reqGR1Detail)
		require.Equal(t, http.StatusOK, wGR1Detail.Code)

		var gr1DetailResp apiResponseEnvelope
		require.NoError(t, json.Unmarshal(wGR1Detail.Body.Bytes(), &gr1DetailResp))
		var gr1Detail supplychain.GoodsReceiptDetail
		require.NoError(t, json.Unmarshal(gr1DetailResp.Data, &gr1Detail))
		require.Len(t, gr1Detail.Items, 1)
		assert.Equal(t, 55, gr1Detail.Items[0].AcceptedQuantity)
		assert.Equal(t, 5, gr1Detail.Items[0].RejectedQuantity)
		assert.Equal(t, 60, gr1Detail.Items[0].DeliveredQuantity)
		assert.Equal(t, "PARTIAL_ACCEPT", gr1Detail.Items[0].QCOutcome)
		assert.Equal(t, 55*50000.0, gr1Detail.TotalValuation)
		assert.NotNil(t, gr1Detail.LedgerEntryNumber)
		assert.NotNil(t, gr1Detail.Items[0].StockMovementID, "stock movement ID must be linked on accepted goods receipt detail item")
	}
}

func TestSupplyChain_GoodsReceipt_StockMovementIntegration(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// Step 1: Create Supplier with Valid Halal Certificate
	supplierCode := fmt.Sprintf("SUP-SM-%d", time.Now().UnixNano())
	supplierReq := supplychain.CreateSupplierRequest{
		Code:          supplierCode,
		CompanyName:   "PT Movement Halal",
		ContactPerson: "Ahmad SM",
		ContactEmail:  "sm@halal.test",
		ContactPhone:  "081234567890",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertificateNumber: fmt.Sprintf("CERT-HALAL-%d", time.Now().UnixNano()),
			CertType:          "HALAL_MUI",
			IssuingAuthority:  "BPJPH",
			Scope:             "Daging Sapi Halal Segar",
			ValidFrom:         time.Now().AddDate(0, 0, -5).Format("2006-01-02"),
			ExpiryDate:        time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	sBody, err := json.Marshal(supplierReq)
	require.NoError(t, err)

	wSup := httptest.NewRecorder()
	reqSup := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(sBody))
	reqSup.Header.Set("Content-Type", "application/json")
	reqSup.Header.Set("Idempotency-Key", fmt.Sprintf("sup-sm-%d", time.Now().UnixNano()))
	router.ServeHTTP(wSup, reqSup)
	require.Equal(t, http.StatusCreated, wSup.Code)

	var sResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wSup.Body.Bytes(), &sResp))
	var createdSupplier supplychain.Supplier
	require.NoError(t, json.Unmarshal(sResp.Data, &createdSupplier))
	require.NotNil(t, createdSupplier.ComplianceCertificate)
	certID := createdSupplier.ComplianceCertificate.ID.String()

	// Step 2: Create Purchase Order for 100 units of Beef (10000000-0000-0000-0000-000000000001)
	productID := "10000000-0000-0000-0000-000000000001"
	poReq := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       createdSupplier.ID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{
				ProductID: productID,
				Quantity:  100,
				UnitCost:  40000,
			},
		},
	}
	poBody, err := json.Marshal(poReq)
	require.NoError(t, err)

	wPO := httptest.NewRecorder()
	reqPO := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	reqPO.Header.Set("Content-Type", "application/json")
	reqPO.Header.Set("Idempotency-Key", fmt.Sprintf("po-sm-%d", time.Now().UnixNano()))
	router.ServeHTTP(wPO, reqPO)
	require.Equal(t, http.StatusCreated, wPO.Code)

	var poResp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wPO.Body.Bytes(), &poResp))
	var createdPO supplychain.PurchaseOrder
	require.NoError(t, json.Unmarshal(poResp.Data, &createdPO))
	require.Equal(t, "ISSUED", createdPO.Status)

	initialStock := stockForSKU(t, db, "SKU-BEEF-01")

	// Step 3: Partial Receipt 1 (60 units accepted)
	grKey1 := fmt.Sprintf("gr-sm-key-%d-1", time.Now().UnixNano())
	grReq1 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Batch 1: 60 units accepted",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 60},
		},
	}
	grBody1, err := json.Marshal(grReq1)
	require.NoError(t, err)

	// Reject the actual movement insert and prove the whole receipt transaction rolls back.
	ctx := context.Background()
	snapshot := func() [4]int {
		t.Helper()
		var counts [4]int
		err := db.Pool.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM tenant_al_barakah_mart.goods_receipts),
			(SELECT count(*) FROM tenant_al_barakah_mart.goods_receipt_items),
			(SELECT count(*) FROM tenant_al_barakah_mart.stock_movements),
			(SELECT count(*) FROM tenant_al_barakah_mart.ledger_entries)`).
			Scan(&counts[0], &counts[1], &counts[2], &counts[3])
		require.NoError(t, err)
		return counts
	}
	beforeFailure := snapshot()
	_, err = db.Pool.Exec(ctx, `ALTER TABLE tenant_al_barakah_mart.stock_movements
		ADD CONSTRAINT test_receipt_movement_failure CHECK
		(NOT (source_document_type = 'GOODS_RECEIPT' AND quantity_delta = 60)) NOT VALID`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Pool.Exec(ctx, `ALTER TABLE tenant_al_barakah_mart.stock_movements
			DROP CONSTRAINT IF EXISTS test_receipt_movement_failure`)
		require.NoError(t, err)
	})
	failed := httptest.NewRecorder()
	failedReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
	failedReq.Header.Set("Content-Type", "application/json")
	failedReq.Header.Set("Idempotency-Key", grKey1)
	router.ServeHTTP(failed, failedReq)
	require.Equal(t, http.StatusInternalServerError, failed.Code, failed.Body.String())
	assert.Equal(t, beforeFailure, snapshot(), "failed movement must leave no receipt, lines, movement, or journal")
	assert.Equal(t, initialStock, stockForSKU(t, db, "SKU-BEEF-01"))
	assertPOStatus(t, db, createdPO.ID, "ISSUED")
	_, err = db.Pool.Exec(ctx, `ALTER TABLE tenant_al_barakah_mart.stock_movements DROP CONSTRAINT test_receipt_movement_failure`)
	require.NoError(t, err)

	// Commit through the service without delivering an HTTP response or writing its cache.
	// The same-key HTTP request below must recover this committed receipt.
	var committedID uuid.UUID
	func() {
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart")
		require.NoError(t, err)
		service := supplychain.NewService(supplychain.NewRepository(), ledger.NewService(ledger.NewRepository()))
		committed, err := service.CreateGoodsReceipt(ctx, conn,
			uuid.MustParse("11111111-1111-1111-1111-111111111111"), grKey1, &grReq1)
		require.NoError(t, err)
		committedID = committed.ID
	}()
	wGR1 := httptest.NewRecorder()
	reqGR1 := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
	reqGR1.Header.Set("Content-Type", "application/json")
	reqGR1.Header.Set("Idempotency-Key", grKey1)
	router.ServeHTTP(wGR1, reqGR1)
	require.Equal(t, http.StatusCreated, wGR1.Code)

	var gr1Resp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wGR1.Body.Bytes(), &gr1Resp))
	var gr1 supplychain.GoodsReceipt
	require.NoError(t, json.Unmarshal(gr1Resp.Data, &gr1))
	require.Len(t, gr1.Items, 1)
	assert.Equal(t, committedID, gr1.ID, "retry must recover the committed receipt")
	assertLedgerJournalExists(t, db, gr1.ID, 60*40000)

	// Invariant assertion: Stock + 60
	assert.Equal(t, initialStock+60, stockForSKU(t, db, "SKU-BEEF-01"))
	assertPOStatus(t, db, createdPO.ID, "PARTIALLY_RECEIVED")

	// Verify stock movement 1 in DB
	func() {
		conn, err := db.Pool.Acquire(context.Background())
		require.NoError(t, err)
		defer conn.Release()

		var count int
		var delta int
		var movType, srcType, location string
		var srcDocID, srcLineID uuid.UUID
		err = conn.QueryRow(context.Background(), `
			SELECT count(*) OVER(), quantity_delta, movement_type, source_document_type, warehouse_location, source_document_id, source_document_line_id
			FROM tenant_al_barakah_mart.stock_movements
			WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id = $1
			LIMIT 1
		`, gr1.ID).Scan(&count, &delta, &movType, &srcType, &location, &srcDocID, &srcLineID)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "exactly one stock movement must exist for GR1")
		assert.Equal(t, 60, delta)
		assert.Equal(t, "IN", movType)
		assert.Equal(t, "GOODS_RECEIPT", srcType)
		assert.Equal(t, "MAIN_STORE", location)
		assert.Equal(t, gr1.ID, srcDocID)
		assert.Equal(t, gr1.Items[0].ID, srcLineID)
	}()

	// Step 4: Idempotent replay of GR1 must NOT add another stock movement or duplicate stock
	func() {
		wGR1Replay := httptest.NewRecorder()
		reqGR1Replay := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody1))
		reqGR1Replay.Header.Set("Content-Type", "application/json")
		reqGR1Replay.Header.Set("Idempotency-Key", grKey1)
		router.ServeHTTP(wGR1Replay, reqGR1Replay)
		assert.Contains(t, []int{http.StatusOK, http.StatusCreated}, wGR1Replay.Code)

		// Stock unchanged
		assert.Equal(t, initialStock+60, stockForSKU(t, db, "SKU-BEEF-01"))

		// Movement count unchanged
		conn, err := db.Pool.Acquire(context.Background())
		require.NoError(t, err)
		defer conn.Release()
		var count int
		err = conn.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM tenant_al_barakah_mart.stock_movements
			WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id = $1
		`, gr1.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "replayed GR must not duplicate stock movements")
		err = conn.QueryRow(context.Background(), `SELECT count(*) FROM tenant_al_barakah_mart.ledger_entries
			WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id = $1`, gr1.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "recovery and replay must not duplicate journals")
	}()

	// Step 5: Partial Receipt 2 (remaining 40 units accepted)
	grKey2 := fmt.Sprintf("gr-sm-key-%d-2", time.Now().UnixNano())
	grReq2 := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: createdPO.ID.String(),
		Notes:           "Batch 2: remaining 40 units accepted",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID, ReceivedQuantity: 40},
		},
	}
	grBody2, err := json.Marshal(grReq2)
	require.NoError(t, err)
	// Two independent commands race for the same remaining 40 units.
	// Use a multi-connection pool so the PO lock, rather than the pool, serializes them.
	concurrentRouter := setupSupplyChainTestRouter(t, newTestDatabase(t))
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	for i := range 2 {
		go func() {
			<-start
			req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody2))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", fmt.Sprintf("%s-%d", grKey2, i))
			w := httptest.NewRecorder()
			concurrentRouter.ServeHTTP(w, req)
			results <- w
		}()
	}
	close(start)
	first, second := <-results, <-results
	wGR2, rejected := first, second
	if first.Code != http.StatusCreated {
		wGR2, rejected = second, first
	}
	require.Equal(t, http.StatusCreated, wGR2.Code, wGR2.Body.String())
	require.Equal(t, http.StatusConflict, rejected.Code, rejected.Body.String())
	assert.Contains(t, rejected.Body.String(), "INVALID_PO_STATUS")

	var gr2Resp apiResponseEnvelope
	require.NoError(t, json.Unmarshal(wGR2.Body.Bytes(), &gr2Resp))
	var gr2 supplychain.GoodsReceipt
	require.NoError(t, json.Unmarshal(gr2Resp.Data, &gr2))

	// Invariant: PO 100 with partial receipts (60 then 40) creates exactly two movements (+60, +40) and total on_hand increases by 100
	assert.Equal(t, initialStock+100, stockForSKU(t, db, "SKU-BEEF-01"))
	assertPOStatus(t, db, createdPO.ID, "RECEIVED")

	func() {
		conn, err := db.Pool.Acquire(context.Background())
		require.NoError(t, err)
		defer conn.Release()

		var totalMovements int
		var totalDelta int
		err = conn.QueryRow(context.Background(), `
			SELECT COUNT(*), COALESCE(SUM(quantity_delta), 0)
			FROM tenant_al_barakah_mart.stock_movements
			WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id IN ($1, $2)
		`, gr1.ID, gr2.ID).Scan(&totalMovements, &totalDelta)
		require.NoError(t, err)
		assert.Equal(t, 2, totalMovements, "exactly two stock movements must exist across both GRs")
		assert.Equal(t, 100, totalDelta, "sum of stock movement deltas must equal exactly 100")
	}()

	// Step 6: Verify GetGoodsReceiptDetail returns linked StockMovementID
	{
		wDetail := httptest.NewRecorder()
		reqDetail := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/goods-receipts/%s", gr2.ID), nil)
		router.ServeHTTP(wDetail, reqDetail)
		require.Equal(t, http.StatusOK, wDetail.Code)

		var detailResp apiResponseEnvelope
		require.NoError(t, json.Unmarshal(wDetail.Body.Bytes(), &detailResp))
		var detail supplychain.GoodsReceiptDetail
		require.NoError(t, json.Unmarshal(detailResp.Data, &detail))
		require.Len(t, detail.Items, 1)
		require.NotNil(t, detail.Items[0].StockMovementID)

		// Cross-check that the ID in detail matches the movement in stock_movements table
		conn, err := db.Pool.Acquire(context.Background())
		require.NoError(t, err)
		defer conn.Release()

		var matchedQuantity int
		err = conn.QueryRow(context.Background(), `
			SELECT quantity_delta
			FROM tenant_al_barakah_mart.stock_movements
			WHERE id = $1
		`, *detail.Items[0].StockMovementID).Scan(&matchedQuantity)
		require.NoError(t, err)
		assert.Equal(t, 40, matchedQuantity)
	}
}
