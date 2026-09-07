package inventory_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/inventory"
	"github.com/b45/tenet-commerce/backend/pkg/database"
)

func setupTestConn(t *testing.T) (*database.PostgresDB, *pgxpool.Conn) {
	ctx := context.Background()
	db, err := database.NewPostgresDB(ctx)
	if err != nil {
		t.Skipf("Skipping inventory test: Database not available: %v", err)
	}

	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		db.Close()
		t.Fatalf("Failed acquiring connection: %v", err)
	}

	_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
	if err != nil {
		conn.Release()
		db.Close()
		t.Fatalf("Failed setting search path: %v", err)
	}

	return db, conn
}

// createDedicatedTestProduct inserts an isolated product + inventory row + opening movement
// ensuring no cross-test pollution from shared seed data.
func createDedicatedTestProduct(t *testing.T, conn *pgxpool.Conn, initialStock int) (uuid.UUID, func()) {
	ctx := context.Background()
	prodID := uuid.New()
	sku := fmt.Sprintf("SKU-INVTEST-%d", time.Now().UnixNano())

	_, err := conn.Exec(ctx, `
		INSERT INTO products (id, sku, name, unit_price, cost_price, is_active)
		VALUES ($1, $2, 'Dedicated Inventory Test SKU', 20000, 15000, TRUE)
	`, prodID, sku)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, `
		INSERT INTO inventory (product_id, stock_quantity, warehouse_location)
		VALUES ($1, $2, 'MAIN_STORE')
	`, prodID, initialStock)
	require.NoError(t, err)

	if initialStock > 0 {
		_, err = conn.Exec(ctx, `
			INSERT INTO stock_movements (
				product_id, warehouse_location, quantity_delta, movement_type,
				source_document_type, source_document_id, reason, occurred_at, created_at
			) VALUES (
				$1, 'MAIN_STORE', $2, 'OPENING',
				'OPENING_BALANCE', $1, 'Initial dedicated test opening', NOW(), NOW()
			)
		`, prodID, initialStock)
		require.NoError(t, err)
	}

	cleanup := func() {
		_, _ = conn.Exec(ctx, `
			DROP TRIGGER IF EXISTS trg_immutable_stock_movements ON stock_movements;
			DELETE FROM stock_movements WHERE product_id = $1;
			CREATE TRIGGER trg_immutable_stock_movements BEFORE UPDATE OR DELETE ON stock_movements FOR EACH ROW EXECUTE FUNCTION prevent_stock_movements_mutation();
			DELETE FROM inventory WHERE product_id = $1;
			DELETE FROM products WHERE id = $1;
		`, prodID)
	}

	return prodID, cleanup
}

