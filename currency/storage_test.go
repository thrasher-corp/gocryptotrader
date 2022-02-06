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
	if !errors.Is(err, errFiatDisplayCurrencyIsNotFiat) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errFiatDisplayCurrencyIsNotFiat)
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
}
