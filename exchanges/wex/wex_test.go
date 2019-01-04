package wex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var w WEX

// Please supply your own keys for better unit testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	isWexEncounteringIssues = true
)

func TestSetDefaults(t *testing.T) {
	w.SetDefaults()
}

func TestSetup(t *testing.T) {
	wexConfig := config.GetConfig()
	wexConfig.LoadConfig("../../testdata/configtest.json")
	conf, err := wexConfig.GetExchangeConfig("WEX")
	if err != nil {
		t.Error("Test Failed - WEX init error")
	}
	conf.APIKey = apiKey
	conf.APISecret = apiSecret
	conf.AuthenticatedAPISupport = true

	w.Setup(conf)
}

func TestGetTradablePairs(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetTradablePairs()
	if err != nil {
		t.Errorf("Test failed. GetTradablePairs err: %s", err)
	}
}

func TestGetInfo(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetInfo()
	if err != nil {
		t.Error("Test Failed - GetInfo() error")
	}
}

func TestGetTicker(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetTicker("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetDepth(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetDepth("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetDepth() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetTrades("btc_usd")
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetAccountInfo(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetAccountInfo()
	if err == nil {
		t.Error("Test Failed - GetAccountInfo() error", err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetActiveOrders("")
	if err == nil {
		t.Error("Test Failed - GetActiveOrders() error", err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetOrderInfo(6196974)
	if err == nil {
		t.Error("Test Failed - GetOrderInfo() error", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.CancelExistingOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelExistingOrder() error", err)
	}
}

func TestTrade(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.Trade("", "buy", 0, 0)
	if err == nil {
		t.Error("Test Failed - Trade() error", err)
	}
}

func TestGetTransactionHistory(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetTransactionHistory(0, 0, 0, "", "", "")
	if err == nil {
		t.Error("Test Failed - GetTransactionHistory() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.GetTradeHistory(0, 0, 0, "", "", "", "")
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestWithdrawCoins(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.WithdrawCoins("", 0, "")
	if err == nil {
		t.Error("Test Failed - WithdrawCoins() error", err)
	}
}

func TestCoinDepositAddress(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.CoinDepositAddress("btc")
	if err == nil {
		t.Error("Test Failed - WithdrawCoins() error", err)
	}
}

func TestCreateCoupon(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.CreateCoupon("bla", 0)
	if err == nil {
		t.Error("Test Failed - CreateCoupon() error", err)
	}
}

func TestRedeemCoupon(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	t.Parallel()
	_, err := w.RedeemCoupon("bla")
	if err == nil {
		t.Error("Test Failed - RedeemCoupon() error", err)
	}
}

func setFeeBuilder() exchange.FeeBuilder {
	return exchange.FeeBuilder{
		Amount:              1,
		Delimiter:           "_",
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
	if isWexEncounteringIssues {
		t.Skip()
	}
	w.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := w.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := w.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := w.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := w.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := w.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := w.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := w.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := w.GetFee(feeBuilder); resp != float64(0.065) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.065), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := w.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	// Arrange
	w.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoWithAPIPermissionText
	// Act
	withdrawPermissions := w.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if w.APIKey != "" && w.APIKey != "Key" &&
		w.APISecret != "" && w.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	w.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var pair = pair.CurrencyPair{
		Delimiter:      "_",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.USD,
	}
	response, err := w.SubmitOrder(pair, exchange.Buy, exchange.Market, 1, 10, "hi")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	// Arrange
	w.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := w.CancelOrder(orderCancellation)

	// Assert
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if isWexEncounteringIssues {
		t.Skip()
	}
	// Arrange
	w.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	resp, err := w.CancelAllOrders(orderCancellation)

	// Assert
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestModifyOrder(t *testing.T) {
	_, err := w.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	w.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    symbol.LTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := w.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when no keys are set: %v", err)
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	w.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:       100,
		Currency:     symbol.BTC,
		Description:  "WITHDRAW IT ALL",
		BankAddress:  "123 Fake St",
		BankCity:     "Tarry Town",
		BankCountry:  "Hyrule",
		BankName:     "Federal Reserve Bank",
		WireCurrency: symbol.AUD,
		SwiftCode:    "Taylor",
	}

	_, err := w.WithdrawFiatFunds(withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', recieved: '%v'", common.ErrNotYetImplemented, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	w.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:       100,
		Currency:     symbol.BTC,
		Description:  "WITHDRAW IT ALL",
		BankAddress:  "123 Fake St",
		BankCity:     "Tarry Town",
		BankCountry:  "Hyrule",
		BankName:     "Federal Reserve Bank",
		WireCurrency: symbol.AUD,
		SwiftCode:    "Taylor",
	}

	_, err := w.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if err != common.ErrNotYetImplemented {
		t.Errorf("Expected '%v', recieved: '%v'", common.ErrNotYetImplemented, err)
	}
}
