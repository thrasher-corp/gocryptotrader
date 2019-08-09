package exchange

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

const (
	defaultTestExchange     = "ANX"
	defaultTestCurrencyPair = "BTC-USD"
)

func TestSupportsRESTTickerBatchUpdates(t *testing.T) {
	b := Base{
		Name:                       "RAWR",
		SupportsRESTTickerBatching: true,
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
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults failed to load config file. Error: %s", err)
	}

	b := Base{
		Name:                     "TESTNAME",
		SupportsAutoPairUpdating: true,
	}

	err = b.SetAutoPairDefaults()
	if err == nil {
		t.Fatal("Test failed. TestSetAutoPairDefaults returned nil error for a non-existent exchange")
	}

	b.Name = "Bitstamp"
	err = b.SetAutoPairDefaults()
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults. Error %s", err)
	}

	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if !exch.SupportsAutoPairUpdates {
		t.Fatalf("Test failed. TestSetAutoPairDefaults Incorrect value")
	}

	if exch.PairsLastUpdated != 0 {
		t.Fatalf("Test failed. TestSetAutoPairDefaults Incorrect value")
	}

	exch.SupportsAutoPairUpdates = false
	err = cfg.UpdateExchangeConfig(&exch)
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults update config failed. Error %s", err)
	}

	exch, err = cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if exch.SupportsAutoPairUpdates {
		t.Fatal("Test failed. TestSetAutoPairDefaults Incorrect value")
	}

	err = b.SetAutoPairDefaults()
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults. Error %s", err)
	}

	exch, err = cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults load config failed. Error %s", err)
	}

	if !exch.SupportsAutoPairUpdates {
		t.Fatal("Test failed. TestSetAutoPairDefaults Incorrect value")
	}

	b.SupportsAutoPairUpdating = false
	err = b.SetAutoPairDefaults()
	if err != nil {
		t.Fatalf("Test failed. TestSetAutoPairDefaults. Error %s", err)
	}

	if b.PairsLastUpdated == 0 {
		t.Fatal("Test failed. TestSetAutoPairDefaults Incorrect value")
	}
}

func TestSupportsAutoPairUpdates(t *testing.T) {
	b := Base{
		Name:                     "TESTNAME",
		SupportsAutoPairUpdating: false,
	}

	if b.SupportsAutoPairUpdates() {
		t.Fatal("Test failed. TestSupportsAutoPairUpdates Incorrect value")
	}
}

func TestGetLastPairsUpdateTime(t *testing.T) {
	testTime := time.Now().Unix()
	b := Base{
		Name:             "TESTNAME",
		PairsLastUpdated: testTime,
	}

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

	err = b.SetAssetTypes()
	if err == nil {
		t.Fatal("Test failed. TestSetAssetTypes returned nil error for a non-existent exchange")
	}

	b.Name = defaultTestExchange
	b.AssetTypes = []string{orderbook.Spot}
	err = b.SetAssetTypes()
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes. Error %s", err)
	}

	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes load config failed. Error %s", err)
	}

	exch.AssetTypes = ""
	err = cfg.UpdateExchangeConfig(&exch)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes update config failed. Error %s", err)
	}

	exch, err = cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes load config failed. Error %s", err)
	}

	if exch.AssetTypes != "" {
		t.Fatal("Test failed. TestSetAssetTypes assetTypes != ''")
	}

	err = b.SetAssetTypes()
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes. Error %s", err)
	}

	if !common.StringDataCompare(b.AssetTypes, ticker.Spot) {
		t.Fatal("Test failed. TestSetAssetTypes assetTypes is not set")
	}
}

func TestGetAssetTypes(t *testing.T) {
	testExchange := Base{
		AssetTypes: []string{orderbook.Spot, "Binary", "Futures"},
	}

	aT := testExchange.GetAssetTypes()
	if len(aT) != 3 {
		t.Error("Test failed. TestGetAssetTypes failed")
	}
}

