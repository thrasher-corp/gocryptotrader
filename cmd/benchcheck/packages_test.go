package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFiles creates each file under dir, along with its parent directories
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700), "creating the fixture directory must not error")
		require.NoError(t, os.WriteFile(path, []byte(body), 0o600), "writing the fixture must not error")
	}
}

func TestLoadPackages(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packages.txt")
	body := "# comment\n\ncurrency\nconfig      # near-deterministic\n" +
		"github.com/thrasher-corp/gocryptotrader/types\n" +
		"exchanges/kraken  excluded  # its TestMain fetches pairs from the live API\n"
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600), "writing the fixture must not error")

	pkgs, err := LoadPackages(path)
	require.NoError(t, err, "LoadPackages must not error")
	exp := &Packages{
		List:     []string{"currency", "config", "types"},
		Excluded: map[string]string{"exchanges/kraken": "its TestMain fetches pairs from the live API"},
	}
	assert.Equal(t, exp, pkgs, "packages should be returned in file order with the module prefix stripped, and exclusions kept with their reason")
}

func TestLoadPackagesRejects(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body string
		err  error
	}{
		{name: "an exclusion without a reason", body: "currency\ntypes excluded\n", err: errExclusionNeedsReason},
		{name: "an exclusion with a blank reason", body: "currency\ntypes excluded #  \n", err: errExclusionNeedsReason},
		{name: "a mistyped marker", body: "currency  exclude # reason\n", err: errUnknownMarker},
		{name: "the retired gated marker", body: "currency  gated\n", err: errUnknownMarker},
		{name: "a marker with extra words", body: "currency excluded now # reason\n", err: errUnknownMarker},
		{name: "a package listed twice", body: "currency\ntypes\n./currency/\n", err: errDuplicatePackage},
		{name: "a package both listed and excluded", body: "currency\n./currency/ excluded # reason\n", err: errDuplicatePackage},
		{name: "a wildcard", body: "currency/...\n", err: errBadPackagePath},
		{name: "only comments", body: "# only comments\n\n", err: errEmptyPackageList},
		{name: "only exclusions", body: "types excluded # reason\n", err: errEmptyPackageList},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "packages.txt")
			require.NoError(t, os.WriteFile(path, []byte(tc.body), 0o600), "writing the fixture must not error")
			_, err := LoadPackages(path)
			assert.ErrorIs(t, err, tc.err, "LoadPackages should reject the list")
		})
	}
}

func TestLoadPackagesMissingFile(t *testing.T) {
	t.Parallel()
	_, err := LoadPackages(filepath.Join(t.TempDir(), "absent.txt"))
	assert.ErrorIs(t, err, errNoPackageList, "an absent package list should error rather than silently gate nothing")
}

func TestLoadPackagesUnreadable(t *testing.T) {
	t.Parallel()
	_, err := LoadPackages(t.TempDir())
	assert.ErrorContains(t, err, "error reading package list", "a package list that cannot be read should be reported")
}

func TestLoadPackagesCanonicalises(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "packages.txt")
	require.NoError(t, os.WriteFile(path, []byte("exchanges/order/\ncurrency/./\n"), 0o600), "writing the fixture must not error")
	pkgs, err := LoadPackages(path)
	require.NoError(t, err, "LoadPackages must not error")
	assert.Equal(t, []string{"exchanges/order", "currency"}, pkgs.List, "entries should be canonicalised")
}

func TestPackagesCovers(t *testing.T) {
	t.Parallel()
	pkgs := &Packages{List: []string{"currency", "types"}, Excluded: map[string]string{"exchanges/kraken": "live API"}}
	require.NoError(t, pkgs.Covers([]string{"currency", "exchanges/kraken", "types"}), "a list matching the declared packages must be accepted")

	err := pkgs.Covers([]string{"currency", "exchanges/kraken", "types", "types/decimal"})
	assert.ErrorIs(t, err, errUnlistedPackage, "a benchmark package the list does not mention should be rejected")
	assert.ErrorContains(t, err, "types/decimal", "the error should name the unlisted package")

	err = pkgs.Covers([]string{"currency", "exchanges/kraken"})
	assert.ErrorIs(t, err, errStalePackage, "a listed package without benchmarks should be rejected")
	assert.ErrorContains(t, err, "types", "the error should name the stale package")

	err = pkgs.Covers([]string{"currency", "types"})
	assert.ErrorIs(t, err, errStalePackage, "an excluded package without benchmarks should be rejected")
	assert.ErrorContains(t, err, "exchanges/kraken", "the error should name the stale exclusion")
}

