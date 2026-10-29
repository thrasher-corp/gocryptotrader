package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOptions returns options for run against a fresh directory listing sampleOutput's packages
func testOptions(t *testing.T) *options {
	t.Helper()
	dir := t.TempDir()
	o := &options{
		baselinePath: filepath.Join(dir, "baseline.json"),
		packagesPath: filepath.Join(dir, "packages.txt"),
		root:         dir,
		samples:      3,
	}
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\nexchanges/orderbook\n"), 0o600),
		"writing the package list must not error")
	return o
}

// TestMainExitStatus runs main in a subprocess, since its exit status is what make and CI read
func TestMainExitStatus(t *testing.T) {
	t.Parallel()
	if baseline := os.Getenv("BENCHCHECK_TEST_MAIN_BASELINE"); baseline != "" {
		os.Args = []string{"benchcheck", "-samples", "3", "-baseline", baseline, "-packages", os.Getenv("BENCHCHECK_TEST_MAIN_PACKAGES")}
		main()
		return
	}
	o := testOptions(t)
	base := Baseline{
		"currency.BenchmarkNewCode":                       {Allocs: 2, Bytes: 16},
		"exchanges/orderbook.BenchmarkProcess":            {Allocs: 0, Bytes: 0},
		"exchanges/orderbook.BenchmarkSortAsksDescending": {Allocs: 1, Bytes: 24},
	}
	for _, tc := range []struct {
		name     string
		budget   uint64
		findings bool
	}{
		{name: "within budget", budget: 2},
		{name: "a regression", budget: 1, findings: true},
		{name: "an improvement", budget: 3},
	} {
		base["currency.BenchmarkNewCode"].Allocs = tc.budget
		require.NoErrorf(t, base.Save(o.baselinePath), "saving the baseline must not error for %s", tc.name)
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMainExitStatus$") //nolint:gosec // re-runs this test binary to exercise main
		cmd.Env = append(os.Environ(), "BENCHCHECK_TEST_MAIN_BASELINE="+o.baselinePath, "BENCHCHECK_TEST_MAIN_PACKAGES="+o.packagesPath)
		cmd.Stdin = strings.NewReader(sampleOutput)
		out, err := cmd.CombinedOutput()
		if !tc.findings {
			assert.NoErrorf(t, err, "main should exit zero for %s: %s", tc.name, out)
			continue
		}
		assert.Errorf(t, err, "main should exit non-zero for %s: %s", tc.name, out)
		assert.Contains(t, string(out), "findings against the baseline", "main should print the findings error")
	}
}

func TestMainRejectsArguments(t *testing.T) {
	t.Parallel()
	if os.Getenv("BENCHCHECK_TEST_MAIN_ARGUMENT") != "" {
		os.Args = []string{"benchcheck", "-samples", "3", "out.txt"}
		main()
		return
	}
	// stdin stays open, so a main that read it before rejecting the argument would wait out the deadline
	stdin, stdinWriter, err := os.Pipe()
	require.NoError(t, err, "os.Pipe must not error")
	t.Cleanup(func() {
		assert.NoError(t, stdin.Close(), "closing stdin should not error")
		assert.NoError(t, stdinWriter.Close(), "closing stdin's writer should not error")
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainRejectsArguments$") //nolint:gosec // re-runs this test binary to exercise main
	cmd.Env = append(os.Environ(), "BENCHCHECK_TEST_MAIN_ARGUMENT=1")
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	require.NoError(t, ctx.Err(), "main must reject the argument without waiting on stdin")
	assert.Error(t, err, "main should exit non-zero when given a file argument")
	assert.Contains(t, string(out), "unexpected argument", "main should refuse a file argument rather than read stdin")
}

func TestRunList(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	const bench = "package p\n\nimport \"testing\"\n\nfunc BenchmarkX(b *testing.B) {}\n"
	writeFiles(t, o.root, map[string]string{"currency/x_test.go": bench, "exchanges/orderbook/x_test.go": bench})

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(""), &out, &options{packagesPath: o.packagesPath, root: o.root, list: true}),
		"-list must not error")
	assert.Equal(t, "./currency/ ./exchanges/orderbook/\n", out.String(), "-list should emit go test package arguments in file order")
}

