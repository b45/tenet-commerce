package entitlement

import (
	"errors"

	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler exposes HTTP endpoints for client capability introspection.
type Handler struct {
	svc *Service
}

// NewHandler creates a new entitlement HTTP handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes registers the capabilities introspection endpoint.
func (h *Handler) RegisterRoutes(router gin.IRoutes) {
	router.GET("/me/capabilities", h.GetCapabilities)
}

// GetCapabilities handles GET /api/v1/me/capabilities
func (h *Handler) GetCapabilities(c *gin.Context) {
	reqLogger := logger.FromContext(c.Request.Context())

	// Resolve verified tenant ID from context
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
		response.Unauthorized(c, "MISSING_TENANT_CONTEXT", "Tenant context could not be resolved")
		return
	}

	capabilities, err := h.svc.GetEffectiveCapabilities(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, ErrNoSubscription) {
			response.OK(c, CapabilitiesResponse{
				TenantID: tenantID,
				Plan: PlanSummary{
					Code:   "none",
					Name:   "No Active Plan",
					Status: StatusCanceled,
				},
				Capabilities:  make(map[string]Decision),
				PolicyVersion: 0,
			})
			return
		}

		reqLogger.Error("Failed to fetch tenant capabilities", "tenant_id", tenantID, "error", err.Error())
		response.InternalServerError(c, "CAPABILITIES_FETCH_FAILED", "Failed to retrieve tenant capabilities")
		return
	}

	response.OK(c, capabilities)
}
