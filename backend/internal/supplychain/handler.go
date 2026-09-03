package supplychain

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgIdempotency "github.com/b45/tenet-commerce/backend/pkg/idempotency"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	pkgRedis "github.com/b45/tenet-commerce/backend/pkg/redis"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes sets up the HTTP endpoints for the supply chain module with RBAC guards
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rdb *pkgRedis.Client) {
	rg.Use(internalAuth.RequirePermission("supply_chain:manage"))
	{
		// Suppliers
		rg.GET("/suppliers", h.ListSuppliers)
		rg.GET("/suppliers/:id", h.GetSupplier)
		rg.POST("/suppliers", h.CreateSupplier)
		rg.PUT("/suppliers/:id", h.UpdateSupplier)

		// Certificates
		rg.GET("/suppliers/:id/certificates", h.GetSupplierCertificates)
		rg.POST("/suppliers/:id/certificates", h.RegisterCertificate)
		rg.PUT("/certificates/:id/revoke", h.RevokeCertificate)

		// Purchase Orders
		rg.GET("/purchase-orders", h.ListPurchaseOrders)
		rg.GET("/purchase-orders/:id", h.GetPurchaseOrderDetail)
		rg.POST("/purchase-orders",
			pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
			h.CreatePurchaseOrder,
		)
		rg.PUT("/purchase-orders/:id/cancel", h.CancelPurchaseOrder)

		// Goods Receipts
		rg.GET("/goods-receipts", h.ListGoodsReceipts)
		rg.GET("/goods-receipts/:id", h.GetGoodsReceiptDetail)
		rg.POST("/goods-receipts",
			pkgIdempotency.DurableIdempotencyMiddleware(rdb, 24*time.Hour),
			h.CreateGoodsReceipt,
		)

		// Document Traceability
		rg.GET("/traceability/product/:product_id", h.GetProductTraceability)
	}
}

func (h *Handler) CreateSupplier(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	var req CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Supplier creation validation failed", "error", err)
		response.AbortBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	conn, exists := tenant.GetConn(c)
	if !exists {
		log.Error("Database connection context not found during supplier creation")
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	supplier, err := h.service.CreateSupplier(c.Request.Context(), conn, &req)
	if err != nil {
		log.Error("Failed to create supplier", "error", err, "company_name", req.CompanyName)
		response.AbortInternalServerError(c, "SUPPLIER_CREATION_FAILED", err.Error())
		return
	}

	log.Info("Supplier created successfully", "supplier_id", supplier.ID, "company_name", supplier.CompanyName, "code", supplier.Code)
	response.Created(c, supplier)
}

func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	var req CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Purchase order validation failed", "error", err)
		response.AbortBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	conn, exists := tenant.GetConn(c)
	if !exists {
		log.Error("Database connection context not found during PO creation")
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	po, err := h.service.CreatePurchaseOrder(c.Request.Context(), conn, &req)
	if err != nil {
		if errors.Is(err, ErrComplianceCertRequired) || errors.Is(err, ErrComplianceCertExpired) {
			log.Warn("Purchase order creation hard-blocked by Halal compliance engine", "supplier_id", req.SupplierID, "error", err)
			response.UnprocessableEntity(c, "COMPLIANCE_ERROR", err.Error())
			c.Abort()
			return
		}
		log.Error("Failed to create purchase order", "error", err, "supplier_id", req.SupplierID)
		response.AbortInternalServerError(c, "PO_CREATION_FAILED", err.Error())
		return
	}

	log.Info("Purchase order created successfully",
		"po_id", po.ID,
		"po_number", po.PONumber,
		"supplier_id", po.SupplierID,
		"total_amount", po.TotalAmount,
	)
	response.Created(c, po)
}

func (h *Handler) CreateGoodsReceipt(c *gin.Context) {
	log := logger.FromContext(c.Request.Context())

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		log.Warn("Goods receipt creation attempted without Idempotency-Key header")
		response.AbortBadRequest(c, "MISSING_IDEMPOTENCY_KEY", "Idempotency-Key header is required")
		return
	}

	var req CreateGoodsReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("Goods receipt validation failed", "error", err)
		response.AbortBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	conn, exists := tenant.GetConn(c)
	if !exists {
		log.Error("Database connection context not found during Goods Receipt creation")
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		log.Warn("Goods receipt creation attempted without user ID in context")
		response.AbortUnauthorized(c, "UNAUTHORIZED", "Missing user ID in context")
		return
	}
	userID, _ := uuid.Parse(userIDVal.(string))

	gr, err := h.service.CreateGoodsReceipt(c.Request.Context(), conn, userID, idempotencyKey, &req)
	if err != nil {
		if errors.Is(err, ErrEmptyReceipt) || errors.Is(err, ErrZeroValueReceipt) || errors.Is(err, ErrDuplicateReceiptItem) {
			log.Warn("Goods receipt validation rejected", "error", err)
			response.AbortBadRequest(c, "INVALID_RECEIPT_ITEMS", err.Error())
			return
		}
		if errors.Is(err, ErrReceiptItemNotOnPO) || errors.Is(err, ErrReceiptQuantityExceeds) {
			log.Warn("Goods receipt line reconciliation failed", "error", err)
			response.UnprocessableEntity(c, "RECEIPT_RECONCILIATION_FAILED", err.Error())
			c.Abort()
			return
		}
		if errors.Is(err, ErrComplianceCertRequired) || errors.Is(err, ErrComplianceCertExpired) || errors.Is(err, ErrComplianceCertInvalid) {
			log.Warn("Goods receipt creation hard-blocked by Halal compliance engine", "po_id", req.PurchaseOrderID, "error", err)
			response.UnprocessableEntity(c, "COMPLIANCE_ERROR", err.Error())
			c.Abort()
			return
		}
		if errors.Is(err, ErrInvalidPOStatus) {
			log.Warn("Goods receipt rejected: invalid PO status", "po_id", req.PurchaseOrderID, "error", err)
			response.AbortConflict(c, "INVALID_PO_STATUS", err.Error())
			return
		}
		if errors.Is(err, ErrIdempotencyKeyConflict) {
			log.Warn("Goods receipt rejected: idempotency key conflict", "idempotency_key", idempotencyKey, "error", err)
			response.AbortConflict(c, "IDEMPOTENCY_KEY_CONFLICT", err.Error())
			return
		}
		log.Error("Failed to process Goods Receipt", "po_id", req.PurchaseOrderID, "error", err)
		response.AbortInternalServerError(c, "GR_CREATION_FAILED", err.Error())
		return
	}

	log.Info("Goods receipt processed and stock restocked successfully",
		"gr_id", gr.ID,
		"gr_number", gr.GRNumber,
		"po_id", gr.PurchaseOrderID,
	)
	response.Created(c, gr)
}

