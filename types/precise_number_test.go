package types

import (
	"strconv"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// highPrecisionValue is a decimal string with 19 significant digits — beyond
// what float64 reliably round-trips through ParseFloat/FormatFloat. Shared by
// the round-trip and head-to-head precision tests.
const highPrecisionValue = "71428.12345678901234"

// TestHighPrecisionValueSanity asserts the shared test constant genuinely
// exceeds float64's round-trip precision, so the tests that rely on it are
// exercising the property they claim to. If a float64 round-trip reproduced
// the string exactly, those tests would pass trivially and prove nothing.
func TestHighPrecisionValueSanity(t *testing.T) {
	t.Parallel()
	f, err := strconv.ParseFloat(highPrecisionValue, 64)
	require.NoError(t, err, "ParseFloat must not error")
	roundTripped := strconv.FormatFloat(f, 'f', -1, 64)
	assert.NotEqual(t, highPrecisionValue, roundTripped,
		"highPrecisionValue should lose precision through a float64 round-trip")
}

// TestPreciseNumberUnmarshalJSON verifies that both quoted and bare numeric
// JSON values are accepted, that the original string is preserved, and that
// invalid input is rejected on every branch of the validator.
func TestPreciseNumberUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var p PreciseNumber

	require.NoError(t, p.UnmarshalJSON([]byte(`"0.00000001"`)), "UnmarshalJSON must not error")
	assert.Equal(t, 1e-8, p.Float64(), "Float64 should return the parsed value")
	assert.Equal(t, "0.00000001", p.String(), "String should return the original digits")

	require.NoError(t, p.UnmarshalJSON([]byte(`""`)), "UnmarshalJSON must not error")
	assert.True(t, p.IsZero(), "empty quoted string should parse to the zero value")

	require.NoError(t, p.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must not error")
	assert.True(t, p.IsZero(), "null should parse to the zero value")

	require.NoError(t, p.UnmarshalJSON([]byte(``)), "UnmarshalJSON must not error")
	assert.True(t, p.IsZero(), "empty payload should parse to the zero value")

	require.NoError(t, p.UnmarshalJSON([]byte(`1337.37`)), "UnmarshalJSON must not error")
	assert.Equal(t, 1337.37, p.Float64(), "Float64 should return the parsed bare number")
	assert.Equal(t, "1337.37", p.String(), "String should return the original bare number")

	require.NoError(t, p.UnmarshalJSON([]byte(`-7.25`)), "leading minus must be accepted")
	assert.Equal(t, "-7.25", p.String(), "String should preserve the leading minus")

	for _, tc := range []struct{ in, reason string }{
		{`"MEOW"`, "quoted non-numeric"},
		{`false`, "false literal"},
		{`true`, "true literal"},
		{`"1337.37`, "missing closing quote"},
		{`abc`, "bare token with non-digit, non-minus first byte"},
		{`+5`, "leading '+' which is not in the accepted set"},
		{`0x1f`, "hexadecimal"},
	} {
		err := p.UnmarshalJSON([]byte(tc.in))
		assert.ErrorIsf(t, err, errInvalidPreciseNumberValue, "%s should be rejected: %q", tc.reason, tc.in)
	}

	require.NoError(t, p.UnmarshalJSON([]byte(`"1e10"`)), "scientific notation must be accepted")
	assert.Equal(t, 1e10, p.Float64(), "Float64 should return the parsed scientific-notation value")
}

// TestPreciseNumberMarshalJSON verifies the zero value emits an empty string
// (matching Number) and other values emit their preserved string verbatim.
func TestPreciseNumberMarshalJSON(t *testing.T) {
	t.Parallel()

	data, err := new(PreciseNumber).MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, `""`, string(data), "zero value should marshal to an empty quoted string")

	p, err := NewPreciseNumberFromString("1337.1337")
	require.NoError(t, err, "NewPreciseNumberFromString must not error")
	data, err = p.MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, `"1337.1337"`, string(data), "value should marshal to its preserved string")
}

// TestPreciseNumberRoundTrip verifies the marshal/unmarshal cycle preserves
// the exact original digit string — the key advantage over Number which
// reformats through float64 and loses trailing precision.
func TestPreciseNumberRoundTrip(t *testing.T) {
	t.Parallel()

	var p PreciseNumber
	require.NoError(t, p.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)), "UnmarshalJSON must not error")

	data, err := p.MarshalJSON()
	require.NoError(t, err, "MarshalJSON must not error")
	assert.Equal(t, `"`+highPrecisionValue+`"`, string(data),
		"PreciseNumber should round-trip the original digit string")
}

// TestPreciseNumberDecimal verifies Decimal() returns the exact decimal value
// parsed from the original string, without going through float64.
func TestPreciseNumberDecimal(t *testing.T) {
	t.Parallel()

	var p PreciseNumber
	require.NoError(t, p.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)), "UnmarshalJSON must not error")

	expected, err := decimal.NewFromString(highPrecisionValue)
	require.NoError(t, err, "decimal.NewFromString must not error")
	assert.Truef(t, p.Decimal().Equal(expected),
		"Decimal should reproduce the original high-precision value exactly without a float64 round-trip; got %s", p.Decimal())

	var zero PreciseNumber
	assert.True(t, zero.Decimal().IsZero(), "zero value Decimal should return decimal.Zero")
}

