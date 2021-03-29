package exchangemanager

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
)

func TestExchangeManagerAdd(t *testing.T) {
	t.Parallel()
	var e Manager
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	e.Add(b)
	if exch := e.GetExchanges(); exch[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerGetExchanges(t *testing.T) {
	t.Parallel()
	var e Manager
	if exchanges := e.GetExchanges(); exchanges != nil {
		t.Error("unexpected value")
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	e.Add(b)
	if exch := e.GetExchanges(); exch[0].GetName() != "Bitfinex" {
		t.Error("unexpected exchange name")
	}
}

func TestExchangeManagerRemoveExchange(t *testing.T) {
	t.Parallel()
	var e Manager
	if err := e.RemoveExchange("Bitfinex"); err != ErrNoExchangesLoaded {
		t.Error("no exchanges should be loaded")
	}
	b := new(bitfinex.Bitfinex)
	b.SetDefaults()
	e.Add(b)
	if err := e.RemoveExchange(events.testExchange); err != ErrExchangeNotFound {
		t.Error("Bitstamp exchange should return an error")
	}
	if err := e.RemoveExchange("BiTFiNeX"); err != nil {
		t.Error("exchange should have been removed")
	}
	if e.Len() != 0 {
		t.Error("exchange manager len should be 0")
	}
}

func TestCheckExchangeExists(t *testing.T) {
	e := engine.CreateTestBot(t)

	if e.GetExchangeByName(events.testExchange) == nil {
		t.Errorf("TestGetExchangeExists: Unable to find exchange")
	}

	if e.GetExchangeByName("Asdsad") != nil {
		t.Errorf("TestGetExchangeExists: Non-existent exchange found")
	}
}

func TestGetExchangeByName(t *testing.T) {
	e := engine.CreateTestBot(t)

	exch := e.GetExchangeByName(events.testExchange)
	if exch == nil {
		t.Errorf("TestGetExchangeByName: Failed to get exchange")
	}

	if !exch.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch.SetEnabled(false)
	bfx := e.GetExchangeByName(events.testExchange)
	if bfx.IsEnabled() {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	if exch.GetName() != events.testExchange {
		t.Errorf("TestGetExchangeByName: Unexpected result")
	}

	exch = e.GetExchangeByName("Asdasd")
	if exch != nil {
		t.Errorf("TestGetExchangeByName: Non-existent exchange found")
	}
}

func TestUnloadExchange(t *testing.T) {
	e := engine.CreateTestBot(t)

	err := e.UnloadExchange("asdf")
	if err == nil || err.Error() != "exchange asdf not found" {
		t.Errorf("TestUnloadExchange: Incorrect result: %s",
			err)
	}

	err = e.UnloadExchange(events.testExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to get exchange. %s",
			err)
	}

	err = e.UnloadExchange(engine.fakePassExchange)
	if err != nil {
		t.Errorf("TestUnloadExchange: Failed to unload exchange. %s",
			err)
	}

	err = e.UnloadExchange(events.testExchange)
	if err != ErrNoExchangesLoaded {
		t.Errorf("TestUnloadExchange: Incorrect result: %s",
			err)
	}
}

func TestDryRunParamInteraction(t *testing.T) {
	bot := engine.CreateTestBot(t)

	// Simulate overiding default settings and ensure that enabling exchange
	// verbose mode will be set on Bitfinex
	var err error
	if err = bot.UnloadExchange(events.testExchange); err != nil {
		t.Error(err)
	}

	bot.Settings.CheckParamInteraction = false
	bot.Settings.EnableExchangeVerbose = false
	if err = bot.LoadExchange(events.testExchange, false, nil); err != nil {
		t.Error(err)
	}

	exchCfg, err := bot.Config.GetExchangeConfig(events.testExchange)
	if err != nil {
		t.Error(err)
	}

	if exchCfg.Verbose {
		t.Error("verbose should have been disabled")
	}

	if err = bot.UnloadExchange(events.testExchange); err != nil {
		t.Error(err)
	}

	// Now set dryrun mode to true,
	// enable exchange verbose mode and verify that verbose mode
	// will be set on Bitfinex
	bot.Settings.EnableDryRun = true
	bot.Settings.CheckParamInteraction = true
	bot.Settings.EnableExchangeVerbose = true
	if err = bot.LoadExchange(events.testExchange, false, nil); err != nil {
		t.Error(err)
	}

	exchCfg, err = bot.Config.GetExchangeConfig(events.testExchange)
	if err != nil {
		t.Error(err)
	}

	if !bot.Settings.EnableDryRun ||
		!exchCfg.Verbose {
		t.Error("dryrun should be true and verbose should be true")
	}
}
