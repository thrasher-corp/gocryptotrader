package decimal

import (
	"database/sql/driver"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImplementation(t *testing.T) {
	t.Parallel()
	assert.Contains(t, []string{"shopspring/decimal", "quagmt/udecimal"}, Implementation,
		"Implementation should identify a supported backend")
}

func TestNewFromInt(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-42", NewFromInt(-42).String(), "NewFromInt should preserve the value")
}

func TestNewFromInt32(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "42", NewFromInt32(42).String(), "NewFromInt32 should preserve the value")
}

func TestNewFromFloat(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.25", NewFromFloat(1.25).String(), "NewFromFloat should preserve the value")
	assert.Equal(t, "0.000305375625", NewFromFloat(0.00030537562500000003).String(),
		"NewFromFloat should truncate binary noise beyond the common precision")
	assert.Panics(t, func() { NewFromFloat(math.Inf(1)) }, "NewFromFloat should panic for infinity")
}

func TestNewFromString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, expected string
	}{
		{name: "integer", input: "42", expected: "42"},
		{name: "fraction", input: ".5", expected: "0.5"},
		{name: "trailing point", input: "5.", expected: "5"},
		{name: "positive exponent", input: "1.25e2", expected: "125"},
		{name: "negative exponent", input: "125e-2", expected: "1.25"},
		{name: "trailing zeroes", input: "1.2300", expected: "1.23"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := NewFromString(tc.input)
			require.NoError(t, err, "NewFromString must parse valid input")
			assert.Equal(t, tc.expected, result.String(), "NewFromString should return the expected value")
		})
	}

	_, err := NewFromString("0." + strings.Repeat("1", maxPrecision+1))
	assert.ErrorIs(t, err, ErrPrecisionOutOfRange, "NewFromString should reject excess precision")
	_, err = NewFromString("not-a-number")
	assert.ErrorIs(t, err, ErrInvalidDecimal, "NewFromString should reject invalid input")
}

func TestRequireFromString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.25", RequireFromString("1.25").String(), "RequireFromString should parse valid input")
	assert.Panics(t, func() { RequireFromString("invalid") }, "RequireFromString should panic for invalid input")
}

func TestDecimalAdd(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.75", RequireFromString("1.25").Add(RequireFromString("2.5")).String(),
		"Add should return the sum")
}

func TestDecimalSub(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-1.25", RequireFromString("1.25").Sub(RequireFromString("2.5")).String(),
		"Sub should return the difference")
}

func TestDecimalMul(t *testing.T) {
	t.Parallel()
	result := RequireFromString("0.1234567890123456789").Mul(RequireFromString("0.1"))
	assert.Equal(t, "0.0123456789012345678", result.String(), "Mul should truncate beyond the common precision")
}

func TestDecimalDiv(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "0.6666666666666667", NewFromInt(2).Div(NewFromInt(3)).String(),
		"Div should round to the common division precision")
	assert.Panics(t, func() { NewFromInt(1).Div(Zero) }, "Div should panic when dividing by zero")
}

func TestDecimalMod(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1", NewFromInt(7).Mod(NewFromInt(3)).String(), "Mod should return the remainder")
	assert.Panics(t, func() { NewFromInt(1).Mod(Zero) }, "Mod should panic when dividing by zero")
}

func TestDecimalPow(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "6.25", RequireFromString("2.5").Pow(NewFromInt(2)).String(), "Pow should apply an integer exponent")
	assert.True(t, Zero.Pow(NewFromInt(-1)).IsZero(), "Pow should preserve shopspring zero edge behaviour")
	assert.Panics(t, func() { NewFromInt(2).Pow(RequireFromString("1.5")) },
		"Pow should panic for a fractional exponent")
}

func TestDecimalAbs(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "2", NewFromInt(-2).Abs().String(), "Abs should return a positive value")
}

func TestDecimalNeg(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-2", NewFromInt(2).Neg().String(), "Neg should invert the sign")
}

func TestDecimalCmp(t *testing.T) {
	t.Parallel()
	assert.Negative(t, NewFromInt(1).Cmp(NewFromInt(2)), "Cmp should report a smaller value")
	assert.Zero(t, NewFromInt(2).Cmp(NewFromInt(2)), "Cmp should report equal values")
	assert.Positive(t, NewFromInt(3).Cmp(NewFromInt(2)), "Cmp should report a larger value")
}

func TestDecimalCompare(t *testing.T) {
	t.Parallel()
	assert.Negative(t, NewFromInt(1).Compare(NewFromInt(2)), "Compare should report a smaller value")
}

func TestDecimalEqual(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(2).Equal(RequireFromString("2.0")), "Equal should compare numeric values")
}

func TestDecimalGreaterThan(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(2).GreaterThan(NewFromInt(1)), "GreaterThan should report a larger value")
}

