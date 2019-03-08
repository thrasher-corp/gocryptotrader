package currency

import "testing"

func TestRunUpdater(t *testing.T) {
	var newStorage Storage

	err := newStorage.RunUpdater(BotOverrides{}, MainConfiguration{}, "", false)
	if err == nil {
		t.Fatal("Test Failed storage RunUpdater() error cannot be nil")
	}

	mainConfig := MainConfiguration{
		Cryptocurrencies: NewCurrenciesFromStringArray([]string{"BTC"}),
	}

	err = newStorage.RunUpdater(BotOverrides{}, mainConfig, "", false)
	if err == nil {
		t.Fatal("Test Failed storage RunUpdater() error cannot be nil")
	}

	err = newStorage.RunUpdater(BotOverrides{}, mainConfig, "/bla", false)
	if err != nil {
		t.Fatal("Test Failed storage RunUpdater() error cannot be nil", err)
	}
}
