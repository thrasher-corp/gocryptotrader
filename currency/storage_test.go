package currency

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

func TestMain(m *testing.M) {
	var err error
	testhelpers.TempDir, err = os.MkdirTemp("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	storage.fiatExchangeMarkets = newMockProvider()

	t := m.Run()

	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestRunUpdater(t *testing.T) {
	var newStorage Storage

	err := newStorage.RunUpdater(BotOverrides{}, &Config{}, "")
	assert.ErrorIs(t, err, errFiatDisplayCurrencyUnset, "No currency should error correctly")

	err = newStorage.RunUpdater(BotOverrides{}, &Config{FiatDisplayCurrency: BTC}, "")
	assert.ErrorIs(t, err, ErrFiatDisplayCurrencyIsNotFiat, "Crypto currency should error as not fiat")

	c := &Config{FiatDisplayCurrency: AUD}
	err = newStorage.RunUpdater(BotOverrides{}, c, "")
	assert.ErrorIs(t, err, errNoFilePathSet, "Should error with no path set")

	tempDir := testhelpers.TempDir
	err = newStorage.RunUpdater(BotOverrides{}, c, tempDir)
	assert.ErrorIs(t, err, errInvalidCurrencyFileUpdateDuration, "Should error invalid file update duration")

	c.CurrencyFileUpdateDuration = DefaultCurrencyFileDelay
	err = newStorage.RunUpdater(BotOverrides{}, c, tempDir)
	assert.ErrorIs(t, err, errInvalidForeignExchangeUpdateDuration, "Should error invalid forex update duration")

	c.ForeignExchangeUpdateDuration = DefaultForeignExchangeDelay
	err = newStorage.RunUpdater(BotOverrides{}, c, tempDir)

	assert.NoError(t, err, "Storage should not error with no forex providers enabled")
	assert.Nil(t, newStorage.fiatExchangeMarkets, "Forex should not be enabled with no providers") // Proxy for testing ForexEnabled

	err = newStorage.Shutdown()
	assert.NoError(t, err, "Shutdown should not error evne though it silently aborted the RunUpdater early")

	// Exchanges which reject a bad APIKey
	for _, n := range []string{"Fixer", "CurrencyConverter", "CurrencyLayer", "ExchangeRates"} {
		c.ForexProviders = AllFXSettings{{Name: n, Enabled: true, APIKey: ""}}
		err = newStorage.RunUpdater(overrideForProvider(n), c, tempDir)
		assert.NoErrorf(t, err, "%s should not error and silently exit without running with no api keys", n)
		assert.Falsef(t, c.ForexProviders[0].Enabled, "%s should not be marked enabled with no api keys", n)
		assert.Nil(t, newStorage.fiatExchangeMarkets, "Forex should not be enabled with no providers")
		c.ForexProviders = AllFXSettings{{Name: n, Enabled: true, APIKey: "sudo shazam!"}}
		err = newStorage.RunUpdater(overrideForProvider(n), c, tempDir)
		assert.Errorf(t, err, "%s should throw some provider originating error with a (hopefully) invalid api key", n)
		assert.Truef(t, c.ForexProviders[0].Enabled, "%s should still be enabled after being chosen but failing", n)
		assert.Nil(t, newStorage.fiatExchangeMarkets, "Forex should not be enabled when provider errored during startup")
		err = newStorage.Shutdown()
		assert.NoError(t, err, "Shutdown should not error")
	}

	// Exchanges which do not error with a bad APIKey on startup
	for _, n := range []string{"OpenExchangeRates"} {
		c.ForexProviders = AllFXSettings{{Name: n, Enabled: true, APIKey: ""}}
		err = newStorage.RunUpdater(overrideForProvider(n), c, tempDir)
		assert.NoErrorf(t, err, "%s should not error and silently exit without running with no api keys", n)
		assert.Nil(t, newStorage.fiatExchangeMarkets, "Forex should not be enabled with no providers")
		c.ForexProviders = AllFXSettings{{Name: n, Enabled: true, APIKey: "sudo shazam!"}}
		err = newStorage.RunUpdater(overrideForProvider(n), c, tempDir)
		assert.NoErrorf(t, err, "%s should not error on Setup with a bad apikey", n)
		assert.NotNil(t, newStorage.fiatExchangeMarkets, "Forex should be enabled now we have a provider with a key")
		err = newStorage.Shutdown()
		assert.NoError(t, err, "Shutdown should not error")
	}

	c.ForexProviders = AllFXSettings{
		{Name: "ExchangeRates"}, // Old Default
		{Name: "OpenExchangeRates", APIKey: "shazam?"},
	}

	// Regression test for old defaults which were enabled when in settings and nothing else was enabled and configured
	err = newStorage.RunUpdater(BotOverrides{}, c, tempDir)
	assert.NoError(t, err, "RunUpdater should not error")
	assert.Nil(t, newStorage.fiatExchangeMarkets, "Forex should not be enabled with no providers") // Proxy for testing ForexEnabled
	assert.False(t, c.ForexProviders[0].Enabled, "Old Default ExchangeRates should not have defaulted to enabled with no enabled overrides")
}

func overrideForProvider(n string) BotOverrides {
	b := BotOverrides{}
	switch n {
	case "Fixer":
		b.Fixer = true
	case "CurrencyConverter":
		b.CurrencyConverter = true
	case "CurrencyLayer":
		b.CurrencyLayer = true
	case "OpenExchangeRates":
		b.OpenExchangeRates = true
	case "ExchangeRates":
		b.ExchangeRates = true
	}
	return b
}
