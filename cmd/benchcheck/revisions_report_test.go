package main

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
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

func TestComparisonCompareRejectsBadTimings(t *testing.T) {
	t.Parallel()
	c := &comparison{base: newRevision("base", ""), head: newRevision("head", ""), samples: 3}
	c.base.samples["p.BenchmarkX"] = steadySamples(3, 100, 0, 0)
	c.head.samples["p.BenchmarkX"] = steadySamples(3, 100, 0, 0)
	c.head.samples["p.BenchmarkX"].NS[1] = math.NaN()
	// benchmath's U test never returns for a NaN, so without the check this would hang rather than fail
	done := make(chan error, 1)
	go func() { done <- c.compare(Baseline{}) }()
	select {
	case err := <-done:
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

	assert.Contains(t, out, "Base `0123456789ab` against head `fedcba987654` with uncommitted changes: 8 benchmarks in 1 package,",
		"the report should say what it compared")
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
