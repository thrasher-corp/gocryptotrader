//go:build !linux && !darwin && !windows && !openbsd

package engine

import (
	"errors"
)

var errNTPClockUnsupported = errors.New("suspend-inclusive NTP scheduling clock is unavailable on this platform")

func readNTPClock() (ntpClockReading, error) {
	return ntpClockReading{}, errNTPClockUnsupported
}
