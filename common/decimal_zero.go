package common

import "github.com/shopspring/decimal"

// DecimalEqualZero returns whether the numbers represented by d equals zero.
func DecimalEqualZero(d decimal.Decimal) bool {
	return d.Equal(decimal.Zero)
}

// DecimalNotZero returns whether d is not zero
func DecimalNotZero(d decimal.Decimal) bool {
	return !DecimalEqualZero(d)
}

// DecimalGreaterThanZero (GT0) returns true when d is greater than zero.
func DecimalGreaterThanZero(d decimal.Decimal) bool {
	return d.GreaterThan(decimal.Zero)
}

// DecimalGreaterThanOrEqualZero (GTE0) returns true when d is greater than or equal to zero.
func DecimalGreaterThanOrEqualZero(d decimal.Decimal) bool {
	return d.GreaterThanOrEqual(decimal.Zero)
}

// DecimalLessThanZero returns true when d is less than zero.
func DecimalLessThanZero(d decimal.Decimal) bool {
	return d.LessThan(decimal.Zero)
}

// DecimalLessThanOrEqualZero returns true when d is less than or equal to zero.
func DecimalLessThanOrEqualZero(d decimal.Decimal) bool {
	return d.LessThanOrEqual(decimal.Zero)
}