// ListSuppliers handles GET /api/v1/supply-chain/suppliers
func (h *Handler) ListSuppliers(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	var isActive *bool
	if activeStr := c.Query("is_active"); activeStr != "" {
		b := strings.ToLower(activeStr) == "true" || activeStr == "1"
		isActive = &b
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	suppliers, err := h.service.ListSuppliers(c.Request.Context(), conn, isActive, limit, offset)
	if err != nil {
		response.AbortInternalServerError(c, "SUPPLIER_LIST_FAILED", err.Error())
		return
	}
	response.OK(c, suppliers)
}

// GetSupplier handles GET /api/v1/supply-chain/suppliers/:id
func (h *Handler) GetSupplier(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_SUPPLIER_ID", "Invalid supplier ID format")
		return
	}

	supplier, err := h.service.GetSupplier(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "SUPPLIER_NOT_FOUND", "Supplier not found")
			return
		}
		response.AbortInternalServerError(c, "SUPPLIER_FETCH_FAILED", err.Error())
		return
	}
	response.OK(c, supplier)
}

// UpdateSupplier handles PUT /api/v1/supply-chain/suppliers/:id
func (h *Handler) UpdateSupplier(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_SUPPLIER_ID", "Invalid supplier ID format")
		return
	}

	var req UpdateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	supplier, err := h.service.UpdateSupplier(c.Request.Context(), conn, id, &req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "SUPPLIER_NOT_FOUND", "Supplier not found")
			return
		}
		response.AbortInternalServerError(c, "SUPPLIER_UPDATE_FAILED", err.Error())
		return
	}
	response.OK(c, supplier)
}

// GetSupplierCertificates handles GET /api/v1/supply-chain/suppliers/:id/certificates
func (h *Handler) GetSupplierCertificates(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_SUPPLIER_ID", "Invalid supplier ID format")
		return
	}

	certs, err := h.service.GetSupplierCertificates(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "SUPPLIER_NOT_FOUND", "Supplier not found")
			return
		}
		response.AbortInternalServerError(c, "CERTIFICATES_FETCH_FAILED", err.Error())
		return
	}
	response.OK(c, certs)
}