func TestGetExchangeAssetTypes(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	result, err := GetExchangeAssetTypes("Bitfinex")
	if err != nil {
		t.Fatal("Test failed. Unable to obtain Bitfinex asset types")
	}

	if !common.StringDataCompare(result, ticker.Spot) {
		t.Fatal("Test failed. Bitfinex does not contain default asset type 'SPOT'")
	}

	_, err = GetExchangeAssetTypes("non-existent-exchange")
	if err == nil {
		t.Fatal("Test failed. Got asset types for non-existent exchange")
	}
}

func TestCompareCurrencyPairFormats(t *testing.T) {
	cfgOne := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Uppercase: true,
		Index:     "",
		Separator: ",",
	}

	cfgTwo := cfgOne
	if !CompareCurrencyPairFormats(cfgOne, &cfgTwo) {
		t.Fatal("Test failed. CompareCurrencyPairFormats should be true")
	}

	cfgTwo.Delimiter = "~"
	if CompareCurrencyPairFormats(cfgOne, &cfgTwo) {
		t.Fatal("Test failed. CompareCurrencyPairFormats should not be true")
	}
}

func TestSetCurrencyPairFormat(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat failed to load config file. Error: %s", err)
	}

	b := Base{
		Name: "TESTNAME",
	}

	err = b.SetCurrencyPairFormat()
	if err == nil {
		t.Fatal("Test failed. TestSetCurrencyPairFormat returned nil error for a non-existent exchange")
	}

	b.Name = defaultTestExchange
	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat. Error %s", err)
	}

	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat load config failed. Error %s", err)
	}

	exch.ConfigCurrencyPairFormat = nil
	exch.RequestCurrencyPairFormat = nil
	err = cfg.UpdateExchangeConfig(&exch)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat update config failed. Error %s", err)
	}

	exch, err = cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat load config failed. Error %s", err)
	}

	if exch.ConfigCurrencyPairFormat != nil && exch.RequestCurrencyPairFormat != nil {
		t.Fatal("Test failed. TestSetCurrencyPairFormat exch values are not nil")
	}

	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat. Error %s", err)
	}

	if b.ConfigCurrencyPairFormat.Delimiter != "" &&
		b.ConfigCurrencyPairFormat.Index != currency.BTC.String() &&
		b.ConfigCurrencyPairFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat ConfigCurrencyPairFormat values are incorrect")
	}

	if b.RequestCurrencyPairFormat.Delimiter != "" &&
		b.RequestCurrencyPairFormat.Index != currency.BTC.String() &&
		b.RequestCurrencyPairFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat RequestCurrencyPairFormat values are incorrect")
	}

	// if currency pairs are the same as the config, should load from config
	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat. Error %s", err)
	}
}

// TestGetAuthenticatedAPISupport logic test
func TestGetAuthenticatedAPISupport(t *testing.T) {
	base := Base{
		AuthenticatedAPISupport:          true,
		AuthenticatedWebsocketAPISupport: false,
	}

	if !base.GetAuthenticatedAPISupport(RestAuthentication) {
		t.Fatal("Test failed. Expected RestAuthentication to return true")
	}
	if base.GetAuthenticatedAPISupport(WebsocketAuthentication) {
		t.Fatal("Test failed. Expected WebsocketAuthentication to return false")
	}
	base.AuthenticatedWebsocketAPISupport = true
	if !base.GetAuthenticatedAPISupport(WebsocketAuthentication) {
		t.Fatal("Test failed. Expected WebsocketAuthentication to return true")
	}
	if base.GetAuthenticatedAPISupport(2) {
		t.Fatal("Test failed. Expected default case of 'false' to be returned")
	}
}

func TestGetName(t *testing.T) {
	GetName := Base{
		Name: "TESTNAME",
	}

	name := GetName.GetName()
	if name != "TESTNAME" {
		t.Error("Test Failed - Exchange getName() returned incorrect name")
	}
}

