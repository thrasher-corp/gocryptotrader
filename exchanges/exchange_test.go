package exchange

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

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

	b.Name = "ANX"
	err = b.SetAssetTypes()
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes. Error %s", err)
	}

	exch, err := cfg.GetExchangeConfig(b.Name)
	if err != nil {
		t.Fatalf("Test failed. TestSetAssetTypes load config failed. Error %s", err)
	}

	exch.AssetTypes = ""
	err = cfg.UpdateExchangeConfig(exch)
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

	b.Name = "ANX"
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
	err = cfg.UpdateExchangeConfig(exch)
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
		b.ConfigCurrencyPairFormat.Index != "BTC" &&
		b.ConfigCurrencyPairFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat ConfigCurrencyPairFormat values are incorrect")
	}

	if b.RequestCurrencyPairFormat.Delimiter != "" &&
		b.RequestCurrencyPairFormat.Index != "BTC" &&
		b.RequestCurrencyPairFormat.Uppercase {
		t.Fatal("Test failed. TestSetCurrencyPairFormat RequestCurrencyPairFormat values are incorrect")
	}

	// if currency pairs are the same as the config, should load from config
	err = b.SetCurrencyPairFormat()
	if err != nil {
		t.Fatalf("Test failed. TestSetCurrencyPairFormat. Error %s", err)
	}
}

func TestGetAuthenticatedAPISupport(t *testing.T) {
	base := Base{
		AuthenticatedAPISupport: false,
	}

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
		t.Error("Test Failed - Exchange getName() returned incorrect name")
	}
}

func TestGetEnabledCurrencies(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.EnabledPairs = []string{"BTC-USD"}
	format := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Index:     "",
	}

	b.RequestCurrencyPairFormat = format
	b.ConfigCurrencyPairFormat = format
	c := b.GetEnabledCurrencies()
	if c[0].Pair().String() != "BTC-USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = "~"
	b.RequestCurrencyPairFormat = format
	c = b.GetEnabledCurrencies()
	if c[0].Pair().String() != "BTC-USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = ""
	b.ConfigCurrencyPairFormat = format
	c = b.GetEnabledCurrencies()
	if c[0].Pair().String() != "BTC-USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = []string{"BTCDOGE"}
	format.Index = "BTC"
	b.ConfigCurrencyPairFormat = format
	c = b.GetEnabledCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "DOGE" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = []string{"BTC_USD"}
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = "_"
	c = b.GetEnabledCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = []string{"BTCDOGE"}
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Index = "BTC"
	c = b.GetEnabledCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "DOGE" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.EnabledPairs = []string{"BTCUSD"}
	b.ConfigCurrencyPairFormat.Index = ""
	c = b.GetEnabledCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}
}

func TestGetAvailableCurrencies(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.AvailablePairs = []string{"BTC-USD"}
	format := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Index:     "",
	}

	b.RequestCurrencyPairFormat = format
	b.ConfigCurrencyPairFormat = format
	c := b.GetAvailableCurrencies()
	if c[0].Pair().String() != "BTC-USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = "~"
	b.RequestCurrencyPairFormat = format
	c = b.GetAvailableCurrencies()
	if c[0].Pair().String() != "BTC-USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	format.Delimiter = ""
	b.ConfigCurrencyPairFormat = format
	c = b.GetAvailableCurrencies()
	if c[0].Pair().String() != "BTC-USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = []string{"BTCDOGE"}
	format.Index = "BTC"
	b.ConfigCurrencyPairFormat = format
	c = b.GetAvailableCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "DOGE" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = []string{"BTC_USD"}
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = "_"
	c = b.GetAvailableCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = []string{"BTCDOGE"}
	b.RequestCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Delimiter = ""
	b.ConfigCurrencyPairFormat.Index = "BTC"
	c = b.GetAvailableCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "DOGE" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}

	b.AvailablePairs = []string{"BTCUSD"}
	b.ConfigCurrencyPairFormat.Index = ""
	c = b.GetAvailableCurrencies()
	if c[0].FirstCurrency.String() != "BTC" && c[0].SecondCurrency.String() != "USD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}
}

