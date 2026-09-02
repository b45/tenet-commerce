package pos

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProductNotFound          = errors.New("product not found or inactive")
	ErrInsufficientStock        = errors.New("insufficient stock for product")
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrAlreadyVoided            = errors.New("transaction is already voided")
	ErrInsufficientCashTendered = errors.New("insufficient cash tendered")
	ErrCategoryNotFound         = errors.New("category not found")
	ErrCategoryCodeExists       = errors.New("category code already exists")
	ErrSKUAlreadyExists         = errors.New("product SKU already exists")
	ErrBarcodeAlreadyExists     = errors.New("product barcode already exists")
	ErrNegativeAdjustmentStock  = errors.New("insufficient stock for negative adjustment")
)

// Repository handles database operations for the POS module within a tenant schema
type Repository struct{}

// NewRepository initializes a new POS repository
func NewRepository() *Repository {
	return &Repository{}
}

// GetProducts returns all active products with current stock quantities for the tenant catalog
func (r *Repository) GetProducts(ctx context.Context, conn *pgxpool.Conn) ([]Product, error) {
	query := `
		SELECT 
			p.id, 
			p.category_id, 
			COALESCE(c.name, '') as category_name,
			p.sku, 
			p.barcode, 
			p.name, 
			p.description, 
			p.unit_price, 
			p.cost_price, 
			COALESCE(i.stock_quantity, 0) as stock_quantity,
			COALESCE(p.compliance_tags, '[]'::jsonb) as compliance_tags, 
			p.is_active, 
			p.created_at, 
			p.updated_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.is_active = TRUE
		ORDER BY p.name ASC
	`

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var tagsBytes []byte
		if err := rows.Scan(
			&p.ID,
			&p.CategoryID,
			&p.CategoryName,
			&p.SKU,
			&p.Barcode,
			&p.Name,
			&p.Description,
			&p.UnitPrice,
			&p.CostPrice,
			&p.StockQuantity,
			&tagsBytes,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed scanning product: %w", err)
		}

		if len(tagsBytes) > 0 {
			_ = json.Unmarshal(tagsBytes, &p.ComplianceTags)
			for _, tag := range p.ComplianceTags {
				if tag == "HALAL_MUI" {
					p.IsHalalCertified = true
					break
				}
			}
		}
		if p.ComplianceTags == nil {
			p.ComplianceTags = []string{}
		}

		products = append(products, p)
	}

	return products, nil
}

// GetProductsBySKUsForUpdate acquires row-level locks on both products and inventory
// within an active transaction to guarantee concurrency safety.
func (r *Repository) GetProductsBySKUsForUpdate(ctx context.Context, tx pgx.Tx, skus []string) (map[string]*Product, error) {
	query := `
		SELECT 
			p.id, 
			p.sku, 
			p.name, 
			p.unit_price, 
			p.cost_price, 
			i.stock_quantity, 
			COALESCE(p.compliance_tags, '[]'::jsonb) as compliance_tags,
			p.is_active
		FROM products p
		INNER JOIN inventory i ON p.id = i.product_id
		WHERE p.sku = ANY($1)
		FOR UPDATE OF p, i
	`

	rows, err := tx.Query(ctx, query, skus)
	if err != nil {
		return nil, fmt.Errorf("failed to lock and query products: %w", err)
	}
	defer rows.Close()

	productMap := make(map[string]*Product)
	for rows.Next() {
		var p Product
		var tagsBytes []byte
		if err := rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.UnitPrice,
			&p.CostPrice,
			&p.StockQuantity,
			&tagsBytes,
			&p.IsActive,
		); err != nil {
			return nil, fmt.Errorf("failed scanning locked product: %w", err)
		}

		if len(tagsBytes) > 0 {
			_ = json.Unmarshal(tagsBytes, &p.ComplianceTags)
			for _, tag := range p.ComplianceTags {
				if tag == "HALAL_MUI" {
					p.IsHalalCertified = true
					break
				}
			}
		}
		if p.ComplianceTags == nil {
			p.ComplianceTags = []string{}
		}

		productMap[p.SKU] = &p
	}

	return productMap, nil
}

