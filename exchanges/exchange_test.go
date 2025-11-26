package exchange

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

const (
	defaultTestExchange = "Bitfinex"
)

var btcusdPair = currency.NewPairWithDelimiter("BTC", "USD", "-")

func TestSupportsRESTTickerBatchUpdates(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "RAWR",
		Features: Features{
			Supports: FeaturesSupported{
				REST: true,
				RESTCapabilities: protocol.Features{
					TickerBatching: true,
				},
			},
		},
	}

	if !b.SupportsRESTTickerBatchUpdates() {
		t.Fatal("TestSupportsRESTTickerBatchUpdates returned false")
	}
}

func TestSetRunningURL(t *testing.T) {
	t.Parallel()
	b := Base{Name: "HELOOOOOOOO"}
	b.API.Endpoints = b.NewEndpoints()
	assert.ErrorIs(t, b.API.Endpoints.SetRunningURL("meep", "http://google.com/"), errInvalidEndpointKey)

	err := b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1url.com/",
		EdgeCase2: "http://test2url.com/",
	})
	assert.NoError(t, err, "SetDefaultEndpoints should not error")
	err = b.API.Endpoints.SetRunningURL(EdgeCase2.String(), "http://google.com/")
	assert.NoError(t, err, "SetRunningURL should not error")

	val, ok := b.API.Endpoints.defaults[EdgeCase2.String()]
	assert.True(t, ok, "SetRunningURL should have set the value in defaults")
	assert.Equal(t, "http://google.com/", val)

	err = b.API.Endpoints.SetRunningURL(EdgeCase3.String(), "Added Edgecase3")
	assert.ErrorContains(t, err, "invalid URI for request", "SetRunningURL should error on invalid endpoint key")
}

func TestGetURL(t *testing.T) {
	t.Parallel()
	b := Base{
		Name: "HELAAAAAOOOOOOOOO",
	}
	b.API.Endpoints = b.NewEndpoints()
	err := b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1.com/",
		EdgeCase2: "http://test2.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	getVal, err := b.API.Endpoints.GetURL(EdgeCase1)
	if err != nil {
		t.Error(err)
	}
	if getVal != "http://test1.com/" {
		t.Errorf("getVal failed")
	}
	err = b.API.Endpoints.SetRunningURL(EdgeCase2.String(), "http://OVERWRITTENBRO.com.au/")
	if err != nil {
		t.Error(err)
	}
	getChangedVal, err := b.API.Endpoints.GetURL(EdgeCase2)
	if err != nil {
		t.Error(err)
	}
	if getChangedVal != "http://OVERWRITTENBRO.com.au/" {
		t.Error("couldn't get changed val")
	}
	_, err = b.API.Endpoints.GetURL(URL(100))
	if err == nil {
		t.Error("expecting error due to invalid URL key parsed")
	}
}

func TestGetAll(t *testing.T) {
	t.Parallel()
	b := Base{
		Name: "HELLLLLLO",
	}
	b.API.Endpoints = b.NewEndpoints()
	err := b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1.com.au/",
		EdgeCase2: "http://test2.com.au/",
	})
	if err != nil {
		t.Error(err)
	}
	allRunning := b.API.Endpoints.GetURLMap()
	if len(allRunning) != 2 {
		t.Error("invalid running map received")
	}
}

func TestSetDefaultEndpoints(t *testing.T) {
	t.Parallel()
	b := Base{Name: "HELLLLLLO"}
	b.API.Endpoints = b.NewEndpoints()
	err := b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1.com.au/",
		EdgeCase2: "http://test2.com.au/",
	})
	assert.NoError(t, err, "SetDefaultEndpoints should not error")
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		URL(1337): "http://test2.com.au/",
	})
	assert.ErrorIs(t, err, errInvalidEndpointKey, "SetDefaultEndpoints should error on invalid endpoint key")
	err = b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "",
	})
	assert.ErrorContains(t, err, "empty url")
}

func TestSetClientProxyAddress(t *testing.T) {
	t.Parallel()

	requester, err := request.New("rawr",
		common.NewHTTPClientWithTimeout(time.Second*15))
	if err != nil {
		t.Fatal(err)
	}

	newBase := Base{
		Name:      "rawr",
		Requester: requester,
	}

	newBase.Websocket = websocket.NewManager()
	err = newBase.SetClientProxyAddress("")
	if err != nil {
		t.Error(err)
	}
	err = newBase.SetClientProxyAddress(":invalid")
	if err == nil {
		t.Error("SetClientProxyAddress parsed invalid URL")
	}

	if newBase.Websocket.GetProxyAddress() != "" {
		t.Error("SetClientProxyAddress error", err)
	}

	err = newBase.SetClientProxyAddress("http://www.valid.com")
	if err != nil {
		t.Error("SetClientProxyAddress error", err)
	}

	// calling this again will cause the ws check to fail
	err = newBase.SetClientProxyAddress("http://www.valid.com")
	if err == nil {
		t.Error("trying to set the same proxy addr should thrown an err for ws")
	}

	if newBase.Websocket.GetProxyAddress() != "http://www.valid.com" {
		t.Error("SetClientProxyAddress error", err)
	}
}

func TestSetFeatureDefaults(t *testing.T) {
	t.Parallel()

	// Test nil features with basic support capabilities
	b := Base{
		Config: &config.Exchange{
			CurrencyPairs: &currency.PairsManager{},
		},
		Features: Features{
			Supports: FeaturesSupported{
				REST: true,
				RESTCapabilities: protocol.Features{
					TickerBatching: true,
				},
				Websocket: true,
			},
		},
	}
	b.SetFeatureDefaults()
	if !b.Config.Features.Supports.REST && b.Config.CurrencyPairs.LastUpdated == 0 {
		t.Error("incorrect values")
	}

	// Test upgrade when SupportsAutoPairUpdates is enabled
	bptr := func(a bool) *bool { return &a }
	b.Config.Features = nil
	b.Config.SupportsAutoPairUpdates = bptr(true)
	b.SetFeatureDefaults()
	if !b.Config.Features.Supports.RESTCapabilities.AutoPairUpdates &&
		!b.Features.Enabled.AutoPairUpdates {
		t.Error("incorrect values")
	}

	// Test non migrated features config
	b.Config.Features.Supports.REST = false
	b.Config.Features.Supports.RESTCapabilities.TickerBatching = false
	b.Config.Features.Supports.Websocket = false
	b.SetFeatureDefaults()

	if !b.Features.Supports.REST ||
		!b.Features.Supports.RESTCapabilities.TickerBatching ||
		!b.Features.Supports.Websocket {
		t.Error("incorrect values")
	}
}

func TestSetAutoPairDefaults(t *testing.T) {
	t.Parallel()
	bs := "Bitstamp"
	cfg := &config.Config{Exchanges: []config.Exchange{
		{
			Name:          bs,
			CurrencyPairs: &currency.PairsManager{},
			Features: &config.FeaturesConfig{
				Supports: config.FeaturesSupportedConfig{
					RESTCapabilities: protocol.Features{
						AutoPairUpdates: true,
					},
				},
			},
		},
	}}

	exch, err := cfg.GetExchangeConfig(bs)
	if err != nil {
		t.Fatalf("TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if !exch.Features.Supports.RESTCapabilities.AutoPairUpdates {
		t.Fatalf("TestSetAutoPairDefaults Incorrect value")
	}

	if exch.CurrencyPairs.LastUpdated != 0 {
		t.Fatalf("TestSetAutoPairDefaults Incorrect value")
	}

	exch.Features.Supports.RESTCapabilities.AutoPairUpdates = false

	exch, err = cfg.GetExchangeConfig(bs)
	if err != nil {
		t.Fatalf("TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if exch.Features.Supports.RESTCapabilities.AutoPairUpdates {
		t.Fatal("TestSetAutoPairDefaults Incorrect value")
	}
}

func TestSupportsAutoPairUpdates(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	if b.SupportsAutoPairUpdates() {
		t.Error("exchange shouldn't support auto pair updates")
	}

	b.Features.Supports.RESTCapabilities.AutoPairUpdates = true
	if !b.SupportsAutoPairUpdates() {
		t.Error("exchange should support auto pair updates")
	}
}

func TestGetLastPairsUpdateTime(t *testing.T) {
	t.Parallel()

	testTime := time.Now().Unix()
	var b Base
	b.CurrencyPairs.LastUpdated = testTime

	if b.GetLastPairsUpdateTime() != testTime {
		t.Fatal("TestGetLastPairsUpdateTim Incorrect value")
	}
}

func TestGetAssetTypes(t *testing.T) {
	t.Parallel()

	testExchange := Base{
		CurrencyPairs: currency.PairsManager{
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot:    new(currency.PairStore),
				asset.Binary:  new(currency.PairStore),
				asset.Futures: new(currency.PairStore),
			},
		},
	}

	aT := testExchange.GetAssetTypes(false)
	if len(aT) != 3 {
		t.Error("TestGetAssetTypes failed")
	}
}

func TestGetClientBankAccounts(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.TestFile, true)
	if err != nil {
		t.Fatal(err)
	}

	var b Base
	var r *banking.Account
	r, err = b.GetClientBankAccounts("Kraken", "USD")
	if err != nil {
		t.Error(err)
	}

	if r.BankName != "test" {
		t.Error("incorrect bank name")
	}

	_, err = b.GetClientBankAccounts("MEOW", "USD")
	if err == nil {
		t.Error("an error should have been thrown for a non-existent exchange")
	}
}

func TestGetExchangeBankAccounts(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.TestFile, true)
	if err != nil {
		t.Fatal(err)
	}

	b := Base{Name: "Bitfinex"}
	r, err := b.GetExchangeBankAccounts("", "USD")
	if err != nil {
		t.Error(err)
	}

	if r.BankName != "Deutsche Bank Privat Und Geschaeftskunden AG" {
		t.Fatal("incorrect bank name")
	}
}

func TestSetCurrencyPairFormat(t *testing.T) {
	t.Parallel()

	b := Base{
		Config: &config.Exchange{},
	}
	err := b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatal(err)
	}
	if b.Config.CurrencyPairs == nil {
		t.Error("currencyPairs shouldn't be nil")
	}

	// Test global format logic
	b.Config.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.UseGlobalFormat = true
	pFmt := &currency.PairFormat{
		Delimiter: "#",
	}
	b.CurrencyPairs.RequestFormat = pFmt
	b.CurrencyPairs.ConfigFormat = pFmt
	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatal(err)
	}
	spot, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	if spot.Delimiter != "#" {
		t.Error("incorrect pair format delimiter")
	}

	// Test individual asset type formatting logic
	b.CurrencyPairs.UseGlobalFormat = false
	// Store non-nil pair stores
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		ConfigFormat: &currency.PairFormat{Delimiter: "~"},
	})
	require.NoError(t, err, "Store must not error")
	err = b.CurrencyPairs.Store(asset.Futures, &currency.PairStore{
		ConfigFormat: &currency.PairFormat{Delimiter: ":)"},
	})
	require.NoError(t, err, "Store must not error")
	require.NoError(t, b.SetCurrencyPairFormat(), "SetCurrencyPairFormat must not error")
	spot, err = b.GetPairFormat(asset.Spot, false)
	require.NoError(t, err, "GetPairFormat must not error")
	assert.Equal(t, "~", spot.Delimiter, "GetPairFormat should return a format with correct delimiter")
	f, err := b.GetPairFormat(asset.Futures, false)
	require.NoError(t, err, "GetPairFormat must not error")
	assert.Equal(t, ":)", f.Delimiter, "GetPairFormat should return a format with correct delimiter")
}

