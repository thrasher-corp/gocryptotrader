package exchange

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/request"
)

const (
	defaultTestExchange     = "ANX"
	defaultTestCurrencyPair = "BTC-USD"
)

func TestSupportsRESTTickerBatchUpdates(t *testing.T) {
	b := Base{
		Name: "RAWR",
		Features: Features{
			Supports: FeaturesSupported{
				REST: true,
				RESTCapabilities: ProtocolFeatures{
					TickerBatching: true,
				},
			},
		},
	}

	if !b.SupportsRESTTickerBatchUpdates() {
		t.Fatal("Test failed. TestSupportsRESTTickerBatchUpdates returned false")
	}
}

func TestHTTPClient(t *testing.T) {
	r := Base{Name: "asdf"}
	r.SetHTTPClientTimeout(time.Second * 5)

	if r.GetHTTPClient().Timeout != time.Second*5 {
		t.Fatalf("Test failed. TestHTTPClient unexpected value")
	}

	r.Requester = nil
	newClient := new(http.Client)
	newClient.Timeout = time.Second * 10

	r.SetHTTPClient(newClient)
	if r.GetHTTPClient().Timeout != time.Second*10 {
		t.Fatalf("Test failed. TestHTTPClient unexpected value")
	}

	r.Requester = nil
	if r.GetHTTPClient() == nil {
		t.Fatalf("Test failed. TestHTTPClient unexpected value")
	}

	b := Base{Name: "RAWR"}
	b.Requester = request.New(b.Name,
		request.NewRateLimit(time.Second, 1),
		request.NewRateLimit(time.Second, 1),
		new(http.Client))

	b.SetHTTPClientTimeout(time.Second * 5)
	if b.GetHTTPClient().Timeout != time.Second*5 {
		t.Fatalf("Test failed. TestHTTPClient unexpected value")
	}

	newClient = new(http.Client)
	newClient.Timeout = time.Second * 10

	b.SetHTTPClient(newClient)
	if b.GetHTTPClient().Timeout != time.Second*10 {
		t.Fatalf("Test failed. TestHTTPClient unexpected value")
	}
}

func TestSetClientProxyAddress(t *testing.T) {
	requester := request.New("testicles",
		&request.RateLimit{},
		&request.RateLimit{},
		&http.Client{})

	newBase := Base{Name: "Testicles", Requester: requester}

	newBase.WebsocketInit()

	err := newBase.SetClientProxyAddress(":invalid")
	if err == nil {
		t.Error("Test failed. SetClientProxyAddress parsed invalid URL")
	}

	if newBase.Websocket.GetProxyAddress() != "" {
		t.Error("Test failed. SetClientProxyAddress error", err)
	}

	err = newBase.SetClientProxyAddress("www.valid.com")
	if err != nil {
		t.Error("Test failed. SetClientProxyAddress error", err)
	}

	if newBase.Websocket.GetProxyAddress() != "www.valid.com" {
		t.Error("Test failed. SetClientProxyAddress error", err)
	}
}

func TestSetAutoPairDefaults(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults failed to load config file. Error: %s", err)
	}

	exch, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if !exch.Features.Supports.RESTCapabilities.AutoPairUpdates {
		t.Fatalf("Test failed. TestSetAutoPairDefaults Incorrect value")
	}

	if exch.CurrencyPairs.LastUpdated != 0 {
		t.Fatalf("Test failed. TestSetAutoPairDefaults Incorrect value")
	}

	exch.Features.Supports.RESTCapabilities.AutoPairUpdates = false
	cfg.UpdateExchangeConfig(exch)

	exch, err = cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if exch.Features.Supports.RESTCapabilities.AutoPairUpdates {
		t.Fatal("Test failed. TestSetAutoPairDefaults Incorrect value")
	}
}

func TestSupportsAutoPairUpdates(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	if b.SupportsAutoPairUpdates() {
		t.Fatal("Test failed. TestSupportsAutoPairUpdates Incorrect value")
	}
}

func TestGetLastPairsUpdateTime(t *testing.T) {
	testTime := time.Now().Unix()
	var b Base
	b.CurrencyPairs.LastUpdated = testTime

	if b.GetLastPairsUpdateTime() != testTime {
		t.Fatal("Test failed. TestGetLastPairsUpdateTim Incorrect value")
	}
}

