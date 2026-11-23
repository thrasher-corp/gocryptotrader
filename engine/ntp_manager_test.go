package engine

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
)

// ntpTimestamp splits a time into NTP's seconds since 1900 and binary fraction of a second
func ntpTimestamp(ts time.Time) (secs, frac uint32) {
	sinceEpoch := ts.Sub(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC))
	fraction := sinceEpoch % time.Second << 32 / time.Second
	return uint32(sinceEpoch / time.Second), uint32(fraction) //nolint:gosec // NTP's 32-bit fields hold these until 2036
}

// newNTPServer answers NTP queries on a loopback UDP port with the time now returns, and returns its address
func newNTPServer(t *testing.T, now func() time.Time) string {
	t.Helper()
	var lc net.ListenConfig
	conn, err := lc.ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	require.NoError(t, err, "ListenPacket must not error")
	t.Cleanup(func() { assert.NoError(t, conn.Close(), "Close should not error") })
	go func() {
		req := make([]byte, 48)
		for {
			_, addr, err := conn.ReadFrom(req)
			if err != nil {
				return
			}
			secs, frac := ntpTimestamp(now())
			var rsp bytes.Buffer
			if err := binary.Write(&rsp, binary.BigEndian, &ntpPacket{Settings: 0x24, TxTimeSec: secs, TxTimeFrac: frac}); err != nil {
				return
			}
			if _, err := conn.WriteTo(rsp.Bytes(), addr); err != nil {
				return
			}
		}
	}()
	return conn.LocalAddr().String()
}

func TestSetupNTPManager(t *testing.T) {
	_, err := setupNTPManager(nil, false)
	assert.ErrorIs(t, err, errNilConfig)

	_, err = setupNTPManager(&config.NTPClientConfig{}, false)
	assert.ErrorIs(t, err, errNilNTPConfigValues)

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
	}
	m, err := setupNTPManager(cfg, false)
	assert.NoError(t, err)

	if m == nil {
		t.Error("expected manager")
	}
}

func TestNTPManagerIsRunning(t *testing.T) {
	var m *ntpManager
	if m.IsRunning() {
		t.Error("expected false")
	}

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err := setupNTPManager(cfg, false)
	assert.NoError(t, err)

	if m.IsRunning() {
		t.Error("expected false")
	}

	err = m.Start()
	if err != nil {
		t.Error(err)
	}
	if !m.IsRunning() {
		t.Error("expected true")
	}
}

func TestNTPManagerStart(t *testing.T) {
	var m *ntpManager
	err := m.Start()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
	}
	m, err = setupNTPManager(cfg, true)
	assert.NoError(t, err)

	err = m.Start()
	assert.ErrorIs(t, err, errNTPManagerDisabled)

	m.level = 1
	err = m.Start()
	assert.NoError(t, err)

	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)
}

func TestNTPManagerStop(t *testing.T) {
	var m *ntpManager
	err := m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err = setupNTPManager(cfg, true)
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)
}

func TestFetchNTPTime(t *testing.T) {
	var m *ntpManager
	_, err := m.FetchNTPTime()
	assert.ErrorIs(t, err, ErrNilSubsystem)

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err = setupNTPManager(cfg, true)
	assert.NoError(t, err)

	_, err = m.FetchNTPTime()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start()
	assert.NoError(t, err)

	tt, err := m.FetchNTPTime()
	assert.NoError(t, err)

	if tt.IsZero() {
		t.Error("expected time")
	}

	served := time.Date(2026, 1, 2, 3, 4, 5, 500_000_000, time.UTC)
	m.pools = []string{newNTPServer(t, func() time.Time { return served })}
	tt, err = m.FetchNTPTime()
	assert.NoError(t, err)
	assert.Truef(t, served.Equal(tt), "FetchNTPTime should return the server's time %s, got %s", served, tt)
}

func TestProcessTime(t *testing.T) {
	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
		Pool:                      []string{newNTPServer(t, time.Now)},
	}
	m, err := setupNTPManager(cfg, true)
	assert.NoError(t, err)

	err = m.processTime()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	err = m.Start()
	assert.NoError(t, err)

	err = m.processTime()
	assert.NoError(t, err)

	m.allowedDifference = time.Duration(1)
	m.allowedNegativeDifference = time.Duration(1)
	err = m.processTime()
	assert.NoError(t, err)
}
