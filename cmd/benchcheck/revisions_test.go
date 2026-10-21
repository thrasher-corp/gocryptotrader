package main

import (
	"archive/tar"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRepo commits files to a new repository and returns its directory
func newRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, gitCommand(t, dir, "init", "-q").Run(), "git init must not error")
	// git archive would otherwise write CRLF line endings where the system config asks for them
	require.NoError(t, gitCommand(t, dir, "config", "core.autocrlf", "false").Run(), "git config must not error")
	commitFiles(t, dir, files)
	return dir
}

// commitFiles writes files to the repository at dir and commits everything
func commitFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	writeFiles(t, dir, files)
	require.NoError(t, gitCommand(t, dir, "add", "-A").Run(), "git add must not error")
	out, err := gitCommand(t, dir, "-c", "user.name=benchcheck", "-c", "user.email=benchcheck@example.invalid",
		"-c", "commit.gpgsign=false", "-c", "core.hooksPath="+os.DevNull, "commit", "-q", "-m", "fixture").CombinedOutput()
	require.NoErrorf(t, err, "git commit must not error: %s", out)
}

func TestBinaryFlags(t *testing.T) {
	t.Parallel()
	benchFlags := []string{"-run", "^$", "-bench", ".", "-benchmem", "-benchtime", "100ms", "-count", "7", "-cpu", "4", "-p", "1", "-v"}
	flags, err := binaryFlags(benchFlags, 7)
	require.NoError(t, err, "binaryFlags must not error for BENCH_FLAGS")
	assert.Equal(t, []string{"-test.run=^$", "-test.bench=.", "-test.benchmem", "-test.benchtime=100ms", "-test.count=1", "-test.cpu=4", "-test.v"},
		flags, "BENCH_FLAGS should become a test binary's flags for one sample, without go test's -p")

	flags, err = binaryFlags([]string{"--bench=.", "-count=3", "-v=true"}, 3)
	require.NoError(t, err, "binaryFlags must not error for values given with =")
	assert.Equal(t, []string{"-test.bench=.", "-test.count=1", "-test.v=true"}, flags, "values given with = and double dashes should carry over")

	for _, tc := range []struct {
		name  string
		flags []string
		exp   error
	}{
		{"an unknown flag", []string{"-race"}, errUnsupportedFlag},
		{"an argument that is not a flag", []string{"./currency/"}, errUnsupportedFlag},
		{"a flag without its value", []string{"-bench"}, errMissingFlagValue},
		{"a count other than the samples", []string{"-count", "5"}, errCountMismatch},
	} {
		_, err := binaryFlags(tc.flags, 7)
		assert.ErrorIsf(t, err, tc.exp, "binaryFlags should refuse %s", tc.name)
	}
}

func TestRoundOrder(t *testing.T) {
	t.Parallel()
	base, head := newRevision("base", ""), newRevision("head", "")
	for round, exp := range [][]string{{"base", "head"}, {"head", "base"}, {"base", "head"}, {"head", "base"}} {
		order := roundOrder(round, base, head)
		got := make([]string, 0, len(order))
		for _, r := range order {
			got = append(got, r.label)
		}
		assert.Equalf(t, exp, got, "round %d should swap which revision runs first", round)
	}
}

func TestSharedPackages(t *testing.T) {
	t.Parallel()
	shared, onlyHead, onlyBase := sharedPackages([]string{"c", "a", "b"}, []string{"b", "d", "c"})
	assert.Equal(t, []string{"c", "b"}, shared, "the shared packages should keep the head's order")
	assert.Equal(t, []string{"a"}, onlyHead, "a package only the head lists should be set apart")
	assert.Equal(t, []string{"d"}, onlyBase, "a package only the base lists should be set apart")
}

// tarArchive builds an archive of the given entries, with each regular file's body after its header
func tarArchive(t *testing.T, entries ...tar.Header) *bytes.Buffer {
	t.Helper()
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	for i := range entries {
		h := &entries[i]
		body := strings.Repeat("x", int(h.Size))
		require.NoErrorf(t, tw.WriteHeader(h), "writing the header for %s must not error", h.Name)
		_, err := tw.Write([]byte(body))
		require.NoErrorf(t, err, "writing the body of %s must not error", h.Name)
	}
	require.NoError(t, tw.Close(), "closing the archive must not error")
	return &archive
}

