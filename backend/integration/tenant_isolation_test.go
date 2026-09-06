package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
)

func TestTenantIsolation_ScopedTransactionLocalSearchPath(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	tenantRepo := tenant.NewRepository(db)

	ctx := context.Background()
	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)
	require.NotNil(t, tenantAB)

	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	scopedDB, err := tenant.NewScopedDB(conn, tenantAB)
	require.NoError(t, err)

	// 1. Begin scoped transaction with SET LOCAL search_path
	tx, err := scopedDB.BeginTx(ctx)
	require.NoError(t, err)

	var activeSchema string
	err = tx.QueryRow(ctx, "SELECT current_schema()").Scan(&activeSchema)
	require.NoError(t, err)
	assert.Equal(t, "tenant_al_barakah_mart", activeSchema)

	// Verify table access inside tenant schema
	var productCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM products").Scan(&productCount)
	require.NoError(t, err)
	assert.Greater(t, productCount, 0)

	// 2. Rollback transaction
	require.NoError(t, tx.Rollback(ctx))

	// 3. Test RunInTx helper commits clean transactions
	executed := false
	err = scopedDB.RunInTx(ctx, func(txTenant pgx.Tx) error {
		var schema string
		if err := txTenant.QueryRow(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
			return err
		}
		assert.Equal(t, "tenant_al_barakah_mart", schema)
		executed = true
		return nil
	})
	require.NoError(t, err)
	assert.True(t, executed)

	// 4. Test RunInTx rolls back when an error occurs
	errExpected := fmt.Errorf("intentional failure to trigger rollback")
	err = scopedDB.RunInTx(ctx, func(txTenant pgx.Tx) error {
		return errExpected
	})
	require.ErrorIs(t, err, errExpected)
}

func TestTenantIsolation_StressAlternationPoolSizeOne(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPoolSizeOneDatabase(t)
	tenantRepo := tenant.NewRepository(db)

	router := gin.New()
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/inspect", func(c *gin.Context) {
		scopedDB, ok := tenant.GetScopedDB(c)
		if !ok {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		var schema string
		var count int
		err := scopedDB.RunInTx(c.Request.Context(), func(tx pgx.Tx) error {
			if err := tx.QueryRow(c.Request.Context(), "SELECT current_schema()").Scan(&schema); err != nil {
				return err
			}
			return tx.QueryRow(c.Request.Context(), "SELECT COUNT(*) FROM products").Scan(&count)
		})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"schema":        schema,
			"product_count": count,
		})
	})

	expectedSchemas := []string{
		"al-barakah-mart",
		"darussalam-store",
	}

	for iteration := 0; iteration < 20; iteration++ {
		slug := expectedSchemas[iteration%2]
		req := httptest.NewRequest(http.MethodGet, "/inspect", nil)
		req.Header.Set("X-Tenant-ID", slug)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		if slug == "al-barakah-mart" {
			assert.Contains(t, w.Body.String(), "tenant_al_barakah_mart")
		} else {
			assert.Contains(t, w.Body.String(), "tenant_darussalam_store")
		}
	}

	// Direct pool check: after all requests have completed, verify that connection
	// released to the pool is completely clean and not locked to any tenant schema.
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()

	var defaultSchema string
	err = conn.QueryRow(context.Background(), "SELECT current_schema()").Scan(&defaultSchema)
	require.NoError(t, err)
	assert.Equal(t, "public", defaultSchema, "a released pooled connection must have returned to public schema")
}

func TestTenantIsolation_TokenClaimsMismatchForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newPoolSizeOneDatabase(t)
	tenantRepo := tenant.NewRepository(db)

	ctx := context.Background()
	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Set claims dynamically based on test header
		if c.GetHeader("X-Test-Claims") == "mismatch" {
			c.Set("tenant_slug", "al-barakah-mart")
			c.Set("jwt_claims", &pkgAuth.CustomClaims{
				TenantID:   "99999999-9999-9999-9999-999999999999", // Mismatched tenant ID
				TenantSlug: "al-barakah-mart",
			})
		} else {
			c.Set("tenant_slug", "al-barakah-mart")
			c.Set("jwt_claims", &pkgAuth.CustomClaims{
				TenantID:   tenantAB.ID,
				TenantSlug: "al-barakah-mart",
			})
		}
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	router.GET("/protected-resource", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Case 1: Valid matching request
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/protected-resource", nil)
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Case 2: Attempting cross-tenant access with X-Tenant-ID header conflicting with token
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/protected-resource", nil)
	req2.Header.Set("X-Tenant-ID", "darussalam-store") // Header mismatch with token
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
	assert.Contains(t, w2.Body.String(), "TENANT_CONTEXT_CONFLICT")

	// Case 3: Token claims tenant ID mismatch rejected with 403 Forbidden
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/protected-resource", nil)
	req3.Header.Set("X-Test-Claims", "mismatch")
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusForbidden, w3.Code)
	assert.Contains(t, w3.Body.String(), "TENANT_ACCESS_DENIED")
}

