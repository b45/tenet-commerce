package pos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePaymentSettlement(t *testing.T) {
	total := 125000.0
	zero := 0.0
	insufficient := 124999.0
	equal := 125000.0
	tendered := 150000.0

	tests := []struct {
		name          string
		paymentMethod string
		cashTendered  *float64
		wantTendered  float64
		wantChange    float64
		wantErr       error
	}{
		{name: "cash requires tender", paymentMethod: "CASH", wantErr: ErrInsufficientCashTendered},
		{name: "cash rejects zero tender", paymentMethod: "CASH", cashTendered: &zero, wantErr: ErrInsufficientCashTendered},
		{name: "cash rejects insufficient tender", paymentMethod: "CASH", cashTendered: &insufficient, wantErr: ErrInsufficientCashTendered},
		{name: "cash accepts exact tender", paymentMethod: "CASH", cashTendered: &equal, wantTendered: equal, wantChange: 0},
		{name: "cash calculates change", paymentMethod: "CASH", cashTendered: &tendered, wantTendered: tendered, wantChange: 25000},
		{name: "non-cash rejects cash tender", paymentMethod: "QRIS", cashTendered: &tendered, wantErr: ErrCashTenderedNotAllowed},
		{name: "non-cash has no cash settlement", paymentMethod: "SIMULATED_CARD", wantTendered: 0, wantChange: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTendered, actualChange, err := validatePaymentSettlement(tt.paymentMethod, tt.cashTendered, total)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantTendered, actualTendered)
			require.Equal(t, tt.wantChange, actualChange)
		})
	}
}
