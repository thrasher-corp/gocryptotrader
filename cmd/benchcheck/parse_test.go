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
BenchmarkProcess
BenchmarkProcess-8   	 5572401	       210.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkProcess-8   	 5570000	       213.1 ns/op	       0 B/op	       0 allocs/op
BenchmarkProcess-8   	 5571000	       211.4 ns/op	       0 B/op	       0 allocs/op
BenchmarkSortAsksDescending
BenchmarkSortAsksDescending-8   	  361266	      3556 ns/op	      24 B/op	       1 allocs/op
BenchmarkSortAsksDescending-8   	  361266	      3561 ns/op	      24 B/op	       1 allocs/op
BenchmarkSortAsksDescending-8   	  361266	      3550 ns/op	      24 B/op	       1 allocs/op
PASS
ok  	github.com/thrasher-corp/gocryptotrader/exchanges/orderbook	3.140s
goos: linux
goarch: amd64
pkg: github.com/thrasher-corp/gocryptotrader/currency
BenchmarkNewCode
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
			Pkg: "exchanges/orderbook", Name: "BenchmarkProcess", NS: []float64{210.9, 213.1, 211.4},
			Bytes: []float64{0, 0, 0}, Allocs: []float64{0, 0, 0},
		},
		"exchanges/orderbook.BenchmarkSortAsksDescending": {
			Pkg: "exchanges/orderbook", Name: "BenchmarkSortAsksDescending", NS: []float64{3556, 3561, 3550},
			Bytes: []float64{24, 24, 24}, Allocs: []float64{1, 1, 1},
		},
		"currency.BenchmarkNewCode": {
			Pkg: "currency", Name: "BenchmarkNewCode", NS: []float64{1050, 1049, 1052},
			Bytes: []float64{16, 16, 16}, Allocs: []float64{2, 2, 2},
		},
	}
	assert.Equal(t, exp, run.Samples,
		"every -count sample should be kept under its package, with the module prefix and GOMAXPROCS suffix stripped")
	assert.Equal(t, map[string]bool{"exchanges/orderbook": true, "currency": true}, run.Finished,
		"each package whose output reached its ok line should be marked finished")
	assert.Empty(t, run.Skipped, "a run in which nothing skipped should record no skips")
	assert.Equal(t, map[string]bool{"exchanges/orderbook": true, "currency": true}, run.Verbose,
		"the bare names -v announces each benchmark with should mark each package's output verbose")
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
	assert.Empty(t, run.Verbose, "output without bare benchmark names should not count as -v output")
}

func TestParseRecordsSkips(t *testing.T) {
	t.Parallel()
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkParentSkip\n" +
		"    code_test.go:9: parent skipped\n" +
		"--- SKIP: BenchmarkParentSkip\n" +
		"BenchmarkSub\n" +
		"BenchmarkSub/a\n" +
		"    code_test.go:15: sub skipped\n" +
		"--- SKIP: BenchmarkSub/a\n" +
		"BenchmarkSub/b\n" +
		"BenchmarkSub/b-4        \t       1\t       400.0 ns/op\t       0 B/op\t       0 allocs/op\n" +
		"BenchmarkSub/c\n" +
		"BenchmarkSub/c/deep\n" +
		"    code_test.go:21: deep skipped\n" +
		"--- SKIP: BenchmarkSub/c/deep\n" +
		"PASS\n" +
		"ok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.015s\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Equal(t, map[string]string{
		"currency.BenchmarkParentSkip": "currency",
		"currency.BenchmarkSub/a":      "currency",
		"currency.BenchmarkSub/c/deep": "currency",
	}, run.Skipped, "every skip should be recorded under its package")
	assert.Contains(t, run.Samples, "currency.BenchmarkSub/b", "the subbenchmark that ran should still be measured")
	assert.Equal(t, map[string]bool{"currency": true}, run.Verbose, "bare benchmark names should mark the package's output as -v output")
}

func TestParseRejectsUnattributedSkips(t *testing.T) {
	t.Parallel()
	_, err := Parse(strings.NewReader("--- SKIP: BenchmarkNewCode\n"))
	assert.ErrorIs(t, err, errNoPackageContext, "a skip with no pkg: header should be refused, since it cannot be attributed")
	_, err = Parse(strings.NewReader("BenchmarkNewCode\nfixture missing: --- SKIP: BenchmarkNewCode\n"))
	assert.ErrorIs(t, err, errNoPackageContext, "a skip record output ran into, with no pkg: header, should be refused too")
	_, err = Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\n--- SKIP: \n"))
	assert.ErrorIs(t, err, errMalformedResult, "a skip naming no benchmark should be rejected")
}

