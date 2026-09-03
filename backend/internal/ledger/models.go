package ledger

import (
	"time"

	"github.com/google/uuid"
)

// Account Types
const (
	AccountTypeAsset     = "ASSET"
	AccountTypeLiability = "LIABILITY"
	AccountTypeEquity    = "EQUITY"
	AccountTypeRevenue   = "REVENUE"
	AccountTypeExpense   = "EXPENSE"
)

// Source Document Types
const (
	SourceDocPOSSale           = "POS_SALE"
	SourceDocPOSVoid           = "POS_VOID"
	SourceDocGoodsReceipt      = "GOODS_RECEIPT"
	SourceDocManualAdjustment  = "MANUAL_ADJUSTMENT"
	SourceDocZakatDisbursement = "ZAKAT_DISBURSEMENT"
	SourceDocReversal          = "REVERSAL"
)

// Journal Entry Statuses
const (
	StatusPosted   = "POSTED"
	StatusReversed = "REVERSED"
)

// Account represents a ledger account (COA)
type Account struct {
	ID              uuid.UUID `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	AccountType     string    `json:"account_type"`
	IsZakatEligible bool      `json:"is_zakat_eligible"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

// Entry represents a journal entry header
type Entry struct {
	ID                 uuid.UUID   `json:"id"`
	EntryNumber        string      `json:"entry_number"`
	EntryDate          time.Time   `json:"entry_date"`
	SourceDocumentType string      `json:"source_document_type"`
	SourceDocumentID   *uuid.UUID  `json:"source_document_id,omitempty"`
	Memo               string      `json:"memo"`
	Status             string      `json:"status"`
	ReversedByEntryID  *uuid.UUID  `json:"reversed_by_entry_id,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	TotalDebit         float64     `json:"total_debit"`
	TotalCredit        float64     `json:"total_credit"`
	Lines              []EntryLine `json:"lines,omitempty"`
}

// EntryLine represents a single debit or credit line in a journal entry
type EntryLine struct {
	ID            uuid.UUID `json:"id"`
	LedgerEntryID uuid.UUID `json:"ledger_entry_id"`
	AccountID     uuid.UUID `json:"account_id"`
	AccountCode   string    `json:"account_code,omitempty"` // For convenience in responses
	AccountName   string    `json:"account_name,omitempty"` // For convenience in responses
	DebitAmount   float64   `json:"debit_amount"`
	CreditAmount  float64   `json:"credit_amount"`
}

// ReverseEntryRequest represents payload for reversing an existing journal entry
type ReverseEntryRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=255"`
}

// CreateEntryRequest represents the payload for creating a manual journal entry
type CreateEntryRequest struct {
	SourceDocumentType string                   `json:"source_document_type" binding:"required"`
	SourceDocumentID   *uuid.UUID               `json:"source_document_id"`
	Memo               string                   `json:"memo" binding:"required"`
	Lines              []CreateEntryLineRequest `json:"lines" binding:"required,min=2,dive"`
}

// CreateEntryLineRequest represents a single line in CreateEntryRequest
type CreateEntryLineRequest struct {
	AccountID    uuid.UUID `json:"account_id" binding:"required"`
	DebitAmount  float64   `json:"debit_amount"`
	CreditAmount float64   `json:"credit_amount"`
}

// TrialBalanceRow represents a single account's balance in the Trial Balance report
type TrialBalanceRow struct {
	AccountCode string  `json:"account_code"`
	AccountName string  `json:"account_name"`
	AccountType string  `json:"account_type"`
	TotalDebit  float64 `json:"total_debit"`
	TotalCredit float64 `json:"total_credit"`
	Balance     float64 `json:"balance"` // Positive means normal balance, negative means contra
}

// TrialBalanceSummary represents the full Trial Balance report
type TrialBalanceSummary struct {
	AsOfDate     string            `json:"as_of_date"`
	Rows         []TrialBalanceRow `json:"rows"`
	TotalDebits  float64           `json:"total_debits"`
	TotalCredits float64           `json:"total_credits"`
	IsBalanced   bool              `json:"is_balanced"`
}