func TestLoadConfigPairs(t *testing.T) {
	t.Parallel()

	pairs := currency.Pairs{
		currency.Pair{Base: currency.BTC, Quote: currency.USD},
		currency.Pair{Base: currency.LTC, Quote: currency.USD},
	}

	b := Base{
		CurrencyPairs: currency.PairsManager{
			UseGlobalFormat: true,
			RequestFormat: &currency.PairFormat{
				Delimiter: ">",
				Uppercase: false,
			},
			ConfigFormat: &currency.PairFormat{
				Delimiter: "^",
				Uppercase: true,
			},
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					RequestFormat: &currency.EMPTYFORMAT,
					ConfigFormat:  &currency.EMPTYFORMAT,
				},
			},
		},
		Config: &config.Exchange{
			CurrencyPairs: &currency.PairsManager{},
		},
	}

	// Test a nil PairsManager
	err := b.SetConfigPairs()
	if err != nil {
		t.Fatal(err)
	}

	// Now setup a proper PairsManager
	b.Config.CurrencyPairs = &currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Delimiter: "!",
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Delimiter: "!",
			Uppercase: true,
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: true,
				Enabled:      pairs,
				Available:    pairs,
			},
		},
	}

	// Test UseGlobalFormat setting of pairs
	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatal(err)
	}

	err = b.SetConfigPairs()
	if err != nil {
		t.Fatal(err)
	}
	// Test four things:
	// 1) Config pairs are set
	// 2) pair format is set for RequestFormat
	// 3) pair format is set for ConfigFormat
	// 4) Config global format delimiter is updated based off exchange.Base
	pFmt, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}
	pairs, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	p := pairs[0].Format(pFmt).String()
	if p != "BTC^USD" {
		t.Errorf("incorrect value, expected BTC^USD")
	}

	avail, err := b.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	format, err := b.FormatExchangeCurrency(avail[0], asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	p = format.String()
	if p != "btc>usd" {
		t.Error("incorrect value, expected btc>usd")
	}
	if b.Config.CurrencyPairs.RequestFormat.Delimiter != ">" ||
		b.Config.CurrencyPairs.RequestFormat.Uppercase ||
		b.Config.CurrencyPairs.ConfigFormat.Delimiter != "^" ||
		!b.Config.CurrencyPairs.ConfigFormat.Uppercase {
		t.Error("incorrect delimiter values")
	}

	// Test !UseGlobalFormat setting of pairs
	err = b.CurrencyPairs.StoreFormat(asset.Spot, &currency.PairFormat{Delimiter: "~"}, false)
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.StoreFormat(asset.Spot, &currency.PairFormat{Delimiter: "/"}, true)
	if err != nil {
		t.Fatal(err)
	}
	pairs = append(pairs, currency.Pair{Base: currency.XRP, Quote: currency.USD})
	err = b.Config.CurrencyPairs.StorePairs(asset.Spot, pairs, false)
	if err != nil {
		t.Fatal(err)
	}
	err = b.Config.CurrencyPairs.StorePairs(asset.Spot, pairs, true)
	if err != nil {
		t.Fatal(err)
	}
	b.Config.CurrencyPairs.UseGlobalFormat = false
	b.CurrencyPairs.UseGlobalFormat = false

	err = b.SetConfigPairs()
	if err != nil {
		t.Fatal(err)
	}
	// Test four things:
	// 1) XRP-USD is set
	// 2) pair format is set for RequestFormat
	// 3) pair format is set for ConfigFormat
	// 4) Config pair store formats are the same as the exchanges
	configFmt, err := b.GetPairFormat(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}
	pairs, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	p = pairs[2].Format(configFmt).String()
	if p != "xrp/usd" {
		t.Error("incorrect value, expected xrp/usd", p)
	}

	avail, err = b.GetAvailablePairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	format, err = b.FormatExchangeCurrency(avail[2], asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	p = format.String()
	if p != "xrp~usd" {
		t.Error("incorrect value, expected xrp~usd", p)
	}
	ps, err := b.Config.CurrencyPairs.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if ps.RequestFormat.Delimiter != "~" ||
		ps.RequestFormat.Uppercase ||
		ps.ConfigFormat.Delimiter != "/" ||
		ps.ConfigFormat.Uppercase {
		t.Error("incorrect delimiter values")
	}
}

func TestGetName(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	if name := b.GetName(); name != "TESTNAME" {
		t.Error("Exchange GetName() returned incorrect name")
	}
}

func TestGetFeatures(t *testing.T) {
	t.Parallel()

	// Test GetEnabledFeatures
	var b Base
	if b.GetEnabledFeatures().AutoPairUpdates {
		t.Error("auto pair updates should be disabled")
	}
	b.Features.Enabled.AutoPairUpdates = true
	if !b.GetEnabledFeatures().AutoPairUpdates {
		t.Error("auto pair updates should be enabled")
	}

	// Test GetSupportedFeatures
	b.Features.Supports.RESTCapabilities.AutoPairUpdates = true
	if !b.GetSupportedFeatures().RESTCapabilities.AutoPairUpdates {
		t.Error("auto pair updates should be supported")
	}
	if b.GetSupportedFeatures().RESTCapabilities.TickerBatching {
		t.Error("ticker batching shouldn't be supported")
	}
}

// TestGetPairFormat ensures that GetPairFormat delegates to PairsManager.GetFormat
func TestGetPairFormat(t *testing.T) {
	t.Parallel()

	b := new(Base)
	_, err := b.GetPairFormat(asset.Spot, true)
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised)
	b.CurrencyPairs = currency.PairsManager{
		Pairs: make(currency.FullStore),
	}
	_, err = b.GetPairFormat(asset.Spot, true)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Must delegate to GetFormat and error")
}

func TestGetPairs(t *testing.T) {
	t.Parallel()

	b := Base{Name: "TESTNAME"}

	for _, d := range []string{"-", "~", "", "_"} {
		b.CurrencyPairs = currency.PairsManager{
			UseGlobalFormat: true,
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: d,
			},
		}

		require.NoError(t, b.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btcusdPair}, false), "StorePairs must not error for available pairs")
		c, err := b.GetAvailablePairs(asset.Spot)
		require.NoError(t, err, "GetAvailablePairs must not error")
		require.Len(t, c, 1, "Must have one enabled pair")
		assert.Equal(t, "BTC"+d+"USD", c[0].String(), "GetAvailablePairs format should use config format")

		require.NoError(t, b.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{btcusdPair}, true), "StorePairs must not error for enabled pairs")
		require.NoError(t, b.CurrencyPairs.SetAssetEnabled(asset.Spot, true), "SetAssetEnabled must not error")

		c, err = b.GetEnabledPairs(asset.Spot)
		require.NoError(t, err, "GetEnabledPairs must not error")
		require.Len(t, c, 1, "Must have one enabled pair")
		assert.Equal(t, "BTC"+d+"USD", c[0].String(), "GetEnabledPairs format should use config format")
	}
}

// TestFormatExchangeCurrencies exercises FormatExchangeCurrencies
func TestFormatExchangeCurrencies(t *testing.T) {
	t.Parallel()

	e := Base{
		CurrencyPairs: currency.PairsManager{
			UseGlobalFormat: true,

			RequestFormat: &currency.PairFormat{
				Uppercase: false,
				Delimiter: "~",
				Separator: "^",
			},

			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
				Delimiter: "_",
			},
		},
	}

	pairs := []currency.Pair{
		currency.NewPairWithDelimiter("BTC", "USD", "_"),
		currency.NewPairWithDelimiter("LTC", "BTC", "_"),
	}

	got, err := e.FormatExchangeCurrencies(pairs, asset.Spot)
	require.NoError(t, err)
	assert.Equal(t, "btc~usd^ltc~btc", got)

	_, err = e.FormatExchangeCurrencies(nil, asset.Spot)
	assert.ErrorContains(t, err, "returned empty string", "FormatExchangeCurrencies should error correctly")
}

func TestFormatExchangeCurrency(t *testing.T) {
	t.Parallel()

	var b Base
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	actual, err := b.FormatExchangeCurrency(btcusdPair, asset.Spot)
	require.NoError(t, err, "FormatExchangeCurrency must not error")
	assert.Equal(t, "BTC-USD", actual.String(), "FormatExchangeCurrency should format pair correctly")
}

func TestSetEnabled(t *testing.T) {
	t.Parallel()

	SetEnabled := Base{
		Name:    "TESTNAME",
		Enabled: false,
	}

	SetEnabled.SetEnabled(true)
	if !SetEnabled.Enabled {
		t.Error("Exchange SetEnabled(true) did not set boolean")
	}
}

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	IsEnabled := Base{
		Name:    "TESTNAME",
		Enabled: false,
	}

	if IsEnabled.IsEnabled() {
		t.Error("Exchange IsEnabled() did not return correct boolean")
	}
}

func TestSetupDefaults(t *testing.T) {
	t.Parallel()

	newRequester, err := request.New("testSetupDefaults", common.NewHTTPClientWithTimeout(0))
	require.NoError(t, err, "request.New must not error")

	b := Base{
		Name:      "awesomeTest",
		Requester: newRequester,
	}
	cfg := config.Exchange{
		HTTPTimeout: time.Duration(-1),
		API: config.APIConfig{
			AuthenticatedSupport: true,
		},
		ConnectionMonitorDelay: time.Second * 5,
	}

	accountsStore := accounts.GetStore()
	require.NoError(t, b.SetupDefaults(&cfg))
	// If this fails, something raced and changed accounts.global under us. Probably accounts.TestGetStore.
	// Highly unlikely, but this check will clarify what happened
	require.Same(t, accountsStore, accounts.GetStore(), "Global accounts Store must not change during SetupDefaults")

	assert.Equal(t, 15*time.Second, cfg.HTTPTimeout, "config.HTTPTimeout should default correctly")

	cfg.HTTPTimeout = time.Second * 30
	require.NoError(t, b.SetupDefaults(&cfg))
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.HTTPTimeout, "config.HTTPTimeout should respect override")

	// Test asset types
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{Enabled: currency.Pairs{btcusdPair}})
	require.NoError(t, err, "Store must not error")
	require.NoError(t, b.SetupDefaults(&cfg))

	ps, err := cfg.CurrencyPairs.Get(asset.Spot)
	require.NoError(t, err, "CurrencyPairs.Get must not error")
	assert.True(t, ps.Enabled.Contains(btcusdPair, true), "default pair should be stored in the configs pair store")

	exp, err := accountsStore.GetExchangeAccounts(&b)
	require.NoError(t, err, "GetExchangeAccounts must not error")
	assert.Same(t, exp, b.Accounts, "SetupDefaults should default accounts from the global accounts store")
	b.Accounts = accounts.MustNewAccounts(&b)
	a := b.Accounts
	require.NoError(t, b.SetupDefaults(&cfg))
	assert.Same(t, a, b.Accounts, "SetDefaults should not overwrite Accounts override")
}

