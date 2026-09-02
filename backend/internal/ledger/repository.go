package ledger

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("record not found")
)

type Repository struct{}

func NewRepository() *Repository {
	return &Repository{}
}

// GetAccounts fetches all active accounts in the Chart of Accounts
func (r *Repository) GetAccounts(ctx context.Context, conn *pgxpool.Conn) ([]Account, error) {
	query := `
		SELECT id, code, name, account_type, is_zakat_eligible, is_active, created_at
		FROM ledger_accounts
		WHERE is_active = TRUE
		ORDER BY code ASC
	`
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.AccountType, &a.IsZakatEligible, &a.IsActive, &a.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

// GetAccountByCode resolves an account code to an Account struct.
// It accepts a generic db interface to work with both conn and tx.
func (r *Repository) GetAccountByCode(ctx context.Context, db pgx.Tx, code string) (*Account, error) {
	query := `
		SELECT id, code, name, account_type, is_zakat_eligible, is_active, created_at
		FROM ledger_accounts
		WHERE code = $1 AND is_active = TRUE
	`
	a := &Account{}
	err := db.QueryRow(ctx, query, code).Scan(&a.ID, &a.Code, &a.Name, &a.AccountType, &a.IsZakatEligible, &a.IsActive, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return a, nil
}

// CreateEntryWithLines inserts an entry and its lines atomically within the provided transaction.
func (r *Repository) CreateEntryWithLines(ctx context.Context, tx pgx.Tx, entry *Entry) error {
	queryEntry := `
		INSERT INTO ledger_entries (id, entry_number, entry_date, source_document_type, source_document_id, memo)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	err := tx.QueryRow(ctx, queryEntry,
		entry.ID, entry.EntryNumber, entry.EntryDate, entry.SourceDocumentType, entry.SourceDocumentID, entry.Memo,
	).Scan(&entry.CreatedAt)
	if err != nil {
		return err
	}

	for i := range entry.Lines {
		line := &entry.Lines[i]
		queryLine := `
			INSERT INTO ledger_entry_lines (id, ledger_entry_id, account_id, debit_amount, credit_amount)
			VALUES ($1, $2, $3, $4, $5)
		`
		_, err := tx.Exec(ctx, queryLine,
			line.ID, line.LedgerEntryID, line.AccountID, line.DebitAmount, line.CreditAmount,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetEntries fetches journal entries with their lines
func (r *Repository) GetEntries(ctx context.Context, conn *pgxpool.Conn, limit, offset int) ([]Entry, error) {
	queryEntries := `
		SELECT id, entry_number, entry_date, source_document_type, source_document_id, memo, created_at
		FROM ledger_entries
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := conn.Query(ctx, queryEntries, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.EntryNumber, &e.EntryDate, &e.SourceDocumentType, &e.SourceDocumentID, &e.Memo, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	// Fetch lines for each entry (N+1 query is acceptable here for small limits, otherwise could use IN clause)
	for i := range entries {
		queryLines := `
			SELECT l.id, l.ledger_entry_id, l.account_id, a.code, a.name, l.debit_amount, l.credit_amount
			FROM ledger_entry_lines l
			JOIN ledger_accounts a ON l.account_id = a.id
			WHERE l.ledger_entry_id = $1
		`
		lineRows, err := conn.Query(ctx, queryLines, entries[i].ID)
		if err != nil {
			return nil, err
		}
		
		var lines []EntryLine
		var totalDebit, totalCredit float64
		for lineRows.Next() {
			var l EntryLine
			if err := lineRows.Scan(&l.ID, &l.LedgerEntryID, &l.AccountID, &l.AccountCode, &l.AccountName, &l.DebitAmount, &l.CreditAmount); err != nil {
				lineRows.Close()
				return nil, err
			}
			totalDebit += l.DebitAmount
			totalCredit += l.CreditAmount
			lines = append(lines, l)
		}
		lineRows.Close()
		entries[i].Lines = lines
		entries[i].TotalDebit = totalDebit
		entries[i].TotalCredit = totalCredit
	}

	return entries, nil
}

// GetTrialBalance computes the Trial Balance as of a specific date
func (r *Repository) GetTrialBalance(ctx context.Context, conn *pgxpool.Conn, asOfDate string) ([]TrialBalanceRow, error) {
	query := `
		WITH account_balances AS (
			SELECT 
				l.account_id,
				SUM(l.debit_amount) as total_debit,
				SUM(l.credit_amount) as total_credit
			FROM ledger_entry_lines l
			JOIN ledger_entries e ON l.ledger_entry_id = e.id
			WHERE e.entry_date <= $1
			GROUP BY l.account_id
		)
		SELECT 
			a.code, 
			a.name, 
			a.account_type, 
			COALESCE(ab.total_debit, 0) as total_debit,
			COALESCE(ab.total_credit, 0) as total_credit
		FROM ledger_accounts a
		LEFT JOIN account_balances ab ON a.id = ab.account_id
		WHERE a.is_active = TRUE
		ORDER BY a.code ASC
	`
	rows, err := conn.Query(ctx, query, asOfDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tbRows []TrialBalanceRow
	for rows.Next() {
		var row TrialBalanceRow
		if err := rows.Scan(&row.AccountCode, &row.AccountName, &row.AccountType, &row.TotalDebit, &row.TotalCredit); err != nil {
			return nil, err
		}

		// Calculate normal balance
		switch row.AccountType {
		case AccountTypeAsset, AccountTypeExpense:
			row.Balance = row.TotalDebit - row.TotalCredit
		case AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue:
			row.Balance = row.TotalCredit - row.TotalDebit
		}

		tbRows = append(tbRows, row)
	}

	return tbRows, nil
}
