package main

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// steadySamples returns n samples of each metric, the timings spread a little around ns
func steadySamples(n int, ns, bytesPerOp, allocsPerOp float64) *Samples {
	s := &Samples{}
	for i := range n {
		s.NS = append(s.NS, ns*(1+float64(i%3-1)/1000))
		s.Bytes = append(s.Bytes, bytesPerOp)
		s.Allocs = append(s.Allocs, allocsPerOp)
	}
	return s
}

// withTimings returns samples of the given timings, each allocating 64 bytes once
func withTimings(ns ...float64) *Samples {
	s := steadySamples(len(ns), 0, 64, 1)
	s.NS = ns
	return s
}

// testComparisonBaseline gives p.BenchmarkTolerated an allocation tolerance its one extra allocation is within
var testComparisonBaseline = Baseline{"p.BenchmarkTolerated": {Allocs: 100, Bytes: 6400, AllocsTolerance: 0.01, Reason: "measured"}}

// testComparison returns a comparison of seven samples covering each outcome a benchmark can have
func testComparison() *comparison {
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), samples: 7}
	for key, pair := range map[string][2]*Samples{
		"p.BenchmarkSame":        {steadySamples(7, 100, 64, 1), steadySamples(7, 100, 64, 1)},
		"p.BenchmarkFaster":      {steadySamples(7, 100, 64, 1), steadySamples(7, 50, 64, 1)},
		"p.BenchmarkSmallShift":  {steadySamples(7, 100, 64, 1), steadySamples(7, 102, 64, 1)},
		"p.BenchmarkAllocs":      {steadySamples(7, 100, 64, 1), steadySamples(7, 100, 64, 2)},
		"p.BenchmarkBytesWithin": {steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 1005, 1)},
		"p.BenchmarkBytesBeyond": {steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 1020, 1)},
		"p.BenchmarkTolerated":   {steadySamples(7, 100, 6400, 100), steadySamples(7, 100, 6400, 101)},
		"p.BenchmarkIncomplete":  {steadySamples(7, 100, 64, 1), steadySamples(6, 100, 64, 1)},
		"p.BenchmarkNoTiming":    {{Bytes: make([]float64, 7), Allocs: make([]float64, 7)}, steadySamples(7, 100, 0, 0)},
		// Medians 8% apart, with samples too spread to make that significant
		"p.BenchmarkNoisy": {
			withTimings(90, 100, 110, 120, 130, 140, 150), withTimings(95, 105, 115, 130, 135, 145, 155),
		},
	} {
		c.base.samples[key], c.head.samples[key] = pair[0], pair[1]
	}
	c.base.samples["p.BenchmarkOnlyBase"] = steadySamples(7, 100, 64, 1)
	c.head.samples["p.BenchmarkOnlyHead"] = steadySamples(7, 100, 64, 1)
	c.head.samples["p.BenchmarkSkippedAtHead"] = steadySamples(0, 0, 0, 0)
	c.base.samples["p.BenchmarkSkippedAtHead"] = steadySamples(7, 100, 64, 1)
	c.head.skipped["p.BenchmarkSkippedAtHead"] = true
	c.base.skipped["p.BenchmarkSkippedAtBase"] = true
	c.base.skipped["p.BenchmarkSkippedAtBoth"] = true
	c.head.skipped["p.BenchmarkSkippedAtBoth"] = true
	return c
}

