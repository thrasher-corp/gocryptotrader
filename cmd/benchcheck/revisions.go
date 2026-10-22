package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	errCompareModes     = errors.New("-compare cannot be combined with -update, -prune or -list")
	errBadRevision      = errors.New("revision to compare must not start with a dash")
	errUnsupportedFlag  = errors.New("go test flag has no test binary equivalent -compare knows")
	errMissingFlagValue = errors.New("go test flag is missing its value")
	errCountMismatch    = errors.New("-count in the go test flags must equal -samples")
	errArchiveEntry     = errors.New("git archive produced an entry -compare cannot extract")
	errProcessTimeout   = errors.New("benchmark process ran past its time limit")
)

// processTimeout bounds each test binary's run, since go test's -timeout does not cover benchmarks and
// a hung one would otherwise hold the job until the workflow's limit
const processTimeout = 10 * time.Minute

// revision is one side of a comparison: a checkout of the module, its test binaries and the samples
// they reported
type revision struct {
	label    string
	root     string
	commit   string
	binaries []string
	samples  map[string]*Samples
	skipped  map[string]bool
	cpu      string
	output   bytes.Buffer
}

// comparison is what measuring two revisions found
type comparison struct {
	base, head *revision
	// dirty notes uncommitted changes measured at the head
	dirty   bool
	samples int
	// predatesList is set when the base has no package list, which leaves nothing to compare
	predatesList bool
	packages     []string
	onlyHead     []string
	onlyBase     []string
	rows         []comparisonRow
	notCompared  map[string]string
	warnings     []string
}

// runComparison measures the revision o.compare names and the working tree, and reports their timings
// and allocations side by side. Every sample is a process of its own, since a process can settle into
// a faster or slower state for its whole life, and the rounds alternate which revision runs first, so
// drift on the machine falls on both.
func runComparison(out io.Writer, o *options) error {
	if o.update || o.prune || o.list {
		return errCompareModes
	}
	if o.samples <= 0 || o.samples%2 == 0 {
		return fmt.Errorf("%w: got %d", errInvalidSamples, o.samples)
	}
	flags, err := binaryFlags(o.args, o.samples)
	if err != nil {
		return err
	}
	pkgs, err := LoadPackages(o.packagesPath)
	if err != nil {
		return err
	}
	baseline, err := LoadBaseline(o.baselinePath, false)
	if err != nil {
		return err
	}
	if err := baseline.Validate(); err != nil {
		return err
	}
	progress := o.progress
	if progress == nil {
		progress = io.Discard
	}
	// Test binaries run from their package directories, where relative paths would no longer resolve
	outDir, err := filepath.Abs(o.out)
	if err != nil {
		return err
	}
	root, err := filepath.Abs(o.root)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("error creating %s: %w", outDir, err)
	}
	// A signal would otherwise end the process before it removes the work directory
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// Under the output directory rather than the system's, which can be a small in-memory filesystem
	work, err := os.MkdirTemp(outDir, ".work-")
	if err != nil {
		return fmt.Errorf("error creating a work directory: %w", err)
	}
	defer os.RemoveAll(work)

	c := &comparison{
		base:    newRevision("base", filepath.Join(work, "base")),
		head:    newRevision("head", root),
		samples: o.samples,
	}
	if c.base.commit, err = extractRevision(ctx, root, o.compare, c.base.root); err != nil {
		return err
	}
	if c.head.commit, err = gitOutput(ctx, root, "rev-parse", "HEAD"); err != nil {
		return err
	}
	status, err := gitOutput(ctx, root, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return err
	}
	c.dirty = status != ""

	basePkgs, err := LoadPackages(filepath.Join(c.base.root, "benchmarks", "packages.txt"))
	switch {
	case errors.Is(err, errNoPackageList):
		c.predatesList = true
		return c.write(out, outDir)
	case err != nil:
		return fmt.Errorf("base: %w", err)
	}
	c.packages, c.onlyHead, c.onlyBase = sharedPackages(pkgs.List, basePkgs.List)
	fmt.Fprintf(progress, "benchcheck: comparing %s with %s across %d packages\n", shortCommit(c.base.commit), shortCommit(c.head.commit), len(c.packages))

	// Building everything first keeps the compiler off the machine while it measures
	for _, r := range []*revision{c.base, c.head} {
		if err := r.build(ctx, c.packages, filepath.Join(work, "bin", r.label)); err != nil {
			return err
		}
	}
	fmt.Fprintln(progress, "benchcheck: built both revisions' test binaries")
	for round := range o.samples {
		order := roundOrder(round, c.base, c.head)
		for i, pkg := range c.packages {
			for _, r := range order {
				if err := r.measure(ctx, i, pkg, flags); err != nil {
					// The output so far shows where it stopped
					return errors.Join(err, c.writeOutput(outDir))
				}
			}
		}
		fmt.Fprintf(progress, "benchcheck: round %d of %d measured\n", round+1, o.samples)
	}
	if err := c.compare(baseline); err != nil {
		return errors.Join(err, c.writeOutput(outDir))
	}
	return c.write(out, outDir)
}

