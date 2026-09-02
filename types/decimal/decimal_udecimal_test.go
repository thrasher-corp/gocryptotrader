//go:build udecimal_on

package decimal

import (
	"database/sql/driver"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/quagmt/udecimal" //nolint:depguard // Verifies errors from the selected backend.
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

func TestMustFromFloat(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.25", MustFromFloat(1.25).String(), "MustFromFloat should preserve the value")
	assert.Equal(t, "0.000305375625", MustFromFloat(0.00030537562500000003).String(),
		"MustFromFloat should apply udecimal precision")
	assert.Equal(t, "0", MustFromFloat(math.SmallestNonzeroFloat64).String(),
		"MustFromFloat should truncate sub-precision finite values")
	assert.NotPanics(t, func() { MustFromFloat(math.MaxFloat64) },
		"MustFromFloat should accept every finite float64 value")
	assert.Panics(t, func() { MustFromFloat(math.Inf(1)) }, "MustFromFloat should panic for infinity")
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
	_, err := NewFromString(highPrecision)
	assert.ErrorIs(t, err, errPrecisionOutOfRange, "NewFromString should reject excess precision")
	_, err = NewFromString("not-a-number")
	assert.ErrorIs(t, err, errInvalidDecimal, "NewFromString should reject invalid input")
	for _, value := range []string{
		strings.Repeat("1", maxStringDigits+1),
		"-" + strings.Repeat("9", maxStringDigits+1) + ".1234567890123456789",
	} {
		result, err := NewFromString(value)
		require.NoError(t, err, "NewFromString must parse extended plain decimal values")
		assert.Equal(t, value, result.String(), "NewFromString should preserve extended plain decimal values")
	}
}

func TestMustFromString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.25", MustFromString("1.25").String(), "MustFromString should parse valid input")
	assert.Panics(t, func() { MustFromString("invalid") }, "MustFromString should panic for invalid input")
}

func TestDecimalAdd(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3.75", MustFromString("1.25").Add(MustFromString("2.5")).String(),
		"Add should return the sum")
}

func TestDecimalSub(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-1.25", MustFromString("1.25").Sub(MustFromString("2.5")).String(),
		"Sub should return the difference")
}

func TestDecimalMul(t *testing.T) {
	t.Parallel()
	result := MustFromString("0.1234567890123456789").Mul(MustFromString("0.1"))
	assert.Equal(t, "0.0123456789012345678", result.String(), "Mul should apply udecimal precision")
}

func TestDecimalDiv(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, left, right, expected string
	}{
		{name: "native precision", left: "2", right: "3", expected: "0.6666666666666666666"},
		{name: "native truncation", left: "1", right: "2818", expected: "0.0003548616039744499"},
		{name: "overflow fallback", left: "340282366920938463463374607431768211455", right: "184467440737095516141553255926290448384", expected: "1.8446744073709551617"},
		{name: "negative overflow dividend", left: "-340282366920938463463374607431768211455", right: "184467440737095516141553255926290448384", expected: "-1.8446744073709551617"},
		{name: "negative overflow divisor", left: "340282366920938463463374607431768211455", right: "-184467440737095516141553255926290448384", expected: "-1.8446744073709551617"},
		{name: "negative overflow operands", left: "-340282366920938463463374607431768211455", right: "-184467440737095516141553255926290448384", expected: "1.8446744073709551617"},
		{name: "signed mixed scale", left: "-12.5", right: "0.2", expected: "-62.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.left).Div(MustFromString(tc.right)).String(),
				"Div should return the expected truncated quotient")
		})
	}
	assert.PanicsWithError(t, udecimal.ErrDivideByZero.Error(), func() { NewFromInt(1).Div(Zero) },
		"Div should expose udecimal's native divide-by-zero error")
}

