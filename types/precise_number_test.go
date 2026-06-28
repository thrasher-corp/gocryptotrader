package types

import (
	"strconv"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// highPrecisionValue is a decimal string with 17 significant digits — beyond
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
	require.NoError(t, err)
	roundTripped := strconv.FormatFloat(f, 'f', -1, 64)
	assert.NotEqual(t, highPrecisionValue, roundTripped,
		"highPrecisionValue must lose precision through float64 for the precision tests to be meaningful")
}

// TestPreciseNumberUnmarshalJSON verifies that both quoted and bare numeric
// JSON values are accepted, that the original string is preserved, and that
// invalid input is rejected on every branch of the validator.
func TestPreciseNumberUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var p PreciseNumber

	require.NoError(t, p.UnmarshalJSON([]byte(`"0.00000001"`)))
	assert.Equal(t, 1e-8, p.Float64())
	assert.Equal(t, "0.00000001", p.String())

	require.NoError(t, p.UnmarshalJSON([]byte(`""`)))
	assert.True(t, p.IsZero(), "empty quoted string parses to zero value")

	require.NoError(t, p.UnmarshalJSON([]byte(`null`)))
	assert.True(t, p.IsZero(), "null parses to zero value")

	require.NoError(t, p.UnmarshalJSON([]byte(``)))
	assert.True(t, p.IsZero(), "empty payload parses to zero value")

	require.NoError(t, p.UnmarshalJSON([]byte(`1337.37`)))
	assert.Equal(t, 1337.37, p.Float64())
	assert.Equal(t, "1337.37", p.String())

	require.NoError(t, p.UnmarshalJSON([]byte(`-7.25`)), "leading minus accepted")
	assert.Equal(t, "-7.25", p.String())

	// Each entry exercises a distinct invalid-input branch in UnmarshalJSON.
	for _, tc := range []struct{ in, reason string }{
		{`"MEOW"`, "quoted non-numeric rejected by parsePreciseNumber"},
		{`false`, "false literal rejected"},
		{`true`, "true literal rejected"},
		{`"1337.37`, "missing closing quote rejected"},
		{`abc`, "bare token with non-digit, non-minus first byte rejected"},
		{`+5`, "leading '+' is not in the accepted set"},
		{`0x1f`, "hexadecimal rejected by decimal.NewFromString"},
	} {
		err := p.UnmarshalJSON([]byte(tc.in))
		assert.ErrorIsf(t, err, errInvalidPreciseNumberValue, "%s: %q", tc.reason, tc.in)
	}

	require.NoError(t, p.UnmarshalJSON([]byte(`"1e10"`)),
		"scientific notation is legitimate base-10 accepted by decimal.NewFromString")
	assert.Equal(t, 1e10, p.Float64())
}

// TestPreciseNumberMarshalJSON verifies the zero value emits an empty string
// (matching Number) and other values emit their preserved string verbatim.
func TestPreciseNumberMarshalJSON(t *testing.T) {
	t.Parallel()

	data, err := new(PreciseNumber).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `""`, string(data))

	p, err := NewPreciseNumberFromString("1337.1337")
	require.NoError(t, err)
	data, err = p.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"1337.1337"`, string(data))
}

// TestPreciseNumberRoundTrip verifies the marshal/unmarshal cycle preserves
// the exact original digit string — the key advantage over Number which
// reformats through float64 and loses trailing precision.
func TestPreciseNumberRoundTrip(t *testing.T) {
	t.Parallel()

	var p PreciseNumber
	require.NoError(t, p.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)))

	data, err := p.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"`+highPrecisionValue+`"`, string(data),
		"PreciseNumber must round-trip the original digit string")
}

// TestPreciseNumberDecimal verifies Decimal() returns the exact decimal value
// parsed from the original string, without going through float64.
func TestPreciseNumberDecimal(t *testing.T) {
	t.Parallel()

	var p PreciseNumber
	require.NoError(t, p.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)))

	expected, err := decimal.NewFromString(highPrecisionValue)
	require.NoError(t, err)
	assert.True(t, p.Decimal().Equal(expected),
		"Decimal() must reproduce the original high-precision value exactly without a float64 round-trip; got %s", p.Decimal())

	var zero PreciseNumber
	assert.True(t, zero.Decimal().IsZero(), "zero value Decimal() returns decimal.Zero")
}