// roundOrder returns the order the revisions run in during a round, swapped every round so that neither
// always runs first
func roundOrder(round int, base, head *revision) []*revision {
	if round%2 == 1 {
		return []*revision{head, base}
	}
	return []*revision{base, head}
}

func newRevision(label, root string) *revision {
	return &revision{label: label, root: root, samples: make(map[string]*Samples), skipped: make(map[string]bool)}
}

// binaryFlags translates the go test flags in BENCH_FLAGS into a test binary's, for one sample per
// run. A flag it does not know is refused rather than dropped, so the comparison cannot drift from what
// the gate measures without anyone noticing.
func binaryFlags(goTestFlags []string, samples int) ([]string, error) {
	// As go test passes it, so that a benchmark calling os.Exit(0) fails rather than quietly ending its
	// package's run
	flags := []string{"-test.paniconexit0"}
	for i := 0; i < len(goTestFlags); i++ {
		arg := goTestFlags[i]
		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if !strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("%w: %q", errUnsupportedFlag, arg)
		}
		switch name {
		case "benchmem", "v":
			flags = append(flags, "-test."+strings.TrimLeft(arg, "-"))
			continue
		case "run", "bench", "benchtime", "cpu", "count", "p":
		default:
			return nil, fmt.Errorf("%w: %q", errUnsupportedFlag, arg)
		}
		if !hasValue {
			if i+1 == len(goTestFlags) {
				return nil, fmt.Errorf("%w: %q", errMissingFlagValue, arg)
			}
			i++
			value = goTestFlags[i]
		}
		switch name {
		case "count":
			if value != strconv.Itoa(samples) {
				return nil, fmt.Errorf("%w: -count %s, -samples %d", errCountMismatch, value, samples)
			}
			flags = append(flags, "-test.count=1")
		case "p":
			// Spreads packages across go test's processes, which the rounds run one at a time anyway
		default:
			flags = append(flags, "-test."+name+"="+value)
		}
	}
	return flags, nil
}

// sharedPackages splits the head's package list into those the base lists too, in the head's order, and
// those only one of them lists
func sharedPackages(head, base []string) (shared, onlyHead, onlyBase []string) {
	for _, pkg := range head {
		if slices.Contains(base, pkg) {
			shared = append(shared, pkg)
		} else {
			onlyHead = append(onlyHead, pkg)
		}
	}
	for _, pkg := range base {
		if !slices.Contains(head, pkg) {
			onlyBase = append(onlyBase, pkg)
		}
	}
	return shared, onlyHead, onlyBase
}

// gitOutput runs git in dir and returns its trimmed output
func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// gitEnv is the environment without the GIT_ variables a git hook exports, which would point git at the
// hook's repository rather than the one being compared
func gitEnv() []string {
	return slices.DeleteFunc(os.Environ(), func(kv string) bool { return strings.HasPrefix(kv, "GIT_") })
}

// extractRevision writes the tree of rev, as git archive exports it, under dir and returns rev's commit
func extractRevision(ctx context.Context, repo, rev, dir string) (string, error) {
	// git would read it as an option
	if strings.HasPrefix(rev, "-") {
		return "", fmt.Errorf("%w: %q", errBadRevision, rev)
	}
	commit, err := gitOutput(ctx, repo, "rev-parse", "--verify", rev+"^{commit}")
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "archive", "--format=tar", commit)
	cmd.Dir = repo
	cmd.Env = gitEnv()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	archive, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error reading git archive: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting git archive: %w", err)
	}
	if err := extractTar(archive, dir); err != nil {
		cancel()
		_ = cmd.Wait()
		return "", err
	}
	// The archive can be padded past its end marker, and git must finish writing before Wait closes the pipe
	if _, err := io.Copy(io.Discard, archive); err != nil {
		cancel()
		_ = cmd.Wait()
		return "", fmt.Errorf("error reading git archive: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("error running git archive %s: %w: %s", commit, err, strings.TrimSpace(stderr.String()))
	}
	return commit, nil
}

