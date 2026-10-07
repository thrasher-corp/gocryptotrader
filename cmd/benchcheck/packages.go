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

// Packages is the benchmarked package list in file order, and the packages excluded from it with
// the reason given for each
type Packages struct {
	List     []string
	Excluded map[string]string
}

// LoadPackages reads the package list. Each line is a package path relative to the module root,
// which is benchmarked and gated, or a path followed by the word "excluded" and a # comment giving
// the reason it is not run. Blank lines and other # comments are ignored. Entries are reduced to
// the form go test reports, via canonicalPackage.
func LoadPackages(path string) (*Packages, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		// Not tolerated: an empty list gates nothing, so a mistyped -packages path would let every
		// benchmark through with no baseline entry and no finding
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
		// Two spellings of one package would otherwise run it twice and double every sample count
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
	// Same reasoning as an absent file: a list that benchmarks nothing gates nothing, and -list then
	// hands go test no packages at all
	if len(pkgs.List) == 0 {
		return nil, fmt.Errorf("%w: %s", errEmptyPackageList, path)
	}
	return pkgs, nil
}

// Covers checks the list against the packages that declare benchmarks. Every one of them must be
// listed or excluded, so a new benchmark package cannot escape the gate by never being added, and
// every listed or excluded package must still declare one, so the list cannot outlive a rename.
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

// canonicalPackage reduces a package list entry to the path go test prints in its pkg: header,
// which is the only form the configured lookups ever see. An entry that resolves to a different
// string runs its benchmarks and gates none of them.
//
// Cleaning handles redundant separators and "." elements. What it cannot resolve is refused: "..."
// covers several packages under one lookup key, and a path leaving the module has no header to
// match. So is anything outside a Go import path's character set, because -list is interpolated
// into go test unquoted and the shell rewrites the rest - "'currency'" loses its quotes,
// "currenc[y]" globs.
func canonicalPackage(entry string) (string, error) {
	root := strings.TrimSuffix(modulePrefix, "/")
	pkg := pathpkg.Clean(entry)
	// Cleaning erases the module boundary, so a module-qualified entry is checked against it before
	// the prefix comes off: ".../gocryptotrader/../../../currency" cleans to a real package and
	// would otherwise be accepted as one, silently naming something the entry never spelled.
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

// importPathRune reports whether r may appear in a Go import path. The set is x/mod/module's
// importPathOK, which is alphanumerics plus "-._~+", together with the element separator. None of
// those carry meaning to a shell, which is the property the caller needs.
func importPathRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '/', r == '.', r == '-', r == '_', r == '~', r == '+':
		return true
	}
	return false
}

// BenchmarkPackages returns, in path order, every package under root, the module root, whose test
// files declare a benchmark, as the relative path go test reports in its pkg: header. Build
// constraints are evaluated for the canonical platform with no extra tags, which is what the gate
// runs. Directories the go command skips are skipped too - testdata, vendor, names starting with
// "." or "_", and nested modules - since `go test ./...` from the root would never run them.
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
		// Most test files declare none, and parsing them all would dominate the run. The filter is
		// the bare name, since a declaration may separate it from "func" with a comment.
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

// isBenchmark applies go test's own rule from cmd/go/internal/load: a top-level function named
// Benchmark, or Benchmark followed by anything but a lower-case letter, taking a single *B. Like go
// test it matches the parameter by its type name, since the testing import may be renamed.
func isBenchmark(fn *ast.FuncDecl) bool {
	suffix, ok := strings.CutPrefix(fn.Name.Name, "Benchmark")
	if !ok || fn.Recv != nil {
		return false
	}
	if r, _ := utf8.DecodeRuneInString(suffix); unicode.IsLower(r) {
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
