package engine

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
	if !errors.Is(err, communications.ErrNoRelayersEnabled) {
		t.Errorf("error '%v', expected '%v'", err, communications.ErrNoRelayersEnabled)
	}

	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SlackConfig: base.SlackConfig{
			Enabled: true,
		},
	})
	assert.NoError(t, err)

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

func TestStart(t *testing.T) {
	t.Parallel()
	m, err := SetupCommunicationManager(&base.CommunicationsConfig{
		SMTPConfig: base.SMTPConfig{
			Enabled: true,
		},
	})
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	_, err = m.GetStatus()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	err = m.Stop()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	err = m.Start()
	assert.NoError(t, err)

	m.PushEvent(base.Event{})
	m.PushEvent(base.Event{})
	m = nil
	m.PushEvent(base.Event{})
}
