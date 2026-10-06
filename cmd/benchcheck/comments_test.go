package main

import (
	"fmt"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// benchmarkResultPattern matches a measurement in the units go test reports: a number followed by a
// time, B/op, allocs/op or throughput unit. The leading digit is what separates a pasted result
// from prose naming a unit ("carries no B/op or allocs/op").
var benchmarkResultPattern = regexp.MustCompile(`\d\s*(?:[nuµm]?s/op|B/op|allocs/op|[KMGT]?B/s)`)

// commentedResults returns a "file:line: text" description of each comment line in src that holds a
// benchmark result. Only comments are scanned, so a measurement inside a string literal is not
// mistaken for one, and every line of a block comment is seen whether or not it starts with a star.
func commentedResults(path string, src []byte) []string {
	var found []string
	fset := token.NewFileSet()
	var s scanner.Scanner
	// The error handler is deliberately a no-op: a file go vet rejects still has comments worth
	// checking, and the build will report the syntax error far more usefully than this would
	s.Init(fset.AddFile(path, -1, len(src)), src, func(token.Position, string) {}, scanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return found
		}
		if tok != token.COMMENT {
			continue
		}
		start := fset.Position(pos).Line
		for i, line := range strings.Split(lit, "\n") {
			if benchmarkResultPattern.MatchString(line) {
				found = append(found, fmt.Sprintf("%s:%d: %s", path, start+i, strings.TrimSpace(line)))
			}
		}
	}
}

func TestCommentedResults(t *testing.T) {
	t.Parallel()
	src := "package x\n" +
		"\n" +
		"// 13636467	        84.26 ns/op	     141 B/op	       1 allocs/op\n" + // line 3
		"func BenchmarkA() {}\n" +
		"\n" +
		"/*\n" +
		"Before: 18.58 µs/op  3120 B/op  65 allocs/op\n" + // line 7, unstarred in a block comment
		"*/\n" +
		"var url = \"https://example.invalid/5 ns/op\"\n" + // a string literal, not a comment
		"var m = metrics[\"ns/op\"] // reads the ns/op metric\n" + // prose naming units
		"// Throughput reached 120 MB/s\n" + // line 11
		"var y = 2 // keeps this at 0 allocs/op\n" // line 12, trailing comment
	exp := []string{
		"x.go:3: // 13636467	        84.26 ns/op	     141 B/op	       1 allocs/op",
		"x.go:7: Before: 18.58 µs/op  3120 B/op  65 allocs/op",
		"x.go:11: // Throughput reached 120 MB/s",
		"x.go:12: // keeps this at 0 allocs/op",
	}
	assert.Equal(t, exp, commentedResults("x.go", []byte(src)),
		"every comment line holding a measurement should be found, and nothing outside comments")
}

// TestNoBenchmarkResultsInComments keeps benchmark numbers out of comments. benchmarks/baseline.json
// is the single source of truth for them and `make bench` verifies it on every run, so a number
// pasted into a comment can only go stale. misc_checks.sh runs this test by name.
func TestNoBenchmarkResultsInComments(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err, "the repository root must resolve")
	var sources []string
	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		// Read the index in a checkout, so untracked and vendored files are never checked
		cmd := exec.CommandContext(t.Context(), "git", "-c", "safe.directory=*", "ls-files", "-z", "*.go")
		cmd.Dir = root
		out, err := cmd.Output()
		require.NoError(t, err, "tracked Go sources must be listed")
		for path := range strings.SplitSeq(string(out), "\x00") {
			if path != "" {
				sources = append(sources, filepath.Join(root, filepath.FromSlash(path)))
			}
		}
	} else {
		// A tree exported without repository metadata has no index to read, so walk it instead
		require.NoError(t, filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && path != root && (d.Name() == ".git" || d.Name() == "vendor") {
				return fs.SkipDir
			}
			if !d.IsDir() && strings.HasSuffix(path, ".go") {
				sources = append(sources, path)
			}
			return nil
		}), "the repository's Go sources must be walked")
	}
	require.NotEmpty(t, sources, "the repository must contain Go sources")

	for _, path := range sources {
		src, err := os.ReadFile(path)
		require.NoErrorf(t, err, "reading %s must not error", path)
		rel, err := filepath.Rel(root, path)
		require.NoErrorf(t, err, "%s must be inside the repository", path)
		assert.Emptyf(t, commentedResults(filepath.ToSlash(rel), src),
			"%s should hold no benchmark results in comments; benchmarks/baseline.json records them", rel)
	}
}
