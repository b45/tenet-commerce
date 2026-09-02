package manager

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles database operations for Manager analytics within a tenant schema
type Repository struct{}

// NewRepository initializes a new Manager repository
func NewRepository() *Repository {
	return &Repository{}
}

// GetSalesSummary queries gross/net sales and transaction counts for today and all time
func (r *Repository) GetSalesSummary(ctx context.Context, conn *pgxpool.Conn) (SalesSummary, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE THEN total_amount ELSE 0 END), 0) AS today_gross,
			COALESCE(SUM(CASE WHEN created_at >= CURRENT_DATE THEN (subtotal_amount - discount_amount) ELSE 0 END), 0) AS today_net,
			COALESCE(COUNT(CASE WHEN created_at >= CURRENT_DATE THEN 1 END), 0) AS today_count,
			COALESCE(COUNT(*), 0) AS all_time_count
		FROM transactions
		WHERE status = 'COMPLETED'
	`

	var s SalesSummary
	err := conn.QueryRow(ctx, query).Scan(
		&s.TodayGrossSales,
		&s.TodayNetSales,
		&s.TodayOrdersCount,
		&s.AllTimeOrdersCount,
	)
	if err != nil {
		return s, fmt.Errorf("failed to fetch sales summary: %w", err)
	}

	if s.TodayOrdersCount > 0 {
		s.AverageOrderValue = s.TodayGrossSales / float64(s.TodayOrdersCount)
	}

	return s, nil
}

// GetInventoryAlerts finds products with stock <= reorder_threshold
func (r *Repository) GetInventoryAlerts(ctx context.Context, conn *pgxpool.Conn, defaultThreshold int) (InventoryAlerts, error) {
	query := `
		SELECT
			p.id::text,
			p.sku,
			p.name,
			COALESCE(c.name, '') AS category_name,
			COALESCE(i.stock_quantity, 0) AS current_stock,
			COALESCE(i.reorder_threshold, $1) AS threshold,
			p.unit_price
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.is_active = true
		  AND COALESCE(i.stock_quantity, 0) <= COALESCE(i.reorder_threshold, $1)
		ORDER BY current_stock ASC, p.name ASC
		LIMIT 50
	`

	rows, err := conn.Query(ctx, query, defaultThreshold)
	if err != nil {
		return InventoryAlerts{}, fmt.Errorf("failed to query inventory alerts: %w", err)
	}
	defer rows.Close()

	items := make([]LowStockItem, 0)
	for rows.Next() {
		var item LowStockItem
		if err := rows.Scan(
			&item.ProductID,
			&item.SKU,
			&item.Name,
			&item.CategoryName,
			&item.CurrentStock,
			&item.Threshold,
			&item.UnitPrice,
		); err != nil {
			return InventoryAlerts{}, fmt.Errorf("failed to scan low stock item: %w", err)
		}
		items = append(items, item)
	}

	return InventoryAlerts{
		LowStockCount: len(items),
		Items:         items,
	}, nil
}

// GetComplianceAlerts checks certificates expiring within daysAhead days or already expired
func (r *Repository) GetComplianceAlerts(ctx context.Context, conn *pgxpool.Conn, daysAhead int) (ComplianceAlerts, error) {
	query := `
		SELECT
			cc.id::text,
			s.id::text,
			s.company_name,
			cc.certificate_number,
			cc.issuing_authority,
			cc.expiry_date,
			(cc.expiry_date - CURRENT_DATE) AS days_remaining
		FROM compliance_certificates cc
		JOIN suppliers s ON cc.supplier_id = s.id
		WHERE cc.expiry_date <= (CURRENT_DATE + $1::integer)
		ORDER BY cc.expiry_date ASC
		LIMIT 50
	`

	rows, err := conn.Query(ctx, query, daysAhead)
	if err != nil {
		return ComplianceAlerts{}, fmt.Errorf("failed to query compliance alerts: %w", err)
	}
	defer rows.Close()

	items := make([]CertificateAlertItem, 0)
	var expiringCount int
	var expiredCount int

	for rows.Next() {
		var item CertificateAlertItem
		if err := rows.Scan(
			&item.CertificateID,
			&item.SupplierID,
			&item.SupplierName,
			&item.CertificateNumber,
			&item.IssuingAuthority,
			&item.ExpiryDate,
			&item.DaysRemaining,
		); err != nil {
			return ComplianceAlerts{}, fmt.Errorf("failed to scan certificate alert item: %w", err)
		}

		if item.DaysRemaining < 0 {
			item.Status = "EXPIRED"
			expiredCount++
		} else {
			item.Status = "EXPIRING_SOON"
			expiringCount++
		}
		items = append(items, item)
	}

	return ComplianceAlerts{
		ExpiringCertificatesCount: expiringCount,
		ExpiredCertificatesCount:  expiredCount,
		Items:                     items,
	}, nil
}

// GetFinancialSummary counts active accounts and today's journal entries
func (r *Repository) GetFinancialSummary(ctx context.Context, conn *pgxpool.Conn) (FinancialSummary, error) {
	query := `
		SELECT
			COALESCE((SELECT COUNT(*) FROM ledger_accounts WHERE is_active = true), 0) AS active_accounts,
			COALESCE((SELECT COUNT(*) FROM ledger_entries WHERE entry_date = CURRENT_DATE), 0) AS today_entries
	`

	var fs FinancialSummary
	err := conn.QueryRow(ctx, query).Scan(
		&fs.ActiveAccountsCount,
		&fs.TodayJournalEntriesCount,
	)
	if err != nil {
		return fs, fmt.Errorf("failed to fetch financial summary: %w", err)
	}

	return fs, nil
}
