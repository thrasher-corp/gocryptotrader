// Copyright 2012 The Chromium Authors. The awake-clock call and conversion
// are adapted from Chromium under the BSD-style licence in THIRD_PARTY_NOTICES at the repository root.

package engine

import (
	"fmt"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var queryUnbiasedInterruptTimePrecise = windows.NewLazySystemDLL("kernel32.dll").NewProc("QueryUnbiasedInterruptTimePrecise")

// readNTPClock reads wall, awake and sleep-inclusive clocks.
func readNTPClock() (ntpClockReading, error) {
	if err := queryUnbiasedInterruptTimePrecise.Find(); err != nil {
		return ntpClockReading{}, fmt.Errorf("load NTP awake clock: %w", err)
	}
	// Read Windows' unbiased interrupt time, which stops while computer sleeps, in 100 ns units.
	// Adapted from Chromium's LiveTicks: https://github.com/chromium/chromium/blob/fcd1720dfbc767af07055b27f303207fab09c45d/base/time/time_win.cc#L755
	var unbiasedInterruptTime uint64
	queryUnbiasedInterruptTimePrecise.Call(uintptr(unsafe.Pointer(&unbiasedInterruptTime))) //nolint:errcheck // Windows function returns nothing, so Call's error is only a leftover from earlier calls
	// DurationSinceBoot reads GetTickCount64: https://cs.opensource.google/go/x/sys/+/refs/tags/v0.48.0:windows/syscall_windows.go
	// GetTickCount64 keeps counting during sleep and hibernation, so it drives check schedule.
	elapsed := windows.DurationSinceBoot()
	return ntpClockReading{wall: time.Now(), elapsed: elapsed, awake: time.Duration(unbiasedInterruptTime) * 100 * time.Nanosecond}, nil
}
