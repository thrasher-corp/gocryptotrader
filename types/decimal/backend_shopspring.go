//go:build !udecimal_on

package decimal

import (
	"fmt"

	shopspring "github.com/shopspring/decimal" //nolint:depguard // Backend implementation for the GCT decimal façade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "shopspring/decimal"

const limitedPrecision = false

type backendDecimal = shopspring.Decimal

func mustParseBackend(value string) backendDecimal {
	return shopspring.RequireFromString(value)
}

func normalisePrecision(digits string, scale int64, _ bool, _ string) (normalised string, normalisedScale int64, err error) {
	if digits == "" {
		return "0", 0, nil
	}
	if scale >= maxStringLength {
		return "", 0, fmt.Errorf("%w: fractional value exceeds %d characters", ErrInvalidDecimal, maxStringLength)
	}
	return digits, scale, nil
}

func mulBackend(left, right backendDecimal) backendDecimal {
	return left.Mul(right)
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
	return value.Pow(exponent), nil
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
