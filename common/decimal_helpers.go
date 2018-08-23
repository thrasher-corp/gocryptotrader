package common

import (
	"github.com/shopspring/decimal"
)

// DecimalToFloat returns the decimal value as a float64
// float is a helper function for float64() and doesnot return bool exact
func DecimalToFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// DecimalNotEqual returns true when d is not equal to d2
func DecimalNotEqual(d, d2 decimal.Decimal) bool {
	return !d.Equal(d2)
}

// DecimalFromInt returns a decimal with the value of int v
func DecimalFromInt(v int) decimal.Decimal {
	return decimal.New(int64(v), 0)
}

// DecimalFromInt32 returns a decimal with the value of int v
func DecimalFromInt32(v int) decimal.Decimal {
	return decimal.New(int64(v), 0)
}

// DecimalFromInt64 returns a decimal with the value of int v
func DecimalFromInt64(v int64) decimal.Decimal {
	return decimal.New(v, 0)
}

// DecimalPercentage calculates the percentage of the AmountItem to the provided AmountItem
func DecimalPercentage(d, d2 decimal.Decimal) (p decimal.Decimal) {
	if d2.IsZero() {
		return decimal.Zero
	}
	return d.Div(d2).Mul(Hundred)
}
