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

	// 5. Calculate final amounts
	taxAmount := 0.00
	discountAmount := req.DiscountAmount
	if discountAmount > subtotal {
		discountAmount = subtotal
	}
	totalAmount := math.Round((subtotal-discountAmount+taxAmount)*100) / 100

	// 5.1 Validate settlement before any inventory or ledger mutation.
	// A completed CASH sale must have a tender amount sufficient to cover the receipt total.
	cashTendered, changeAmount, err := validatePaymentSettlement(req.PaymentMethod, req.CashTendered, totalAmount)
	if err != nil {
		return nil, err
	}

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
		CustomerName:      req.CustomerName,
		Notes:             req.Notes,
		CashTendered:      cashTendered,
		ChangeAmount:      changeAmount,
		PaymentReference:  req.PaymentReference,
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
		CustomerName:      masterTxn.CustomerName,
		Notes:             masterTxn.Notes,
		CashTendered:      masterTxn.CashTendered,
		ChangeAmount:      masterTxn.ChangeAmount,
		PaymentReference:  masterTxn.PaymentReference,
		Items:             lineItems,
		SubtotalAmount:    masterTxn.SubtotalAmount,
		TaxAmount:         masterTxn.TaxAmount,
		DiscountAmount:    masterTxn.DiscountAmount,
		TotalAmount:       masterTxn.TotalAmount,
		CreatedAt:         masterTxn.CreatedAt,
	}, nil
}

func validatePaymentSettlement(paymentMethod string, cashTendered *float64, totalAmount float64) (float64, float64, error) {
	if paymentMethod != "CASH" {
		if cashTendered != nil {
			return 0, 0, ErrCashTenderedNotAllowed
		}
		return 0, 0, nil
	}

	if cashTendered == nil {
		return 0, 0, fmt.Errorf("%w: cash_tendered is required", ErrInsufficientCashTendered)
	}

	if *cashTendered < totalAmount {
		return 0, 0, fmt.Errorf("%w: paid %.2f, required %.2f", ErrInsufficientCashTendered, *cashTendered, totalAmount)
	}

	return *cashTendered, math.Round((*cashTendered-totalAmount)*100) / 100, nil
}

// VoidTransaction executes an atomic void/refund of a completed transaction.
// It acquires a row-level lock, restores stock quantities to inventory,
// and posts a balanced reversal journal to the Sharia ledger.
func (s *Service) VoidTransaction(
	ctx context.Context,
	conn *pgxpool.Conn,
	cashierID string,
	txnID string,
	req VoidRequest,
) (*VoidResponse, error) {
	reqLogger := logger.FromContext(ctx)
	tStart := time.Now()

	// 1. Begin PostgreSQL Transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin database transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Lock the transaction row
	txn, err := s.repo.GetTransactionForUpdate(ctx, tx, txnID)
	if err != nil {
		return nil, err
	}

	// 3. Domain invariant validations
	if txn.Status == "VOIDED" {
		return nil, ErrAlreadyVoided
	}
	if txn.Status != "COMPLETED" {
		return nil, fmt.Errorf("transaction with status '%s' cannot be voided", txn.Status)
	}

	// 4. Retrieve transaction line items
	items, err := s.repo.GetTransactionItems(ctx, tx, txn.ID)
	if err != nil {
		return nil, fmt.Errorf("failed retrieving transaction line items: %w", err)
	}

	// 5. Restock inventory and compute total COGS
	var totalCOGS float64
	for _, item := range items {
		if err := s.repo.IncrementStock(ctx, tx, item.ProductID, item.Quantity); err != nil {
			reqLogger.Error("Failed restocking inventory during void", "product_id", item.ProductID, "error", err)
			return nil, fmt.Errorf("failed restocking product %s: %w", item.SKU, err)
		}
		totalCOGS += math.Round(item.CostPrice*float64(item.Quantity)*100) / 100
	}

	// 6. Mark transaction VOIDED
	if err := s.repo.MarkTransactionVoided(ctx, tx, txn.ID, cashierID, req.Reason); err != nil {
		reqLogger.Error("Failed marking transaction as voided", "error", err)
		return nil, fmt.Errorf("failed updating transaction status: %w", err)
	}

	// 7. Post Sharia Ledger Reversal Journal Entry
	txnUUID, _ := uuid.Parse(txn.ID)
	if err := s.ledgerService.PostPOSVoidReversalJournal(ctx, tx, txnUUID, txn.TransactionNumber, txn.TotalAmount, totalCOGS, txn.PaymentMethod, req.Reason); err != nil {
		reqLogger.Error("Failed posting POS void reversal journal", "error", err)
		return nil, fmt.Errorf("failed posting ledger reversal: %w", err)
	}

	// 8. Commit atomically
	if err := tx.Commit(ctx); err != nil {
		reqLogger.Error("Failed committing POS void transaction", "error", err)
		return nil, fmt.Errorf("commit void failed: %w", err)
	}

	reqLogger.Info("POS Transaction Voided Successfully",
		slog.String("transaction_number", txn.TransactionNumber),
		slog.Float64("total_refunded", txn.TotalAmount),
		slog.Int("items_restocked", len(items)),
		slog.String("void_reason", req.Reason),
		slog.Float64("duration_ms", float64(time.Since(tStart).Microseconds())/1000.0),
	)

	return &VoidResponse{
		TransactionID:     txn.ID,
		TransactionNumber: txn.TransactionNumber,
		Status:            "VOIDED",
		VoidReason:        req.Reason,
		VoidedAt:          time.Now(),
		VoidedBy:          cashierID,
		ItemsRestocked:    len(items),
		TotalRefunded:     txn.TotalAmount,
	}, nil
}

