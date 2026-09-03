package tenant

import (
	"context"
	"fmt"
	"strings"
	"time"

	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextMiddleware injects the tenant schema into the PostgreSQL search_path
// for every incoming HTTP request. It resolves the tenant from:
//  1. The `tenant_slug` key set by JWTAuthMiddleware (authenticated routes), or
//  2. The `X-Tenant-ID` request header (fallback for public/unauthenticated routes).
func ContextMiddleware(db *database.PostgresDB, repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLogger := logger.FromContext(c.Request.Context())

		// 1. Resolve tenant slug: JWT context takes priority over manual header.
		tokenSlug := strings.TrimSpace(c.GetString("tenant_slug"))
		headerSlug := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))

		// Cross-tenant guard: prevent sending a token for tenant A while targeting tenant B in header.
		if tokenSlug != "" && headerSlug != "" && tokenSlug != headerSlug {
			reqLogger.Warn("Conflicting tenant identifiers in header and token",
				"header_tenant", headerSlug,
				"token_tenant", tokenSlug,
			)
			response.AbortBadRequest(c, "TENANT_CONTEXT_CONFLICT", "X-Tenant-ID header conflicts with authenticated token tenant")
			return
		}

		tenantSlug := tokenSlug
		if tenantSlug == "" {
			tenantSlug = headerSlug
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

		// 3. Cross-tenant token verification: verify JWT claims match resolved tenant attributes
		if claimsVal, exists := c.Get("jwt_claims"); exists {
			if claims, ok := claimsVal.(*pkgAuth.CustomClaims); ok {
				if claims.TenantID != "" && claims.TenantID != tenantData.ID {
					reqLogger.Warn("Token tenant ID does not match active tenant context",
						"claims_tenant_id", claims.TenantID,
						"resolved_tenant_id", tenantData.ID,
					)
					response.AbortForbidden(c, "TENANT_ACCESS_DENIED", "Token tenant ID does not match active tenant")
					return
				}
				if claims.TenantSlug != "" && claims.TenantSlug != tenantData.Slug {
					reqLogger.Warn("Token tenant slug does not match active tenant context",
						"claims_tenant_slug", claims.TenantSlug,
						"resolved_tenant_slug", tenantData.Slug,
					)
					response.AbortForbidden(c, "TENANT_ACCESS_DENIED", "Token tenant slug does not match active tenant")
					return
				}
			}
		}

		// 4. Validate schema name against strict identifier invariants
		if !IsValidSchemaName(tenantData.SchemaName) {
			reqLogger.Error("Invalid or unsafe schema name in tenant registry", "schema_name", tenantData.SchemaName)
			response.AbortInternalServerError(c, "INVALID_SCHEMA_CONFIGURATION", "Tenant schema configuration is invalid")
			return
		}

		// 5. Acquire a dedicated connection from the pool for this request lifetime
		conn, err := db.Pool.Acquire(c.Request.Context())
		if err != nil {
			reqLogger.Error("Failed to acquire DB connection from pool",
				"tenant_slug", tenantSlug,
				"error", err.Error(),
			)
			response.AbortInternalServerError(c, "DATABASE_UNAVAILABLE", "Failed to connect to the database")
			return
		}
		defer func() {
			// pgxpool reuses physical connections. Never return a tenant-scoped
			// session to the pool before its session settings have been cleared.
			resetCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			if _, err := conn.Exec(resetCtx, "RESET ALL"); err != nil {
				reqLogger.Error("Failed to reset database session before pool release; closing connection",
					"tenant_slug", tenantSlug,
					"error", err.Error(),
				)
				_ = conn.Conn().Close(resetCtx)
			}
			conn.Release()
		}()

		// 6. Set the schema search path dynamically.
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

		// 7. Initialize ScopedDB wrapper for transaction-local SET LOCAL search_path execution
		scopedDB, err := NewScopedDB(conn, tenantData)
		if err != nil {
			reqLogger.Error("Failed to initialize scoped tenant database wrapper", "error", err)
			response.AbortInternalServerError(c, "TENANT_CONTEXT_FAILURE", "Failed to initialize scoped tenant database context")
			return
		}

		// 8. Inject the scoped wrapper, raw connection, and tenant data into Gin context.
		c.Set("tenant_db", scopedDB)
		c.Set("db_conn", conn)
		c.Set("tenant", tenantData)

		c.Next()
	}
}

// GetConn retrieves the tenant-scoped database connection from the Gin context.
func GetConn(c *gin.Context) (*pgxpool.Conn, bool) {
	conn, exists := c.Get("db_conn")
	if !exists {
		return nil, false
	}
	pgxConn, ok := conn.(*pgxpool.Conn)
	return pgxConn, ok
}
