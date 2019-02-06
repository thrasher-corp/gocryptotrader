package okgroup

import (
	"fmt"
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
	OKGroupExchange         = "OKEX"
	canManipulateRealOrders = false
)

var o = OKGroup{
	APIURL:       fmt.Sprintf("%v%v", "https://www.okex.com/", OkGroupAPIPath),
	APIVersion:   "/v3/",
	ExchangeName: OKGroupExchange,
	WebsocketURL: "wss://real.okex.com:10440/websocket/okexapi",
}

func TestSetDefaults(t *testing.T) {
	if o.Name != "OKEX" {
		o.SetDefaults()
	}
	if o.GetName() != "OKEX" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
	t.Parallel()
	TestSetup(t)
}

func TestSetup(t *testing.T) {
	if o.APIKey == apiKey && o.APISecret == apiSecret &&
		o.ClientID == passphrase {
		return
	}
	o.ExchangeName = OKGroupExchange
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")

	okexConfig, err := cfg.GetExchangeConfig(OKGroupExchange)
	if err != nil {
		t.Error("Test Failed - Okex Setup() init error")
	}

	okexConfig.AuthenticatedAPISupport = true
	okexConfig.APIKey = apiKey
	okexConfig.APISecret = apiSecret
	okexConfig.ClientID = passphrase
	okexConfig.Verbose = true
	o.Setup(okexConfig)
}

func areTestAPIKeysSet() bool {
	if o.APIKey != "" && o.APIKey != "Key" &&
		o.APISecret != "" && o.APISecret != "Secret" {
		return true
	}
	return false
}

func testStandardErrorHandling(t *testing.T, err error) {
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Encountered error: %v", err)
	}
}

func TestGetSpotInstruments(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotInstruments()
	if err != nil {
		t.Errorf("Test failed - okex GetSpotInstruments() failed: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetCurrencies()
	testStandardErrorHandling(t, err)
}

func TestGetWalletInformation(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetWalletInformation("")
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) == 0 {
		t.Error("No wallets returned")
	}
}

func TestGetWalletInformationForCurrency(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetWalletInformation(symbol.BTC)
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) != 1 {
		t.Errorf("Error recieving wallet information for currency: %v", symbol.BTC)
	}
}

func TestTransferFunds(t *testing.T) {
	TestSetDefaults(t)
	request := FundTransferRequest{
		Amount:   10,
		Currency: symbol.BTC,
		From:     6,
		To:       1,
	}

	_, err := o.TransferFunds(request)
	testStandardErrorHandling(t, err)
}

func TestBaseWithdraw(t *testing.T) {
	TestSetDefaults(t)
	request := WithdrawRequest{
		Amount:      10,
		Currency:    symbol.BTC,
		TradePwd:    "1234",
		Destination: 4,
		ToAddress:   "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Fee:         1,
	}

	_, err := o.Withdraw(request)
	testStandardErrorHandling(t, err)
}

func TestGetWithdrawalFee(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetWithdrawalFee("")
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) == 0 {
		t.Error("Expected fees")
	}
}

func TestGetWithdrawalFeeForCurrency(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetWithdrawalFee(symbol.BTC)
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) != 1 {
		t.Error("Expected fee for one currency")
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetWithdrawalHistory("")
	testStandardErrorHandling(t, err)
}

func TestGetWithdrawalHistoryForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetWithdrawalHistory(symbol.BTC)
	testStandardErrorHandling(t, err)
}

func TestGetBillDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetBillDetails(GetBillDetailsRequest{})
	testStandardErrorHandling(t, err)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetDepositAddressForCurrency(symbol.BTC)
	testStandardErrorHandling(t, err)
}

func TestGetDepositHistory(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetDepositHistory("")
	testStandardErrorHandling(t, err)
}

func TestGetDepositHistoryForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetDepositHistory(symbol.BTC)
	testStandardErrorHandling(t, err)
}

func TestGetSpotTradingAccounts(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotTradingAccounts()
	testStandardErrorHandling(t, err)
}

func TestGetSpotTradingAccountsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotTradingAccountForCurrency(symbol.BTC)
	testStandardErrorHandling(t, err)
}

func TestGetSpotBillDetailsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotBillDetailsForCurrencyRequest{
		Currency: symbol.BTC,
		Limit:    100,
	}

	_, err := o.GetSpotBillDetailsForCurrency(request)
	testStandardErrorHandling(t, err)

	request.Limit = -1
	_, err = o.GetSpotBillDetailsForCurrency(request)
	if areTestAPIKeysSet() && err == nil {
		t.Errorf("Expecting an error when invalid request sent")
	}

}

func TestPlaceSpotOrderLimit(t *testing.T) {
	TestSetDefaults(t)
	request := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Type:          "limit",
		Side:          "buy",
		MarginTrading: "1",
		Price:         "100",
		Size:          "100",
	}

	_, err := o.PlaceSpotOrder(request)
	testStandardErrorHandling(t, err)
}

func TestPlaceSpotOrderMarket(t *testing.T) {
	TestSetDefaults(t)
	request := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	_, err := o.PlaceSpotOrder(request)
	testStandardErrorHandling(t, err)
}

