package supplychain

import (
	"errors"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	"github.com/b45/tenet-commerce/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes sets up the HTTP endpoints for the supply chain module
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/supply-chain")
	{
		group.POST("/suppliers", h.CreateSupplier)
		group.POST("/purchase-orders", h.CreatePurchaseOrder)
		group.POST("/goods-receipts", h.CreateGoodsReceipt)
	}
}

func (h *Handler) CreateSupplier(c *gin.Context) {
	var req CreateSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	supplier, err := h.service.CreateSupplier(c.Request.Context(), conn, &req)
	if err != nil {
		response.AbortInternalServerError(c, "SUPPLIER_CREATION_FAILED", err.Error())
		return
	}

	response.Created(c, supplier)
}

func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	var req CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	po, err := h.service.CreatePurchaseOrder(c.Request.Context(), conn, &req)
	if err != nil {
		if errors.Is(err, ErrComplianceCertRequired) || errors.Is(err, ErrComplianceCertExpired) {
			response.UnprocessableEntity(c, "COMPLIANCE_ERROR", err.Error())
			c.Abort()
			return
		}
		response.AbortInternalServerError(c, "PO_CREATION_FAILED", err.Error())
		return
	}

	response.Created(c, po)
}

func (h *Handler) CreateGoodsReceipt(c *gin.Context) {
	var req CreateGoodsReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, "INVALID_REQUEST", err.Error())
		return
	}

	conn, exists := tenant.GetConn(c)
	if !exists {
		response.AbortInternalServerError(c, "TENANT_CONN_NOT_FOUND", "Database connection not available")
		return
	}

	userIDVal, exists := c.Get("user_id")
	if !exists {
		response.AbortUnauthorized(c, "UNAUTHORIZED", "Missing user ID in context")
		return
	}
	userID, _ := uuid.Parse(userIDVal.(string))

	gr, err := h.service.CreateGoodsReceipt(c.Request.Context(), conn, userID, &req)
	if err != nil {
		if errors.Is(err, ErrComplianceCertRequired) || errors.Is(err, ErrComplianceCertExpired) {
			response.UnprocessableEntity(c, "COMPLIANCE_ERROR", err.Error())
			c.Abort()
			return
		}
		if errors.Is(err, ErrInvalidPOStatus) {
			response.AbortConflict(c, "INVALID_PO_STATUS", err.Error())
			return
		}
		response.AbortInternalServerError(c, "GR_CREATION_FAILED", err.Error())
		return
	}

	response.Created(c, gr)
}
