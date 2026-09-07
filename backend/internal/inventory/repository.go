package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInsufficientStock       = errors.New("insufficient stock for requested operation")
	ErrInvalidMovementType     = errors.New("invalid stock movement type")
	ErrInvalidQuantityDelta    = errors.New("quantity delta violates movement type direction constraint")
	ErrProductNotFound         = errors.New("product not found in inventory")
	ErrDuplicateMovementSource = errors.New("stock movement source document already posted")
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// LockInventoryForUpdate locks the product's inventory record with row-level locking (SELECT ... FOR UPDATE).
// It accepts caller's active pgx.Tx and never self-commits.
func (r *Repository) LockInventoryForUpdate(ctx context.Context, tx pgx.Tx, productID uuid.UUID) (stockQty int, warehouseLocation string, err error) {
	query := `
		SELECT stock_quantity, COALESCE(warehouse_location, 'MAIN_STORE')
		FROM inventory
		WHERE product_id = $1
		FOR UPDATE
	`
	err = tx.QueryRow(ctx, query, productID).Scan(&stockQty, &warehouseLocation)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", ErrProductNotFound
		}
		return 0, "", fmt.Errorf("failed to lock inventory row: %w", err)
	}
	return stockQty, warehouseLocation, nil
}

// PostMovementTx posts an append-only stock movement and atomically adjusts the inventory balance.
// Invariant: The caller MUST supply an active pgx.Tx. Self-commit is strictly forbidden.
func (r *Repository) PostMovementTx(ctx context.Context, tx pgx.Tx, p PostMovementParams) (*StockMovement, error) {
	// 1. Validate movement type & signed delta direction
	if err := validateMovementParams(p); err != nil {
		return nil, err
	}

	location := p.WarehouseLocation
	if strings.TrimSpace(location) == "" {
		location = DefaultCoreLocation
	}

	occurredAt := p.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	// 2. Lock inventory row before writing movement or updating balance
	currQty, _, err := r.LockInventoryForUpdate(ctx, tx, p.ProductID)
	if err != nil {
		return nil, err
	}

	newQty := currQty + p.QuantityDelta
	if newQty < 0 {
		return nil, ErrInsufficientStock
	}

	// 3. Insert append-only movement
	movementID := uuid.New()
	insertQuery := `
		INSERT INTO stock_movements (
			id, product_id, warehouse_location, quantity_delta, movement_type,
			source_document_type, source_document_id, source_document_line_id,
			actor_id, reason, occurred_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, NOW()
		)
		RETURNING created_at
	`
	var createdAt time.Time
	err = tx.QueryRow(
		ctx,
		insertQuery,
		movementID,
		p.ProductID,
		location,
		p.QuantityDelta,
		p.MovementType,
		p.SourceDocumentType,
		p.SourceDocumentID,
		p.SourceDocumentLineID,
		p.ActorID,
		p.Reason,
		occurredAt,
	).Scan(&createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "uq_stock_movement_source") {
			return nil, ErrDuplicateMovementSource
		}
		return nil, fmt.Errorf("failed to insert stock movement: %w", err)
	}

	// 4. Update inventory balance atomically
	updateQuery := `
		UPDATE inventory
		SET stock_quantity = $1, updated_at = NOW()
		WHERE product_id = $2
	`
	cmdTag, err := tx.Exec(ctx, updateQuery, newQty, p.ProductID)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory balance: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, ErrProductNotFound
	}

	return &StockMovement{
		ID:                   movementID,
		ProductID:            p.ProductID,
		WarehouseLocation:    location,
		QuantityDelta:        p.QuantityDelta,
		MovementType:         p.MovementType,
		SourceDocumentType:   p.SourceDocumentType,
		SourceDocumentID:     p.SourceDocumentID,
		SourceDocumentLineID: p.SourceDocumentLineID,
		ActorID:              p.ActorID,
		Reason:               p.Reason,
		OccurredAt:           occurredAt,
		CreatedAt:            createdAt,
	}, nil
}

