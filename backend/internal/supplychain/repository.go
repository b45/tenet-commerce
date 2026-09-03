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

// ListSuppliers fetches suppliers with optional is_active filter and pagination
func (r *Repository) ListSuppliers(ctx context.Context, conn *pgxpool.Conn, isActive *bool, limit, offset int) ([]Supplier, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT s.id, s.code, s.company_name, s.contact_person, s.contact_email, s.contact_phone, s.is_active, s.created_at
		FROM suppliers s
		WHERE ($1::boolean IS NULL OR s.is_active = $1)
		ORDER BY s.company_name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := conn.Query(ctx, query, isActive, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppliers []Supplier
	for rows.Next() {
		var s Supplier
		if err := rows.Scan(&s.ID, &s.Code, &s.CompanyName, &s.ContactPerson, &s.ContactEmail, &s.ContactPhone, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		suppliers = append(suppliers, s)
	}
	rows.Close()

	// Attach active certificate if available
	for i := range suppliers {
		certQuery := `
			SELECT id, supplier_id, cert_type, certificate_number, issuing_authority, scope, valid_from, expiry_date, document_url, created_at,
			CASE
				WHEN valid_from > CURRENT_DATE THEN 'NOT_YET_VALID'
				WHEN expiry_date < CURRENT_DATE THEN 'EXPIRED'
				WHEN expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'EXPIRING_SOON'
				ELSE 'VALID'
			END AS computed_status
			FROM compliance_certificates
			WHERE supplier_id = $1
			ORDER BY expiry_date DESC
			LIMIT 1
		`
		var cert ComplianceCertificate
		err := conn.QueryRow(ctx, certQuery, suppliers[i].ID).Scan(
			&cert.ID, &cert.SupplierID, &cert.CertType, &cert.CertificateNumber,
			&cert.IssuingAuthority, &cert.Scope, &cert.ValidFrom, &cert.ExpiryDate,
			&cert.DocumentURL, &cert.CreatedAt, &cert.ComputedStatus,
		)
		if err == nil {
			suppliers[i].ComplianceCertificate = &cert
		}
	}

	return suppliers, nil
}

// GetSupplierWithCertificates fetches a single supplier with all its historical compliance certificates
func (r *Repository) GetSupplierWithCertificates(ctx context.Context, conn *pgxpool.Conn, id uuid.UUID) (*SupplierDetail, error) {
	query := `
		SELECT id, code, company_name, contact_person, contact_email, contact_phone, is_active, created_at
		FROM suppliers
		WHERE id = $1
	`
	var sd SupplierDetail
	err := conn.QueryRow(ctx, query, id).Scan(
		&sd.ID, &sd.Code, &sd.CompanyName, &sd.ContactPerson,
		&sd.ContactEmail, &sd.ContactPhone, &sd.IsActive, &sd.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	certs, err := r.GetCertificatesBySupplierID(ctx, conn, id)
	if err != nil {
		return nil, err
	}
	sd.Certificates = certs
	if len(certs) > 0 {
		sd.ComplianceCertificate = &certs[0]
	}

	return &sd, nil
}

// UpdateSupplier updates supplier contact information or active status
func (r *Repository) UpdateSupplier(ctx context.Context, conn *pgxpool.Conn, id uuid.UUID, req *UpdateSupplierRequest) (*Supplier, error) {
	supplier, err := r.GetSupplierByID(ctx, conn, id)
	if err != nil {
		return nil, err
	}

	if req.CompanyName != nil {
		supplier.CompanyName = *req.CompanyName
	}
	if req.ContactPerson != nil {
		supplier.ContactPerson = *req.ContactPerson
	}
	if req.ContactEmail != nil {
		supplier.ContactEmail = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		supplier.ContactPhone = *req.ContactPhone
	}
	if req.IsActive != nil {
		supplier.IsActive = *req.IsActive
	}

	query := `
		UPDATE suppliers
		SET company_name = $1, contact_person = $2, contact_email = $3, contact_phone = $4, is_active = $5
		WHERE id = $6
	`
	_, err = conn.Exec(ctx, query, supplier.CompanyName, supplier.ContactPerson, supplier.ContactEmail, supplier.ContactPhone, supplier.IsActive, id)
	if err != nil {
		return nil, err
	}

	return supplier, nil
}

// GetSupplierByID fetches basic supplier by ID
func (r *Repository) GetSupplierByID(ctx context.Context, db queryRower, id uuid.UUID) (*Supplier, error) {
	query := `
		SELECT id, code, company_name, contact_person, contact_email, contact_phone, is_active, created_at
		FROM suppliers
		WHERE id = $1
	`
	var s Supplier
	err := db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Code, &s.CompanyName, &s.ContactPerson,
		&s.ContactEmail, &s.ContactPhone, &s.IsActive, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// GetCertificatesBySupplierID returns all certificates for a supplier ordered by expiry descending
func (r *Repository) GetCertificatesBySupplierID(ctx context.Context, conn *pgxpool.Conn, supplierID uuid.UUID) ([]ComplianceCertificate, error) {
	query := `
		SELECT id, supplier_id, cert_type, certificate_number, issuing_authority, scope, valid_from, expiry_date, document_url, created_at,
		CASE
			WHEN valid_from > CURRENT_DATE THEN 'NOT_YET_VALID'
			WHEN expiry_date < CURRENT_DATE THEN 'EXPIRED'
			WHEN expiry_date <= CURRENT_DATE + INTERVAL '30 days' THEN 'EXPIRING_SOON'
			ELSE 'VALID'
		END AS computed_status
		FROM compliance_certificates
		WHERE supplier_id = $1
		ORDER BY expiry_date DESC
	`
	rows, err := conn.Query(ctx, query, supplierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []ComplianceCertificate
	for rows.Next() {
		var c ComplianceCertificate
		if err := rows.Scan(
			&c.ID, &c.SupplierID, &c.CertType, &c.CertificateNumber,
			&c.IssuingAuthority, &c.Scope, &c.ValidFrom, &c.ExpiryDate,
			&c.DocumentURL, &c.CreatedAt, &c.ComputedStatus,
		); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, nil
}

// RevokeCertificate marks a certificate as expired immediately
func (r *Repository) RevokeCertificate(ctx context.Context, conn *pgxpool.Conn, certID uuid.UUID) error {
	cmd, err := conn.Exec(ctx, `
		UPDATE compliance_certificates
		SET expiry_date = CURRENT_DATE - INTERVAL '1 day'
		WHERE id = $1
	`, certID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListPurchaseOrders returns a paginated list of PO summaries with optional status filter
func (r *Repository) ListPurchaseOrders(ctx context.Context, conn *pgxpool.Conn, status string, limit, offset int) ([]PurchaseOrderSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT po.id, po.po_number, po.supplier_id, s.company_name, po.compliance_cert_id, po.total_amount, po.status, po.issued_date, po.created_at,
		       (SELECT COUNT(*) FROM purchase_order_items poi WHERE poi.purchase_order_id = po.id) AS item_count
		FROM purchase_orders po
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE ($1::text = '' OR po.status = $1)
		ORDER BY po.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := conn.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pos []PurchaseOrderSummary
	for rows.Next() {
		var po PurchaseOrderSummary
		if err := rows.Scan(
			&po.ID, &po.PONumber, &po.SupplierID, &po.SupplierName, &po.ComplianceCertID,
			&po.TotalAmount, &po.Status, &po.IssuedDate, &po.CreatedAt, &po.ItemCount,
		); err != nil {
			return nil, err
		}
		pos = append(pos, po)
	}
	return pos, nil
}

// GetPurchaseOrderDetail returns comprehensive PO information including items with remaining balances and linked GRs
func (r *Repository) GetPurchaseOrderDetail(ctx context.Context, conn *pgxpool.Conn, poID uuid.UUID) (*PurchaseOrderDetail, error) {
	queryPO := `
		SELECT po.id, po.po_number, po.supplier_id, s.company_name, po.compliance_cert_id, po.total_amount, po.status, po.issued_date, po.created_at
		FROM purchase_orders po
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE po.id = $1
	`
	var pod PurchaseOrderDetail
	err := conn.QueryRow(ctx, queryPO, poID).Scan(
		&pod.ID, &pod.PONumber, &pod.SupplierID, &pod.SupplierName,
		&pod.ComplianceCertID, &pod.TotalAmount, &pod.Status, &pod.IssuedDate, &pod.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Fetch items with received quantities
	queryItems := `
		SELECT poi.id, poi.product_id, p.name, p.sku, poi.quantity,
		       COALESCE((
		           SELECT SUM(gri.received_quantity)
		           FROM goods_receipt_items gri
		           JOIN goods_receipts gr ON gr.id = gri.goods_receipt_id
		           WHERE gr.purchase_order_id = poi.purchase_order_id AND gri.product_id = poi.product_id
		       ), 0) AS received_quantity,
		       poi.unit_cost, poi.subtotal
		FROM purchase_order_items poi
		JOIN products p ON p.id = poi.product_id
		WHERE poi.purchase_order_id = $1
		ORDER BY poi.id
	`
	itemRows, err := conn.Query(ctx, queryItems, poID)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var line PurchaseOrderDetailLine
		if err := itemRows.Scan(
			&line.ID, &line.ProductID, &line.ProductName, &line.ProductSKU,
			&line.Quantity, &line.ReceivedQuantity, &line.UnitCost, &line.Subtotal,
		); err != nil {
			return nil, err
		}
		line.RemainingQuantity = line.Quantity - line.ReceivedQuantity
		if line.RemainingQuantity < 0 {
			line.RemainingQuantity = 0
		}
		pod.Items = append(pod.Items, line)
	}
	itemRows.Close()

	// Fetch linked goods receipts
	queryGRs := `
		SELECT gr.id, gr.gr_number, gr.purchase_order_id, po.po_number, s.company_name, gr.received_by, gr.received_date,
		       (SELECT COUNT(*) FROM goods_receipt_items gri WHERE gri.goods_receipt_id = gr.id) AS total_items,
		       COALESCE((
		           SELECT SUM(gri.received_quantity * poi.unit_cost)
		           FROM goods_receipt_items gri
		           JOIN purchase_order_items poi ON poi.purchase_order_id = gr.purchase_order_id AND poi.product_id = gri.product_id
		           WHERE gri.goods_receipt_id = gr.id
		       ), 0) AS total_valuation,
		       gr.created_at
		FROM goods_receipts gr
		JOIN purchase_orders po ON po.id = gr.purchase_order_id
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE gr.purchase_order_id = $1
		ORDER BY gr.created_at DESC
	`
	grRows, err := conn.Query(ctx, queryGRs, poID)
	if err != nil {
		return nil, err
	}
	defer grRows.Close()

	for grRows.Next() {
		var gr GoodsReceiptSummary
		if err := grRows.Scan(
			&gr.ID, &gr.GRNumber, &gr.PurchaseOrderID, &gr.PONumber, &gr.SupplierName,
			&gr.ReceivedBy, &gr.ReceivedDate, &gr.TotalItems, &gr.TotalValuation, &gr.CreatedAt,
		); err != nil {
			return nil, err
		}
		pod.GoodsReceipts = append(pod.GoodsReceipts, gr)
	}

	return &pod, nil
}

// CancelPurchaseOrder atomically cancels an unfulfilled PO (status DRAFT or ISSUED with 0 items received)
func (r *Repository) CancelPurchaseOrder(ctx context.Context, conn *pgxpool.Conn, poID uuid.UUID) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	po, err := r.LockPurchaseOrder(ctx, tx, poID)
	if err != nil {
		return err
	}

	if po.Status != "DRAFT" && po.Status != "ISSUED" {
		return errors.New("only DRAFT or ISSUED purchase orders can be cancelled")
	}

	receivedMap, err := r.GetReceivedQuantities(ctx, tx, poID)
	if err != nil {
		return err
	}

	totalReceived := 0
	for _, qty := range receivedMap {
		totalReceived += qty
	}
	if totalReceived > 0 {
		return errors.New("cannot cancel purchase order with received goods")
	}

	_, err = tx.Exec(ctx, "UPDATE purchase_orders SET status = 'CANCELLED' WHERE id = $1", poID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListGoodsReceipts returns a paginated list of goods receipt summaries
func (r *Repository) ListGoodsReceipts(ctx context.Context, conn *pgxpool.Conn, limit, offset int) ([]GoodsReceiptSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT gr.id, gr.gr_number, gr.purchase_order_id, po.po_number, s.company_name, gr.received_by, gr.received_date,
		       (SELECT COUNT(*) FROM goods_receipt_items gri WHERE gri.goods_receipt_id = gr.id) AS total_items,
		       COALESCE((
		           SELECT SUM(gri.received_quantity * poi.unit_cost)
		           FROM goods_receipt_items gri
		           JOIN purchase_order_items poi ON poi.purchase_order_id = gr.purchase_order_id AND poi.product_id = gri.product_id
		           WHERE gri.goods_receipt_id = gr.id
		       ), 0) AS total_valuation,
		       gr.created_at
		FROM goods_receipts gr
		JOIN purchase_orders po ON po.id = gr.purchase_order_id
		JOIN suppliers s ON s.id = po.supplier_id
		ORDER BY gr.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := conn.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grs []GoodsReceiptSummary
	for rows.Next() {
		var gr GoodsReceiptSummary
		if err := rows.Scan(
			&gr.ID, &gr.GRNumber, &gr.PurchaseOrderID, &gr.PONumber, &gr.SupplierName,
			&gr.ReceivedBy, &gr.ReceivedDate, &gr.TotalItems, &gr.TotalValuation, &gr.CreatedAt,
		); err != nil {
			return nil, err
		}
		grs = append(grs, gr)
	}
	return grs, nil
}

// GetGoodsReceiptDetail returns full detail of a goods receipt including product lines and accounting cross-reference
func (r *Repository) GetGoodsReceiptDetail(ctx context.Context, conn *pgxpool.Conn, grID uuid.UUID) (*GoodsReceiptDetail, error) {
	queryHeader := `
		SELECT gr.id, gr.gr_number, gr.idempotency_key, gr.purchase_order_id, po.po_number,
		       s.id, s.company_name, gr.received_by, gr.received_date, gr.notes, gr.created_at
		FROM goods_receipts gr
		JOIN purchase_orders po ON po.id = gr.purchase_order_id
		JOIN suppliers s ON s.id = po.supplier_id
		WHERE gr.id = $1
	`
	var grd GoodsReceiptDetail
	err := conn.QueryRow(ctx, queryHeader, grID).Scan(
		&grd.ID, &grd.GRNumber, &grd.IdempotencyKey, &grd.PurchaseOrderID, &grd.PONumber,
		&grd.SupplierID, &grd.SupplierName, &grd.ReceivedBy, &grd.ReceivedDate, &grd.Notes, &grd.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Fetch items with unit cost and subtotal valuation
	queryItems := `
		SELECT gri.id, gri.product_id, p.name, p.sku, gri.received_quantity, poi.unit_cost,
		       (gri.received_quantity * poi.unit_cost) AS subtotal_valuation
		FROM goods_receipt_items gri
		JOIN products p ON p.id = gri.product_id
		JOIN goods_receipts gr ON gr.id = gri.goods_receipt_id
		JOIN purchase_order_items poi ON poi.purchase_order_id = gr.purchase_order_id AND poi.product_id = gri.product_id
		WHERE gri.goods_receipt_id = $1
		ORDER BY gri.id
	`
	rows, err := conn.Query(ctx, queryItems, grID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalValuation float64
	for rows.Next() {
		var item GoodsReceiptDetailItem
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.ProductSKU, &item.ReceivedQuantity, &item.UnitCost, &item.SubtotalValuation); err != nil {
			return nil, err
		}
		totalValuation += item.SubtotalValuation
		grd.Items = append(grd.Items, item)
	}
	grd.TotalValuation = totalValuation

	// Lookup linked ledger entry number
	var ledgerEntryNumber string
	err = conn.QueryRow(ctx, `
		SELECT entry_number
		FROM ledger_entries
		WHERE source_document_type = 'GOODS_RECEIPT' AND source_document_id = $1
		LIMIT 1
	`, grID).Scan(&ledgerEntryNumber)
	if err == nil {
		grd.LedgerEntryNumber = &ledgerEntryNumber
	}

	return &grd, nil
}

// GetProductTraceability reconstructs the document provenance of a product from Halal cert -> PO -> GR -> stock
func (r *Repository) GetProductTraceability(ctx context.Context, conn *pgxpool.Conn, productID uuid.UUID) (*ProductTraceabilityReport, error) {
	// Product header and inventory stock
	queryProd := `
		SELECT p.id, p.sku, p.name, COALESCE(i.stock_quantity, 0)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.id = $1
	`
	var report ProductTraceabilityReport
	err := conn.QueryRow(ctx, queryProd, productID).Scan(&report.ProductID, &report.SKU, &report.Name, &report.CurrentStock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Suppliers that have supplied this product through purchase orders
	querySuppliers := `
		SELECT DISTINCT s.id, s.company_name, s.code
		FROM suppliers s
		JOIN purchase_orders po ON po.supplier_id = s.id
		JOIN purchase_order_items poi ON poi.purchase_order_id = po.id
		WHERE poi.product_id = $1
		ORDER BY s.company_name
	`
	suppRows, err := conn.Query(ctx, querySuppliers, productID)
	if err != nil {
		return nil, err
	}
	for suppRows.Next() {
		var s ProductTraceabilitySupplierInfo
		if err := suppRows.Scan(&s.SupplierID, &s.CompanyName, &s.Code); err != nil {
			suppRows.Close()
			return nil, err
		}
		report.Suppliers = append(report.Suppliers, s)
	}
	suppRows.Close()

	// Query certificates for each supplier after suppRows is closed
	for i := range report.Suppliers {
		certs, _ := r.GetCertificatesBySupplierID(ctx, conn, report.Suppliers[i].SupplierID)
		report.Suppliers[i].Certificates = certs
	}

	// Purchase orders containing this product
	queryPOs := `
		SELECT po.id, po.po_number, po.status, po.issued_date, poi.quantity, poi.unit_cost
		FROM purchase_orders po
		JOIN purchase_order_items poi ON poi.purchase_order_id = po.id
		WHERE poi.product_id = $1
		ORDER BY po.created_at DESC
	`
	poRows, err := conn.Query(ctx, queryPOs, productID)
	if err != nil {
		return nil, err
	}
	for poRows.Next() {
		var po ProductTraceabilityPOInfo
		if err := poRows.Scan(&po.POID, &po.PONumber, &po.Status, &po.IssuedDate, &po.OrderedQuantity, &po.UnitCost); err != nil {
			poRows.Close()
			return nil, err
		}
		report.PurchaseOrders = append(report.PurchaseOrders, po)
	}
	poRows.Close()

	// Goods receipts containing this product
	queryGRs := `
		SELECT gr.id, gr.gr_number, po.po_number, gr.received_date, gri.received_quantity, gr.received_by
		FROM goods_receipts gr
		JOIN purchase_orders po ON po.id = gr.purchase_order_id
		JOIN goods_receipt_items gri ON gri.goods_receipt_id = gr.id
		WHERE gri.product_id = $1
		ORDER BY gr.created_at DESC
	`
	grRows, err := conn.Query(ctx, queryGRs, productID)
	if err != nil {
		return nil, err
	}
	for grRows.Next() {
		var gr ProductTraceabilityGRInfo
		if err := grRows.Scan(&gr.GRID, &gr.GRNumber, &gr.PONumber, &gr.ReceivedDate, &gr.ReceivedQuantity, &gr.ReceivedBy); err != nil {
			grRows.Close()
			return nil, err
		}
		report.GoodsReceipts = append(report.GoodsReceipts, gr)
	}
	grRows.Close()

	return &report, nil
}
