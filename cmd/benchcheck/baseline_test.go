package main

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBaselineMissingFile(t *testing.T) {
	t.Parallel()
	b, err := LoadBaseline(filepath.Join(t.TempDir(), "absent.json"), true)
	require.NoError(t, err, "LoadBaseline must not error when the file is absent")
	assert.Empty(t, b, "an absent baseline should load as empty so the first -update can seed it")
}

func TestLoadBaselineMissingInCheckMode(t *testing.T) {
	t.Parallel()
	_, err := LoadBaseline(filepath.Join(t.TempDir(), "absent.json"), false)
	assert.ErrorIs(t, err, errNoBaseline,
		"an absent baseline should error in check mode; an empty one passes any numbers")
}

func TestLoadBaselineUnreadable(t *testing.T) {
	t.Parallel()
	_, err := LoadBaseline(t.TempDir(), false)
	assert.ErrorContains(t, err, "error reading baseline", "a baseline that cannot be read should not be mistaken for an absent one")
}

func TestLoadBaselineInvalid(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	require.NoError(t, os.WriteFile(path, []byte("{nope"), 0o600), "writing the fixture must not error")
	_, err := LoadBaseline(path, false)
	assert.ErrorContains(t, err, "error parsing baseline", "an unparsable baseline should report the path")
}

func TestLoadBaselineRejectsDuplicateEntries(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	body := `{"currency.BenchmarkNewCode": {"allocs": 2, "bytes": 16}, "currency.BenchmarkNewCode": {"allocs": 40, "bytes": 4000}}`
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600), "writing the fixture must not error")
	_, err := LoadBaseline(path, false)
	assert.ErrorIs(t, err, errDuplicateName, "a second entry for a benchmark should be rejected rather than replace the first")
}

func TestDuplicateName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		data string
		exp  string
	}{
		{name: "distinct keys", data: `{"a.B": {"allocs": 1}, "a.C": {"allocs": 1}}`},
		{name: "fields repeated across entries", data: `{"a.B": {"allocs": 1, "bytes": 2}, "a.C": {"allocs": 1, "bytes": 2}}`},
		{name: "keys differing only in case", data: `{"a.B": {}, "a.b": {}}`},
		{name: "braces and quotes inside a string", data: `{"a.B": {"reason": "a {brace} and a \"quote\""}, "a.C": {}}`},
		{name: "a repeated key", data: `{"a.B": {}, "a.C": {}, "a.B": {}}`, exp: "a.B"},
		{name: "a repeat spelled with an escape", data: `{"a.B": {}, "a\u002eB": {}}`, exp: "a.B"},
		{name: "a repeated field", data: `{"a.B": {"allocs": 2, "bytes": 16, "allocs": 40}}`, exp: "a.B: allocs"},
		{name: "a field repeated in another case", data: `{"a.B": {"bytes_tolerance": 0.001, "reason": "x", "Bytes_Tolerance": 0.9}}`, exp: "a.B: Bytes_Tolerance"},
		{name: "a field repeated in a later entry only", data: `{"a.B": {"allocs": 1}, "a.C": {"allocs": 1, "ALLOCS": 2}}`, exp: "a.C: ALLOCS"},
		{name: "an unterminated string", data: `{"a.B": {}, "a.B`},
	} {
		got, ok := duplicateName([]byte(tc.data))
		assert.Equalf(t, tc.exp != "", ok, "duplicateName should report whether %s holds a duplicate", tc.name)
		assert.Equalf(t, tc.exp, got, "duplicateName should name the duplicate in %s", tc.name)
	}
}

func TestLoadBaselineNullFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	require.NoError(t, os.WriteFile(path, []byte("null"), 0o600), "writing the fixture must not error")

	b, err := LoadBaseline(path, false)
	require.NoError(t, err, "a null baseline must not error")
	require.NotNil(t, b, "a null baseline must load as an empty map, not a nil one")
	b.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), false)
	assert.Contains(t, b, "currency.BenchmarkNewCode", "updating a null baseline should not panic")
}