func TestComparisonCompare(t *testing.T) {
	t.Parallel()
	c := testComparison()
	require.NoError(t, c.compare(testComparisonBaseline), "compare must not error")

	rows := make(map[string]comparisonRow, len(c.rows))
	for _, row := range c.rows {
		rows[row.key] = row
	}
	for _, tc := range []struct {
		key                        string
		significant, allocs, moved bool
	}{
		{key: "p.BenchmarkSame"},
		{key: "p.BenchmarkFaster", significant: true, moved: true},
		{key: "p.BenchmarkSmallShift", significant: true},
		{key: "p.BenchmarkAllocs", allocs: true, moved: true},
		{key: "p.BenchmarkBytesWithin"},
		{key: "p.BenchmarkBytesBeyond", allocs: true, moved: true},
		{key: "p.BenchmarkTolerated"},
		{key: "p.BenchmarkNoisy"},
	} {
		row, ok := rows[tc.key]
		require.Truef(t, ok, "%s must be compared", tc.key)
		assert.Equalf(t, tc.significant, row.significant, "%s should have the expected timing significance", tc.key)
		assert.Equalf(t, tc.allocs, row.allocationsMoved, "%s should have the expected allocation movement", tc.key)
		assert.Equalf(t, tc.moved, row.moved(), "%s should be led with only when it moved", tc.key)
	}
	assert.InDelta(t, 100.0, rows["p.BenchmarkFaster"].baseNS, 1e-9, "the base timing should be the median")
	assert.InDelta(t, 50.0, rows["p.BenchmarkFaster"].headNS, 1e-9, "the head timing should be the median")
	assert.Equal(t, uint64(2), rows["p.BenchmarkAllocs"].headAllocs, "the head's allocations should be the median")
	assert.Equal(t, uint64(1020), rows["p.BenchmarkBytesBeyond"].headBytes, "the head's bytes should be the median")
	assert.InDelta(t, 120.0, rows["p.BenchmarkNoisy"].baseNS, 1e-9, "a spread timing's median should still be reported")
	assert.Len(t, c.rows, 8, "only benchmarks complete at both revisions should be compared")

	assert.Equal(t, map[string]string{
		"p.BenchmarkIncomplete":    "expected 7 samples of each metric, got 7 ns/op, 7 B/op and 7 allocs/op at the base and 6 ns/op, 6 B/op and 6 allocs/op at the head",
		"p.BenchmarkNoTiming":      "expected 7 samples of each metric, got 0 ns/op, 7 B/op and 7 allocs/op at the base and 7 ns/op, 7 B/op and 7 allocs/op at the head",
		"p.BenchmarkOnlyBase":      "only at the base",
		"p.BenchmarkOnlyHead":      "only at the head",
		"p.BenchmarkSkippedAtBase": "skipped at the base",
		"p.BenchmarkSkippedAtBoth": "skipped at both",
		"p.BenchmarkSkippedAtHead": "skipped at the head",
	}, c.notCompared, "each benchmark not compared should say why")
	assert.Empty(t, c.warnings, "seven samples should be enough for the test to find a difference")
}

// guardHeap ends the test binary once the heap has grown by 256 MiB, until stop returns, since
// benchmath's U test allocates without bound once given a NaN and only exiting stops it
func guardHeap(t *testing.T) (stop func()) {
	t.Helper()
	var start runtime.MemStats
	runtime.ReadMemStats(&start)
	name := t.Name()
	done, exited := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(exited)
		tick := time.NewTicker(20 * time.Millisecond)
		defer tick.Stop()
		var ms runtime.MemStats
		for {
			select {
			case <-done:
				return
			case <-tick.C:
			}
			runtime.ReadMemStats(&ms)
			if ms.HeapAlloc > start.HeapAlloc+256<<20 {
				// Exits even if nothing reads stderr and the message blocks
				time.AfterFunc(time.Second, func() { os.Exit(1) })
				fmt.Fprintf(os.Stderr, "%s: the heap grew by more than 256 MiB, as it does once a NaN timing reaches benchmath's U test\n", name)
				os.Exit(1)
			}
		}
	}()
	// Waits for a poll already under way, which could otherwise end the test binary after its test passed
	return func() {
		close(done)
		<-exited
	}
}

func TestComparisonCompareRejectsBadTimings(t *testing.T) {
	t.Parallel()
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), samples: 3}
	c.base.samples["p.BenchmarkX"] = steadySamples(3, 100, 0, 0)
	c.head.samples["p.BenchmarkX"] = steadySamples(3, 100, 0, 0)
	c.head.samples["p.BenchmarkX"].NS[1] = math.NaN()
	// Stopped only once compare returns, which it never does once the NaN reaches the U test
	stop := guardHeap(t)
	done := make(chan error, 1)
	go func() { done <- c.compare(Baseline{}) }()
	select {
	case err := <-done:
		stop()
		assert.ErrorIs(t, err, errBadTiming, "a timing Parse could not read should fail the comparison")
	case <-time.After(time.Minute):
		require.FailNow(t, "compare must refuse a timing Parse could not read rather than hang")
	}
}

