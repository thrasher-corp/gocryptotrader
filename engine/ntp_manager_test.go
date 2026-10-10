package engine

import (
	"context"
	"errors"
	"net/netip"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/beevik/ntp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func testNTPManager(t *testing.T, level int) *ntpManager {
	t.Helper()
	tolerance := 50 * time.Millisecond
	manager, err := setupNTPManager(&config.NTPClientConfig{
		Level: level, Pool: []string{"test.invalid"}, AllowedDifference: &tolerance, AllowedNegativeDifference: &tolerance,
	}, false)
	require.NoError(t, err, "test manager must be configured")
	start := time.Now()
	manager.clock = func() (ntpClockReading, error) {
		return ntpClockReading{wall: time.Now(), elapsed: time.Since(start), awake: time.Since(start)}, nil
	}
	manager.interval = func() time.Duration { return 15 * time.Minute }
	manager.survey.resolve = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
	}
	manager.survey.query = func(context.Context, string) (*ntp.Response, error) {
		return &ntp.Response{RootDistance: time.Millisecond}, nil
	}
	t.Cleanup(func() { _ = manager.Stop() })
	return manager
}

func TestSetupNTPManager(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, testNTPManager(t, 1).Stop(), ErrSubSystemNotStarted, "Stop before Start should report the subsystem state")
	_, err := setupNTPManager(nil, false)
	assert.ErrorIs(t, err, errNilConfig, "nil config should be rejected")
	_, err = setupNTPManager(&config.NTPClientConfig{}, false)
	assert.ErrorIs(t, err, errNilNTPConfigValues, "uninitialised tolerance should be rejected")
	tolerance := time.Second
	_, err = setupNTPManager(&config.NTPClientConfig{AllowedDifference: &tolerance}, false)
	assert.ErrorIs(t, err, errNilNTPConfigValues, "missing negative tolerance should be rejected")
	_, err = setupNTPManager(&config.NTPClientConfig{AllowedNegativeDifference: &tolerance}, false)
	assert.ErrorIs(t, err, errNilNTPConfigValues, "missing positive tolerance should be rejected")
	var zero time.Duration
	_, err = setupNTPManager(&config.NTPClientConfig{AllowedDifference: &zero, AllowedNegativeDifference: &zero}, false)
	assert.NoError(t, err, "zero tolerance should be accepted")
	_, err = setupNTPManager(&config.NTPClientConfig{Level: 2, AllowedDifference: &tolerance, AllowedNegativeDifference: &tolerance}, false)
	assert.ErrorIs(t, err, errNTPConfig, "invalid level should be rejected")
}

func TestNTPManagerModes(t *testing.T) {
	t.Parallel()
	for _, level := range []int{-1, 0, 1} {
		t.Run(map[int]string{-1: "disabled", 0: "legacy startup value", 1: "periodic"}[level], func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				manager := testNTPManager(t, level)
				var calls atomic.Int32
				manager.survey.query = func(context.Context, string) (*ntp.Response, error) {
					calls.Add(1)
					return &ntp.Response{RootDistance: time.Millisecond}, nil
				}
				err := manager.Start(t.Context())
				if level == -1 {
					require.ErrorIs(t, err, errNTPManagerDisabled, "disabled manager must not start")
					assert.Zero(t, calls.Load(), "disabled manager should not query")
					return
				}
				require.NoError(t, err, "enabled manager must start without waiting for an observation")
				synctest.Wait()
				assert.Equal(t, int32(1), calls.Load(), "enabled manager should check promptly even without logging")
				time.Sleep(15 * time.Minute)
				synctest.Wait()
				assert.Equal(t, int32(2), calls.Load(), "enabled manager should check periodically")
				assert.True(t, manager.IsRunning(), "enabled manager should keep running after its checks")
			})
		})
	}
}

