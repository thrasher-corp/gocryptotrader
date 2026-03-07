package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	ntpEpochOffset      = 2208988800
	ntpDialTimeout      = 5 * time.Second
	ntpReadWriteTimeout = 5 * time.Second
)

// errNoValidNTPServer is returned when no valid NTP server could be reached
var errNoValidNTPServer = errors.New("no valid NTP server could be reached")

// CheckNTPOffset performs a one-time NTP check and returns the time offset.
// This can be called before the NTP manager is started to verify time sync.
// It uses the RFC 5905 formula: offset = ((T2-T1) + (T3-T4)) / 2
func CheckNTPOffset(ctx context.Context, pools []string) (time.Duration, error) {
	if len(pools) == 0 {
		return 0, errors.New("no NTP pools configured")
	}

	dialer := &net.Dialer{
		Timeout: ntpDialTimeout,
	}

	for i := range pools {
		conn, err := dialer.DialContext(ctx, "udp", pools[i])
		if err != nil {
			log.Warnf(log.TimeMgr, "NTP check: Unable to connect to %v, attempting next", pools[i])
			continue
		}

		if err = conn.SetDeadline(time.Now().Add(ntpReadWriteTimeout)); err != nil {
			log.Warnf(log.TimeMgr, "NTP check: Unable to set deadline on %v. Error %s. Attempting next\n", pools[i], err)
			if err = conn.Close(); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		// T1: Record time before sending request (origin timestamp)
		t1 := time.Now()

		req := &ntpPacket{Settings: 0x1B}
		if err = binary.Write(conn, binary.BigEndian, req); err != nil {
			log.Warnf(log.TimeMgr, "NTP check: Unable to write to %v. Error %s. Attempting next\n", pools[i], err)
			if err = conn.Close(); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		rsp := &ntpPacket{}
		if err = binary.Read(conn, binary.BigEndian, rsp); err != nil {
			log.Warnf(log.TimeMgr, "NTP check: Unable to read from %v. Error: %s. Attempting next\n", pools[i], err)
			if err = conn.Close(); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		// T4: Record time after receiving response (Destination timestamp)
		t4 := time.Now()

		if err = conn.Close(); err != nil {
			log.Errorln(log.TimeMgr, err)
		}

		// T2: Server receive timestamp (when server received our request)
		t2 := ntpTimestampToTime(rsp.RxTimeSec, rsp.RxTimeFrac)
		// T3: Server transmit timestamp (when server sent our response)
		t3 := ntpTimestampToTime(rsp.TxTimeSec, rsp.TxTimeFrac)

		// RFC 5905 offset calculation: ((T2-T1) + (T3-T4)) / 2
		// This formula cancels out the network round-trip time
		offset := (t2.Sub(t1) + t3.Sub(t4)) / 2
		return offset, nil
	}
	return 0, errNoValidNTPServer
}

// ntpTimestampToTime converts timestamp (seconds and fractional) to time.Time
func ntpTimestampToTime(seconds, fractional uint32) time.Time {
	unixSeconds := int64(seconds) - ntpEpochOffset
	nanos := (int64(fractional) * 1.e9) >> 32
	return time.Unix(unixSeconds, nanos)
}

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
			err := m.processTime(context.Background())
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
			err := m.processTime(context.Background())
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
	offset, err := m.getTimeOffset(context.Background())
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(offset), nil
}

// processTime determines the difference between system time and NTP time to discover discrepancies
func (m *ntpManager) processTime(ctx context.Context) error {
	if atomic.LoadInt32(&m.started) == 0 {
		return fmt.Errorf("NTP manager %w", ErrSubSystemNotStarted)
	}
	offset, err := m.getTimeOffset(ctx)
	if err != nil {
		return err
	}
	configNTPTime := m.allowedDifference
	negDiff := m.allowedNegativeDifference
	configNTPNegativeTime := -negDiff
	if offset > configNTPTime || offset < configNTPNegativeTime {
		log.Warnf(log.TimeMgr, "NTP manager: Time out of sync (Offset): %v | (Allowed) +%v / %v\n", offset, configNTPTime, configNTPNegativeTime)
	}
	return nil
}

// getTimeOffset queries NTP servers and returns the calculated time offset
// using the RFC5905 formula: offset = ((T2-T1) + (T3-T4)) / 2
// This properly accounts for network round-trip time
func (m *ntpManager) getTimeOffset(ctx context.Context) (time.Duration, error) {
	dialer := &net.Dialer{
		Timeout: ntpDialTimeout,
	}

	for i := range m.pools {
		conn, err := dialer.DialContext(ctx, "udp", m.pools[i])
		if err != nil {
			log.Warnf(log.TimeMgr, "Unable to connect to hosts %v attempting to next", m.pools[i])
			continue
		}

		if err = conn.SetDeadline(time.Now().Add(ntpReadWriteTimeout)); err != nil {
			log.Warnf(log.TimeMgr, "Unable to set deadline on hosts %v. Error %s. attempting to next\n", m.pools[i], err)
			if err = conn.Close(); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		// T1: Record time before sending request (origin timestamp)
		t1 := time.Now()

		req := &ntpPacket{Settings: 0x1B}
		if err = binary.Write(conn, binary.BigEndian, req); err != nil {
			log.Warnf(log.TimeMgr, "Unable to write to hosts %v. Error %s. Attempting to next\n", m.pools[i], err)
			if err = conn.Close(); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		rsp := &ntpPacket{}
		if err = binary.Read(conn, binary.BigEndian, rsp); err != nil {
			log.Warnf(log.TimeMgr, "Unable to read from hosts %v. Error: %s. Attempting to next\n", m.pools[i], err)
			if err = conn.Close(); err != nil {
				log.Errorln(log.TimeMgr, err)
			}
			continue
		}

		// T4L Record time after receiving response (Destination timestamp)
		t4 := time.Now()

		if err = conn.Close(); err != nil {
			log.Errorln(log.TimeMgr, err)
		}

		// T2: Server receive timestamp (when server received our request)
		t2 := ntpTimestampToTime(rsp.RxTimeSec, rsp.RxTimeFrac)
		// T3: Server transmit timestamp (when server sent our response)
		t3 := ntpTimestampToTime(rsp.TxTimeSec, rsp.TxTimeFrac)

		// RFC 5905 offset calculation: ((T2-T1) + (T3-T4)) / 2
		// This formula cancels out the network round-trip time
		offset := (t2.Sub(t1) + t3.Sub(t4)) / 2
		return offset, nil
	}
	log.Warnln(log.TimeMgr, "No valid NTP servers found")
	return 0, errNoValidNTPServer
}