func TestGetEnabledCurrencies(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.EnabledPairs = currency.NewPairsFromStrings([]string{"BTC-USD"})
	format := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	b.RequestCurrencyPairFormat = format
	b.ConfigCurrencyPairFormat = format
	c := b.GetEnabledCurrencies()
	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = "~"
	b.RequestCurrencyPairFormat = format
	c = b.GetEnabledCurrencies()
	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = ""
	b.ConfigCurrencyPairFormat = format
	c = b.GetEnabledCurrencies()
	if c[0].String() != "BTCUSD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = currency.NewPairsFromStrings([]string{"BTCDOGE"})
	format.Index = "BTC"
	b.ConfigCurrencyPairFormat = format
	c = b.GetEnabledCurrencies()
	if c[0].Base.String() != "BTC" && c[0].Quote.String() != "DOGE" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = currency.NewPairsFromStrings([]string{"BTC_USD"})
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = "_"
	c = b.GetEnabledCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = currency.NewPairsFromStrings([]string{"BTCDOGE"})
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Index = currency.BTC.String()
	c = b.GetEnabledCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = currency.NewPairsFromStrings([]string{"BTCUSD"})
	b.ConfigCurrencyPairFormat.Index = ""
	c = b.GetEnabledCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}
}

func TestGetAvailableCurrencies(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.AvailablePairs = currency.NewPairsFromStrings([]string{"BTC-USD"})
	format := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Index:     "",
		Uppercase: true,
	}

	b.RequestCurrencyPairFormat = format
	b.ConfigCurrencyPairFormat = format
	c := b.GetAvailableCurrencies()
	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = "~"
	b.RequestCurrencyPairFormat = format
	c = b.GetAvailableCurrencies()
	if c[0].String() != defaultTestCurrencyPair {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = ""
	b.ConfigCurrencyPairFormat = format
	c = b.GetAvailableCurrencies()
	if c[0].String() != "BTCUSD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string", c[0])
	}

	b.AvailablePairs = currency.NewPairsFromStrings([]string{"BTCDOGE"})
	format.Index = currency.BTC.String()
	b.ConfigCurrencyPairFormat = format
	c = b.GetAvailableCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = currency.NewPairsFromStrings([]string{"BTC_USD"})
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = "_"
	c = b.GetAvailableCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = currency.NewPairsFromStrings([]string{"BTCDOGE"})
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Index = currency.BTC.String()
	c = b.GetAvailableCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.DOGE {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = currency.NewPairsFromStrings([]string{"BTCUSD"})
	b.ConfigCurrencyPairFormat.Index = ""
	c = b.GetAvailableCurrencies()
	if c[0].Base != currency.BTC && c[0].Quote != currency.USD {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}
}

func TestSupportsCurrency(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.AvailablePairs = currency.NewPairsFromStrings([]string{defaultTestCurrencyPair, "ETH-USD"})
	b.EnabledPairs = currency.NewPairsFromStrings([]string{defaultTestCurrencyPair})

	format := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Index:     "",
	}

	b.RequestCurrencyPairFormat = format
	b.ConfigCurrencyPairFormat = format

	if !b.SupportsCurrency(currency.NewPairFromStrings("BTC", "USD"), true) {
		t.Error("Test Failed - Exchange SupportsCurrency() incorrect value")
	}

	if !b.SupportsCurrency(currency.NewPairFromStrings("ETH", "USD"), false) {
		t.Error("Test Failed - Exchange SupportsCurrency() incorrect value")
	}

	if b.SupportsCurrency(currency.NewPairFromStrings("ASD", "ASDF"), true) {
		t.Error("Test Failed - Exchange SupportsCurrency() incorrect value")
	}
}
func TestGetExchangeFormatCurrencySeperator(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	expected := true
	actual := GetExchangeFormatCurrencySeperator("Yobit")

	if expected != actual {
		t.Errorf("Test failed - TestGetExchangeFormatCurrencySeperator expected %v != actual %v",
			expected, actual)
	}

	expected = false
	actual = GetExchangeFormatCurrencySeperator("LocalBitcoins")

	if expected != actual {
		t.Errorf("Test failed - TestGetExchangeFormatCurrencySeperator expected %v != actual %v",
			expected, actual)
	}

	expected = false
	actual = GetExchangeFormatCurrencySeperator("blah")

	if expected != actual {
		t.Errorf("Test failed - TestGetExchangeFormatCurrencySeperator expected %v != actual %v",
			expected, actual)
	}
}

func TestGetAndFormatExchangeCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	var pairs = []currency.Pair{
		currency.NewPairDelimiter("BTC_USD", "_"),
		currency.NewPairDelimiter("LTC_BTC", "_"),
	}

	actual, err := GetAndFormatExchangeCurrencies("Yobit", pairs)
	if err != nil {
		t.Errorf("Test failed - Exchange TestGetAndFormatExchangeCurrencies error %s", err)
	}
	expected := "btc_usd-ltc_btc"

	if actual != expected {
		t.Errorf("Test failed - Exchange TestGetAndFormatExchangeCurrencies %s != %s",
			actual, expected)
	}

	_, err = GetAndFormatExchangeCurrencies("non-existent", pairs)
	if err == nil {
		t.Errorf("Test failed - Exchange TestGetAndFormatExchangeCurrencies returned nil error on non-existent exchange")
	}
}

func TestFormatExchangeCurrency(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	expected := defaultTestCurrencyPair
	actual := FormatExchangeCurrency("CoinbasePro", p)

	if actual.String() != expected {
		t.Errorf("Test failed - Exchange TestFormatExchangeCurrency %s != %s",
			actual, expected)
	}
}

func TestFormatCurrency(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	p := currency.NewPair(currency.BTC, currency.USD)
	expected := defaultTestCurrencyPair
	actual := FormatCurrency(p).String()
	if actual != expected {
		t.Errorf("Test failed - Exchange TestFormatCurrency %s != %s",
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

// TestSetAPIKeys logic test
func TestSetAPIKeys(t *testing.T) {
	SetAPIKeys := Base{
		Name:                             "TESTNAME",
		Enabled:                          false,
		AuthenticatedAPISupport:          false,
		AuthenticatedWebsocketAPISupport: false,
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)
	if SetAPIKeys.APIKey != "" && SetAPIKeys.APISecret != "" && SetAPIKeys.ClientID != "" {
		t.Error("Test Failed - SetAPIKeys() set values without authenticated API support enabled")
	}

	SetAPIKeys.AuthenticatedAPISupport = true
	SetAPIKeys.AuthenticatedWebsocketAPISupport = true
	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)
	if SetAPIKeys.APIKey != "RocketMan" && SetAPIKeys.APISecret != "Digereedoo" && SetAPIKeys.ClientID != "007" {
		t.Error("Test Failed - Exchange SetAPIKeys() did not set correct values")
	}

	SetAPIKeys.AuthenticatedAPISupport = false
	SetAPIKeys.AuthenticatedWebsocketAPISupport = true
	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)
	if SetAPIKeys.APIKey != "RocketMan" && SetAPIKeys.APISecret != "Digereedoo" && SetAPIKeys.ClientID != "007" {
		t.Error("Test Failed - Exchange SetAPIKeys() did not set correct values")
	}

	SetAPIKeys.AuthenticatedAPISupport = true
	SetAPIKeys.AuthenticatedWebsocketAPISupport = false
	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)
	if SetAPIKeys.APIKey != "RocketMan" && SetAPIKeys.APISecret != "Digereedoo" && SetAPIKeys.ClientID != "007" {
		t.Error("Test Failed - Exchange SetAPIKeys() did not set correct values")
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", true)
}

func TestSetCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestSetCurrencies failed to load config")
	}

	UAC := Base{Name: "ASDF"}
	UAC.AvailablePairs = currency.NewPairsFromStrings([]string{"ETHLTC", "LTCBTC"})
	UAC.EnabledPairs = currency.NewPairsFromStrings([]string{"ETHLTC"})
	newPair := currency.NewPairDelimiter("ETH_USDT", "_")

	err = UAC.SetCurrencies([]currency.Pair{newPair}, true)
	if err == nil {
		t.Fatal("Test failed. TestSetCurrencies returned nil error on non-existent exchange")
	}

	anxCfg, err := cfg.GetExchangeConfig(defaultTestExchange)
	if err != nil {
		t.Fatal("Test failed. TestSetCurrencies failed to load config")
	}

	UAC.Name = defaultTestExchange
	UAC.ConfigCurrencyPairFormat.Delimiter = anxCfg.ConfigCurrencyPairFormat.Delimiter
	UAC.SetCurrencies(currency.Pairs{newPair}, true)
	if !UAC.GetEnabledCurrencies().Contains(newPair, true) {
		t.Fatal("Test failed. TestSetCurrencies failed to set currencies")
	}

	UAC.SetCurrencies(currency.Pairs{newPair}, false)
	if !UAC.GetAvailableCurrencies().Contains(newPair, true) {
		t.Fatal("Test failed. TestSetCurrencies failed to set currencies")
	}

	err = UAC.SetCurrencies(nil, false)
	if err == nil {
		t.Fatal("Test failed. TestSetCurrencies should return an error when attempting to set an empty pairs array")
	}
}

func TestUpdateCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestUpdateEnabledCurrencies failed to load config")
	}

	UAC := Base{Name: "ANX"}
	exchangeProducts := currency.NewPairsFromStrings([]string{"ltc", "btc", "usd", "aud", ""})

	// Test updating exchange products for an exchange which doesn't exist
	UAC.Name = "Blah"
	err = UAC.UpdateCurrencies(exchangeProducts, true, false)
	if err == nil {
		t.Errorf("Test Failed - Exchange TestUpdateCurrencies succeeded on an exchange which doesn't exist")
	}

	// Test updating exchange products
	UAC.Name = defaultTestExchange
	err = UAC.UpdateCurrencies(exchangeProducts, true, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange TestUpdateCurrencies error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	UAC.Name = defaultTestExchange
	err = UAC.UpdateCurrencies(exchangeProducts, true, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange TestUpdateCurrencies error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = currency.NewPairsFromStrings([]string{"btc"})
	err = UAC.UpdateCurrencies(exchangeProducts, true, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange TestUpdateCurrencies error: %s", err)
	}

	exchangeProducts = currency.NewPairsFromStrings([]string{"ltc", "btc", "usd", "aud"})
	// Test updating exchange products for an exchange which doesn't exist
	UAC.Name = "Blah"
	err = UAC.UpdateCurrencies(exchangeProducts, false, false)
	if err == nil {
		t.Errorf("Test Failed - Exchange UpdateCurrencies() succeeded on an exchange which doesn't exist")
	}

	// Test updating exchange products
	UAC.Name = defaultTestExchange
	err = UAC.UpdateCurrencies(exchangeProducts, false, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdateCurrencies() error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	UAC.Name = defaultTestExchange
	err = UAC.UpdateCurrencies(exchangeProducts, false, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdateCurrencies() error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = currency.NewPairsFromStrings([]string{"btc"})
	err = UAC.UpdateCurrencies(exchangeProducts, false, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdateCurrencies() error: %s", err)
	}

	// Test update currency pairs with btc excluded
	exchangeProducts = currency.NewPairsFromStrings([]string{"ltc", "eth"})
	err = UAC.UpdateCurrencies(exchangeProducts, false, false)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdateCurrencies() error: %s", err)
	}

	// Test that empty exchange products should return an error
	exchangeProducts = nil
	err = UAC.UpdateCurrencies(exchangeProducts, false, false)
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

	test := config.ExchangeConfig{}

	err := tester.SetAPIURL(&test)
	if err == nil {
		t.Error("test failed - setting zero value config")
	}

	test.APIURL = testURL
	test.APIURLSecondary = testURLSecondary

	tester.APIUrlDefault = testURLDefault
	tester.APIUrlSecondaryDefault = testURLSecondaryDefault

	err = tester.SetAPIURL(&test)
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

	test.APIURL = "https://api.something.com"
	test.APIURLSecondary = "https://api.somethingelse.com"

	tester.APIUrlDefault = "https://api.defaultsomething.com"
	tester.APIUrlSecondaryDefault = "https://api.defaultsomethingelse.com"

	for i := 0; i < b.N; i++ {
		err := tester.SetAPIURL(&test)
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
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestUpdateEnabledCurrencies failed to load config")
	}

	UAC := Base{Name: defaultTestExchange}
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

func TestOrderTypes(t *testing.T) {
	var ot OrderType = "Mo'Money"

	if ot.ToString() != "Mo'Money" {
		t.Errorf("test failed - unexpected string %s", ot.ToString())
	}

	var os OrderSide = "BUY"

	if os.ToString() != "BUY" {
		t.Errorf("test failed - unexpected string %s", os.ToString())
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
