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
	ReceivedQuantity int    `json:"received_quantity" binding:"required,min=0"`
}
