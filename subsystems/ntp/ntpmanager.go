package ntp

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
	"github.com/thrasher-corp/gocryptotrader/subsystems/ntp/ntpclient"
)

// vars related to the NTP manager
var (
	NTPCheckInterval = time.Second * 30
	NTPRetryLimit    = 3
	errNTPDisabled   = errors.New("ntp client disabled")
)

// Manager starts the NTP manager
type Manager struct {
	started      int32
	initialCheck bool
	shutdown     chan struct{}
}

func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Start() error {
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}

	if engine.Bot.Config.NTPClient.Level == -1 {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
		return errors.New("NTP client disabled")
	}

	log.Debugln(log.TimeMgr, "NTP manager starting...")
	if engine.Bot.Config.NTPClient.Level == 0 && *engine.Bot.Config.Logging.Enabled {
		// Initial NTP check (prompts user on how we should proceed)
		m.initialCheck = true
		// Sometimes the NTP client can have transient issues due to UDP, try
		// the default retry limits before giving up
	check:
		for i := 0; i < NTPRetryLimit; i++ {
			err := m.processTime()
			switch err {
			case nil:
				break check
			case errNTPDisabled:
				log.Debugln(log.TimeMgr, "NTP manager: User disabled NTP prompts. Exiting.")
				atomic.CompareAndSwapInt32(&m.started, 1, 0)
				return nil
			default:
				if i == NTPRetryLimit-1 {
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
	t := time.NewTicker(NTPCheckInterval)
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
	return ntpclient.NTPClient(engine.Bot.Config.NTPClient.Pool)
}

func (m *Manager) processTime() error {
	NTPTime := m.FetchNTPTime()
	currentTime := time.Now()
	diff := NTPTime.Sub(currentTime)
	configNTPTime := *engine.Bot.Config.NTPClient.AllowedDifference
	negDiff := *engine.Bot.Config.NTPClient.AllowedNegativeDifference
	configNTPNegativeTime := -negDiff
	if diff > configNTPTime || diff < configNTPNegativeTime {
		log.Warnf(log.TimeMgr, "NTP manager: Time out of sync (NTP): %v | (time.Now()): %v | (Difference): %v | (Allowed): +%v / %v\n",
			NTPTime,
			currentTime,
			diff,
			configNTPTime,
			configNTPNegativeTime)
		if m.initialCheck {
			m.initialCheck = false
			disable, err := engine.Bot.Config.DisableNTPCheck(os.Stdin)
			if err != nil {
				return fmt.Errorf("unable to disable NTP check: %s", err)
			}
			log.Infoln(log.TimeMgr, disable)
			if engine.Bot.Config.NTPClient.Level == -1 {
				return errNTPDisabled
			}
		}
	}
	return nil
}
