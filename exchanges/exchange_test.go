package exchange

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	defaultTestExchange     = "Bitfinex"
	defaultTestCurrencyPair = "BTC-USD"
)

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

func TestCreateMap(t *testing.T) {
	t.Parallel()
	b := Base{
		Name: "HELOOOOOOOO",
	}
	b.API.Endpoints = b.NewEndpoints()
	err := b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1url.com/",
		EdgeCase2: "http://test2url.com/",
	})
	if err != nil {
		t.Error(err)
	}
	val, ok := b.API.Endpoints.defaults[EdgeCase1.String()]
	if !ok || val != "http://test1url.com/" {
		t.Errorf("CreateMap failed, incorrect value received for the given key")
	}
}

func TestSet(t *testing.T) {
	t.Parallel()
	b := Base{
		Name: "HELOOOOOOOO",
	}
	b.API.Endpoints = b.NewEndpoints()
	err := b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1url.com/",
		EdgeCase2: "http://test2url.com/",
	})
	if err != nil {
		t.Error(err)
	}
	err = b.API.Endpoints.SetRunning(EdgeCase2.String(), "http://google.com/")
	if err != nil {
		t.Error(err)
	}
	val, ok := b.API.Endpoints.defaults[EdgeCase2.String()]
	if !ok {
		t.Error("set method or createmap failed")
	}
	if val != "http://google.com/" {
		t.Errorf("vals didn't match. expecting: %s, got: %s\n", "http://google.com/", val)
	}
	err = b.API.Endpoints.SetRunning(EdgeCase3.String(), "Added Edgecase3")
	if err != nil {
		t.Errorf("not expecting an error since invalid url val err should be logged but received: %v", err)
	}
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
	err = b.API.Endpoints.SetRunning(EdgeCase2.String(), "http://OVERWRITTENBRO.com.au/")
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
	b.API.Endpoints = b.NewEndpoints()
	err = b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		URL(1337): "http://test2.com.au/",
	})
	if err == nil {
		t.Error("expecting an error due to invalid url key")
	}
	err = b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "",
	})
	if err != nil {
		t.Errorf("expecting a warning due to invalid url value but got an error: %v", err)
	}
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
		Requester: requester}

	newBase.Websocket = stream.New()
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

	var b = Base{Name: "Bitfinex"}
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
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.Store(asset.Futures, &currency.PairStore{
		ConfigFormat: &currency.PairFormat{Delimiter: ":)"},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatal(err)
	}
	spot, err = b.GetPairFormat(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}
	if spot.Delimiter != "~" {
		t.Error("incorrect pair format delimiter")
	}
	futures, err := b.GetPairFormat(asset.Futures, false)
	if err != nil {
		t.Fatal(err)
	}
	if futures.Delimiter != ":)" {
		t.Error("incorrect pair format delimiter")
	}
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
				AssetEnabled: convert.BoolPtr(true),
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

func TestGetPairFormat(t *testing.T) {
	t.Parallel()

	// Test global formatting
	var b Base
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.ConfigFormat = &currency.PairFormat{
		Uppercase: true,
	}
	b.CurrencyPairs.RequestFormat = &currency.PairFormat{
		Delimiter: "~",
	}
	pFmt, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}
	if pFmt.Delimiter != "~" && !pFmt.Uppercase {
		t.Error("incorrect pair format values")
	}
	pFmt, err = b.GetPairFormat(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}
	if pFmt.Delimiter != "" && pFmt.Uppercase {
		t.Error("incorrect pair format values")
	}

	// Test individual asset pair store formatting
	b.CurrencyPairs.UseGlobalFormat = false
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		ConfigFormat:  &pFmt,
		RequestFormat: &currency.PairFormat{Delimiter: "/", Uppercase: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	pFmt, err = b.GetPairFormat(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}
	if pFmt.Delimiter != "" && pFmt.Uppercase {
		t.Error("incorrect pair format values")
	}
	pFmt, err = b.GetPairFormat(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}
	if pFmt.Delimiter != "~" && !pFmt.Uppercase {
		t.Error("incorrect pair format values")
	}
}

func TestGetEnabledPairs(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	defaultPairs, err := currency.NewPairsFromStrings([]string{defaultTestCurrencyPair})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, defaultPairs, true)
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.StorePairs(asset.Spot, defaultPairs, false)
	if err != nil {
		t.Fatal(err)
	}
	format := currency.PairFormat{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	err = b.CurrencyPairs.SetAssetEnabled(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &format
	b.CurrencyPairs.ConfigFormat = &format

	c, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = "~"
	b.CurrencyPairs.RequestFormat = &format
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].String() != "BTC~USD" {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = ""
	b.CurrencyPairs.ConfigFormat = &format
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].String() != "BTCUSD" {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	btcdoge, err := currency.NewPairsFromStrings([]string{"BTCDOGE"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, true)
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, false)
	if err != nil {
		t.Fatal(err)
	}
	format.Index = currency.BTC.String()
	b.CurrencyPairs.ConfigFormat = &format
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	btcusdUnderscore, err := currency.NewPairsFromStrings([]string{"BTC_USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, btcusdUnderscore, true)
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.StorePairs(asset.Spot, btcusdUnderscore, false)
	if err != nil {
		t.Fatal(err)
	}
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, true)
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, false)
	if err != nil {
		t.Fatal(err)
	}
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Index = currency.BTC.String()
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	btcusd, err := currency.NewPairsFromStrings([]string{"BTCUSD"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, btcusd, true)
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.StorePairs(asset.Spot, btcusd, false)
	if err != nil {
		t.Fatal(err)
	}
	b.CurrencyPairs.ConfigFormat.Index = ""
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}
}

func TestGetAvailablePairs(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	defaultPairs, err := currency.NewPairsFromStrings([]string{defaultTestCurrencyPair})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, defaultPairs, false)
	if err != nil {
		t.Fatal(err)
	}
	format := currency.PairFormat{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	assetType := asset.Spot
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &format
	b.CurrencyPairs.ConfigFormat = &format

	c, err := b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = "~"
	b.CurrencyPairs.RequestFormat = &format
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].String() != "BTC~USD" {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = ""
	b.CurrencyPairs.ConfigFormat = &format
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].String() != "BTCUSD" {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	dogePairs, err := currency.NewPairsFromStrings([]string{"BTCDOGE"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, dogePairs, false)
	if err != nil {
		t.Fatal(err)
	}

	format.Index = currency.BTC.String()
	b.CurrencyPairs.ConfigFormat = &format
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	btcusdUnderscore, err := currency.NewPairsFromStrings([]string{"BTC_USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, btcusdUnderscore, false)
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, dogePairs, false)
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	b.CurrencyPairs.ConfigFormat.Index = currency.BTC.String()
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	btcusd, err := currency.NewPairsFromStrings([]string{"BTCUSD"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, btcusd, false)
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.ConfigFormat.Index = ""
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}
}

func TestSupportsPair(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
		CurrencyPairs: currency.PairsManager{
			Pairs: map[asset.Item]*currency.PairStore{
				asset.Spot: {
					AssetEnabled: convert.BoolPtr(true),
				},
			},
		},
	}

	pairs, err := currency.NewPairsFromStrings([]string{defaultTestCurrencyPair,
		"ETH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, pairs, false)
	if err != nil {
		t.Fatal(err)
	}

	defaultpairs, err := currency.NewPairsFromStrings([]string{defaultTestCurrencyPair})
	if err != nil {
		t.Fatal(err)
	}

	err = b.CurrencyPairs.StorePairs(asset.Spot, defaultpairs, true)
	if err != nil {
		t.Fatal(err)
	}

	format := &currency.PairFormat{
		Delimiter: "-",
		Index:     "",
	}

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = format
	b.CurrencyPairs.ConfigFormat = format
	assetType := asset.Spot

	if b.SupportsPair(currency.NewPair(currency.BTC, currency.USD), true, assetType) != nil {
		t.Error("Exchange SupportsPair() incorrect value")
	}

	if b.SupportsPair(currency.NewPair(currency.ETH, currency.USD), false, assetType) != nil {
		t.Error("Exchange SupportsPair() incorrect value")
	}

	asdasdf, err := currency.NewPairFromStrings("ASD", "ASDF")
	if err != nil {
		t.Fatal(err)
	}

	if b.SupportsPair(asdasdf, true, assetType) == nil {
		t.Error("Exchange SupportsPair() incorrect value")
	}
}

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
	p1, err := currency.NewPairDelimiter("BTC_USD", "_")
	if err != nil {
		t.Fatal(err)
	}
	p2, err := currency.NewPairDelimiter("LTC_BTC", "_")
	if err != nil {
		t.Fatal(err)
	}
	var pairs = []currency.Pair{
		p1,
		p2,
	}

	actual, err := e.FormatExchangeCurrencies(pairs, asset.Spot)
	if err != nil {
		t.Errorf("Exchange TestFormatExchangeCurrencies error %s", err)
	}
	if expected := "btc~usd^ltc~btc"; actual != expected {
		t.Errorf("Exchange TestFormatExchangeCurrencies %s != %s",
			actual, expected)
	}

	_, err = e.FormatExchangeCurrencies(nil, asset.Spot)
	if err == nil {
		t.Error("nil pairs should return an error")
	}
}

func TestFormatExchangeCurrency(t *testing.T) {
	t.Parallel()

	var b Base
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	expected := defaultTestCurrencyPair
	actual, err := b.FormatExchangeCurrency(p, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if actual.String() != expected {
		t.Errorf("Exchange TestFormatExchangeCurrency %s != %s",
			actual, expected)
	}
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

	newRequester, err := request.New("testSetupDefaults",
		common.NewHTTPClientWithTimeout(0))
	if err != nil {
		t.Fatal(err)
	}

	var b = Base{
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

	err = b.SetupDefaults(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPTimeout.String() != "15s" {
		t.Error("HTTP timeout should be set to 15s")
	}

	// Test custom HTTP timeout is set
	cfg.HTTPTimeout = time.Second * 30
	err = b.SetupDefaults(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPTimeout.String() != "30s" {
		t.Error("HTTP timeout should be set to 30s")
	}

	// Test asset types
	p, err := currency.NewPairDelimiter(defaultTestCurrencyPair, "-")
	if err != nil {
		t.Fatal(err)
	}
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		Enabled: currency.Pairs{p},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = b.SetupDefaults(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	ps, err := cfg.CurrencyPairs.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if !ps.Enabled.Contains(p, true) {
		t.Error("default pair should be stored in the configs pair store")
	}

	// Test websocket support
	b.Websocket = stream.New()
	b.Features.Supports.Websocket = true
	err = b.Websocket.Setup(&stream.WebsocketSetup{
		ExchangeConfig: &config.Exchange{
			WebsocketTrafficTimeout: time.Second * 30,
			Name:                    "test",
			Features:                &config.FeaturesConfig{},
		},
		Features:              &protocol.Features{},
		DefaultURL:            "ws://something.com",
		RunningURL:            "ws://something.com",
		Connector:             func() error { return nil },
		GenerateSubscriptions: func() ([]stream.ChannelSubscription, error) { return []stream.ChannelSubscription{}, nil },
		Subscriber:            func(cs []stream.ChannelSubscription) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	err = b.Websocket.Enable()
	if err != nil {
		t.Fatal(err)
	}
	if !b.IsWebsocketEnabled() {
		t.Error("websocket should be enabled")
	}
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
						AssetEnabled: convert.BoolPtr(true),
					},
				},
			},
		},
	}

	if err := b.SetPairs(nil, asset.Spot, true); err == nil {
		t.Error("nil pairs should throw an error")
	}

	pairs := currency.Pairs{
		currency.NewPair(currency.BTC, currency.USD),
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
					AssetEnabled: convert.BoolPtr(true),
				},
			},
			ConfigFormat:    &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
			UseGlobalFormat: true,
		},
	}
	UAC.Config = exchCfg
	exchangeProducts, err := currency.NewPairsFromStrings([]string{"ltcusd",
		"btcusd",
		"usdbtc",
		"audusd"})
	if err != nil {
		t.Fatal(err)
	}
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true, false)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true, false)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"btcusd"})
	if err != nil {
		t.Fatal(err)
	}

	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true, true)
	if err != nil {
		t.Errorf("TestUpdatePairs error: %s", err)
	}

	// Test updating exchange products
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"ltcusd",
		"btcusd",
		"usdbtc",
		"audbtc"})
	if err != nil {
		t.Fatal(err)
	}
	UAC.Name = defaultTestExchange
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("Exchange UpdatePairs() error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("Exchange UpdatePairs() error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"btcusd"})
	if err != nil {
		t.Fatal(err)
	}
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, true)
	if err != nil {
		t.Errorf("Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test update currency pairs with btc excluded
	exchangeProducts, err = currency.NewPairsFromStrings([]string{"ltcusd", "ethusd"})
	if err != nil {
		t.Fatal(err)
	}
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("Exchange UpdatePairs() error: %s", err)
	}

	// Test empty pair
	p, err := currency.NewPairDelimiter(defaultTestCurrencyPair, "-")
	if err != nil {
		t.Fatal(err)
	}
	pairs := currency.Pairs{currency.EMPTYPAIR, p}
	err = UAC.UpdatePairs(pairs, asset.Spot, true, true)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pairs = currency.Pairs{p, p}
	err = UAC.UpdatePairs(pairs, asset.Spot, false, true)
	if !errors.Is(err, currency.ErrPairDuplication) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrPairDuplication)
	}

	pairs = currency.Pairs{p}
	err = UAC.UpdatePairs(pairs, asset.Spot, false, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = UAC.UpdatePairs(pairs, asset.Spot, true, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	UAC.CurrencyPairs.UseGlobalFormat = true
	UAC.CurrencyPairs.ConfigFormat = &currency.PairFormat{
		Delimiter: "-",
	}

	uacPairs, err := UAC.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if !uacPairs.Contains(p, true) {
		t.Fatal("expected currency pair not found")
	}

	pairs = currency.Pairs{
		currency.NewPair(currency.XRP, currency.USD),
		currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, true, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	pairs = currency.Pairs{
		currency.NewPair(currency.WABI, currency.USD),
		currency.NewPair(currency.EASY, currency.USD),
		currency.NewPair(currency.LARIX, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, false, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	uacEnabledPairs, err := UAC.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if uacEnabledPairs.Contains(currency.NewPair(currency.XRP, currency.USD), true) {
		t.Fatal("expected currency pair not found")
	}
	if uacEnabledPairs.Contains(currency.NewPair(currency.BTC, currency.USD), true) {
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
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	pairs = currency.Pairs{
		currency.NewPair(currency.WABI, currency.USD),
		currency.NewPair(currency.EASY, currency.USD),
		currency.NewPair(currency.LARIX, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
		unintentionalInput,
	}

	err = UAC.UpdatePairs(pairs, asset.Spot, true, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	pairs = currency.Pairs{
		currency.NewPair(currency.WABI, currency.USD),
		currency.NewPair(currency.EASY, currency.USD),
		currency.NewPair(currency.LARIX, currency.USD),
		currency.NewPair(currency.LTC, currency.USDT),
		currency.NewPair(currency.LINK, currency.USD),
	}

	err = UAC.UpdatePairs(pairs, asset.Spot, false, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	uacEnabledPairs, err = UAC.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if !uacEnabledPairs.Contains(currency.NewPair(currency.LINK, currency.USD), true) {
		t.Fatalf("received: '%v' but expected: '%v'", false, true)
	}
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
	if b.IsWebsocketEnabled() {
		t.Error("exchange doesn't support websocket")
	}

	b.Websocket = stream.New()
	err := b.Websocket.Setup(&stream.WebsocketSetup{
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
		GenerateSubscriptions: func() ([]stream.ChannelSubscription, error) { return nil, nil },
		Subscriber:            func(cs []stream.ChannelSubscription) error { return nil },
	})
	if err != nil {
		t.Error(err)
	}
	if !b.IsWebsocketEnabled() {
		t.Error("websocket should be enabled")
	}
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
		asset.Spot: {},
	}
	if !b.SupportsAsset(asset.Spot) {
		t.Error("spot should be supported")
	}
	if b.SupportsAsset(asset.Index) {
		t.Error("index shouldn't be supported")
	}
}

func TestPrintEnabledPairs(t *testing.T) {
	t.Parallel()

	var b Base
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Enabled: currency.Pairs{
			currency.NewPair(currency.BTC, currency.USD),
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
	p := currency.NewPair(currency.BTC, currency.USD)
	if _, err := b.GetPairAssetType(p); err == nil {
		t.Fatal("error cannot be nil")
	}
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		Enabled: currency.Pairs{
			currency.NewPair(currency.BTC, currency.USD),
		},
		Available: currency.Pairs{
			currency.NewPair(currency.BTC, currency.USD),
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
		AssetEnabled: convert.BoolPtr(true),
		Enabled: currency.Pairs{
			currency.NewPair(currency.BTC, currency.USD),
		},
		Available: currency.Pairs{
			currency.NewPair(currency.BTC, currency.USD),
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

func TestStoreAssetPairFormat(t *testing.T) {
	b := Base{
		Config: &config.Exchange{Name: "kitties"},
	}

	err := b.StoreAssetPairFormat(asset.Empty, currency.PairStore{})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.StoreAssetPairFormat(asset.Spot, currency.PairStore{})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.StoreAssetPairFormat(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true}})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.StoreAssetPairFormat(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true}})
	if !errors.Is(err, errConfigPairFormatRequiresDelimiter) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigPairFormatRequiresDelimiter)
	}

	err = b.StoreAssetPairFormat(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}})
	if err != nil {
		t.Error(err)
	}

	err = b.StoreAssetPairFormat(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}})
	if err != nil {
		t.Error(err)
	}
}

func TestSetGlobalPairsManager(t *testing.T) {
	b := Base{
		Config: &config.Exchange{Name: "kitties"},
	}

	err := b.SetGlobalPairsManager(nil, nil, asset.Empty)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, nil, asset.Empty)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true}, asset.Empty)
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true},
		asset.Spot,
		asset.Binary)
	if !errors.Is(err, errConfigPairFormatRequiresDelimiter) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errConfigPairFormatRequiresDelimiter)
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
		asset.Spot,
		asset.Binary)
	if err != nil {
		t.Error(err)
	}

	if !b.SupportsAsset(asset.Binary) || !b.SupportsAsset(asset.Spot) {
		t.Fatal("global pairs manager not set correctly")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true}, asset.Spot, asset.Binary)
	if err == nil {
		t.Error("error cannot be nil")
	}
}
func Test_FormatExchangeKlineInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"60",
		},
		{
			"OneDay",
			kline.OneDay,
			"86400",
		},
	}

	b := Base{}
	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := b.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestBase_ValidateKline(t *testing.T) {
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
					AssetEnabled: convert.BoolPtr(true),
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

	err := b.ValidateKline(availablePairs[0], asset.Spot, kline.OneMin)
	if err != nil {
		t.Fatalf("expected validation to pass received error: %v", err)
	}

	err = b.ValidateKline(availablePairs[1], asset.Spot, kline.OneYear)
	if err == nil {
		t.Fatal("expected validation to fail")
	}

	err = b.ValidateKline(availablePairs[1], asset.Index, kline.OneYear)
	if err == nil {
		t.Fatal("expected validation to fail")
	}
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
	if !errors.Is(err, request.ErrRequestSystemIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, request.ErrRequestSystemIsNil)
	}

	b.Requester, err = request.New("testingRateLimiter", common.NewHTTPClientWithTimeout(0))
	if err != nil {
		t.Fatal(err)
	}

	err = b.DisableRateLimiter()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = b.DisableRateLimiter()
	if !errors.Is(err, request.ErrRateLimiterAlreadyDisabled) {
		t.Fatalf("received: '%v' but expected: '%v'", err, request.ErrRateLimiterAlreadyDisabled)
	}

	err = b.EnableRateLimiter()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = b.EnableRateLimiter()
	if !errors.Is(err, request.ErrRateLimiterAlreadyEnabled) {
		t.Fatalf("received: '%v' but expected: '%v'", err, request.ErrRateLimiterAlreadyEnabled)
	}
}

