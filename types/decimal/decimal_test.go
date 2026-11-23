package decimal

import (
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textDecoder decodes decimal text through a method that parses it.
type textDecoder struct {
	name   string
	decode func(*Decimal, string) error
}

var (
	unmarshalJSONDecoders = []textDecoder{
		{name: "UnmarshalJSON quoted", decode: func(d *Decimal, value string) error { return d.UnmarshalJSON([]byte(`"` + value + `"`)) }},
		{name: "UnmarshalJSON bare", decode: func(d *Decimal, value string) error { return d.UnmarshalJSON([]byte(value)) }},
	}
	unmarshalTextDecoders = []textDecoder{
		{name: "UnmarshalText", decode: func(d *Decimal, value string) error { return d.UnmarshalText([]byte(value)) }},
	}
	scanDecoders = []textDecoder{
		{name: "Scan string", decode: func(d *Decimal, value string) error { return d.Scan(value) }},
		{name: "Scan bytes", decode: func(d *Decimal, value string) error { return d.Scan([]byte(value)) }},
	}
	textDecoders = slices.Concat(unmarshalJSONDecoders, unmarshalTextDecoders, scanDecoders)
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
		{name: "u128 division band", value: "184467440737095516160", precision: -20, expected: "100000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, MustFromString(tc.value).Truncate(tc.precision).String(),
				"Truncate should return the expected value")
		})
	}
}

func TestDecimalFormat(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		format, value, expected string
	}{
		{format: "%v", value: "3.14", expected: "3.14"},
		{format: "%+v", value: "-3.14", expected: "-3.14"},
		{format: "%s", value: "0", expected: "0"},
		{format: "%q", value: "3.14", expected: `"3.14"`},
		{format: "%x", value: "3.14", expected: "332e3134"},
		{format: "%X", value: "3.14", expected: "332E3134"},
		{format: "%8v", value: "3.14", expected: "    3.14"},
		{format: "%-8s|", value: "3.14", expected: "3.14    |"},
		{format: "%f", value: "3.14159", expected: "3.14159"},
		{format: "%F", value: "-3.14159", expected: "-3.14159"},
		{format: "%f", value: "1234567890123456789012345678901234567890", expected: "1234567890123456789012345678901234567890"},
		{format: "%.2f", value: "3.14159", expected: "3.14"},
		{format: "%.2f", value: "2.675", expected: "2.68"},
		{format: "%.2f", value: "-2.675", expected: "-2.68"},
		{format: "%.0f", value: "2.5", expected: "3"},
		{format: "%.3f", value: "1.5", expected: "1.500"},
		{format: "%.2f", value: "-0.001", expected: "0.00"},
		{format: "%+.2f", value: "3.14159", expected: "+3.14"},
		{format: "% .2f", value: "3.14159", expected: " 3.14"},
		{format: "%8.2f", value: "3.14159", expected: "    3.14"},
		{format: "%-8.2f|", value: "3.14159", expected: "3.14    |"},
		{format: "%08.2f", value: "-3.14159", expected: "-0003.14"},
		{format: "%+08.2f", value: "3.14159", expected: "+0003.14"},
		{format: "%2.1f", value: "123.45", expected: "123.5"},
		{format: "%d", value: "3.14", expected: "%!d(decimal.Decimal=3.14)"},
		{format: "%e", value: "-3.14", expected: "%!e(decimal.Decimal=-3.14)"},
	} {
		t.Run(tc.format+" "+tc.value, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, fmt.Sprintf(tc.format, MustFromString(tc.value)),
				"Sprintf should format the decimal as the default backend does")
		})
	}
	t.Run("Go syntax", testDecimalFormatGoSyntax)
	t.Run("struct field", func(t *testing.T) {
		t.Parallel()
		price := struct{ Price Decimal }{Price: MustFromString("3.14")}
		assert.Equal(t, "{Price:3.14}", fmt.Sprintf("%+v", price), "Sprintf should format a decimal field with its value")
	})
}

// The cases below cover behaviour that both builds share. Each build's test for
// the method runs them, so the default build pins what the udecimal build must
// match.

func testNewFromStringExponentLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value, expected string
	}{
		{name: "positive limit", value: "0e10000", expected: "0"},
		{name: "negative limit", value: "0e-10000", expected: "0"},
		{name: "fractional digits offsetting the exponent", value: "0.0e10001", expected: "0"},
		{name: "non-zero fractional digits offsetting the exponent", value: "0." + strings.Repeat("0", 10000) + "1e10001", expected: "1"},
		{name: "fractional digits at limit", value: "1." + strings.Repeat("0", 10000), expected: "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := NewFromString(tc.value)
			require.NoError(t, err, "NewFromString must accept an exponent at the decode limit")
			assert.Equal(t, tc.expected, result.String(), "NewFromString should return the expected value")
			for _, decoder := range textDecoders {
				t.Run(decoder.name, func(t *testing.T) {
					t.Parallel()
					// Start from a non-zero value so that a decoder which returns
					// without decoding cannot pass on the zero rows.
					decoded := NewFromInt(7)
					require.NoError(t, decoder.decode(&decoded, tc.value), "decoder must accept an exponent at the decode limit")
					assert.Equal(t, tc.expected, decoded.String(), "decoder should return the expected value")
				})
			}
		})
	}
	for _, tc := range []struct {
		name, value string
	}{
		{name: "beyond positive limit", value: "0e10001"},
		{name: "beyond negative limit", value: "0e-10001"},
		{name: "uppercase exponent beyond limit", value: "0E10001"},
		{name: "signed exponent beyond limit", value: "0e+10001"},
		{name: "signed mantissa beyond limit", value: "+0e10001"},
		{name: "fractional digits beyond limit", value: "1." + strings.Repeat("0", 10001)},
		{name: "negative fractional digits beyond limit", value: "-1." + strings.Repeat("0", 10001)},
		{name: "integer zeroes cancelling the exponent", value: "1" + strings.Repeat("0", 10001) + "e-10001"},
		{name: "minimum int32 exponent", value: "0e-2147483648"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewFromString(tc.value)
			assert.Error(t, err, "NewFromString should reject an exponent beyond the decode limit")
			for _, decoder := range textDecoders {
				t.Run(decoder.name, func(t *testing.T) {
					t.Parallel()
					assert.Error(t, decoder.decode(new(Decimal), tc.value), "decoder should reject an exponent beyond the decode limit")
				})
			}
		})
	}
}

func testNewFromStringUnicodeMinus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value, expected string
	}{
		{name: "integer", value: "\u22125", expected: "-5"},
		{name: "fraction", value: "\u22120.125", expected: "-0.125"},
		{name: "exponent", value: "1.25e\u22122", expected: "0.0125"},
		{name: "mantissa and exponent", value: "\u22121.25e\u22122", expected: "-0.0125"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := NewFromString(tc.value)
			require.NoError(t, err, "NewFromString must accept the Unicode minus sign")
			assert.Equal(t, tc.expected, result.String(), "NewFromString should read the Unicode minus sign as '-'")
			for _, decoder := range textDecoders {
				t.Run(decoder.name, func(t *testing.T) {
					t.Parallel()
					var decoded Decimal
					require.NoError(t, decoder.decode(&decoded, tc.value), "decoder must accept the Unicode minus sign")
					assert.Equal(t, tc.expected, decoded.String(), "decoder should read the Unicode minus sign as '-'")
				})
			}
		})
	}
	for _, tc := range []struct {
		name, value string
	}{
		{name: "sign only", value: "\u2212"},
		{name: "repeated sign", value: "\u2212\u22125"},
		{name: "trailing sign", value: "5\u2212"},
		{name: "sign after the decimal point", value: ".\u22125"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewFromString(tc.value)
			assert.Error(t, err, "NewFromString should reject invalid use of the Unicode minus sign")
			for _, decoder := range textDecoders {
				t.Run(decoder.name, func(t *testing.T) {
					t.Parallel()
					assert.Error(t, decoder.decode(new(Decimal), tc.value), "decoder should reject invalid use of the Unicode minus sign")
				})
			}
		})
	}
}

func testDecimalUnmarshalJSONBareNull(t *testing.T) {
	t.Parallel()
	value := NewFromInt(7)
	require.NoError(t, value.UnmarshalJSON([]byte("null")), "UnmarshalJSON must accept a bare null")
	assert.Equal(t, "7", value.String(), "UnmarshalJSON should leave the value unchanged for a bare null")
}

func testDecimalUnmarshalTextQuoted(t *testing.T) {
	t.Parallel()
	decoded := NewFromInt(7)
	require.Error(t, decoded.UnmarshalText([]byte(`"1.25"`)), "UnmarshalText must reject quoted text")
	assert.True(t, decoded.IsZero(), "UnmarshalText should leave the value zero after rejecting quoted text")
}

