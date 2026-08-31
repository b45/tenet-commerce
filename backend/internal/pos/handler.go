package pos

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
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

// GetProducts returns the product catalog with stock levels for the active tenant
// GET /api/v1/pos/products
func (h *Handler) GetProducts(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}

	conn, ok := connVal.(*pgxpool.Conn)
	if !ok {
		response.InternalServerError(c, "DATABASE_TYPE_ERROR", "Invalid connection context type")
		return
	}

	products, err := h.service.GetCatalog(c.Request.Context(), conn)
	if err != nil {
		response.InternalServerError(c, "CATALOG_FETCH_FAILED", "Failed to retrieve product catalog")
		return
	}

	response.OKWithMeta(c, products, response.Meta{
		Total: len(products),
	})
}

// Checkout processes a sales transaction atomically with inventory decrement and receipt generation
// POST /api/v1/pos/checkout
func (h *Handler) Checkout(c *gin.Context) {
	connVal, exists := c.Get("db_conn")
	if !exists {
		response.InternalServerError(c, "DATABASE_CONTEXT_LOST", "Database connection context not found")
		return
	}

	conn, ok := connVal.(*pgxpool.Conn)
	if !ok {
		response.InternalServerError(c, "DATABASE_TYPE_ERROR", "Invalid connection context type")
		return
	}

	cashierID := c.GetString("user_id")
	if cashierID == "" {
		response.Unauthorized(c, "UNAUTHORIZED_CASHIER", "Cashier identity not found in token")
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		response.BadRequest(c, "MISSING_IDEMPOTENCY_KEY", "Idempotency-Key header is required")
		return
	}

	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid checkout request format", err.Error())
		return
	}

	receipt, err := h.service.Checkout(c.Request.Context(), conn, cashierID, idempotencyKey, req)
	if err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			response.Conflict(c, "INSUFFICIENT_STOCK", err.Error())
			return
		}
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "PRODUCT_NOT_FOUND", err.Error())
			return
		}
		response.InternalServerError(c, "CHECKOUT_FAILED", err.Error())
		return
	}

	response.Created(c, receipt)
}
