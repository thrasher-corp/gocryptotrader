package main

import (
	"cmp"
	"fmt"
)

// Kind classifies a comparison finding
type Kind uint8

// Finding kinds, ordered so that the most actionable sort first
const (
	// Regression is a benchmark that now allocates more than its recorded budget
	Regression Kind = iota
	// Missing is a baseline entry whose benchmark was not produced by a package that did run
	Missing
	// Untracked is a benchmark in a gated package with no baseline entry
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

// Compare checks results against the baseline. Regressions fail because the code got worse;
// stale entries fail because the code got better and the budget must be tightened to keep the
// improvement, which is what makes the baseline ratchet downwards rather than act as a ceiling that
// only ever moves up.
// configured is the set of packages the run was meant to cover, from packages.txt. Missing
// detection uses it rather than the packages that actually reported: a package whose last benchmark
// is deleted produces no results at all, so keying off what reported would let its baseline entry
// linger unnoticed. A deliberately filtered run will therefore report the benchmarks it skipped.
func Compare(base Baseline, results map[string]*Result, configured, gated map[string]bool, bytesTolerance float64) []Finding {
	var findings []Finding
	for _, key := range sortedKeys(results) {
		r := results[key]
		entry, ok := base[key]
		if !ok {
			if gated[r.Pkg] {
				findings = append(findings, Finding{Untracked, key, fmt.Sprintf(
					"benchmark in gated package %s has no baseline entry; run 'make bench_update'", r.Pkg)})
			}
			continue
		}
		if entry.Ignore {
			continue
		}
		findings = append(findings, compareMetric(key, "allocs/op", entry.Allocs, r.Allocs, entry.AllocsTolerance)...)
		findings = append(findings, compareMetric(key, "B/op", entry.Bytes, r.Bytes, cmp.Or(entry.BytesTolerance, bytesTolerance))...)
	}

	// Fall back to the packages that reported when no package list is configured, so a bare
	// invocation with no packages.txt still detects renames within the packages it did see.
	scope := configured
	if len(scope) == 0 {
		scope = seenPackages(results)
	}
	for _, key := range sortedKeys(base) {
		if _, ok := results[key]; ok || packageOf(key, scope) == "" {
			continue
		}
		// Ignore drops a benchmark from the gate entirely, which has to include the missing check;
		// otherwise deleting an ignored benchmark fails the build it was meant to be exempt from
		if base[key].Ignore {
			continue
		}
		findings = append(findings, Finding{Missing, key, "baseline entry has no matching benchmark; " +
			"it was renamed or deleted, run 'make bench_update'"})
	}
	return findings
}

// compareMetric checks one metric's median against its budget, ignoring movement inside tolerance
func compareMetric(key, unit string, budget uint64, samples []float64, tolerance float64) []Finding {
	if len(samples) == 0 {
		return nil
	}
	got := uint64(median(samples))
	switch {
	case float64(got) > float64(budget)*(1+tolerance):
		return []Finding{{Regression, key, fmt.Sprintf("%s %d -> %d", unit, budget, got)}}
	case float64(got) < float64(budget)*(1-tolerance):
		return []Finding{{Stale, key, fmt.Sprintf("%s improved %d -> %d; run 'make bench_update'", unit, budget, got)}}
	}
	return nil
}
