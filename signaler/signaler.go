package signaler

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var s = make(chan os.Signal, 1)

func init() {
	sigs := getPlatformSignals()
	signal.Notify(s, sigs...)
}

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

// WaitForInterrupt waits until a os.Signal is
// received and returns the result
func WaitForInterrupt() os.Signal {
	return <-s
}
