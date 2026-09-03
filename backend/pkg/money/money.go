package money

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	CurrencyIDR = "IDR"
)

var (
	ErrCurrencyMismatch = errors.New("cannot perform arithmetic on different currencies")
	ErrDivisionByZero   = errors.New("cannot divide by zero or negative parts")
	ErrOverflow         = errors.New("monetary arithmetic integer overflow")
	ErrInvalidAmount    = errors.New("invalid monetary amount")
)

// Money represents an immutable exact monetary amount stored as an integer in minor units.
// For IDR, 1 minor unit = 1 Rupiah.
type Money struct {
	amount   int64
	currency string
}

// New creates a new Money value object with the given minor units and currency.
func New(amount int64, currency string) Money {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if c == "" {
		c = CurrencyIDR
	}
	return Money{
		amount:   amount,
		currency: c,
	}
}

// IDR is a convenience constructor for Indonesian Rupiah (1 unit = Rp 1).
func IDR(amount int64) Money {
	return New(amount, CurrencyIDR)
}

// FromFloat converts a float64 into an exact Money object by rounding to the nearest minor unit.
func FromFloat(val float64, currency string) (Money, error) {
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return Money{}, ErrInvalidAmount
	}
	if val > float64(math.MaxInt64) || val < float64(math.MinInt64) {
		return Money{}, ErrOverflow
	}

	rounded := int64(math.Round(val))
	return New(rounded, currency), nil
}

// Amount returns the integer minor unit amount.
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the 3-letter ISO currency code.
func (m Money) Currency() string {
	if m.currency == "" {
		return CurrencyIDR
	}
	return m.currency
}

// ToFloat returns the amount as float64 (for serialization/reporting compatibility).
func (m Money) ToFloat() float64 {
	return float64(m.amount)
}

// IsZero returns true if the amount is 0.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsPositive returns true if the amount is greater than 0.
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// IsNegative returns true if the amount is less than 0.
func (m Money) IsNegative() bool {
	return m.amount < 0
}

// Equal returns true if amounts and currencies match.
func (m Money) Equal(other Money) bool {
	return m.amount == other.amount && m.Currency() == other.Currency()
}

// GreaterThan returns true if m has greater amount than other.
func (m Money) GreaterThan(other Money) (bool, error) {
	if m.Currency() != other.Currency() {
		return false, ErrCurrencyMismatch
	}
	return m.amount > other.amount, nil
}

// LessThan returns true if m has smaller amount than other.
func (m Money) LessThan(other Money) (bool, error) {
	if m.Currency() != other.Currency() {
		return false, ErrCurrencyMismatch
	}
	return m.amount < other.amount, nil
}

// Add safely adds two Money objects with overflow checking.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency() != other.Currency() {
		return Money{}, ErrCurrencyMismatch
	}

	// Overflow detection
	if (other.amount > 0 && m.amount > math.MaxInt64-other.amount) ||
		(other.amount < 0 && m.amount < math.MinInt64-other.amount) {
		return Money{}, ErrOverflow
	}

	return New(m.amount+other.amount, m.Currency()), nil
}

// Sub safely subtracts two Money objects with overflow checking.
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency() != other.Currency() {
		return Money{}, ErrCurrencyMismatch
	}

	// Overflow detection
	if (other.amount < 0 && m.amount > math.MaxInt64+other.amount) ||
		(other.amount > 0 && m.amount < math.MinInt64+other.amount) {
		return Money{}, ErrOverflow
	}

	return New(m.amount-other.amount, m.Currency()), nil
}

// Mul multiplies the Money object by an integer scalar.
func (m Money) Mul(multiplier int64) (Money, error) {
	if m.amount == 0 || multiplier == 0 {
		return New(0, m.Currency()), nil
	}

	result := m.amount * multiplier
	if result/multiplier != m.amount {
		return Money{}, ErrOverflow
	}

	return New(result, m.Currency()), nil
}

