package ntpmanager

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

func (m *Manager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

func Setup(cfg *config.NTPClientConfig, loggingEnabled bool) (*Manager, error) {
	m := &Manager{
		started:                   1,
		shutdown:                  make(chan struct{}),
		level:                     int64(cfg.Level),
		allowedDifference:         *cfg.AllowedDifference,
		allowedNegativeDifference: *cfg.AllowedNegativeDifference,
		pools:                     cfg.Pool,
		checkInterval:             defaultNTPCheckInterval,
		retryLimit:                defaultRetryLimit,
	}

	if cfg.Level != 1 {
		atomic.CompareAndSwapInt32(&m.started, 1, 0)
		return nil, errors.New("NTP client disabled")
	}

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
				return m, nil
			default:
				if i == m.retryLimit-1 {
					return m, err
				}
			}
		}
	}
	go m.run()
	log.Debugln(log.TimeMgr, "NTP manager started.")
	return m, nil
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
	return checkTimeInPools(m.pools)
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
