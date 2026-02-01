# GoCryptoTrader Coding Guidelines

This document outlines the coding, formatting, and testing standards for implementing or refactoring exchange API integrations or any related functionality within the codebase. These practices ensure consistency, maintainability, and performance throughout the project.

## General Standards

- Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
- Code must adhere to these [Effective Go](https://go.dev/doc/effective_go) guidelines.
- Code must also follow these [Go Style](https://google.github.io/styleguide/go/) guidelines.

## Exchange Implementation Guidelines

Refer to the [ADD_NEW_EXCHANGE.md](/docs/ADD_NEW_EXCHANGE.md) document for comprehensive steps on integrating a new exchange.

### Endpoint Organisation

- Implement API endpoints in the order they are presented in the API documentation to maintain alignment with the source.
- Group related endpoints into files that follow the documented API structure.
- Inline endpoint paths directly in the method implementation. Avoid defining them as constants elsewhere.
- Export exchange types, functions and methods by default (e.g. `func (e *Exchange) GetOrderBook(...)`) so that GoCryptoTrader can be consumed as both a standalone library and interfaced via the engine package.

### Type Usage

- Use the most appropriate native Go types for struct fields:
  - If the API returns numbers as strings, use float64 with the `json:",string"` tag.
  - For timestamps, use `time.Time` if Go's JSON unmarshalling supports the format directly.
- If native Go types are not supported directly, use the following built-in types:
  - `types.Time` for Unix timestamps that require custom unmarshalling.
  - `types.Number` for numerical float values where an exchange API may return either a `string` or `float64` value.
- Always use full and descriptive field names for clarity and consistency. Avoid short API-provided aliases unless compatibility requires it.
- Default to `uint64` for exchange API parameters and structs for integers where appropriate.
  - Avoid `int` (size varies by architecture) or `int64` (allows negatives where they don't make sense).
  - Aligns well with `strconv.FormatUint`.

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

- Prefer meaningful and specific error messages that identify the operation being performed.
- Always include enough context in errors to aid in debugging and traceability.
- Do not use panic; always return and propagate errors cleanly.

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
- Use the *f* variants when using format specifiers (e.g., `require.Equalf`).

#### assert

- Use when the test can proceed regardless of the check.
- Messages must contain **"should"** (e.g., "status code should be 200").
- Use `assert.Equalf`, etc., when applicable.

### Test Coverage

- Maintain original test inputs unless they are incorrect.
- Full test coverage is preferable; mock external calls as needed.
- All unit tests must pass before finalising changes.

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
- Only retain comments for complex logic or where external behavior needs clarification.

## Formatting

Run the following after completing changes:

```console
    make gofumpt
```

This ensures proper formatting across the codebase.

## Linters and other miscellaneous checks

Run the following to check for linting issues:

```console
    golangci-lint run ./... (or make lint)
```

Run the following tool to check for Go modernise issues:

```console
    make modernise
```

Several other miscellaneous checks will be run via [GitHub actions](/.github/workflows/misc.yml).

- All lint warnings and errors must be resolved before merging.
- Use `//nolint:linter-name` sparingly and always explain the reason in a comment next to the code.
- Examples of valid use:

```go
    extension := "strat" //nolint:misspell // its shorthand for strategy
```
