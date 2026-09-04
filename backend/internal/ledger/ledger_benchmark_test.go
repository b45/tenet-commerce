package ledger

import (
	"testing"

	"github.com/google/uuid"
)

func BenchmarkValidateBalance_Standard(b *testing.B) {
	s := &Service{}
	lines := []EntryLine{
		{ID: uuid.New(), DebitAmount: 150000.00, CreditAmount: 0},
		{ID: uuid.New(), DebitAmount: 0, CreditAmount: 150000.00},
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := s.validateBalance(lines); err != nil {
			b.Fatalf("unexpected validation error: %v", err)
		}
	}
}

func BenchmarkValidateBalance_POSComplex(b *testing.B) {
	s := &Service{}
	lines := []EntryLine{
		{ID: uuid.New(), DebitAmount: 150000.00, CreditAmount: 0}, // Cash
		{ID: uuid.New(), DebitAmount: 0, CreditAmount: 150000.00}, // Sales Revenue
		{ID: uuid.New(), DebitAmount: 85000.00, CreditAmount: 0},  // COGS
		{ID: uuid.New(), DebitAmount: 0, CreditAmount: 85000.00},  // Merchandise Inventory
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := s.validateBalance(lines); err != nil {
			b.Fatalf("unexpected validation error: %v", err)
		}
	}
}

func BenchmarkValidateBalance_Parallel(b *testing.B) {
	s := &Service{}
	lines := []EntryLine{
		{ID: uuid.New(), DebitAmount: 250000.00, CreditAmount: 0},
		{ID: uuid.New(), DebitAmount: 0, CreditAmount: 250000.00},
		{ID: uuid.New(), DebitAmount: 140000.00, CreditAmount: 0},
		{ID: uuid.New(), DebitAmount: 0, CreditAmount: 140000.00},
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := s.validateBalance(lines); err != nil {
				b.Fatalf("unexpected validation error: %v", err)
			}
		}
	})
}