func TestSetPairs(t *testing.T) {
	t.Parallel()

	b := Base{
		CurrencyPairs: currency.PairsManager{
			UseGlobalFormat: true,
			ConfigFormat: &currency.PairFormat{
				Uppercase: true,
			},
		},
		Config: &config.Exchange{
			CurrencyPairs: &currency.PairsManager{
				UseGlobalFormat: true,
				ConfigFormat: &currency.PairFormat{
					Uppercase: true,
				},
				Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot: {
						AssetEnabled: true,
					},
				},
			},
		},
	}

	if err := b.SetPairs(nil, asset.Spot, true); err == nil {
		t.Error("nil pairs should throw an error")
	}

	pairs := currency.Pairs{
		currency.NewBTCUSD(),
	}
	err := b.SetPairs(pairs, asset.Spot, true)
	if err != nil {
		t.Error(err)
	}

	err = b.SetPairs(pairs, asset.Spot, false)
	if err != nil {
		t.Error(err)
	}

	err = b.SetConfigPairs()
	if err != nil {
		t.Fatal(err)
	}

	p, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if len(p) != 1 {
		t.Error("pairs shouldn't be nil")
	}
}

func TestUpdatePairs(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Exchanges: []config.Exchange{
			{
				Name:          defaultTestExchange,
				CurrencyPairs: &currency.PairsManager{},
			},
		},
	}

	exchCfg, err := cfg.GetExchangeConfig(defaultTestExchange)
	if err != nil {
		t.Fatal("TestUpdatePairs failed to load config")
	}

	UAC := Base{
		Name: defaultTestExchange,
		CurrencyPairs: currency.PairsManager{
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					AssetEnabled: true,
				},
			},
			ConfigFormat:    &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
			UseGlobalFormat: true,
		},
	}
	UAC.Config = exchCfg
	exchangeProducts, err := currency.NewPairsFromStrings([]string{
		"ltcusd",
		"btcusd",
		"usdbtc",
		"audusd",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"btcusd"})
	if err != nil {
		t.Fatal(err)
	}

	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	// Test updating exchange products
	exchangeProducts, err = currency.NewPairsFromStrings([]string{
		"ltcusd",
		"btcusd",
		"usdbtc",
		"audbtc",
	})
	if err != nil {
		t.Fatal(err)
	}
	UAC.Name = defaultTestExchange
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false)
	if err != nil {
		t.Errorf("Exchange UpdatePairs() error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false)
	if err != nil {
		t.Errorf("Exchange UpdatePairs() error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"btcusd"})
	if err != nil {
		t.Fatal(err)
	}
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false)
	if err != nil {
		t.Errorf("Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test update currency pairs with btc excluded
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"ltcusd", "ethusd"})
	if err != nil {
		t.Fatal(err)
	}
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false)
	if err != nil {
		t.Errorf("Exchange UpdatePairs() error: %s", err)
	}

	err = UAC.UpdatePairs(currency.Pairs{currency.EMPTYPAIR, btcusdPair}, asset.Spot, true)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdatePairs should error on empty pairs")

	err = UAC.UpdatePairs(currency.Pairs{btcusdPair, btcusdPair}, asset.Spot, false)
	assert.ErrorIs(t, err, currency.ErrPairDuplication, "UpdatePairs should error on Duplicates")

	err = UAC.UpdatePairs(currency.Pairs{btcusdPair}, asset.Spot, false)
	assert.NoError(t, err, "UpdatePairs should not error")

	err = UAC.UpdatePairs(currency.Pairs{btcusdPair}, asset.Spot, true)
	assert.NoError(t, err, "UpdatePairs should not error")

	UAC.CurrencyPairs.UseGlobalFormat = true
	UAC.CurrencyPairs.ConfigFormat = &currency.PairFormat{Delimiter: "-"}

	uacPairs, err := UAC.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	assert.True(t, uacPairs.Contains(btcusdPair, true), "Should contain currency pair")

	pairs := currency.Pairs{
		currency.NewPair(currency.XRP, currency.USD),
		currency.NewBTCUSD(),
		currency.NewPair(currency.LTC, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, true)
	require.NoError(t, err)

	pairs = currency.Pairs{
		currency.NewPair(currency.WABI, currency.USD),
		currency.NewPair(currency.EASY, currency.USD),
		currency.NewPair(currency.LARIX, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, false)
	require.NoError(t, err)

	uacEnabledPairs, err := UAC.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if uacEnabledPairs.Contains(currency.NewPair(currency.XRP, currency.USD), true) {
		t.Fatal("expected currency pair not found")
	}
	if uacEnabledPairs.Contains(currency.NewBTCUSD(), true) {
		t.Fatal("expected currency pair not found")
	}
	if uacEnabledPairs.Contains(currency.NewPair(currency.LTC, currency.USD), true) {
		t.Fatal("expected currency pair not found")
	}
	if !uacEnabledPairs.Contains(currency.NewPair(currency.LTC, currency.USDT), true) {
		t.Fatal("expected currency pair not found")
	}

	// This should be matched and formatted to `link-usd`
	unintentionalInput, err := currency.NewPairFromString("linkusd")
	require.NoError(t, err)

	pairs = currency.Pairs{
		currency.NewPair(currency.WABI, currency.USD),
		currency.NewPair(currency.EASY, currency.USD),
		currency.NewPair(currency.LARIX, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
		unintentionalInput,
	}

	err = UAC.UpdatePairs(pairs, asset.Spot, true)
	require.NoError(t, err)

	pairs = currency.Pairs{
		currency.NewPair(currency.WABI, currency.USD),
		currency.NewPair(currency.EASY, currency.USD),
		currency.NewPair(currency.LARIX, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
		currency.NewPair(currency.LINK, currency.USD),
	}

	err = UAC.UpdatePairs(pairs, asset.Spot, false)
	require.NoError(t, err)

	uacEnabledPairs, err = UAC.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if !uacEnabledPairs.Contains(currency.NewPair(currency.LINK, currency.USD), true) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}

	err = UAC.UpdatePairs(currency.Pairs{}, asset.Spot, true)
	require.NoError(t, err, "purging all pairs must not error")

	pairs, err = UAC.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)
	require.Empty(t, pairs)

	err = UAC.UpdatePairs(currency.Pairs{}, asset.Spot, false)
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "Purging all available pairs must error")

	avail, err := UAC.GetAvailablePairs(asset.Spot)
	require.NoError(t, err)
	assert.NotEmpty(t, avail, "Failed attempt to purge available pairs should not affect store")
}

func TestSupportsWebsocket(t *testing.T) {
	t.Parallel()

	var b Base
	if b.SupportsWebsocket() {
		t.Error("exchange doesn't support websocket")
	}

	b.Features.Supports.Websocket = true
	if !b.SupportsWebsocket() {
		t.Error("exchange supports websocket")
	}
}

func TestSupportsREST(t *testing.T) {
	t.Parallel()

	var b Base
	if b.SupportsREST() {
		t.Error("exchange doesn't support REST")
	}

	b.Features.Supports.REST = true
	if !b.SupportsREST() {
		t.Error("exchange supports REST")
	}
}

func TestIsWebsocketEnabled(t *testing.T) {
	t.Parallel()

	var b Base
	require.False(t, b.IsWebsocketEnabled(), "IsWebsocketEnabled must return false on an empty Base")

	b.Websocket = websocket.NewManager()
	err := b.Websocket.Setup(&websocket.ManagerSetup{
		ExchangeConfig: &config.Exchange{
			Enabled:                 true,
			WebsocketTrafficTimeout: time.Second * 30,
			Name:                    "test",
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{
					Websocket: true,
				},
			},
		},
		Features:              &protocol.Features{},
		DefaultURL:            "ws://something.com",
		RunningURL:            "ws://something.com",
		Connector:             func() error { return nil },
		GenerateSubscriptions: func() (subscription.List, error) { return nil, nil },
		Subscriber:            func(subscription.List) error { return nil },
	})
	require.NoError(t, err, "Websocket.Setup must not error")
	assert.True(t, b.IsWebsocketEnabled(), "websocket should be enabled")
	require.NoError(t, b.Websocket.Disable(), "Websocket.Disable must not error")
	assert.False(t, b.IsWebsocketEnabled(), "websocket should not be enabled")
}

func TestSupportsWithdrawPermissions(t *testing.T) {
	t.Parallel()

	UAC := Base{Name: defaultTestExchange}
	UAC.Features.Supports.WithdrawPermissions = AutoWithdrawCrypto | AutoWithdrawCryptoWithAPIPermission
	withdrawPermissions := UAC.SupportsWithdrawPermissions(AutoWithdrawCrypto)

	if !withdrawPermissions {
		t.Errorf("Expected: %v, Received: %v", true, withdrawPermissions)
	}

	withdrawPermissions = UAC.SupportsWithdrawPermissions(AutoWithdrawCrypto | AutoWithdrawCryptoWithAPIPermission)
	if !withdrawPermissions {
		t.Errorf("Expected: %v, Received: %v", true, withdrawPermissions)
	}

	withdrawPermissions = UAC.SupportsWithdrawPermissions(AutoWithdrawCrypto | WithdrawCryptoWith2FA)
	if withdrawPermissions {
		t.Errorf("Expected: %v, Received: %v", false, withdrawPermissions)
	}

	withdrawPermissions = UAC.SupportsWithdrawPermissions(AutoWithdrawCrypto | AutoWithdrawCryptoWithAPIPermission | WithdrawCryptoWith2FA)
	if withdrawPermissions {
		t.Errorf("Expected: %v, Received: %v", false, withdrawPermissions)
	}

	withdrawPermissions = UAC.SupportsWithdrawPermissions(WithdrawCryptoWith2FA)
	if withdrawPermissions {
		t.Errorf("Expected: %v, Received: %v", false, withdrawPermissions)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()

	UAC := Base{Name: defaultTestExchange}
	UAC.Features.Supports.WithdrawPermissions = AutoWithdrawCrypto |
		AutoWithdrawCryptoWithAPIPermission |
		AutoWithdrawCryptoWithSetup |
		WithdrawCryptoWith2FA |
		WithdrawCryptoWithSMS |
		WithdrawCryptoWithEmail |
		WithdrawCryptoWithWebsiteApproval |
		WithdrawCryptoWithAPIPermission |
		AutoWithdrawFiat |
		AutoWithdrawFiatWithAPIPermission |
		AutoWithdrawFiatWithSetup |
		WithdrawFiatWith2FA |
		WithdrawFiatWithSMS |
		WithdrawFiatWithEmail |
		WithdrawFiatWithWebsiteApproval |
		WithdrawFiatWithAPIPermission |
		WithdrawCryptoViaWebsiteOnly |
		WithdrawFiatViaWebsiteOnly |
		NoFiatWithdrawals |
		1<<19
	withdrawPermissions := UAC.FormatWithdrawPermissions()
	if withdrawPermissions != "AUTO WITHDRAW CRYPTO & AUTO WITHDRAW CRYPTO WITH API PERMISSION & AUTO WITHDRAW CRYPTO WITH SETUP & WITHDRAW CRYPTO WITH 2FA & WITHDRAW CRYPTO WITH SMS & WITHDRAW CRYPTO WITH EMAIL & WITHDRAW CRYPTO WITH WEBSITE APPROVAL & WITHDRAW CRYPTO WITH API PERMISSION & AUTO WITHDRAW FIAT & AUTO WITHDRAW FIAT WITH API PERMISSION & AUTO WITHDRAW FIAT WITH SETUP & WITHDRAW FIAT WITH 2FA & WITHDRAW FIAT WITH SMS & WITHDRAW FIAT WITH EMAIL & WITHDRAW FIAT WITH WEBSITE APPROVAL & WITHDRAW FIAT WITH API PERMISSION & WITHDRAW CRYPTO VIA WEBSITE ONLY & WITHDRAW FIAT VIA WEBSITE ONLY & NO FIAT WITHDRAWAL & UNKNOWN[1<<19]" {
		t.Errorf("Expected: %s, Received: %s", AutoWithdrawCryptoText+" & "+AutoWithdrawCryptoWithAPIPermissionText, withdrawPermissions)
	}

	UAC.Features.Supports.WithdrawPermissions = NoAPIWithdrawalMethods
	withdrawPermissions = UAC.FormatWithdrawPermissions()

	if withdrawPermissions != NoAPIWithdrawalMethodsText {
		t.Errorf("Expected: %s, Received: %s", NoAPIWithdrawalMethodsText, withdrawPermissions)
	}
}

func TestSupportsAsset(t *testing.T) {
	t.Parallel()
	var b Base
	b.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{
		asset.Spot: {
			AssetEnabled: true,
		},
	}
	assert.True(t, b.SupportsAsset(asset.Spot), "Spot should be supported")
	assert.False(t, b.SupportsAsset(asset.Index), "Index should not be supported")
}

func TestPrintEnabledPairs(t *testing.T) {
	t.Parallel()

	var b Base
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Enabled: currency.Pairs{
			currency.NewBTCUSD(),
		},
	}

	b.PrintEnabledPairs()
}

func TestGetBase(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "MEOW",
	}

	p := b.GetBase()
	p.Name = "rawr"

	if b.Name != "rawr" {
		t.Error("name should be rawr")
	}
}

