package main

import (
	"bytes"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writePackages creates the package list that run() now requires, covering the packages used by the
// sample output in these tests
func writePackages(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "packages.txt")
	body := "currency\nexchanges/orderbook\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600), "writing the package list must not error")
	return path
}

func TestLoadBaselineMissingFile(t *testing.T) {
	t.Parallel()
	b, err := LoadBaseline(filepath.Join(t.TempDir(), "absent.json"), true)
	require.NoError(t, err, "LoadBaseline must not error when the file is absent")
	assert.Empty(t, b, "an absent baseline should load as empty so the first -update can seed it")
}

func TestLoadBaselineInvalid(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	require.NoError(t, os.WriteFile(path, []byte("{nope"), 0o600), "writing the fixture must not error")
	_, err := LoadBaseline(path, false)
	assert.ErrorContains(t, err, "error parsing baseline", "an unparsable baseline should report the path")
}

func TestBaselineSaveAndLoad(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	in := Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16, NSHint: 1050, Reason: "interned lookup"}}
	require.NoError(t, in.Save(path), "Save must not error")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading the saved baseline must not error")
	assert.True(t, bytes.HasSuffix(data, []byte("\n")), "the baseline should end with a newline")

	out, err := LoadBaseline(path, false)
	require.NoError(t, err, "LoadBaseline must not error")
	assert.Equal(t, in, out, "a saved baseline should round-trip")
}

func TestBaselineUpdate(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode":         {Allocs: 9, Bytes: 99, Reason: "keep me"},
		"currency.BenchmarkGone":            {Allocs: 1, Bytes: 8},
		"exchanges/orderbook.BenchmarkSort": {Allocs: 1, Bytes: 8},
	}
	_ = base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), true, nil)

	require.Contains(t, base, "currency.BenchmarkNewCode", "the updated benchmark must remain")
	entry := base["currency.BenchmarkNewCode"]
	assert.Equal(t, uint64(2), entry.Allocs, "allocs should be taken from the run")
	assert.Equal(t, uint64(16), entry.Bytes, "bytes should be taken from the run")
	assert.Equal(t, "keep me", entry.Reason, "a hand-written reason should be preserved across updates")

	assert.NotContains(t, base, "currency.BenchmarkGone", "a benchmark absent from a package that ran should be dropped")
	assert.Contains(t, base, "exchanges/orderbook.BenchmarkSort", "a package that did not run should be left untouched")
}

func TestBaselineUpdatePrunesVanishedPackage(t *testing.T) {
	t.Parallel()
	// A package whose last benchmark was deleted reports nothing at all, so it never appears in the
	// results. Compare still fails on its orphaned entry, so if pruning cannot reach it, `make
	// bench` demands a `make bench_update` that changes nothing and the gate stays red for good.
	base := Baseline{
		"common.BenchmarkCounter":   {Allocs: 0, Bytes: 0},
		"currency.BenchmarkNewCode": {Allocs: 1, Bytes: 8},
	}
	got := resultsOf(result("currency", "BenchmarkNewCode", 1, 8))
	configured := map[string]bool{"common": true, "currency": true}

	pruned := base.Update(got, true, configured)
	assert.Equal(t, []string{"common.BenchmarkCounter"}, pruned, "the vanished package's entry should be pruned")
	assert.NotContains(t, base, "common.BenchmarkCounter", "and removed from the baseline")
	assert.Empty(t, Compare(base, got, configured, nil, 0.01), "the gate should then be clean")

	// A package outside the configured scope is still left alone, so updating from a run that only
	// covered some packages does not delete the rest.
	outOfScope := Baseline{"exchanges/okx.BenchmarkMessageID": {Allocs: 2, Bytes: 48}}
	assert.Empty(t, outOfScope.Update(got, true, configured), "an unconfigured package should not be pruned")
	assert.Contains(t, outOfScope, "exchanges/okx.BenchmarkMessageID", "and should keep its entry")
}

