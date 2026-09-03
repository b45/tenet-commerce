package supplychain

import (
	"time"

	"github.com/google/uuid"
)

// TenantConfig represents the compliance settings for the tenant
type TenantConfig struct {
	ID          uuid.UUID              `json:"id"`
	ConfigKey   string                 `json:"config_key"`
	ConfigValue map[string]interface{} `json:"config_value"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Supplier represents a supplier entity
type Supplier struct {
	ID            uuid.UUID `json:"id"`
	Code          string    `json:"code"`
	CompanyName   string    `json:"company_name"`
	ContactPerson string    `json:"contact_person"`
	ContactEmail  string    `json:"contact_email"`
	ContactPhone  string    `json:"contact_phone"`
	IsActive              bool                   `json:"is_active"`
	CreatedAt             time.Time              `json:"created_at"`
	ComplianceCertificate *ComplianceCertificate `json:"compliance_certificate,omitempty"`
}

// ComplianceCertificate represents a certificate tied to a supplier
type ComplianceCertificate struct {
	ID                 uuid.UUID `json:"id"`
	SupplierID         uuid.UUID `json:"supplier_id"`
	CertType           string    `json:"cert_type"`
	CertificateNumber  string    `json:"certificate_number"`
	IssuingAuthority   string    `json:"issuing_authority"`
	Scope              string    `json:"scope"`
	ValidFrom          time.Time `json:"valid_from"`
	ExpiryDate         time.Time `json:"expiry_date"`
	DocumentURL        *string   `json:"document_url"`
	ComputedStatus     string    `json:"computed_status"` // Calculated on the fly (VALID, EXPIRING_SOON, EXPIRED)
	CreatedAt          time.Time `json:"created_at"`
}

// PurchaseOrder represents a PO sent to a supplier
type PurchaseOrder struct {
	ID                 uuid.UUID           `json:"id"`
	PONumber           string              `json:"po_number"`
	SupplierID         uuid.UUID           `json:"supplier_id"`
	ComplianceCertID   *uuid.UUID          `json:"compliance_cert_id"`
	TotalAmount        float64             `json:"total_amount"`
	Status             string              `json:"status"` // DRAFT, ISSUED, RECEIVED, CANCELLED
	IssuedDate         time.Time           `json:"issued_date"`
	CreatedAt          time.Time           `json:"created_at"`
	Items              []PurchaseOrderItem `json:"items,omitempty"`
}

// PurchaseOrderItem represents a line item in a PO
type PurchaseOrderItem struct {
	ID                uuid.UUID `json:"id"`
	PurchaseOrderID   uuid.UUID `json:"purchase_order_id"`
	ProductID         uuid.UUID `json:"product_id"`
	Quantity          int       `json:"quantity"`
	UnitCost          float64   `json:"unit_cost"`
	Subtotal          float64   `json:"subtotal"`
}

// GoodsReceipt represents a GR matching a PO
type GoodsReceipt struct {
	ID                uuid.UUID          `json:"id"`
	GRNumber          string             `json:"gr_number"`
	IdempotencyKey    string             `json:"idempotency_key"`
	PurchaseOrderID   uuid.UUID          `json:"purchase_order_id"`
	ReceivedBy        uuid.UUID          `json:"received_by"`
	ReceivedDate      time.Time          `json:"received_date"`
	Notes             string             `json:"notes"`
	CreatedAt         time.Time          `json:"created_at"`
	Items             []GoodsReceiptItem `json:"items,omitempty"`
}

// GoodsReceiptItem represents a line item in a GR
type GoodsReceiptItem struct {
	ID                uuid.UUID `json:"id"`
	GoodsReceiptID    uuid.UUID `json:"goods_receipt_id"`
	ProductID         uuid.UUID `json:"product_id"`
	ReceivedQuantity  int       `json:"received_quantity"`
}

// -----------------------------------------------------------------------------
// Request / Response Payloads
// -----------------------------------------------------------------------------

type CreateSupplierRequest struct {
	Code                   string                           `json:"code" binding:"required"`
	CompanyName            string                           `json:"company_name" binding:"required"`
	ContactPerson          string                           `json:"contact_person"`
	ContactEmail           string                           `json:"contact_email"`
	ContactPhone           string                           `json:"contact_phone"`
	ComplianceCertificate  *CreateComplianceCertRequest     `json:"compliance_certificate,omitempty"`
}

type CreateComplianceCertRequest struct {
	CertType          string  `json:"cert_type" binding:"required"`
	CertificateNumber string  `json:"certificate_number" binding:"required"`
	IssuingAuthority  string  `json:"issuing_authority" binding:"required"`
	Scope             string  `json:"scope" binding:"required"`
	ValidFrom         string  `json:"valid_from" binding:"required"` // Format YYYY-MM-DD
	ExpiryDate        string  `json:"expiry_date" binding:"required"` // Format YYYY-MM-DD
	DocumentURL       *string `json:"document_url"`
}

type CreatePurchaseOrderRequest struct {
	SupplierID         string                     `json:"supplier_id" binding:"required,uuid"`
	ComplianceCertID   *string                    `json:"compliance_cert_id" binding:"omitempty,uuid"`
	Items              []CreatePOItemRequest      `json:"items" binding:"required,min=1,dive"`
}

type CreatePOItemRequest struct {
	ProductID string  `json:"product_id" binding:"required,uuid"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	UnitCost  float64 `json:"unit_cost" binding:"required,min=0"`
}

