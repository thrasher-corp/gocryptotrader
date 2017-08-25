package exchange

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestGetName(t *testing.T) {
	GetName := Base{
		Name: "TESTNAME",
	}

	name := GetName.GetName()
	if name != "TESTNAME" {
		t.Error("Test Failed - Exchange getName() returned incorrect name")
	}
}

func TestSetCurrencyPairFormat(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	exch, err := cfg.GetExchangeConfig("GDAX")
	if err != nil {
		t.Fatalf("Failed to load GDAX exchange config. Error: %s", err)
	}

	exch.RequestCurrencyPairFormat = nil
	exch.ConfigCurrencyPairFormat = nil

	err = cfg.UpdateExchangeConfig(exch)
	if err != nil {
		t.Fatalf("Failed to update GDAX config. Error: %s", err)
	}

	// to-do
}

func TestGetEnabledCurrencies(t *testing.T) {
	enabledPairs := []string{"BTCUSD", "BTCAUD", "LTCUSD", "LTCAUD"}
	GetEnabledCurrencies := Base{
		Name:         "TESTNAME",
		EnabledPairs: enabledPairs,
	}

	enCurr := GetEnabledCurrencies.GetEnabledCurrencies()
	if enCurr[0].Pair().String() != "BTCUSD" {
		t.Error("Test Failed - Exchange GetEnabledCurrencies() incorrect string")
	}
}

func TestGetAvailableCurrencies(t *testing.T) {
	availablePairs := []string{"BTCUSD", "BTCAUD", "LTCUSD", "LTCAUD"}
	GetEnabledCurrencies := Base{
		Name:           "TESTNAME",
		AvailablePairs: availablePairs,
	}

	enCurr := GetEnabledCurrencies.GetAvailableCurrencies()
	if enCurr[0].Pair().String() != "BTCUSD" {
		t.Error("Test Failed - Exchange GetAvailableCurrencies() incorrect string")
	}
}

func TestGetExchangeFormatCurrencySeperator(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	if err != nil {
		t.Fatalf("Failed to load config file. Error: %s", err)
	}

	expected := true
	actual := GetExchangeFormatCurrencySeperator("BTCE")

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
		Name:    "TESTNAME",
		Enabled: false,
	}

	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", false)
	if SetAPIKeys.APIKey != "RocketMan" && SetAPIKeys.APISecret != "Digereedoo" && SetAPIKeys.ClientID != "007" {
		t.Error("Test Failed - Exchange SetAPIKeys() did not set correct values")
	}
	SetAPIKeys.SetAPIKeys("RocketMan", "Digereedoo", "007", true)
}

func TestUpdateEnabledCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	UAC := Base{Name: "ANX"}
	enabledCurrencies := []string{"ltc", "btc", "usd", "aud"}

	if err != nil {
		t.Error(
			"Test Failed - Exchange UpdateEnabledCurrencies() did not set correct values",
		)
	}
	err2 := UAC.UpdateEnabledCurrencies(enabledCurrencies, false)
	if err2 != nil {
		t.Errorf("Test Failed - Exchange UpdateEnabledCurrencies() error: %s", err2)
	}
}

func TestUpdateAvailableCurrencies(t *testing.T) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig(config.ConfigTestFile)
	UAC := Base{Name: "ANX"}
	exchangeProducts := []string{"ltc", "btc", "usd", "aud"}

	if err != nil {
		t.Error(
			"Test Failed - Exchange UpdateAvailableCurrencies() did not set correct values",
		)
	}
	err2 := UAC.UpdateAvailableCurrencies(exchangeProducts, false)
	if err2 != nil {
		t.Errorf("Test Failed - Exchange UpdateAvailableCurrencies() error: %s", err2)
	}
}
