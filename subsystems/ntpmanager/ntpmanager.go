package ntpmanager

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

func Setup(cfg *config.NTPClientConfig, loggingEnabled bool) (*Manager, error) {
	if cfg == nil {
		return nil, errNilConfig
	}
	if cfg.AllowedNegativeDifference == nil ||
		cfg.AllowedDifference == nil {
		return nil, errNilConfigValues
	}
	return &Manager{
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

func (m *Manager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

func (m *Manager) Start() error {
	if m == nil {
		return fmt.Errorf("ntp manager %w", subsystems.ErrNilSubsystem)
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemAlreadyStarted)
	}
	if m.level == 0 && m.loggingEnabled {
		// Sometimes the NTP client can have transient issues due to UDP, try
		// the default retry limits before giving up
	check:
		for i := 0; i < m.retryLimit; i++ {
			err := m.processTime()
			switch err {
			case nil:
				break check
			case subsystems.ErrSubSystemNotStarted:
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
	log.Debugf(log.TimeMgr, "NTP manager %s", subsystems.MsgSubSystemStarted)
	return nil
}

func (m *Manager) Stop() error {
	if m == nil {
		return fmt.Errorf("ntp manager %w", subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemNotStarted)
	}
	defer func() {
		log.Debugf(log.TimeMgr, "NTP manager %s", subsystems.MsgSubSystemShutdown)
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
	}()
	log.Debugf(log.TimeMgr, "NTP manager %s", subsystems.MsgSubSystemShuttingDown)
	close(m.shutdown)
	return nil
}

func (m *Manager) run() {
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
				log.Error(log.TimeMgr, err)
			}
		}
	}
}

func (m *Manager) FetchNTPTime() (time.Time, error) {
	if m == nil {
		return time.Time{}, fmt.Errorf("ntp manager %w", subsystems.ErrNilSubsystem)
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return time.Time{}, fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemNotStarted)
	}
	return checkTimeInPools(m.pools), nil
}

func (m *Manager) processTime() error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("NTP manager %w", subsystems.ErrSubSystemNotStarted)
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
func checkTimeInPools(pool []string) time.Time {
	for i := range pool {
		con, err := net.DialTimeout("udp", pool[i], 5*time.Second)
		if err != nil {
			log.Warnf(log.TimeMgr, "Unable to connect to hosts %v attempting next", pool[i])
			continue
		}

		if err := con.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Warnf(log.TimeMgr, "Unable to SetDeadline. Error: %s\n", err)
			err = con.Close()
			if err != nil {
				log.Error(log.TimeMgr, err)
			}
			continue
		}

		req := &ntpPacket{Settings: 0x1B}
		if err := binary.Write(con, binary.BigEndian, req); err != nil {
			log.Warnf(log.TimeMgr, "Unable to write. Error: %s\n", err)
			err = con.Close()
			if err != nil {
				log.Error(log.TimeMgr, err)
			}
			continue
		}

		rsp := &ntpPacket{}
		if err := binary.Read(con, binary.BigEndian, rsp); err != nil {
			log.Warnf(log.TimeMgr, "Unable to read. Error: %s\n", err)
			err = con.Close()
			if err != nil {
				log.Error(log.TimeMgr, err)
			}
			continue
		}

		secs := float64(rsp.TxTimeSec) - 2208988800
		nanos := (int64(rsp.TxTimeFrac) * 1e9) >> 32

		err = con.Close()
		if err != nil {
			log.Error(log.TimeMgr, err)
		}
		return time.Unix(int64(secs), nanos)
	}
	log.Warnln(log.TimeMgr, "No valid NTP servers found, using current system time")
	return time.Now().UTC()
}
