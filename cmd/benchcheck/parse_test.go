package main

import (
	"bufio"
	"math"
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
BenchmarkProcess-8   	 5571000	       211.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkSortAsksDecending-8   	  361266	      3556 ns/op	      24 B/op	       1 allocs/op
BenchmarkSortAsksDecending-8   	  361266	      3561 ns/op	      24 B/op	       1 allocs/op
BenchmarkSortAsksDecending-8   	  361266	      3550 ns/op	      24 B/op	       1 allocs/op
PASS
ok  	github.com/thrasher-corp/gocryptotrader/exchanges/orderbook	3.140s
goos: linux
goarch: amd64
pkg: github.com/thrasher-corp/gocryptotrader/currency
BenchmarkNewCode-8   	 1000000	      1050 ns/op	      16 B/op	       2 allocs/op
BenchmarkNewCode-8   	 1000000	      1049 ns/op	      16 B/op	       2 allocs/op
BenchmarkNewCode-8   	 1000000	      1052 ns/op	      16 B/op	       2 allocs/op
PASS
ok  	github.com/thrasher-corp/gocryptotrader/currency	1.200s
`

func TestParse(t *testing.T) {
	t.Parallel()
	run, err := Parse(strings.NewReader(sampleOutput))
	require.NoError(t, err, "Parse must not error")
	assert.Equal(t, "linux/amd64", run.Platform, "Platform should come from the goos and goarch headers")

	exp := map[string]*Samples{
		"exchanges/orderbook.BenchmarkProcess": {
			Pkg: "exchanges/orderbook", Name: "BenchmarkProcess", Bytes: []float64{0, 0, 0}, Allocs: []float64{0, 0, 0},
		},
		"exchanges/orderbook.BenchmarkSortAsksDecending": {
			Pkg: "exchanges/orderbook", Name: "BenchmarkSortAsksDecending", Bytes: []float64{24, 24, 24}, Allocs: []float64{1, 1, 1},
		},
		"currency.BenchmarkNewCode": {
			Pkg: "currency", Name: "BenchmarkNewCode", Bytes: []float64{16, 16, 16}, Allocs: []float64{2, 2, 2},
		},
	}
	assert.Equal(t, exp, run.Samples,
		"every -count sample should be kept under its package, with the module prefix and GOMAXPROCS suffix stripped")
	assert.Equal(t, map[string]bool{"exchanges/orderbook": true, "currency": true}, run.Finished,
		"each package whose output reached its ok line should be marked finished")
}

func TestParseWithoutPlatformHeaders(t *testing.T) {
	t.Parallel()
	run, err := Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n"))
	require.NoError(t, err, "Parse must not error")
	assert.Empty(t, run.Platform, "output without goos and goarch headers should leave the platform unknown")
}

func TestParseRejectsMixedPlatforms(t *testing.T) {
	t.Parallel()
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"goos: darwin\ngoarch: arm64\npkg: github.com/thrasher-corp/gocryptotrader/types\n"
	_, err := Parse(strings.NewReader(in))
	assert.ErrorIs(t, err, errMixedPlatforms, "output concatenated from two machines should be rejected")
}

func TestParseIgnoresNonResultLines(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"cpu: AMD Ryzen 9 5950X 16-Core Processor\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n" +
		"--- BENCH: BenchmarkNewCode-8\n" +
		"    code_test.go:12: logged from the benchmark\n" +
		"PASS\n" +
		"?   \tgithub.com/thrasher-corp/gocryptotrader/currency/sub\t[no test files]\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	require.Len(t, run.Samples, 1, "only the result line must be parsed")
	assert.Equal(t, []float64{2}, run.Samples["currency.BenchmarkNewCode"].Allocs, "the result line should be recorded")
	assert.Empty(t, run.Finished, "output without an ok line should finish no package")
}

func TestParseVerboseOutput(t *testing.T) {
	t.Parallel()
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/common/timeperiods\n" +
		"BenchmarkFind\n" +
		"BenchmarkFind/empty\n" +
		"BenchmarkFind/empty-4   \t       1\t    122100 ns/op\t  119456 B/op\t      13 allocs/op\n" +
		"PASS\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must accept the bare names -v announces each benchmark with")
	require.Contains(t, run.Samples, "common/timeperiods.BenchmarkFind/empty", "the result line must be parsed")
	assert.Equal(t, []float64{13}, run.Samples["common/timeperiods.BenchmarkFind/empty"].Allocs, "the result should be recorded")
	assert.Len(t, run.Samples, 1, "an announced name should not become a benchmark of its own")
}

func TestParseRejectsMalformedResults(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"BenchmarkNewCode-8   \t notanumber\t      1050 ns/op",
		"BenchmarkNewCode-8   \tsomething the benchmark printed",
		"BenchmarkNewCode-4 100 10 ns/op INVALID B/op 10 allocs/op",
		"BenchmarkNewCode-4 100 10 ns/op 16 B/op 2",
	} {
		in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" + line + "\n"
		_, err := Parse(strings.NewReader(in))
		assert.ErrorIsf(t, err, errMalformedResult, "%q should be rejected rather than silently dropping a sample", line)
	}
}

func TestParseRejectsFailedRun(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"--- FAIL: BenchmarkNewCode-8",
		"    --- FAIL: BenchmarkNewCode/sub-8",
		"BenchmarkNewCode-8   \t--- FAIL: BenchmarkNewCode-8",
		"FAIL",
		"FAIL\tgithub.com/thrasher-corp/gocryptotrader/currency\t1.200s",
	} {
		in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
			"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n" +
			line + "\n"
		_, err := Parse(strings.NewReader(in))
		assert.ErrorIsf(t, err, errRunFailed, "a run reporting %q should be rejected rather than read as a short one", line)
	}
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
		"a result with no pkg: header cannot be attributed and should not be silently accepted")
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

func TestParseAcceptsNegativeCustomMetrics(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\t      -3.000 delta/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "a negative custom metric must not fail the run, since b.ReportMetric allows one")
	assert.Equal(t, []float64{2}, run.Samples["currency.BenchmarkNewCode"].Allocs, "the gated metrics should still be recorded")
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
		"a bad measurement should fail the run even when another package parses cleanly")
}

func TestParseReportsReadErrors(t *testing.T) {
	t.Parallel()
	_, err := Parse(strings.NewReader(strings.Repeat("x", 2*1024*1024)))
	assert.ErrorIs(t, err, bufio.ErrTooLong, "a line too long to scan should fail the parse rather than end it early")
}

func TestParseNoResults(t *testing.T) {
	t.Parallel()
	run, err := Parse(strings.NewReader("PASS\nok  \tgithub.com/x/y\t0.1s\n"))
	require.NoError(t, err, "Parse must not error on benchmark-free output")
	assert.Empty(t, run.Samples, "output with no benchmarks should yield no samples")
}

func TestParseBenchLine(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		line    string
		name    string
		metrics map[string]float64
	}{
		{
			line:    "BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op",
			name:    "BenchmarkNewCode",
			metrics: map[string]float64{"ns/op": 1050, "B/op": 16, "allocs/op": 2},
		},
		{
			line:    "BenchmarkSort/case.1-4 10 5.5 ns/op 0 B/op 0 allocs/op 3 widgets/op",
			name:    "BenchmarkSort/case.1",
			metrics: map[string]float64{"ns/op": 5.5, "B/op": 0, "allocs/op": 0, "widgets/op": 3},
		},
		{line: "BenchmarkNewCode-8"},
		{line: "--- BENCH: BenchmarkNewCode-8 1 2 ns/op"},
		{line: "BenchmarkNewCode-8 notanumber 1050 ns/op"},
		{line: "BenchmarkNewCode-8 100 1050 ns/op 16"},
		{line: "BenchmarkNewCode-8 100 ten ns/op"},
	} {
		name, metrics, ok := parseBenchLine(tc.line)
		if tc.name == "" {
			assert.Falsef(t, ok, "parseBenchLine should reject %q", tc.line)
			continue
		}
		require.Truef(t, ok, "parseBenchLine must accept %q", tc.line)
		assert.Equalf(t, tc.name, name, "parseBenchLine should name %q", tc.line)
		assert.Equalf(t, tc.metrics, metrics, "parseBenchLine should read every metric of %q", tc.line)
	}

	_, metrics, ok := parseBenchLine("BenchmarkNewCode-8 100 1050 ns/op 16 B/op -2 allocs/op")
	require.True(t, ok, "a negative measurement must still be returned, so Parse can reject it")
	assert.True(t, math.IsNaN(metrics["allocs/op"]), "a negative measurement should be marked NaN")
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
	for _, tc := range []struct {
		samples []float64
		exp     float64
	}{
		{samples: []float64{3}, exp: 3},
		{samples: []float64{3, 1, 2}, exp: 2},
		{samples: []float64{10, 10, 10, 10, 20, 20, 20}, exp: 10},
		{samples: []float64{20, 10, 20, 10, 20, 10, 20}, exp: 20},
	} {
		got, err := median(tc.samples)
		require.NoErrorf(t, err, "median must accept %v", tc.samples)
		assert.Equalf(t, tc.exp, got, "median should return the middle of %v", tc.samples)
	}

	for _, even := range [][]float64{nil, {1, 2}, {1, 2, 3, 4, 5, 6}} {
		_, err := median(even)
		assert.ErrorIsf(t, err, errEvenSampleCount, "median should refuse %d samples, which have no middle", len(even))
	}
}

func TestMedianDoesNotReorderSamples(t *testing.T) {
	t.Parallel()
	samples := []float64{3, 1, 2}
	_, err := median(samples)
	require.NoError(t, err, "median must not error")
	assert.Equal(t, []float64{3, 1, 2}, samples, "median should sort a copy, not the caller's samples")
}

func TestSampleCountMismatches(t *testing.T) {
	t.Parallel()
	full := func(n int) *Samples {
		return &Samples{Bytes: make([]float64, n), Allocs: make([]float64, n)}
	}
	shortBytes := &Samples{Bytes: make([]float64, 6), Allocs: make([]float64, 7)}
	shortAllocs := &Samples{Bytes: make([]float64, 7), Allocs: make([]float64, 6)}

	for _, tc := range []struct {
		name    string
		samples map[string]*Samples
		exp     []string
	}{
		{name: "complete", samples: map[string]*Samples{"a.BenchmarkOne": full(7)}},
		{name: "short B/op", samples: map[string]*Samples{"a.BenchmarkOne": shortBytes}, exp: []string{"a.BenchmarkOne"}},
		{name: "short allocs/op", samples: map[string]*Samples{"a.BenchmarkOne": shortAllocs}, exp: []string{"a.BenchmarkOne"}},
		{name: "two -cpu values doubling the count", samples: map[string]*Samples{"a.BenchmarkOne": full(14)}, exp: []string{"a.BenchmarkOne"}},
		{
			name:    "reported in key order",
			samples: map[string]*Samples{"b.BenchmarkTwo": full(6), "a.BenchmarkOne": full(5), "c.BenchmarkThree": full(7)},
			exp:     []string{"a.BenchmarkOne", "b.BenchmarkTwo"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, SampleCountMismatches(tc.samples, 7), "SampleCountMismatches should report the expected benchmarks")
		})
	}
}

func TestDescribeSamples(t *testing.T) {
	t.Parallel()
	s := &Samples{Bytes: make([]float64, 6), Allocs: make([]float64, 5)}
	assert.Equal(t, "B/op=6 allocs/op=5, expected 7", describeSamples(s, 7), "describeSamples should give both counts and the expectation")
}

func TestSummarise(t *testing.T) {
	t.Parallel()
	results, err := Summarise(map[string]*Samples{
		"currency.BenchmarkNewCode": {Pkg: "currency", Name: "BenchmarkNewCode", Bytes: []float64{16, 32, 16}, Allocs: []float64{3, 2, 2}},
	})
	require.NoError(t, err, "Summarise must not error")
	exp := map[string]*Result{
		"currency.BenchmarkNewCode": {Pkg: "currency", Name: "BenchmarkNewCode", Allocs: 2, Bytes: 16},
	}
	assert.Equal(t, exp, results, "Summarise should reduce each metric to its median")

	_, err = Summarise(map[string]*Samples{"a.BenchmarkOne": {Bytes: []float64{1}, Allocs: []float64{1, 2}}})
	assert.ErrorIs(t, err, errEvenSampleCount, "an even allocs/op count should be refused")
	_, err = Summarise(map[string]*Samples{"a.BenchmarkOne": {Bytes: []float64{1, 2}, Allocs: []float64{1}}})
	assert.ErrorIs(t, err, errEvenSampleCount, "an even B/op count should be refused")
}

func TestKeys(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "exchanges/orderbook.BenchmarkProcess", (&Samples{Pkg: "exchanges/orderbook", Name: "BenchmarkProcess"}).Key(),
		"Samples.Key should join package and name")
	assert.Equal(t, "exchanges/orderbook.BenchmarkProcess", (&Result{Pkg: "exchanges/orderbook", Name: "BenchmarkProcess"}).Key(),
		"Result.Key should join package and name")
}
