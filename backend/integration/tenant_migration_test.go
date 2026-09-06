package integration_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/database/migration"
)

func cleanupTenantMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	dropFn := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `
			DROP TABLE IF EXISTS tenant_al_barakah_mart.schema_migrations CASCADE;
			DROP TABLE IF EXISTS tenant_darussalam_store.schema_migrations CASCADE;
			DROP TABLE IF EXISTS tenant_al_barakah_mart.operational_logs CASCADE;
			DROP TABLE IF EXISTS tenant_darussalam_store.operational_logs CASCADE;
			DROP TABLE IF EXISTS tenant_al_barakah_mart.test_chk CASCADE;
			DROP TABLE IF EXISTS tenant_al_barakah_mart.tenant_valid_tbl CASCADE;
			DROP TABLE IF EXISTS tenant_darussalam_store.tenant_valid_tbl CASCADE;
			DROP TABLE IF EXISTS tenant_al_barakah_mart.test_concurrent_run CASCADE;
			DROP TABLE IF EXISTS tenant_darussalam_store.test_concurrent_run CASCADE;
			ALTER TABLE tenant_al_barakah_mart.products DROP COLUMN IF EXISTS audit_tag;
			ALTER TABLE tenant_al_barakah_mart.suppliers DROP COLUMN IF EXISTS metadata;
			ALTER TABLE tenant_darussalam_store.suppliers DROP COLUMN IF EXISTS metadata;
		`)
	}
	dropFn()
	t.Cleanup(dropFn)
}

func TestTenantMigration_CleanInstallAndRerunIdempotency(t *testing.T) {
	db := newTestDatabase(t)
	cleanupTenantMigrations(t, db.Pool)
	tenantRepo := tenant.NewRepository(db)

	ctx := context.Background()
	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)

	migrations := []migration.Migration{
		{
			Version:     1,
			Description: "Add audit tag column to products",
			SQL:         "ALTER TABLE products ADD COLUMN IF NOT EXISTS audit_tag VARCHAR(50);",
		},
		{
			Version:     2,
			Description: "Create tenant operational log",
			SQL: `CREATE TABLE IF NOT EXISTS operational_logs (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				event_name VARCHAR(100) NOT NULL,
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			);`,
		},
	}

	runner, err := migration.NewRunner(db.Pool, tenantRepo, migrations)
	require.NoError(t, err)

	// 1. Initial Migration Run
	err = runner.MigrateTenant(ctx, tenantAB)
	require.NoError(t, err)

	// Verify column and table exist
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	var colExists bool
	err = conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_schema = 'tenant_al_barakah_mart' 
			  AND table_name = 'products' 
			  AND column_name = 'audit_tag'
		);
	`).Scan(&colExists)
	require.NoError(t, err)
	assert.True(t, colExists)

	// 2. Rerun Migration on same tenant (Must be completely idempotent, zero errors, no duplicate rows)
	err = runner.MigrateTenant(ctx, tenantAB)
	require.NoError(t, err)

	var migrationCount int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM tenant_al_barakah_mart.schema_migrations;").Scan(&migrationCount)
	require.NoError(t, err)
	assert.Equal(t, 2, migrationCount, "must have exactly 2 recorded migrations")
}

func TestTenantMigration_MultiTenantConvergence(t *testing.T) {
	db := newTestDatabase(t)
	cleanupTenantMigrations(t, db.Pool)
	tenantRepo := tenant.NewRepository(db)
	ctx := context.Background()

	migrations := []migration.Migration{
		{
			Version:     1,
			Description: "Add metadata jsonb to suppliers",
			SQL:         "ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;",
		},
	}

	runner, err := migration.NewRunner(db.Pool, tenantRepo, migrations)
	require.NoError(t, err)

	// Run across all active tenants
	results, err := runner.MigrateAllTenants(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 2)
	assert.NoError(t, results["al-barakah-mart"])
	assert.NoError(t, results["darussalam-store"])

	// Check convergence in both schemas
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	for _, schema := range []string{"tenant_al_barakah_mart", "tenant_darussalam_store"} {
		var colExists bool
		err = conn.QueryRow(ctx, fmt.Sprintf(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_schema = '%s' 
				  AND table_name = 'suppliers' 
				  AND column_name = 'metadata'
			);
		`, schema)).Scan(&colExists)
		require.NoError(t, err)
		assert.True(t, colExists, "column metadata must exist in %s", schema)
	}
}