// TestPreciseNumberDecimalBeatsNumber demonstrates the precision benefit
// against the existing Number type: a value with more significant digits
// than float64 reliably preserves round-trips exactly through PreciseNumber
// but is silently truncated by Number.Decimal().
func TestPreciseNumberDecimalBeatsNumber(t *testing.T) {
	t.Parallel()

	var n Number
	require.NoError(t, n.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)), "Number.UnmarshalJSON must not error")

	var p PreciseNumber
	require.NoError(t, p.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)), "PreciseNumber.UnmarshalJSON must not error")

	expected, err := decimal.NewFromString(highPrecisionValue)
	require.NoError(t, err, "decimal.NewFromString must not error")

	assert.Truef(t, p.Decimal().Equal(expected),
		"PreciseNumber.Decimal should equal the wire value exactly; got %s", p.Decimal())

	assert.Falsef(t, n.Decimal().Equal(expected),
		"Number.Decimal should differ for high-precision values; got %s", n.Decimal())
}

// TestPreciseNumberInt64 verifies integer values parse without float
// round-trip, that fractional values are truncated toward zero, and that
// the zero value returns 0.
func TestPreciseNumberInt64(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in   string
		want int64
	}{
		{"42", 42},
		{"42.00000064", 42},
		{"43.99999964", 43},
		{"-7", -7},
		{"-7.9", -7},
		{"0", 0},
	} {
		p, err := NewPreciseNumberFromString(tc.in)
		require.NoErrorf(t, err, "NewPreciseNumberFromString must not error for %q", tc.in)
		assert.Equalf(t, tc.want, p.Int64(), "Int64 should truncate %q toward zero", tc.in)
	}

	assert.Zero(t, PreciseNumber{}.Int64(), "zero value Int64 should return 0")
}

// TestPreciseNumberFloat64 verifies Float64() returns the cached float.
func TestPreciseNumberFloat64(t *testing.T) {
	t.Parallel()
	p, err := NewPreciseNumberFromString("0.04200064")
	require.NoError(t, err, "NewPreciseNumberFromString must not error")
	assert.Equal(t, 0.04200064, p.Float64(), "Float64 should return the cached float")
}

// TestPreciseNumberString verifies the original digits are preserved by
// String() and that the zero value returns "0".
func TestPreciseNumberString(t *testing.T) {
	t.Parallel()

	p, err := NewPreciseNumberFromString("3.14159")
	require.NoError(t, err, "NewPreciseNumberFromString must not error")
	assert.Equal(t, "3.14159", p.String(), "String should preserve the original digits")

	assert.Equal(t, "0", PreciseNumber{}.String(), "zero value String should return 0")
}

// TestNewPreciseNumberFromString verifies the constructor parses valid strings
// and rejects empty input, non-numeric input, and magnitudes that overflow
// float64.
func TestNewPreciseNumberFromString(t *testing.T) {
	t.Parallel()

	p, err := NewPreciseNumberFromString("1.5")
	require.NoError(t, err, "NewPreciseNumberFromString must not error")
	assert.Equal(t, "1.5", p.String(), "String should preserve the input")
	assert.Equal(t, 1.5, p.Float64(), "Float64 should return the parsed value")

	_, err = NewPreciseNumberFromString("")
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue, "empty input should be rejected")

	_, err = NewPreciseNumberFromString("garbage")
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue, "non-numeric input should be rejected")

	_, err = NewPreciseNumberFromString("1" + strings.Repeat("0", 400))
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue, "a magnitude that overflows float64 should be rejected")
}

// TestPreciseNumberIsZero pins the contract that IsZero is true only for the
// uninitialized struct, never for an explicit "0" — the property that
// protects accounting fields from omitempty data loss.
func TestPreciseNumberIsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, PreciseNumber{}.IsZero(), "uninitialized value should be IsZero")

	explicit, err := NewPreciseNumberFromString("0")
	require.NoError(t, err, "NewPreciseNumberFromString must not error")
	assert.False(t, explicit.IsZero(), "explicitly set \"0\" should not be IsZero")

	var fromNull PreciseNumber
	require.NoError(t, fromNull.UnmarshalJSON([]byte(`null`)), "UnmarshalJSON must not error")
	assert.True(t, fromNull.IsZero(), "null should parse to IsZero")

	var fromEmpty PreciseNumber
	require.NoError(t, fromEmpty.UnmarshalJSON([]byte(`""`)), "UnmarshalJSON must not error")
	assert.True(t, fromEmpty.IsZero(), "empty quoted string should parse to IsZero")
}

// BenchmarkPreciseNumberUnmarshalJSON measures the cost of UnmarshalJSON for
// a typical exchange string value.
// Ballpark: 271.2 ns/op        392 B/op         16 allocs/op
func BenchmarkPreciseNumberUnmarshalJSON(b *testing.B) {
	var p PreciseNumber
	for b.Loop() {
		if err := p.UnmarshalJSON([]byte(`"0.04200074"`)); err != nil {
			require.NoError(b, err, "UnmarshalJSON must not error")
		}
	}
}

// BenchmarkPreciseNumberDecimal measures the cost of Decimal().
// Ballpark: 55.11 ns/op         56 B/op          3 allocs/op
func BenchmarkPreciseNumberDecimal(b *testing.B) {
	p, err := NewPreciseNumberFromString("0.04200074")
	require.NoError(b, err, "NewPreciseNumberFromString must not error")
	for b.Loop() {
		if !p.Decimal().IsPositive() {
			b.Fatal("unexpected non-positive decimal")
		}
	}
}
