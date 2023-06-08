package telegram

import (
	"errors"
	"testing"

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
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected %s, got %s", ErrNotConnected, err)
	}

	T.Connected = true
	T.AuthorisedClients = map[string]int64{"sender": 0}
	err = T.PushEvent(base.Event{})
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}

	T.AuthorisedClients = map[string]int64{"sender": 1337}
	err = T.PushEvent(base.Event{})
	if err.Error() != testErrNotFound {
		t.Errorf("telegram PushEvent() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestHandleMessages(t *testing.T) {
	t.Parallel()
	var T Telegram
	chatID := int64(1337)
	err := T.HandleMessages(cmdHelp, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdStart, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdStatus, chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages("Not a command", chatID)
	if err.Error() != testErrNotFound {
		t.Errorf("telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
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
	err := T.SendMessage("Test message", int64(1337))
	if err.Error() != testErrNotFound {
		t.Errorf("telegram SendMessage() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	var T Telegram
	err := T.SendHTTPRequest("0.0.0.0", nil, nil)
	if err == nil {
		t.Error("telegram SendHTTPRequest() error")
	}
}
