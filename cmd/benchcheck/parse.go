package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
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
)

const modulePrefix = "github.com/thrasher-corp/gocryptotrader/"

// Run is what one `go test -bench` invocation reported
type Run struct {
	// Platform is the goos/goarch pair from the output's headers, such as "linux/amd64", or empty
	// when the output carries none
	Platform string
	Samples  map[string]*Samples
	// Finished holds the packages whose output ran to go test's closing "ok" line
	Finished map[string]bool
}

// Samples holds every measurement recorded for a single benchmark across all -count runs
type Samples struct {
	Pkg    string
	Name   string
	Bytes  []float64
	Allocs []float64
}

// Key returns the baseline key for the benchmark, in the form "exchanges/orderbook.BenchmarkProcess"
func (s *Samples) Key() string {
	return s.Pkg + "." + s.Name
}

// Result is a benchmark's median measurements, the values compared against its budget
type Result struct {
	Pkg    string
	Name   string
	Allocs uint64
	Bytes  uint64
}

// Key returns the baseline key for a result, in the form "exchanges/orderbook.BenchmarkProcess"
func (r *Result) Key() string {
	return r.Pkg + "." + r.Name
}

// Parse reads `go test -bench` output and gathers the samples for each benchmark. Package
// attribution comes from the "pkg:" header Go emits before each package's results, so the input
// must not be filtered through anything that strips those lines.
func Parse(r io.Reader) (*Run, error) {
	run := &Run{Samples: make(map[string]*Samples), Finished: make(map[string]bool)}
	var pkg, goos, goarch string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		// A failed benchmark still prints a result line for each sample it completed, so the output
		// of a broken run reads as a clean one with nothing to say it stopped short. One that fails
		// partway through a run has its failure printed after its name, on the same line.
		if line == "FAIL" || strings.HasPrefix(line, "FAIL\t") || strings.Contains(line, "--- FAIL:") {
			return nil, fmt.Errorf("%w: %q", errRunFailed, line)
		}
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "ok" {
			run.Finished[strings.TrimPrefix(f[1], modulePrefix)] = true
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
			pkg = strings.TrimPrefix(strings.TrimSpace(p), modulePrefix)
			if goos != "" || goarch != "" {
				platform := goos + "/" + goarch
				if run.Platform != "" && run.Platform != platform {
					return nil, fmt.Errorf("%w: %s and %s", errMixedPlatforms, run.Platform, platform)
				}
				run.Platform = platform
			}
			continue
		}
		// -v announces each benchmark with its bare name on a line of its own
		if !strings.HasPrefix(line, "Benchmark") || len(strings.Fields(line)) == 1 {
			continue
		}
		// Otherwise go test prints a line starting with the benchmark's name for every sample, and
		// nothing else starts that way, so one that fails to parse is a result something has written
		// into: the benchmark's own output, or a line cut short. Skipping it would drop the sample
		// silently, and a benchmark with every sample hit would vanish from the run along with any
		// finding.
		name, metrics, ok := parseBenchLine(line)
		if !ok {
			return nil, fmt.Errorf("%w: %q", errMalformedResult, line)
		}
		// A key such as ".BenchmarkX" matches no baseline entry and no listed package, so an
		// unattributed result would pass the gate rather than fail it
		if pkg == "" {
			return nil, fmt.Errorf("%w: %s", errNoPackageContext, name)
		}
		s, ok := run.Samples[pkg+"."+name]
		if !ok {
			s = &Samples{Pkg: pkg, Name: name}
			run.Samples[s.Key()] = s
		}
		// A line without -benchmem carries no B/op or allocs/op. Recording it would leave the
		// gated metrics empty, so the benchmark could never fail the gate.
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
		s.Bytes = append(s.Bytes, b)
		s.Allocs = append(s.Allocs, a)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return run, nil
}

// SampleCountMismatches reports, in key order, the benchmarks that did not record exactly expected
// samples of each gated metric. go test runs every benchmark -count times, so any other number is a
// truncated run, a result line that failed to parse, or -cpu naming several values, whose results
// the suffix stripping files under one key.
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

// describeSamples renders a mismatched benchmark's sample counts for an error line
func describeSamples(s *Samples, expected int) string {
	return fmt.Sprintf("B/op=%d allocs/op=%d, expected %d", len(s.Bytes), len(s.Allocs), expected)
}

// Summarise reduces each benchmark's samples to their medians, which are the values compared
// against its budget
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

// parseBenchLine extracts the benchmark name and its value/unit pairs from a single result line,
// reporting false for any line that is not a well-formed benchmark result
func parseBenchLine(line string) (name string, metrics map[string]float64, ok bool) {
	f := strings.Fields(line)
	if len(f) < 4 || !strings.HasPrefix(f[0], "Benchmark") {
		return "", nil, false
	}
	// The iteration count separates a result line from output such as "--- BENCH: BenchmarkX"
	if _, err := strconv.ParseUint(f[1], 10, 64); err != nil {
		return "", nil, false
	}
	pairs := f[2:]
	if len(pairs)%2 != 0 {
		return "", nil, false
	}
	metrics = make(map[string]float64, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		v, err := strconv.ParseFloat(pairs[i], 64)
		if err != nil {
			return "", nil, false
		}
		// NaN and infinities parse cleanly but convert to nonsense budgets via uint64, and a
		// negative measurement is not something the testing package emits. Recorded rather than
		// skipped: silently dropping the line removes its package from the results entirely, and
		// with any other package still parsing, the gate would report success.
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			metrics[pairs[i+1]] = math.NaN()
			continue
		}
		metrics[pairs[i+1]] = v
	}
	return trimProcSuffix(f[0]), metrics, true
}

// trimProcSuffix removes the -N GOMAXPROCS suffix that the testing package appends to result names
func trimProcSuffix(name string) string {
	i := strings.LastIndexByte(name, '-')
	if i <= 0 {
		return name
	}
	if _, err := strconv.ParseUint(name[i+1:], 10, 64); err != nil {
		return name
	}
	return name[:i]
}

// median returns the middle sample, rather than a mean, so one noisy iteration on a shared CI
// runner cannot move the compared value. It refuses an even count, which has no middle sample:
// averaging the two middles compares a value the run never measured, and picking either one biases
// the comparison towards that side.
func median(samples []float64) (float64, error) {
	if len(samples)%2 == 0 {
		return 0, fmt.Errorf("%w: got %d", errEvenSampleCount, len(samples))
	}
	s := slices.Clone(samples)
	slices.Sort(s)
	return s[len(s)/2], nil
}
