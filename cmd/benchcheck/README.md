# benchcheck

`benchcheck` gates benchmark allocations against a checked-in baseline, so that allocation counts in
GoCryptoTrader ratchet downwards over time instead of drifting up unnoticed.

## Usage

```bash
make bench                      # check the current tree against benchmarks/baseline.json
make bench_update               # fold the current measurements back into the baseline
make bench_pkg PKG=./currency/  # measure one package with the gate's exact flags
make bench_trend                # long-run measurements appended to the ns/op history
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
| `allocs/op` | Reproducible and machine independent | Hard gate; an increase in the median fails, unless the entry carries a tolerance |
| `B/op` | Near-reproducible | Hard gate with a 1% default tolerance |
| `ns/op` | Depends on the host and its current load | **Never gated**; recorded as an advisory `ns_hint` and tracked in the scheduled history |

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

## Files

| Path | Purpose |
| --- | --- |
| `benchmarks/baseline.json` | The budgets. Checked in, so widening one is a visible line in a PR diff that a reviewer has to approve |
| `benchmarks/packages.txt` | Which packages are benchmarked, and which are `gated` |
| `scripts/bench_history.sh` | Fetches and publishes the ns/op history branch |
| `scripts/bench_history_test.sh` | Exercises that script against a throwaway repository; run by `misc_checks.sh` |

A package marked `gated` requires every one of its benchmarks to have a baseline entry, so a new
`func Benchmark` fails CI until it is seeded. Benchmarks in unmarked packages are still checked
whenever they already have an entry; the marker only makes that coverage mandatory.

One benchmark escapes that: one that calls `b.Skip` emits no result line at all without `-v`, so a
new skipping benchmark is invisible here rather than untracked. A skip is still caught the moment
the benchmark has an entry — it then reads as missing — so this only hides one that never had one.

`packages.txt` is the source of truth for coverage: which packages are gated, which are only checked
against an entry they already have, and which are excluded along with the reason why.

## Updating the baseline

`-update` never deletes. A run that selected a subset of benchmarks looks, from the output alone,
exactly like a full run in which the rest disappeared, so deleting on that evidence would destroy
the baseline. Entries with no matching result are reported and kept.

`make bench_update` passes `-prune`, which does delete them. That is safe there and only there,
because `BENCH_FLAGS` uses `-bench .` and therefore matches everything. Pass `-prune` by hand only
when the same is true of your own invocation.

## Escape hatches

Both require a `reason`, enforced by `Baseline.Validate`, and a tolerance must be in `[0, 1)`:

```jsonc
{
  "dispatch.BenchmarkSubscribe": {
    "allocs": 1,
    "bytes": 156,
    "bytes_tolerance": 0.1,           // widen one metric's budget
    "reason": "worker goroutines make B/op scheduler dependent; observed 151-161 across runs"
  },
  "some/pkg.BenchmarkExample": {
    "ignore": true,                   // drop the benchmark from the gate entirely
    "reason": "why no tolerance can work here"
  }
}
```

Prefer a tolerance to `ignore`: an ignored benchmark still costs CI time but tells nobody anything.
Note that `ignore` only skips *comparison* — the benchmark still runs, so one that is unsafe to
execute must be excluded from `packages.txt` instead.

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
reproducible. The `bench_trend` target uses `-count 15 -benchtime 1s` instead, because tight
confidence intervals do matter for the `ns/op` history and a long runtime is affordable on a
scheduled run.

Both counts are **odd, and must stay odd**. The compared value is the median, and an even count has
no middle sample, so `median` falls back to the lower of the two middles. That is biased, and biased
asymmetrically for a gate that fails on improvement as well as regression: at six samples three low
ones move the value while four high ones are needed to.

The count is also passed to benchcheck as `-samples`, which rejects any benchmark that did not
report exactly that many measurements. A truncated run, or a result line that failed to parse, would
otherwise shift which sample sits in the middle with nothing to notice it. `bench_trend` pairs
`-samples` with `-warn`, which drops the offending records and appends the rest rather than losing a
ten minute measurement to one short benchmark.

Keep `-cpu` single valued. benchcheck strips the `-N` suffix Go appends to result names, so `-cpu
1,4` files two populations under one key and `-samples` cannot tell that from a single clean run.

## The ns/op history

`make bench_trend` appends one JSON line per benchmark to a series file. In CI a scheduled job
publishes it to the `benchmarks-history` branch, which is append-only: `bench_history.sh` refuses to
publish a series that does not extend what is already there, so a run built from a stale fetch
cannot silently drop another run's records.
