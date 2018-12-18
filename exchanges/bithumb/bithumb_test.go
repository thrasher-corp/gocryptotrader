package bithumb

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey              = ""
	testAPISecret           = ""
	canManipulateRealOrders = false
)

var b Bithumb

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bitConfig, err := cfg.GetExchangeConfig("Bithumb")
	if err != nil {
		t.Error("Test Failed - Bithumb Setup() init error")
	}

	bitConfig.AuthenticatedAPISupport = true
	bitConfig.APIKey = testAPIKey
	bitConfig.APISecret = testAPISecret

	b.Setup(bitConfig)
}

// TestAreAPIKeysSet is part of a pre-commit hook to prevent commiting your API keys
func TestAreAPIKeysSet(t *testing.T) {
	var errMsg string
	// Local keys
	if testAPIKey != "" && testAPIKey != "Key" {
		errMsg += "Cannot commit populated testAPIKey. "
	}
	if testAPISecret != "" && testAPISecret != "Secret" {
		errMsg += "Cannot commit populated testAPISecret. "
	}
	if canManipulateRealOrders {
		errMsg += "Cannot commit with canManipulateRealOrders enabled."
	}
	//configtest.json keys
	b.SetDefaults()
	TestSetup(t)
	if b.APIKey != "" && b.APIKey != "Key" {
		errMsg += "API key present in testconfig.json"
	}
	if b.APISecret != "" && b.APISecret != "Key" {
		errMsg += "API secret key present in testconfig.json"
	}
	if len(errMsg) > 0 {
		t.Error(errMsg)
	}
}

func TestGetTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetTradablePairs()
	if err != nil {
		t.Error("test failed - Bithumb GetTradablePairs() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("btc")
	if err != nil {
		t.Error("test failed - Bithumb GetTicker() error", err)
	}
}

func TestGetAllTickers(t *testing.T) {
	t.Parallel()
	_, err := b.GetAllTickers()
	if err != nil {
		t.Error("test failed - Bithumb GetAllTickers() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook("btc")
	if err != nil {
		t.Error("test failed - Bithumb GetOrderBook() error", err)
	}
}

func TestGetTransactionHistory(t *testing.T) {
	t.Parallel()
	_, err := b.GetTransactionHistory("btc")
	if err != nil {
		t.Error("test failed - Bithumb GetTransactionHistory() error", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.GetAccountBalance("BTC")
	if err == nil {
		t.Error("test failed - Bithumb GetAccountBalance() error", err)
	}
}

func TestGetWalletAddress(t *testing.T) {
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	t.Parallel()
	_, err := b.GetWalletAddress("")
	if err == nil {
		t.Error("test failed - Bithumb GetWalletAddress() error", err)
	}
}

func TestGetLastTransaction(t *testing.T) {
	t.Parallel()
	_, err := b.GetLastTransaction()
	if err == nil {
		t.Error("test failed - Bithumb GetLastTransaction() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrders("1337", "bid", "100", "", "BTC")
	if err == nil {
		t.Error("test failed - Bithumb GetOrders() error", err)
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserTransactions()
	if err == nil {
		t.Error("test failed - Bithumb GetUserTransactions() error", err)
	}
}

func TestPlaceTrade(t *testing.T) {
	t.Parallel()
	_, err := b.PlaceTrade("btc", "bid", 0, 0)
	if err == nil {
		t.Error("test failed - Bithumb PlaceTrade() error", err)
	}
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderDetails("1337", "bid", "btc")
	if err == nil {
		t.Error("test failed - Bithumb GetOrderDetails() error", err)
	}
}

func TestCancelTrade(t *testing.T) {
	t.Parallel()
	_, err := b.CancelTrade("", "", "")
	if err == nil {
		t.Error("test failed - Bithumb CancelTrade() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := b.WithdrawCrypto("LQxiDhKU7idKiWQhx4ALKYkBx8xKEQVxJR", "", "ltc", 0)
	if err == nil {
		t.Error("test failed - Bithumb WithdrawCrypto() error", err)
	}
}

func TestRequestKRWDepositDetails(t *testing.T) {
	t.Parallel()
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}
	_, err := b.RequestKRWDepositDetails()
	if err == nil {
		t.Error("test failed - Bithumb RequestKRWDepositDetails() error", err)
	}
}

func TestRequestKRWWithdraw(t *testing.T) {
	t.Parallel()
	_, err := b.RequestKRWWithdraw("102_bank", "1337", 1000)
	if err == nil {
		t.Error("test failed - Bithumb RequestKRWWithdraw() error", err)
	}
}

func TestMarketBuyOrder(t *testing.T) {
	t.Parallel()
	_, err := b.MarketBuyOrder("btc", 0)
	if err == nil {
		t.Error("test failed - Bithumb MarketBuyOrder() error", err)
	}
}

func TestMarketSellOrder(t *testing.T) {
	t.Parallel()
	_, err := b.MarketSellOrder("btc", 0)
	if err == nil {
		t.Error("test failed - Bithumb MarketSellOrder() error", err)
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
	b.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0015) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0015), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(1500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(1500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0015) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.0015), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.AutoWithdrawFiatText
	// Act
	withdrawPermissions := b.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Recieved: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func isRealOrderTestEnabled() bool {
	if b.APIKey == "" || b.APISecret == "" ||
		b.APIKey == "Key" || b.APISecret == "Secret" ||
		!canManipulateRealOrders {
		return false
	}
	return true
}

func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.LTC,
	}
	response, err := b.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 1, "clientId")
	if err != nil || !response.IsOrderPlaced {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	err := b.CancelOrder(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	b.SetDefaults()
	TestSetup(t)

	if !isRealOrderTestEnabled() {
		t.Skip()
	}

	currencyPair := pair.NewCurrencyPair(symbol.LTC, symbol.BTC)

	var orderCancellation = exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	// Act
	resp, err := b.CancelAllOrders(orderCancellation)

	// Assert
	if err != nil {
		t.Errorf("Could not cancel order: %s", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if testAPIKey != "" || testAPISecret != "" {
		_, err := b.GetAccountInfo()
		if err != nil {
			t.Error("test failed - Bithumb GetAccountInfo() error", err)
		}
	} else {
		_, err := b.GetAccountInfo()
		if err == nil {
			t.Error("test failed - Bithumb GetAccountInfo() error")
		}
	}
}
