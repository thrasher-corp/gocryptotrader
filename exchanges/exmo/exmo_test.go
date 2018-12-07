package exmo

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

const (
	APIKey                  = ""
	APISecret               = ""
	canManipulateRealOrders = false
)

var (
	e EXMO
)

func TestDefault(t *testing.T) {
	e.SetDefaults()
}

func TestSetup(t *testing.T) {
	e.AuthenticatedAPISupport = true
	e.APIKey = APIKey
	e.APISecret = APISecret
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades("BTC_USD")
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook("BTC_USD")
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker("BTC_USD")
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetPairSettings(t *testing.T) {
	t.Parallel()
	_, err := e.GetPairSettings()
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrency()
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	if APIKey == "" || APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := e.GetUserInfo()
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetRequiredAmount(t *testing.T) {
	t.Parallel()
	if APIKey == "" || APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := e.GetRequiredAmount("BTC_USD", 100)
	if err != nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func TestGetCryptoDepositAddress(t *testing.T) {
	t.Parallel()
	if APIKey == "" || APISecret == "" {
		t.Skip()
	}
	TestSetup(t)
	_, err := e.GetCryptoDepositAddress()
	if err == nil {
		t.Errorf("Test failed. Err: %s", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "",
		FeeType:             exchange.CryptocurrencyTradeFee,
		FirstCurrency:       symbol.BTC,
		SecondCurrency:      symbol.LTC,
		IsMaker:             false,
		PurchasePrice:       1,
		CurrencyItem:        symbol.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFee(t *testing.T) {
	e.SetDefaults()
	TestSetup(t)
	t.Parallel()

	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := e.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := e.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := e.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := e.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := e.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0005), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := e.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := e.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.RUB
	if resp, err := e.GetFee(feeBuilder); resp != float64(1600) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(1600), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.PLN
	if resp, err := e.GetFee(feeBuilder); resp != float64(30) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(30), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.PLN
	if resp, err := e.GetFee(feeBuilder); resp != float64(125) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(125), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.TRY
	if resp, err := e.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.EUR
	if resp, err := e.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.RUB
	if resp, err := e.GetFee(feeBuilder); resp != float64(3200) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(3200), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	e.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText
	// Act
	withdrawPermissions := e.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if e.APIKey == "" || e.APISecret == "" ||
		e.APIKey == "Key" || e.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	e.SetDefaults()
	TestSetup(t)
	e.Verbose = true

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.USD,
	}
	response, err := e.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 10, "1234234")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	e.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	e.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := e.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	e.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	e.Verbose = true
	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := e.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}