// DecrementStock safely reduces inventory quantity within an active transaction.
// Row-level lock was already acquired via GetProductsBySKUsForUpdate.
func (r *Repository) DecrementStock(ctx context.Context, tx pgx.Tx, productID string, quantity int) error {
	query := `
		UPDATE inventory 
		SET stock_quantity = stock_quantity - $1, 
		    updated_at = NOW()
		WHERE product_id = $2 
		  AND stock_quantity >= $1
	`

	cmdTag, err := tx.Exec(ctx, query, quantity, productID)
	if err != nil {
		return fmt.Errorf("failed executing stock decrement: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrInsufficientStock
	}

	return nil
}

// IncrementStock safely restores inventory quantity within an active transaction upon void
func (r *Repository) IncrementStock(ctx context.Context, tx pgx.Tx, productID string, quantity int) error {
	query := `
		UPDATE inventory 
		SET stock_quantity = stock_quantity + $1, 
		    updated_at = NOW()
		WHERE product_id = $2
	`

	_, err := tx.Exec(ctx, query, quantity, productID)
	if err != nil {
		return fmt.Errorf("failed executing stock increment: %w", err)
	}

	return nil
}

// GenerateTransactionNumber generates a unique, sortable transaction code: TXN-YYYYMMDD-HEX
func (r *Repository) GenerateTransactionNumber() string {
	datePart := time.Now().Format("20060102")
	randomBytes := make([]byte, 3)
	_, _ = rand.Read(randomBytes)
	hexPart := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("TXN-%s-%s", datePart, hexPart)
}

// CreateTransaction inserts a master transaction record inside an active database transaction
func (r *Repository) CreateTransaction(ctx context.Context, tx pgx.Tx, txn *Transaction) error {
	query := `
		INSERT INTO transactions (
			transaction_number,
			idempotency_key,
			cashier_id,
			subtotal_amount,
			tax_amount,
			discount_amount,
			total_amount,
			payment_method,
			status,
			customer_name,
			notes,
			cash_tendered,
			change_amount,
			payment_reference
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at
	`

	err := tx.QueryRow(
		ctx,
		query,
		txn.TransactionNumber,
		txn.IdempotencyKey,
		txn.CashierID,
		txn.SubtotalAmount,
		txn.TaxAmount,
		txn.DiscountAmount,
		txn.TotalAmount,
		txn.PaymentMethod,
		txn.Status,
		txn.CustomerName,
		txn.Notes,
		txn.CashTendered,
		txn.ChangeAmount,
		txn.PaymentReference,
	).Scan(&txn.ID, &txn.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert transaction record: %w", err)
	}

	return nil
}

// CreateTransactionItems bulk inserts items for a transaction in a single multi-row query for high throughput
func (r *Repository) CreateTransactionItems(ctx context.Context, tx pgx.Tx, items []TransactionItem) error {
	if len(items) == 0 {
		return nil
	}

	query := `
		INSERT INTO transaction_items (
			transaction_id,
			product_id,
			quantity,
			unit_price,
			cost_price,
			subtotal
		) VALUES `

	values := make([]interface{}, 0, len(items)*6)
	valuePlaceholders := make([]string, len(items))

	for i, item := range items {
		offset := i * 6
		valuePlaceholders[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5, offset+6)
		values = append(values,
			item.TransactionID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.CostPrice,
			item.Subtotal,
		)
	}

	query += fmt.Sprintf("%s", strings.Join(valuePlaceholders, ", "))
	if _, err := tx.Exec(ctx, query, values...); err != nil {
		return fmt.Errorf("failed executing bulk insert transaction items: %w", err)
	}

	return nil
}

// GetOrders retrieves a paginated and filterable list of orders
func (r *Repository) GetOrders(ctx context.Context, conn *pgxpool.Conn, filter OrderFilter) ([]OrderSummary, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.StartDate != "" {
		conditions = append(conditions, fmt.Sprintf("t.created_at >= $%d::timestamptz", argIdx))
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("t.created_at <= $%d::timestamptz + INTERVAL '1 day'", argIdx))
		args = append(args, filter.EndDate)
		argIdx++
	}
	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("t.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.PaymentMethod != "" {
		conditions = append(conditions, fmt.Sprintf("t.payment_method = $%d", argIdx))
		args = append(args, filter.PaymentMethod)
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(t.transaction_number ILIKE '%%' || $%d || '%%' OR t.customer_name ILIKE '%%' || $%d || '%%' OR t.notes ILIKE '%%' || $%d || '%%')", argIdx, argIdx, argIdx))
		args = append(args, filter.Search)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total matching orders
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transactions t %s", whereClause)
	var total int
	if err := conn.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed counting orders: %w", err)
	}

	// Fetch paginated summaries
	query := fmt.Sprintf(`
		SELECT 
			t.id,
			t.transaction_number,
			t.cashier_id,
			t.total_amount,
			t.payment_method,
			t.status,
			t.customer_name,
			COALESCE(SUM(ti.quantity), 0) AS total_items,
			t.void_reason,
			t.created_at
		FROM transactions t
		LEFT JOIN transaction_items ti ON t.id = ti.transaction_id
		%s
		GROUP BY t.id, t.transaction_number, t.cashier_id, t.total_amount, t.payment_method, t.status, t.customer_name, t.void_reason, t.created_at
		ORDER BY t.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed querying orders: %w", err)
	}
	defer rows.Close()

	var orders []OrderSummary
	for rows.Next() {
		var o OrderSummary
		if err := rows.Scan(
			&o.ID,
			&o.TransactionNumber,
			&o.CashierID,
			&o.TotalAmount,
			&o.PaymentMethod,
			&o.Status,
			&o.CustomerName,
			&o.TotalItems,
			&o.VoidReason,
			&o.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed scanning order summary: %w", err)
		}
		orders = append(orders, o)
	}

	if orders == nil {
		orders = []OrderSummary{}
	}

	return orders, total, nil
}

// GetOrderByID retrieves the full order detail with line items by UUID or transaction number
func (r *Repository) GetOrderByID(ctx context.Context, conn *pgxpool.Conn, idOrNumber string) (*OrderDetailResponse, error) {
	queryTxn := `
		SELECT 
			id,
			transaction_number,
			idempotency_key,
			cashier_id,
			subtotal_amount,
			tax_amount,
			discount_amount,
			total_amount,
			payment_method,
			status,
			customer_name,
			notes,
			COALESCE(cash_tendered, 0),
			COALESCE(change_amount, 0),
			payment_reference,
			void_reason,
			voided_at,
			voided_by,
			created_at
		FROM transactions
		WHERE id::text = $1 OR transaction_number = $1
	`

	var t Transaction
	err := conn.QueryRow(ctx, queryTxn, idOrNumber).Scan(
		&t.ID,
		&t.TransactionNumber,
		&t.IdempotencyKey,
		&t.CashierID,
		&t.SubtotalAmount,
		&t.TaxAmount,
		&t.DiscountAmount,
		&t.TotalAmount,
		&t.PaymentMethod,
		&t.Status,
		&t.CustomerName,
		&t.Notes,
		&t.CashTendered,
		&t.ChangeAmount,
		&t.PaymentReference,
		&t.VoidReason,
		&t.VoidedAt,
		&t.VoidedBy,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed querying transaction by id: %w", err)
	}

	queryItems := `
		SELECT 
			ti.id,
			ti.transaction_id,
			ti.product_id,
			p.sku,
			p.name,
			ti.quantity,
			ti.unit_price,
			ti.cost_price,
			ti.subtotal
		FROM transaction_items ti
		JOIN products p ON ti.product_id = p.id
		WHERE ti.transaction_id = $1
		ORDER BY ti.id ASC
	`
	rows, err := conn.Query(ctx, queryItems, t.ID)
	if err != nil {
		return nil, fmt.Errorf("failed querying transaction items: %w", err)
	}
	defer rows.Close()

	var items []TransactionItem
	for rows.Next() {
		var item TransactionItem
		if err := rows.Scan(
			&item.ID,
			&item.TransactionID,
			&item.ProductID,
			&item.SKU,
			&item.Name,
			&item.Quantity,
			&item.UnitPrice,
			&item.CostPrice,
			&item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("failed scanning transaction item: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []TransactionItem{}
	}

	return &OrderDetailResponse{
		Transaction: t,
		Items:       items,
	}, nil
}

// GetTransactionForUpdate locks the transaction row inside an active database transaction
func (r *Repository) GetTransactionForUpdate(ctx context.Context, tx pgx.Tx, txnID string) (*Transaction, error) {
	query := `
		SELECT 
			id,
			transaction_number,
			idempotency_key,
			cashier_id,
			subtotal_amount,
			tax_amount,
			discount_amount,
			total_amount,
			payment_method,
			status,
			customer_name,
			notes,
			COALESCE(cash_tendered, 0),
			COALESCE(change_amount, 0),
			payment_reference,
			void_reason,
			voided_at,
			voided_by,
			created_at
		FROM transactions
		WHERE id::text = $1 OR transaction_number = $1
		FOR UPDATE
	`

	var t Transaction
	err := tx.QueryRow(ctx, query, txnID).Scan(
		&t.ID,
		&t.TransactionNumber,
		&t.IdempotencyKey,
		&t.CashierID,
		&t.SubtotalAmount,
		&t.TaxAmount,
		&t.DiscountAmount,
		&t.TotalAmount,
		&t.PaymentMethod,
		&t.Status,
		&t.CustomerName,
		&t.Notes,
		&t.CashTendered,
		&t.ChangeAmount,
		&t.PaymentReference,
		&t.VoidReason,
		&t.VoidedAt,
		&t.VoidedBy,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed querying transaction for update: %w", err)
	}

	return &t, nil
}

// GetTransactionItems retrieves line items for a transaction within an active transaction
func (r *Repository) GetTransactionItems(ctx context.Context, tx pgx.Tx, txnID string) ([]TransactionItem, error) {
	query := `
		SELECT 
			ti.id,
			ti.transaction_id,
			ti.product_id,
			p.sku,
			p.name,
			ti.quantity,
			ti.unit_price,
			ti.cost_price,
			ti.subtotal
		FROM transaction_items ti
		JOIN products p ON ti.product_id = p.id
		WHERE ti.transaction_id = $1
	`
	rows, err := tx.Query(ctx, query, txnID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transaction items: %w", err)
	}
	defer rows.Close()

	var items []TransactionItem
	for rows.Next() {
		var item TransactionItem
		if err := rows.Scan(
			&item.ID,
			&item.TransactionID,
			&item.ProductID,
			&item.SKU,
			&item.Name,
			&item.Quantity,
			&item.UnitPrice,
			&item.CostPrice,
			&item.Subtotal,
		); err != nil {
			return nil, fmt.Errorf("failed scanning item: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// MarkTransactionVoided updates the transaction status to VOIDED with reason and audit metadata
func (r *Repository) MarkTransactionVoided(ctx context.Context, tx pgx.Tx, txnID, voidedBy, reason string) error {
	query := `
		UPDATE transactions
		SET status = 'VOIDED',
		    void_reason = $1,
		    voided_at = NOW(),
		    voided_by = $2
		WHERE id = $3
	`
	_, err := tx.Exec(ctx, query, reason, voidedBy, txnID)
	if err != nil {
		return fmt.Errorf("failed marking transaction as voided: %w", err)
	}
	return nil
}

// GetDailySummary aggregates sales metrics, COGS, and payment breakdowns for a given date
func (r *Repository) GetDailySummary(ctx context.Context, conn *pgxpool.Conn, date string, cashierID *string) (*DailySummaryResponse, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	whereClause := "WHERE t.created_at::date = $1::date"
	args := []interface{}{date}
	if cashierID != nil && *cashierID != "" {
		whereClause += " AND t.cashier_id = $2"
		args = append(args, *cashierID)
	}

	summaryQuery := fmt.Sprintf(`
		SELECT 
			COUNT(t.id) AS total_orders,
			COUNT(t.id) FILTER (WHERE t.status = 'COMPLETED') AS completed_orders,
			COUNT(t.id) FILTER (WHERE t.status = 'VOIDED') AS voided_orders,
			COALESCE(SUM(t.subtotal_amount) FILTER (WHERE t.status = 'COMPLETED'), 0) AS gross_sales,
			COALESCE(SUM(t.discount_amount) FILTER (WHERE t.status = 'COMPLETED'), 0) AS discounts,
			COALESCE(SUM(t.total_amount) FILTER (WHERE t.status = 'COMPLETED'), 0) AS net_sales
		FROM transactions t
		%s
	`, whereClause)

	res := &DailySummaryResponse{
		Date:             date,
		CashierID:        cashierID,
		PaymentBreakdown: make(map[string]PaymentSummary),
	}

	if err := conn.QueryRow(ctx, summaryQuery, args...).Scan(
		&res.TotalOrders,
		&res.CompletedOrders,
		&res.VoidedOrders,
		&res.GrossSales,
		&res.Discounts,
		&res.NetSales,
	); err != nil {
		return nil, fmt.Errorf("failed querying daily summary totals: %w", err)
	}

	// Query COGS for completed orders
	cogsQuery := fmt.Sprintf(`
		SELECT COALESCE(SUM(ti.quantity * ti.cost_price), 0)
		FROM transaction_items ti
		JOIN transactions t ON ti.transaction_id = t.id
		%s AND t.status = 'COMPLETED'
	`, whereClause)

	if err := conn.QueryRow(ctx, cogsQuery, args...).Scan(&res.TotalCOGS); err != nil {
		return nil, fmt.Errorf("failed querying daily summary cogs: %w", err)
	}
	res.GrossProfit = res.NetSales - res.TotalCOGS

	// Query payment breakdown for completed orders
	breakdownQuery := fmt.Sprintf(`
		SELECT 
			t.payment_method,
			COUNT(t.id),
			COALESCE(SUM(t.total_amount), 0)
		FROM transactions t
		%s AND t.status = 'COMPLETED'
		GROUP BY t.payment_method
	`, whereClause)

	rows, err := conn.Query(ctx, breakdownQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed querying payment breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var method string
		var count int
		var amount float64
		if err := rows.Scan(&method, &count, &amount); err != nil {
			return nil, fmt.Errorf("failed scanning payment breakdown: %w", err)
		}
		res.PaymentBreakdown[method] = PaymentSummary{
			Count:       count,
			TotalAmount: amount,
		}
	}

	return res, nil
}

// GetQRISConfig retrieves the tenant's QRIS payload from store_settings
func (r *Repository) GetQRISConfig(ctx context.Context, conn *pgxpool.Conn) (*QRISConfig, error) {
	query := `SELECT value FROM store_settings WHERE key = 'qris'`
	var valBytes []byte
	err := conn.QueryRow(ctx, query).Scan(&valBytes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Return default fallback configuration
			return &QRISConfig{
				MerchantName: "Tenet Bakery & Store",
				NMID:         "ID1020030040050",
				QRString:     "00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924TENET BAKERY & STORE6010JAKARTA SE61051234062070703A0163041D2B",
				QRImageURL:   "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924",
			}, nil
		}
		return nil, fmt.Errorf("failed to retrieve QRIS config: %w", err)
	}

	var config QRISConfig
	if err := json.Unmarshal(valBytes, &config); err != nil {
		return nil, fmt.Errorf("failed parsing QRIS config json: %w", err)
	}
	return &config, nil
}

// UpdateQRISConfig upserts the tenant's QRIS configuration in store_settings
func (r *Repository) UpdateQRISConfig(ctx context.Context, conn *pgxpool.Conn, cfg QRISConfig) error {
	valBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed serializing QRIS config: %w", err)
	}

	query := `
		INSERT INTO store_settings (key, value, updated_at)
		VALUES ('qris', $1, NOW())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()
	`
	if _, err := conn.Exec(ctx, query, valBytes); err != nil {
		return fmt.Errorf("failed updating QRIS config: %w", err)
	}
	return nil
}

// GetCategories returns all product categories with their current active product counts
func (r *Repository) GetCategories(ctx context.Context, conn *pgxpool.Conn) ([]Category, error) {
	query := `
		SELECT 
			c.id, 
			c.name, 
			c.code, 
			c.parent_id, 
			COUNT(p.id) as product_count, 
			c.created_at
		FROM categories c
		LEFT JOIN products p ON c.id = p.category_id AND p.is_active = TRUE
		GROUP BY c.id, c.name, c.code, c.parent_id, c.created_at
		ORDER BY c.name ASC
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed querying categories: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		var parentID *string
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Code, &parentID, &cat.ProductCount, &cat.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed scanning category: %w", err)
		}
		cat.ParentID = parentID
		categories = append(categories, cat)
	}
	if categories == nil {
		categories = []Category{}
	}
	return categories, nil
}