func TestTenantMigration_ChecksumMismatchRejection(t *testing.T) {
	db := newTestDatabase(t)
	cleanupTenantMigrations(t, db.Pool)
	tenantRepo := tenant.NewRepository(db)
	ctx := context.Background()

	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)

	// Step 1: Apply original migration v1
	v1Original := []migration.Migration{
		{
			Version:     1,
			Description: "Initial test migration",
			SQL:         "CREATE TABLE IF NOT EXISTS test_chk (id INT);",
		},
	}
	runner1, err := migration.NewRunner(db.Pool, tenantRepo, v1Original)
	require.NoError(t, err)
	require.NoError(t, runner1.MigrateTenant(ctx, tenantAB))

	// Step 2: Tamper with v1 SQL content (Checksum mismatch)
	v1Tampered := []migration.Migration{
		{
			Version:     1,
			Description: "Initial test migration",
			SQL:         "CREATE TABLE IF NOT EXISTS test_chk (id INT, tampered TEXT);", // Modified!
		},
	}
	runner2, err := migration.NewRunner(db.Pool, tenantRepo, v1Tampered)
	require.NoError(t, err)

	err = runner2.MigrateTenant(ctx, tenantAB)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestTenantMigration_FailureIsolationAndResume(t *testing.T) {
	db := newTestDatabase(t)
	cleanupTenantMigrations(t, db.Pool)
	tenantRepo := tenant.NewRepository(db)
	ctx := context.Background()

	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)
	tenantDS, err := tenantRepo.GetTenantBySlug(ctx, "darussalam-store")
	require.NoError(t, err)

	// 1. Successful migration on Tenant A
	mValid := []migration.Migration{
		{
			Version:     1,
			Description: "Valid table",
			SQL:         "CREATE TABLE IF NOT EXISTS tenant_valid_tbl (id INT);",
		},
	}
	rValid, _ := migration.NewRunner(db.Pool, tenantRepo, mValid)
	require.NoError(t, rValid.MigrateTenant(ctx, tenantAB))

	// 2. Erroneous migration targeting Tenant B (syntax error)
	mInvalid := []migration.Migration{
		{
			Version:     1,
			Description: "Invalid table syntax",
			SQL:         "CREATE TABLE INCORRECT SYNTAX ERROR %%%;",
		},
	}
	rInvalid, _ := migration.NewRunner(db.Pool, tenantRepo, mInvalid)
	err = rInvalid.MigrateTenant(ctx, tenantDS)
	require.Error(t, err)

	// 3. Resume Tenant B with fixed migration
	mFixed := []migration.Migration{
		{
			Version:     1,
			Description: "Fixed table syntax",
			SQL:         "CREATE TABLE IF NOT EXISTS tenant_valid_tbl (id INT);",
		},
	}
	rFixed, _ := migration.NewRunner(db.Pool, tenantRepo, mFixed)
	err = rFixed.MigrateTenant(ctx, tenantDS)
	require.NoError(t, err, "resumed migration on Tenant B should succeed cleanly")
}

func TestTenantMigration_ConcurrentRunnersAdvisoryLock(t *testing.T) {
	db := newTestDatabase(t)
	cleanupTenantMigrations(t, db.Pool)
	tenantRepo := tenant.NewRepository(db)
	ctx := context.Background()

	migrations := []migration.Migration{
		{
			Version:     1,
			Description: "Concurrent test table",
			SQL:         "CREATE TABLE IF NOT EXISTS test_concurrent_run (id INT);",
		},
	}

	runner1, err := migration.NewRunner(db.Pool, tenantRepo, migrations)
	require.NoError(t, err)
	runner2, err := migration.NewRunner(db.Pool, tenantRepo, migrations)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error
	go func() {
		defer wg.Done()
		_, err1 = runner1.MigrateAllTenants(ctx)
	}()

	go func() {
		defer wg.Done()
		_, err2 = runner2.MigrateAllTenants(ctx)
	}()

	wg.Wait()

	// At least one runner succeeds, and if there's overlap the second runner is blocked safely by advisory lock
	if err1 != nil {
		assert.Contains(t, err1.Error(), "advisory lock")
	}
	if err2 != nil {
		assert.Contains(t, err2.Error(), "advisory lock")
	}
	assert.True(t, err1 == nil || err2 == nil, "at least one runner must succeed")
}
