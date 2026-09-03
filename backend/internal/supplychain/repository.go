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

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// GetTenantConfig fetches a specific config value for the tenant
func (r *Repository) GetTenantConfig(ctx context.Context, db queryRower, configKey string) (map[string]interface{}, error) {
	query := `SELECT config_value FROM tenant_config WHERE config_key = $1`
	var configData []byte
	err := db.QueryRow(ctx, query, configKey).Scan(&configData)
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
func (r *Repository) GetComplianceCertificateByID(ctx context.Context, db queryRower, certID uuid.UUID) (*ComplianceCertificate, error) {
	query := `
		SELECT id, supplier_id, cert_type, certificate_number, issuing_authority, scope, valid_from, expiry_date, document_url, created_at,
		CASE
			WHEN valid_from > CURRENT_DATE THEN 'NOT_YET_VALID'
			WHEN expiry_date < CURRENT_DATE THEN 'EXPIRED'
			WHEN expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'EXPIRING_SOON'
			ELSE 'VALID'
		END AS computed_status
		FROM compliance_certificates
		WHERE id = $1
	`
	
	cert := &ComplianceCertificate{}
	err := db.QueryRow(ctx, query, certID).Scan(
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

// LockPurchaseOrder acquires a row lock so a receipt decision and its state update
// are serialized for a single purchase order.
func (r *Repository) LockPurchaseOrder(ctx context.Context, tx pgx.Tx, poID uuid.UUID) (*PurchaseOrder, error) {
	query := `
		SELECT id, po_number, supplier_id, compliance_cert_id, total_amount, status, issued_date, created_at
		FROM purchase_orders
		WHERE id = $1
		FOR UPDATE
	`
	po := &PurchaseOrder{}
	err := tx.QueryRow(ctx, query, poID).Scan(
		&po.ID, &po.PONumber, &po.SupplierID, &po.ComplianceCertID,
		&po.TotalAmount, &po.Status, &po.IssuedDate, &po.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return po, nil
}

// GetPurchaseOrderItems returns the approved quantities and costs used to value a receipt.
func (r *Repository) GetPurchaseOrderItems(ctx context.Context, tx pgx.Tx, poID uuid.UUID) ([]PurchaseOrderItem, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, purchase_order_id, product_id, quantity, unit_cost, subtotal
		FROM purchase_order_items
		WHERE purchase_order_id = $1
		ORDER BY id
	`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PurchaseOrderItem, 0)
	for rows.Next() {
		var item PurchaseOrderItem
		if err := rows.Scan(&item.ID, &item.PurchaseOrderID, &item.ProductID, &item.Quantity, &item.UnitCost, &item.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetReceivedQuantities returns the cumulative quantities already posted for a PO.
func (r *Repository) GetReceivedQuantities(ctx context.Context, tx pgx.Tx, poID uuid.UUID) (map[uuid.UUID]int, error) {
	rows, err := tx.Query(ctx, `
		SELECT gri.product_id, SUM(gri.received_quantity)
		FROM goods_receipt_items gri
		JOIN goods_receipts gr ON gr.id = gri.goods_receipt_id
		WHERE gr.purchase_order_id = $1
		GROUP BY gri.product_id
	`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	received := make(map[uuid.UUID]int)
	for rows.Next() {
		var productID uuid.UUID
		var quantity int
		if err := rows.Scan(&productID, &quantity); err != nil {
			return nil, err
		}
		received[productID] = quantity
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return received, nil
}

// GetGoodsReceiptByIdempotencyKey returns a previously committed receipt for replay.
func (r *Repository) GetGoodsReceiptByIdempotencyKey(ctx context.Context, tx pgx.Tx, idempotencyKey string) (*GoodsReceipt, error) {
	gr := &GoodsReceipt{}
	err := tx.QueryRow(ctx, `
		SELECT id, gr_number, idempotency_key, purchase_order_id, received_by, received_date, notes, created_at
		FROM goods_receipts
		WHERE idempotency_key = $1
	`, idempotencyKey).Scan(
		&gr.ID, &gr.GRNumber, &gr.IdempotencyKey, &gr.PurchaseOrderID,
		&gr.ReceivedBy, &gr.ReceivedDate, &gr.Notes, &gr.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, goods_receipt_id, product_id, received_quantity
		FROM goods_receipt_items
		WHERE goods_receipt_id = $1
		ORDER BY id
	`, gr.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item GoodsReceiptItem
		if err := rows.Scan(&item.ID, &item.GoodsReceiptID, &item.ProductID, &item.ReceivedQuantity); err != nil {
			return nil, err
		}
		gr.Items = append(gr.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return gr, nil
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
	result, err := tx.Exec(ctx, `
		UPDATE purchase_orders
		SET status = $1
		WHERE id = $2 AND status IN ('ISSUED', 'PARTIALLY_RECEIVED')
	`, status, poID)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

// CreateGoodsReceipt inserts a GR and increments inventory stock atomically
func (r *Repository) CreateGoodsReceipt(ctx context.Context, tx pgx.Tx, gr *GoodsReceipt) error {
	queryGR := `
		INSERT INTO goods_receipts (id, gr_number, idempotency_key, purchase_order_id, received_by, received_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`
	err := tx.QueryRow(ctx, queryGR,
		gr.ID, gr.GRNumber, gr.IdempotencyKey, gr.PurchaseOrderID, gr.ReceivedBy, gr.ReceivedDate, gr.Notes,
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
