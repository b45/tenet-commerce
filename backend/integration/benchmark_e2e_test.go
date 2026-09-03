package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/b45/tenet-commerce/backend/internal/pos"
)

// BenchmarkE2E_RealDatabase_POSCheckout benchmarks the full ACID POS checkout transaction
// across Redis 7 and PostgreSQL 16 testcontainers:
// - Redis Distributed Idempotency Key Lock (SETNX)
// - PostgreSQL Transaction BEGIN
// - PostgreSQL Row-level Lock (SELECT ... FOR UPDATE) on inventory
// - Atomic stock decrement
// - Insertion of pos_transactions master record
// - Insertion of pos_transaction_items lines
// - Double-entry ledger journal creation (Merchandise Inventory, Sales Revenue, COGS, Cash)
// - Database balance trigger execution (sum Debits == sum Credits)
// - PostgreSQL Transaction COMMIT (WAL flush)
// - Redis Idempotency Cache response storage
func BenchmarkE2E_RealDatabase_POSCheckout(b *testing.B) {
	// Setup testcontainers
	if testing.Short() {
		b.Skip("skipping benchmark in short mode")
	}

	db := newTestDatabase(&testing.T{})
	rdb := newTestRedisClient(&testing.T{})
	router := setupFullCommerceRouter(&testing.T{}, db, rdb)

	// Ensure large stock on SKU-BEEF-01 for continuous benchmarking
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		if err != nil {
			b.Fatalf("failed to acquire connection: %v", err)
		}
		_, _ = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		_, _ = conn.Exec(ctx, "UPDATE inventory SET stock_quantity = 1000000 WHERE product_id = (SELECT id FROM products WHERE sku = 'SKU-BEEF-01')")
		conn.Release()
	}

	var reqCounter int64
	cashTender := 500000.0
	payload := pos.CheckoutRequest{
		Items: []pos.CartItemRequest{
			{SKU: "SKU-BEEF-01", Quantity: 1},
		},
		PaymentMethod: "CASH",
		CashTendered:  &cashTender,
	}
	bodyBytes, _ := json.Marshal(payload)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := atomic.AddInt64(&reqCounter, 1)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/pos/checkout", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", fmt.Sprintf("bench-e2e-%d-%d", time.Now().UnixNano(), n))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				b.Errorf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
			}
		}
	})
}
