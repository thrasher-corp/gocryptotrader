package main

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var testSetup = false

func SetupTest(t *testing.T) {
	if !testSetup {
		bot.config = &config.Cfg
		err := bot.config.LoadConfig("./testdata/configtest.json")
		if err != nil {
			t.Fatalf("Test failed. SetupTest: Failed to load config: %s", err)
		}
		testSetup = true
	}

	if CheckExchangeExists("Bitfinex") {
		return
	}
	err := LoadExchange("Bitfinex")
	if err != nil {
		t.Errorf("Test failed. SetupTest: Failed to load exchange: %s", err)
	}
}

func CleanupTest(t *testing.T) {
	if !CheckExchangeExists("Bitfinex") {
		return
	}

	err := UnloadExchange("Bitfinex")
	if err != nil {
		t.Fatalf("Test failed. CleanupTest: Failed to unload exchange: %s",
			err)
	}
}

func TestCheckExchangeExists(t *testing.T) {
	SetupTest(t)

	if !CheckExchangeExists("Bitfinex") {
		t.Errorf("Test failed. TestGetExchangeExists: Unable to find exchange")
	}

	if CheckExchangeExists("Asdsad") {
		t.Errorf("Test failed. TestGetExchangeExists: Non-existant exchange found")
	}

	CleanupTest(t)
}

func TestGetExchangeByName(t *testing.T) {
	SetupTest(t)

	exch := GetExchangeByName("Bitfinex")
	if exch == nil {
		t.Errorf("Test failed. TestGetExchangeByName: Failed to get exchange")
	}

	if !exch.IsEnabled() {
		t.Errorf("Test failed. TestGetExchangeByName: Unexpected result")
	}

	exch.SetEnabled(false)
	bfx := GetExchangeByName("Bitfinex")
	if bfx.IsEnabled() {
		t.Errorf("Test failed. TestGetExchangeByName: Unexpected result")
	}

	if exch.GetName() != "Bitfinex" {
		t.Errorf("Test failed. TestGetExchangeByName: Unexpected result")
	}

	exch = GetExchangeByName("Asdasd")
	if exch != nil {
		t.Errorf("Test failed. TestGetExchangeByName: Non-existant exchange found")
	}

	CleanupTest(t)
}

func TestReloadExchange(t *testing.T) {
	SetupTest(t)

	err := ReloadExchange("asdf")
	if err != ErrExchangeNotFound {
		t.Errorf("Test failed. TestReloadExchange: Incorrect result: %s",
			err)
	}

	err = ReloadExchange("Bitfinex")
	if err != nil {
		t.Errorf("Test failed. TestReloadExchange: Incorrect result: %s",
			err)
	}

	CleanupTest(t)

	err = ReloadExchange("asdf")
	if err != ErrNoExchangesLoaded {
		t.Errorf("Test failed. TestReloadExchange: Incorrect result: %s",
			err)
	}
}

func TestUnloadExchange(t *testing.T) {
	SetupTest(t)

	err := UnloadExchange("asdf")
	if err != ErrExchangeNotFound {
		t.Errorf("Test failed. TestUnloadExchange: Incorrect result: %s",
			err)
	}

	err = UnloadExchange("Bitfinex")
	if err != nil {
		t.Errorf("Test failed. TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = UnloadExchange("asdf")
	if err != ErrNoExchangesLoaded {
		t.Errorf("Test failed. TestUnloadExchange: Incorrect result: %s",
			err)
	}

	CleanupTest(t)
}

func TestSetupExchanges(t *testing.T) {
	SetupTest(t)
	SetupExchanges()
	CleanupTest(t)
}
