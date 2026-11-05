package exchange

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"weak"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/binance"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bybit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

const multiConnectionFilter = "multi-connection-test"

type multiConnectionSetupExchange struct {
	sharedtestvalues.CustomEx
}

func (e *multiConnectionSetupExchange) GetBase() *exchange.Base {
	return &e.Base
}

func newMultiConnectionSetupExchange(tb testing.TB, websocketURL string, dialer *gws.Dialer, trafficTimeout time.Duration) *multiConnectionSetupExchange {
	tb.Helper()

	e := &multiConnectionSetupExchange{}
	e.Base.Name = "MultiConnectionSetupExchange"
	e.Base.Features.Subscriptions = subscription.List{{Channel: "ticker"}}
	e.Base.Websocket = websocket.NewManager()

	err := e.Base.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: &config.Exchange{
			Name: "MultiConnectionSetupExchange",
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{Websocket: true},
			},
			WebsocketTrafficTimeout: trafficTimeout,
		},
		Features:                     &protocol.Features{},
		UseMultiConnectionManagement: true,
	})
	require.NoError(tb, err, "Setup must not error for the multi-connection manager")

	err = e.Base.Websocket.SetupNewConnection(&websocket.ConnectionSetup{
		URL: websocketURL,
		Connector: func(ctx context.Context, conn websocket.Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return e.Base.Features.Subscriptions.Clone(), nil
		},
		Subscriber:    func(context.Context, websocket.Connection, subscription.List) error { return nil },
		Handler:       func(context.Context, websocket.Connection, []byte) error { return nil },
		MessageFilter: multiConnectionFilter,
	})
	require.NoError(tb, err, "SetupNewConnection must not error for the multi-connection manager")

	return e
}

// accountsProbeExchange records the accounts the exchange's own Setup receives
type accountsProbeExchange struct {
	binance.Exchange
	setupErr      error
	setupAccounts *accounts.Accounts
}

func (e *accountsProbeExchange) Setup(exch *config.Exchange) error {
	e.setupAccounts = e.Accounts
	if e.setupErr != nil {
		return e.setupErr
	}
	return e.Exchange.Setup(exch)
}

// failingMarshaler fails marshalling with err, so copy errors can be checked for their cause
type failingMarshaler struct {
	err error
}

func (m failingMarshaler) MarshalJSON() ([]byte, error) {
	return nil, m.err
}

