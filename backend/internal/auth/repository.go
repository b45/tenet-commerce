package auth

import (
	"context"
	"fmt"

	"github.com/b45/tenet-commerce/backend/pkg/database"
)

// Repository handles database operations for authentication against the public schema
type Repository struct {
	db *database.PostgresDB
}

// NewRepository initializes a new auth repository
func NewRepository(db *database.PostgresDB) *Repository {
	return &Repository{db: db}
}

// GetUserByEmailAndTenantSlug finds an active user within an active tenant
func (r *Repository) GetUserByEmailAndTenantSlug(ctx context.Context, tenantSlug, email string) (*User, error) {
	query := `
		SELECT 
			u.id, 
			u.tenant_id, 
			t.slug as tenant_slug, 
			u.email, 
			u.password_hash, 
			u.full_name, 
			u.role, 
			u.is_active, 
			u.created_at, 
			u.updated_at
		FROM public.users u
		INNER JOIN public.tenants t ON u.tenant_id = t.id
		WHERE t.slug = $1 
		  AND u.email = $2 
		  AND u.is_active = TRUE 
		  AND t.status = 'ACTIVE'
	`

	var u User
	err := r.db.Pool.QueryRow(ctx, query, tenantSlug, email).Scan(
		&u.ID,
		&u.TenantID,
		&u.TenantSlug,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found or inactive: %w", err)
	}

	return &u, nil
}

// GetUserByID fetches a user by their UUID
func (r *Repository) GetUserByID(ctx context.Context, userID string) (*User, error) {
	query := `
		SELECT 
			u.id, 
			u.tenant_id, 
			t.slug as tenant_slug, 
			u.email, 
			u.password_hash, 
			u.full_name, 
			u.role, 
			u.is_active, 
			u.created_at, 
			u.updated_at
		FROM public.users u
		INNER JOIN public.tenants t ON u.tenant_id = t.id
		WHERE u.id = $1 
		  AND u.is_active = TRUE 
		  AND t.status = 'ACTIVE'
	`

	var u User
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&u.ID,
		&u.TenantID,
		&u.TenantSlug,
		&u.Email,
		&u.PasswordHash,
		&u.FullName,
		&u.Role,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &u, nil
}
