package decimal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimalIsInteger(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    Decimal
		expected bool
	}{
		{name: "positive integer", value: NewFromInt(42), expected: true},
		{name: "negative integer", value: MustFromString("-42.000"), expected: true},
		{name: "zero", value: Zero, expected: true},
		{name: "positive fraction", value: MustFromString("42.1"), expected: false},
		{name: "negative fraction", value: MustFromString("-0.1"), expected: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.value.IsInteger(),
				"IsInteger should identify values without a fractional component")
		})
	}
}

func TestDecimalTruncate(t *testing.T) {
	t.Parallel()
	beyondU128 := "5" + strings.Repeat("0", 40) + "7"
	for _, tc := range []struct {
		name, value, expected string
		precision             int32
	}{
		{name: "fractional digits", value: "1.239", precision: 2, expected: "1.23"},
		{name: "negative fractional digits", value: "-1.239", precision: 2, expected: "-1.23"},
		{name: "negative fraction to zero", value: "-0.5", precision: 0, expected: "0"},
		{name: "unused precision", value: "1.239", precision: 5, expected: "1.239"},
		{name: "nineteenth fractional digit", value: "0.1234567890123456789", precision: 18, expected: "0.123456789012345678"},
		{name: "integer digits", value: "5432", precision: -2, expected: "5400"},
		{name: "negative integer digits", value: "-5432.987", precision: -2, expected: "-5400"},
		{name: "every integer digit", value: "-5432", precision: -4, expected: "0"},
		{name: "beyond integer digits", value: "5432", precision: -5, expected: "0"},
		{name: "fraction only", value: "1.239", precision: -1, expected: "0"},
		{name: "beyond u128", value: beyondU128, precision: -1, expected: "5" + strings.Repeat("0", 41)},
		{name: "u128 division overflow", value: "184467440737095516160", precision: -20, expected: "100000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).Truncate(tc.precision).String(),
				"Truncate should return the expected value")
		})
	}
}

func TestNewFromStringExponentLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value, expected string
	}{
		{name: "positive limit", value: "0e10000", expected: "0"},
		{name: "negative limit", value: "0e-10000", expected: "0"},
		{name: "fractional digits offsetting the exponent", value: "0.0e10001", expected: "0"},
		{name: "fractional digits at limit", value: "1." + strings.Repeat("0", 10000), expected: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := NewFromString(tc.value)
			require.NoError(t, err, "NewFromString must accept an exponent at the decode limit")
			assert.Equal(t, tc.expected, result.String(), "NewFromString should return the expected value")
		})
	}
	for _, tc := range []struct {
		name, value string
	}{
		{name: "beyond positive limit", value: "0e10001"},
		{name: "beyond negative limit", value: "0e-10001"},
		{name: "fractional digits beyond limit", value: "1." + strings.Repeat("0", 10001)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewFromString(tc.value)
			assert.Error(t, err, "NewFromString should reject an exponent beyond the decode limit")
			assert.Error(t, new(Decimal).UnmarshalJSON([]byte(`"`+tc.value+`"`)),
				"UnmarshalJSON should reject an exponent beyond the decode limit")
		})
	}
}