// TestPreciseNumberDecimalBeatsNumber demonstrates the precision benefit
// against the existing Number type: a value with more significant digits
// than float64 reliably preserves round-trips exactly through PreciseNumber
// but is silently truncated by Number.Decimal().
func TestPreciseNumberDecimalBeatsNumber(t *testing.T) {
	t.Parallel()

	var n Number
	require.NoError(t, n.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)))

	var p PreciseNumber
	require.NoError(t, p.UnmarshalJSON([]byte(`"`+highPrecisionValue+`"`)))

	expected, err := decimal.NewFromString(highPrecisionValue)
	require.NoError(t, err)

	assert.True(t, p.Decimal().Equal(expected),
		"PreciseNumber should equal the wire value exactly; got %s", p.Decimal())

	assert.False(t, n.Decimal().Equal(expected),
		"Number.Decimal() is expected to differ for high-precision values; got %s", n.Decimal())
}

// TestPreciseNumberInt64 verifies integer values parse without float
// round-trip, that fractional values are truncated toward zero, and that
// the zero value returns 0.
func TestPreciseNumberInt64(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want int64
	}{
		{"42", 42},
		{"42.00000064", 42},
		{"43.99999964", 43},
		{"-7", -7},
		{"-7.9", -7},
		{"0", 0},
	}
	for _, c := range cases {
		p, err := NewPreciseNumberFromString(c.in)
		require.NoErrorf(t, err, "constructor for %q", c.in)
		assert.Equalf(t, c.want, p.Int64(), "Int64() for %q", c.in)
	}

	assert.Zero(t, PreciseNumber{}.Int64(), "zero value Int64() returns 0")
}

// TestPreciseNumberFloat64 verifies Float64() returns the cached float.
func TestPreciseNumberFloat64(t *testing.T) {
	t.Parallel()
	p, err := NewPreciseNumberFromString("0.04200064")
	require.NoError(t, err)
	assert.Equal(t, 0.04200064, p.Float64())
}

// TestPreciseNumberString verifies the original digits are preserved by
// String() and that the zero value returns "0".
func TestPreciseNumberString(t *testing.T) {
	t.Parallel()

	p, err := NewPreciseNumberFromString("3.14159")
	require.NoError(t, err)
	assert.Equal(t, "3.14159", p.String())

	assert.Equal(t, "0", PreciseNumber{}.String(), "zero value formats as 0")
}

// TestNewPreciseNumberFromString verifies the constructor parses valid
// strings, rejects empty input and invalid values, and rejects non-finite
// magnitudes that overflow float64.
func TestNewPreciseNumberFromString(t *testing.T) {
	t.Parallel()

	p, err := NewPreciseNumberFromString("1.5")
	require.NoError(t, err)
	assert.Equal(t, "1.5", p.String())
	assert.Equal(t, 1.5, p.Float64())

	_, err = NewPreciseNumberFromString("")
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue,
		"empty constructor input must error rather than return a silent zero value; callers wanting zero can use PreciseNumber{}")

	_, err = NewPreciseNumberFromString("garbage")
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue,
		"non-numeric input must be rejected")

	_, err = NewPreciseNumberFromString("1" + strings.Repeat("0", 400))
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue,
		"a magnitude that overflows float64 to Inf must be rejected so the cached val stays finite")
}

// TestPreciseNumberIsZero pins the contract that IsZero is true only for the
// uninitialized struct, never for an explicit "0" — the property that
// protects accounting fields from omitempty data loss.
func TestPreciseNumberIsZero(t *testing.T) {
	t.Parallel()

	assert.True(t, PreciseNumber{}.IsZero(), "uninitialized value is zero")

	explicit, err := NewPreciseNumberFromString("0")
	require.NoError(t, err)
	assert.False(t, explicit.IsZero(), "explicit \"0\" is not IsZero")

	var fromNull PreciseNumber
	require.NoError(t, fromNull.UnmarshalJSON([]byte(`null`)))
	assert.True(t, fromNull.IsZero(), "null parses to IsZero")

	var fromEmpty PreciseNumber
	require.NoError(t, fromEmpty.UnmarshalJSON([]byte(`""`)))
	assert.True(t, fromEmpty.IsZero(), "empty quoted string parses to IsZero")
}

// BenchmarkPreciseNumberUnmarshalJSON measures the cost of UnmarshalJSON for
// a typical exchange string value.
// Ballpark: 271.2 ns/op        392 B/op         16 allocs/op
func BenchmarkPreciseNumberUnmarshalJSON(b *testing.B) {
	var p PreciseNumber
	for b.Loop() {
		if err := p.UnmarshalJSON([]byte(`"0.04200074"`)); err != nil {
			require.NoError(b, err)
		}
	}
}

// BenchmarkPreciseNumberDecimal measures the cost of Decimal().
// Ballpark: 55.11 ns/op         56 B/op          3 allocs/op
func BenchmarkPreciseNumberDecimal(b *testing.B) {
	p, err := NewPreciseNumberFromString("0.04200074")
	require.NoError(b, err)
	for b.Loop() {
		if !p.Decimal().IsPositive() {
			b.Fatal("unexpected non-positive decimal")
		}
	}
}
