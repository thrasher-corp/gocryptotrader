package engine

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupNTPManager creates a new NTP manager.
func setupNTPManager(cfg *config.NTPClientConfig) (*ntpManager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if cfg.AllowedNegativeDifference == nil || cfg.AllowedDifference == nil {
		return nil, errNilNTPConfigValues
	}
	return &ntpManager{
		shutdown:                  make(chan struct{}),
		level:                     cfg.Level,
		allowedDifference:         *cfg.AllowedDifference,
		allowedNegativeDifference: *cfg.AllowedNegativeDifference,
		pools:                     cfg.Pool,
		checkInterval:             defaultNTPCheckInterval,
		measure:                   clockMeasurer{queryOne: queryClock}.measure,
	}, nil
}

// IsRunning safely checks whether the subsystem is running.
func (m *ntpManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return m.started.Load()
}

// Start runs the subsystem.
func (m *ntpManager) Start() error {
	if m == nil {
		return fmt.Errorf("ntp manager %w", ErrNilSubsystem)
	}
	if !m.started.CompareAndSwap(false, true) {
		return fmt.Errorf("NTP manager %w", ErrSubSystemAlreadyStarted)
	}
	if m.level != config.NTPClientPeriodic {
		m.started.CompareAndSwap(true, false)
		return errNTPManagerDisabled
	}
	m.shutdown = make(chan struct{})
	go m.run()
	log.Debugf(log.TimeMgr, "NTP manager %s", MsgSubSystemStarted)
	return nil
}

// Stop attempts to shutdown the subsystem.
func (m *ntpManager) Stop() error {
	if m == nil {
		return fmt.Errorf("ntp manager %w", ErrNilSubsystem)
	}
	if !m.started.Load() {
		return fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	defer func() {
		log.Debugf(log.TimeMgr, "NTP manager %s", MsgSubSystemShutdown)
		m.started.CompareAndSwap(true, false)
	}()
	log.Debugf(log.TimeMgr, "NTP manager %s", MsgSubSystemShuttingDown)
	close(m.shutdown)
	return nil
}

// run periodically checks the configured time sources.
func (m *ntpManager) run() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdown:
			return
		case <-ticker.C:
			if err := m.processTime(context.Background()); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
		}
	}
}

func (m *ntpManager) checkClock(ctx context.Context) (clockObservation, clockSyncState, error) {
	observation, err := m.measure(ctx, m.pools)
	if err != nil {
		return clockObservation{}, clockSyncInconclusive, fmt.Errorf("NTP clock measurement failed: %w", err)
	}
	return observation,
		evaluateClock(observation, m.allowedDifference, m.allowedNegativeDifference),
		nil
}

// processTime checks and reports the current clock state for a running manager.
func (m *ntpManager) processTime(ctx context.Context) error {
	if !m.started.Load() {
		return fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	observation, state, err := m.checkClock(ctx)
	if err != nil {
		return err
	}
	m.logClockObservation(observation, state)
	return nil
}

func (bot *Engine) setupNTPClient(parent context.Context, input io.Reader) error {
	if !bot.Settings.EnableNTPClient {
		return nil
	}

	manager, err := setupNTPManager(&bot.Config.NTPClient)
	bot.ntpManager = manager
	if err != nil {
		log.Errorf(log.Global, "NTP manager unable to start: %s", err)
		return nil
	}
	return bot.handleStartupNTPPolicy(parent, input, ntpStartupBudget)
}

func (bot *Engine) handleStartupNTPPolicy(parent context.Context, input io.Reader, budget time.Duration) error {
	if bot.Config.NTPClient.Level != config.NTPClientStartup {
		return nil
	}

	ctx, cancel := context.WithTimeout(parent, budget)
	defer cancel()

	observation, state, err := bot.ntpManager.checkClock(ctx)
	if err != nil {
		if parentErr := parent.Err(); parentErr != nil {
			return fmt.Errorf("NTP startup check aborted: %w", parentErr)
		}
		log.Warnf(log.TimeMgr, "NTP manager startup check failed: %v", err)
		return nil
	}

	bot.ntpManager.logClockObservation(observation, state)
	if state != clockSyncAhead && state != clockSyncBehind {
		return nil
	}

	responseMessage, err := bot.Config.SetNTPCheck(input)
	if err != nil {
		return fmt.Errorf("unable to set NTP check: %w", err)
	}
	log.Infoln(log.TimeMgr, responseMessage)
	bot.ntpManager.level = bot.Config.NTPClient.Level
	return nil
}

func (m *ntpManager) logClockObservation(observation clockObservation, state clockSyncState) {
	message, warning := m.clockObservationReport(observation, state)
	if warning {
		log.Warnln(log.TimeMgr, message)
		return
	}
	log.Debugln(log.TimeMgr, message)
}

func (m *ntpManager) clockObservationReport(observation clockObservation, state clockSyncState) (string, bool) {
	correction := observation.clockCorrection.String()
	if observation.clockCorrection > 0 {
		correction = "+" + correction
	}
	message := fmt.Sprintf(
		"NTP manager: state=%s correction=%s (server-local; positive means local behind) root-distance=±%s RTT=%s source=%q allowed=[-%s ahead,+%s behind]",
		state,
		correction,
		observation.uncertainty,
		observation.roundTrip,
		observation.source,
		m.allowedNegativeDifference,
		m.allowedDifference,
	)
	if state == clockSyncInconclusive {
		message += "; no conclusion was possible; configure a closer or lower-latency NTP server"
	}
	return message, state != clockSyncInSync
}

func (s clockSyncState) String() string {
	switch s {
	case clockSyncInSync:
		return "in-sync"
	case clockSyncAhead:
		return "ahead"
	case clockSyncBehind:
		return "behind"
	default:
		return "inconclusive"
	}
}
