package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func result(pkg, name string, allocs, bytes float64) *Result {
	return &Result{Pkg: pkg, Name: name, NS: []float64{100}, Bytes: []float64{bytes}, Allocs: []float64{allocs}}
}

func resultsOf(rs ...*Result) map[string]*Result {
	m := make(map[string]*Result, len(rs))
	for _, r := range rs {
		m[r.Key()] = r
	}
	return m
}

func TestCompareWithinBudget(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}}
	got := resultsOf(result("currency", "BenchmarkNewCode", 2, 16))
	assert.Empty(t, Compare(base, got, nil, nil, 0.01), "a benchmark matching its baseline should produce no findings")
}

func TestCompareRegression(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}}
	got := resultsOf(result("currency", "BenchmarkNewCode", 3, 32))
	findings := Compare(base, got, nil, nil, 0.01)
	require.Len(t, findings, 2, "both allocs and bytes regressions must be reported")
	for _, f := range findings {
		assert.Equal(t, Regression, f.Kind, "an increase should be reported as a regression")
	}
	assert.Contains(t, findings[0].Msg, "allocs/op 2 -> 3", "the message should show the delta")
}

func TestCompareStaleRatchetsDown(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}}
	got := resultsOf(result("currency", "BenchmarkNewCode", 1, 8))
	findings := Compare(base, got, nil, nil, 0.01)
	require.Len(t, findings, 2, "an improvement in both metrics must be reported so the baseline is tightened")
	for _, f := range findings {
		assert.Equal(t, Stale, f.Kind, "an improvement should be reported as stale")
		assert.Contains(t, f.Msg, "bench_update", "a stale finding should tell the user how to fix it")
	}
}

func TestCompareBytesTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 1000}}
	assert.Empty(t, Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 1005)), nil, nil, 0.01),
		"B/op movement inside the tolerance should not be reported")
	assert.Len(t, Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 1020)), nil, nil, 0.01), 1,
		"B/op movement outside the tolerance should be reported")
}

func TestComparePerEntryTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"exchanges/alert.BenchmarkWait": {
		Allocs: 10, Bytes: 736, BytesTolerance: 0.15, Reason: "goroutine scheduling",
	}}
	assert.Empty(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 10, 795)), nil, nil, 0.01),
		"a per-entry tolerance should override the global one")
	assert.Len(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 10, 900)), nil, nil, 0.01), 1,
		"movement beyond the per-entry tolerance should still be reported")
	assert.Len(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 11, 736)), nil, nil, 0.01), 1,
		"a bytes tolerance should not loosen the allocs gate")
}

func TestCompareAllocsTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"config.BenchmarkUpdateConfig": {
		Allocs: 40731, AllocsTolerance: 0.001, Bytes: 9934399, BytesTolerance: 0.01, Reason: "config file IO",
	}}
	assert.Empty(t, Compare(base, resultsOf(result("config", "BenchmarkUpdateConfig", 40732, 9934985)), nil, nil, 0.01),
		"an allocs tolerance should absorb small run-to-run drift")
	assert.Len(t, Compare(base, resultsOf(result("config", "BenchmarkUpdateConfig", 41500, 9934399)), nil, nil, 0.01), 1,
		"an allocs tolerance should not hide a real regression")
}

func TestCompareIgnore(t *testing.T) {
	t.Parallel()
	base := Baseline{"exchanges/alert.BenchmarkWait": {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"}}
	assert.Empty(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 99, 9999)), nil, map[string]bool{"exchanges/alert": true}, 0.01),
		"an ignored benchmark should produce no findings however far it moves")
}

func TestCompareUntrackedOnlyInGatedPackages(t *testing.T) {
	t.Parallel()
	got := resultsOf(result("currency", "BenchmarkNewCode", 2, 16))
	assert.Empty(t, Compare(Baseline{}, got, nil, nil, 0.01), "an unlisted package should not require baseline entries")

	findings := Compare(Baseline{}, got, nil, map[string]bool{"currency": true}, 0.01)
	require.Len(t, findings, 1, "a gated package must require a baseline entry for every benchmark")
	assert.Equal(t, Untracked, findings[0].Kind, "a missing entry in a gated package should be untracked")
}

func TestCompareMissingOnlyForRunPackages(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode":         {Allocs: 2, Bytes: 16},
		"currency.BenchmarkGone":            {Allocs: 1, Bytes: 8},
		"exchanges/orderbook.BenchmarkSort": {Allocs: 1, Bytes: 8},
	}
	got := resultsOf(result("currency", "BenchmarkNewCode", 2, 16))
	findings := Compare(base, got, nil, nil, 0.01)
	require.Len(t, findings, 1, "only benchmarks from packages that actually ran must be reported missing")
	assert.Equal(t, Missing, findings[0].Kind, "a vanished benchmark should be reported as missing")
	assert.Equal(t, "currency.BenchmarkGone", findings[0].Key, "the missing benchmark should be the one that ran but did not report")
}

func TestCompareIgnoredBenchmarkNotReportedMissing(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"exchanges/alert.BenchmarkWait":  {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"},
		"exchanges/alert.BenchmarkAlert": {Allocs: 0, Bytes: 0},
	}
	got := resultsOf(result("exchanges/alert", "BenchmarkAlert", 0, 0))
	assert.Empty(t, Compare(base, got, nil, nil, 0.01),
		"deleting an ignored benchmark should not fail the gate it is exempt from")
}

func TestCompareDetectsDeletedLastBenchmarkInPackage(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"common.BenchmarkCounter":   {Allocs: 0, Bytes: 0},
		"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16},
	}
	// common's only benchmark was deleted, so common reports nothing at all. Keying missing
	// detection off the packages that reported would let its entry linger unnoticed forever.
	got := resultsOf(result("currency", "BenchmarkNewCode", 2, 16))
	configured := map[string]bool{"common": true, "currency": true}

	findings := Compare(base, got, configured, nil, 0.01)
	require.Len(t, findings, 1, "the vanished package's entry must be reported")
	assert.Equal(t, Missing, findings[0].Kind, "it should be reported as missing")
	assert.Equal(t, "common.BenchmarkCounter", findings[0].Key, "the missing key should be the deleted benchmark")

	assert.Empty(t, Compare(base, got, nil, nil, 0.01),
		"with no configured list, only packages that reported are considered")
}

func TestKindString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "regression", Regression.String(), "Regression should stringify")
	assert.Equal(t, "missing", Missing.String(), "Missing should stringify")
	assert.Equal(t, "untracked", Untracked.String(), "Untracked should stringify")
	assert.Equal(t, "stale", Stale.String(), "Stale should stringify")
	assert.Equal(t, "unknown", Kind(255).String(), "an out of range kind should stringify as unknown")
}
