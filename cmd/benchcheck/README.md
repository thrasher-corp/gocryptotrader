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
be set so loose it catches nothing. Allocation counts do not have that problem: when the baseline was
seeded, `allocs/op` was identical across every sample for all but one of the benchmarks measured.
That is what makes them worth gating and timings not.

"Reproducible" is not the same as "constant". A few benchmarks move by one or two allocations between
runs; those carry a per-entry tolerance with the observed range recorded as its reason. Benchmarks
that move more than that are excluded outright rather than given a tolerance wide enough to be
meaningless.

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

The repository currently holds 63 benchmarks, of which 50 are in the baseline. The gap is the
packages excluded below.

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
    "bytes": 158,
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
execute must be excluded from `packages.txt` instead. Nothing in the current baseline uses `ignore`.

## Flag settings

`BENCH_FLAGS` in the Makefile is the single definition of how a benchmark is measured; the workflow
calls `make` rather than restating it, and `make bench_pkg` reuses it so an audit of one package
matches the gate.

It pins `-cpu 4` and `-p 1` so that CI and a developer's machine measure the same thing whatever the
host's core count: `-cpu` fixes GOMAXPROCS inside each test binary, and `-p` stops several package
binaries competing for the same cores. Without both, scheduler-dependent benchmarks record different
values on each machine and the baseline is only valid on the one that produced it.

`-count 6 -benchtime 100ms` is deliberately cheap (~45s). A longer benchtime or higher count buys
the allocation gate nothing, because `allocs/op` is per-iteration and reproducible. The
`bench_trend` target uses `-count 15 -benchtime 1s` instead, because tight confidence intervals do
matter for the `ns/op` history and a long runtime is affordable on a scheduled run.

## The ns/op history

`make bench_trend` appends one JSON line per benchmark to a series file. In CI a scheduled job
publishes it to the `benchmarks-history` branch, which is append-only: `bench_history.sh` refuses to
publish a series that does not extend what is already there, so a run built from a stale fetch
cannot silently drop another run's records.

## Known exclusions

Recorded in `benchmarks/packages.txt`, repeated here for visibility:

- `exchanges/kucoin` — its `TestMain` calls `getFirstTradablePairOfAssets` unconditionally, so the
  package fetches tradable pairs from the live KuCoin API even under `-run '^$'`. A rate limit or
  outage would fail the whole gate job, for one benchmark that allocates nothing.
- `log` — the logger writes to stdout during measurement, and those writes interleave into the
  result lines and corrupt them. Route the benchmarks at `io.Discard`, as `BenchmarkNewLogEvent`
  already does, to re-enable the package.
- `exchange/websocket/buffer` — every benchmark fails with `channel buffer is full: failed to relay
  <*orderbook.Depth>` once it runs enough iterations. They pass at `-benchtime 10x` and fail at
  `100ms`, alone or together, so the relay channel simply overruns.
- `exchanges/alert` — `BenchmarkWait` calls `Wait(nil)` and never reads the reply, leaking the
  goroutine that `alert.hold` documents as leakable. Harmless at 100ms, but the scheduled `1s × 15` run
  accumulates millions and can exhaust memory.
- `exchanges/binance` — `BenchmarkWsHandleData` is hermetic and its output is clean, but its counts
  are not reproducible: `allocs/op` moves between 198 and 209 and `B/op` by around 20% even at a
  fixed iteration count, so no tolerance gates anything useful.
