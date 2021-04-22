package ntpmanager

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystems"
)

func TestSetup(t *testing.T) {
	_, err := Setup(nil, false)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}
	_, err = Setup(&config.NTPClientConfig{}, false)
	if !errors.Is(err, errNilConfigValues) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfigValues)
	}
	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
	}
	m, err := Setup(cfg, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsRunning(t *testing.T) {
	var m *Manager
	if m.IsRunning() {
		t.Error("expected false")
	}

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err := Setup(cfg, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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

func TestStart(t *testing.T) {
	var m *Manager
	err := m.Start()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
	}
	m, err = Setup(cfg, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, errNTPManagerDisabled) {
		t.Errorf("error '%v', expected '%v'", err, errNTPManagerDisabled)
	}

	m.level = 1
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}
}

func TestStop(t *testing.T) {
	var m *Manager
	err := m.Stop()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}

	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err = Setup(cfg, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}

	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestFetchNTPTime(t *testing.T) {
	var m *Manager
	_, err := m.FetchNTPTime()
	if !errors.Is(err, subsystems.ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrNilSubsystem)
	}
	sec := time.Second
	cfg := &config.NTPClientConfig{
		AllowedDifference:         &sec,
		AllowedNegativeDifference: &sec,
		Level:                     1,
	}
	m, err = Setup(cfg, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.FetchNTPTime()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}

	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	tt, err := m.FetchNTPTime()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if tt.IsZero() {
		t.Error("expected time")
	}

	m.pools = []string{"0.pool.ntp.org:123"}
	tt, err = m.FetchNTPTime()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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
	m, err := Setup(cfg, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.processTime()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}

	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.processTime()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.allowedDifference = time.Duration(1)
	m.allowedNegativeDifference = time.Duration(1)
	err = m.processTime()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}
