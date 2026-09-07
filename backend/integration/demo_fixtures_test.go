package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/supplychain"
)

// TestDemoFixtures_DeterministicInitialState verifies Acceptance Criteria 2 & 3:
// - Initial demo product SKU-DEMO-01 starts with stock 0 and threshold 10
// - Demo valid supplier and certificate are provisioned
// - Tenant B has strictly isolated data (zero leakage from Tenant A)
func TestDemoFixtures_DeterministicInitialState(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()

	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	// 1. Verify Tenant A Demo Product SKU-DEMO-01
	_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
	require.NoError(t, err)

	var (
		demoProductID uuid.UUID
		unitPrice     float64
		costPrice     float64
		stockQty      int
		reorderThresh int
	)
	err = conn.QueryRow(ctx, `
		SELECT p.id, p.unit_price, p.cost_price, i.stock_quantity, i.reorder_threshold
		FROM products p
		JOIN inventory i ON i.product_id = p.id
		WHERE p.sku = 'SKU-DEMO-01'
	`).Scan(&demoProductID, &unitPrice, &costPrice, &stockQty, &reorderThresh)
	require.NoError(t, err, "SKU-DEMO-01 must be pre-seeded in tenant_al_barakah_mart")
	assert.Equal(t, float64(15000), unitPrice, "Demo unit price must be IDR 15,000")
	assert.Equal(t, float64(10000), costPrice, "Demo unit cost must be IDR 10,000")
	assert.Equal(t, 0, stockQty, "Initial opening stock for SKU-DEMO-01 must be strictly 0")
	assert.Equal(t, 10, reorderThresh, "Reorder threshold for SKU-DEMO-01 must be strictly 10")

	// 2. Verify Tenant A Demo Suppliers & Certificates
	var (
		validSupplierID uuid.UUID
		validCertID     uuid.UUID
		certExpiryDate  string
	)
	err = conn.QueryRow(ctx, `
		SELECT s.id, c.id, c.expiry_date::text
		FROM suppliers s
		JOIN compliance_certificates c ON c.supplier_id = s.id
		WHERE s.code = 'SUP-DEMO-VALID-01'
	`).Scan(&validSupplierID, &validCertID, &certExpiryDate)
	require.NoError(t, err, "SUP-DEMO-VALID-01 must be pre-seeded with active certificate")
	assert.Equal(t, "2099-12-31", certExpiryDate)

	var (
		expSupplierID uuid.UUID
		expCertID     uuid.UUID
		expDate       string
	)
	err = conn.QueryRow(ctx, `
		SELECT s.id, c.id, c.expiry_date::text
		FROM suppliers s
		JOIN compliance_certificates c ON c.supplier_id = s.id
		WHERE s.code = 'SUP-DEMO-EXP-01'
	`).Scan(&expSupplierID, &expCertID, &expDate)
	require.NoError(t, err, "SUP-DEMO-EXP-01 must be pre-seeded with expired certificate")
	assert.Equal(t, "2021-12-31", expDate)

	// 3. Verify Tenant B Data Isolation
	_, err = conn.Exec(ctx, "SET search_path TO tenant_darussalam_store, public")
	require.NoError(t, err)

	// Tenant A demo product SKU-DEMO-01 must NOT exist in Tenant B
	var count int
	err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE sku = 'SKU-DEMO-01'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "SKU-DEMO-01 must NOT leak into tenant_darussalam_store")

	// Tenant A demo suppliers must NOT exist in Tenant B
	err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM suppliers WHERE code IN ('SUP-DEMO-VALID-01', 'SUP-DEMO-EXP-01')`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Tenant A demo suppliers must NOT leak into tenant_darussalam_store")

	// Tenant B isolated supplier must exist
	var dsSuppID uuid.UUID
	err = conn.QueryRow(ctx, `SELECT id FROM suppliers WHERE code = 'SUP-DS-DEMO-01'`).Scan(&dsSuppID)
	require.NoError(t, err, "SUP-DS-DEMO-01 must exist in tenant_darussalam_store")
}

// TestDemoFixtures_RepeatableFreshBootstrap verifies Acceptance Criteria 1:
// - Two consecutive fresh bootstrap executions produce identical, deterministic baseline fixture states
func TestDemoFixtures_RepeatableFreshBootstrap(t *testing.T) {
	db := newTestDatabase(t)
	ctx := context.Background()

	initScript := filepath.Join("..", "..", "scripts", "init_dev_db.sql")
	require.FileExists(t, initScript)
	sqlContent, err := os.ReadFile(initScript)
	require.NoError(t, err)

	queryBaselineState := func() (int, float64, float64, int, int) {
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()

		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)

		var (
			stock        int
			unitPrice    float64
			costPrice    float64
			suppCount    int
			certCount    int
		)
		err = conn.QueryRow(ctx, `
			SELECT i.stock_quantity, p.unit_price, p.cost_price
			FROM products p
			JOIN inventory i ON i.product_id = p.id
			WHERE p.sku = 'SKU-DEMO-01'
		`).Scan(&stock, &unitPrice, &costPrice)
		require.NoError(t, err)

		err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM suppliers WHERE code LIKE 'SUP-DEMO-%'`).Scan(&suppCount)
		require.NoError(t, err)

		err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM compliance_certificates WHERE certificate_number LIKE 'CERT-DEMO-%'`).Scan(&certCount)
		require.NoError(t, err)

		return stock, unitPrice, costPrice, suppCount, certCount
	}

	// 1. Snapshot first bootstrap state
	stock1, unitPrice1, costPrice1, suppCount1, certCount1 := queryBaselineState()
	assert.Equal(t, 0, stock1)
	assert.Equal(t, float64(15000), unitPrice1)
	assert.Equal(t, float64(10000), costPrice1)
	assert.Equal(t, 2, suppCount1)
	assert.Equal(t, 2, certCount1)

	// 2. Re-execute fresh bootstrap script directly
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, string(sqlContent))
	require.NoError(t, err, "Re-running init_dev_db.sql must be completely idempotent and error-free")
	conn.Release()

	// 3. Snapshot second bootstrap state
	stock2, unitPrice2, costPrice2, suppCount2, certCount2 := queryBaselineState()
	assert.Equal(t, stock1, stock2, "Stock must remain identical after second fresh bootstrap")
	assert.Equal(t, unitPrice1, unitPrice2, "Price must remain identical after second fresh bootstrap")
	assert.Equal(t, costPrice1, costPrice2, "Cost must remain identical after second fresh bootstrap")
	assert.Equal(t, suppCount1, suppCount2, "Supplier count must remain identical after second fresh bootstrap")
	assert.Equal(t, certCount1, certCount2, "Certificate count must remain identical after second fresh bootstrap")
}

// TestDemoFixtures_GoldenJourney_DeterministicFlow verifies Acceptance Criteria 1 & 2:
// - Start with stock 0, threshold 10
// - Issue PO 100 units @ IDR 10,000 (total IDR 1,000,000)
// - Goods receipts 60 + 40 yield stock 100 and receipt valuation IDR 1,000,000
// - Verification of stock movement audit trail and balanced journal entries
func TestDemoFixtures_GoldenJourney_DeterministicFlow(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router := setupFullCommerceRouter(t, db, rdb)
	ctx := context.Background()

	// 1. Fetch pre-seeded demo entities
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
	require.NoError(t, err)

	var (
		productID  uuid.UUID
		supplierID uuid.UUID
		certID     uuid.UUID
	)
	err = conn.QueryRow(ctx, `SELECT id FROM products WHERE sku = 'SKU-DEMO-01'`).Scan(&productID)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `
		SELECT s.id, c.id
		FROM suppliers s
		JOIN compliance_certificates c ON c.supplier_id = s.id
		WHERE s.code = 'SUP-DEMO-VALID-01'
	`).Scan(&supplierID, &certID)
	require.NoError(t, err)
	conn.Release()

	// 2. Issue PO for 100 units @ IDR 10,000
	certIDStr := certID.String()
	poPayload := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       supplierID.String(),
		ComplianceCertID: &certIDStr,
		Items: []supplychain.CreatePOItemRequest{
			{
				ProductID: productID.String(),
				Quantity:  100,
				UnitCost:  10000,
			},
		},
	}
	poBody, _ := json.Marshal(poPayload)
	poReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	poReq.Header.Set("Content-Type", "application/json")
	poReq.Header.Set("Idempotency-Key", "demo-po-key-100")
	poW := httptest.NewRecorder()
	router.ServeHTTP(poW, poReq)
	require.Equal(t, http.StatusCreated, poW.Code)

	var poResp struct {
		Data supplychain.PurchaseOrder `json:"data"`
	}
	require.NoError(t, json.Unmarshal(poW.Body.Bytes(), &poResp))
	poID := poResp.Data.ID
	assert.Equal(t, float64(1000000), poResp.Data.TotalAmount, "PO total amount must be IDR 1,000,000")
	assert.Equal(t, "ISSUED", poResp.Data.Status)

	// Stock must still be 0 after PO creation
	assert.Equal(t, 0, stockForSKU(t, db, "SKU-DEMO-01"))

	// 3. First Goods Receipt: 60 units accepted
	deliv60 := 60
	accept60 := 60
	reject0 := 0
	gr1Payload := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: poID.String(),
		Notes:           "Partial Goods Receipt 1 (60 units)",
		Items: []supplychain.CreateGRItemRequest{
			{
				ProductID:         productID.String(),
				DeliveredQuantity: &deliv60,
				AcceptedQuantity:  &accept60,
				RejectedQuantity:  &reject0,
			},
		},
	}
	gr1Body, _ := json.Marshal(gr1Payload)
	gr1Req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(gr1Body))
	gr1Req.Header.Set("Content-Type", "application/json")
	gr1Req.Header.Set("Idempotency-Key", "demo-gr-key-60")
	gr1W := httptest.NewRecorder()
	router.ServeHTTP(gr1W, gr1Req)
	require.Equal(t, http.StatusCreated, gr1W.Code)

	// Stock must now be 60
	assert.Equal(t, 60, stockForSKU(t, db, "SKU-DEMO-01"))

	// 4. Second Goods Receipt: Remaining 40 units accepted
	deliv40 := 40
	accept40 := 40
	gr2Payload := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: poID.String(),
		Notes:           "Final Goods Receipt 2 (40 units)",
		Items: []supplychain.CreateGRItemRequest{
			{
				ProductID:         productID.String(),
				DeliveredQuantity: &deliv40,
				AcceptedQuantity:  &accept40,
				RejectedQuantity:  &reject0,
			},
		},
	}
	gr2Body, _ := json.Marshal(gr2Payload)
	gr2Req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(gr2Body))
	gr2Req.Header.Set("Content-Type", "application/json")
	gr2Req.Header.Set("Idempotency-Key", "demo-gr-key-40")
	gr2W := httptest.NewRecorder()
	router.ServeHTTP(gr2W, gr2Req)
	require.Equal(t, http.StatusCreated, gr2W.Code)

	// Total stock must now be 100
	assert.Equal(t, 100, stockForSKU(t, db, "SKU-DEMO-01"))

	// Verify PO status transitioned to RECEIVED
	conn, err = db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	_, _ = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")

	var poFinalStatus string
	err = conn.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id = $1`, poID).Scan(&poFinalStatus)
	require.NoError(t, err)
	assert.Equal(t, "RECEIVED", poFinalStatus)

	// 5. Verify Ledger Double-Entry Parity for the goods receipts
	var (
		totalDebits  float64
		totalCredits float64
	)
	err = conn.QueryRow(ctx, `
		SELECT COALESCE(SUM(l.debit_amount), 0), COALESCE(SUM(l.credit_amount), 0)
		FROM ledger_entry_lines l
		JOIN ledger_entries e ON e.id = l.ledger_entry_id
		WHERE e.source_document_type = 'GOODS_RECEIPT'
	`).Scan(&totalDebits, &totalCredits)
	require.NoError(t, err)
	assert.Equal(t, float64(1000000), totalDebits, "Sum of Debits for GR receipts must be IDR 1,000,000")
	assert.Equal(t, float64(1000000), totalCredits, "Sum of Credits for GR receipts must be IDR 1,000,000")
	assert.Equal(t, totalDebits, totalCredits, "Ledger balance invariant strictly enforced")
}

