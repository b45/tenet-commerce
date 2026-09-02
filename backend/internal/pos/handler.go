package pos

import (
	"errors"
	"strconv"
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
		internalAuth.RequirePermission("inventory:read"),
		h.GetProducts,
	)
	rg.POST("/checkout",
		internalAuth.RequirePermission("pos:checkout"),
		pkgRedis.IdempotencyMiddleware(rdb, 24*time.Hour),
		h.Checkout,
	)
	rg.GET("/orders",
		internalAuth.RequirePermission("pos:read"),
		h.GetOrders,
	)
	rg.GET("/orders/:id",
		internalAuth.RequirePermission("pos:read"),
		h.GetOrderDetail,
	)
	rg.POST("/orders/:id/void",
		internalAuth.RequirePermission("pos:void"),
		pkgRedis.IdempotencyMiddleware(rdb, 24*time.Hour),
		h.VoidOrder,
	)
	rg.GET("/daily-summary",
		internalAuth.RequirePermission("pos:read"),
		h.GetDailySummary,
	)
	rg.GET("/qris",
		internalAuth.RequirePermission("inventory:read"),
		h.GetQRISConfig,
	)
	rg.PUT("/qris",
		internalAuth.RequirePermission("inventory:write"),
		h.UpdateQRISConfig,
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
		if errors.Is(err, ErrInsufficientCashTendered) {
			log.Warn("Checkout rejected: insufficient cash tendered", "error", err, "cashier_id", cashierID)
			response.BadRequest(c, "INSUFFICIENT_CASH_TENDERED", err.Error())
			return
		}
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

// GetOrders retrieves a paginated and filterable list of transactions
// GET /api/v1/pos/orders
func (h *Handler) GetOrders(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

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

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := OrderFilter{
		Limit:         limit,
		Offset:        offset,
		StartDate:     c.Query("start_date"),
		EndDate:       c.Query("end_date"),
		Status:        c.Query("status"),
		PaymentMethod: c.Query("payment_method"),
		Search:        c.Query("search"),
	}

	orders, total, err := h.service.GetOrders(c.Request.Context(), conn, filter)
	if err != nil {
		log.Error("Failed to retrieve order history", "error", err)
		response.InternalServerError(c, "ORDERS_FETCH_FAILED", err.Error())
		return
	}

	response.OKWithMeta(c, orders, response.Meta{
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// GetOrderDetail retrieves full details of a specific transaction including line items
// GET /api/v1/pos/orders/:id
func (h *Handler) GetOrderDetail(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

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

	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		response.BadRequest(c, "INVALID_ORDER_ID", "Order ID or Transaction Number is required")
		return
	}

	detail, err := h.service.GetOrderDetail(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrTransactionNotFound) {
			response.NotFound(c, "ORDER_NOT_FOUND", "Order not found")
			return
		}
		log.Error("Failed to retrieve order detail", "id", id, "error", err)
		response.InternalServerError(c, "ORDER_DETAIL_FAILED", err.Error())
		return
	}

	response.OK(c, detail)
}

// VoidOrder cancels a completed transaction, restores inventory, and posts reversal journal
// POST /api/v1/pos/orders/:id/void
func (h *Handler) VoidOrder(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

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
		response.Unauthorized(c, "UNAUTHORIZED", "User identity not found in token")
		return
	}

	txnID := c.Param("id")
	if strings.TrimSpace(txnID) == "" {
		response.BadRequest(c, "INVALID_ORDER_ID", "Transaction ID is required")
		return
	}

	var req VoidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid void request format", err.Error())
		return
	}

	voidRes, err := h.service.VoidTransaction(c.Request.Context(), conn, cashierID, txnID, req)
	if err != nil {
		if errors.Is(err, ErrTransactionNotFound) {
			response.NotFound(c, "TRANSACTION_NOT_FOUND", "Transaction not found")
			return
		}
		if errors.Is(err, ErrAlreadyVoided) {
			response.Conflict(c, "TRANSACTION_ALREADY_VOIDED", "Transaction is already voided")
			return
		}
		log.Error("Failed voiding transaction", "txn_id", txnID, "error", err)
		response.InternalServerError(c, "VOID_FAILED", err.Error())
		return
	}

	response.OK(c, voidRes)
}

// GetDailySummary aggregates sales and payment breakdown for a date
// GET /api/v1/pos/daily-summary
func (h *Handler) GetDailySummary(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

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

	date := c.Query("date")
	cashierIDParam := c.Query("cashier_id")
	var cashierID *string
	if cashierIDParam != "" {
		cashierID = &cashierIDParam
	}

	summary, err := h.service.GetDailySummary(c.Request.Context(), conn, date, cashierID)
	if err != nil {
		log.Error("Failed retrieving daily POS summary", "error", err)
		response.InternalServerError(c, "DAILY_SUMMARY_FAILED", err.Error())
		return
	}

	response.OK(c, summary)
}

// GetQRISConfig returns active tenant's QRIS details for customer display
// GET /api/v1/pos/qris
func (h *Handler) GetQRISConfig(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

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

	qris, err := h.service.GetQRISConfig(c.Request.Context(), conn)
	if err != nil {
		log.Error("Failed retrieving QRIS config", "error", err)
		response.InternalServerError(c, "QRIS_FETCH_FAILED", err.Error())
		return
	}

	response.OK(c, qris)
}

// UpdateQRISConfig allows updating the active tenant's QRIS details
// PUT /api/v1/pos/qris
func (h *Handler) UpdateQRISConfig(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

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

	var cfg QRISConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Invalid QRIS config format", err.Error())
		return
	}

	if strings.TrimSpace(cfg.MerchantName) == "" || strings.TrimSpace(cfg.QRString) == "" {
		response.BadRequest(c, "VALIDATION_ERROR", "merchant_name and qr_string are required")
		return
	}

	if err := h.service.UpdateQRISConfig(c.Request.Context(), conn, cfg); err != nil {
		log.Error("Failed updating QRIS config", "error", err)
		response.InternalServerError(c, "QRIS_UPDATE_FAILED", err.Error())
		return
	}

	response.OK(c, gin.H{
		"message": "QRIS configuration updated successfully",
		"qris":    cfg,
	})
}

