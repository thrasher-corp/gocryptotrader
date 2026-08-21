package main

import (
	"bufio"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Entry is the recorded budget for a single benchmark. Allocs and Bytes are gated; NSHint is
// advisory, because ns/op on a shared CI runner is too noisy to compare against a stored value.
//
// The tolerances are fractional per-benchmark overrides, for counts that are not reproducible.
// Ignore drops a benchmark from the gate entirely; prefer a tolerance, since an ignored benchmark
// still costs CI time and tells nobody anything. Both require a Reason.
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
	errEmptyPackageList     = errors.New("package list names no packages")
	errBadPackagePath       = errors.New("package path is not a package inside this module")
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
		// Trimmed, or " " buys the same exemption as a justification while reviewing as an empty diff
		if (e.AllocsTolerance != 0 || e.BytesTolerance != 0 || e.Ignore) && strings.TrimSpace(e.Reason) == "" {
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

// Update folds results into the baseline, preserving any hand-written reason, and returns the keys
// it pruned. Callers set allowPrune only when the run covered every configured package: a filtered
// run reports a subset, and pruning on that evidence deletes the rest of the package.
//
// scope is the set of packages the run was meant to cover, and must be the same set Compare uses to
// report a benchmark missing, or a package whose last benchmark was deleted is failed by one and
// unreachable by the other.
func (b Baseline) Update(results map[string]*Result, allowPrune bool, scope map[string]bool) []string {
	var pruned []string
	if len(scope) == 0 {
		scope = seenPackages(results)
	}
	if allowPrune {
		for _, key := range sortedKeys(b) {
			if _, ok := results[key]; !ok && packageOf(key, scope) != "" {
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
// module root, optionally followed by the word "gated"; blank lines and # comments are ignored.
// Entries are reduced to the form go test reports, via canonicalPackage.
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
		pkg, err := canonicalPackage(fields[0])
		if err != nil {
			return nil, err
		}
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
	// Same reasoning as an absent file: a list that names nothing gates nothing, and -list then
	// hands go test no packages at all
	if len(pkgs.List) == 0 {
		return nil, fmt.Errorf("%w: %s", errEmptyPackageList, path)
	}
	return pkgs, nil
}

// canonicalPackage reduces a package list entry to the path go test prints in its pkg: header,
// which is the only form the gated and configured lookups ever see. An entry that resolves to a
// different string runs its benchmarks and gates none of them.
//
// Cleaning handles redundant separators and "." elements. What it cannot resolve is refused: "..."
// covers several packages under one lookup key, and a path leaving the module has no header to
// match. So is anything outside a Go import path's character set, because -list is interpolated
// into go test unquoted and the shell rewrites the rest - "'currency'" loses its quotes,
// "currenc[y]" globs.
func canonicalPackage(entry string) (string, error) {
	root := strings.TrimSuffix(modulePrefix, "/")
	pkg := pathpkg.Clean(entry)
	// Cleaning erases the module boundary, so a module-qualified entry is checked against it before
	// the prefix comes off: ".../gocryptotrader/../../../currency" cleans to a real package and
	// would otherwise be accepted as one, silently naming something the entry never spelled.
	if strings.HasPrefix(entry, root) && pkg != root && !strings.HasPrefix(pkg, modulePrefix) {
		return "", fmt.Errorf("%w: %q traverses outside the module", errBadPackagePath, entry)
	}
	if pkg == root || pkg == "." {
		return "", fmt.Errorf("%w: %q resolves to the module root", errBadPackagePath, entry)
	}
	pkg = strings.TrimPrefix(pkg, modulePrefix)
	switch {
	case strings.Contains(pkg, "..."):
		return "", fmt.Errorf("%w: %q is a wildcard, which covers several packages under one entry", errBadPackagePath, entry)
	case strings.HasPrefix(pkg, "/"), strings.HasPrefix(pkg, ".."):
		return "", fmt.Errorf("%w: %q resolves outside the module", errBadPackagePath, entry)
	}
	for _, r := range pkg {
		if !importPathRune(r) {
			return "", fmt.Errorf("%w: %q contains %q, which the shell would rewrite before go test saw it",
				errBadPackagePath, entry, string(r))
		}
	}
	return pkg, nil
}

// importPathRune reports whether r may appear in a Go import path. The set is x/mod/module's
// importPathOK, which is alphanumerics plus "-._~+", together with the element separator. None of
// those carry meaning to a shell, which is the property the caller needs.
func importPathRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '/', r == '.', r == '-', r == '_', r == '~', r == '+':
		return true
	}
	return false
}

// packageOf resolves a baseline key to one of the known packages, returning "" when none matches.
// Keys cannot be split on the final dot: both package paths and subbenchmark names contain dots,
// so matching against the known set is the only unambiguous attribution.
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