func TestRunListRejectsUncoveredPackages(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	const bench = "package p\n\nimport \"testing\"\n\nfunc BenchmarkX(b *testing.B) {}\n"
	writeFiles(t, o.root, map[string]string{
		"currency/x_test.go":            bench,
		"exchanges/orderbook/x_test.go": bench,
		"types/decimal/x_test.go":       bench,
	})
	var out bytes.Buffer
	err := run(strings.NewReader(""), &out, &options{packagesPath: o.packagesPath, root: o.root, list: true})
	assert.ErrorIs(t, err, errUnlistedPackage, "-list should refuse while a benchmark package is neither listed nor excluded")
	assert.Empty(t, out.String(), "-list should print nothing for make to run when it fails")
}

func TestRunUpdateThenCheck(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o), "seeding the baseline must not error")
	assert.Contains(t, out.String(), "added currency.BenchmarkNewCode: allocs/op 2, B/op 16", "the update should describe each addition")
	assert.Contains(t, out.String(), "3 benchmarks folded into", "the update should report what it recorded")

	out.Reset()
	o.update = false
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o),
		"checking the same output against the seeded baseline must not error")
	assert.Contains(t, out.String(), "3 benchmarks within budget", "a clean check should say so")
}

func TestRunReportsRegression(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	base := Baseline{
		"currency.BenchmarkNewCode":                       {Allocs: 0, Bytes: 0},
		"exchanges/orderbook.BenchmarkProcess":            {Allocs: 0, Bytes: 0},
		"exchanges/orderbook.BenchmarkSortAsksDescending": {Allocs: 1, Bytes: 24},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	var out bytes.Buffer
	err := run(strings.NewReader(sampleOutput), &out, o)
	assert.ErrorIs(t, err, errFindings, "a regression should fail the run")
	assert.EqualError(t, err, errFindings.Error()+": 2", "the error should count both findings")
	assert.Contains(t, out.String(), "regression currency.BenchmarkNewCode: allocs/op 0 -> 2", "the finding should be printed")
}

func TestRunFailsOnUntrackedAndMissing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, packages, exp string
		edit                func(Baseline)
	}{
		{name: "a benchmark without a budget", edit: func(b Baseline) { delete(b, "currency.BenchmarkNewCode") }, exp: "untracked  currency.BenchmarkNewCode:"},
		{name: "a budget without a benchmark", edit: func(b Baseline) { b["currency.BenchmarkGone"] = &Entry{Allocs: 1, Bytes: 8} }, exp: "missing    currency.BenchmarkGone:"},
		{name: "a listed package that reported nothing", packages: "currency\nexchanges/orderbook\ntypes\n", exp: "missing    types:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := testOptions(t)
			if tc.packages != "" {
				require.NoError(t, os.WriteFile(o.packagesPath, []byte(tc.packages), 0o600), "writing the package list must not error")
			}
			base := Baseline{
				"currency.BenchmarkNewCode":                       {Allocs: 2, Bytes: 16},
				"exchanges/orderbook.BenchmarkProcess":            {Allocs: 0, Bytes: 0},
				"exchanges/orderbook.BenchmarkSortAsksDescending": {Allocs: 1, Bytes: 24},
			}
			if tc.edit != nil {
				tc.edit(base)
			}
			require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")
			var out bytes.Buffer
			err := run(strings.NewReader(sampleOutput), &out, o)
			assert.ErrorIs(t, err, errFindings, "the finding should fail the run")
			assert.EqualError(t, err, errFindings.Error()+": 1", "the error should count the one finding")
			assert.Contains(t, out.String(), tc.exp, "the finding should be printed")
		})
	}
}

