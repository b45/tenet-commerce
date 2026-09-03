package money_test

import (
	"encoding/json"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/b45/tenet-commerce/backend/pkg/money"
)

func TestMoney_Basics(t *testing.T) {
	m := money.IDR(150000)
	assert.Equal(t, int64(150000), m.Amount())
	assert.Equal(t, "IDR", m.Currency())
	assert.Equal(t, float64(150000), m.ToFloat())
	assert.True(t, m.IsPositive())
	assert.False(t, m.IsZero())
	assert.False(t, m.IsNegative())
	assert.Equal(t, "Rp 150.000", m.Format())

	zero := money.IDR(0)
	assert.True(t, zero.IsZero())
	assert.False(t, zero.IsPositive())

	neg := money.IDR(-25000)
	assert.True(t, neg.IsNegative())
	assert.Equal(t, "-Rp 25.000", neg.Format())
}

func TestMoney_FromFloat(t *testing.T) {
	tests := []struct {
		input    float64
		expected int64
	}{
		{100.0, 100},
		{100.4, 100},
		{100.5, 101},
		{100.6, 101},
		{-50.2, -50},
		{-50.6, -51},
	}

	for _, tc := range tests {
		m, err := money.FromFloat(tc.input, "IDR")
		require.NoError(t, err)
		assert.Equal(t, tc.expected, m.Amount())
	}

	_, err := money.FromFloat(math.NaN(), "IDR")
	assert.ErrorIs(t, err, money.ErrInvalidAmount)

	_, err = money.FromFloat(math.Inf(1), "IDR")
	assert.ErrorIs(t, err, money.ErrInvalidAmount)
}

func TestMoney_Arithmetic(t *testing.T) {
	m1 := money.IDR(50000)
	m2 := money.IDR(30000)

	sum, err := m1.Add(m2)
	require.NoError(t, err)
	assert.Equal(t, int64(80000), sum.Amount())

	diff, err := m1.Sub(m2)
	require.NoError(t, err)
	assert.Equal(t, int64(20000), diff.Amount())

	prod, err := m1.Mul(3)
	require.NoError(t, err)
	assert.Equal(t, int64(150000), prod.Amount())

	// Multiply with 11% PPN tax factor
	taxed, err := m1.MulFloat(0.11)
	require.NoError(t, err)
	assert.Equal(t, int64(5500), taxed.Amount())

	// Currency mismatch check
	usd := money.New(50, "USD")
	_, err = m1.Add(usd)
	assert.ErrorIs(t, err, money.ErrCurrencyMismatch)

	_, err = m1.Sub(usd)
	assert.ErrorIs(t, err, money.ErrCurrencyMismatch)
}

func TestMoney_SplitConservesTotal(t *testing.T) {
	// Rp 100 split into 3 parts cannot divide evenly into integers (33.333...)
	// Split must distribute remainder pennies so 34 + 33 + 33 = 100!
	m := money.IDR(100)
	parts, err := m.Split(3)
	require.NoError(t, err)
	require.Len(t, parts, 3)

	assert.Equal(t, int64(34), parts[0].Amount())
	assert.Equal(t, int64(33), parts[1].Amount())
	assert.Equal(t, int64(33), parts[2].Amount())

	total := int64(0)
	for _, p := range parts {
		total += p.Amount()
	}
	assert.Equal(t, m.Amount(), total, "split parts sum must strictly equal the original amount")

	// Test negative split
	mNeg := money.IDR(-100)
	negParts, err := mNeg.Split(3)
	require.NoError(t, err)
	assert.Equal(t, int64(-34), negParts[0].Amount())
	assert.Equal(t, int64(-33), negParts[1].Amount())
	assert.Equal(t, int64(-33), negParts[2].Amount())

	_, err = m.Split(0)
	assert.ErrorIs(t, err, money.ErrDivisionByZero)
}

func TestMoney_JSONSerialization(t *testing.T) {
	m := money.IDR(250000)

	bytes, err := json.Marshal(m)
	require.NoError(t, err)
	assert.Equal(t, "250000", string(bytes))

	var deserialized money.Money
	require.NoError(t, json.Unmarshal(bytes, &deserialized))
	assert.True(t, m.Equal(deserialized))

	// Deserializing from string "250000" or "Rp 250.000"
	var fromStr money.Money
	require.NoError(t, json.Unmarshal([]byte(`"Rp 250.000"`), &fromStr))
	assert.Equal(t, int64(250000), fromStr.Amount())

	// Deserializing from float
	var fromFloat money.Money
	require.NoError(t, json.Unmarshal([]byte(`250000.45`), &fromFloat))
	assert.Equal(t, int64(250000), fromFloat.Amount())
}

func TestMoney_SQLScanAndValue(t *testing.T) {
	var m money.Money

	// Scan from int64
	require.NoError(t, m.Scan(int64(75000)))
	assert.Equal(t, int64(75000), m.Amount())

	// Scan from string from PostgreSQL numeric e.g. "75000.00"
	require.NoError(t, m.Scan("75000.00"))
	assert.Equal(t, int64(75000), m.Amount())

	// Value driver
	val, err := m.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(75000), val)
}

func TestMoney_RandomizedCartConservationProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Run 1,000 simulated cart operations with discounts, quantities, and splits
	for i := 0; i < 1000; i++ {
		numItems := rng.Intn(10) + 1
		cartTotal := money.IDR(0)

		for j := 0; j < numItems; j++ {
			price := int64(rng.Intn(500000) + 1000)
			qty := int64(rng.Intn(5) + 1)
			itemMoney := money.IDR(price)
			subtotal, err := itemMoney.Mul(qty)
			require.NoError(t, err)

			cartTotal, err = cartTotal.Add(subtotal)
			require.NoError(t, err)
		}

		// Apply random discount percentage (0% to 50%)
		discountPct := rng.Float64() * 0.5
		discount, err := cartTotal.MulFloat(discountPct)
		require.NoError(t, err)

		netTotal, err := cartTotal.Sub(discount)
		require.NoError(t, err)

		// Verification invariant: cartTotal == netTotal + discount
		reconstructed, err := netTotal.Add(discount)
		require.NoError(t, err)
		assert.Equal(t, cartTotal.Amount(), reconstructed.Amount(), "Net + Discount must strictly equal Gross Cart Total")

		// Split netTotal across 2-5 payment installments / tenders
		splits := rng.Intn(4) + 2
		parts, err := netTotal.Split(splits)
		require.NoError(t, err)

		splitSum := int64(0)
		for _, p := range parts {
			splitSum += p.Amount()
		}
		assert.Equal(t, netTotal.Amount(), splitSum, "Split sum must strictly equal netTotal")
	}
}
