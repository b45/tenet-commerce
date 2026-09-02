package pos

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service handles business logic and orchestration for the POS domain
type Service struct {
	repo          *Repository
	ledgerService *ledger.Service
}

// NewService initializes a new POS service
func NewService(repo *Repository, ledgerService *ledger.Service) *Service {
	return &Service{repo: repo, ledgerService: ledgerService}
}

// GetCatalog retrieves all available products with live inventory quantities
func (s *Service) GetCatalog(ctx context.Context, conn *pgxpool.Conn) ([]Product, error) {
	return s.repo.GetProducts(ctx, conn)
}

// Checkout executes an atomic, ACID-compliant POS sale within a single database transaction.
// Row-level locks (SELECT ... FOR UPDATE) prevent race conditions and inventory overselling.
// It instruments APM-style sub-operation timings for Grafana Loki observability.
func (s *Service) Checkout(
	ctx context.Context,
	conn *pgxpool.Conn,
	cashierID string,
	idempotencyKey string,
	req CheckoutRequest,
) (*CheckoutResponse, error) {
	reqLogger := logger.FromContext(ctx)
	tStart := time.Now()

	// 1. Begin PostgreSQL Transaction on the current tenant schema
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Safe: no-op if transaction has already been committed

	// 2. Extract and aggregate SKUs
	skuMap := make(map[string]int)
	var skus []string
	for _, item := range req.Items {
		if _, exists := skuMap[item.SKU]; !exists {
			skus = append(skus, item.SKU)
		}
		skuMap[item.SKU] += item.Quantity
	}

	// 3. APM SPAN: Acquire row-level locks on products and inventory
	tLock := time.Now()
	productMap, err := s.repo.GetProductsBySKUsForUpdate(ctx, tx, skus)
	lockDuration := time.Since(tLock)
	if err != nil {
		reqLogger.Error("Failed to lock products during checkout", "error", err)
		return nil, fmt.Errorf("database lock failure: %w", err)
	}

	// 4. Validate stock availability and calculate subtotals
	var subtotal float64
	var totalCOGS float64
	var lineItems []TransactionItem

	for _, item := range req.Items {
		product, exists := productMap[item.SKU]
		if !exists || !product.IsActive {
			return nil, fmt.Errorf("%w: sku '%s'", ErrProductNotFound, item.SKU)
		}

		totalRequested := skuMap[item.SKU]
		if product.StockQuantity < totalRequested {
			return nil, fmt.Errorf("%w: '%s' (requested: %d, available: %d)",
				ErrInsufficientStock, product.Name, totalRequested, product.StockQuantity)
		}

		itemSubtotal := math.Round(product.UnitPrice*float64(item.Quantity)*100) / 100
		subtotal += itemSubtotal
		totalCOGS += math.Round(product.CostPrice*float64(item.Quantity)*100) / 100

		lineItems = append(lineItems, TransactionItem{
			ProductID: product.ID,
			SKU:       product.SKU,
			Name:      product.Name,
			Quantity:  item.Quantity,
			UnitPrice: product.UnitPrice,
			CostPrice: product.CostPrice,
			Subtotal:  itemSubtotal,
		})
	}

	// 5. Calculate final amounts (Option B: Tax is 0 for MVP)
	taxAmount := 0.00
	discountAmount := req.DiscountAmount
	if discountAmount > subtotal {
		discountAmount = subtotal
	}
	totalAmount := math.Round((subtotal-discountAmount+taxAmount)*100) / 100

	// 6. APM SPAN: Atomically decrement stock for each locked item
	tDec := time.Now()
	for sku, qty := range skuMap {
		product := productMap[sku]
		if err := s.repo.DecrementStock(ctx, tx, product.ID, qty); err != nil {
			reqLogger.Error("Failed to decrement inventory stock", "sku", sku, "error", err)
			return nil, fmt.Errorf("failed decrementing stock for %s: %w", sku, err)
		}
	}
	decDuration := time.Since(tDec)

	// 7. APM SPAN: Insert Master Transaction Record
	txnNumber := s.repo.GenerateTransactionNumber()
	masterTxn := &Transaction{
		TransactionNumber: txnNumber,
		IdempotencyKey:    idempotencyKey,
		CashierID:         cashierID,
		SubtotalAmount:    subtotal,
		TaxAmount:         taxAmount,
		DiscountAmount:    discountAmount,
		TotalAmount:       totalAmount,
		PaymentMethod:     req.PaymentMethod,
		Status:            "COMPLETED",
	}

	tTxn := time.Now()
	if err := s.repo.CreateTransaction(ctx, tx, masterTxn); err != nil {
		reqLogger.Error("Failed to insert transaction record", "error", err)
		return nil, fmt.Errorf("failed saving transaction: %w", err)
	}
	txnDuration := time.Since(tTxn)

	// 8. APM SPAN: Bulk Insert Transaction Line Items
	for i := range lineItems {
		lineItems[i].TransactionID = masterTxn.ID
	}
	tItems := time.Now()
	if err := s.repo.CreateTransactionItems(ctx, tx, lineItems); err != nil {
		reqLogger.Error("Failed to insert transaction items", "error", err)
		return nil, fmt.Errorf("failed saving line items: %w", err)
	}
	itemsDuration := time.Since(tItems)

	// 8.5. APM SPAN: Post automatic journal entry
	tLedger := time.Now()
	txnUUID, _ := uuid.Parse(masterTxn.ID)
	if err := s.ledgerService.PostPOSSaleJournal(ctx, tx, txnUUID, masterTxn.TransactionNumber, masterTxn.TotalAmount, totalCOGS, masterTxn.PaymentMethod); err != nil {
		reqLogger.Error("Failed to post POS sale journal entry", "error", err)
		return nil, fmt.Errorf("failed posting ledger journal: %w", err)
	}
	ledgerDuration := time.Since(tLedger)

	// 9. APM SPAN: Commit Transaction atomically
	tCommit := time.Now()
	if err := tx.Commit(ctx); err != nil {
		reqLogger.Error("Failed to commit POS checkout transaction", "error", err)
		return nil, fmt.Errorf("commit failed: %w", err)
	}
	commitDuration := time.Since(tCommit)
	totalDuration := time.Since(tStart)

	// 10. Emit APM-Grade Structured Log with Sub-Operation Latencies
	reqLogger.Info("POS Checkout Transaction Completed",
		slog.String("transaction_number", masterTxn.TransactionNumber),
		slog.Float64("total_amount", masterTxn.TotalAmount),
		slog.Int("items_count", len(lineItems)),
		slog.Float64("duration_total_ms", float64(totalDuration.Microseconds())/1000.0),
		slog.Float64("duration_lock_products_ms", float64(lockDuration.Microseconds())/1000.0),
		slog.Float64("duration_stock_decrement_ms", float64(decDuration.Microseconds())/1000.0),
		slog.Float64("duration_insert_txn_ms", float64(txnDuration.Microseconds())/1000.0),
		slog.Float64("duration_insert_items_ms", float64(itemsDuration.Microseconds())/1000.0),
		slog.Float64("duration_post_journal_ms", float64(ledgerDuration.Microseconds())/1000.0),
		slog.Float64("duration_commit_ms", float64(commitDuration.Microseconds())/1000.0),
	)

	// 11. Construct Receipt Response
	return &CheckoutResponse{
		TransactionID:     masterTxn.ID,
		TransactionNumber: masterTxn.TransactionNumber,
		IdempotencyKey:    masterTxn.IdempotencyKey,
		CashierID:         masterTxn.CashierID,
		PaymentMethod:     masterTxn.PaymentMethod,
		Status:            masterTxn.Status,
		Items:             lineItems,
		SubtotalAmount:    masterTxn.SubtotalAmount,
		TaxAmount:         masterTxn.TaxAmount,
		DiscountAmount:    masterTxn.DiscountAmount,
		TotalAmount:       masterTxn.TotalAmount,
		CreatedAt:         masterTxn.CreatedAt,
	}, nil
}