func TestGetAssetType(t *testing.T) {
	var b Base
	p := currency.NewBTCUSD()
	if _, err := b.GetPairAssetType(p); err == nil {
		t.Fatal("error cannot be nil")
	}
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: true,
		Enabled: currency.Pairs{
			currency.NewBTCUSD(),
		},
		Available: currency.Pairs{
			currency.NewBTCUSD(),
		},
		ConfigFormat: &currency.PairFormat{Delimiter: "-"},
	}

	a, err := b.GetPairAssetType(p)
	if err != nil {
		t.Fatal(err)
	}

	if a != asset.Spot {
		t.Error("should be spot but is", a)
	}
}

func TestGetFormattedPairAndAssetType(t *testing.T) {
	t.Parallel()
	b := Base{
		Config: &config.Exchange{},
	}
	err := b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatal(err)
	}
	b.Config.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.UseGlobalFormat = true
	pFmt := &currency.PairFormat{
		Delimiter: "#",
	}
	b.CurrencyPairs.RequestFormat = pFmt
	b.CurrencyPairs.ConfigFormat = pFmt
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: true,
		Enabled: currency.Pairs{
			currency.NewBTCUSD(),
		},
		Available: currency.Pairs{
			currency.NewBTCUSD(),
		},
	}
	p, a, err := b.GetRequestFormattedPairAndAssetType("btc#usd")
	if err != nil {
		t.Error(err)
	}
	if p.String() != "btc#usd" {
		t.Error("Expected pair to match")
	}
	if a != asset.Spot {
		t.Error("Expected spot asset")
	}
	_, _, err = b.GetRequestFormattedPairAndAssetType("btcusd")
	if err == nil {
		t.Error("Expected error")
	}
}

func TestSetAssetPairStore(t *testing.T) {
	b := Base{
		Config: &config.Exchange{Name: "kitties"},
	}

	err := b.SetAssetPairStore(asset.Empty, currency.PairStore{})
	assert.ErrorIs(t, err, asset.ErrInvalidAsset)

	err = b.SetAssetPairStore(asset.Spot, currency.PairStore{})
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	err = b.SetAssetPairStore(asset.Spot, currency.PairStore{RequestFormat: &currency.PairFormat{Uppercase: true}})
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	err = b.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
	})
	assert.ErrorIs(t, err, errConfigPairFormatRequiresDelimiter)

	err = b.SetAssetPairStore(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	})
	assert.NoError(t, err)
	assert.False(t, b.CurrencyPairs.Pairs[asset.Futures].AssetEnabled, "SetAssetPairStore should not magically enable AssetTypes")

	err = b.SetAssetPairStore(asset.Futures, currency.PairStore{
		AssetEnabled:  true,
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
	})
	assert.NoError(t, err)
	assert.True(t, b.CurrencyPairs.Pairs[asset.Futures].AssetEnabled, "AssetEnabled should be respected")
}

func TestSetGlobalPairsManager(t *testing.T) {
	b := Base{Config: &config.Exchange{Name: "kitties"}}

	err := b.SetGlobalPairsManager(nil, nil, asset.Empty)
	assert.ErrorContains(t, err, "cannot set pairs manager, request pair format not provided")

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, nil, asset.Empty)
	assert.ErrorContains(t, err, "cannot set pairs manager, config pair format not provided")

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, &currency.PairFormat{Uppercase: true})
	assert.ErrorContains(t, err, " cannot set pairs manager, no assets provided")

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, &currency.PairFormat{Uppercase: true}, asset.Empty)
	assert.ErrorContains(t, err, " cannot set global pairs manager config pair format requires delimiter for assets")

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true},
		asset.Spot,
		asset.Binary)
	assert.ErrorIs(t, err, errConfigPairFormatRequiresDelimiter)

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}, asset.Spot, asset.Binary)
	require.NoError(t, err, "SetGlobalPairsManager must not error")

	assert.True(t, b.SupportsAsset(asset.Binary), "Pairs Manager should support Binary")
	assert.True(t, b.SupportsAsset(asset.Spot), "Pairs Manager should support Spot")

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, &currency.PairFormat{Uppercase: true}, asset.Spot, asset.Binary)
	assert.ErrorIs(t, err, errConfigPairFormatRequiresDelimiter, "SetGlobalPairsManager should error correctly")
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	b := Base{}
	for _, tc := range []struct {
		interval kline.Interval
		output   string
	}{
		{
			kline.OneMin,
			"60",
		},
		{
			kline.OneDay,
			"86400",
		},
	} {
		t.Run(tc.interval.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.output, b.FormatExchangeKlineInterval(tc.interval))
		})
	}
}

func TestVerifyKlineParameters(t *testing.T) {
	pairs := currency.Pairs{
		currency.Pair{Base: currency.BTC, Quote: currency.USDT},
	}

	availablePairs := currency.Pairs{
		currency.Pair{Base: currency.BTC, Quote: currency.USDT},
		currency.Pair{Base: currency.BTC, Quote: currency.AUD},
	}

	b := Base{
		Name: "TESTNAME",
		CurrencyPairs: currency.PairsManager{
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					AssetEnabled: true,
					Enabled:      pairs,
					Available:    availablePairs,
				},
			},
		},
		Features: Features{
			Enabled: FeaturesEnabled{
				Kline: kline.ExchangeCapabilitiesEnabled{
					Intervals: kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneMin}),
				},
			},
		},
	}

	assert.ErrorIs(t, b.verifyKlineParameters(availablePairs[0], asset.Index, kline.OneYear), currency.ErrAssetNotFound)
	assert.ErrorIs(t, b.verifyKlineParameters(currency.EMPTYPAIR, asset.Spot, kline.OneMin), currency.ErrCurrencyPairEmpty)
	assert.ErrorIs(t, b.verifyKlineParameters(availablePairs[1], asset.Spot, kline.OneYear), currency.ErrPairNotEnabled)
	assert.ErrorIs(t, b.verifyKlineParameters(availablePairs[0], asset.Spot, kline.OneYear), kline.ErrInvalidInterval)
	assert.NoError(t, b.verifyKlineParameters(availablePairs[0], asset.Spot, kline.OneMin), "verifyKlineParameters should not error")
}

func TestCheckTransientError(t *testing.T) {
	b := Base{}
	err := b.CheckTransientError(nil)
	if err != nil {
		t.Fatal(err)
	}

	err = b.CheckTransientError(errors.New("wow"))
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	nErr := net.DNSError{}
	err = b.CheckTransientError(&nErr)
	if err != nil {
		t.Fatal("error cannot be nil")
	}
}

func TestDisableEnableRateLimiter(t *testing.T) {
	b := Base{}
	err := b.EnableRateLimiter()
	require.ErrorIs(t, err, request.ErrRequestSystemIsNil)

	b.Requester, err = request.New("testingRateLimiter", common.NewHTTPClientWithTimeout(0))
	if err != nil {
		t.Fatal(err)
	}

	err = b.DisableRateLimiter()
	require.NoError(t, err)

	err = b.DisableRateLimiter()
	require.ErrorIs(t, err, request.ErrRateLimiterAlreadyDisabled)

	err = b.EnableRateLimiter()
	require.NoError(t, err)

	err = b.EnableRateLimiter()
	require.ErrorIs(t, err, request.ErrRateLimiterAlreadyEnabled)
}

func TestGetWebsocket(t *testing.T) {
	b := Base{}
	_, err := b.GetWebsocket()
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	b.Websocket = websocket.NewManager()
	_, err = b.GetWebsocket()
	if err != nil {
		t.Fatal(err)
	}
}

func TestFlushWebsocketChannels(t *testing.T) {
	b := Base{}
	err := b.FlushWebsocketChannels()
	if err != nil {
		t.Fatal(err)
	}

	b.Websocket = websocket.NewManager()
	err = b.FlushWebsocketChannels()
	if err == nil {
		t.Fatal(err)
	}
}

func TestSubscribeToWebsocketChannels(t *testing.T) {
	b := Base{}
	err := b.SubscribeToWebsocketChannels(nil)
	if err == nil {
		t.Fatal(err)
	}

	b.Websocket = websocket.NewManager()
	err = b.SubscribeToWebsocketChannels(nil)
	if err == nil {
		t.Fatal(err)
	}
}

func TestUnsubscribeToWebsocketChannels(t *testing.T) {
	b := Base{}
	err := b.UnsubscribeToWebsocketChannels(nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "UnsubscribeToWebsocketChannels should error correctly with a nil Websocket")

	b.Websocket = websocket.NewManager()
	err = b.UnsubscribeToWebsocketChannels(nil)
	assert.NoError(t, err, "UnsubscribeToWebsocketChannels from an empty/nil list should not error")
}