type CreateGoodsReceiptRequest struct {
	PurchaseOrderID string                     `json:"purchase_order_id" binding:"required,uuid"`
	Notes           string                     `json:"notes"`
	Items           []CreateGRItemRequest      `json:"items" binding:"required,min=1,dive"`
}

type CreateGRItemRequest struct {
	ProductID        string `json:"product_id" binding:"required,uuid"`
	ReceivedQuantity int    `json:"received_quantity" binding:"required,min=1"`
}

// UpdateSupplierRequest represents payload for modifying supplier information
type UpdateSupplierRequest struct {
	CompanyName   *string `json:"company_name" binding:"omitempty,min=2,max=255"`
	ContactPerson *string `json:"contact_person" binding:"omitempty,max=255"`
	ContactEmail  *string `json:"contact_email" binding:"omitempty,email"`
	ContactPhone  *string `json:"contact_phone" binding:"omitempty,max=64"`
	IsActive      *bool   `json:"is_active"`
}

// RevokeComplianceCertRequest represents payload for revoking an active certificate
type RevokeComplianceCertRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=255"`
}

// CancelPurchaseOrderRequest represents payload for cancelling a purchase order
type CancelPurchaseOrderRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=255"`
}

// SupplierDetail represents a supplier with its full certificate history
type SupplierDetail struct {
	Supplier
	Certificates []ComplianceCertificate `json:"certificates"`
}

// PurchaseOrderSummary represents a lightweight summary for listing POs
type PurchaseOrderSummary struct {
	ID               uuid.UUID  `json:"id"`
	PONumber         string     `json:"po_number"`
	SupplierID       uuid.UUID  `json:"supplier_id"`
	SupplierName     string     `json:"supplier_name"`
	ComplianceCertID *uuid.UUID `json:"compliance_cert_id,omitempty"`
	TotalAmount      float64    `json:"total_amount"`
	Status           string     `json:"status"`
	IssuedDate       time.Time  `json:"issued_date"`
	CreatedAt        time.Time  `json:"created_at"`
	ItemCount        int        `json:"item_count"`
}

