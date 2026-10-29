package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
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

// newBenchmarkRepo commits a module whose package list names one package, fixture, holding files
func newBenchmarkRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	all := map[string]string{
		"go.mod":                   "module github.com/thrasher-corp/gocryptotrader\n\ngo 1.24\n",
		"benchmarks/packages.txt":  "fixture\n",
		"benchmarks/baseline.json": "{}\n",
	}
	for name, body := range files {
		all["fixture/"+name] = body
	}
	return newRepo(t, all)
}

// compareOptions returns the options comparing repo's working tree with base in the given number of
// rounds, each passing args as go test's flags
func compareOptions(t *testing.T, repo, base string, samples int, args ...string) *options {
	t.Helper()
	return &options{
		root:         repo,
		packagesPath: filepath.Join(repo, "benchmarks", "packages.txt"),
		baselinePath: filepath.Join(repo, "benchmarks", "baseline.json"),
		samples:      samples,
		compare:      base,
		out:          filepath.Join(t.TempDir(), "out"),
		args:         args,
	}
}

func TestBinaryFlags(t *testing.T) {
	t.Parallel()
	benchFlags := []string{"-run", "^$", "-bench", ".", "-benchmem", "-benchtime", "100ms", "-count", "7", "-cpu", "4", "-p", "1", "-v"}
	flags, err := binaryFlags(benchFlags, 7)
	require.NoError(t, err, "binaryFlags must not error for BENCH_FLAGS")
	assert.Equal(t, []string{"-test.paniconexit0", "-test.run=^$", "-test.bench=.", "-test.benchmem", "-test.benchtime=100ms", "-test.count=1", "-test.cpu=4", "-test.v"},
		flags, "BENCH_FLAGS should become a test binary's flags for one sample, with go test's -test.paniconexit0 and without its -p")

	flags, err = binaryFlags([]string{"--bench=.", "-count=3", "-v=true"}, 3)
	require.NoError(t, err, "binaryFlags must not error for values given with =")
	assert.Equal(t, []string{"-test.paniconexit0", "-test.bench=.", "-test.count=1", "-test.v=true"}, flags, "values given with = and double dashes should carry over")

	for _, tc := range []struct {
		name    string
		flags   []string
		samples int
		exp     error
	}{
		{"an unknown flag", []string{"-race"}, 7, errUnsupportedFlag},
		{"an argument that is not a flag", []string{"./currency/"}, 7, errUnsupportedFlag},
		{"a flag without its value", []string{"-bench"}, 7, errMissingFlagValue},
		{"a count other than the samples", []string{"-count", "5"}, 7, errCountMismatch},
		{"a count other than a single sample", []string{"-count", "3"}, 1, errCountMismatch},
	} {
		_, err := binaryFlags(tc.flags, tc.samples)
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
		{Typeflag: tar.TypeReg, Name: "pkg/../../outside.go", Mode: 0o644},
		{Typeflag: tar.TypeReg, Name: "/outside.go", Mode: 0o644},
	} {
		err := extractTar(tarArchive(t, h), t.TempDir())
		assert.ErrorIsf(t, err, errArchiveEntry, "extractTar should refuse %s", h.Name)
	}
}

func TestExtractTarReportsErrors(t *testing.T) {
	t.Parallel()
	file := tar.Header{Typeflag: tar.TypeReg, Name: "x.go", Mode: 0o644, Size: 3}
	whole := tarArchive(t, file).Bytes()
	for _, tc := range []struct {
		name    string
		archive []byte
		err     error
		text    string
	}{
		{"a truncated header", whole[:100], io.ErrUnexpectedEOF, "error reading git archive"},
		{"a truncated body", whole[:514], io.ErrUnexpectedEOF, "error extracting x.go"},
		{"a repeated file", tarArchive(t, file, file).Bytes(), os.ErrExist, "error extracting x.go"},
		{"a directory where a file is", tarArchive(t, tar.Header{Typeflag: tar.TypeReg, Name: "pkg", Mode: 0o644},
			tar.Header{Typeflag: tar.TypeDir, Name: "pkg/", Mode: 0o755}).Bytes(), syscall.ENOTDIR, "error extracting pkg/"},
	} {
		err := extractTar(bytes.NewReader(tc.archive), t.TempDir())
		assert.ErrorIsf(t, err, tc.err, "extractTar should report the cause of %s", tc.name)
		assert.ErrorContainsf(t, err, tc.text, "extractTar should say where it failed for %s", tc.name)
	}
}

func TestWriteArchiveFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "run.sh")
	h := &tar.Header{Typeflag: tar.TypeReg, Name: "a/b/run.sh", Mode: 0o755, Size: 4}
	require.NoError(t, writeArchiveFile(path, strings.NewReader("body and more"), h), "writeArchiveFile must not error")
	got, err := os.ReadFile(path)
	require.NoError(t, err, "reading the written file must not error")
	assert.Equal(t, "body", string(got), "only as many bytes as the header gives should be written")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err, "stat on the written file must not error")
		assert.NotZero(t, info.Mode().Perm()&0o100, "the header's mode should be kept")
	}
	assert.ErrorIs(t, writeArchiveFile(path, strings.NewReader("body"), h), os.ErrExist, "a file already written should not be replaced")
	assert.ErrorIs(t, writeArchiveFile(filepath.Join(path, "below"), strings.NewReader("body"), h), syscall.ENOTDIR, "a file below a file should fail")
	assert.ErrorIs(t, writeArchiveFile(filepath.Join(dir, "short"), strings.NewReader("abc"), h), io.EOF,
		"a body shorter than the header gives should fail")
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

	// git archive reads tar.umask before it writes anything, so it fails after an archive that reads as empty
	require.NoError(t, gitCommand(t, repo, "config", "tar.umask", "invalid").Run(), "git config must not error")
	_, err = extractRevision(t.Context(), repo, "HEAD", filepath.Join(t.TempDir(), "umask"))
	var exitErr *exec.ExitError
	assert.ErrorAs(t, err, &exitErr, "a git archive that fails should fail the export with its exit status")
	assert.ErrorContains(t, err, "error running git archive", "a git archive that fails should fail the export, naming it")
}

func TestExtractRevisionRefusesSymlinks(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink needs privileges on windows")
	}
	repo := newRepo(t, map[string]string{"target.txt": "target\n"})
	require.NoError(t, os.Symlink("target.txt", filepath.Join(repo, "link.txt")), "os.Symlink must not error")
	commitFiles(t, repo, nil)
	_, err := extractRevision(t.Context(), repo, "HEAD", filepath.Join(t.TempDir(), "base"))
	assert.ErrorIs(t, err, errArchiveEntry, "a symlink in the revision should fail the export rather than go missing")
}

func TestRevisionBuild(t *testing.T) {
	t.Parallel()
	repo := newBenchmarkRepo(t, map[string]string{
		"fixture_test.go": "package fixture\n\nimport \"testing\"\n\nfunc BenchmarkNothing(b *testing.B) {\n\tfor b.Loop() {\n\t}\n}\n",
	})
	base := newRevision("base", filepath.Join(t.TempDir(), "base"))
	_, err := extractRevision(t.Context(), repo, "HEAD", base.root)
	require.NoError(t, err, "extractRevision must not error")
	head := newRevision("head", repo)
	bin := t.TempDir()
	sums := make(map[string][sha256.Size]byte)
	for _, r := range []*revision{base, head} {
		require.NoErrorf(t, r.build(t.Context(), []string{"fixture"}, filepath.Join(bin, r.label)), "building the %s test binary must not error", r.label)
		require.Lenf(t, r.binaries, 1, "the %s revision must have a test binary for its one package", r.label)
		binary, err := os.ReadFile(r.binaries[0])
		require.NoErrorf(t, err, "reading the %s test binary must not error", r.label)
		sums[r.label] = sha256.Sum256(binary)
	}
	assert.Equal(t, sums["base"], sums["head"], "the same code should build into the same binary at both revisions, wherever each is checked out")

	writeFiles(t, repo, map[string]string{"fixture/fixture_test.go": "package fixture\n\nfunc broken() {\n"})
	err = head.build(t.Context(), []string{"fixture"}, filepath.Join(bin, "broken"))
	assert.ErrorContains(t, err, "error building the head test binary for fixture", "a package that does not compile should fail the build, naming it")
}

