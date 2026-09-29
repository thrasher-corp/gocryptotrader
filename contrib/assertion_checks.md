# Check assertion messages

Run the checker and its regression fixtures from a Git checkout:

```console
make assertion_checks
```

Install Git, Make and the Go version required by `go.mod`. The target builds the
checker with that toolchain and checks both decimal backends. Local miscellaneous
checks and CI use this same target through `scripts/misc_checks.sh`.

The checker resolves Testify functions and methods through Go type information.
It checks the actual message argument, including multiline calls, import aliases,
dot imports, method expressions and embedded assertion methods. Constant strings
and constant concatenations are supported.

It reports these cases:

- An `assert` message contains the word `must`, or a `require` message contains
  `should`, regardless of capitalisation. Words inside expected values do not count.
- An assertion uses its formatted variant for a message without a percent sign.
- An assertion uses its plain variant for a message with a valid printf operation.
  Escaped percents, indexed arguments, dynamic widths and custom verbs count as
  formatting operations.

An assertion must have the corresponding alternative function or method before
the checker recommends it. For example, `assert.CollectT.Errorf` has no plain
`Error` method and is left alone.

The check scans tracked Go sources selected by the build, excluding generated
files. Type checking still requires the complete package to compile. Malformed
format strings and argument mismatches remain covered by testifylint.

Missing messages and messages containing neither `must` nor `should` remain
outside this check's scope. Existing tests contain many of these. The coding
guidelines remain the writing standard, but this focused replacement does not
enforce every guideline retroactively. Nonconstant messages, expanded message
slices, indirect function calls and arbitrary helper wrappers are not evaluated.

The message-parameter lookup follows
[testifylint's signature-based approach](https://github.com/antonboom/testifylint/blob/720065d36f434c616017441aa8f2069e621b5e44/internal/checkers/formatter.go).
Its [MIT notice](../THIRD_PARTY_NOTICES) is retained.
To decide whether a message formats something, the checker formats it with
Go's `fmt` package and no arguments. Any change means a real formatting
operation. It does not duplicate testifylint's full formatting validation.
