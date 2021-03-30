package ntp

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/ntp/ntpclient"
)

const (
	defaultNTPCheckInterval = time.Second * 30
	defaultRetryLimit       = 3
)

// vars related to the NTP manager
var (
	errNTPDisabled = errors.New("ntp client disabled")
)

// Manager starts the NTP manager
type Manager struct {
	started                   int32
	shutdown                  chan struct{}
	level                     int64
	allowedDifference         time.Duration
	allowedNegativeDifference time.Duration
	pool                      []string
	checkInterval             time.Duration
	retryLimit                int
}

func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Start(cfg *config.NTPClientConfig, loggingEnabled bool) error {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	if cfg.Level != 1 {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
		return errors.New("NTP client disabled")
	}
	m.level = int64(cfg.Level)
	m.allowedDifference = *cfg.AllowedDifference
	m.allowedNegativeDifference = *cfg.AllowedNegativeDifference
	m.pool = cfg.Pool
	m.retryLimit = defaultRetryLimit
	m.checkInterval = defaultNTPCheckInterval

	log.Debugln(log.TimeMgr, "NTP manager starting...")
	if m.level == 0 && loggingEnabled {
		// Sometimes the NTP client can have transient issues due to UDP, try
		// the default retry limits before giving up
	check:
		for i := 0; i < m.retryLimit; i++ {
			err := m.processTime()
			switch err {
			case nil:
				break check
			case errNTPDisabled:
				log.Debugln(log.TimeMgr, "NTP manager: User disabled NTP prompts. Exiting.")
				atomic.CompareAndSwapInt32(&m.started, 1, 0)
				return nil
			default:
				if i == m.retryLimit-1 {
					return err
				}
			}
		}
	}
	m.shutdown = make(chan struct{})
	go m.run()
	log.Debugln(log.TimeMgr, "NTP manager started.")
	return nil
}

func (m *Manager) Stop() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()
	log.Debugln(log.TimeMgr, "NTP manager shutting down...")
	close(m.shutdown)
	return nil
}

func (m *Manager) run() {
	t := time.NewTicker(m.checkInterval)
	defer func() {
		t.Stop()
		log.Debugln(log.TimeMgr, "NTP manager shutdown.")
	}()

	for {
		select {
		case <-m.shutdown:
			return
		case <-t.C:
			err := m.processTime()
			if err != nil {
				log.Error(log.TimeMgr, err)
			}
		}
	}
}

func (m *Manager) FetchNTPTime() time.Time {
	return ntpclient.NTPClient(m.pool)
}

func (m *Manager) processTime() error {
	NTPTime := m.FetchNTPTime()
	currentTime := time.Now()
	diff := NTPTime.Sub(currentTime)
	configNTPTime := m.allowedDifference
	negDiff := m.allowedNegativeDifference
	configNTPNegativeTime := -negDiff
	if diff > configNTPTime || diff < configNTPNegativeTime {
		log.Warnf(log.TimeMgr, "NTP manager: Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v\n",
			NTPTime,
			currentTime,
			diff,
			configNTPTime,
			configNTPNegativeTime)
	}
	return nil
}
