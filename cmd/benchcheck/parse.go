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
)

const modulePrefix = "github.com/thrasher-corp/gocryptotrader/"

// Result holds every sample recorded for a single benchmark across all -count runs
type Result struct {
	Pkg    string
	Name   string
	NS     []float64
	Bytes  []float64
	Allocs []float64
}

// Key returns the baseline key for a result, in the form "exchanges/orderbook.BenchmarkProcess"
func (r *Result) Key() string {
	return r.Pkg + "." + r.Name
}

// Parse reads `go test -bench` output and aggregates the samples for each benchmark. Package
// attribution comes from the "pkg:" header Go emits before each package's results, so the input
// must not be filtered through anything that strips those lines.
func Parse(r io.Reader) (map[string]*Result, error) {
	results := make(map[string]*Result)
	var pkg string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if p, ok := strings.CutPrefix(line, "pkg:"); ok {
			pkg = strings.TrimPrefix(strings.TrimSpace(p), modulePrefix)
			continue
		}
		name, metrics, ok := parseBenchLine(line)
		if !ok {
			continue
		}
		// Without a pkg: header the result cannot be attributed, and a key such as ".BenchmarkX"
		// silently matches no baseline entry and no gated package, so the gate would pass anything
		if pkg == "" {
			return nil, fmt.Errorf("%w: %s", errNoPackageContext, name)
		}
		res, ok := results[pkg+"."+name]
		if !ok {
			res = &Result{Pkg: pkg, Name: name}
			results[res.Key()] = res
		}
		// A line without -benchmem carries no B/op or allocs/op. Recording it would leave the
		// gated metrics empty, which compareMetric skips, so the gate would silently pass.
		b, hasBytes := metrics["B/op"]
		a, hasAllocs := metrics["allocs/op"]
		if !hasBytes || !hasAllocs {
			return nil, fmt.Errorf("%w: %s.%s (is -benchmem set?)", errNoMemoryMetrics, pkg, name)
		}
		for unit, v := range metrics {
			if math.IsNaN(v) {
				return nil, fmt.Errorf("%w: %s.%s has a non-finite or negative %s", errBadMeasurement, pkg, name, unit)
			}
		}
		if v, ok := metrics["ns/op"]; ok {
			res.NS = append(res.NS, v)
		}
		res.Bytes = append(res.Bytes, b)
		res.Allocs = append(res.Allocs, a)
	}
	if err := sc.Err(); err != nil {
		return nil, err
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
	// The iteration count separates a result line from output such as "--- FAIL: BenchmarkX"
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

// median returns the middle sample, taking the lower of the two middles for an even sample count so
// that alloc and byte counts stay whole numbers. Samples are a median rather than a mean because a
// single noisy iteration on a shared CI runner should not move the compared value.
func median(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	s := slices.Clone(samples)
	slices.Sort(s)
	return s[(len(s)-1)/2]
}
