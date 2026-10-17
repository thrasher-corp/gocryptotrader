package engine

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// readNTPClock reads wall, awake and sleep-inclusive clocks.
// Apple's Libc maps CLOCK_MONOTONIC_RAW and CLOCK_UPTIME_RAW to mach_continuous_time and mach_absolute_time: https://github.com/apple-oss-distributions/Libc/blob/768d166d42689471e1e8fd1ddde5eee25db02381/gen/clock_gettime.c#L111
func readNTPClock() (ntpClockReading, error) {
	var awake, elapsed unix.Timespec
	// CLOCK_UPTIME_RAW stops during sleep, as does Go's timer clock on macOS.
	// Chromium's macOS awake clock reads same mach_absolute_time source: https://github.com/chromium/chromium/blob/fcd1720dfbc767af07055b27f303207fab09c45d/base/time/time_apple.mm#L84
	if err := unix.ClockGettime(unix.CLOCK_UPTIME_RAW, &awake); err != nil {
		return ntpClockReading{}, fmt.Errorf("read NTP awake clock: %w", err)
	}
	// CLOCK_MONOTONIC_RAW keeps counting during sleep, so it drives schedule.
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC_RAW, &elapsed); err != nil {
		return ntpClockReading{}, fmt.Errorf("read NTP scheduling clock: %w", err)
	}
	return ntpClockReading{wall: time.Now(), elapsed: time.Duration(elapsed.Nano()), awake: time.Duration(awake.Nano())}, nil
}
