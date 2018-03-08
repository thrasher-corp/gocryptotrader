package main

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

func getEnabledExchanges() []string {
	bot.config = &config.Cfg
	err := bot.config.LoadConfig("./testdata/configtest.json")
	if err != nil {
		return []string{}
	}
	return bot.config.GetEnabledExchanges()

}

func TestNSetupExchanges(t *testing.T) {
	exchs := getEnabledExchanges()
	t.Logf("\nNumber of exchanges is %d", len(exchs))
	for _, e := range exchs {
		t.Logf(e)
		t.Run(e, func(t *testing.T) {
			err := LoadExchange(e)
			if err != nil {
				t.Errorf("Test failed. SetupTest: Failed to load exchange: %s", err)
			}
			SetupExchanges()
			err = UnloadExchange(e)
			if err != nil {
				t.Fatalf("Test failed. Failed to unload exchange: %s",
					err)
			}
		})
	}

}

func TestNExchangeExists(t *testing.T) {
	exchs := getEnabledExchanges()
	for _, e := range exchs {
		t.Run(e, func(t *testing.T) {
			err := LoadExchange(e)
			if err != nil {
				t.Errorf("Test failed. SetupTest: Failed to load exchange: %s", err)
			}

			if !CheckExchangeExists(e) {
				t.Errorf("Test failed. TestGetExchangeExists: Unable to find exchange")
			}
			err = UnloadExchange(e)
			if err != nil {
				t.Fatalf("Test failed. Failed to unload exchange: %s",
					err)
			}
		})
	}

}

func TestNUnloadExchange(t *testing.T) {
	exchs := getEnabledExchanges()
	for _, e := range exchs {
		t.Run(e, func(t *testing.T) {
			err := LoadExchange(e)
			if err != nil {
				t.Errorf("Test failed. SetupTest: Failed to load exchange: %s", err)
			}
			err = UnloadExchange(e)
			if err != nil {
				t.Errorf("Test failed. TestUnloadExchange: Failed to get exchange. %s",
					err)
			}

			err = UnloadExchange(e)
			if err == nil {
				t.Fatalf("Test failed. Failed to unload exchange: %s",
					err)
			}

		})
	}

}

func TestNReloadExchange(t *testing.T) {
	exchs := getEnabledExchanges()
	for _, e := range exchs {
		t.Run(e, func(t *testing.T) {
			err := LoadExchange(e)
			if err != nil {
				t.Errorf("Test failed. SetupTest: Failed to load exchange: %s", err)
			}
			err = ReloadExchange(e)
			if err != nil {
				t.Errorf("Test failed. TestReloadExchange: Incorrect result: %s",
					err)
			}
			err = UnloadExchange(e)
			if err != nil {
				t.Fatalf("Test failed. Failed to unload exchange: %s",
					err)
			}
		})
	}
}

func TestNGetExchangeByName(t *testing.T) {
	exchs := getEnabledExchanges()
	for _, e := range exchs {
		t.Run(e, func(t *testing.T) {
			err := LoadExchange(e)
			if err != nil {
				t.Errorf("Test failed. SetupTest: Failed to load exchange: %s", err)
			}
			exch := GetExchangeByName(e)
			if exch == nil {
				t.Errorf("Test failed. TestGetExchangeByName: Failed to get exchange")
			}

			if !exch.IsEnabled() {
				t.Errorf("Test failed. TestGetExchangeByName: Unexpected result")
			}

			exch.SetEnabled(false)
			bfx := GetExchangeByName(e)
			if bfx.IsEnabled() {
				t.Errorf("Test failed. TestGetExchangeByName: Unexpected result")
			}

			if exch.GetName() != e {
				t.Errorf("Test failed. TestGetExchangeByName: Unexpected result")
			}
			err = UnloadExchange(e)
			if err != nil {
				t.Fatalf("Test failed. Failed to unload exchange: %s",
					err)
			}

		})
	}
}