func TestSupportsCurrency(t *testing.T) {
	b := Base{
		Name: "TESTNAME",
	}

	b.AvailablePairs = []string{"BTC-USD", "ETH-USD"}
	b.EnabledPairs = []string{"BTC-USD"}

	format := config.CurrencyPairFormatConfig{
		Delimiter: "-",
		Index:     "",
	}

	b.RequestCurrencyPairFormat = format
	b.ConfigCurrencyPairFormat = format

	if !b.SupportsCurrency(pair.NewCurrencyPair("BTC", "USD"), true) {
		t.Error("Test Failed - Exchange SupportsCurrency() incorrect value")
	}

	if !b.SupportsCurrency(pair.NewCurrencyPair("ETH", "USD"), false) {
		t.Error("Test Failed - Exchange SupportsCurrency() incorrect value")
	}

	if b.SupportsCurrency(pair.NewCurrencyPair("ASD", "ASDF"), true) {
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
	actual := GetExchangeFormatCurrencySeperator("WEX")

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

	var pairs []pair.CurrencyPair
	pairs = append(pairs, pair.NewCurrencyPairDelimiter("BTC_USD", "_"))
	pairs = append(pairs, pair.NewCurrencyPairDelimiter("LTC_BTC", "_"))

	actual, err := GetAndFormatExchangeCurrencies("Liqui", pairs)
	if err != nil {
		t.Errorf("Test failed - Exchange TestGetAndFormatExchangeCurrencies error %s", err)
	}
	expected := pair.CurrencyItem("btc_usd-ltc_btc")

	if actual.String() != expected.String() {
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

	pair := pair.NewCurrencyPair("BTC", "USD")
	expected := "BTC-USD"
	actual := FormatExchangeCurrency("GDAX", pair)

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

	currency := pair.NewCurrencyPair("btc", "usd")
	expected := "BTC-USD"
	actual := FormatCurrency(currency).String()
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

func TestSetAPIKeys(t *testing.T) {
	SetAPIKeys := Base{
		Name:                    "TESTNAME",
		Enabled:                 false,
		AuthenticatedAPISupport: false,
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)
	if SetAPIKeys.APIKey != "" && SetAPIKeys.APISecret != "" && SetAPIKeys.ClientID != "" {
		t.Error("Test Failed - SetAPIKeys() set values without authenticated API support enabled")
	}

	SetAPIKeys.AuthenticatedAPISupport = true
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
	UAC.AvailablePairs = []string{"ETHLTC", "LTCBTC"}
	UAC.EnabledPairs = []string{"ETHLTC"}
	newPair := pair.NewCurrencyPair("ETH", "USDT")

	err = UAC.SetCurrencies([]pair.CurrencyPair{newPair}, true)
	if err == nil {
		t.Fatal("Test failed. TestSetCurrencies returned nil error on non-existant exchange")
	}

	UAC.Name = "ANX"
	UAC.SetCurrencies([]pair.CurrencyPair{newPair}, true)
	if !pair.Contains(UAC.GetEnabledCurrencies(), newPair) {
		t.Fatal("Test failed. TestSetCurrencies failed to set currencies")
	}

	UAC.SetCurrencies([]pair.CurrencyPair{newPair}, false)
	if !pair.Contains(UAC.GetAvailableCurrencies(), newPair) {
		t.Fatal("Test failed. TestSetCurrencies failed to set currencies")
	}
}

func TestUpdateEnabledCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestUpdateEnabledCurrencies failed to load config")
	}

	UAC := Base{Name: "ANX"}
	exchangeProducts := []string{"ltc", "btc", "usd", "aud"}

	// Test updating exchange products for an exchange which doesn't exist
	UAC.Name = "Blah"
	err = UAC.UpdateEnabledCurrencies(exchangeProducts, false)
	if err == nil {
		t.Errorf("Test Failed - Exchange TestUpdateEnabledCurrencies succeeded on an exchange which doesn't exist")
	}

	// Test updating exchange products
	UAC.Name = "ANX"
	err = UAC.UpdateEnabledCurrencies(exchangeProducts, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange TestUpdateEnabledCurrencies error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	UAC.Name = "ANX"
	err = UAC.UpdateEnabledCurrencies(exchangeProducts, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange TestUpdateEnabledCurrencies error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = []string{"btc"}
	err = UAC.UpdateEnabledCurrencies(exchangeProducts, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange TestUpdateEnabledCurrencies error: %s", err)
	}
}

func TestUpdateAvailableCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatal("Test failed. TestUpdateAvailableCurrencies failed to load config")
	}

	UAC := Base{Name: "ANX"}
	exchangeProducts := []string{"ltc", "btc", "usd", "aud"}

	// Test updating exchange products for an exchange which doesn't exist
	UAC.Name = "Blah"
	err = UAC.UpdateAvailableCurrencies(exchangeProducts, false)
	if err == nil {
		t.Errorf("Test Failed - Exchange UpdateAvailableCurrencies() succeeded on an exchange which doesn't exist")
	}

	// Test updating exchange products
	UAC.Name = "ANX"
	err = UAC.UpdateAvailableCurrencies(exchangeProducts, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdateAvailableCurrencies() error: %s", err)
	}

	// Test updating the same new products, diff should be 0
	UAC.Name = "ANX"
	err = UAC.UpdateAvailableCurrencies(exchangeProducts, false)
	if err != nil {
		t.Errorf("Test Failed - Exchange UpdateAvailableCurrencies() error: %s", err)
	}

	// Test force updating to only one product
	exchangeProducts = []string{"btc"}
	err = UAC.UpdateAvailableCurrencies(exchangeProducts, true)
	if err != nil {
		t.Errorf("Test Failed - Forced Exchange UpdateAvailableCurrencies() error: %s", err)
	}
}
