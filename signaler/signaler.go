// Package signaler provides cross-platform signal handling for graceful application shutdown
package signaler

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForInterrupt returns a channel to receive termination signals
func WaitForInterrupt() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	return c
}