func TestRevisionBuildEnvironment(t *testing.T) {
	// No t.Parallel, since t.Setenv changes the process's environment. Set here so that the caller's own
	// settings, such as the GOTOOLCHAIN=local that CI's setup-go exports, cannot stand in for the build's
	t.Setenv("GOTOOLCHAIN", "auto")
	t.Setenv("GOWORK", "")
	const bench = "package fixture\n\nimport \"testing\"\n\nfunc BenchmarkNothing(b *testing.B) {\n\tfor b.Loop() {\n\t}\n}\n"
	later := newRepo(t, map[string]string{"go.mod": "module github.com/thrasher-corp/gocryptotrader\n\ngo 1.99.0\n", "fixture/fixture_test.go": bench})
	err := newRevision("head", later).build(t.Context(), []string{"fixture"}, t.TempDir())
	assert.ErrorContains(t, err, "GOTOOLCHAIN=local", "a revision whose go.mod asks for a later release should fail to build rather than switch toolchains")

	repo := newBenchmarkRepo(t, map[string]string{"fixture_test.go": bench})
	writeFiles(t, repo, map[string]string{"go.work": "go 1.24\n\nuse ./other\n", "other/go.mod": "module example.invalid/other\n\ngo 1.24\n"})
	assert.NoError(t, newRevision("head", repo).build(t.Context(), []string{"fixture"}, t.TempDir()), "a workspace over the checkout should not change how a revision builds")
}