func TestParseIgnoresLoggedRecords(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n" +
		"--- BENCH: BenchmarkNewCode-8\n" +
		"    code_test.go:12: parser input: --- FAIL: Example\n" +
		"    code_test.go:13: a message spanning lines\n" +
		"        --- FAIL: BenchmarkPretend\n" +
		"    code_test.go:14: captured subprocess output:\n" +
		"        ok  \tgithub.com/thrasher-corp/gocryptotrader/types\t0.001s\n" +
		"        --- SKIP: BenchmarkElsewhere\n" +
		"    code_test.go:15: reference: BenchmarkProcess-8 5572401 210.9 ns/op 0 B/op 0 allocs/op\n" +
		"    code_test.go:16: references:\n" +
		"        BenchmarkProcess-8 5572401 210.9 ns/op 0 B/op 0 allocs/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not read logged text as a failure or a result")
	assert.Empty(t, run.Finished, "a logged ok line should not mark a package finished")
	assert.Empty(t, run.Skipped, "a logged skip line should not record a skip")
	assert.Equal(t, []float64{2}, run.Samples["currency.BenchmarkNewCode"].Allocs, "the result should still be recorded")
}

func TestParseZeroIterationsIsSkip(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkLate\n" +
		"    code_test.go:11: skip during calibration\n" +
		strings.Repeat("BenchmarkLate-4   \t       0\t               NaN ns/op\t       0 B/op\t       0 allocs/op\n", 3) +
		"BenchmarkNewCode\n" +
		"BenchmarkNewCode-4   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Equal(t, map[string]string{"currency.BenchmarkLate": "currency"}, run.Skipped, "a zero-iteration result should be recorded as a skip")
	assert.NotContains(t, run.Samples, "currency.BenchmarkLate", "a zero-iteration result should not be recorded as a measurement")
	assert.Contains(t, run.Samples, "currency.BenchmarkNewCode", "the benchmark that ran should still be measured")
}

func TestParseRecordsSkipsOutputRanInto(t *testing.T) {
	t.Parallel()
	header := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n"
	result := "BenchmarkAlive-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n"
	// Ends the first package at a pkg: header rather than at EOF
	next := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/types\nBenchmarkOther\n" +
		"BenchmarkOther-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\nPASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/types\t0.9s\n"
	for _, tc := range []struct{ name, in, key string }{
		{"text before b.SkipNow", "BenchmarkCase\npreparing fixture: --- SKIP: BenchmarkCase\n", "currency.BenchmarkCase"},
		{"a cleanup printing after b.Skip", "BenchmarkCase\n    code_test.go:34: fixture unavailable\nclosing fixture: --- SKIP: BenchmarkCase\n", "currency.BenchmarkCase"},
		{"a subbenchmark", "BenchmarkCase\nBenchmarkCase/child\npreparing fixture: --- SKIP: BenchmarkCase/child\n", "currency.BenchmarkCase/child"},
		{"a parent printing before b.Run", "BenchmarkCase\npreparing fixture: BenchmarkCase/child\n--- SKIP: BenchmarkCase/child\n", "currency.BenchmarkCase/child"},
		{"a benchmark beside a longer name", "BenchmarkCase\npreparing fixture: --- SKIP: BenchmarkCase\nBenchmarkCaseExtra\n" +
			"BenchmarkCaseExtra-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", "currency.BenchmarkCase"},
		{"text laid out as a logged line before b.SkipNow, beside a longer name", "BenchmarkCase\n        preparing fixture: --- SKIP: BenchmarkCase\n" +
			"BenchmarkCaseExtra\nBenchmarkCaseExtra/child\nBenchmarkCaseExtra/child-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", "currency.BenchmarkCase"},
		{"text laid out as a logged line before b.SkipNow, beside a shorter name", "BenchmarkCas\n" +
			"BenchmarkCas-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\nBenchmarkCase\n        preparing fixture: --- SKIP: BenchmarkCase\n", "currency.BenchmarkCase"},
		{"text laid out as a logged line before b.SkipNow, beside subbenchmarks named after it", "BenchmarkCase\n        preparing fixture: --- SKIP: BenchmarkCase\n" +
			"BenchmarkOther\nBenchmarkOther/currency.BenchmarkCase\nBenchmarkOther/currency.BenchmarkCase-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n" +
			"BenchmarkOther/currency.BenchmarkCase/child\nBenchmarkOther/currency.BenchmarkCase/child-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", "currency.BenchmarkCase"},
		{"text laid out as a logged line before b.SkipNow", "BenchmarkCase\n        preparing fixture: --- SKIP: BenchmarkCase\n", "currency.BenchmarkCase"},
		{"text laid out as a logged line before the b.SkipNow of a subbenchmark named after a measured benchmark", "BenchmarkCase\n" +
			"BenchmarkCase-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\nBenchmarkOther\nBenchmarkOther/currency.BenchmarkCase\n" +
			"        preparing fixture: --- SKIP: BenchmarkOther/currency.BenchmarkCase\n", "currency.BenchmarkOther/currency.BenchmarkCase"},
		{"text laid out as a logged line before a subbenchmark's b.SkipNow", "BenchmarkCase\nBenchmarkCase/child\n" +
			"    fixture.go:12: --- SKIP: BenchmarkCase/child\n", "currency.BenchmarkCase/child"},
		{"a parent printing before b.Run and its child before b.SkipNow", "BenchmarkCase\nBenchmarkCase/small\n" +
			"BenchmarkCase/small-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n" +
			"opening fixture: BenchmarkCase/large\n fixture missing: --- SKIP: BenchmarkCase/large\n", "currency.BenchmarkCase/large"},
	} {
		// Also covers a run in which everything skipped
		for _, alive := range []string{"", "BenchmarkAlive\n" + result} {
			for _, tail := range []string{"", next} {
				run, err := Parse(strings.NewReader(header + alive + tc.in + "PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.9s\n" + tail))
				require.NoErrorf(t, err, "Parse must not error on %s", tc.name)
				assert.Equalf(t, map[string]string{tc.key: "currency"}, run.Skipped,
					"an announced benchmark that reported nothing should be recorded as skipped after %s, measured alongside: %t, followed by more output: %t",
					tc.name, alive != "", tail != "")
			}
		}
	}

	run, err := Parse(strings.NewReader(header + "BenchmarkAlive\n" + result + "BenchmarkHang\n"))
	require.NoError(t, err, "Parse must not error on output cut short")
	assert.Empty(t, run.Skipped, "a benchmark in a package that did not finish should not be recorded as skipped")
}

