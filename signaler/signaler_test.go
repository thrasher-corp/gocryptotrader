//go:build !windows
// +build !windows

package signaler

import (
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitForInterrupt_SIGTERM(t *testing.T) {
	done := make(chan os.Signal, 1)

	go func() {
		sig := WaitForInterrupt()
		done <- sig
	}()

	// Give the waiter time to register signal.Notify
	time.Sleep(50 * time.Millisecond)

	proc, err := os.FindProcess(os.Getpid())

	require.NoError(t, err, "os.FindProcess must not error")
	require.NoError(t, proc.Signal(syscall.SIGTERM), "os.FindProcess must not error")

	select {
	case sig := <-done:
		require.Equal(t, sig, syscall.SIGTERM, "expected SIGTERM, got %v", sig)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for WaitForInterrupt to return")
	}
}