// TestCanonicalPackage pins every spelling go test resolves to a package while the configured
// lookups, which only ever see the pkg: header, would miss it and gate nothing.
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
		{entry: "exchange//websocket/orderbookmanager", want: "exchange/websocket/orderbookmanager"},
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
		"currency", "exchanges/order/", "./currency", "currency/./", "exchange//websocket/orderbookmanager",
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

func TestImportPathRune(t *testing.T) {
	t.Parallel()
	for _, r := range "azAZ09/.-_~+" {
		assert.Truef(t, importPathRune(r), "%q should be allowed in an import path", r)
	}
	for _, r := range " '\"*?[]$;&|`\\#é" {
		assert.Falsef(t, importPathRune(r), "%q should not be allowed in an import path", r)
	}
}

func TestBenchmarkPackages(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const bench = "package %s\n\nimport \"testing\"\n\nfunc BenchmarkX(b *testing.B) {}\n"
	writeFiles(t, root, map[string]string{
		"go.mod":                              "module github.com/thrasher-corp/gocryptotrader\n",
		"plain/plain.go":                      "package plain\n",
		"plain/plain_test.go":                 strings.ReplaceAll(bench, "%s", "plain"),
		"external/external.go":                "package external\n",
		"external/external_test.go":           strings.ReplaceAll(bench, "%s", "external_test"),
		"testonly/testonly_test.go":           strings.ReplaceAll(bench, "%s", "testonly"),
		"nested/deeper/deeper_test.go":        strings.ReplaceAll(bench, "%s", "deeper"),
		"tests/tests_test.go":                 "package tests\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n",
		"nontest/nontest.go":                  strings.ReplaceAll(bench, "%s", "nontest"),
		"tagged/tagged_test.go":               "//go:build sonic_on\n\n" + strings.ReplaceAll(bench, "%s", "tagged"),
		"windows/bench_windows_test.go":       strings.ReplaceAll(bench, "%s", "windows"),
		"testdata/fixture/fixture_test.go":    strings.ReplaceAll(bench, "%s", "fixture"),
		"vendor/dep/dep_test.go":              strings.ReplaceAll(bench, "%s", "dep"),
		".hidden/hidden_test.go":              strings.ReplaceAll(bench, "%s", "hidden"),
		"_skipped/skipped_test.go":            strings.ReplaceAll(bench, "%s", "skipped"),
		"submodule/go.mod":                    "module example.com/submodule\n",
		"submodule/submodule_test.go":         strings.ReplaceAll(bench, "%s", "submodule"),
		"docs/README.md":                      "no Go here\n",
		"method/method_test.go":               "package method\n\nimport \"testing\"\n\ntype s struct{}\n\nfunc (s) BenchmarkX(b *testing.B) {}\n",
		"lowercase/lowercase_test.go":         "package lowercase\n\nimport \"testing\"\n\nfunc Benchmarkx(b *testing.B) {}\n",
		"renamedimport/renamedimport_test.go": "package renamedimport\n\nimport tt \"testing\"\n\nfunc Benchmark(b *tt.B) {}\n",
		"commented/commented_test.go":         "package commented\n\nimport \"testing\"\n\nfunc /* bench */ BenchmarkX(b *testing.B) {}\n",
	})

	got, err := BenchmarkPackages(root)
	require.NoError(t, err, "BenchmarkPackages must not error")
	assert.Equal(t, []string{"commented", "external", "nested/deeper", "plain", "renamedimport", "testonly"}, got,
		"only packages go test would benchmark in the canonical configuration should be found")
}

func TestBenchmarkPackagesReportsBrokenFiles(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"broken/broken_test.go": "package broken\n\nfunc Benchmark(b *testing.B {\n"})
	_, err := BenchmarkPackages(root)
	assert.ErrorContains(t, err, "error parsing", "an unparsable benchmark file should fail discovery rather than hide its package")
}

