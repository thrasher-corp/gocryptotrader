package engine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/config"
)

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

	m.pools = []string{"0.pool.ntp.org:123"}
	tt, err = m.FetchNTPTime()
	assert.NoError(t, err)

	if tt.IsZero() {
		t.Error("expected time")
	}
}

func TestProcessTime(t *testing.T) {
	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
		Pool:                      []string{"0.pool.ntp.org:123"},
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
