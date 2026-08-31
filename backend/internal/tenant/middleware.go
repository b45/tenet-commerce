package tenant

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// ContextMiddleware injects the tenant schema into the PostgreSQL search_path
// for every incoming HTTP request. It resolves the tenant from:
//  1. The `tenant_slug` key set by JWTAuthMiddleware (authenticated routes), or
//  2. The `X-Tenant-ID` request header (fallback for public/unauthenticated routes).
func ContextMiddleware(db *database.PostgresDB, repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLogger := logger.FromContext(c.Request.Context())

		// 1. Resolve tenant slug: JWT context takes priority over manual header.
		tenantSlug := c.GetString("tenant_slug")
		if tenantSlug == "" {
			tenantSlug = c.GetHeader("X-Tenant-ID")
		}

		if tenantSlug == "" {
			response.AbortBadRequest(c, "MISSING_TENANT_CONTEXT",
				"Tenant context could not be resolved. Provide a valid JWT or X-Tenant-ID header.")
			return
		}

		// 2. Validate tenant exists and is ACTIVE in the registry
		tenantData, err := repo.GetTenantBySlug(c.Request.Context(), tenantSlug)
		if err != nil {
			reqLogger.Warn("Tenant lookup failed",
				"tenant_slug", tenantSlug,
				"error", err.Error(),
			)
			response.AbortUnauthorized(c, "INVALID_TENANT", "Tenant not found or inactive")
			return
		}

		// 3. Acquire a dedicated connection from the pool for this request lifetime
		conn, err := db.Pool.Acquire(c.Request.Context())
		if err != nil {
			reqLogger.Error("Failed to acquire DB connection from pool",
				"tenant_slug", tenantSlug,
				"error", err.Error(),
			)
			response.AbortInternalServerError(c, "DATABASE_UNAVAILABLE", "Failed to connect to the database")
			return
		}
		defer conn.Release()

		// 4. Set the schema search path dynamically.
		// SECURITY: schemaName is retrieved from our trusted public.tenants registry,
		// never directly from user input, and sanitized by pgx.Identifier to prevent SQL injection.
		searchPathQuery := fmt.Sprintf("SET search_path TO %s, public;",
			pgx.Identifier{tenantData.SchemaName}.Sanitize())

		if _, err := conn.Exec(c.Request.Context(), searchPathQuery); err != nil {
			reqLogger.Error("Failed to set search_path",
				"schema_name", tenantData.SchemaName,
				"error", err.Error(),
			)
			response.AbortInternalServerError(c, "TENANT_CONTEXT_FAILURE",
				"Failed to initialize tenant database context")
			return
		}

		// 5. Inject the scoped connection and tenant data into Gin context.
		// Downstream handlers MUST use c.Get("db_conn") and NOT touch the global pool directly.
		c.Set("db_conn", conn)
		c.Set("tenant", tenantData)

		c.Next()
	}
}