func TestRevisionMeasure(t *testing.T) {
	t.Parallel()
	// Reported from the iteration count rather than measured, which a stray allocation in the runtime can
	// move, so that each metric and each process's differ
	repo := newBenchmarkRepo(t, map[string]string{
		"fixture_test.go": "package fixture\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
			"func BenchmarkWork(b *testing.B) {\n\tif _, err := os.Stat(\"fixture_test.go\"); err != nil {\n\t\tb.Fatal(err)\n\t}\n" +
			"\tvar n float64\n\tfor b.Loop() {\n\t\tn++\n\t}\n" +
			"\tb.ReportMetric(n, \"ns/op\")\n\tb.ReportMetric(n+1, \"B/op\")\n\tb.ReportMetric(n+2, \"allocs/op\")\n}\n\n" +
			"func BenchmarkSkipped(b *testing.B) {\n\tb.Skip(\"not here\")\n}\n\n" +
			"func BenchmarkNegative(b *testing.B) {\n\tfor b.Loop() {\n\t}\n\tb.ReportMetric(-1, \"B/op\")\n}\n\n" +
			"func BenchmarkFailing(b *testing.B) {\n\tb.Fatal(\"broken\")\n}\n",
	})
	r := newRevision("head", repo)
	require.NoError(t, r.build(t.Context(), []string{"fixture"}, t.TempDir()), "building the test binary must not error")
	// The package is measured as the second, so a measurement that ran the first binary would fail. It has the
	// built binary's name, since on Windows exec rejects a name without .exe before trying to start it
	r.binaries = append([]string{filepath.Join(t.TempDir(), filepath.Base(r.binaries[0]))}, r.binaries...)
	benchmarks := func(pattern, benchtime string) []string {
		t.Helper()
		flags, err := binaryFlags([]string{"-run", "^$", "-bench", pattern, "-benchmem", "-benchtime", benchtime, "-count", "1", "-v"}, 1)
		require.NoError(t, err, "binaryFlags must not error")
		return flags
	}
	for _, benchtime := range []string{"10x", "20x"} {
		require.NoErrorf(t, r.measure(t.Context(), 1, "fixture", benchmarks("^Benchmark(Work|Skipped)$", benchtime)), "measure must not error for -benchtime %s", benchtime)
	}
	assert.Equal(t, map[string]*Samples{
		"fixture.BenchmarkWork": {Pkg: "fixture", Name: "BenchmarkWork", NS: []float64{10, 20}, Bytes: []float64{11, 21}, Allocs: []float64{12, 22}},
	}, r.samples, "each process's metrics should be kept, in the order measured")
	flags := benchmarks("^Benchmark(Work|Skipped)$", "10x")
	assert.Equal(t, map[string]bool{"fixture.BenchmarkSkipped": true}, r.skipped, "a skipped benchmark should be recorded")
	assert.Equal(t, 2, strings.Count(r.output.String(), "pkg: github.com/thrasher-corp/gocryptotrader/fixture"), "each process's output should be kept")
	assert.Equal(t, cpuModel(r.output.Bytes()), r.cpu, "the processor should be read from the output")

	r.cpu = "Earlier CPU"
	require.NoError(t, r.measure(t.Context(), 1, "fixture", flags), "measure must not error")
	assert.Equal(t, "Earlier CPU", r.cpu, "the processor first read should be kept")

	err := r.measure(t.Context(), 1, "fixture", benchmarks("^BenchmarkNegative$", "10x"))
	assert.ErrorIs(t, err, errBadMeasurement, "output Parse rejects should fail the measurement")
	assert.ErrorContains(t, err, "head fixture: ", "a failed measurement should name the revision and package")
	assert.Contains(t, r.output.String(), "BenchmarkNegative", "output Parse rejects should still be kept")

	err = r.measure(t.Context(), 1, "fixture", benchmarks("^BenchmarkFailing$", "10x"))
	var exitErr *exec.ExitError
	assert.ErrorAs(t, err, &exitErr, "a failing process should fail the measurement with its exit status")
	assert.ErrorContains(t, err, "head fixture: benchmark process failed", "a failed measurement should name the revision and package")
	assert.Contains(t, r.output.String(), "broken", "a failing process's output should still be kept")
	assert.NotContains(t, r.samples, "fixture.BenchmarkNegative", "a measurement that failed should add no samples")

	var pathErr *fs.PathError
	assert.ErrorAs(t, r.measure(t.Context(), 0, "fixture", flags), &pathErr, "a binary that cannot be run should fail the measurement")
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

func TestGitEnv(t *testing.T) {
	// No t.Parallel, since t.Setenv changes the process's environment
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "index"))
	t.Setenv("GIT_WORK_TREE", "")
	t.Setenv("BENCHCHECK_GIT_DIR", "kept")
	t.Setenv("GITHUB_BENCHCHECK", "kept")
	// Asserted as booleans, since a failure would otherwise print the whole environment
	env := gitEnv()
	assert.True(t, slices.Contains(env, "BENCHCHECK_GIT_DIR=kept"), "gitEnv should keep a variable that does not start with GIT_")
	assert.True(t, slices.Contains(env, "GITHUB_BENCHCHECK=kept"), "gitEnv should keep a variable that starts with GIT but not GIT_")
	assert.False(t, slices.ContainsFunc(env, func(kv string) bool { return strings.HasPrefix(kv, "GIT_") }), "gitEnv should drop every GIT_ variable")
}

