package supplychain

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
)

// This test requires a running database on localhost:5432 with the tenet_commerce db
// To run: make test

func TestSupplyChain_ConfigurableCompliance(t *testing.T) {
	// 1. Setup DB connection
	// Assuming test DB is available as per Makefile
	ctx := context.Background()
	connString := "postgres://postgres:postgres@localhost:5432/tenet_commerce?sslmode=disable"
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Skip("Database not available, skipping integration test", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Database not pingable, skipping integration test", err)
	}

	repo := NewRepository()
	ledgerService := ledger.NewService(ledger.NewRepository())
	svc := NewService(repo, ledgerService)

	// We use two different tenants seeded in init_dev_db.sql
	// tenant_al_barakah_mart -> strict_compliance_mode = true
	// tenant_darussalam_store -> strict_compliance_mode = false

	t.Run("tenant_darussalam_store (strict mode OFF)", func(t *testing.T) {
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()

		// Set search path for this test connection
		_, err = conn.Exec(ctx, "SET search_path TO tenant_darussalam_store")
		require.NoError(t, err)

		// 1. Create Supplier WITHOUT Certificate
		reqSupplier := &CreateSupplierRequest{
			Code:          "SUP-DS-01",
			CompanyName:   "Supplier Without Cert",
			ContactPerson: "Budi",
		}
		supplier, err := svc.CreateSupplier(ctx, conn, reqSupplier)
		require.NoError(t, err)
		require.NotNil(t, supplier)

		// 2. Create PO referencing supplier (no cert) - Should SUCCEED
		reqPO := &CreatePurchaseOrderRequest{
			SupplierID: supplier.ID.String(),
			Items: []CreatePOItemRequest{
				{ProductID: "20000000-0000-0000-0000-000000000001", Quantity: 10, UnitCost: 10000},
			},
		}
		po, err := svc.CreatePurchaseOrder(ctx, conn, reqPO)
		require.NoError(t, err)
		assert.Equal(t, "ISSUED", po.Status)

		// 3. Create GR - Should SUCCEED
		reqGR := &CreateGoodsReceiptRequest{
			PurchaseOrderID: po.ID.String(),
			Items: []CreateGRItemRequest{
				{ProductID: "20000000-0000-0000-0000-000000000001", ReceivedQuantity: 10},
			},
		}
		gr, err := svc.CreateGoodsReceipt(ctx, conn, uuid.New(), "test-gr-key-"+uuid.NewString(), reqGR)
		require.NoError(t, err)
		require.NotNil(t, gr)
	})

	t.Run("tenant_al_barakah_mart (strict mode ON)", func(t *testing.T) {
		conn, err := pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()

		// Set search path for this test connection
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart")
		require.NoError(t, err)

		// 1. Try to create PO without cert - Should FAIL with ErrComplianceCertRequired
		reqSupplier := &CreateSupplierRequest{
			Code:          "SUP-AB-01",
			CompanyName:   "Supplier Try Bypass",
		}
		supplierNoCert, err := svc.CreateSupplier(ctx, conn, reqSupplier)
		require.NoError(t, err)

		reqPO := &CreatePurchaseOrderRequest{
			SupplierID: supplierNoCert.ID.String(),
			Items: []CreatePOItemRequest{
				{ProductID: "10000000-0000-0000-0000-000000000001", Quantity: 10, UnitCost: 10000},
			},
		}
		_, err = svc.CreatePurchaseOrder(ctx, conn, reqPO)
		require.ErrorIs(t, err, ErrComplianceCertRequired)

		// 2. Create Supplier WITH Valid Cert
		validDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
		expiryDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02") // 1 year later
		reqSupplierValid := &CreateSupplierRequest{
			Code:          "SUP-AB-02",
			CompanyName:   "Supplier Valid Cert",
			ComplianceCertificate: &CreateComplianceCertRequest{
				CertType:          "HALAL_MUI",
				CertificateNumber: "CERT-002",
				IssuingAuthority:  "MUI",
				Scope:             "Meat",
				ValidFrom:         validDate,
				ExpiryDate:        expiryDate,
			},
		}
		supplierValid, err := svc.CreateSupplier(ctx, conn, reqSupplierValid)
		require.NoError(t, err)

		// We need to fetch the cert ID from DB to pass to PO request
		var validCertID string
		err = conn.QueryRow(ctx, "SELECT id FROM compliance_certificates WHERE supplier_id = $1", supplierValid.ID).Scan(&validCertID)
		require.NoError(t, err)

		// 3. Create PO with Valid Cert - Should SUCCEED
		reqPOValid := &CreatePurchaseOrderRequest{
			SupplierID:       supplierValid.ID.String(),
			ComplianceCertID: &validCertID,
			Items: []CreatePOItemRequest{
				{ProductID: "10000000-0000-0000-0000-000000000001", Quantity: 10, UnitCost: 10000},
			},
		}
		poValid, err := svc.CreatePurchaseOrder(ctx, conn, reqPOValid)
		require.NoError(t, err)
		assert.Equal(t, "ISSUED", poValid.Status)


		// 4. Create Supplier WITH Expired Cert
		expiredDate := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
		reqSupplierExpired := &CreateSupplierRequest{
			Code:          "SUP-AB-03",
			CompanyName:   "Supplier Expired Cert",
			ComplianceCertificate: &CreateComplianceCertRequest{
				CertType:          "HALAL_MUI",
				CertificateNumber: "CERT-EXPIRED",
				IssuingAuthority:  "MUI",
				Scope:             "Meat",
				ValidFrom:         time.Now().AddDate(-2, 0, 0).Format("2006-01-02"),
				ExpiryDate:        expiredDate,
			},
		}
		supplierExpired, err := svc.CreateSupplier(ctx, conn, reqSupplierExpired)
		require.NoError(t, err)

		var expiredCertID string
		err = conn.QueryRow(ctx, "SELECT id FROM compliance_certificates WHERE supplier_id = $1", supplierExpired.ID).Scan(&expiredCertID)
		require.NoError(t, err)

		// 5. Create PO with Expired Cert - Should FAIL with ErrComplianceCertExpired
		reqPOExpired := &CreatePurchaseOrderRequest{
			SupplierID:       supplierExpired.ID.String(),
			ComplianceCertID: &expiredCertID,
			Items: []CreatePOItemRequest{
				{ProductID: "10000000-0000-0000-0000-000000000001", Quantity: 10, UnitCost: 10000},
			},
		}
		_, err = svc.CreatePurchaseOrder(ctx, conn, reqPOExpired)
		require.ErrorIs(t, err, ErrComplianceCertExpired)
	})
}

