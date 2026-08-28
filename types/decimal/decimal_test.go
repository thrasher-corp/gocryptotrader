//go:build udecimal_on

package decimal

import (
	"database/sql/driver"
	"math"
	"strconv"
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
	expected := "0.00030537562500000003"
	if limitedPrecision {
		expected = "0.000305375625"
	}
	assert.Equal(t, expected, NewFromFloat(0.00030537562500000003).String(),
		"NewFromFloat should apply the selected backend precision")
	assert.Equal(t, "0", NewFromFloat(math.SmallestNonzeroFloat64).String(),
		"NewFromFloat should truncate sub-precision finite values")
	assert.NotPanics(t, func() { NewFromFloat(math.MaxFloat64) },
		"NewFromFloat should accept every finite float64 value")
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

	highPrecision := "0." + strings.Repeat("1", 20)
	result, err := NewFromString(highPrecision)
	if limitedPrecision {
		assert.ErrorIs(t, err, ErrPrecisionOutOfRange, "NewFromString should reject excess precision")
	} else {
		require.NoError(t, err, "NewFromString must accept shopspring high precision")
		assert.Equal(t, highPrecision, result.String(), "NewFromString should preserve shopspring high precision")
	}
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
	expected := "0.01234567890123456789"
	if limitedPrecision {
		expected = "0.0123456789012345678"
	}
	assert.Equal(t, expected, result.String(), "Mul should apply the selected backend precision")
}

func TestDecimalDiv(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "0.6666666666666667", NewFromInt(2).Div(NewFromInt(3)).String(),
		"Div should round to the common division precision")
	assert.Equal(t, "0.0003548616039744", NewFromInt(1).Div(NewFromInt(2818)).String(),
		"Div should round directly to the common division precision")
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
	assert.Panics(t, func() { Zero.Pow(RequireFromString("1.5")) },
		"Pow should reject a fractional exponent for a zero base")
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
	for _, tc := range []struct {
		name     string
		value    string
		places   int32
		expected string
	}{
		{name: "rounded", value: "1.235", places: 2, expected: "1.24"},
		{name: "padded", value: "1.2", places: 4, expected: "1.2000"},
		{name: "integer", value: "1", places: 2, expected: "1.00"},
		{name: "zero places", value: "1.6", places: 0, expected: "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, RequireFromString(tc.value).StringFixed(tc.places),
				"StringFixed should return the expected representation")
		})
	}
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
	for _, input := range []string{`"\x"`, `invalid`} {
		assert.ErrorIs(t, result.UnmarshalJSON([]byte(input)), ErrInvalidDecimal,
			"UnmarshalJSON should reject invalid input")
	}
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
	assert.ErrorIs(t, result.UnmarshalText([]byte("invalid")), ErrInvalidDecimal,
		"UnmarshalText should reject invalid input")
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
	truncatedExpected := "0.12345678901234567899"
	if limitedPrecision {
		truncatedExpected = "0.1234567890123456789"
	}
	for _, tc := range []struct {
		name     string
		input    string
		truncate bool
		expected string
	}{
		{name: "scientific", input: "-1.25e2", expected: "-125"},
		{name: "expand exponent", input: "1e2", expected: "100"},
		{name: "zero", input: "000.000", expected: "0"},
		{name: "truncate", input: "0.12345678901234567899", truncate: true, expected: truncatedExpected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := normalise(tc.input, tc.truncate)
			require.NoError(t, err, "normalise must parse valid input")
			assert.Equal(t, tc.expected, result, "normalise should return the expected value")
		})
	}
	result, err := normalise("1e-20", true)
	require.NoError(t, err, "normalise must truncate excess backend precision")
	expected := "0"
	if !limitedPrecision {
		expected = "0." + strings.Repeat("0", 19) + "1"
	}
	assert.Equal(t, expected, result, "normalise should apply the selected backend precision")

	errorTests := []struct {
		name, input string
		expected    error
	}{
		{name: "empty", input: "", expected: ErrInvalidDecimal},
		{name: "out of range exponent", input: "1e999999999999", expected: ErrInvalidDecimal},
		{name: "missing mantissa", input: "+", expected: ErrInvalidDecimal},
		{name: "multiple decimal points", input: "1.2.3", expected: ErrInvalidDecimal},
		{name: "decimal point only", input: ".", expected: ErrInvalidDecimal},
		{name: "expanded value too long", input: "1e200", expected: ErrInvalidDecimal},
		{name: "negative expanded value too long", input: "-1e200", expected: ErrInvalidDecimal},
	}
	if limitedPrecision {
		errorTests = append(errorTests, struct {
			name, input string
			expected    error
		}{name: "precision", input: "0.12345678901234567899", expected: ErrPrecisionOutOfRange})
	} else {
		errorTests = append(errorTests, struct {
			name, input string
			expected    error
		}{name: "fractional value too long", input: "1e-200", expected: ErrInvalidDecimal})
	}
	for _, tc := range errorTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalise(tc.input, false)
			assert.ErrorIs(t, err, tc.expected, "normalise should return the expected error")
		})
	}
	for _, input := range []string{"1e199", "-1e199"} {
		_, err := normalise(input, false)
		require.NoError(t, err, "normalise must apply the digit cap independently of the sign")
	}
}

