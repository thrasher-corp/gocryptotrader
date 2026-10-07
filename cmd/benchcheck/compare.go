package main

import "fmt"

// Kind classifies a comparison finding
type Kind uint8

// Finding kinds, ordered so that the most actionable sort first
const (
	// Regression is a benchmark that now allocates more than its recorded budget
	Regression Kind = iota
	// Missing is a budget or listed package that the run produced no measurement for
	Missing
	// Untracked is a measured benchmark with no budget
	Untracked
	// Stale is a benchmark that beat its recorded budget; the baseline needs tightening
	Stale
)

func (k Kind) String() string {
	switch k {
	case Regression:
		return "regression"
	case Missing:
		return "missing"
	case Untracked:
		return "untracked"
	case Stale:
		return "stale"
	default:
		return "unknown"
	}
}

// Finding is a single discrepancy between a benchmark run and the baseline
type Finding struct {
	Kind Kind
	Key  string
	Msg  string
}

func (f Finding) String() string {
	return fmt.Sprintf("%-10s %s: %s", f.Kind, f.Key, f.Msg)
}

// Compare checks results against the baseline. A regression fails because the code got worse, a
// stale entry because it got better and the budget must be tightened to keep the improvement, which
// is what makes the baseline ratchet down rather than act as a ceiling.
//
// listed is the package list, every package of which is expected to report. One that reported
// nothing is a single finding rather than one per budget: it failed to run, every benchmark in it
// skipped, or the run deliberately covered other packages. A budget belonging to no listed package
// is reported too, since nothing will measure it again.
func Compare(base Baseline, results map[string]*Result, listed []string) []Finding {
	var findings []Finding
	for _, pkg := range silentPackages(results, listed) {
		findings = append(findings, Finding{
			Missing, pkg,
			"listed package reported no benchmarks; it failed to run, or every benchmark in it skipped",
		})
	}

	for _, key := range sortedKeys(results) {
		r := results[key]
		entry, ok := base[key]
		if !ok {
			findings = append(findings, Finding{Untracked, key, "benchmark has no baseline entry; run 'make bench_update'"})
			continue
		}
		if entry.Ignore {
			continue
		}
		findings = append(findings, compareMetric(key, "allocs/op", entry.Allocs, r.Allocs, entry.AllocsTolerance)...)
		findings = append(findings, compareMetric(key, "B/op", entry.Bytes, r.Bytes, entry.bytesTolerance())...)
	}

	configured := configuredSet(listed)
	reported := seenPackages(results)
	for _, key := range sortedKeys(base) {
		// Ignore drops a benchmark from the gate entirely, which has to include the missing check;
		// otherwise deleting an ignored benchmark fails the build it was meant to be exempt from
		if _, ok := results[key]; ok || base[key].Ignore {
			continue
		}
		switch pkg := packageOf(key, configured); {
		case pkg == "":
			findings = append(findings, Finding{Missing, key, "baseline entry belongs to no listed package; " +
				"the package was renamed, excluded or removed, run 'make bench_update'"})
		case reported[pkg]:
			findings = append(findings, Finding{Missing, key, "baseline entry has no matching benchmark; " +
				"it was renamed or deleted, run 'make bench_update'"})
		}
	}
	return findings
}

// compareMetric checks one metric's median against its budget, ignoring movement inside tolerance
func compareMetric(key, unit string, budget, got uint64, tolerance float64) []Finding {
	switch budgetDrift(got, budget, tolerance) {
	case 1:
		return []Finding{{Regression, key, fmt.Sprintf("%s %d -> %d", unit, budget, got)}}
	case -1:
		return []Finding{{Stale, key, fmt.Sprintf("%s improved %d -> %d; run 'make bench_update'", unit, budget, got)}}
	}
	return nil
}

// budgetDrift reports which side of its budget a measurement falls once the fractional tolerance is
// applied: 1 above it, -1 below it and 0 within it
func budgetDrift(got, budget uint64, tolerance float64) int {
	switch {
	case float64(got) > float64(budget)*(1+tolerance):
		return 1
	case float64(got) < float64(budget)*(1-tolerance):
		return -1
	}
	return 0
}
