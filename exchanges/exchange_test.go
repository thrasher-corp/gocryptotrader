package exchange

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
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
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
)

const (
	defaultTestExchange     = "Bitfinex"
	defaultTestCurrencyPair = "BTC-USD"
)

func TestMain(m *testing.M) {
	c := log.GenDefaultSettings()
	log.RWM.Lock()
	log.GlobalLogConfig = &c
	log.RWM.Unlock()
	log.SetupGlobalLogger()
	os.Exit(m.Run())
}

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
		t.Errorf("vals didnt match. expecting: %s, got: %s\n", "http://google.com/", val)
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
	b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "http://test1.com/",
		EdgeCase2: "http://test2.com/",
	})
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
		t.Error("couldnt get changed val")
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
		URL(15): "http://test2.com.au/",
	})
	if err == nil {
		t.Error("expecting an error due to invalid url key")
	}
	err = b.API.Endpoints.SetDefaultEndpoints(map[URL]string{
		EdgeCase1: "",
	})
	if err != nil {
		t.Errorf("expecting a warning due due to invalid url val but got an error: %v", err)
	}
}

func TestHTTPClient(t *testing.T) {
	t.Parallel()
	r := Base{Name: "asdf"}
	r.SetHTTPClientTimeout(time.Second * 5)

	if r.GetHTTPClient().Timeout != time.Second*5 {
		t.Fatalf("TestHTTPClient unexpected value")
	}

	r.Requester = nil
	newClient := new(http.Client)
	newClient.Timeout = time.Second * 10

	r.SetHTTPClient(newClient)
	if r.GetHTTPClient().Timeout != time.Second*10 {
		t.Fatalf("TestHTTPClient unexpected value")
	}

	r.Requester = nil
	if r.GetHTTPClient() == nil {
		t.Fatalf("TestHTTPClient unexpected value")
	}

	b := Base{Name: "RAWR"}
	b.Requester = request.New(b.Name,
		new(http.Client))

	b.SetHTTPClientTimeout(time.Second * 5)
	if b.GetHTTPClient().Timeout != time.Second*5 {
		t.Fatalf("TestHTTPClient unexpected value")
	}

	newClient = new(http.Client)
	newClient.Timeout = time.Second * 10

	b.SetHTTPClient(newClient)
	if b.GetHTTPClient().Timeout != time.Second*10 {
		t.Fatalf("TestHTTPClient unexpected value")
	}

	b.SetHTTPClientUserAgent("epicUserAgent")
	if !strings.Contains(b.GetHTTPClientUserAgent(), "epicUserAgent") {
		t.Error("user agent not set properly")
	}
}

func TestSetClientProxyAddress(t *testing.T) {
	t.Parallel()

	requester := request.New("rawr",
		common.NewHTTPClientWithTimeout(time.Second*15))

	newBase := Base{
		Name:      "rawr",
		Requester: requester}

	newBase.Websocket = stream.New()
	err := newBase.SetClientProxyAddress("")
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

	// Nil out transport
	newBase.Requester.HTTPClient.Transport = nil
	err = newBase.SetClientProxyAddress("http://www.valid.com")
	if err == nil {
		t.Error("error cannot be nil")
	}
}

