# GoCryptoTrader Coding Guidelines

This document outlines the coding, formatting, and testing standards for implementing or refactoring exchange API integrations or any related functionality within the codebase. These practices ensure consistency, maintainability, and performance throughout the project.

## General Standards

- Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
- Code must adhere to these [Effective Go](https://go.dev/doc/effective_go) guidelines.
- Code must also follow these [Go Style](https://google.github.io/styleguide/go/) guidelines.
- Use Australian/British English in project-owned prose, comments, test
    messages and identifiers. Prefer `-ise` and `-isation` spellings over
    `-ize` and `-ization`, for example `normalise`, `serialise` and
    `standardisation`. Preserve externally defined API names, protocol fields,
    quotations and compatibility-sensitive public identifiers. When codespell
    reports an American spelling, use the replacement in the
    [custom dictionary](../contrib/spellcheck/codespell_custom_dictionary.txt)
    unless an external contract requires the original spelling.

## Security

See [SECURITY.md](../SECURITY.md) for the project's security policy, supported versions and reporting process.

- Never commit API keys, secrets, client IDs or a populated `config.json`. Use placeholder values in tests and examples.
- Never log or return credentials in error messages, RPC responses or test output.
- If you discover a vulnerability while working on the codebase, report it privately as described in [SECURITY.md](../SECURITY.md). Do not describe it in a public issue, pull request or commit message.

## Exchange Implementation Guidelines

Refer to the [ADD_NEW_EXCHANGE.md](../docs/ADD_NEW_EXCHANGE.md) document for comprehensive steps on integrating a new exchange.

### Endpoint Organisation

- Implement API endpoints in the order they are presented in the API documentation to maintain alignment with the source.
- Group related endpoints into files that follow the documented API structure.
- Inline endpoint paths directly in the method implementation. Avoid defining them as constants elsewhere.
- Export exchange types, functions and methods by default (e.g. `func (e *Exchange) GetOrderBook(...)`) so that GoCryptoTrader can be consumed as both a standalone library and interfaced via the engine package.

### Type Usage

- Use the most appropriate native Go types for struct fields:
  - If the API always returns a number as a bare JSON number, use `float64`.
  - If the API returns a number as a quoted JSON string, or may return either a string or a bare number, use `types.Number` — **do not** use `float64` with the `json:",string"` tag.
  - For timestamps, use `time.Time` if Go's JSON unmarshalling supports the format directly; otherwise use `types.Time` for Unix timestamps that require custom unmarshalling.
- Always use full and descriptive field names for clarity and consistency. Avoid short API-provided aliases unless compatibility requires it.
- Default to `uint64` for exchange API parameters and structs for integers where appropriate.
  - Avoid `int` (size varies by architecture) or `int64` (allows negatives where they don't make sense).
  - Aligns well with `strconv.FormatUint`.
- Use a dedicated currency pair constructor when one exists (for example, `currency.NewBTCUSD()` or `currency.NewBTCUSDT()`) instead of constructing the same pair with `currency.NewPair`. Use `currency.NewPair` when no dedicated constructor exists or the currencies are selected at runtime.

### TestMain usage

- TestMain must avoid API calls, so that individual unit tests can run quickly. Use sync.Once or similar patterns to bootstrap common data without burdening all unit tests with the same overhaed. See `UpdatePairsOnce` for an example of this.

### Struct Naming

- Request structs must be named in the form `XRequest`.
- Response structs must be named in the form `XResponse`.
- All request and response structs should be used as pointers in implementations:

```go
    var x *XResponse
```

### Parameter Handling

- Use pointer structs for passing request parameters.
- Use idiomatic Go types (e.g., `time.Time`) in the parameter definition and convert them within the method as needed when preparing the request.
- Time related requests should default to UTC.

### Path Construction

- Path API endpoints must be inlined within the calling method.
- Use basic string concatenation instead of `fmt.Sprintf`:

```go
    path := "/api/v1/" + id
```

- For multi-part strings, consider using `strings.Builder`:
  - Use only after benchmarking with `testing.B` to ensure it improves performance for realistic input sizes.

- Use the following function:

```go
    path = common.EncodeURLValues(path, params)
```

  to append query parameters efficiently. This handles both empty and set params and will automatically handle the "?" for you.

## Error Handling

- Wrap external errors using fmt.Errorf with context, using the following format:

```go
    return nil, fmt.Errorf("error fetching order: %w", err)
```

- You may define and return your own custom errors when appropriate, especially for known API error codes or validation failures:

```go
    var errInvalidSymbol = errors.New("invalid symbol provided")

    if symbol == "" {
        return nil, errInvalidSymbol
    }
```

- Prefer package-level sentinel errors for validation, parsing, and custom unmarshalling failures that produce deterministic, testable outcomes when callers or tests need stable `errors.Is` matching.
- When returning additional context for a sentinel error, wrap the sentinel with `fmt.Errorf` and `%w` rather than replacing it with a new inline `errors.New(...)` value.

```go
    var errInvalidOrderSide = errors.New("invalid order side")

    if side == "" {
        return fmt.Errorf("%w: empty input", errInvalidOrderSide)
    }
```

- Prefer meaningful and specific error messages that identify the operation being performed.
- Always include enough context in errors to aid in debugging and traceability.
- Do not use panic; always return and propagate errors cleanly.

## Configuration Migrations

Migration code lives in [config/versions](../config/versions), with each version in its own `vN` package. Start with the package instructions and the `ExchangeVersion` and `ConfigVersion` interfaces in [config/versions/versions.go](../config/versions/versions.go). Register new versions in [config/versions/register.go](../config/versions/register.go). For an exchange-specific example, see [config/versions/v14/v14.go](../config/versions/v14/v14.go) and its tests in [config/versions/v14/v14_test.go](../config/versions/v14/v14_test.go).

- Add a new version for subsequent configuration changes rather than rewriting historical migrations to match new types. Keep migration-specific types local to the version package instead of depending on evolving types in the config package.
- For every configuration change, assess how existing saved configurations behave after upgrade. Implement a versioned migration when existing values would otherwise lose functionality, change meaning, or prevent adoption of an intended replacement.
- Updating defaults or example configurations does not migrate existing installations. Test a representative configuration from the previous version through the real configuration loader.
- When no migration is needed, explain why existing configurations remain compatible and whether retaining their previous behaviour is intentional.
- Preserve explicit user choices. Apply default-changing migrations only to configurations that can be reliably identified as using the previous defaults.

### Migration Guards

- Before adding a guard that skips migration, identify the downstream dependency it protects and whether the affected entry can reach that code path. A disabled entry must not block migration solely because its channel exists.
- Match configuration values using the same normalisation rules as the runtime. If runtime parsing is case-insensitive or canonicalises identifiers, test every accepted representation relevant to the migration, such as `spot`, `Spot` and `SPOT`.
- Evaluate effective scope before filtering literal values. Wildcards such as `asset:"all"` can overlap a specific asset after expansion; test both wildcard and explicitly scoped entries.
- Evaluate equivalent channel names together. Generic names and exchange-specific aliases may resolve to the same downstream subscription; guards must cover both spellings across explicit and wildcard scopes.
- Distinguish explicit `false`, explicit `true`, omitted and `null` values where their meanings differ. Use presence-aware decoding when necessary, and document any conservative treatment of unspecified values.
- Test both sides of each guard: a configuration that must remain unchanged and a minimally different configuration that must migrate. Exercise upgrade and downgrade when the guard is shared, and verify unrelated entries remain unchanged.
- Validate migrated configurations through the real downstream loading, expansion and validation paths. Assert that the result remains usable, preserves the intended coverage and introduces no conflicting or exclusive entries.
- Account for version advancement when a migration makes no changes. Do not assume a skipped transformation will be retried after the user changes their configuration.

## Testing Guidelines

### General testing

Verify all tests pass by:

```console
    go test ./... -race -count 1
```

### Assertion Usage

Use `require` and `assert` appropriately:

#### require

- Use when test flow depends on the result.
- Messages must contain **"must"** (e.g., "response must not be nil").

#### assert

- Use when the test can proceed regardless of the check.
- Messages must contain **"should"** (e.g., "status code should be 200").

#### `f` variants (`assert.ErrorIsf`, `require.NoErrorf`, etc.)

- Only use `f` variants when the message contains **format verbs** (e.g., `%s`, `%d`, `%v`).
- If the message is a plain string with no format verbs, use the non-`f` variant.

```go
    // Correct — format verb %s requires the f variant:
    assert.NoErrorf(t, err, "UpdateAccountInfo should not error for asset %s", a)

    // Correct — plain message, no format verbs:
    assert.ErrorIs(t, err, errInvalidOrderSize, "validate should return expected error")

    // Wrong — f variant used without format verbs:
    assert.ErrorIsf(t, err, errInvalidOrderSize, "validate should return expected error")
```

#### Sentinel error coverage

- When returning sentinel errors directly or wrapped, add direct `assert.ErrorIs` or `require.ErrorIs` coverage for that path.
- For validation and custom unmarshalling helpers, prefer a focused test for the exact sentinel error path rather than relying only on indirect endpoint coverage.

### Test Coverage

- Maintain original test inputs unless they are incorrect.
- Derive expected outcomes from intended behaviour and downstream requirements, not solely from the current implementation. Passing tests can preserve an incorrect policy.
- When fixing behaviour for one accepted representation, extend the regression matrix to its equivalent forms. Cross relevant input dimensions, such as aliases, wildcards, accepted capitalisation, authentication or authorisation states, explicit, omitted or `null` values, and forward or reverse lifecycle transitions, where they can affect runtime behaviour.
- Integration tests must reproduce the registration order, ownership and lookup paths relevant to the bug. For isolation tests, make the competing entry reachable first so lookup order cannot conceal a missing discriminator. Where practical, verify the test fails with the targeted fix removed, then restore the fix and verify it passes.
- When resolving review feedback, fix the underlying source of truth, add
    focused regression coverage, regenerate derived files when applicable and
    avoid unrelated behavioural or formatting changes.
- Full test coverage is preferable; mock external calls as needed.
- Distinguish mocked verification from live API verification when reporting results. A credential-gated test that skips does not establish endpoint compatibility; explicitly report the unverified behaviour without exposing credentials.
- All unit tests must pass before finalising changes.

### Interface Contracts

- Name tests and files according to what they prove. Compile-time interface assertions establish API compatibility, not equivalent runtime behaviour; reserve "parity" for tests that compare behaviour.
- Assert conformance against existing standard-library or dependency interfaces instead of duplicating their method signatures. Use custom contract interfaces only for the additional API being checked.
- For build-tag-selected implementations, validate shared contracts and run relevant tests under every supported backend configuration affected by the change.
- Keep signature compatibility separate from behavioural expectations. Document intentional backend differences and test them explicitly where relevant.

### Test deduplication

- Test deduplication should be the default approach for exchanges and across the codebase, an example can be seen below:

```diff
--- a/gateio_test.go
+++ b/gateio_test.go
@@ -89,19 +89,11 @@ func TestGetAccountInfo(t *testing.T) {
     t.Parallel()
     sharedtestvalues.SkipTestIfCredentialsUnset(t, g)
-    _, err := g.UpdateAccountInfo(t.Context(), asset.Spot)
-    if err != nil {
-        t.Error("GetAccountInfo() error", err)
-    }
-    if _, err := g.UpdateAccountInfo(t.Context(), asset.Margin); err != nil {
-        t.Errorf("%s UpdateAccountInfo() error %v", g.Name, err)
-    }
-    if _, err := g.UpdateAccountInfo(t.Context(), asset.CrossMargin); err != nil {
-        t.Error("%s UpdateAccountInfo() error %v", g.Name, err)
-    }
-    if _, err := g.UpdateAccountInfo(t.Context(), asset.Options); err != nil {
-        t.Error("%s UpdateAccountInfo() error %v", g.Name, err)
-    }
-    if _, err := g.UpdateAccountInfo(t.Context(), asset.Futures); err != nil {
-        t.Error("%s UpdateAccountInfo() error %v", g.Name, err)
-    }
-    if _, err := g.UpdateAccountInfo(t.Context(), asset.DeliveryFutures); err != nil {
-        t.Error("%s UpdateAccountInfo() error %v", g.Name, err)
-    }
+    for _, a := range g.GetAssetTypes(false) {
+        _, err := g.UpdateAccountInfo(t.Context(), a)
+        assert.NoErrorf(t, err, "UpdateAccountInfo should not error for asset %s", a)
+    }
}
```

## Comments

- API methods and public types must have comments for GoDoc.
- Comments should explain **why** the code is doing something, not **what** it's doing, which should be self-explanatory.
- Self-explanatory comments must be avoided.
- Only retain comments for complex logic or where external behaviour needs clarification.

## Formatting

Run the following after completing changes:

```console
    make gofumpt
```

This ensures proper formatting across the codebase.

### Agent instruction entry points

- Keep `AGENTS.md`, `CLAUDE.md` and `.github/copilot-instructions.md` as concise
    regular Markdown files that direct their respective tools to read this
    document completely before beginning work.
- Keep detailed coding instructions in this document as the single source of
    truth rather than duplicating them across tool-specific entry points.
- Verify these files remain mode `100644` after bulk documentation changes.

## Documentation and Markdown

- Update the source templates under `cmd/documentation` before changing their
    generated Markdown. Regenerate documentation from `cmd/documentation` with
    `go run .` and include the resulting output in the change.
- Run normal contributor fetching when regenerating the root README. Never
    replace the contributor list with output generated from an empty list.
- Generated documentation must use regular source-file permissions (`0644`)
    and must not be executable.
- Markdown normalisation may standardise prose whitespace and line endings,
    but must preserve whitespace inside backtick and tilde fenced code blocks.
    Add focused regression coverage when changing normalisation behaviour.
- Keep code samples correctly formatted. Markdown auto-fixes must not alter
    indentation or semantics inside fenced code blocks.
- Use paths relative to each README for repository images and links so they
    resolve at the checked-out revision on GitHub and pkg.go.dev. Generated
    documentation must use `{{.RepoRoot}}` in templates for repository-root
    resources.
- Validate repository-relative resource references from the location of every
    generated file. Checks must cover source templates and generated outputs
    and verify that each referenced repository file exists.
- When upgrading `markdownlint-cli2` in the `markdownlint` Makefile target,
    review newly introduced rules before changing the config.
- Lint both Markdown and template sources using the same scope as CI:

    ```console
    make markdownlint
    ```

- Run the documentation generator twice when templates or normalisation
    change. The second run must produce no additional diff.
- Before submitting documentation changes, run `git diff --check` and verify
    that generated files have no unexpected mode, encoding, or line-ending
    changes.

## Linters and other miscellaneous checks

- Checks for structured formats such as Markdown, JSON, YAML and Go should use
    an appropriate parser where practical. Scan tracked source files and
    templates, ignore examples inside code blocks and add a regression fixture
    that proves the prohibited form is detected.
- When a repository check requires a tool or runtime, provision that dependency
    explicitly in its CI workflow and document the local prerequisite. Local
    and CI execution must use the same check entry point, and a missing
    dependency must fail clearly rather than silently skip validation.

Run the following to check for linting issues:

```console
    golangci-lint run ./... (or make lint)
```

Run the miscellaneous repository checks locally with:

```console
    make misc_checks
```

The full local verification flow can be run with:

```console
    make check
```

This includes linting, miscellaneous checks and tests. The same miscellaneous checks are also run via [GitHub actions](../.github/workflows/misc.yml).

- All lint warnings and errors must be resolved before merging.
- Use `//nolint:linter-name` sparingly and always explain the reason in a comment next to the code.
- Examples of valid use:

```go
    extension := "strat" //nolint:misspell // its shorthand for strategy
```
