package exchange

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

const (
	defaultTestExchange     = "ANX"
	defaultTestCurrencyPair = "BTC-USD"
)

func TestSupportsRESTTickerBatchUpdates(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

	b.SetHTTPClientUserAgent("epicUserAgent")
	if !strings.Contains(b.GetHTTPClientUserAgent(), "epicUserAgent") {
		t.Error("user agent not set properly")
	}
}

func TestSetClientProxyAddress(t *testing.T) {
	t.Parallel()

	requester := request.New("rawr",
		&request.RateLimit{},
		&request.RateLimit{},
		&http.Client{})

	newBase := Base{
		Name:      "rawr",
		Requester: requester}

	newBase.Websocket = wshandler.New()
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
	err := cfg.LoadConfig(config.ConfigTestFile, true)
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
		t.Fatal("Test failed. TestGetLastPairsUpdateTim Incorrect value")
	}
}

func TestSetAssetTypes(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile, true)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes failed to load config file. Error: %s", err)
	}

	b := Base{
		Name: "TESTNAME",
	}

	b.Name = "ANX"
	b.CurrencyPairs.AssetTypes = asset.Items{asset.Spot}
	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes load config failed. Error %s", err)
	}

	exch.CurrencyPairs.AssetTypes = asset.New("")
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
	if !common.StringDataCompare(b.CurrencyPairs.AssetTypes.Strings(), asset.Spot.String()) {
		t.Fatal("Test failed. TestSetAssetTypes assetTypes is not set")
	}
}

func TestGetAssetTypes(t *testing.T) {
	t.Parallel()

	testExchange := Base{
		CurrencyPairs: currency.PairsManager{
			AssetTypes: asset.Items{
				asset.Spot,
				asset.Binary,
				asset.Futures,
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
	err := cfg.LoadConfig(config.ConfigTestFile, true)
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
	if exch.CurrencyPairs.Get(asset.Spot).ConfigFormat.Delimiter != "-" && !exch.CurrencyPairs.ConfigFormat.Uppercase {
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
	if exch.CurrencyPairs.Get(asset.Spot).ConfigFormat.Delimiter != "-" && !exch.CurrencyPairs.ConfigFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat exch values are not nil")
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
		t.Fatal("Test failed. Expected RestAuthentication to return true")
	}
	if base.GetAuthenticatedAPISupport(WebsocketAuthentication) {
		t.Fatal("Test failed. Expected WebsocketAuthentication to return false")
	}
	base.API.AuthenticatedWebsocketSupport = true
	if !base.GetAuthenticatedAPISupport(WebsocketAuthentication) {
		t.Fatal("Test failed. Expected WebsocketAuthentication to return true")
	}
	if base.GetAuthenticatedAPISupport(2) {
		t.Fatal("Test failed. Expected default case of 'false' to be returned")
	}
}

func TestGetName(t *testing.T) {
	t.Parallel()

	GetName := Base{
		Name: "TESTNAME",
	}

	name := GetName.GetName()
	if name != "TESTNAME" {
		t.Error("Test Failed - Exchange GetName() returned incorrect name")
	}
}

func TestGetEnabledPairs(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTC-USD"}), true)
	format := currency.PairFormat{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	assetType := asset.Spot
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

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), true)
	format.Index = currency.BTC.String()
	b.CurrencyPairs.ConfigFormat = &format
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTC_USD"}), true)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), true)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Index = currency.BTC.String()
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTCUSD"}), true)
	b.CurrencyPairs.ConfigFormat.Index = ""
	c = b.GetEnabledPairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}
}

func TestGetAvailablePairs(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{defaultTestCurrencyPair}), false)
	format := currency.PairFormat{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	assetType := asset.Spot
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

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), false)
	format.Index = currency.BTC.String()
	b.CurrencyPairs.ConfigFormat = &format
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTC_USD"}), false)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTCDOGE"}), false)
	b.CurrencyPairs.RequestFormat.Delimiter = ""
	b.CurrencyPairs.ConfigFormat.Delimiter = "_"
	b.CurrencyPairs.ConfigFormat.Index = currency.BTC.String()
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{"BTCUSD"}), false)
	b.CurrencyPairs.ConfigFormat.Index = ""
	c = b.GetAvailablePairs(assetType)
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailablePairs() incorrect string")
	}
}

func TestSupportsPair(t *testing.T) {
	t.Parallel()

	b := Base{
		Name: "TESTNAME",
	}

	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{
			defaultTestCurrencyPair, "ETH-USD"}), false)
	b.CurrencyPairs.StorePairs(asset.Spot,
		currency.NewPairsFromStrings([]string{defaultTestCurrencyPair}), true)

	format := &currency.PairFormat{
		Delimiter: "-",
		Index:     "",
	}

	b.CurrencyPairs.UseGlobalFormat = true
	b.CurrencyPairs.RequestFormat = format
	b.CurrencyPairs.ConfigFormat = format
	assetType := asset.Spot

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

	var pairs = []currency.Pair{
		currency.NewPairDelimiter("BTC_USD", "_"),
		currency.NewPairDelimiter("LTC_BTC", "_"),
	}

	actual, err := e.FormatExchangeCurrencies(pairs, asset.Spot)
	if err != nil {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrencies error %s", err)
	}
	expected := "btc~usd^ltc~btc"
	if actual != expected {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrencies %s != %s",
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
	actual := b.FormatExchangeCurrency(p, asset.Spot)

	if actual.String() != expected {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrency %s != %s",
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
		t.Error("Test Failed - Exchange SetEnabled(true) did not set boolean")
	}
}

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	IsEnabled := Base{
		Name:    "TESTNAME",
		Enabled: false,
	}

	if IsEnabled.IsEnabled() {
		t.Error("Test Failed - Exchange IsEnabled() did not return correct boolean")
	}
}

