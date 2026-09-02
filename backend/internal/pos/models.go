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
	Items            []CartItemRequest `json:"items" binding:"required,min=1,dive"`
	PaymentMethod    string            `json:"payment_method" binding:"required,oneof=CASH SIMULATED_CARD QRIS"`
	DiscountAmount   float64           `json:"discount_amount" binding:"omitempty,gte=0"`
	CashTendered     *float64          `json:"cash_tendered,omitempty" binding:"omitempty,gte=0"`
	CustomerName     *string           `json:"customer_name,omitempty" binding:"omitempty,max=127"`
	Notes            *string           `json:"notes,omitempty"`
	PaymentReference *string           `json:"payment_reference,omitempty" binding:"omitempty,max=127"`
}

// Transaction represents the master record in the transactions table
type Transaction struct {
	ID                string     `json:"id"`
	TransactionNumber string     `json:"transaction_number"`
	IdempotencyKey    string     `json:"idempotency_key"`
	CashierID         string     `json:"cashier_id"`
	SubtotalAmount    float64    `json:"subtotal_amount"`
	TaxAmount         float64    `json:"tax_amount"`
	DiscountAmount    float64    `json:"discount_amount"`
	TotalAmount       float64    `json:"total_amount"`
	PaymentMethod     string     `json:"payment_method"`
	Status            string     `json:"status"`
	CustomerName      *string    `json:"customer_name,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	CashTendered      float64    `json:"cash_tendered"`
	ChangeAmount      float64    `json:"change_amount"`
	PaymentReference  *string    `json:"payment_reference,omitempty"`
	VoidReason        *string    `json:"void_reason,omitempty"`
	VoidedAt          *time.Time `json:"voided_at,omitempty"`
	VoidedBy          *string    `json:"voided_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
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
	CustomerName      *string           `json:"customer_name,omitempty"`
	Notes             *string           `json:"notes,omitempty"`
	CashTendered      float64           `json:"cash_tendered"`
	ChangeAmount      float64           `json:"change_amount"`
	PaymentReference  *string           `json:"payment_reference,omitempty"`
	Items             []TransactionItem `json:"items"`
	SubtotalAmount    float64           `json:"subtotal_amount"`
	TaxAmount         float64           `json:"tax_amount"`
	DiscountAmount    float64           `json:"discount_amount"`
	TotalAmount       float64           `json:"total_amount"`
	CreatedAt         time.Time         `json:"created_at"`
}

// VoidRequest represents the request payload for voiding/refunding an order
type VoidRequest struct {
	Reason string  `json:"reason" binding:"required,min=3,max=255"`
	Notes  *string `json:"notes,omitempty"`
}

// VoidResponse represents the response when an order is successfully voided
type VoidResponse struct {
	TransactionID     string    `json:"transaction_id"`
	TransactionNumber string    `json:"transaction_number"`
	Status            string    `json:"status"`
	VoidReason        string    `json:"void_reason"`
	VoidedAt          time.Time `json:"voided_at"`
	VoidedBy          string    `json:"voided_by"`
	ItemsRestocked    int       `json:"items_restocked"`
	TotalRefunded     float64   `json:"total_refunded"`
}

// OrderFilter contains parameters for querying order history
type OrderFilter struct {
	Limit         int
	Offset        int
	StartDate     string
	EndDate       string
	Status        string
	PaymentMethod string
	Search        string
}

// OrderSummary represents a single row in the order history list
type OrderSummary struct {
	ID                string    `json:"id"`
	TransactionNumber string    `json:"transaction_number"`
	CashierID         string    `json:"cashier_id"`
	TotalAmount       float64   `json:"total_amount"`
	PaymentMethod     string    `json:"payment_method"`
	Status            string    `json:"status"`
	CustomerName      *string   `json:"customer_name,omitempty"`
	TotalItems        int       `json:"total_items"`
	VoidReason        *string   `json:"void_reason,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

// OrderDetailResponse represents full order details including line items
type OrderDetailResponse struct {
	Transaction Transaction       `json:"transaction"`
	Items       []TransactionItem `json:"items"`
}

// PaymentSummary holds count and sum for each payment method
type PaymentSummary struct {
	Count       int     `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

// DailySummaryResponse represents the end-of-day / shift cashier report
type DailySummaryResponse struct {
	Date             string                    `json:"date"`
	CashierID        *string                   `json:"cashier_id,omitempty"`
	TotalOrders      int                       `json:"total_orders"`
	CompletedOrders  int                       `json:"completed_orders"`
	VoidedOrders     int                       `json:"voided_orders"`
	GrossSales       float64                   `json:"gross_sales"`
	Discounts        float64                   `json:"discounts"`
	NetSales         float64                   `json:"net_sales"`
	TotalCOGS        float64                   `json:"total_cogs"`
	GrossProfit      float64                   `json:"gross_profit"`
	PaymentBreakdown map[string]PaymentSummary `json:"payment_breakdown"`
}

// QRISConfig represents the tenant's QRIS payload and merchant identity
type QRISConfig struct {
	MerchantName string `json:"merchant_name"`
	NMID         string `json:"nmid"`
	QRString     string `json:"qr_string"`
	QRImageURL   string `json:"qr_image_url"`
}