func TestGitOutput(t *testing.T) {
	// No t.Parallel, since t.Setenv changes the process's environment, here so git prints untranslated messages
	t.Setenv("LC_ALL", "C")
	repo := newRepo(t, map[string]string{"x.txt": "x\n"})
	head, err := gitCommand(t, repo, "rev-parse", "HEAD").Output()
	require.NoError(t, err, "git rev-parse must not error")
	got, err := gitOutput(t.Context(), repo, "rev-parse", "HEAD")
	require.NoError(t, err, "gitOutput must not error")
	assert.Equal(t, strings.TrimSpace(string(head)), got, "gitOutput should return git's output without its trailing newline")

	_, err = gitOutput(t.Context(), repo, "rev-parse", "--verify", "no-such-ref")
	var exitErr *exec.ExitError
	assert.ErrorAs(t, err, &exitErr, "gitOutput should return git's exit status")
	assert.ErrorContains(t, err, "error running git rev-parse --verify no-such-ref", "gitOutput should name the git command that failed")
	assert.ErrorContains(t, err, "Needed a single revision", "gitOutput should carry what git printed on failing")
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
		wd, err := os.Getwd()
		require.NoError(t, err, "os.Getwd must not error")
		fmt.Println("measured in " + wd)
		return
	case "spawn":
		self, err := os.Executable()
		require.NoError(t, err, "os.Executable must not error")
		// Outlives this process, holding its output open
		descendant := exec.CommandContext(context.WithoutCancel(t.Context()), self, "-test.run=^TestRunTestBinary$", "hang")
		descendant.Stdout = os.Stdout
		require.NoError(t, descendant.Start(), "starting the descendant must not error")
		fmt.Println("started", descendant.Process.Pid)
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
	firstLine, _, _ := strings.Cut(string(out), "\n")
	wd, ok := strings.CutPrefix(firstLine, "measured in ")
	require.True(t, ok, "the process's output must be returned")
	// Compared as files, since a temporary directory can be reached through a symlink, as on macOS
	got, err := os.Stat(wd)
	require.NoError(t, err, "stat on the process's working directory must not error")
	want, err := os.Stat(dir)
	require.NoError(t, err, "stat on the directory given must not error")
	assert.True(t, os.SameFile(want, got), "the process should run from the directory given, as go test runs a test binary from its package")

	start = time.Now()
	out, err = runTestBinary(t.Context(), self, dir, args("spawn"), time.Minute)
	assert.ErrorIs(t, err, exec.ErrWaitDelay, "a process that leaves a descendant holding its output should fail")
	assert.Less(t, time.Since(start), 30*time.Second, "a descendant holding the output should not keep the run waiting")
	firstLine, _, _ = strings.Cut(string(out), "\n")
	pid, err := strconv.Atoi(strings.TrimPrefix(firstLine, "started "))
	require.NoError(t, err, "the process must report the descendant it started")
	if descendant, err := os.FindProcess(pid); err == nil {
		_ = descendant.Kill()
	}
}

func TestCPUModel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Example CPU @ 3.00GHz", cpuModel([]byte("goos: linux\ngoarch: amd64\npkg: example\ncpu: Example CPU @ 3.00GHz\nBenchmarkX\n")),
		"the processor should be read from go test's cpu: header")
	assert.Empty(t, cpuModel([]byte("goos: linux\ngoarch: amd64\npkg: example\nBenchmarkX\n")), "output without a cpu: header should name no processor")
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
	// Every run writes to the same directory, so each must replace the files of the run before
	outDir := filepath.Join(t.TempDir(), "out")
	newOptions := func(base string) *options {
		// Enough iterations that the thread the runtime can start as b.Loop first resets the timer, once -cpu 2 has
		// raised GOMAXPROCS from 1, adds less than a byte an operation
		o := compareOptions(t, repo, base, 3, "-run", "^$", "-bench", ".", "-benchmem", "-benchtime", "100000x", "-count", "3", "-cpu", "2", "-v")
		o.out = outDir
		return o
	}

	o := newOptions("HEAD~1")
	var out bytes.Buffer
	require.NoError(t, runComparison(&out, o), "-compare must not error against a base without a package list")
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
	require.NoError(t, runComparison(&out, o), "-compare must not error")
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
	assert.Equal(t, []string{"base.txt", "head.txt", "summary.md"}, dirNames(t, outDir), "the work directory should be removed once the comparison is done")

	require.Error(t, runComparison(&bytes.Buffer{}, newOptions("no-such-revision")), "a base naming no commit must fail the comparison")
	assert.Equal(t, []string{"base.txt", "head.txt", "summary.md"}, dirNames(t, outDir), "a comparison that measured nothing should leave the files already there")
	kept, err := os.ReadFile(filepath.Join(outDir, "summary.md"))
	require.NoError(t, err, "reading summary.md must not error")
	assert.Equal(t, report, string(kept), "a comparison that measured nothing should leave the report already there")

	// A benchmark failing at the head stops the comparison, which keeps what each revision printed
	writeFiles(t, repo, map[string]string{"fixture/fixture_test.go": "package fixture\n\nimport \"testing\"\n\n" +
		"func BenchmarkWork(b *testing.B) {\n\tb.Fatal(\"broken at the head\")\n}\n"})
	o = newOptions("HEAD")
	require.Error(t, runComparison(&bytes.Buffer{}, o), "a failing benchmark must fail the comparison")
	raw, err := os.ReadFile(filepath.Join(o.out, "head.txt"))
	require.NoError(t, err, "head.txt must be saved when the comparison fails")
	assert.Contains(t, string(raw), "broken at the head", "the failing process's output should be kept")
	assert.Equal(t, []string{"base.txt", "head.txt"}, dirNames(t, outDir),
		"a failed comparison should keep the raw output, write no report, remove the report of the run before and remove its work directory")

	require.NoError(t, runComparison(&bytes.Buffer{}, newOptions("HEAD~1")), "-compare must not error against a base without a package list")
	assert.Equal(t, []string{"summary.md"}, dirNames(t, outDir), "a base without a package list should leave only its report, not the raw output of the run before")
}

