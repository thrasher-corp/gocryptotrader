package currency

import (
	"errors"
	"testing"
)

func TestRunUpdater(t *testing.T) {
	var newStorage Storage

	emptyMainConfig := Config{}
	err := newStorage.RunUpdater(BotOverrides{}, &emptyMainConfig, "")
	if err == nil {
		t.Fatal("storage RunUpdater() error cannot be nil")
	}

	mainConfig := Config{}
	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "")
	if !errors.Is(err, errFiatDisplayCurrencyUnset) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errFiatDisplayCurrencyUnset)
	}

	mainConfig.FiatDisplayCurrency = BTC
	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "")
	if !errors.Is(err, ErrFiatDisplayCurrencyIsNotFiat) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrFiatDisplayCurrencyIsNotFiat)
	}

	mainConfig.FiatDisplayCurrency = AUD
	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "")
	if !errors.Is(err, errNoFilePathSet) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoFilePathSet)
	}

	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "/bla")
	if !errors.Is(err, errInvalidCurrencyFileUpdateDuration) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidCurrencyFileUpdateDuration)
	}

	mainConfig.CurrencyFileUpdateDuration = DefaultCurrencyFileDelay
	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "/bla")
	if !errors.Is(err, errInvalidForeignExchangeUpdateDuration) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidForeignExchangeUpdateDuration)
	}

	mainConfig.ForeignExchangeUpdateDuration = DefaultForeignExchangeDelay
	err = newStorage.RunUpdater(BotOverrides{}, &mainConfig, "/bla")
	if !errors.Is(err, errNoForeignExchangeProvidersEnabled) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errNoForeignExchangeProvidersEnabled)
	}

	settings := FXSettings{
		Name:    "Fixer",
		Enabled: true,
		APIKey:  "wo",
	}

	mainConfig.ForexProviders = AllFXSettings{settings}
	err = newStorage.RunUpdater(BotOverrides{Fixer: true}, &mainConfig, "/bla")
	if errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, "an error")
	}

	err = newStorage.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	settings.Name = "CurrencyConverter"
	mainConfig.ForexProviders = AllFXSettings{settings}
	err = newStorage.RunUpdater(BotOverrides{CurrencyConverter: true}, &mainConfig, "/bla")
	if errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, "an error")
	}

	err = newStorage.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	settings.Name = "CurrencyLayer"
	mainConfig.ForexProviders = AllFXSettings{settings}
	err = newStorage.RunUpdater(BotOverrides{CurrencyLayer: true}, &mainConfig, "/bla")
	if errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, "an error")
	}

	err = newStorage.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	settings.Name = "OpenExchangeRates"
	mainConfig.ForexProviders = AllFXSettings{settings}
	err = newStorage.RunUpdater(BotOverrides{OpenExchangeRates: true}, &mainConfig, "/bla")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = newStorage.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	settings.Name = "ExchangeRates"
	mainConfig.ForexProviders = AllFXSettings{settings}
	err = newStorage.RunUpdater(BotOverrides{ExchangeRates: true}, &mainConfig, "/bla")
	if errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, "an error")
	}

	err = newStorage.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	settings.Name = "ExchangeRateHost"
	mainConfig.ForexProviders = AllFXSettings{settings}
	err = newStorage.RunUpdater(BotOverrides{ExchangeRateHost: true}, &mainConfig, "/bla")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}
