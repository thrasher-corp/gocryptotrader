package engine

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := SetupCommunicationManager(nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = SetupCommunicationManager(&base.CommunicationsConfig{})
	if !errors.Is(err, communications.ErrNoCommunicationRelayersEnabled) {
		t.Errorf("error '%v', expected '%v'", err, communications.ErrNoCommunicationRelayersEnabled)
	}

	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SlackConfig: base.SlackConfig{
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
	t.Parallel()
	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SMSGlobalConfig: base.SMSGlobalConfig{
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
	t.Parallel()
	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SMTPConfig: base.SMTPConfig{
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
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemAlreadyStarted)
	}
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		TelegramConfig: base.TelegramConfig{
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
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
}

func TestStop(t *testing.T) {
	t.Parallel()
	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SlackConfig: base.SlackConfig{
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
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SlackConfig: base.SlackConfig{
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
	m.PushEvent(base.Event{})
	m = nil
	m.PushEvent(base.Event{})
}
