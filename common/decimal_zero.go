package common

import "github.com/shopspring/decimal"

// EqualZero returns whether the numbers represented by d equals zero.
func EqualZero(d decimal.Decimal) bool {
	return d.Equal(decimal.Zero)
}

// NotZero returns whether d is not zero
func NotZero(d decimal.Decimal) bool {
	return !EqualZero(d)
}

// GreaterThanZero (GT0) returns true when d is greater than zero.
func GreaterThanZero(d decimal.Decimal) bool {
	return d.GreaterThan(decimal.Zero)
}

// GreaterThanOrEqualZero (GTE0) returns true when d is greater than or equal to zero.
func GreaterThanOrEqualZero(d decimal.Decimal) bool {
	return d.GreaterThanOrEqual(decimal.Zero)
}

// LessThanZero returns true when d is less than zero.
func LessThanZero(d decimal.Decimal) bool {
	return d.LessThan(decimal.Zero)
}

// LessThanOrEqualZero returns true when d is less than or equal to zero.
func LessThanOrEqualZero(d decimal.Decimal) bool {
	return d.LessThanOrEqual(decimal.Zero)
}
