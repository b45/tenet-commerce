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
	ErrProductNotFound   = errors.New("product not found or inactive")
	ErrInsufficientStock = errors.New("insufficient stock for product")
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
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
