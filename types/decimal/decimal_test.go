package decimal

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"testing"

	shopspring "github.com/shopspring/decimal" //nolint:depguard // Reference output for the udecimal build's Format.
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/internal/bsonwire"
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
	values := []string{"0", "3.14", "-3.14159", "1.20", "2.675", "-0.001", "9.995", "54.33000183105469", strings.Repeat("9", 40)}
	decoded := make([]Decimal, len(values))
	for i, value := range values {
		// UnmarshalText keeps trailing zeros in both builds, as shopspring's parser does.
		require.NoError(t, decoded[i].UnmarshalText([]byte(value)), "UnmarshalText must decode the format test value")
	}
	for _, verb := range "vsqxXfFdeEgGbcoOUt" {
		for flagSet := range 1 << 5 {
			var flags strings.Builder
			for i, flag := range "+-# 0" {
				if flagSet&(1<<i) != 0 {
					flags.WriteRune(flag)
				}
			}
			if verb == 'v' && strings.ContainsRune(flags.String(), '#') {
				// The udecimal build prints %#v as a MustFromString call instead.
				continue
			}
			directive := "%" + flags.String() + string(verb)
			t.Run("matches shopspring "+directive, func(t *testing.T) {
				t.Parallel()
				for _, width := range []string{"", "1", "8", "12"} {
					for _, precision := range []string{"", ".0", ".2", ".5", ".20", ".256"} {
						format := "%" + flags.String() + width + precision + string(verb)
						for i, value := range values {
							assert.Equalf(t, fmt.Sprintf(format, shopspring.RequireFromString(value)), fmt.Sprintf(format, decoded[i]),
								"Sprintf(%q) should print %s as shopspring does", format, value)
						}
					}
				}
			})
		}
	}
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
		{name: "lone minus sign", value: "-"},
		{name: "unbalanced quote", value: `"1.5`},
		{name: "closing quote only", value: `1.5"`},
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
		{name: "float64 beyond float32 precision", input: 1.234567891234, expected: "1.234567891234"},
		{name: "float64 holding a float32 value", input: float64(float32(54.33)), expected: "54.33000183105469"},
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
		{name: "zero big.Int", input: big.Int{}, expected: "0"},
		{name: "zero big.Int pointer", input: new(big.Int), expected: "0"},
		{name: "negative float32 with eight significant digits", input: float32(-1.2345678), expected: "-1.2345678"},
		{name: "negative float64 with 17 significant digits", input: -0.0012345678912345677, expected: "-0.0012345678912345677"},
		{name: "float64 integer past its shortest form", input: float64(1_000_000_000_000_000_128), expected: "1000000000000000100"},
		{name: "big.Int beyond int64", input: *new(big.Int).Lsh(big.NewInt(1), 63), expected: "9223372036854775808"},
		{name: "big.Int beyond uint64", input: *new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1)), expected: "18446744073709551617"},
		{name: "big.Int pointer beyond int64", input: new(big.Int).Lsh(big.NewInt(1), 63), expected: "9223372036854775808"},
		{name: "small negative big.Int pointer", input: big.NewInt(-42), expected: "-42"},
		{name: "Decimal", input: MustFromString("-1.25"), expected: "-1.25"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Start from a non-zero value so that a case which returns without
			// storing its input cannot pass on the zero row.
			result := NewFromInt(7)
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

// bsonDecimal128 returns the BSON Decimal128 written as hex in its stored,
// little-endian byte order.
func bsonDecimal128(t *testing.T, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	require.NoError(t, err, "the Decimal128 hex must decode")
	require.Len(t, data, 16, "a Decimal128 must be 16 bytes")
	return data
}

func testDecimalMarshalBSONValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, value, expected string
	}{
		{name: "zero", value: "0", expected: "00000000000000000000000000004030"},
		{name: "integer", value: "1", expected: "01000000000000000000000000004030"},
		{name: "negative integer", value: "-1", expected: "010000000000000000000000000040b0"},
		{name: "fraction", value: "1.5", expected: "0f000000000000000000000000003e30"},
		{name: "trailing fractional zero", value: "1.50", expected: "96000000000000000000000000003c30"},
		{name: "negative with trailing fractional zeros", value: "-12.3400", expected: "08e201000000000000000000000038b0"},
		{name: "trailing integer zeros", value: "100", expected: "64000000000000000000000000004030"},
		{name: "nineteen fractional digits", value: "-0.0000000000000000001", expected: "01000000000000000000000000001ab0"},
		{name: "34 digits", value: "9999999999999999999999999999999999", expected: "ffffffff638e8d37c087adbe09ed4130"},
		{name: "34 digits with nineteen fractional digits", value: "999999999999999.9999999999999999999", expected: "ffffffff638e8d37c087adbe09ed1b30"},
		{name: "trailing zero beyond 34 digits", value: "99999999999999999999999999999999990", expected: "ffffffff638e8d37c087adbe09ed4330"},
		{name: "power of ten beyond 34 digits", value: "1" + strings.Repeat("0", 50), expected: "000000000a5bc138938d44c64d316230"},
		{name: "trailing zero beyond 34 digits with a fraction", value: "999999999999999999999999999999999.10", expected: "f7ffffff638e8d37c087adbe09ed3f30"},
		{name: "negative 34 digits", value: "-9999999999999999999999999999999999", expected: "ffffffff638e8d37c087adbe09ed41b0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// UnmarshalText keeps trailing zeros in both builds, as shopspring's parser does.
			var value Decimal
			require.NoError(t, value.UnmarshalText([]byte(tc.value)), "UnmarshalText must decode the test value")
			typ, data, err := value.MarshalBSONValue()
			require.NoError(t, err, "MarshalBSONValue must store the value")
			assert.Equal(t, bsonwire.TypeDecimal128, typ, "MarshalBSONValue should store a BSON Decimal128")
			assert.Equal(t, tc.expected, hex.EncodeToString(data), "MarshalBSONValue should store the expected Decimal128")
		})
	}
	t.Run("power of ten at the largest exponent", func(t *testing.T) {
		t.Parallel()
		// The udecimal build caps decoding at 200 digits, so build 10^6144 with Pow.
		typ, data, err := MustFromString("1e96").Pow(NewFromInt(64)).MarshalBSONValue()
		require.NoError(t, err, "MarshalBSONValue must store 10^6144")
		assert.Equal(t, bsonwire.TypeDecimal128, typ, "MarshalBSONValue should store a BSON Decimal128")
		assert.Equal(t, "000000000a5bc138938d44c64d31fe5f", hex.EncodeToString(data), "MarshalBSONValue should store 10^6144 at the largest exponent")
	})
	for _, tc := range []struct {
		name  string
		value Decimal
	}{
		{name: "35 significant digits", value: MustFromString("12345678901234567890123456789012345")},
		{name: "35 significant digits with a fraction", value: MustFromString("1234567890123456.1234567890123456789")},
		{name: "power of ten beyond the largest exponent", value: MustFromString("1e5").Pow(NewFromInt(1229))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			typ, data, err := tc.value.MarshalBSONValue()
			assert.Error(t, err, "MarshalBSONValue should refuse a value that Decimal128 holds only by rounding")
			assert.Zero(t, typ, "MarshalBSONValue should return no BSON type with its error")
			assert.Nil(t, data, "MarshalBSONValue should return no data with its error")
		})
	}
}

func testDecimalUnmarshalBSONValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, expected string
		typ            byte
		data           []byte
	}{
		{name: "null", typ: bsonwire.TypeNull, expected: "0"},
		{name: "null with stray bytes", typ: bsonwire.TypeNull, data: []byte{42}, expected: "0"},
		{name: "undefined", typ: bsonwire.TypeUndefined, expected: "0"},
		{name: "undefined with stray bytes", typ: bsonwire.TypeUndefined, data: []byte{42}, expected: "0"},
		{name: "Decimal128", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "0f000000000000000000000000003e30"), expected: "1.5"},
		{name: "Decimal128 trailing zero", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "96000000000000000000000000003c30"), expected: "1.5"},
		{name: "Decimal128 negative", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "d2040000000000000000000000003cb0"), expected: "-12.34"},
		{name: "Decimal128 nineteen fractional digits", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "01000000000000000000000000001a30"), expected: "0.0000000000000000001"},
		{name: "Decimal128 trailing zero beyond nineteen fractional digits", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "0a000000000000000000000000001830"), expected: "0.0000000000000000001"},
		{name: "Decimal128 trailing zeros beyond nineteen fractional digits", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "e8030000000000000000000000001630"), expected: "0.000000000000000001"},
		{name: "Decimal128 positive exponent", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "0c000000000000000000000000000831"), expected: "12" + strings.Repeat("0", 100)},
		{name: "Decimal128 negative with a positive exponent", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "050000000000000000000000000046b0"), expected: "-5000"},
		{name: "Decimal128 exponent of one", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "05000000000000000000000000004230"), expected: "50"},
		{name: "Decimal128 34 digits", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "ffffffff638e8d37c087adbe09ed4130"), expected: "9999999999999999999999999999999999"},
		{name: "Decimal128 zero", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "00000000000000000000000000004030"), expected: "0"},
		{name: "Decimal128 zero at the smallest exponent", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "00000000000000000000000000000000"), expected: "0"},
		{name: "Decimal128 zero at the largest exponent", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "0000000000000000000000000000fe5f"), expected: "0"},
		{name: "Decimal128 negative zero", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "000000000000000000000000000040b0"), expected: "0"},
		{name: "Decimal128 coefficient beyond 34 digits", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "00000000648e8d37c087adbe09ed4130"), expected: "0"},
		{name: "Decimal128 large coefficient form", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "01000000000000000000000000000060"), expected: "0"},
		{name: "string", typ: bsonwire.TypeString, data: bsonwire.MarshalString("-1.25"), expected: "-1.25"},
		{name: "string in scientific notation", typ: bsonwire.TypeString, data: bsonwire.MarshalString("1.25e3"), expected: "1250"},
		{name: "string with the Unicode minus sign", typ: bsonwire.TypeString, data: bsonwire.MarshalString("\u22121.25"), expected: "-1.25"},
		{name: "string with an unchecked length", typ: bsonwire.TypeString, data: []byte{0xFF, 0, 0, 0, '1', '2', 0}, expected: "12"},
		{name: "string with an unchecked final byte", typ: bsonwire.TypeString, data: []byte{2, 0, 0, 0, '8', 'X'}, expected: "8"},
		{name: "double", typ: bsonwire.TypeDouble, data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(54.33)), expected: "54.33"},
		{name: "double integer", typ: bsonwire.TypeDouble, data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(1e20)), expected: "100000000000000000000"},
		{name: "double negative zero", typ: bsonwire.TypeDouble, data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(math.Copysign(0, -1))), expected: "0"},
		{name: "int32", typ: bsonwire.TypeInt32, data: []byte{0, 0, 0, 0x80}, expected: "-2147483648"},
		{name: "int32 across every byte", typ: bsonwire.TypeInt32, data: []byte{5, 1, 2, 0}, expected: "131333"},
		{name: "int64", typ: bsonwire.TypeInt64, data: []byte{0, 0, 0, 0, 0, 0, 0, 0x80}, expected: "-9223372036854775808"},
		{name: "int64 across both words", typ: bsonwire.TypeInt64, data: []byte{5, 0, 0, 0, 1, 0, 0, 0}, expected: "4294967301"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Start from a non-zero value so that a case which returns without
			// decoding cannot pass on the zero rows.
			decoded := NewFromInt(7)
			require.NoError(t, decoded.UnmarshalBSONValue(tc.typ, tc.data), "UnmarshalBSONValue must decode the BSON value")
			assert.Equal(t, tc.expected, decoded.String(), "UnmarshalBSONValue should decode the expected value")
		})
	}
	for _, tc := range []struct {
		name, expected string
		typ            byte
		data           []byte
	}{
		{name: "Decimal128 NaN", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "0000000000000000000000000000007c"), expected: "0"},
		{name: "Decimal128 infinity", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "00000000000000000000000000000078"), expected: "0"},
		{name: "Decimal128 negative infinity", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "000000000000000000000000000000f8"), expected: "0"},
		{name: "Decimal128 NaN with a payload", typ: bsonwire.TypeDecimal128, data: bsonDecimal128(t, "1200000000000000000000000000007e"), expected: "0"},
		{name: "invalid string", typ: bsonwire.TypeString, data: bsonwire.MarshalString("invalid"), expected: "0"},
		{name: "empty string", typ: bsonwire.TypeString, data: bsonwire.MarshalString(""), expected: "0"},
		{name: "quoted string", typ: bsonwire.TypeString, data: bsonwire.MarshalString(`"1.25"`), expected: "0"},
		{name: "string with surrounding spaces", typ: bsonwire.TypeString, data: bsonwire.MarshalString(" 1.25 "), expected: "0"},
		{name: "double NaN", typ: bsonwire.TypeDouble, data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(math.NaN())), expected: "7"},
		{name: "double infinity", typ: bsonwire.TypeDouble, data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(math.Inf(1))), expected: "7"},
		{name: "double negative infinity", typ: bsonwire.TypeDouble, data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(math.Inf(-1))), expected: "7"},
		{name: "short Decimal128", typ: bsonwire.TypeDecimal128, data: make([]byte, 15), expected: "7"},
		{name: "long Decimal128", typ: bsonwire.TypeDecimal128, data: make([]byte, 17), expected: "7"},
		{name: "short string", typ: bsonwire.TypeString, data: []byte{1, 0, 0, 0}, expected: "7"},
		{name: "four-byte double", typ: bsonwire.TypeDouble, data: make([]byte, 4), expected: "7"},
		{name: "short double", typ: bsonwire.TypeDouble, data: make([]byte, 7), expected: "7"},
		{name: "long double", typ: bsonwire.TypeDouble, data: make([]byte, 9), expected: "7"},
		{name: "short int32", typ: bsonwire.TypeInt32, data: make([]byte, 3), expected: "7"},
		{name: "long int32", typ: bsonwire.TypeInt32, data: make([]byte, 5), expected: "7"},
		{name: "eight-byte int32", typ: bsonwire.TypeInt32, data: make([]byte, 8), expected: "7"},
		{name: "four-byte int64", typ: bsonwire.TypeInt64, data: make([]byte, 4), expected: "7"},
		{name: "short int64", typ: bsonwire.TypeInt64, data: make([]byte, 7), expected: "7"},
		{name: "long int64", typ: bsonwire.TypeInt64, data: make([]byte, 9), expected: "7"},
		{name: "boolean", typ: 0x08, data: []byte{1}, expected: "7"},
		{name: "symbol", typ: bsonwire.TypeSymbol, data: bsonwire.MarshalString("1.25"), expected: "7"},
		{name: "document", typ: bsonwire.TypeEmbeddedDocument, data: bsonwire.MarshalDocument(), expected: "7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			decoded := NewFromInt(7)
			assert.Error(t, decoded.UnmarshalBSONValue(tc.typ, tc.data), "UnmarshalBSONValue should reject the BSON value")
			assert.Equal(t, tc.expected, decoded.String(), "UnmarshalBSONValue should leave the expected value after rejecting it")
		})
	}
}