func TestDecimalMod(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, left, right, expected string
	}{
		{name: "native remainder", left: "7", right: "3", expected: "1"},
		{name: "fractional remainder", left: "7.5", right: "2", expected: "1.5"},
		{name: "overflow fallback", left: "340282366920938463463374607431768211455", right: "18446744073709551614.1553255926290448384", expected: "15581492618384294731.1821351505477763072"},
		{name: "signed overflow fallback", left: "-340282366920938463463374607431768211455", right: "18446744073709551614.1553255926290448384", expected: "-15581492618384294731.1821351505477763072"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.left).Mod(MustFromString(tc.right)).String(),
				"Mod should return the expected remainder")
		})
	}
	assert.PanicsWithError(t, udecimal.ErrDivideByZero.Error(), func() { NewFromInt(1).Mod(Zero) },
		"Mod should expose udecimal's native divide-by-zero error")
}

func TestDecimalPow(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "6.25", MustFromString("2.5").Pow(NewFromInt(2)).String(), "Pow should apply an integer exponent")
	assert.True(t, Zero.Pow(NewFromInt(-1)).IsZero(), "Pow should preserve shopspring zero edge behaviour")
	assert.Panics(t, func() { NewFromInt(2).Pow(MustFromString("1.5")) },
		"Pow should panic for a fractional exponent")
	assert.Panics(t, func() { Zero.Pow(MustFromString("1.5")) },
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
	assert.True(t, NewFromInt(2).Equal(MustFromString("2.0")), "Equal should compare numeric values")
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
	largeHalf := "5" + strings.Repeat("0", maxStringDigits)
	largeRounded := "1" + strings.Repeat("0", maxStringDigits+1)
	for _, tc := range []struct {
		name, value, expected string
		places                int32
	}{
		{name: "fractional", value: "1.235", places: 2, expected: "1.24"},
		{name: "integer places", value: "545", places: -1, expected: "550"},
		{name: "negative value", value: "-545", places: -1, expected: "-550"},
		{name: "beyond parser cap", value: "1.25", places: -maxStringDigits - 1, expected: "0"},
		{name: "large half", value: largeHalf, places: -maxStringDigits - 1, expected: largeRounded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).Round(tc.places).String(),
				"Round should return the expected value")
		})
	}
}

func TestDecimalTruncate(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.23", MustFromString("1.239").Truncate(2).String(), "Truncate should remove excess digits")
	assert.Equal(t, "1.239", MustFromString("1.239").Truncate(-1).String(), "Truncate should ignore negative precision")
}

func TestDecimalFloor(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-2", MustFromString("-1.2").Floor().String(), "Floor should round toward negative infinity")
}

func TestDecimalCeil(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "2", MustFromString("1.2").Ceil().String(), "Ceil should round toward positive infinity")
}

func TestDecimalString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1.23", MustFromString("1.2300").String(), "String should return the canonical value")
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
		{name: "beyond parser cap", value: "1.6", places: -maxStringDigits - 1, expected: "0"},
		{name: "minimum int32 places", value: "1.6", places: math.MinInt32, expected: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).StringFixed(tc.places),
				"StringFixed should return the expected representation")
		})
	}
}

func TestDecimalFloat64(t *testing.T) {
	t.Parallel()
	value, exact := MustFromString("0.5").Float64()
	assert.Equal(t, 0.5, value, "Float64 should return the nearest value")
	assert.True(t, exact, "Float64 should report exact binary conversions")
	_, exact = MustFromString("0.1").Float64()
	assert.False(t, exact, "Float64 should report inexact binary conversions")
}

func TestDecimalInexactFloat64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.1, MustFromString("0.1").InexactFloat64(), "InexactFloat64 should return the nearest value")
}

func TestDecimalIntPart(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(-12), MustFromString("-12.9").IntPart(), "IntPart should truncate toward zero")
}

func TestDecimalMarshalJSON(t *testing.T) {
	t.Parallel()
	data, err := MustFromString("1.25").MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, `"1.25"`, string(data), "MarshalJSON should quote the decimal")
}

func TestDecimalUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input, expected string
	}{
		{input: `"1.25"`, expected: "1.25"},
		{input: `1.25`, expected: "1.25"},
		{input: `"1e-8"`, expected: "0.00000001"},
		{input: `1E5`, expected: "100000"},
		{input: `"-2.5e-4"`, expected: "-0.00025"},
		{input: `.5`, expected: "0.5"},
		{input: `1.`, expected: "1"},
	} {
		var result Decimal
		require.NoError(t, result.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must parse valid input")
		assert.Equal(t, tc.expected, result.String(), "UnmarshalJSON should preserve the value")
	}
	result := NewFromInt(2)
	require.NoError(t, result.UnmarshalJSON([]byte("null")), "UnmarshalJSON must accept null")
	assert.Equal(t, "2", result.String(), "UnmarshalJSON should leave the value unchanged for null")
	for _, input := range []string{"", `"`, `""`, `"\x"`, `invalid`} {
		assert.Error(t, result.UnmarshalJSON([]byte(input)),
			"UnmarshalJSON should reject invalid input")
	}
	largeInteger := strings.Repeat("1", maxStringDigits+1)
	largeFraction := "-" + strings.Repeat("9", maxStringDigits+1) + ".1234567890123456789"
	for _, tc := range []struct {
		name, input, expected string
	}{
		{name: "bare large integer", input: largeInteger, expected: largeInteger},
		{name: "quoted large integer", input: `"` + largeInteger + `"`, expected: largeInteger},
		{name: "bare negative large fraction", input: largeFraction, expected: largeFraction},
		{name: "quoted negative large fraction", input: `"` + largeFraction + `"`, expected: largeFraction},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var decoded Decimal
			require.NoError(t, decoded.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must decode extended plain decimal values")
			assert.Equal(t, tc.expected, decoded.String(), "UnmarshalJSON should preserve extended plain decimal values")
		})
	}
	assert.ErrorIs(t, result.UnmarshalJSON([]byte(`"`+largeInteger+`x"`)), errInvalidDecimal,
		"UnmarshalJSON should wrap the extended parser error")
}

func TestDecimalMarshalText(t *testing.T) {
	t.Parallel()
	data, err := MustFromString("1.25").MarshalText()
	require.NoError(t, err, "MarshalText must not error")
	assert.Equal(t, "1.25", string(data), "MarshalText should return the decimal text")
}

func TestDecimalUnmarshalText(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input, expected string
	}{
		{input: "1.25", expected: "1.25"},
		{input: "1e-8", expected: "0.00000001"},
		{input: "1E5", expected: "100000"},
		{input: "-2.5e-4", expected: "-0.00025"},
		{input: ".5", expected: "0.5"},
		{input: "1.", expected: "1"},
	} {
		var decoded Decimal
		require.NoError(t, decoded.UnmarshalText([]byte(tc.input)), "UnmarshalText must parse valid input")
		assert.Equal(t, tc.expected, decoded.String(), "UnmarshalText should preserve the value")
	}
	var result Decimal
	assert.Error(t, result.UnmarshalText([]byte("invalid")),
		"UnmarshalText should reject invalid input")
	for _, value := range []string{
		strings.Repeat("1", maxStringDigits+1),
		"-" + strings.Repeat("9", maxStringDigits+1) + ".1234567890123456789",
	} {
		var decoded Decimal
		require.NoError(t, decoded.UnmarshalText([]byte(value)), "UnmarshalText must decode extended plain decimal values")
		assert.Equal(t, value, decoded.String(), "UnmarshalText should preserve extended plain decimal values")
	}
	assert.ErrorIs(t, result.UnmarshalText([]byte(strings.Repeat("1", maxStringDigits+1)+"x")), errInvalidDecimal,
		"UnmarshalText should wrap the extended parser error")
}

