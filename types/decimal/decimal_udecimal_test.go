//go:build udecimal_on

package decimal

import (
	"database/sql/driver"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"strconv"
	"strings"
	"testing"
	"time"

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

	for _, tc := range []struct {
		name, input string
		expected    error
	}{
		{name: "excess precision", input: "0." + strings.Repeat("1", 20), expected: errPrecisionOutOfRange},
		{name: "exponent beyond decode limit", input: "0e10001", expected: errExponentOutOfRange},
		{name: "invalid input", input: "not-a-number", expected: errInvalidDecimal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewFromString(tc.input)
			assert.ErrorIs(t, err, tc.expected, "NewFromString should return the expected error")
		})
	}
	for _, value := range []string{
		strings.Repeat("1", maxStringDigits+1),
		"-" + strings.Repeat("9", maxStringDigits+1) + ".1234567890123456789",
	} {
		result, err := NewFromString(value)
		require.NoError(t, err, "NewFromString must parse extended plain decimal values")
		assert.Equal(t, value, result.String(), "NewFromString should preserve extended plain decimal values")
	}
	t.Run("decode exponent limit", testNewFromStringExponentLimit)
	t.Run("Unicode minus sign", testNewFromStringUnicodeMinus)
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
		{name: "exact quotient", left: "1", right: "2", expected: "0.5"},
		{name: "u128 division band", left: "340282366920938463463374607431768211455", right: "184467440737095516141553255926290448384", expected: "1.8446744073709551617"},
		{name: "negative dividend in u128 division band", left: "-340282366920938463463374607431768211455", right: "184467440737095516141553255926290448384", expected: "-1.8446744073709551617"},
		{name: "negative divisor in u128 division band", left: "340282366920938463463374607431768211455", right: "-184467440737095516141553255926290448384", expected: "-1.8446744073709551617"},
		{name: "negative operands in u128 division band", left: "-340282366920938463463374607431768211455", right: "-184467440737095516141553255926290448384", expected: "1.8446744073709551617"},
		{name: "fractional operands in u128 division band", left: "34028236692093846346337460743176821145.5", right: "18446744073709551614155325592629044838.4", expected: "1.8446744073709551617"},
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
		{name: "u128 division band", left: "340282366920938463463374607431768211455", right: "18446744073709551614.1553255926290448384", expected: "15581492618384294731.1821351505477763072"},
		{name: "negative dividend in u128 division band", left: "-340282366920938463463374607431768211455", right: "18446744073709551614.1553255926290448384", expected: "-15581492618384294731.1821351505477763072"},
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
	assert.Panics(t, func() { NewFromInt(2).Pow(MustFromString("1.5")) },
		"Pow should panic for a fractional exponent")
	assert.Panics(t, func() { Zero.Pow(MustFromString("1.5")) },
		"Pow should reject a fractional exponent for a zero base")
	t.Run("zero base", testDecimalPowZeroBase)
	for _, tc := range []struct {
		name, value string
		exponent    int64
		expected    string
	}{
		{name: "exponent at positive int32 limit", value: "-1", exponent: math.MaxInt32, expected: "-1"},
		{name: "exponent at negative int32 limit", value: "1", exponent: -math.MaxInt32, expected: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).Pow(NewFromInt(tc.exponent)).String(),
				"Pow should accept an exponent at the int32 limit")
		})
	}
	for _, tc := range []struct {
		name     string
		exponent int64
	}{
		{name: "exponent beyond positive int32 limit", exponent: math.MaxInt32 + 1},
		{name: "exponent beyond negative int32 limit", exponent: math.MinInt32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.PanicsWithError(t, udecimal.ErrExponentTooLarge.Error(), func() { NewFromInt(1).Pow(NewFromInt(tc.exponent)) },
				"Pow should panic for an exponent beyond the int32 limit")
		})
	}
	for _, tc := range []struct {
		name, value string
		exponent    int64
		expected    string
	}{
		{name: "u128 division band", value: "1.226421929545717699", exponent: 3, expected: "1.8446744073709551615"},
		{name: "negative base in u128 division band", value: "-1.226421929545717699", exponent: 3, expected: "-1.8446744073709551615"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).Pow(NewFromInt(tc.exponent)).String(),
				"Pow should truncate the exact power to udecimal precision")
		})
	}
	for _, value := range []string{"1.5", "-1.5", "0.1234567890123456789", "12345.678", "-0.000000001"} {
		for _, exponent := range []int64{-3, -2, -1, 1, 2, 3, 5} {
			t.Run(value+" to the power of "+strconv.FormatInt(exponent, 10), func(t *testing.T) {
				t.Parallel()
				expected := truncatedPower(t, value, exponent)
				result := MustFromString(value).Pow(NewFromInt(exponent))
				assert.Truef(t, expected.Equal(result), "Pow should return %s, the exact power truncated to udecimal precision, not %s", expected, result)
			})
		}
	}
}

