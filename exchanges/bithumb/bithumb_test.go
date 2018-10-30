package bithumb

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply your own keys here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
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

	bitConfig.API.AuthenticatedSupport = true
	bitConfig.API.Credentials.Key = apiKey
	bitConfig.API.Credentials.Secret = apiSecret

	b.Setup(bitConfig)
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
	if apiKey == "" || apiSecret == "" {
		t.Skip()
	}

	_, err := b.GetAccountBalance("BTC")
	if err == nil {
		t.Error("test failed - Bithumb GetAccountBalance() error", err)
	}
}

func TestGetWalletAddress(t *testing.T) {
	if apiKey == "" || apiSecret == "" {
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
	if apiKey == "" || apiSecret == "" {
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

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	b.GetFeeByType(feeBuilder)
	if apiKey == "" || apiSecret == "" {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestGetFee(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	var feeBuilder = setFeeBuilder()

	// CryptocurrencyTradeFee Basic
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0025), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(2500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(2500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.0025) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0025), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.HKD
	if resp, err := b.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	b.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.AutoWithdrawFiatText

	withdrawPermissions := b.FormatWithdrawPermissions()

	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
		OrderSide: exchange.SellOrderSide,
	}

	_, err := b.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType: exchange.AnyOrderType,
	}

	_, err := b.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = currency.Pair{
		Delimiter: "",
		Base:      currency.BTC,
		Quote:     currency.LTC,
	}
	response, err := b.SubmitOrder(p, exchange.BuyOrderSide, exchange.LimitOrderType, 1, 1, "clientId")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	err := b.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel order: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)

	var orderCancellation = &exchange.OrderCancellation{
		OrderID:       "1",
		WalletAddress: "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		AccountID:     "1",
		CurrencyPair:  currencyPair,
	}

	resp, err := b.CancelAllOrders(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel order: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	if apiKey != "" || apiSecret != "" {
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

func TestModifyOrder(t *testing.T) {
	curr := currency.NewPairFromString("BTCUSD")
	_, err := b.ModifyOrder(&exchange.ModifyOrder{OrderID: "1337",
		Price:        100,
		Amount:       1000,
		OrderSide:    exchange.SellOrderSide,
		CurrencyPair: curr})
	if err == nil {
		t.Error("Test Failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:      100,
		Currency:    currency.BTC,
		Address:     "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description: "WITHDRAW IT ALL",
	}

	_, err := b.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{
		Amount:                   100,
		Currency:                 currency.KRW,
		Description:              "WITHDRAW IT ALL",
		BankAccountName:          "Satoshi Nakamoto",
		BankAccountNumber:        12345,
		BankCode:                 123,
		BankAddress:              "123 Fake St",
		BankCity:                 "Tarry Town",
		BankCountry:              "Hyrule",
		BankName:                 "Federal Reserve Bank",
		WireCurrency:             currency.KRW.String(),
		SwiftCode:                "Taylor",
		RequiresIntermediaryBank: false,
		IsExpressWire:            false,
	}

	_, err := b.WithdrawFiatFunds(&withdrawFiatRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	b.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := b.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	if apiKey != "" && apiSecret != "" {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err != nil {
			t.Error("Test Failed - GetDepositAddress() error", err)
		}
	} else {
		_, err := b.GetDepositAddress(currency.BTC, "")
		if err == nil {
			t.Error("Test Failed - GetDepositAddress() error cannot be nil")
		}
	}
}
