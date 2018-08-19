package common

import (
	"github.com/shopspring/decimal"
)

const sqrtMaxIterations = 10000

// Float returns the decimal value as a float64
// float is a helper function for float64() and doesnot return bool exact
func Float(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// NotEqual returns true when d is not equal to d2
func NotEqual(d, d2 decimal.Decimal) bool {
	return !d.Equal(d2)
}

//NewFromInt returns a decimal with the value of int v
func NewFromInt(v int) decimal.Decimal {
	return decimal.New(int64(v), 0)
}

//NewFromInt32 returns a decimal with the value of int v
func NewFromInt32(v int) decimal.Decimal {
	return decimal.New(int64(v), 0)
}

//NewFromInt64 returns a decimal with the value of int v
func NewFromInt64(v int64) decimal.Decimal {
	return decimal.New(v, 0)
}

// Percentage calculates the percentage of the AmountItem to the provided AmountItem
func Percentage(d, d2 decimal.Decimal) (p decimal.Decimal) {
	if d2.IsZero() {
		return decimal.Zero
	}
	return d.Div(d2).Mul(Hundred)
}
