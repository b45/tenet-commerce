package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/entitlement"
	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/manager"
	"github.com/b45/tenet-commerce/backend/internal/pos"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

func setupEntitlementTestRouter(
	t *testing.T,
	db *database.PostgresDB,
	tenantSlug string,
	role string,
	permissions []string,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	tenantRepo := tenant.NewRepository(db)
	rdb := newTestRedisClient(t)

	entitlementRepo := entitlement.NewRepository(db)
	entitlementSvc := entitlement.NewService(entitlementRepo, rdb)
	entitlementHandler := entitlement.NewHandler(entitlementSvc)

	ledgerSvc := ledger.NewService(ledger.NewRepository())
	posRepo := pos.NewRepository()
	posSvc := pos.NewService(posRepo, ledgerSvc)
	posHandler := pos.NewHandler(posSvc, entitlementSvc)

	managerRepo := manager.NewRepository()
	managerSvc := manager.NewService(managerRepo)
	managerHandler := manager.NewHandler(managerSvc, entitlementSvc)

	router := gin.New()

	// Simulate verified JWT and tenant scoping middleware
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", tenantSlug)
		c.Set("role", role)
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			Role:        role,
			Permissions: permissions,
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))

	apiV1 := router.Group("/api/v1")
	{
		entitlementHandler.RegisterRoutes(apiV1)
		posHandler.RegisterRoutes(apiV1.Group("/pos"), rdb)
		managerHandler.RegisterRoutes(apiV1.Group("/manager"))
	}

	return router
}

func TestEntitlement_CapabilitiesIntrospection(t *testing.T) {
	db := newTestDatabase(t)

	t.Run("Growth tier (al-barakah-mart) has pos.daily_summary enabled", func(t *testing.T) {
		router := setupEntitlementTestRouter(t, db, "al-barakah-mart", "MANAGER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/me/capabilities", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				TenantID     string                          `json:"tenant_id"`
				Plan         entitlement.PlanSummary         `json:"plan"`
				Capabilities map[string]entitlement.Decision `json:"capabilities"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "growth", resp.Data.Plan.Code)
		assert.Equal(t, entitlement.StatusActive, resp.Data.Plan.Status)

		// Check capability decisions
		dailySummary, exists := resp.Data.Capabilities["pos.daily_summary"]
		require.True(t, exists, "pos.daily_summary must exist in capabilities map")
		assert.True(t, dailySummary.Allowed)
		assert.Equal(t, entitlement.ReasonEntitled, dailySummary.Reason)

		checkout, exists := resp.Data.Capabilities["pos.checkout"]
		require.True(t, exists)
		assert.True(t, checkout.Allowed)
	})

	t.Run("Starter tier (darussalam-store) has pos.daily_summary disabled", func(t *testing.T) {
		router := setupEntitlementTestRouter(t, db, "darussalam-store", "CASHIER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/me/capabilities", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				TenantID     string                          `json:"tenant_id"`
				Plan         entitlement.PlanSummary         `json:"plan"`
				Capabilities map[string]entitlement.Decision `json:"capabilities"`
			} `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "starter", resp.Data.Plan.Code)

		dailySummary, exists := resp.Data.Capabilities["pos.daily_summary"]
		require.True(t, exists)
		assert.False(t, dailySummary.Allowed)
		assert.Equal(t, entitlement.ReasonFeatureDisabled, dailySummary.Reason)
	})

	t.Run("Canceled subscription has inactive status and all capabilities denied", func(t *testing.T) {
		_, err := db.Pool.Exec(context.Background(), "UPDATE public.tenant_subscriptions SET status = 'CANCELED' WHERE tenant_id = 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22'")
		require.NoError(t, err)
		defer func() {
			_, _ = db.Pool.Exec(context.Background(), "UPDATE public.tenant_subscriptions SET status = 'ACTIVE' WHERE tenant_id = 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22'")
		}()

		router := setupEntitlementTestRouter(t, db, "darussalam-store", "MANAGER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/me/capabilities", nil)
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp struct {
			Success bool `json:"success"`
			Data    struct {
				Plan         entitlement.PlanSummary         `json:"plan"`
				Capabilities map[string]entitlement.Decision `json:"capabilities"`
			} `json:"data"`
		}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, entitlement.StatusCanceled, resp.Data.Plan.Status)

		for key, cap := range resp.Data.Capabilities {
			assert.False(t, cap.Allowed, "capability %s should not be allowed for canceled subscription", key)
			assert.Equal(t, entitlement.ReasonSubscriptionInactive, cap.Reason)
		}
	})
}

func TestEntitlement_VerticalSlice_POSDailySummary(t *testing.T) {
	db := newTestDatabase(t)

	t.Run("Starter tier is rejected with 403 FEATURE_NOT_ENTITLED", func(t *testing.T) {
		router := setupEntitlementTestRouter(t, db, "darussalam-store", "CASHIER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/pos/daily-summary", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var errResp struct {
			Success bool `json:"success"`
			Error   struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.False(t, errResp.Success)
		assert.Equal(t, "FEATURE_NOT_ENTITLED", errResp.Error.Code)
		assert.Contains(t, errResp.Error.Message, "pos.daily_summary")
	})

	t.Run("Growth tier is permitted with 200 OK", func(t *testing.T) {
		router := setupEntitlementTestRouter(t, db, "al-barakah-mart", "CASHIER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/pos/daily-summary", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestEntitlement_VerticalSlice_ManagerDashboard(t *testing.T) {
	db := newTestDatabase(t)

	t.Run("Starter tier with MANAGER role is rejected with 403 FEATURE_NOT_ENTITLED", func(t *testing.T) {
		router := setupEntitlementTestRouter(t, db, "darussalam-store", "MANAGER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/manager/dashboard", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var errResp struct {
			Success bool `json:"success"`
			Error   struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Equal(t, "FEATURE_NOT_ENTITLED", errResp.Error.Code)
	})

	t.Run("Growth tier with MANAGER role is permitted with 200 OK", func(t *testing.T) {
		router := setupEntitlementTestRouter(t, db, "al-barakah-mart", "MANAGER", []string{"pos:read"})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/manager/dashboard", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
