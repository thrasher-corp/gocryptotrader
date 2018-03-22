package platform

import (
	"testing"
)

var (
	newbot   *Bot
	exchName string
)

func TestSetupExchanges(t *testing.T) {
	newbot = GetBot(false, true, "")
	newbot.SetConfig()
	newbot.SetExchanges()
	exchName = "Bitfinex"
}

func TestCheckExchangeExists(t *testing.T) {
	if !newbot.CheckExchangeExists(exchName) {
		t.Error("test failed - exchange CheckExchangeExists() error")
	}
}

func TestGetExchangeByName(t *testing.T) {
	exch := newbot.GetExchangeByName(exchName)
	if exch.GetName() != exchName {
		t.Error("test failed - exchange GetExchangeByName() error")
	}
	exchTwo := newbot.GetExchangeByName("ning-nong")
	if exchTwo != nil {
		t.Error("test failed - exchange GetExchangeByName() error")
	}
}

func TestReloadExchange(t *testing.T) {
	if err := newbot.ReloadExchange(exchName); err != nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}

	if err := newbot.ReloadExchange("ning-nong"); err == nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}
}

func TestUnloadExchange(t *testing.T) {
	if err := newbot.UnloadExchange(exchName); err != nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}

	if err := newbot.UnloadExchange("ning-nong"); err == nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}
}

func TestLoadExchange(t *testing.T) {
	if err := newbot.LoadExchange(exchName); err != nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}

	if err := newbot.LoadExchange(exchName); err == nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}

	if err := newbot.LoadExchange("ning-nong"); err == nil {
		t.Error("test failed - exchange ReloadExchange() error", err)
	}
}
