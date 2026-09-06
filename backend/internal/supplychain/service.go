package supplychain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/pkg/money"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrComplianceCertRequired = errors.New("compliance certificate is required under strict mode")
	ErrComplianceCertExpired  = errors.New("compliance certificate is expired")
	ErrComplianceCertInvalid  = errors.New("compliance certificate is not valid for this transaction")
	ErrSupplierInactive       = errors.New("supplier is inactive")
	ErrInvalidPOStatus        = errors.New("invalid purchase order status for this operation")
	ErrEmptyReceipt           = errors.New("goods receipt must contain at least one item")
	ErrZeroValueReceipt       = errors.New("goods receipt inbound valuation must be greater than zero")
	ErrReceiptItemNotOnPO     = errors.New("goods receipt item does not exist on purchase order")
	ErrDuplicateReceiptItem   = errors.New("goods receipt contains a duplicate product")
	ErrReceiptQuantityExceeds = errors.New("goods receipt quantity exceeds purchase order outstanding quantity")
	ErrIdempotencyKeyConflict = errors.New("idempotency key is already associated with another purchase order")
	ErrInvalidMonetaryAmount  = errors.New("invalid monetary amount: must be non-fractional, non-negative, and within bounds")
)

type Service struct {
	repo          *Repository
	ledgerService *ledger.Service
	now           func() time.Time
}

func NewService(repo *Repository, ledgerService *ledger.Service) *Service {
	return NewServiceWithClock(repo, ledgerService, time.Now)
}

// NewServiceWithClock allows deterministic boundary tests without changing global time.
// The clock must be safe for concurrent use and must not be derived from HTTP input.
func NewServiceWithClock(repo *Repository, ledgerService *ledger.Service, now func() time.Time) *Service {
	if now == nil {
		panic("supplychain: nil clock")
	}
	return &Service{repo: repo, ledgerService: ledgerService, now: now}
}

// CreateSupplier registers a supplier and optionally its compliance certificate
func (s *Service) CreateSupplier(ctx context.Context, conn *pgxpool.Conn, req *CreateSupplierRequest) (*Supplier, error) {
	supplier := &Supplier{
		ID:            uuid.New(),
		Code:          req.Code,
		CompanyName:   req.CompanyName,
		ContactPerson: req.ContactPerson,
		ContactEmail:  req.ContactEmail,
		ContactPhone:  req.ContactPhone,
		IsActive:      true,
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.CreateSupplier(ctx, tx, supplier); err != nil {
		return nil, err
	}

	if req.ComplianceCertificate != nil {
		validFrom, _ := time.Parse("2006-01-02", req.ComplianceCertificate.ValidFrom)
		expiryDate, _ := time.Parse("2006-01-02", req.ComplianceCertificate.ExpiryDate)

		cert := &ComplianceCertificate{
			ID:                uuid.New(),
			SupplierID:        supplier.ID,
			CertType:          req.ComplianceCertificate.CertType,
			CertificateNumber: req.ComplianceCertificate.CertificateNumber,
			IssuingAuthority:  req.ComplianceCertificate.IssuingAuthority,
			Scope:             req.ComplianceCertificate.Scope,
			ValidFrom:         validFrom,
			ExpiryDate:        expiryDate,
			DocumentURL:       req.ComplianceCertificate.DocumentURL,
		}

		if err := s.repo.CreateComplianceCertificate(ctx, tx, cert); err != nil {
			return nil, err
		}
		supplier.ComplianceCertificate = cert
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return supplier, nil
}

// CreatePurchaseOrder creates a PO, enforcing compliance rules
func (s *Service) CreatePurchaseOrder(ctx context.Context, conn *pgxpool.Conn, req *CreatePurchaseOrderRequest) (*PurchaseOrder, error) {
	supplierID, err := uuid.Parse(req.SupplierID)
	if err != nil {
		return nil, fmt.Errorf("parse supplier id: %w", err)
	}

	var certID *uuid.UUID
	if req.ComplianceCertID != nil {
		id, err := uuid.Parse(*req.ComplianceCertID)
		if err != nil {
			return nil, fmt.Errorf("parse compliance certificate id: %w", err)
		}
		certID = &id
	}

	po := &PurchaseOrder{
		ID:               uuid.New(),
		PONumber:         "PO-" + time.Now().Format("20060102150405") + "-" + uuid.NewString()[:8],
		SupplierID:       supplierID,
		ComplianceCertID: certID,
		Status:           "ISSUED",
		IssuedDate:       time.Now(),
	}

	totalMoney := money.IDR(0)
	for _, reqItem := range req.Items {
		productID, _ := uuid.Parse(reqItem.ProductID)
		if reqItem.Quantity <= 0 || reqItem.Quantity > money.MaxLineItemQuantity {
			return nil, fmt.Errorf("invalid quantity %d: must be between 1 and %d", reqItem.Quantity, money.MaxLineItemQuantity)
		}

		unitCostMoney, err := money.ValidateIDR(reqItem.UnitCost, money.MaxTransactionAmount)
		if err != nil {
			return nil, fmt.Errorf("%w: unit cost: %v", ErrInvalidMonetaryAmount, err)
		}

		subtotalMoney, err := unitCostMoney.Mul(int64(reqItem.Quantity))
		if err != nil {
			return nil, fmt.Errorf("%w: subtotal calculation: %v", ErrInvalidMonetaryAmount, err)
		}

		totalMoney, err = totalMoney.Add(subtotalMoney)
		if err != nil {
			return nil, fmt.Errorf("%w: total calculation: %v", ErrInvalidMonetaryAmount, err)
		}
		if totalMoney.Amount() > money.MaxTransactionAmount {
			return nil, fmt.Errorf("%w: total PO amount exceeds %d IDR", ErrInvalidMonetaryAmount, money.MaxTransactionAmount)
		}

		po.Items = append(po.Items, PurchaseOrderItem{
			ID:              uuid.New(),
			PurchaseOrderID: po.ID,
			ProductID:       productID,
			Quantity:        reqItem.Quantity,
			UnitCost:        unitCostMoney.ToFloat(),
			Subtotal:        subtotalMoney.ToFloat(),
		})
	}
	po.TotalAmount = totalMoney.ToFloat()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// COMPLIANCE INTERCEPTOR (HARD-BLOCK), evaluated inside the PO transaction.
	po.ComplianceEvaluation, err = s.checkCompliance(ctx, tx, supplierID, certID)
	if err != nil {
		return nil, err
	}

	startInsert := time.Now()
	if err := s.repo.CreatePurchaseOrder(ctx, tx, po); err != nil {
		return nil, err
	}
	if err := s.repo.recordComplianceDecision(ctx, tx, "PO", po.ID, po.ComplianceEvaluation); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "po_insert_completed", slog.Duration("duration_insert_po_ms", time.Since(startInsert)))

	startCommit := time.Now()
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "tx_commit_completed", slog.Duration("duration_commit_ms", time.Since(startCommit)))

	return po, nil
}

