package utils

import (
	"errors"
	"path/filepath"
	"runtime"
)

const (
	defaultTLSDir = "tls"
)

// Util vars
var (
	ErrGoMaxProcsFailure = errors.New("failed to set GOMAXPROCS")
)

// AdjustGoMaxProcs sets the runtime GOMAXPROCS val
// Since Go 1.5, Go will use the total number of logical processors that the
// system has available. Caveats to this are if someone has set the GOMAXPROCS
// env var set or wish to limit usage of the number of logical processors
// between a range from 1 to NumCPUs
func AdjustGoMaxProcs(procs int) error {
	// Check for default settings, plus respecting GOMAXPROCS env but
	// don't allow for values which will cause thread contention
	n := runtime.NumCPU()
	if procs == runtime.GOMAXPROCS(-1) {
		if procs <= n {
			return nil
		}
	}

	// Sanitise the procs value (defaults to NumCPUs)
	if procs < 1 || procs > n {
		procs = n
	}

	runtime.GOMAXPROCS(procs)
	if i := runtime.GOMAXPROCS(procs); i != procs {
		return ErrGoMaxProcsFailure
	}
	return nil
}

// GetTLSDir returns the default TLS dir
func GetTLSDir(dir string) string {
	return filepath.Join(dir, defaultTLSDir)
}
