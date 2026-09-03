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

func setupSupplyChainTestRouter(t *testing.T, db *database.PostgresDB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tenantRepo := tenant.NewRepository(db)
	ledgerService := ledger.NewService(ledger.NewRepository())
	scService := supplychain.NewService(supplychain.NewRepository(), ledgerService)
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