func TestComparisonCompareWarnsOnTooFewSamples(t *testing.T) {
	t.Parallel()
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), samples: 3}
	c.base.samples["p.BenchmarkX"] = steadySamples(3, 100, 0, 0)
	c.head.samples["p.BenchmarkX"] = steadySamples(3, 200, 0, 0)
	require.NoError(t, c.compare(Baseline{}), "compare must not error")
	require.Len(t, c.rows, 1, "the benchmark must be compared")
	assert.False(t, c.rows[0].significant, "three samples a side cannot reach significance")
	assert.NotEmpty(t, c.warnings, "the comparison should warn that it had too few samples to find a difference")

	var report bytes.Buffer
	c.render(&report)
	assert.Contains(t, report.String(), "### Notes\n\n- "+c.warnings[0], "the report should carry the warning")
}

func TestComparisonRender(t *testing.T) {
	t.Parallel()
	c := testComparison()
	c.base.commit, c.head.commit = "0123456789abcdef0123", "fedcba9876543210fedc"
	c.base.cpu, c.head.cpu = "Example CPU", "Example CPU"
	c.dirty = true
	c.packages = []string{"p"}
	c.onlyHead, c.onlyBase = []string{"newpkg"}, []string{"oldpkg"}
	require.NoError(t, c.compare(testComparisonBaseline), "compare must not error")
	var report bytes.Buffer
	c.render(&report)
	out := report.String()

	assert.Contains(t, out, "Base `0123456789ab` against head `fedcba987654` with uncommitted changes: 8 benchmarks in 1 package, measured in 7 rounds",
		"the report should say what it compared, and in how many rounds")
	assert.Contains(t, out, ", on Example CPU.", "the report should name the processor both revisions ran on")
	assert.Contains(t, out, "### Moved (3)", "the report should lead with the benchmarks that moved")
	moved, all, ok := strings.Cut(out, "<details>")
	require.True(t, ok, "the report must hold every row in a details block")
	faster := "| `p.BenchmarkFaster` | 100 → 50.0 (-50.0%, p=0.001) | 64 | 1 |"
	allocs := "| `p.BenchmarkAllocs` | 100 → 100 (~, p=1.000) | 64 | 1 → 2 (+100.0%) |"
	bytesBeyond := "| `p.BenchmarkBytesBeyond` | 100 → 100 (~, p=1.000) | 1000 → 1020 (+2.0%) | 1 |"
	smallShift := "| `p.BenchmarkSmallShift` | 100 → 102 (+2.0%, p=0.001) | 64 | 1 |"
	for _, row := range []string{faster, allocs, bytesBeyond} {
		assert.Containsf(t, moved, row, "a benchmark that moved should lead: %s", row)
	}
	assert.NotContains(t, moved, smallShift, "a significant change below the highlight threshold should not lead")
	assert.Contains(t, all, smallShift, "every compared benchmark should be in the full table")
	assert.Contains(t, all, "<summary>All 8 benchmarks</summary>", "the full table should count its rows")
	assert.Contains(t, all, "</summary>\n\n| Benchmark |", "a blank line should end the summary's HTML, which would otherwise swallow the table")
	header := "| Benchmark | ns/op | B/op | allocs/op |\n| --- | --- | --- | --- |\n"
	assert.Contains(t, moved, header, "the benchmarks that moved should form a table")
	assert.Contains(t, all, header, "every compared benchmark should be in a table")
	_, afterDetails, ok := strings.Cut(all, "</details>")
	require.True(t, ok, "the details block must be closed")
	assert.Contains(t, afterDetails, "### Not compared (9)", "what was not compared should follow the details block rather than hide in it")
	assert.Contains(t, all, "</details>\n\n### Not compared (9)", "a blank line should end the details block's HTML, which would otherwise swallow the heading after it")

	for _, line := range []string{
		"### Not compared (9)",
		"- `newpkg`: package only listed at the head",
		"- `oldpkg`: package only listed at the base",
		"- `p.BenchmarkSkippedAtHead`: skipped at the head",
	} {
		assert.Containsf(t, out, line, "the report should list what it did not compare: %s", line)
	}
	assert.NotContains(t, out, "### Notes", "a comparison without warnings should have no notes")
}

