package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOptions returns options for run() against a fresh directory, whose package list covers the
// packages in sampleOutput
func testOptions(t *testing.T) *options {
	t.Helper()
	dir := t.TempDir()
	o := &options{
		baselinePath: filepath.Join(dir, "baseline.json"),
		packagesPath: filepath.Join(dir, "packages.txt"),
		root:         dir,
		samples:      3,
	}
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\nexchanges/orderbook\n"), 0o600),
		"writing the package list must not error")
	return o
}

func TestRunList(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	const bench = "package p\n\nimport \"testing\"\n\nfunc BenchmarkX(b *testing.B) {}\n"
	writeFiles(t, o.root, map[string]string{"currency/x_test.go": bench, "exchanges/orderbook/x_test.go": bench})

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(""), &out, &options{packagesPath: o.packagesPath, root: o.root, list: true}),
		"-list must not error")
	assert.Equal(t, "./currency/ ./exchanges/orderbook/\n", out.String(), "-list should emit go test package arguments in file order")
}

func TestRunListRejectsUncoveredPackages(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	const bench = "package p\n\nimport \"testing\"\n\nfunc BenchmarkX(b *testing.B) {}\n"
	writeFiles(t, o.root, map[string]string{
		"currency/x_test.go":            bench,
		"exchanges/orderbook/x_test.go": bench,
		"types/decimal/x_test.go":       bench,
	})
	var out bytes.Buffer
	err := run(strings.NewReader(""), &out, &options{packagesPath: o.packagesPath, root: o.root, list: true})
	assert.ErrorIs(t, err, errUnlistedPackage, "-list should refuse while a benchmark package is neither listed nor excluded")
	assert.Empty(t, out.String(), "-list should print nothing for make to run when it fails")
}

func TestRunUpdateThenCheck(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o), "seeding the baseline must not error")
	assert.Contains(t, out.String(), "added currency.BenchmarkNewCode: allocs/op 2, B/op 16", "the update should describe each addition")
	assert.Contains(t, out.String(), "3 benchmarks folded into", "the update should report what it recorded")

	out.Reset()
	o.update = false
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o),
		"checking the same output against the seeded baseline must not error")
	assert.Contains(t, out.String(), "3 benchmarks within budget", "a clean check should say so")
}

func TestRunReportsRegression(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	base := Baseline{
		"currency.BenchmarkNewCode":                      {Allocs: 0, Bytes: 0},
		"exchanges/orderbook.BenchmarkProcess":           {Allocs: 0, Bytes: 0},
		"exchanges/orderbook.BenchmarkSortAsksDecending": {Allocs: 1, Bytes: 24},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	var out bytes.Buffer
	err := run(strings.NewReader(sampleOutput), &out, o)
	assert.ErrorContains(t, err, "2 findings", "a regression should fail the run")
	assert.Contains(t, out.String(), "regression currency.BenchmarkNewCode: allocs/op 0 -> 2", "the finding should be printed")
}

func TestRunRejectsInvalidSamples(t *testing.T) {
	t.Parallel()
	for _, samples := range []int{0, -1, -7, 2, 6} {
		o := testOptions(t)
		o.samples = samples
		err := run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o)
		assert.ErrorIsf(t, err, errInvalidSamples, "-samples %d should be rejected", samples)
	}
}

func TestRunRejectsWrongSampleCount(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.samples = 5
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	var out bytes.Buffer
	err := run(strings.NewReader(sampleOutput), &out, o)
	assert.ErrorIs(t, err, errSampleCount, "a run with the wrong number of samples should not be compared against the baseline")
	assert.Contains(t, out.String(), "currency.BenchmarkNewCode reported B/op=3 allocs/op=3, expected 5",
		"the offending benchmark should be named with its counts")
}

func TestRunRejectsPruneWithoutUpdate(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.prune = true
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errPruneWithoutUpdate,
		"-prune should not be silently ignored in check mode")
}

func TestRunRejectsUnlistedOutput(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	err := run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errUnlistedOutput, "output from a package the list does not benchmark should be rejected")
	assert.ErrorContains(t, err, "exchanges/orderbook", "the error should name the package")
}

