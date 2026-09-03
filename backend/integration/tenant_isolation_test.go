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
