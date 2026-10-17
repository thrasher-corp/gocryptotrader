package main

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// defaultBytesTolerance is the fractional B/op movement ignored when an entry sets none, since a few
// bytes of runtime bookkeeping can move between identical runs
const defaultBytesTolerance = 0.01

// Entry is a benchmark's budget: its median allocs/op and B/op on the canonical platform. The
// tolerances are fractional overrides, BytesTolerance replacing defaultBytesTolerance whenever it is
// set, and Ignore drops the benchmark from the gate; each needs a Reason.
type Entry struct {
	Allocs          uint64   `json:"allocs"`
	Bytes           uint64   `json:"bytes"`
	AllocsTolerance float64  `json:"allocs_tolerance,omitempty"`
	BytesTolerance  *float64 `json:"bytes_tolerance,omitempty"`
	Ignore          bool     `json:"ignore,omitempty"`
	Reason          string   `json:"reason,omitempty"`
}

func (e *Entry) bytesTolerance() float64 {
	if e.BytesTolerance != nil {
		return *e.BytesTolerance
	}
	return defaultBytesTolerance
}

// Baseline maps a benchmark key to its budget
type Baseline map[string]*Entry

var (
	errNullEntry            = errors.New("baseline entry is null")
	errToleranceNeedsReason = errors.New("tolerance override requires a reason")
	errToleranceOutOfRange  = errors.New("tolerance out of range")
	errNoBaseline           = errors.New("baseline file does not exist")
	errDuplicateName        = errors.New("baseline repeats a name")
	errUnknownField         = errors.New("baseline entry has a field Save does not write")
)

// entryFields are the member names Save writes for an entry
var entryFields = map[string]bool{
	"allocs": true, "bytes": true, "allocs_tolerance": true, "bytes_tolerance": true, "ignore": true, "reason": true,
}

// LoadBaseline reads a baseline file. seeding allows the file to be absent, which only an update may;
// a check would otherwise report each measured benchmark as untracked rather than name the file.
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
	if err := checkNames(data); err != nil {
		return nil, fmt.Errorf("%w in %s", err, path)
	}
	if b == nil {
		// A literal null decodes to a nil map, which an update would panic on
		b = Baseline{}
	}
	return b, nil
}

// checkNames rejects what the JSON decoder accepts silently: a name repeated within an object, where
// the last copy wins, and an entry field spelt other than as Save writes it, which the decoder matches
// case-insensitively or ignores.
func checkNames(data []byte) error {
	type object struct {
		names map[string]bool
		key   string // the top-level name whose value this object is
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
				return nil
			}
			next := end + 1
			for next < len(data) && strings.ContainsRune(" \t\r\n", rune(data[next])) {
				next++
			}
			var name string
			if next < len(data) && data[next] == ':' && len(stack) > 0 && stack[len(stack)-1] != nil &&
				json.Unmarshal(data[i:end+1], &name) == nil {
				obj := stack[len(stack)-1]
				if len(stack) > 1 && !entryFields[name] {
					return fmt.Errorf("%w: %s: %q", errUnknownField, obj.key, name)
				}
				if obj.names[name] {
					if len(stack) > 1 {
						return fmt.Errorf("%w: %s: %s", errDuplicateName, obj.key, name)
					}
					return fmt.Errorf("%w: %s", errDuplicateName, name)
				}
				obj.names[name] = true
				if len(stack) == 1 {
					lastName = name
				}
			}
			i = end
		}
	}
	return nil
}

// Validate rejects overrides without a reason and tolerances outside [0, 1)
func (b Baseline) Validate() error {
	for _, key := range sortedKeys(b) {
		e := b[key]
		if e == nil {
			return fmt.Errorf("%w: %s", errNullEntry, key)
		}
		// A blank reason is no reason
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

// validTolerance rejects tolerances outside [0, 1): a finite one of 1 or more can never report an
// improvement, and any other would silently act as zero
func validTolerance(t float64) error {
	if math.IsNaN(t) || t < 0 || t >= 1 {
		return fmt.Errorf("%w: %v is not in [0, 1)", errToleranceOutOfRange, t)
	}
	return nil
}

// Save writes the baseline with sorted keys, renaming a temporary file from the same directory into
// place so an interrupted run cannot truncate it
func (b Baseline) Save(path string) error {
	var data bytes.Buffer
	enc := json.NewEncoder(&data)
	enc.SetIndent("", "  ")
	// Reasons are written by hand, so an update must leave their <, > and & as written
	enc.SetEscapeHTML(false)
	if err := enc.Encode(b); err != nil {
		return fmt.Errorf("error encoding baseline: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("error creating temporary baseline: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp) // No-op once the rename has succeeded
	if _, err := f.Write(data.Bytes()); err != nil {
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

// Update folds results into the baseline and describes each change, in key order. An entry without a
// result is kept when a skip accounts for it, and prune deletes the rest. A skip with no entry is
// named too, since the check that follows fails on it.
func (b Baseline) Update(results map[string]*Result, skipped map[string]string, prune bool) []string {
	var changes []string
	for _, key := range sortedKeys(b) {
		if _, ok := results[key]; ok {
			continue
		}
		if s, ok := skipCovering(skipped, key); ok {
			if s == key {
				changes = append(changes, key+" skipped in this run; keeping it")
			} else {
				changes = append(changes, key+" has no result because "+s+" skipped; keeping it")
			}
			continue
		}
		if !prune {
			changes = append(changes, key+" has no result in this run; keeping it (pass -prune to delete)")
			continue
		}
		delete(b, key)
		changes = append(changes, "pruned "+key)
	}
	for _, s := range sortedKeys(skipped) {
		if _, ok := b[s]; !ok {
			changes = append(changes, s+" skipped in this run and has no entry; 'make bench' reports it until it stops skipping or the baseline ignores it")
		}
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

// updatedBudget returns the budget recorded for a metric measured at got: raised only past the
// tolerance, where the check reports a regression, and lowered once got is more than half the exact
// tolerance below it
func updatedBudget(got, budget uint64, tolerance float64) uint64 {
	exact := exactTolerance(tolerance)
	var half *big.Rat
	if exact != nil {
		half = new(big.Rat).Mul(exact, big.NewRat(1, 2))
	}
	if exactDrift(got, budget, exact) > 0 || exactDrift(got, budget, half) < 0 {
		return got
	}
	return budget
}

// packageOf returns the longest known package key falls under, or "". Package paths and subbenchmark
// names both contain dots, so a key cannot simply be split.
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

// reportedPackages returns the packages that reported a result or a skip
func reportedPackages(results map[string]*Result, skipped map[string]string) map[string]bool {
	reported := seenPackages(results)
	for _, pkg := range skipped {
		reported[pkg] = true
	}
	return reported
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}

func configuredSet(list []string) map[string]bool {
	set := make(map[string]bool, len(list))
	for _, p := range list {
		set[p] = true
	}
	return set
}