// GetCategoryByID returns a category by its UUID
func (r *Repository) GetCategoryByID(ctx context.Context, conn *pgxpool.Conn, id string) (*Category, error) {
	query := `
		SELECT 
			c.id, 
			c.name, 
			c.code, 
			c.parent_id, 
			COUNT(p.id) as product_count, 
			c.created_at
		FROM categories c
		LEFT JOIN products p ON c.id = p.category_id AND p.is_active = TRUE
		WHERE c.id = $1
		GROUP BY c.id, c.name, c.code, c.parent_id, c.created_at
	`
	var cat Category
	var parentID *string
	err := conn.QueryRow(ctx, query, id).Scan(&cat.ID, &cat.Name, &cat.Code, &parentID, &cat.ProductCount, &cat.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed querying category: %w", err)
	}
	cat.ParentID = parentID
	return &cat, nil
}

// CreateCategory inserts a new product category
func (r *Repository) CreateCategory(ctx context.Context, conn *pgxpool.Conn, req CreateCategoryRequest) (*Category, error) {
	query := `
		INSERT INTO categories (name, code, parent_id, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, name, code, parent_id, 0 as product_count, created_at
	`
	var cat Category
	var parentID *string
	err := conn.QueryRow(ctx, query, req.Name, strings.ToUpper(strings.TrimSpace(req.Code)), req.ParentID).
		Scan(&cat.ID, &cat.Name, &cat.Code, &parentID, &cat.ProductCount, &cat.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "categories_code_key") {
			return nil, ErrCategoryCodeExists
		}
		return nil, fmt.Errorf("failed creating category: %w", err)
	}
	cat.ParentID = parentID
	return &cat, nil
}

