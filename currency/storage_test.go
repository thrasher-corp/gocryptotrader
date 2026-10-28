package currency

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

//nolint:forbidigo // TestMain reports setup and teardown failures before or after a *testing.T exists
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

func TestRunUpdaterFXMacroDataOverrideRequiresAPIKey(t *testing.T) {
	var newStorage Storage
	c := &Config{
		FiatDisplayCurrency:           AUD,
		CurrencyFileUpdateDuration:    DefaultCurrencyFileDelay,
		ForeignExchangeUpdateDuration: DefaultForeignExchangeDelay,
		ForexProviders: AllFXSettings{
			{Name: "FXMacroData", Enabled: true, APIKey: ""},
		},
	}

	err := newStorage.RunUpdater(overrideForProvider("FXMacroData"), c, testhelpers.TempDir)
	assert.NoError(t, err, "FXMacroData should not error when disabled due to missing API key")
	assert.False(t, c.ForexProviders[0].Enabled, "FXMacroData should be disabled when API key is missing")
	assert.Nil(t, newStorage.fiatExchangeMarkets, "forex markets should not start without an API key")
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
	case "FXMacroData":
		b.FXMacroData = true
	}
	return b
}

func TestStorageUpdaterLifecycleAndCachedReads(t *testing.T) {
	fxProviders, mock := newCountingMockProvider()

	var s Storage
	s.SetupConversionRates()
	s.defaultBaseCurrency = USD
	s.baseCurrency = USD
	s.defaultFiatCurrencies = Currencies{USD, EUR, AUD}
	s.fiatCurrencies = Currencies{USD, EUR, AUD}
	s.fiatExchangeMarkets = fxProviders
	s.foreignExchangeUpdateDelay = time.Hour
	s.currencyFileUpdateDelay = time.Hour
	s.shutdown = make(chan struct{})
	s.path = filepath.Join(t.TempDir(), DefaultStorageFile)

	// 1. Before updater is started, updaterRunning is false
	assert.False(t, s.updaterRunning.Load())

	// 2. Initial reads without background updater trigger one-off seeding
	r1, err := s.GetExchangeRates()
	assert.NoError(t, err)
	assert.NotEmpty(t, r1)
	assert.Equal(t, int64(1), mock.calls.Load(), "initial GetExchangeRates should call provider")

	r2, err := s.GetDefaultForeignExchangeRates()
	assert.NoError(t, err)
	assert.NotEmpty(t, r2)
	assert.Equal(t, int64(2), mock.calls.Load(), "initial GetDefaultForeignExchangeRates should call provider")

	// 3. Start ForeignExchangeUpdater in background
	s.mtx.Lock()
	s.wg.Add(1)
	go s.ForeignExchangeUpdater()

	// Wait for initial seed in ForeignExchangeUpdater to complete
	assert.Eventually(t, s.updaterRunning.Load, 5*time.Second, 10*time.Millisecond)

	initialCalls := mock.calls.Load()

	// 4. Repeated reads while updater is running should use cached rates without extra provider calls
	_, err = s.GetExchangeRates()
	assert.NoError(t, err)
	assert.Equal(t, initialCalls, mock.calls.Load(), "GetExchangeRates should read from cache while updater is running")

	_, err = s.GetDefaultForeignExchangeRates()
	assert.NoError(t, err)
	assert.Equal(t, initialCalls, mock.calls.Load(), "GetDefaultForeignExchangeRates should read from cache while updater is running")

	// 5. Shutdown transitions IsUpdaterRunning to false
	err = s.Shutdown()
	assert.NoError(t, err)
	assert.False(t, s.updaterRunning.Load(), "updaterRunning should be false after Shutdown")
}

// newCountingStorage returns a Storage set up as RunUpdater leaves it, using the counting mock provider
func newCountingStorage(t *testing.T, foreignExchangeUpdateDelay time.Duration) (*Storage, *CountingMockProvider) {
	t.Helper()
	fxProviders, mock := newCountingMockProvider()
	s := &Storage{
		baseCurrency:               USD,
		defaultBaseCurrency:        USD,
		fiatCurrencies:             Currencies{USD, EUR, AUD},
		defaultFiatCurrencies:      Currencies{USD, EUR, AUD},
		fiatExchangeMarkets:        fxProviders,
		path:                       filepath.Join(t.TempDir(), DefaultStorageFile),
		currencyFileUpdateDelay:    time.Hour,
		foreignExchangeUpdateDelay: foreignExchangeUpdateDelay,
		shutdown:                   make(chan struct{}),
	}
	s.SetupConversionRates()
	return s, mock
}

// startForeignExchangeUpdater starts the updater as RunUpdater does and waits for its initial seed
func startForeignExchangeUpdater(t *testing.T, s *Storage) {
	t.Helper()
	s.mtx.Lock()
	s.wg.Add(1)
	go s.ForeignExchangeUpdater()
	t.Cleanup(func() { assert.NoError(t, s.Shutdown(), "Shutdown should not error") })
	require.Eventually(t, s.updaterRunning.Load, 5*time.Second, 10*time.Millisecond, "updater must finish its initial seed")
}

func TestStorageUpdaterFailedInitialSeed(t *testing.T) {
	s, mock := newCountingStorage(t, time.Hour)
	mock.fail.Store(true)
	startForeignExchangeUpdater(t, s)
	require.Equal(t, int64(1), mock.calls.Load(), "initial seed must call the provider once")

	_, err := s.GetDefaultForeignExchangeRates()
	assert.Error(t, err, "GetDefaultForeignExchangeRates should retry the seed and return its error while no rates are cached")
	_, err = s.GetExchangeRates()
	assert.Error(t, err, "GetExchangeRates should retry the seed and return its error while no rates are cached")
	assert.Equal(t, int64(3), mock.calls.Load(), "each read should retry the seed while no rates are cached")

	mock.fail.Store(false)
	rates, err := s.GetExchangeRates()
	require.NoError(t, err, "GetExchangeRates must not error once the provider recovers")
	assert.NotEmpty(t, rates, "GetExchangeRates should return the rates it seeded")
	rates, err = s.GetDefaultForeignExchangeRates()
	require.NoError(t, err, "GetDefaultForeignExchangeRates must not error once rates are cached")
	assert.NotEmpty(t, rates, "GetDefaultForeignExchangeRates should return the cached rates")
	assert.Equal(t, int64(4), mock.calls.Load(), "reads should use the cache once a seed has succeeded")
}

func TestStorageUpdaterTickRefreshesRates(t *testing.T) {
	s, mock := newCountingStorage(t, 10*time.Millisecond)
	startForeignExchangeUpdater(t, s)
	seeded, err := s.fxRates.GetRate(USD, EUR)
	require.NoError(t, err, "GetRate must not error after the initial seed")
	assert.Eventually(t, func() bool {
		rate, err := s.fxRates.GetRate(USD, EUR)
		return err == nil && rate > seeded
	}, 5*time.Second, 10*time.Millisecond, "a foreign exchange tick should refresh the cached rates")

	// Tick goroutines outlive Shutdown and update rates through the package storage: fail those still to
	// fetch, and let HasData's lock wait out one already updating, before another test uses that storage
	mock.fail.Store(true)
	require.NoError(t, s.Shutdown(), "Shutdown must not error")
	assert.True(t, s.fxRates.HasData(), "rates should stay cached after Shutdown")
}
