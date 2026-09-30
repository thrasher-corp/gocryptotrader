package engine

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNTPClockChangeDuringQuery(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		wall, elapsed time.Duration
		awake         time.Duration
		valid         bool
	}{
		{name: "slow collection", valid: true},
		{name: "forward step", wall: time.Second},
		{name: "backward step", wall: -time.Second},
		{name: "suspend", wall: time.Hour, elapsed: time.Hour},
		{name: "Windows suspend advances Go monotonic time", awake: -10 * time.Second},
		{name: "clock resolution skew", elapsed: 20 * time.Millisecond, valid: true},
		{name: "awake clock reading delay", awake: 20 * time.Millisecond, valid: true},
		{name: "elapsed clock moves backwards", elapsed: -time.Hour},
		{name: "awake clock moves backwards", awake: -time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				manager := testNTPManager(t, 1)
				start := time.Now()
				var wall, elapsed, awake time.Duration
				manager.clock = func() (ntpClockReading, error) {
					return ntpClockReading{wall: time.Now().Add(wall), elapsed: time.Since(start) + elapsed, awake: time.Since(start) + awake}, nil
				}
				manager.survey.query = func(context.Context, string) (*ntp.Response, error) {
					time.Sleep(30 * time.Second)
					wall, elapsed = tc.wall, tc.elapsed
					awake = tc.awake
					return &ntp.Response{RootDistance: time.Millisecond}, nil
				}
				require.NoError(t, manager.Start(t.Context()), "worker must start")
				time.Sleep(31 * time.Second)
				require.NoError(t, manager.Stop(), "worker must join before inspecting its evidence")
				if tc.valid {
					assert.Equal(t, ntpHealthy, manager.history.last, "slow collection should remain healthy within its uncertainty bound")
				} else {
					assert.Equal(t, ntpUnknown, manager.history.last, "clock discontinuity should discard in-flight evidence")
					assert.False(t, manager.history.unknown, "detector event should not count as scheduled verification failure")
				}
			})
		})
	}
}

func TestNTPClockDiscontinuity(t *testing.T) {
	t.Parallel()
	previous := ntpClockReading{wall: time.Unix(1_700_000_000, 0), elapsed: time.Minute, awake: time.Minute}
	for _, tc := range []struct {
		name                 string
		wall, elapsed, awake time.Duration
		changed              bool
	}{
		{name: "wall reading delay", wall: 15*time.Second + 20*time.Millisecond, elapsed: 15 * time.Second, awake: 15 * time.Second},
		{name: "elapsed clock moves backwards alone", wall: -time.Millisecond, elapsed: -time.Millisecond, changed: true},
		{name: "awake clock moves backwards alone", awake: -time.Millisecond, changed: true},
	} {
		current := ntpClockReading{wall: previous.wall.Add(tc.wall), elapsed: previous.elapsed + tc.elapsed, awake: previous.awake + tc.awake}
		assert.Equalf(t, tc.changed, ntpClockDiscontinuity(previous, current), "%s should give the expected discontinuity result", tc.name)
	}
}
