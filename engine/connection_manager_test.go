package engine

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
)

func TestSetupConnectionManager(t *testing.T) {
	t.Parallel()
	_, err := setupConnectionManager(nil)
	if !errors.Is(err, subsystem.ErrNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNilConfig)
	}

	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestConnectionMonitorIsRunning(t *testing.T) {
	t.Parallel()
	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, subsystem.ErrAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrAlreadyStarted)
	}
	m = nil
	err = m.Start()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}
}

func TestConnectionMonitorStop(t *testing.T) {
	t.Parallel()
	err := (&connectionManager{started: 1}).Stop()
	if !errors.Is(err, errConnectionCheckerIsNil) {
		t.Errorf("error '%v', expected '%v'", err, errConnectionCheckerIsNil)
	}
	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, subsystem.ErrNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, subsystem.ErrNil) {
		t.Errorf("error '%v', expected '%v'", err, subsystem.ErrNil)
	}
}

func TestConnectionMonitorIsOnline(t *testing.T) {
	t.Parallel()
	m, err := setupConnectionManager(&config.ConnectionMonitorConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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