func TestGetWebsocket(t *testing.T) {
	b := Base{}
	_, err := b.GetWebsocket()
	if err == nil {
		t.Fatal("error cannot be nil")
	}
	b.Websocket = &stream.Websocket{}
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

	b.Websocket = &stream.Websocket{}
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

	b.Websocket = &stream.Websocket{}
	err = b.SubscribeToWebsocketChannels(nil)
	if err == nil {
		t.Fatal(err)
	}
}

func TestUnsubscribeToWebsocketChannels(t *testing.T) {
	b := Base{}
	err := b.UnsubscribeToWebsocketChannels(nil)
	if err == nil {
		t.Fatal(err)
	}

	b.Websocket = &stream.Websocket{}
	err = b.UnsubscribeToWebsocketChannels(nil)
	if err == nil {
		t.Fatal(err)
	}
}

func TestGetSubscriptions(t *testing.T) {
	b := Base{}
	_, err := b.GetSubscriptions()
	if err == nil {
		t.Fatal(err)
	}

	b.Websocket = &stream.Websocket{}
	_, err = b.GetSubscriptions()
	if err != nil {
		t.Fatal(err)
	}
}

func TestAuthenticateWebsocket(t *testing.T) {
	b := Base{}
	if err := b.AuthenticateWebsocket(context.Background()); err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestKlineIntervalEnabled(t *testing.T) {
	b := Base{}
	if b.klineIntervalEnabled(kline.EightHour) {
		t.Fatal("unexpected value")
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	b := Base{}
	if b.FormatExchangeKlineInterval(kline.EightHour) != "28800" {
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
	if RestSpot.String() != restSpotURL {
		t.Errorf("received '%v' expected '%v'", RestSpot, restSpotURL)
	}
	if RestSpotSupplementary.String() != restSpotSupplementaryURL {
		t.Errorf("received '%v' expected '%v'", RestSpotSupplementary, restSpotSupplementaryURL)
	}
	if RestUSDTMargined.String() != "RestUSDTMarginedFuturesURL" {
		t.Errorf("received '%v' expected '%v'", RestUSDTMargined, "RestUSDTMarginedFuturesURL")
	}
	if RestCoinMargined.String() != restCoinMarginedFuturesURL {
		t.Errorf("received '%v' expected '%v'", RestCoinMargined, restCoinMarginedFuturesURL)
	}
	if RestFutures.String() != restFuturesURL {
		t.Errorf("received '%v' expected '%v'", RestFutures, restFuturesURL)
	}
	if RestFuturesSupplementary.String() != restFuturesSupplementaryURL {
		t.Errorf("received '%v' expected '%v'", RestFutures, restFuturesSupplementaryURL)
	}
	if RestUSDCMargined.String() != restUSDCMarginedFuturesURL {
		t.Errorf("received '%v' expected '%v'", RestUSDCMargined, restUSDCMarginedFuturesURL)
	}
	if RestSandbox.String() != restSandboxURL {
		t.Errorf("received '%v' expected '%v'", RestSandbox, restSandboxURL)
	}
	if RestSwap.String() != restSwapURL {
		t.Errorf("received '%v' expected '%v'", RestSwap, restSwapURL)
	}
	if WebsocketSpot.String() != websocketSpotURL {
		t.Errorf("received '%v' expected '%v'", WebsocketSpot, websocketSpotURL)
	}
	if WebsocketSpotSupplementary.String() != websocketSpotSupplementaryURL {
		t.Errorf("received '%v' expected '%v'", WebsocketSpotSupplementary, websocketSpotSupplementaryURL)
	}
	if ChainAnalysis.String() != chainAnalysisURL {
		t.Errorf("received '%v' expected '%v'", ChainAnalysis, chainAnalysisURL)
	}
	if EdgeCase1.String() != edgeCase1URL {
		t.Errorf("received '%v' expected '%v'", EdgeCase1, edgeCase1URL)
	}
	if EdgeCase2.String() != edgeCase2URL {
		t.Errorf("received '%v' expected '%v'", EdgeCase2, edgeCase2URL)
	}
	if EdgeCase3.String() != edgeCase3URL {
		t.Errorf("received '%v' expected '%v'", EdgeCase3, edgeCase3URL)
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
	err := b.StoreAssetPairFormat(asset.Spot, spotStore)
	if err != nil {
		t.Error(err)
	}
	pair, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	sym, err := b.FormatSymbol(pair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
	if sym != "BTCUSD" {
		t.Error("formatting failed")
	}
	_, err = b.FormatSymbol(pair, asset.Futures)
	if err == nil {
		t.Error("expecting an error since asset pair format has not been set")
	}
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
	err = b.SetAPIURL()
	if err != nil {
		t.Errorf("expecting a warning since invalid oldendpoints url but got an error: %v", err)
	}
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

func TestSetRunning(t *testing.T) {
	b := Base{
		Name: "HELOOOOOOOO",
	}
	b.API.Endpoints = b.NewEndpoints()
	err := b.API.Endpoints.SetRunning(EdgeCase1.String(), "http://google.com/")
	if err != nil {
		t.Error(err)
	}
}

func TestAssetWebsocketFunctionality(t *testing.T) {
	b := Base{}
	if !b.IsAssetWebsocketSupported(asset.Spot) {
		t.Fatal("error asset is not turned off, unexpected response")
	}

	err := b.DisableAssetWebsocketSupport(asset.Spot)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("expected error: %v but received: %v", asset.ErrNotSupported, err)
	}

	err = b.StoreAssetPairFormat(asset.Spot, currency.PairStore{
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		ConfigFormat: &currency.PairFormat{
			Uppercase: true,
			Delimiter: currency.DashDelimiter,
		},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = b.DisableAssetWebsocketSupport(asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("expected error: %v but received: %v", nil, err)
	}

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
		{Endpoint: "RestSpotURL", Expected: RestSpot},
		{Endpoint: "RestSpotSupplementaryURL", Expected: RestSpotSupplementary},
		{Endpoint: "RestUSDTMarginedFuturesURL", Expected: RestUSDTMargined},
		{Endpoint: "RestCoinMarginedFuturesURL", Expected: RestCoinMargined},
		{Endpoint: "RestFuturesURL", Expected: RestFutures},
		{Endpoint: "RestUSDCMarginedFuturesURL", Expected: RestUSDCMargined},
		{Endpoint: "RestSandboxURL", Expected: RestSandbox},
		{Endpoint: "RestSwapURL", Expected: RestSwap},
		{Endpoint: "WebsocketSpotURL", Expected: WebsocketSpot},
		{Endpoint: "WebsocketSpotSupplementaryURL", Expected: WebsocketSpotSupplementary},
		{Endpoint: "ChainAnalysisURL", Expected: ChainAnalysis},
		{Endpoint: "EdgeCase1URL", Expected: EdgeCase1},
		{Endpoint: "EdgeCase2URL", Expected: EdgeCase2},
		{Endpoint: "EdgeCase3URL", Expected: EdgeCase3},
		{Endpoint: "sillyMcSillyBilly", Expected: 0, Error: errEndpointStringNotFound},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.Endpoint, func(t *testing.T) {
			t.Parallel()
			u, err := getURLTypeFromString(tt.Endpoint)
			if !errors.Is(err, tt.Error) {
				t.Fatalf("received: %v but expected: %v", err, tt.Error)
			}

			if u != tt.Expected {
				t.Fatalf("received: %v but expected: %v", u, tt.Expected)
			}
		})
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.GetAvailableTransferChains(context.Background(), currency.BTC); !errors.Is(err, common.ErrFunctionNotSupported) {
		t.Errorf("received: %v, expected: %v", err, common.ErrFunctionNotSupported)
	}
}

func TestCalculatePNL(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.CalculatePNL(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestScaleCollateral(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.ScaleCollateral(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestCalculateTotalCollateral(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.CalculateTotalCollateral(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	var b Base
	if err := b.UpdateCurrencyStates(context.Background(), asset.Spot); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	var b Base
	if err := b.UpdateOrderExecutionLimits(context.Background(), asset.Spot); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
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
	if _, err := b.GetMarginRatesHistory(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.GetPositionSummary(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.GetFuturesPositions(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestGetFundingPaymentDetails(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.GetFundingPaymentDetails(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.GetLatestFundingRate(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.GetFundingRates(context.Background(), nil); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	var b Base
	if _, err := b.IsPerpetualFutureCurrency(asset.Spot, currency.NewPair(currency.BTC, currency.USD)); !errors.Is(err, common.ErrNotYetImplemented) {
		t.Errorf("received: %v, expected: %v", err, common.ErrNotYetImplemented)
	}
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
					AssetEnabled:  convert.BoolPtr(true),
					Enabled:       enabledPairs,
					Available:     availablePairs,
					RequestFormat: &currency.PairFormat{Delimiter: "-", Uppercase: true},
					ConfigFormat:  &currency.EMPTYFORMAT,
				},
				asset.PerpetualContract: {
					AssetEnabled:  convert.BoolPtr(true),
					Enabled:       enabledPairs,
					Available:     availablePairs,
					RequestFormat: &currency.PairFormat{Delimiter: "_", Uppercase: true},
					ConfigFormat:  &currency.EMPTYFORMAT,
				},
			},
		},
	}

	_, _, err := b.GetPairAndAssetTypeRequestFormatted("")
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	_, _, err = b.GetPairAndAssetTypeRequestFormatted("BTCAUD")
	if !errors.Is(err, errSymbolCannotBeMatched) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errSymbolCannotBeMatched)
	}

	_, _, err = b.GetPairAndAssetTypeRequestFormatted("BTCUSDT")
	if !errors.Is(err, errSymbolCannotBeMatched) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errSymbolCannotBeMatched)
	}

	p, a, err := b.GetPairAndAssetTypeRequestFormatted("BTC-USDT")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if a != asset.Spot {
		t.Fatal("unexpected value", a)
	}
	if !p.Equal(expected) {
		t.Fatalf("received: '%v' but expected: '%v'", p, expected)
	}

	p, a, err = b.GetPairAndAssetTypeRequestFormatted("BTC_USDT")
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
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
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNotYetImplemented)
	}
}

func TestGetCurrencyForRealisedPNL(t *testing.T) {
	t.Parallel()
	b := Base{}
	_, _, err := b.GetCurrencyForRealisedPNL(asset.Empty, currency.EMPTYPAIR)
	if !errors.Is(err, common.ErrNotYetImplemented) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNotYetImplemented)
	}
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
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pair := currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.GetKlineRequest(pair, asset.Empty, 0, time.Time{}, time.Time{}, false)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	_, err = b.GetKlineRequest(pair, asset.Spot, 0, time.Time{}, time.Time{}, false)
	if !errors.Is(err, kline.ErrInvalidInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrInvalidInterval)
	}

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneDay, Capacity: 1439})
	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		Enabled:      []currency.Pair{pair},
		Available:    []currency.Pair{pair},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = b.GetKlineRequest(pair, asset.Spot, 0, time.Time{}, time.Time{}, false)
	if !errors.Is(err, kline.ErrInvalidInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrInvalidInterval)
	}

	_, err = b.GetKlineRequest(pair, asset.Spot, kline.OneMin, time.Time{}, time.Time{}, false)
	if !errors.Is(err, kline.ErrCannotConstructInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrCannotConstructInterval)
	}

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneMin})
	b.Features.Enabled.Kline.GlobalResultLimit = 1439
	_, err = b.GetKlineRequest(pair, asset.Spot, kline.OneHour, time.Time{}, time.Time{}, false)
	if !errors.Is(err, errAssetRequestFormatIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAssetRequestFormatIsNil)
	}

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		Enabled:       []currency.Pair{pair},
		Available:     []currency.Pair{pair},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	start := time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	_, err = b.GetKlineRequest(pair, asset.Spot, kline.OneMin, start, end, true)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}

	_, err = b.GetKlineRequest(pair, asset.Spot, kline.OneMin, start, end, false)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}

	_, err = b.GetKlineRequest(pair, asset.Futures, kline.OneHour, start, end, false)
	if !errors.Is(err, asset.ErrNotEnabled) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotEnabled)
	}

	err = b.CurrencyPairs.Store(asset.Futures, &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		Enabled:       []currency.Pair{pair},
		Available:     []currency.Pair{pair},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	_, err = b.GetKlineRequest(pair, asset.Futures, kline.OneHour, start, end, false)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneHour})
	r, err := b.GetKlineRequest(pair, asset.Spot, kline.OneHour, start, end, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r.Exchange != "klineTest" {
		t.Fatalf("received: '%v' but expected: '%v'", r.Exchange, "klineTest")
	}

	if !r.Pair.Equal(pair) {
		t.Fatalf("received: '%v' but expected: '%v'", r.Pair, pair)
	}

	if r.Asset != asset.Spot {
		t.Fatalf("received: '%v' but expected: '%v'", r.Asset, asset.Spot)
	}

	if r.ExchangeInterval != kline.OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", r.ExchangeInterval, kline.OneHour)
	}

	if r.ClientRequired != kline.OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", r.ClientRequired, kline.OneHour)
	}

	if r.Start != start {
		t.Fatalf("received: '%v' but expected: '%v'", r.Start, start)
	}

	if r.End != end {
		t.Fatalf("received: '%v' but expected: '%v'", r.End, end)
	}

	if r.RequestFormatted.String() != "BTCUSDT" {
		t.Fatalf("received: '%v' but expected: '%v'", r.RequestFormatted.String(), "BTCUSDT")
	}

	end = time.Now().Truncate(kline.OneHour.Duration()).UTC()
	start = end.Add(-kline.OneHour.Duration() * 1439)

	r, err = b.GetKlineRequest(pair, asset.Spot, kline.OneHour, start, end, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r.Exchange != "klineTest" {
		t.Fatalf("received: '%v' but expected: '%v'", r.Exchange, "klineTest")
	}

	if !r.Pair.Equal(pair) {
		t.Fatalf("received: '%v' but expected: '%v'", r.Pair, pair)
	}

	if r.Asset != asset.Spot {
		t.Fatalf("received: '%v' but expected: '%v'", r.Asset, asset.Spot)
	}

	if r.ExchangeInterval != kline.OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", r.ExchangeInterval, kline.OneHour)
	}

	if r.ClientRequired != kline.OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", r.ClientRequired, kline.OneHour)
	}

	if r.Start != start {
		t.Fatalf("received: '%v' but expected: '%v'", r.Start, start)
	}

	if r.End != end {
		t.Fatalf("received: '%v' but expected: '%v'", r.End, end)
	}

	if r.RequestFormatted.String() != "BTCUSDT" {
		t.Fatalf("received: '%v' but expected: '%v'", r.RequestFormatted.String(), "BTCUSDT")
	}
}