// extractTar writes a git archive's directories and regular files under dir. git archive emits nothing
// else for a tree without symlinks or submodules, so any other entry fails rather than going missing.
func extractTar(r io.Reader, dir string) error {
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("error reading git archive: %w", err)
		}
		// git archive's record of the commit it exported
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if !filepath.IsLocal(h.Name) {
			return fmt.Errorf("%w: %q leaves the checkout", errArchiveEntry, h.Name)
		}
		path := filepath.Join(dir, filepath.FromSlash(h.Name))
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, 0o750)
		case tar.TypeReg:
			err = writeArchiveFile(path, tr, h)
		default:
			return fmt.Errorf("%w: %q has type %q", errArchiveEntry, h.Name, h.Typeflag)
		}
		if err != nil {
			return fmt.Errorf("error extracting %s: %w", h.Name, err)
		}
	}
}

func writeArchiveFile(path string, r io.Reader, h *tar.Header) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, h.FileInfo().Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.CopyN(f, r, h.Size); err != nil {
		return errors.Join(err, f.Close())
	}
	return f.Close()
}

// build compiles each package's test binary. -trimpath leaves the checkout's location out of the build,
// so the packages both revisions share come from the build cache as identical binaries, and
// GOTOOLCHAIN=local builds both with the running toolchain, since a go.mod naming another release could
// otherwise switch one of them and the comparison would measure the toolchain.
func (r *revision) build(ctx context.Context, pkgs []string, dir string) error {
	r.binaries = make([]string, len(pkgs))
	for i, pkg := range pkgs {
		binary := filepath.Join(dir, strconv.Itoa(i)+".test")
		if runtime.GOOS == "windows" {
			binary += ".exe"
		}
		cmd := exec.CommandContext(ctx, "go", "test", "-c", "-trimpath", "-o", binary, "./"+pkg+"/")
		cmd.Dir = r.root
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("error building the %s test binary for %s: %w\n%s", r.label, pkg, err, out)
		}
		r.binaries[i] = binary
	}
	return nil
}

// measure runs one round of a package's benchmarks and folds in what the process reported
func (r *revision) measure(ctx context.Context, i int, pkg string, flags []string) error {
	output, err := runTestBinary(ctx, r.binaries[i], filepath.Join(r.root, filepath.FromSlash(pkg)), flags, processTimeout)
	r.output.Write(output)
	if err != nil {
		return fmt.Errorf("%s %s: %w", r.label, pkg, err)
	}
	run, err := Parse(bytes.NewReader(output))
	if err != nil {
		return fmt.Errorf("%s %s: %w", r.label, pkg, err)
	}
	for key, s := range run.Samples {
		merged, ok := r.samples[key]
		if !ok {
			merged = &Samples{Pkg: s.Pkg, Name: s.Name}
			r.samples[key] = merged
		}
		merged.NS = append(merged.NS, s.NS...)
		merged.Bytes = append(merged.Bytes, s.Bytes...)
		merged.Allocs = append(merged.Allocs, s.Allocs...)
	}
	for key := range run.Skipped {
		r.skipped[key] = true
	}
	if r.cpu == "" {
		r.cpu = cpuModel(output)
	}
	return nil
}

// runTestBinary runs a test binary from dir, as go test does, killing it once timeout passes
func runTestBinary(ctx context.Context, binary, dir string, flags []string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, flags...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	// A process the benchmark started can hold the output open after the kill
	cmd.WaitDelay = 10 * time.Second
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return out.Bytes(), fmt.Errorf("%w: %s", errProcessTimeout, timeout)
	}
	if err != nil {
		return out.Bytes(), fmt.Errorf("benchmark process failed: %w\n%s", err, out.Bytes())
	}
	return out.Bytes(), nil
}

// cpuModel returns the processor named by go test's cpu: header, or ""
func cpuModel(output []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(output))
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "cpu:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