func TestBaselineSaveAndLoad(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "baseline.json")
	in := Baseline{
		"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16},
		"config.BenchmarkUpdate":    {Allocs: 9, Bytes: 99, AllocsTolerance: 0.001, BytesTolerance: new(0.0), Reason: "file IO"},
	}
	require.NoError(t, in.Save(path), "Save must not error")

	data, err := os.ReadFile(path)
	require.NoError(t, err, "reading the saved baseline must not error")
	assert.True(t, bytes.HasSuffix(data, []byte("\n")), "the baseline should end with a newline")
	assert.Contains(t, string(data), `"bytes_tolerance": 0`, "an explicit zero tolerance should be written rather than dropped")
	assert.NotContains(t, string(data), `"allocs_tolerance": 0,`, "an unset allocs tolerance should be omitted")

	out, err := LoadBaseline(path, false)
	require.NoError(t, err, "LoadBaseline must not error")
	assert.Equal(t, in, out, "a saved baseline should round-trip, explicit zero tolerance included")
}

func TestSaveLeavesPreviousBaselineOnFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	require.NoError(t, Baseline{"a.BenchmarkOne": {Allocs: 1, Bytes: 16}}.Save(path), "the first save must not error")
	before, err := os.ReadFile(path)
	require.NoError(t, err, "reading the saved baseline must not error")

	// NaN cannot be encoded, so the marshal fails after the caller has already committed to saving
	err = Baseline{"a.BenchmarkOne": {Allocs: 1, Bytes: 16, AllocsTolerance: math.NaN()}}.Save(path)
	require.Error(t, err, "saving an unencodable baseline must error")

	after, err := os.ReadFile(path)
	require.NoError(t, err, "the previous baseline must still be readable")
	assert.Equal(t, before, after, "a failed save should leave the previous baseline untouched")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "reading the baseline directory must not error")
	assert.Len(t, entries, 1, "a failed save should not leave a temporary file behind")
}

func TestSaveReportsFilesystemErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	err := Baseline{}.Save(filepath.Join(dir, "missing", "baseline.json"))
	assert.ErrorContains(t, err, "error creating temporary baseline", "a missing directory should be reported")

	// A non-empty directory in the way cannot be replaced by the rename
	path := filepath.Join(dir, "occupied")
	writeFiles(t, path, map[string]string{"file": "x"})
	err = Baseline{}.Save(path)
	assert.ErrorContains(t, err, "error replacing baseline", "a failed rename should be reported")
}

func TestEntryBytesTolerance(t *testing.T) {
	t.Parallel()
	assert.Equal(t, defaultBytesTolerance, (&Entry{}).bytesTolerance(), "an unset tolerance should fall back to the default")
	assert.Zero(t, (&Entry{BytesTolerance: new(0.0)}).bytesTolerance(), "an explicit zero should not fall back to the default")
	assert.Equal(t, 0.2, (&Entry{BytesTolerance: new(0.2)}).bytesTolerance(), "a set tolerance should be used as is")
}

