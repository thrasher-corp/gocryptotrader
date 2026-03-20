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

var (
	errInvalidNTPResponse  = errors.New("invalid NTP response")
	errInvalidNTPMode      = errors.New("invalid NTP mode")
	errZeroNTPTransmitTime = errors.New("zero NTP transmit timestamp")
	errZeroNTPReceiveTime  = errors.New("zero NTP receive timestamp")
	errInvalidNTPStratum   = errors.New("invalid NTP stratum")
)

var queryNTPOffsetFunc = queryNTPOffset

// checkNTPOffset performs a one-time NTP check and returns the measured offset.
// It is used during startup before the NTP manager loop begins.
func checkNTPOffset(ctx context.Context, pools []string) (time.Duration, error) {
	return queryNTPOffsetFunc(ctx, pools)
}

// queryNTPOffset centralises NTP offset measurement so startup checks and the
// long-running NTP manager use identical transport and calculation logic
func queryNTPOffset(ctx context.Context, pools []string) (time.Duration, error) {
	if len(pools) == 0 {
		return 0, errors.New("no NTP pools configured")
	}

	dialer := &net.Dialer{Timeout: ntpDialTimeout}

	for i := range pools {
		offset, err := queryNTPOffsetFromPool(ctx, dialer, pools[i])
		if err == nil {
			return offset, nil
		}
		log.Warnf(log.TimeMgr, "Unable to query NTP host %v: %v. Attempting next", pools[i], err)
	}
	return 0, errNoValidNTPServer
}

// queryNTPOffsetFromPool performs a single NTP exchange against one pool and
// returns the RFC 5905 offset derived from that response.
func queryNTPOffsetFromPool(ctx context.Context, dialer *net.Dialer, pool string) (time.Duration, error) {
	conn, err := dialer.DialContext(ctx, "udp", pool)
	if err != nil {
		return 0, fmt.Errorf("unable to connect to %v: %w", pool, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorln(log.TimeMgr, err)
		}
	}()

	if err := conn.SetDeadline(time.Now().Add(ntpReadWriteTimeout)); err != nil {
		return 0, fmt.Errorf("unable to set deadline on %v: %w", pool, err)
	}

	originTimestamp := time.Now()

	req := &ntpPacket{Settings: 0x1B}
	if err := binary.Write(conn, binary.BigEndian, req); err != nil {
		return 0, fmt.Errorf("unable to write request to %v: %w", pool, err)
	}

	rsp := &ntpPacket{}
	if err := binary.Read(conn, binary.BigEndian, rsp); err != nil {
		return 0, fmt.Errorf("unable to read response from %v: %w", pool, err)
	}

	destinationTimestamp := time.Now()

	if err := validateNTPResponse(rsp); err != nil {
		return 0, fmt.Errorf("invalid response from %v: %w", pool, err)
	}

	receiveTimestamp := ntpTimestampToTime(rsp.RxTimeSec, rsp.RxTimeFrac)
	transmitTimestamp := ntpTimestampToTime(rsp.TxTimeSec, rsp.TxTimeFrac)

	return calculateNTPOffset(originTimestamp, receiveTimestamp, transmitTimestamp, destinationTimestamp), nil
}

// ntpTimestampToTime converts timestamp (seconds and fractional) to time.Time
func ntpTimestampToTime(seconds, fractional uint32) time.Time {
	unixSeconds := int64(seconds) - ntpEpochOffset
	nanos := (int64(fractional) * 1.e9) >> 32
	return time.Unix(unixSeconds, nanos)
}

// calculateNTPOffset applies the RFC 5905 clock offset formula using the four
// timestamps involved in one NTP request/response exchange.
func calculateNTPOffset(origin, receive, transmit, destination time.Time) time.Duration {
	return (receive.Sub(origin) + transmit.Sub(destination)) / 2
}

// validateNTPResponse rejects obviously invalid server replies before their
// timestamps are trusted for offset calculation.
func validateNTPResponse(rsp *ntpPacket) error {
	if rsp == nil {
		return errInvalidNTPResponse
	}

	if rsp.Settings&0x07 != 4 {
		return errInvalidNTPMode
	}

	if rsp.Stratum == 0 {
		return errInvalidNTPStratum
	}

	if rsp.RxTimeSec == 0 && rsp.RxTimeFrac == 0 {
		return errZeroNTPReceiveTime
	}

	if rsp.TxTimeSec == 0 && rsp.TxTimeFrac == 0 {
		return errZeroNTPTransmitTime
	}

	return nil
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

// continuously checks the internet connection at intervals
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
		log.Warnf(log.TimeMgr, "NTP manager: Time out of sync (Offset): %v | (Allowed) +%v / %v", offset, configNTPTime, configNTPNegativeTime)
	}
	return nil
}

// getTimeOffset returns the measured NTP offset for the manager's configured pools
func (m *ntpManager) getTimeOffset(ctx context.Context) (time.Duration, error) {
	return queryNTPOffsetFunc(ctx, m.pools)
}
