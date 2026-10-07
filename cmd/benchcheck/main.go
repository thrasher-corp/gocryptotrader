// benchcheck compares `go test -bench` output against a checked-in baseline of per-benchmark
// allocation budgets, failing on any regression and on any improvement that has not been folded
// back into the baseline.
//
// Normally invoked through the Makefile, which owns the measurement flags:
//
//	make bench          # check
//	make bench_update   # record, pruning entries with no result
//	make bench_apply    # record from output already measured, such as CI's
//
// Called directly, read from a file rather than a pipe. In a pipeline both processes run at once,
// so this can read partial output and, with -update, save a baseline before the shell has seen
// whether go test succeeded:
//
//	go test -run '^$' -bench . -benchmem -benchtime 100ms -count 7 -cpu 4 -p 1 -timeout 20m $(benchcheck -list) > out.txt
//	benchcheck -samples 7 < out.txt
//
// BENCH_FLAGS in the Makefile is the definition of those flags; keep this example in step with it.
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

// canonicalGOOS and canonicalGOARCH are the platform the benchmarks workflow measures on. Budgets
// are only recorded from output produced there, because allocation counts are not guaranteed to
// match across platforms, and package discovery evaluates build constraints for it.
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
	errWrongPlatform      = errors.New("budgets are only recorded from " + canonicalGOOS + "/" + canonicalGOARCH + " output")
)

type options struct {
	baselinePath string
	packagesPath string
	root         string
	samples      int
	update       bool
	prune        bool
	list         bool
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

	if err := run(os.Stdin, os.Stdout, &o); err != nil {
		fmt.Fprintln(os.Stderr, "benchcheck:", err)
		os.Exit(1)
	}
}

func run(in io.Reader, out io.Writer, o *options) error {
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

	// Required rather than defaulted: the compared value is the median, which only exists for an
	// odd count, and only the caller knows how many samples it asked go test for
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
	if len(measured.Samples) == 0 {
		return errNoResults
	}
	// A result from a package outside the list has nowhere to go: recording it creates a budget
	// that every later check reports as belonging to no listed package
	configured := configuredSet(pkgs.List)
	var unlisted []string
	for _, key := range sortedKeys(measured.Samples) {
		if pkg := measured.Samples[key].Pkg; !configured[pkg] && !slices.Contains(unlisted, pkg) {
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
		// A budget recorded elsewhere fails the gate the first time CI measures it, and nothing in
		// the diff would say why
		if measured.Platform != platform {
			return fmt.Errorf("%w, not %q; fold in the output CI uploads with 'make bench_apply' instead",
				errWrongPlatform, measured.Platform)
		}
		if o.prune {
			if unfinished := unfinishedPackages(measured.Finished, results, pkgs.List); len(unfinished) > 0 {
				return fmt.Errorf("%w: %s", errIncompleteRun, strings.Join(unfinished, ", "))
			}
		}
		for _, change := range base.Update(results, o.prune) {
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
	findings := Compare(base, results, pkgs.List)
	slices.SortStableFunc(findings, func(a, b Finding) int { return int(a.Kind) - int(b.Kind) })
	for _, f := range findings {
		fmt.Fprintln(out, f)
	}
	if len(findings) == 0 {
		fmt.Fprintf(out, "benchcheck: %d benchmarks within budget\n", len(results))
		return nil
	}
	return fmt.Errorf("%d findings", len(findings))
}

// silentPackages returns, in list order, the listed packages with no result
func silentPackages(results map[string]*Result, listed []string) []string {
	reported := seenPackages(results)
	var silent []string
	for _, pkg := range listed {
		if !reported[pkg] {
			silent = append(silent, pkg)
		}
	}
	return silent
}

// unfinishedPackages returns, in list order, the listed packages a run cannot vouch for: those with
// no result, and those whose output stopped before go test's closing "ok" line. Pruning on the
// second would read every benchmark the run never reached as one that had been deleted.
func unfinishedPackages(finished map[string]bool, results map[string]*Result, listed []string) []string {
	reported := seenPackages(results)
	var unfinished []string
	for _, pkg := range listed {
		if !finished[pkg] || !reported[pkg] {
			unfinished = append(unfinished, pkg)
		}
	}
	return unfinished
}
