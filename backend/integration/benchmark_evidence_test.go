package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http/httptest"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestBenchmarkEvidence measures an in-process HTTP handler with real database
// dependencies. Authentication is injected by the integration router; transport,
// TLS, JWT verification and the frontend are deliberately outside this workload.
func TestBenchmarkEvidence(t *testing.T) {
	db := newTestDatabase(t)
	rdb := newTestRedisClient(t)
	router := setupFullCommerceRouter(t, db, rdb)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Exec(ctx, `SET search_path TO tenant_al_barakah_mart`)
	require.NoError(t, err)
	var pgVersion string
	require.NoError(t, conn.QueryRow(ctx, `SHOW server_version`).Scan(&pgVersion))
	// Compare the other tenant's business tables before and after all workloads.
	otherSnapshot := func() string {
		var value string
		require.NoError(t, conn.QueryRow(ctx, `SELECT md5(concat(
   (SELECT coalesce(jsonb_agg(to_jsonb(i) ORDER BY product_id)::text,'[]') FROM tenant_darussalam_store.inventory i),
   (SELECT coalesce(jsonb_agg(to_jsonb(s) ORDER BY id)::text,'[]') FROM tenant_darussalam_store.stock_movements s),
   (SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY id)::text,'[]') FROM tenant_darussalam_store.transactions x),
   (SELECT coalesce(jsonb_agg(to_jsonb(l) ORDER BY id)::text,'[]') FROM tenant_darussalam_store.ledger_entries l)))`).Scan(&value))
		return value
	}
	before := otherSnapshot()
	createProduct := func(stock int) (string, string) {
		id, sku := uuid.NewString(), "BENCH-"+uuid.NewString()
		tx, e := conn.Begin(ctx)
		require.NoError(t, e)
		defer tx.Rollback(ctx)
		_, e = tx.Exec(ctx, `INSERT INTO products(id,sku,name,unit_price,cost_price) VALUES($1,$2,'Synthetic benchmark product',1000,500)`, id, sku)
		require.NoError(t, e)
		_, e = tx.Exec(ctx, `INSERT INTO inventory(product_id,stock_quantity) VALUES($1,$2)`, id, stock)
		require.NoError(t, e)
		_, e = tx.Exec(ctx, `INSERT INTO stock_movements(product_id,quantity_delta,movement_type,source_document_type,source_document_id,reason) VALUES($1,$2,'OPENING','OPENING_BALANCE',$1,'Isolated benchmark fixture')`, id, stock)
		require.NoError(t, e)
		require.NoError(t, tx.Commit(ctx))
		return id, sku
	}
	type sample struct {
		Milliseconds float64 `json:"ms"`
		Status       int     `json:"status"`
		Code         string  `json:"code,omitempty"`
	}
	request := func(method, path, body, key string) sample {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		start := time.Now()
		router.ServeHTTP(w, req)
		result := sample{Milliseconds: float64(time.Since(start).Nanoseconds()) / 1e6, Status: w.Code}
		var envelope struct {
			Error *struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(w.Body.Bytes(), &envelope) == nil && envelope.Error != nil {
			result.Code = envelope.Error.Code
		}
		return result
	}
	run := func(name, method, path, body string, n, workers int) {
		samples := make([]sample, n)
		jobs := make(chan int)
		var wg sync.WaitGroup
		start := time.Now()
		for j := 0; j < workers; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range jobs {
					samples[i] = request(method, path, body, uuid.NewString())
				}
			}()
		}
		for i := 0; i < n; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		duration := time.Since(start).Seconds()
		latencies := make([]float64, n)
		counts := map[int]int{}
		conflicts := 0
		unexpected := 0
		for i, s := range samples {
			latencies[i] = s.Milliseconds
			counts[s.Status]++
			if name == "final_unit" && s.Status == 409 && s.Code == "INSUFFICIENT_STOCK" {
				conflicts++
			} else if (method == "GET" && s.Status != 200) || (method == "POST" && s.Status != 201) {
				unexpected++
			}
		}
		sort.Float64s(latencies)
		quantile := func(q float64) float64 { return latencies[int(math.Ceil(q*float64(n)))-1] }
		report := map[string]any{"scenario": name, "requests": n, "workers": workers, "duration_seconds": duration, "requests_per_second": float64(n) / duration, "p50_ms": quantile(.5), "p95_ms": quantile(.95), "p99_ms": quantile(.99), "statuses": counts, "expected_conflicts": conflicts, "unexpected": unexpected, "samples": samples, "go": runtime.Version(), "gomaxprocs": runtime.GOMAXPROCS(0), "postgres": pgVersion, "pool_max": 10, "transport": "httptest; injected auth"}
		data, e := json.Marshal(report)
		require.NoError(t, e)
		t.Log("BENCHMARK_EVIDENCE " + string(data))
		require.Zero(t, unexpected)
		if name == "final_unit" {
			require.Equal(t, 1, counts[201])
			require.Equal(t, n-1, conflicts)
		}
	}
	for i := 0; i < 10; i++ {
		require.Equal(t, 200, request("GET", "/api/v1/pos/products", "", "").Status)
	}
	run("catalog", "GET", "/api/v1/pos/products", "", 100, 4)
	id, sku := createProduct(101)
	body := fmt.Sprintf(`{"items":[{"sku":%q,"quantity":1}],"payment_method":"CASH","cash_tendered":1000}`, sku)
	warmKey := uuid.NewString()
	require.Equal(t, 201, request("POST", "/api/v1/pos/checkout", body, warmKey).Status)
	run("checkout", "POST", "/api/v1/pos/checkout", body, 100, 4)
	// An exact replay must not add another effect after all stock is sold.
	require.Equal(t, 201, request("POST", "/api/v1/pos/checkout", body, warmKey).Status)
	lastID, lastSKU := createProduct(1)
	lastBody := fmt.Sprintf(`{"items":[{"sku":%q,"quantity":1}],"payment_method":"CASH","cash_tendered":1000}`, lastSKU)
	run("final_unit", "POST", "/api/v1/pos/checkout", lastBody, 16, 8)
	for _, product := range []string{id, lastID} {
		var stock, movement, sold int
		require.NoError(t, conn.QueryRow(ctx, `SELECT stock_quantity FROM inventory WHERE product_id=$1`, product).Scan(&stock))
		require.Zero(t, stock)
		require.NoError(t, conn.QueryRow(ctx, `SELECT coalesce(sum(quantity_delta),0) FROM stock_movements WHERE product_id=$1`, product).Scan(&movement))
		require.Equal(t, stock, movement)
		require.NoError(t, conn.QueryRow(ctx, `SELECT coalesce(sum(quantity),0) FROM transaction_items WHERE product_id=$1`, product).Scan(&sold))
		if product == id {
			require.Equal(t, 101, sold)
		} else {
			require.Equal(t, 1, sold)
		}
	}
	var invalid int
	require.NoError(t, conn.QueryRow(ctx, `SELECT count(*) FROM (SELECT e.id FROM ledger_entries e LEFT JOIN ledger_entry_lines l ON l.ledger_entry_id=e.id GROUP BY e.id HAVING count(l.id)<2 OR coalesce(sum(l.debit_amount),0)<=0 OR coalesce(sum(l.debit_amount),0)<>coalesce(sum(l.credit_amount),0)) broken`).Scan(&invalid))
	require.Zero(t, invalid)
	require.Equal(t, before, otherSnapshot())
	t.Log("BENCHMARK_INVARIANTS passed: stock, movement balance, exact replay effects, journals, other-tenant snapshot")
}