func TestSetAssetTypes(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes failed to load config file. Error: %s", err)
	}

	b := Base{
		Name: "TESTNAME",
	}

	b.Name = "ANX"
	b.CurrencyPairs.AssetTypes = assets.AssetTypes{assets.AssetTypeSpot}
	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes load config failed. Error %s", err)
	}

	exch.CurrencyPairs.AssetTypes = assets.New("")
	err = cfg.UpdateExchangeConfig(exch)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes update config failed. Error %s", err)
	}

	exch, err = cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes load config failed. Error %s", err)
	}
	b.Config = exch

	if exch.CurrencyPairs.AssetTypes.JoinToString(",") != "" {
		t.Fatal("Test failed. TestSetAssetTypes assetTypes != ''")
	}

	b.SetAssetTypes()
	if !common.StringDataCompare(b.CurrencyPairs.AssetTypes.Strings(), assets.AssetTypeSpot.String()) {
		t.Fatal("Test failed. TestSetAssetTypes assetTypes is not set")
	}
}

func TestGetAssetTypes(t *testing.T) {
	testExchange := Base{
		CurrencyPairs: currency.PairsManager{
			AssetTypes: assets.AssetTypes{
				assets.AssetTypeSpot,
				assets.AssetTypeBinary,
				assets.AssetTypeFutures,
			},
		},
	}

	aT := testExchange.GetAssetTypes()
	if len(aT) != 3 {
		t.Error("Test failed. TestGetAssetTypes failed")
	}
}

func TestSetCurrencyPairFormat(t *testing.T) {
	t.Skip()
	// TO-DO
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat failed to load config file. Error: %s", err)
	}

	b := Base{
		Name: "TESTNAME",
	}

	b.Name = "ANX"
	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat load config failed. Error %s", err)
	}
	b.Config = exch
	b.SetCurrencyPairFormat()
	if exch.CurrencyPairs.ConfigFormat.Delimiter != "-" && !exch.CurrencyPairs.ConfigFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat exch values are not nil")
	}

	exch.CurrencyPairs.ConfigFormat = nil
	exch.CurrencyPairs.RequestFormat = nil
	err = cfg.UpdateExchangeConfig(exch)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat update config failed. Error %s", err)
	}

	exch, err = cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat load config failed. Error %s", err)
	}

	if exch.CurrencyPairs.ConfigFormat != nil && exch.CurrencyPairs.RequestFormat != nil {
		t.Fatal("Test failed. TestSetCurrencyPairFormat exch values are not nil")
	}

	b.SetCurrencyPairFormat()
	if exch.CurrencyPairs.ConfigFormat.Delimiter != "-" && !exch.CurrencyPairs.ConfigFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat exch values are not nil")
	}
}

func TestGetAuthenticatedAPISupport(t *testing.T) {
	var base Base
	if base.GetAuthenticatedAPISupport() {
		t.Fatal("Test failed. TestGetAuthenticatedAPISupport returned true when it should of been false.")
	}
}

func TestGetName(t *testing.T) {
	GetName := Base{
		Name: "TESTNAME",
	}

	name := GetName.GetName()
	if name != "TESTNAME" {
		t.Error("Test Failed - Exchange GetName() returned incorrect name")
	}
}

func TestGetEnabledPairs(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTC-USD"}), true)
	format := currency.PairFormat{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	assetType := assets.AssetTypeSpot
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &format
	b.CurrencyPairs.ConfigFormat = &format

	c := b.GetEnabledPairs(assetType)
	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = "~"
	b.CurrencyPairs.RequestFormat = &format
	c = b.GetEnabledPairs(assetType)
	if c[0].String() != "BTC~USD" {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = ""
	b.CurrencyPairs.ConfigFormat = &format
	c = b.GetEnabledPairs(assetType)
	if c[0].String() != "BTCUSD" {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), true)
	format.Index = currency.BTC.String()
	b.CurrencyPairs.ConfigFormat = &format
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTC_USD"}), true)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), true)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Index = currency.BTC.String()
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTCUSD"}), true)
	b.CurrencyPairs.ConfigFormat.Index = ""
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}
}

