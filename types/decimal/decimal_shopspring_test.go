//go:build !udecimal_on

package decimal

import (
	"fmt"
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
	t.Run("decode exponent limit", testNewFromStringExponentLimit)
	t.Run("Unicode minus sign", testNewFromStringUnicodeMinus)
}

func TestMustFromString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, shopspring.RequireFromString("1.25"), MustFromString("1.25"),
		"MustFromString should preserve shopspring parsing")
	assert.Panics(t, func() { MustFromString("invalid") }, "MustFromString should panic for invalid input")
}

func TestDecimalPow(t *testing.T) {
	t.Parallel()
	t.Run("zero base", testDecimalPowZeroBase)
}

func TestDecimalUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("bare null leaves the value unchanged", testDecimalUnmarshalJSONBareNull)
	t.Run("decoding failure leaves zero", func(t *testing.T) {
		t.Parallel()
		testDecodeFailureLeavesZero(t, unmarshalJSONDecoders)
	})
}

func TestDecimalUnmarshalText(t *testing.T) {
	t.Parallel()
	t.Run("quoted text", testDecimalUnmarshalTextQuoted)
	t.Run("decoding failure leaves zero", func(t *testing.T) {
		t.Parallel()
		testDecodeFailureLeavesZero(t, unmarshalTextDecoders)
	})
}

func TestDecimalScan(t *testing.T) {
	t.Parallel()
	t.Run("input types", testDecimalScanInputTypes)
	t.Run("decoding failure leaves zero", func(t *testing.T) {
		t.Parallel()
		testDecodeFailureLeavesZero(t, scanDecoders)
	})
}

// testDecimalFormatGoSyntax checks shopspring's %#v form, which the udecimal
// build replaces with a MustFromString call.
func testDecimalFormatGoSyntax(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value, expected string
	}{
		{name: "fraction", value: "3.14", expected: "decimal.New(314, -2)"},
		{name: "negative fraction", value: "-0.5", expected: "decimal.New(-5, -1)"},
		{name: "zero", value: "0", expected: "decimal.New(0, 0)"},
		{name: "positive exponent", value: "1e3", expected: "decimal.New(1, 3)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, fmt.Sprintf("%#v", MustFromString(tc.value)),
				"Go syntax formatting should print shopspring's New call")
		})
	}
}
