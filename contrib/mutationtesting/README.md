# Run the mutation-testing pilot

From the repository root, run:

```console
make mutation_math
```

Use the Go version required by `go.mod`. The command downloads and runs Gremlins
v0.6.0, then tests `common/math` with the default decimal backend and with
`udecimal_on`. It runs two mutation workers.
The target is optional and is not part of `make check` or required CI.

Do not pass `--test-cpu`. Gremlins v0.6.0 passes it to `go test` as the single
argument `-cpu N`, so the intended package is not tested and every mutant is
reported as killed. See the
[argument construction](https://github.com/go-gremlins/gremlins/blob/e05b1d47b8c55748e50abc28ff6b132c536bacca/internal/engine/executor.go#L231-L233).

Read the individual mutant results as well as the summary. Investigate surviving
mutants, uncovered code, timeouts and nonviable mutations before drawing a
conclusion about the tests. A successful exit and 100% efficacy do not establish
that every mutation was tested successfully.

Gremlins v0.6.0 reports a mutant that fails to compile as killed, not as
nonviable, because `go test` exits with status 1 for both. See its
[exit-code mapping](https://github.com/go-gremlins/gremlins/blob/e05b1d47b8c55748e50abc28ff6b132c536bacca/internal/engine/executor.go#L258-L267).
Apply a surprising kill by hand before relying on it.

The timeout coefficient is 20 because the default coefficient of 3 made every
mutant time out on the development host. With the larger allowance, the
default-backend run killed 102 mutants, 17 survived and seven were in code not
covered by that backend. The `udecimal_on` run killed 109 mutants and 17
survived. These measurements are observations, not required scores or runtime
limits.

Gremlins creates temporary working copies before applying mutations. See its
[working-directory implementation](https://github.com/go-gremlins/gremlins/blob/e05b1d47b8c55748e50abc28ff6b132c536bacca/internal/engine/workdir/workdir.go)
and [test executor](https://github.com/go-gremlins/gremlins/blob/e05b1d47b8c55748e50abc28ff6b132c536bacca/internal/engine/executor.go).
The executor derives each mutation's timeout from the coverage run duration.
Keep the package scope small and check results again after changing the tool,
toolchain, backend or worker count.

## Mutate the NTP checks

From the repository root, run:

```console
GOFLAGS='-trimpath -run=NTP|QueryNTP' go run github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0 unleash ./engine \
  -E '(^|/)([^n/]|n[^t]|nt[^p]|ntp[^_])[^/]*$' \
  --invert-logical --invert-loopctrl --invert-bitwise --invert-assignments --remove-self-assignments \
  --workers 2 --timeout-coefficient 60
```

The exclusion pattern mutates only `engine/ntp_*.go` files.
`-run` limits each mutant to the NTP tests.
`-trimpath` lets the temporary working copies reuse the checkout's build cache,
so cold builds don't consume each mutant's time allowance.
On the development host, runs without either flag timed out on every mutant.
Platform clock files for other operating systems are reported as not covered.

## Limits

This pilot checks arithmetic and conditional mutations. Lifecycle tests must
still prove cancellation, worker completion and restart ownership directly.
The pilot does not establish those properties.
