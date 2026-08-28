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
	shopspringValue := shopspring.NewFromInt(42)
	facadeValue := shopspringValue
	roundTrip := facadeValue
	assert.Equal(t, shopspringValue, roundTrip, "Decimal should preserve shopspring type identity")
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

func TestNewFromFloat(t *testing.T) {
	t.Parallel()
	for _, value := range []float64{1.25, 1e-300, 1e200, math.SmallestNonzeroFloat64, math.MaxFloat64} {
		assert.Equal(t, shopspring.NewFromFloat(value), NewFromFloat(value),
			"NewFromFloat should delegate finite values to shopspring")
	}
}

func TestNewFromString(t *testing.T) {
	t.Parallel()
	result, err := NewFromString("1e-300")
	require.NoError(t, err, "NewFromString must accept shopspring input")
	assert.Equal(t, shopspring.RequireFromString("1e-300"), result, "NewFromString should delegate to shopspring")
	_, err = NewFromString("invalid")
	assert.Error(t, err, "NewFromString should return the shopspring parse error")
}

func TestRequireFromString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, shopspring.RequireFromString("1.25"), RequireFromString("1.25"),
		"RequireFromString should delegate to shopspring")
	assert.Panics(t, func() { RequireFromString("invalid") }, "RequireFromString should panic for invalid input")
}