func TestDecimalMarshalBinary(t *testing.T) {
	t.Parallel()
	data, err := MustFromString("1.25").MarshalBinary()
	require.NoError(t, err, "MarshalBinary must not error")
	var result Decimal
	require.NoError(t, result.UnmarshalBinary(data), "UnmarshalBinary must parse native binary data")
	assert.Equal(t, "1.25", result.String(), "MarshalBinary should round-trip the value")
	_, err = MustFromString(strings.Repeat("9", 598)).MarshalBinary()
	assert.ErrorIs(t, err, udecimal.ErrInvalidBinaryData,
		"MarshalBinary should reject a native encoding whose length byte would overflow")
}

func TestDecimalUnmarshalBinary(t *testing.T) {
	t.Parallel()
	var result Decimal
	assert.Error(t, result.UnmarshalBinary([]byte("1.25")), "UnmarshalBinary should reject non-native binary data")
	assert.ErrorIs(t, result.UnmarshalBinary([]byte{0, 255, 11, 0, 0, 0, 0, 0, 0, 0, 125}),
		udecimal.ErrInvalidBinaryData, "UnmarshalBinary should reject unsupported precision")
}

func TestDecimalScan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    any
		expected string
	}{
		{name: "string", input: "1.25", expected: "1.25"},
		{name: "bytes", input: []byte("1.25"), expected: "1.25"},
		{name: "quoted string", input: `"1.25"`, expected: "1.25"},
		{name: "quoted bytes", input: []byte(`"1.25"`), expected: "1.25"},
		{name: "exponent string", input: "1e-8", expected: "0.00000001"},
		{name: "quoted exponent bytes", input: []byte(`"1e-8"`), expected: "0.00000001"},
		{name: "fraction bytes", input: []byte(".5"), expected: "0.5"},
		{name: "int64", input: int64(1), expected: "1"},
		{name: "uint64", input: uint64(1), expected: "1"},
		{name: "float32", input: float32(1.25), expected: "1.25"},
		{name: "float64", input: 1.25, expected: "1.25"},
		{name: "sub-precision float64", input: 1e-20, expected: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result Decimal
			require.NoError(t, result.Scan(tc.input), "Scan must accept supported input")
			assert.Equal(t, tc.expected, result.String(), "Scan should preserve the supported input")
		})
	}
	for _, input := range []any{int(1), int32(1)} {
		var result Decimal
		assert.Error(t, result.Scan(input), "Scan should match the default backend's unsupported integer types")
	}
	var result Decimal
	assert.Error(t, result.Scan(struct{}{}), "Scan should reject unsupported input")
	assert.Error(t, result.Scan(nil), "Scan should reject nil using native udecimal semantics")
	assert.ErrorIs(t, result.Scan(math.NaN()), errInvalidDecimal,
		"Scan should reject non-finite floats through the Decimal façade")
	largeInteger := strings.Repeat("1", maxStringDigits+1)
	largeFraction := "-" + strings.Repeat("9", maxStringDigits+1) + ".1234567890123456789"
	for _, tc := range []struct {
		name     string
		input    any
		expected string
	}{
		{name: "large string", input: largeInteger, expected: largeInteger},
		{name: "large bytes", input: []byte(largeInteger), expected: largeInteger},
		{name: "quoted large string", input: `"` + largeInteger + `"`, expected: largeInteger},
		{name: "quoted large bytes", input: []byte(`"` + largeInteger + `"`), expected: largeInteger},
		{name: "negative large fractional string", input: largeFraction, expected: largeFraction},
		{name: "negative large fractional bytes", input: []byte(largeFraction), expected: largeFraction},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var scanned Decimal
			require.NoError(t, scanned.Scan(tc.input), "Scan must decode extended plain decimal values")
			assert.Equal(t, tc.expected, scanned.String(), "Scan should preserve extended plain decimal values")
		})
	}
	assert.ErrorIs(t, result.Scan(strings.Repeat("1", maxStringDigits+1)+"x"), errInvalidDecimal,
		"Scan should wrap the extended parser error")
}