// dirNames returns the names of the entries in dir, sorted
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "reading the directory must not error")
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestRunComparisonErrors(t *testing.T) {
	t.Parallel()
	const bench = "package fixture\n\nimport \"testing\"\n\nfunc BenchmarkNothing(b *testing.B) {\n\tfor b.Loop() {\n\t}\n}\n"
	newOptions := func(t *testing.T, benchmarks string) *options {
		t.Helper()
		repo := newBenchmarkRepo(t, map[string]string{"fixture_test.go": benchmarks})
		return compareOptions(t, repo, "HEAD", 1, "-run", "^$", "-bench", ".", "-benchmem", "-count", "1")
	}
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) *options
		err   error
		text  string
		// nanTiming marks a row whose timing reaches the U test as a NaN should the comparison stop refusing one
		nanTiming bool
	}{
		{name: "a go test flag without a test binary equivalent", err: errUnsupportedFlag, setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			o.args = append(o.args, "-race")
			return o
		}},
		{name: "a missing package list", err: errNoPackageList, setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			o.packagesPath = filepath.Join(t.TempDir(), "packages.txt")
			return o
		}},
		{name: "a missing baseline", err: errNoBaseline, setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			o.baselinePath = filepath.Join(t.TempDir(), "baseline.json")
			return o
		}},
		{name: "a tolerance without a reason", err: errToleranceNeedsReason, setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			writeFiles(t, o.root, map[string]string{"benchmarks/baseline.json": `{"fixture.BenchmarkNothing": {"allocs": 0, "bytes": 0, "allocs_tolerance": 0.1}}`})
			return o
		}},
		{name: "an output directory under a file", text: "error creating", setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			dir := t.TempDir()
			writeFiles(t, dir, map[string]string{"file": "not a directory\n"})
			o.out = filepath.Join(dir, "file", "out")
			return o
		}},
		{name: "an output directory that cannot be written", err: os.ErrPermission, setup: func(t *testing.T) *options {
			t.Helper()
			if runtime.GOOS == "windows" || os.Geteuid() == 0 {
				t.Skip("a read-only directory stops neither windows nor root from writing")
			}
			o := newOptions(t, bench)
			require.NoError(t, os.Mkdir(o.out, 0o500), "creating the read-only output directory must not error")
			return o
		}},
		{name: "a checkout on a branch without commits", text: "error running git rev-parse HEAD", setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			commit, err := gitCommand(t, o.root, "rev-parse", "HEAD").Output()
			require.NoError(t, err, "git rev-parse must not error")
			require.NoError(t, gitCommand(t, o.root, "checkout", "-q", "--orphan", "unborn").Run(), "git checkout must not error")
			o.compare = strings.TrimSpace(string(commit))
			return o
		}},
		{name: "an index git cannot read", text: "error running git status", setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			require.NoError(t, os.WriteFile(filepath.Join(o.root, ".git", "index"), []byte("invalid"), 0o600), "overwriting the index must not error")
			return o
		}},
		{name: "a base whose package list is malformed", err: errUnknownMarker, setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			commitFiles(t, o.root, map[string]string{"benchmarks/packages.txt": "fixture  included\n"})
			writeFiles(t, o.root, map[string]string{"benchmarks/packages.txt": "fixture\n"})
			return o
		}},
		{name: "a head that does not build", text: "error building the head test binary for fixture", setup: func(t *testing.T) *options {
			t.Helper()
			o := newOptions(t, bench)
			writeFiles(t, o.root, map[string]string{"fixture/fixture_test.go": "package fixture\n\nfunc broken() {\n"})
			return o
		}},
		{name: "a negative timing", err: errBadTiming, nanTiming: true, setup: func(t *testing.T) *options {
			t.Helper()
			return newOptions(t, "package fixture\n\nimport \"testing\"\n\n"+
				"func BenchmarkNegative(b *testing.B) {\n\tfor b.Loop() {\n\t}\n\tb.ReportMetric(-1, \"ns/op\")\n}\n")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := tc.setup(t)
			if tc.nanTiming {
				defer guardHeap(t)()
			}
			err := runComparison(&bytes.Buffer{}, o)
			if tc.err != nil {
				assert.ErrorIsf(t, err, tc.err, "runComparison should fail for %s", tc.name)
			} else {
				assert.ErrorContainsf(t, err, tc.text, "runComparison should fail for %s", tc.name)
			}
		})
	}
}