// truncatedPower returns a non-zero value raised to a small exponent, computed
// exactly and then truncated toward zero to udecimal's precision.
func truncatedPower(t *testing.T, value string, exponent int64) Decimal {
	t.Helper()
	base, ok := new(big.Rat).SetString(value)
	require.True(t, ok, "big.Rat must parse the base")
	power := new(big.Rat).SetInt64(1)
	for range max(exponent, -exponent) {
		power.Mul(power, base)
	}
	if exponent < 0 {
		power.Inv(power)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(maxPrecision), nil)
	// Quo truncates toward zero, as udecimal does beyond its precision.
	scaled := new(big.Int).Quo(new(big.Int).Mul(power.Num(), scale), power.Denom())
	result, err := NewFromString(new(big.Rat).SetFrac(scaled, scale).FloatString(maxPrecision))
	require.NoError(t, err, "NewFromString must parse the truncated power")
	return result
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

func TestDecimalIsIntegerScaledValues(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    Decimal
		expected bool
	}{
		{name: "stored scale integer", value: Decimal{value: udecimal.MustParse("-42.000")}, expected: true},
		{name: "computed integer", value: MustFromString("0.5").Add(MustFromString("0.5")), expected: true},
		{name: "computed integer at max scale", value: MustFromString("3").Div(MustFromString("1.5")), expected: true},
		{
			name:     "integer beyond u128",
			value:    Decimal{value: udecimal.MustParse(strings.Repeat("9", 40) + "." + strings.Repeat("0", maxPrecision))},
			expected: true,
		},
		{
			name:     "fraction beyond u128",
			value:    Decimal{value: udecimal.MustParse(strings.Repeat("9", 40) + ".5")},
			expected: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.NotZero(t, tc.value.value.Prec(), "test value should retain a fractional scale")
			assert.Equal(t, tc.expected, tc.value.IsInteger(),
				"IsInteger should identify values without a fractional component")
		})
	}
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
		{name: "u128 division band", value: "184467440737095516160", places: -20, expected: "200000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).Round(tc.places).String(),
				"Round should return the expected value")
		})
	}
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
		{name: "u128 division band", value: "184467440737095516160", places: -20, expected: "200000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value := MustFromString(tc.value)
			assert.Equal(t, tc.expected, requirePrompt(t, func() string { return value.StringFixed(tc.places) }),
				"StringFixed should return the expected representation")
		})
	}
}