func TestGetSubscriptions(t *testing.T) {
	b := Base{}
	_, err := b.GetSubscriptions()
	if err == nil {
		t.Fatal(err)
	}

	b.Websocket = websocket.NewManager()
	_, err = b.GetSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticateWebsocket(t *testing.T) {
	b := Base{}
	if err := b.AuthenticateWebsocket(t.Context()); err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestKlineIntervalEnabled(t *testing.T) {
	b := Base{}
	if b.klineIntervalEnabled(kline.EightHour) {
		t.Fatal("unexpected value")
	}
}

func TestSetSaveTradeDataStatus(t *testing.T) {
	b := Base{
		Features: Features{
			Enabled: FeaturesEnabled{
				SaveTradeData: false,
			},
		},
		Config: &config.Exchange{
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{},
			},
		},
	}

	if b.IsSaveTradeDataEnabled() {
		t.Errorf("expected false")
	}
	b.SetSaveTradeDataStatus(true)
	if !b.IsSaveTradeDataEnabled() {
		t.Errorf("expected true")
	}
	b.SetSaveTradeDataStatus(false)
	if b.IsSaveTradeDataEnabled() {
		t.Errorf("expected false")
	}
	// data race this
	go b.SetSaveTradeDataStatus(false)
	go b.SetSaveTradeDataStatus(true)
}

func TestAddTradesToBuffer(t *testing.T) {
	b := Base{
		Features: Features{
			Enabled: FeaturesEnabled{},
		},
		Config: &config.Exchange{
			Features: &config.FeaturesConfig{
				Enabled: config.FeaturesEnabledConfig{},
			},
		},
	}
	err := b.AddTradesToBuffer()
	if err != nil {
		t.Error(err)
	}

	b.SetSaveTradeDataStatus(true)
	err = b.AddTradesToBuffer()
	if err != nil {
		t.Error(err)
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		url      URL
		expected string
	}{
		{0, ""},
		{RestSpot, restSpotURL},
		{RestSpotSupplementary, restSpotSupplementaryURL},
		{RestUSDTMargined, restUSDTMarginedFuturesURL},
		{RestCoinMargined, restCoinMarginedFuturesURL},
		{RestFutures, restFuturesURL},
		{RestFuturesSupplementary, restFuturesSupplementaryURL},
		{RestUSDCMargined, restUSDCMarginedFuturesURL},
		{RestSandbox, restSandboxURL},
		{RestSwap, restSwapURL},
		{WebsocketSpot, websocketSpotURL},
		{WebsocketCoinMargined, websocketCoinMarginedURL},
		{WebsocketUSDTMargined, websocketUSDTMarginedURL},
		{WebsocketUSDCMargined, websocketUSDCMarginedURL},
		{WebsocketOptions, websocketOptionsURL},
		{WebsocketTrade, websocketTradeURL},
		{WebsocketPrivate, websocketPrivateURL},
		{WebsocketSpotSupplementary, websocketSpotSupplementaryURL},
		{ChainAnalysis, chainAnalysisURL},
		{EdgeCase1, edgeCase1URL},
		{EdgeCase2, edgeCase2URL},
		{EdgeCase3, edgeCase3URL},
		{420, ""},
	} {
		t.Run(tc.url.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.url.String(), "String() should return the expected URL")
		})
	}
}

