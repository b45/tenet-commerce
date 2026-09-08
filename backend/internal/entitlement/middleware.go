package entitlement

import (
	"fmt"

	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// RequireFeature creates a Gin middleware that enforces tenant entitlement for a specific feature key.
// If the tenant's active plan does not grant the feature, the request is rejected with HTTP 403 Forbidden.
func RequireFeature(svc *Service, featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLogger := logger.FromContext(c.Request.Context())

		// Resolve tenant ID from verified tenant context or JWT claims
		var tenantID string
		if tVal, exists := c.Get("tenant"); exists {
			if tObj, ok := tVal.(*tenant.Tenant); ok && tObj.ID != "" {
				tenantID = tObj.ID
			}
		}
		if tenantID == "" {
			if claimsVal, exists := c.Get("jwt_claims"); exists {
				if claims, ok := claimsVal.(*pkgAuth.CustomClaims); ok {
					tenantID = claims.TenantID
				}
			}
		}

		if tenantID == "" {
			response.AbortUnauthorized(c, "MISSING_TENANT_CONTEXT", "Tenant context could not be resolved")
			return
		}

		decision, err := svc.Evaluate(c.Request.Context(), tenantID, featureKey)
		if err != nil {
			reqLogger.Error("Entitlement evaluation error",
				"tenant_id", tenantID,
				"feature_key", featureKey,
				"error", err.Error(),
			)
			response.AbortInternalServerError(c, "ENTITLEMENT_CHECK_FAILED", "Failed to evaluate feature capability")
			return
		}

		if !decision.Allowed {
			reqLogger.Warn("Access denied: tenant not entitled to feature",
				"tenant_id", tenantID,
				"feature_key", featureKey,
				"reason", string(decision.Reason),
			)
			response.AbortForbidden(c, "FEATURE_NOT_ENTITLED", fmt.Sprintf("Tenant is not entitled to capability '%s' (reason: %s)", featureKey, decision.Reason))
			return
		}

		c.Next()
	}
}
