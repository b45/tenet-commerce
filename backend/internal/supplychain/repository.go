package supplychain

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("record not found")
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// GetTenantConfig fetches a specific config value for the tenant
func (r *Repository) GetTenantConfig(ctx context.Context, conn *pgxpool.Conn, configKey string) (map[string]interface{}, error) {
	query := `SELECT config_value FROM tenant_config WHERE config_key = $1`
	var configData []byte
	err := conn.QueryRow(ctx, query, configKey).Scan(&configData)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// CreateSupplier inserts a new supplier
func (r *Repository) CreateSupplier(ctx context.Context, tx pgx.Tx, supplier *Supplier) error {
	query := `
		INSERT INTO suppliers (id, code, company_name, contact_person, contact_email, contact_phone, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	return tx.QueryRow(ctx, query,
		supplier.ID, supplier.Code, supplier.CompanyName, supplier.ContactPerson,
		supplier.ContactEmail, supplier.ContactPhone, supplier.IsActive,
	).Scan(&supplier.CreatedAt)
}

// CreateComplianceCertificate inserts a new compliance certificate
func (r *Repository) CreateComplianceCertificate(ctx context.Context, tx pgx.Tx, cert *ComplianceCertificate) error {
	query := `
		INSERT INTO compliance_certificates (id, supplier_id, cert_type, certificate_number, issuing_authority, scope, valid_from, expiry_date, document_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at
	`
	return tx.QueryRow(ctx, query,
		cert.ID, cert.SupplierID, cert.CertType, cert.CertificateNumber,
		cert.IssuingAuthority, cert.Scope, cert.ValidFrom, cert.ExpiryDate, cert.DocumentURL,
	).Scan(&cert.CreatedAt)
}

// GetComplianceCertificateByID fetches a certificate and computes its status dynamically
func (r *Repository) GetComplianceCertificateByID(ctx context.Context, conn *pgxpool.Conn, certID uuid.UUID) (*ComplianceCertificate, error) {
	query := `
		SELECT id, supplier_id, cert_type, certificate_number, issuing_authority, scope, valid_from, expiry_date, document_url, created_at,
		CASE 
			WHEN expiry_date < CURRENT_DATE THEN 'EXPIRED'
			WHEN expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'EXPIRING_SOON'
			ELSE 'VALID'
		END AS computed_status
		FROM compliance_certificates
		WHERE id = $1
	`
	
	cert := &ComplianceCertificate{}
	err := conn.QueryRow(ctx, query, certID).Scan(
		&cert.ID, &cert.SupplierID, &cert.CertType, &cert.CertificateNumber,
		&cert.IssuingAuthority, &cert.Scope, &cert.ValidFrom, &cert.ExpiryDate,
		&cert.DocumentURL, &cert.CreatedAt, &cert.ComputedStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return cert, nil
}

// CreatePurchaseOrder inserts a PO and its items
func (r *Repository) CreatePurchaseOrder(ctx context.Context, tx pgx.Tx, po *PurchaseOrder) error {
	queryPO := `
		INSERT INTO purchase_orders (id, po_number, supplier_id, compliance_cert_id, total_amount, status, issued_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	err := tx.QueryRow(ctx, queryPO,
		po.ID, po.PONumber, po.SupplierID, po.ComplianceCertID,
		po.TotalAmount, po.Status, po.IssuedDate,
	).Scan(&po.CreatedAt)
	if err != nil {
		return err
	}

	for _, item := range po.Items {
		queryItem := `
			INSERT INTO purchase_order_items (id, purchase_order_id, product_id, quantity, unit_cost, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err := tx.Exec(ctx, queryItem, item.ID, item.PurchaseOrderID, item.ProductID, item.Quantity, item.UnitCost, item.Subtotal)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPurchaseOrderByID fetches a PO
func (r *Repository) GetPurchaseOrderByID(ctx context.Context, conn *pgxpool.Conn, poID uuid.UUID) (*PurchaseOrder, error) {
	query := `
		SELECT id, po_number, supplier_id, compliance_cert_id, total_amount, status, issued_date, created_at
		FROM purchase_orders WHERE id = $1
	`
	po := &PurchaseOrder{}
	err := conn.QueryRow(ctx, query, poID).Scan(
		&po.ID, &po.PONumber, &po.SupplierID, &po.ComplianceCertID,
		&po.TotalAmount, &po.Status, &po.IssuedDate, &po.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return po, nil
}

// UpdatePurchaseOrderStatus updates the PO status
func (r *Repository) UpdatePurchaseOrderStatus(ctx context.Context, tx pgx.Tx, poID uuid.UUID, status string) error {
	query := `UPDATE purchase_orders SET status = $1 WHERE id = $2`
	_, err := tx.Exec(ctx, query, status, poID)
	return err
}

// CreateGoodsReceipt inserts a GR and increments inventory stock atomically
func (r *Repository) CreateGoodsReceipt(ctx context.Context, tx pgx.Tx, gr *GoodsReceipt) error {
	queryGR := `
		INSERT INTO goods_receipts (id, gr_number, purchase_order_id, received_by, received_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	err := tx.QueryRow(ctx, queryGR,
		gr.ID, gr.GRNumber, gr.PurchaseOrderID, gr.ReceivedBy, gr.ReceivedDate, gr.Notes,
	).Scan(&gr.CreatedAt)
	if err != nil {
		return err
	}

	for _, item := range gr.Items {
		// 1. Insert GR Item
		queryItem := `
			INSERT INTO goods_receipt_items (id, goods_receipt_id, product_id, received_quantity)
			VALUES ($1, $2, $3, $4)
		`
		_, err := tx.Exec(ctx, queryItem, item.ID, item.GoodsReceiptID, item.ProductID, item.ReceivedQuantity)
		if err != nil {
			return err
		}

		// 2. Increment Stock Atomically
		queryStock := `
			UPDATE inventory
			SET stock_quantity = stock_quantity + $1, updated_at = NOW()
			WHERE product_id = $2
		`
		_, err = tx.Exec(ctx, queryStock, item.ReceivedQuantity, item.ProductID)
		if err != nil {
			return err
		}
	}
	return nil
}
