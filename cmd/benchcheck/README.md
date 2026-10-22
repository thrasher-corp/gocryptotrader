# benchcheck

`benchcheck` gates benchmark allocations against a checked-in baseline, so that allocation counts in
GoCryptoTrader cannot drift up unnoticed.

## Usage

```bash
make bench                                # check the current tree against benchmarks/baseline.json
make bench_update                         # fold the current measurements back in
make bench_apply BENCH_OUT=f              # fold in output measured elsewhere, such as CI's bench-output artifact
make bench_pkg PKG=./currency/            # measure one package with the gate's exact flags
make bench_compare BASE=upstream/master   # compare timings and allocations with another commit
```

`make bench` fails when a benchmark allocates **more** than its recorded budget. One that allocates
**less** passes and is reported as `stale`: write it back with `make bench_update`, which lowers the
budget so a later regression cannot quietly take the improvement back.

Each budget is compared against the **median** of the samples, not any single one, so one unlucky
run does not trip the gate. Per-entry tolerances are applied on top of that median, and include
their edge: a measurement exactly a tolerance away from its budget is within it. A tolerance is
taken exactly as the baseline spells it, so `0.15` means 15% rather than the binary value a hair
below it.

## What is gated, and what is not

| Metric | Reproducibility | Treatment |
| --- | --- | --- |
| `allocs/op` | Reproducible on one platform and toolchain | Hard gate; any increase in the median fails, unless the entry carries a tolerance |
| `B/op` | Near-reproducible | Hard gate with a 1% default tolerance |
| `ns/op` | Depends on the host and its current load | **Never gated or recorded**; compared with the base in an advisory report, see [Timings](#timings) |

Timings are not gated because they move substantially between runs on a shared CI runner, in a way
that has nothing to do with the code under test. A gate on `ns/op` would either flap constantly or
be set so loose it catches nothing. Allocation counts do not have that problem: `allocs/op` is
identical across every sample for all but a handful of the benchmarks measured. That is what makes
them worth gating and timings not.

"Reproducible" is not the same as "constant", but most movement between runs is a defect in the
benchmark rather than a fact about the code. A loop that stages work on a queue and never waits for
it leaves the amount the worker happened to finish to chance; a loop that accumulates state measures
a structure that grows as the run goes on, and pays its amortised growth wherever the run stopped.
Both move `B/op`. Waiting for the work, or releasing what the iteration acquired, removes the
movement rather than accommodating it, and is what nearly every entry here does.

Reach for a per-entry tolerance only once the movement is in the code under test, and size it to the
range actually observed rather than to a round number: a tolerance is a fraction of the budget, so on
a large budget an innocuous-looking `0.001` is worth dozens of allocations.

One caveat on reading `allocs/op`: the testing package reports it as `MemAllocs / N` in **integer**
arithmetic, so it is a truncated average rather than a count. A benchmark allocating once every
tenth iteration reports `0`, and one allocating on nine iterations in ten still reports `0`. A zero
in the baseline therefore means "under one allocation per operation", not "allocation free", and a
regression of less than a whole allocation per operation can leave the number unchanged. `B/op` is
the finer-grained of the two gated metrics for exactly that reason, and is what catches sub-integer
movement — worth remembering before widening a `bytes_tolerance`.

## The measurement profile

Allocation counts are reproducible on one platform and toolchain, not across them: escape analysis
changes between Go releases, and code paths can differ between platforms. The budgets are therefore
recorded against one profile, the one the `benchmarks` workflow measures:

- `linux/amd64`
- the Go version `go.mod` names, which every workflow installs
- no build tags or `GOEXPERIMENT` settings, so the default backends, such as `encoding/json/v2` and
  shopspring's decimal, and the runtime's default garbage collection settings
- `BENCH_ENV` and `BENCH_FLAGS` in the Makefile

`benchcheck` refuses to record budgets from output produced on any other platform, and a check run
elsewhere says so above its findings. To update the baseline from another machine, push the change
and let CI measure it: every run uploads its raw output as the `bench-output` artifact, whether or not
the check passed, and `make bench_apply BENCH_OUT=path/to/bench-output.txt` folds it in.

A Go upgrade, a dependency bump or a backend change can move many budgets at once. Refresh them on the
profile in the same change, with `make bench_update` or from CI's output, and check that every move
the diff shows is one the change explains.

## Files

| Path | Purpose |
| --- | --- |
| `benchmarks/baseline.json` | The budgets. Checked in, so widening one is a visible line in a PR diff that a reviewer has to approve |
| `benchmarks/packages.txt` | Which packages are benchmarked, and which are excluded and why |

## Which packages are benchmarked

Every benchmark in a package listed in `packages.txt` must have a baseline entry, so a new
`func Benchmark` fails CI until it is seeded. A package whose benchmarks must not run, for instance
because its `TestMain` reaches the network, is listed with the `excluded` marker and a reason
instead.

The list cannot fall behind the code. `benchcheck -list`, which `make bench` and `make bench_update` run first,
finds every package whose test files declare a benchmark, applying `go test`'s own rules for the
profile's build constraints, and fails while any of them is neither listed nor excluded, or while
the list names a package that declares none.

A skip cannot hide a benchmark either, unless printed text disguises its record, as described below. `BENCH_FLAGS` runs go test with `-v`, without which a
benchmark that calls `b.Skip` before measuring anything prints nothing at all. One that skips after
go test's first round prints a result of zero iterations instead, in either mode, and `benchcheck`
reads that as a skip too. Each skip is a `skipped` finding, whether or not the benchmark has a
budget, and the budgets beneath a skipped benchmark are not reported missing as well: it still
exists, so the cure is to stop it skipping rather than to delete its budgets. Text a benchmark prints
without a newline just before go test's skip record runs into it, so `benchcheck` reads a record
that ends a line as well as one that starts it, and reads a benchmark that `-v` announced and that
then reported nothing at all, in a package that ran to its `ok` line, as a skip too.

Printed text can still disguise a skip record. `benchcheck` does not read a record run into text
laid out like `b.Log` output, indented by eight spaces or by four before a `file.go:12:` prefix,
since a benchmark can log exactly that text itself and plain output cannot tell the two apart. A
benchmark that has measured a sub-benchmark and then skips after printing such text therefore shows
no `skipped` finding: a sub-benchmark the skip left unmeasured is still reported `missing` if it has
a budget, and not reported at all if it has none.

## Updating the baseline

An update never loosens a budget quietly. A new benchmark gains an entry, and an existing one has a
metric raised only where the check reports a regression, so every increase in the diff of
`make bench_update` is a finding a reviewer has seen, and drift too small to report on any one run
accumulates against a fixed budget instead of being absorbed by each update in turn. A metric is
lowered once the run measured more than half its tolerance below the budget, or on any decrease
where the tolerance is zero. That keeps run-to-run noise out of the diff, and leaves every
measurement at least half a tolerance clear of reading as stale.

`-update` deletes nothing by itself. A run that selected a subset of benchmarks looks, from the
output alone, exactly like a full run in which the rest disappeared, so entries with no matching
result are reported and kept.

`make bench_update` and `make bench_apply` pass `-prune`, which deletes them: entries whose
benchmark was renamed or deleted, and entries whose whole package has left the list. Entries a skip
accounts for are kept. `benchcheck` refuses to prune unless every listed package reported, its
output ran to go test's closing `ok` line, and it came from `go test -v`, so running either target
against a subset of the packages, on output from a run that stopped early, or on output that could
not have shown a skip, fails rather than deleting budgets. It also refuses to delete an entry that
text in go test's skip record layout names, unless a skip `benchcheck` read already accounts for it:
output a benchmark prints can hide its own record, so the benchmark may have skipped. Remove such an
entry by hand once its benchmark is gone.

## Escape hatches

Every override requires a `reason`, enforced by `Baseline.Validate`, and a tolerance must be in
`[0, 1)`:

```jsonc
{
  "some/pkg.BenchmarkScheduled": {
    "allocs": 1,
    "bytes": 156,
    "bytes_tolerance": 0.1,           // widen one metric's budget
    "reason": "worker goroutines make B/op scheduler dependent; observed 151-161 across runs"
  },
  "some/pkg.BenchmarkFileIO": {
    "allocs": 37991,
    "bytes": 9383689,
    "allocs_tolerance": 0.004,        // allocs/op otherwise has to match exactly
    "reason": "reads a file; the median moves by tens of allocations between runs"
  },
  "some/pkg.BenchmarkExact": {
    "allocs": 2,
    "bytes": 64,
    "bytes_tolerance": 0,             // an explicit zero asks for an exact B/op match
    "reason": "fixed-size output; any change in B/op is a real change"
  },
  "some/pkg.BenchmarkExample": {
    "ignore": true,                   // drop the benchmark from the gate entirely
    "reason": "why no tolerance can work here"
  }
}
```

Prefer a tolerance to `ignore`: an ignored benchmark still costs CI time but tells nobody anything.
Note that `ignore` only skips *comparison* — the benchmark still runs, so one that is unsafe to
execute must be excluded in `packages.txt` instead. An ignored entry also exempts its benchmark from
the skip check.

`benchcheck` accepts only the field names shown here, spelt exactly as `make bench_update` writes
them. The JSON decoder matches field names regardless of case, so a variant spelling could otherwise
replace a budget while its diff read as an addition, and a misspelt override would do nothing at all.

## Flag settings

`BENCH_FLAGS` in the Makefile is the single definition of how a benchmark is measured, and
`BENCH_ENV` of the environment it runs in; the workflow calls `make` rather than restating them, and
`make bench_pkg` reuses both so an audit of one package matches the gate.

`BENCH_ENV` clears `GOFLAGS`, `GOEXPERIMENT`, `GOGC` and `GOMEMLIMIT`. go test's output records
none of them, so a `GOFLAGS=-tags=sonic_on` left set in a shell would otherwise measure a different
JSON backend under the same `linux/amd64` headers, and `make bench_update` would record it; a low
`GOGC` or a `GOMEMLIMIT` runs the garbage collector more often, which moves allocation counts that
depend on it. benchcheck runs under it too, since `GOFLAGS` can make `go run` exit zero without
running it, as `-exec=/bin/true` and `-n` do. An empty variable does not override the go env file,
so the measuring targets and `make bench_apply` refuse to run while `go env -w` has set `GOFLAGS` or
`GOEXPERIMENT`, which `go env -u GOFLAGS GOEXPERIMENT` removes. The measuring targets also refuse a
Go release other than the major and minor version `go.mod` names, which a newer local toolchain or
`GOTOOLCHAIN` can select.

It pins `-cpu 4` and `-p 1` so that CI and a developer's machine measure the same thing whatever the
host's core count: `-cpu` fixes GOMAXPROCS inside each test binary, and `-p` stops several package
binaries competing for the same cores. Without both, scheduler-dependent benchmarks record different
values on each machine and the baseline is only valid on the one that produced it.

`-count 7 -benchtime 100ms` is deliberately cheap, a couple of minutes for the whole repository. A
longer benchtime buys the allocation gate nothing, because `allocs/op` is per-iteration and
reproducible.

The count is **odd, and must stay odd**. The compared value is the median, and an even count has no
middle sample: averaging the two middles compares a value the run never measured, and picking either
one biases the comparison towards that side.
`benchcheck` refuses an even count rather than choosing.

The count is also passed to benchcheck as `-samples`, which is required and rejects any benchmark
that did not report exactly that many measurements. A truncated run, or a result line that failed to
parse, would otherwise change which sample sits in the middle with nothing to notice it.

Keep `-cpu` a single value above 1. benchcheck strips the `-N` suffix go test appends to result
names, and refuses output in which results carry different ones, as `-cpu 1,4` produces. In output
without `-v` that check is also what tells the suffix from a name ending in digits: under `-cpu 1`,
which appends no suffix, a sub-benchmark named `size-1` would otherwise lose its `-1`.

`-v` is there for its skip reports, described under
[Which packages are benchmarked](#which-packages-are-benchmarked). It also announces each benchmark
by name before its results, which settles exactly where each name ends, and shows a result under any
other name to be output a benchmark printed, which fails the run.

## Timings

`ns/op` is neither gated nor stored. A stored timing says nothing about a run on another machine, and
two runs of the same code on one machine still report different timings. What a timing can answer is
whether a change made something faster or slower, and that takes both revisions measured on one
machine at one time:

```bash
make bench_compare BASE=upstream/master
```

It builds the test binaries of `BASE` and of the working tree first, then measures them in
`BENCH_SAMPLES` rounds of one sample each, swapping which revision runs first every round so that
drift in the machine's load falls on both. Each sample is a process of its own: some benchmarks
settle into a faster or a slower state for the life of a process, so samples from one process agree
with each other more closely than with another process of the same binary, and a comparison of two
single processes reads that difference as a change. `BENCH_FLAGS` is translated into the test
binaries' flags, and a flag `benchcheck` cannot translate is refused rather than dropped.

The report puts each benchmark's medians at both revisions, `ns/op`, `B/op` and `allocs/op`, on one
row, and leads with the benchmarks that moved: allocations whose medians differ by more than the
baseline's tolerance, and timings that changed significantly by 5% or more. A timing's significance
is benchstat's default test, a Mann-Whitney U test at p < 0.05, and `~` marks a timing without a
significant change. With this many benchmarks a few pass that test by chance alone, so a small
timing change is a reason to look again rather than a result. The report is saved as `summary.md`,
beside each revision's raw output for
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) to read.

Both revisions are built with the running toolchain and `-trimpath`, which shares the build cache
between them and builds a package whose code and dependencies are unchanged into the same binary at
both, so a Go upgrade's own effect does not show. A benchmark whose body changed measures different
work at each revision, and a base without `benchmarks/packages.txt` has nothing to compare.

Pull requests get the same report from the `benchmark comparison` workflow, which compares the merge
commit with the base it merges into and writes the report to the job's summary. The workflow is
advisory: it is not a required check, holds no write permission, and fails only when it cannot
measure.

A long-run history is only worth storing once something reads it, and once it records enough of the
environment - processor, toolchain and runner image - to tell a change in the code from a change in
the machine.
