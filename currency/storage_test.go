package currency

import "testing"

func TestRunUpdater(t *testing.T) {
	var newStorage Storage

	emptyMainConfig := Config{}
	err := newStorage.RunUpdater(BotOverrides{}, &emptyMainConfig, "")
	if err == nil {
		t.Fatal("storage RunUpdater() error cannot be nil")
	}

	mainConfig := Config{
		// Cryptocurrencies:    NewCurrenciesFromStringArray([]string{"BTC"}),
		FiatDisplayCurrency: USD,
	}

	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "")
	if err == nil {
		t.Fatal("storage RunUpdater() error cannot be nil")
	}

	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "/bla")
	if err != nil {
		t.Fatal("storage RunUpdater() error", err)
	}
}