func TestGetAvailablePairs(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{defaultTestCurrencyPair}), false)
	format := currency.PairFormat{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	assetType := assets.AssetTypeSpot
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &format
	b.CurrencyPairs.ConfigFormat = &format

	c := b.GetAvailablePairs(assetType)
	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = "~"
	b.CurrencyPairs.RequestFormat = &format
	c = b.GetAvailablePairs(assetType)
	if c[0].String() != "BTC~USD" {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	format.Delimiter = ""
	b.CurrencyPairs.ConfigFormat = &format
	c = b.GetAvailablePairs(assetType)
	if c[0].String() != "BTCUSD" {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), false)
	format.Index = currency.BTC.String()
	b.CurrencyPairs.ConfigFormat = &format
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTC_USD"}), false)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), false)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Index = currency.BTC.String()
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{"BTCUSD"}), false)
	b.CurrencyPairs.ConfigFormat.Index = ""
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}
}

func TestSupportsPair(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{
			defaultTestCurrencyPair, "ETH-USD"}), false)
	b.CurrencyPairs.StorePairs(assets.AssetTypeSpot,
		currency.NewPairsFromStrings([]string{defaultTestCurrencyPair}), true)

	format := &currency.PairFormat{
		Delimiter: "-",
		Index:     "",
	}

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = format
	b.CurrencyPairs.ConfigFormat = format
	assetType := assets.AssetTypeSpot

	if !b.SupportsPair(currency.NewPair(currency.BTC, currency.USD), true, assetType) {
		t.Error("Test Failed - Exchange SupportsPair() incorrect value")
	}

	if !b.SupportsPair(currency.NewPair(currency.ETH, currency.USD), false, assetType) {
		t.Error("Test Failed - Exchange SupportsPair() incorrect value")
	}

	if b.SupportsPair(currency.NewPairFromStrings("ASD", "ASDF"), true, assetType) {
		t.Error("Test Failed - Exchange SupportsPair() incorrect value")
	}
}

func TestFormatExchangeCurrencies(t *testing.T) {
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

	var pairs = []currency.Pair{
		currency.NewPairDelimiter("BTC_USD", "_"),
		currency.NewPairDelimiter("LTC_BTC", "_"),
	}

	actual, err := e.FormatExchangeCurrencies(pairs, assets.AssetTypeSpot)
	if err != nil {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrencies error %s", err)
	}
	expected := "btc~usd^ltc~btc"

	if actual != expected {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrencies %s != %s",
			actual, expected)
	}
}

func TestFormatExchangeCurrency(t *testing.T) {
	var b Base
	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = &currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	expected := defaultTestCurrencyPair
	actual := b.FormatExchangeCurrency(p, assets.AssetTypeSpot)

	if actual.String() != expected {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrency %s != %s",
			actual, expected)
	}
}

func TestSetEnabled(t *testing.T) {
	SetEnabled := Base{
		Name:    "TESTNAME",
		Enabled: false,
	}

	SetEnabled.SetEnabled(true)
	if !SetEnabled.Enabled {
		t.Error("Test Failed - Exchange SetEnabled(true) did not set boolean")
	}
}

func TestIsEnabled(t *testing.T) {
	IsEnabled := Base{
		Name:    "TESTNAME",
		Enabled: false,
	}

	if IsEnabled.IsEnabled() {
		t.Error("Test Failed - Exchange IsEnabled() did not return correct boolean")
	}
}

func TestSetAPIKeys(t *testing.T) {
	SetAPIKeys := Base{
		Name:    "TESTNAME",
		Enabled: false,
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007")
	if SetAPIKeys.API.Credentials.Key != "RocketMan" && SetAPIKeys.API.Credentials.Secret != "Digereedoo" && SetAPIKeys.API.Credentials.ClientID != "007" {
		t.Error("Test Failed - SetAPIKeys() unable to set API credentials")
	}

	SetAPIKeys.API.CredentialsValidator.RequiresBase64DecodeSecret = true
	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007")
}

func TestSetPairs(t *testing.T) {
	t.Skip()
	// TO-DO
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestSetPairs failed to load config")
	}

	anxCfg, err := cfg.GetExchangeConfig(defaultTestExchange)
	if err != nil {
		t.Fatal("Test failed. TestSetPairs failed to load config")
	}

	newPair := currency.NewPairDelimiter("ETH_USDT", "_")
	assetType := assets.AssetTypeSpot

	var UAC Base
	UAC.Name = "ANX"
	UAC.Config = anxCfg
	err = UAC.SetupDefaults(anxCfg)
	if err != nil {
		t.Fatalf("Test failed. TestSetPairs unable to set defaults: %s", err)
	}

	err = UAC.SetPairs([]currency.Pair{newPair}, assets.AssetTypeSpot, true)
	if err != nil {
		t.Fatalf("Test failed. TestSetPairs failed to set currencies: %s", err)
	}

	if !UAC.GetEnabledPairs(assetType).Contains(newPair, true) {
		t.Fatal("Test failed. TestSetPairs failed to set currencies")
	}

	UAC.SetPairs([]currency.Pair{newPair}, assets.AssetTypeSpot, false)
	if !UAC.GetAvailablePairs(assetType).Contains(newPair, true) {
		t.Fatal("Test failed. TestSetPairs failed to set currencies")
	}

	err = UAC.SetPairs(nil, assets.AssetTypeSpot, false)
	if err == nil {
		t.Fatal("Test failed. TestSetPairs should return an error when attempting to set an empty pairs array")
	}
}

