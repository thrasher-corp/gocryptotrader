package communicationmanager

import (
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/subsystems"
)

func TestSetup(t *testing.T) {
	_, err := Setup(nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = Setup(&config.CommunicationsConfig{})
	if !errors.Is(err, communications.ErrNoCommunicationRelayersEnabled) {
		t.Errorf("error '%v', expected '%v'", err, communications.ErrNoCommunicationRelayersEnabled)
	}

	m, err := Setup(&config.CommunicationsConfig{
		SlackConfig: config.SlackConfig{
			Enabled: true,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestIsRunning(t *testing.T) {
	m, err := Setup(&config.CommunicationsConfig{
		SlackConfig: config.SlackConfig{
			Enabled: true,
		},
	})
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
	m, err := Setup(&config.CommunicationsConfig{
		SlackConfig: config.SlackConfig{
			Enabled: true,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.started = 1
	err = m.Start()
	if !errors.Is(err, subsystems.ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemAlreadyStarted)
	}
}

func TestGetStatus(t *testing.T) {
	m, err := Setup(&config.CommunicationsConfig{
		SlackConfig: config.SlackConfig{
			Enabled: true,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.GetStatus()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.started = 0
	_, err = m.GetStatus()
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}
}

func TestStop(t *testing.T) {
	m, err := Setup(&config.CommunicationsConfig{
		SlackConfig: config.SlackConfig{
			Enabled: true,
		},
	})
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
	if !errors.Is(err, subsystems.ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, subsystems.ErrSubSystemNotStarted)
	}
}

func TestPushEvent(t *testing.T) {
	m, err := Setup(&config.CommunicationsConfig{
		SlackConfig: config.SlackConfig{
			Enabled: true,
		},
	})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.PushEvent(base.Event{})
	time.Sleep(time.Second)
	m.PushEvent(base.Event{})
	m = nil
	m.PushEvent(base.Event{})
}