func TestComparisonRowMoved(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		row  comparisonRow
		exp  bool
	}{
		{"a significant 5% slowdown", comparisonRow{baseNS: 100, headNS: 105, significant: true}, true},
		{"a significant 5% speed-up", comparisonRow{baseNS: 100, headNS: 95, significant: true}, true},
		{"a significant 5.1% slowdown", comparisonRow{baseNS: 100, headNS: 105.1, significant: true}, true},
		{"a significant 5.1% speed-up", comparisonRow{baseNS: 100, headNS: 94.9, significant: true}, true},
		{"a significant 4.9% change", comparisonRow{baseNS: 100, headNS: 104.9, significant: true}, false},
		{"an insignificant 10% change", comparisonRow{baseNS: 100, headNS: 110}, false},
		{"allocations that moved", comparisonRow{baseNS: 100, headNS: 100, allocationsMoved: true}, true},
	} {
		assert.Equalf(t, tc.exp, tc.row.moved(), "moved should be %t for %s", tc.exp, tc.name)
	}
}

func TestRemoveEarlierOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"summary.md": "report\n", "base.txt": "base\n", "head.txt": "head\n", "notes.txt": "kept\n"})
	require.NoError(t, removeEarlierOutput(dir), "removeEarlierOutput must not error")
	assert.Equal(t, []string{"notes.txt"}, dirNames(t, dir), "only the files a comparison writes should be removed")
	assert.NoError(t, removeEarlierOutput(dir), "removeEarlierOutput should not error when there is nothing to remove")

	names := []string{"summary.md", "base.txt", "head.txt"}
	for i, name := range names {
		dir := t.TempDir()
		// A directory that is not empty cannot be removed in a file's place
		files := map[string]string{name + "/inside.txt": "in the way\n"}
		for _, other := range names {
			if other != name {
				files[other] = "earlier\n"
			}
		}
		writeFiles(t, dir, files)
		assert.ErrorContainsf(t, removeEarlierOutput(dir), "error removing the earlier "+name, "a %s that cannot be removed should fail, naming it", name)
		for _, before := range names[:i] {
			assert.NoFileExistsf(t, filepath.Join(dir, before), "%s should be removed before %s, the report first", before, name)
		}
		for _, after := range names[i+1:] {
			assert.FileExistsf(t, filepath.Join(dir, after), "%s should be left once removing %s fails", after, name)
		}
	}
}

// outputComparison returns a comparison whose revisions printed one line each
func outputComparison(predatesList bool) *comparison {
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), samples: 7, predatesList: predatesList}
	c.base.commit, c.head.commit = "0123456789abcdef", "fedcba9876543210"
	c.base.output.WriteString("base output\n")
	c.head.output.WriteString("head output\n")
	return c
}

// assertFiles asserts that dir holds exactly the files given, with their contents
func assertFiles(t *testing.T, dir string, exp map[string]string) {
	t.Helper()
	got := make(map[string]string, len(exp))
	for _, name := range dirNames(t, dir) {
		body, err := os.ReadFile(filepath.Join(dir, name))
		require.NoErrorf(t, err, "reading %s must not error", name)
		got[name] = string(body)
	}
	assert.Equal(t, exp, got, "the directory should hold the files expected")
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestComparisonWrite(t *testing.T) {
	t.Parallel()
	earlier := map[string]string{"summary.md": "earlier report\n", "base.txt": "earlier base\n", "head.txt": "earlier head\n", "notes.txt": "kept\n"}

	dir := t.TempDir()
	writeFiles(t, dir, earlier)
	var out bytes.Buffer
	require.NoError(t, outputComparison(false).write(&out, dir), "write must not error")
	assert.Contains(t, out.String(), "## Benchmark comparison", "write should print the report")
	assertFiles(t, dir, map[string]string{"summary.md": out.String(), "base.txt": "base output\n", "head.txt": "head output\n", "notes.txt": "kept\n"})

	dir = t.TempDir()
	writeFiles(t, dir, earlier)
	out.Reset()
	require.NoError(t, outputComparison(true).write(&out, dir), "write must not error for a base without a package list")
	assert.Contains(t, out.String(), "predates `benchmarks/packages.txt`", "write should print the report")
	assertFiles(t, dir, map[string]string{"summary.md": out.String(), "notes.txt": "kept\n"})

	missing := filepath.Join(t.TempDir(), "missing")
	assert.ErrorContains(t, outputComparison(false).write(&bytes.Buffer{}, missing), "error saving the base output", "write should fail when the raw output cannot be saved")
	assert.ErrorContains(t, outputComparison(true).write(&bytes.Buffer{}, missing), "error saving the summary", "write should fail when the report cannot be saved")
	for _, predatesList := range []bool{false, true} {
		dir = t.TempDir()
		writeFiles(t, dir, map[string]string{"base.txt/inside.txt": "in the way\n"})
		assert.ErrorContainsf(t, outputComparison(predatesList).write(&bytes.Buffer{}, dir), "error removing the earlier base.txt",
			"write should fail when an earlier file cannot be removed, with predatesList %t", predatesList)
	}
	errPrint := errors.New("printing failed")
	assert.ErrorIs(t, outputComparison(false).write(failingWriter{errPrint}, t.TempDir()), errPrint, "write should return the error printing the report")
}

func TestComparisonWriteOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"summary.md": "earlier report\n", "head.txt": "earlier head\n", "notes.txt": "kept\n"})
	require.NoError(t, outputComparison(false).writeOutput(dir), "writeOutput must not error")
	assertFiles(t, dir, map[string]string{"base.txt": "base output\n", "head.txt": "head output\n", "notes.txt": "kept\n"})

	assert.ErrorContains(t, outputComparison(false).writeOutput(filepath.Join(t.TempDir(), "missing")), "error saving the base output",
		"writeOutput should fail when the output cannot be saved")
	dir = t.TempDir()
	writeFiles(t, dir, map[string]string{"summary.md/inside.txt": "in the way\n"})
	assert.ErrorContains(t, outputComparison(false).writeOutput(dir), "error removing the earlier summary.md", "writeOutput should fail when the earlier report cannot be removed")
}

