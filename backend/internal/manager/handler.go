package manager

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// Handler handles HTTP requests for the manager domain
type Handler struct {
	service *Service
}

// NewHandler initializes a new Manager HTTP handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts manager endpoints onto the given router group with RBAC role guards
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(internalAuth.RequireRole("MANAGER", "SUPER_ADMIN"))
	{
		rg.GET("/dashboard", h.GetDashboardSummary)
	}
}

// GetDashboardSummary returns aggregated metrics across sales, inventory, and compliance
// GET /api/v1/manager/dashboard
func (h *Handler) GetDashboardSummary(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during manager dashboard fetch")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}

	conn, ok := connVal.(*pgxpool.Conn)
	if !ok {
		log.Error("Invalid database connection type in context")
		response.InternalServerError(c, "DATABASE_TYPE_ERROR", "Invalid connection context type")
		return
	}

	summary, err := h.service.GetDashboardSummary(c.Request.Context(), conn)
	if err != nil {
		log.Error("Failed to generate manager dashboard summary", "error", err)
		response.InternalServerError(c, "DASHBOARD_FETCH_FAILED", err.Error())
		return
	}

	log.Info("Manager dashboard summary aggregated successfully",
		"today_orders", summary.SalesSummary.TodayOrdersCount,
		"today_gross_sales", summary.SalesSummary.TodayGrossSales,
		"low_stock_count", summary.InventoryAlerts.LowStockCount,
		"expiring_certs", summary.ComplianceAlerts.ExpiringCertificatesCount,
	)

	response.OK(c, summary)
}
