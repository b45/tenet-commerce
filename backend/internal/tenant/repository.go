package tenant

import (
	"context"
	"fmt"

	"github.com/b45/tenet-commerce/backend/pkg/database"
)

// Repository handles data access for the public tenant registry
type Repository struct {
	db *database.PostgresDB
}

// NewRepository creates a new tenant repository
func NewRepository(db *database.PostgresDB) *Repository {
	return &Repository{db: db}
}

// GetTenantBySlug queries the public.tenants table to find schema routing info
func (r *Repository) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	query := `
		SELECT id, slug, company_name, schema_name, status, created_at, updated_at
		FROM public.tenants
		WHERE slug = $1 AND status = 'ACTIVE'
	`
	
	var t Tenant
	err := r.db.Pool.QueryRow(ctx, query, slug).Scan(
		&t.ID,
		&t.Slug,
		&t.Name,
		&t.SchemaName,
		&t.Status,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get active tenant by slug: %w", err)
	}
	
	return &t, nil
}
