package signaler

import (
	"os"
	"syscall"
	"testing"
	"time"
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
	if err != nil {
		t.Fatalf("failed to find process: %v", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send SIGTERM: %v", err)
	}

	select {
	case sig := <-done:
		if sig != syscall.SIGTERM {
			t.Fatalf("expected SIGTERM, got %v", sig)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for WaitForInterrupt to return")
	}
}