func TestComparisonRenderNothingMoved(t *testing.T) {
	t.Parallel()
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), samples: 7}
	c.base.samples["p.BenchmarkSame"] = steadySamples(7, 100, 64, 1)
	c.head.samples["p.BenchmarkSame"] = steadySamples(7, 100, 64, 1)
	c.base.cpu, c.head.cpu = "One CPU", "Another CPU"
	require.NoError(t, c.compare(Baseline{}), "compare must not error")
	var report bytes.Buffer
	c.render(&report)
	assert.Contains(t, report.String(), "### Moved (0)\n\nNo allocations moved", "the report should say when nothing moved")
	assert.Contains(t, report.String(), "<summary>All 1 benchmark</summary>", "a single compared benchmark should still be in the full table")
	assert.NotContains(t, report.String(), " CPU", "the report should name no processor when the revisions' outputs disagree")
	assert.NotContains(t, report.String(), "### Not compared", "a comparison that compared everything should list nothing as not compared")
}

func TestComparisonRenderPredatesList(t *testing.T) {
	t.Parallel()
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), predatesList: true}
	c.base.commit, c.head.commit = "0123456789abcdef", "fedcba9876543210"
	var report bytes.Buffer
	c.render(&report)
	assert.Equal(t, "## Benchmark comparison\n\nThe base, `0123456789ab`, predates `benchmarks/packages.txt`, "+
		"so there are no benchmarks listed to compare `fedcba987654` against.\n", report.String(),
		"a base without a package list should be reported as having nothing to compare")
}

func TestFormatting(t *testing.T) {
	t.Parallel()
	for ns, exp := range map[float64]string{0.7454: "0.75", 9.996: "10.00", 10: "10.0", 27.33: "27.3", 99.96: "100.0", 100: "100", 1234.5: "1234"} {
		assert.Equalf(t, exp, formatNS(ns), "formatNS should round %v to three or four significant figures", ns)
	}
	for _, tc := range []struct {
		base, head uint64
		exp        string
	}{
		{16, 16, "16"},
		{0, 8, "0 → 8"},
		{8, 0, "8 → 0 (-100.0%)"},
		{100, 150, "100 → 150 (+50.0%)"},
	} {
		assert.Equalf(t, tc.exp, countCell(tc.base, tc.head), "countCell should describe %d to %d", tc.base, tc.head)
	}
	assert.Equal(t, "`a/b.BenchmarkX/x\\|y`", codeSpan("a/b.BenchmarkX/x|y"), "a pipe should not end the table cell")
	assert.Equal(t, "``a/b.BenchmarkX/x`y``", codeSpan("a/b.BenchmarkX/x`y"), "a backtick should not end the code span")
	assert.Equal(t, "``` a/b.BenchmarkX/``y` ```", codeSpan("a/b.BenchmarkX/``y`"), "a key ending with a backtick should be set apart from the run around it")
	assert.Equal(t, "`` `x ``", codeSpan("`x"), "a key starting with a backtick should be set apart from the run around it")
	assert.Equal(t, "````a/b.BenchmarkX/x```y````", codeSpan("a/b.BenchmarkX/x```y"), "a run of three backticks should not end the code span")
	assert.Equal(t, "0123456789ab", shortCommit("0123456789abcdef"), "a commit should be shortened to twelve characters")
	assert.Equal(t, "abc", shortCommit("abc"), "a short name should be left as it is")
	assert.True(t, math.IsInf(relativeChange(0, 1), 1), "a change from zero should be infinite")
	assert.Zero(t, relativeChange(0, 0), "no change from zero should be zero")
}

