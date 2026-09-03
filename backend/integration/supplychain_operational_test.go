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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/supplychain"
)

func TestSupplyChain_SupplierOperations_ListDetailUpdate(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// 1. Create a supplier
	code := fmt.Sprintf("SUP-OP-%d", time.Now().UnixNano())
	createPayload := supplychain.CreateSupplierRequest{
		Code:          code,
		CompanyName:   "CV Nusantara Jaya",
		ContactPerson: "Ahmad Fauzi",
		ContactEmail:  "fauzi@nusantarajaya.id",
		ContactPhone:  "08123456789",
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Success bool                  `json:"success"`
		Data    supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	supplierID := createResp.Data.ID
	assert.Equal(t, "CV Nusantara Jaya", createResp.Data.CompanyName)

	// 2. List suppliers and find created one
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/supply-chain/suppliers?is_active=true", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)

	var listResp struct {
		Success bool                    `json:"success"`
		Data    []supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listResp))
	var found bool
	for _, s := range listResp.Data {
		if s.ID == supplierID {
			found = true
			break
		}
	}
	assert.True(t, found, "created supplier must be present in active supplier list")

	// 3. Get supplier details
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/suppliers/%s", supplierID), nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	require.Equal(t, http.StatusOK, getW.Code)

	var getResp struct {
		Success bool                        `json:"success"`
		Data    supplychain.SupplierDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getW.Body.Bytes(), &getResp))
	assert.Equal(t, supplierID, getResp.Data.ID)
	assert.Equal(t, "Ahmad Fauzi", getResp.Data.ContactPerson)

	// 4. Update supplier contact and active status
	newName := "PT Nusantara Jaya Perkasa"
	newPhone := "08987654321"
	updatePayload := supplychain.UpdateSupplierRequest{
		CompanyName:  &newName,
		ContactPhone: &newPhone,
	}
	updBody, _ := json.Marshal(updatePayload)
	updReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/supply-chain/suppliers/%s", supplierID), bytes.NewReader(updBody))
	updReq.Header.Set("Content-Type", "application/json")
	updW := httptest.NewRecorder()
	router.ServeHTTP(updW, updReq)
	require.Equal(t, http.StatusOK, updW.Code)

	var updResp struct {
		Success bool                  `json:"success"`
		Data    supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(updW.Body.Bytes(), &updResp))
	assert.Equal(t, newName, updResp.Data.CompanyName)
	assert.Equal(t, newPhone, updResp.Data.ContactPhone)
}

func TestSupplyChain_CertificateRenewalAndRevoke(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// 1. Create a supplier
	code := fmt.Sprintf("SUP-CERT-%d", time.Now().UnixNano())
	createPayload := supplychain.CreateSupplierRequest{
		Code:          code,
		CompanyName:   "PT Berkah Unggas Sejahtera",
		ContactPerson: "Haji Sukri",
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Data supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	supplierID := createResp.Data.ID

	// 2. Register renewal certificate
	now := time.Now()
	validFrom := now.AddDate(0, -1, 0).Format("2006-01-02")
	expiryDate := now.AddDate(1, 0, 0).Format("2006-01-02")
	certPayload := supplychain.CreateComplianceCertRequest{
		CertType:          "HALAL_MUI",
		CertificateNumber: fmt.Sprintf("MUI-RENEW-%d", time.Now().UnixNano()),
		IssuingAuthority:  "LPPOM MUI",
		Scope:             "Poultry Processing",
		ValidFrom:         validFrom,
		ExpiryDate:        expiryDate,
	}
	certBody, _ := json.Marshal(certPayload)

	certReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/supply-chain/suppliers/%s/certificates", supplierID), bytes.NewReader(certBody))
	certReq.Header.Set("Content-Type", "application/json")
	certW := httptest.NewRecorder()
	router.ServeHTTP(certW, certReq)
	require.Equal(t, http.StatusCreated, certW.Code)

	var certResp struct {
		Data supplychain.ComplianceCertificate `json:"data"`
	}
	require.NoError(t, json.Unmarshal(certW.Body.Bytes(), &certResp))
	certID := certResp.Data.ID
	assert.Equal(t, "VALID", certResp.Data.ComputedStatus)

	// 3. List supplier certificates
	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/suppliers/%s/certificates", supplierID), nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)

	var listResp struct {
		Data []supplychain.ComplianceCertificate `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listResp))
	require.Len(t, listResp.Data, 1)
	assert.Equal(t, "VALID", listResp.Data[0].ComputedStatus)

	// 4. Revoke the certificate
	revokeReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/supply-chain/certificates/%s/revoke", certID), nil)
	revokeW := httptest.NewRecorder()
	router.ServeHTTP(revokeW, revokeReq)
	require.Equal(t, http.StatusOK, revokeW.Code)

	// 5. Verify certificate computed status is now EXPIRED
	listReq2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/suppliers/%s/certificates", supplierID), nil)
	listW2 := httptest.NewRecorder()
	router.ServeHTTP(listW2, listReq2)
	require.Equal(t, http.StatusOK, listW2.Code)

	var listResp2 struct {
		Data []supplychain.ComplianceCertificate `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listW2.Body.Bytes(), &listResp2))
	require.Len(t, listResp2.Data, 1)
	assert.Equal(t, "EXPIRED", listResp2.Data[0].ComputedStatus)
}

