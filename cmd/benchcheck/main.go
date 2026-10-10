// benchcheck compares `go test -bench` output against a checked-in baseline of per-benchmark
// allocation budgets, failing on any regression and reporting any improvement not yet folded into it.
//
// The Makefile owns the measurement flags:
//
//	make bench          # check
//	make bench_update   # record, pruning entries with no result
//	make bench_apply    # record from output already measured, such as CI's
//
// Called directly, give it go test's output in a file, and only once go test has succeeded, since
// output cut short can still pass a check:
//
//	GOFLAGS= GOEXPERIMENT= GOGC= GOMEMLIMIT= go test -run '^$' -bench . -benchmem -benchtime 100ms -count 7 -cpu 4 -p 1 -v $(benchcheck -list) > out.txt &&
//		benchcheck -samples 7 < out.txt
//
// Keep this example in step with BENCH_ENV and BENCH_FLAGS in the Makefile.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// canonicalGOOS and canonicalGOARCH are the platform budgets are recorded on, since allocation counts
// can differ between platforms
const (
	canonicalGOOS   = "linux"
	canonicalGOARCH = "amd64"
)

var (
	errNoResults          = errors.New("no benchmark results found on stdin")
	errInvalidSamples     = errors.New("-samples must be the odd -count passed to go test")
	errPruneWithoutUpdate = errors.New("-prune only applies with -update")
	errUnlistedOutput     = errors.New("benchmark output includes packages the package list does not benchmark")
	errIncompleteRun      = errors.New("refusing to prune on a run that did not finish every listed package")
	errPruneNeedsVerbose  = errors.New("refusing to prune on output from go test without -v, which says nothing of a benchmark that skips before measuring")
	errWrongPlatform      = errors.New("budgets are only recorded from " + canonicalGOOS + "/" + canonicalGOARCH + " output")
	errFindings           = errors.New("findings against the baseline")
	errUnexpectedArgument = errors.New("unexpected argument")
	errSkipRecordBudgets  = errors.New("refusing to prune budgets that a skip record in the output names, since output a benchmark printed can hide its skip; remove them by hand once the benchmark is gone")
)

type options struct {
	baselinePath string
	packagesPath string
	root         string
	samples      int
	update       bool
	prune        bool
	list         bool
	args         []string
}

