package tenant

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

// ContextMiddleware injects the tenant schema into the PostgreSQL search_path
// for every incoming HTTP request.
func ContextMiddleware(db *database.PostgresDB, repo *Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract Tenant ID from header (or JWT in Phase 2)
		tenantSlug := c.GetHeader("X-Tenant-ID")
		if tenantSlug == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "MISSING_TENANT_HEADER",
					"message": "X-Tenant-ID header is required",
				},
			})
			return
		}

		// 2. Validate tenant and get schema name
		tenant, err := repo.GetTenantBySlug(c.Request.Context(), tenantSlug)
		if err != nil {
			log.Printf("Tenant lookup failed for slug '%s': %v", tenantSlug, err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "INVALID_TENANT",
					"message": "Tenant not found or inactive",
				},
			})
			return
		}

		// 3. Acquire a dedicated connection from the pool for this request
		conn, err := db.Pool.Acquire(c.Request.Context())
		if err != nil {
			log.Printf("Failed to acquire DB connection: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "DATABASE_UNAVAILABLE",
					"message": "Failed to connect to the database",
				},
			})
			return
		}
		
		// Ensure the connection is released back to the pool after the request
		defer conn.Release()

		// 4. Dynamically set the PostgreSQL search_path to the tenant's isolated schema
		// WARNING: We must sanitize/validate the schema name to prevent SQL injection.
		// The repository layer ensures schemaName comes strictly from our trusted registry table.
		setPathQuery := fmt.Sprintf("SET search_path TO %s, public;", pgx.Identifier{tenant.SchemaName}.Sanitize())
		
		if _, err := conn.Exec(c.Request.Context(), setPathQuery); err != nil {
			log.Printf("Failed to set search_path to %s: %v", tenant.SchemaName, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TENANT_CONTEXT_FAILURE",
					"message": "Failed to initialize tenant database context",
				},
			})
			return
		}

		// 5. Inject the scoped connection into the Gin context
		// All subsequent domain logic MUST retrieve this specific connection 
		// via c.Get("db_conn") instead of using the global pool directly.
		c.Set("db_conn", conn)
		c.Set("tenant", tenant)

		// Proceed to the next handler
		c.Next()
	}
}