// CreateGoodsReceipt creates a GR, revalidates compliance, and applies inventory,
// PO state, and ledger effects within one serialized transaction.
func (s *Service) CreateGoodsReceipt(ctx context.Context, conn *pgxpool.Conn, userID uuid.UUID, idempotencyKey string, req *CreateGoodsReceiptRequest) (*GoodsReceipt, error) {
	poID, err := uuid.Parse(req.PurchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("parse purchase order id: %w", err)
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, ErrIdempotencyKeyConflict
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	po, err := s.repo.LockPurchaseOrder(ctx, tx, poID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetGoodsReceiptByIdempotencyKey(ctx, tx, idempotencyKey)
	if err == nil {
		if existing.PurchaseOrderID != po.ID {
			return nil, ErrIdempotencyKeyConflict
		}
		existing.ComplianceEvaluation, err = s.repo.getComplianceDecision(ctx, tx, "GR", existing.ID)
		return existing, err
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if po.Status != "ISSUED" && po.Status != "PARTIALLY_RECEIVED" {
		return nil, ErrInvalidPOStatus
	}

	// The certificate may have changed since the PO was issued, so enforce the
	// tenant's strict-compliance configuration within this receipt transaction.
	decision, err := s.checkCompliance(ctx, tx, po.SupplierID, po.ComplianceCertID)
	if err != nil {
		return nil, err
	}

	poItems, err := s.repo.GetPurchaseOrderItems(ctx, tx, po.ID)
	if err != nil {
		return nil, err
	}
	receivedQuantities, err := s.repo.GetReceivedQuantities(ctx, tx, po.ID)
	if err != nil {
		return nil, err
	}

	gr := &GoodsReceipt{
		ComplianceEvaluation: decision,
		ID:                   uuid.New(),
		GRNumber:             "GR-" + time.Now().Format("20060102150405") + "-" + uuid.NewString()[:8],
		IdempotencyKey:       idempotencyKey,
		PurchaseOrderID:      po.ID,
		ReceivedBy:           userID,
		ReceivedDate:         time.Now(),
		Notes:                req.Notes,
	}
	inboundValue, fullyReceived, err := reconcileReceiptItems(gr, req.Items, poItems, receivedQuantities)
	if err != nil {
		return nil, err
	}

	startInsert := time.Now()
	if err := s.repo.CreateGoodsReceipt(ctx, tx, gr); err != nil {
		return nil, err
	}
	if err := s.repo.recordComplianceDecision(ctx, tx, "GR", gr.ID, decision); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "gr_and_stock_update_completed", slog.Duration("duration_stock_increment_ms", time.Since(startInsert)))

	status := "PARTIALLY_RECEIVED"
	if fullyReceived {
		status = "RECEIVED"
	}
	if err := s.repo.UpdatePurchaseOrderStatus(ctx, tx, po.ID, status); err != nil {
		return nil, err
	}

	if err := s.ledgerService.PostGoodsReceiptJournal(ctx, tx, gr.ID, gr.GRNumber, inboundValue); err != nil {
		slog.ErrorContext(ctx, "failed_to_post_gr_journal", slog.Any("error", err))
		return nil, err
	}

	startCommit := time.Now()
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "tx_commit_completed", slog.Duration("duration_commit_ms", time.Since(startCommit)))

	return gr, nil
}

func reconcileReceiptItems(gr *GoodsReceipt, requested []CreateGRItemRequest, poItems []PurchaseOrderItem, received map[uuid.UUID]int) (float64, bool, error) {
	if len(requested) == 0 {
		return 0, false, ErrEmptyReceipt
	}

	poByProduct := make(map[uuid.UUID]PurchaseOrderItem, len(poItems))
	for _, poItem := range poItems {
		if _, exists := poByProduct[poItem.ProductID]; exists {
			return 0, false, fmt.Errorf("purchase order contains duplicate product %s", poItem.ProductID)
		}
		poByProduct[poItem.ProductID] = poItem
	}

	requestedByProduct := make(map[uuid.UUID]int, len(requested))
	inboundMoney := money.IDR(0)
	for _, requestItem := range requested {
		productID, err := uuid.Parse(requestItem.ProductID)
		if err != nil {
			return 0, false, fmt.Errorf("parse goods receipt product id: %w", err)
		}
		if _, exists := requestedByProduct[productID]; exists {
			return 0, false, ErrDuplicateReceiptItem
		}
		poItem, exists := poByProduct[productID]
		if !exists {
			return 0, false, ErrReceiptItemNotOnPO
		}
		if requestItem.ReceivedQuantity <= 0 || requestItem.ReceivedQuantity > poItem.Quantity-received[productID] {
			return 0, false, ErrReceiptQuantityExceeds
		}

		unitCostMoney, err := money.FromExactFloat(poItem.UnitCost, money.CurrencyIDR)
		if err != nil {
			return 0, false, fmt.Errorf("invalid purchase order unit cost: %w", err)
		}
		itemInboundMoney, err := unitCostMoney.Mul(int64(requestItem.ReceivedQuantity))
		if err != nil {
			return 0, false, fmt.Errorf("failed calculating inbound valuation: %w", err)
		}
		inboundMoney, err = inboundMoney.Add(itemInboundMoney)
		if err != nil {
			return 0, false, fmt.Errorf("failed aggregating inbound valuation: %w", err)
		}

		requestedByProduct[productID] = requestItem.ReceivedQuantity
		gr.Items = append(gr.Items, GoodsReceiptItem{
			ID:               uuid.New(),
			GoodsReceiptID:   gr.ID,
			ProductID:        productID,
			ReceivedQuantity: requestItem.ReceivedQuantity,
		})
	}

	if !inboundMoney.IsPositive() {
		return 0, false, ErrZeroValueReceipt
	}
	inboundValue := inboundMoney.ToFloat()

	for productID, poItem := range poByProduct {
		if received[productID]+requestedByProduct[productID] != poItem.Quantity {
			return inboundValue, false, nil
		}
	}
	return inboundValue, true, nil
}

// ListSuppliers returns a paginated list of suppliers with optional is_active filter
func (s *Service) ListSuppliers(ctx context.Context, conn *pgxpool.Conn, isActive *bool, limit, offset int) ([]Supplier, error) {
	return s.repo.ListSuppliers(ctx, conn, isActive, limit, offset)
}

// GetSupplier retrieves a single supplier along with all its compliance certificates
func (s *Service) GetSupplier(ctx context.Context, conn *pgxpool.Conn, id uuid.UUID) (*SupplierDetail, error) {
	return s.repo.GetSupplierWithCertificates(ctx, conn, id)
}

// UpdateSupplier updates supplier contact and active status
func (s *Service) UpdateSupplier(ctx context.Context, conn *pgxpool.Conn, id uuid.UUID, req *UpdateSupplierRequest) (*Supplier, error) {
	return s.repo.UpdateSupplier(ctx, conn, id, req)
}

// GetSupplierCertificates returns all certificates for a supplier
func (s *Service) GetSupplierCertificates(ctx context.Context, conn *pgxpool.Conn, supplierID uuid.UUID) ([]ComplianceCertificate, error) {
	// Verify supplier exists first
	_, err := s.repo.GetSupplierByID(ctx, conn, supplierID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCertificatesBySupplierID(ctx, conn, supplierID)
}

// RegisterCertificate adds a new or renewal compliance certificate for a supplier
func (s *Service) RegisterCertificate(ctx context.Context, conn *pgxpool.Conn, supplierID uuid.UUID, req *CreateComplianceCertRequest) (*ComplianceCertificate, error) {
	// Verify supplier exists
	_, err := s.repo.GetSupplierByID(ctx, conn, supplierID)
	if err != nil {
		return nil, err
	}

	validFrom, err := time.Parse("2006-01-02", req.ValidFrom)
	if err != nil {
		return nil, fmt.Errorf("invalid valid_from date format (must be YYYY-MM-DD): %w", err)
	}
	expiryDate, err := time.Parse("2006-01-02", req.ExpiryDate)
	if err != nil {
		return nil, fmt.Errorf("invalid expiry_date date format (must be YYYY-MM-DD): %w", err)
	}
	if expiryDate.Before(validFrom) {
		return nil, errors.New("expiry_date cannot be before valid_from")
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	cert := &ComplianceCertificate{
		ID:                uuid.New(),
		SupplierID:        supplierID,
		CertType:          req.CertType,
		CertificateNumber: req.CertificateNumber,
		IssuingAuthority:  req.IssuingAuthority,
		Scope:             req.Scope,
		ValidFrom:         validFrom,
		ExpiryDate:        expiryDate,
		DocumentURL:       req.DocumentURL,
	}

	if err := s.repo.CreateComplianceCertificate(ctx, tx, cert); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	cert.ComputedStatus = certificateStatus(cert, s.now())

	return cert, nil
}

// RevokeCertificate revokes a certificate without rewriting its validity dates.
func (s *Service) RevokeCertificate(ctx context.Context, conn *pgxpool.Conn, certID uuid.UUID) error {
	return s.repo.RevokeCertificate(ctx, conn, certID)
}

// ListPurchaseOrders returns a paginated list of purchase order summaries
func (s *Service) ListPurchaseOrders(ctx context.Context, conn *pgxpool.Conn, status string, limit, offset int) ([]PurchaseOrderSummary, error) {
	return s.repo.ListPurchaseOrders(ctx, conn, status, limit, offset)
}

// GetPurchaseOrderDetail returns comprehensive PO details including lines and linked goods receipts
func (s *Service) GetPurchaseOrderDetail(ctx context.Context, conn *pgxpool.Conn, poID uuid.UUID) (*PurchaseOrderDetail, error) {
	return s.repo.GetPurchaseOrderDetail(ctx, conn, poID)
}

// CancelPurchaseOrder atomically cancels an unfulfilled purchase order
func (s *Service) CancelPurchaseOrder(ctx context.Context, conn *pgxpool.Conn, poID uuid.UUID) error {
	return s.repo.CancelPurchaseOrder(ctx, conn, poID)
}

// ListGoodsReceipts returns a paginated list of goods receipts
func (s *Service) ListGoodsReceipts(ctx context.Context, conn *pgxpool.Conn, limit, offset int) ([]GoodsReceiptSummary, error) {
	return s.repo.ListGoodsReceipts(ctx, conn, limit, offset)
}

// GetGoodsReceiptDetail returns full detail of a goods receipt with product valuation and ledger cross-reference
func (s *Service) GetGoodsReceiptDetail(ctx context.Context, conn *pgxpool.Conn, grID uuid.UUID) (*GoodsReceiptDetail, error) {
	return s.repo.GetGoodsReceiptDetail(ctx, conn, grID)
}

// GetProductTraceability reconstructs product provenance from Halal cert to current shelf stock
func (s *Service) GetProductTraceability(ctx context.Context, conn *pgxpool.Conn, productID uuid.UUID) (*ProductTraceabilityReport, error) {
	return s.repo.GetProductTraceability(ctx, conn, productID)
}