func main() {
	o := options{root: "."}
	flag.StringVar(&o.baselinePath, "baseline", "benchmarks/baseline.json", "path to the baseline file")
	flag.StringVar(&o.packagesPath, "packages", "benchmarks/packages.txt", "path to the benchmarked package list")
	flag.IntVar(&o.samples, "samples", 0, "the -count passed to go test, which must be odd; every benchmark must report exactly this many samples")
	flag.BoolVar(&o.update, "update", false, "fold the results into the baseline instead of checking them")
	flag.BoolVar(&o.prune, "prune", false, "with -update, delete baseline entries with no result; refused unless every listed package reported")
	flag.BoolVar(&o.list, "list", false, "check the package list against the packages declaring benchmarks, then print it as go test arguments")
	flag.Parse()
	o.args = flag.Args()

	if err := run(os.Stdin, os.Stdout, &o); err != nil {
		fmt.Fprintln(os.Stderr, "benchcheck:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, o *options) error {
	// A file named as an argument would be ignored while benchcheck waited on stdin
	if len(o.args) > 0 {
		return fmt.Errorf("%w %q: pass go test's output on stdin", errUnexpectedArgument, o.args[0])
	}
	pkgs, err := LoadPackages(o.packagesPath)
	if err != nil {
		return err
	}
	if o.list {
		declared, err := BenchmarkPackages(o.root)
		if err != nil {
			return err
		}
		if err := pkgs.Covers(declared); err != nil {
			return err
		}
		args := make([]string, len(pkgs.List))
		for i, p := range pkgs.List {
			args[i] = "./" + p + "/"
		}
		fmt.Fprintln(out, strings.Join(args, " "))
		return nil
	}

	if o.samples <= 0 || o.samples%2 == 0 {
		return fmt.Errorf("%w: got %d", errInvalidSamples, o.samples)
	}
	if o.prune && !o.update {
		return errPruneWithoutUpdate
	}

	measured, err := Parse(in)
	if err != nil {
		return err
	}
	// A run in which everything skipped still has skips to report
	if len(measured.Samples) == 0 && len(measured.Skipped) == 0 {
		return errNoResults
	}
	// An unlisted package has no budgets to check against, and its header may be text a benchmark printed
	configured := configuredSet(pkgs.List)
	var unlisted []string
	for _, pkg := range sortedKeys(measured.Headers) {
		if !configured[pkg] {
			unlisted = append(unlisted, pkg)
		}
	}
	if len(unlisted) > 0 {
		return fmt.Errorf("%w: %s", errUnlistedOutput, strings.Join(unlisted, ", "))
	}
	if mismatched := SampleCountMismatches(measured.Samples, o.samples); len(mismatched) > 0 {
		for _, key := range mismatched {
			fmt.Fprintf(out, "benchcheck: %s reported %s\n", key, describeSamples(measured.Samples[key], o.samples))
		}
		return fmt.Errorf("%w: %d of %d benchmarks", errSampleCount, len(mismatched), len(measured.Samples))
	}
	results, err := Summarise(measured.Samples)
	if err != nil {
		return err
	}

	base, err := LoadBaseline(o.baselinePath, o.update)
	if err != nil {
		return err
	}
	if err := base.Validate(); err != nil {
		return err
	}

	platform := canonicalGOOS + "/" + canonicalGOARCH
	if o.update {
		// A budget recorded elsewhere would fail CI's next check with nothing in the diff to say why
		if measured.Platform != platform {
			return fmt.Errorf("%w, not %q; fold in the output CI uploads with 'make bench_apply' instead",
				errWrongPlatform, measured.Platform)
		}
		if o.prune {
			if unfinished := unfinishedPackages(measured.Finished, results, measured.Skipped, pkgs.List); len(unfinished) > 0 {
				return fmt.Errorf("%w: %s", errIncompleteRun, strings.Join(unfinished, ", "))
			}
			if quiet := quietPackages(measured.Verbose, pkgs.List); len(quiet) > 0 {
				return fmt.Errorf("%w: %s", errPruneNeedsVerbose, strings.Join(quiet, ", "))
			}
			if named := skipRecordBudgets(base, results, measured.Skipped, measured.SkipRecords); len(named) > 0 {
				return fmt.Errorf("%w: %s", errSkipRecordBudgets, strings.Join(named, ", "))
			}
		}
		for _, change := range base.Update(results, measured.Skipped, o.prune) {
			fmt.Fprintln(out, "benchcheck:", change)
		}
		if err := base.Save(o.baselinePath); err != nil {
			return err
		}
		fmt.Fprintf(out, "benchcheck: %d benchmarks folded into %s\n", len(results), o.baselinePath)
		return nil
	}

	if measured.Platform != platform {
		fmt.Fprintf(out, "benchcheck: budgets are recorded on %s but this output is from %q, so a finding may be the platform rather than the code\n",
			platform, measured.Platform)
	}
	findings := Compare(base, results, measured.Skipped, pkgs.List)
	slices.SortStableFunc(findings, func(a, b Finding) int { return int(a.Kind) - int(b.Kind) })
	var failing int
	improved := make(map[string]bool)
	for _, f := range findings {
		fmt.Fprintln(out, f)
		if f.Kind == Stale {
			improved[f.Key] = true
		} else {
			failing++
		}
	}
	if failing > 0 {
		return fmt.Errorf("%w: %d", errFindings, failing)
	}
	summary := fmt.Sprintf("benchcheck: %d benchmarks within budget", len(results))
	if len(improved) > 0 {
		summary += fmt.Sprintf(", %d improved; run 'make bench_update' to lower their budgets", len(improved))
	}
	fmt.Fprintln(out, summary)
	return nil
}

func silentPackages(results map[string]*Result, skipped map[string]string, listed []string) []string {
	reported := reportedPackages(results, skipped)
	var silent []string
	for _, pkg := range listed {
		if !reported[pkg] {
			silent = append(silent, pkg)
		}
	}
	return silent
}

// skipRecordBudgets returns, in key order, the budgets a prune would delete that skip record text
// names, since printed text can hide go test's own record from Skipped
func skipRecordBudgets(base Baseline, results map[string]*Result, skipped, records map[string]string) []string {
	var named []string
	for _, key := range sortedKeys(base) {
		if _, ok := results[key]; ok {
			continue
		}
		if _, ok := skipCovering(skipped, key); ok {
			continue
		}
		if _, ok := skipCovering(records, key); ok {
			named = append(named, key)
		}
	}
	return named
}

func quietPackages(verbose map[string]bool, listed []string) []string {
	var quiet []string
	for _, pkg := range listed {
		if !verbose[pkg] {
			quiet = append(quiet, pkg)
		}
	}
	return quiet
}

// unfinishedPackages returns, in list order, the listed packages that reported nothing or whose output
// stopped before go test's ok line
func unfinishedPackages(finished map[string]bool, results map[string]*Result, skipped map[string]string, listed []string) []string {
	reported := reportedPackages(results, skipped)
	var unfinished []string
	for _, pkg := range listed {
		if !finished[pkg] || !reported[pkg] {
			unfinished = append(unfinished, pkg)
		}
	}
	return unfinished
}