func TestParseRecordsSkipRecords(t *testing.T) {
	t.Parallel()
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkCase\nBenchmarkCase/small\nBenchmarkCase/small-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n" +
		"BenchmarkCase/skipped\n--- SKIP: BenchmarkCase/skipped\n" +
		"opening fixture: BenchmarkCase/large\n        --- SKIP: BenchmarkCase/large\n" +
		"    code_test.go:9: --- SKIP: BenchmarkCase\n" +
		"fixture ready: BenchmarkHidden\nprogress: --- SKIP: BenchmarkHidden\n" +
		"    code_test.go:10: --- SKIP: BenchmarkCase/other and more\n" +
		"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.9s\n" +
		// Text after the ok line, such as the next package's TestMain output
		"    main_test.go:5: --- SKIP: BenchmarkAfter\n" +
		"goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/types\nBenchmarkOther\n" +
		"BenchmarkOther-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\nPASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/types\t0.9s\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Equal(t, map[string]string{
		"currency.BenchmarkCase/skipped": "currency",
		"currency.BenchmarkCase/large":   "currency",
		"currency.BenchmarkCase":         "currency",
		"currency.BenchmarkHidden":       "currency",
		"currency.BenchmarkAfter":        "currency",
	}, run.SkipRecords, "every skip record ending a line should be kept under the last package header, wherever it starts")
	assert.Equal(t, map[string]string{"currency.BenchmarkCase/skipped": "currency"}, run.Skipped,
		"only go test's own record, and records run into announced names, should be read as skips")

	run, err = Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"transcript: --- SKIP: BenchmarkPrior--- SKIP: BenchmarkTwice/BenchmarkInner\ntranscript SKIP: BenchmarkPlain\n" +
		"transcript: --- SKIP: BenchmarkOuterSKIP: BenchmarkInner\ntranscript: --- SKIP: BenchmarkOuter--- SKIP: TestInner\n"))
	require.NoError(t, err, "Parse must not error on two skip records in one line")
	assert.Equal(t, map[string]string{"currency.BenchmarkTwice/BenchmarkInner": "currency"}, run.SkipRecords,
		"only the skip record ending a line should be kept")
}

func TestParseRejectsMalformedPackageHeaders(t *testing.T) {
	t.Parallel()
	for _, header := range []string{
		"pkg: fixture ready: BenchmarkMissing", "pkg: fixture\tready:BenchmarkMissing", "pkg: fixture:BenchmarkMissing",
		"pkg: fixture\u00a0ready", "pkg:", "pkg: \t",
		// go test prints each package's header once
		"pkg: github.com/thrasher-corp/gocryptotrader/currency",
	} {
		in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkSeed\n" +
			"BenchmarkSeed-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n" + header + "\n    fixture.go:12: --- SKIP: BenchmarkMissing\n"
		_, err := Parse(strings.NewReader(in))
		assert.ErrorIsf(t, err, errMalformedResult, "%q should be rejected rather than read as a package header", header)
	}
}

func TestParseRejectsResultsUnderUnannouncedNames(t *testing.T) {
	t.Parallel()
	for _, printed := range []string{"BenchmarkProgress:", "BenchmarkX"} {
		in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkCase\n" +
			strings.Repeat(printed+"BenchmarkCase-4     \t 6745969\t        17.66 ns/op\t      32 B/op\t       1 allocs/op\n", 3)
		_, err := Parse(strings.NewReader(in))
		assert.ErrorIsf(t, err, errMalformedResult, "a result run into by %q should be rejected rather than filed under a name -v never announced", printed)
	}
}