// TestTenantIsolation_ProductionRouterPoolSizeOne ensures request A/B through full production-style
// router with pool size 1 cannot read or mutate each other's data across iterations.
func TestTenantIsolation_ProductionRouterPoolSizeOne(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	rdb := newTestRedisClient(t)

	tenantRepo := tenant.NewRepository(db)
	ctx := context.Background()
	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)
	tenantDS, err := tenantRepo.GetTenantBySlug(ctx, "darussalam-store")
	require.NoError(t, err)

	ledgerService := ledger.NewService(ledger.NewRepository())
	posService := pos.NewService(pos.NewRepository(), ledgerService)
	posHandler := pos.NewHandler(posService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		slug := c.GetHeader("X-Tenant-ID")
		if slug == "" {
			slug = "al-barakah-mart"
		}
		tenantID := tenantAB.ID
		if slug == "darussalam-store" {
			tenantID = tenantDS.ID
		}
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", slug)
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			UserID:   "11111111-1111-1111-1111-111111111111",
			TenantID: tenantID,
			Role:     "MANAGER",
			Permissions: []string{
				"pos:checkout", "pos:void", "inventory:read", "inventory:write",
				"supply_chain:manage", "ledger:read", "ledger:write", "analytics:read",
			},
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))
	posHandler.RegisterRoutes(router.Group("/api/v1/pos"), rdb)

	tenants := []string{"al-barakah-mart", "darussalam-store"}

	for i := 0; i < 10; i++ {
		tenantSlug := tenants[i%2]

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/pos/products", nil)
		req.Header.Set("X-Tenant-ID", tenantSlug)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		if tenantSlug == "al-barakah-mart" {
			assert.Contains(t, w.Body.String(), "SKU-BEEF-01")
		} else {
			assert.Contains(t, w.Body.String(), "SKU-ZAMZAM-01")
		}
	}
}

// TestTenantIsolation_NoPublicFallbackWhenTableMissing proves domain queries fail-closed
// instead of silently reading from public schema if a table is missing in tenant schema.
func TestTenantIsolation_NoPublicFallbackWhenTableMissing(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	tenantRepo := tenant.NewRepository(db)

	ctx := context.Background()
	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)

	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	scopedDB, err := tenant.NewScopedDB(conn, tenantAB)
	require.NoError(t, err)

	// Attempting to query a public-only table (like public.plans) unqualified
	// inside tenant-scoped transaction must fail because search_path excludes public schema.
	err = scopedDB.ReadTx(ctx, func(tx pgx.Tx) error {
		var count int
		return tx.QueryRow(ctx, "SELECT COUNT(*) FROM plans").Scan(&count)
	})
	require.Error(t, err, "unqualified query to public-only table must fail when public schema is excluded from search_path")
}

// TestTenantIsolation_FaultInjectionAndPoolReuse tests that panic, rollback, or timeout
// inside a transaction never leaks the tenant search path to subsequent pooled connections.
func TestTenantIsolation_FaultInjectionAndPoolReuse(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	tenantRepo := tenant.NewRepository(db)

	ctx := context.Background()
	tenantAB, err := tenantRepo.GetTenantBySlug(ctx, "al-barakah-mart")
	require.NoError(t, err)

	// 1. Fault injection: Rollback inside RunInTx
	{
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		scopedDB, err := tenant.NewScopedDB(conn, tenantAB)
		require.NoError(t, err)

		_ = scopedDB.RunInTx(ctx, func(tx pgx.Tx) error {
			return fmt.Errorf("simulated transaction failure")
		})
		conn.Release()
	}

	// 2. Fault injection: Panic recovery inside RunInTx
	{
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		scopedDB, err := tenant.NewScopedDB(conn, tenantAB)
		require.NoError(t, err)

		assert.Panics(t, func() {
			_ = scopedDB.RunInTx(ctx, func(tx pgx.Tx) error {
				panic("simulated unhandled panic in handler")
			})
		})
		conn.Release()
	}

	// 3. Acquire connection again from pool and verify it is completely clean (public schema)
	{
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		defer conn.Release()

		var schema string
		err = conn.QueryRow(ctx, "SELECT current_schema()").Scan(&schema)
		require.NoError(t, err)
		assert.Equal(t, "public", schema, "connection from pool after panic/rollback must have default public schema")
	}
}