// UpdateCategory updates an existing category
func (r *Repository) UpdateCategory(ctx context.Context, conn *pgxpool.Conn, id string, req UpdateCategoryRequest) (*Category, error) {
	query := `
		UPDATE categories
		SET name = $1, code = $2, parent_id = $3
		WHERE id = $4
		RETURNING id, name, code, parent_id, created_at
	`
	var cat Category
	var parentID *string
	err := conn.QueryRow(ctx, query, req.Name, strings.ToUpper(strings.TrimSpace(req.Code)), req.ParentID, id).
		Scan(&cat.ID, &cat.Name, &cat.Code, &parentID, &cat.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "categories_code_key") {
			return nil, ErrCategoryCodeExists
		}
		return nil, fmt.Errorf("failed updating category: %w", err)
	}
	cat.ParentID = parentID

	_ = conn.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE category_id = $1 AND is_active = TRUE`, id).Scan(&cat.ProductCount)
	return &cat, nil
}

// DeleteCategory removes a category, unlinking any assigned products
func (r *Repository) DeleteCategory(ctx context.Context, conn *pgxpool.Conn, id string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE products SET category_id = NULL WHERE category_id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed unlinking category products: %w", err)
	}

	res, err := tx.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed deleting category: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}

	return tx.Commit(ctx)
}

// GetProductByID retrieves a product by its ID including category name and real-time inventory quantity
func (r *Repository) GetProductByID(ctx context.Context, conn *pgxpool.Conn, id string) (*Product, error) {
	query := `
		SELECT 
			p.id, 
			p.category_id, 
			COALESCE(c.name, '') as category_name, 
			p.sku, 
			p.barcode, 
			p.name, 
			p.description, 
			p.unit_price, 
			p.cost_price, 
			COALESCE(i.stock_quantity, 0) as stock_quantity, 
			COALESCE(p.compliance_tags, '[]'::jsonb) as compliance_tags, 
			p.is_active, 
			p.created_at, 
			p.updated_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		LEFT JOIN inventory i ON p.id = i.product_id
		WHERE p.id = $1
	`
	var p Product
	var tagsBytes []byte
	err := conn.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.CategoryID,
		&p.CategoryName,
		&p.SKU,
		&p.Barcode,
		&p.Name,
		&p.Description,
		&p.UnitPrice,
		&p.CostPrice,
		&p.StockQuantity,
		&tagsBytes,
		&p.IsActive,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed querying product: %w", err)
	}

	if len(tagsBytes) > 0 {
		_ = json.Unmarshal(tagsBytes, &p.ComplianceTags)
	}
	for _, tag := range p.ComplianceTags {
		if strings.Contains(strings.ToUpper(tag), "HALAL") {
			p.IsHalalCertified = true
			break
		}
	}
	return &p, nil
}

// CreateProduct inserts a new product and initializes its inventory entry atomically
func (r *Repository) CreateProduct(ctx context.Context, conn *pgxpool.Conn, req CreateProductRequest) (*Product, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tagsJSON, err := json.Marshal(req.ComplianceTags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	warehouseLocation := "MAIN_STORE"
	if strings.TrimSpace(req.WarehouseLocation) != "" {
		warehouseLocation = req.WarehouseLocation
	}

	reorderThreshold := 10
	if req.ReorderThreshold > 0 {
		reorderThreshold = req.ReorderThreshold
	}

	productQuery := `
		INSERT INTO products (
			name, sku, barcode, description, category_id, unit_price, cost_price, compliance_tags, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
		) RETURNING id, created_at, updated_at
	`
	var productID string
	var createdAt, updatedAt time.Time

	err = tx.QueryRow(
		ctx,
		productQuery,
		req.Name,
		strings.TrimSpace(req.SKU),
		req.Barcode,
		req.Description,
		req.CategoryID,
		req.UnitPrice,
		req.CostPrice,
		tagsJSON,
		isActive,
	).Scan(&productID, &createdAt, &updatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "products_sku_key") || strings.Contains(err.Error(), "sku") {
			return nil, ErrSKUAlreadyExists
		}
		if strings.Contains(err.Error(), "products_barcode_key") || strings.Contains(err.Error(), "barcode") {
			return nil, ErrBarcodeAlreadyExists
		}
		return nil, fmt.Errorf("failed inserting product: %w", err)
	}

	inventoryQuery := `
		INSERT INTO inventory (product_id, stock_quantity, reorder_threshold, warehouse_location, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	if _, err := tx.Exec(ctx, inventoryQuery, productID, req.InitialStock, reorderThreshold, warehouseLocation); err != nil {
		return nil, fmt.Errorf("failed inserting inventory record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed committing product creation: %w", err)
	}

	var categoryName string
	if req.CategoryID != nil {
		_ = conn.QueryRow(ctx, `SELECT name FROM categories WHERE id = $1`, *req.CategoryID).Scan(&categoryName)
	}

	isHalal := false
	for _, tag := range req.ComplianceTags {
		if strings.Contains(strings.ToUpper(tag), "HALAL") {
			isHalal = true
			break
		}
	}

	return &Product{
		ID:               productID,
		CategoryID:       req.CategoryID,
		CategoryName:     categoryName,
		SKU:              req.SKU,
		Barcode:          req.Barcode,
		Name:             req.Name,
		Description:      req.Description,
		UnitPrice:        req.UnitPrice,
		CostPrice:        req.CostPrice,
		StockQuantity:    req.InitialStock,
		ComplianceTags:   req.ComplianceTags,
		IsHalalCertified: isHalal,
		IsActive:         isActive,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}, nil
}

