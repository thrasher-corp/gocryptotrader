package decimal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	benchmarkDecimalSink Decimal
	benchmarkBoolSink    bool
	benchmarkFloatSink   float64
	benchmarkStringSink  string
)

func BenchmarkNewFromInt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkDecimalSink = NewFromInt(123456789)
	}
}

func BenchmarkMustFromFloat(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchmarkDecimalSink = MustFromFloat(12345.6789012345)
	}
}

func BenchmarkNewFromString(b *testing.B) {
	var err error
	b.ReportAllocs()
	for b.Loop() {
		benchmarkDecimalSink, err = NewFromString("12345.6789012345")
	}
	require.NoError(b, err, "NewFromString must parse benchmark input")
}

func BenchmarkDecimalAdd(b *testing.B) {
	left := MustFromString("12345.6789012345")
	right := MustFromString("98765.4321098765")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkDecimalSink = left.Add(right)
	}
}

func BenchmarkDecimalMul(b *testing.B) {
	left := MustFromString("12345.6789012345")
	right := MustFromString("1.000123456789")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkDecimalSink = left.Mul(right)
	}
}

func BenchmarkDecimalDiv(b *testing.B) {
	left := MustFromString("12345.6789012345")
	right := MustFromString("1.000123456789")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkDecimalSink = left.Div(right)
	}
}

func BenchmarkDecimalEqual(b *testing.B) {
	left := MustFromString("12345.6789012345")
	right := MustFromString("12345.6789012345")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkBoolSink = left.Equal(right)
	}
}

func BenchmarkDecimalString(b *testing.B) {
	value := MustFromString("12345.6789012345")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkStringSink = value.String()
	}
}

func BenchmarkDecimalInexactFloat64(b *testing.B) {
	value := MustFromString("12345.6789012345")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkFloatSink = value.InexactFloat64()
	}
}

func BenchmarkDecimalFloat64(b *testing.B) {
	value := MustFromString("12345.6789012345")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		benchmarkFloatSink, benchmarkBoolSink = value.Float64()
	}
}

func BenchmarkDecimalUnmarshalJSON(b *testing.B) {
	input := []byte(`"12345.6789012345"`)
	var (
		result Decimal
		err    error
	)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		err = result.UnmarshalJSON(input)
	}
	benchmarkDecimalSink = result
	require.NoError(b, err, "UnmarshalJSON must parse benchmark input")
}