func TestParseIgnoresPrintedNames(t *testing.T) {
	t.Parallel()
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkCase\nBenchmarkProgress:\n" +
		"BenchmarkCase-4     \t 6745969\t        17.66 ns/op\t      32 B/op\t       1 allocs/op\n" +
		"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.9s\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error on a printed line that cannot be a benchmark name")
	assert.Empty(t, run.Skipped, "a printed line should not be taken for a benchmark that skipped")
	assert.Contains(t, run.Samples, "currency.BenchmarkCase", "the benchmark's result should still be recorded")

	run, err = Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkProgress:\n" +
		"BenchmarkCase-4     \t 6745969\t        17.66 ns/op\t      32 B/op\t       1 allocs/op\n"))
	require.NoError(t, err, "Parse must not error on a printed line in output without -v")
	assert.Empty(t, run.Verbose, "a printed line should not mark the output as -v output")
}

func TestBenchmarkName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		exp  bool
	}{
		{"Benchmark", true},
		{"BenchmarkX", true},
		{"Benchmark_x", true},
		{"BenchmarkX/sub name: with anything", true},
		{"BenchmarkX/BenchmarkY", true},
		{"Benchmarkx", false},
		{"Benchmarké", false},
		{"BenchmarkProgress:", false},
		{"BenchmarkProgress:BenchmarkCase", false},
		{"Bench", false},
	} {
		assert.Equalf(t, tc.exp, benchmarkName(tc.name), "benchmarkName should judge %q as go test names benchmarks", tc.name)
	}
}

func TestParseNamesFromAnnouncements(t *testing.T) {
	t.Parallel()
	// Under -cpu 1 there is no suffix, so only the announcements show "small-4" is a whole name
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkCases\n" +
		"BenchmarkCases/small-4\n" +
		"BenchmarkCases/small-4 100 10 ns/op 0 B/op 0 allocs/op\n" +
		"BenchmarkCases/large-4\n" +
		"BenchmarkCases/large-4 100 10 ns/op 0 B/op 0 allocs/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Contains(t, run.Samples, "currency.BenchmarkCases/small-4", "the announced name should be kept whole")
	assert.Contains(t, run.Samples, "currency.BenchmarkCases/large-4", "the announced name should be kept whole")

	run, err = Parse(strings.NewReader(strings.ReplaceAll(in, "-4 100", "-4-4 100")))
	require.NoError(t, err, "Parse must not error")
	assert.Contains(t, run.Samples, "currency.BenchmarkCases/small-4", "only the suffix after the announced name should be stripped")
}

func TestParseAnnouncementStaysInItsPackage(t *testing.T) {
	t.Parallel()
	// The second package's output lacks -v, so the first package's announcements must not name its results
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkZ\nBenchmarkZ/n-4\n" +
		"BenchmarkZ/n-4-4 100 10 ns/op 0 B/op 0 allocs/op\n" +
		"pkg: github.com/thrasher-corp/gocryptotrader/types\n" +
		"BenchmarkZ/n-4 100 10 ns/op 0 B/op 0 allocs/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Contains(t, run.Samples, "currency.BenchmarkZ/n-4", "the announced name should be kept whole in its own package")
	assert.Contains(t, run.Samples, "types.BenchmarkZ/n", "another package's result should not be read against that announcement")
}

func TestParseNamesAParentAfterItsSubbenchmarks(t *testing.T) {
	t.Parallel()
	// A parent that runs a child only once b.N > 1 reports its own result after the child's
	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkLateSub\n" +
		"BenchmarkLateSub/small-4\n" +
		"BenchmarkLateSub/small-4/child\n" +
		"BenchmarkLateSub/small-4/child   \t       2\t       150.0 ns/op\t       0 B/op\t       0 allocs/op\n" +
		"BenchmarkLateSub/small-4         \t       2\t    162950 ns/op\t    2092 B/op\t      19 allocs/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Contains(t, run.Samples, "currency.BenchmarkLateSub/small-4", "the parent should keep its whole name")
	assert.Contains(t, run.Samples, "currency.BenchmarkLateSub/small-4/child", "the child should keep its whole name")
}

func TestParseIgnoresTestSkips(t *testing.T) {
	t.Parallel()
	// With tests run too, a test's skip comes before any pkg: header
	in := "=== RUN   TestOptional\n" +
		"    code_test.go:5: optional\n" +
		"--- SKIP: TestOptional (0.00s)\n" +
		"goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode\n" +
		"BenchmarkNewCode-4   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not read a skipped test as an unattributed benchmark")
	assert.Empty(t, run.Skipped, "a skipped test should not be recorded as a skipped benchmark")
	assert.Contains(t, run.Samples, "currency.BenchmarkNewCode", "the benchmark should still be measured")
}

func TestParseRejectsMixedProcs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, lines string }{
		{"two -cpu values", "BenchmarkA-4 100 10 ns/op 0 B/op 0 allocs/op\nBenchmarkB-8 100 10 ns/op 0 B/op 0 allocs/op\n"},
		// Under -cpu 1, "size-1" would read as a suffix
		{"a name ending in digits under -cpu 1", "BenchmarkA 100 10 ns/op 0 B/op 0 allocs/op\nBenchmarkCases/size-1 100 10 ns/op 0 B/op 0 allocs/op\n"},
	} {
		_, err := Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\n" + tc.lines))
		assert.ErrorIsf(t, err, errMixedProcs, "Parse should reject %s", tc.name)
	}
	run, err := Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkA 100 10 ns/op 0 B/op 0 allocs/op\nBenchmarkB 100 10 ns/op 0 B/op 0 allocs/op\n"))
	require.NoError(t, err, "output with no suffix on any result must parse")
	assert.Len(t, run.Samples, 2, "both results should be recorded")
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