func TestRunReportsImprovementsWithoutFailing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		entry *Entry
		fails bool
	}{
		{"an improvement in both metrics", &Entry{Allocs: 3, Bytes: 32}, false},
		{"an improvement beside a regression", &Entry{Allocs: 3, Bytes: 8}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := testOptions(t)
			base := Baseline{
				"currency.BenchmarkNewCode":                       tc.entry,
				"exchanges/orderbook.BenchmarkProcess":            {Allocs: 0, Bytes: 0},
				"exchanges/orderbook.BenchmarkSortAsksDescending": {Allocs: 1, Bytes: 24},
			}
			require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")
			var out bytes.Buffer
			err := run(strings.NewReader(sampleOutput), &out, o)
			assert.Contains(t, out.String(), "stale      currency.BenchmarkNewCode: allocs/op improved 3 -> 2", "the improvement should be printed")
			if tc.fails {
				assert.ErrorIs(t, err, errFindings, "the regression should fail the run")
				assert.EqualError(t, err, errFindings.Error()+": 1", "the error should count the regression alone")
				return
			}
			require.NoError(t, err, "an improvement alone must not fail the run")
			assert.Contains(t, out.String(), "3 benchmarks within budget, 1 improved; run 'make bench_update' to lower their budgets",
				"the summary should count the improvement")
		})
	}
}

func TestRunRejectsInvalidSamples(t *testing.T) {
	t.Parallel()
	for _, samples := range []int{0, -1, -7, 2, 6} {
		o := testOptions(t)
		o.samples = samples
		err := run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o)
		assert.ErrorIsf(t, err, errInvalidSamples, "-samples %d should be rejected", samples)
	}
}

func TestRunRejectsWrongSampleCount(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.samples = 5
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	var out bytes.Buffer
	err := run(strings.NewReader(sampleOutput), &out, o)
	assert.ErrorIs(t, err, errSampleCount, "a run with the wrong number of samples should not be compared against the baseline")
	assert.Contains(t, out.String(), "currency.BenchmarkNewCode reported B/op=3 allocs/op=3, expected 5",
		"the offending benchmark should be named with its counts")
}

func TestRunRejectsOneShortBenchmark(t *testing.T) {
	t.Parallel()
	header := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n"
	complete := "BenchmarkA\n" + strings.Repeat("BenchmarkA-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 7)
	skipped := "BenchmarkC\n--- SKIP: BenchmarkC\n"
	// Only the last benchmark is short of samples
	short := "BenchmarkB\n" + strings.Repeat("BenchmarkB-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 5)
	done := "PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t1.2s\n"
	for _, tc := range []struct {
		name string
		in   string
		base Baseline
	}{
		{
			name: "after a complete benchmark",
			in:   header + complete + short,
			base: Baseline{"currency.BenchmarkA": {Allocs: 2, Bytes: 16}, "currency.BenchmarkB": {Allocs: 2, Bytes: 16}},
		},
		{
			name: "after an ignored skip",
			in:   header + skipped + short,
			base: Baseline{"currency.BenchmarkB": {Allocs: 2, Bytes: 16}, "currency.BenchmarkC": {Ignore: true, Reason: "needs a fixture"}},
		},
		{name: "alone", in: header + short, base: Baseline{"currency.BenchmarkB": {Allocs: 2, Bytes: 16}}},
	} {
		o := testOptions(t)
		require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
		o.samples = 7
		require.NoError(t, tc.base.Save(o.baselinePath), "seeding the baseline must not error")
		var out bytes.Buffer
		assert.ErrorIsf(t, run(strings.NewReader(tc.in+done), &out, o), errSampleCount, "a short benchmark %s should fail the run", tc.name)
		assert.Containsf(t, out.String(), "currency.BenchmarkB reported B/op=5 allocs/op=5, expected 7",
			"a short benchmark %s should be named with its counts", tc.name)
		// make bench_update and make bench_apply prune as they update
		for _, prune := range []bool{false, true} {
			o.update, o.prune = true, prune
			assert.ErrorIsf(t, run(strings.NewReader(tc.in+done), &bytes.Buffer{}, o), errSampleCount,
				"an update with prune %t should refuse a short benchmark %s", prune, tc.name)
		}
		after, err := LoadBaseline(o.baselinePath, false)
		require.NoErrorf(t, err, "reloading the baseline must not error after a short benchmark %s", tc.name)
		assert.Equalf(t, tc.base, after, "a refused update should leave the baseline untouched after a short benchmark %s", tc.name)
	}
}

func TestRunChecksSampleCountOfOne(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.samples = 1
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	var out bytes.Buffer
	err := run(strings.NewReader(sampleOutput), &out, o)
	assert.ErrorIs(t, err, errSampleCount, "-samples 1 is odd, so its count should be checked like any other")
	assert.Contains(t, out.String(), "currency.BenchmarkNewCode reported B/op=3 allocs/op=3, expected 1",
		"the benchmark reporting more samples should be named with its counts")
}

func TestRunRejectsArguments(t *testing.T) {
	t.Parallel()
	for _, o := range []*options{testOptions(t), {list: true}} {
		o.args = []string{"bench-output.txt", "-samples", "7"}
		err := run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o)
		assert.ErrorIs(t, err, errUnexpectedArgument, "an argument should be refused rather than ignored for stdin")
		assert.ErrorContains(t, err, "bench-output.txt", "the error should name the argument")
	}
}

func TestRunRejectsPruneWithoutUpdate(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.prune = true
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errPruneWithoutUpdate,
		"-prune should not be silently ignored in check mode")
}

