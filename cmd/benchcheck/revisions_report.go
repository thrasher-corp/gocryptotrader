package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/perf/benchmath"
)

var errBadTiming = errors.New("benchmark timing is not a finite, non-negative number")

// highlightDelta is the smallest significant timing change the summary leads with. Runs of identical
// code on one machine still find a few significant changes by chance, which this keeps out of the lead
// only when they are small.
const highlightDelta = 0.05

// comparisonRow holds one benchmark's medians at both revisions
type comparisonRow struct {
	key              string
	baseNS, headNS   float64
	p                float64
	significant      bool
	baseBytes        uint64
	headBytes        uint64
	baseAllocs       uint64
	headAllocs       uint64
	allocationsMoved bool
}

// compare reduces each benchmark measured at both revisions to a row, and records why each other one
// was not compared
func (c *comparison) compare(baseline Baseline) error {
	c.notCompared = make(map[string]string)
	keys := make(map[string]bool)
	for _, r := range []*revision{c.base, c.head} {
		for key := range r.samples {
			keys[key] = true
		}
		for key := range r.skipped {
			keys[key] = true
		}
	}
	warnings := make(map[string]bool)
	for _, key := range sortedKeys(keys) {
		b, atBase := c.base.samples[key]
		h, atHead := c.head.samples[key]
		switch {
		case c.base.skipped[key] && c.head.skipped[key]:
			c.notCompared[key] = "skipped at both"
		case c.base.skipped[key]:
			c.notCompared[key] = "skipped at the base"
		case c.head.skipped[key]:
			c.notCompared[key] = "skipped at the head"
		case !atBase:
			c.notCompared[key] = "only at the head"
		case !atHead:
			c.notCompared[key] = "only at the base"
		case !complete(b, c.samples) || !complete(h, c.samples):
			c.notCompared[key] = fmt.Sprintf("expected %d samples of each metric, got %s at the base and %s at the head",
				c.samples, describeCounts(b), describeCounts(h))
		default:
			row, rowWarnings, err := compareSamples(key, b, h, baseline[key])
			if err != nil {
				return err
			}
			c.rows = append(c.rows, row)
			for _, w := range rowWarnings {
				warnings[w.Error()] = true
			}
		}
	}
	c.warnings = sortedKeys(warnings)
	return nil
}

func complete(s *Samples, n int) bool {
	return len(s.NS) == n && len(s.Bytes) == n && len(s.Allocs) == n
}

func describeCounts(s *Samples) string {
	return fmt.Sprintf("%d ns/op, %d B/op and %d allocs/op", len(s.NS), len(s.Bytes), len(s.Allocs))
}

// compareSamples tests the timings with benchstat's default, a Mann-Whitney U test, and counts the
// allocations as moved once their medians differ by more than the gate's tolerance for the benchmark
func compareSamples(key string, base, head *Samples, entry *Entry) (comparisonRow, []error, error) {
	// Parse records a timing it cannot read as NaN, and benchmath's U test never returns for one
	if slices.ContainsFunc(slices.Concat(base.NS, head.NS), math.IsNaN) {
		return comparisonRow{}, nil, fmt.Errorf("%w: %s", errBadTiming, key)
	}
	row := comparisonRow{key: key}
	medians := []struct {
		dst     *float64
		samples []float64
	}{
		{&row.baseNS, base.NS}, {&row.headNS, head.NS},
	}
	for _, m := range medians {
		v, err := median(m.samples)
		if err != nil {
			return comparisonRow{}, nil, fmt.Errorf("%s ns/op: %w", key, err)
		}
		*m.dst = v
	}
	baseResult, err := Summarise(map[string]*Samples{key: base})
	if err != nil {
		return comparisonRow{}, nil, err
	}
	headResult, err := Summarise(map[string]*Samples{key: head})
	if err != nil {
		return comparisonRow{}, nil, err
	}
	row.baseBytes, row.baseAllocs = baseResult[key].Bytes, baseResult[key].Allocs
	row.headBytes, row.headAllocs = headResult[key].Bytes, headResult[key].Allocs

	// NewSample sorts what it is given
	result := benchmath.AssumeNothing.Compare(
		benchmath.NewSample(slices.Clone(base.NS), &benchmath.DefaultThresholds),
		benchmath.NewSample(slices.Clone(head.NS), &benchmath.DefaultThresholds))
	row.p, row.significant = result.P, result.P < result.Alpha

	if entry == nil {
		entry = &Entry{}
	}
	row.allocationsMoved = budgetDrift(row.headAllocs, row.baseAllocs, entry.AllocsTolerance) != 0 ||
		budgetDrift(row.headBytes, row.baseBytes, entry.bytesTolerance()) != 0
	return row, result.Warnings, nil
}