func TestStockMovement_OpeningBalanceAndReconciliationInvariant(t *testing.T) {
	db, conn := setupTestConn(t)
	defer db.Close()
	defer conn.Release()

	ctx := context.Background()
	repo := inventory.NewRepository()
	service := inventory.NewService(repo)

	prodID, cleanup := createDedicatedTestProduct(t, conn, 50)
	defer cleanup()

	// 1. Invariant check on fresh opening balance:
	// opening + sum(subsequent delta) == on_hand
	bal, err := service.GetReconciledBalance(ctx, conn, prodID)
	require.NoError(t, err)
	assert.Equal(t, 50, bal.OnHandQuantity)
	assert.Equal(t, 50, bal.OpeningQuantity)
	assert.Equal(t, 0, bal.SubsequentDelta)
	assert.Equal(t, bal.OnHandQuantity, bal.OpeningQuantity+bal.SubsequentDelta, "Invariant violated: opening + sum(delta) != on_hand")

	// 2. Perform a transaction-owned IN movement (+15)
	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	docID := uuid.New()
	lineID := uuid.New()
	actorID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	reason := "Receipt batch A1"

	mov, err := service.PostMovementTx(ctx, tx, inventory.PostMovementParams{
		ProductID:            prodID,
		WarehouseLocation:    inventory.DefaultCoreLocation,
		QuantityDelta:        15,
		MovementType:         inventory.MovementTypeIn,
		SourceDocumentType:   inventory.SourceDocGoodsReceipt,
		SourceDocumentID:     &docID,
		SourceDocumentLineID: &lineID,
		ActorID:              &actorID,
		Reason:               &reason,
		OccurredAt:           time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, 15, mov.QuantityDelta)
	assert.Equal(t, inventory.MovementTypeIn, mov.MovementType)

	// Commit transaction
	require.NoError(t, tx.Commit(ctx))

	// Re-check reconciled balance
	balAfterIn, err := service.GetReconciledBalance(ctx, conn, prodID)
	require.NoError(t, err)
	assert.Equal(t, 65, balAfterIn.OnHandQuantity)
	assert.Equal(t, 50, balAfterIn.OpeningQuantity)
	assert.Equal(t, 15, balAfterIn.SubsequentDelta)
	assert.Equal(t, balAfterIn.OnHandQuantity, balAfterIn.OpeningQuantity+balAfterIn.SubsequentDelta)

	// 3. Perform a transaction-owned OUT movement (-5)
	tx2, err := conn.Begin(ctx)
	require.NoError(t, err)

	saleDocID := uuid.New()
	saleLineID := uuid.New()
	saleReason := "POS Sale test"

	movOut, err := service.PostMovementTx(ctx, tx2, inventory.PostMovementParams{
		ProductID:            prodID,
		WarehouseLocation:    inventory.DefaultCoreLocation,
		QuantityDelta:        -5,
		MovementType:         inventory.MovementTypeOut,
		SourceDocumentType:   inventory.SourceDocPOSSale,
		SourceDocumentID:     &saleDocID,
		SourceDocumentLineID: &saleLineID,
		ActorID:              &actorID,
		Reason:               &saleReason,
		OccurredAt:           time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, -5, movOut.QuantityDelta)

	require.NoError(t, tx2.Commit(ctx))

	// Re-check reconciled balance after OUT
	balAfterOut, err := service.GetReconciledBalance(ctx, conn, prodID)
	require.NoError(t, err)
	assert.Equal(t, 60, balAfterOut.OnHandQuantity)
	assert.Equal(t, 50, balAfterOut.OpeningQuantity)
	assert.Equal(t, 10, balAfterOut.SubsequentDelta)
	assert.Equal(t, balAfterOut.OnHandQuantity, balAfterOut.OpeningQuantity+balAfterOut.SubsequentDelta)
}

func TestStockMovement_TriggerBlocksUpdateAndDelete(t *testing.T) {
	db, conn := setupTestConn(t)
	defer db.Close()
	defer conn.Release()

	ctx := context.Background()
	prodID, cleanup := createDedicatedTestProduct(t, conn, 20)
	defer cleanup()

	repo := inventory.NewRepository()
	service := inventory.NewService(repo)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	docID := uuid.New()
	mov, err := service.PostMovementTx(ctx, tx, inventory.PostMovementParams{
		ProductID:          prodID,
		WarehouseLocation:  inventory.DefaultCoreLocation,
		QuantityDelta:      1,
		MovementType:       inventory.MovementTypeIn,
		SourceDocumentType: "TEST_TRIGGER_DOC",
		SourceDocumentID:   &docID,
		OccurredAt:         time.Now(),
	})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	// 1. Attempt UPDATE on stock_movements -> Must be blocked by trigger
	_, err = conn.Exec(ctx, `UPDATE stock_movements SET quantity_delta = 999 WHERE id = $1`, mov.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Stock movement violation: Posted stock movements are strictly append-only")

	// 2. Attempt DELETE on stock_movements -> Must be blocked by trigger
	_, err = conn.Exec(ctx, `DELETE FROM stock_movements WHERE id = $1`, mov.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Stock movement violation: Posted stock movements are strictly append-only")
}

func TestStockMovement_SourceUniqueness_PreventsDuplicatePosting(t *testing.T) {
	db, conn := setupTestConn(t)
	defer db.Close()
	defer conn.Release()

	ctx := context.Background()
	prodID, cleanup := createDedicatedTestProduct(t, conn, 30)
	defer cleanup()

	repo := inventory.NewRepository()
	service := inventory.NewService(repo)

	docID := uuid.New()
	lineID := uuid.New()

	// First post succeeds
	tx1, err := conn.Begin(ctx)
	require.NoError(t, err)

	_, err = service.PostMovementTx(ctx, tx1, inventory.PostMovementParams{
		ProductID:            prodID,
		WarehouseLocation:    inventory.DefaultCoreLocation,
		QuantityDelta:        5,
		MovementType:         inventory.MovementTypeIn,
		SourceDocumentType:   inventory.SourceDocGoodsReceipt,
		SourceDocumentID:     &docID,
		SourceDocumentLineID: &lineID,
		OccurredAt:           time.Now(),
	})
	require.NoError(t, err)
	require.NoError(t, tx1.Commit(ctx))

	// Second post with identical source document type, ID, and line ID must fail with duplicate error
	tx2, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer tx2.Rollback(ctx)

	_, err = service.PostMovementTx(ctx, tx2, inventory.PostMovementParams{
		ProductID:            prodID,
		WarehouseLocation:    inventory.DefaultCoreLocation,
		QuantityDelta:        5,
		MovementType:         inventory.MovementTypeIn,
		SourceDocumentType:   inventory.SourceDocGoodsReceipt,
		SourceDocumentID:     &docID,
		SourceDocumentLineID: &lineID,
		OccurredAt:           time.Now(),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, inventory.ErrDuplicateMovementSource)
}

func TestStockMovement_TransactionRollback(t *testing.T) {
	db, conn := setupTestConn(t)
	defer db.Close()
	defer conn.Release()

	ctx := context.Background()
	prodID, cleanup := createDedicatedTestProduct(t, conn, 40)
	defer cleanup()

	repo := inventory.NewRepository()
	service := inventory.NewService(repo)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	docID := uuid.New()
	mov, err := service.PostMovementTx(ctx, tx, inventory.PostMovementParams{
		ProductID:          prodID,
		WarehouseLocation:  inventory.DefaultCoreLocation,
		QuantityDelta:      10,
		MovementType:       inventory.MovementTypeIn,
		SourceDocumentType: "ROLLBACK_TEST_DOC",
		SourceDocumentID:   &docID,
		OccurredAt:         time.Now(),
	})
	require.NoError(t, err)
	require.NotNil(t, mov)

	// Explicit Rollback
	require.NoError(t, tx.Rollback(ctx))

	// Verify inventory unchanged
	var currentQty int
	err = conn.QueryRow(ctx, "SELECT stock_quantity FROM inventory WHERE product_id = $1", prodID).Scan(&currentQty)
	require.NoError(t, err)
	assert.Equal(t, 40, currentQty)

	// Verify movement record does NOT exist
	var count int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM stock_movements WHERE id = $1", mov.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestStockMovement_RerunCutoverDoesNotDuplicateOpeningOrCreateZeroDeltas(t *testing.T) {
	db, conn := setupTestConn(t)
	defer db.Close()
	defer conn.Release()

	ctx := context.Background()

	// 1. Create a zero-stock product to verify zero-stock does not create meaningless zero movements
	zeroProdID := uuid.New()
	skuZero := fmt.Sprintf("SKU-ZERO-%d", time.Now().UnixNano())
	_, err := conn.Exec(ctx, `
		INSERT INTO products (id, sku, name, unit_price, cost_price, is_active)
		VALUES ($1, $2, 'Zero Stock Product Test', 10000, 8000, TRUE)
	`, zeroProdID, skuZero)
	require.NoError(t, err)

	_, err = conn.Exec(ctx, `
		INSERT INTO inventory (product_id, stock_quantity, warehouse_location)
		VALUES ($1, 0, 'MAIN_STORE')
	`, zeroProdID)
	require.NoError(t, err)

	defer func() {
		_, _ = conn.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, zeroProdID)
		_, _ = conn.Exec(ctx, `DELETE FROM products WHERE id = $1`, zeroProdID)
	}()

	// Count existing opening movements
	var countBefore int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM stock_movements WHERE movement_type = 'OPENING'").Scan(&countBefore)
	require.NoError(t, err)

	// 2. Rerun the exact cutover query from migration
	_, err = conn.Exec(ctx, `
		INSERT INTO stock_movements (
			product_id,
			warehouse_location,
			quantity_delta,
			movement_type,
			source_document_type,
			source_document_id,
			source_document_line_id,
			reason,
			occurred_at,
			created_at
		)
		SELECT
			i.product_id,
			COALESCE(i.warehouse_location, 'MAIN_STORE'),
			i.stock_quantity,
			'OPENING',
			'OPENING_BALANCE',
			i.product_id,
			NULL,
			'Initial opening stock cutover',
			i.updated_at,
			NOW()
		FROM inventory i
		WHERE i.stock_quantity > 0
		  AND NOT EXISTS (
			  SELECT 1 FROM stock_movements sm
			  WHERE sm.product_id = i.product_id
				AND sm.movement_type = 'OPENING'
		  )
	`)
	require.NoError(t, err)

	// Count after rerun
	var countAfter int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM stock_movements WHERE movement_type = 'OPENING'").Scan(&countAfter)
	require.NoError(t, err)
	assert.Equal(t, countBefore, countAfter, "Rerunning cutover must not duplicate opening balances")

	// Verify zero-stock product has no OPENING movement
	var zeroCount int
	err = conn.QueryRow(ctx, "SELECT COUNT(*) FROM stock_movements WHERE product_id = $1", zeroProdID).Scan(&zeroCount)
	require.NoError(t, err)
	assert.Equal(t, 0, zeroCount, "Zero-stock product must not produce meaningless zero movements")
}

func TestStockMovement_PropertyStyleSequenceReconciliation(t *testing.T) {
	db, conn := setupTestConn(t)
	defer db.Close()
	defer conn.Release()

	ctx := context.Background()
	prodID, cleanup := createDedicatedTestProduct(t, conn, 50)
	defer cleanup()

	repo := inventory.NewRepository()
	service := inventory.NewService(repo)

	// Apply a deterministic sequence of positive and negative adjustments
	deltas := []int{10, -3, -2, 25, -15, 7, -4}
	expectedBalance := 50

	for i, delta := range deltas {
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)

		mType := inventory.MovementTypeIn
		if delta < 0 {
			mType = inventory.MovementTypeOut
		}
		docID := uuid.New()
		lineID := uuid.New()
		reason := fmt.Sprintf("Sequence step %d", i)

		_, err = service.PostMovementTx(ctx, tx, inventory.PostMovementParams{
			ProductID:            prodID,
			WarehouseLocation:    inventory.DefaultCoreLocation,
			QuantityDelta:        delta,
			MovementType:         mType,
			SourceDocumentType:   "TEST_SEQUENCE",
			SourceDocumentID:     &docID,
			SourceDocumentLineID: &lineID,
			Reason:               &reason,
			OccurredAt:           time.Now(),
		})
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))

		expectedBalance += delta

		// Assert invariant holds at EVERY step
		bal, err := service.GetReconciledBalance(ctx, conn, prodID)
		require.NoError(t, err)
		assert.Equal(t, expectedBalance, bal.OnHandQuantity)
		assert.Equal(t, expectedBalance, bal.OpeningQuantity+bal.SubsequentDelta, "Sequence invariant failed at step %d", i)
	}
}
