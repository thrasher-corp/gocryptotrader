package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	errUnknownMarker        = errors.New("unknown package marker")
	errExclusionNeedsReason = errors.New("excluded package requires a reason after #")
	errDuplicatePackage     = errors.New("package listed more than once")
	errNoPackageList        = errors.New("package list does not exist")
	errEmptyPackageList     = errors.New("package list names no packages to benchmark")
	errBadPackagePath       = errors.New("package path is not a package inside this module")
	errUnlistedPackage      = errors.New("package declares benchmarks but is neither listed nor excluded")
	errStalePackage         = errors.New("package is listed but declares no benchmarks")
	errRootBenchmarks       = errors.New("the module root declares benchmarks, which the package list cannot name; move them into a package")
)

// Packages is the benchmarked package list in file order, and the excluded packages with their reasons
type Packages struct {
	List     []string
	Excluded map[string]string
}

// LoadPackages reads the package list: a package path per line, relative to the module root, or a
// path followed by "excluded" and a # comment giving the reason
func LoadPackages(path string) (*Packages, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		// An empty list gates nothing, so a mistyped path must not pass as one
		return nil, fmt.Errorf("%w: %s", errNoPackageList, path)
	}
	if err != nil {
		return nil, fmt.Errorf("error reading package list: %w", err)
	}
	defer f.Close()

	pkgs := &Packages{Excluded: map[string]string{}}
	seen := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line, comment, _ := strings.Cut(sc.Text(), "#")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg, err := canonicalPackage(fields[0])
		if err != nil {
			return nil, err
		}
		// One entry per package, so a package cannot be both listed and excluded
		if seen[pkg] {
			return nil, fmt.Errorf("%w: %q", errDuplicatePackage, pkg)
		}
		seen[pkg] = true
		switch {
		case len(fields) == 1:
			pkgs.List = append(pkgs.List, pkg)
		case len(fields) == 2 && fields[1] == "excluded":
			reason := strings.TrimSpace(comment)
			if reason == "" {
				return nil, fmt.Errorf("%w: %q", errExclusionNeedsReason, pkg)
			}
			pkgs.Excluded[pkg] = reason
		default:
			return nil, fmt.Errorf("%w: %q on package %q", errUnknownMarker, strings.Join(fields[1:], " "), pkg)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("error reading package list: %w", err)
	}
	if len(pkgs.List) == 0 {
		return nil, fmt.Errorf("%w: %s", errEmptyPackageList, path)
	}
	return pkgs, nil
}

// Covers checks that every package declaring benchmarks is listed or excluded, and that every listed
// or excluded package still declares one
func (p *Packages) Covers(declared []string) error {
	have := make(map[string]bool, len(declared))
	var errs []error
	for _, pkg := range declared {
		have[pkg] = true
		if _, excluded := p.Excluded[pkg]; !excluded && !slices.Contains(p.List, pkg) {
			errs = append(errs, fmt.Errorf("%w: %s", errUnlistedPackage, pkg))
		}
	}
	for _, pkg := range slices.Concat(p.List, sortedKeys(p.Excluded)) {
		if !have[pkg] {
			errs = append(errs, fmt.Errorf("%w: %s", errStalePackage, pkg))
		}
	}
	return errors.Join(errs...)
}