func TestExtractTar(t *testing.T) {
	t.Parallel()
	archive := tarArchive(t,
		tar.Header{Typeflag: tar.TypeXGlobalHeader, Name: "pax_global_header", PAXRecords: map[string]string{"comment": "0123abcd"}},
		tar.Header{Typeflag: tar.TypeDir, Name: "pkg/", Mode: 0o755},
		tar.Header{Typeflag: tar.TypeReg, Name: "pkg/x.go", Mode: 0o644, Size: 3},
		tar.Header{Typeflag: tar.TypeReg, Name: "scripts/run.sh", Mode: 0o755, Size: 2},
	)
	dir := t.TempDir()
	require.NoError(t, extractTar(archive, dir), "extractTar must not error")

	got, err := os.ReadFile(filepath.Join(dir, "pkg", "x.go"))
	require.NoError(t, err, "reading an extracted file must not error")
	assert.Equal(t, "xxx", string(got), "a file should be extracted with its contents")
	got, err = os.ReadFile(filepath.Join(dir, "scripts", "run.sh"))
	require.NoError(t, err, "reading a file whose directory had no entry must not error")
	assert.Equal(t, "xx", string(got), "a file should be extracted even when its directory has no entry")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "scripts", "run.sh"))
		require.NoError(t, err, "stat on an extracted file must not error")
		assert.NotZero(t, info.Mode().Perm()&0o100, "an executable file should stay executable")
	}
	_, err = os.Stat(filepath.Join(dir, "pax_global_header"))
	assert.ErrorIs(t, err, os.ErrNotExist, "git archive's record of the commit should not be written out")
}

func TestExtractTarRefusesUnexpectedEntries(t *testing.T) {
	t.Parallel()
	for _, h := range []tar.Header{
		{Typeflag: tar.TypeSymlink, Name: "link", Linkname: "elsewhere"},
		{Typeflag: tar.TypeReg, Name: "../outside.go", Mode: 0o644},
	} {
		err := extractTar(tarArchive(t, h), t.TempDir())
		assert.ErrorIsf(t, err, errArchiveEntry, "extractTar should refuse %s", h.Name)
	}
}

func TestExtractRevision(t *testing.T) {
	t.Parallel()
	repo := newRepo(t, map[string]string{"go.mod": "module example.invalid/m\n", "pkg/x.go": "package pkg\n"})
	writeFiles(t, repo, map[string]string{"pkg/x.go": "package pkg // uncommitted\n"})
	dir := filepath.Join(t.TempDir(), "base")
	commit, err := extractRevision(t.Context(), repo, "HEAD", dir)
	require.NoError(t, err, "extractRevision must not error")

	head, err := gitCommand(t, repo, "rev-parse", "HEAD").Output()
	require.NoError(t, err, "git rev-parse must not error")
	assert.Equal(t, strings.TrimSpace(string(head)), commit, "extractRevision should return the commit it exported")
	got, err := os.ReadFile(filepath.Join(dir, "pkg", "x.go"))
	require.NoError(t, err, "reading the exported file must not error")
	assert.Equal(t, "package pkg\n", string(got), "the committed file should be exported, not the working tree's")

	_, err = extractRevision(t.Context(), repo, "-o", filepath.Join(t.TempDir(), "dash"))
	assert.ErrorIs(t, err, errBadRevision, "a revision git would read as an option should be refused")
	_, err = extractRevision(t.Context(), repo, "no-such-branch", filepath.Join(t.TempDir(), "missing"))
	assert.Error(t, err, "a revision naming no commit should fail")
}