// testDecimalFormatGoSyntax checks that %#v prints the MustFromString call that
// recreates the value, since the façade has no New.
func testDecimalFormatGoSyntax(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value string
	}{
		{name: "fraction", value: "3.14"},
		{name: "negative fraction", value: "-0.5"},
		{name: "zero", value: "0"},
		{name: "beyond parser cap", value: strings.Repeat("9", maxStringDigits+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, "decimal.MustFromString("+strconv.Quote(tc.value)+")", fmt.Sprintf("%#v", MustFromString(tc.value)),
				"Go syntax formatting should print the MustFromString call that recreates the value")
		})
	}
	t.Run("width", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, `     decimal.MustFromString("3.14")`, fmt.Sprintf("%#35v", MustFromString("3.14")),
			"Go syntax formatting should apply the width")
	})
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
	var result Decimal
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
	// The parser limit counts the quotes. 200 bytes of quoted JSON still go to
	// udecimal's parser, which keeps a trailing zero's precision, while 199 or
	// 200 quoted digits go to NewFromString, which stores a small value in
	// udecimal's u128 form.
	for _, tc := range []struct {
		name, text   string
		nativeParser bool
	}{
		{name: "quoted fraction at the parser limit", text: strings.Repeat("9", maxStringDigits-4) + ".0", nativeParser: true},
		{name: "199 quoted digits", text: strings.Repeat("0", maxStringDigits-2) + "3"},
		{name: "200 quoted digits", text: strings.Repeat("0", maxStringDigits-1) + "3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var decoded Decimal
			require.NoError(t, decoded.UnmarshalJSON([]byte(`"`+tc.text+`"`)), "UnmarshalJSON must decode the quoted text")
			var expected []byte
			var err error
			if tc.nativeParser {
				expected, err = udecimal.MustParse(tc.text).MarshalBinary()
			} else {
				expected, err = MustFromString(tc.text).MarshalBinary()
			}
			require.NoError(t, err, "MarshalBinary must encode the expected value")
			actual, err := decoded.MarshalBinary()
			require.NoError(t, err, "MarshalBinary must encode the decoded value")
			assert.Equal(t, expected, actual, "UnmarshalJSON should decode the quoted text with the expected parser")
		})
	}
	t.Run("bare null leaves the value unchanged", testDecimalUnmarshalJSONBareNull)
	t.Run("decoding failure leaves zero", func(t *testing.T) {
		t.Parallel()
		testDecodeFailureLeavesZero(t, unmarshalJSONDecoders)
	})
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
	t.Run("quoted text", testDecimalUnmarshalTextQuoted)
	t.Run("decoding failure leaves zero", func(t *testing.T) {
		t.Parallel()
		testDecodeFailureLeavesZero(t, unmarshalTextDecoders)
	})
}

func TestDecimalMarshalBinary(t *testing.T) {
	t.Parallel()
	// The native encoding stores the coefficient in whole machine words, so the
	// most digits that fit its one-byte length depend on the word size.
	maxEncodableNines := 597
	if bits.UintSize == 32 {
		maxEncodableNines = 606
	}
	for _, tc := range []struct {
		name, value string
		expected    error
	}{
		{name: "short value", value: "1.25"},
		{name: "longest encodable value", value: strings.Repeat("9", maxEncodableNines)},
		{name: "length byte overflow", value: strings.Repeat("9", maxEncodableNines+1), expected: udecimal.ErrInvalidBinaryData},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, err := MustFromString(tc.value).MarshalBinary()
			require.ErrorIs(t, err, tc.expected, "MarshalBinary must return the expected error")
			if tc.expected != nil {
				return
			}
			var result Decimal
			require.NoError(t, result.UnmarshalBinary(data), "UnmarshalBinary must parse native binary data")
			assert.Equal(t, tc.value, result.String(), "MarshalBinary should round-trip the value")
		})
	}
}

