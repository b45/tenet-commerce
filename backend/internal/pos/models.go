package pos

import (
	"time"
)

// Product represents a product catalog item in the tenant schema
type Product struct {
	ID               string    `json:"id"`
	CategoryID       *string   `json:"category_id,omitempty"`
	CategoryName     string    `json:"category_name,omitempty"`
	SKU              string    `json:"sku"`
	Barcode          *string   `json:"barcode,omitempty"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	UnitPrice        float64   `json:"unit_price"`
	CostPrice        float64   `json:"cost_price"`
	StockQuantity    int       `json:"stock_quantity"`
	ComplianceTags   []string  `json:"compliance_tags"`
	IsHalalCertified bool      `json:"is_halal_certified"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CartItemRequest represents an individual item in the checkout payload
type CartItemRequest struct {
	SKU      string `json:"sku" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
}

// CheckoutRequest represents the payload for POST /api/v1/pos/checkout
type CheckoutRequest struct {
	Items          []CartItemRequest `json:"items" binding:"required,min=1,dive"`
	PaymentMethod  string            `json:"payment_method" binding:"required,oneof=CASH SIMULATED_CARD QRIS"`
	DiscountAmount float64           `json:"discount_amount" binding:"omitempty,gte=0"`
}

// Transaction represents the master record in the transactions table
type Transaction struct {
	ID                string    `json:"id"`
	TransactionNumber string    `json:"transaction_number"`
	IdempotencyKey    string    `json:"idempotency_key"`
	CashierID         string    `json:"cashier_id"`
	SubtotalAmount    float64   `json:"subtotal_amount"`
	TaxAmount         float64   `json:"tax_amount"`
	DiscountAmount    float64   `json:"discount_amount"`
	TotalAmount       float64   `json:"total_amount"`
	PaymentMethod     string    `json:"payment_method"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

// TransactionItem represents a line item in transaction_items table
type TransactionItem struct {
	ID            string  `json:"id"`
	TransactionID string  `json:"transaction_id"`
	ProductID     string  `json:"product_id"`
	SKU           string  `json:"sku"`
	Name          string  `json:"name"`
	Quantity      int     `json:"quantity"`
	UnitPrice     float64 `json:"unit_price"`
	CostPrice     float64 `json:"cost_price"`
	Subtotal      float64 `json:"subtotal"`
}

// CheckoutResponse represents the detailed receipt returned to the POS client
type CheckoutResponse struct {
	TransactionID     string            `json:"transaction_id"`
	TransactionNumber string            `json:"transaction_number"`
	IdempotencyKey    string            `json:"idempotency_key"`
	CashierID         string            `json:"cashier_id"`
	PaymentMethod     string            `json:"payment_method"`
	Status            string            `json:"status"`
	Items             []TransactionItem `json:"items"`
	SubtotalAmount    float64           `json:"subtotal_amount"`
	TaxAmount         float64           `json:"tax_amount"`
	DiscountAmount    float64           `json:"discount_amount"`
	TotalAmount       float64           `json:"total_amount"`
	CreatedAt         time.Time         `json:"created_at"`
}