func TestBaselineUpdateSeedsNewEntries(t *testing.T) {
	t.Parallel()
	base := Baseline{}
	_ = base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), true, nil)
	require.Contains(t, base, "currency.BenchmarkNewCode", "an unseen benchmark must be added")
	assert.Equal(t, float64(100), base["currency.BenchmarkNewCode"].NSHint, "the advisory ns hint should be recorded")
}

func TestBaselineValidate(t *testing.T) {
	t.Parallel()
	assert.NoError(t, Baseline{"a.B": {Allocs: 1}}.Validate(), "an entry without tolerances should not need a reason")
	assert.NoError(t, Baseline{"a.B": {BytesTolerance: 0.1, Reason: "noisy"}}.Validate(),
		"a justified tolerance should validate")

	assert.ErrorIs(t, Baseline{"a.B": {BytesTolerance: 0.1}}.Validate(), errToleranceNeedsReason,
		"a bytes tolerance without a reason should be rejected")
	assert.ErrorIs(t, Baseline{"a.B": {AllocsTolerance: 0.1}}.Validate(), errToleranceNeedsReason,
		"an allocs tolerance without a reason should be rejected")
	assert.ErrorIs(t, Baseline{"a.B": nil}.Validate(), errNullEntry, "a null entry should be rejected")

	// A blank reason reviews as an empty diff while buying the same exemption as a justification
	assert.ErrorIs(t, Baseline{"a.B": {Ignore: true, Reason: "  \t"}}.Validate(), errToleranceNeedsReason,
		"a whitespace-only reason should not satisfy the justification requirement")
}

func TestBaselineUpdatePreservesTolerances(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 9, BytesTolerance: 0.15, Reason: "noisy"}}
	_ = base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), true, nil)
	entry := base["currency.BenchmarkNewCode"]
	assert.Equal(t, 0.15, entry.BytesTolerance, "a tolerance override should survive an update")
	assert.Equal(t, uint64(2), entry.Allocs, "the measured value should still be refreshed")
}

