package tenant_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *database.PostgresDB) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()

	db, err := database.NewPostgresDB(ctx)
	if err != nil {
		t.Skipf("Skipping integration test: Postgres not reachable at 127.0.0.1:5432: %v", err)
	}

	repo := tenant.NewRepository(db)
	router := gin.New()

	v1 := router.Group("/api/v1")
	v1.Use(tenant.ContextMiddleware(db, repo))
	v1.GET("/test-tenant", func(c *gin.Context) {
		tenantObj, _ := c.Get("tenant")
		c.JSON(http.StatusOK, gin.H{"tenant": tenantObj})
	})

	return router, db
}

func TestContextMiddleware_MissingHeader(t *testing.T) {
	router, db := setupTestRouter(t)
	if db != nil {
		defer db.Close()
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test-tenant", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "MISSING_TENANT_CONTEXT")
}

func TestContextMiddleware_InvalidTenant(t *testing.T) {
	router, db := setupTestRouter(t)
	if db != nil {
		defer db.Close()
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test-tenant", nil)
	req.Header.Set("X-Tenant-ID", "non-existent-tenant-slug")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_TENANT")
}

func TestContextMiddleware_ValidTenant(t *testing.T) {
	router, db := setupTestRouter(t)
	if db != nil {
		defer db.Close()
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test-tenant", nil)
	req.Header.Set("X-Tenant-ID", "al-barakah-mart")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "al-barakah-mart")
}

func TestIsValidSchemaName(t *testing.T) {
	tests := []struct {
		schema string
		valid  bool
	}{
		{"tenant_al_barakah_mart", true},
		{"tenant_darussalam_store", true},
		{"public", true},
		{"tenant_123", true},
		{"", false},
		{"tenant-with-hyphen", false},
		{"tenant; DROP TABLE users;--", false},
		{"tenant with spaces", false},
		{"tenant$special", false},
		{"tenant/slash", false},
		{string(make([]byte, 64)), false},
	}

	for _, tc := range tests {
		t.Run(tc.schema, func(t *testing.T) {
			assert.Equal(t, tc.valid, tenant.IsValidSchemaName(tc.schema))
		})
	}
}

func TestContextMiddleware_HeaderTokenConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_slug", "tenant-alpha")
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(nil, nil))
	router.GET("/conflict-test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/conflict-test", nil)
	req.Header.Set("X-Tenant-ID", "tenant-beta") // Conflicts with tenant-alpha in token context
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "TENANT_CONTEXT_CONFLICT")
}
