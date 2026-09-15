//go:build !udecimal_on

package decimal

import shopspring "github.com/shopspring/decimal" //nolint:depguard // Default backend implementation for the GCT decimal facade.

// Implementation identifies the selected decimal backend.
const Implementation = "shopspring/decimal"

// Zero is the zero-value Decimal.
var Zero = shopspring.Zero

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

// MustFromFloat returns the shortest decimal representation that round-trips
// to value. It panics for non-finite values.
func MustFromFloat(value float64) Decimal {
	return shopspring.NewFromFloat(value)
}

// NewFromString parses value using shopspring.NewFromString.
func NewFromString(value string) (Decimal, error) {
	return shopspring.NewFromString(value)
}

// MustFromString parses value and panics if it is invalid.
func MustFromString(value string) Decimal {
	return shopspring.RequireFromString(value)
}
