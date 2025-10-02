package signaler

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitForInterrupt_SIGTERM(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM not supported on Windows")
	}
	done := make(chan os.Signal, 1)

	go func() {
		sig := WaitForInterrupt()
		done <- sig
	}()

	// Yield to allow the waiter to register signal.Notify
	runtime.Gosched()

	proc, err := os.FindProcess(os.Getpid())

	require.NoError(t, err, "os.FindProcess must not error")
	require.NoError(t, proc.Signal(syscall.SIGTERM), "proc.Signal must not error")

	var got os.Signal
	require.Eventually(t, func() bool {
		select {
		case got = <-done:
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "timeout waiting for WaitForInterrupt to return")
	require.Equalf(t, syscall.SIGTERM, got, "expected SIGTERM, got %v", got)
}

func TestWaitForInterrupt_Interrupt(t *testing.T) {
	done := make(chan os.Signal, 1)
	go func() {
		sig := WaitForInterrupt()
		done <- sig
	}()

	// Yield to allow the waiter to register signal.Notify
	runtime.Gosched()

	proc, err := os.FindProcess(os.Getpid())
	require.NoError(t, err, "os.FindProcess must not error")
	if err := proc.Signal(os.Interrupt); err != nil {
		// On Windows, delivering os.Interrupt programmatically via os.Process.Signal
		// may not be supported. If so, skip rather than fail the test as the feature
		// is still valid (Ctrl+C generates os.Interrupt).
		if runtime.GOOS == "windows" {
			t.Skipf("os.Process.Signal(os.Interrupt) not supported on Windows: %v", err)
		}
		require.NoError(t, err, "proc.Signal must not error")
	}

	var got os.Signal
	require.Eventually(t, func() bool {
		select {
		case got = <-done:
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond, "timeout waiting for WaitForInterrupt to return")
	require.Equalf(t, os.Interrupt, got, "expected os.Interrupt, got %v", got)
}