// UpdateProduct updates product metadata and inventory reorder parameters atomically
func (r *Repository) UpdateProduct(ctx context.Context, conn *pgxpool.Conn, id string, req UpdateProductRequest) (*Product, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	tagsJSON, err := json.Marshal(req.ComplianceTags)
	if err != nil {
		tagsJSON = []byte("[]")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	query := `
		UPDATE products
		SET name = $1, barcode = $2, description = $3, category_id = $4,
		    unit_price = $5, cost_price = $6, compliance_tags = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING sku, created_at, updated_at
	`
	var sku string
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(
		ctx,
		query,
		req.Name,
		req.Barcode,
		req.Description,
		req.CategoryID,
		req.UnitPrice,
		req.CostPrice,
		tagsJSON,
		isActive,
		id,
	).Scan(&sku, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		if strings.Contains(err.Error(), "products_barcode_key") {
			return nil, ErrBarcodeAlreadyExists
		}
		return nil, fmt.Errorf("failed updating product: %w", err)
	}

	if req.ReorderThreshold > 0 || req.WarehouseLocation != "" {
		invUpdate := `
			UPDATE inventory
			SET reorder_threshold = CASE WHEN $1 > 0 THEN $1 ELSE reorder_threshold END,
			    warehouse_location = CASE WHEN $2 <> '' THEN $2 ELSE warehouse_location END,
			    updated_at = NOW()
			WHERE product_id = $3
		`
		if _, err := tx.Exec(ctx, invUpdate, req.ReorderThreshold, req.WarehouseLocation, id); err != nil {
			return nil, fmt.Errorf("failed updating inventory threshold: %w", err)
		}
	}

	var stockQty int
	_ = tx.QueryRow(ctx, `SELECT stock_quantity FROM inventory WHERE product_id = $1`, id).Scan(&stockQty)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed committing product update: %w", err)
	}

	var categoryName string
	if req.CategoryID != nil {
		_ = conn.QueryRow(ctx, `SELECT name FROM categories WHERE id = $1`, *req.CategoryID).Scan(&categoryName)
	}

	isHalal := false
	for _, tag := range req.ComplianceTags {
		if strings.Contains(strings.ToUpper(tag), "HALAL") {
			isHalal = true
			break
		}
	}

	return &Product{
		ID:               id,
		CategoryID:       req.CategoryID,
		CategoryName:     categoryName,
		SKU:              sku,
		Barcode:          req.Barcode,
		Name:             req.Name,
		Description:      req.Description,
		UnitPrice:        req.UnitPrice,
		CostPrice:        req.CostPrice,
		StockQuantity:    stockQty,
		ComplianceTags:   req.ComplianceTags,
		IsHalalCertified: isHalal,
		IsActive:         isActive,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}, nil
}

