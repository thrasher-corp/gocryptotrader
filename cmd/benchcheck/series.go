package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// SeriesRecord is one benchmark's measurements from one run, appended to the series file as a
// single JSON line so history can be charted without re-running anything.
//
// Median is the value the gate compares; Min is carried alongside it because the fastest sample is
// the least contaminated by scheduler noise, which makes it the better estimator when reading the
// ns/op trend across runs on different machines.
type SeriesRecord struct {
	Timestamp   time.Time `json:"ts"`
	Commit      string    `json:"commit,omitempty"`
	Package     string    `json:"pkg"`
	Benchmark   string    `json:"bench"`
	Samples     int       `json:"samples"`
	NSMedian    float64   `json:"ns_median"`
	NSMin       float64   `json:"ns_min"`
	BytesMedian uint64    `json:"bytes_median"`
	AllocsMedn  uint64    `json:"allocs_median"`
}

// AppendSeries appends one record per benchmark to path, creating it if absent. Records are only
// ever appended, never rewritten, so the file is an append-only history and merges cleanly.
func AppendSeries(path, commit string, results map[string]*Result, now time.Time) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("error opening series file: %w", err)
	}
	defer f.Close()

	// An existing file whose last line is unterminated would otherwise have the next record
	// concatenated onto it, producing a line that is not valid JSON. The append-only prefix check
	// in scripts/bench_history.sh cannot catch that: the damaged line is still a byte prefix.
	if err := terminateLastLine(f, path); err != nil {
		return err
	}

	for _, key := range sortedKeys(results) {
		r := results[key]
		rec := &SeriesRecord{
			Timestamp:   now.UTC(),
			Commit:      commit,
			Package:     r.Pkg,
			Benchmark:   r.Name,
			Samples:     len(r.NS),
			NSMedian:    median(r.NS),
			NSMin:       minimum(r.NS),
			BytesMedian: uint64(median(r.Bytes)),
			AllocsMedn:  uint64(median(r.Allocs)),
		}
		line, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("error encoding series record for %s: %w", key, err)
		}
		if _, err := f.Write(append(line, '\n')); err != nil {
			return fmt.Errorf("error writing series record for %s: %w", key, err)
		}
	}
	return nil
}

// terminateLastLine writes a newline if path is non-empty and does not already end with one
func terminateLastLine(f *os.File, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("error inspecting series file: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}
	r, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error reading series file: %w", err)
	}
	defer r.Close()

	last := make([]byte, 1)
	if _, err := r.ReadAt(last, info.Size()-1); err != nil {
		return fmt.Errorf("error reading end of series file: %w", err)
	}
	if last[0] == '\n' {
		return nil
	}
	if _, err := f.Write([]byte{'\n'}); err != nil {
		return fmt.Errorf("error terminating series file: %w", err)
	}
	return nil
}

func minimum(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	m := samples[0]
	for _, v := range samples[1:] {
		if v < m {
			m = v
		}
	}
	return m
}