// TestSetup exercises Setup
func TestSetup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := new(binance.Exchange)
		require.NoError(t, Setup(b), "Setup must not error")
		assert.NotNil(t, b.Websocket, "Websocket should not be nil after Setup")
	})

	t.Run("missing exchange", func(t *testing.T) {
		assert.ErrorIs(t, Setup(new(sharedtestvalues.CustomEx)), config.ErrExchangeNotFound, "Setup should error correctly on a missing exchange")
	})

	t.Run("verbose", func(t *testing.T) {
		b := new(binance.Exchange)
		require.NoError(t, Setup(b, true), "Setup must not error when verbose is overridden")
		assert.True(t, b.IsVerbose(), "Setup should honour the verbose override")
	})

	t.Run("accounts assigned before exchange setup", func(t *testing.T) {
		e := new(accountsProbeExchange)
		require.NoError(t, Setup(e), "Setup must not error")
		require.NotNil(t, e.setupAccounts, "Accounts must be assigned before the exchange's Setup runs")
		assert.Same(t, e.setupAccounts, e.Accounts, "Setup should keep the accounts assigned before the exchange's Setup")
	})

	t.Run("exchange setup error", func(t *testing.T) {
		errSetup := errors.New("setup failure")
		e := &accountsProbeExchange{setupErr: errSetup}
		assert.ErrorIs(t, Setup(e), errSetup, "Setup should return the exchange's Setup error")
		assert.NotNil(t, e.setupAccounts, "Accounts should be assigned before a failing exchange Setup runs")
	})

	t.Run("does not retain exchange", func(t *testing.T) {
		exch := func() weak.Pointer[binance.Exchange] {
			e := new(binance.Exchange)
			require.NoError(t, Setup(e), "Setup must not error")
			return weak.Make(e)
		}()
		assert.Eventually(t, func() bool {
			runtime.GC()
			return exch.Value() == nil
		}, 10*time.Second, 10*time.Millisecond, "Setup should not keep the exchange reachable once the caller drops it")
	})

	t.Run("own config per exchange", func(t *testing.T) {
		a, b := new(binance.Exchange), new(binance.Exchange)
		require.NoError(t, Setup(a), "Setup must not error")
		require.NoError(t, Setup(b), "Setup must not error for a second exchange")
		require.NotSame(t, a.Config, b.Config, "Setup must give each exchange its own config")
		a.Config.HTTPTimeout = time.Nanosecond
		assert.NotEqual(t, time.Nanosecond, b.Config.HTTPTimeout, "Changing one exchange's config should not affect another's")
	})

	t.Run("concurrent", func(t *testing.T) {
		var ec common.ErrorCollector
		for range 8 {
			ec.Go(func() error { return Setup(new(binance.Exchange)) })
		}
		done := make(chan error, 1)
		go func() { done <- ec.Collect() }()
		select {
		case err := <-done:
			assert.NoError(t, err, "Concurrent Setup should not error")
		case <-time.After(time.Minute):
			assert.Fail(t, "Concurrent Setup should finish within a minute")
		}
	})

	t.Run("config load error", func(t *testing.T) {
		errLoad := errors.New("load failure")
		load := testConfig
		t.Cleanup(func() { testConfig = load })
		testConfig = func() (*config.Config, error) { return nil, errLoad }
		e := new(binance.Exchange)
		assert.ErrorIs(t, Setup(e), errLoad, "Setup should return the config load error")
		assert.Equal(t, new(binance.Exchange), e, "Setup should leave the exchange untouched when the config cannot be loaded")
	})

	t.Run("config copy error", func(t *testing.T) {
		load := testConfig
		t.Cleanup(func() { testConfig = load })
		errMarshal := errors.New("marshal failure")
		bad := &config.Config{Exchanges: []config.Exchange{{
			Name:     "Binance",
			Features: &config.FeaturesConfig{Subscriptions: subscription.List{{Channel: "x", Params: map[string]any{"bad": failingMarshaler{err: errMarshal}}}}},
		}}}
		testConfig = func() (*config.Config, error) { return bad, nil }
		err := Setup(new(binance.Exchange))
		assert.ErrorIs(t, err, errCopyingExchangeConfig, "Setup should return the config copy error")
		assert.ErrorIs(t, err, errMarshal, "Setup should keep the cause of the config copy error")
		assert.ErrorContains(t, err, "Binance error copying exchange config", "Setup should name the exchange in the config copy error")
	})
}

func TestTestConfig(t *testing.T) {
	a, err := testConfig()
	require.NoError(t, err, "testConfig must not error")
	b, err := testConfig()
	require.NoError(t, err, "testConfig must not error when called again")
	assert.Same(t, a, b, "testConfig should load the config once")
}

func TestLoadTestConfig(t *testing.T) {
	cfg, err := loadTestConfig()
	require.NoError(t, err, "loadTestConfig must not error from within the repository")
	_, err = cfg.GetExchangeConfig("Binance")
	assert.NoError(t, err, "loadTestConfig should load the exchange configs")

	t.Run("outside the repository", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := testutils.RootPath(dir); err == nil {
			t.Skipf("Temporary directory %s is inside a GoCryptoTrader checkout", dir)
		}
		t.Chdir(dir)
		_, err := loadTestConfig()
		assert.ErrorIs(t, err, testutils.ErrRootNotFound, "loadTestConfig should error outside the repository")
	})

	t.Run("invalid config", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "LICENSE"), nil, 0o600), "Writing LICENSE must not error")
		require.NoError(t, os.Mkdir(filepath.Join(root, "testdata"), 0o700), "Creating testdata must not error")
		require.NoError(t, os.WriteFile(filepath.Join(root, "testdata", "configtest.json"), []byte("{"), 0o600), "Writing configtest.json must not error")
		t.Chdir(root)
		_, err := loadTestConfig()
		assert.ErrorIs(t, err, config.ErrFailureOpeningConfig, "loadTestConfig should error on an invalid config")
	})
}