func TestSetFeatureDefaults(t *testing.T) {
	t.Parallel()

	// Test nil features with basic support capabilities
	b := Base{
		Config: &config.ExchangeConfig{
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

func TestSetAPICredentialDefaults(t *testing.T) {
	t.Parallel()

	b := Base{
		Config: &config.ExchangeConfig{},
	}
	b.API.CredentialsValidator.RequiresKey = true
	b.API.CredentialsValidator.RequiresSecret = true
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.CredentialsValidator.RequiresClientID = true
	b.API.CredentialsValidator.RequiresPEM = true
	b.SetAPICredentialDefaults()

	if !b.Config.API.CredentialsValidator.RequiresKey ||
		!b.Config.API.CredentialsValidator.RequiresSecret ||
		!b.Config.API.CredentialsValidator.RequiresBase64DecodeSecret ||
		!b.Config.API.CredentialsValidator.RequiresClientID ||
		!b.Config.API.CredentialsValidator.RequiresPEM {
		t.Error("incorrect values")
	}
}

func TestSetAutoPairDefaults(t *testing.T) {
	t.Parallel()
	bs := "Bitstamp"
	cfg := &config.Config{Exchanges: []config.ExchangeConfig{
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
		Config: &config.ExchangeConfig{},
	}
	b.SetCurrencyPairFormat()
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
	b.SetCurrencyPairFormat()
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
	b.CurrencyPairs.Store(asset.Spot, currency.PairStore{
		ConfigFormat: &currency.PairFormat{
			Delimiter: "~",
		},
	})
	b.CurrencyPairs.Store(asset.Futures, currency.PairStore{
		ConfigFormat: &currency.PairFormat{
			Delimiter: ":)",
		},
	})
	b.SetCurrencyPairFormat()
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
					RequestFormat: &currency.PairFormat{},
					ConfigFormat:  &currency.PairFormat{},
				},
			},
		},
		Config: &config.ExchangeConfig{
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
	b.SetCurrencyPairFormat()
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

	p := pairs[0].Format(pFmt.Delimiter, pFmt.Uppercase).String()
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
	exchPS, err := b.CurrencyPairs.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	exchPS.RequestFormat.Delimiter = "~"
	exchPS.RequestFormat.Uppercase = false
	exchPS.ConfigFormat.Delimiter = "/"
	exchPS.ConfigFormat.Uppercase = false
	pairs = append(pairs, currency.Pair{Base: currency.XRP, Quote: currency.USD})
	b.Config.CurrencyPairs.StorePairs(asset.Spot, pairs, false)
	b.Config.CurrencyPairs.StorePairs(asset.Spot, pairs, true)
	b.Config.CurrencyPairs.UseGlobalFormat = false
	b.CurrencyPairs.UseGlobalFormat = false

	b.SetConfigPairs()
	// Test four things:
	// 1) XRP-USD is set
	// 2) pair format is set for RequestFormat
	// 3) pair format is set for ConfigFormat
	// 4) Config pair store formats are the same as the exchanges
	pFmt, err = b.GetPairFormat(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}
	pairs, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	p = pairs[2].Format(pFmt.Delimiter, pFmt.Uppercase).String()
	if p != "xrp/usd" {
		t.Error("incorrect value, expected xrp/usd")
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
		t.Error("incorrect value, expected xrp~usd")
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

// TestGetAuthenticatedAPISupport logic test
func TestGetAuthenticatedAPISupport(t *testing.T) {
	t.Parallel()

	base := Base{
		API: API{
			AuthenticatedSupport:          true,
			AuthenticatedWebsocketSupport: false,
		},
	}

	if !base.GetAuthenticatedAPISupport(RestAuthentication) {
		t.Fatal("Expected RestAuthentication to return true")
	}
	if base.GetAuthenticatedAPISupport(WebsocketAuthentication) {
		t.Fatal("Expected WebsocketAuthentication to return false")
	}
	base.API.AuthenticatedWebsocketSupport = true
	if !base.GetAuthenticatedAPISupport(WebsocketAuthentication) {
		t.Fatal("Expected WebsocketAuthentication to return true")
	}
	if base.GetAuthenticatedAPISupport(2) {
		t.Fatal("Expected default case of 'false' to be returned")
	}
}

func TestGetName(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	name := b.GetName()
	if name != "TESTNAME" {
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
	b.CurrencyPairs.Store(asset.Spot, currency.PairStore{
		ConfigFormat: &pFmt,
		RequestFormat: &currency.PairFormat{
			Delimiter: "/",
			Uppercase: true,
		},
	})
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

	b.CurrencyPairs.StorePairs(asset.Spot, defaultPairs, true)
	b.CurrencyPairs.StorePairs(asset.Spot, defaultPairs, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, true)
	b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, btcusdUnderscore, true)
	b.CurrencyPairs.StorePairs(asset.Spot, btcusdUnderscore, false)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c, err = b.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, true)
	b.CurrencyPairs.StorePairs(asset.Spot, btcdoge, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, btcusd, true)
	b.CurrencyPairs.StorePairs(asset.Spot, btcusd, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, defaultPairs, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, dogePairs, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, btcusdUnderscore, false)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c, err = b.GetAvailablePairs(assetType)
	if err != nil {
		t.Fatal(err)
	}

	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot, dogePairs, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, btcusd, false)
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

	b.CurrencyPairs.StorePairs(asset.Spot, pairs, false)

	defaultpairs, err := currency.NewPairsFromStrings([]string{defaultTestCurrencyPair})
	if err != nil {
		t.Fatal(err)
	}

	b.CurrencyPairs.StorePairs(asset.Spot, defaultpairs, true)

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
	expected := "btc~usd^ltc~btc"
	if actual != expected {
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

// TestSetAPIKeys logic test
func TestSetAPIKeys(t *testing.T) {
	t.Parallel()

	b := Base{
		Name:    "TESTNAME",
		Enabled: false,
		API: API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
		},
	}

	b.SetAPIKeys("RocketMan", "Digereedoo", "007")
	if b.API.Credentials.Key != "RocketMan" && b.API.Credentials.Secret != "Digereedoo" && b.API.Credentials.ClientID != "007" {
		t.Error("invalid API credentials")
	}

	// Invalid secret
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.AuthenticatedSupport = true
	b.SetAPIKeys("RocketMan", "%%", "007")
	if b.API.AuthenticatedSupport || b.API.AuthenticatedWebsocketSupport {
		t.Error("invalid secret should disable authenticated API support")
	}

	// valid secret
	b.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	b.API.AuthenticatedSupport = true
	b.SetAPIKeys("RocketMan", "aGVsbG8gd29ybGQ=", "007")
	if !b.API.AuthenticatedSupport && b.API.Credentials.Secret != "hello world" {
		t.Error("invalid secret should disable authenticated API support")
	}
}

func TestSetupDefaults(t *testing.T) {
	t.Parallel()

	var b Base
	cfg := config.ExchangeConfig{
		HTTPTimeout: time.Duration(-1),
		API: config.APIConfig{
			AuthenticatedSupport: true,
		},
	}

	err := b.SetupDefaults(&cfg)
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
	b.CurrencyPairs.Store(asset.Spot,
		currency.PairStore{
			Enabled: currency.Pairs{
				p,
			},
		},
	)
	b.SetupDefaults(&cfg)
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
	b.SetupDefaults(&cfg)
	err = b.Websocket.Setup(&stream.WebsocketSetup{
		Enabled:          false,
		WebsocketTimeout: time.Second * 30,
		Features:         &protocol.Features{},
		DefaultURL:       "ws://something.com",
		RunningURL:       "ws://something.com",
		ExchangeName:     "test",
		Connector:        func() error { return nil },
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

func TestAllowAuthenticatedRequest(t *testing.T) {
	t.Parallel()

	b := Base{
		SkipAuthCheck: true,
	}

	// Test SkipAuthCheck
	if r := b.AllowAuthenticatedRequest(); !r {
		t.Error("skip auth check should allow authenticated requests")
	}

	// Test credentials failure
	b.SkipAuthCheck = false
	b.API.CredentialsValidator.RequiresKey = true
	if r := b.AllowAuthenticatedRequest(); r {
		t.Error("should fail with an empty key")
	}

	// Test bot usage with authenticated API support disabled, but with
	// valid credentials
	b.LoadedByConfig = true
	b.API.Credentials.Key = "k3y"
	if r := b.AllowAuthenticatedRequest(); r {
		t.Error("should fail when authenticated support is disabled")
	}

	// Test enabled authenticated API support and loaded by config
	// but invalid credentials
	b.API.AuthenticatedSupport = true
	b.API.Credentials.Key = ""
	if r := b.AllowAuthenticatedRequest(); r {
		t.Error("should fail with invalid credentials")
	}

	// Finally a valid one
	b.API.Credentials.Key = "k3y"
	if r := b.AllowAuthenticatedRequest(); !r {
		t.Error("show allow an authenticated request")
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()

	var b Base
	type tester struct {
		Key                        string
		Secret                     string
		ClientID                   string
		PEMKey                     string
		RequiresPEM                bool
		RequiresKey                bool
		RequiresSecret             bool
		RequiresClientID           bool
		RequiresBase64DecodeSecret bool
		Expected                   bool
		Result                     bool
	}

	tests := []tester{
		// test key
		{RequiresKey: true},
		{RequiresKey: true, Key: "k3y", Expected: true},
		// test secret
		{RequiresSecret: true},
		{RequiresSecret: true, Secret: "s3cr3t", Expected: true},
		// test pem
		{RequiresPEM: true},
		{RequiresPEM: true, PEMKey: "p3mK3y", Expected: true},
		// test clientID
		{RequiresClientID: true},
		{RequiresClientID: true, ClientID: "cli3nt1D", Expected: true},
		// test requires base64 decode secret
		{RequiresBase64DecodeSecret: true, RequiresSecret: true},
		{RequiresBase64DecodeSecret: true, Secret: "%%", Expected: false},
		{RequiresBase64DecodeSecret: true, Secret: "aGVsbG8gd29ybGQ=", Expected: true},
	}

	for x := range tests {
		setupBase := func(b *Base, tData tester) {
			b.API.Credentials.Key = tData.Key
			b.API.Credentials.Secret = tData.Secret
			b.API.Credentials.ClientID = tData.ClientID
			b.API.Credentials.PEMKey = tData.PEMKey
			b.API.CredentialsValidator.RequiresKey = tData.RequiresKey
			b.API.CredentialsValidator.RequiresSecret = tData.RequiresSecret
			b.API.CredentialsValidator.RequiresPEM = tData.RequiresPEM
			b.API.CredentialsValidator.RequiresClientID = tData.RequiresClientID
			b.API.CredentialsValidator.RequiresBase64DecodeSecret = tData.RequiresBase64DecodeSecret
		}

		setupBase(&b, tests[x])
		if r := b.ValidateAPICredentials(); r != tests[x].Expected {
			t.Errorf("Test %d: expected: %v: got %v", x, tests[x].Expected, r)
		}
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
		Config: &config.ExchangeConfig{
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
		Exchanges: []config.ExchangeConfig{
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
		t.Errorf("Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test empty pair
	p, err := currency.NewPairDelimiter(defaultTestCurrencyPair, "-")
	if err != nil {
		t.Fatal(err)
	}
	pairs := currency.Pairs{
		currency.Pair{},
		p,
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, true, true)
	if err != nil {
		t.Errorf("Forced Exchange UpdatePairs() error: %s", err)
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, false, true)
	if err != nil {
		t.Errorf("Forced Exchange UpdatePairs() error: %s", err)
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
		Enabled:          true,
		WebsocketTimeout: time.Second * 30,
		Features:         &protocol.Features{},
		DefaultURL:       "ws://something.com",
		RunningURL:       "ws://something.com",
		ExchangeName:     "test",
		Connector:        func() error { return nil },
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
	_, err := b.GetPairAssetType(p)
	if err == nil {
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
		Config: &config.ExchangeConfig{},
	}
	b.SetCurrencyPairFormat()
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
		Config: &config.ExchangeConfig{Name: "kitties"},
	}

	err := b.StoreAssetPairFormat(asset.Item(""), currency.PairStore{})
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
	if err != nil {
		t.Error(err)
	}

	err = b.StoreAssetPairFormat(asset.Futures, currency.PairStore{
		RequestFormat: &currency.PairFormat{Uppercase: true},
		ConfigFormat:  &currency.PairFormat{Uppercase: true}})
	if err != nil {
		t.Error(err)
	}
}

func TestSetGlobalPairsManager(t *testing.T) {
	b := Base{
		Config: &config.ExchangeConfig{Name: "kitties"},
	}

	err := b.SetGlobalPairsManager(nil, nil, "")
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true}, nil, "")
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true})
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true}, "")
	if err == nil {
		t.Error("error cannot be nil")
	}

	err = b.SetGlobalPairsManager(&currency.PairFormat{Uppercase: true},
		&currency.PairFormat{Uppercase: true}, asset.Spot, asset.Binary)
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
					Intervals: map[string]bool{
						kline.OneMin.Word(): true,
					},
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
	b.checkAndInitRequester()
	err := b.EnableRateLimiter()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = b.DisableRateLimiter()
	if err != nil {
		t.Fatal(err)
	}

	err = b.DisableRateLimiter()
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	err = b.EnableRateLimiter()
	if err != nil {
		t.Fatal(err)
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
	if err := b.AuthenticateWebsocket(); err == nil {
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
		Config: &config.ExchangeConfig{
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
		Config: &config.ExchangeConfig{
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
	if RestSpot.String() != "RestSpotURL" {
		t.Errorf("invalid string conversion")
	}
	if RestSpotSupplementary.String() != "RestSpotSupplementaryURL" {
		t.Errorf("invalid string conversion")
	}
	if RestUSDTMargined.String() != "RestUSDTMarginedFuturesURL" {
		t.Errorf("invalid string conversion")
	}
	if RestCoinMargined.String() != "RestCoinMarginedFuturesURL" {
		t.Errorf("invalid string conversion")
	}
	if RestFutures.String() != "RestFuturesURL" {
		t.Errorf("invalid string conversion")
	}
	if RestSandbox.String() != "RestSandboxURL" {
		t.Errorf("invalid string conversion")
	}
	if RestSwap.String() != "RestSwapURL" {
		t.Errorf("invalid string conversion")
	}
	if WebsocketSpot.String() != "WebsocketSpotURL" {
		t.Errorf("invalid string conversion")
	}
	if WebsocketSpotSupplementary.String() != "WebsocketSpotSupplementaryURL" {
		t.Errorf("invalid string conversion")
	}
	if ChainAnalysis.String() != "ChainAnalysisURL" {
		t.Errorf("invalid string conversion")
	}
	if EdgeCase1.String() != "EdgeCase1URL" {
		t.Errorf("invalid string conversion")
	}
	if EdgeCase2.String() != "EdgeCase2URL" {
		t.Errorf("invalid string conversion")
	}
	if EdgeCase3.String() != "EdgeCase3URL" {
		t.Errorf("invalid string conversion")
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
	b.Config = &config.ExchangeConfig{}
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
		},
	})
	if err != nil {
		log.Errorln(log.ExchangeSys, err)
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