func TestParseRejectsOutputRunIntoResults(t *testing.T) {
	t.Parallel()
	header := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n"
	result := "BenchmarkNoisy-4   \t  122286\t   906.4 ns/op\t    4096 B/op\t       1 allocs/op\n"
	for _, tc := range []struct{ name, in string }{
		{"a dot", "BenchmarkNoisy\n" + strings.Repeat("."+result, 3)},
		{"a dot from a subbenchmark", "BenchmarkParent\nBenchmarkParent/quiet\n" +
			strings.Repeat("BenchmarkParent/quiet-4   \t 4356048\t   30.31 ns/op\t      64 B/op\t       1 allocs/op\n", 3) +
			".BenchmarkParent/noisy\n" + strings.Repeat(".BenchmarkParent/noisy-4   \t  152587\t   874.7 ns/op\t    4096 B/op\t       1 allocs/op\n", 3)},
		{"text naming a benchmark", "BenchmarkNoisy\n" + strings.Repeat("working on BenchmarkNoisy: "+result, 3)},
		{"text read as go test's ok line", "BenchmarkNoisy\n" + strings.Repeat("ok "+result, 3)},
		{"text read as a goos header", "BenchmarkNoisy\n" + strings.Repeat("goos: "+result, 3)},
		{"spaces after a logged message", "BenchmarkNoisy\n    code_test.go:4: progress\n" + strings.Repeat("        "+result, 3)},
		{"text shaped like a logged message", "BenchmarkNoisy\n" + strings.Repeat("    worker.go:12: "+result, 3)},
		// Text written between a result's name and its measurements
		{"text and a tab after the name", "BenchmarkNoisy\n" + strings.Repeat("prefix \t"+strings.Replace(result, "\t", "\t\t", 1), 3)},
		{"text and numbers after the name", "BenchmarkNoisy\n" + strings.Repeat("prefix "+strings.Replace(result, "\t", "\t1 2 ", 1), 3)},
		{"text and a no-break space after the name", "BenchmarkNoisy\n" + strings.Repeat("prefix "+strings.Replace(result, "\t", "\t\u00a0", 1), 3)},
		// Or a newline written there
		{"a newline after the name", "BenchmarkNoisy\n" + strings.Repeat(strings.Replace(result, "\t", "\t\n\n", 1), 3)},
		{"a vertical tab and a newline after the name", "BenchmarkNoisy\n" + strings.Repeat(strings.Replace(result, "\t", "\t\v\n", 1), 3)},
		{"a no-break space and a newline after the name", "BenchmarkNoisy\n" + strings.Repeat(strings.Replace(result, "\t", "\t\u00a0\n", 1), 3)},
	} {
		_, err := Parse(strings.NewReader(header + tc.in))
		assert.ErrorIsf(t, err, errMalformedResult, "output with %s should be rejected", tc.name)
	}

	run, err := Parse(strings.NewReader(header + "BenchmarkNewCode\n" +
		"BenchmarkNewCode-4   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n" +
		"--- BENCH: BenchmarkNewCode-4\n" +
		"    code_test.go:9: compared against BenchmarkPretend\n" +
		"testing: BenchmarkNewCode-4 left GOMAXPROCS set to 2\n" +
		"BenchmarkSkipper\n--- SKIP: BenchmarkSkipper\n" +
		"BenchmarkNest\nBenchmarkNest/BenchmarkInner\n" +
		"BenchmarkNest/BenchmarkInner-4   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n"))
	require.NoError(t, err, "Parse must accept lines that only name a benchmark")
	assert.Equal(t, []float64{2}, run.Samples["currency.BenchmarkNewCode"].Allocs, "the result should still be recorded")
	assert.Contains(t, run.Samples, "currency.BenchmarkNest/BenchmarkInner", "a result naming a benchmark twice should be recorded")
	assert.Contains(t, run.Skipped, "currency.BenchmarkSkipper", "the skip should still be recorded")
}

