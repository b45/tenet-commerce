package ledger

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateBalance(t *testing.T) {
	s := &Service{} // We don't need a repo for validateBalance

	tests := []struct {
		name    string
		lines   []EntryLine
		wantErr error
	}{
		{
			name: "Balanced Entry Succeeds",
			lines: []EntryLine{
				{ID: uuid.New(), DebitAmount: 100.50, CreditAmount: 0},
				{ID: uuid.New(), DebitAmount: 0, CreditAmount: 100.50},
			},
			wantErr: nil,
		},
		{
			name: "Unbalanced Entry Rejected",
			lines: []EntryLine{
				{ID: uuid.New(), DebitAmount: 100.50, CreditAmount: 0},
				{ID: uuid.New(), DebitAmount: 0, CreditAmount: 100.00},
			},
			wantErr: ErrUnbalancedEntry,
		},
		{
			name: "Zero Amount Entry Rejected",
			lines: []EntryLine{
				{ID: uuid.New(), DebitAmount: 0, CreditAmount: 0},
				{ID: uuid.New(), DebitAmount: 0, CreditAmount: 0},
			},
			wantErr: ErrZeroAmountEntry,
		},
		{
			name: "Single Line Entry Rejected",
			lines: []EntryLine{
				{ID: uuid.New(), DebitAmount: 100, CreditAmount: 0},
			},
			wantErr: ErrInsufficientLines,
		},
		{
			name: "Complex Balanced Entry Succeeds",
			lines: []EntryLine{
				{ID: uuid.New(), DebitAmount: 150.00, CreditAmount: 0}, // Cash
				{ID: uuid.New(), DebitAmount: 0, CreditAmount: 150.00}, // Revenue
				{ID: uuid.New(), DebitAmount: 90.00, CreditAmount: 0},  // COGS
				{ID: uuid.New(), DebitAmount: 0, CreditAmount: 90.00},  // Inventory
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validateBalance(tt.lines)
			if err != tt.wantErr {
				t.Errorf("validateBalance() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