// moved reports whether the summary leads with the row
func (r *comparisonRow) moved() bool {
	return r.allocationsMoved || (r.significant && math.Abs(relativeChange(r.baseNS, r.headNS)) >= highlightDelta)
}

// relativeChange returns the change from base to head as a fraction of base
func relativeChange(base, head float64) float64 {
	if base == 0 {
		if head == 0 {
			return 0
		}
		return math.Inf(1)
	}
	return head/base - 1
}

// writeOutput replaces an earlier run's files in dir with each revision's raw go test output, which
// benchstat also reads
func (c *comparison) writeOutput(dir string) error {
	if err := removeEarlierOutput(dir); err != nil {
		return err
	}
	for _, r := range []*revision{c.base, c.head} {
		if err := os.WriteFile(filepath.Join(dir, r.label+".txt"), r.output.Bytes(), 0o644); err != nil {
			return fmt.Errorf("error saving the %s output: %w", r.label, err)
		}
	}
	return nil
}

// write saves the raw output and the summary to dir, and prints the summary
func (c *comparison) write(out io.Writer, dir string) error {
	var summary bytes.Buffer
	c.render(&summary)
	if c.predatesList {
		if err := removeEarlierOutput(dir); err != nil {
			return err
		}
	} else if err := c.writeOutput(dir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.md"), summary.Bytes(), 0o644); err != nil {
		return fmt.Errorf("error saving the summary: %w", err)
	}
	_, err := out.Write(summary.Bytes())
	return err
}

// removeEarlierOutput removes the files an earlier comparison wrote to dir, its report first, so that no
// report is left beside output it does not describe
func removeEarlierOutput(dir string) error {
	for _, name := range []string{"summary.md", "base.txt", "head.txt"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("error removing the earlier %s: %w", name, err)
		}
	}
	return nil
}