func TestRunComparisonWithoutWorkingDirectory(t *testing.T) {
	// No t.Parallel, since t.Chdir changes the process's working directory
	if runtime.GOOS != "linux" {
		t.Skip("relies on linux failing os.Getwd once the working directory is removed")
	}
	repo := newBenchmarkRepo(t, nil)
	for _, tc := range []struct {
		name     string
		relative func(o *options)
	}{
		{"a relative output directory", func(o *options) { o.out = "out" }},
		{"a relative checkout", func(o *options) { o.root = "." }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gone := t.TempDir()
			t.Chdir(gone)
			require.NoError(t, os.Remove(gone), "removing the working directory must not error")
			o := compareOptions(t, repo, "HEAD", 1, "-count", "1")
			tc.relative(o)
			assert.ErrorIs(t, runComparison(&bytes.Buffer{}, o), os.ErrNotExist, "a path that cannot be made absolute should fail the comparison")
		})
	}
}

func TestRunComparisonAlternatesRevisions(t *testing.T) {
	t.Parallel()
	// Each process's TestMain records the revision its rev.txt names and the directory it runs from
	order := filepath.Join(t.TempDir(), "order.log")
	const mainTest = "package fixture\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
		"func TestMain(m *testing.M) {\n\trev, _ := os.ReadFile(\"rev.txt\")\n\twd, _ := os.Getwd()\n" +
		"\tif f, err := os.OpenFile(%q, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {\n" +
		"\t\t_, _ = f.WriteString(string(rev) + \" \" + wd + \"\\n\")\n\t\t_ = f.Close()\n\t}\n\tos.Exit(m.Run())\n}\n"
	repo := newBenchmarkRepo(t, map[string]string{
		"fixture_test.go": "package fixture\n\nimport \"testing\"\n\nfunc BenchmarkNothing(b *testing.B) {\n\tfor b.Loop() {\n\t}\n}\n",
		"main_test.go":    fmt.Sprintf(mainTest, order),
		"rev.txt":         "base",
	})
	writeFiles(t, repo, map[string]string{"fixture/rev.txt": "head"})
	o := compareOptions(t, repo, "HEAD", 5, "-run", "^$", "-bench", ".", "-benchmem", "-benchtime", "1x", "-count", "5")
	require.NoError(t, runComparison(&bytes.Buffer{}, o), "-compare must not error")
	recorded, err := os.ReadFile(order)
	require.NoError(t, err, "reading the order the processes ran in must not error")
	var got []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(recorded)), "\n") {
		rev, wd, _ := strings.Cut(line, " ")
		got = append(got, rev)
		if rev == "base" {
			assert.Truef(t, strings.HasPrefix(wd, filepath.Join(o.out, ".work-")), "the base should run from its export in the output directory, not %s", wd)
		}
	}
	assert.Equal(t, []string{"base", "head", "head", "base", "base", "head", "head", "base", "base", "head"}, got,
		"each round should swap which revision runs first")
}