func TestExtractRevisionIgnoresHookEnvironment(t *testing.T) {
	// No t.Parallel, since t.Setenv changes the process's environment
	hook := newRepo(t, map[string]string{"hook.txt": "hook\n"})
	repo := newRepo(t, map[string]string{"repo.txt": "repo\n"})
	t.Setenv("GIT_DIR", filepath.Join(hook, ".git"))
	t.Setenv("GIT_WORK_TREE", hook)

	dir := filepath.Join(t.TempDir(), "base")
	_, err := extractRevision(t.Context(), repo, "HEAD", dir)
	require.NoError(t, err, "extractRevision must not error while a hook's variables are set")
	assert.FileExists(t, filepath.Join(dir, "repo.txt"), "the repository asked for should be exported")
	assert.NoFileExists(t, filepath.Join(dir, "hook.txt"), "the hook's repository should not be exported")
}

func TestRunTestBinary(t *testing.T) {
	t.Parallel()
	// Run again by the test below as the process it measures
	switch flag.Arg(0) {
	case "hang":
		time.Sleep(time.Minute)
		return
	case "fail":
		fmt.Println("partial output")
		os.Exit(3)
	case "print":
		fmt.Println("measured")
		return
	}
	self, err := os.Executable()
	require.NoError(t, err, "os.Executable must not error")
	args := func(mode string) []string { return []string{"-test.run=^TestRunTestBinary$", mode} }
	dir := t.TempDir()

	start := time.Now()
	_, err = runTestBinary(t.Context(), self, dir, args("hang"), 100*time.Millisecond)
	assert.ErrorIs(t, err, errProcessTimeout, "a process running past its limit should be stopped and reported")
	assert.Less(t, time.Since(start), 30*time.Second, "a process running past its limit should be stopped promptly")

	out, err := runTestBinary(t.Context(), self, dir, args("fail"), time.Minute)
	require.Error(t, err, "a process exiting non-zero must be an error")
	assert.NotErrorIs(t, err, errProcessTimeout, "a failing process should not read as a timeout")
	assert.Contains(t, string(out), "partial output", "a failing process's output should still be returned")

	out, err = runTestBinary(t.Context(), self, dir, args("print"), time.Minute)
	require.NoError(t, err, "a process exiting zero must not error")
	assert.Contains(t, string(out), "measured", "the process's output should be returned")
}

func TestRunComparisonRefusesOtherModes(t *testing.T) {
	t.Parallel()
	for _, o := range []*options{
		{compare: "HEAD", samples: 7, update: true},
		{compare: "HEAD", samples: 7, prune: true},
		{compare: "HEAD", samples: 7, list: true},
	} {
		assert.ErrorIs(t, run(strings.NewReader(""), &bytes.Buffer{}, o), errCompareModes, "-compare should refuse another mode")
	}
	err := run(strings.NewReader(""), &bytes.Buffer{}, &options{compare: "HEAD", samples: 4})
	assert.ErrorIs(t, err, errInvalidSamples, "-compare should refuse an even sample count")
}