func TestCompareSamples(t *testing.T) {
	t.Parallel()
	row, warnings, err := compareSamples("p.BenchmarkX", steadySamples(7, 100, 64, 1), steadySamples(7, 50, 80, 2), nil)
	require.NoError(t, err, "compareSamples must not error")
	assert.Empty(t, warnings, "seven samples a side should be enough for the test to find a difference")
	assert.Less(t, row.p, 0.05, "timings that never overlap should differ significantly")
	assert.Equal(t, comparisonRow{
		key: "p.BenchmarkX", baseNS: 100, headNS: 50, p: row.p, significant: true,
		baseBytes: 64, headBytes: 80, baseAllocs: 1, headAllocs: 2, allocationsMoved: true,
	}, row, "the row should hold both revisions' medians and the test's result")
	noisyBase, noisyHead := steadySamples(7, 100, 64, 1), steadySamples(7, 50, 80, 2)
	noisyBase.Bytes, noisyBase.Allocs = []float64{1000, 62, 64, 0, 66, 63, 2000}, []float64{9, 1, 2, 0, 4, 3, 10}
	noisyHead.Bytes, noisyHead.Allocs = []float64{2000, 78, 80, 0, 82, 79, 3000}, []float64{10, 4, 5, 0, 7, 6, 20}
	row, _, err = compareSamples("p.BenchmarkX", noisyBase, noisyHead, nil)
	require.NoError(t, err, "compareSamples must not error")
	assert.Equal(t, []uint64{64, 80, 3, 6}, []uint64{row.baseBytes, row.headBytes, row.baseAllocs, row.headAllocs},
		"the row should hold each revision's allocation medians, which outlying samples do not move")
	row, _, err = compareSamples("p.BenchmarkX", withTimings(100, 110, 120, 130, 140, 150, 160), withTimings(130, 140, 150, 160, 170, 180, 190), nil)
	require.NoError(t, err, "compareSamples must not error")
	assert.True(t, row.significant, "timings at p = 0.037 should differ significantly, as benchstat's p < 0.05 counts them")
	row, _, err = compareSamples("p.BenchmarkX", withTimings(100, 110, 120, 130, 140, 150, 160), withTimings(125, 135, 145, 155, 165, 175, 185), nil)
	require.NoError(t, err, "compareSamples must not error")
	assert.False(t, row.significant, "timings at p = 0.073 should not differ significantly, as benchstat's p < 0.05 counts them")

	base, head := steadySamples(7, 100, 6400, 100), steadySamples(7, 100, 6400, 101)
	row, _, err = compareSamples("p.BenchmarkX", base, head, &Entry{AllocsTolerance: 0.01, Reason: "measured"})
	require.NoError(t, err, "compareSamples must not error")
	assert.False(t, row.significant, "identical timings should not differ significantly")
	assert.InDelta(t, 1.0, row.p, 1e-9, "identical timings should have a p of 1")
	assert.False(t, row.allocationsMoved, "allocations within the entry's tolerance should not count as moved")
	row, _, err = compareSamples("p.BenchmarkX", base, head, nil)
	require.NoError(t, err, "compareSamples must not error")
	assert.True(t, row.allocationsMoved, "allocations that differ should count as moved without an entry")

	for _, tc := range []struct {
		name       string
		base, head *Samples
		entry      *Entry
		moved      bool
	}{
		{"bytes alone beyond the default tolerance", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 1020, 1), nil, true},
		{"bytes alone within the default tolerance", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 1005, 1), nil, false},
		{"bytes alone falling beyond the default tolerance", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 980, 1), nil, true},
		{"allocations alone falling", steadySamples(7, 100, 64, 2), steadySamples(7, 100, 64, 1), nil, true},
		// Measured against the base, a fall to the edge of the tolerance stays within it
		{"bytes falling to the default tolerance", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 990, 1), nil, false},
		{"allocations falling to the entry's tolerance", steadySamples(7, 100, 6400, 100), steadySamples(7, 100, 6400, 90), &Entry{AllocsTolerance: 0.1, Reason: "measured"}, false},
		{"bytes rising within the entry's tolerance, beyond the default", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 1020, 1), &Entry{BytesTolerance: new(0.05), Reason: "measured"}, false},
		{"bytes falling within the entry's tolerance, beyond the default", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 980, 1), &Entry{BytesTolerance: new(0.05), Reason: "measured"}, false},
		{"bytes rising within the default tolerance, beyond the entry's", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 1005, 1), &Entry{BytesTolerance: new(0.0), Reason: "measured"}, true},
		{"bytes falling within the default tolerance, beyond the entry's", steadySamples(7, 100, 1000, 1), steadySamples(7, 100, 995, 1), &Entry{BytesTolerance: new(0.0), Reason: "measured"}, true},
		{"bytes rising beyond the default tolerance under an allocations tolerance", steadySamples(7, 100, 1000, 100), steadySamples(7, 100, 1020, 100), &Entry{AllocsTolerance: 0.1, Reason: "measured"}, true},
		{"bytes falling beyond the default tolerance under an allocations tolerance", steadySamples(7, 100, 1000, 100), steadySamples(7, 100, 980, 100), &Entry{AllocsTolerance: 0.1, Reason: "measured"}, true},
		{"allocations rising under a bytes tolerance", steadySamples(7, 100, 1000, 100), steadySamples(7, 100, 1000, 101), &Entry{BytesTolerance: new(0.05), Reason: "measured"}, true},
		{"allocations falling under a bytes tolerance", steadySamples(7, 100, 1000, 100), steadySamples(7, 100, 1000, 99), &Entry{BytesTolerance: new(0.05), Reason: "measured"}, true},
	} {
		row, _, err := compareSamples("p.BenchmarkX", tc.base, tc.head, tc.entry)
		require.NoErrorf(t, err, "compareSamples must not error for %s", tc.name)
		assert.Equalf(t, tc.moved, row.allocationsMoved, "allocations should count as moved only beyond the tolerance, for %s", tc.name)
	}

	_, warnings, err = compareSamples("p.BenchmarkX", steadySamples(3, 100, 0, 0), steadySamples(3, 200, 0, 0), nil)
	require.NoError(t, err, "compareSamples must not error")
	assert.NotEmpty(t, warnings, "three samples a side should come with the test's warning")

	for _, side := range []string{"base", "head"} {
		samples := map[string]*Samples{"base": steadySamples(7, 100, 64, 1), "head": steadySamples(7, 100, 64, 1)}
		samples[side].NS[3] = math.NaN()
		stop := guardHeap(t)
		_, _, err = compareSamples("p.BenchmarkX", samples["base"], samples["head"], nil)
		stop()
		assert.ErrorIsf(t, err, errBadTiming, "a NaN timing at the %s should be refused", side)
	}

	shortened := func(metric func(s *Samples) *[]float64) *Samples {
		s := steadySamples(7, 100, 64, 1)
		*metric(s) = (*metric(s))[:6]
		return s
	}
	for _, tc := range []struct {
		name       string
		base, head *Samples
		text       string
	}{
		{"an even count of the base's timings", shortened(func(s *Samples) *[]float64 { return &s.NS }), steadySamples(7, 100, 64, 1), "p.BenchmarkX ns/op"},
		{"an even count of the base's allocations", shortened(func(s *Samples) *[]float64 { return &s.Allocs }), steadySamples(7, 100, 64, 1), "p.BenchmarkX allocs/op"},
		{"an even count of the head's bytes", steadySamples(7, 100, 64, 1), shortened(func(s *Samples) *[]float64 { return &s.Bytes }), "p.BenchmarkX B/op"},
	} {
		_, _, err := compareSamples("p.BenchmarkX", tc.base, tc.head, nil)
		assert.ErrorIsf(t, err, errEvenSampleCount, "compareSamples should refuse %s", tc.name)
		assert.ErrorContainsf(t, err, tc.text, "compareSamples should name the benchmark and metric for %s", tc.name)
	}
}

