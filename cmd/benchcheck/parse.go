package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/token"
	"io"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	errNoPackageContext = errors.New("benchmark result with no preceding pkg: header")
	errNoMemoryMetrics  = errors.New("benchmark result without B/op and allocs/op")
	errBadMeasurement   = errors.New("benchmark measurement is not a finite, non-negative number")
	errRunFailed        = errors.New("go test reported a failure")
	errMixedPlatforms   = errors.New("benchmark output comes from more than one platform")
	errSampleCount      = errors.New("benchmark did not produce the expected number of samples")
	errEvenSampleCount  = errors.New("a median needs an odd number of samples")
	errMalformedResult  = errors.New("benchmark result line is malformed")
	errMixedProcs       = errors.New("benchmark results end in different -N suffixes")
)

const modulePrefix = "github.com/thrasher-corp/gocryptotrader/"

// Run holds what one `go test -bench` run reported
type Run struct {
	// Platform is the goos/goarch pair from the output's headers, such as "linux/amd64"
	Platform string
	Samples  map[string]*Samples
	// Skipped maps the key of each benchmark that skipped to its package
	Skipped map[string]string
	// Finished holds the packages whose output reached go test's ok line
	Finished map[string]bool
	// Verbose holds the packages whose output came from go test -v
	Verbose map[string]bool
	// Headers holds each package a pkg: header named
	Headers map[string]bool
	// SkipRecords maps the key of each benchmark named by skip record text ending a line, go test's
	// or not, to the package it appeared under
	SkipRecords map[string]string
}

// Samples holds every measurement recorded for one benchmark across all -count runs. NS is only read by
// -compare, which validates it; the gate never reads it, so a bad value cannot fail a check.
type Samples struct {
	Pkg    string
	Name   string
	NS     []float64
	Bytes  []float64
	Allocs []float64
}

// Key returns the baseline key, such as "exchanges/orderbook.BenchmarkProcess"
func (s *Samples) Key() string {
	return s.Pkg + "." + s.Name
}

// Result holds a benchmark's median measurements
type Result struct {
	Pkg    string
	Name   string
	Allocs uint64
	Bytes  uint64
}

// Key returns the baseline key, such as "exchanges/orderbook.BenchmarkProcess"
func (r *Result) Key() string {
	return r.Pkg + "." + r.Name
}