func TestEmbeddedResult(t *testing.T) {
	t.Parallel()
	result := "BenchmarkNoisy-4   \t  122286\t   906.4 ns/op\t    4096 B/op\t       1 allocs/op"
	for _, tc := range []struct {
		line string
		exp  bool
	}{
		{"." + result, true},
		{"ok " + result, true},
		{"working on BenchmarkNoisy: " + result, true},
		{result, false},
		{"BenchmarkNest/BenchmarkInner-4   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op", false},
		{"--- BENCH: BenchmarkNoisy-4", false},
		{"    code_test.go:9: compared against BenchmarkPretend", false},
		{"testing: BenchmarkNoisy-4 left GOMAXPROCS set to 2", false},
		{"", false},
		// Only go test's layout counts: the name, padded with spaces, then a tab
		{"    code_test.go:15: reference: BenchmarkNoisy-4 122286 906.4 ns/op 4096 B/op 1 allocs/op", false},
		// Copied from a terminal, which shows tabs as spaces
		{"    code_test.go:16: BenchmarkNoisy-4      122286         906.4 ns/op        4096 B/op          1 allocs/op", false},
		{"working on BenchmarkNoisy-4 122286\t906.4 ns/op\t4096 B/op\t1 allocs/op", false},
		{"working on BenchmarkNoisy-4\f\t122286\t906.4 ns/op\t4096 B/op\t1 allocs/op", false},
		{"    code_test.go:17: BenchmarkNoisy-4\t122286\t906.4 ns/op\t4096 B/op\t1 allocs/op", false},
		{"    code_test.go:18: BenchmarkNoisy-4 122286 906.4 ns/op 1 BenchmarkOther \t1 allocs/op", false},
		{"working on BenchmarkNoisy-4 \t122286 906.4 ns/op\t4096 B/op\t1 allocs/op", true},
		{"x" + strings.Replace(result, "Noisy-4", "", 1), true}, // a benchmark named Benchmark alone, under -cpu 1
		{"        " + result, true},
		{"    worker.go:12: " + result, true},
		{"x" + strings.Replace(result, "  122286", "123456789", 1), true},
		// Anything can come between the name and the measurements
		{"x" + strings.Replace(result, "\t", "\t\t\t", 1), true},
		{"x" + strings.Replace(result, "\t", "\t1 2 ", 1), true},
		{"x" + strings.Replace(result, "\t", "\t\v\u00a0\u2003 ", 1), true},
	} {
		assert.Equalf(t, tc.exp, embeddedResult(tc.line, nil), "embeddedResult should judge %q", tc.line)
	}
}

