package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func result(pkg, name string, allocs, bytes uint64) *Result {
	return &Result{Pkg: pkg, Name: name, Allocs: allocs, Bytes: bytes}
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
	assert.Empty(t, Compare(base, got, []string{"currency"}), "a benchmark matching its baseline should produce no findings")
}

func TestCompareRegression(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}}
	got := resultsOf(result("currency", "BenchmarkNewCode", 3, 32))
	exp := []Finding{
		{Regression, "currency.BenchmarkNewCode", "allocs/op 2 -> 3"},
		{Regression, "currency.BenchmarkNewCode", "B/op 16 -> 32"},
	}
	assert.Equal(t, exp, Compare(base, got, []string{"currency"}), "both regressions should be reported with their delta")
}

func TestCompareStaleRatchetsDown(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}}
	got := resultsOf(result("currency", "BenchmarkNewCode", 1, 8))
	exp := []Finding{
		{Stale, "currency.BenchmarkNewCode", "allocs/op improved 2 -> 1; run 'make bench_update'"},
		{Stale, "currency.BenchmarkNewCode", "B/op improved 16 -> 8; run 'make bench_update'"},
	}
	assert.Equal(t, exp, Compare(base, got, []string{"currency"}),
		"an improvement in both metrics should be reported so the baseline is tightened")
}

func TestCompareBytesTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 1000}}
	assert.Empty(t, Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 1005)), []string{"currency"}),
		"B/op movement inside the default tolerance should not be reported")
	assert.Len(t, Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 1020)), []string{"currency"}), 1,
		"B/op movement outside the default tolerance should be reported")
}

func TestComparePerEntryTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"exchanges/alert.BenchmarkWait": {
		Allocs: 10, Bytes: 736, BytesTolerance: new(0.15), Reason: "goroutine scheduling",
	}}
	listed := []string{"exchanges/alert"}
	assert.Empty(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 10, 795)), listed),
		"a per-entry tolerance should override the default one")
	assert.Len(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 10, 900)), listed), 1,
		"movement beyond the per-entry tolerance should still be reported")
	assert.Len(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 11, 736)), listed), 1,
		"a bytes tolerance should not loosen the allocs gate")
}

func TestCompareExplicitZeroBytesTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 1000, BytesTolerance: new(0.0), Reason: "exactly reproducible"}}
	assert.Len(t, Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 1005)), []string{"currency"}), 1,
		"an explicit zero tolerance should report movement the default tolerance would absorb")
}

func TestCompareAllocsTolerance(t *testing.T) {
	t.Parallel()
	base := Baseline{"config.BenchmarkUpdateConfig": {
		Allocs: 40731, AllocsTolerance: 0.001, Bytes: 9934399, Reason: "config file IO",
	}}
	listed := []string{"config"}
	assert.Empty(t, Compare(base, resultsOf(result("config", "BenchmarkUpdateConfig", 40732, 9934985)), listed),
		"an allocs tolerance should absorb small run-to-run drift")
	assert.Len(t, Compare(base, resultsOf(result("config", "BenchmarkUpdateConfig", 41500, 9934399)), listed), 1,
		"an allocs tolerance should not hide a real regression")
}

func TestCompareIgnore(t *testing.T) {
	t.Parallel()
	base := Baseline{"exchanges/alert.BenchmarkWait": {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"}}
	assert.Empty(t, Compare(base, resultsOf(result("exchanges/alert", "BenchmarkWait", 99, 9999)), []string{"exchanges/alert"}),
		"an ignored benchmark should produce no findings however far it moves")
}

func TestCompareUntracked(t *testing.T) {
	t.Parallel()
	findings := Compare(Baseline{}, resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), []string{"currency"})
	exp := []Finding{{Untracked, "currency.BenchmarkNewCode", "benchmark has no baseline entry; run 'make bench_update'"}}
	assert.Equal(t, exp, findings, "every measured benchmark should need a baseline entry")
}

func TestCompareMissingBenchmark(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16},
		"currency.BenchmarkGone":    {Allocs: 1, Bytes: 8},
	}
	findings := Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), []string{"currency"})
	require.Len(t, findings, 1, "the vanished benchmark must be the only finding")
	assert.Equal(t, Missing, findings[0].Kind, "a vanished benchmark should be reported as missing")
	assert.Equal(t, "currency.BenchmarkGone", findings[0].Key, "the missing benchmark should be the one that did not report")
	assert.Contains(t, findings[0].Msg, "renamed or deleted", "the finding should say why the entry is missing")
}

