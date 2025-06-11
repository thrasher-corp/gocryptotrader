package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	_, err := SetupCommunicationManager(nil)
	assert.ErrorIs(t, err, errNilConfig)

	_, err = SetupCommunicationManager(&base.CommunicationsConfig{})
	assert.ErrorIs(t, err, communications.ErrNoRelayersEnabled)

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
	assert.ErrorIs(t, err, ErrSubSystemAlreadyStarted)
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
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)
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
	assert.ErrorIs(t, err, ErrSubSystemNotStarted)

	m = nil
	err = m.Stop()
	assert.ErrorIs(t, err, ErrNilSubsystem)
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