func TestGetKlineExtendedRequest(t *testing.T) {
	t.Parallel()
	b := Base{Name: "klineTest"}
	_, err := b.GetKlineExtendedRequest(currency.EMPTYPAIR, asset.Empty, 0, time.Time{}, time.Time{})
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyPairEmpty)
	}

	pair := currency.NewPair(currency.BTC, currency.USDT)
	_, err = b.GetKlineExtendedRequest(pair, asset.Empty, 0, time.Time{}, time.Time{})
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	_, err = b.GetKlineExtendedRequest(pair, asset.Spot, 0, time.Time{}, time.Time{})
	if !errors.Is(err, kline.ErrInvalidInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrInvalidInterval)
	}

	_, err = b.GetKlineExtendedRequest(pair, asset.Spot, kline.OneHour, time.Time{}, time.Time{})
	if !errors.Is(err, kline.ErrCannotConstructInterval) {
		t.Fatalf("received: '%v' but expected: '%v'", err, kline.ErrCannotConstructInterval)
	}

	b.Features.Enabled.Kline.Intervals = kline.DeployExchangeIntervals(kline.IntervalCapacity{Interval: kline.OneMin})
	b.Features.Enabled.Kline.GlobalResultLimit = 100
	start := time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	_, err = b.GetKlineExtendedRequest(pair, asset.Spot, kline.OneHour, start, end)
	if !errors.Is(err, asset.ErrNotEnabled) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotEnabled)
	}

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled: convert.BoolPtr(true),
		Enabled:      []currency.Pair{pair},
		Available:    []currency.Pair{pair},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	_, err = b.GetKlineExtendedRequest(pair, asset.Spot, kline.OneHour, start, end)
	if !errors.Is(err, errAssetRequestFormatIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAssetRequestFormatIsNil)
	}

	err = b.CurrencyPairs.Store(asset.Spot, &currency.PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		Enabled:       []currency.Pair{pair},
		Available:     []currency.Pair{pair},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	// The one hour interval is not supported by the exchange. This scenario
	// demonstrates the conversion from the supported 1 minute candles into
	// one hour candles
	r, err := b.GetKlineExtendedRequest(pair, asset.Spot, kline.OneHour, start, end)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r.Exchange != "klineTest" {
		t.Fatalf("received: '%v' but expected: '%v'", r.Exchange, "klineTest")
	}

	if !r.Pair.Equal(pair) {
		t.Fatalf("received: '%v' but expected: '%v'", r.Pair, pair)
	}

	if r.Asset != asset.Spot {
		t.Fatalf("received: '%v' but expected: '%v'", r.Asset, asset.Spot)
	}

	if r.ExchangeInterval != kline.OneMin {
		t.Fatalf("received: '%v' but expected: '%v'", r.ExchangeInterval, kline.OneMin)
	}

	if r.ClientRequired != kline.OneHour {
		t.Fatalf("received: '%v' but expected: '%v'", r.ClientRequired, kline.OneHour)
	}

	if r.Request.Start != start {
		t.Fatalf("received: '%v' but expected: '%v'", r.Request.Start, start)
	}

	if r.Request.End != end {
		t.Fatalf("received: '%v' but expected: '%v'", r.Request.End, end)
	}

	if r.RequestFormatted.String() != "BTCUSDT" {
		t.Fatalf("received: '%v' but expected: '%v'", r.RequestFormatted.String(), "BTCUSDT")
	}

	if len(r.RangeHolder.Ranges) != 15 { // 15 request at max 100 candles == 1440 1 min candles.
		t.Fatalf("received: '%v' but expected: '%v'", len(r.RangeHolder.Ranges), 15)
	}
}

func TestEnsureOnePairEnabled(t *testing.T) {
	t.Parallel()
	b := Base{Name: "test"}
	err := b.EnsureOnePairEnabled()
	if !errors.Is(err, currency.ErrCurrencyPairsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, currency.ErrCurrencyPairsEmpty)
	}
	b.CurrencyPairs = currency.PairsManager{
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Futures: {},
			asset.Spot: {
				AssetEnabled: convert.BoolPtr(true),
				Available: []currency.Pair{
					currency.NewPair(currency.BTC, currency.USDT),
				},
			},
		},
	}
	err = b.EnsureOnePairEnabled()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(b.CurrencyPairs.Pairs[asset.Spot].Enabled) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(b.CurrencyPairs.Pairs[asset.Spot].Enabled), 1)
	}

	err = b.EnsureOnePairEnabled()
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(b.CurrencyPairs.Pairs[asset.Spot].Enabled) != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", len(b.CurrencyPairs.Pairs[asset.Spot].Enabled), 1)
	}
}
