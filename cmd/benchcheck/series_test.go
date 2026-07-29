package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestAppendSeries(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "series.jsonl")
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)

	r := &Result{Pkg: "currency", Name: "BenchmarkNewCode", NS: []float64{110, 98, 104}, Bytes: []float64{8, 8, 8}, Allocs: []float64{1, 1, 1}}
	require.NoError(t, AppendSeries(path, "abc123", resultsOf(r), now), "AppendSeries must not error")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading the series file must not error")
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 1, "one record per benchmark must be written")

	var rec SeriesRecord
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &rec), "the record must be valid JSON")
	assert.Equal(t, "currency", rec.Package, "the package should be recorded")
	assert.Equal(t, "BenchmarkNewCode", rec.Benchmark, "the benchmark name should be recorded")
	assert.Equal(t, "abc123", rec.Commit, "the commit should be recorded")
	assert.Equal(t, now, rec.Timestamp, "the timestamp should be recorded in UTC")
	assert.Equal(t, 3, rec.Samples, "the sample count should be recorded")
	assert.Equal(t, float64(104), rec.NSMedian, "the median ns should be recorded")
	assert.Equal(t, float64(98), rec.NSMin, "the fastest sample should be recorded alongside the median")
	assert.Equal(t, uint64(8), rec.BytesMedian, "the median bytes should be recorded")
	assert.Equal(t, uint64(1), rec.AllocsMedn, "the median allocs should be recorded")
}

func TestAppendSeriesAppends(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "series.jsonl")
	got := resultsOf(result("currency", "BenchmarkNewCode", 1, 8), result("types", "BenchmarkNumberMarshalJSON", 1, 16))

	for range 3 {
		require.NoError(t, AppendSeries(path, "abc123", got, time.Now()), "AppendSeries must not error")
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading the series file must not error")
	assert.Len(t, strings.Split(strings.TrimSpace(string(data)), "\n"), 6,
		"repeated runs must append rather than overwrite, giving history")
}

func TestAppendSeriesTerminatesUnterminatedFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "series.jsonl")
	// A history whose last line has no newline: appending straight onto it produces a line that is
	// not valid JSON, and the byte-prefix guard in bench_history.sh cannot see it because the
	// damaged line is still a prefix of the new content.
	require.NoError(t, os.WriteFile(path, []byte(`{"ts":"2026-07-29T01:00:00Z","bench":"A"}`), 0o600),
		"writing the fixture must not error")

	require.NoError(t, AppendSeries(path, "abc", resultsOf(result("currency", "BenchmarkNewCode", 1, 8)), time.Now()),
		"AppendSeries must not error")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading the series file must not error")
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	require.Len(t, lines, 2, "the record must land on its own line")
	for i, ln := range lines {
		var v map[string]any
		assert.NoErrorf(t, json.Unmarshal([]byte(ln), &v), "line %d should be valid JSON", i+1)
	}
}

func TestMinimum(t *testing.T) {
	t.Parallel()
	assert.Zero(t, minimum(nil), "the minimum of no samples should be zero")
	assert.Equal(t, 2.0, minimum([]float64{5, 2, 9}), "the minimum should be the smallest sample")
}

func TestRunWritesSeries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		seriesPath:     filepath.Join(dir, "series.jsonl"),
		commit:         "deadbeef",
		bytesTolerance: 0.01,
		warn:           true,
	}
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	require.NoError(t, run(strings.NewReader(sampleOutput), &strings.Builder{}, o), "run must not error in warn mode")

	data, err := os.ReadFile(o.seriesPath)
	require.NoError(t, err, "the series file must have been written")
	assert.Len(t, strings.Split(strings.TrimSpace(string(data)), "\n"), 3, "one record per benchmark should be appended")
	assert.Contains(t, string(data), "deadbeef", "records should carry the supplied commit")
}
