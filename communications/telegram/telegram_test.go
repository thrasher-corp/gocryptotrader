package telegram

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/communications/base"
	"github.com/thrasher-/gocryptotrader/config"
)

var T Telegram

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	T.Setup(cfg.GetCommunicationsConfig())
	if T.Name != "Telegram" || T.Enabled ||
		T.Token != "testest" || T.Verbose {
		t.Error("test failed - telegram Setup() error, unexpected setup values",
			T.Name, T.Enabled, T.Token, T.Verbose)
	}
}

func TestConnect(t *testing.T) {
	err := T.Connect()
	if err == nil {
		t.Error("test failed - telegram Connect() error")
	}
}

func TestPushEvent(t *testing.T) {
	err := T.PushEvent(base.Event{})
	if err != nil {
		t.Error("test failed - telegram PushEvent() error", err)
	}
	T.AuthorisedClients = append(T.AuthorisedClients, 1337)
	err = T.PushEvent(base.Event{})
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram PushEvent() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestHandleMessages(t *testing.T) {
	t.Parallel()
	chatID := int64(1337)
	err := T.HandleMessages(cmdHelp, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdStart, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdOrders, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdStatus, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdTicker, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdSettings, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages(cmdPortfolio, chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
	err = T.HandleMessages("Not a command", chatID)
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram HandleMessages() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestGetUpdates(t *testing.T) {
	t.Parallel()
	_, err := T.GetUpdates()
	if err != nil {
		t.Error("test failed - telegram GetUpdates() error", err)
	}
}

func TestTestConnection(t *testing.T) {
	t.Parallel()
	err := T.TestConnection()
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram TestConnection() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestSendMessage(t *testing.T) {
	t.Parallel()
	err := T.SendMessage("Test message", int64(1337))
	if err.Error() != "Not Found" {
		t.Errorf("test failed - telegram SendMessage() error, expected 'Not found' got '%s'",
			err)
	}
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	err := T.SendHTTPRequest("0.0.0.0", nil, nil)
	if err == nil {
		t.Error("test failed - telegram SendHTTPRequest() error")
	}
}
