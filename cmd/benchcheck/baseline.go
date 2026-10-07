package main

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// defaultBytesTolerance is the fractional B/op movement ignored when an entry does not set its own.
// B/op is near-reproducible rather than exact: a few bytes of runtime bookkeeping can move between
// runs of an otherwise identical benchmark.
const defaultBytesTolerance = 0.01

// Entry is the recorded budget for a single benchmark: the median allocs/op and B/op measured on
// the canonical platform.
//
// The tolerances are fractional per-benchmark overrides, for counts that are not reproducible.
// AllocsTolerance defaults to zero. BytesTolerance replaces defaultBytesTolerance whenever it is
// set, so an explicit zero asks for an exact match. Ignore drops a benchmark from the gate
// entirely; prefer a tolerance, since an ignored benchmark still costs CI time and tells nobody
// anything. Each requires a Reason.
type Entry struct {
	Allocs          uint64   `json:"allocs"`
	Bytes           uint64   `json:"bytes"`
	AllocsTolerance float64  `json:"allocs_tolerance,omitempty"`
	BytesTolerance  *float64 `json:"bytes_tolerance,omitempty"`
	Ignore          bool     `json:"ignore,omitempty"`
	Reason          string   `json:"reason,omitempty"`
}

// bytesTolerance returns the B/op tolerance that applies to the entry
func (e *Entry) bytesTolerance() float64 {
	if e.BytesTolerance != nil {
		return *e.BytesTolerance
	}
	return defaultBytesTolerance
}

// Baseline maps a benchmark key to its recorded budget
type Baseline map[string]*Entry

var (
	errNullEntry            = errors.New("baseline entry is null")
	errToleranceNeedsReason = errors.New("tolerance override requires a reason")
	errToleranceOutOfRange  = errors.New("tolerance out of range")
	errNoBaseline           = errors.New("baseline file does not exist")
	errDuplicateName        = errors.New("baseline repeats a name")
)

// LoadBaseline reads a baseline file. seeding permits an absent file so the first -update can
// create one; in check mode it must be an error, since an empty baseline passes any numbers at all.
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
	if name, ok := duplicateName(data); ok {
		return nil, fmt.Errorf("%w: %s in %s", errDuplicateName, name, path)
	}
	if b == nil {
		// A file containing literal null unmarshals to a nil map, which -update would then panic on
		b = Baseline{}
	}
	return b, nil
}

// duplicateName returns the first name repeated within one object of data, a baseline that has
// already decoded cleanly: a benchmark key, or an entry's field qualified by its benchmark. The JSON
// package keeps the last of two members sharing a name, and matches an entry's fields without
// regard to case, so a repeat would replace a budget or widen a tolerance while its diff read as
// something being added.
func duplicateName(data []byte) (string, bool) {
	type object struct {
		names map[string]bool
		key   string // the top-level name this object is the value of
	}
	var stack []*object // nil for an array
	var lastName string
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '{':
			stack = append(stack, &object{names: make(map[string]bool), key: lastName})
		case '[':
			stack = append(stack, nil)
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case '"':
			end := i + 1
			for ; end < len(data) && data[end] != '"'; end++ {
				if data[end] == '\\' {
					end++
				}
			}
			if end >= len(data) {
				return "", false
			}
			next := end + 1
			for next < len(data) && strings.ContainsRune(" \t\r\n", rune(data[next])) {
				next++
			}
			var name string
			if next < len(data) && data[next] == ':' && len(stack) > 0 && stack[len(stack)-1] != nil &&
				json.Unmarshal(data[i:end+1], &name) == nil {
				obj := stack[len(stack)-1]
				// Benchmark keys are matched exactly, an entry's fields without regard to case
				folded := name
				if len(stack) > 1 {
					folded = strings.ToLower(name)
				}
				if obj.names[folded] {
					if len(stack) > 1 {
						return obj.key + ": " + name, true
					}
					return name, true
				}
				obj.names[folded] = true
				if len(stack) == 1 {
					lastName = name
				}
			}
			i = end
		}
	}
	return "", false
}