type progressTrigger struct {
	trigger string
	reached chan struct{}
	once    sync.Once
}

func (w *progressTrigger) Write(p []byte) (int, error) {
	if strings.Contains(string(p), w.trigger) {
		w.once.Do(func() { close(w.reached) })
	}
	return len(p), nil
}

func TestRunComparisonRemovesWorkWhenInterrupted(t *testing.T) {
	// No t.Parallel, since the signal reaches the whole process and would stop any other comparison
	if runtime.GOOS == "windows" {
		t.Skip("a benchmark cannot send its parent an interrupt on windows")
	}
	// Caught here as well until every comparison has returned, so that one that does not handle the signal
	// fails this test, and a later benchmark's signal cannot end the test binary
	held := make(chan os.Signal, 1)
	signal.Notify(held, syscall.SIGINT, syscall.SIGTERM)
	var comparisons sync.WaitGroup
	defer func() {
		go func() {
			comparisons.Wait()
			signal.Stop(held)
		}()
	}()
	// The base's benchmark signals the comparison, its parent, as Ctrl-C would while it runs, then sleeps
	// for longer than the test waits
	const slow = "package fixture\n\nimport (\n\t\"os\"\n\t\"syscall\"\n\t\"testing\"\n\t\"time\"\n)\n\n" +
		"func BenchmarkSlow(b *testing.B) {\n\tif p, err := os.FindProcess(os.Getppid()); err == nil {\n\t\t_ = p.Signal(syscall.Signal(%d))\n\t}\n" +
		"\ttime.Sleep(time.Minute)\n\tfor b.Loop() {\n\t}\n}\n"
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		repo := newBenchmarkRepo(t, map[string]string{"fixture_test.go": fmt.Sprintf(slow, int(sig))})
		o := compareOptions(t, repo, "HEAD", 1, "-run", "^$", "-bench", ".", "-benchmem", "-count", "1", "-v")
		built := &progressTrigger{trigger: "built both revisions' test binaries", reached: make(chan struct{})}
		o.progress = built
		done := make(chan error, 1)
		comparisons.Go(func() { done <- runComparison(&bytes.Buffer{}, o) })
		var err error
		select {
		case <-built.reached:
			select {
			case err = <-done:
			case <-time.After(30 * time.Second):
				require.FailNowf(t, "a signalled comparison must stop rather than measure on", "signal %s", sig)
			}
		case err = <-done:
			// The comparison can finish before this goroutine sees the build complete
			select {
			case <-built.reached:
			default:
				require.FailNowf(t, "the comparison must build both revisions before it is signalled", "%s: %v", sig, err)
			}
		}
		// Only the comparison's own cancellation kills the base's process
		require.ErrorContainsf(t, err, "base fixture: benchmark process failed: signal: killed", "a comparison sent %s must stop measuring the base", sig)
		assert.Equalf(t, []string{"base.txt", "head.txt"}, dirNames(t, o.out), "a comparison sent %s should keep the raw output and remove its work directory", sig)
		raw, err := os.ReadFile(filepath.Join(o.out, "base.txt"))
		require.NoError(t, err, "reading base.txt must not error")
		assert.Containsf(t, string(raw), "pkg: github.com/thrasher-corp/gocryptotrader/fixture", "the base's output before %s should be kept", sig)
	}
}

func TestRunComparisonFailsABenchmarkThatExits(t *testing.T) {
	t.Parallel()
	repo := newBenchmarkRepo(t, map[string]string{
		"fixture_test.go": "package fixture\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n" +
			"func BenchmarkExits(b *testing.B) {\n\tos.Exit(0)\n}\n",
	})
	o := compareOptions(t, repo, "HEAD", 1, "-run", "^$", "-bench", ".", "-benchmem", "-count", "1")
	err := runComparison(&bytes.Buffer{}, o)
	assert.ErrorContains(t, err, "unexpected call to os.Exit(0) during test",
		"a benchmark ending its process with status 0 should fail the comparison, as it fails go test")
}