// render writes the comparison as Markdown, for a terminal or a GitHub job summary
func (c *comparison) render(w io.Writer) {
	fmt.Fprint(w, "## Benchmark comparison\n\n")
	base, head := codeSpan(shortCommit(c.base.commit)), codeSpan(shortCommit(c.head.commit))
	if c.dirty {
		head += " with uncommitted changes"
	}
	if c.predatesList {
		fmt.Fprintf(w, "The base, %s, predates `benchmarks/packages.txt`, so there are no benchmarks listed to compare %s against.\n", base, head)
		return
	}
	fmt.Fprintf(w, "Base %s against head %s: %s in %s, measured in %d rounds that alternate which revision runs first, each sample a process of its own",
		base, head, plural(len(c.rows), "benchmark"), plural(len(c.packages), "package"), c.samples)
	if c.cpu() != "" {
		fmt.Fprintf(w, ", on %s", c.cpu())
	}
	fmt.Fprint(w, ".\n\n")
	fmt.Fprintf(w, "This comparison is advisory: no difference fails it. A timing marked `~` shows no significant change "+
		"(Mann-Whitney U test, p ≥ %v), and with this many benchmarks a few can test significant by chance alone. "+
		"Allocations count as moved once their medians differ by more than the baseline's tolerance.\n\n", benchmath.DefaultThresholds.CompareAlpha)

	var moved []comparisonRow
	for _, row := range c.rows {
		if row.moved() {
			moved = append(moved, row)
		}
	}
	fmt.Fprintf(w, "### Moved (%d)\n\n", len(moved))
	if len(moved) == 0 {
		fmt.Fprintf(w, "No allocations moved, and no timing changed significantly by %v%% or more.\n\n", highlightDelta*100)
	} else {
		fmt.Fprintf(w, "Allocations that moved, and timings that changed significantly by %v%% or more.\n\n", highlightDelta*100)
		renderRows(w, moved)
		fmt.Fprintln(w)
	}
	if len(c.rows) > 0 {
		fmt.Fprintf(w, "<details>\n<summary>All %s</summary>\n\n", plural(len(c.rows), "benchmark"))
		renderRows(w, c.rows)
		fmt.Fprint(w, "\n</details>\n\n")
	}

	notCompared := len(c.onlyHead) + len(c.onlyBase) + len(c.notCompared)
	if notCompared > 0 {
		fmt.Fprintf(w, "### Not compared (%d)\n\n", notCompared)
		for _, pkg := range c.onlyHead {
			fmt.Fprintf(w, "- %s: package only listed at the head\n", codeSpan(pkg))
		}
		for _, pkg := range c.onlyBase {
			fmt.Fprintf(w, "- %s: package only listed at the base\n", codeSpan(pkg))
		}
		for _, key := range sortedKeys(c.notCompared) {
			fmt.Fprintf(w, "- %s: %s\n", codeSpan(key), c.notCompared[key])
		}
		fmt.Fprintln(w)
	}
	if len(c.warnings) > 0 {
		fmt.Fprint(w, "### Notes\n\n")
		for _, warning := range c.warnings {
			fmt.Fprintf(w, "- %s\n", warning)
		}
		fmt.Fprintln(w)
	}
}

// cpu returns the processor both revisions ran on, or "" if their outputs disagree or name none
func (c *comparison) cpu() string {
	if c.base.cpu != c.head.cpu {
		return ""
	}
	return c.base.cpu
}

func renderRows(w io.Writer, rows []comparisonRow) {
	fmt.Fprint(w, "| Benchmark | ns/op | B/op | allocs/op |\n| --- | --- | --- | --- |\n")
	for _, row := range rows {
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", codeSpan(row.key), row.timingCell(),
			countCell(row.baseBytes, row.headBytes), countCell(row.baseAllocs, row.headAllocs))
	}
}

func (r *comparisonRow) timingCell() string {
	delta := "~"
	if r.significant {
		delta = formatChange(relativeChange(r.baseNS, r.headNS))
	}
	return fmt.Sprintf("%s → %s (%s, p=%.3f)", formatNS(r.baseNS), formatNS(r.headNS), delta, r.p)
}

func countCell(base, head uint64) string {
	if base == head {
		return strconv.FormatUint(base, 10)
	}
	if base == 0 {
		return fmt.Sprintf("0 → %d", head)
	}
	return fmt.Sprintf("%d → %d (%s)", base, head, formatChange(relativeChange(float64(base), float64(head))))
}

func formatChange(change float64) string {
	return fmt.Sprintf("%+.1f%%", change*100)
}

// formatNS gives a timing three or four significant figures, which is as many as a run on a shared
// machine can support
func formatNS(ns float64) string {
	switch {
	case ns >= 100:
		return strconv.FormatFloat(ns, 'f', 0, 64)
	case ns >= 10:
		return strconv.FormatFloat(ns, 'f', 1, 64)
	default:
		return strconv.FormatFloat(ns, 'f', 2, 64)
	}
}

// codeSpan quotes a benchmark key for a Markdown table cell, where a pipe would end the cell. A key
// holding backticks gets a longer run of them around it, padded with spaces where it starts or ends
// with one, so that its own cannot end the span.
func codeSpan(s string) string {
	fence := "`"
	for strings.Contains(s, fence) {
		fence += "`"
	}
	s = strings.ReplaceAll(s, "|", `\|`)
	if strings.HasPrefix(s, "`") || strings.HasSuffix(s, "`") {
		s = " " + s + " "
	}
	return fence + s + fence
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}
