// Package signaler provides cross-platform signal handling for graceful application shutdown.
//
// The package automatically handles platform-specific signal differences and provides
// both a simple legacy API and enhanced testing capabilities.
//
// Basic usage:
//
//	sig := signaler.WaitForInterrupt()
//	log.Printf("Received %v, shutting down...", sig)
//
// The package correctly excludes uncatchable signals (like SIGKILL on Unix systems)
// and includes appropriate signals for each platform.
package signaler

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// s is the global signal channel used by WaitForInterrupt
// it receives OS signals as configured by getPlatformSignals
var s = make(chan os.Signal, 1)

// notifier handles actual signal registration with the OS
// I can be replaced with a mock for testing
var notifier SignalNotifier = &osSignalNotifier{}

func init() {
	sigs := getPlatformSignals()
	notifier.Notify(s, sigs...)
}

// getPlatformSignals returns the appropriate signals to listen for on the current platform.
// On Unix-like systems (Linux, macOS, BSD), sigkill is excluded because it cannot be caught
// or ignored. On Windows, sigkill is included because it can be caught.
func getPlatformSignals() []os.Signal {
	signals := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGABRT,
	}

	// Add os.Kill only for windows
	// os.Kill cannot be caught or ignored on Unix-based systems
	if runtime.GOOS == "windows" {
		signals = append(signals, os.Kill)
	}
	return signals
}

// SignalNotifier is an interface for signal notification
type SignalNotifier interface {
	Notify(c chan<- os.Signal, sig ...os.Signal)
	Stop(c chan<- os.Signal)
}

// osSignalNotifier is the default implementation of SignalNotifier
type osSignalNotifier struct{}

// Notify registers the signal channel with the OS
func (o *osSignalNotifier) Notify(c chan<- os.Signal, sig ...os.Signal) {
	signal.Notify(c, sig...)
}

// Stop stops the signal registration with the OS
func (o *osSignalNotifier) Stop(c chan<- os.Signal) {
	signal.Stop(c)
}

// NewSignalNotifier creates a new SignalNotifier
// This function returns a SignalNotifier that uses the operating system's signal handling mechanism
func NewSignalNotifier() SignalNotifier {
	return &osSignalNotifier{}
}

// WaitForInterrupt waits until a os.Signal is received and returns the result
// This function blocks until of one the following signals is received:
// - SIGINT (Ctrl+C)
// - SIGTERM (termination request)
// - SIGABRT (abort signal)
// - SIGKILL (windows only - cannot be caught on Unix systems)
//
// The function automatically handles platform-specific signal behavior
func WaitForInterrupt() os.Signal {
	return <-s
}