func TestCompareReportsSilentPackageOnce(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"common.BenchmarkCounter":   {Allocs: 0, Bytes: 0},
		"common.BenchmarkOther":     {Allocs: 0, Bytes: 0},
		"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16},
	}
	// common reported nothing at all: it failed to run, every benchmark skipped, or the run
	// deliberately covered only currency. One finding says so, rather than one per budget.
	findings := Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), []string{"common", "currency"})
	require.Len(t, findings, 1, "a silent package must be reported once")
	assert.Equal(t, Finding{Missing, "common", "listed package reported no benchmarks; it failed to run, or every benchmark in it skipped"},
		findings[0], "the finding should name the silent package")
}

func TestCompareReportsSilentPackageWithoutBudgets(t *testing.T) {
	t.Parallel()
	// A newly listed package whose benchmarks all skip has no budgets to go missing, so without a
	// package-level check it would never be measured and nothing would say so
	findings := Compare(Baseline{}, resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), []string{"currency", "types"})
	assert.Contains(t, findings, Finding{Missing, "types", "listed package reported no benchmarks; it failed to run, or every benchmark in it skipped"},
		"a listed package with no results should be reported even with no budgets")
}

func TestCompareReportsOrphanedEntry(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode":                      {Allocs: 2, Bytes: 16},
		"exchange/websocket/buffer.BenchmarkBufferPerf":  {Allocs: 1, Bytes: 8},
		"exchange/websocket/buffer.BenchmarkIgnoredPerf": {Allocs: 1, Bytes: 8, Ignore: true, Reason: "noisy"},
	}
	findings := Compare(base, resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), []string{"currency"})
	require.Len(t, findings, 1, "only the unignored orphan must be reported")
	assert.Equal(t, Missing, findings[0].Kind, "an entry whose package is no longer listed should be reported as missing")
	assert.Equal(t, "exchange/websocket/buffer.BenchmarkBufferPerf", findings[0].Key, "the orphaned entry should be named")
	assert.Contains(t, findings[0].Msg, "belongs to no listed package", "the finding should say why nothing will measure it")
}

func TestCompareIgnoredBenchmarkNotReportedMissing(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"exchanges/alert.BenchmarkWait":  {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"},
		"exchanges/alert.BenchmarkAlert": {Allocs: 0, Bytes: 0},
	}
	got := resultsOf(result("exchanges/alert", "BenchmarkAlert", 0, 0))
	assert.Empty(t, Compare(base, got, []string{"exchanges/alert"}),
		"deleting an ignored benchmark should not fail the gate it is exempt from")
}

func TestCompareMetric(t *testing.T) {
	t.Parallel()
	assert.Empty(t, compareMetric("a.B", "B/op", 100, 100, 0), "an unchanged metric should produce no finding")
	assert.Equal(t, []Finding{{Regression, "a.B", "B/op 100 -> 101"}}, compareMetric("a.B", "B/op", 100, 101, 0),
		"a metric above its budget should be a regression")
	assert.Equal(t, []Finding{{Stale, "a.B", "B/op improved 100 -> 99; run 'make bench_update'"}}, compareMetric("a.B", "B/op", 100, 99, 0),
		"a metric below its budget should be stale")
	assert.Empty(t, compareMetric("a.B", "B/op", 100, 101, 0.01), "movement within tolerance should produce no finding")
}

func TestBudgetDrift(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		got, budget uint64
		tolerance   float64
		exp         int
	}{
		{got: 100, budget: 100, exp: 0},
		{got: 101, budget: 100, exp: 1},
		{got: 99, budget: 100, exp: -1},
		{got: 101, budget: 100, tolerance: 0.01, exp: 0},
		{got: 102, budget: 100, tolerance: 0.01, exp: 1},
		{got: 99, budget: 100, tolerance: 0.01, exp: 0},
		{got: 98, budget: 100, tolerance: 0.01, exp: -1},
		{got: 1, budget: 0, tolerance: 0.5, exp: 1},
		{got: 0, budget: 0, tolerance: 0.5, exp: 0},
	} {
		assert.Equalf(t, tc.exp, budgetDrift(tc.got, tc.budget, tc.tolerance),
			"budgetDrift should place %d against a budget of %d with tolerance %v", tc.got, tc.budget, tc.tolerance)
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "regression", Regression.String(), "Regression should stringify")
	assert.Equal(t, "missing", Missing.String(), "Missing should stringify")
	assert.Equal(t, "untracked", Untracked.String(), "Untracked should stringify")
	assert.Equal(t, "stale", Stale.String(), "Stale should stringify")
	assert.Equal(t, "unknown", Kind(255).String(), "an out of range kind should stringify as unknown")
}

func TestFindingString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "regression currency.BenchmarkNewCode: allocs/op 2 -> 3",
		Finding{Regression, "currency.BenchmarkNewCode", "allocs/op 2 -> 3"}.String(), "a finding should render as kind, key and message")
}