func TestUpdatePairs(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestUpdatePairs failed to load config")
	}

	anxCfg, err := cfg.GetExchangeConfig("ANX")
	if err != nil {
		t.Fatal("Test failed. TestUpdatePairs failed to load config")
	}

	UAC := Base{Name: "ANX"}
	UAC.Config = anxCfg
	exchangeProducts := currency.NewPairsFromStrings([]string{"ltc", "btc", "usd", "aud", ""})
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, true, false)
	if err != nil {
		t.Errorf("Test Failed - TestUpdatePairs error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, true, false)
	if err != nil {
		t.Errorf("Test Failed - TestUpdatePairs error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = currency.NewPairsFromStrings([]string{"btc"})
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, true, true)
	if err != nil {
		t.Errorf("Test Failed - TestUpdatePairs error: %s", err)
	}

	// Test updating exchange products
	exchangeProducts = currency.NewPairsFromStrings([]string{"ltc", "btc", "usd", "aud"})
	UAC.Name = "ANX"
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, false, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdatePairs() error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, false, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdatePairs() error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = currency.NewPairsFromStrings([]string{"btc"})
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, false, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test update currency pairs with btc excluded
	exchangeProducts = currency.NewPairsFromStrings([]string{"ltc", "eth"})
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, false, false)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test that empty exchange products should return an error
	exchangeProducts = nil
	err = UAC.UpdatePairs(exchangeProducts, assets.AssetTypeSpot, false, false)
	if err == nil {
		t.Errorf("Test failed - empty available pairs should return an error")
	}
}

func TestSetAPIURL(t *testing.T) {
	testURL := "https://api.something.com"
	testURLSecondary := "https://api.somethingelse.com"
	testURLDefault := "https://api.defaultsomething.com"
	testURLSecondaryDefault := "https://api.defaultsomethingelse.com"

	tester := Base{Name: "test"}
	tester.Config = new(config.ExchangeConfig)

	err := tester.SetAPIURL()
	if err == nil {
		t.Error("test failed - setting zero value config")
	}

	tester.Config.API.Endpoints.URL = testURL
	tester.Config.API.Endpoints.URLSecondary = testURLSecondary

	tester.API.Endpoints.URLDefault = testURLDefault
	tester.API.Endpoints.URLSecondaryDefault = testURLSecondaryDefault

	err = tester.SetAPIURL()
	if err != nil {
		t.Error("test failed", err)
	}

	if tester.GetAPIURL() != testURL {
		t.Error("test failed - incorrect return URL")
	}

	if tester.GetSecondaryAPIURL() != testURLSecondary {
		t.Error("test failed - incorrect return URL")
	}

	if tester.GetAPIURLDefault() != testURLDefault {
		t.Error("test failed - incorrect return URL")
	}

	if tester.GetAPIURLSecondaryDefault() != testURLSecondaryDefault {
		t.Error("test failed - incorrect return URL")
	}
}

func BenchmarkSetAPIURL(b *testing.B) {
	tester := Base{Name: "test"}

	test := config.ExchangeConfig{}

	test.API.Endpoints.URL = "https://api.something.com"
	test.API.Endpoints.URLSecondary = "https://api.somethingelse.com"

	tester.API.Endpoints.URLDefault = "https://api.defaultsomething.com"
	tester.API.Endpoints.URLDefault = "https://api.defaultsomethingelse.com"

	tester.Config = &test

	for i := 0; i < b.N; i++ {
		err := tester.SetAPIURL()
		if err != nil {
			b.Errorf("Benchmark failed %v", err)
		}
	}
}

func TestSupportsWithdrawPermissions(t *testing.T) {
	UAC := Base{Name: defaultTestExchange}
	UAC.APIWithdrawPermissions = AutoWithdrawCrypto | AutoWithdrawCryptoWithAPIPermission
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
	UAC := Base{Name: "ANX"}
	UAC.APIWithdrawPermissions = AutoWithdrawCrypto |
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

	UAC.APIWithdrawPermissions = NoAPIWithdrawalMethods
	withdrawPermissions = UAC.FormatWithdrawPermissions()

	if withdrawPermissions != NoAPIWithdrawalMethodsText {
		t.Errorf("Expected: %s, Received: %s", NoAPIWithdrawalMethodsText, withdrawPermissions)
	}
}

func TestOrderSides(t *testing.T) {
	var os = BuyOrderSide
	if os.ToString() != "BUY" {
		t.Errorf("test failed - unexpected string %s", os.ToString())
	}

	if os.ToLower() != "buy" {
		t.Errorf("test failed - unexpected string %s", os.ToString())
	}
}

func TestOrderTypes(t *testing.T) {
	var ot OrderType = "Mo'Money"

	if ot.ToString() != "Mo'Money" {
		t.Errorf("test failed - unexpected string %s", ot.ToString())
	}

	if ot.ToLower() != "mo'money" {
		t.Errorf("test failed - unexpected string %s", ot.ToString())
	}
}

func TestFilterOrdersByType(t *testing.T) {
	var orders = []OrderDetail{
		{
			OrderType: ImmediateOrCancelOrderType,
		},
		{
			OrderType: LimitOrderType,
		},
	}

	FilterOrdersByType(&orders, AnyOrderType)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	FilterOrdersByType(&orders, LimitOrderType)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	FilterOrdersByType(&orders, StopOrderType)
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

func TestFilterOrdersBySide(t *testing.T) {
	var orders = []OrderDetail{
		{
			OrderSide: BuyOrderSide,
		},
		{
			OrderSide: SellOrderSide,
		},
		{},
	}

	FilterOrdersBySide(&orders, AnyOrderSide)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersBySide(&orders, BuyOrderSide)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	FilterOrdersBySide(&orders, SellOrderSide)
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

func TestFilterOrdersByTickRange(t *testing.T) {
	var orders = []OrderDetail{
		{
			OrderDate: time.Unix(100, 0),
		},
		{
			OrderDate: time.Unix(110, 0),
		},
		{
			OrderDate: time.Unix(111, 0),
		},
	}

	FilterOrdersByTickRange(&orders, time.Unix(0, 0), time.Unix(0, 0))
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersByTickRange(&orders, time.Unix(100, 0), time.Unix(111, 0))
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	FilterOrdersByTickRange(&orders, time.Unix(101, 0), time.Unix(111, 0))
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	FilterOrdersByTickRange(&orders, time.Unix(200, 0), time.Unix(300, 0))
	if len(orders) != 0 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 0, len(orders))
	}
}

func TestFilterOrdersByCurrencies(t *testing.T) {
	var orders = []OrderDetail{
		{
			CurrencyPair: currency.NewPair(currency.BTC, currency.USD),
		},
		{
			CurrencyPair: currency.NewPair(currency.LTC, currency.EUR),
		},
		{
			CurrencyPair: currency.NewPair(currency.DOGE, currency.RUB),
		},
	}

	currencies := []currency.Pair{currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.EUR),
		currency.NewPair(currency.DOGE, currency.RUB)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 3 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 3, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.LTC, currency.EUR)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 2 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 2, len(orders))
	}

	currencies = []currency.Pair{currency.NewPair(currency.BTC, currency.USD)}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}

	currencies = []currency.Pair{}
	FilterOrdersByCurrencies(&orders, currencies)
	if len(orders) != 1 {
		t.Errorf("Orders failed to be filtered. Expected %v, received %v", 1, len(orders))
	}
}

