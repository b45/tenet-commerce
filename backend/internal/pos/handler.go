package pos

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

// Handler handles HTTP requests for the POS domain
type Handler struct {
	service *Service
}

// NewHandler initializes a new POS HTTP handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes mounts all POS endpoints with RBAC and idempotency middleware
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rdb *pkgRedis.Client) {
	rg.GET("/products",
		internalAuth.RequirePermission("pos:read"),
		h.GetProducts,
	)
	rg.POST("/checkout",
		internalAuth.RequirePermission("pos:checkout"),
		pkgRedis.IdempotencyMiddleware(rdb, 24*time.Hour),
		h.Checkout,
	)
}

// GetProducts returns the product catalog with stock levels for the active tenant
// GET /api/v1/pos/products
func (h *Handler) GetProducts(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during catalog fetch")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}

	conn, ok := connVal.(*pgxpool.Conn)
	if !ok {
		log.Error("Invalid database connection type in context")
		response.InternalServerError(c, "DATABASE_TYPE_ERROR", "Invalid connection context type")
		return
	}

	products, err := h.service.GetCatalog(c.Request.Context(), conn)
	if err != nil {
		log.Error("Failed to retrieve product catalog", "error", err)
		response.InternalServerError(c, "CATALOG_FETCH_FAILED", "Failed to retrieve product catalog")
		return
	}

	log.Debug("Product catalog retrieved successfully", "count", len(products))
	response.OKWithMeta(c, products, response.Meta{
		Total: len(products),
	})
}

// Checkout processes a sales transaction atomically with inventory decrement and receipt generation
// POST /api/v1/pos/checkout
func (h *Handler) Checkout(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	connVal, exists := c.Get("db_conn")
	if !exists {
		log.Error("Database connection context not found during checkout")
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}

	conn, ok := connVal.(*pgxpool.Conn)
	if !ok {
		log.Error("Invalid database connection type in context during checkout")
		response.InternalServerError(c, "DATABASE_TYPE_ERROR", "Invalid connection context type")
		return
	}

	cashierID := c.GetString("user_id")
	if cashierID == "" {
		log.Warn("Checkout attempted without authenticated cashier identity")
		response.Unauthorized(c, "UNAUTHORIZED_CASHIER", "Cashier identity not found in token")
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		log.Warn("Checkout attempted without Idempotency-Key header", "cashier_id", cashierID)
		response.BadRequest(c, "MISSING_IDEMPOTENCY_KEY", "Idempotency-Key header is required")
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Invalid checkout payload format", "error", err, "cashier_id", cashierID)
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid checkout request format", err.Error())
		return
	}

	receipt, err := h.service.Checkout(c.Request.Context(), conn, cashierID, idempotencyKey, req)
	if err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			log.Warn("Checkout rejected due to insufficient stock", "error", err, "cashier_id", cashierID)
			response.Conflict(c, "INSUFFICIENT_STOCK", err.Error())
			return
		}
		if errors.Is(err, ErrProductNotFound) {
			log.Warn("Checkout rejected: product not found", "error", err, "cashier_id", cashierID)
			response.NotFound(c, "PRODUCT_NOT_FOUND", err.Error())
			return
		}
		log.Error("Checkout processing failed", "error", err, "cashier_id", cashierID)
		response.InternalServerError(c, "CHECKOUT_FAILED", err.Error())
		return
	}

	log.Info("POS checkout completed successfully",
		"transaction_id", receipt.TransactionID,
		"transaction_number", receipt.TransactionNumber,
		"total_amount", receipt.TotalAmount,
		"cashier_id", cashierID,
		"payment_method", receipt.PaymentMethod,
	)

	response.Created(c, receipt)
}