func TestCopyExchangeConfig(t *testing.T) {
	cfg, err := testConfig()
	require.NoError(t, err, "testConfig must not error")

	t.Run("independent copy", func(t *testing.T) {
		shared, err := cfg.GetExchangeConfig("Binance")
		require.NoError(t, err, "GetExchangeConfig must not error")
		sharedSpot := shared.CurrencyPairs.Pairs[asset.Spot]
		require.NotEmpty(t, shared.BaseCurrencies, "Binance's config must have base currencies")
		require.NotEmpty(t, sharedSpot.Enabled, "Binance's spot config must have enabled pairs")
		require.NotNil(t, sharedSpot.ConfigFormat, "Binance's spot config must have a config format")
		require.NotEmpty(t, shared.API.Endpoints, "Binance's config must have endpoints")
		require.NotEmpty(t, shared.Features.Subscriptions, "Binance's config must have subscriptions")
		baseCurrency, enabledPair, delimiter := shared.BaseCurrencies[0], sharedSpot.Enabled[0], sharedSpot.ConfigFormat.Delimiter
		endpoints, channel := maps.Clone(shared.API.Endpoints), shared.Features.Subscriptions[0].Channel

		a, err := copyExchangeConfig(shared)
		require.NoError(t, err, "copyExchangeConfig must not error")
		require.NotSame(t, shared, a, "copyExchangeConfig must return a new config")
		aSpot := a.CurrencyPairs.Pairs[asset.Spot]
		a.Name = "changed"
		a.BaseCurrencies[0] = currency.XRP
		aSpot.Enabled[0] = currency.NewPair(currency.XRP, currency.USD)
		aSpot.ConfigFormat.Delimiter = "?"
		a.API.Endpoints["changed"] = "changed"
		a.Features.Subscriptions[0].Channel = "changed"

		assert.Equal(t, "Binance", shared.Name, "Changing a copy's name should not affect the shared config")
		assert.Equal(t, baseCurrency, shared.BaseCurrencies[0], "Changing a copy's base currencies should not affect the shared config")
		assert.Equal(t, enabledPair, sharedSpot.Enabled[0], "Changing a copy's pairs should not affect the shared config")
		assert.Equal(t, delimiter, sharedSpot.ConfigFormat.Delimiter, "Changing a copy's pair format should not affect the shared config")
		assert.Equal(t, endpoints, shared.API.Endpoints, "Changing a copy's endpoints should not affect the shared config")
		assert.Equal(t, channel, shared.Features.Subscriptions[0].Channel, "Changing a copy's subscriptions should not affect the shared config")

		b, err := copyExchangeConfig(shared)
		require.NoError(t, err, "copyExchangeConfig must not error after a copy is changed")
		assert.Equal(t, baseCurrency, b.BaseCurrencies[0], "Changing a copy's base currencies should not affect a later copy")
		assert.Equal(t, enabledPair, b.CurrencyPairs.Pairs[asset.Spot].Enabled[0], "Changing a copy's pairs should not affect a later copy")
	})

	t.Run("no currency pairs", func(t *testing.T) {
		c, err := copyExchangeConfig(&config.Exchange{Name: "NoPairs"})
		require.NoError(t, err, "copyExchangeConfig must not error without currency pairs")
		assert.Equal(t, &config.Exchange{Name: "NoPairs"}, c, "copyExchangeConfig should copy a config without currency pairs")
	})

	t.Run("matches a fresh load", func(t *testing.T) {
		fresh, err := loadTestConfig()
		require.NoError(t, err, "loadTestConfig must not error")
		require.NotEmpty(t, fresh.Exchanges, "configtest.json must contain exchanges")
		for i := range fresh.Exchanges {
			want := fresh.Exchanges[i]
			t.Run(want.Name, func(t *testing.T) {
				t.Parallel()
				src, err := cfg.GetExchangeConfig(want.Name)
				require.NoErrorf(t, err, "GetExchangeConfig must not error for %s", want.Name)
				got, err := copyExchangeConfig(src)
				require.NoErrorf(t, err, "copyExchangeConfig must not error for %s", want.Name)
				require.NotNilf(t, want.CurrencyPairs, "%s's config must have currency pairs", want.Name)
				require.NotNilf(t, got.CurrencyPairs, "copyExchangeConfig must copy the currency pairs for %s", want.Name)
				assert.Equalf(t, want.CurrencyPairs.Pairs, got.CurrencyPairs.Pairs, "copyExchangeConfig should copy the pair stores for %s", want.Name)
				assert.Equalf(t, want.CurrencyPairs.RequestFormat, got.CurrencyPairs.RequestFormat, "copyExchangeConfig should copy the request format for %s", want.Name)
				assert.Equalf(t, want.CurrencyPairs.ConfigFormat, got.CurrencyPairs.ConfigFormat, "copyExchangeConfig should copy the config format for %s", want.Name)
				assert.Equalf(t, want.CurrencyPairs.UseGlobalFormat, got.CurrencyPairs.UseGlobalFormat, "copyExchangeConfig should copy UseGlobalFormat for %s", want.Name)
				assert.Equalf(t, want.CurrencyPairs.BypassConfigFormatUpgrades, got.CurrencyPairs.BypassConfigFormatUpgrades, "copyExchangeConfig should copy BypassConfigFormatUpgrades for %s", want.Name)
				assert.Equalf(t, want.CurrencyPairs.LastUpdated, got.CurrencyPairs.LastUpdated, "copyExchangeConfig should copy LastUpdated for %s", want.Name)
				rest := *got
				want.CurrencyPairs, rest.CurrencyPairs = nil, nil
				assert.Equalf(t, want, rest, "copyExchangeConfig should copy the rest of the config for %s", want.Name)
			})
		}
	})

	t.Run("copy errors", func(t *testing.T) {
		errMarshal := errors.New("marshal failure")
		for _, tc := range []struct {
			name     string
			features *config.FeaturesConfig
			cause    error
		}{
			{"marshal failure", &config.FeaturesConfig{Subscriptions: subscription.List{{Channel: "x", Params: map[string]any{"bad": failingMarshaler{err: errMarshal}}}}}, errMarshal},
			{"unmarshal failure", &config.FeaturesConfig{Subscriptions: subscription.List{{Channel: "x", Pairs: currency.Pairs{currency.NewPair(currency.NewCode("A"), currency.EMPTYCODE)}}}}, currency.ErrCreatingPair},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				c, err := copyExchangeConfig(&config.Exchange{Name: "Bad", Features: tc.features})
				assert.ErrorIs(t, err, errCopyingExchangeConfig, "copyExchangeConfig should error when the config cannot be copied")
				assert.ErrorIs(t, err, tc.cause, "copyExchangeConfig should wrap the cause of the copy error")
				assert.Nil(t, c, "copyExchangeConfig should return no config when the config cannot be copied")
			})
		}
	})
}

