// benchcheck compares `go test -bench` output against a checked-in baseline of per-benchmark
// allocation budgets, failing on any regression and on any improvement that has not been folded
// back into the baseline.
//
// Normally invoked through the Makefile, which owns the measurement flags:
//
//	make bench          # check
//	make bench_update   # record, pruning entries with no result
//	make bench_trend    # append to the ns/op history
//
// Called directly, read from a file rather than a pipe. In a pipeline both processes run at once,
// so this can read partial output and, with -update, save a baseline before the shell has seen
// whether go test succeeded:
//
//	go test -run '^$' -bench . -benchmem -count 6 -cpu 4 -p 1 ./currency/ > out.txt
//	benchcheck < out.txt
//	benchcheck -list
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"
)

var errNoResults = errors.New("no benchmark results found on stdin")

type options struct {
	baselinePath   string
	packagesPath   string
	seriesPath     string
	commit         string
	update         bool
	prune          bool
	list           bool
	bytesTolerance float64
	warn           bool
}

func main() {
	var o options
	flag.StringVar(&o.baselinePath, "baseline", "benchmarks/baseline.json", "path to the baseline file")
	flag.StringVar(&o.packagesPath, "packages", "benchmarks/packages.txt", "path to the benchmarked package list")
	flag.StringVar(&o.seriesPath, "series", "", "append the results to this JSON lines history file")
	flag.StringVar(&o.commit, "commit", "", "commit SHA to label series records with")
	flag.BoolVar(&o.update, "update", false, "fold the results into the baseline instead of checking them")
	flag.BoolVar(&o.prune, "prune", false, "with -update, delete baseline entries with no result; only safe when -bench matched every benchmark")
	flag.BoolVar(&o.list, "list", false, "print the benchmarked packages as go test arguments and exit")
	flag.Float64Var(&o.bytesTolerance, "bytes-tolerance", 0.01, "fractional B/op movement to ignore")
	flag.BoolVar(&o.warn, "warn", false, "report findings but always exit 0")
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
		args := make([]string, len(pkgs.List))
		for i, p := range pkgs.List {
			args[i] = "./" + p + "/"
		}
		fmt.Fprintln(out, strings.Join(args, " "))
		return nil
	}

	results, err := Parse(in)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return errNoResults
	}

	base, err := LoadBaseline(o.baselinePath, o.update)
	if err != nil {
		return err
	}
	if err := base.Validate(); err != nil {
		return err
	}
	// The per-entry overrides are validated by Validate; the global flag reached the comparison
	// unchecked, so -bytes-tolerance NaN disabled every B/op comparison silently.
	if err := validTolerance(o.bytesTolerance); err != nil {
		return fmt.Errorf("-bytes-tolerance: %w", err)
	}

	// Appended only once nothing else can reject the run. Writing first meant an invalid baseline or
	// tolerance appended a full run and then exited non-zero, so the obvious retry duplicated it.
	if o.seriesPath != "" {
		if err := AppendSeries(o.seriesPath, o.commit, results, time.Now()); err != nil {
			return err
		}
		fmt.Fprintf(out, "benchcheck: appended %d records to %s\n", len(results), o.seriesPath)
	}

	if o.update {
		// Pruning is opt-in rather than inferred. "every package reported something" is not
		// evidence of a full run: `-bench BenchmarkOne` across every package satisfies it while
		// selecting a single benchmark each, and deleting on that basis destroys the baseline.
		// Only the caller knows whether -bench matched everything, so only the caller may say so.
		if !o.prune {
			for _, key := range sortedKeys(base) {
				if _, ok := results[key]; !ok && packageOf(key, seenPackages(results)) != "" {
					fmt.Fprintf(out, "benchcheck: %s has no result in this run; keeping it (pass -prune to delete)\n", key)
				}
			}
		}
		for _, key := range base.Update(results, o.prune) {
			fmt.Fprintf(out, "benchcheck: pruned %s\n", key)
		}
		if err := base.Save(o.baselinePath); err != nil {
			return err
		}
		fmt.Fprintf(out, "benchcheck: recorded %d benchmarks in %s\n", len(results), o.baselinePath)
		return nil
	}

	findings := Compare(base, results, configuredSet(pkgs.List), pkgs.Gated, o.bytesTolerance)
	slices.SortStableFunc(findings, func(a, b Finding) int { return int(a.Kind) - int(b.Kind) })
	for _, f := range findings {
		fmt.Fprintln(out, f)
	}
	if len(findings) == 0 {
		fmt.Fprintf(out, "benchcheck: %d benchmarks within budget\n", len(results))
		return nil
	}
	if o.warn {
		fmt.Fprintf(out, "benchcheck: %d findings (warn mode)\n", len(findings))
		return nil
	}
	return fmt.Errorf("%d findings", len(findings))
}
