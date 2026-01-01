package signaler

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForInterrupt(t *testing.T) {
	t.Parallel()
	for _, sig := range []os.Signal{syscall.SIGTERM, os.Interrupt} {
		sigC := WaitForInterrupt()
		proc, err := os.FindProcess(os.Getpid())
		require.NoError(t, err, "os.FindProcess must not error")

		if err := proc.Signal(sig); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("proc.Signal(%s) not supported on Windows: %v", sig, err)
			}
			require.NoErrorf(t, err, "proc.Signal(%s) must not error", sig)
		}

		assert.Eventuallyf(t, func() bool {
			select {
			case got := <-sigC:
				return got == sig
			default:
				return false
			}
		}, 2*time.Second, 10*time.Millisecond, "Signal %s should be received within timeout", sig)
	}
}