// DeleteProduct performs a soft delete by marking is_active = FALSE
func (r *Repository) DeleteProduct(ctx context.Context, conn *pgxpool.Conn, id string) error {
	res, err := conn.Exec(ctx, `UPDATE products SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed soft-deleting product: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

// AdjustInventoryStock applies an atomic stock adjustment to an inventory item with row-level locking
func (r *Repository) AdjustInventoryStock(ctx context.Context, tx pgx.Tx, productID string, adjType string, qty int) (prevQty int, newQty int, deltaQty int, err error) {
	err = tx.QueryRow(ctx, `SELECT stock_quantity FROM inventory WHERE product_id = $1 FOR UPDATE`, productID).Scan(&prevQty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, 0, ErrProductNotFound
		}
		return 0, 0, 0, fmt.Errorf("failed locking inventory: %w", err)
	}

	switch adjType {
	case "ADD":
		deltaQty = qty
		newQty = prevQty + qty
	case "SUBTRACT":
		deltaQty = -qty
		newQty = prevQty - qty
	case "SET":
		deltaQty = qty - prevQty
		newQty = qty
	default:
		return 0, 0, 0, fmt.Errorf("invalid adjustment type: %s", adjType)
	}

	if newQty < 0 {
		return 0, 0, 0, ErrNegativeAdjustmentStock
	}

	_, err = tx.Exec(ctx, `UPDATE inventory SET stock_quantity = $1, updated_at = NOW() WHERE product_id = $2`, newQty, productID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed updating inventory stock: %w", err)
	}

	return prevQty, newQty, deltaQty, nil
}

// RecordInventoryAdjustment inserts an audit log for an inventory adjustment
func (r *Repository) RecordInventoryAdjustment(ctx context.Context, tx pgx.Tx, adjID string, productID string, adjType string, deltaQty int, prevQty int, newQty int, reason string, notes string, userID string, ledgerEntryID *string) error {
	query := `
		INSERT INTO inventory_adjustments (
			id, product_id, adjustment_type, quantity_delta, previous_quantity, new_quantity, reason, notes, adjusted_by, ledger_entry_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW()
		)
	`
	_, err := tx.Exec(ctx, query, adjID, productID, adjType, deltaQty, prevQty, newQty, reason, notes, userID, ledgerEntryID)
	if err != nil {
		return fmt.Errorf("failed recording inventory adjustment audit: %w", err)
	}
	return nil
}

// GetLowStockProducts returns all active products whose stock quantity is at or below their reorder threshold
func (r *Repository) GetLowStockProducts(ctx context.Context, conn *pgxpool.Conn) ([]Product, error) {
	query := `
		SELECT 
			p.id, 
			p.category_id, 
			COALESCE(c.name, '') as category_name, 
			p.sku, 
			p.barcode, 
			p.name, 
			p.description, 
			p.unit_price, 
			p.cost_price, 
			COALESCE(i.stock_quantity, 0) as stock_quantity, 
			COALESCE(p.compliance_tags, '[]'::jsonb) as compliance_tags, 
			p.is_active, 
			p.created_at, 
			p.updated_at
		FROM products p
		LEFT JOIN categories c ON p.category_id = c.id
		JOIN inventory i ON p.id = i.product_id
		WHERE p.is_active = TRUE AND i.stock_quantity <= i.reorder_threshold
		ORDER BY (i.stock_quantity - i.reorder_threshold) ASC, p.name ASC
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed querying low stock products: %w", err)
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		var tagsBytes []byte
		if err := rows.Scan(
			&p.ID,
			&p.CategoryID,
			&p.CategoryName,
			&p.SKU,
			&p.Barcode,
			&p.Name,
			&p.Description,
			&p.UnitPrice,
			&p.CostPrice,
			&p.StockQuantity,
			&tagsBytes,
			&p.IsActive,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed scanning low stock product: %w", err)
		}

		if len(tagsBytes) > 0 {
			_ = json.Unmarshal(tagsBytes, &p.ComplianceTags)
		}
		for _, tag := range p.ComplianceTags {
			if strings.Contains(strings.ToUpper(tag), "HALAL") {
				p.IsHalalCertified = true
				break
			}
		}
		products = append(products, p)
	}
	if products == nil {
		products = []Product{}
	}
	return products, nil
}