func TestNTPManagerStopJoinsBeforeRestart(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		manager := testNTPManager(t, 1)
		entered := make(chan struct{}, 2)
		release := make(chan struct{})
		var calls, active, overlap atomic.Int32
		manager.survey.query = func(ctx context.Context, _ string) (*ntp.Response, error) {
			calls.Add(1)
			if active.Add(1) != 1 {
				overlap.Add(1)
			}
			defer active.Add(-1)
			entered <- struct{}{}
			<-ctx.Done()
			<-release
			return nil, ctx.Err()
		}
		require.NoError(t, manager.Start(t.Context()), "first run must start")
		select {
		case <-entered:
		case <-time.After(time.Hour):
			// Let a late query return, so the cleanup's Stop can finish.
			close(release)
			require.FailNow(t, "first run must start its query")
		}
		assert.ErrorIs(t, manager.Start(t.Context()), ErrSubSystemAlreadyStarted, "duplicate Start should not launch another worker")
		stopped := make(chan error, 1)
		go func() { stopped <- manager.Stop() }()
		synctest.Wait()
		select {
		case <-stopped:
			t.Fatal("Stop must wait for the in-flight query to exit")
		default:
		}
		restarted := make(chan error, 1)
		go func() { restarted <- manager.Start(t.Context()) }()
		// synctest.Wait cannot finish while restart goroutine waits for mutex, so release query first.
		// Results below verify worker restarts without overlapping previous run.
		close(release)
		require.NoError(t, <-stopped, "Stop must join the old run")
		require.NoError(t, <-restarted, "restart must succeed after the join")
		synctest.Wait()
		assert.Zero(t, active.Load(), "restart should preserve the next query deadline")
		time.Sleep(15 * time.Minute)
		select {
		case <-entered:
		case <-time.After(time.Hour):
			require.FailNow(t, "restarted run must start its due query")
		}
		require.NoError(t, manager.Stop(), "restarted run must also stop cleanly")
		assert.Equal(t, int32(2), calls.Load(), "there should be exactly one query per due round")
		assert.Zero(t, overlap.Load(), "old and new rounds should never overlap")
		assert.Zero(t, active.Load(), "Stop should return with no query left running")
	})
}

func TestNTPManagerClockChanges(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		manager := testNTPManager(t, 1)
		origin := time.Now()
		var extraElapsed, extraWall atomic.Int64
		manager.clock = func() (ntpClockReading, error) {
			return ntpClockReading{wall: time.Now().Add(time.Duration(extraWall.Load())), elapsed: time.Since(origin) + time.Duration(extraElapsed.Load()), awake: time.Since(origin)}, nil
		}
		var calls atomic.Int32
		manager.survey.query = func(context.Context, string) (*ntp.Response, error) {
			calls.Add(1)
			return &ntp.Response{RootDistance: time.Millisecond}, nil
		}
		require.NoError(t, manager.Start(t.Context()), "manager must start")
		synctest.Wait()
		extraWall.Add(int64(time.Hour))
		time.Sleep(15 * time.Second)
		synctest.Wait()
		require.NoError(t, manager.Stop(), "worker must join before the restart")
		assert.Equal(t, int32(1), calls.Load(), "wall step should not grant an extra network round")
		require.NoError(t, manager.Start(t.Context()), "worker must restart with its deadline preserved")
		synctest.Wait()
		extraWall.Add(int64(-2 * time.Hour))
		time.Sleep(15 * time.Second)
		synctest.Wait()
		assert.Equal(t, int32(1), calls.Load(), "backward step should not grant an extra round")
		extraElapsed.Add(int64(2 * time.Hour))
		extraWall.Add(int64(2 * time.Hour))
		time.Sleep(15 * time.Second)
		synctest.Wait()
		assert.Equal(t, int32(2), calls.Load(), "long suspend should make one normal round due")
		time.Sleep(time.Minute)
		synctest.Wait()
		assert.Equal(t, int32(2), calls.Load(), "resume should not replay missed rounds")
	})
}

func TestNTPManagerCancelledDNSRecoversNextRound(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		manager := testNTPManager(t, 1)
		entered := make(chan struct{})
		var lookups atomic.Int32
		manager.survey.resolve = func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
			if lookups.Add(1) == 1 {
				close(entered)
				<-ctx.Done()
				return nil, ctx.Err()
			}
			return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
		}
		require.NoError(t, manager.Start(t.Context()), "worker must start the first lookup")
		select {
		case <-entered:
		case <-time.After(time.Hour):
			require.FailNow(t, "worker must reach its first lookup")
		}
		require.NoError(t, manager.Stop(), "Stop must cancel and join DNS resolution")
		assert.False(t, manager.history.unknown, "shutdown alone should not open an unknown episode")
		require.NoError(t, manager.Start(t.Context()), "worker must restart with request deadlines preserved")
		time.Sleep(14 * time.Minute)
		synctest.Wait()
		assert.Equal(t, int32(1), lookups.Load(), "restart should not grant an early DNS retry")
		time.Sleep(time.Minute)
		synctest.Wait()
		require.NoError(t, manager.Stop(), "worker must join after the next scheduled round")
		assert.Equal(t, int32(2), lookups.Load(), "next scheduled round should retry interrupted resolution")
		assert.Equal(t, ntpHealthy, manager.history.last, "recovery should use new clock evidence")
		assert.False(t, manager.history.unknown, "fresh healthy evidence should close the unknown episode")
	})
}

