package engine

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// readNTPClock reads wall, awake and sleep-inclusive clocks.
// CLOCK_BOOTTIME keeps counting during sleep and CLOCK_UPTIME stops.
// Comparing how far each clock advanced estimates time spent asleep.
// OpenBSD documents this pairing in its clock_gettime example.
// Its kernel builds UPTIME by subtracting accumulated sleep (nanoruntime via binruntime).
// https://man.openbsd.org/clock_gettime.2#EXAMPLES
// https://github.com/openbsd/src/blob/82041f51fffa2d899cdb05fc2023a3d959b11b96/sys/kern/kern_time.c#L124
// https://github.com/openbsd/src/blob/82041f51fffa2d899cdb05fc2023a3d959b11b96/sys/kern/kern_tc.c#L271
func readNTPClock() (ntpClockReading, error) {
	var awake, elapsed unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_UPTIME, &awake); err != nil {
		return ntpClockReading{}, fmt.Errorf("read NTP awake clock: %w", err)
	}
	if err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &elapsed); err != nil {
		return ntpClockReading{}, fmt.Errorf("read NTP scheduling clock: %w", err)
	}
	return ntpClockReading{wall: time.Now(), elapsed: time.Duration(elapsed.Nano()), awake: time.Duration(awake.Nano())}, nil
}
