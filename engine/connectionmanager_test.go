package engine

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystems"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := SetupConnectionManager(nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	m, err := SetupConnectionManager(&config.ConnectionMonitorConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsRunning(t *testing.T) {
	t.Parallel()
	m, err := SetupConnectionManager(&config.ConnectionMonitorConfig{})
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

func TestStart(t *testing.T) {
	t.Parallel()
	m, err := SetupConnectionManager(&config.ConnectionMonitorConfig{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}
	m = nil
	err = m.Start()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}

func TestStop(t *testing.T) {
	t.Parallel()
	m, err := SetupConnectionManager(&config.ConnectionMonitorConfig{})
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
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
}

func TestIsOnline(t *testing.T) {
	t.Parallel()
	m, err := SetupConnectionManager(&config.ConnectionMonitorConfig{})
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