func TestFormatSymbol(t *testing.T) {
	b := Base{}
	spotStore := currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat: &currency.PairFormat{
			Delimiter: currency.DashDelimiter,
			Uppercase: true,
		},
	}
	err := b.SetAssetPairStore(asset.Spot, spotStore)
	require.NoError(t, err, "SetAssetPairStore must not error")
	p := currency.NewBTCUSD().Format(*spotStore.ConfigFormat)
	sym, err := b.FormatSymbol(p, asset.Spot)
	require.NoError(t, err, "FormatSymbol must not error")
	assert.Equal(t, "BTCUSD", sym, "FormatSymbol should format the pair correctly")
	_, err = b.FormatSymbol(p, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestSetAPIURL(t *testing.T) {
	b := Base{
		Name: "SomeExchange",
	}
	b.Config = &config.Exchange{}
	var mappy struct {
		Mappymap map[string]string `json:"urlEndpoints"`
	}
	mappy.Mappymap = make(map[string]string)
	mappy.Mappymap["hi"] = "http://google.com/"
	b.Config.API.Endpoints = mappy.Mappymap
	b.API.Endpoints = b.NewEndpoints()
	err := b.SetAPIURL()
	if err == nil {
		t.Error("expecting an error since the key provided is invalid")
	}
	mappy.Mappymap = make(map[string]string)
	b.Config.API.Endpoints = mappy.Mappymap
	mappy.Mappymap["RestSpotURL"] = "hi"
	b.API.Endpoints = b.NewEndpoints()
	err = b.SetAPIURL()
	if err != nil {
		t.Errorf("expecting no error since invalid url value should be logged but received the following error: %v", err)
	}
	mappy.Mappymap = make(map[string]string)
	b.Config.API.Endpoints = mappy.Mappymap
	mappy.Mappymap["RestSpotURL"] = "http://google.com/"
	b.API.Endpoints = b.NewEndpoints()
	err = b.SetAPIURL()
	if err != nil {
		t.Error(err)
	}
	mappy.Mappymap = make(map[string]string)
	b.Config.API.OldEndPoints = &config.APIEndpointsConfig{}
	b.Config.API.Endpoints = mappy.Mappymap
	mappy.Mappymap["RestSpotURL"] = "http://google.com/"
	b.API.Endpoints = b.NewEndpoints()
	b.Config.API.OldEndPoints.URL = "heloo"
	assert.ErrorContains(t, b.SetAPIURL(), "invalid URI for request")

	mappy.Mappymap = make(map[string]string)
	b.Config.API.OldEndPoints = &config.APIEndpointsConfig{}
	b.Config.API.Endpoints = mappy.Mappymap
	mappy.Mappymap["RestSpotURL"] = "http://google.com/"
	b.API.Endpoints = b.NewEndpoints()
	b.Config.API.OldEndPoints.URL = "https://www.bitstamp.net/"
	b.Config.API.OldEndPoints.URLSecondary = "https://www.secondary.net/"
	b.Config.API.OldEndPoints.WebsocketURL = "https://www.websocket.net/"
	err = b.SetAPIURL()
	if err != nil {
		t.Error(err)
	}
	var urlLookup URL
	for x := range keyURLs {
		if keyURLs[x].String() == "RestSpotURL" {
			urlLookup = keyURLs[x]
		}
	}
	urlData, err := b.API.Endpoints.GetURL(urlLookup)
	if err != nil {
		t.Error(err)
	}
	if urlData != "https://www.bitstamp.net/" {
		t.Error("oldendpoints url setting failed")
	}
}

func TestAssetWebsocketFunctionality(t *testing.T) {
	b := Base{}
	if !b.IsAssetWebsocketSupported(asset.Spot) {
		t.Fatal("error asset is not turned off, unexpected response")
	}

	err := b.DisableAssetWebsocketSupport(asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = b.SetAssetPairStore(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	})
	require.NoError(t, err)

	err = b.DisableAssetWebsocketSupport(asset.Spot)
	require.NoError(t, err)

	if b.IsAssetWebsocketSupported(asset.Spot) {
		t.Fatal("error asset is not turned off, unexpected response")
	}

	// Edge case
	b.AssetWebsocketSupport.unsupported = make(map[asset.Item]bool)
	b.AssetWebsocketSupport.unsupported[asset.Spot] = true
	b.AssetWebsocketSupport.unsupported[asset.Futures] = false

	if b.IsAssetWebsocketSupported(asset.Spot) {
		t.Fatal("error asset is turned off, unexpected response")
	}

	if !b.IsAssetWebsocketSupported(asset.Futures) {
		t.Fatal("error asset is not turned off, unexpected response")
	}
}

func TestGetGetURLTypeFromString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Endpoint string
		Expected URL
		Error    error
	}{
		{Endpoint: restSpotURL, Expected: RestSpot},
		{Endpoint: restSpotSupplementaryURL, Expected: RestSpotSupplementary},
		{Endpoint: restUSDTMarginedFuturesURL, Expected: RestUSDTMargined},
		{Endpoint: restCoinMarginedFuturesURL, Expected: RestCoinMargined},
		{Endpoint: restFuturesURL, Expected: RestFutures},
		{Endpoint: restFuturesSupplementaryURL, Expected: RestFuturesSupplementary},
		{Endpoint: restUSDCMarginedFuturesURL, Expected: RestUSDCMargined},
		{Endpoint: restSandboxURL, Expected: RestSandbox},
		{Endpoint: restSwapURL, Expected: RestSwap},
		{Endpoint: websocketSpotURL, Expected: WebsocketSpot},
		{Endpoint: websocketCoinMarginedURL, Expected: WebsocketCoinMargined},
		{Endpoint: websocketUSDTMarginedURL, Expected: WebsocketUSDTMargined},
		{Endpoint: websocketUSDCMarginedURL, Expected: WebsocketUSDCMargined},
		{Endpoint: websocketOptionsURL, Expected: WebsocketOptions},
		{Endpoint: websocketTradeURL, Expected: WebsocketTrade},
		{Endpoint: websocketPrivateURL, Expected: WebsocketPrivate},
		{Endpoint: websocketSpotSupplementaryURL, Expected: WebsocketSpotSupplementary},
		{Endpoint: chainAnalysisURL, Expected: ChainAnalysis},
		{Endpoint: edgeCase1URL, Expected: EdgeCase1},
		{Endpoint: edgeCase2URL, Expected: EdgeCase2},
		{Endpoint: edgeCase3URL, Expected: EdgeCase3},
		{Endpoint: "sillyMcSillyBilly", Expected: 0, Error: errEndpointStringNotFound},
	}

	for _, tt := range testCases {
		t.Run(tt.Endpoint, func(t *testing.T) {
			t.Parallel()
			u, err := getURLTypeFromString(tt.Endpoint)
			require.ErrorIs(t, err, tt.Error)

			if u != tt.Expected {
				t.Fatalf("received: %v but expected: %v", u, tt.Expected)
			}
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetAvailableTransferChains(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestCalculatePNL(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.CalculatePNL(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestScaleCollateral(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.ScaleCollateral(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestCalculateTotalCollateral(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.CalculateTotalCollateral(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	var b Base
	err := b.UpdateCurrencyStates(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestSetTradeFeedStatus(t *testing.T) {
	t.Parallel()
	b := Base{
		Config: &config.Exchange{
			Features: &config.FeaturesConfig{},
		},
		Verbose: true,
	}
	b.SetTradeFeedStatus(true)
	if !b.IsTradeFeedEnabled() {
		t.Error("expected true")
	}
	b.SetTradeFeedStatus(false)
	if b.IsTradeFeedEnabled() {
		t.Error("expected false")
	}
}

func TestSetFillsFeedStatus(t *testing.T) {
	t.Parallel()
	b := Base{
		Config: &config.Exchange{
			Features: &config.FeaturesConfig{},
		},
		Verbose: true,
	}
	b.SetFillsFeedStatus(true)
	if !b.IsFillsFeedEnabled() {
		t.Error("expected true")
	}
	b.SetFillsFeedStatus(false)
	if b.IsFillsFeedEnabled() {
		t.Error("expected false")
	}
}

func TestGetMarginRateHistory(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetMarginRatesHistory(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetFuturesPositionSummary(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetFuturesPositionOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetHistoricalFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetHistoricalFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.IsPerpetualFutureCurrency(asset.Spot, currency.NewBTCUSD())
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetPairAndAssetTypeRequestFormatted(t *testing.T) {
	t.Parallel()

	expected := currency.Pair{Base: currency.BTC, Quote: currency.USDT}
	enabledPairs := currency.Pairs{expected}
	availablePairs := currency.Pairs{
		currency.Pair{Base: currency.BTC, Quote: currency.USDT},
		currency.Pair{Base: currency.BTC, Quote: currency.AUD},
	}

	b := Base{
		CurrencyPairs: currency.PairsManager{
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					AssetEnabled:  true,
					Enabled:       enabledPairs,
					Available:     availablePairs,
					RequestFormat: &currency.PairFormat{Delimiter: "-", Uppercase: true},
					ConfigFormat:  &currency.EMPTYFORMAT,
				},
				asset.PerpetualContract: {
					AssetEnabled:  true,
					Enabled:       enabledPairs,
					Available:     availablePairs,
					RequestFormat: &currency.PairFormat{Delimiter: "_", Uppercase: true},
					ConfigFormat:  &currency.EMPTYFORMAT,
				},
			},
		},
	}

	_, _, err := b.GetPairAndAssetTypeRequestFormatted("")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	_, _, err = b.GetPairAndAssetTypeRequestFormatted("BTCAUD")
	require.ErrorIs(t, err, ErrSymbolNotMatched)

	_, _, err = b.GetPairAndAssetTypeRequestFormatted("BTCUSDT")
	require.ErrorIs(t, err, ErrSymbolNotMatched)

	p, a, err := b.GetPairAndAssetTypeRequestFormatted("BTC-USDT")
	require.NoError(t, err)

	if a != asset.Spot {
		t.Fatal("unexpected value", a)
	}
	if !p.Equal(expected) {
		t.Fatalf("received: '%v' but expected: '%v'", p, expected)
	}

	p, a, err = b.GetPairAndAssetTypeRequestFormatted("BTC_USDT")
	require.NoError(t, err)

	if a != asset.PerpetualContract {
		t.Fatal("unexpected value", a)
	}
	if !p.Equal(expected) {
		t.Fatalf("received: '%v' but expected: '%v'", p, expected)
	}
}

func TestSetRequester(t *testing.T) {
	t.Parallel()

	b := Base{
		Config:    &config.Exchange{Name: "kitties"},
		Requester: nil,
	}

	err := b.SetRequester(nil)
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	requester, err := request.New("testingRequester", common.NewHTTPClientWithTimeout(0))
	if err != nil {
		t.Fatal(err)
	}

	err = b.SetRequester(requester)
	if err != nil {
		t.Fatalf("expected no error, received %v", err)
	}

	if b.Requester == nil {
		t.Fatal("requester not set correctly")
	}
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, _, err := b.GetCollateralCurrencyForContract(asset.Futures, currency.NewPair(currency.XRP, currency.BABYDOGE))
	require.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetCurrencyForRealisedPNL(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, _, err := b.GetCurrencyForRealisedPNL(asset.Empty, currency.EMPTYPAIR)
	require.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestHasAssetTypeAccountSegregation(t *testing.T) {
	t.Parallel()
	b := Base{
		Name: "RAWR",
		Features: Features{
			Supports: FeaturesSupported{
				REST: true,
				RESTCapabilities: protocol.Features{
					HasAssetTypeAccountSegregation: true,
				},
			},
		},
	}

	has := b.HasAssetTypeAccountSegregation()
	if !has {
		t.Errorf("expected '%v' received '%v'", true, false)
	}
}

func TestGetKlineRequest(t *testing.T) {
	t.Parallel()
	b := Base{Name: "klineTest"}
	_, err := b.GetKlineRequest(currency.EMPTYPAIR, asset.Empty, 0, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	p := currency.NewBTCUSDT()
	_, err = b.GetKlineRequest(p, asset.Empty, 0, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetKlineRequest(p, asset.Spot, 0, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, kline.ErrInvalidInterval)

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneDay, Capacity: 1439})
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: true,
		Enabled:      []currency.Pair{p},
		Available:    []currency.Pair{p},
	})
	require.NoError(t, err, "CurrencyPairs.Store must not error")

	_, err = b.GetKlineRequest(p, asset.Spot, 0, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, kline.ErrInvalidInterval)

	_, err = b.GetKlineRequest(p, asset.Spot, kline.OneMin, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, kline.ErrCannotConstructInterval)

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneMin})
	b.Features.Enabled.Kline.GlobalResultLimit = 1439
	_, err = b.GetKlineRequest(p, asset.Spot, kline.OneHour, time.Time{}, time.Time{}, false)
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil)

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  true,
		Enabled:       []currency.Pair{p},
		Available:     []currency.Pair{p},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	})
	require.NoError(t, err, "CurrencyPairs.Store must not error")

	start := time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	_, err = b.GetKlineRequest(p, asset.Spot, kline.OneMin, start, end, true)
	assert.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits)

	_, err = b.GetKlineRequest(p, asset.Spot, kline.OneMin, start, end, false)
	assert.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits)

	_, err = b.GetKlineRequest(p, asset.Futures, kline.OneHour, start, end, false)
	assert.ErrorIs(t, err, currency.ErrAssetNotFound)

	err = b.CurrencyPairs.Store(asset.Futures, &currency.PairStore{
		AssetEnabled:  true,
		Enabled:       []currency.Pair{p},
		Available:     []currency.Pair{p},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	})
	require.NoError(t, err, "CurrencyPairs.Store must not error")

	_, err = b.GetKlineRequest(p, asset.Futures, kline.OneHour, start, end, false)
	assert.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits)

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneHour})
	r, err := b.GetKlineRequest(p, asset.Spot, kline.OneHour, start, end, false)
	require.NoError(t, err, "GetKlineRequest must not error")

	exp := &kline.Request{
		Exchange:         b.Name,
		Pair:             p,
		Asset:            asset.Spot,
		ExchangeInterval: kline.OneHour,
		ClientRequired:   kline.OneHour,
		Start:            start,
		End:              end,
		RequestFormatted: p,
		RequestLimit:     1439,
	}
	assert.Equal(t, exp, r, "GetKlineRequest should return the expected request result")

	end = time.Now().Truncate(kline.OneHour.Duration()).UTC()
	start = end.Add(-kline.OneHour.Duration() * 1439)
	r, err = b.GetKlineRequest(p, asset.Spot, kline.OneHour, start, end, true)
	require.NoError(t, err, "GetKlineRequest must not error")

	exp.Start = start
	exp.End = end
	assert.Equal(t, exp, r, "GetKlineRequest should return the expected request result")
}

func TestGetKlineExtendedRequest(t *testing.T) {
	t.Parallel()
	b := Base{Name: "klineTest"}
	_, err := b.GetKlineExtendedRequest(currency.EMPTYPAIR, asset.Empty, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)

	p := currency.NewBTCUSDT()
	_, err = b.GetKlineExtendedRequest(p, asset.Empty, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = b.GetKlineExtendedRequest(p, asset.Spot, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, kline.ErrInvalidInterval)

	_, err = b.GetKlineExtendedRequest(p, asset.Spot, kline.OneHour, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, kline.ErrCannotConstructInterval)

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneMin})
	b.Features.Enabled.Kline.GlobalResultLimit = 100
	start := time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	_, err = b.GetKlineExtendedRequest(p, asset.Spot, kline.OneHour, start, end)
	assert.ErrorIs(t, err, currency.ErrPairManagerNotInitialised)

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: true,
		Enabled:      []currency.Pair{p},
		Available:    []currency.Pair{p},
	})
	require.NoError(t, err, "CurrencyPairs.Store must not error")

	_, err = b.GetKlineExtendedRequest(p, asset.Spot, kline.OneHour, start, end)
	assert.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetKlineExtendedRequest should error correctly")

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  true,
		Enabled:       []currency.Pair{p},
		Available:     []currency.Pair{p},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	})
	require.NoError(t, err, "CurrencyPairs.Store must not error")

	// The one hour interval is not supported by the exchange. This scenario
	// demonstrates the conversion from the supported 1 minute candles into
	// one hour candles
	r, err := b.GetKlineExtendedRequest(p, asset.Spot, kline.OneHour, start, end)
	require.NoError(t, err, "GetKlineExtendedRequest must not error")

	assert.Equal(t, "klineTest", r.Exchange, "Exchange name should match")
	assert.Equal(t, p, r.Pair, "Pair should match")
	assert.Equal(t, asset.Spot, r.Asset, "Asset should match")
	assert.Equal(t, kline.OneMin, r.ExchangeInterval, "ExchangeInterval should match")
	assert.Equal(t, kline.OneHour, r.ClientRequired, "ClientRequired should match")
	assert.Equal(t, start, r.Request.Start, "Request.Start should match")
	assert.Equal(t, end, r.Request.End, "Request.End should match")
	assert.Equal(t, "BTCUSDT", r.RequestFormatted.String(), "RequestFormatted should match")
	assert.Equal(t, 15, len(r.RangeHolder.Ranges), "RangeHolder.Ranges length should match")
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	b := Base{}
	err := b.SetCollateralMode(t.Context(), asset.Spot, collateral.SingleMode)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, err := b.GetCollateralMode(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	b := Base{}
	err := b.SetMarginType(t.Context(), asset.Spot, currency.NewBTCUSD(), margin.Multi)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, err := b.ChangePositionMargin(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	b := Base{}
	err := b.SetLeverage(t.Context(), asset.Spot, currency.NewBTCUSD(), margin.Multi, 1, order.UnknownSide)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, err := b.GetLeverage(t.Context(), asset.Spot, currency.NewBTCUSD(), margin.Multi, order.UnknownSide)
	assert.ErrorIs(t, err, common.ErrNotYetImplemented)
}

func TestEnsureOnePairEnabled(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	err := b.EnsureOnePairEnabled()
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty)

	b.CurrencyPairs = currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Futures: {},
			asset.Spot: {
				AssetEnabled: true,
				Available: []currency.Pair{
					currency.NewBTCUSDT(),
				},
			},
		},
	}
	err = b.EnsureOnePairEnabled()
	require.NoError(t, err)

	if len(b.CurrencyPairs.Pairs[asset.Spot].Enabled) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(b.CurrencyPairs.Pairs[asset.Spot].Enabled), 1)
	}

	err = b.EnsureOnePairEnabled()
	require.NoError(t, err)

	if len(b.CurrencyPairs.Pairs[asset.Spot].Enabled) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(b.CurrencyPairs.Pairs[asset.Spot].Enabled), 1)
	}
}

