package pos

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	pkgIdempotency "github.com/b45/tenet-commerce/backend/pkg/idempotency"
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
	// Product Catalog & CRUD
	rg.GET("/products",
		internalAuth.RequirePermission("inventory:read"),
		h.GetProducts,
	)
	rg.GET("/products/:id",
		internalAuth.RequirePermission("inventory:read"),
		h.GetProductByID,
	)
	rg.POST("/products",
		internalAuth.RequirePermission("inventory:write"),
		h.CreateProduct,
	)
	rg.PUT("/products/:id",
		internalAuth.RequirePermission("inventory:write"),
		h.UpdateProduct,
	)
	rg.DELETE("/products/:id",
		internalAuth.RequirePermission("inventory:write"),
		h.DeleteProduct,
	)

	// Category Management
	rg.GET("/categories",
		internalAuth.RequirePermission("inventory:read"),
		h.GetCategories,
	)
	rg.GET("/categories/:id",
		internalAuth.RequirePermission("inventory:read"),
		h.GetCategoryByID,
	)
	rg.POST("/categories",
		internalAuth.RequirePermission("inventory:write"),
		h.CreateCategory,
	)
	rg.PUT("/categories/:id",
		internalAuth.RequirePermission("inventory:write"),
		h.UpdateCategory,
	)
	rg.DELETE("/categories/:id",
		internalAuth.RequirePermission("inventory:write"),
		h.DeleteCategory,
	)

	// Inventory Stock Adjustment (Stock Opname & Spoilage Write-Offs)
	rg.POST("/inventory/adjust",
		internalAuth.RequirePermission("inventory:write"),
		pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
		h.AdjustStock,
	)
	rg.GET("/inventory/low-stock",
		internalAuth.RequirePermission("inventory:read"),
		h.GetLowStock,
	)

	// Checkout & Orders
	rg.POST("/checkout",
		internalAuth.RequirePermission("pos:checkout"),
		pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
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
		pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
		h.VoidOrder,
	)
	rg.GET("/daily-summary",
		internalAuth.RequirePermission("pos:read"),
		h.GetDailySummary,
	)

	// QRIS Configuration
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
		if errors.Is(err, ErrCashTenderedNotAllowed) {
			log.Warn("Checkout rejected: cash tendered supplied for non-cash payment", "cashier_id", cashierID)
			response.BadRequest(c, "INVALID_CASH_TENDERED", err.Error())
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

// GetProductByID retrieves single product specification and real-time inventory
// GET /api/v1/pos/products/:id
func (h *Handler) GetProductByID(c *gin.Context) {
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
	product, err := h.service.GetProduct(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "PRODUCT_NOT_FOUND", "Product not found")
			return
		}
		log.Error("Failed retrieving product by ID", "id", id, "error", err)
		response.InternalServerError(c, "PRODUCT_FETCH_FAILED", err.Error())
		return
	}

	response.OK(c, product)
}

// CreateProduct adds a new product to the catalog with initial inventory
// POST /api/v1/pos/products
func (h *Handler) CreateProduct(c *gin.Context) {
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

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}

	product, err := h.service.CreateProduct(c.Request.Context(), conn, req)
	if err != nil {
		if errors.Is(err, ErrSKUAlreadyExists) {
			response.Conflict(c, "SKU_ALREADY_EXISTS", "Product SKU already exists")
			return
		}
		if errors.Is(err, ErrBarcodeAlreadyExists) {
			response.Conflict(c, "BARCODE_ALREADY_EXISTS", "Product barcode already exists")
			return
		}
		log.Error("Failed creating product", "sku", req.SKU, "error", err)
		response.InternalServerError(c, "PRODUCT_CREATE_FAILED", err.Error())
		return
	}

	response.Created(c, product)
}

// UpdateProduct updates product metadata and reorder levels
// PUT /api/v1/pos/products/:id
func (h *Handler) UpdateProduct(c *gin.Context) {
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
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}

	product, err := h.service.UpdateProduct(c.Request.Context(), conn, id, req)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "PRODUCT_NOT_FOUND", "Product not found")
			return
		}
		if errors.Is(err, ErrBarcodeAlreadyExists) {
			response.Conflict(c, "BARCODE_ALREADY_EXISTS", "Product barcode already exists")
			return
		}
		log.Error("Failed updating product", "id", id, "error", err)
		response.InternalServerError(c, "PRODUCT_UPDATE_FAILED", err.Error())
		return
	}

	response.OK(c, product)
}