// TestSetAPIKeys logic test
func TestSetAPIKeys(t *testing.T) {
	t.Parallel()

	SetAPIKeys := Base{
		Name:    "TESTNAME",
		Enabled: false,
		API: API{
			AuthenticatedSupport:          false,
			AuthenticatedWebsocketSupport: false,
		},
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007")
	if SetAPIKeys.API.Credentials.Key != "RocketMan" && SetAPIKeys.API.Credentials.Secret != "Digereedoo" && SetAPIKeys.API.Credentials.ClientID != "007" {
		t.Error("Test Failed - SetAPIKeys() unable to set API credentials")
	}
}

func TestAllowAuthenticatedRequest(t *testing.T) {
	t.Parallel()

	b := Base{
		SkipAuthCheck: true,
	}

	if r := b.AllowAuthenticatedRequest(); !r {
		t.Error("skip auth check should allow authenticated requests")
	}

	b.SkipAuthCheck = false
	b.API.CredentialsValidator.RequiresKey = true
	if r := b.AllowAuthenticatedRequest(); r {
		t.Error("should fail with an empty key")
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
				Pairs: map[asset.Item]*currency.PairStore{},
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

	if p := b.GetEnabledPairs(asset.Spot); len(p) != 1 {
		t.Error("pairs shouldn't be nil")
	}
}

func TestUpdatePairs(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile, true)
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
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true, false)
	if err != nil {
		t.Errorf("Test Failed - TestUpdatePairs error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true, false)
	if err != nil {
		t.Errorf("Test Failed - TestUpdatePairs error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = currency.NewPairsFromStrings([]string{"btc"})
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, true, true)
	if err != nil {
		t.Errorf("Test Failed - TestUpdatePairs error: %s", err)
	}

	// Test updating exchange products
	exchangeProducts = currency.NewPairsFromStrings([]string{"ltc", "btc", "usd", "aud"})
	UAC.Name = "ANX"
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdatePairs() error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdatePairs() error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = currency.NewPairsFromStrings([]string{"btc"})
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test update currency pairs with btc excluded
	exchangeProducts = currency.NewPairsFromStrings([]string{"ltc", "eth"})
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdatePairs() error: %s", err)
	}

	// Test that empty exchange products should return an error
	exchangeProducts = nil
	err = UAC.UpdatePairs(exchangeProducts, asset.Spot, false, false)
	if err == nil {
		t.Errorf("Test failed - empty available pairs should return an error")
	}

	// Test empty pair
	p := currency.NewPairDelimiter(defaultTestCurrencyPair, "-")
	pairs := currency.Pairs{
		currency.Pair{},
		p,
	}
	err = UAC.UpdatePairs(pairs, asset.Spot, true, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdatePairs() error: %s", err)
	}
	UAC.CurrencyPairs.UseGlobalFormat = true
	UAC.CurrencyPairs.ConfigFormat = &currency.PairFormat{
		Delimiter: "-",
	}
	if !UAC.GetEnabledPairs(asset.Spot).Contains(p, true) {
		t.Fatal("expected currency pair not found")
	}
}

func TestSetAPIURL(t *testing.T) {
	t.Parallel()

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

	b.Websocket = wshandler.New()
	err := b.Websocket.Setup(&wshandler.WebsocketSetup{Enabled: true})
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

	UAC := Base{Name: "ANX"}
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

func TestOrderSides(t *testing.T) {
	t.Parallel()

	var os = BuyOrderSide
	if os.ToString() != "BUY" {
		t.Errorf("test failed - unexpected string %s", os.ToString())
	}

	if os.ToLower() != "buy" {
		t.Errorf("test failed - unexpected string %s", os.ToString())
	}
}

func TestOrderTypes(t *testing.T) {
	t.Parallel()

	var ot OrderType = "Mo'Money"

	if ot.ToString() != "Mo'Money" {
		t.Errorf("test failed - unexpected string %s", ot.ToString())
	}

	if ot.ToLower() != "mo'money" {
		t.Errorf("test failed - unexpected string %s", ot.ToString())
	}
}

func TestFilterOrdersByType(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestIsAssetTypeSupported(t *testing.T) {
	t.Parallel()

	var b Base
	b.CurrencyPairs.AssetTypes = asset.Items{
		asset.Spot,
	}

	if !b.IsAssetTypeSupported(asset.Spot) {
		t.Error("spot should be supported")
	}
	if b.IsAssetTypeSupported(asset.Index) {
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
