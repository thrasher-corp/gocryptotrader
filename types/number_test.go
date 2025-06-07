package types

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// TestNumberUnmarshalJSON asserts the following behaviour:
// * Literal numbers and quoted are valid
// * Anything else returns errInvalidNumberValue
func TestNumberUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var n Number

	err := n.UnmarshalJSON([]byte(`"0.00000001"`))
	assert.NoError(t, err, "Unmarshal should not error")
	assert.Equal(t, 1e-8, n.Float64(), "Float64() should return the correct value")

	err = n.UnmarshalJSON([]byte(`""`))
	assert.NoError(t, err, "Unmarshal should not error")
	assert.Zero(t, n.Float64(), "UnmarshalJSON should parse empty as 0")

	err = n.UnmarshalJSON([]byte(`1337.37`))
	assert.NoError(t, err, "Unmarshal should not error on number types")
	assert.Equal(t, 1337.37, n.Float64(), "UnmarshalJSON should handle raw numerics")

	// Invalid value checking
	for _, i := range []string{`"MEOW"`, `null`, `false`, `true`, `"1337.37`} {
		err = n.UnmarshalJSON([]byte(i))
		assert.ErrorIsf(t, err, errInvalidNumberValue, "UnmarshalJSON should error with invalid Value for %q", i)
	}
}

// TestNumberMarshalJSON asserts the following behaviour:
// 0 marshalls to quoted empty string
// Anything else marshalls as a quoted number
func TestNumberMarshalJSON(t *testing.T) {
	data, err := new(Number).MarshalJSON()
	assert.NoError(t, err, "MarshalJSON should not error")
	assert.Equal(t, `""`, string(data), "MarshalJSON should return the correct value")

	data, err = Number(1337.1337).MarshalJSON()
	assert.NoError(t, err, "MarshalJSON should not error")
	assert.Equal(t, `"1337.1337"`, string(data), "MarshalJSON should return the correct value")
}

// TestNumberFloat64 asserts Float64() returns a valid float64
func TestNumberFloat64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.04200064, Number(0.04200064).Float64(), "Float64() should return the correct value")
}

// TestNumberDecimal asserts Decimal() returns a valid decimal.Decimal
func TestNumberDecimal(t *testing.T) {
	t.Parallel()
	assert.Equal(t, decimal.NewFromFloat(0.04200064), Number(0.04200064).Decimal(), "Decimal() should return the correct value")
}

// TestNumberInt64 asserts Int64() returns a valid truncated int64
func TestNumberInt64(t *testing.T) {
	t.Parallel()
	assert.Equal(t, int64(42), Number(42.00000064).Int64(), "Int64() should return the correct truncated value")
	assert.Equal(t, int64(43), Number(43.99999964).Int64(), "Int64() should not round the number")
}

// BenchmarkNumberUnmarshalJSON provides a barebones benchmark of Unmarshaling a string value
// Ballpark: 42.78 ns/op        16 B/op          1 allocs/op
func BenchmarkNumberUnmarshalJSON(b *testing.B) {
	var n Number
	for b.Loop() {
		_ = n.UnmarshalJSON([]byte(`"0.04200074"`))
	}
}

// BenchmarkNumberMarshalJSON provides a barebones benchmark of Marshaling a string value
// Ballpark: 118.2 ns/op            56 B/op          3 allocs/op
func BenchmarkNumberMarshalJSON(b *testing.B) {
	for b.Loop() {
		_, _ = Number(1337.1337).MarshalJSON()
	}
}
