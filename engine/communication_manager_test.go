package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/communications"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
)

// startCommunicationManager starts m and stops its run routine when the test ends, whatever the test does to the
// started flag
func startCommunicationManager(t *testing.T, m *CommunicationManager) {
	t.Helper()
	require.NoError(t, m.Start(), "Start must not error")
	t.Cleanup(func() {
		m.started.Store(true)
		assert.NoError(t, m.Stop(), "Stop should not error")
	})
}

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

	startCommunicationManager(t, m)

	if !m.IsRunning() {
		t.Error("expected true")
	}
	m.started.Store(false)
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

	startCommunicationManager(t, m)

	m.started.Store(true)
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

	startCommunicationManager(t, m)

	_, err = m.GetStatus()
	assert.NoError(t, err)

	m.started.Store(false)
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

	startCommunicationManager(t, m)

	m.PushEvent(base.Event{})
	m.PushEvent(base.Event{})
	m = nil
	m.PushEvent(base.Event{})
}