func TestRunRejectsUnlistedOutput(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	err := run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errUnlistedOutput, "output from a package the list does not benchmark should be rejected")
	assert.ErrorContains(t, err, "exchanges/orderbook", "the error should name the package")

	skipOnly := sampleOutput + "pkg: github.com/thrasher-corp/gocryptotrader/types\nBenchmarkNumber\n--- SKIP: BenchmarkNumber\n"
	err = run(strings.NewReader(skipOnly), &bytes.Buffer{}, testOptions(t))
	assert.ErrorIs(t, err, errUnlistedOutput, "a skip from a package the list does not benchmark should be rejected too")
	assert.ErrorContains(t, err, "types", "the error should name the package that only skipped")

	// A header printed mid-package files the skip record after it under the printed package
	printed := strings.Replace(sampleOutput, "PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency",
		"BenchmarkNew\npkg: github.com/thrasher-corp/gocryptotrader/types\nprogress: --- SKIP: BenchmarkNew\n"+
			"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency", 1)
	err = run(strings.NewReader(printed), &bytes.Buffer{}, testOptions(t))
	assert.ErrorIs(t, err, errUnlistedOutput, "a printed header naming a package the list does not benchmark should be rejected")
}

func TestRunPruneRequiresCompleteRun(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update, o.prune = true, true
	base := Baseline{
		"currency.BenchmarkNewCode":            {Allocs: 2, Bytes: 16},
		"exchanges/orderbook.BenchmarkProcess": {Allocs: 0, Bytes: 0},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	// What `make bench_update BENCH_PKGS=./currency/` produces: one package of two
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkNewCode\n" +
		strings.Repeat("BenchmarkNewCode-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3)
	err := run(strings.NewReader(in), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errIncompleteRun, "pruning on a run that skipped a listed package should be refused")

	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")

	o.prune = false
	require.NoError(t, run(strings.NewReader(in), &bytes.Buffer{}, o), "the same partial run must update without -prune")
	after, err = LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Contains(t, after, "exchanges/orderbook.BenchmarkProcess", "an update without -prune should keep what it did not measure")
}

func TestRunPruneRequiresFinishedPackages(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	o.update, o.prune = true, true
	base := Baseline{
		"currency.BenchmarkA": {Allocs: 2, Bytes: 16},
		"currency.BenchmarkB": {Allocs: 2, Bytes: 16},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	// Cut off after BenchmarkA, so BenchmarkB never ran
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkA\n" +
		strings.Repeat("BenchmarkA-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3)
	err := run(strings.NewReader(in), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errIncompleteRun, "pruning on output that stopped before the ok line should be refused")
	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")

	// Run to its ok line, BenchmarkB really is gone
	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(in+"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t1.2s\n"), &out, o),
		"a finished run must prune")
	assert.Contains(t, out.String(), "pruned currency.BenchmarkB", "the benchmark the finished run did not report should be pruned")
}

func TestRunPruneDeletesOrphans(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update, o.prune = true, true
	base := Baseline{
		"currency.BenchmarkNewCode":                     {Allocs: 2, Bytes: 16},
		"exchange/websocket/buffer.BenchmarkBufferPerf": {Allocs: 1, Bytes: 8},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o), "a complete run must prune")
	assert.Contains(t, out.String(), "pruned exchange/websocket/buffer.BenchmarkBufferPerf", "the orphan should be pruned")

	o.update, o.prune = false, false
	require.NoError(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "the check after pruning must be clean")
}

func TestRunPruneNeedsVerboseOutput(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update, o.prune = true, true
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}, "currency.BenchmarkGone": {Allocs: 1, Bytes: 8}}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	quiet := strings.NewReplacer("BenchmarkProcess\n", "", "BenchmarkSortAsksDescending\n", "", "BenchmarkNewCode\n", "").Replace(sampleOutput)
	assert.ErrorIs(t, run(strings.NewReader(quiet), &bytes.Buffer{}, o), errPruneNeedsVerbose,
		"pruning on output without -v should be refused")
	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")

	// Output concatenated from a -v run and one without
	mixed := strings.Replace(sampleOutput, "BenchmarkProcess\n", "", 1)
	mixed = strings.Replace(mixed, "BenchmarkSortAsksDescending\n", "", 1)
	err = run(strings.NewReader(mixed), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errPruneNeedsVerbose, "pruning should be refused while any package's output lacks -v")
	assert.ErrorContains(t, err, "exchanges/orderbook", "the error should name the package without -v")
	assert.NotContains(t, err.Error(), "currency", "the package with -v output should not be named")

	o.prune = false
	require.NoError(t, run(strings.NewReader(quiet), &bytes.Buffer{}, o), "an update that deletes nothing must accept output without -v")
}