func TestRunComparison(t *testing.T) {
	t.Parallel()
	const work = "package fixture\n\n// Work allocates the slice it returns\nfunc Work(n int) []byte { return make([]byte, n) }\n"
	const bench = "package fixture\n\nimport \"testing\"\n\nvar sink []byte\n\n" +
		"func BenchmarkWork(b *testing.B) {\n\tfor b.Loop() {\n\t\tsink = Work(64)\n\t}\n}\n\n" +
		"func BenchmarkSkipped(b *testing.B) {\n\t%s\n\tfor b.Loop() {\n\t\tsink = Work(8)\n\t}\n}\n"
	repo := newRepo(t, map[string]string{
		"go.mod":                  "module github.com/thrasher-corp/gocryptotrader\n\ngo 1.24\n",
		"fixture/fixture.go":      work,
		"fixture/fixture_test.go": fmt.Sprintf(bench, "b.ReportAllocs()"),
	})
	commitFiles(t, repo, map[string]string{"benchmarks/packages.txt": "fixture\n", "benchmarks/baseline.json": "{}\n"})
	// Untracked, as CI's output directory is, so it must not count as a change
	writeFiles(t, repo, map[string]string{"notes.txt": "untracked\n"})
	newOptions := func(base string) *options {
		return &options{
			root:         repo,
			packagesPath: filepath.Join(repo, "benchmarks", "packages.txt"),
			baselinePath: filepath.Join(repo, "benchmarks", "baseline.json"),
			samples:      3,
			compare:      base,
			out:          filepath.Join(t.TempDir(), "out"),
			args:         []string{"-run", "^$", "-bench", ".", "-benchmem", "-benchtime", "100x", "-count", "3", "-cpu", "2", "-v"},
		}
	}

	o := newOptions("HEAD~1")
	var out bytes.Buffer
	require.NoError(t, run(strings.NewReader(""), &out, o), "-compare must not error against a base without a package list")
	assert.Contains(t, out.String(), "predates `benchmarks/packages.txt`", "a base without a package list should have nothing to compare")
	assert.NotContains(t, out.String(), "uncommitted", "an untracked file should not count as an uncommitted change")
	assert.NoFileExists(t, filepath.Join(o.out, "base.txt"), "nothing should be measured against a base without a package list")

	// The working tree allocates twice per operation, and skips the second benchmark
	writeFiles(t, repo, map[string]string{
		"fixture/fixture.go": "package fixture\n\nvar kept []byte\n\n" +
			"// Work allocates the slice it returns, and another it keeps\nfunc Work(n int) []byte {\n\tkept = make([]byte, n)\n\treturn make([]byte, n)\n}\n",
		"fixture/fixture_test.go": fmt.Sprintf(bench, `b.Skip("not at the head")`),
	})
	o = newOptions("HEAD")
	out.Reset()
	require.NoError(t, run(strings.NewReader(""), &out, o), "-compare must not error")
	report := out.String()
	allocations := "| 64 → 128 (+100.0%) | 1 → 2 (+100.0%) |"
	assert.Contains(t, report, "### Moved (1)", "the doubled allocation should lead the report")
	assert.Contains(t, report, allocations, "the report should show the allocations at both revisions")
	assert.Contains(t, report, "with uncommitted changes", "the report should say the head was a working tree with changes")
	assert.Contains(t, report, "- `fixture.BenchmarkSkipped`: skipped at the head", "a benchmark the head skips should be reported as such")

	base, err := os.ReadFile(filepath.Join(o.out, "base.txt"))
	require.NoError(t, err, "reading base.txt must not error")
	if cpu := cpuModel(base); cpu != "" {
		assert.Contains(t, report, ", on "+cpu+".", "the report should name the processor go test reported")
	}

	summary, err := os.ReadFile(filepath.Join(o.out, "summary.md"))
	require.NoError(t, err, "reading summary.md must not error")
	assert.Equal(t, report, string(summary), "summary.md should hold the report printed")
	for _, side := range []string{"base", "head"} {
		raw, err := os.ReadFile(filepath.Join(o.out, side+".txt"))
		require.NoErrorf(t, err, "reading %s.txt must not error", side)
		assert.Equalf(t, 3, strings.Count(string(raw), "pkg: github.com/thrasher-corp/gocryptotrader/fixture"),
			"%s.txt should hold the output of each of the three processes", side)
	}
	entries, err := os.ReadDir(o.out)
	require.NoError(t, err, "reading the output directory must not error")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	assert.Equal(t, []string{"base.txt", "head.txt", "summary.md"}, names, "the work directory should be removed once the comparison is done")

	// A benchmark failing at the head stops the comparison, which keeps what each revision printed
	writeFiles(t, repo, map[string]string{"fixture/fixture_test.go": "package fixture\n\nimport \"testing\"\n\n" +
		"func BenchmarkWork(b *testing.B) {\n\tb.Fatal(\"broken at the head\")\n}\n"})
	o = newOptions("HEAD")
	require.Error(t, run(strings.NewReader(""), &bytes.Buffer{}, o), "a failing benchmark must fail the comparison")
	raw, err := os.ReadFile(filepath.Join(o.out, "head.txt"))
	require.NoError(t, err, "head.txt must be saved when the comparison fails")
	assert.Contains(t, string(raw), "broken at the head", "the failing process's output should be kept")
	entries, err = os.ReadDir(o.out)
	require.NoError(t, err, "reading the output directory must not error")
	names = names[:0]
	for _, e := range entries {
		names = append(names, e.Name())
	}
	slices.Sort(names)
	assert.Equal(t, []string{"base.txt", "head.txt"}, names, "a failed comparison should keep the raw output, write no report and remove its work directory")
}
