package engine

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
)

var testSetup = false

func SetupTest(t *testing.T) {
	if !testSetup {
		if Bot == nil {
			Bot = new(Engine)
		}
		Bot.Config = &config.Cfg
		err := Bot.Config.LoadConfig("", true)
		if err != nil {
			t.Fatalf("SetupTest: Failed to load config: %s", err)
		}
		testSetup = true
	}

	if GetExchangeByName(testExchange) != nil {
		return
	}
	err := LoadExchange(testExchange, false, nil)
	if err != nil {
		t.Errorf("SetupTest: Failed to load exchange: %s", err)
	}
}

func CleanupTest(t *testing.T) {
	if GetExchangeByName(testExchange) == nil {
		return
	}

	err := UnloadExchange(testExchange)
	if err != nil {
		t.Fatalf("CleanupTest: Failed to unload exchange: %s",
			err)
	}
}

func TestCheckExchangeExists(t *testing.T) {
	SetupTest(t)

	if GetExchangeByName(testExchange) == nil {
		t.Errorf("TestGetExchangeExists: Unable to find exchange")
	}

	if GetExchangeByName("Asdsad") != nil {
		t.Errorf("TestGetExchangeExists: Non-existent exchange found")
	}

	CleanupTest(t)
}

func TestGetExchangeByName(t *testing.T) {
	SetupTest(t)

	exch := GetExchangeByName(testExchange)
	if exch == nil {
		t.Errorf("TestGetExchangeByName: Failed to get exchange")
	}

	if !exch.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch.SetEnabled(false)
	bfx := GetExchangeByName(testExchange)
	if bfx.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	if exch.GetName() != testExchange {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch = GetExchangeByName("Asdasd")
	if exch != nil {
		t.Errorf("TestGetExchangeByName: Non-existent exchange found")
	}

	CleanupTest(t)
}

func TestReloadExchange(t *testing.T) {
	SetupTest(t)

	err := ReloadExchange("asdf")
	if err != ErrExchangeNotFound {
		t.Errorf("TestReloadExchange: Incorrect result: %s",
			err)
	}

	err = ReloadExchange(testExchange)
	if err != nil {
		t.Errorf("TestReloadExchange: Incorrect result: %s",
			err)
	}

	CleanupTest(t)

	err = ReloadExchange("asdf")
	if err != ErrNoExchangesLoaded {
		t.Errorf("TestReloadExchange: Incorrect result: %s",
			err)
	}
}

func TestUnloadExchange(t *testing.T) {
	SetupTest(t)

	err := UnloadExchange("asdf")
	if err != ErrExchangeNotFound {
		t.Errorf("TestUnloadExchange: Incorrect result: %s",
			err)
	}

	err = UnloadExchange(testExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = UnloadExchange("asdf")
	if err != ErrNoExchangesLoaded {
		t.Errorf("TestUnloadExchange: Incorrect result: %s",
			err)
	}

	CleanupTest(t)
}

func TestDryRunParamInteraction(t *testing.T) {
	SetupTest(t)

	// Load bot as per normal, dry run and verbose for Bitfinex should be
	// disabled
	exchCfg, err := Bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}

	if Bot.Settings.EnableDryRun ||
		exchCfg.Verbose {
		t.Error("dryrun and verbose should have been disabled")
	}

	// Simulate overiding default settings and ensure that enabling exchange
	// verbose mode will be set on Bitfinex
	if err = UnloadExchange(testExchange); err != nil {
		t.Error(err)
	}

	Bot.Settings.CheckParamInteraction = true
	Bot.Settings.EnableExchangeVerbose = true
	if err = LoadExchange(testExchange, false, nil); err != nil {
		t.Error(err)
	}

	exchCfg, err = Bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}

	if !Bot.Settings.EnableDryRun ||
		!exchCfg.Verbose {
		t.Error("dryrun and verbose should have been enabled")
	}

	if err = UnloadExchange(testExchange); err != nil {
		t.Error(err)
	}

	// Now set dryrun mode to false (via flagset and the previously enabled
	// setting), enable exchange verbose mode and verify that verbose mode
	// will be set on Bitfinex
	Bot.Settings.EnableDryRun = false
	Bot.Settings.CheckParamInteraction = true
	Bot.Settings.EnableExchangeVerbose = true
	flagSet["dryrun"] = true
	if err = LoadExchange(testExchange, false, nil); err != nil {
		t.Error(err)
	}

	exchCfg, err = Bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Error(err)
	}

	if Bot.Settings.EnableDryRun ||
		!exchCfg.Verbose {
		t.Error("dryrun should be false and verbose should be true")
	}
}