func TestDecimalGreaterThanOrEqual(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(2).GreaterThanOrEqual(NewFromInt(2)), "GreaterThanOrEqual should include equality")
}

func TestDecimalLessThan(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(1).LessThan(NewFromInt(2)), "LessThan should report a smaller value")
}

func TestDecimalLessThanOrEqual(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(2).LessThanOrEqual(NewFromInt(2)), "LessThanOrEqual should include equality")
}

func TestDecimalIsZero(t *testing.T) {
	t.Parallel()
	assert.True(t, Zero.IsZero(), "IsZero should identify zero")
}

func TestDecimalIsPositive(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(1).IsPositive(), "IsPositive should identify positive values")
	assert.False(t, Zero.IsPositive(), "IsPositive should reject zero")
}

func TestDecimalIsNegative(t *testing.T) {
	t.Parallel()
	assert.True(t, NewFromInt(-1).IsNegative(), "IsNegative should identify negative values")
	assert.False(t, Zero.IsNegative(), "IsNegative should reject zero")
}

func TestDecimalRound(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.24", RequireFromString("1.235").Round(2).String(), "Round should use half away from zero")
	assert.Equal(t, "550", NewFromInt(545).Round(-1).String(), "Round should support integer places")
	assert.Equal(t, "-550", NewFromInt(-545).Round(-1).String(), "Round should support negative values")
}

func TestDecimalTruncate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.23", RequireFromString("1.239").Truncate(2).String(), "Truncate should remove excess digits")
	assert.Equal(t, "1.239", RequireFromString("1.239").Truncate(-1).String(), "Truncate should ignore negative precision")
}

func TestDecimalFloor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-2", RequireFromString("-1.2").Floor().String(), "Floor should round toward negative infinity")
}

func TestDecimalCeil(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "2", RequireFromString("1.2").Ceil().String(), "Ceil should round toward positive infinity")
}

func TestDecimalString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.23", RequireFromString("1.2300").String(), "String should return the canonical value")
}

func TestDecimalStringFixed(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.24", RequireFromString("1.235").StringFixed(2), "StringFixed should round to the requested places")
	assert.Equal(t, "1.2000", RequireFromString("1.2").StringFixed(4), "StringFixed should pad trailing zeroes")
}

func TestDecimalFloat64(t *testing.T) {
	t.Parallel()
	value, exact := RequireFromString("0.5").Float64()
	assert.Equal(t, 0.5, value, "Float64 should return the nearest value")
	assert.True(t, exact, "Float64 should report exact binary conversions")
	_, exact = RequireFromString("0.1").Float64()
	assert.False(t, exact, "Float64 should report inexact binary conversions")
}

func TestDecimalInexactFloat64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.1, RequireFromString("0.1").InexactFloat64(), "InexactFloat64 should return the nearest value")
}

func TestDecimalIntPart(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(-12), RequireFromString("-12.9").IntPart(), "IntPart should truncate toward zero")
}

func TestDecimalMarshalJSON(t *testing.T) {
	t.Parallel()
	data, err := RequireFromString("1.25").MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, `"1.25"`, string(data), "MarshalJSON should quote the decimal")
}

func TestDecimalUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, input := range []string{`"1.25"`, `1.25`} {
		var result Decimal
		require.NoError(t, result.UnmarshalJSON([]byte(input)), "UnmarshalJSON must parse valid input")
		assert.Equal(t, "1.25", result.String(), "UnmarshalJSON should preserve the value")
	}
	result := NewFromInt(2)
	require.NoError(t, result.UnmarshalJSON([]byte("null")), "UnmarshalJSON must accept null")
	assert.Equal(t, "2", result.String(), "UnmarshalJSON should leave the value unchanged for null")
}

func TestDecimalMarshalText(t *testing.T) {
	t.Parallel()
	data, err := RequireFromString("1.25").MarshalText()
	require.NoError(t, err, "MarshalText must not error")
	assert.Equal(t, "1.25", string(data), "MarshalText should return the decimal text")
}

func TestDecimalUnmarshalText(t *testing.T) {
	t.Parallel()
	var result Decimal
	require.NoError(t, result.UnmarshalText([]byte("1.25")), "UnmarshalText must parse valid input")
	assert.Equal(t, "1.25", result.String(), "UnmarshalText should preserve the value")
}

func TestDecimalMarshalBinary(t *testing.T) {
	t.Parallel()
	data, err := RequireFromString("1.25").MarshalBinary()
	require.NoError(t, err, "MarshalBinary must not error")
	assert.Equal(t, "1.25", string(data), "MarshalBinary should return the canonical text")
}

func TestDecimalUnmarshalBinary(t *testing.T) {
	t.Parallel()
	var result Decimal
	require.NoError(t, result.UnmarshalBinary([]byte("1.25")), "UnmarshalBinary must parse valid input")
	assert.Equal(t, "1.25", result.String(), "UnmarshalBinary should preserve the value")
}