// DeleteProduct soft-deletes a product by setting is_active = FALSE
// DELETE /api/v1/pos/products/:id
func (h *Handler) DeleteProduct(c *gin.Context) {
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
	if err := h.service.DeleteProduct(c.Request.Context(), conn, id); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "PRODUCT_NOT_FOUND", "Product not found")
			return
		}
		log.Error("Failed soft-deleting product", "id", id, "error", err)
		response.InternalServerError(c, "PRODUCT_DELETE_FAILED", err.Error())
		return
	}

	response.OK(c, gin.H{
		"message": "Product soft-deleted successfully",
		"id":      id,
	})
}

// GetCategories returns all product categories with product counts
// GET /api/v1/pos/categories
func (h *Handler) GetCategories(c *gin.Context) {
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

	categories, err := h.service.GetCategories(c.Request.Context(), conn)
	if err != nil {
		log.Error("Failed fetching categories", "error", err)
		response.InternalServerError(c, "CATEGORIES_FETCH_FAILED", err.Error())
		return
	}

	response.OKWithMeta(c, categories, response.Meta{
		Total: len(categories),
	})
}

// GetCategoryByID returns a category by ID
// GET /api/v1/pos/categories/:id
func (h *Handler) GetCategoryByID(c *gin.Context) {
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
	category, err := h.service.GetCategoryByID(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.NotFound(c, "CATEGORY_NOT_FOUND", "Category not found")
			return
		}
		log.Error("Failed fetching category by ID", "id", id, "error", err)
		response.InternalServerError(c, "CATEGORY_FETCH_FAILED", err.Error())
		return
	}

	response.OK(c, category)
}

// CreateCategory adds a new product category
// POST /api/v1/pos/categories
func (h *Handler) CreateCategory(c *gin.Context) {
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

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}

	category, err := h.service.CreateCategory(c.Request.Context(), conn, req)
	if err != nil {
		if errors.Is(err, ErrCategoryCodeExists) {
			response.Conflict(c, "CATEGORY_CODE_EXISTS", "Category code already exists")
			return
		}
		log.Error("Failed creating category", "code", req.Code, "error", err)
		response.InternalServerError(c, "CATEGORY_CREATE_FAILED", err.Error())
		return
	}

	response.Created(c, category)
}

// UpdateCategory modifies an existing category
// PUT /api/v1/pos/categories/:id
func (h *Handler) UpdateCategory(c *gin.Context) {
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
	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}

	category, err := h.service.UpdateCategory(c.Request.Context(), conn, id, req)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.NotFound(c, "CATEGORY_NOT_FOUND", "Category not found")
			return
		}
		if errors.Is(err, ErrCategoryCodeExists) {
			response.Conflict(c, "CATEGORY_CODE_EXISTS", "Category code already exists")
			return
		}
		log.Error("Failed updating category", "id", id, "error", err)
		response.InternalServerError(c, "CATEGORY_UPDATE_FAILED", err.Error())
		return
	}

	response.OK(c, category)
}

// DeleteCategory deletes a category unlinking any assigned products
// DELETE /api/v1/pos/categories/:id
func (h *Handler) DeleteCategory(c *gin.Context) {
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
	if err := h.service.DeleteCategory(c.Request.Context(), conn, id); err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			response.NotFound(c, "CATEGORY_NOT_FOUND", "Category not found")
			return
		}
		log.Error("Failed deleting category", "id", id, "error", err)
		response.InternalServerError(c, "CATEGORY_DELETE_FAILED", err.Error())
		return
	}

	response.OK(c, gin.H{
		"message": "Category deleted successfully",
		"id":      id,
	})
}

// AdjustStock adjusts product stock level and records Sharia shrinkage journals if write-off
// POST /api/v1/pos/inventory/adjust
func (h *Handler) AdjustStock(c *gin.Context) {
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

	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c, "UNAUTHORIZED", "User identity not found in token")
		return
	}

	var req StockAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}

	res, err := h.service.AdjustStock(c.Request.Context(), conn, userID, req)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.NotFound(c, "PRODUCT_NOT_FOUND", "Product not found")
			return
		}
		if errors.Is(err, ErrNegativeAdjustmentStock) {
			response.BadRequest(c, "INSUFFICIENT_STOCK", "Insufficient stock for negative adjustment")
			return
		}
		log.Error("Failed adjusting stock", "product_id", req.ProductID, "error", err)
		response.InternalServerError(c, "STOCK_ADJUST_FAILED", err.Error())
		return
	}

	response.OK(c, res)
}

// GetLowStock retrieves all products at or below their reorder threshold
// GET /api/v1/pos/inventory/low-stock
func (h *Handler) GetLowStock(c *gin.Context) {
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

	products, err := h.service.GetLowStock(c.Request.Context(), conn)
	if err != nil {
		log.Error("Failed querying low stock products", "error", err)
		response.InternalServerError(c, "LOW_STOCK_FETCH_FAILED", err.Error())
		return
	}

	response.OKWithMeta(c, products, response.Meta{
		Total: len(products),
	})
}