func TestNormalisePrecision(t *testing.T) {
	t.Parallel()
	result, scale, err := normalisePrecision("123", 2, false, "1.23")
	require.NoError(t, err, "normalisePrecision must accept supported precision")
	assert.Equal(t, "123", result, "normalisePrecision should preserve supported digits")
	assert.Equal(t, int64(2), scale, "normalisePrecision should preserve supported scale")

	result, scale, err = normalisePrecision("", 2, false, "0.00")
	require.NoError(t, err, "normalisePrecision must accept zero")
	assert.Equal(t, "0", result, "normalisePrecision should canonicalise zero")
	assert.Zero(t, scale, "normalisePrecision should remove zero scale")

	if !limitedPrecision {
		result, scale, err = normalisePrecision("123", 20, true, "0.00000000000000000123")
		require.NoError(t, err, "normalisePrecision must retain shopspring precision")
		assert.Equal(t, "123", result, "normalisePrecision should retain shopspring digits")
		assert.Equal(t, int64(20), scale, "normalisePrecision should retain shopspring scale")
		_, _, err = normalisePrecision("1", maxStringDigits, false, "1e-200")
		assert.ErrorIs(t, err, ErrInvalidDecimal, "normalisePrecision should reject unsafe fixed notation")
		return
	}

	_, _, err = normalisePrecision("123", 20, false, "0.00000000000000000123")
	assert.ErrorIs(t, err, ErrPrecisionOutOfRange, "normalisePrecision should reject excess udecimal precision")
	result, scale, err = normalisePrecision("1", 20, true, "0.00000000000000000001")
	require.NoError(t, err, "normalisePrecision must truncate a sub-precision value")
	assert.Equal(t, "0", result, "normalisePrecision should truncate a sub-precision value to zero")
	assert.Zero(t, scale, "normalisePrecision should remove truncated zero scale")
	result, scale, err = normalisePrecision("101", 20, true, "0.00000000000000000101")
	require.NoError(t, err, "normalisePrecision must truncate excess udecimal precision")
	assert.Equal(t, "1", result, "normalisePrecision should remove trailing zeroes after truncation")
	assert.Equal(t, int64(18), scale, "normalisePrecision should adjust scale after removing trailing zeroes")
}