// GetOrders queries the order history with filtering and pagination
func (s *Service) GetOrders(ctx context.Context, conn *pgxpool.Conn, filter OrderFilter) ([]OrderSummary, int, error) {
	return s.repo.GetOrders(ctx, conn, filter)
}

// GetOrderDetail retrieves a single transaction with all line items
func (s *Service) GetOrderDetail(ctx context.Context, conn *pgxpool.Conn, id string) (*OrderDetailResponse, error) {
	return s.repo.GetOrderByID(ctx, conn, id)
}

// GetDailySummary aggregates end-of-day/shift cashier metrics
func (s *Service) GetDailySummary(ctx context.Context, conn *pgxpool.Conn, date string, cashierID *string) (*DailySummaryResponse, error) {
	return s.repo.GetDailySummary(ctx, conn, date, cashierID)
}

// GetQRISConfig returns the active tenant's QRIS payload
func (s *Service) GetQRISConfig(ctx context.Context, conn *pgxpool.Conn) (*QRISConfig, error) {
	return s.repo.GetQRISConfig(ctx, conn)
}

// UpdateQRISConfig updates the active tenant's QRIS payload
func (s *Service) UpdateQRISConfig(ctx context.Context, conn *pgxpool.Conn, cfg QRISConfig) error {
	return s.repo.UpdateQRISConfig(ctx, conn, cfg)
}

// GetCategories returns all categories
func (s *Service) GetCategories(ctx context.Context, conn *pgxpool.Conn) ([]Category, error) {
	return s.repo.GetCategories(ctx, conn)
}

// GetCategoryByID returns a category by ID
func (s *Service) GetCategoryByID(ctx context.Context, conn *pgxpool.Conn, id string) (*Category, error) {
	return s.repo.GetCategoryByID(ctx, conn, id)
}

// CreateCategory creates a new category
func (s *Service) CreateCategory(ctx context.Context, conn *pgxpool.Conn, req CreateCategoryRequest) (*Category, error) {
	return s.repo.CreateCategory(ctx, conn, req)
}

// UpdateCategory updates an existing category
func (s *Service) UpdateCategory(ctx context.Context, conn *pgxpool.Conn, id string, req UpdateCategoryRequest) (*Category, error) {
	return s.repo.UpdateCategory(ctx, conn, id, req)
}