func TestBaselineValidate(t *testing.T) {
	t.Parallel()
	assert.NoError(t, Baseline{"a.B": {Allocs: 1}}.Validate(), "an entry without overrides should not need a reason")
	assert.NoError(t, Baseline{"a.B": {BytesTolerance: new(0.1), Reason: "noisy"}}.Validate(),
		"a justified tolerance should validate")

	assert.ErrorIs(t, Baseline{"a.B": {BytesTolerance: new(0.1)}}.Validate(), errToleranceNeedsReason,
		"a bytes tolerance without a reason should be rejected")
	assert.ErrorIs(t, Baseline{"a.B": {BytesTolerance: new(0.0)}}.Validate(), errToleranceNeedsReason,
		"an explicit zero bytes tolerance is still an override and should need a reason")
	assert.ErrorIs(t, Baseline{"a.B": {AllocsTolerance: 0.1}}.Validate(), errToleranceNeedsReason,
		"an allocs tolerance without a reason should be rejected")
	assert.ErrorIs(t, Baseline{"a.B": nil}.Validate(), errNullEntry, "a null entry should be rejected")

	// A blank reason reviews as an empty diff while buying the same exemption as a justification
	assert.ErrorIs(t, Baseline{"a.B": {Ignore: true, Reason: "  \t"}}.Validate(), errToleranceNeedsReason,
		"a whitespace-only reason should not satisfy the justification requirement")
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
		err := Baseline{"a.B": {BytesTolerance: new(tc.tol), Reason: "x"}}.Validate()
		assert.ErrorIsf(t, err, errToleranceOutOfRange, "a bytes tolerance that is %s should be rejected", tc.name)
		err = Baseline{"a.B": {AllocsTolerance: tc.tol, Reason: "x"}}.Validate()
		assert.ErrorIsf(t, err, errToleranceOutOfRange, "an allocs tolerance that is %s should be rejected", tc.name)
	}
	assert.NoError(t, Baseline{"a.B": {BytesTolerance: new(0.99), Reason: "x"}}.Validate(),
		"a tolerance just inside the range should be accepted")
}

func TestBaselineUpdateNeverLoosensQuietly(t *testing.T) {
	t.Parallel()
	base := Baseline{
		// allocs improved while B/op rose inside the default tolerance: only allocs moves, or the
		// update would quietly raise the B/op budget to a value no check reported
		"currency.BenchmarkNewCode": {Allocs: 10, Bytes: 1000, Reason: "keep me"},
		// B/op regressed past the default tolerance
		"currency.BenchmarkPairs": {Allocs: 1, Bytes: 100},
		// B/op rose, but not past the default tolerance
		"currency.BenchmarkSteady": {Allocs: 3, Bytes: 500},
		// B/op fell by more than half the default tolerance, but not past it
		"currency.BenchmarkTighter": {Allocs: 3, Bytes: 500},
		// B/op fell by less than half the default tolerance: noise
		"currency.BenchmarkNudged": {Allocs: 3, Bytes: 500},
	}
	got := resultsOf(
		result("currency", "BenchmarkNewCode", 9, 1009),
		result("currency", "BenchmarkPairs", 1, 120),
		result("currency", "BenchmarkSteady", 3, 504),
		result("currency", "BenchmarkTighter", 3, 497),
		result("currency", "BenchmarkNudged", 3, 499),
	)
	changes := base.Update(got, false)

	assert.Equal(t, []string{
		"currency.BenchmarkNewCode allocs/op 10 -> 9",
		"currency.BenchmarkPairs B/op 100 -> 120",
		"currency.BenchmarkTighter B/op 500 -> 497",
	}, changes, "Update should describe exactly the metrics it rewrote")
	assert.Equal(t, Baseline{
		"currency.BenchmarkNewCode": {Allocs: 9, Bytes: 1000, Reason: "keep me"},
		"currency.BenchmarkPairs":   {Allocs: 1, Bytes: 120},
		"currency.BenchmarkSteady":  {Allocs: 3, Bytes: 500},
		"currency.BenchmarkTighter": {Allocs: 3, Bytes: 497},
		"currency.BenchmarkNudged":  {Allocs: 3, Bytes: 500},
	}, base, "a budget should rise only past its tolerance and fall only past half of it, keeping a hand-written reason")
	assert.Empty(t, Compare(base, got, []string{"currency"}), "a check of the same results should then be clean")
}

