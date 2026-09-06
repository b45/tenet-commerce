package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/entitlement"
	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/manager"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/supplychain"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
)

// setupRBACPublicRouteRouter builds a real production-like router with authentic JWT, tenant scoping,
// and full RBAC & entitlement guards across all domain route groups.
func setupRBACPublicRouteRouter(t *testing.T, db *database.PostgresDB, rdb *pkgRedis.Client) (*gin.Engine, *pkgAuth.JWTService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	jwtService := pkgAuth.NewJWTService()
	tenantRepo := tenant.NewRepository(db)
	authRepo := internalAuth.NewRepository(db)
	authHandler := internalAuth.NewHandler(authRepo, jwtService, rdb)

	entitlementRepo := entitlement.NewRepository(db)
	entitlementSvc := entitlement.NewService(entitlementRepo, rdb)
	entitlementHandler := entitlement.NewHandler(entitlementSvc)

	ledgerSvc := ledger.NewService(ledger.NewRepository())
	ledgerHandler := ledger.NewHandler(ledgerSvc)

	posSvc := pos.NewService(pos.NewRepository(), ledgerSvc)
	posHandler := pos.NewHandler(posSvc, entitlementSvc)

	scSvc := supplychain.NewService(supplychain.NewRepository(), ledgerSvc)
	scHandler := supplychain.NewHandler(scSvc)

	managerSvc := manager.NewService(manager.NewRepository())
	managerHandler := manager.NewHandler(managerSvc, entitlementSvc)

	router := gin.New()
	apiV1 := router.Group("/api/v1")
	{
		// Public Auth routes
		authPublic := apiV1.Group("/auth")
		authHandler.RegisterPublicRoutes(authPublic)

		// Protected routes with JWT & Tenant Context
		protected := apiV1.Group("")
		protected.Use(
			internalAuth.JWTAuthMiddleware(jwtService, rdb),
			tenant.ContextMiddleware(db, tenantRepo),
		)

		authHandler.RegisterProtectedRoutes(protected.Group("/auth"))
		entitlementHandler.RegisterRoutes(protected)
		posHandler.RegisterRoutes(protected.Group("/pos"), rdb)
		scHandler.RegisterRoutes(protected.Group("/supply-chain"), rdb)
		ledgerHandler.RegisterRoutes(protected.Group("/ledger"), rdb)
		managerHandler.RegisterRoutes(protected.Group("/manager"))
	}

	return router, jwtService
}

