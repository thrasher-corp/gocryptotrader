package okgroup

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

func TestSetDefaults(t *testing.T) {
	Okex.SetDefaults()
	if Okex.GetName() != "OKEX" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	okexConfig, err := cfg.GetExchangeConfig(Okex.Name)
	if err != nil {
		t.Error("Test Failed - Okex Setup() init error")
	}

	okexConfig.AuthenticatedAPISupport = true
	okexConfig.APIKey = apiKey
	okexConfig.APISecret = apiSecret
	okexConfig.ClientID = passphrase
	okexConfig.Verbose = true
	Okex.Setup(okexConfig)
}

func TestGetSpotInstruments(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)

	_, err := Okex.GetSpotInstruments()
	if err != nil {
		t.Errorf("Test failed - okex GetSpotInstruments() failed: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)

	_, err := Okex.GetCurrencies()
	if err != nil {
		t.Errorf("Test failed - okex GetSpotInstruments() failed: %s", err)
	}
}

func TestGetContractPrice(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractPrice("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractPrice() error", err)
	}
	_, err = Okex.GetContractPrice("btc_bla", "123525")
	if err == nil {
		t.Error("Test failed - okex GetContractPrice() error", err)
	}
	_, err = Okex.GetContractPrice("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractPrice() error", err)
	}
}

func TestGetContractMarketDepth(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractMarketDepth("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractMarketDepth() error", err)
	}
	_, err = Okex.GetContractMarketDepth("btc_bla", "123525")
	if err == nil {
		t.Error("Test failed - okex GetContractMarketDepth() error", err)
	}
	_, err = Okex.GetContractMarketDepth("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractMarketDepth() error", err)
	}
}

func TestGetContractTradeHistory(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractTradeHistory("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
	_, err = Okex.GetContractTradeHistory("btc_bla", "123525")
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
	_, err = Okex.GetContractTradeHistory("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
}

func TestGetContractIndexPrice(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractIndexPrice("btc_usd")
	if err != nil {
		t.Error("Test failed - okex GetContractIndexPrice() error", err)
	}
	_, err = Okex.GetContractIndexPrice("lol123")
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
}

func TestGetContractExchangeRate(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractExchangeRate()
	if err != nil {
		t.Error("Test failed - okex GetContractExchangeRate() error", err)
	}
}

func TestGetContractCandlestickData(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractCandlestickData("btc_usd", "1min", "this_week", 1, 2)
	if err != nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
	_, err = Okex.GetContractCandlestickData("btc_bla", "1min", "this_week", 1, 2)
	if err == nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
	_, err = Okex.GetContractCandlestickData("btc_usd", "min", "this_week", 1, 2)
	if err == nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
	_, err = Okex.GetContractCandlestickData("btc_usd", "1min", "this_wok", 1, 2)
	if err == nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
}

func TestGetContractHoldingsNumber(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, _, err := Okex.GetContractHoldingsNumber("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractHoldingsNumber() error", err)
	}
	_, _, err = Okex.GetContractHoldingsNumber("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractHoldingsNumber() error", err)
	}
	_, _, err = Okex.GetContractHoldingsNumber("btc_usd", "this_bla")
	if err == nil {
		t.Error("Test failed - okex GetContractHoldingsNumber() error", err)
	}
}

func TestGetContractlimit(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetContractlimit("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractlimit() error", err)
	}
	_, err = Okex.GetContractlimit("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractlimit() error", err)
	}
	_, err = Okex.GetContractlimit("btc_usd", "this_bla")
	if err == nil {
		t.Error("Test failed - okex GetContractlimit() error", err)
	}
}

func TestGetContractUserInfo(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	err := Okex.GetContractUserInfo()
	if err == nil {
		t.Error("Test failed - okex GetContractUserInfo() error", err)
	}
}

func TestGetContractPosition(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	err := Okex.GetContractPosition("btc_usd", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractPosition() error", err)
	}
}

func TestPlaceContractOrders(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.PlaceContractOrders("btc_usd", "this_week", "1", 10, 1, 1, true)
	if err == nil {
		t.Error("Test failed - okex PlaceContractOrders() error", err)
	}
}

func TestGetContractFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	err := Okex.GetContractFuturesTradeHistory("btc_usd", "1972-01-01", 0)
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetLatestSpotPrice("ltc_btc")
	if err != nil {
		t.Error("Test failed - okex GetLatestSpotPrice() error", err)
	}
}

func TestGetSpotTicker(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetSpotTicker("ltc_btc")
	if err != nil {
		t.Error("Test failed - okex GetSpotTicker() error", err)
	}
}

func TestGetSpotMarketDepth(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetSpotMarketDepth(ActualSpotDepthRequestParams{
		Symbol: "eth_btc",
		Size:   2,
	})
	if err != nil {
		t.Error("Test failed - okex GetSpotMarketDepth() error", err)
	}
}

func TestGetSpotRecentTrades(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.GetSpotRecentTrades(ActualSpotTradeHistoryRequestParams{
		Symbol: "ltc_btc",
		Since:  0,
	})
	if err != nil {
		t.Error("Test failed - okex GetSpotRecentTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)
	arg := KlinesRequestParams{
		Symbol: "ltc_btc",
		Type:   TimeIntervalFiveMinutes,
		Size:   100,
	}
	_, err := Okex.GetSpotKline(arg)
	if err != nil {
		t.Error("Test failed - okex GetSpotCandleStick() error", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)

	_, err := Okex.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: "ltc_btc",
		Amount: 1.1,
		Price:  10.1,
		Type:   SpotNewOrderRequestTypeBuy,
	})
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Expected no errors, recieved '%v'", err)
	}
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected an error when no keys are set")
	}
}

func TestSpotCancelOrder(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)

	_, err := Okex.SpotCancelOrder("ltc_btc", 519158961)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Expected no errors, recieved '%v'", err)
	}
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected an error when no keys are set")
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	Okex.SetDefaults()
	TestSetup(t)

	_, err := Okex.GetUserInfo()
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Expected no errors, recieved '%v'", err)
	}
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expected an error when no keys are set")
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
	Okex.SetDefaults()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0.0015) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0015), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(1500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(1500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := Okex.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	Okex.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	// Act
	withdrawPermissions := Okex.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	if Okex.APIKey != "" && Okex.APIKey != "Key" &&
		Okex.APISecret != "" && Okex.APISecret != "Secret" {
		return true
	}
	return false
}

func TestSubmitOrder(t *testing.T) {
	Okex.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.EUR,
	}
	response, err := Okex.SubmitOrder(p, exchange.Buy, exchange.Market, 1, 10, "hi")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	Okex.SetDefaults()
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
	err := Okex.CancelOrder(orderCancellation)

	// Assert
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	Okex.SetDefaults()
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
	resp, err := Okex.CancelAllOrders(orderCancellation)

	// Assert
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.OrderStatus) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.OrderStatus))
	}
}

func TestGetAccountInfo(t *testing.T) {
	_, err := Okex.GetAccountInfo()
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestModifyOrder(t *testing.T) {
	Okex.SetDefaults()
	TestSetup(t)
	_, err := Okex.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	Okex.SetDefaults()
	TestSetup(t)
	var withdrawCryptoRequest = exchange.WithdrawRequest{
		Amount:        100,
		Currency:      "btc_usd",
		Address:       "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Description:   "WITHDRAW IT ALL",
		TradePassword: "Password",
		FeeAmount:     1,
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := Okex.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	Okex.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := Okex.WithdrawFiatFunds(withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', recieved: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	Okex.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := Okex.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', recieved: '%v'", common.ErrFunctionNotSupported, err)
	}
}