func TestGetStandardConfig(t *testing.T) {
	t.Parallel()

	var b *Base
	_, err := b.GetStandardConfig()
	require.ErrorIs(t, err, errExchangeIsNil)

	b = &Base{}
	_, err = b.GetStandardConfig()
	require.ErrorIs(t, err, errSetDefaultsNotCalled)

	b.Name = "test"
	b.Features.Supports.Websocket = true

	cfg, err := b.GetStandardConfig()
	require.NoError(t, err)

	if cfg.Name != "test" {
		t.Fatalf("received: '%v' but expected: '%v'", cfg.Name, "test")
	}

	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Fatalf("received: '%v' but expected: '%v'", cfg.HTTPTimeout, DefaultHTTPTimeout)
	}

	if cfg.WebsocketResponseCheckTimeout != config.DefaultWebsocketResponseCheckTimeout {
		t.Fatalf("received: '%v' but expected: '%v'", cfg.WebsocketResponseCheckTimeout, config.DefaultWebsocketResponseCheckTimeout)
	}

	if cfg.WebsocketResponseMaxLimit != config.DefaultWebsocketResponseMaxLimit {
		t.Fatalf("received: '%v' but expected: '%v'", cfg.WebsocketResponseMaxLimit, config.DefaultWebsocketResponseMaxLimit)
	}

	if cfg.WebsocketTrafficTimeout != config.DefaultWebsocketTrafficTimeout {
		t.Fatalf("received: '%v' but expected: '%v'", cfg.WebsocketTrafficTimeout, config.DefaultWebsocketTrafficTimeout)
	}
}

func TestMatchSymbolWithAvailablePairs(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	whatIWant := currency.NewBTCUSDT()
	err := b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: true,
		Available:    []currency.Pair{whatIWant},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.MatchSymbolWithAvailablePairs("sillBillies", asset.Futures, false)
	require.ErrorIs(t, err, currency.ErrPairNotFound)

	whatIGot, err := b.MatchSymbolWithAvailablePairs("btcusdT", asset.Spot, false)
	require.NoError(t, err)

	if !whatIGot.Equal(whatIWant) {
		t.Fatalf("received: '%v' but expected: '%v'", whatIGot, whatIWant)
	}

	whatIGot, err = b.MatchSymbolWithAvailablePairs("btc-usdT", asset.Spot, true)
	require.NoError(t, err)

	if !whatIGot.Equal(whatIWant) {
		t.Fatalf("received: '%v' but expected: '%v'", whatIGot, whatIWant)
	}
}

func TestMatchSymbolCheckEnabled(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	whatIWant := currency.NewBTCUSDT()
	availButNoEnabled := currency.NewPair(currency.BTC, currency.AUD)
	err := b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: true,
		Available:    []currency.Pair{whatIWant, availButNoEnabled},
		Enabled:      []currency.Pair{whatIWant},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = b.MatchSymbolCheckEnabled("sillBillies", asset.Futures, false)
	require.ErrorIs(t, err, currency.ErrPairNotFound)

	whatIGot, enabled, err := b.MatchSymbolCheckEnabled("btcusdT", asset.Spot, false)
	require.NoError(t, err)

	if !enabled {
		t.Fatal("expected true")
	}

	if !whatIGot.Equal(whatIWant) {
		t.Fatalf("received: '%v' but expected: '%v'", whatIGot, whatIWant)
	}

	whatIGot, enabled, err = b.MatchSymbolCheckEnabled("btc-usdT", asset.Spot, true)
	require.NoError(t, err)

	if !whatIGot.Equal(whatIWant) {
		t.Fatalf("received: '%v' but expected: '%v'", whatIGot, whatIWant)
	}

	if !enabled {
		t.Fatal("expected true")
	}

	whatIGot, enabled, err = b.MatchSymbolCheckEnabled("btc-AUD", asset.Spot, true)
	require.NoError(t, err)

	if !whatIGot.Equal(availButNoEnabled) {
		t.Fatalf("received: '%v' but expected: '%v'", whatIGot, whatIWant)
	}

	if enabled {
		t.Fatal("expected false")
	}
}

func TestIsPairEnabled(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	whatIWant := currency.NewBTCUSDT()
	availButNoEnabled := currency.NewPair(currency.BTC, currency.AUD)
	err := b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: true,
		Available:    []currency.Pair{whatIWant, availButNoEnabled},
		Enabled:      []currency.Pair{whatIWant},
	})
	if err != nil {
		t.Fatal(err)
	}

	enabled, err := b.IsPairEnabled(currency.NewPair(currency.AAA, currency.CYC), asset.Spot)
	require.NoError(t, err)

	if enabled {
		t.Fatal("expected false")
	}

	enabled, err = b.IsPairEnabled(availButNoEnabled, asset.Spot)
	require.NoError(t, err)

	if enabled {
		t.Fatal("expected false")
	}

	enabled, err = b.IsPairEnabled(whatIWant, asset.Spot)
	require.NoError(t, err)

	if !enabled {
		t.Fatal("expected true")
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	var b Base
	_, err := b.GetOpenInterest(t.Context())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetCachedOpenInterest(t *testing.T) {
	t.Parallel()
	var b FakeBase
	b.Features.Supports.FuturesCapabilities.OpenInterest = OpenInterestSupport{
		Supported: true,
	}
	_, err := b.GetCachedOpenInterest(t.Context())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
	b.Features.Supports.FuturesCapabilities.OpenInterest.SupportedViaTicker = true
	b.Name = "test"
	err = ticker.ProcessTicker(&ticker.Price{
		ExchangeName: "test",
		Pair:         currency.NewPair(currency.BTC, currency.BONK),
		AssetType:    asset.Futures,
		OpenInterest: 1337,
	})
	assert.NoError(t, err)

	_, err = b.GetCachedOpenInterest(t.Context())
	assert.NoError(t, err)

	_, err = b.GetCachedOpenInterest(t.Context(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: currency.BONK.Item,
		Asset: asset.Futures,
	})
	assert.NoError(t, err)
}

// TestSetSubscriptionsFromConfig tests the setting and loading of subscriptions from config and exchange defaults
func TestSetSubscriptionsFromConfig(t *testing.T) {
	t.Parallel()
	b := Base{Config: &config.Exchange{Features: &config.FeaturesConfig{}}}
	subs := subscription.List{
		{Channel: subscription.CandlesChannel, Interval: kline.OneDay, Enabled: true},
		{Channel: subscription.OrderbookChannel, Enabled: false},
	}
	b.Features.Subscriptions = subs
	b.SetSubscriptionsFromConfig()
	assert.ElementsMatch(t, subs, b.Config.Features.Subscriptions, "Config Subscriptions should be updated")
	assert.ElementsMatch(t, subscription.List{subs[0]}, b.Features.Subscriptions, "Actual Subscriptions should only contain Enabled")

	subs = subscription.List{
		{Channel: subscription.OrderbookChannel, Enabled: true},
		{Channel: subscription.CandlesChannel, Interval: kline.OneDay, Enabled: false},
	}
	b.Config.Features.Subscriptions = subs
	b.SetSubscriptionsFromConfig()
	assert.ElementsMatch(t, subs, b.Config.Features.Subscriptions, "Config Subscriptions should be the same")
	assert.ElementsMatch(t, subscription.List{subs[0]}, b.Features.Subscriptions, "Subscriptions should only contain Enabled from Config")
}

// TestParallelChanOp unit tests the helper func ParallelChanOp
func TestParallelChanOp(t *testing.T) {
	t.Parallel()
	c := subscription.List{
		{Channel: "red"},
		{Channel: "blue"},
		{Channel: "violent"},
		{Channel: "spin"},
		{Channel: "charm"},
	}
	run := make(chan struct{}, len(c)*2)
	b := Base{}
	errC := make(chan error, 1)
	go func() {
		errC <- b.ParallelChanOp(t.Context(), c, func(_ context.Context, c subscription.List) error {
			time.Sleep(300 * time.Millisecond)
			run <- struct{}{}
			switch c[0].Channel {
			case "spin", "violent":
				return errors.New(c[0].Channel)
			}
			return nil
		}, 1)
	}()
	f := func(ct *assert.CollectT) {
		if assert.Len(ct, errC, 1, "Should eventually have an error") {
			err := <-errC
			assert.ErrorContains(ct, err, "violent", "Should get a violent error")
			assert.ErrorContains(ct, err, "spin", "Should get a spin error")
		}
	}
	assert.EventuallyWithT(t, f, 500*time.Millisecond, 50*time.Millisecond, "ParallelChanOp should complete within 500ms not 5*300ms")
	assert.Len(t, run, len(c), "Every channel was run to completion")
}

func TestGetDefaultConfig(t *testing.T) {
	t.Parallel()

	exch := &FakeBase{}

	_, err := GetDefaultConfig(t.Context(), nil)
	assert.ErrorIs(t, err, errExchangeIsNil)

	c, err := GetDefaultConfig(t.Context(), exch)
	require.NoError(t, err)

	assert.Equal(t, "test", c.Name)
	cpy := exch.Requester

	// Test below demonstrates that the requester is not overwritten so that
	// SetDefaults is not called twice.
	c, err = GetDefaultConfig(t.Context(), exch)
	require.NoError(t, err)

	assert.Equal(t, "test", c.Name)
	assert.Equal(t, cpy, exch.Requester)
}

// TestCanUseAuthenticatedWebsocketEndpoints exercises CanUseAuthenticatedWebsocketEndpoints
func TestCanUseAuthenticatedWebsocketEndpoints(t *testing.T) {
	t.Parallel()
	e := &FakeBase{}
	assert.False(t, e.CanUseAuthenticatedWebsocketEndpoints(), "CanUseAuthenticatedWebsocketEndpoints should return false with nil websocket")
	e.Websocket = websocket.NewManager()
	assert.False(t, e.CanUseAuthenticatedWebsocketEndpoints())
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	assert.True(t, e.CanUseAuthenticatedWebsocketEndpoints())
}

func TestGetCachedTicker(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	pair := currency.NewBTCUSDT()
	_, err := b.GetCachedTicker(pair, asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound)

	err = ticker.ProcessTicker(&ticker.Price{ExchangeName: "test", Pair: pair, AssetType: asset.Spot})
	assert.NoError(t, err)

	tickerPrice, err := b.GetCachedTicker(pair, asset.Spot)
	assert.NoError(t, err)
	assert.Equal(t, pair, tickerPrice.Pair)
}

func TestGetCachedOrderbook(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	pair := currency.NewBTCUSDT()
	_, err := b.GetCachedOrderbook(pair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookNotFound)

	err = (&orderbook.Book{Exchange: "test", Pair: pair, Asset: asset.Spot}).Process()
	assert.NoError(t, err)

	ob, err := b.GetCachedOrderbook(pair, asset.Spot)
	assert.NoError(t, err)
	assert.Equal(t, pair, ob.Pair)
}

func TestGetCachedSubAccounts(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}

	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:    "test",
		Secret: "test",
	})
	_, err := b.GetCachedSubAccounts(ctx, asset.Spot)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	b.Accounts = accounts.MustNewAccounts(&b)
	_, err = b.GetCachedSubAccounts(ctx, asset.Spot)
	assert.ErrorIs(t, err, accounts.ErrNoSubAccounts)

	err = b.Accounts.Save(ctx, accounts.SubAccounts{
		{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.BTC: {Total: 1}}},
	}, true)
	require.NoError(t, err, "b.Accounts.Save must not error")

	_, err = b.GetCachedSubAccounts(ctx, asset.Spot)
	assert.NoError(t, err)
}

