package types

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestPreciseNumberUnmarshalJSON verifies that both quoted and bare numeric
// JSON values are accepted, that the original string is preserved, and that
// invalid input is rejected.
func TestPreciseNumberUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var p PreciseNumber

	err := p.UnmarshalJSON([]byte(`"0.00000001"`))
	assert.NoError(t, err)
	assert.Equal(t, 1e-8, p.Float64())
	assert.Equal(t, "0.00000001", p.String())

	err = p.UnmarshalJSON([]byte(`""`))
	assert.NoError(t, err)
	assert.True(t, p.IsZero())

	err = p.UnmarshalJSON([]byte(`null`))
	assert.NoError(t, err)
	assert.True(t, p.IsZero())

	err = p.UnmarshalJSON([]byte(`1337.37`))
	assert.NoError(t, err)
	assert.Equal(t, 1337.37, p.Float64())
	assert.Equal(t, "1337.37", p.String())

	for _, in := range []string{`"MEOW"`, `false`, `true`, `"1337.37`} {
		err = p.UnmarshalJSON([]byte(in))
		assert.ErrorIsf(t, err, errInvalidPreciseNumberValue, "rejects invalid %q", in)
	}
}

// TestPreciseNumberMarshalJSON verifies the zero value emits an empty string
// (matching Number) and other values emit their preserved string verbatim.
func TestPreciseNumberMarshalJSON(t *testing.T) {
	t.Parallel()

	data, err := new(PreciseNumber).MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `""`, string(data))

	p, err := NewPreciseNumberFromString("1337.1337")
	assert.NoError(t, err)
	data, err = p.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"1337.1337"`, string(data))
}

// TestPreciseNumberRoundTrip verifies the marshal/unmarshal cycle preserves
// the exact original digit string — the key advantage over Number which
// reformats through float64 and loses trailing precision.
func TestPreciseNumberRoundTrip(t *testing.T) {
	t.Parallel()

	// 17 significant digits — beyond what float64 reliably round-trips
	// through ParseFloat/FormatFloat
	const high = "71428.12345678901234"

	var p PreciseNumber
	err := p.UnmarshalJSON([]byte(`"` + high + `"`))
	assert.NoError(t, err)

	data, err := p.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"`+high+`"`, string(data),
		"PreciseNumber must round-trip the original digit string")
}

// TestPreciseNumberDecimal verifies Decimal() returns the exact decimal value
// parsed from the original string, without going through float64.
func TestPreciseNumberDecimal(t *testing.T) {
	t.Parallel()

	var p PreciseNumber
	err := p.UnmarshalJSON([]byte(`"71428.12345678"`))
	assert.NoError(t, err)

	expected, _ := decimal.NewFromString("71428.12345678")
	assert.True(t, p.Decimal().Equal(expected),
		"Decimal() must reproduce the original value exactly; got %s", p.Decimal())

	// And the zero value
	var zero PreciseNumber
	assert.True(t, zero.Decimal().IsZero())
}

// TestPreciseNumberDecimalBeatsNumber demonstrates the precision benefit
// against the existing Number type: a value with more significant digits
// than float64 reliably preserves round-trips exactly through PreciseNumber
// but is silently truncated by Number.Decimal().
func TestPreciseNumberDecimalBeatsNumber(t *testing.T) {
	t.Parallel()

	// 17 digits — beyond float64's reliable round-trip range
	const wireValue = "71428.12345678901234"

	var n Number
	require := assert.New(t)
	require.NoError(n.UnmarshalJSON([]byte(`"` + wireValue + `"`)))

	var p PreciseNumber
	require.NoError(p.UnmarshalJSON([]byte(`"` + wireValue + `"`)))

	expected, _ := decimal.NewFromString(wireValue)

	// PreciseNumber preserves the wire value exactly.
	require.True(p.Decimal().Equal(expected),
		"PreciseNumber should equal the wire value exactly; got %s", p.Decimal())

	// Number routes through float64 and loses the trailing digits.
	require.False(n.Decimal().Equal(expected),
		"Number.Decimal() is expected to differ for high-precision values; got %s", n.Decimal())
}

// TestPreciseNumberInt64 verifies integer values parse without float
// round-trip, and that fractional values are truncated toward zero.
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
		assert.NoError(t, err)
		assert.Equalf(t, c.want, p.Int64(), "Int64() for %q", c.in)
	}

	assert.Zero(t, PreciseNumber{}.Int64(), "zero value Int64() returns 0")
}

// TestPreciseNumberFloat64 verifies Float64() returns the cached float.
func TestPreciseNumberFloat64(t *testing.T) {
	t.Parallel()
	p, err := NewPreciseNumberFromString("0.04200064")
	assert.NoError(t, err)
	assert.Equal(t, 0.04200064, p.Float64())
}

// TestNewPreciseNumberFromString verifies the constructor parses valid
// strings and rejects invalid ones.
func TestNewPreciseNumberFromString(t *testing.T) {
	t.Parallel()

	p, err := NewPreciseNumberFromString("1.5")
	assert.NoError(t, err)
	assert.Equal(t, "1.5", p.String())
	assert.Equal(t, 1.5, p.Float64())

	zero, err := NewPreciseNumberFromString("")
	assert.NoError(t, err)
	assert.True(t, zero.IsZero())

	_, err = NewPreciseNumberFromString("garbage")
	assert.ErrorIs(t, err, errInvalidPreciseNumberValue)
}

// BenchmarkPreciseNumberUnmarshalJSON measures the cost of UnmarshalJSON for
// a typical exchange string value.
func BenchmarkPreciseNumberUnmarshalJSON(b *testing.B) {
	var p PreciseNumber
	for b.Loop() {
		_ = p.UnmarshalJSON([]byte(`"0.04200074"`))
	}
}

// BenchmarkPreciseNumberDecimal measures the cost of Decimal() so we can
// compare against [BenchmarkNumberDecimalConversion] in number_test.go.
func BenchmarkPreciseNumberDecimal(b *testing.B) {
	p, _ := NewPreciseNumberFromString("0.04200074")
	for b.Loop() {
		_ = p.Decimal()
	}
}