// DeleteCategory deletes a category
func (s *Service) DeleteCategory(ctx context.Context, conn *pgxpool.Conn, id string) error {
	return s.repo.DeleteCategory(ctx, conn, id)
}

// GetProduct retrieves a product by its UUID
func (s *Service) GetProduct(ctx context.Context, conn *pgxpool.Conn, id string) (*Product, error) {
	return s.repo.GetProductByID(ctx, conn, id)
}

// CreateProduct adds a new product to the catalog with initial inventory
func (s *Service) CreateProduct(ctx context.Context, conn *pgxpool.Conn, req CreateProductRequest) (*Product, error) {
	return s.repo.CreateProduct(ctx, conn, req)
}

// UpdateProduct updates product metadata
func (s *Service) UpdateProduct(ctx context.Context, conn *pgxpool.Conn, id string, req UpdateProductRequest) (*Product, error) {
	return s.repo.UpdateProduct(ctx, conn, id, req)
}

// DeleteProduct soft-deletes a product
func (s *Service) DeleteProduct(ctx context.Context, conn *pgxpool.Conn, id string) error {
	return s.repo.DeleteProduct(ctx, conn, id)
}

// AdjustStock executes an atomic inventory adjustment and posts a balanced Sharia ledger shrinkage journal if applicable
func (s *Service) AdjustStock(ctx context.Context, conn *pgxpool.Conn, userID string, req StockAdjustmentRequest) (*StockAdjustmentResponse, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed starting adjustment transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	product, err := s.repo.GetProductByID(ctx, conn, req.ProductID)
	if err != nil {
		return nil, err
	}

	prevQty, newQty, deltaQty, err := s.repo.AdjustInventoryStock(ctx, tx, req.ProductID, req.AdjustmentType, req.Quantity)
	if err != nil {
		return nil, err
	}

	adjID := uuid.New()
	var ledgerEntryNum *string
	var ledgerEntryIDStr *string

	if deltaQty != 0 && product.CostPrice > 0 {
		entry, err := s.ledgerService.PostInventoryAdjustmentJournal(ctx, tx, adjID, product.Name, deltaQty, product.CostPrice, req.Reason, req.Notes)
		if err != nil {
			return nil, fmt.Errorf("failed posting inventory adjustment journal: %w", err)
		}
		if entry != nil {
			ledgerEntryNum = &entry.EntryNumber
			idStr := entry.ID.String()
			ledgerEntryIDStr = &idStr
		}
	}

	if err := s.repo.RecordInventoryAdjustment(ctx, tx, adjID.String(), req.ProductID, req.AdjustmentType, deltaQty, prevQty, newQty, req.Reason, req.Notes, userID, ledgerEntryIDStr); err != nil {
		return nil, fmt.Errorf("failed recording adjustment audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed committing stock adjustment: %w", err)
	}

	reqLogger := logger.FromContext(ctx)
	reqLogger.Info("Inventory Stock Adjusted",
		slog.String("product_id", req.ProductID),
		slog.String("product_name", product.Name),
		slog.Int("previous_quantity", prevQty),
		slog.Int("new_quantity", newQty),
		slog.Int("quantity_delta", deltaQty),
		slog.String("reason", req.Reason),
	)

	return &StockAdjustmentResponse{
		AdjustmentID:      adjID.String(),
		ProductID:         req.ProductID,
		ProductName:       product.Name,
		PreviousQuantity:  prevQty,
		NewQuantity:       newQty,
		QuantityDelta:     deltaQty,
		Reason:            req.Reason,
		LedgerEntryNumber: ledgerEntryNum,
		AdjustedAt:        time.Now(),
	}, nil
}

// GetLowStock retrieves all products at or below their reorder threshold
func (s *Service) GetLowStock(ctx context.Context, conn *pgxpool.Conn) ([]Product, error) {
	return s.repo.GetLowStockProducts(ctx, conn)
}