func TestSupplyChain_PurchaseOrder_ListDetailAndCancel(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// 1. Create a supplier with valid cert
	now := time.Now()
	createPayload := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-PO-%d", now.UnixNano()),
		CompanyName:   "PT Berkah Pangan Mandiri",
		ContactPerson: "Fajar",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("MUI-PO-%d", now.UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Meat Distribution",
			ValidFrom:         now.AddDate(0, -1, 0).Format("2006-01-02"),
			ExpiryDate:        now.AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Data supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	supplierID := createResp.Data.ID
	certID := createResp.Data.ComplianceCertificate.ID.String()

	// 2. Fetch a valid product ID
	var productID uuid.UUID
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		err = conn.QueryRow(ctx, "SELECT id FROM products WHERE is_active = true ORDER BY name LIMIT 1").Scan(&productID)
		require.NoError(t, err)
		conn.Release()
	}

	// 3. Create PO via POST /purchase-orders
	poPayload := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       supplierID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{ProductID: productID.String(), Quantity: 20, UnitCost: 25000},
		},
	}
	poBody, _ := json.Marshal(poPayload)
	poReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	poReq.Header.Set("Content-Type", "application/json")
	poReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-po-%d", now.UnixNano()))
	poW := httptest.NewRecorder()
	router.ServeHTTP(poW, poReq)
	require.Equal(t, http.StatusCreated, poW.Code)

	var poResp struct {
		Data supplychain.PurchaseOrder `json:"data"`
	}
	require.NoError(t, json.Unmarshal(poW.Body.Bytes(), &poResp))
	poID := poResp.Data.ID

	// 4. List POs filtered by ISSUED
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/supply-chain/purchase-orders?status=ISSUED", nil)
	listW := httptest.NewRecorder()
	router.ServeHTTP(listW, listReq)
	require.Equal(t, http.StatusOK, listW.Code)

	var listResp struct {
		Data []supplychain.PurchaseOrderSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listW.Body.Bytes(), &listResp))
	var poFound bool
	for _, p := range listResp.Data {
		if p.ID == poID {
			poFound = true
			assert.Equal(t, "ISSUED", p.Status)
			assert.Equal(t, float64(500000), p.TotalAmount)
			break
		}
	}
	assert.True(t, poFound, "created PO must be in list")

	// 5. Get PO details and check remaining balance
	detailReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/purchase-orders/%s", poID), nil)
	detailW := httptest.NewRecorder()
	router.ServeHTTP(detailW, detailReq)
	require.Equal(t, http.StatusOK, detailW.Code)

	var detailResp struct {
		Data supplychain.PurchaseOrderDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detailW.Body.Bytes(), &detailResp))
	require.Len(t, detailResp.Data.Items, 1)
	assert.Equal(t, 20, detailResp.Data.Items[0].Quantity)
	assert.Equal(t, 0, detailResp.Data.Items[0].ReceivedQuantity)
	assert.Equal(t, 20, detailResp.Data.Items[0].RemainingQuantity)

	// 6. Cancel PO
	cancelReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/supply-chain/purchase-orders/%s/cancel", poID), nil)
	cancelW := httptest.NewRecorder()
	router.ServeHTTP(cancelW, cancelReq)
	require.Equal(t, http.StatusOK, cancelW.Code)

	// 7. Verify status changed to CANCELLED
	detailReq2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/purchase-orders/%s", poID), nil)
	detailW2 := httptest.NewRecorder()
	router.ServeHTTP(detailW2, detailReq2)
	require.Equal(t, http.StatusOK, detailW2.Code)

	var detailResp2 struct {
		Data supplychain.PurchaseOrderDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detailW2.Body.Bytes(), &detailResp2))
	assert.Equal(t, "CANCELLED", detailResp2.Data.Status)
}