func TestSortOrdersByPrice(t *testing.T) {
	orders := []OrderDetail{
		{
			Price: 100,
		}, {
			Price: 0,
		}, {
			Price: 50,
		},
	}

	SortOrdersByPrice(&orders, false)
	if orders[0].Price != 0 {
		t.Errorf("Test failed. Expected: '%v', received: '%v'", 0, orders[0].Price)
	}

	SortOrdersByPrice(&orders, true)
	if orders[0].Price != 100 {
		t.Errorf("Test failed. Expected: '%v', received: '%v'", 100, orders[0].Price)
	}
}

func TestSortOrdersByDate(t *testing.T) {
	orders := []OrderDetail{
		{
			OrderDate: time.Unix(0, 0),
		}, {
			OrderDate: time.Unix(1, 0),
		}, {
			OrderDate: time.Unix(2, 0),
		},
	}

	SortOrdersByDate(&orders, false)
	if orders[0].OrderDate.Unix() != time.Unix(0, 0).Unix() {
		t.Errorf("Test failed. Expected: '%v', received: '%v'",
			time.Unix(0, 0).Unix(),
			orders[0].OrderDate.Unix())
	}

	SortOrdersByDate(&orders, true)
	if orders[0].OrderDate.Unix() != time.Unix(2, 0).Unix() {
		t.Errorf("Test failed. Expected: '%v', received: '%v'",
			time.Unix(2, 0).Unix(),
			orders[0].OrderDate.Unix())
	}
}

