package currency

import "testing"

func TestRunUpdater(t *testing.T) {
	var newStorage Storage

	emptyMainConfig := MainConfiguration{}
	err := newStorage.RunUpdater(BotOverrides{}, &emptyMainConfig, "", false)
	if err == nil {
		t.Fatal("Test Failed storage RunUpdater() error cannot be nil")
	}

	mainConfig := MainConfiguration{
		Cryptocurrencies:    NewCurrenciesFromStringArray([]string{"BTC"}),
		FiatDisplayCurrency: USD,
	}

	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "", false)
	if err == nil {
		t.Fatal("Test Failed storage RunUpdater() error cannot be nil")
	}

	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "/bla", false)
	if err != nil {
		t.Fatal("Test Failed storage RunUpdater() error", err)
	}
}
