package engine

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

func setupNTPManager(cfg *config.NTPClientConfig, loggingEnabled bool) (*ntpManager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if cfg.AllowedNegativeDifference == nil || cfg.AllowedDifference == nil {
		return nil, errNilNTPConfigValues
	}
	if cfg.Level < -1 || cfg.Level > 1 || *cfg.AllowedDifference < 0 || *cfg.AllowedNegativeDifference < 0 {
		return nil, errNTPConfig
	}
	pools := cfg.Pool
	if len(pools) == 0 {
		// An empty list means "use GCT's default servers".
		// Default is filled in here and never written back to saved config, so later default changes reach existing users.
		// Syncthing expands its stored default setting into separate runtime list the same way: https://github.com/syncthing/syncthing/blob/94c3c1cdef718d568686620cbff268eeaaf2c87d/lib/config/optionsconfiguration.go#L210
		pools = defaultNTPServers
	}
	survey, err := newNTPSurvey(pools)
	if err != nil {
		return nil, err
	}
	return &ntpManager{
		level:                     cfg.Level,
		allowedDifference:         *cfg.AllowedDifference,
		allowedNegativeDifference: *cfg.AllowedNegativeDifference,
		survey:                    survey,
		loggingEnabled:            loggingEnabled,
		clock:                     readNTPClock,
		interval: func() time.Duration {
			return defaultNTPCheckInterval + time.Duration(rand.Int64N(int64(time.Minute))) //nolint:gosec // Scheduling jitter is not a security token
		},
	}, nil
}

// IsRunning reports whether the NTP worker is still running.
func (m *ntpManager) IsRunning() bool {
	if m == nil {
		return false
	}
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	return m.running()
}

func (m *ntpManager) running() bool {
	if m.run == nil {
		return false
	}
	select {
	case <-m.run.done:
		return false
	default:
		return true
	}
}

// Start launches the NTP worker without blocking engine startup.
func (m *ntpManager) Start(parent context.Context) error {
	if m == nil {
		return fmt.Errorf("NTP manager %w", ErrNilSubsystem)
	}
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	if m.running() {
		return fmt.Errorf("NTP manager %w", ErrSubSystemAlreadyStarted)
	}
	if m.level == -1 {
		return errNTPManagerDisabled
	}
	ctx, cancel := context.WithCancel(parent)
	run := &ntpRun{cancel: cancel, done: make(chan struct{})}
	m.run = run
	go func() {
		defer close(run.done)
		defer cancel()
		m.observe(ctx)
	}()
	return nil
}

// Stop cancels the NTP worker and waits for it to exit.
func (m *ntpManager) Stop() error {
	if m == nil {
		return fmt.Errorf("NTP manager %w", ErrNilSubsystem)
	}
	// Worker never takes this lock, so waiting while holding it cannot deadlock.
	// Same pattern as Consul's CheckHTTP Start/Stop, plus a guard against starting twice: https://github.com/hashicorp/consul/blob/c3f767b1146f4e394dc147f6d9438f02b7039928/agent/checks/check.go#L379
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	if m.run == nil {
		return fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	m.run.cancel()
	<-m.run.done
	m.run = nil
	return nil
}

// observe runs the NTP worker loop until ctx is cancelled or a clock read fails.
// It checks once per interval, first check immediately on first start, and discards a result if clock changed or computer slept during it.
func (m *ntpManager) observe(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		// If shutdown and timer happen together, select may pick timer.
		// Check for shutdown again before starting query.
		// Kubernetes' wait loop does the same: https://github.com/kubernetes/kubernetes/blob/6c1c7702cf2052245ef10e699d45f071af306f59/staging/src/k8s.io/apimachinery/pkg/util/wait/backoff.go#L236
		if ctx.Err() != nil {
			return
		}
		current, err := m.clock()
		if err != nil {
			if m.loggingEnabled {
				log.Warnln(log.TimeMgr, err)
			}
			return
		}
		if current.elapsed >= m.nextRound {
			// Schedule the next check one interval from now.
			// After sleep, run one overdue check instead of catching up every missed one.
			// Wall-clock steps never cause extra queries.
			m.nextRound = current.elapsed + m.interval()
			observation := m.survey.observe(ctx, current.elapsed, m.allowedDifference, m.allowedNegativeDifference)
			if ctx.Err() != nil {
				return
			}
			after, err := m.clock()
			if err != nil {
				if m.loggingEnabled {
					log.Warnln(log.TimeMgr, err)
				}
				return
			}
			if ntpClockDiscontinuity(current, after) {
				if m.loggingEnabled {
					log.Debugln(log.TimeMgr, errNTPClockChanged)
				}
			} else {
				m.report(&observation, after.elapsed)
			}
			current = after
		}
		// Go's timers stop during sleep on some platforms, so wake regularly to notice an overdue check.
		timer.Reset(min(ntpClockWatchInterval, max(time.Millisecond, m.nextRound-current.elapsed)))
	}
}

// ntpClockDiscontinuity reports whether clock changed or computer slept between two readings.
// Compare how far wall clock and sleep-inclusive clock advanced, allowing up to 50 ms difference in either direction.
// Apply same limit to sleep-inclusive and awake clocks.
// Also reject either of those clock readings going backwards.
// Inspired by Tailscale's wall-time watch for resume, but using explicit clock comparisons instead: https://github.com/tailscale/tailscale/blob/2d4379386a5f02342f1a52006555764a22320d6e/net/netmon/netmon.go#L703
func ntpClockDiscontinuity(previous, current ntpClockReading) bool {
	elapsed := current.elapsed - previous.elapsed
	awake := current.awake - previous.awake
	wall := current.wall.Round(0).Sub(previous.wall.Round(0))
	delta := wall - elapsed
	// Go's clock includes sleep on Windows, so use separate awake clock.
	// Both elapsed clocks keep counting during ordinary slow checks, which helps distinguish them from sleep.
	suspended := elapsed - awake
	return elapsed < 0 || awake < 0 || delta > 50*time.Millisecond || delta < -50*time.Millisecond || suspended > 50*time.Millisecond || suspended < -50*time.Millisecond
}
