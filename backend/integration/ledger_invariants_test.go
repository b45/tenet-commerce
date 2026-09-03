package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/internal/ledger"
	"github.com/b45/tenet-commerce/backend/internal/tenant"
	pkgAuth "github.com/b45/tenet-commerce/backend/pkg/auth"
	"github.com/b45/tenet-commerce/backend/pkg/database"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupLedgerTestRouter(t *testing.T, db *database.PostgresDB) (*gin.Engine, *ledger.Service) {
	gin.SetMode(gin.TestMode)

	tenantRepo := tenant.NewRepository(db)
	ledgerRepo := ledger.NewRepository()
	ledgerService := ledger.NewService(ledgerRepo)
	ledgerHandler := ledger.NewHandler(ledgerService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "11111111-1111-1111-1111-111111111111")
		c.Set("tenant_slug", "al-barakah-mart")
		c.Set("jwt_claims", &pkgAuth.CustomClaims{
			Permissions: []string{"ledger:read", "ledger:write"},
		})
		c.Next()
	})
	router.Use(tenant.ContextMiddleware(db, tenantRepo))

	rdb := newTestRedisClient(t)
	ledgerHandler.RegisterRoutes(router.Group("/api/v1/ledger"), rdb)

	return router, ledgerService
}

func getTwoAccountIDs(t *testing.T, conn *pgxpool.Conn) (uuid.UUID, uuid.UUID) {
	ctx := context.Background()
	rows, err := conn.Query(ctx, "SELECT id FROM ledger_accounts ORDER BY code LIMIT 2")
	require.NoError(t, err)
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	require.Len(t, ids, 2)
	return ids[0], ids[1]
}

func TestLedgerInvariants_DatabaseTrigger_BlocksUnbalancedEntry(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
	require.NoError(t, err)

	acc1, acc2 := getTwoAccountIDs(t, conn)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	entryID := uuid.New()
	entryNumber := fmt.Sprintf("JE-TEST-UNBALANCED-%d", time.Now().UnixNano())
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, entry_number, entry_date, source_document_type, memo, status)
		VALUES ($1, $2, CURRENT_DATE, 'MANUAL_ADJUSTMENT', 'Direct DB Test Unbalanced', 'POSTED')
	`, entryID, entryNumber)
	require.NoError(t, err)

	// Insert unbalanced lines: Debit 100,000 vs Credit 90,000
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entry_lines (id, ledger_entry_id, account_id, debit_amount, credit_amount)
		VALUES ($1, $2, $3, 100000, 0), ($4, $2, $5, 0, 90000)
	`, uuid.New(), entryID, acc1, uuid.New(), acc2)
	require.NoError(t, err)

	// Commit must trigger trg_verify_ledger_balance and FAIL
	err = tx.Commit(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Sharia Ledger Invariant Violation: Total Debits (100000.00) must equal Total Credits (90000.00)")
}

func TestLedgerInvariants_DatabaseTrigger_BlocksSingleLineEntry(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
	require.NoError(t, err)

	acc1, _ := getTwoAccountIDs(t, conn)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	entryID := uuid.New()
	entryNumber := fmt.Sprintf("JE-TEST-SINGLE-%d", time.Now().UnixNano())
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, entry_number, entry_date, source_document_type, memo, status)
		VALUES ($1, $2, CURRENT_DATE, 'MANUAL_ADJUSTMENT', 'Direct DB Test Single Line', 'POSTED')
	`, entryID, entryNumber)
	require.NoError(t, err)

	// Insert only 1 line
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entry_lines (id, ledger_entry_id, account_id, debit_amount, credit_amount)
		VALUES ($1, $2, $3, 100000, 0)
	`, uuid.New(), entryID, acc1)
	require.NoError(t, err)

	// Commit must fail on line_count < 2
	err = tx.Commit(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least 2 lines")
}

