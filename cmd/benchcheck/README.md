# benchcheck

`benchcheck` gates benchmark allocations against a checked-in baseline, so that allocation counts in
GoCryptoTrader ratchet downwards over time instead of drifting up unnoticed.

## Usage

```bash
make bench                      # check the current tree against benchmarks/baseline.json
make bench_update               # fold the current measurements back in
make bench_apply BENCH_OUT=f    # fold in output measured elsewhere, such as CI's bench-output artifact
make bench_pkg PKG=./currency/  # measure one package with the gate's exact flags
```

`make bench` fails when a benchmark allocates **more** than its recorded budget, and equally when it
allocates **less**. The second case is the point of the tool: an improvement is only permanent once
it has been written back with `make bench_update`, which makes every optimisation a new floor and
turns the baseline into a ratchet rather than a ceiling that only ever moves up.

Each budget is compared against the **median** of the samples, not any single one, so one unlucky
run does not trip the gate. Per-entry tolerances are applied on top of that median.

## What is gated, and what is not

| Metric | Reproducibility | Treatment |
| --- | --- | --- |
| `allocs/op` | Reproducible on one platform and toolchain | Hard gate; any change in the median fails, unless the entry carries a tolerance |
| `B/op` | Near-reproducible | Hard gate with a 1% default tolerance |
| `ns/op` | Depends on the host and its current load | **Never gated or recorded**; it stays in the raw output CI uploads |

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
regression of less than a whole allocation per operation cannot move the number at all. `B/op` is
the finer-grained of the two gated metrics for exactly that reason, and is what catches sub-integer
movement — worth remembering before widening a `bytes_tolerance`.

## The measurement profile

Allocation counts are reproducible on one platform and toolchain, not across them: escape analysis
changes between Go releases, and code paths can differ between platforms. The budgets are therefore
recorded against one profile, the one the `benchmarks` workflow measures:

- `linux/amd64`
- the Go version the workflow installs, which matches the other workflows
- no build tags, so the default backends, such as `encoding/json/v2` and shopspring's decimal
- `BENCH_FLAGS` in the Makefile

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

Skipping is the one way a benchmark escapes. One that calls `b.Skip` emits no result line without
`-v`, so it is invisible here rather than untracked. A skip is still caught once the benchmark has
an entry, which then reads as missing, and a listed package whose benchmarks all skip is reported as
reporting nothing, so this only hides a newly added benchmark that skips from the start.

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
benchmark was renamed or deleted, and entries whose whole package has left the list. `benchcheck`
refuses to prune unless every listed package reported and its output ran to go test's closing `ok`
line, so running either target against a subset of the packages, or on output from a run that
stopped early, fails rather than deleting budgets.

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
execute must be excluded in `packages.txt` instead.

## Flag settings

`BENCH_FLAGS` in the Makefile is the single definition of how a benchmark is measured; the workflow
calls `make` rather than restating it, and `make bench_pkg` reuses it so an audit of one package
matches the gate.

It pins `-cpu 4` and `-p 1` so that CI and a developer's machine measure the same thing whatever the
host's core count: `-cpu` fixes GOMAXPROCS inside each test binary, and `-p` stops several package
binaries competing for the same cores. Without both, scheduler-dependent benchmarks record different
values on each machine and the baseline is only valid on the one that produced it.

`-count 7 -benchtime 100ms` is deliberately cheap, a couple of minutes for the whole repository. A
longer benchtime buys the allocation gate nothing, because `allocs/op` is per-iteration and
reproducible.

The count is **odd, and must stay odd**. The compared value is the median, and an even count has no
middle sample: averaging the two middles compares a value the run never measured, and picking either
one biases the gate, which fails on improvements as well as regressions, towards that side.
`benchcheck` refuses an even count rather than choosing.

The count is also passed to benchcheck as `-samples`, which is required and rejects any benchmark
that did not report exactly that many measurements. A truncated run, or a result line that failed to
parse, would otherwise change which sample sits in the middle with nothing to notice it.

Keep `-cpu` single valued. benchcheck strips the `-N` suffix Go appends to result names, so
`-cpu 1,4` files two populations under one key; `-samples` then sees twice the count and rejects the
run.

## Timings

`ns/op` is neither gated nor stored. The raw output each CI run uploads carries it, together with the
`cpu:` line naming the runner's processor, and stays available for the artifact retention period.

A question about whether a change made something slower is best answered on one machine: run the
benchmarks at the base and the head, interleaved, and compare them with
[benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat). A long-run history is only worth
storing once something reads it, and once it records enough of the environment - processor,
toolchain and runner image - to tell a change in the code from a change in the machine.