func TestBenchmarkPackagesReportsUnloadablePackages(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"mixed/a.go": "package a\n", "mixed/b.go": "package b\n"})
	_, err := BenchmarkPackages(root)
	assert.ErrorContains(t, err, "error loading package", "a directory go test cannot load should fail discovery")

	_, err = BenchmarkPackages(filepath.Join(root, "absent"))
	assert.Error(t, err, "a missing root should fail discovery")
}

func TestBenchmarkPackagesRejectsRootBenchmarks(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"root_test.go": "package root\n\nimport \"testing\"\n\nfunc BenchmarkX(b *testing.B) {}\n"})
	_, err := BenchmarkPackages(root)
	assert.ErrorIs(t, err, errRootBenchmarks, "a benchmark in the module root, which the list cannot name, should be refused")
}

func TestBenchmarkPackagesFindsThisModule(t *testing.T) {
	t.Parallel()
	// The real tree is the case that matters: discovery must succeed on it and find, at the very
	// least, this repository's own long-standing benchmark packages
	got, err := BenchmarkPackages(filepath.Join("..", ".."))
	require.NoError(t, err, "BenchmarkPackages must not error on the repository")
	assert.Subset(t, got, []string{"currency", "exchanges/orderbook", "types"}, "the repository's benchmark packages should be discovered")
}

func TestDeclaresBenchmark(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		"a_test.go": "package a\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n",
		"b_test.go": "package a\n\nimport \"testing\"\n\nfunc BenchmarkB(b *testing.B) {}\n",
		"c_test.go": "package a\n\nimport \"testing\"\n\nfunc /* bench */ BenchmarkC(b *testing.B) {}\n",
	})
	found, err := declaresBenchmark(dir, []string{"a_test.go"})
	require.NoError(t, err, "declaresBenchmark must not error")
	assert.False(t, found, "a file without benchmarks should not count")

	found, err = declaresBenchmark(dir, []string{"a_test.go", "b_test.go"})
	require.NoError(t, err, "declaresBenchmark must not error")
	assert.True(t, found, "a file declaring a benchmark should count")

	found, err = declaresBenchmark(dir, []string{"c_test.go"})
	require.NoError(t, err, "declaresBenchmark must not error")
	assert.True(t, found, "a declaration with a comment between func and its name should count")

	_, err = declaresBenchmark(dir, []string{"absent_test.go"})
	assert.ErrorContains(t, err, "error reading", "an unreadable file should be reported")
}

func TestIsBenchmark(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		decl string
		exp  bool
	}{
		{decl: "func BenchmarkX(b *testing.B) {}", exp: true},
		{decl: "func Benchmark(b *testing.B) {}", exp: true},
		{decl: "func Benchmark_x(b *testing.B) {}", exp: true},
		{decl: "func BenchmarkX(b *B) {}", exp: true},
		{decl: "func Benchmarkx(b *testing.B) {}"},
		{decl: "func BenchX(b *testing.B) {}"},
		{decl: "func (s) BenchmarkX(b *testing.B) {}"},
		{decl: "func BenchmarkX(b testing.B) {}"},
		{decl: "func BenchmarkX(b *testing.T) {}"},
		{decl: "func BenchmarkX() {}"},
		{decl: "func BenchmarkX(a, b *testing.B) {}"},
		{decl: "func BenchmarkX(b *testing.B, n int) {}"},
		{decl: "func BenchmarkX(b *testing.B) error { return nil }"},
		{decl: "func BenchmarkX(b *[]testing.B) {}"},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), "x_test.go", "package x\n\n"+tc.decl+"\n", parser.SkipObjectResolution)
		require.NoErrorf(t, err, "the fixture %q must parse", tc.decl)
		fn, ok := f.Decls[0].(*ast.FuncDecl)
		require.Truef(t, ok, "the fixture %q must declare a function", tc.decl)
		assert.Equalf(t, tc.exp, isBenchmark(fn), "isBenchmark should judge %q as go test does", tc.decl)
	}
}