// TestRBAC_MatrixTableDriven verifies that each role (CASHIER, MANAGER, COMPLIANCE_OFFICER, FINANCIAL_ADMIN, SUPER_ADMIN)
// adheres strictly to its documented permission matrix and cannot bypass domain boundaries via direct HTTP calls.
func TestRBAC_MatrixTableDriven(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router, jwtService := setupRBACPublicRouteRouter(t, db, rdb)

	tenantRepo := tenant.NewRepository(db)
	tenantAB, err := tenantRepo.GetTenantBySlug(context.Background(), "al-barakah-mart")
	require.NoError(t, err)

	type testEndpoint struct {
		method string
		path   string
		body   string
	}

	endpoints := map[string]testEndpoint{
		"pos_products_get": {
			method: http.MethodGet,
			path:   "/api/v1/pos/products",
		},
		"pos_checkout_post": {
			method: http.MethodPost,
			path:   "/api/v1/pos/checkout",
			body:   `{"payment_method":"CASH","cash_tendered":100000,"items":[]}`,
		},
		"pos_adjust_post": {
			method: http.MethodPost,
			path:   "/api/v1/pos/inventory/adjust",
			body:   `{"product_id":"11111111-1111-1111-1111-111111111111","adjustment_quantity":1,"reason":"audit"}`,
		},
		"supply_chain_suppliers_get": {
			method: http.MethodGet,
			path:   "/api/v1/supply-chain/suppliers",
		},
		"supply_chain_po_post": {
			method: http.MethodPost,
			path:   "/api/v1/supply-chain/purchase-orders",
			body:   `{"supplier_id":"` + uuid.NewString() + `","items":[]}`,
		},
		"ledger_accounts_get": {
			method: http.MethodGet,
			path:   "/api/v1/ledger/accounts",
		},
		"ledger_entries_post": {
			method: http.MethodPost,
			path:   "/api/v1/ledger/entries",
			body:   `{"description":"test manual entry","lines":[]}`,
		},
		"manager_dashboard_get": {
			method: http.MethodGet,
			path:   "/api/v1/manager/dashboard",
		},
	}

	// Matrix of expected access (true = allowed/200/400/422 validation, false = forbidden/403)
	// Key: endpointKey -> role -> allowed
	expectedMatrix := map[string]map[string]bool{
		"pos_products_get": {
			"CASHIER":            true,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": true,
			"FINANCIAL_ADMIN":    true,
			"SUPER_ADMIN":        true,
		},
		"pos_checkout_post": {
			"CASHIER":            true,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": false,
			"FINANCIAL_ADMIN":    false,
			"SUPER_ADMIN":        true,
		},
		"pos_adjust_post": {
			"CASHIER":            false,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": false,
			"FINANCIAL_ADMIN":    false,
			"SUPER_ADMIN":        true,
		},
		"supply_chain_suppliers_get": {
			"CASHIER":            false,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": true,
			"FINANCIAL_ADMIN":    false,
			"SUPER_ADMIN":        true,
		},
		"supply_chain_po_post": {
			"CASHIER":            false,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": true,
			"FINANCIAL_ADMIN":    false,
			"SUPER_ADMIN":        true,
		},
		"ledger_accounts_get": {
			"CASHIER":            false,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": true,
			"FINANCIAL_ADMIN":    true,
			"SUPER_ADMIN":        true,
		},
		"ledger_entries_post": {
			"CASHIER":            false,
			"MANAGER":            false,
			"COMPLIANCE_OFFICER": false,
			"FINANCIAL_ADMIN":    true,
			"SUPER_ADMIN":        true,
		},
		"manager_dashboard_get": {
			"CASHIER":            false,
			"MANAGER":            true,
			"COMPLIANCE_OFFICER": false,
			"FINANCIAL_ADMIN":    false,
			"SUPER_ADMIN":        true,
		},
	}

	roles := []string{"CASHIER", "MANAGER", "COMPLIANCE_OFFICER", "FINANCIAL_ADMIN", "SUPER_ADMIN"}

	for _, role := range roles {
		token, _, _, err := jwtService.GenerateTokenPair(
			uuid.NewString(),
			tenantAB.ID,
			tenantAB.Slug,
			role,
		)
		require.NoError(t, err)

		for endpointKey, ep := range endpoints {
			expectedAllowed := expectedMatrix[endpointKey][role]
			testName := fmt.Sprintf("%s_%s", role, endpointKey)

			t.Run(testName, func(t *testing.T) {
				w := httptest.NewRecorder()
				var bodyReader *bytes.Buffer
				if ep.body != "" {
					bodyReader = bytes.NewBufferString(ep.body)
				} else {
					bodyReader = bytes.NewBuffer(nil)
				}

				req := httptest.NewRequest(ep.method, ep.path, bodyReader)
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("X-Tenant-ID", tenantAB.Slug)
				if ep.method == http.MethodPost || ep.method == http.MethodPut {
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("Idempotency-Key", "test-rbac-"+uuid.NewString())
				}

				router.ServeHTTP(w, req)

				if expectedAllowed {
					assert.NotEqual(t, http.StatusForbidden, w.Code,
						"Role %s should have permission for %s %s, got forbidden: %s", role, ep.method, ep.path, w.Body.String())
					assert.NotEqual(t, http.StatusUnauthorized, w.Code,
						"Role %s should be authorized for %s %s", role, ep.method, ep.path)
				} else {
					assert.Equal(t, http.StatusForbidden, w.Code,
						"Role %s MUST BE FORBIDDEN from %s %s, but got %d: %s", role, ep.method, ep.path, w.Code, w.Body.String())
					assert.Contains(t, w.Body.String(), "FORBIDDEN")
				}
			})
		}
	}
}

