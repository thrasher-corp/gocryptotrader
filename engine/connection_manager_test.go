package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestSetupConnectionManager(t *testing.T) {
	t.Parallel()
	_, err := setupConnectionManager(nil)
	assert.ErrorIs(t, err, errNilConfig)

	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	assert.NoError(t, err)

	if m == nil {
		t.Error("expected manager")
	}
}

func TestConnectionMonitorIsRunning(t *testing.T) {
	t.Parallel()
	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	if !m.IsRunning() {
		t.Error("expected true")
	}
	m.started = 0
	if m.IsRunning() {
		t.Error("expected false")
	}
	m = nil
	if m.IsRunning() {
		t.Error("expected false")
	}
}

func TestConnectionMonitorStart(t *testing.T) {
	t.Parallel()
	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Start()
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)

	m = nil
	err = m.Start()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestConnectionMonitorStop(t *testing.T) {
	t.Parallel()
	err := (&connectionManager{started: 1}).Stop()
	assert.ErrorIs(t, err, errConnectionCheckerIsNil)

	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)

	err = m.Stop()
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)
}

func TestConnectionMonitorIsOnline(t *testing.T) {
	t.Parallel()
	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	// If someone runs this offline, who are we to fail them?
	m.IsOnline()
	err = m.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if m.IsOnline() {
		t.Error("expected false")
	}
	m.conn = nil
	if m.IsOnline() {
		t.Error("expected false")
	}
	m = nil
	if m.IsOnline() {
		t.Error("expected false")
	}
}