func TestParseRejectsFailedRun(t *testing.T) {
	t.Parallel()
	for _, line := range []string{
		"--- FAIL: BenchmarkNewCode-8",
		"BenchmarkNewCode/sub-8   \t--- FAIL: BenchmarkNewCode/sub-8",
		"BenchmarkNewCode-8   \t--- FAIL: BenchmarkNewCode-8",
		"--- FAIL: BenchmarkNewCode/x/y",
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

func TestParseEndsPackageAtItsOkRecord(t *testing.T) {
	t.Parallel()
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkShared\n" +
		"BenchmarkShared-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\nPASS\n" +
		"ok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.9s\n" +
		"transcript: --- SKIP: BenchmarkShared\n" +
		"goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/types\nBenchmarkOther\n" +
		"BenchmarkOther-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\nPASS\n" +
		"ok  \tgithub.com/thrasher-corp/gocryptotrader/types\t0.9s\n"
	run, err := Parse(strings.NewReader(in))
	require.NoError(t, err, "Parse must not error")
	assert.Empty(t, run.Skipped, "text after a package's ok record should not be read as one of its skips")
	assert.Equal(t, map[string]bool{"currency": true, "types": true}, run.Finished, "both packages should be finished")

	run, err = Parse(strings.NewReader("goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkCodec\nBenchmarkCodec/small\nBenchmarkCodec/small-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n" +
		"ok  \tgithub.com/thrasher-corp/gocryptotrader/types\t0.1s\n" +
		"opening fixture: BenchmarkCodec/large\nfixture missing: --- SKIP: BenchmarkCodec/large\n" +
		"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.9s\n"))
	require.NoError(t, err, "Parse must not error")
	assert.Equal(t, map[string]string{"currency.BenchmarkCodec/large": "currency"}, run.Skipped,
		"a skip record run into after another package's printed ok record should still be read")

	run, err = Parse(strings.NewReader("goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkDiagnostic\nok github.com/thrasher-corp/gocryptotrader/currency\n" +
		strings.Repeat("BenchmarkDiagnostic-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3) +
		"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.9s\n"))
	require.NoError(t, err, "Parse must not error")
	assert.Empty(t, run.Skipped, "a benchmark that printed its package's ok record before measuring should not be read as skipped")
	assert.Len(t, run.Samples["currency.BenchmarkDiagnostic"].Allocs, 3, "its samples should all be recorded")
}

func TestEmbeddedSkip(t *testing.T) {
	t.Parallel()
	announced := []string{"BenchmarkCodec", "BenchmarkCodec/small"}
	for _, tc := range []struct {
		line, exp string
	}{
		{line: "fixture missing: --- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: " fixture missing: --- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: "fixture missing: --- SKIP: BenchmarkCodec", exp: "BenchmarkCodec"},
		{line: ".--- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: "✓--- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: " --- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: "ab--- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		// Only the last record counts, since go test's ends the line
		{line: "transcript: --- SKIP: BenchmarkPrior--- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: "transcript: --- SKIP: BenchmarkCodec/large and more"},
		// Unannounced names are printed text
		{line: "transcript: --- SKIP: BenchmarkGhost"},
		{line: "transcript: --- SKIP: BenchmarkCodecs"},
		{line: "transcript: --- SKIP: BenchmarkCode"},
		{line: "transcript: --- SKIP: BenchmarkCodecs/large"},
		{line: "transcript: --- SKIP: BenchmarkOther/BenchmarkCodec/small"},
		{line: "transcript: --- SKIP: TestCodec"},
		// b.Log's layout
		{line: "    codec_test.go:9: --- SKIP: BenchmarkCodec/large"},
		{line: "    generated.template:10: --- SKIP: BenchmarkCodec/large"},
		{line: "    generated source.template:10: --- SKIP: BenchmarkCodec/large"},
		{line: "        --- SKIP: BenchmarkCodec/large"},
		// Text indented some other way is a benchmark's own
		{line: "    fixture unavailable. --- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: "   step:1: --- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		{line: "fixture missing:    codec_test.go:9: --- SKIP: BenchmarkCodec/large", exp: "BenchmarkCodec/large"},
		// Parse reads a record starting the line itself
		{line: "--- SKIP: BenchmarkCodec/large"},
		{line: ""},
	} {
		got, ok := embeddedSkip(tc.line, announced)
		assert.Equalf(t, tc.exp != "", ok, "embeddedSkip should report whether %q holds go test's record", tc.line)
		assert.Equalf(t, tc.exp, got, "embeddedSkip should return the name in %q", tc.line)
	}
	_, ok := embeddedSkip("transcript: --- SKIP: BenchmarkInner", []string{"BenchmarkCodec/BenchmarkInner"})
	assert.False(t, ok, "embeddedSkip should not match a record naming only the end of an announced name")
}

func TestFailureRecord(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		line string
		exp  bool
	}{
		{"FAIL", true},
		{"FAIL\tgithub.com/thrasher-corp/gocryptotrader/currency\t1.200s", true},
		{"--- FAIL: BenchmarkNewCode-8", true},
		{"--- FAIL: BenchmarkNewCode/x/y", true},
		{"BenchmarkNewCode-8   \t--- FAIL: BenchmarkNewCode-8", true},
		{"BenchmarkNewCode/sub-8   \t--- FAIL: BenchmarkNewCode/sub-8", true},
		// Logged text is indented, so it is never go test's record
		{"    code_test.go:12: parser input: --- FAIL: Example", false},
		{"    code_test.go:12: --- FAIL: BenchmarkNewCode", false},
		{"        --- FAIL: BenchmarkPretend", false},
		{"FAILED", false},
		{"PASS", false},
		{"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op", false},
	} {
		assert.Equalf(t, tc.exp, failureRecord(tc.line), "failureRecord should judge %q", tc.line)
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
	// go test prints custom metrics before B/op and allocs/op, as in the first three lines
	for _, line := range []string{
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      -3.000 delta/op\t      16 B/op\t       2 allocs/op\n",
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t             NaN ratio/op\t      16 B/op\t       2 allocs/op\n",
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      +Inf ratio/op\t      16 B/op\t       2 allocs/op\n",
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\t      -3.000 delta/op\n",
	} {
		run, err := Parse(strings.NewReader("pkg: github.com/thrasher-corp/gocryptotrader/currency\n" + line))
		require.NoErrorf(t, err, "a negative or non-finite custom metric must not fail the run, since b.ReportMetric allows one: %q", line)
		assert.Equalf(t, &Samples{Pkg: "currency", Name: "BenchmarkNewCode", NS: []float64{1050}, Bytes: []float64{16}, Allocs: []float64{2}},
			run.Samples["currency.BenchmarkNewCode"], "the gated metrics should still be recorded: %q", line)
	}
}

func TestParseBadMeasurementDoesNotVanish(t *testing.T) {
	t.Parallel()
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
		line      string
		announced []string
		exp       benchLine
	}{
		{
			line: "BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op",
			exp: benchLine{
				name: "BenchmarkNewCode", procs: "8", iterations: 1000000,
				metrics: map[string]float64{"ns/op": 1050, "B/op": 16, "allocs/op": 2},
			},
		},
		{
			line: "BenchmarkSort/case.1-4 10 5.5 ns/op 0 B/op 0 allocs/op 3 widgets/op",
			exp: benchLine{
				name: "BenchmarkSort/case.1", procs: "4", iterations: 10,
				metrics: map[string]float64{"ns/op": 5.5, "B/op": 0, "allocs/op": 0, "widgets/op": 3},
			},
		},
		{
			line: "BenchmarkNewCode 10 5.5 ns/op 0 B/op 0 allocs/op",
			exp:  benchLine{name: "BenchmarkNewCode", iterations: 10, metrics: map[string]float64{"ns/op": 5.5, "B/op": 0, "allocs/op": 0}},
		},
		{
			line:      "BenchmarkCases/size-4 10 5.5 ns/op 0 B/op 0 allocs/op",
			announced: []string{"BenchmarkCases/size-4"},
			exp:       benchLine{name: "BenchmarkCases/size-4", iterations: 10, metrics: map[string]float64{"ns/op": 5.5, "B/op": 0, "allocs/op": 0}},
		},
		{line: "BenchmarkNewCode-8"},
		{line: "--- BENCH: BenchmarkNewCode-8 1 2 ns/op"},
		{line: "BenchmarkNewCode-8 notanumber 1050 ns/op"},
		{line: "BenchmarkNewCode-8 100 1050 ns/op 16"},
		{line: "BenchmarkNewCode-8 100 ten ns/op"},
	} {
		got, ok := parseBenchLine(tc.line, tc.announced)
		if tc.exp.name == "" {
			assert.Falsef(t, ok, "parseBenchLine should reject %q", tc.line)
			continue
		}
		require.Truef(t, ok, "parseBenchLine must accept %q", tc.line)
		assert.Equalf(t, tc.exp, got, "parseBenchLine should read the name, suffix, iterations and every metric of %q", tc.line)
	}

	got, ok := parseBenchLine("BenchmarkNewCode-8 100 1050 ns/op 16 B/op -2 allocs/op", nil)
	require.True(t, ok, "a negative measurement must still be returned, so Parse can reject it")
	assert.True(t, math.IsNaN(got.metrics["allocs/op"]), "a negative measurement should be marked NaN")
}

func TestSplitProcSuffix(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in        string
		announced []string
		name      string
		procs     string
	}{
		{"BenchmarkProcess-8", nil, "BenchmarkProcess", "8"},
		{"BenchmarkProcess-128", nil, "BenchmarkProcess", "128"},
		{"BenchmarkProcess", nil, "BenchmarkProcess", ""},
		{"BenchmarkUpdateInsertByID_asks-8", nil, "BenchmarkUpdateInsertByID_asks", "8"},
		{"BenchmarkSort-Ascending", nil, "BenchmarkSort-Ascending", ""},
		{"BenchmarkCases/size-1-4", nil, "BenchmarkCases/size-1", "4"},
		// An announcement settles it, even under -cpu 1
		{"BenchmarkCases/size-4", []string{"BenchmarkCases/size-4"}, "BenchmarkCases/size-4", ""},
		{"BenchmarkCases/size-4-4", []string{"BenchmarkCases/size-4"}, "BenchmarkCases/size-4", "4"},
		// An announcement belonging to another benchmark is no guide
		{"BenchmarkOther-4", []string{"BenchmarkCases/size-4"}, "BenchmarkOther", "4"},
		{"BenchmarkCases/size-4x", []string{"BenchmarkCases/size-4"}, "BenchmarkCases/size-4x", ""},
		{"BenchmarkX/size4", []string{"BenchmarkX/size"}, "BenchmarkX/size4", ""},
		{"BenchmarkX/size-large", []string{"BenchmarkX/size"}, "BenchmarkX/size-large", ""},
		{"BenchmarkX/size--4", []string{"BenchmarkX/size"}, "BenchmarkX/size-", "4"},
		{"BenchmarkY/BenchmarkX/size-4", []string{"BenchmarkX/size-4"}, "BenchmarkY/BenchmarkX/size", "4"},
		{"BenchmarkX/size-4", []string{"BenchmarkY/BenchmarkX/size-4"}, "BenchmarkX/size", "4"},
		{"BenchmarkX/size-0x10", []string{"BenchmarkX/size"}, "BenchmarkX/size-0x10", ""},
		{"BenchmarkX/size-+4", []string{"BenchmarkX/size"}, "BenchmarkX/size-+4", ""},
		{"BenchmarkX/size-4.", []string{"BenchmarkX/size"}, "BenchmarkX/size-4.", ""},
		{"BenchmarkX/size-4", []string{"BenchmarkX/size", "BenchmarkX/size-4-more"}, "BenchmarkX/size", "4"},
		// A parent's result can follow its subbenchmarks' announcements
		{"BenchmarkLateSub/small-4", []string{"BenchmarkLateSub", "BenchmarkLateSub/small-4", "BenchmarkLateSub/small-4/child"}, "BenchmarkLateSub/small-4", ""},
		// The newest announcement wins
		{"BenchmarkX/size-4", []string{"BenchmarkX/size"}, "BenchmarkX/size", "4"},
		{"BenchmarkX/size-4-4", []string{"BenchmarkX/size", "BenchmarkX/size-4"}, "BenchmarkX/size-4", "4"},
		{"BenchmarkX/size-4", []string{"BenchmarkX/size", "BenchmarkX/size-4"}, "BenchmarkX/size-4", ""},
		{"BenchmarkX/size-4", []string{"BenchmarkX/size-4", "BenchmarkX/size"}, "BenchmarkX/size", "4"},
	} {
		name, procs := splitProcSuffix(tc.in, tc.announced)
		assert.Equalf(t, tc.name, name, "splitProcSuffix should return the name of %s announced as %q", tc.in, tc.announced)
		assert.Equalf(t, tc.procs, procs, "splitProcSuffix should return the suffix of %s announced as %q", tc.in, tc.announced)
	}
}

func TestDescribeProcSuffix(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "-4", describeProcSuffix("4"), "a suffix should be shown as go test prints it")
	assert.Equal(t, "no suffix", describeProcSuffix(""), "a missing suffix should be named")
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