func TestQuietPackages(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"common", "types"}, quietPackages(map[string]bool{"currency": true}, []string{"common", "currency", "types"}),
		"quietPackages should list the packages without -v output, in list order")
}

func TestRunKeepsSkippedBudgets(t *testing.T) {
	t.Parallel()
	header := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n"
	alive := "BenchmarkAlive\n" + strings.Repeat("BenchmarkAlive-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3)
	done := "PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t1.2s\n"
	full := header + alive + "BenchmarkConditional\n" +
		strings.Repeat("BenchmarkConditional-4   \t 100\t 10 ns/op\t 8 B/op\t 1 allocs/op\n", 3) + done
	for _, tc := range []struct{ name, skip string }{
		{name: "a skip record", skip: "BenchmarkConditional\n    code_test.go:8: fixture unavailable\n--- SKIP: BenchmarkConditional\n"},
		{name: "a skip record output ran into", skip: "BenchmarkConditional\npreparing fixture: --- SKIP: BenchmarkConditional\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := testOptions(t)
			require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
			skip := header + alive + tc.skip + done

			o.update, o.prune = true, true
			require.NoError(t, run(strings.NewReader(full), &bytes.Buffer{}, o), "seeding the baseline must not error")

			var out bytes.Buffer
			require.NoError(t, run(strings.NewReader(skip), &out, o), "updating on a run with a skip must not error")
			assert.Contains(t, out.String(), "currency.BenchmarkConditional skipped in this run; keeping it", "the update should say why it kept the budget")
			after, err := LoadBaseline(o.baselinePath, false)
			require.NoError(t, err, "reloading the baseline must not error")
			assert.Contains(t, after, "currency.BenchmarkConditional", "a prune should keep the budget of a benchmark that skipped")

			o.update, o.prune = false, false
			out.Reset()
			assert.ErrorIs(t, run(strings.NewReader(skip), &out, o), errFindings, "a check on a run with a skip should fail")
			assert.Contains(t, out.String(), "skipped    currency.BenchmarkConditional: benchmark skipped", "the skip should be reported as one")
			assert.NotContains(t, out.String(), "renamed or deleted", "a skipped benchmark should not be called deleted")
		})
	}
}