// GetStockMovementsByProduct returns historical stock movements for a product ordered by occurred_at DESC, created_at DESC
func (r *Repository) GetStockMovementsByProduct(ctx context.Context, conn *pgxpool.Conn, productID uuid.UUID, limit, offset int) ([]StockMovement, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT
			id, product_id, warehouse_location, quantity_delta, movement_type,
			source_document_type, source_document_id, source_document_line_id,
			actor_id, reason, occurred_at, created_at
		FROM stock_movements
		WHERE product_id = $1
		ORDER BY occurred_at DESC, created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := conn.Query(ctx, query, productID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed querying stock movements: %w", err)
	}
	defer rows.Close()

	var movements []StockMovement
	for rows.Next() {
		var sm StockMovement
		if err := rows.Scan(
			&sm.ID,
			&sm.ProductID,
			&sm.WarehouseLocation,
			&sm.QuantityDelta,
			&sm.MovementType,
			&sm.SourceDocumentType,
			&sm.SourceDocumentID,
			&sm.SourceDocumentLineID,
			&sm.ActorID,
			&sm.Reason,
			&sm.OccurredAt,
			&sm.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed scanning stock movement: %w", err)
		}
		movements = append(movements, sm)
	}
	return movements, nil
}

// GetReconciledBalance returns the current on-hand quantity alongside opening balance and sum of subsequent deltas
// Verifies acceptance invariant: opening + sum(subsequent delta) == on_hand
func (r *Repository) GetReconciledBalance(ctx context.Context, conn *pgxpool.Conn, productID uuid.UUID) (*CurrentBalance, error) {
	query := `
		SELECT
			i.product_id,
			COALESCE(i.warehouse_location, 'MAIN_STORE') as warehouse_location,
			i.stock_quantity as on_hand,
			COALESCE(sm.opening, 0) as opening,
			COALESCE(sm.subsequent_delta, 0) as subsequent_delta
		FROM inventory i
		LEFT JOIN (
			SELECT
				product_id,
				SUM(CASE WHEN movement_type = 'OPENING' THEN quantity_delta ELSE 0 END) as opening,
				SUM(CASE WHEN movement_type <> 'OPENING' THEN quantity_delta ELSE 0 END) as subsequent_delta
			FROM stock_movements
			WHERE product_id = $1
			GROUP BY product_id
		) sm ON sm.product_id = i.product_id
		WHERE i.product_id = $1
	`
	var bal CurrentBalance
	err := conn.QueryRow(ctx, query, productID).Scan(
		&bal.ProductID,
		&bal.WarehouseLocation,
		&bal.OnHandQuantity,
		&bal.OpeningQuantity,
		&bal.SubsequentDelta,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("failed querying reconciled balance: %w", err)
	}

	return &bal, nil
}

func validateMovementParams(p PostMovementParams) error {
	switch p.MovementType {
	case MovementTypeOpening:
		if p.QuantityDelta < 0 {
			return fmt.Errorf("%w: OPENING movement delta must be non-negative", ErrInvalidQuantityDelta)
		}
	case MovementTypeIn:
		if p.QuantityDelta <= 0 {
			return fmt.Errorf("%w: IN movement delta must be strictly positive", ErrInvalidQuantityDelta)
		}
	case MovementTypeOut:
		if p.QuantityDelta >= 0 {
			return fmt.Errorf("%w: OUT movement delta must be strictly negative", ErrInvalidQuantityDelta)
		}
	case MovementTypeAdjustment:
		if p.QuantityDelta == 0 {
			return fmt.Errorf("%w: ADJUSTMENT movement delta cannot be zero", ErrInvalidQuantityDelta)
		}
	default:
		return fmt.Errorf("%w: %s", ErrInvalidMovementType, p.MovementType)
	}

	if strings.TrimSpace(p.SourceDocumentType) == "" {
		return errors.New("source_document_type is required")
	}

	return nil
}