// RegisterCertificate handles POST /api/v1/supply-chain/suppliers/:id/certificates
func (h *Handler) RegisterCertificate(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_SUPPLIER_ID", "Invalid supplier ID format")
		return
	}

	var req CreateComplianceCertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	cert, err := h.service.RegisterCertificate(c.Request.Context(), conn, id, &req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "SUPPLIER_NOT_FOUND", "Supplier not found")
			return
		}
		response.BadRequest(c, "CERTIFICATE_REGISTRATION_FAILED", err.Error())
		return
	}
	response.Created(c, cert)
}

// RevokeCertificate handles PUT /api/v1/supply-chain/certificates/:id/revoke
func (h *Handler) RevokeCertificate(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_CERTIFICATE_ID", "Invalid certificate ID format")
		return
	}

	if err := h.service.RevokeCertificate(c.Request.Context(), conn, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "CERTIFICATE_NOT_FOUND", "Certificate not found")
			return
		}
		response.AbortInternalServerError(c, "CERTIFICATE_REVOCATION_FAILED", err.Error())
		return
	}
	response.OK(c, gin.H{"revoked": true})
}

// ListPurchaseOrders handles GET /api/v1/supply-chain/purchase-orders
func (h *Handler) ListPurchaseOrders(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	pos, err := h.service.ListPurchaseOrders(c.Request.Context(), conn, status, limit, offset)
	if err != nil {
		response.AbortInternalServerError(c, "PO_LIST_FAILED", err.Error())
		return
	}
	response.OK(c, pos)
}

// GetPurchaseOrderDetail handles GET /api/v1/supply-chain/purchase-orders/:id
func (h *Handler) GetPurchaseOrderDetail(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_PO_ID", "Invalid purchase order ID format")
		return
	}

	poDetail, err := h.service.GetPurchaseOrderDetail(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "PO_NOT_FOUND", "Purchase order not found")
			return
		}
		response.AbortInternalServerError(c, "PO_FETCH_FAILED", err.Error())
		return
	}
	response.OK(c, poDetail)
}

// CancelPurchaseOrder handles PUT /api/v1/supply-chain/purchase-orders/:id/cancel
func (h *Handler) CancelPurchaseOrder(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_PO_ID", "Invalid purchase order ID format")
		return
	}

	if err := h.service.CancelPurchaseOrder(c.Request.Context(), conn, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "PO_NOT_FOUND", "Purchase order not found")
			return
		}
		response.UnprocessableEntity(c, "PO_CANCELLATION_FAILED", err.Error())
		return
	}
	response.OK(c, gin.H{"cancelled": true})
}

// ListGoodsReceipts handles GET /api/v1/supply-chain/goods-receipts
func (h *Handler) ListGoodsReceipts(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	grs, err := h.service.ListGoodsReceipts(c.Request.Context(), conn, limit, offset)
	if err != nil {
		response.AbortInternalServerError(c, "GR_LIST_FAILED", err.Error())
		return
	}
	response.OK(c, grs)
}

// GetGoodsReceiptDetail handles GET /api/v1/supply-chain/goods-receipts/:id
func (h *Handler) GetGoodsReceiptDetail(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "INVALID_GR_ID", "Invalid goods receipt ID format")
		return
	}

	grDetail, err := h.service.GetGoodsReceiptDetail(c.Request.Context(), conn, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "GR_NOT_FOUND", "Goods receipt not found")
			return
		}
		response.AbortInternalServerError(c, "GR_FETCH_FAILED", err.Error())
		return
	}
	response.OK(c, grDetail)
}

// GetProductTraceability handles GET /api/v1/supply-chain/traceability/product/:product_id
func (h *Handler) GetProductTraceability(c *gin.Context) {
	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	productID, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		response.BadRequest(c, "INVALID_PRODUCT_ID", "Invalid product ID format")
		return
	}

	report, err := h.service.GetProductTraceability(c.Request.Context(), conn, productID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.NotFound(c, "PRODUCT_NOT_FOUND", "Product not found")
			return
		}
		response.AbortInternalServerError(c, "TRACEABILITY_FAILED", err.Error())
		return
	}
	response.OK(c, report)
}