// Parse reads `go test -bench` output and gathers each benchmark's samples. Results are attributed
// to packages by go test's pkg: headers, so the output must not be filtered.
func Parse(r io.Reader) (*Run, error) {
	run := &Run{
		Samples:     make(map[string]*Samples),
		Skipped:     make(map[string]string),
		Finished:    make(map[string]bool),
		Verbose:     make(map[string]bool),
		Headers:     make(map[string]bool),
		SkipRecords: make(map[string]string),
	}
	var pkg, goos, goarch, procs string
	var seenProcs bool
	var announced []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		// A failed run still prints a result for each sample it completed
		if failureRecord(line) {
			return nil, fmt.Errorf("%w: %q", errRunFailed, line)
		}
		// Checked first, since printed text joined to a result can make the line read as another record
		if embeddedResult(line, announced) {
			return nil, fmt.Errorf("%w: %q has output run into its start", errMalformedResult, line)
		}
		if name, ok := skipRecordName(line); ok {
			run.SkipRecords[pkg+"."+name] = pkg
		}
		// A logged "ok" is indented, so only one starting the line is go test's
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "ok" && strings.HasPrefix(line, "ok") {
			run.Finished[strings.TrimPrefix(f[1], modulePrefix)] = true
			continue
		}
		// After a package's ok line, text before the next pkg: header is the next package's TestMain's
		if name, ok := embeddedSkip(line, announced); ok && !run.Finished[pkg] {
			line = "--- SKIP: " + name
		}
		if v, ok := strings.CutPrefix(line, "--- SKIP: "); ok {
			f := strings.Fields(v)
			if len(f) == 0 {
				return nil, fmt.Errorf("%w: %q", errMalformedResult, line)
			}
			// A test's skip, when tests ran too
			if !strings.HasPrefix(f[0], "Benchmark") {
				continue
			}
			if pkg == "" {
				return nil, fmt.Errorf("%w: %s", errNoPackageContext, f[0])
			}
			run.Skipped[pkg+"."+f[0]] = pkg
			continue
		}
		if v, ok := strings.CutPrefix(line, "goos:"); ok {
			goos = strings.TrimSpace(v)
			continue
		}
		if v, ok := strings.CutPrefix(line, "goarch:"); ok {
			goarch = strings.TrimSpace(v)
			continue
		}
		if p, ok := strings.CutPrefix(line, "pkg:"); ok {
			// go test prints an import path, so anything else is text a benchmark printed
			if p = strings.TrimSpace(p); p == "" || strings.ContainsFunc(p, func(r rune) bool { return !importPathRune(r) }) {
				return nil, fmt.Errorf("%w: %q is not a package header", errMalformedResult, line)
			}
			recordUnreported(run, pkg, announced)
			pkg = strings.TrimPrefix(p, modulePrefix)
			// go test prints each package's header once, so a repeat is text a benchmark printed
			if run.Headers[pkg] {
				return nil, fmt.Errorf("%w: %q repeats a package header", errMalformedResult, line)
			}
			run.Headers[pkg] = true
			announced = nil
			if goos != "" || goarch != "" {
				platform := goos + "/" + goarch
				if run.Platform != "" && run.Platform != platform {
					return nil, fmt.Errorf("%w: %s and %s", errMixedPlatforms, run.Platform, platform)
				}
				run.Platform = platform
			}
			continue
		}
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		// -v announces each benchmark by name on a line of its own
		if f := strings.Fields(line); len(f) == 1 {
			// go test writes a result's padded name and its measurements separately, so a name with
			// whitespace after it is a result that a printed newline split in two
			if f[0] != line {
				return nil, fmt.Errorf("%w: %q is cut short", errMalformedResult, line)
			}
			if !benchmarkName(f[0]) {
				continue
			}
			if pkg != "" {
				run.Verbose[pkg] = true
			}
			announced = append(announced, strings.TrimSpace(line))
			continue
		}
		// Only go test starts a line with a benchmark's name, so one that fails to parse is a result
		// something printed into, and skipping it would drop the sample silently
		result, ok := parseBenchLine(line, announced)
		if !ok {
			return nil, fmt.Errorf("%w: %q", errMalformedResult, line)
		}
		name, metrics := result.name, result.metrics
		// An unattributed result matches no budget, so it would pass the gate
		if pkg == "" {
			return nil, fmt.Errorf("%w: %s", errNoPackageContext, name)
		}
		// -v announces every benchmark before its results, so any other name is printed text joined
		// to a result
		if len(announced) > 0 && !slices.Contains(announced, name) {
			return nil, fmt.Errorf("%w: %q names no benchmark -v announced (is a benchmark writing to stdout or stderr?)", errMalformedResult, line)
		}
		// Every result of a run carries the same suffix, so differing ones mean a name ending in digits
		// was misread as one, as under -cpu 1
		if seenProcs && result.procs != procs {
			return nil, fmt.Errorf("%w: %s and %s, at %s.%s; run with one -cpu value above 1, as BENCH_FLAGS does",
				errMixedProcs, describeProcSuffix(procs), describeProcSuffix(result.procs), pkg, name)
		}
		procs, seenProcs = result.procs, true
		// A benchmark that skips after go test's first round reports zero iterations, not a skip record
		if result.iterations == 0 {
			run.Skipped[pkg+"."+name] = pkg
			continue
		}
		s, ok := run.Samples[pkg+"."+name]
		if !ok {
			s = &Samples{Pkg: pkg, Name: name}
			run.Samples[s.Key()] = s
		}
		b, hasBytes := metrics["B/op"]
		a, hasAllocs := metrics["allocs/op"]
		if !hasBytes || !hasAllocs {
			return nil, fmt.Errorf("%w: %s.%s (is -benchmem set?)", errNoMemoryMetrics, pkg, name)
		}
		// Only the gated metrics: b.ReportMetric can legitimately report a negative custom one
		for _, m := range []struct {
			unit  string
			value float64
		}{{"B/op", b}, {"allocs/op", a}} {
			if math.IsNaN(m.value) {
				return nil, fmt.Errorf("%w: %s.%s has a non-finite or negative %s", errBadMeasurement, pkg, name, m.unit)
			}
		}
		// b.ReportMetric(0, "ns/op") removes the timing, so a result can lack one
		if ns, ok := metrics["ns/op"]; ok {
			s.NS = append(s.NS, ns)
		}
		s.Bytes = append(s.Bytes, b)
		s.Allocs = append(s.Allocs, a)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("error reading benchmark output: %w", err)
	}
	recordUnreported(run, pkg, announced)
	return run, nil
}

