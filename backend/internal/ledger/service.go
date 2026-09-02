package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUnbalancedEntry = errors.New("ledger entry is unbalanced: total debits must equal total credits")
	ErrZeroAmountEntry = errors.New("ledger entry has zero amount")
	ErrInsufficientLines = errors.New("ledger entry must have at least two lines")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetChartOfAccounts returns the active Chart of Accounts
func (s *Service) GetChartOfAccounts(ctx context.Context, conn *pgxpool.Conn) ([]Account, error) {
	return s.repo.GetAccounts(ctx, conn)
}

// GetJournalEntries returns a paginated list of journal entries
func (s *Service) GetJournalEntries(ctx context.Context, conn *pgxpool.Conn, limit, offset int) ([]Entry, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetEntries(ctx, conn, limit, offset)
}

// GetTrialBalance returns the Trial Balance and checks if it's balanced
func (s *Service) GetTrialBalance(ctx context.Context, conn *pgxpool.Conn, asOfDate string) (*TrialBalanceSummary, error) {
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	rows, err := s.repo.GetTrialBalance(ctx, conn, asOfDate)
	if err != nil {
		return nil, err
	}

	var totalDebit, totalCredit float64
	for _, row := range rows {
		totalDebit += row.TotalDebit
		totalCredit += row.TotalCredit
	}

	// Deal with floating point issues by rounding to 2 decimal places before comparison
	totalDebitRounded := float64(int(totalDebit*100)) / 100
	totalCreditRounded := float64(int(totalCredit*100)) / 100

	return &TrialBalanceSummary{
		AsOfDate:     asOfDate,
		Rows:         rows,
		TotalDebits:  totalDebit,
		TotalCredits: totalCredit,
		IsBalanced:   totalDebitRounded == totalCreditRounded,
	}, nil
}

// validateBalance checks if an entry satisfies the accounting invariants before hitting DB
func (s *Service) validateBalance(lines []EntryLine) error {
	if len(lines) < 2 {
		return ErrInsufficientLines
	}

	var sumDebit, sumCredit float64
	for _, line := range lines {
		sumDebit += line.DebitAmount
		sumCredit += line.CreditAmount
	}

	sumDebitRounded := float64(int(sumDebit*100)) / 100
	sumCreditRounded := float64(int(sumCredit*100)) / 100

	if sumDebitRounded == 0 && sumCreditRounded == 0 {
		return ErrZeroAmountEntry
	}

	if sumDebitRounded != sumCreditRounded {
		return ErrUnbalancedEntry
	}

	return nil
}

// CreateManualEntry creates a manual journal entry, ensuring it's balanced
func (s *Service) CreateManualEntry(ctx context.Context, conn *pgxpool.Conn, req *CreateEntryRequest) (*Entry, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	entry := &Entry{
		ID:                 uuid.New(),
		EntryNumber:        "JE-" + time.Now().Format("20060102150405"),
		EntryDate:          time.Now(),
		SourceDocumentType: req.SourceDocumentType,
		SourceDocumentID:   req.SourceDocumentID,
		Memo:               req.Memo,
	}

	var totalDebit, totalCredit float64
	for _, reqLine := range req.Lines {
		totalDebit += reqLine.DebitAmount
		totalCredit += reqLine.CreditAmount
		entry.Lines = append(entry.Lines, EntryLine{
			ID:            uuid.New(),
			LedgerEntryID: entry.ID,
			AccountID:     reqLine.AccountID,
			DebitAmount:   reqLine.DebitAmount,
			CreditAmount:  reqLine.CreditAmount,
		})
	}
	entry.TotalDebit = totalDebit
	entry.TotalCredit = totalCredit

	if err := s.validateBalance(entry.Lines); err != nil {
		return nil, err
	}

	if err := s.repo.CreateEntryWithLines(ctx, tx, entry); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return entry, nil
}

// PostPOSSaleJournal creates an automatic journal entry for a POS checkout.
// It is intended to be called within the same transaction as the checkout.
func (s *Service) PostPOSSaleJournal(ctx context.Context, tx pgx.Tx, txnID uuid.UUID, txnNumber string, totalAmount, totalCOGS float64, paymentMethod string) error {
	entry := &Entry{
		ID:                 uuid.New(),
		EntryNumber:        "JE-POS-" + time.Now().Format("20060102150405") + "-" + txnNumber[len(txnNumber)-6:],
		EntryDate:          time.Now(),
		SourceDocumentType: SourceDocPOSSale,
		SourceDocumentID:   &txnID,
		Memo:               "POS sale " + txnNumber,
	}

	// 1. Debit Cash/Bank
	var assetAccountCode string
	if paymentMethod == "CASH" {
		assetAccountCode = "1010" // Cash on Hand
	} else {
		assetAccountCode = "1020" // Bank Operating Account
	}
	
	assetAccount, err := s.repo.GetAccountByCode(ctx, tx, assetAccountCode)
	if err != nil {
		return fmt.Errorf("failed to get asset account: %w", err)
	}

	// 2. Credit Sales Revenue
	revenueAccount, err := s.repo.GetAccountByCode(ctx, tx, "4010")
	if err != nil {
		return fmt.Errorf("failed to get revenue account: %w", err)
	}

	// 3. Debit COGS
	cogsAccount, err := s.repo.GetAccountByCode(ctx, tx, "5010")
	if err != nil {
		return fmt.Errorf("failed to get cogs account: %w", err)
	}

	// 4. Credit Inventory
	inventoryAccount, err := s.repo.GetAccountByCode(ctx, tx, "1030")
	if err != nil {
		return fmt.Errorf("failed to get inventory account: %w", err)
	}

	entry.Lines = []EntryLine{
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: assetAccount.ID, DebitAmount: totalAmount, CreditAmount: 0},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: revenueAccount.ID, DebitAmount: 0, CreditAmount: totalAmount},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: cogsAccount.ID, DebitAmount: totalCOGS, CreditAmount: 0},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: inventoryAccount.ID, DebitAmount: 0, CreditAmount: totalCOGS},
	}

	if err := s.validateBalance(entry.Lines); err != nil {
		return fmt.Errorf("failed POS sale journal balance validation: %w", err)
	}

	return s.repo.CreateEntryWithLines(ctx, tx, entry)
}

