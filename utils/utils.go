package utils

import (
	"errors"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/thrasher-/gocryptotrader/common"
)

const (
	logFile = "debug.log"
)

// Util vars
var (
	ErrGoMaxProcsFailure = errors.New("failed to set GOMAXPROCS")
	LogFileHandle        *os.File
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

// InitLogFile initialises the log file
func InitLogFile(lFile string) error {
	if LogFileHandle != nil {
		return errors.New("log file already initialised")
	}

	var err error
	LogFileHandle, err = os.OpenFile(lFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	wrt := io.MultiWriter(os.Stdout, LogFileHandle)
	log.SetOutput(wrt)
	return nil
}

// GetLogFile returns the debug.log file
func GetLogFile(dir string) string {
	return dir + common.GetOSPathSlash() + logFile
}
