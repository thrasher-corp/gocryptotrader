package main

import (
	"bufio"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Entry is the recorded budget for a single benchmark. Allocs and Bytes are gated; NSHint is
// advisory only, because ns/op on a shared CI runner is too noisy to compare against a stored value.
//
// AllocsTolerance and BytesTolerance are fractional per-benchmark overrides for benchmarks whose
// counts are not reproducible — typically those that spawn goroutines, where the scheduler decides
// how much of the work is attributed to the measured iterations.
//
// Ignore drops a benchmark from the gate entirely and stops -update from recording values for it,
// for benchmarks too non-deterministic to carry any budget. Prefer a tolerance where one works: an
// ignored benchmark is measured by CI but tells nobody anything.
//
// Ignore and both tolerances must carry a Reason.
type Entry struct {
	Allocs          uint64  `json:"allocs"`
	Bytes           uint64  `json:"bytes"`
	NSHint          float64 `json:"ns_hint,omitempty"`
	AllocsTolerance float64 `json:"allocs_tolerance,omitempty"`
	BytesTolerance  float64 `json:"bytes_tolerance,omitempty"`
	Ignore          bool    `json:"ignore,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}

// Baseline maps a benchmark key to its recorded budget
type Baseline map[string]*Entry

var (
	errNullEntry            = errors.New("baseline entry is null")
	errToleranceNeedsReason = errors.New("tolerance override requires a reason")
	errToleranceOutOfRange  = errors.New("tolerance out of range")
	errUnknownMarker        = errors.New("unknown package marker")
	errNoBaseline           = errors.New("baseline file does not exist")
	errNoPackageList        = errors.New("package list does not exist")
)

// LoadBaseline reads a baseline file. seeding permits an absent file, so the first -update run can
// create one; in check mode an absent file must be an error, because an empty baseline compares
// clean against any numbers at all and a mistyped -baseline path would silently pass everything.
func LoadBaseline(path string, seeding bool) (Baseline, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if seeding {
			return Baseline{}, nil
		}
		return nil, fmt.Errorf("%w: %s", errNoBaseline, path)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading baseline: %w", err)
	}
	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("error parsing baseline %q: %w", path, err)
	}
	if b == nil {
		// A file containing literal null unmarshals to a nil map, which -update would then panic on
		b = Baseline{}
	}
	return b, nil
}

// Validate rejects tolerance overrides that carry no justification, so that widening a budget
// cannot be slipped through as a one-character diff
func (b Baseline) Validate() error {
	for _, key := range sortedKeys(b) {
		e := b[key]
		if e == nil {
			return fmt.Errorf("%w: %s", errNullEntry, key)
		}
		if (e.AllocsTolerance != 0 || e.BytesTolerance != 0 || e.Ignore) && e.Reason == "" {
			return fmt.Errorf("%w: %s", errToleranceNeedsReason, key)
		}
		if err := validTolerance(e.AllocsTolerance); err != nil {
			return fmt.Errorf("%s allocs_tolerance: %w", key, err)
		}
		if err := validTolerance(e.BytesTolerance); err != nil {
			return fmt.Errorf("%s bytes_tolerance: %w", key, err)
		}
	}
	return nil
}

// validTolerance rejects tolerances that would invert or disable the gate. A negative value reports
// an unchanged benchmark as a regression; a value of 1 or more puts the lower bound at or below zero
// so the ratchet can never fire; NaN makes every comparison false.
func validTolerance(t float64) error {
	if math.IsNaN(t) || t < 0 || t >= 1 {
		return fmt.Errorf("%w: %v is not in [0, 1)", errToleranceOutOfRange, t)
	}
	return nil
}

// Save writes the baseline with sorted keys and a trailing newline so diffs stay reviewable
func (b Baseline) Save(path string) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding baseline: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Update folds results into the baseline, preserving any hand-written reason. Benchmarks that have
// disappeared from a package present in the results are removed; packages absent from the results
// are left untouched so that updating from a partial run does not discard unrelated entries.
// A filtered run (-bench selecting one benchmark, or a package that failed early) reports only a
// subset, and pruning on that evidence would delete every other entry in the package. Pruning is
// therefore gated on allowPrune, which callers set only when the run covered every configured
// package; a partial run updates the values it measured and leaves the rest alone.
// Pruned keys are returned rather than silently dropped. A benchmark that skips at runtime emits no
// result at all — Go only marks it under -v — so it is indistinguishable here from one that was
// deleted, and its budget would vanish without trace. Reporting makes that visible in CI output.
func (b Baseline) Update(results map[string]*Result, allowPrune bool) []string {
	var pruned []string
	seen := seenPackages(results)
	if allowPrune {
		for _, key := range sortedKeys(b) {
			if _, ok := results[key]; !ok && packageOf(key, seen) != "" {
				delete(b, key)
				pruned = append(pruned, key)
			}
		}
	}
	for key, r := range results {
		e, ok := b[key]
		if !ok {
			e = &Entry{}
			b[key] = e
		}
		if e.Ignore {
			continue
		}
		e.Allocs = uint64(median(r.Allocs))
		e.Bytes = uint64(median(r.Bytes))
		e.NSHint = roundNS(median(r.NS))
	}
	return pruned
}

// roundNS trims ns/op to three significant-ish digits; it is advisory, and storing full precision
// would produce baseline churn on every update for no benefit
func roundNS(ns float64) float64 {
	switch {
	case ns >= 1000:
		return float64(int64(ns))
	case ns >= 10:
		return float64(int64(ns*10)) / 10
	default:
		return float64(int64(ns*100)) / 100
	}
}

// Packages is the benchmarked package list, in file order, with the packages whose benchmarks must
// all appear in the baseline flagged as gated
type Packages struct {
	List  []string
	Gated map[string]bool
}

// LoadPackages reads the benchmarked package list. Each line is a package path relative to the
// module root, optionally followed by the word "gated"; blank lines, # comments and trailing
// # comments are ignored.
func LoadPackages(path string) (*Packages, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		// Not tolerated: an empty list leaves Gated empty, so a mistyped -packages path would let a
		// new benchmark in a gated package through with no baseline entry and no finding.
		return nil, fmt.Errorf("%w: %s", errNoPackageList, path)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading package list: %w", err)
	}
	defer f.Close()

	pkgs := &Packages{Gated: map[string]bool{}}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg := strings.TrimPrefix(fields[0], modulePrefix)
		if len(fields) > 1 {
			if fields[1] != "gated" {
				return nil, fmt.Errorf("%w: %q on package %q", errUnknownMarker, fields[1], pkg)
			}
			pkgs.Gated[pkg] = true
		}
		pkgs.List = append(pkgs.List, pkg)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("error reading package list: %w", err)
	}
	return pkgs, nil
}

// packageOf resolves a baseline key to one of the packages that actually reported results. Keys
// cannot be split on the final dot: package paths legitimately contain dots (for example
// currency/forexprovider/exchangeratesapi.io) and so do subbenchmark names, so matching against the
// known set is the only unambiguous attribution. An unmatched key belongs to a package that did not
// run, which is exactly what the callers need to know.
func packageOf(key string, known map[string]bool) string {
	for pkg := range known {
		if strings.HasPrefix(key, pkg+".") {
			return pkg
		}
	}
	return ""
}

func seenPackages(results map[string]*Result) map[string]bool {
	seen := make(map[string]bool)
	for _, r := range results {
		seen[r.Pkg] = true
	}
	return seen
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

// configuredSet turns the ordered package list into a set for lookups
func configuredSet(list []string) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, p := range list {
		set[p] = true
	}
	return set
}
