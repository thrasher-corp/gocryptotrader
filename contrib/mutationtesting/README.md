# Run the mutation-testing pilot

From the repository root, run:

```console
make mutation_math
```

Use the Go version required by `go.mod`. The command downloads and runs Gremlins
v0.6.0, then tests `common/math` with the default decimal backend and with
`udecimal_on`. It runs two mutation workers. Each test invocation uses `-cpu 1`.
The target is optional and is not part of `make check` or required CI.

Read the individual mutant results as well as the summary. Investigate surviving
mutants, uncovered code, timeouts and nonviable mutations before drawing a
conclusion about the tests. A successful exit and 100% efficacy do not establish
that every mutation was tested successfully.

The timeout coefficient is 20 because the default coefficient of 3 produced
66 timeouts during the initial Go 1.27 pilot. With the larger allowance, the
same default-backend run killed 119 mutants without timeouts. Seven mutations
were in code not covered by that backend. The `udecimal_on` run killed all
126 mutants without timeouts. Each run took about 30 seconds on the development
host. These measurements are observations, not required scores or runtime limits.

Gremlins creates temporary working copies before applying mutations. See its
[working-directory implementation](https://github.com/go-gremlins/gremlins/blob/e05b1d47b8c55748e50abc28ff6b132c536bacca/internal/engine/workdir/workdir.go)
and [test executor](https://github.com/go-gremlins/gremlins/blob/e05b1d47b8c55748e50abc28ff6b132c536bacca/internal/engine/executor.go).
The executor derives each mutation's timeout from the coverage run duration.
Keep the package scope small and check results again after changing the tool,
toolchain, backend or worker count.

This pilot checks arithmetic and conditional mutations. Lifecycle tests must
still prove cancellation, worker completion and restart ownership directly.
The pilot does not establish those properties.
