//go:build !udecimal_on

package decimal

import shopspring "github.com/shopspring/decimal" //nolint:depguard // Backend implementation for the GCT decimal façade.

// Implementation identifies the selected decimal backend.
const Implementation = "shopspring/decimal"

type backendDecimal = shopspring.Decimal

func parseBackend(value string) (backendDecimal, error) {
	return shopspring.NewFromString(value)
}

func mulBackend(left, right backendDecimal) backendDecimal {
	return left.Mul(right).Truncate(maxPrecision)
}

func divBackend(left, right backendDecimal) (backendDecimal, error) {
	if right.IsZero() {
		return backendDecimal{}, ErrDivideByZero
	}
	return left.Div(right), nil
}

func modBackend(left, right backendDecimal) (backendDecimal, error) {
	if right.IsZero() {
		return backendDecimal{}, ErrDivideByZero
	}
	return left.Mod(right), nil
}

func powBackend(value, exponent backendDecimal) (backendDecimal, error) {
	if !exponent.IsInteger() {
		return backendDecimal{}, ErrInvalidDecimal
	}
	return value.Pow(exponent).Truncate(maxPrecision), nil
}

func isPositiveBackend(value backendDecimal) bool {
	return value.IsPositive()
}

func isNegativeBackend(value backendDecimal) bool {
	return value.IsNegative()
}

func roundBackend(value backendDecimal, places int32) backendDecimal {
	return value.Round(places)
}

func truncateBackend(value backendDecimal, precision int32) backendDecimal {
	return value.Truncate(precision)
}