func BenchmarkSetup(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		require.NoError(b, Setup(new(binance.Exchange)), "Setup must not error")
	}
}

// TestMockHTTPInstance exercises MockHTTPInstance
func TestMockHTTPInstance(t *testing.T) {
	b := new(binance.Exchange)
	require.NoError(t, Setup(b), "Test exchange Setup must not error")
	require.NoError(t, MockHTTPInstance(b), "MockHTTPInstance with no optional path must not error")
	require.NoError(t, MockHTTPInstance(b, "api"), "MockHTTPInstance with optional path must not error")
}

// TestMockWebsocketInstance exercises MockWebsocketInstance
func TestMockWebsocketInstance(t *testing.T) {
	newInstance := func(tb testing.TB) *binance.Exchange {
		tb.Helper()
		return MockWebsocketInstance[binance.Exchange](tb, mockws.CurryWsMockUpgrader(tb, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }))
	}

	t.Run("success", func(t *testing.T) {
		require.NotNil(t, newInstance(t), "MockWebsocketInstance must not be nil")
	})

	t.Run("verbose", func(t *testing.T) {
		b := MockWebsocketInstance[binance.Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }), true)
		require.NotNil(t, b, "MockWebsocketInstance must not be nil when verbose is overridden")
		assert.True(t, b.IsVerbose(), "MockWebsocketInstance should honour the verbose override")
	})

	t.Run("multi-connection management", func(t *testing.T) {
		b := MockWebsocketInstance[bybit.Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, _ []byte, _ *gws.Conn) error { return nil }))
		require.NotNil(t, b, "MockWebsocketInstance must not be nil for multi-connection websocket exchanges")
		t.Cleanup(func() {
			if b.GetBase().Websocket.IsConnected() {
				assert.NoError(t, b.GetBase().Websocket.Shutdown(), "Websocket shutdown should not error for multi-connection websocket exchanges")
			}
		})
		assert.True(t, b.GetBase().Websocket.IsConnected(), "Websocket manager should be connected for multi-connection websocket exchanges")
	})

	t.Run("shut down once its test ends", func(t *testing.T) {
		var e *binance.Exchange
		t.Run("instance", func(t *testing.T) {
			e = newInstance(t)
			require.True(t, e.Websocket.IsConnected(), "Websocket must be connected during its test")
		})
		assert.False(t, e.Websocket.IsEnabled(), "Websocket should be disabled once its test ends")
		assert.False(t, e.Websocket.IsConnected(), "Websocket should be shut down once its test ends")
	})

	t.Run("already shut down", func(t *testing.T) {
		require.NoError(t, newInstance(t).Shutdown(), "Shutdown must not error")
	})

	t.Run("websocket shut down but still enabled", func(t *testing.T) {
		var e *binance.Exchange
		t.Run("instance", func(t *testing.T) {
			e = newInstance(t)
			require.NoError(t, e.Websocket.Shutdown(), "Websocket Shutdown must not error")
		})
		assert.False(t, e.Websocket.IsEnabled(), "Websocket should be disabled once its test ends, so its monitor stops reconnecting")
	})

	t.Run("does not retain exchange", func(t *testing.T) {
		var ref weak.Pointer[binance.Exchange]
		t.Run("instance", func(t *testing.T) {
			ref = weak.Make(newInstance(t))
		})
		assert.Eventually(t, func() bool {
			runtime.GC()
			return ref.Value() == nil
		}, 10*time.Second, 10*time.Millisecond, "MockWebsocketInstance should not keep the exchange reachable once its test ends")
	})
}