func TestBaselineUpdateSkipsIgnored(t *testing.T) {
	t.Parallel()
	base := Baseline{"exchanges/alert.BenchmarkWait": {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"}}
	_ = base.Update(resultsOf(result("exchanges/alert", "BenchmarkWait", 99, 9999)), true, nil)
	entry := base["exchanges/alert.BenchmarkWait"]
	assert.Equal(t, uint64(4), entry.Allocs, "an ignored benchmark should keep its recorded allocs")
	assert.Equal(t, uint64(778), entry.Bytes, "an ignored benchmark should keep its recorded bytes")
	assert.True(t, entry.Ignore, "the ignore flag should survive an update")
}

func TestRoundNS(t *testing.T) {
	t.Parallel()
	assert.Equal(t, float64(3556), roundNS(3556.482), "values over 1000 should drop the fraction")
	assert.Equal(t, 210.9, roundNS(210.94), "values over 10 should keep one decimal")
	assert.Equal(t, 1.23, roundNS(1.2345), "small values should keep two decimals")
}

func TestLoadPackages(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packages.txt")
	body := "# comment\n\ncurrency    gated\nconfig      # not hermetic\n" +
		"github.com/thrasher-corp/gocryptotrader/types  gated\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600), "writing the fixture must not error")

	pkgs, err := LoadPackages(path)
	require.NoError(t, err, "LoadPackages must not error")
	assert.Equal(t, []string{"currency", "config", "types"}, pkgs.List,
		"packages should be returned in file order with the module prefix stripped")
	assert.Equal(t, map[string]bool{"currency": true, "types": true}, pkgs.Gated,
		"only packages carrying the gated marker should be gated")
}

// TestCanonicalPackage pins every spelling go test resolves to a package while the gated and
// configured lookups, which only ever see the pkg: header, would miss it and gate nothing.
func TestCanonicalPackage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		entry string
		want  string
		err   error
	}{
		{entry: "currency", want: "currency"},
		{entry: "exchanges/order/", want: "exchanges/order"},
		{entry: "exchanges/order//", want: "exchanges/order"},
		{entry: "./currency", want: "currency"},
		{entry: "././currency", want: "currency"},
		{entry: "currency/./", want: "currency"},
		{entry: "exchange//websocket/buffer", want: "exchange/websocket/buffer"},
		{entry: "currency/../types", want: "types"},
		{entry: modulePrefix + "types", want: "types"},
		{entry: modulePrefix + "types/", want: "types"},
		{entry: modulePrefix + "/currency", want: "currency"},
		{entry: "currency/forexprovider/exchangeratesapi.io", want: "currency/forexprovider/exchangeratesapi.io"},
		{entry: strings.TrimSuffix(modulePrefix, "/"), err: errBadPackagePath},
		// Cleaning erases the module boundary, so these must be caught before the prefix comes off:
		// the second cleans to a real package the entry never named
		{entry: modulePrefix + "../outside", err: errBadPackagePath},
		{entry: modulePrefix + "../../../currency", err: errBadPackagePath},
		// -list is interpolated into go test unquoted, so the shell rewrites these into a real
		// package after the gate has already filed them under the spelling in the file
		{entry: "'currency'", err: errBadPackagePath},
		{entry: `"currency"`, err: errBadPackagePath},
		{entry: "typ[e]s", err: errBadPackagePath},
		{entry: "typ*s", err: errBadPackagePath},
		{entry: "$CURRENCY", err: errBadPackagePath},
		{entry: "currency;rm", err: errBadPackagePath},
		{entry: `currency\`, err: errBadPackagePath},
		{entry: "currency/...", err: errBadPackagePath},
		{entry: "./...", err: errBadPackagePath},
		{entry: "./", err: errBadPackagePath},
		{entry: ".", err: errBadPackagePath},
		{entry: "../outside", err: errBadPackagePath},
		{entry: "/abs/path", err: errBadPackagePath},
	} {
		t.Run(tc.entry, func(t *testing.T) {
			t.Parallel()
			got, err := canonicalPackage(tc.entry)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "an entry go test resolves elsewhere should be refused")
				return
			}
			require.NoError(t, err, "canonicalPackage must not error")
			assert.Equal(t, tc.want, got, "the entry should reduce to the path go test reports")
		})
	}
}

// TestListOutputSurvivesTheShell pins the property the gate actually depends on: make interpolates
// -list into the go test command unquoted, so an accepted entry must mean the same thing after the
// shell has had it. Any entry whose emitted argument the shell would rewrite must be refused, or
// go test measures one package while the gate waits on a name nothing reports.
func TestListOutputSurvivesTheShell(t *testing.T) {
	t.Parallel()
	for _, entry := range []string{
		"currency", "exchanges/order/", "./currency", "currency/./", "exchange//websocket/buffer",
		"currency/forexprovider/exchangeratesapi.io", modulePrefix + "types",
		"'currency'", `"currency"`, "typ[e]s", "typ*s", "$CURRENCY", "types dispatch", "currency;rm",
		"currency`x`", "currency|x", "typ?s", "currency&", "types~x", "~currency", "currency\\",
		"cmd/g++", "foo+bar", "a-b_c.d~e/f",
	} {
		t.Run(entry, func(t *testing.T) {
			t.Parallel()
			pkg, err := canonicalPackage(entry)
			if err != nil {
				return // refused, so it never reaches the shell
			}
			arg := "./" + pkg + "/"
			for _, sh := range []string{"sh", "bash", "dash"} {
				if _, err := exec.LookPath(sh); err != nil {
					continue
				}
				// Interpolated unquoted on purpose: that is what make does, and reproducing it is
				// the only way to check the emitted argument against a real shell rather than
				// against the same assumption the code was written from. Run from the module root
				// so a glob has the real package tree to expand against; from this package's own
				// directory nothing matches and a globbing entry would pass by accident.
				cmd := exec.CommandContext(t.Context(), sh, "-c", "printf %s "+arg)
				cmd.Dir = filepath.Join("..", "..")
				out, shErr := cmd.CombinedOutput()
				require.NoErrorf(t, shErr, "%s must accept the emitted argument", sh)
				assert.Equalf(t, arg, string(out),
					"an accepted entry should reach go test as the exact path the gate recorded, under %s", sh)
			}
		})
	}
}

func TestLoadPackagesCanonicalises(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "packages.txt")
	require.NoError(t, os.WriteFile(path, []byte("exchanges/order/  gated\ncurrency/./  gated\n"), 0o600),
		"writing the fixture must not error")

	pkgs, err := LoadPackages(path)
	require.NoError(t, err, "LoadPackages must not error")
	assert.Equal(t, []string{"exchanges/order", "currency"}, pkgs.List, "entries should be canonicalised")
	assert.Equal(t, map[string]bool{"exchanges/order": true, "currency": true}, pkgs.Gated,
		"a noncanonical entry should still gate the package go test actually runs")

	bad := filepath.Join(dir, "bad.txt")
	require.NoError(t, os.WriteFile(bad, []byte("currency/...  gated\n"), 0o600), "writing the fixture must not error")
	_, err = LoadPackages(bad)
	assert.ErrorIs(t, err, errBadPackagePath, "a wildcard entry should be refused rather than gating one of its packages")
}

func TestLoadPackagesEmpty(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packages.txt")
	require.NoError(t, os.WriteFile(path, []byte("# only comments\n\n"), 0o600), "writing the fixture must not error")
	_, err := LoadPackages(path)
	assert.ErrorIs(t, err, errEmptyPackageList,
		"a list naming no packages should error rather than silently gate nothing")
}

func TestLoadPackagesUnknownMarker(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packages.txt")
	require.NoError(t, os.WriteFile(path, []byte("currency  gatd\n"), 0o600), "writing the fixture must not error")
	_, err := LoadPackages(path)
	assert.ErrorIs(t, err, errUnknownMarker, "a mistyped marker should be rejected rather than silently ignored")
}

func TestLoadPackagesMissingFile(t *testing.T) {
	t.Parallel()
	_, err := LoadPackages(filepath.Join(t.TempDir(), "absent.txt"))
	assert.ErrorIs(t, err, errNoPackageList,
		"an absent package list must error rather than silently gate nothing")
}

func TestLoadBaselineMissingInCheckMode(t *testing.T) {
	t.Parallel()
	_, err := LoadBaseline(filepath.Join(t.TempDir(), "absent.json"), false)
	assert.ErrorIs(t, err, errNoBaseline,
		"an absent baseline must error in check mode; an empty one passes any numbers")
}

func TestRunList(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packages.txt")
	require.NoError(t, os.WriteFile(path, []byte("currency gated\nconfig\n"), 0o600), "writing the fixture must not error")

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(""), &out, &options{packagesPath: path, list: true}), "-list must not error")
	assert.Equal(t, "./currency/ ./config/\n", out.String(), "-list should emit go test package arguments in file order")
}

func TestPackageOf(t *testing.T) {
	t.Parallel()
	known := map[string]bool{
		"exchanges/orderbook":                        true,
		"currency/forexprovider/exchangeratesapi.io": true,
	}
	assert.Equal(t, "exchanges/orderbook", packageOf("exchanges/orderbook.BenchmarkProcess", known),
		"a key should resolve to its reporting package")
	assert.Equal(t, "currency/forexprovider/exchangeratesapi.io",
		packageOf("currency/forexprovider/exchangeratesapi.io.BenchmarkGetRates", known),
		"a package path containing dots should resolve correctly")
	assert.Equal(t, "exchanges/orderbook", packageOf("exchanges/orderbook.BenchmarkSort/case.1", known),
		"a subbenchmark name containing dots should resolve to its package")
	assert.Empty(t, packageOf("currency.BenchmarkNewCode", known), "a package that did not run should not resolve")
	assert.Empty(t, packageOf("nodot", known), "a key without a package prefix should yield no package")
}

func TestBaselineUpdateWithoutPruneKeepsEntries(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode": {Allocs: 9, Bytes: 99},
		"currency.BenchmarkOther":   {Allocs: 1, Bytes: 8},
	}
	_ = base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), false, nil)
	assert.Contains(t, base, "currency.BenchmarkOther",
		"a partial run must not delete entries it simply did not select")
	assert.Equal(t, uint64(2), base["currency.BenchmarkNewCode"].Allocs, "measured entries should still be refreshed")
}

func TestLoadBaselineNullFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	require.NoError(t, os.WriteFile(path, []byte("null"), 0o600), "writing the fixture must not error")

	b, err := LoadBaseline(path, false)
	require.NoError(t, err, "a null baseline must not error")
	require.NotNil(t, b, "a null baseline must load as an empty map, not a nil one")
	b.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), false, nil)
	assert.Contains(t, b, "currency.BenchmarkNewCode", "updating a null baseline should not panic")
}

func TestValidateToleranceRange(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		tol  float64
	}{
		{"negative reports an unchanged benchmark as a regression", -0.01},
		{"one puts the ratchet lower bound at zero", 1},
		{"above one disables the gate", 2},
		{"NaN makes every comparison false", math.NaN()},
	} {
		err := Baseline{"a.B": {BytesTolerance: tc.tol, Reason: "x"}}.Validate()
		assert.ErrorIsf(t, err, errToleranceOutOfRange, "a bytes tolerance that is %s should be rejected", tc.name)
		err = Baseline{"a.B": {AllocsTolerance: tc.tol, Reason: "x"}}.Validate()
		assert.ErrorIsf(t, err, errToleranceOutOfRange, "an allocs tolerance that is %s should be rejected", tc.name)
	}
	assert.NoError(t, Baseline{"a.B": {BytesTolerance: 0.99, Reason: "x"}}.Validate(),
		"a tolerance just inside the range should be accepted")
}

func TestRunUpdateThenCheck(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
		update:         true,
	}

	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o), "seeding the baseline must not error")
	assert.Contains(t, out.String(), "recorded 3 benchmarks", "the update should report what it recorded")

	out.Reset()
	o.update = false
	require.NoError(t, run(strings.NewReader(sampleOutput), &out, o),
		"checking the same output against the seeded baseline must not error")
	assert.Contains(t, out.String(), "3 benchmarks within budget", "a clean check should say so")
}

func TestRunReportsRegression(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
	}
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 0, Bytes: 0}}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode-8   \t 1000000\t      1050 ns/op\t      16 B/op\t       2 allocs/op\n"

	var out bytes.Buffer
	err := run(strings.NewReader(in), &out, o)
	assert.ErrorContains(t, err, "2 findings", "a regression should fail the run")
	assert.Contains(t, out.String(), "regression", "the finding should be printed")

	out.Reset()
	o.warn = true
	assert.NoError(t, run(strings.NewReader(in), &out, o), "warn mode should not fail the run")
	assert.Contains(t, out.String(), "warn mode", "warn mode should be flagged in the output")
}

func TestRunUpdateDoesNotPruneWithoutFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
		update:         true,
	}
	// Every configured package reports one benchmark, which a filtered `-bench BenchmarkOne` run
	// also satisfies. Inferring a complete run from that deleted 51 real entries.
	require.NoError(t, os.WriteFile(o.packagesPath, []byte("currency\nexchanges/orderbook\n"), 0o600),
		"writing the fixture must not error")
	base := Baseline{
		"currency.BenchmarkNewCode":          {Allocs: 1, Bytes: 8},
		"currency.BenchmarkOther":            {Allocs: 2, Bytes: 16},
		"exchanges/orderbook.BenchmarkSort":  {Allocs: 3, Bytes: 24},
		"exchanges/orderbook.BenchmarkOther": {Allocs: 4, Bytes: 32},
	}
	require.NoError(t, base.Save(o.baselinePath), "seeding the baseline must not error")

	in := "pkg: github.com/thrasher-corp/gocryptotrader/currency\n" +
		"BenchmarkNewCode-4   \t 100\t 10 ns/op\t 8 B/op\t 1 allocs/op\n" +
		"pkg: github.com/thrasher-corp/gocryptotrader/exchanges/orderbook\n" +
		"BenchmarkSort-4   \t 100\t 10 ns/op\t 24 B/op\t 3 allocs/op\n"
	require.NoError(t, run(strings.NewReader(in), &bytes.Buffer{}, o), "update must not error")

	after, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Len(t, after, 4, "a filtered run should not delete the entries it did not select")

	o.prune = true
	require.NoError(t, run(strings.NewReader(in), &bytes.Buffer{}, o), "update with -prune must not error")
	after, err = LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "reloading the baseline must not error")
	assert.Len(t, after, 2, "-prune should delete unmatched entries when the caller vouches for the run")
}

func TestRunRejectsBadGlobalTolerance(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: math.NaN(),
	}
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	err := run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o)
	assert.ErrorIs(t, err, errToleranceOutOfRange,
		"a NaN global tolerance would disable every B/op comparison and must be rejected")
}

func TestRunRejectsShortSampleCount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
		samples:        7,
	}
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	var out bytes.Buffer
	err := run(strings.NewReader(sampleOutput), &out, o)
	assert.ErrorIs(t, err, errSampleCount, "a run with too few samples should not be compared against the baseline")
	assert.Contains(t, out.String(), "currency.BenchmarkNewCode reported", "the offending benchmark should be named")
}

func TestRunWarnSkipsShortSampleCounts(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
		samples:        2,
		warn:           true,
		update:         true,
	}
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	require.NoError(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), "warn mode must not fail on a short benchmark")

	base, err := LoadBaseline(o.baselinePath, false)
	require.NoError(t, err, "loading the baseline must not error")
	assert.Contains(t, base, "exchanges/orderbook.BenchmarkProcess", "the complete benchmark should still be recorded")
	assert.NotContains(t, base, "currency.BenchmarkNewCode", "a benchmark with the wrong sample count should not be recorded")
}

func TestRunRejectsNegativeSamples(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
		samples:        -1,
	}
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	assert.ErrorIs(t, run(strings.NewReader(sampleOutput), &bytes.Buffer{}, o), errNegativeSamples,
		"a negative -samples must be rejected")
}

func TestSaveLeavesPreviousBaselineOnFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	require.NoError(t, Baseline{"a.BenchmarkOne": {Allocs: 1, Bytes: 16}}.Save(path), "the first save must not error")
	before, err := os.ReadFile(path)
	require.NoError(t, err, "reading the saved baseline must not error")

	// NaN cannot be encoded, so the marshal fails after the caller has already committed to saving
	err = Baseline{"a.BenchmarkOne": {Allocs: 1, Bytes: 16, BytesTolerance: math.NaN()}}.Save(path)
	require.Error(t, err, "saving an unencodable baseline must error")

	after, err := os.ReadFile(path)
	require.NoError(t, err, "the previous baseline must still be readable")
	assert.Equal(t, before, after, "a failed save should leave the previous baseline untouched")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "reading the baseline directory must not error")
	assert.Len(t, entries, 1, "a failed save should not leave a temporary file behind")
}

func TestRunNoResults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	o := &options{
		baselinePath:   filepath.Join(dir, "baseline.json"),
		packagesPath:   writePackages(t, dir),
		bytesTolerance: 0.01,
	}
	require.NoError(t, Baseline{}.Save(o.baselinePath), "seeding an empty baseline must not error")
	assert.ErrorIs(t, run(strings.NewReader("PASS\n"), &bytes.Buffer{}, o), errNoResults,
		"empty input should return errNoResults")
}