func testDecodeFailureLeavesZero(t *testing.T, decoders []textDecoder) {
	t.Helper()
	for _, tc := range []struct {
		name, value string
	}{
		{name: "invalid text", value: "invalid"},
		{name: "invalid text beyond the udecimal parser limit", value: strings.Repeat("1", 300) + "x"},
		{name: "quoted null", value: `"null"`},
		{name: "minus then plus sign before 40 digits", value: "-+" + strings.Repeat("1", 40)},
		{name: "quoted minus then plus sign before 40 digits", value: `"-+` + strings.Repeat("1", 40) + `"`},
		{name: "two minus signs before 40 zeros", value: "--" + strings.Repeat("0", 40)},
		{name: "minus then plus sign before a 39-digit fraction", value: "-+" + strings.Repeat("1", 38) + ".1"},
		{name: "two minus signs before a zero fraction", value: "--" + strings.Repeat("0", 38) + ".0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for _, decoder := range decoders {
				t.Run(decoder.name, func(t *testing.T) {
					t.Parallel()
					decoded := NewFromInt(7)
					require.Error(t, decoder.decode(&decoded, tc.value), "decoder must reject the input")
					assert.True(t, decoded.IsZero(), "decoder should leave the value zero after a decoding failure")
				})
			}
		})
	}
}

func testDecimalScanInputTypes(t *testing.T) {
	t.Parallel()
	largeInteger, ok := new(big.Int).SetString("-1267650600228229401496703205376", 10)
	require.True(t, ok, "big.Int must parse the test value")
	for _, tc := range []struct {
		name     string
		input    any
		expected string
	}{
		{name: "string", input: "1.25", expected: "1.25"},
		{name: "bytes", input: []byte("1.25"), expected: "1.25"},
		{name: "quoted string", input: `"1.25"`, expected: "1.25"},
		{name: "Unicode minus string", input: "\u22121.25", expected: "-1.25"},
		{name: "float32", input: float32(54.33), expected: "54.33"},
		{name: "float64", input: 54.33, expected: "54.33"},
		{name: "int", input: int(-1), expected: "-1"},
		{name: "int8", input: int8(math.MinInt8), expected: "-128"},
		{name: "int16", input: int16(math.MinInt16), expected: "-32768"},
		{name: "int32", input: int32(math.MinInt32), expected: "-2147483648"},
		{name: "int64", input: int64(math.MinInt64), expected: "-9223372036854775808"},
		{name: "uint", input: uint(1), expected: "1"},
		{name: "uint8", input: uint8(math.MaxUint8), expected: "255"},
		{name: "uint16", input: uint16(math.MaxUint16), expected: "65535"},
		{name: "uint32", input: uint32(math.MaxUint32), expected: "4294967295"},
		{name: "uint64", input: uint64(math.MaxUint64), expected: "18446744073709551615"},
		{name: "big.Int", input: *largeInteger, expected: largeInteger.String()},
		{name: "big.Int pointer", input: largeInteger, expected: largeInteger.String()},
		{name: "Decimal", input: MustFromString("-1.25"), expected: "-1.25"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result Decimal
			require.NoError(t, result.Scan(tc.input), "Scan must accept the input type")
			assert.Equal(t, tc.expected, result.String(), "Scan should preserve the input value")
		})
	}
	for _, tc := range []struct {
		name, expected string
		input          any
	}{
		{name: "nil", input: nil, expected: "7"},
		{name: "nil big.Int pointer", input: (*big.Int)(nil), expected: "7"},
		{name: "Decimal pointer", input: new(Decimal), expected: "7"},
		{name: "float32 NaN", input: float32(math.NaN()), expected: "7"},
		{name: "float64 infinity", input: math.Inf(-1), expected: "7"},
		{name: "bool", input: true, expected: "7"},
		{name: "invalid string", input: "invalid", expected: "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := NewFromInt(7)
			assert.Error(t, result.Scan(tc.input), "Scan should reject the input")
			assert.Equal(t, tc.expected, result.String(), "Scan should leave the expected value after rejecting the input")
		})
	}
}

func testDecimalPowZeroBase(t *testing.T) {
	t.Parallel()
	for _, exponent := range []int64{math.MinInt32 - 1, -2, -1, 0, 1, 3, math.MaxInt32 + 1} {
		t.Run("exponent "+strconv.FormatInt(exponent, 10), func(t *testing.T) {
			t.Parallel()
			assert.True(t, Zero.Pow(NewFromInt(exponent)).IsZero(), "Pow should return zero for a zero base")
		})
	}
}