func TestMustParseBackend(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "ordinary", value: "1.25"},
		{name: "maximum negative integer", value: "-" + strings.Repeat("1", maxStringDigits)},
		{name: "maximum negative decimal", value: "-" + strings.Repeat("1", 181) + "." + strings.Repeat("1", 19)},
		{name: "multi-chunk internal value", value: strings.Repeat("1", maxStringDigits+50)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.value, mustParseBackend(tc.value).String(), "backend parser must preserve the value")
		})
	}
	require.Panics(t, func() { mustParseBackend("invalid") }, "backend parser must panic for invalid input")
}

func TestNewFromFloatBackend(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    float64
		expected string
	}{
		{name: "ordinary", value: 1.25, expected: "1.25"},
		{name: "negative exponent", value: math.SmallestNonzeroFloat64, expected: "0"},
		{name: "positive exponent", value: math.MaxFloat64, expected: strconv.FormatFloat(math.MaxFloat64, 'f', -1, 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, newFromFloatBackend(tc.value).String(),
				"newFromFloatBackend should preserve the supported value")
		})
	}
	assert.Panics(t, func() { newFromFloatBackend(math.NaN()) },
		"newFromFloatBackend should panic for a non-finite value")
}

func TestPowerOfTenBackend(t *testing.T) {
	t.Parallel()
	for _, exponent := range []int{0, 1, maxStringDigits} {
		expected := "1" + strings.Repeat("0", exponent)
		assert.Equal(t, expected, powerOfTenBackend(exponent).String(),
			"powerOfTenBackend should return the requested power")
	}
}

func TestMulBackend(t *testing.T) {
	t.Parallel()
	left := mustParseBackend("2")
	right := mustParseBackend("3")
	assert.Equal(t, "6", mulBackend(left, right).String(), "mulBackend should multiply values")
}

func TestDivBackend(t *testing.T) {
	t.Parallel()
	left := mustParseBackend("2")
	right := mustParseBackend("3")
	result, err := divBackend(left, right)
	require.NoError(t, err, "divBackend must divide valid values")
	assert.Equal(t, "0.6666666666666667", result.String(), "divBackend should apply common rounding")
}

func TestModBackend(t *testing.T) {
	t.Parallel()
	left := mustParseBackend("7")
	right := mustParseBackend("3")
	result, err := modBackend(left, right)
	require.NoError(t, err, "modBackend must calculate a valid remainder")
	assert.Equal(t, "1", result.String(), "modBackend should return the remainder")
}

func TestPowBackend(t *testing.T) {
	t.Parallel()
	value := mustParseBackend("2")
	exponent := mustParseBackend("3")
	result, err := powBackend(value, exponent)
	require.NoError(t, err, "powBackend must apply an integer exponent")
	assert.Equal(t, "8", result.String(), "powBackend should return the power")
}

func TestIsPositiveBackend(t *testing.T) {
	t.Parallel()
	value := mustParseBackend("1")
	assert.True(t, isPositiveBackend(value), "isPositiveBackend should identify positive values")
}

func TestIsNegativeBackend(t *testing.T) {
	t.Parallel()
	value := mustParseBackend("-1")
	assert.True(t, isNegativeBackend(value), "isNegativeBackend should identify negative values")
}

func TestRoundBackend(t *testing.T) {
	t.Parallel()
	value := mustParseBackend("1.25")
	assert.Equal(t, "1.3", roundBackend(value, 1).String(), "roundBackend should round half away from zero")
	if limitedPrecision {
		assert.Equal(t, value, roundBackend(value, 19),
			"roundBackend should leave values unchanged beyond maximum precision")
		assert.Panics(t, func() { roundBackend(value, -maxStringDigits) },
			"roundBackend should panic when negative precision exceeds the supported range")
	}
}

func TestTruncateBackend(t *testing.T) {
	t.Parallel()
	value := mustParseBackend("1.29")
	assert.Equal(t, "1.2", truncateBackend(value, 1).String(), "truncateBackend should remove excess digits")
	if limitedPrecision {
		assert.Equal(t, value, truncateBackend(value, 19),
			"truncateBackend should leave values unchanged beyond maximum precision")
	}
}