func TestDecimalScan(t *testing.T) {
	t.Parallel()
	for _, input := range []any{"1.25", []byte("1.25"), int64(1), 1.25} {
		var result Decimal
		require.NoError(t, result.Scan(input), "Scan must accept supported input")
		assert.False(t, result.IsZero(), "Scan should set a non-zero value")
	}
	var result Decimal
	assert.ErrorIs(t, result.Scan(struct{}{}), ErrInvalidDecimal, "Scan should reject unsupported input")
	require.NoError(t, result.Scan(nil), "Scan must accept nil")
	assert.True(t, result.IsZero(), "Scan should set nil to zero")
}

func TestDecimalValue(t *testing.T) {
	t.Parallel()
	value, err := RequireFromString("1.25").Value()
	require.NoError(t, err, "Value must not error")
	assert.Equal(t, driver.Value("1.25"), value, "Value should return the canonical string")
}

func TestNormalise(t *testing.T) {
	t.Parallel()
	result, err := normalise("-1.25e2", false)
	require.NoError(t, err, "normalise must parse valid scientific notation")
	assert.Equal(t, "-125", result, "normalise should return fixed notation")
	_, err = normalise("1e999999999999", false)
	assert.ErrorIs(t, err, ErrInvalidDecimal, "normalise should reject an out-of-range exponent")
	result, err = normalise("0.12345678901234567899", true)
	require.NoError(t, err, "normalise must truncate excess precision when requested")
	assert.Equal(t, "0.1234567890123456789", result, "normalise should truncate to the common precision")
}

func TestParseBackend(t *testing.T) {
	t.Parallel()
	result, err := parseBackend("1.25")
	require.NoError(t, err, "parseBackend must parse valid input")
	assert.Equal(t, "1.25", result.String(), "parseBackend should preserve the value")
}

func TestMulBackend(t *testing.T) {
	t.Parallel()
	left, err := parseBackend("2")
	require.NoError(t, err, "parseBackend must parse the left value")
	right, err := parseBackend("3")
	require.NoError(t, err, "parseBackend must parse the right value")
	assert.Equal(t, "6", mulBackend(left, right).String(), "mulBackend should multiply values")
}

func TestDivBackend(t *testing.T) {
	t.Parallel()
	left, err := parseBackend("2")
	require.NoError(t, err, "parseBackend must parse the left value")
	right, err := parseBackend("3")
	require.NoError(t, err, "parseBackend must parse the right value")
	result, err := divBackend(left, right)
	require.NoError(t, err, "divBackend must divide valid values")
	assert.Equal(t, "0.6666666666666667", result.String(), "divBackend should apply common rounding")
}

func TestModBackend(t *testing.T) {
	t.Parallel()
	left, err := parseBackend("7")
	require.NoError(t, err, "parseBackend must parse the left value")
	right, err := parseBackend("3")
	require.NoError(t, err, "parseBackend must parse the right value")
	result, err := modBackend(left, right)
	require.NoError(t, err, "modBackend must calculate a valid remainder")
	assert.Equal(t, "1", result.String(), "modBackend should return the remainder")
}

func TestPowBackend(t *testing.T) {
	t.Parallel()
	value, err := parseBackend("2")
	require.NoError(t, err, "parseBackend must parse the value")
	exponent, err := parseBackend("3")
	require.NoError(t, err, "parseBackend must parse the exponent")
	result, err := powBackend(value, exponent)
	require.NoError(t, err, "powBackend must apply an integer exponent")
	assert.Equal(t, "8", result.String(), "powBackend should return the power")
}

func TestIsPositiveBackend(t *testing.T) {
	t.Parallel()
	value, err := parseBackend("1")
	require.NoError(t, err, "parseBackend must parse the value")
	assert.True(t, isPositiveBackend(value), "isPositiveBackend should identify positive values")
}

func TestIsNegativeBackend(t *testing.T) {
	t.Parallel()
	value, err := parseBackend("-1")
	require.NoError(t, err, "parseBackend must parse the value")
	assert.True(t, isNegativeBackend(value), "isNegativeBackend should identify negative values")
}

func TestRoundBackend(t *testing.T) {
	t.Parallel()
	value, err := parseBackend("1.25")
	require.NoError(t, err, "parseBackend must parse the value")
	assert.Equal(t, "1.3", roundBackend(value, 1).String(), "roundBackend should round half away from zero")
}

func TestTruncateBackend(t *testing.T) {
	t.Parallel()
	value, err := parseBackend("1.29")
	require.NoError(t, err, "parseBackend must parse the value")
	assert.Equal(t, "1.2", truncateBackend(value, 1).String(), "truncateBackend should remove excess digits")
}