func TestGetCurrencyBalances(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}

	_, err := b.GetCachedCurrencyBalances(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, ErrCredentialsAreEmpty)

	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:    "test",
		Secret: "test",
	})
	_, err = b.GetCachedCurrencyBalances(ctx, asset.Spot)
	assert.ErrorIs(t, err, common.ErrNilPointer)

	b.Accounts = accounts.MustNewAccounts(&b)
	_, err = b.GetCachedCurrencyBalances(ctx, asset.Spot)
	assert.ErrorIs(t, err, accounts.ErrNoBalances)

	err = b.Accounts.Save(ctx, accounts.SubAccounts{
		{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.BTC: {Total: 1.4}}},
	}, true)
	require.NoError(t, err, "b.Accounts.Save must not error")

	a, err := b.GetCachedCurrencyBalances(ctx, asset.Spot)
	require.NoError(t, err)
	require.Contains(t, a, currency.BTC)
	assert.Equal(t, 1.4, a[currency.BTC].Total, "BTC Total should be correct")
}

func TestSubscribeAccountBalances(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}

	_, err := b.SubscribeAccountBalances()
	assert.ErrorIs(t, err, common.ErrNilPointer)

	err = dispatch.EnsureRunning(dispatch.DefaultMaxWorkers, dispatch.DefaultJobsLimit)
	require.NoError(t, err, "dispatch.EnsureRunning must not error")

	b.Accounts = accounts.MustNewAccounts(&b)
	p, err := b.SubscribeAccountBalances()
	require.NoError(t, err)

	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:    "test",
		Secret: "test",
	})
	exp := &accounts.SubAccount{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{currency.BTC: {Total: 1.4}}}
	err = b.Accounts.Save(ctx, accounts.SubAccounts{exp}, true)
	require.NoError(t, err, "b.Accounts.Save must not error")
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		select {
		case a := <-p.Channel():
			require.IsType(c, &accounts.SubAccount{}, a, "Save must publish *SubAccount")
			subAcct, _ := a.(*accounts.SubAccount)
			assert.Equal(c, exp, subAcct, "Save should publish the same update")
		default:
			require.Fail(c, "Data must eventually arrive")
		}
	}, time.Second, time.Millisecond, "Publish must eventually send to Channel")
}

// FakeBase is used to override functions
type FakeBase struct{ Base }

func (f *FakeBase) GetOpenInterest(context.Context, ...key.PairAsset) ([]futures.OpenInterest, error) {
	return []futures.OpenInterest{
		{
			Key:          key.NewExchangeAssetPair(f.Name, asset.Futures, currency.NewPair(currency.BTC, currency.BONK)),
			OpenInterest: 1337,
		},
	}, nil
}

func (f *FakeBase) SetDefaults() {
	f.Name = "test"
	f.Requester, _ = request.New("test", common.NewHTTPClientWithTimeout(time.Second))
	f.Features.Supports.RESTCapabilities.AutoPairUpdates = true
}
func (f *FakeBase) UpdateTradablePairs(context.Context) error { return nil }

func (f *FakeBase) Setup(*config.Exchange) error {
	return nil
}

func (f *FakeBase) CancelAllOrders(context.Context, *order.Cancel) (order.CancelAllResponse, error) {
	return order.CancelAllResponse{}, nil
}

func (f *FakeBase) CancelBatchOrders(context.Context, []order.Cancel) (*order.CancelBatchResponse, error) {
	return nil, nil
}

func (f *FakeBase) CancelOrder(context.Context, *order.Cancel) error {
	return nil
}

func (f *FakeBase) GetCachedSubAccounts(context.Context, asset.Item) (accounts.SubAccounts, error) {
	return accounts.SubAccounts{}, nil
}

func (f *FakeBase) GetCachedOrderbook(currency.Pair, asset.Item) (*orderbook.Book, error) {
	return nil, nil
}

func (f *FakeBase) GetCachedTicker(currency.Pair, asset.Item) (*ticker.Price, error) {
	return nil, nil
}

func (f *FakeBase) FetchTradablePairs(context.Context, asset.Item) (currency.Pairs, error) {
	return nil, nil
}

func (f *FakeBase) GetAccountFundingHistory(context.Context) ([]FundingHistory, error) {
	return nil, nil
}

func (f *FakeBase) ValidateAPICredentials(context.Context, asset.Item) error {
	return nil
}

func (f *FakeBase) UpdateTickers(context.Context, asset.Item) error {
	return nil
}

func (f *FakeBase) UpdateTicker(context.Context, currency.Pair, asset.Item) (*ticker.Price, error) {
	return nil, nil
}

func (f *FakeBase) UpdateOrderbook(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
	return nil, nil
}

func (f *FakeBase) UpdateAccountBalances(context.Context, asset.Item) (accounts.SubAccounts, error) {
	return accounts.SubAccounts{}, nil
}

func (f *FakeBase) GetRecentTrades(context.Context, currency.Pair, asset.Item) ([]trade.Data, error) {
	return nil, nil
}

func (f *FakeBase) GetHistoricTrades(context.Context, currency.Pair, asset.Item, time.Time, time.Time) ([]trade.Data, error) {
	return nil, nil
}

func (f *FakeBase) GetServerTime(context.Context, asset.Item) (time.Time, error) {
	return time.Now(), nil
}

func (f *FakeBase) GetFeeByType(context.Context, *FeeBuilder) (float64, error) {
	return 0.0, nil
}

func (f *FakeBase) SubmitOrder(context.Context, *order.Submit) (*order.SubmitResponse, error) {
	return nil, nil
}

func (f *FakeBase) ModifyOrder(context.Context, *order.Modify) (*order.ModifyResponse, error) {
	return nil, nil
}

func (f *FakeBase) GetOrderInfo(context.Context, string, currency.Pair, asset.Item) (*order.Detail, error) {
	return nil, nil
}

func (f *FakeBase) GetDepositAddress(context.Context, currency.Code, string, string) (*deposit.Address, error) {
	return nil, nil
}

func (f *FakeBase) GetOrderHistory(context.Context, *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return nil, nil
}

func (f *FakeBase) GetWithdrawalsHistory(context.Context, currency.Code, asset.Item) ([]WithdrawalHistory, error) {
	return []WithdrawalHistory{}, nil
}

func (f *FakeBase) GetActiveOrders(context.Context, *order.MultiOrderRequest) (order.FilteredOrders, error) {
	return []order.Detail{}, nil
}

func (f *FakeBase) WithdrawCryptocurrencyFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (f *FakeBase) WithdrawFiatFunds(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (f *FakeBase) WithdrawFiatFundsToInternationalBank(context.Context, *withdraw.Request) (*withdraw.ExchangeResponse, error) {
	return nil, nil
}

func (f *FakeBase) GetHistoricCandles(context.Context, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (*kline.Item, error) {
	return &kline.Item{}, nil
}

func (f *FakeBase) GetHistoricCandlesExtended(context.Context, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (*kline.Item, error) {
	return &kline.Item{}, nil
}

func (f *FakeBase) UpdateOrderExecutionLimits(context.Context, asset.Item) error {
	return nil
}

func (f *FakeBase) GetLatestFundingRates(context.Context, *fundingrate.LatestRateRequest) ([]fundingrate.LatestRateResponse, error) {
	return nil, nil
}

func (f *FakeBase) GetFuturesContractDetails(context.Context, asset.Item) ([]futures.Contract, error) {
	return nil, common.ErrFunctionNotSupported
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, err := b.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.NewBTCUSDT())
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetTradingRequirements(t *testing.T) {
	t.Parallel()
	requirements := (*Base)(nil).GetTradingRequirements()
	require.Empty(t, requirements)
	requirements = (&Base{Features: Features{TradingRequirements: protocol.TradingRequirements{ClientOrderID: true}}}).GetTradingRequirements()
	require.NotEmpty(t, requirements)
}

func TestSetConfigPairFormatFromExchange(t *testing.T) {
	t.Parallel()
	b := Base{Config: &config.Exchange{CurrencyPairs: &currency.PairsManager{}}}
	err := b.setConfigPairFormatFromExchange(asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "setConfigPairFormatFromExchange should error correctly without pairs")
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		Enabled:       currency.Pairs{btcusdPair},
		ConfigFormat:  &currency.PairFormat{Delimiter: "🐋"},
		RequestFormat: &currency.PairFormat{Delimiter: "🦥"},
	})
	require.NoError(t, err, "CurrencyPairs.Store must not error")
	err = b.setConfigPairFormatFromExchange(asset.Spot)
	require.NoError(t, err)
	assert.Equal(t, "🐋", b.Config.CurrencyPairs.Pairs[asset.Spot].ConfigFormat.Delimiter, "ConfigFormat should be correct and have a blow hole")
	assert.Equal(t, "🦥", b.Config.CurrencyPairs.Pairs[asset.Spot].RequestFormat.Delimiter, "RequestFormat should be correct and kinda lazy")
}

func TestGetOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	exch := Base{
		Name: "TESTNAME",
	}
	cp := currency.NewBTCUSDT()
	k := key.NewExchangeAssetPair("TESTNAME", asset.Spread, cp)
	l := limits.MinMaxLevel{
		Key:      k,
		MaxPrice: 1337,
	}
	err := limits.Load([]limits.MinMaxLevel{l})
	require.NoError(t, err, "Load must not error")

	_, err = exch.GetOrderExecutionLimits(asset.Spread, cp)
	require.NoError(t, err)
}

func TestCheckOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	exch := Base{
		Name: "TESTNAME",
	}
	cp := currency.NewBTCUSDT()
	k := key.NewExchangeAssetPair("TESTNAME", asset.Spread, cp)
	l := limits.MinMaxLevel{
		Key:      k,
		MaxPrice: 1337,
	}
	err := limits.Load([]limits.MinMaxLevel{
		l,
	})
	require.NoError(t, err, "Load must not error")

	err = exch.CheckOrderExecutionLimits(asset.Spread, cp, 1338.0, 1.0, order.Market)
	require.NoError(t, err, "CheckOrderExecutionLimits must not error")
}

func TestWebsocketSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := (&Base{}).WebsocketSubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketSubmitOrders(t *testing.T) {
	t.Parallel()
	_, err := (&Base{}).WebsocketSubmitOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := (&Base{}).WebsocketModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWebsocketCancelOrder(t *testing.T) {
	t.Parallel()
	err := (&Base{}).WebsocketCancelOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestMessageID(t *testing.T) {
	t.Parallel()
	id := (new(Base)).MessageID()
	require.NotEmpty(t, id, "MessageID must return a non-empty message ID")
	u, err := uuid.FromString(id)
	require.NoError(t, err, "MessageID must return a valid UUID")
	assert.Equal(t, uuid.V7, u.Version(), "MessageID should return a V7 uuid")
}