func TestDecimalUnmarshalBinary(t *testing.T) {
	t.Parallel()
	seededData, err := NewFromInt(7).MarshalBinary()
	require.NoError(t, err, "MarshalBinary must encode the seeded receiver")
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "short payload", data: []byte{1, 2}},
		{name: "invalid native text", data: []byte("1.25")},
		{name: "unsupported precision", data: []byte{0, 255, 11, 0, 0, 0, 0, 0, 0, 0, 125}},
		{name: "precision above maximum", data: []byte{0, maxPrecision + 1, 11, 0, 0, 0, 0, 0, 0, 0, 125}},
		{name: "big integer precision above maximum", data: []byte{16, maxPrecision + 1, 4, 1}},
		{name: "u128 length mismatch", data: []byte{9, 9, 9}},
		{name: "unsupported u128 length", data: []byte{0, 0, 3}},
		{name: "big integer length mismatch", data: []byte{17, 3, 200, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := NewFromInt(7)
			assert.ErrorIs(t, result.UnmarshalBinary(tc.data), udecimal.ErrInvalidBinaryData,
				"UnmarshalBinary should reject malformed data")
			data, err := result.MarshalBinary()
			require.NoError(t, err, "MarshalBinary must encode the receiver")
			assert.Equal(t, seededData, data, "UnmarshalBinary should preserve the receiver after an error")
		})
	}

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "maximum precision", data: []byte{0, maxPrecision, 11, 0, 0, 0, 0, 0, 0, 0, 125}},
		{name: "big integer maximum precision", data: []byte{16, maxPrecision, 4, 125}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var fresh Decimal
			require.NoError(t, fresh.UnmarshalBinary(tc.data), "UnmarshalBinary must accept the maximum precision")
			assert.Equal(t, "0.0000000000000000125", fresh.String(), "UnmarshalBinary should decode the maximum precision")
			reused := MustFromString(strings.Repeat("9", 45))
			require.NoError(t, reused.UnmarshalBinary(tc.data), "UnmarshalBinary must decode into a used receiver")
			freshData, err := fresh.MarshalBinary()
			require.NoError(t, err, "MarshalBinary must encode the value decoded into a fresh receiver")
			reusedData, err := reused.MarshalBinary()
			require.NoError(t, err, "MarshalBinary must encode the value decoded into a used receiver")
			assert.Equal(t, freshData, reusedData, "UnmarshalBinary should replace the receiver's previous value")
		})
	}

	zeroData, err := Zero.MarshalBinary()
	require.NoError(t, err, "MarshalBinary must encode zero")
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "negative zero", data: []byte{1, 0, 11, 0, 0, 0, 0, 0, 0, 0, 0}},
		{name: "negative zero with precision", data: []byte{1, 5, 11, 0, 0, 0, 0, 0, 0, 0, 0}},
		{name: "big integer negative zero with precision", data: []byte{17, 3, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result Decimal
			require.NoError(t, result.UnmarshalBinary(tc.data), "UnmarshalBinary must accept a native zero payload")
			assert.True(t, result.IsZero(), "UnmarshalBinary should decode zero")
			assert.Equal(t, "0", result.String(), "UnmarshalBinary should decode zero without sign or precision")
			assert.True(t, result.Equal(Zero), "UnmarshalBinary should canonicalise zero equality")
			assert.Zero(t, result.Cmp(Zero), "UnmarshalBinary should canonicalise zero comparison")
			data, err := result.MarshalBinary()
			require.NoError(t, err, "MarshalBinary must encode the decoded zero")
			assert.Equal(t, zeroData, data, "UnmarshalBinary should decode the canonical zero")
		})
	}

	t.Run("nil receiver", func(t *testing.T) {
		t.Parallel()
		var result *Decimal
		assert.ErrorIs(t, result.UnmarshalBinary([]byte{1, 2}), udecimal.ErrInvalidBinaryData,
			"UnmarshalBinary should reject malformed data before using the receiver")
	})
}