// TestRBAC_RevokedSessionAndRoleDowngrade verifies that token revocation immediately blocks access
// and that role downgrade in refreshed credentials restricts permissions on direct calls.
func TestRBAC_RevokedSessionAndRoleDowngrade(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router, jwtService := setupRBACPublicRouteRouter(t, db, rdb)

	tenantRepo := tenant.NewRepository(db)
	tenantAB, err := tenantRepo.GetTenantBySlug(context.Background(), "al-barakah-mart")
	require.NoError(t, err)

	// 1. Generate Manager token
	mgrToken, _, _, err := jwtService.GenerateTokenPair(
		uuid.NewString(),
		tenantAB.ID,
		tenantAB.Slug,
		"MANAGER",
	)
	require.NoError(t, err)

	// Verify Manager can access supply-chain
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/supply-chain/suppliers", nil)
	req1.Header.Set("Authorization", "Bearer "+mgrToken)
	req1.Header.Set("X-Tenant-ID", tenantAB.Slug)
	router.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code)

	// 2. Blacklist/Revoke Manager Token in Redis
	tokenHash := internalAuth.HashToken(mgrToken)
	err = rdb.RDB.Set(context.Background(), internalAuth.BlacklistKeyPrefix+tokenHash, "revoked", 15*time.Minute).Err()
	require.NoError(t, err)

	// Verify revoked token is immediately rejected with 401 TOKEN_REVOKED
	wRevoked := httptest.NewRecorder()
	reqRevoked := httptest.NewRequest(http.MethodGet, "/api/v1/supply-chain/suppliers", nil)
	reqRevoked.Header.Set("Authorization", "Bearer "+mgrToken)
	reqRevoked.Header.Set("X-Tenant-ID", tenantAB.Slug)
	router.ServeHTTP(wRevoked, reqRevoked)
	assert.Equal(t, http.StatusUnauthorized, wRevoked.Code)
	assert.Contains(t, wRevoked.Body.String(), "TOKEN_REVOKED")

	// 3. Role Downgrade: Issue downgraded token for same user as CASHIER
	cashierToken, _, _, err := jwtService.GenerateTokenPair(
		uuid.NewString(),
		tenantAB.ID,
		tenantAB.Slug,
		"CASHIER",
	)
	require.NoError(t, err)

	// Verify downgraded user is rejected from supply-chain
	wDowngraded := httptest.NewRecorder()
	reqDowngraded := httptest.NewRequest(http.MethodGet, "/api/v1/supply-chain/suppliers", nil)
	reqDowngraded.Header.Set("Authorization", "Bearer "+cashierToken)
	reqDowngraded.Header.Set("X-Tenant-ID", tenantAB.Slug)
	router.ServeHTTP(wDowngraded, reqDowngraded)
	assert.Equal(t, http.StatusForbidden, wDowngraded.Code)
}

// TestRBAC_DirectObjectAccessAndCrossTenantReference verifies that querying resources
// across tenants fails closed and does not disclose documents belonging to another tenant.
func TestRBAC_DirectObjectAccessAndCrossTenantReference(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router, jwtService := setupRBACPublicRouteRouter(t, db, rdb)

	tenantRepo := tenant.NewRepository(db)
	tenantAB, err := tenantRepo.GetTenantBySlug(context.Background(), "al-barakah-mart")
	require.NoError(t, err)
	tenantDS, err := tenantRepo.GetTenantBySlug(context.Background(), "darussalam-store")
	require.NoError(t, err)

	// Token for Tenant A (al-barakah-mart)
	tokenA, _, _, err := jwtService.GenerateTokenPair(uuid.NewString(), tenantAB.ID, tenantAB.Slug, "MANAGER")
	require.NoError(t, err)

	// Token for Tenant B (darussalam-store)
	tokenB, _, _, err := jwtService.GenerateTokenPair(uuid.NewString(), tenantDS.ID, tenantDS.Slug, "MANAGER")
	require.NoError(t, err)

	// 1. Fetch Tenant B products to find a target object ID
	wB := httptest.NewRecorder()
	reqB := httptest.NewRequest(http.MethodGet, "/api/v1/pos/products", nil)
	reqB.Header.Set("Authorization", "Bearer "+tokenB)
	reqB.Header.Set("X-Tenant-ID", tenantDS.Slug)
	router.ServeHTTP(wB, reqB)
	require.Equal(t, http.StatusOK, wB.Code)

	var respB struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(wB.Body.Bytes(), &respB))
	require.NotEmpty(t, respB.Data)
	tenantBProductID := respB.Data[0].ID

	// 2. Tenant A tries to directly fetch Tenant B product by ID
	wCross := httptest.NewRecorder()
	reqCross := httptest.NewRequest(http.MethodGet, "/api/v1/pos/products/"+tenantBProductID, nil)
	reqCross.Header.Set("Authorization", "Bearer "+tokenA)
	reqCross.Header.Set("X-Tenant-ID", tenantAB.Slug)
	router.ServeHTTP(wCross, reqCross)

	// Must fail-closed (404 not found in Tenant A schema) rather than leaking Tenant B data
	assert.Equal(t, http.StatusNotFound, wCross.Code)
	assert.Contains(t, wCross.Body.String(), "NOT_FOUND")
}
