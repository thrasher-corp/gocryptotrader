package engine

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// readNTPClock reads wall, awake and sleep-inclusive clocks.
func readNTPClock() (ntpClockReading, error) {
	var awake, elapsed unix.Timespec
	// CLOCK_MONOTONIC stops while computer sleeps.
	// Comparing how far each clock advanced estimates time spent asleep.
	// Chromium uses this clock as its awake clock: https://github.com/chromium/chromium/blob/fcd1720dfbc767af07055b27f303207fab09c45d/base/time/time_now_posix.cc#L100
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &awake); err != nil {
		return ntpClockReading{}, fmt.Errorf("read NTP awake clock: %w", err)
	}
	// CLOCK_BOOTTIME keeps counting while computer sleeps and ignores wall-clock steps, so it drives check schedule.
	// systemd-timesyncd schedules its polls with same clock: https://github.com/systemd/systemd/blob/885fe07ee37cff7316680b5088d11081e01813b1/src/timesync/timesyncd-manager.c#L206
	if err := unix.ClockGettime(unix.CLOCK_BOOTTIME, &elapsed); err != nil {
		return ntpClockReading{}, fmt.Errorf("read NTP scheduling clock: %w", err)
	}
	return ntpClockReading{wall: time.Now(), elapsed: time.Duration(elapsed.Nano()), awake: time.Duration(awake.Nano())}, nil
}
