// Package signaler provides cross-platform signal handling for graceful application shutdown
package signaler

import (
	"os"
	"os/signal"
	"syscall"
)

// WaitForInterrupt blocks until a termination signal is received and returns it.
// It registers a temporary channel, unregisters it via signal.Stop() and returns the received signal.
func WaitForInterrupt() os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	s := <-c
	signal.Stop(c) // unregister to avoid keeping channel referenced/registered
	return s
}