// recordUnreported records as skipped each benchmark -v announced in a finished package that reported
// nothing for itself or a subbenchmark. Printed text can hide go test's skip record, and without one
// the budget would read as deleted.
func recordUnreported(run *Run, pkg string, announced []string) {
	if !run.Finished[pkg] {
		return
	}
	for _, name := range announced {
		key := pkg + "." + name
		if reportsUnder(run.Samples, key) || reportsUnder(run.Skipped, key) ||
			slices.ContainsFunc(announced, func(a string) bool { return strings.HasPrefix(a, name+"/") }) {
			continue
		}
		run.Skipped[key] = pkg
	}
}

// reportsUnder reports whether keys holds key or the key of one of its subbenchmarks
func reportsUnder[V any](keys map[string]V, key string) bool {
	for k := range keys {
		if k == key || strings.HasPrefix(k, key+"/") {
			return true
		}
	}
	return false
}

// benchmarkName reports whether name is one go test could announce
func benchmarkName(name string) bool {
	top, _, _ := strings.Cut(name, "/")
	return token.IsIdentifier(top) && benchmarkFuncName(top)
}

// SampleCountMismatches returns, in key order, the benchmarks without exactly expected samples of each
// gated metric
func SampleCountMismatches(samples map[string]*Samples, expected int) []string {
	var mismatched []string
	for _, key := range sortedKeys(samples) {
		s := samples[key]
		if len(s.Bytes) != expected || len(s.Allocs) != expected {
			mismatched = append(mismatched, key)
		}
	}
	return mismatched
}

func describeSamples(s *Samples, expected int) string {
	return fmt.Sprintf("B/op=%d allocs/op=%d, expected %d", len(s.Bytes), len(s.Allocs), expected)
}

// Summarise reduces each benchmark's samples to their medians
func Summarise(samples map[string]*Samples) (map[string]*Result, error) {
	results := make(map[string]*Result, len(samples))
	for _, key := range sortedKeys(samples) {
		s := samples[key]
		allocs, err := median(s.Allocs)
		if err != nil {
			return nil, fmt.Errorf("%s allocs/op: %w", key, err)
		}
		bytes, err := median(s.Bytes)
		if err != nil {
			return nil, fmt.Errorf("%s B/op: %w", key, err)
		}
		results[key] = &Result{Pkg: s.Pkg, Name: s.Name, Allocs: uint64(allocs), Bytes: uint64(bytes)}
	}
	return results, nil
}

// resultStart matches go test's first write of a result: the name, padded with spaces, and a tab. The
// measurements are a separate write, so printed text can come between them.
var resultStart = regexp.MustCompile(`^Benchmark\S* +\t`)

// embeddedResult reports whether printed text has run into a result, pushing it off the start of the
// line where it would be skipped. Only an occurrence in go test's layout counts, so a logged result
// written out with spaces does not fail a passing run.
func embeddedResult(line string, announced []string) bool {
	if strings.HasPrefix(line, "Benchmark") {
		return false
	}
	for i := 0; i < len(line); i++ {
		next := strings.Index(line[i:], "Benchmark")
		if next < 0 {
			return false
		}
		i += next
		if !resultStart.MatchString(line[i:]) {
			continue
		}
		if _, ok := parseBenchLine(line[i:], announced); ok {
			return true
		}
	}
	return false
}