func TestRunPruneRefusesSkipRecordBudgets(t *testing.T) {
	t.Parallel()
	header := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkCase\nBenchmarkCase/small\n" +
		strings.Repeat("BenchmarkCase/small-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3)
	done := "PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t1.2s\n"
	for _, tc := range []struct{ name, in, key string }{
		{"a sub-benchmark behind a hidden announcement", "opening fixture: BenchmarkCase/large\n        --- SKIP: BenchmarkCase/large\n", "currency.BenchmarkCase/large"},
		{"a parent that skips after a sub-benchmark measured", "    fixture.go:12: --- SKIP: BenchmarkCase\n", "currency.BenchmarkCase/large"},
		{"a benchmark behind a hidden announcement", "fixture ready: BenchmarkHidden\nprogress: --- SKIP: BenchmarkHidden\n", "currency.BenchmarkHidden"},
		{"a benchmark after a printed ok line", "ok  github.com/thrasher-corp/gocryptotrader/currency\n" +
			"fixture ready: BenchmarkHidden\n    fixture.go:12: --- SKIP: BenchmarkHidden\n", "currency.BenchmarkHidden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := testOptions(t)
			require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
			base := Baseline{"currency.BenchmarkCase/small": {Allocs: 2, Bytes: 16}, tc.key: {Allocs: 2, Bytes: 16}}
			require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

			o.update, o.prune = true, true
			err := run(strings.NewReader(header+tc.in+done), &bytes.Buffer{}, o)
			assert.ErrorIs(t, err, errSkipRecordBudgets, "pruning a budget a skip record names should be refused")
			assert.ErrorContains(t, err, tc.key, "the error should name the budget")
			after, err := LoadBaseline(o.baselinePath, false)
			require.NoError(t, err, "reloading the baseline must not error")
			assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")
		})
	}

	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	base := Baseline{
		"currency.BenchmarkCase/small": {Allocs: 2, Bytes: 16},
		"currency.BenchmarkCase/large": {Allocs: 2, Bytes: 16},
		"currency.BenchmarkGone":       {Allocs: 1, Bytes: 8},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")
	o.update, o.prune = true, true
	var out bytes.Buffer
	in := header + "    code_test.go:9: --- SKIP: BenchmarkCase/small\n" +
		"BenchmarkCase/large\n    code_test.go:10: retrying --- SKIP: BenchmarkCase/large\n--- SKIP: BenchmarkCase/large\n" + done
	require.NoError(t, run(strings.NewReader(in), &out, o), "a prune whose deletions no skip record names must not error")
	assert.Contains(t, out.String(), "pruned currency.BenchmarkGone", "a budget no skip record names should still be pruned")
	assert.Contains(t, out.String(), "currency.BenchmarkCase/large skipped in this run; keeping it", "a budget a reported skip covers should be kept")
}

func TestRunPruneRefusesSkipRecordBudgetsAfterASkip(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	base := Baseline{
		"currency.BenchmarkAlpha":      {Allocs: 1, Bytes: 8},
		"currency.BenchmarkCase/small": {Allocs: 2, Bytes: 16},
		"currency.BenchmarkHidden":     {Allocs: 2, Bytes: 16},
		"currency.BenchmarkLater":      {Allocs: 2, Bytes: 16},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkAlpha\n--- SKIP: BenchmarkAlpha\n" +
		"BenchmarkCase\nBenchmarkCase/small\n" +
		strings.Repeat("BenchmarkCase/small-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3) +
		"fixture ready: BenchmarkHidden\nprogress: --- SKIP: BenchmarkHidden\n" +
		"fixture ready: BenchmarkLater\nprogress: --- SKIP: BenchmarkLater\n" +
		"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t1.2s\n"
	o.update, o.prune = true, true
	err := run(strings.NewReader(in), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errSkipRecordBudgets, "a skip covering an earlier budget should not exempt a later budget a skip record names")
	assert.ErrorContains(t, err, "currency.BenchmarkHidden, currency.BenchmarkLater", "the error should name every such budget")
}

func TestRunPruneIgnoresLoggedOkLine(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	o.update, o.prune = true, true
	base := Baseline{"currency.BenchmarkA": {Allocs: 2, Bytes: 16}, "currency.BenchmarkB": {Allocs: 2, Bytes: 16}}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	// Cut off before BenchmarkB, with only a logged ok line
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\nBenchmarkA\n" +
		strings.Repeat("BenchmarkA-4   \t 100\t 10 ns/op\t 16 B/op\t 2 allocs/op\n", 3) +
		"--- BENCH: BenchmarkA-4\n    code_test.go:4: captured subprocess output:\n" +
		"        ok github.com/thrasher-corp/gocryptotrader/currency 0.001s\n"
	assert.ErrorIs(t, run(strings.NewReader(in), &bytes.Buffer{}, o), errIncompleteRun,
		"a logged ok line should not vouch for a package whose output stopped short")
	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "a refused prune should leave the baseline untouched")
}

func TestRunUpdateRefusesOtherPlatforms(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true
	for _, in := range []string{
		strings.ReplaceAll(sampleOutput, "goarch: amd64", "goarch: arm64"),
		strings.NewReplacer("goos: linux\n", "", "goarch: amd64\n", "").Replace(sampleOutput),
	} {
		err := run(strings.NewReader(in), &bytes.Buffer{}, o)
		assert.ErrorIs(t, err, errWrongPlatform, "budgets should only be recorded from the canonical platform")
		_, statErr := os.Stat(o.baselinePath)
		assert.True(t, os.IsNotExist(statErr), "a refused update should not write a baseline")
	}
}

func TestRunCheckNotesOtherPlatforms(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true
	require.NoError(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "seeding the baseline must not error")

	o.update = false
	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(strings.ReplaceAll(sampleOutput, "goos: linux", "goos: darwin")), &out, o),
		"a check from another platform must still run")
	assert.Contains(t, out.String(), `this output is from "darwin/amd64"`, "the check should say the platform differs")
}

