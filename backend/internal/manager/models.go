package manager

import "time"

// DashboardSummary represents the aggregated analytics returned by GET /api/v1/manager/dashboard
type DashboardSummary struct {
	GeneratedAt      time.Time         `json:"generated_at"`
	SalesSummary     SalesSummary      `json:"sales_summary"`
	InventoryAlerts  InventoryAlerts   `json:"inventory_alerts"`
	ComplianceAlerts ComplianceAlerts  `json:"compliance_alerts"`
	FinancialSummary FinancialSummary  `json:"financial_summary"`
}

// SalesSummary contains aggregated order and revenue figures
type SalesSummary struct {
	TodayGrossSales    float64 `json:"today_gross_sales"`
	TodayNetSales      float64 `json:"today_net_sales"`
	TodayOrdersCount   int     `json:"today_orders_count"`
	AllTimeOrdersCount int     `json:"all_time_orders_count"`
	AverageOrderValue  float64 `json:"average_order_value"`
}

// InventoryAlerts summarizes stock depletion warnings
type InventoryAlerts struct {
	LowStockCount int            `json:"low_stock_count"`
	Items         []LowStockItem `json:"items"`
}

// LowStockItem represents an individual SKU falling below the reorder threshold
type LowStockItem struct {
	ProductID    string  `json:"product_id"`
	SKU          string  `json:"sku"`
	Name         string  `json:"name"`
	CategoryName string  `json:"category_name"`
	CurrentStock int     `json:"current_stock"`
	Threshold    int     `json:"threshold"`
	UnitPrice    float64 `json:"unit_price"`
}

// ComplianceAlerts lists suppliers with expiring or expired Halal certificates
type ComplianceAlerts struct {
	ExpiringCertificatesCount int                    `json:"expiring_certificates_count"`
	ExpiredCertificatesCount  int                    `json:"expired_certificates_count"`
	Items                     []CertificateAlertItem `json:"items"`
}

// CertificateAlertItem describes an impending or active Halal certificate lapse
type CertificateAlertItem struct {
	CertificateID     string    `json:"certificate_id"`
	SupplierID        string    `json:"supplier_id"`
	SupplierName      string    `json:"supplier_name"`
	CertificateNumber string    `json:"certificate_number"`
	IssuingAuthority  string    `json:"issuing_authority"`
	ExpiryDate        time.Time `json:"expiry_date"`
	DaysRemaining     int       `json:"days_remaining"`
	Status            string    `json:"status"` // "EXPIRING_SOON" or "EXPIRED"
}

// FinancialSummary provides high-level double-entry ledger status
type FinancialSummary struct {
	ActiveAccountsCount      int `json:"active_accounts_count"`
	TodayJournalEntriesCount int `json:"today_journal_entries_count"`
}