func TestDecimalValue(t *testing.T) {
	t.Parallel()
	value, err := MustFromString("1.25").Value()
	require.NoError(t, err, "Value must not error")
	assert.Equal(t, driver.Value("1.25"), value, "Value should return the canonical string")
}

func TestParseDecimalParts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input, expectedDigits string
		expectedScale               int64
		expectedNegative            bool
	}{
		{name: "integer", input: "+00120", expectedDigits: "120"},
		{name: "negative exponent expansion", input: "-001.2500e2", expectedDigits: "125", expectedNegative: true},
		{name: "negative exponent", input: "1E-8", expectedDigits: "1", expectedScale: 8},
		{name: "leading decimal point", input: ".5", expectedDigits: "5", expectedScale: 1},
		{name: "trailing decimal point", input: "1.", expectedDigits: "1"},
		{name: "signed zero", input: "-000.000", expectedDigits: "0", expectedNegative: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			digits, scale, negative, err := parseDecimalParts(tc.input)
			require.NoError(t, err, "parseDecimalParts must parse valid input")
			assert.Equal(t, tc.expectedDigits, digits, "parseDecimalParts should return the expected digits")
			assert.Equal(t, tc.expectedScale, scale, "parseDecimalParts should return the expected scale")
			assert.Equal(t, tc.expectedNegative, negative, "parseDecimalParts should return the expected sign")
		})
	}

	for _, tc := range []struct {
		name, input string
	}{
		{name: "empty", input: ""},
		{name: "invalid exponent", input: "1e"},
		{name: "multiple exponents", input: "1e2e3"},
		{name: "out of range exponent", input: "1e999999999999"},
		{name: "missing mantissa", input: "+"},
		{name: "multiple decimal points", input: "1.2.3"},
		{name: "decimal point only", input: "."},
		{name: "invalid digit", input: "12x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, err := parseDecimalParts(tc.input)
			assert.ErrorIs(t, err, errInvalidDecimal, "parseDecimalParts should reject invalid input")
		})
	}
}

func TestFormatDecimalParts(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, digits, expected string
		scale                  int64
		negative               bool
	}{
		{name: "integer", digits: "123", expected: "123"},
		{name: "fraction", digits: "123", scale: 2, expected: "1.23"},
		{name: "leading fractional zeroes", digits: "12", scale: 3, expected: "0.012"},
		{name: "positive exponent expansion", digits: "12", scale: -2, expected: "1200"},
		{name: "negative", digits: "123", scale: 2, negative: true, expected: "-1.23"},
		{name: "negative zero", digits: "0", negative: true, expected: "0"},
		{name: "maximum expansion", digits: "1", scale: 1 - maxStringDigits, expected: "1" + strings.Repeat("0", maxStringDigits-1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := formatDecimalParts(tc.digits, tc.scale, tc.negative)
			require.NoError(t, err, "formatDecimalParts must format supported decimal parts")
			assert.Equal(t, tc.expected, result, "formatDecimalParts should return canonical decimal text")
		})
	}

	_, err := formatDecimalParts("1", -maxStringDigits, false)
	assert.ErrorIs(t, err, errInvalidDecimal, "formatDecimalParts should reject expansion beyond the parser cap")
}

