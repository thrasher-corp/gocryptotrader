# Signaler

A cross-platform helper for graceful shutdown in Go. It blocks until a termination signal is received and returns that signal.

What it does
- Listens for SIGINT (Ctrl+C) and SIGTERM.
- Blocks the caller until a termination signal arrives, then returns it.
- Unregisters its channel via signal.Stop before returning.

Platform notes
- Unix/macOS: SIGINT and SIGTERM are commonly delivered by the OS and process managers.
- Windows: Console events (Ctrl+C, Ctrl+Break) are surfaced by Go as os.Interrupt. Including SIGTERM is harmless on Windows even if not typically delivered.

Testing
- Run package tests: go test ./signaler -v -count=1
- With race detector: go test ./signaler -race -v -count=1
- With coverage: go test ./signaler -cover -v
- Coverage report: go test -coverprofile=cover.out ./signaler && go tool cover -func=cover.out

References
    * Go os.Signal guarantees: https://pkg.go.dev/os#Signal
    * Windows console control handlers: https://learn.microsoft.com/en-us/windows/console/setconsolectrlhandler
    * Windows TerminateProcess: https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-terminateprocess

Notes
    * If you decide to include SIGABRT, be aware that catching it suppresses the default abort+core-dump diagnostics. To preserve diagnostics, reset and re-raise SIGABRT after cleanup (Unix-only).

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***