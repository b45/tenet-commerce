package entitlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/b45/tenet-commerce/backend/pkg/database"
)

var (
	ErrNoSubscription = errors.New("no active subscription found for tenant")
	ErrPlanNotFound   = errors.New("plan not found")
)

// Repository handles data access for subscription plans, grants, and tenant subscriptions.
type Repository struct {
	db *database.PostgresDB
}

// NewRepository initializes a new entitlement repository.
func NewRepository(db *database.PostgresDB) *Repository {
	return &Repository{db: db}
}

// GetSubscriptionByTenantID queries public.tenant_subscriptions with associated plan and features.
func (r *Repository) GetSubscriptionByTenantID(ctx context.Context, tenantID string) (*TenantSubscription, error) {
	subQuery := `
		SELECT 
			s.id, s.tenant_id, s.plan_id, s.status, 
			s.current_period_start, s.current_period_end, s.canceled_at,
			s.created_at, s.updated_at,
			p.id, p.code, p.name, p.version, p.is_active, p.created_at, p.updated_at
		FROM public.tenant_subscriptions s
		JOIN public.plans p ON s.plan_id = p.id
		WHERE s.tenant_id = $1
	`

	var sub TenantSubscription
	var plan Plan
	var statusStr string

	err := r.db.Pool.QueryRow(ctx, subQuery, tenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.PlanID, &statusStr,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CanceledAt,
		&sub.CreatedAt, &sub.UpdatedAt,
		&plan.ID, &plan.Code, &plan.Name, &plan.Version, &plan.IsActive, &plan.CreatedAt, &plan.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoSubscription
		}
		return nil, fmt.Errorf("failed to query tenant subscription: %w", err)
	}

	sub.Status = SubscriptionStatus(statusStr)
	plan.Features = make(map[string]PlanFeature)

	// Fetch plan features
	featQuery := `
		SELECT id, plan_id, feature_key, grant_type, quota_limit, is_enabled, created_at, updated_at
		FROM public.plan_features
		WHERE plan_id = $1
	`

	rows, err := r.db.Pool.Query(ctx, featQuery, plan.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query plan features: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pf PlanFeature
		var grantTypeStr string
		if err := rows.Scan(
			&pf.ID, &pf.PlanID, &pf.FeatureKey, &grantTypeStr,
			&pf.QuotaLimit, &pf.IsEnabled, &pf.CreatedAt, &pf.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan plan feature: %w", err)
		}
		pf.GrantType = GrantType(grantTypeStr)
		plan.Features[pf.FeatureKey] = pf
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading plan feature rows: %w", err)
	}

	sub.Plan = &plan
	return &sub, nil
}

// canceledAtScanHelper assists in scanning nullable canceled_at timestamp
func (s *TenantSubscription) canceledAtScanHelper() any {
	return &s.CanceledAt
}