func TestSupplyChain_GoodsReceiptAndTraceability(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	router := setupSupplyChainTestRouter(t, db)

	// 1. Create supplier with Halal cert
	now := time.Now()
	createPayload := supplychain.CreateSupplierRequest{
		Code:          fmt.Sprintf("SUP-TRACE-%d", now.UnixNano()),
		CompanyName:   "PT Sumber Segar Halal",
		ContactPerson: "Budi",
		ComplianceCertificate: &supplychain.CreateComplianceCertRequest{
			CertType:          "HALAL_MUI",
			CertificateNumber: fmt.Sprintf("BPJPH-TRACE-%d", now.UnixNano()),
			IssuingAuthority:  "BPJPH",
			Scope:             "Fresh Broiler Chicken",
			ValidFrom:         now.AddDate(0, -1, 0).Format("2006-01-02"),
			ExpiryDate:        now.AddDate(1, 0, 0).Format("2006-01-02"),
		},
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/suppliers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		Data supplychain.Supplier `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	supplierID := createResp.Data.ID
	certID := createResp.Data.ComplianceCertificate.ID.String()

	// 2. Fetch a product
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
			WHERE p.sku = 'SKU-CHICKEN-01'
		`).Scan(&productID, &initialStock)
		require.NoError(t, err)
		conn.Release()
	}

	// 3. Create PO
	poPayload := supplychain.CreatePurchaseOrderRequest{
		SupplierID:       supplierID.String(),
		ComplianceCertID: &certID,
		Items: []supplychain.CreatePOItemRequest{
			{ProductID: productID.String(), Quantity: 15, UnitCost: 30000},
		},
	}
	poBody, _ := json.Marshal(poPayload)
	poReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/purchase-orders", bytes.NewReader(poBody))
	poReq.Header.Set("Content-Type", "application/json")
	poReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-po-trace-%d", now.UnixNano()))
	poW := httptest.NewRecorder()
	router.ServeHTTP(poW, poReq)
	require.Equal(t, http.StatusCreated, poW.Code)

	var poResp struct {
		Data supplychain.PurchaseOrder `json:"data"`
	}
	require.NoError(t, json.Unmarshal(poW.Body.Bytes(), &poResp))
	poID := poResp.Data.ID

	// 4. Create Goods Receipt via POST /goods-receipts
	grPayload := supplychain.CreateGoodsReceiptRequest{
		PurchaseOrderID: poID.String(),
		Notes:           "Received 15 fresh chickens in chilled containers",
		Items: []supplychain.CreateGRItemRequest{
			{ProductID: productID.String(), ReceivedQuantity: 15},
		},
	}
	grBody, _ := json.Marshal(grPayload)
	grReq := httptest.NewRequest(http.MethodPost, "/api/v1/supply-chain/goods-receipts", bytes.NewReader(grBody))
	grReq.Header.Set("Content-Type", "application/json")
	grReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-gr-trace-%d", now.UnixNano()))
	grW := httptest.NewRecorder()
	router.ServeHTTP(grW, grReq)
	require.Equal(t, http.StatusCreated, grW.Code)

	var grResp struct {
		Data supplychain.GoodsReceipt `json:"data"`
	}
	require.NoError(t, json.Unmarshal(grW.Body.Bytes(), &grResp))
	grID := grResp.Data.ID

	// 5. List Goods Receipts
	listGRReq := httptest.NewRequest(http.MethodGet, "/api/v1/supply-chain/goods-receipts", nil)
	listGRW := httptest.NewRecorder()
	router.ServeHTTP(listGRW, listGRReq)
	require.Equal(t, http.StatusOK, listGRW.Code)

	var listGRResp struct {
		Data []supplychain.GoodsReceiptSummary `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listGRW.Body.Bytes(), &listGRResp))
	var grFound bool
	for _, g := range listGRResp.Data {
		if g.ID == grID {
			grFound = true
			assert.Equal(t, float64(450000), g.TotalValuation)
			break
		}
	}
	assert.True(t, grFound, "created GR must appear in goods receipt list")

	// 6. Get Goods Receipt Detail
	grDetailReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/supply-chain/goods-receipts/%s", grID), nil)
	grDetailW := httptest.NewRecorder()
	router.ServeHTTP(grDetailW, grDetailReq)
	require.Equal(t, http.StatusOK, grDetailW.Code)

	var grDetailResp struct {
		Data supplychain.GoodsReceiptDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(grDetailW.Body.Bytes(), &grDetailResp))
	assert.Equal(t, grID, grDetailResp.Data.ID)
	assert.Equal(t, float64(450000), grDetailResp.Data.TotalValuation)
	require.NotNil(t, grDetailResp.Data.LedgerEntryNumber, "ledger journal entry must be cross-referenced")

	// 7. Test Document Traceability endpoint GET /traceability/product/:product_id
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
	assert.Equal(t, "SKU-CHICKEN-01", report.SKU)
	assert.Equal(t, initialStock+15, report.CurrentStock)

	// Verify supplier and certificate lineage
	require.NotEmpty(t, report.Suppliers)
	var sFound *supplychain.ProductTraceabilitySupplierInfo
	for i := range report.Suppliers {
		if report.Suppliers[i].CompanyName == "PT Sumber Segar Halal" {
			sFound = &report.Suppliers[i]
			break
		}
	}
	require.NotNil(t, sFound, "PT Sumber Segar Halal must be in traceability suppliers list")
	require.NotEmpty(t, sFound.Certificates)
	assert.Equal(t, "BPJPH", sFound.Certificates[0].IssuingAuthority)

	// Verify PO and GR history
	require.NotEmpty(t, report.PurchaseOrders)
	require.NotEmpty(t, report.GoodsReceipts)
	assert.Equal(t, 15, report.GoodsReceipts[0].ReceivedQuantity)
}