// PurchaseOrderDetailLine represents an item line within a PO detail
type PurchaseOrderDetailLine struct {
	ID                uuid.UUID `json:"id"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductName       string    `json:"product_name"`
	ProductSKU        string    `json:"product_sku"`
	Quantity          int       `json:"quantity"`
	ReceivedQuantity  int       `json:"received_quantity"`
	RemainingQuantity int       `json:"remaining_quantity"`
	UnitCost          float64   `json:"unit_cost"`
	Subtotal          float64   `json:"subtotal"`
}

// GoodsReceiptSummary represents a lightweight summary of a Goods Receipt
type GoodsReceiptSummary struct {
	ID              uuid.UUID `json:"id"`
	GRNumber        string    `json:"gr_number"`
	PurchaseOrderID uuid.UUID `json:"purchase_order_id"`
	PONumber        string    `json:"po_number"`
	SupplierName    string    `json:"supplier_name"`
	ReceivedBy      uuid.UUID `json:"received_by"`
	ReceivedDate    time.Time `json:"received_date"`
	TotalItems      int       `json:"total_items"`
	TotalValuation  float64   `json:"total_valuation"`
	CreatedAt       time.Time `json:"created_at"`
}

// PurchaseOrderDetail represents full PO details including lines, remaining balance, and receipts
type PurchaseOrderDetail struct {
	ID               uuid.UUID                 `json:"id"`
	PONumber         string                    `json:"po_number"`
	SupplierID       uuid.UUID                 `json:"supplier_id"`
	SupplierName     string                    `json:"supplier_name"`
	ComplianceCertID *uuid.UUID                `json:"compliance_cert_id,omitempty"`
	TotalAmount      float64                   `json:"total_amount"`
	Status           string                    `json:"status"`
	IssuedDate       time.Time                 `json:"issued_date"`
	CreatedAt        time.Time                 `json:"created_at"`
	Items            []PurchaseOrderDetailLine `json:"items"`
	GoodsReceipts    []GoodsReceiptSummary     `json:"goods_receipts"`
}

// GoodsReceiptDetailItem represents a line item in GoodsReceiptDetail
type GoodsReceiptDetailItem struct {
	ID                uuid.UUID `json:"id"`
	ProductID         uuid.UUID `json:"product_id"`
	ProductName       string    `json:"product_name"`
	ProductSKU        string    `json:"product_sku"`
	ReceivedQuantity  int       `json:"received_quantity"`
	UnitCost          float64   `json:"unit_cost"`
	SubtotalValuation float64   `json:"subtotal_valuation"`
}

// GoodsReceiptDetail represents full GR details with line items and accounting cross-reference
type GoodsReceiptDetail struct {
	ID                uuid.UUID                `json:"id"`
	GRNumber          string                   `json:"gr_number"`
	IdempotencyKey    string                   `json:"idempotency_key"`
	PurchaseOrderID   uuid.UUID                `json:"purchase_order_id"`
	PONumber          string                   `json:"po_number"`
	SupplierID        uuid.UUID                `json:"supplier_id"`
	SupplierName      string                   `json:"supplier_name"`
	ReceivedBy        uuid.UUID                `json:"received_by"`
	ReceivedDate      time.Time                `json:"received_date"`
	Notes             string                   `json:"notes"`
	CreatedAt         time.Time                `json:"created_at"`
	TotalValuation    float64                  `json:"total_valuation"`
	LedgerEntryNumber *string                  `json:"ledger_entry_number,omitempty"`
	Items             []GoodsReceiptDetailItem `json:"items"`
}

// ProductTraceabilitySupplierInfo represents a supplier that provided the product
type ProductTraceabilitySupplierInfo struct {
	SupplierID   uuid.UUID               `json:"supplier_id"`
	CompanyName  string                  `json:"company_name"`
	Code         string                  `json:"code"`
	Certificates []ComplianceCertificate `json:"certificates"`
}

// ProductTraceabilityPOInfo represents a PO that included the product
type ProductTraceabilityPOInfo struct {
	POID            uuid.UUID `json:"po_id"`
	PONumber        string    `json:"po_number"`
	Status          string    `json:"status"`
	IssuedDate      time.Time `json:"issued_date"`
	OrderedQuantity int       `json:"ordered_quantity"`
	UnitCost        float64   `json:"unit_cost"`
}

// ProductTraceabilityGRInfo represents a receipt for the product
type ProductTraceabilityGRInfo struct {
	GRID             uuid.UUID `json:"gr_id"`
	GRNumber         string    `json:"gr_number"`
	PONumber         string    `json:"po_number"`
	ReceivedDate     time.Time `json:"received_date"`
	ReceivedQuantity int       `json:"received_quantity"`
	ReceivedBy       uuid.UUID `json:"received_by"`
}

// ProductTraceabilityReport represents end-to-end document lineage for a product
type ProductTraceabilityReport struct {
	ProductID      uuid.UUID                         `json:"product_id"`
	SKU            string                            `json:"sku"`
	Name           string                            `json:"name"`
	CurrentStock   int                               `json:"current_stock"`
	Suppliers      []ProductTraceabilitySupplierInfo `json:"suppliers"`
	PurchaseOrders []ProductTraceabilityPOInfo       `json:"purchase_orders"`
	GoodsReceipts  []ProductTraceabilityGRInfo       `json:"goods_receipts"`
}
