package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
)

const (
	// AdvisoryLockKey is the 64-bit advisory lock identifier for tenant migrations
	AdvisoryLockKey int64 = 0x74656e65746d6967 // 'tenetmig' in hex
)

// Migration represents a versioned DDL/DML script applied to tenant schemas
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// Checksum returns the SHA-256 hex digest of the normalized SQL script
func (m Migration) Checksum() string {
	h := sha256.New()
	_, _ = h.Write([]byte(strings.TrimSpace(m.SQL)))
	return hex.EncodeToString(h.Sum(nil))
}

// AppliedMigration represents an audit record in schema_migrations table
type AppliedMigration struct {
	Version     int       `json:"version"`
	Description string    `json:"description"`
	Checksum    string    `json:"checksum"`
	AppliedAt   time.Time `json:"applied_at"`
}

// Runner executes versioned migrations across trusted tenant schemas
type Runner struct {
	pool       *pgxpool.Pool
	tenantRepo *tenant.Repository
	migrations []Migration
	mu         sync.Mutex
}

// NewRunner creates a new migration Runner with verified migrations
func NewRunner(pool *pgxpool.Pool, tenantRepo *tenant.Repository, migrations []Migration) (*Runner, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool cannot be nil")
	}
	if tenantRepo == nil {
		return nil, fmt.Errorf("tenant repository cannot be nil")
	}

	// Validate migration versions are strictly sequential
	for i, m := range migrations {
		expectedVersion := i + 1
		if m.Version != expectedVersion {
			return nil, fmt.Errorf("non-sequential migration version at index %d: expected %d, got %d", i, expectedVersion, m.Version)
		}
		if strings.TrimSpace(m.SQL) == "" {
			return nil, fmt.Errorf("migration %d (%s) has empty SQL content", m.Version, m.Description)
		}
	}

	return &Runner{
		pool:       pool,
		tenantRepo: tenantRepo,
		migrations: migrations,
	}, nil
}

// EnsureSchemaMigrationsTable ensures the audit table exists within the given tenant schema
func EnsureSchemaMigrationsTable(ctx context.Context, tx pgx.Tx, schemaName string) error {
	sanitized := pgx.Identifier{schemaName}.Sanitize()
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.schema_migrations (
			version INTEGER PRIMARY KEY,
			description VARCHAR(255) NOT NULL,
			checksum VARCHAR(64) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`, sanitized)

	_, err := tx.Exec(ctx, query)
	return err
}

// MigrateTenant applies pending migrations to a single tenant schema within an isolated transaction
func (r *Runner) MigrateTenant(ctx context.Context, t *tenant.Tenant) error {
	if !tenant.IsValidSchemaName(t.SchemaName) {
		return fmt.Errorf("untrusted or invalid tenant schema name: %s", t.SchemaName)
	}

	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection for tenant %s: %w", t.Slug, err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. SET LOCAL search_path strictly to tenant schema (no public fallback)
	sanitized := pgx.Identifier{t.SchemaName}.Sanitize()
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %s;", sanitized)); err != nil {
		return fmt.Errorf("failed to set local search_path: %w", err)
	}

	// 2. Ensure schema_migrations table exists
	if err := EnsureSchemaMigrationsTable(ctx, tx, t.SchemaName); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	// 3. Query already applied migrations
	rows, err := tx.Query(ctx, fmt.Sprintf("SELECT version, checksum FROM %s.schema_migrations ORDER BY version ASC;", sanitized))
	if err != nil {
		return fmt.Errorf("failed to query existing migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]string)
	for rows.Next() {
		var v int
		var cs string
		if err := rows.Scan(&v, &cs); err != nil {
			return fmt.Errorf("failed scanning applied migration: %w", err)
		}
		applied[v] = cs
	}

	// 4. Validate checksums of previously applied migrations
	for _, m := range r.migrations {
		if existingChecksum, found := applied[m.Version]; found {
			if existingChecksum != m.Checksum() {
				return fmt.Errorf("migration %d checksum mismatch in tenant %s: expected %s, found in database %s",
					m.Version, t.Slug, m.Checksum(), existingChecksum)
			}
		}
	}

	// 5. Apply pending migrations sequentially
	for _, m := range r.migrations {
		if _, found := applied[m.Version]; found {
			continue // already applied
		}

		logger.Info("Applying migration to tenant",
			"tenant", t.Slug,
			"schema", t.SchemaName,
			"version", m.Version,
			"description", m.Description,
		)

		if _, err := tx.Exec(ctx, m.SQL); err != nil {
			return fmt.Errorf("failed executing migration %d (%s) on tenant %s: %w", m.Version, m.Description, t.Slug, err)
		}

		insertQuery := fmt.Sprintf(`
			INSERT INTO %s.schema_migrations (version, description, checksum, applied_at)
			VALUES ($1, $2, $3, NOW());
		`, sanitized)
		if _, err := tx.Exec(ctx, insertQuery, m.Version, m.Description, m.Checksum()); err != nil {
			return fmt.Errorf("failed recording migration %d on tenant %s: %w", m.Version, t.Slug, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tenant migration: %w", err)
	}

	return nil
}

// MigrateAllTenants runs all pending migrations across all active tenants in the registry.
// It acquires an exclusive advisory lock to prevent concurrent runner collisions.
func (r *Runner) MigrateAllTenants(ctx context.Context) (map[string]error, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire migration coordination connection: %w", err)
	}
	defer conn.Release()

	// Acquire global advisory lock
	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", AdvisoryLockKey).Scan(&locked); err != nil {
		return nil, fmt.Errorf("failed trying to acquire advisory lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("another migration runner is actively executing (advisory lock held)")
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", AdvisoryLockKey)
	}()

	tenants, err := r.tenantRepo.GetAllActiveTenants(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed fetching active tenants: %w", err)
	}

	results := make(map[string]error)
	for _, t := range tenants {
		tenantCopy := t
		err := r.MigrateTenant(ctx, &tenantCopy)
		if err != nil {
			results[t.Slug] = err
			logger.Error("Tenant migration failed",
				"tenant", t.Slug,
				"error", err.Error(),
			)
			// Return immediately so operator can identify the failing tenant without side-effects
			return results, fmt.Errorf("migration halted due to error on tenant %s: %w", t.Slug, err)
		}
		results[t.Slug] = nil
	}

	return results, nil
}
