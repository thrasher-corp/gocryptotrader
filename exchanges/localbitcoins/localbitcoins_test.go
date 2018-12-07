package localbitcoins

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var l LocalBitcoins

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	l.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
	if err != nil {
		t.Error("Test Failed - LakeBTC Setup() init error")
	}

	localbitcoinsConfig.AuthenticatedAPISupport = true
	localbitcoinsConfig.APIKey = apiKey
	localbitcoinsConfig.APISecret = apiSecret

	l.Setup(localbitcoinsConfig)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := l.GetTicker()
	if err != nil {
		t.Errorf("Test failed - GetTicker() returned: %s", err)
	}
}

func TestGetTradableCurrencies(t *testing.T) {
	t.Parallel()
	_, err := l.GetTradableCurrencies()
	if err != nil {
		t.Errorf("Test failed - GetTradableCurrencies() returned: %s", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.GetAccountInformation("", true)
	if err == nil {
		t.Error("Test Failed - GetAccountInformation() error", err)
	}
	_, err = l.GetAccountInformation("bitcoinbaron", false)
	if err != nil {
		t.Error("Test Failed - GetAccountInformation() error", err)
	}
}

func TestGetads(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	_, err := l.Getads("")
	if err == nil {
		t.Error("Test Failed - Getads() - Full list, error", err)
	}
	_, err = l.Getads("1337")
	if err == nil {
		t.Error("Test Failed - Getads() error", err)
	}
}

func TestEditAd(t *testing.T) {
	t.Parallel()
	if l.APIKey == "" || l.APISecret == "" {
		t.Skip()
	}
	edit := AdEdit{}
	err := l.EditAd(edit, "1337")
	if err == nil {
		t.Error("Test Failed - EditAd() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "-",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.LTC,
		SecondCurrency:      symbol.BTC,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	l.SetDefaults()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := l.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	l.SetDefaults()
	expectedResult := exchange.WithdrawCryptoViaWebsiteOnlyText
	// Act
	withdrawPermissions := l.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if l.APIKey == "" || l.APISecret == "" ||
		l.APIKey == "Key" || l.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	l.SetDefaults()
	TestSetup(t)
	l.Verbose = true

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.EUR,
	}
	response, err := l.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 10, "hi")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	l.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	l.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := l.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	l.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	l.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := l.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}