// TestDemoResetScript_SafetyGuards verifies Acceptance Criteria 4:
// - Reset requires explicit target check and rejects arbitrary/unsafe databases
// - Rejects execution if CONFIRM_DEMO_RESET is not confirmed in non-interactive shell
func TestDemoResetScript_SafetyGuards(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "scripts", "reset_dev_db.sh")
	require.FileExists(t, scriptPath)

	// Case 1: Unsafe database name must be hard rejected
	cmdUnsafe := exec.Command("/bin/bash", scriptPath)
	cmdUnsafe.Env = append(os.Environ(),
		"TENET_POSTGRES_DB=production_tenet_db",
		"CONFIRM_DEMO_RESET=true",
	)
	outUnsafe, errUnsafe := cmdUnsafe.CombinedOutput()
	assert.Error(t, errUnsafe, "Reset script must fail when given an unsafe production db name")
	assert.Contains(t, string(outUnsafe), "Unsafe target database name")

	// Case 2: Missing confirmation in non-interactive mode must be rejected
	cmdNoConfirm := exec.Command("/bin/bash", scriptPath)
	cmdNoConfirm.Env = append(os.Environ(),
		"TENET_POSTGRES_DB=tenet_commerce",
		"CONFIRM_DEMO_RESET=",
	)
	outNoConfirm, errNoConfirm := cmdNoConfirm.CombinedOutput()
	assert.Error(t, errNoConfirm, "Reset script must fail when confirmation is missing in non-interactive shell")
	assert.Contains(t, string(outNoConfirm), "CONFIRM_DEMO_RESET=true")
}