func TestComparisonCPU(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ base, head, exp string }{
		{"Example CPU", "Example CPU", "Example CPU"},
		{"One CPU", "Another CPU", ""},
		{"Example CPU", "", ""},
		{"", "", ""},
	} {
		c := &comparison{base: newRevision("base", ""), head: newRevision("head", "")}
		c.base.cpu, c.head.cpu = tc.base, tc.head
		assert.Equalf(t, tc.exp, c.cpu(), "cpu should name a processor only when both revisions name the same one, not for %q and %q", tc.base, tc.head)
	}
}

func TestComparisonRowTimingCell(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		row  comparisonRow
		exp  string
	}{
		{"a significant change as a percentage", comparisonRow{baseNS: 100, headNS: 50, p: 0.0012, significant: true}, "100 → 50.0 (-50.0%, p=0.001)"},
		{"a change that is not significant as ~", comparisonRow{baseNS: 100, headNS: 102, p: 0.25}, "100 → 102 (~, p=0.250)"},
	} {
		assert.Equalf(t, tc.exp, tc.row.timingCell(), "timingCell should give %s", tc.name)
	}
}

func TestComplete(t *testing.T) {
	t.Parallel()
	assert.True(t, complete(steadySamples(7, 100, 64, 1), 7), "seven samples of each metric should be complete for seven")
	assert.False(t, complete(steadySamples(7, 100, 64, 1), 5), "seven samples of each metric should not be complete for five")
	for name, short := range map[string]*Samples{
		"ns/op":     {NS: make([]float64, 6), Bytes: make([]float64, 7), Allocs: make([]float64, 7)},
		"B/op":      {NS: make([]float64, 7), Bytes: make([]float64, 6), Allocs: make([]float64, 7)},
		"allocs/op": {NS: make([]float64, 7), Bytes: make([]float64, 7), Allocs: make([]float64, 6)},
	} {
		assert.Falsef(t, complete(short, 7), "samples one %s short should not be complete", name)
	}
}

