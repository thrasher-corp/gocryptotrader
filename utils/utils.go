package utils

import (
	"errors"
	"runtime"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	defaultTLSDir = "tls"
)

// Util vars
var (
	ErrGoMaxProcsFailure = errors.New("failed to set GOMAXPROCS")
)

// AdjustGoMaxProcs sets the runtime GOMAXPROCS val
func AdjustGoMaxProcs(maxProcs int) error {
	n := runtime.NumCPU()
	if maxProcs < 0 || maxProcs > n {
		maxProcs = n
	}

	if i := runtime.GOMAXPROCS(maxProcs); i != maxProcs {
		return ErrGoMaxProcsFailure
	}

	return nil
}

// GetTLSDir returns the default TLS dir
func GetTLSDir(dir string) string {
	return dir + common.GetOSPathSlash() + defaultTLSDir
}
