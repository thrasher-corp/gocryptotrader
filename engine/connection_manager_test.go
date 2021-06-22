package engine

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

func TestSetupConnectionManager(t *testing.T) {
	t.Parallel()
	_, err := setupConnectionManager(nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
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
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemAlreadyStarted)
	}
	m = nil
	err = m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestConnectionMonitorStop(t *testing.T) {
	t.Parallel()
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
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
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