func TestLedgerInvariants_DatabaseTrigger_BlocksMutationAndDeletion(t *testing.T) {
	db := newPoolSizeOneDatabase(t)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()

	_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
	require.NoError(t, err)

	acc1, acc2 := getTwoAccountIDs(t, conn)

	// Insert a valid balanced entry
	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	entryID := uuid.New()
	entryNumber := fmt.Sprintf("JE-TEST-IMMUTABLE-%d", time.Now().UnixNano())
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, entry_number, entry_date, source_document_type, memo, status)
		VALUES ($1, $2, CURRENT_DATE, 'MANUAL_ADJUSTMENT', 'Original Immutable Entry', 'POSTED')
	`, entryID, entryNumber)
	require.NoError(t, err)

	lineID1 := uuid.New()
	lineID2 := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger_entry_lines (id, ledger_entry_id, account_id, debit_amount, credit_amount)
		VALUES ($1, $2, $3, 50000, 0), ($4, $2, $5, 0, 50000)
	`, lineID1, entryID, acc1, lineID2, acc2)
	require.NoError(t, err)

	require.NoError(t, tx.Commit(ctx))

	// 1. Attempt DELETE on ledger_entries -> Must be blocked by trigger
	_, err = conn.Exec(ctx, "DELETE FROM ledger_entries WHERE id = $1", entryID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Posted ledger entries are immutable and cannot be deleted")

	// 2. Attempt UPDATE on memo -> Must be blocked by trigger
	_, err = conn.Exec(ctx, "UPDATE ledger_entries SET memo = 'tampered memo' WHERE id = $1", entryID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Posted ledger entries cannot be modified")

	// 3. Attempt DELETE on ledger_entry_lines -> Must be blocked by trigger
	_, err = conn.Exec(ctx, "DELETE FROM ledger_entry_lines WHERE id = $1", lineID1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Posted ledger entry lines are strictly immutable")

	// 4. Attempt UPDATE on ledger_entry_lines -> Must be blocked by trigger
	_, err = conn.Exec(ctx, "UPDATE ledger_entry_lines SET debit_amount = 99999 WHERE id = $1", lineID1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Posted ledger entry lines are strictly immutable")
}

func TestLedgerInvariants_ReversalEndpoint_ProducesBalancedAuditTrail(t *testing.T) {
	db := newPoolSizeOneDatabase(t)

	var acc1, acc2 uuid.UUID
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)
		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)
		acc1, acc2 = getTwoAccountIDs(t, conn)
		conn.Release()
	}

	router, _ := setupLedgerTestRouter(t, db)

	// Step 1: Create a valid manual entry via POST /entries
	createPayload := ledger.CreateEntryRequest{
		SourceDocumentType: ledger.SourceDocManualAdjustment,
		Memo:               "Test Office Supplies Purchase",
		Lines: []ledger.CreateEntryLineRequest{
			{AccountID: acc1, DebitAmount: 175000, CreditAmount: 0},
			{AccountID: acc2, DebitAmount: 0, CreditAmount: 175000},
		},
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/entries", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("idem-entry-%d", time.Now().UnixNano()))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var createResp struct {
		Success bool         `json:"success"`
		Data    ledger.Entry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	origEntry := createResp.Data
	assert.Equal(t, ledger.StatusPosted, origEntry.Status)
	assert.Nil(t, origEntry.ReversedByEntryID)

	// Step 2: Call Reversal Endpoint POST /entries/:id/reverse
	reversalPayload := ledger.ReverseEntryRequest{
		Reason: "Accounting error: wrongly categorized expense",
	}
	revBody, _ := json.Marshal(reversalPayload)

	revReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/ledger/entries/%s/reverse", origEntry.ID), bytes.NewReader(revBody))
	revReq.Header.Set("Content-Type", "application/json")
	revReq.Header.Set("Idempotency-Key", fmt.Sprintf("idem-rev-%d", time.Now().UnixNano()))
	revW := httptest.NewRecorder()
	router.ServeHTTP(revW, revReq)

	require.Equal(t, http.StatusCreated, revW.Code)
	var revResp struct {
		Success bool         `json:"success"`
		Data    ledger.Entry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(revW.Body.Bytes(), &revResp))
	reversalEntry := revResp.Data

	// Verify reversal entry properties
	assert.Equal(t, ledger.SourceDocReversal, reversalEntry.SourceDocumentType)
	assert.Equal(t, &origEntry.ID, reversalEntry.SourceDocumentID)
	assert.Equal(t, origEntry.TotalCredit, reversalEntry.TotalDebit)
	assert.Equal(t, origEntry.TotalDebit, reversalEntry.TotalCredit)
	assert.Contains(t, reversalEntry.Memo, "Accounting error: wrongly categorized expense")

	// Step 3: Verify the original entry status in DB is now REVERSED with reversed_by_entry_id set
	{
		ctx := context.Background()
		conn, err := db.Pool.Acquire(ctx)
		require.NoError(t, err)

		_, err = conn.Exec(ctx, "SET search_path TO tenant_al_barakah_mart, public")
		require.NoError(t, err)

		var origStatus string
		var reversedBy *uuid.UUID
		err = conn.QueryRow(ctx, "SELECT status, reversed_by_entry_id FROM ledger_entries WHERE id = $1", origEntry.ID).Scan(&origStatus, &reversedBy)
		require.NoError(t, err)
		assert.Equal(t, ledger.StatusReversed, origStatus)
		require.NotNil(t, reversedBy)
		assert.Equal(t, reversalEntry.ID, *reversedBy)
		conn.Release()
	}

	// Step 4: Verify that attempting to reverse again is rejected with HTTP 422
	revReq2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/ledger/entries/%s/reverse", origEntry.ID), bytes.NewReader(revBody))
	revReq2.Header.Set("Content-Type", "application/json")
	revReq2.Header.Set("Idempotency-Key", fmt.Sprintf("idem-rev2-%d", time.Now().UnixNano()))
	revW2 := httptest.NewRecorder()
	router.ServeHTTP(revW2, revReq2)

	assert.Equal(t, http.StatusUnprocessableEntity, revW2.Code)
	assert.Contains(t, revW2.Body.String(), "ENTRY_ALREADY_REVERSED")

	// Step 5: Verify Trial Balance is still completely balanced
	tbReq := httptest.NewRequest(http.MethodGet, "/api/v1/ledger/trial-balance", nil)
	tbW := httptest.NewRecorder()
	router.ServeHTTP(tbW, tbReq)

	require.Equal(t, http.StatusOK, tbW.Code)
	var tbResp struct {
		Success bool                        `json:"success"`
		Data    ledger.TrialBalanceSummary  `json:"data"`
	}
	require.NoError(t, json.Unmarshal(tbW.Body.Bytes(), &tbResp))
	assert.True(t, tbResp.Data.IsBalanced)
}