func TestDescribeCounts(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "6 ns/op, 7 B/op and 5 allocs/op", describeCounts(&Samples{NS: make([]float64, 6), Bytes: make([]float64, 7), Allocs: make([]float64, 5)}),
		"describeCounts should count each metric's samples")
}

func TestFormatChange(t *testing.T) {
	t.Parallel()
	for change, exp := range map[float64]string{0.5: "+50.0%", -0.25: "-25.0%", 0: "+0.0%", 1.234: "+123.4%", -1: "-100.0%"} {
		assert.Equalf(t, exp, formatChange(change), "formatChange should give %v as a signed percentage", change)
	}
}

func TestPlural(t *testing.T) {
	t.Parallel()
	for n, exp := range map[int]string{0: "0 packages", 1: "1 package", 2: "2 packages", 12: "12 packages"} {
		assert.Equalf(t, exp, plural(n, "package"), "plural should count %d packages", n)
	}
}

func TestRenderRows(t *testing.T) {
	t.Parallel()
	header := "| Benchmark | ns/op | B/op | allocs/op |\n| --- | --- | --- | --- |\n"
	var out bytes.Buffer
	renderRows(&out, []comparisonRow{
		{key: "p.BenchmarkSame/a|b", baseNS: 100, headNS: 100, p: 1, baseBytes: 64, headBytes: 64, baseAllocs: 1, headAllocs: 1},
		{key: "p.BenchmarkFaster", baseNS: 100, headNS: 50, p: 0.001, significant: true, baseBytes: 64, headBytes: 128, baseAllocs: 1, headAllocs: 2},
	})
	assert.Equal(t, header+
		"| `p.BenchmarkSame/a\\|b` | 100 → 100 (~, p=1.000) | 64 | 1 |\n"+
		"| `p.BenchmarkFaster` | 100 → 50.0 (-50.0%, p=0.001) | 64 → 128 (+100.0%) | 1 → 2 (+100.0%) |\n",
		out.String(), "renderRows should write a table row per benchmark, in the order given")
	out.Reset()
	renderRows(&out, nil)
	assert.Equal(t, header, out.String(), "renderRows should write only the header for no rows")
}