func TestNormalise(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    string
		truncate bool
		expected string
	}{
		{name: "scientific", input: "-1.25e2", expected: "-125"},
		{name: "expand exponent", input: "1e2", expected: "100"},
		{name: "zero", input: "000.000", expected: "0"},
		{name: "truncate", input: "0.12345678901234567899", truncate: true, expected: "0.1234567890123456789"},
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
	assert.Equal(t, "0", result, "normalise should apply udecimal precision")

	errorTests := []struct {
		name, input string
		expected    error
	}{
		{name: "empty", input: "", expected: errInvalidDecimal},
		{name: "out of range exponent", input: "1e999999999999", expected: errInvalidDecimal},
		{name: "missing mantissa", input: "+", expected: errInvalidDecimal},
		{name: "multiple decimal points", input: "1.2.3", expected: errInvalidDecimal},
		{name: "decimal point only", input: ".", expected: errInvalidDecimal},
		{name: "expanded value too long", input: "1e200", expected: errInvalidDecimal},
		{name: "negative expanded value too long", input: "-1e200", expected: errInvalidDecimal},
		{name: "precision", input: "0.12345678901234567899", expected: errPrecisionOutOfRange},
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

func TestApplyUdecimalPrecision(t *testing.T) {
	t.Parallel()
	result, scale, err := applyUdecimalPrecision("123", 2, false, "1.23")
	require.NoError(t, err, "applyUdecimalPrecision must accept supported precision")
	assert.Equal(t, "123", result, "applyUdecimalPrecision should preserve supported digits")
	assert.Equal(t, int64(2), scale, "applyUdecimalPrecision should preserve supported scale")

	result, scale, err = applyUdecimalPrecision("", 2, false, "0.00")
	require.NoError(t, err, "applyUdecimalPrecision must accept zero")
	assert.Equal(t, "0", result, "applyUdecimalPrecision should canonicalise zero")
	assert.Zero(t, scale, "applyUdecimalPrecision should remove zero scale")

	_, _, err = applyUdecimalPrecision("123", 20, false, "0.00000000000000000123")
	assert.ErrorIs(t, err, errPrecisionOutOfRange, "applyUdecimalPrecision should reject excess precision")
	result, scale, err = applyUdecimalPrecision("1", 20, true, "0.00000000000000000001")
	require.NoError(t, err, "applyUdecimalPrecision must truncate a sub-precision value")
	assert.Equal(t, "0", result, "applyUdecimalPrecision should truncate a sub-precision value to zero")
	assert.Zero(t, scale, "applyUdecimalPrecision should remove truncated zero scale")
	result, scale, err = applyUdecimalPrecision("101", 20, true, "0.00000000000000000101")
	require.NoError(t, err, "applyUdecimalPrecision must truncate excess precision")
	assert.Equal(t, "1", result, "applyUdecimalPrecision should remove trailing zeroes after truncation")
	assert.Equal(t, int64(18), scale, "applyUdecimalPrecision should adjust scale after removing trailing zeroes")
}

func TestParseUdecimal(t *testing.T) {
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
			result, err := parseUdecimal(tc.value)
			require.NoError(t, err, "parseUdecimal must parse the value")
			assert.Equal(t, tc.value, result.String(), "parseUdecimal should preserve the value")
		})
	}
	_, err := parseUdecimal("invalid")
	assert.Error(t, err, "parseUdecimal should reject invalid input")
}

func TestDivideUdecimal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, left, right, expected string
	}{
		{name: "native division", left: "1", right: "2", expected: "0.5"},
		{name: "overflow fallback", left: "340282366920938463463374607431768211455", right: "184467440737095516141553255926290448384", expected: "1.8446744073709551617"},
		{name: "negative overflow fallback", left: "-340282366920938463463374607431768211455", right: "184467440737095516141553255926290448384", expected: "-1.8446744073709551617"},
		{name: "fractional overflow fallback", left: "34028236692093846346337460743176821145.5", right: "18446744073709551614155325592629044838.4", expected: "1.8446744073709551617"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			left := udecimal.MustParse(tc.left)
			right := udecimal.MustParse(tc.right)
			assert.Equal(t, tc.expected, divideUdecimal(left, right).String(),
				"divideUdecimal should return the expected native quotient")
		})
	}
	t.Run("unrelated backend panic", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithError(t, udecimal.ErrDivideByZero.Error(),
			func() { divideUdecimal(udecimal.One, udecimal.Zero) },
			"divideUdecimal should preserve unrelated backend panics")
	})
	//nolint:dogsled // Only the u128 fit flag is under test here.
	_, _, _, _, fitsU128 := udecimalBigIntDivisionFactor.ToHiLo()
	assert.False(t, fitsU128, "udecimalBigIntDivisionFactor should exceed u128")
}