func TestPlaceMultipleSpotOrders(t *testing.T) {
	TestSetDefaults(t)
	order := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []PlaceSpotOrderRequest{
		order,
	}

	_, errs := o.PlaceMultipleSpotOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

func TestPlaceMultipleSpotOrdersOverCurrencyLimits(t *testing.T) {
	TestSetDefaults(t)
	order := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []PlaceSpotOrderRequest{
		order,
		order,
		order,
		order,
		order,
	}

	_, errs := o.PlaceMultipleSpotOrders(request)
	if errs[0].Error() != "maximum 4 orders for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", errs[0])
	}
}

func TestPlaceMultipleSpotOrdersOverPairLimits(t *testing.T) {
	TestSetDefaults(t)
	order := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []PlaceSpotOrderRequest{
		order,
	}

	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.LTC, symbol.USDT, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.DOGE, symbol.USDT, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.XMR, symbol.USDT, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.BCH, symbol.USDT, "-").Pair().Lower().String()
	request = append(request, order)

	_, errs := o.PlaceMultipleSpotOrders(request)
	if errs[0].Error() != "up to 4 trading pairs" {
		t.Error("Expecting an error when more than 4 trading pairs supplied", errs[0])
	}
}

func TestCancelSpotOrder(t *testing.T) {
	TestSetDefaults(t)
	request := CancelSpotOrderRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderID:      1234,
	}

	_, err := o.CancelSpotOrder(request)
	testStandardErrorHandling(t, err)
}

func TestCancelMultipleSpotOrders(t *testing.T) {
	TestSetDefaults(t)
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4},
	}

	_, errs := o.CancelMultipleSpotOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

func TestCancelMultipleSpotOrdersOverCurrencyLimits(t *testing.T) {
	TestSetDefaults(t)
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4, 5},
	}

	_, errs := o.CancelMultipleSpotOrders(request)
	if errs[0].Error() != "maximum 4 order cancellations for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", errs[0])
	}
}

func TestGetSpotOrders(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Status:       "all",
	}
	_, err := o.GetSpotOrders(request)
	testStandardErrorHandling(t, err)
}

func TestGetSpotOpenOrders(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOpenOrdersRequest{}
	_, err := o.GetSpotOpenOrders(request)
	testStandardErrorHandling(t, err)
}

func TestGetSpotOrder(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOrderRequest{
		OrderID:      -1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Upper().String(),
	}
	_, err := o.GetSpotOrder(request)
	testStandardErrorHandling(t, err)
}

func TestGetSpotTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotTransactionDetailsRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotTransactionDetails(request)
	testStandardErrorHandling(t, err)
}

func TestGetSpotTokenPairDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotTokenPairDetails()
	testStandardErrorHandling(t, err)
}

func TestGetSpotOrderBook(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOrderBookRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotOrderBook(request)
	testStandardErrorHandling(t, err)
}

func TestGetSpotAllTokenPairsInformation(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotAllTokenPairsInformation()
	testStandardErrorHandling(t, err)
}

func TestGetSpotAllTokenPairsInformationForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotAllTokenPairsInformationForCurrency(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

func TestGetSpotFilledOrdersInformation(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotFilledOrdersInformationRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotFilledOrdersInformation(request)
	testStandardErrorHandling(t, err)
}

func TestGetSpotMarketData(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotMarketDataRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Granularity:  604800,
	}
	_, err := o.GetSpotMarketData(request)
	testStandardErrorHandling(t, err)
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
	o.SetDefaults()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := o.GetFee(feeBuilder); resp != float64(0.0015) || err != nil {
		t.Error(err)
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0015), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := o.GetFee(feeBuilder); resp != float64(1500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(1500), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := o.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := o.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := o.GetFee(feeBuilder); resp != float64(0.001) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.001), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.FirstCurrency = "hello"
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := o.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := o.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := o.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.CurrencyItem = symbol.USD
	if resp, err := o.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	// Arrange
	o.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	// Act
	withdrawPermissions := o.FormatWithdrawPermissions()
	// Assert
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------

func TestSubmitOrder(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var p = pair.CurrencyPair{
		Delimiter:      "",
		FirstCurrency:  symbol.BTC,
		SecondCurrency: symbol.EUR,
	}
	response, err := o.SubmitOrder(p, exchange.BuyOrderSide, exchange.MarketOrderType, 1, 10, "hi")
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	// Arrange
	o.SetDefaults()
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

	err := o.CancelOrder(orderCancellation)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	// Arrange
	o.SetDefaults()
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
	resp, err := o.CancelAllOrders(orderCancellation)

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
	_, err := o.GetAccountInfo()
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestModifyOrder(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)
	_, err := o.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

func TestWithdraw(t *testing.T) {
	o.SetDefaults()
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

	_, err := o.WithdrawCryptocurrencyFunds(withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := o.WithdrawFiatFunds(withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = exchange.WithdrawRequest{}

	_, err := o.WithdrawFiatFundsToInternationalBank(withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}