func TestDecimalScan(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		input    any
		expected string
	}{
		{name: "quoted bytes", input: []byte(`"1.25"`), expected: "1.25"},
		{name: "exponent string", input: "1e-8", expected: "0.00000001"},
		{name: "quoted exponent bytes", input: []byte(`"1e-8"`), expected: "0.00000001"},
		{name: "fraction bytes", input: []byte(".5"), expected: "0.5"},
		{name: "int64", input: int64(1), expected: "1"},
		{name: "uint64", input: uint64(1), expected: "1"},
		{name: "float32", input: float32(1.25), expected: "1.25"},
		{name: "float32 shortest form", input: float32(0.1), expected: "0.1"},
		{name: "sub-precision float32", input: float32(1e-20), expected: "0"},
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
	for _, tc := range []struct {
		name     string
		input    any
		expected error
	}{
		{name: "unsupported type", input: struct{}{}, expected: errUnsupportedScanType},
		{name: "nil", input: nil, expected: errUnsupportedScanType},
		{name: "nil big.Int pointer", input: (*big.Int)(nil), expected: errInvalidDecimal},
		{name: "non-finite float64", input: math.NaN(), expected: errInvalidDecimal},
		{name: "non-finite float32", input: float32(math.Inf(1)), expected: errInvalidDecimal},
		{name: "invalid text beyond the parser cap", input: strings.Repeat("1", maxStringDigits+1) + "x", expected: errInvalidDecimal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result Decimal
			assert.ErrorIs(t, result.Scan(tc.input), tc.expected, "Scan should return the expected error")
		})
	}
	largeInteger := strings.Repeat("1", maxStringDigits+1)
	largeBigInt, ok := new(big.Int).SetString(largeInteger, 10)
	require.True(t, ok, "big.Int must parse the large integer")
	largeFraction := "-" + strings.Repeat("9", maxStringDigits+1) + ".1234567890123456789"
	for _, tc := range []struct {
		name     string
		input    any
		expected string
	}{
		{name: "large big.Int", input: largeBigInt, expected: largeInteger},
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
	t.Run("input types", testDecimalScanInputTypes)
	t.Run("decoding failure leaves zero", func(t *testing.T) {
		t.Parallel()
		testDecodeFailureLeavesZero(t, scanDecoders)
	})
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
		{name: "positive exponent limit", input: "0e10000", expectedDigits: "0"},
		{name: "fractional digits at exponent limit", input: "1." + strings.Repeat("0", shopspringMaxDecodeExponent), expectedDigits: "1"},
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

	for _, tc := range []struct {
		name, input string
	}{
		{name: "beyond positive exponent limit", input: "0e10001"},
		{name: "beyond negative exponent limit", input: "-0e-10001"},
		{name: "fractional digits beyond exponent limit", input: "1." + strings.Repeat("0", shopspringMaxDecodeExponent+1)},
		{name: "integer digits cancelling the exponent", input: "1" + strings.Repeat("0", shopspringMaxDecodeExponent+1) + "e-10001"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, err := parseDecimalParts(tc.input)
			assert.ErrorIs(t, err, errExponentOutOfRange, "parseDecimalParts should reject an exponent beyond the decode limit")
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
		{name: "exponent beyond decode limit", input: "0e-10001", expected: errExponentOutOfRange},
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
	t.Run("u128 division band", func(t *testing.T) {
		t.Parallel()
		// Leading zeroes take a u128 value down the chunked path, where dividing
		// by 10^20 reaches the u128 division band of quagmt/udecimal#46.
		result, err := parseUdecimal(strings.Repeat("0", maxStringDigits) + "1.84467440737095516169")
		require.NoError(t, err, "parseUdecimal must parse a long value in the u128 division band")
		assert.Equal(t, "1.8446744073709551616", result.String(), "parseUdecimal should truncate the long value to udecimal precision")
	})
	_, err := parseUdecimal("invalid")
	assert.Error(t, err, "parseUdecimal should reject invalid input")
}

func TestNewUdecimalFromFloat(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    float64
		bitSize  int
		expected string
	}{
		{name: "ordinary", value: 1.25, bitSize: 64, expected: "1.25"},
		{name: "negative exponent", value: math.SmallestNonzeroFloat64, bitSize: 64, expected: "0"},
		{name: "positive exponent", value: math.MaxFloat64, bitSize: 64, expected: strconv.FormatFloat(math.MaxFloat64, 'f', -1, 64)},
		{name: "float32 shortest form", value: float64(float32(54.33)), bitSize: 32, expected: "54.33"},
		{name: "float32 value at float64 precision", value: float64(float32(54.33)), bitSize: 64, expected: "54.33000183105469"},
		{name: "float32 negative exponent", value: math.SmallestNonzeroFloat32, bitSize: 32, expected: "0"},
		{name: "float32 positive exponent", value: math.MaxFloat32, bitSize: 32, expected: strconv.FormatFloat(math.MaxFloat32, 'f', -1, 32)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := newUdecimalFromFloat(tc.value, tc.bitSize)
			require.NoError(t, err, "newUdecimalFromFloat must convert the supported value")
			assert.Equal(t, tc.expected, result.String(),
				"newUdecimalFromFloat should preserve the supported value")
		})
	}
	_, err := newUdecimalFromFloat(math.NaN(), 64)
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
	for _, tc := range []struct {
		name, value, exponent, expected string
	}{
		{name: "positive exponent", value: "2", exponent: "3", expected: "8"},
		{name: "negative exponent", value: "2", exponent: "-2", expected: "0.25"},
		{name: "negative base", value: "-2", exponent: "3", expected: "-8"},
		{name: "zero exponent", value: "1.5", exponent: "0", expected: "1"},
		{name: "zero base", value: "0", exponent: "3", expected: "0"},
		{name: "u128 division band", value: "1.226421929545717699", exponent: "3", expected: "1.8446744073709551615"},
		{name: "negative base in u128 division band", value: "-1.226421929545717699", exponent: "3", expected: "-1.8446744073709551615"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := powUdecimal(udecimal.MustParse(tc.value), udecimal.MustParse(tc.exponent))
			require.NoError(t, err, "powUdecimal must apply an integer exponent")
			assert.Equal(t, tc.expected, result.String(), "powUdecimal should return the power")
		})
	}
	for _, tc := range []struct {
		name, value, exponent string
		expected              error
	}{
		{name: "fractional exponent", value: "2", exponent: "0.5", expected: errInvalidDecimal},
		{name: "exponent beyond int32 limit", value: "2", exponent: "2147483648", expected: udecimal.ErrExponentTooLarge},
		{name: "fractional base beyond int32 limit", value: "1.5", exponent: "2147483648", expected: udecimal.ErrExponentTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := powUdecimal(udecimal.MustParse(tc.value), udecimal.MustParse(tc.exponent))
			assert.ErrorIs(t, err, tc.expected, "powUdecimal should return the expected error")
		})
	}
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
		{name: "u128 division band", value: "184467440737095516160", places: -20, expected: "200000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := parseUdecimal(tc.value)
			require.NoError(t, err, "parseUdecimal must parse the value")
			assert.Equal(t, tc.expected, requirePrompt(t, func() string { return roundUdecimal(value, tc.places).String() }),
				"roundUdecimal should return the expected value")
		})
	}
}