func TestBaselineUpdateWithAllocsTolerance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		allocs uint64
		exp    uint64
	}{
		{name: "an improvement inside half the tolerance keeps the budget", allocs: 999, exp: 1000},
		{name: "an improvement past half the tolerance lowers the budget", allocs: 994, exp: 994},
		{name: "an increase inside the tolerance keeps the budget", allocs: 1005, exp: 1000},
		{name: "an increase past the tolerance raises the budget", allocs: 1020, exp: 1020},
	} {
		base := Baseline{"config.BenchmarkUpdateConfig": {Allocs: 1000, AllocsTolerance: 0.01, Reason: "file IO"}}
		base.Update(resultsOf(result("config", "BenchmarkUpdateConfig", tc.allocs, 0)), false)
		assert.Equalf(t, tc.exp, base["config.BenchmarkUpdateConfig"].Allocs, "Update should apply the rule: %s", tc.name)
	}
}

func TestBaselineUpdateIsIdempotent(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 10, Bytes: 1000}}
	got := resultsOf(result("currency", "BenchmarkNewCode", 9, 900))
	require.NotEmpty(t, base.Update(got, true), "the first update must change the baseline")
	assert.Empty(t, base.Update(got, true), "updating again from the same results should change nothing")
}

func TestBaselineUpdateSeedsNewEntries(t *testing.T) {
	t.Parallel()
	base := Baseline{}
	changes := base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), true)
	assert.Equal(t, []string{"added currency.BenchmarkNewCode: allocs/op 2, B/op 16"}, changes, "the addition should be described")
	assert.Equal(t, Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16}}, base, "an unseen benchmark should be added")
}

func TestBaselineUpdatePreservesOverrides(t *testing.T) {
	t.Parallel()
	base := Baseline{"currency.BenchmarkNewCode": {Allocs: 9, Bytes: 16, BytesTolerance: new(0.15), Reason: "noisy"}}
	_ = base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), true)
	assert.Equal(t, Baseline{"currency.BenchmarkNewCode": {Allocs: 2, Bytes: 16, BytesTolerance: new(0.15), Reason: "noisy"}}, base,
		"a tolerance override and its reason should survive an update")
}