// failureRecord reports whether line is go test reporting a failure. go test starts those records at
// the start of a line and indents logged text, so a logged "--- FAIL:" does not count.
func failureRecord(line string) bool {
	if line == "FAIL" || strings.HasPrefix(line, "FAIL\t") {
		return true
	}
	const record = "--- FAIL: Benchmark"
	return strings.HasPrefix(line, record) ||
		(strings.HasPrefix(line, "Benchmark") && strings.Contains(line, "\t"+record))
}

// loggedLine matches b.Log output: four spaces and the file and line, which a //line directive can
// name anything, then eight spaces on the message's later lines
var loggedLine = regexp.MustCompile(`^(?: {4}.+?:\d+: | {8})`)

// embeddedSkip returns the benchmark named by a skip record that printed text has pushed off the start
// of the line. The record must end a line that is not logged, and name a benchmark announced in the
// package or a subbenchmark of one.
func embeddedSkip(line string, announced []string) (string, bool) {
	if loggedLine.MatchString(line) {
		return "", false
	}
	i := strings.LastIndex(line, "--- SKIP: Benchmark")
	if i <= 0 {
		return "", false
	}
	name := line[i+len("--- SKIP: "):]
	if strings.ContainsAny(name, " \t") {
		return "", false
	}
	for _, a := range announced {
		if name == a || strings.HasPrefix(name, a+"/") {
			return name, true
		}
	}
	return "", false
}

// skipRecordName returns the benchmark named by skip record text ending line, wherever it starts
func skipRecordName(line string) (string, bool) {
	i := strings.LastIndex(line, "--- SKIP: Benchmark")
	if i < 0 {
		return "", false
	}
	name := line[i+len("--- SKIP: "):]
	return name, !strings.ContainsAny(name, " \t")
}

type benchLine struct {
	name string
	// procs is the GOMAXPROCS suffix without its dash, or "" for none
	procs      string
	iterations uint64
	metrics    map[string]float64
}

// parseBenchLine parses a result line, reporting false for any other line
func parseBenchLine(line string, announced []string) (benchLine, bool) {
	f := strings.Fields(line)
	if len(f) < 4 || !strings.HasPrefix(f[0], "Benchmark") {
		return benchLine{}, false
	}
	iterations, err := strconv.ParseUint(f[1], 10, 64)
	if err != nil {
		return benchLine{}, false
	}
	pairs := f[2:]
	if len(pairs)%2 != 0 {
		return benchLine{}, false
	}
	metrics := make(map[string]float64, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		v, err := strconv.ParseFloat(pairs[i], 64)
		if err != nil {
			return benchLine{}, false
		}
		// Kept as NaN for Parse to reject, rather than dropped, which would lose the sample silently
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			metrics[pairs[i+1]] = math.NaN()
			continue
		}
		metrics[pairs[i+1]] = v
	}
	name, procs := splitProcSuffix(f[0], announced)
	return benchLine{name: name, procs: procs, iterations: iterations, metrics: metrics}, true
}

// splitProcSuffix splits off the -N GOMAXPROCS suffix go test appends when N is above 1. The newest
// announcement that matches settles where the name ends, so a subbenchmark named "size-4" keeps its
// digits; failing that, a final dash and digits is the suffix.
func splitProcSuffix(name string, announced []string) (base, procs string) {
	for _, a := range slices.Backward(announced) {
		if name == a {
			return name, ""
		}
		if rest, ok := strings.CutPrefix(name, a+"-"); ok {
			if _, err := strconv.ParseUint(rest, 10, 64); err == nil {
				return a, rest
			}
		}
	}
	i := strings.LastIndexByte(name, '-')
	if i <= 0 {
		return name, ""
	}
	if _, err := strconv.ParseUint(name[i+1:], 10, 64); err != nil {
		return name, ""
	}
	return name[:i], name[i+1:]
}

func describeProcSuffix(procs string) string {
	if procs == "" {
		return "no suffix"
	}
	return "-" + procs
}

// median returns the middle sample, so one noisy sample cannot move the result. An even count has no
// middle sample and is refused.
func median(samples []float64) (float64, error) {
	if len(samples)%2 == 0 {
		return 0, fmt.Errorf("%w: got %d", errEvenSampleCount, len(samples))
	}
	s := slices.Clone(samples)
	slices.Sort(s)
	return s[len(s)/2], nil
}
