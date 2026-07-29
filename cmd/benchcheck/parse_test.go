package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleOutput = `goos: linux
goarch: amd64
pkg: github.com/thrasher-corp/gocryptotrader/exchanges/orderbook
cpu: AMD Ryzen 9 5950X 16-Core Processor
BenchmarkProcess-8   	 5572401	       210.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkProcess-8   	 5570000	       213.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkSortAsksDecending-8   	  361266	      3556 ns/op	      24 B/op	       1 allocs/op
PASS
ok  	github.com/thrasher-corp/gocryptotrader/exchanges/orderbook	3.140s
goos: linux
goarch: amd64
pkg: github.com/thrasher-corp/gocryptotrader/currency
BenchmarkNewCode-8   	 1000000	      1050 ns/op	      16 B/op	       2 allocs/op
PASS
ok  	github.com/thrasher-corp/gocryptotrader/currency	1.200s
`

func TestParse(t *testing.T) {
	t.Parallel()
	results, err := Parse(strings.NewReader(sampleOutput))
	require.NoError(t, err, "Parse must not error")
	require.Len(t, results, 3, "Parse must return one result per benchmark")

	proc := results["exchanges/orderbook.BenchmarkProcess"]
	require.NotNil(t, proc, "BenchmarkProcess must be present")
	assert.Equal(t, "exchanges/orderbook", proc.Pkg, "Pkg should have the module prefix stripped")
	assert.Equal(t, "BenchmarkProcess", proc.Name, "Name should have the GOMAXPROCS suffix stripped")
	assert.Equal(t, []float64{210.9, 213.1}, proc.NS, "every -count sample should be retained")
	assert.Equal(t, []float64{0, 0}, proc.Bytes, "B/op samples should be retained")
	assert.Equal(t, []float64{0, 0}, proc.Allocs, "allocs/op samples should be retained")

	code := results["currency.BenchmarkNewCode"]
	require.NotNil(t, code, "results must be attributed to the package of the preceding pkg: header")
	assert.Equal(t, "currency", code.Pkg, "Pkg should track the most recent pkg: header")
}

func TestParseIgnoresNonResultLines(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"--- FAIL: BenchmarkNewCode\n" +
		"BenchmarkNewCode-8\n" +
		"BenchmarkNewCode-8   \t notanumber\t      1050 ns/op\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n" +
		"FAIL\n"
	results, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	require.Len(t, results, 1, "only the well-formed result line must be parsed")
	assert.Equal(t, []float64{2}, results["currency.BenchmarkNewCode"].Allocs, "the well-formed line should be recorded")
}

func TestParseRequiresMemoryMetrics(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\n"
	_, err := Parse(strings.NewReader(in))
	assert.ErrorIs(t, err, errNoMemoryMetrics,
		"a result without -benchmem should error rather than silently leave the gated metrics empty")
}

func TestParseRequiresPackageContext(t *testing.T) {
	t.Parallel()
	in := "BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n"
	_, err := Parse(strings.NewReader(in))
	assert.ErrorIs(t, err, errNoPackageContext,
		"a result with no pkg: header cannot be attributed and must not be silently accepted")
}

func TestParseRejectsNonFiniteAndNegative(t *testing.T) {
	t.Parallel()
	for _, bad := range []string{"NaN", "+Inf", "-1"} {
		in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
			"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       " + bad + " allocs/op\n"
		_, err := Parse(strings.NewReader(in))
		assert.ErrorIsf(t, err, errBadMeasurement, "a %s measurement should be rejected", bad)
	}
}

func TestParseBadMeasurementDoesNotVanish(t *testing.T) {
	t.Parallel()
	// Skipping the line instead of failing removes its package from the results entirely. With any
	// other package still parsing, the run stays non-empty and the gate reports success.
	in := "pkg: github.com/thrasher-corp/gocryptotrader/common\n" +
		"BenchmarkCounter-4   \t 100\t 10 ns/op\t 0 B/op\t NaN allocs/op\n" +
		"pkg: github.com/thrasher-corp/gocryptotrader/exchanges/okx\n" +
		"BenchmarkMessageID-4   \t 100\t 175 ns/op\t 48 B/op\t 2 allocs/op\n"
	_, err := Parse(strings.NewReader(in))
	assert.ErrorIs(t, err, errBadMeasurement,
		"a bad measurement must fail the run even when another package parses cleanly")
}

func TestParseNoResults(t *testing.T) {
	t.Parallel()
	results, err := Parse(strings.NewReader("PASS\nok  \tgithub.com/x/y\t0.1s\n"))
	require.NoError(t, err, "Parse must not error on benchmark-free output")
	assert.Empty(t, results, "output with no benchmarks should yield no results")
}

func TestTrimProcSuffix(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ in, exp string }{
		{"BenchmarkProcess-8", "BenchmarkProcess"},
		{"BenchmarkProcess-128", "BenchmarkProcess"},
		{"BenchmarkProcess", "BenchmarkProcess"},
		{"BenchmarkUpdateInsertByID_asks-8", "BenchmarkUpdateInsertByID_asks"},
		{"BenchmarkSort-Ascending", "BenchmarkSort-Ascending"},
	} {
		assert.Equalf(t, tc.exp, trimProcSuffix(tc.in), "trimProcSuffix should handle %s", tc.in)
	}
}

func TestMedian(t *testing.T) {
	t.Parallel()
	assert.Zero(t, median(nil), "median of no samples should be zero")
	assert.Equal(t, 3.0, median([]float64{3}), "median of one sample should be that sample")
	assert.Equal(t, 2.0, median([]float64{3, 1, 2}), "median should ignore sample order")
	assert.Equal(t, 2.0, median([]float64{1, 2, 3, 4}), "an even count should take the lower middle to stay whole")
}

func TestResultKey(t *testing.T) {
	t.Parallel()
	r := &Result{Pkg: "exchanges/orderbook", Name: "BenchmarkProcess"}
	assert.Equal(t, "exchanges/orderbook.BenchmarkProcess", r.Key(), "Key should join package and name")
}