func TestRunPruneRequiresCompleteRun(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update, o.prune = true, true
	base := Baseline{
		"currency.BenchmarkNewCode":            {Allocs: 2, Bytes: 16},
		"exchanges/orderbook.BenchmarkProcess": {Allocs: 0, Bytes: 0},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	// What `make bench_update BENCH_PKGS=./currency/` produces: one package of two. Pruning on it
	// would delete every budget in the package that was never asked to run.
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		strings.Repeat("BenchmarkNewCode-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3)
	err := run(strings.NewReader(in), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errIncompleteRun, "pruning on a run that skipped a listed package should be refused")

	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")

	o.prune = false
	require.NoError(t, run(strings.NewReader(in), &bytes.Buffer{}, o), "the same partial run must update without -prune")
	after, err = LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Contains(t, after, "exchanges/orderbook.BenchmarkProcess", "an update without -prune should keep what it did not measure")
}

func TestRunPruneRequiresFinishedPackages(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	o.update, o.prune = true, true
	base := Baseline{
		"currency.BenchmarkA": {Allocs: 2, Bytes: 16},
		"currency.BenchmarkB": {Allocs: 2, Bytes: 16},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	// Output cut off after BenchmarkA: every listed package reported something and every reported
	// benchmark has its samples, but BenchmarkB never ran
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		strings.Repeat("BenchmarkA-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3)
	err := run(strings.NewReader(in), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errIncompleteRun, "pruning on output that stopped before the ok line should be refused")
	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")

	// The same output run to its ok line is a run in which BenchmarkB really is gone
	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(in+"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t1.2s\n"), &out, o),
		"a finished run must prune")
	assert.Contains(t, out.String(), "pruned currency.BenchmarkB", "the benchmark the finished run did not report should be pruned")
}

func TestRunPruneDeletesOrphans(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update, o.prune = true, true
	base := Baseline{
		"currency.BenchmarkNewCode":                     {Allocs: 2, Bytes: 16},
		"exchange/websocket/buffer.BenchmarkBufferPerf": {Allocs: 1, Bytes: 8},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o), "a complete run must prune")
	assert.Contains(t, out.String(), "pruned exchange/websocket/buffer.BenchmarkBufferPerf", "the orphan should be pruned")

	o.update, o.prune = false, false
	require.NoError(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "the check after pruning must be clean")
}

func TestRunUpdateRefusesOtherPlatforms(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true
	for _, in := range []string{
		strings.ReplaceAll(sampleOutput, "goarch: amd64", "goarch: arm64"),
		strings.NewReplacer("goos: linux\n", "", "goarch: amd64\n", "").Replace(sampleOutput),
	} {
		err := run(strings.NewReader(in), &bytes.Buffer{}, o)
		assert.ErrorIs(t, err, errWrongPlatform, "budgets should only be recorded from the canonical platform")
		_, statErr := os.Stat(o.baselinePath)
		assert.True(t, os.IsNotExist(statErr), "a refused update should not write a baseline")
	}
}

func TestRunCheckNotesOtherPlatforms(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true
	require.NoError(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "seeding the baseline must not error")

	o.update = false
	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(strings.ReplaceAll(sampleOutput, "goos: linux", "goos: darwin")), &out, o),
		"a check from another platform must still run")
	assert.Contains(t, out.String(), `this output is from "darwin/amd64"`, "the check should say the platform differs")
}

func TestRunReportsFailedRun(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	err := run(strings.NewReader(sampleOutput+"--- FAIL: BenchmarkNewCode-8\n"), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errRunFailed, "output from a failed run should be rejected")
}

func TestRunNoResults(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	assert.ErrorIs(t, run(strings.NewReader("PASS\n"), &bytes.Buffer{}, o), errNoResults, "empty input should return errNoResults")
}

func TestRunRejectsInvalidBaseline(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16, AllocsTolerance: 0.5}}.Save(o.baselinePath),
		"seeding the baseline must not error")
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errToleranceNeedsReason,
		"a baseline that fails validation should not be compared against")
}

func TestRunListReportsDiscoveryErrors(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	writeFiles(t, o.root, map[string]string{"currency/x_test.go": "package currency\n\nfunc Benchmark(b *testing.B {\n"})
	err := run(strings.NewReader(""), &bytes.Buffer{}, &options{packagesPath: o.packagesPath, root: o.root, list: true})
	assert.ErrorContains(t, err, "error parsing", "-list should fail when discovery cannot read a package")
}

func TestRunReportsUnreadableBaseline(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.baselinePath, []byte("{nope"), 0o600), "writing the fixture must not error")
	assert.ErrorContains(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "error parsing baseline",
		"an unparsable baseline should fail the run")
}

func TestRunUpdateReportsSaveErrors(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true
	o.baselinePath = filepath.Join(t.TempDir(), "missing", "baseline.json")
	assert.ErrorContains(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "error creating temporary baseline",
		"a baseline that cannot be written should fail the update")
}

func TestRunMissingPackageList(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.packagesPath = filepath.Join(t.TempDir(), "absent.txt")
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errNoPackageList,
		"run should fail without a package list")
}

func TestUnfinishedPackages(t *testing.T) {
	t.Parallel()
	results := resultsOf(result("currency", "BenchmarkNewCode", 2, 16), result("types", "BenchmarkNumber", 1, 8))
	finished := map[string]bool{"currency": true, "common": true}
	assert.Equal(t, []string{"common", "types"}, unfinishedPackages(finished, results, []string{"common", "currency", "types"}),
		"unfinishedPackages should list packages without results and packages without an ok line, in list order")
}

func TestSilentPackages(t *testing.T) {
	t.Parallel()
	got := silentPackages(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), []string{"common", "currency", "types"})
	assert.Equal(t, []string{"common", "types"}, got, "silentPackages should list the packages without results, in list order")
}
