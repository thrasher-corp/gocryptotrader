package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Kind classifies a comparison finding
type Kind uint8

// Finding kinds, most actionable first
const (
	// Regression is a benchmark allocating more than its budget
	Regression Kind = iota
	// Missing is a budget or listed package the run measured nothing for
	Missing
	// Skipped is a benchmark go test reported skipping
	Skipped
	// Untracked is a measured benchmark with no budget
	Untracked
	// Stale is a benchmark that beat its budget, which the check reports without failing
	Stale
)

// String returns the kind's name as a report prints it
func (k Kind) String() string {
	switch k {
	case Regression:
		return "regression"
	case Missing:
		return "missing"
	case Skipped:
		return "skipped"
	case Untracked:
		return "untracked"
	case Stale:
		return "stale"
	default:
		return "unknown"
	}
}

// Finding is a discrepancy between a run and the baseline
type Finding struct {
	Kind Kind
	Key  string
	Msg  string
}

// String renders the finding as one line of the report
func (f Finding) String() string {
	return fmt.Sprintf("%-10s %s: %s", f.Kind, f.Key, f.Msg)
}

// Compare checks results against the baseline. A listed package that reported nothing is one
// finding, and a skip is one finding that also covers the budgets beneath it, since the benchmark
// still exists.
func Compare(base Baseline, results map[string]*Result, skipped map[string]string, listed []string) []Finding {
	var findings []Finding
	for _, pkg := range silentPackages(results, skipped, listed) {
		findings = append(findings, Finding{
			Missing, pkg,
			"listed package reported no benchmarks; it failed to run, or every benchmark in it skipped",
		})
	}
	for _, key := range sortedKeys(skipped) {
		if e, ok := base[key]; ok && e.Ignore {
			continue
		}
		findings = append(findings, Finding{Skipped, key, "benchmark skipped, so this run measured nothing for it; " +
			"stop it skipping, or ignore it in the baseline with a reason"})
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
	reported := reportedPackages(results, skipped)
	for _, key := range sortedKeys(base) {
		// Deleting an ignored benchmark must not fail the gate it is exempt from
		if _, ok := results[key]; ok || base[key].Ignore {
			continue
		}
		if _, ok := skipCovering(skipped, key); ok {
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

func compareMetric(key, unit string, budget, got uint64, tolerance float64) []Finding {
	switch budgetDrift(got, budget, tolerance) {
	case 1:
		return []Finding{{Regression, key, fmt.Sprintf("%s %d -> %d", unit, budget, got)}}
	case -1:
		return []Finding{{Stale, key, fmt.Sprintf("%s improved %d -> %d; run 'make bench_update'", unit, budget, got)}}
	}
	return nil
}

// budgetDrift reports which side of its budget a measurement falls once the tolerance is applied: 1
// above it, -1 below it and 0 within it, edges included
func budgetDrift(got, budget uint64, tolerance float64) int {
	return exactDrift(got, budget, exactTolerance(tolerance))
}

// exactTolerance returns a tolerance as the exact decimal the baseline spells, since float arithmetic
// puts 115 outside 15% of 100. NaN and the infinities give nil.
func exactTolerance(tolerance float64) *big.Rat {
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(tolerance, 'g', -1, 64))
	if !ok {
		return nil
	}
	return r
}

// exactDrift is budgetDrift for an exact tolerance, nil counting as none
func exactDrift(got, budget uint64, tolerance *big.Rat) int {
	var distance uint64
	var drift int
	switch {
	case got > budget:
		distance, drift = got-budget, 1
	case got < budget:
		distance, drift = budget-got, -1
	default:
		return 0
	}
	if tolerance == nil {
		return drift
	}
	allowance := new(big.Rat).Mul(tolerance, new(big.Rat).SetUint64(budget))
	if new(big.Rat).SetUint64(distance).Cmp(allowance) > 0 {
		return drift
	}
	return 0
}

// skipCovering returns the first skip, in key order, that is key or a benchmark key falls under
func skipCovering(skipped map[string]string, key string) (string, bool) {
	for _, s := range sortedKeys(skipped) {
		if key == s || strings.HasPrefix(key, s+"/") {
			return s, true
		}
	}
	return "", false
}
