# Signaler

A cross-platform helper for graceful shutdown in Go. It blocks until the process receives termination signal and then returns that signal

* Minimal portable set: SIGINT (Ctrl+C) and SIGTERM
* No os.Kill: cannot be caught on any platform
* No SIGABRT by default: preserves default abort+core-dump diagnostics
* Clean unregistration via signal.Stop

* WaitForInterrupt() blocks until a termination signal arrives and returns it.
* Signals are defined in getPlatformSignals().

Platform behavior:
    * Unix/macOS:
        * Receives SIGINT and SIGTERM by default.
        * SIGKILL (os.Kill) and SIGSTOP cannot be caught or ignored.
        * SIGABRT default action is abort + core dump. Catching it changes that behavior, hence excluded here.
    * Windows:
        * Receiving: Go maps console events (Ctrl_C, Ctrl_Break) to os.Interrupt. Our handler will receive os.Interrupt.
        * SIGTERM is included but typically not delivered on windows.
        * os.Process.Kill forcibly terminates the process and cannot be caught.
    * Go:
        * The only signal values guaranteed to exist in the os package on all systems are os.Interrupt and os.Kill. We still register syscall.SIGTERM for Unix portability. On windows, it is usually not delivered but harmless.
    
 * Unix/macOS listen for SIGINT and SIGTERM by default. SIGKILL and SIGSTOP are uncatchable by design and are never registered.
 * Windows maps console events (Ctrl+C, Ctrl+Break) to os.Interrupt including. SIGTERM is harmess but typically not delivered on windows.
 * SIGABRT excluded to preserve the default abort+core-dump diagnostics.

Testing
* Run package tests: go test ./signaler -v count=1
* With race detector: go test ./signaler -race -v count=1
* With coverage: go test ./signaler -cover -v
* Coverage report: go test ./signaler -coverprofile=cover.out ./signaler && go tool cover -func=cover.out

References
    * Go os.Signal guarantees: https://pkg.go.dev/os#Signal
    * Windows console control handlers: https://learn.microsoft.com/en-us/windows/console/setconsolectrlhandler
    * Windows TerminateProcess: https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-terminateprocess

Notes
    * If there's a decision to taken to include SIGABRT, keep in mind that catching it suppresses the default abort+core-dump diagnostics/behavior. To preserve diagnostics, reset and re-raise SIGABRT after cleanup (Unix-only).

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