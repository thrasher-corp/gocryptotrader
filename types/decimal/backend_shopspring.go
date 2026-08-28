//go:build !udecimal_on

// Package decimal preserves shopspring/decimal as the default implementation
// while allowing an alternative implementation to be selected at build time.
package decimal

import (
	"errors"

	shopspring "github.com/shopspring/decimal" //nolint:depguard // Default backend implementation for the GCT decimal facade.
)

// Implementation identifies the selected decimal backend.
const Implementation = "shopspring/decimal"

var (
	// ErrInvalidDecimal is returned when input cannot be represented as a decimal.
	ErrInvalidDecimal = errors.New("invalid decimal")
	// ErrPrecisionOutOfRange is returned when input exceeds the selected backend's precision.
	ErrPrecisionOutOfRange = errors.New("decimal precision out of range")
	// ErrDivideByZero is returned when a division operation has a zero divisor.
	ErrDivideByZero = errors.New("decimal division by zero")

	// Zero is the zero-value Decimal.
	Zero = shopspring.Zero
)

// Decimal aliases shopspring.Decimal in the default build so existing library
// consumers retain source compatibility with exported decimal fields.
type Decimal = shopspring.Decimal

// NewFromInt returns a Decimal equal to value.
func NewFromInt(value int64) Decimal {
	return shopspring.NewFromInt(value)
}

// NewFromInt32 returns a Decimal equal to value.
func NewFromInt32(value int32) Decimal {
	return shopspring.NewFromInt32(value)
}

// NewFromFloat returns the shortest decimal representation that round-trips
// to value, matching shopspring.NewFromFloat.
func NewFromFloat(value float64) Decimal {
	return shopspring.NewFromFloat(value)
}

// NewFromString parses value using shopspring.NewFromString.
func NewFromString(value string) (Decimal, error) {
	return shopspring.NewFromString(value)
}

// RequireFromString parses value and panics if it is invalid.
func RequireFromString(value string) Decimal {
	return shopspring.RequireFromString(value)
}