func TestSetupNTPManagerDefaultServers(t *testing.T) {
	t.Parallel()
	tolerance := 50 * time.Millisecond
	manager, err := setupNTPManager(&config.NTPClientConfig{AllowedDifference: &tolerance, AllowedNegativeDifference: &tolerance}, false)
	require.NoError(t, err, "empty pool must use default servers")
	hosts := make([]string, 0, len(manager.survey.sources))
	for _, source := range manager.survey.sources {
		hosts = append(hosts, source.host)
	}
	assert.Equal(t, []string{"time.cloudflare.com", "any.time.nl", "time.nist.gov", "ntp.nict.jp"}, hosts, "empty pool should use four servers from different organisations")
}

func TestNilNTPManager(t *testing.T) {
	t.Parallel()
	var manager *ntpManager
	assert.False(t, manager.IsRunning(), "nil manager should not be running")
	assert.ErrorIs(t, manager.Start(t.Context()), ErrNilSubsystem, "nil Start should fail safely")
	assert.ErrorIs(t, manager.Stop(), ErrNilSubsystem, "nil Stop should fail safely")
}

func TestNTPUnknownEscalationUsesCompletedObservations(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		manager := testNTPManager(t, 1)
		calls := 0
		manager.survey.query = func(context.Context, string) (*ntp.Response, error) {
			calls++
			if calls == 1 {
				time.Sleep(10 * time.Second)
			} else {
				time.Sleep(time.Second)
			}
			return nil, errors.New("test server did not reply")
		}
		for _, tc := range []struct {
			wait   time.Duration
			warned bool
		}{
			{wait: 11 * time.Second},
			// Second completion is only 14m51s after first, even though scheduled attempts started 15 minutes apart.
			{wait: 14*time.Minute + 51*time.Second},
			{wait: 15 * time.Minute, warned: true},
		} {
			require.NoError(t, manager.Start(t.Context()), "manager must start with its previous deadline preserved")
			time.Sleep(tc.wait)
			synctest.Wait()
			require.NoError(t, manager.Stop(), "worker must join before inspecting notification history")
			require.True(t, manager.history.unknown, "failed collection must leave an unknown episode")
			assert.Equal(t, tc.warned, manager.history.warned, "warning should require prolonged inability measured from first completed failure")
		}
		assert.Equal(t, 3, calls, "reporting persistence should not create extra network attempts")
	})
}

func TestNTPManagerClockFailureStopsWorker(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		failingRead int32
	}{
		{name: "before survey", failingRead: 1},
		{name: "after survey", failingRead: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				manager := testNTPManager(t, 1)
				reading := manager.clock
				var reads, queries atomic.Int32
				manager.clock = func() (ntpClockReading, error) {
					if reads.Add(1) == tc.failingRead {
						return ntpClockReading{}, errors.New("test clock unavailable")
					}
					return reading()
				}
				manager.survey.query = func(context.Context, string) (*ntp.Response, error) {
					queries.Add(1)
					return &ntp.Response{RootDistance: time.Millisecond}, nil
				}
				require.NoError(t, manager.Start(t.Context()), "worker must start")
				synctest.Wait()
				assert.False(t, manager.IsRunning(), "worker should exit when its clock cannot be read")
				assert.Equal(t, tc.failingRead-1, queries.Load(), "worker should stop at the failed clock read")
				assert.Equal(t, ntpReporting{}, manager.history, "a round without a clock reading should not be reported")
				assert.NoError(t, manager.Stop(), "Stop should join a worker that already exited")
			})
		})
	}
}