func TestRunReportsFailedRun(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	err := run(strings.NewReader(sampleOutput+"--- FAIL: BenchmarkNewCode-8\n"), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errRunFailed, "output from a failed run should be rejected")
}

func TestRunReportsRunsThatOnlySkipped(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\n"), 0o600), "writing the package list must not error")
	base := Baseline{"currency.BenchmarkA": {Allocs: 1, Bytes: 8}}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")
	in := "goos: linux\ngoarch: amd64\npkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkA\n    code_test.go:3: skip A\n--- SKIP: BenchmarkA\n" +
		"BenchmarkB\nBenchmarkB/child\n    code_test.go:4: skip B\n--- SKIP: BenchmarkB/child\n" +
		"PASS\nok  \tgithub.com/thrasher-corp/gocryptotrader/currency\t0.1s\n"

	var out bytes.Buffer
	err := run(strings.NewReader(in), &out, o)
	assert.ErrorIs(t, err, errFindings, "a run in which everything skipped should report the skips")
	assert.EqualError(t, err, errFindings.Error()+": 2", "the error should count both skips")
	assert.Contains(t, out.String(), "skipped    currency.BenchmarkA:", "the budgeted skip should be reported")
	assert.Contains(t, out.String(), "skipped    currency.BenchmarkB/child:", "the unbudgeted skip should be reported")

	base["currency.BenchmarkB/child"] = &Entry{Ignore: true, Reason: "needs credentials"}
	base["currency.BenchmarkA"].Ignore, base["currency.BenchmarkA"].Reason = true, "needs credentials"
	require.NoError(t, base.Save(o.baselinePath), "saving the ignored entries must not error")
	require.NoError(t, run(strings.NewReader(in), &bytes.Buffer{}, o), "skips whose entries are ignored must pass")

	o.update, o.prune = true, true
	require.NoError(t, run(strings.NewReader(in), &bytes.Buffer{}, o), "an update on a run that only skipped must not error")
	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Equal(t, base, after, "an update should keep every entry a skip accounts for")
}