// canonicalPackage reduces a package list entry to the path go test prints in its pkg: header, since
// an entry spelt any other way would run its benchmarks and gate none of them. It refuses wildcards,
// paths outside the module, and characters a shell would rewrite, since -list reaches go test
// unquoted.
func canonicalPackage(entry string) (string, error) {
	root := strings.TrimSuffix(modulePrefix, "/")
	pkg := pathpkg.Clean(entry)
	// Cleaning erases the module boundary, so ".../gocryptotrader/../../../currency" is caught here
	if strings.HasPrefix(entry, root) && pkg != root && !strings.HasPrefix(pkg, modulePrefix) {
		return "", fmt.Errorf("%w: %q traverses outside the module", errBadPackagePath, entry)
	}
	if pkg == root || pkg == "." {
		return "", fmt.Errorf("%w: %q resolves to the module root", errBadPackagePath, entry)
	}
	pkg = strings.TrimPrefix(pkg, modulePrefix)
	switch {
	case strings.Contains(pkg, "..."):
		return "", fmt.Errorf("%w: %q is a wildcard, which covers several packages under one entry", errBadPackagePath, entry)
	case strings.HasPrefix(pkg, "/"), strings.HasPrefix(pkg, ".."):
		return "", fmt.Errorf("%w: %q resolves outside the module", errBadPackagePath, entry)
	}
	for _, r := range pkg {
		if !importPathRune(r) {
			return "", fmt.Errorf("%w: %q contains %q, which the shell would rewrite before go test saw it",
				errBadPackagePath, entry, string(r))
		}
	}
	return pkg, nil
}

// importPathRune reports whether r may appear in a Go import path, as x/mod/module's importPathOK
// allows, or is its separator; none of those mean anything to a shell
func importPathRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '/', r == '.', r == '-', r == '_', r == '~', r == '+':
		return true
	}
	return false
}

// BenchmarkPackages returns, in path order, every package under the module root whose test files
// declare a benchmark under the gate's build constraints, skipping what `go test ./...` skips
func BenchmarkPackages(root string) ([]string, error) {
	ctx := build.Default
	ctx.GOOS, ctx.GOARCH = canonicalGOOS, canonicalGOARCH
	var pkgs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if path != root {
			if name := d.Name(); name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
				return filepath.SkipDir
			}
		}
		p, err := ctx.ImportDir(path, 0)
		if err != nil {
			if _, ok := errors.AsType[*build.NoGoError](err); ok {
				return nil
			}
			return fmt.Errorf("error loading package %s: %w", path, err)
		}
		found, err := declaresBenchmark(path, slices.Concat(p.TestGoFiles, p.XTestGoFiles))
		if err != nil || !found {
			return err
		}
		if path == root {
			return errRootBenchmarks
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		pkgs = append(pkgs, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error discovering benchmark packages: %w", err)
	}
	return pkgs, nil
}

// declaresBenchmark reports whether any of the named files in dir declares a benchmark
func declaresBenchmark(dir string, files []string) (bool, error) {
	fset := token.NewFileSet()
	for _, name := range files {
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			return false, fmt.Errorf("error reading %s: %w", path, err)
		}
		// Parsing every test file would dominate the run. The filter is the bare name, since a comment
		// can separate it from "func".
		if !bytes.Contains(src, []byte("Benchmark")) {
			continue
		}
		f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			return false, fmt.Errorf("error parsing %s: %w", path, err)
		}
		for _, decl := range f.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && isBenchmark(fn) {
				return true, nil
			}
		}
	}
	return false, nil
}

// isBenchmark applies go test's rule from cmd/go/internal/load: a top-level function with a
// benchmark's name taking a single *B, matched by type name since the testing import can be renamed
func isBenchmark(fn *ast.FuncDecl) bool {
	if fn.Recv != nil || !benchmarkFuncName(fn.Name.Name) {
		return false
	}
	params := fn.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 || (fn.Type.Results != nil && len(fn.Type.Results.List) > 0) {
		return false
	}
	ptr, ok := params[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	switch x := ptr.X.(type) {
	case *ast.Ident:
		return x.Name == "B"
	case *ast.SelectorExpr:
		return x.Sel.Name == "B"
	}
	return false
}

// benchmarkFuncName reports whether name is Benchmark alone or followed by anything but a lower-case
// letter, as go test requires
func benchmarkFuncName(name string) bool {
	suffix, ok := strings.CutPrefix(name, "Benchmark")
	if !ok {
		return false
	}
	r, _ := utf8.DecodeRuneInString(suffix)
	return !unicode.IsLower(r)
}