// MulFloat multiplies the Money object by a floating-point factor with deterministic rounding.
func (m Money) MulFloat(factor float64) (Money, error) {
	if math.IsNaN(factor) || math.IsInf(factor, 0) {
		return Money{}, ErrInvalidAmount
	}

	val := float64(m.amount) * factor
	return FromFloat(val, m.Currency())
}

// Split divides the amount into n parts, distributing any remainder penny/rupiah deterministically
// such that the sum of the resulting parts strictly equals the original amount.
func (m Money) Split(parts int) ([]Money, error) {
	if parts <= 0 {
		return nil, ErrDivisionByZero
	}

	base := m.amount / int64(parts)
	remainder := m.amount % int64(parts)

	results := make([]Money, parts)
	for i := 0; i < parts; i++ {
		extra := int64(0)
		if m.amount >= 0 {
			if int64(i) < remainder {
				extra = 1
			}
		} else {
			if int64(i) < -remainder {
				extra = -1
			}
		}
		results[i] = New(base+extra, m.Currency())
	}

	return results, nil
}

// Format returns a formatted string representation e.g. "Rp 150.000"
func (m Money) Format() string {
	sign := ""
	val := m.amount
	if val < 0 {
		sign = "-"
		val = -val
	}

	strVal := strconv.FormatInt(val, 10)
	var formatted []byte
	n := len(strVal)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			formatted = append(formatted, '.')
		}
		formatted = append(formatted, strVal[i])
	}

	if m.Currency() == CurrencyIDR {
		return fmt.Sprintf("%sRp %s", sign, string(formatted))
	}
	return fmt.Sprintf("%s%s %s", sign, m.Currency(), string(formatted))
}

// String implements fmt.Stringer
func (m Money) String() string {
	return m.Format()
}

// MarshalJSON serializes Money to a numeric minor unit in JSON.
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.amount)
}

// UnmarshalJSON deserializes Money from JSON number or string.
func (m *Money) UnmarshalJSON(data []byte) error {
	var num int64
	if err := json.Unmarshal(data, &num); err == nil {
		*m = IDR(num)
		return nil
	}

	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		parsed, err := FromFloat(f, CurrencyIDR)
		if err != nil {
			return err
		}
		*m = parsed
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		clean := strings.TrimSpace(s)
		clean = strings.ReplaceAll(clean, "Rp", "")
		clean = strings.ReplaceAll(clean, "IDR", "")
		clean = strings.ReplaceAll(clean, ".", "")
		clean = strings.ReplaceAll(clean, ",", "")
		clean = strings.TrimSpace(clean)
		parsedInt, err := strconv.ParseInt(clean, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot parse money string %q: %w", s, err)
		}
		*m = IDR(parsedInt)
		return nil
	}

	return fmt.Errorf("invalid json data for Money: %s", string(data))
}

// Scan implements the database/sql.Scanner interface for pgx.
func (m *Money) Scan(value any) error {
	if value == nil {
		*m = IDR(0)
		return nil
	}

	switch v := value.(type) {
	case int64:
		*m = IDR(v)
		return nil
	case int32:
		*m = IDR(int64(v))
		return nil
	case int:
		*m = IDR(int64(v))
		return nil
	case float64:
		parsed, err := FromFloat(v, CurrencyIDR)
		if err != nil {
			return err
		}
		*m = parsed
		return nil
	case []byte:
		return m.scanString(string(v))
	case string:
		return m.scanString(v)
	default:
		return fmt.Errorf("cannot scan type %T into Money", value)
	}
}

func (m *Money) scanString(s string) error {
	s = strings.TrimSpace(s)
	// Could be numeric string with decimals from PostgreSQL NUMERIC e.g. "150000.00"
	if strings.Contains(s, ".") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		parsed, err := FromFloat(f, CurrencyIDR)
		if err != nil {
			return err
		}
		*m = parsed
		return nil
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*m = IDR(val)
	return nil
}

// Value implements the database/sql/driver.Valuer interface.
func (m Money) Value() (driver.Value, error) {
	return m.amount, nil
}