func TestSortOrdersByCurrency(t *testing.T) {
	orders := []OrderDetail{
		{
			CurrencyPair: currency.NewPairWithDelimiter(currency.BTC.String(),
				currency.USD.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.DOGE.String(),
				currency.USD.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.BTC.String(),
				currency.RUB.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.LTC.String(),
				currency.EUR.String(),
				"-"),
		}, {
			CurrencyPair: currency.NewPairWithDelimiter(currency.LTC.String(),
				currency.AUD.String(),
				"-"),
		},
	}

	SortOrdersByCurrency(&orders, false)
	if orders[0].CurrencyPair.String() != currency.BTC.String()+"-"+currency.RUB.String() {
		t.Errorf("Test failed. Expected: '%v', received: '%v'",
			currency.BTC.String()+"-"+currency.RUB.String(),
			orders[0].CurrencyPair.String())
	}

	SortOrdersByCurrency(&orders, true)
	if orders[0].CurrencyPair.String() != currency.LTC.String()+"-"+currency.EUR.String() {
		t.Errorf("Test failed. Expected: '%v', received: '%v'",
			currency.LTC.String()+"-"+currency.EUR.String(),
			orders[0].CurrencyPair.String())
	}
}

func TestSortOrdersByOrderSide(t *testing.T) {
	orders := []OrderDetail{
		{
			OrderSide: BuyOrderSide,
		}, {
			OrderSide: SellOrderSide,
		}, {
			OrderSide: SellOrderSide,
		}, {
			OrderSide: BuyOrderSide,
		},
	}

	SortOrdersBySide(&orders, false)
	if !strings.EqualFold(orders[0].OrderSide.ToString(), BuyOrderSide.ToString()) {
		t.Errorf("Test failed. Expected: '%v', received: '%v'",
			BuyOrderSide,
			orders[0].OrderSide)
	}

	SortOrdersBySide(&orders, true)
	if !strings.EqualFold(orders[0].OrderSide.ToString(), SellOrderSide.ToString()) {
		t.Errorf("Test failed. Expected: '%v', received: '%v'",
			SellOrderSide,
			orders[0].OrderSide)
	}
}

func TestSortOrdersByOrderType(t *testing.T) {
	orders := []OrderDetail{
		{
			OrderType: MarketOrderType,
		}, {
			OrderType: LimitOrderType,
		}, {
			OrderType: ImmediateOrCancelOrderType,
		}, {
			OrderType: TrailingStopOrderType,
		},
	}

	SortOrdersByType(&orders, false)
	if !strings.EqualFold(orders[0].OrderType.ToString(), ImmediateOrCancelOrderType.ToString()) {
		t.Errorf("Test failed. Expected: '%v', received: '%v'", ImmediateOrCancelOrderType, orders[0].OrderType)
	}

	SortOrdersByType(&orders, true)
	if !strings.EqualFold(orders[0].OrderType.ToString(), TrailingStopOrderType.ToString()) {
		t.Errorf("Test failed. Expected: '%v', received: '%v'", TrailingStopOrderType, orders[0].OrderType)
	}
}