func TestBaselineUpdateSkipsIgnored(t *testing.T) {
	t.Parallel()
	base := Baseline{"exchanges/alert.BenchmarkWait": {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"}}
	assert.Empty(t, base.Update(resultsOf(result("exchanges/alert", "BenchmarkWait", 99, 9999)), true),
		"an ignored benchmark should not be changed")
	assert.Equal(t, Baseline{"exchanges/alert.BenchmarkWait": {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"}}, base,
		"an ignored benchmark should keep its recorded values")
}

func TestBaselineUpdateWithoutPruneKeepsEntries(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode":                     {Allocs: 9, Bytes: 99},
		"currency.BenchmarkOther":                       {Allocs: 1, Bytes: 8},
		"exchange/websocket/buffer.BenchmarkBufferPerf": {Allocs: 1, Bytes: 8},
	}
	changes := base.Update(resultsOf(result("currency", "BenchmarkNewCode", 2, 16)), false)
	assert.Contains(t, changes, "currency.BenchmarkOther has no result in this run; keeping it (pass -prune to delete)",
		"an entry without a result should be reported as kept")
	assert.Contains(t, base, "currency.BenchmarkOther", "a partial run should not delete entries it simply did not select")
	assert.Contains(t, base, "exchange/websocket/buffer.BenchmarkBufferPerf", "an entry outside the run's packages should be kept too")
	assert.Equal(t, uint64(2), base["currency.BenchmarkNewCode"].Allocs, "measured entries should still be refreshed")
}

func TestBaselineUpdatePrunes(t *testing.T) {
	t.Parallel()
	base := Baseline{
		"currency.BenchmarkNewCode": {Allocs: 1, Bytes: 8},
		"currency.BenchmarkGone":    {Allocs: 1, Bytes: 8},
		// The rename this branch inherited from master: the package left the list, so nothing can
		// measure these again, and neither check nor update could reach them when pruning was
		// scoped to the listed packages
		"exchange/websocket/buffer.BenchmarkBufferPerf": {Allocs: 1, Bytes: 8},
		"exchanges/alert.BenchmarkWait":                 {Allocs: 4, Bytes: 778, Ignore: true, Reason: "non-deterministic"},
	}
	got := resultsOf(result("currency", "BenchmarkNewCode", 1, 8))
	changes := base.Update(got, true)
	assert.Equal(t, []string{
		"pruned currency.BenchmarkGone",
		"pruned exchange/websocket/buffer.BenchmarkBufferPerf",
		"pruned exchanges/alert.BenchmarkWait",
	}, changes, "every entry without a result should be pruned, ignored or orphaned alike")
	assert.Equal(t, Baseline{"currency.BenchmarkNewCode": {Allocs: 1, Bytes: 8}}, base, "only the measured entry should remain")
	assert.Empty(t, Compare(base, got, []string{"currency"}), "the gate should then be clean")
}

func TestUpdatedBudget(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		got, budget uint64
		tolerance   float64
		exp         uint64
	}{
		{got: 100, budget: 100, exp: 100},
		{got: 99, budget: 100, exp: 99},
		{got: 101, budget: 100, exp: 101},
		{got: 1004, budget: 1000, tolerance: 0.01, exp: 1000},
		{got: 1010, budget: 1000, tolerance: 0.01, exp: 1000},
		{got: 1011, budget: 1000, tolerance: 0.01, exp: 1011},
		{got: 996, budget: 1000, tolerance: 0.01, exp: 1000},
		{got: 995, budget: 1000, tolerance: 0.01, exp: 1000},
		{got: 994, budget: 1000, tolerance: 0.01, exp: 994},
		{got: 0, budget: 0, tolerance: 0.01, exp: 0},
	} {
		assert.Equalf(t, tc.exp, updatedBudget(tc.got, tc.budget, tc.tolerance),
			"updatedBudget should record %d for %d against a budget of %d with tolerance %v", tc.exp, tc.got, tc.budget, tc.tolerance)
	}
}

func TestPackageOf(t *testing.T) {
	t.Parallel()
	known := map[string]bool{
		"exchanges/orderbook":                        true,
		"currency/forexprovider/exchangeratesapi.io": true,
		"currency/forexprovider/exchangeratesapi":    true,
	}
	assert.Equal(t, "exchanges/orderbook", packageOf("exchanges/orderbook.BenchmarkProcess", known),
		"a key should resolve to its reporting package")
	assert.Equal(t, "currency/forexprovider/exchangeratesapi.io",
		packageOf("currency/forexprovider/exchangeratesapi.io.BenchmarkGetRates", known),
		"the longest matching package should win over one that only prefixes it")
	assert.Equal(t, "exchanges/orderbook", packageOf("exchanges/orderbook.BenchmarkSort/case.1", known),
		"a subbenchmark name containing dots should resolve to its package")
	assert.Empty(t, packageOf("currency.BenchmarkNewCode", known), "an unknown package should not resolve")
	assert.Empty(t, packageOf("nodot", known), "a key without a package prefix should yield no package")
}

func TestSeenPackages(t *testing.T) {
	t.Parallel()
	got := seenPackages(resultsOf(result("currency", "BenchmarkA", 0, 0), result("currency", "BenchmarkB", 0, 0), result("types", "BenchmarkC", 0, 0)))
	assert.Equal(t, map[string]bool{"currency": true, "types": true}, got, "seenPackages should return each reporting package once")
}

func TestSortedKeys(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"a", "b", "c"}, sortedKeys(map[string]int{"c": 1, "a": 2, "b": 3}), "sortedKeys should sort the keys")
}

func TestConfiguredSet(t *testing.T) {
	t.Parallel()
	assert.Equal(t, map[string]bool{"currency": true, "types": true}, configuredSet([]string{"currency", "types"}),
		"configuredSet should hold every listed package")
}
