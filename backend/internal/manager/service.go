package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Service encapsulates business logic for the manager dashboard
type Service struct {
	repo *Repository
}

// NewService initializes a new Manager domain service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetDashboardSummary aggregates all metrics across sales, inventory, compliance, and ledger
func (s *Service) GetDashboardSummary(ctx context.Context, conn *pgxpool.Conn) (*DashboardSummary, error) {
	// 1. Sales Summary
	sales, err := s.repo.GetSalesSummary(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("error aggregating sales: %w", err)
	}

	// 2. Inventory Alerts (reorder threshold default = 10)
	inventory, err := s.repo.GetInventoryAlerts(ctx, conn, 10)
	if err != nil {
		return nil, fmt.Errorf("error aggregating inventory alerts: %w", err)
	}

	// 3. Compliance Alerts (check within next 30 days)
	compliance, err := s.repo.GetComplianceAlerts(ctx, conn, 30)
	if err != nil {
		return nil, fmt.Errorf("error aggregating compliance alerts: %w", err)
	}

	// 4. Financial Summary
	finance, err := s.repo.GetFinancialSummary(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("error aggregating financial summary: %w", err)
	}

	return &DashboardSummary{
		GeneratedAt:      time.Now().UTC(),
		SalesSummary:     sales,
		InventoryAlerts:  inventory,
		ComplianceAlerts: compliance,
		FinancialSummary: finance,
	}, nil
}