// PostPOSVoidReversalJournal creates an automatic reversal journal entry for a voided/refunded POS sale.
// It reverses revenue, asset (cash/bank), COGS, and inventory within the active transaction.
func (s *Service) PostPOSVoidReversalJournal(ctx context.Context, tx pgx.Tx, txnID uuid.UUID, txnNumber string, totalAmount, totalCOGS float64, paymentMethod string, voidReason string) error {
	entry := &Entry{
		ID:                 uuid.New(),
		EntryNumber:        "JE-VOID-" + time.Now().Format("20060102150405") + "-" + txnNumber[len(txnNumber)-6:],
		EntryDate:          time.Now(),
		SourceDocumentType: SourceDocPOSVoid,
		SourceDocumentID:   &txnID,
		Memo:               fmt.Sprintf("Void POS sale %s: %s", txnNumber, voidReason),
	}

	// 1. Credit Cash/Bank (Refund asset)
	var assetAccountCode string
	if paymentMethod == "CASH" {
		assetAccountCode = "1010" // Cash on Hand
	} else {
		assetAccountCode = "1020" // Bank Operating Account (QRIS / Card)
	}

	assetAccount, err := s.repo.GetAccountByCode(ctx, tx, assetAccountCode)
	if err != nil {
		return fmt.Errorf("failed to get asset account for void: %w", err)
	}

	// 2. Debit Sales Revenue (Reverse Revenue)
	revenueAccount, err := s.repo.GetAccountByCode(ctx, tx, "4010")
	if err != nil {
		return fmt.Errorf("failed to get revenue account for void: %w", err)
	}

	// 3. Credit COGS (Reverse Expense)
	cogsAccount, err := s.repo.GetAccountByCode(ctx, tx, "5010")
	if err != nil {
		return fmt.Errorf("failed to get cogs account for void: %w", err)
	}

	// 4. Debit Inventory (Restock Asset value)
	inventoryAccount, err := s.repo.GetAccountByCode(ctx, tx, "1030")
	if err != nil {
		return fmt.Errorf("failed to get inventory account for void: %w", err)
	}

	entry.Lines = []EntryLine{
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: revenueAccount.ID, DebitAmount: totalAmount, CreditAmount: 0},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: assetAccount.ID, DebitAmount: 0, CreditAmount: totalAmount},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: inventoryAccount.ID, DebitAmount: totalCOGS, CreditAmount: 0},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: cogsAccount.ID, DebitAmount: 0, CreditAmount: totalCOGS},
	}

	if err := s.validateBalance(entry.Lines); err != nil {
		return fmt.Errorf("failed POS void reversal journal balance validation: %w", err)
	}

	return s.repo.CreateEntryWithLines(ctx, tx, entry)
}

// PostGoodsReceiptJournal creates an automatic journal entry for a Goods Receipt.
// It is intended to be called within the same transaction as the GR creation.
func (s *Service) PostGoodsReceiptJournal(ctx context.Context, tx pgx.Tx, grID uuid.UUID, grNumber string, inboundValue float64) error {
	entry := &Entry{
		ID:                 uuid.New(),
		EntryNumber:        "JE-GR-" + time.Now().Format("20060102150405") + "-" + grNumber[len(grNumber)-6:],
		EntryDate:          time.Now(),
		SourceDocumentType: SourceDocGoodsReceipt,
		SourceDocumentID:   &grID,
		Memo:               "Goods receipt " + grNumber,
	}

	// 1. Debit Inventory
	inventoryAccount, err := s.repo.GetAccountByCode(ctx, tx, "1030")
	if err != nil {
		return fmt.Errorf("failed to get inventory account: %w", err)
	}

	// 2. Credit Accounts Payable
	payableAccount, err := s.repo.GetAccountByCode(ctx, tx, "2010")
	if err != nil {
		return fmt.Errorf("failed to get payable account: %w", err)
	}

	entry.Lines = []EntryLine{
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: inventoryAccount.ID, DebitAmount: inboundValue, CreditAmount: 0},
		{ID: uuid.New(), LedgerEntryID: entry.ID, AccountID: payableAccount.ID, DebitAmount: 0, CreditAmount: inboundValue},
	}

	if err := s.validateBalance(entry.Lines); err != nil {
		return fmt.Errorf("failed GR journal balance validation: %w", err)
	}

	return s.repo.CreateEntryWithLines(ctx, tx, entry)
}