func TestModuloUdecimal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, left, right, expected string
	}{
		{name: "native remainder", left: "7", right: "3", expected: "1"},
		{name: "overflow fallback", left: "340282366920938463463374607431768211455", right: "18446744073709551614.1553255926290448384", expected: "15581492618384294731.1821351505477763072"},
		{name: "negative overflow fallback", left: "-340282366920938463463374607431768211455", right: "18446744073709551614.1553255926290448384", expected: "-15581492618384294731.1821351505477763072"},
		{name: "fractional operands", left: "7.5", right: "2", expected: "1.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			left := udecimal.MustParse(tc.left)
			right := udecimal.MustParse(tc.right)
			assert.Equal(t, tc.expected, moduloUdecimal(left, right).String(),
				"moduloUdecimal should return the expected native remainder")
		})
	}
	t.Run("unrelated backend panic", func(t *testing.T) {
		t.Parallel()
		assert.PanicsWithError(t, udecimal.ErrDivideByZero.Error(),
			func() { moduloUdecimal(udecimal.One, udecimal.Zero) },
			"moduloUdecimal should preserve unrelated backend panics")
	})
}

func TestNewUdecimalFromFloat(t *testing.T) {
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
			result, err := newUdecimalFromFloat(tc.value)
			require.NoError(t, err, "newUdecimalFromFloat must convert the supported value")
			assert.Equal(t, tc.expected, result.String(),
				"newUdecimalFromFloat should preserve the supported value")
		})
	}
	_, err := newUdecimalFromFloat(math.NaN())
	assert.ErrorIs(t, err, errInvalidDecimal, "newUdecimalFromFloat should reject a non-finite value")
}

func TestPowerOfTenUdecimal(t *testing.T) {
	t.Parallel()
	for _, exponent := range []int{0, 1, maxStringDigits} {
		expected := "1" + strings.Repeat("0", exponent)
		assert.Equal(t, expected, powerOfTenUdecimal(exponent).String(),
			"powerOfTenUdecimal should return the requested power")
	}
}

func TestPowUdecimal(t *testing.T) {
	t.Parallel()
	value, err := parseUdecimal("2")
	require.NoError(t, err, "parseUdecimal must parse the base")
	exponent, err := parseUdecimal("3")
	require.NoError(t, err, "parseUdecimal must parse the exponent")
	result, err := powUdecimal(value, exponent)
	require.NoError(t, err, "powUdecimal must apply an integer exponent")
	assert.Equal(t, "8", result.String(), "powUdecimal should return the power")
}

func TestRoundUdecimal(t *testing.T) {
	t.Parallel()
	largeHalf := "5" + strings.Repeat("0", maxStringDigits)
	largeRounded := "1" + strings.Repeat("0", maxStringDigits+1)
	for _, tc := range []struct {
		name, value, expected string
		places                int32
	}{
		{name: "fractional", value: "1.25", places: 1, expected: "1.3"},
		{name: "maximum precision", value: "1.25", places: maxPrecision, expected: "1.25"},
		{name: "beyond parser cap", value: "1.25", places: -maxStringDigits - 1, expected: "0"},
		{name: "minimum int32 places", value: "1.25", places: math.MinInt32, expected: "0"},
		{name: "large positive half", value: largeHalf, places: -maxStringDigits - 1, expected: largeRounded},
		{name: "large negative half", value: "-" + largeHalf, places: -maxStringDigits - 1, expected: "-" + largeRounded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := parseUdecimal(tc.value)
			require.NoError(t, err, "parseUdecimal must parse the value")
			assert.Equal(t, tc.expected, roundUdecimal(value, tc.places).String(),
				"roundUdecimal should return the expected value")
		})
	}
}