func TestTruncateUdecimal(t *testing.T) {
	t.Parallel()
	largeValue := "5" + strings.Repeat("0", maxStringDigits-1) + "7"
	largeTruncated := "5" + strings.Repeat("0", maxStringDigits)
	for _, tc := range []struct {
		name, value, expected string
		precision             int32
	}{
		{name: "fractional", value: "1.239", precision: 2, expected: "1.23"},
		{name: "maximum precision", value: "0.1234567890123456789", precision: maxPrecision, expected: "0.1234567890123456789"},
		{name: "precision beyond uint8", value: "1.239", precision: math.MaxUint8 + 3, expected: "1.239"},
		{name: "integer digits", value: "-5432.987", precision: -2, expected: "-5400"},
		{name: "every integer digit", value: "5432", precision: -4, expected: "0"},
		{name: "beyond parser cap", value: "1.25", precision: -maxStringDigits - 1, expected: "0"},
		{name: "minimum int32 precision", value: "1.25", precision: math.MinInt32, expected: "0"},
		{name: "minimum int32 precision plus one", value: "1.25", precision: math.MinInt32 + 1, expected: "0"},
		{name: "large positive value", value: largeValue, precision: -1, expected: largeTruncated},
		{name: "large negative value", value: "-" + largeValue, precision: -1, expected: "-" + largeTruncated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := parseUdecimal(tc.value)
			require.NoError(t, err, "parseUdecimal must parse the value")
			assert.Equal(t, tc.expected, requirePrompt(t, func() string { return truncateUdecimal(value, tc.precision).String() }),
				"truncateUdecimal should return the expected value")
		})
	}
}

// requirePrompt returns compute's result within a deadline. A regressed zero
// guard can make extreme precision build a ginormous power of ten, and the
// deadline fails such a test instead of letting it hang.
func requirePrompt(t *testing.T, compute func() string) string {
	t.Helper()
	result := make(chan string, 1)
	go func() { result <- compute() }()
	select {
	case got := <-result:
		return got
	case <-time.After(10 * time.Second):
		require.FailNow(t, "call must return promptly")
		return ""
	}
}

func TestIntegerDigitCount(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value string
		expected    int64
	}{
		{name: "zero", value: "0", expected: 0},
		{name: "fraction only", value: "-0.5", expected: 0},
		{name: "integer and fraction", value: "5432.987", expected: 4},
		{name: "negative integer", value: "-5432", expected: 4},
		{name: "beyond parser cap", value: "1" + strings.Repeat("0", maxStringDigits), expected: maxStringDigits + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			value, err := parseUdecimal(tc.value)
			require.NoError(t, err, "parseUdecimal must parse the value")
			assert.Equal(t, tc.expected, integerDigitCount(value), "integerDigitCount should count the integer digits")
		})
	}
}
