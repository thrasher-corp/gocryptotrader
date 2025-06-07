package telegram

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/communications/base"
	"github.com/thrasher-corp/gocryptotrader/config"
)

const (
	testErrNotFound = "Not Found"
)

func TestSetup(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Communications: base.CommunicationsConfig{
		TelegramConfig: base.TelegramConfig{
			Name:              "Telegram",
			Enabled:           false,
			Verbose:           false,
			VerificationToken: "testest",
			AuthorisedClients: map[string]int64{"sender": 0},
		},
	}}
	commsCfg := cfg.GetCommunicationsConfig()
	var T Telegram
	T.Setup(&commsCfg)
	if T.Name != "Telegram" || T.Enabled || T.Token != "testest" || T.Verbose || len(T.AuthorisedClients) != 1 {
		t.Error("telegram Setup() error, unexpected setup values",
			T.Name,
			T.Enabled,
			T.Token,
			T.Verbose)
	}
}

func TestConnect(t *testing.T) {
	t.Parallel()
	var T Telegram
	if err := T.Connect(); err == nil {
		t.Error("expected error")
	}
}

func TestPushEvent(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.PushEvent(base.Event{})
	assert.ErrorIs(t, err, ErrNotConnected)

	T.Connected = true
	T.AuthorisedClients = map[string]int64{"sender": 0}
	err = T.PushEvent(base.Event{})
	assert.NoError(t, err, "PushEvent should not error")

	T.AuthorisedClients = map[string]int64{"sender": 1337}
	err = T.PushEvent(base.Event{})
	assert.ErrorContains(t, err, testErrNotFound)
}

func TestHandleMessages(t *testing.T) {
	t.Parallel()
	var T Telegram
	for _, c := range []string{cmdHelp, cmdStart, cmdStatus, "Not a command"} {
		assert.ErrorContainsf(t, T.HandleMessages(c, 1337), testErrNotFound,
			"HandleMessages with command %q should error correctly", c)
	}
}

func TestGetUpdates(t *testing.T) {
	t.Parallel()
	var T Telegram
	if _, err := T.GetUpdates(); err != nil {
		t.Error("telegram GetUpdates() error", err)
	}
}

func TestTestConnection(t *testing.T) {
	t.Parallel()
	var T Telegram
	if err := T.TestConnection(); err.Error() != testErrNotFound {
		t.Errorf("received %s, expected: %s", err, testErrNotFound)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.SendMessage("Test message", 1337)
	assert.ErrorContains(t, err, testErrNotFound, "SendMessage should error correctly")
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.SendHTTPRequest("0.0.0.0", nil, nil)
	if err == nil {
		t.Error("telegram SendHTTPRequest() error")
	}
}