func TestReconcileReceiptItems(t *testing.T) {
	prod1 := uuid.New()
	prod2 := uuid.New()
	unknownProd := uuid.New()

	poItems := []PurchaseOrderItem{
		{
			ID:              uuid.New(),
			PurchaseOrderID: uuid.New(),
			ProductID:       prod1,
			Quantity:        10,
			UnitCost:        25000,
			Subtotal:        250000,
		},
		{
			ID:              uuid.New(),
			PurchaseOrderID: uuid.New(),
			ProductID:       prod2,
			Quantity:        5,
			UnitCost:        50000,
			Subtotal:        250000,
		},
	}

	t.Run("empty receipt items rejected", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		_, _, err := reconcileReceiptItems(gr, nil, poItems, map[uuid.UUID]int{})
		require.ErrorIs(t, err, ErrEmptyReceipt)
	})

	t.Run("invalid product UUID format rejected", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		requested := []CreateGRItemRequest{
			{ProductID: "invalid-uuid", ReceivedQuantity: 5},
		}
		_, _, err := reconcileReceiptItems(gr, requested, poItems, map[uuid.UUID]int{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse goods receipt product id")
	})

	t.Run("duplicate product in requested items rejected", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		requested := []CreateGRItemRequest{
			{ProductID: prod1.String(), ReceivedQuantity: 3},
			{ProductID: prod1.String(), ReceivedQuantity: 2},
		}
		_, _, err := reconcileReceiptItems(gr, requested, poItems, map[uuid.UUID]int{})
		require.ErrorIs(t, err, ErrDuplicateReceiptItem)
	})

	t.Run("product not on purchase order rejected", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		requested := []CreateGRItemRequest{
			{ProductID: unknownProd.String(), ReceivedQuantity: 2},
		}
		_, _, err := reconcileReceiptItems(gr, requested, poItems, map[uuid.UUID]int{})
		require.ErrorIs(t, err, ErrReceiptItemNotOnPO)
	})

	t.Run("zero or negative quantity rejected", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		requested := []CreateGRItemRequest{
			{ProductID: prod1.String(), ReceivedQuantity: 0},
		}
		_, _, err := reconcileReceiptItems(gr, requested, poItems, map[uuid.UUID]int{})
		require.ErrorIs(t, err, ErrReceiptQuantityExceeds)
	})

	t.Run("quantity exceeding remaining outstanding quantity rejected", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		// Already received 7 of 10. Outstanding is 3. Attempt to receive 4.
		received := map[uuid.UUID]int{prod1: 7}
		requested := []CreateGRItemRequest{
			{ProductID: prod1.String(), ReceivedQuantity: 4},
		}
		_, _, err := reconcileReceiptItems(gr, requested, poItems, received)
		require.ErrorIs(t, err, ErrReceiptQuantityExceeds)
	})

	t.Run("partial receipt returns fullyReceived false and correct valuation", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		received := map[uuid.UUID]int{}
		requested := []CreateGRItemRequest{
			{ProductID: prod1.String(), ReceivedQuantity: 4},
		}
		val, fullyReceived, err := reconcileReceiptItems(gr, requested, poItems, received)
		require.NoError(t, err)
		assert.False(t, fullyReceived)
		assert.Equal(t, 4*25000.0, val)
		require.Len(t, gr.Items, 1)
		assert.Equal(t, prod1, gr.Items[0].ProductID)
		assert.Equal(t, 4, gr.Items[0].ReceivedQuantity)
	})

	t.Run("full receipt completing all lines returns fullyReceived true", func(t *testing.T) {
		gr := &GoodsReceipt{ID: uuid.New()}
		// Already received 4 of prod1 and 2 of prod2
		received := map[uuid.UUID]int{
			prod1: 4,
			prod2: 2,
		}
		// Now receiving remaining 6 of prod1 and 3 of prod2
		requested := []CreateGRItemRequest{
			{ProductID: prod1.String(), ReceivedQuantity: 6},
			{ProductID: prod2.String(), ReceivedQuantity: 3},
		}
		val, fullyReceived, err := reconcileReceiptItems(gr, requested, poItems, received)
		require.NoError(t, err)
		assert.True(t, fullyReceived)
		expectedVal := (6 * 25000.0) + (3 * 50000.0)
		assert.Equal(t, expectedVal, val)
		require.Len(t, gr.Items, 2)
	})
}