func TestRunNoResults(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	assert.ErrorIs(t, run(strings.NewReader("PASS\n"), &bytes.Buffer{}, o), errNoResults, "empty input should return errNoResults")
}

func TestRunRejectsInvalidBaseline(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16, AllocsTolerance: 0.5}}.Save(o.baselinePath),
		"seeding the baseline must not error")
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errToleranceNeedsReason,
		"a baseline that fails validation should not be compared against")
}

func TestRunListReportsDiscoveryErrors(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	writeFiles(t, o.root, map[string]string{"currency/x_test.go": "package currency\n\nfunc Benchmark(b *testing.B {\n"})
	err := run(strings.NewReader(""), &bytes.Buffer{}, &options{packagesPath: o.packagesPath, root: o.root, list: true})
	assert.ErrorContains(t, err, "error parsing", "-list should fail when discovery cannot read a package")
}

func TestRunReportsUnreadableBaseline(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	require.NoError(t, os.WriteFile(o.baselinePath, []byte("{nope"), 0o600), "writing the fixture must not error")
	assert.ErrorContains(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "error parsing baseline",
		"an unparsable baseline should fail the run")
}

func TestRunUpdateReportsSaveErrors(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.update = true
	o.baselinePath = filepath.Join(t.TempDir(), "missing", "baseline.json")
	assert.ErrorContains(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "error creating temporary baseline",
		"a baseline that cannot be written should fail the update")
}

func TestRunMissingPackageList(t *testing.T) {
	t.Parallel()
	o := testOptions(t)
	o.packagesPath = filepath.Join(t.TempDir(), "absent.txt")
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errNoPackageList,
		"run should fail without a package list")
}

func TestUnfinishedPackages(t *testing.T) {
	t.Parallel()
	results := resultsOf(result("currency", "BenchmarkNewCode", 2, 16), result("types", "BenchmarkNumber", 1, 8))
	finished := map[string]bool{"currency": true, "common": true}
	assert.Equal(t, []string{"common", "types"}, unfinishedPackages(finished, results, nil, []string{"common", "currency", "types"}),
		"unfinishedPackages should list packages without results and packages without an ok line, in list order")
	assert.Equal(t, []string{"types"}, unfinishedPackages(finished, results, map[string]string{"common.BenchmarkCounter": "common"},
		[]string{"common", "currency", "types"}), "a finished package whose benchmarks skipped should not count as unfinished")
}

func TestSilentPackages(t *testing.T) {
	t.Parallel()
	got := silentPackages(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), nil, []string{"common", "currency", "types"})
	assert.Equal(t, []string{"common", "types"}, got, "silentPackages should list the packages without results, in list order")
	got = silentPackages(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), map[string]string{"types.BenchmarkNumber": "types"},
		[]string{"common", "currency", "types"})
	assert.Equal(t, []string{"common"}, got, "a package that reported a skip should not count as silent")
}

func TestSkipRecordBudgets(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkMeasured": {}, "currency.BenchmarkNamed": {}, "currency.BenchmarkNamed/sub": {},
		"currency.BenchmarkNamedX": {}, "currency.BenchmarkSkipped": {}, "currency.BenchmarkSkipped/sub": {}, "currency.BenchmarkGone": {},
	}
	results := resultsOf(result("currency", "BenchmarkMeasured", 1, 8))
	skipped := map[string]string{"currency.BenchmarkSkipped": "currency"}
	records := map[string]string{"currency.BenchmarkMeasured": "currency", "currency.BenchmarkNamed": "currency", "currency.BenchmarkSkipped": "currency"}
	assert.Equal(t, []string{"currency.BenchmarkNamed", "currency.BenchmarkNamed/sub"}, skipRecordBudgets(base, results, skipped, records),
		"skipRecordBudgets should name, in key order, the unmeasured and unskipped budgets that skip record text covers")
	assert.Empty(t, skipRecordBudgets(base, results, skipped, nil), "skipRecordBudgets should name nothing without skip record text")
}
