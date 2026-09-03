package supplychain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrComplianceCertRequired = errors.New("compliance certificate is required under strict mode")
	ErrComplianceCertExpired  = errors.New("compliance certificate is expired")
	ErrComplianceCertInvalid  = errors.New("compliance certificate is not valid for this transaction")
	ErrInvalidPOStatus        = errors.New("invalid purchase order status for this operation")
	ErrEmptyReceipt           = errors.New("goods receipt must contain at least one item")
	ErrZeroValueReceipt       = errors.New("goods receipt inbound valuation must be greater than zero")
	ErrReceiptItemNotOnPO     = errors.New("goods receipt item does not exist on purchase order")
	ErrDuplicateReceiptItem   = errors.New("goods receipt contains a duplicate product")
	ErrReceiptQuantityExceeds = errors.New("goods receipt quantity exceeds purchase order outstanding quantity")
	ErrIdempotencyKeyConflict = errors.New("idempotency key is already associated with another purchase order")
)

type Service struct {
	repo          *Repository
	ledgerService *ledger.Service
}

func NewService(repo *Repository, ledgerService *ledger.Service) *Service {
	return &Service{repo: repo, ledgerService: ledgerService}
}

// checkCompliance is an internal helper that implements the Configurable Compliance Engine logic.
// It verifies if the tenant is in strict mode and, if so, whether the provided certificate is valid.
func (s *Service) checkCompliance(ctx context.Context, db queryRower, supplierID uuid.UUID, certID *uuid.UUID) error {
	start := time.Now()
	defer func() {
		slog.InfoContext(ctx, "compliance_check_completed",
			slog.Duration("duration_cert_validation_ms", time.Since(start)),
		)
	}()

	// 1. Fetch Tenant Config
	config, err := s.repo.GetTenantConfig(ctx, db, "compliance")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // No config found means no strict mode
		}
		return err
	}

	strictMode, ok := config["strict_compliance_mode"].(bool)
	if !ok || !strictMode {
		return nil // Strict mode is disabled. Bypass check.
	}

	// 2. Strict mode is ON. We must have a certificate.
	if certID == nil {
		return ErrComplianceCertRequired
	}

	// 3. Validate Certificate Status
	cert, err := s.repo.GetComplianceCertificateByID(ctx, db, *certID)
	if err != nil {
		return err
	}
	if cert.SupplierID != supplierID {
		return ErrComplianceCertInvalid
	}

	switch cert.ComputedStatus {
	case "EXPIRED", "NOT_YET_VALID":
		slog.ErrorContext(ctx, "compliance_hard_block",
			slog.String("cert_id", cert.ID.String()),
			slog.String("cert_type", cert.CertType),
		)
		return ErrComplianceCertExpired
	case "EXPIRING_SOON":
		slog.WarnContext(ctx, "compliance_expiring_soon",
			slog.String("cert_id", cert.ID.String()),
			slog.String("expiry_date", cert.ExpiryDate.String()),
		)
		return nil
	}

	requiredTypes, ok := config["required_compliance"].([]interface{})
	if !ok {
		return nil
	}
	for _, value := range requiredTypes {
		requiredType, ok := value.(string)
		if ok && requiredType == cert.CertType {
			return nil
		}
	}
	return ErrComplianceCertInvalid
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

	var totalAmount float64
	for _, reqItem := range req.Items {
		productID, _ := uuid.Parse(reqItem.ProductID)
		subtotal := float64(reqItem.Quantity) * reqItem.UnitCost
		totalAmount += subtotal

		po.Items = append(po.Items, PurchaseOrderItem{
			ID:              uuid.New(),
			PurchaseOrderID: po.ID,
			ProductID:       productID,
			Quantity:        reqItem.Quantity,
			UnitCost:        reqItem.UnitCost,
			Subtotal:        subtotal,
		})
	}
	po.TotalAmount = totalAmount

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// COMPLIANCE INTERCEPTOR (HARD-BLOCK), evaluated inside the PO transaction.
	if err := s.checkCompliance(ctx, tx, supplierID, certID); err != nil {
		return nil, err
	}

	startInsert := time.Now()
	if err := s.repo.CreatePurchaseOrder(ctx, tx, po); err != nil {
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
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if po.Status != "ISSUED" && po.Status != "PARTIALLY_RECEIVED" {
		return nil, ErrInvalidPOStatus
	}

	// The certificate may have changed since the PO was issued, so enforce the
	// tenant's strict-compliance configuration within this receipt transaction.
	if err := s.checkCompliance(ctx, tx, po.SupplierID, po.ComplianceCertID); err != nil {
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
		ID:              uuid.New(),
		GRNumber:        "GR-" + time.Now().Format("20060102150405") + "-" + uuid.NewString()[:8],
		IdempotencyKey:  idempotencyKey,
		PurchaseOrderID: po.ID,
		ReceivedBy:      userID,
		ReceivedDate:    time.Now(),
		Notes:           req.Notes,
	}
	inboundValue, fullyReceived, err := reconcileReceiptItems(gr, req.Items, poItems, receivedQuantities)
	if err != nil {
		return nil, err
	}

	startInsert := time.Now()
	if err := s.repo.CreateGoodsReceipt(ctx, tx, gr); err != nil {
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
	var inboundValue float64
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

		requestedByProduct[productID] = requestItem.ReceivedQuantity
		gr.Items = append(gr.Items, GoodsReceiptItem{
			ID:               uuid.New(),
			GoodsReceiptID:   gr.ID,
			ProductID:        productID,
			ReceivedQuantity: requestItem.ReceivedQuantity,
		})
		inboundValue += float64(requestItem.ReceivedQuantity) * poItem.UnitCost
	}

	if inboundValue <= 0 {
		return 0, false, ErrZeroValueReceipt
	}

	for productID, poItem := range poByProduct {
		if received[productID]+requestedByProduct[productID] != poItem.Quantity {
			return inboundValue, false, nil
		}
	}
	return inboundValue, true, nil
}
