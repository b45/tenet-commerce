package supplychain

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	internalAuth "github.com/b45/tenet-commerce/backend/internal/auth"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/b45/tenet-commerce/backend/pkg/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes sets up the HTTP endpoints for the supply chain module with RBAC guards
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.Use(internalAuth.RequirePermission("supply_chain:manage"))
	{
		rg.POST("/suppliers", h.CreateSupplier)
		rg.POST("/purchase-orders", h.CreatePurchaseOrder)
		rg.POST("/goods-receipts", h.CreateGoodsReceipt)
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
