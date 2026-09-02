package supplychain

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrComplianceCertRequired = errors.New("compliance certificate is required under strict mode")
	ErrComplianceCertExpired  = errors.New("compliance certificate is expired")
	ErrInvalidPOStatus        = errors.New("invalid purchase order status for this operation")
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
func (s *Service) checkCompliance(ctx context.Context, conn *pgxpool.Conn, certID *uuid.UUID) error {
	start := time.Now()
	defer func() {
		slog.InfoContext(ctx, "compliance_check_completed",
			slog.Duration("duration_cert_validation_ms", time.Since(start)),
		)
	}()

	// 1. Fetch Tenant Config
	config, err := s.repo.GetTenantConfig(ctx, conn, "compliance")
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
	cert, err := s.repo.GetComplianceCertificateByID(ctx, conn, *certID)
	if err != nil {
		return err
	}

	switch cert.ComputedStatus {
	case "EXPIRED":
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
	default:
		return nil
	}
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
	var certID *uuid.UUID
	if req.ComplianceCertID != nil {
		id, err := uuid.Parse(*req.ComplianceCertID)
		if err == nil {
			certID = &id
		}
	}

	// COMPLIANCE INTERCEPTOR (HARD-BLOCK)
	if err := s.checkCompliance(ctx, conn, certID); err != nil {
		return nil, err
	}

	supplierID, _ := uuid.Parse(req.SupplierID)
	po := &PurchaseOrder{
		ID:               uuid.New(),
		PONumber:         "PO-" + time.Now().Format("20060102150405"),
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

// CreateGoodsReceipt creates a GR, enforces compliance again, and increments stock atomically
func (s *Service) CreateGoodsReceipt(ctx context.Context, conn *pgxpool.Conn, userID uuid.UUID, req *CreateGoodsReceiptRequest) (*GoodsReceipt, error) {
	poID, _ := uuid.Parse(req.PurchaseOrderID)
	po, err := s.repo.GetPurchaseOrderByID(ctx, conn, poID)
	if err != nil {
		return nil, err
	}

	if po.Status != "ISSUED" {
		return nil, ErrInvalidPOStatus
	}

	// COMPLIANCE INTERCEPTOR (RE-VALIDATION)
	// Certificate might have expired between PO creation and Goods Receipt
	if err := s.checkCompliance(ctx, conn, po.ComplianceCertID); err != nil {
		return nil, err
	}

	gr := &GoodsReceipt{
		ID:              uuid.New(),
		GRNumber:        "GR-" + time.Now().Format("20060102150405"),
		PurchaseOrderID: po.ID,
		ReceivedBy:      userID,
		ReceivedDate:    time.Now(),
		Notes:           req.Notes,
	}

	// Fetch PO items to determine unit costs for valuation
	queryPOItems := `SELECT product_id, unit_cost FROM purchase_order_items WHERE purchase_order_id = $1`
	rows, err := conn.Query(ctx, queryPOItems, po.ID)
	if err != nil {
		return nil, err
	}
	poUnitCosts := make(map[uuid.UUID]float64)
	for rows.Next() {
		var pid uuid.UUID
		var cost float64
		if err := rows.Scan(&pid, &cost); err != nil {
			rows.Close()
			return nil, err
		}
		poUnitCosts[pid] = cost
	}
	rows.Close()

	var inboundValue float64
	for _, reqItem := range req.Items {
		productID, _ := uuid.Parse(reqItem.ProductID)
		gr.Items = append(gr.Items, GoodsReceiptItem{
			ID:               uuid.New(),
			GoodsReceiptID:   gr.ID,
			ProductID:        productID,
			ReceivedQuantity: reqItem.ReceivedQuantity,
		})
		inboundValue += float64(reqItem.ReceivedQuantity) * poUnitCosts[productID]
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	startInsert := time.Now()
	if err := s.repo.CreateGoodsReceipt(ctx, tx, gr); err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "gr_and_stock_update_completed", slog.Duration("duration_stock_increment_ms", time.Since(startInsert)))

	if err := s.repo.UpdatePurchaseOrderStatus(ctx, tx, po.ID, "RECEIVED"); err != nil {
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