// Validate rejects tolerance overrides that carry no justification, so that widening a budget
// cannot be slipped through as a one-character diff
func (b Baseline) Validate() error {
	for _, key := range sortedKeys(b) {
		e := b[key]
		if e == nil {
			return fmt.Errorf("%w: %s", errNullEntry, key)
		}
		// Trimmed, or " " buys the same exemption as a justification while reviewing as an empty diff
		if (e.AllocsTolerance != 0 || e.BytesTolerance != nil || e.Ignore) && strings.TrimSpace(e.Reason) == "" {
			return fmt.Errorf("%w: %s", errToleranceNeedsReason, key)
		}
		if err := validTolerance(e.AllocsTolerance); err != nil {
			return fmt.Errorf("%s allocs_tolerance: %w", key, err)
		}
		if err := validTolerance(e.bytesTolerance()); err != nil {
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

// Save writes the baseline with sorted keys and a trailing newline so diffs stay reviewable. It
// renames a temporary file from the same directory into place, which keeps an interrupted run from
// truncating the baseline; the same directory because rename is only atomic within a filesystem.
// Concurrent updates are still unsafe - the last rename wins.
func (b Baseline) Save(path string) error {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("error encoding baseline: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("error creating temporary baseline: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp) // No-op once the rename has succeeded
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return fmt.Errorf("error writing temporary baseline: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("error closing temporary baseline: %w", err)
	}
	if err := os.Chmod(tmp, 0o644); err != nil {
		return fmt.Errorf("error setting baseline permissions: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("error replacing baseline: %w", err)
	}
	return nil
}

// Update folds results into the baseline and returns a line describing each change, in key order.
//
// A new benchmark gains an entry, and an existing one has each metric moved as updatedBudget decides.
// Ignored entries are left as they are.
//
// prune deletes every entry without a result: in a listed package its benchmark was renamed or
// deleted, and outside one its package was. Callers prune only on a run that covered every listed
// package, or a package that merely did not run loses every budget it had.
func (b Baseline) Update(results map[string]*Result, prune bool) []string {
	var changes []string
	for _, key := range sortedKeys(b) {
		if _, ok := results[key]; ok {
			continue
		}
		if !prune {
			changes = append(changes, key+" has no result in this run; keeping it (pass -prune to delete)")
			continue
		}
		delete(b, key)
		changes = append(changes, "pruned "+key)
	}
	for _, key := range sortedKeys(results) {
		r := results[key]
		e, ok := b[key]
		if !ok {
			b[key] = &Entry{Allocs: r.Allocs, Bytes: r.Bytes}
			changes = append(changes, fmt.Sprintf("added %s: allocs/op %d, B/op %d", key, r.Allocs, r.Bytes))
			continue
		}
		if e.Ignore {
			continue
		}
		if budget := updatedBudget(r.Allocs, e.Allocs, e.AllocsTolerance); budget != e.Allocs {
			changes = append(changes, fmt.Sprintf("%s allocs/op %d -> %d", key, e.Allocs, budget))
			e.Allocs = budget
		}
		if budget := updatedBudget(r.Bytes, e.Bytes, e.bytesTolerance()); budget != e.Bytes {
			changes = append(changes, fmt.Sprintf("%s B/op %d -> %d", key, e.Bytes, budget))
			e.Bytes = budget
		}
	}
	return changes
}

// updatedBudget returns the budget an update records for a metric measured at got. It rises only past
// the tolerance, where Compare reports a regression, so a budget never loosens without a finding
// saying so, and drift too small to report on any one run accumulates against a fixed budget. It
// falls once got is more than half the tolerance below it, or on any decrease when the tolerance is
// zero, which keeps run-to-run noise out of the diff while leaving every measurement at least half a
// tolerance clear of reading as stale.
func updatedBudget(got, budget uint64, tolerance float64) uint64 {
	if budgetDrift(got, budget, tolerance) > 0 || budgetDrift(got, budget, tolerance/2) < 0 {
		return got
	}
	return budget
}

// packageOf resolves a baseline key to one of the known packages, returning "" when none matches.
// Keys cannot be split on the final dot: both package paths and subbenchmark names contain dots,
// so matching against the known set is the only unambiguous attribution. The longest match wins, so
// a key under "a/b.c" is never attributed to a package "a/b" that happens to be listed too.
func packageOf(key string, known map[string]bool) string {
	var match string
	for pkg := range known {
		if len(pkg) > len(match) && strings.HasPrefix(key, pkg+".") {
			match = pkg
		}
	}
	return match
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
