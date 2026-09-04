//go:build !udecimal_on

package decimal

import (
	"math"
	"testing"

	shopspring "github.com/shopspring/decimal" //nolint:depguard // Confirms default-backend source compatibility.
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ *Decimal = (*shopspring.Decimal)(nil)

func TestShopspringCompatibility(t *testing.T) {
	t.Parallel()
	assert.Equal(t, shopspring.Zero, Zero, "Zero should preserve the shopspring value")
	assert.Equal(t, "shopspring/decimal", Implementation, "Implementation should identify shopspring")
}

func TestNewFromInt(t *testing.T) {
	t.Parallel()
	assert.Equal(t, shopspring.NewFromInt(-42), NewFromInt(-42), "NewFromInt should delegate to shopspring")
}

func TestNewFromInt32(t *testing.T) {
	t.Parallel()
	assert.Equal(t, shopspring.NewFromInt32(42), NewFromInt32(42), "NewFromInt32 should delegate to shopspring")
}

func TestMustFromFloat(t *testing.T) {
	t.Parallel()
	for _, value := range []float64{1.25, 1e-300, 1e200, math.SmallestNonzeroFloat64, math.MaxFloat64} {
		assert.Equal(t, shopspring.NewFromFloat(value), MustFromFloat(value),
			"MustFromFloat should preserve shopspring's finite conversion")
	}
	assert.Panics(t, func() { MustFromFloat(math.Inf(1)) }, "MustFromFloat should panic for infinity")
}

func TestNewFromString(t *testing.T) {
	t.Parallel()
	result, err := NewFromString("1e-300")
	require.NoError(t, err, "NewFromString must accept shopspring input")
	assert.Equal(t, shopspring.RequireFromString("1e-300"), result, "NewFromString should delegate to shopspring")
	_, err = NewFromString("invalid")
	assert.Error(t, err, "NewFromString should return the shopspring parse error")
}

func TestMustFromString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, shopspring.RequireFromString("1.25"), MustFromString("1.25"),
		"MustFromString should preserve shopspring parsing")
	assert.Panics(t, func() { MustFromString("invalid") }, "MustFromString should panic for invalid input")
}

func TestDecimalIsInteger(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    string
		expected bool
	}{
		{name: "positive integer", value: "42", expected: true},
		{name: "negative integer", value: "-42.000", expected: true},
		{name: "zero", value: "0", expected: true},
		{name: "positive fraction", value: "42.1", expected: false},
		{name: "negative fraction", value: "-0.1", expected: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).IsInteger(),
				"IsInteger should identify values without a fractional component")
		})
	}
}