func TestSetupWebsocket(t *testing.T) {
	server, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
	const trafficTimeout = 5 * time.Second

	t.Run("multi-connection management", func(t *testing.T) {
		e := newMultiConnectionSetupExchange(t, websocketURL, dialer, trafficTimeout)
		t.Cleanup(func() {
			if e.Base.Websocket.IsConnected() {
				assert.NoError(t, e.Base.Websocket.Shutdown(), "Websocket shutdown should not error after SetupWebsocket")
			}
		})

		SetupWebsocket(t, e)

		assert.Empty(t, e.Base.Features.Subscriptions, "Features.Subscriptions should be cleared by SetupWebsocket")
		assert.True(t, e.Base.Websocket.IsConnected(), "Websocket manager should be connected after SetupWebsocket")

		conn, err := e.Base.Websocket.GetConnection(multiConnectionFilter)
		require.NoError(t, err, "GetConnection must not error after SetupWebsocket on a multi-connection manager")
		assert.NotNil(t, conn, "GetConnection should return a connection after SetupWebsocket on a multi-connection manager")
		assert.Empty(t, conn.Subscriptions().List(), "Connection subscriptions should remain empty when subscriptions are not required")
	})

	t.Run("already set up", func(t *testing.T) {
		e := newMultiConnectionSetupExchange(t, websocketURL, dialer, trafficTimeout)
		SetupWebsocket(t, e)
		// Disabled, so only the set-up-once check can stop the second call skipping its test
		require.NoError(t, e.Base.Websocket.Disable(), "Websocket disable must not error")
		var skipped bool
		t.Run("again", func(t *testing.T) {
			t.Cleanup(func() { skipped = t.Skipped() })
			SetupWebsocket(t, e)
		})
		assert.False(t, skipped, "SetupWebsocket should return early for an exchange it has already set up")
	})

	t.Run("already connected", func(t *testing.T) {
		// A silent connection must outlast any pause before SetupWebsocket, or the traffic monitor drops it
		e := newMultiConnectionSetupExchange(t, websocketURL, dialer, time.Hour)
		t.Cleanup(func() { assert.NoError(t, e.Base.Websocket.Disable(), "Websocket disable should not error") })
		e.Base.Websocket.SetSubscriptionsNotRequired()
		require.NoError(t, e.Base.Websocket.Connect(t.Context()), "Connect must not error")
		SetupWebsocket(t, e)
		assert.Len(t, e.Base.Features.Subscriptions, 1, "SetupWebsocket should leave a connected exchange alone")
	})

	t.Run("websocket disabled", func(t *testing.T) {
		e := newMultiConnectionSetupExchange(t, websocketURL, dialer, trafficTimeout)
		require.NoError(t, e.Base.Websocket.Disable(), "Websocket disable must not error")
		var skipped bool
		t.Run("setup", func(t *testing.T) {
			t.Cleanup(func() { skipped = t.Skipped() })
			SetupWebsocket(t, e)
		})
		assert.True(t, skipped, "SetupWebsocket should skip a test whose websocket is disabled")
	})

	t.Run("does not retain exchange", func(t *testing.T) {
		ref := func() weak.Pointer[multiConnectionSetupExchange] {
			e := newMultiConnectionSetupExchange(t, websocketURL, dialer, trafficTimeout)
			SetupWebsocket(t, e)
			require.True(t, e.Base.Websocket.IsConnected(), "Websocket must be connected after SetupWebsocket")
			require.NoError(t, e.Base.Websocket.Disable(), "Websocket disable must not error")
			return weak.Make(e)
		}()
		assert.Eventually(t, func() bool {
			runtime.GC()
			return ref.Value() == nil
		}, 10*time.Second, 10*time.Millisecond, "SetupWebsocket should not keep a disabled exchange reachable")
	})
}
