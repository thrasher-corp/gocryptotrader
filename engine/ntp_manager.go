package engine

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// setupNTPManager creates a new NTP manager
func setupNTPManager(cfg *config.NTPClientConfig, loggingEnabled bool) (*ntpManager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if cfg.AllowedNegativeDifference == nil ||
		cfg.AllowedDifference == nil {
		return nil, errNilNTPConfigValues
	}
	return &ntpManager{
		shutdown:                  make(chan struct{}),
		level:                     int64(cfg.Level),
		allowedDifference:         *cfg.AllowedDifference,
		allowedNegativeDifference: *cfg.AllowedNegativeDifference,
		pools:                     cfg.Pool,
		checkInterval:             defaultNTPCheckInterval,
		retryLimit:                defaultRetryLimit,
		loggingEnabled:            loggingEnabled,
	}, nil
}

// IsRunning safely checks whether the subsystem is running
func (m *ntpManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Start runs the subsystem
func (m *ntpManager) Start() error {
	if m == nil {
		return fmt.Errorf("ntp manager %w", ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("NTP manager %w", ErrSubSystemAlreadyStarted)
	}
	if m.level == 0 && m.loggingEnabled {
		// Sometimes the NTP client can have transient issues due to UDP, try
		// the default retry limits before giving up
	check:
		for i := range m.retryLimit {
			err := m.processTime()
			switch err {
			case nil:
				break check
			case ErrSubSystemNotStarted:
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
	if m.level != 1 {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
		return errNTPManagerDisabled
	}
	m.shutdown = make(chan struct{})
	go m.run()
	log.Debugf(log.TimeMgr, "NTP manager %s", MsgSubSystemStarted)
	return nil
}

// Stop attempts to shutdown the subsystem
func (m *ntpManager) Stop() error {
	if m == nil {
		return fmt.Errorf("ntp manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	defer func() {
		log.Debugf(log.TimeMgr, "NTP manager %s", MsgSubSystemShutdown)
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()
	log.Debugf(log.TimeMgr, "NTP manager %s", MsgSubSystemShuttingDown)
	close(m.shutdown)
	return nil
}

// run continuously checks the internet connection at intervals
func (m *ntpManager) run() {
	t := time.NewTicker(m.checkInterval)
	defer func() {
		t.Stop()
	}()

	for {
		select {
		case <-m.shutdown:
			return
		case <-t.C:
			err := m.processTime()
			if err != nil {
				log.Errorln(log.TimeMgr, err)
			}
		}
	}
}

// FetchNTPTime returns the time from defined NTP pools
func (m *ntpManager) FetchNTPTime() (time.Time, error) {
	if m == nil {
		return time.Time{}, fmt.Errorf("ntp manager %w", ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return time.Time{}, fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	return m.checkTimeInPools(), nil
}

// processTime determines the difference between system time and NTP time
// to discover discrepancies
func (m *ntpManager) processTime() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	NTPTime, err := m.FetchNTPTime()
	if err != nil {
		return err
	}
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

// checkTimeInPools returns local based on ntp servers provided timestamp
// if no server can be reached will return local time in UTC()
func (m *ntpManager) checkTimeInPools() time.Time {
	for i := range m.pools {
		con, err := net.DialTimeout("udp", m.pools[i], 5*time.Second) //nolint:noctx // TODO: #2006 Use (*net.Dialer).DialContext with (*net.Dialer).Timeout
		if err != nil {
			log.Warnf(log.TimeMgr, "Unable to connect to hosts %v attempting next", m.pools[i])
			continue
		}

		if err = con.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Warnf(log.TimeMgr, "Unable to SetDeadline. Error: %s\n", err)
			err = con.Close()
			if err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		req := &ntpPacket{Settings: 0x1B}
		if err = binary.Write(con, binary.BigEndian, req); err != nil {
			log.Warnf(log.TimeMgr, "Unable to write. Error: %s\n", err)
			err = con.Close()
			if err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		rsp := &ntpPacket{}
		if err = binary.Read(con, binary.BigEndian, rsp); err != nil {
			log.Warnf(log.TimeMgr, "Unable to read. Error: %s\n", err)
			err = con.Close()
			if err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		secs := float64(rsp.TxTimeSec) - 2208988800
		nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32

		err = con.Close()
		if err != nil {
			log.Errorln(log.TimeMgr, err)
		}
		return time.Unix(int64(secs), nanos)
	}
	log.Warnln(log.TimeMgr, "No valid NTP servers found, using current system time")
	return time.Now().UTC()
}
