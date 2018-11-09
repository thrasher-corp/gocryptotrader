package anx

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var a ANX

const (
	canPlaceOrders = false
)

func TestSetDefaults(t *testing.T) {
	a.SetDefaults()

	if a.Name != "ANX" {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.Enabled != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.TakerFee != 0.02 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.MakerFee != 0.01 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.Verbose != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.Websocket.IsEnabled() != false {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
	if a.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX SetDefaults() incorrect values set")
	}
}

func TestSetup(t *testing.T) {
	anxSetupConfig := config.GetConfig()
	anxSetupConfig.LoadConfig("../../testdata/configtest.json")
	anxConfig, err := anxSetupConfig.GetExchangeConfig("ANX")
	anxConfig.AuthenticatedAPISupport = true

	if err != nil {
		t.Error("Test Failed - ANX Setup() init error")
	}
	a.Setup(anxConfig)

	if a.Enabled != true {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}

	if a.RESTPollingDelay != 10 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if a.Verbose != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if a.Websocket.IsEnabled() != false {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.BaseCurrencies) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.AvailablePairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
	if len(a.EnabledPairs) <= 0 {
		t.Error("Test Failed - ANX Setup() incorrect values set")
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := a.GetCurrencies()
	if err != nil {
		t.Fatalf("Test failed. TestGetCurrencies failed. Err: %s", err)
	}
}

func TestGetTradablePairs(t *testing.T) {
	_, err := a.GetTradablePairs()
	if err != nil {
		t.Fatalf("Test failed. TestGetTradablePairs failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	ticker, err := a.GetTicker("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetTicker() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetTicker() unsuccessful")
	}
}

func TestGetDepth(t *testing.T) {
	ticker, err := a.GetDepth("BTCUSD")
	if err != nil {
		t.Errorf("Test Failed - ANX GetDepth() error: %s", err)
	}
	if ticker.Result != "success" {
		t.Error("Test Failed - ANX GetDepth() unsuccessful")
	}
}

func TestGetAPIKey(t *testing.T) {
	apiKey, apiSecret, err := a.GetAPIKey("userName", "passWord", "", "1337")
	if err == nil {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
	if apiKey != "" {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
	if apiSecret != "" {
		t.Error("Test Failed - ANX GetAPIKey() Incorrect")
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:         1,
		Delimiter:      "",
		FeeType:        exchange.CryptocurrencyTradeFee,
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
		IsMaker:        false,
		PurchasePrice:  1,
	}
}

func TestGetFee(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.02) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := a.GetFee(feeBuilder); resp != float64(20000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(20000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.01) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.01), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := a.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := a.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := a.GetFee(feeBuilder); resp != float64(250.01) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(250.01), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	a.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.WithdrawCryptoWith2FAText + " & " +
		exchange.WithdrawCryptoWithEmailText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	// Act
	withdrawPermissions := a.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// This will really really use the API to place an order
// If you're going to test this, make sure you're willing to place real orders on the exchange
func TestSubmitOrder(t *testing.T) {
	a.SetDefaults()
	TestSetup(t)

	if a.APIKey == "" || a.APISecret == "" ||
		a.APIKey == "Key" || a.APISecret == "Secret" ||
		!canPlaceOrders {
		t.Skip()
	}
	var p = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  "BTC",
		SecondCurrency: "USD",
	}
	response, err := a.SubmitExchangeOrder(p, exchange.Buy, exchange.Market, 1, 1, "clientId")
	if err != nil {
		t.Error("Something happehned: ", err)
	}
	if response == "" {
		t.Errorf("OrderId not returned")
	}
}
