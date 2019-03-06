package okcoin

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/okgroup"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	OKGroupExchange         = "OKCOIN International"
	canManipulateRealOrders = false
)

var o = OKCoin{}

func TestSetDefaults(t *testing.T) {
	if o.Name != OKGroupExchange {
		o.SetDefaults()
	}
	if o.GetName() != OKGroupExchange {
		t.Errorf("Test Failed - %v - SetDefaults() error", OKGroupExchange)
	}
	t.Parallel()
	TestSetup(t)
}

func TestSetRealOrderDefaults(t *testing.T) {
	TestSetDefaults(t)
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
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
		t.Errorf("Test Failed - %v Setup() init error", OKGroupExchange)
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

// setupWSConnection Connect to WS, but pass back error so test can handle it if needed
func setupWSConnection() error {
	o.Enabled = true
	err := o.WebsocketSetup(o.WsConnect,
		o.Name,
		true,
		o.WebsocketURL,
		o.WebsocketURL)
	if err != nil {
		return err
	}
	o.Websocket.SetWsStatusAndConnection(true)
	return nil
}

func connectToWs() error {
	err := o.Websocket.Connect()
	if err != nil {
		return err
	}
	return nil
}

// disconnectFromWS disconnect to WS, but pass back error so test can handle it if needed
func disconnectFromWS() error {
	err := o.Websocket.Shutdown()
	if err != nil {
		return err
	}
	return nil
}

// TestGetAccountCurrencies API endpoint test
func TestGetAccountCurrencies(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountCurrencies()
	testStandardErrorHandling(t, err)
}

// TestGetAccountWalletInformation API endpoint test
func TestGetAccountWalletInformation(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetAccountWalletInformation("")
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) == 0 {
		t.Error("No wallets returned")
	}
}

// TestGetAccountWalletInformationForCurrency API endpoint test
func TestGetAccountWalletInformationForCurrency(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetAccountWalletInformation(symbol.BTC)
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) != 1 {
		t.Errorf("Error receiving wallet information for currency: %v", symbol.BTC)
	}
}

// TestTransferAccountFunds API endpoint test
func TestTransferAccountFunds(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.TransferAccountFundsRequest{
		Amount:   10,
		Currency: symbol.BTC,
		From:     6,
		To:       1,
	}

	_, err := o.TransferAccountFunds(request)
	testStandardErrorHandling(t, err)
}

// TestBaseWithdraw API endpoint test
func TestAccountWithdrawRequest(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.AccountWithdrawRequest{
		Amount:      10,
		Currency:    symbol.BTC,
		TradePwd:    "1234",
		Destination: 4,
		ToAddress:   "1F5zVDgNjorJ51oGebSvNCrSAHpwGkUdDB",
		Fee:         1,
	}

	_, err := o.AccountWithdraw(request)
	testStandardErrorHandling(t, err)
}

// TestGetAccountWithdrawalFee API endpoint test
func TestGetAccountWithdrawalFee(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetAccountWithdrawalFee("")
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) == 0 {
		t.Error("Expected fees")
	}
}

// TestGetWithdrawalFeeForCurrency API endpoint test
func TestGetAccountWithdrawalFeeForCurrency(t *testing.T) {
	TestSetDefaults(t)
	resp, err := o.GetAccountWithdrawalFee(symbol.BTC)
	testStandardErrorHandling(t, err)

	if areTestAPIKeysSet() && len(resp) != 1 {
		t.Error("Expected fee for one currency")
	}
}

// TestGetAccountWithdrawalHistory API endpoint test
func TestGetAccountWithdrawalHistory(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountWithdrawalHistory("")
	testStandardErrorHandling(t, err)
}

// TestGetAccountWithdrawalHistoryForCurrency API endpoint test
func TestGetAccountWithdrawalHistoryForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountWithdrawalHistory(symbol.BTC)
	testStandardErrorHandling(t, err)
}

// TestGetAccountBillDetails API endpoint test
func TestGetAccountBillDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountBillDetails(okgroup.GetAccountBillDetailsRequest{})
	testStandardErrorHandling(t, err)
}

// TestGetAccountDepositAddressForCurrency API endpoint test
func TestGetAccountDepositAddressForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountDepositAddressForCurrency(symbol.BTC)
	testStandardErrorHandling(t, err)
}

// TestGetAccountDepositHistory API endpoint test
func TestGetAccountDepositHistory(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountDepositHistory("")
	testStandardErrorHandling(t, err)
}

// TestGetAccountDepositHistoryForCurrency API endpoint test
func TestGetAccountDepositHistoryForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAccountDepositHistory(symbol.BTC)
	testStandardErrorHandling(t, err)
}

// TestGetSpotTradingAccounts API endpoint test
func TestGetSpotTradingAccounts(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotTradingAccounts()
	testStandardErrorHandling(t, err)
}

// TestGetSpotTradingAccountsForCurrency API endpoint test
func TestGetSpotTradingAccountsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotTradingAccountForCurrency(symbol.BTC)
	testStandardErrorHandling(t, err)
}

// TestGetSpotBillDetailsForCurrency API endpoint test
func TestGetSpotBillDetailsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotBillDetailsForCurrencyRequest{
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

// TestPlaceSpotOrderLimit API endpoint test
func TestPlaceSpotOrderLimit(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "limit",
		Side:          "buy",
		MarginTrading: "1",
		Price:         "100",
		Size:          "100",
	}

	_, err := o.PlaceSpotOrder(request)
	testStandardErrorHandling(t, err)
}

// TestPlaceSpotOrderMarket API endpoint test
func TestPlaceSpotOrderMarket(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	_, err := o.PlaceSpotOrder(request)
	testStandardErrorHandling(t, err)
}

// TestPlaceMultipleSpotOrders API endpoint test
func TestPlaceMultipleSpotOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	order := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []okgroup.PlaceSpotOrderRequest{
		order,
	}

	_, errs := o.PlaceMultipleSpotOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

// TestPlaceMultipleSpotOrdersOverCurrencyLimits API logic test
func TestPlaceMultipleSpotOrdersOverCurrencyLimits(t *testing.T) {
	TestSetDefaults(t)
	order := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []okgroup.PlaceSpotOrderRequest{
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

// TestPlaceMultipleSpotOrdersOverPairLimits API logic test
func TestPlaceMultipleSpotOrdersOverPairLimits(t *testing.T) {
	TestSetDefaults(t)
	order := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []okgroup.PlaceSpotOrderRequest{
		order,
	}

	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.LTC, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.DOGE, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.XMR, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.BCH, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)

	_, errs := o.PlaceMultipleSpotOrders(request)
	if errs[0].Error() != "up to 4 trading pairs" {
		t.Error("Expecting an error when more than 4 trading pairs supplied", errs[0])
	}
}

// TestCancelSpotOrder API endpoint test
func TestCancelSpotOrder(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.CancelSpotOrderRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		OrderID:      1234,
	}

	_, err := o.CancelSpotOrder(request)
	testStandardErrorHandling(t, err)
}

// TestCancelMultipleSpotOrders API endpoint test
func TestCancelMultipleSpotOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4},
	}

	cancellations, err := o.CancelMultipleSpotOrders(request)
	testStandardErrorHandling(t, err)
	for _, cancellationsPerCurrency := range cancellations {
		for _, cancellation := range cancellationsPerCurrency {
			if !cancellation.Result {
				t.Error(cancellation.Error)
			}
		}
	}
}

// TestCancelMultipleSpotOrdersOverCurrencyLimits API logic test
func TestCancelMultipleSpotOrdersOverCurrencyLimits(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4, 5},
	}

	_, err := o.CancelMultipleSpotOrders(request)
	if err.Error() != "maximum 4 order cancellations for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", err)
	}
}

// TestGetSpotOrders API endpoint test
func TestGetSpotOrders(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Status:       "all",
	}
	_, err := o.GetSpotOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotOpenOrders API endpoint test
func TestGetSpotOpenOrders(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOpenOrdersRequest{}
	_, err := o.GetSpotOpenOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotOrder API endpoint test
func TestGetSpotOrder(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOrderRequest{
		OrderID:      -1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Upper().String(),
	}
	_, err := o.GetSpotOrder(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotTransactionDetails API endpoint test
func TestGetSpotTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotTransactionDetailsRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotTransactionDetails(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotTokenPairDetails API endpoint test
func TestGetSpotTokenPairDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotTokenPairDetails()
	testStandardErrorHandling(t, err)
}

// TestGetSpotOrderBook API endpoint test
func TestGetSpotOrderBook(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOrderBookRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotOrderBook(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotAllTokenPairsInformation API endpoint test
func TestGetSpotAllTokenPairsInformation(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotAllTokenPairsInformation()
	testStandardErrorHandling(t, err)
}

// TestGetSpotAllTokenPairsInformationForCurrency API endpoint test
func TestGetSpotAllTokenPairsInformationForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSpotAllTokenPairsInformationForCurrency(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

// TestGetSpotFilledOrdersInformation API endpoint test
func TestGetSpotFilledOrdersInformation(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotFilledOrdersInformationRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotFilledOrdersInformation(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotMarketData API endpoint test
func TestGetSpotMarketData(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotMarketDataRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Granularity:  604800,
	}
	_, err := o.GetSpotMarketData(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginTradingAccounts API endpoint test
func TestGetMarginTradingAccounts(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetMarginTradingAccounts()
	testStandardErrorHandling(t, err)
}

// TestGetMarginTradingAccountsForCurrency API endpoint test
func TestGetMarginTradingAccountsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetMarginTradingAccountsForCurrency(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

// TestGetMarginBillDetails API endpoint test
func TestGetMarginBillDetails(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetMarginBillDetailsRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Limit:        100,
	}
	_, err := o.GetMarginBillDetails(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginAccountSettings API endpoint test
func TestGetMarginAccountSettings(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetMarginAccountSettings("")
	testStandardErrorHandling(t, err)
}

// TestGetMarginAccountSettingsForCurrency API endpoint test
func TestGetMarginAccountSettingsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetMarginAccountSettings(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

// TestOpenMarginLoan API endpoint test
func TestOpenMarginLoan(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.OpenMarginLoanRequest{
		Amount:        100,
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		QuoteCurrency: symbol.USD,
	}

	_, err := o.OpenMarginLoan(request)
	testStandardErrorHandling(t, err)
}

// TestRepayMarginLoan API endpoint test
func TestRepayMarginLoan(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.RepayMarginLoanRequest{
		Amount:        100,
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		QuoteCurrency: symbol.USD,
		BorrowID:      1,
	}

	_, err := o.RepayMarginLoan(request)
	testStandardErrorHandling(t, err)
}

// TestPlaceMarginOrderLimit API endpoint test
func TestPlaceMarginOrderLimit(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "limit",
		Side:          "buy",
		MarginTrading: "2",
		Price:         "100",
		Size:          "100",
	}

	_, err := o.PlaceMarginOrder(request)
	testStandardErrorHandling(t, err)
}

// TestPlaceMarginOrderMarket API endpoint test
func TestPlaceMarginOrderMarket(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "2",
		Size:          "100",
		Notional:      "100",
	}

	_, err := o.PlaceMarginOrder(request)
	testStandardErrorHandling(t, err)
}

// TestPlaceMultipleMarginOrders API endpoint test
func TestPlaceMultipleMarginOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	order := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []okgroup.PlaceSpotOrderRequest{
		order,
	}

	_, errs := o.PlaceMultipleMarginOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

// TestPlaceMultipleMarginOrdersOverCurrencyLimits API logic test

func TestPlaceMultipleMarginOrdersOverCurrencyLimits(t *testing.T) {
	TestSetDefaults(t)
	order := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []okgroup.PlaceSpotOrderRequest{
		order,
		order,
		order,
		order,
		order,
	}

	_, errs := o.PlaceMultipleMarginOrders(request)
	if errs[0].Error() != "maximum 4 orders for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", errs[0])
	}
}

// TestPlaceMultipleMarginOrdersOverPairLimits API logic test
func TestPlaceMultipleMarginOrdersOverPairLimits(t *testing.T) {
	TestSetDefaults(t)
	order := okgroup.PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Type:          "market",
		Side:          "buy",
		MarginTrading: "1",
		Size:          "100",
		Notional:      "100",
	}

	request := []okgroup.PlaceSpotOrderRequest{
		order,
	}

	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.LTC, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.DOGE, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.XMR, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)
	order.InstrumentID = pair.NewCurrencyPairWithDelimiter(symbol.BCH, symbol.USD, "-").Pair().Lower().String()
	request = append(request, order)

	_, errs := o.PlaceMultipleMarginOrders(request)
	if errs[0].Error() != "up to 4 trading pairs" {
		t.Error("Expecting an error when more than 4 trading pairs supplied", errs[0])
	}
}

// TestCancelMarginOrder API endpoint test
func TestCancelMarginOrder(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.CancelSpotOrderRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		OrderID:      1234,
	}

	_, err := o.CancelMarginOrder(request)
	testStandardErrorHandling(t, err)
}

// TestCancelMultipleMarginOrders API endpoint test
func TestCancelMultipleMarginOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4},
	}

	_, errs := o.CancelMultipleMarginOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

// TestCancelMultipleMarginOrdersOverCurrencyLimits API logic test
func TestCancelMultipleMarginOrdersOverCurrencyLimits(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := okgroup.CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4, 5},
	}

	_, errs := o.CancelMultipleMarginOrders(request)
	if errs[0].Error() != "maximum 4 order cancellations for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", errs[0])
	}
}

// TestGetMarginOrders API endpoint test
func TestGetMarginOrders(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
		Status:       "all",
	}
	_, err := o.GetMarginOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginOpenOrders API endpoint test
func TestGetMarginOpenOrders(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOpenOrdersRequest{}
	_, err := o.GetMarginOpenOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginOrder API endpoint test
func TestGetMarginOrder(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotOrderRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Upper().String(),
	}
	_, err := o.GetMarginOrder(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginTransactionDetails API endpoint test
func TestGetMarginTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	request := okgroup.GetSpotTransactionDetailsRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USD, "-").Pair().Lower().String(),
	}
	_, err := o.GetMarginTransactionDetails(request)
	testStandardErrorHandling(t, err)
}

// Websocket tests ----------------------------------------------------------------------------------------------

// TestSubscribeToPewDiePie API endpoint test
func TestWSSetup(t *testing.T) {
	defer disconnectFromWS()
	TestSetDefaults(t)
	err := setupWSConnection()
	testStandardErrorHandling(t, err)
	if err != nil {
		t.Error(err)
	}
}

// TestSubscribeToPewDiePie API endpoint test
func TestSubscribeToPewDiePie(t *testing.T) {
	defer disconnectFromWS()
	TestSetDefaults(t)
	err := setupWSConnection()
	testStandardErrorHandling(t, err)
	err = o.WsSubscribeToChannel("Pewdiepie")
	testStandardErrorHandling(t, err)
	err = o.WsUnsubscribeToChannel("T-Series")
	testStandardErrorHandling(t, err)
}

// TestWsLogin API endpoint test
func TestWsLogin(t *testing.T) {
	defer disconnectFromWS()
	TestSetDefaults(t)
	err := setupWSConnection()
	testStandardErrorHandling(t, err)
	err = o.WsLogin()
	testStandardErrorHandling(t, err)
}

// TestOrderBookUpdateChecksumCalculator logic test
func TestOrderBookUpdateChecksumCalculator(t *testing.T) {
	o.Verbose = true
	original := `{"table":"spot/depth","action":"partial",
	"data":[{"instrument_id":"BTC-USDT",
	"asks":[["3725.1736","0.01469563",1],["3725.1995","0.002",1],["3725.3994","0.5433996",1],["3725.3995","0.004",1],["3725.4308","0.001",1],["3725.5867","0.125",1],["3725.5993","0.64230999",2],["3725.5994","0.59999986",1],["3725.5995","0.004",1],["3725.9995","0.004",1],["3726.0342","0.0402052",1],["3726.1126","1.85971361",1],["3726.1995","0.004",1],["3726.3995","0.004",1],["3726.5994","0.59999998",1],["3726.5995","0.004",1],["3726.6795","0.26833539",1],["3726.7995","0.004",1],["3726.9995","0.004",1],["3727.1149","0.4298",2],["3727.1528","0.08041043",1],["3727.2499","0.533",1],["3727.4879","0.015",1],["3727.5074","0.93989126",3],["3727.5262","0.27568946",1],["3727.5928","0.33246791",1],["3727.6074","1.07416144",1],["3727.7568","1.1",1],["3727.7997","0.53650951",1],["3728.0272","0.02599999",1],["3728.1172","1.288",1],["3728.3263","0.03932609",1],["3728.6969","0.027",1],["3728.8714","0.241",1],["3729","0.999",1],["3729.0075","0.001",1],["3729.0534","0.009",1],["3729.1385","0.201",1],["3729.359","0.052",1],["3729.4234","0.26845385",1],["3729.4297","0.07864757",1],["3729.5235","0.13422692",1],["3729.5613","0.01",1],["3729.9091","1.1",1],["3729.9402","0.02438212",1],["3730.1307","0.53691926",1],["3730.5042","0.2412313",1],["3730.7777","0.07",1],["3731.4758","0.015784",1],["3731.7216","1.4",1],["3731.8172","0.00539524",1],["3732.0846","0.0536",1],["3732.4044","0.27",1],["3732.6112","0.0107384",1],["3732.8198","8.0978328",1],["3733.3151","0.001055",1],["3733.97","0.345",1],["3733.9709","0.115",1],["3733.99","0.46",1],["3734","0.02876938",1],["3734.23","1.5",5],["3734.2864","0.53695077",1],["3735","0.15",1],["3735.5314","0.36185026",1],["3735.575","2.322",1],["3735.8318","6",1],["3736.1865","0.14999998",1],["3736.1866","0.12",2],["3736.2202","0.06",1],["3736.5724","1.8",1],["3736.6762","0.03",1],["3736.9769","0.06",1],["3737","10",1],["3737.1677","2.95607883",3],["3737.2991","0.50404083",1],["3737.3032","0.06",1],["3738.712","0.0537",1],["3740","0.04802962",3],["3740.7777","0.07",1],["3741.5968","1.02572022",1],["3741.7317","6.48",1],["3741.9309","0.00268043",1],["3742.01","0.16741356",1],["3742.169","0.06",1],["3742.5796","0.001",1],["3743.0946","18",1],["3743.3458","1.09058853",1],["3744.9","0.02",1],["3745","0.01",1],["3745.1592","0.001",1],["3747.7388","0.001",1],["3748","2.14653432",1],["3748.1316","6.3",1],["3749.29","0.00008016",1],["3750","0.3681107",1],["3750.3184","0.001",1],["3750.7777","0.07",1],["3751.7225","1.71003342",1],["3752.5534","4.878",1],["3752.898","0.001",1],["3753.1567","0.00267241",1],["3754.0445","0.06",1],["3754.5996","0.2684654",1],["3755.4776","0.001",1],["3756","0.4885",1],["3758.0572","0.001",1],["3760","0.45515728",1],["3760.6367","1.02527379",1],["3760.6368","0.001",1],["3760.7777","0.07",1],["3763.2164","0.001",1],["3764.4162","0.00266442",1],["3765.796","0.001",1],["3766.1704","0.15839999",1],["3768","0.1327",1],["3768.3756","0.001",1],["3769.4102","8.417",1],["3770.5443","0.13499933",1],["3770.7777","0.07",1],["3770.9552","0.001",1],["3771","0.02867241",1],["3773.5348","0.001",1],["3774","0.189613",2],["3774.8005","0.0011135",1],["3775.7094","0.00265645",1],["3776.1144","0.001",1],["3777","0.10628471",2],["3777.5703","0.08054662",1],["3777.8882","7.352",1],["3778.694","0.001",1],["3779.4791","0.12535065",1],["3780.7777","0.07",1],["3781.2736","0.001",1],["3782","0.06636968",1],["3783.8532","0.001",1],["3783.9233","0.561",1],["3785","1",1],["3786.267","12.176",1],["3786.4328","0.001",1],["3787.0365","0.0026485",1],["3788","0.808",2],["3789.0124","0.001",1],["3790","0.33457",1],["3790.7777","0.07",1],["3791.31","0.035",1],["3791.592","0.001",1],["3794.1716","0.001",1],["3795.5225","0.02",1],["3796","0.1",1],["3797.3963","1.31669165",1],["3798","3",1],["3798.3977","0.00264058",1],["3799","1.07",2],["3799.999","0.05",1],["3800","2.18314491",9],["3800.7777","0.07",1],["3803.1238","14.678",1],["3809.7928","0.00263268",1],["3810","1.5",1],["3810.7777","0.07",1],["3815","0.15",1],["3817","0.1",1],["3818.3819","0.00639869",1],["3820","14.71129448",5],["3821","1.05453573",2],["3821.2222","0.00262481",1],["3821.2257","0.07",1],["3822.2889","0.20968321",1],["3823","0.11950477",1],["3824","1",1],["3825","282.90879371",2],["3826","0.07",1],["3826.47","0.11902332",1],["3828","0.80804343",1],["3829.58","0.62104863",1],["3830","8.86079401",6],["3831.2257","0.07",1],["3832","0.01503",1],["3832.6859","0.00261696",1],["3834.427","0.0011",1],["3835","0.535",2],["3835.9579","0.04105526",1],["3838","0.1",1],["3839.58","0.62104863",1],["3840","5.55158672",3],["3844","0.1301",1],["3844.184","0.00260913",1],["3846","1.10139862",2],["3848","4.03438135",2],["3849","1",1],["3849.7866","0.3226041",1],["3850","2.55673517",7],["3850.3333","0.0202551",1],["3850.4005","0.08008079",1],["3851","0.06212366",1],["3855.7165","0.00260133",1],["3857","0.049",1],["3858","0.02830316",2],["3858.888","0.072927",1],["3859","1.95969614",1]],
	"bids":[["3724.736","0.45240613",2],["3724.7359","0.4163833",1],["3724.6149","0.00985064",1],["3724.5416","0.174",1],["3724.5289","0.4",1],["3724.5032","0.01036624",1],["3724.317","0.01576102",1],["3724.2054","0.05660039",1],["3724.2053","0.01658598",1],["3724.019","0.02521764",1],["3724.0132","0.538",1],["3723.9074","0.02653758",1],["3723.7211","0.04034823",1],["3723.6095","0.04246013",1],["3723.4271","0.03576489",1],["3723.4233","0.06455717",1],["3723.3942","0.7168353",1],["3723.3116","0.06793622",1],["3723.1023","1.42412798",2],["3723.0776","0.82706839",1],["3723.0287","0.01826911",1],["3722.9545","0.25640737",1],["3722.9211","0.01943454",1],["3722.8598","0.00709864",1],["3722.8359","0.007",1],["3722.7995","0.03404989",1],["3722.562","0.01135783",1],["3722.407","0.0106281",1],["3722.2963","0.026",1],["3722.2642","0.01817252",1],["3722.1922","0.04020287",1],["3722.1437","0.026",1],["3722.0927","0.40270618",2],["3721.9665","0.02907604",1],["3721.8399","0.19083695",1],["3721.6849","0.052",1],["3721.6687","0.04652167",1],["3721.6475","0.40268239",2],["3721.4005","0.03266516",1],["3721.384","0.056",1],["3721.3078","0.53690986",1],["3721.2005","0.025",1],["3721.1433","0.53702303",1],["3720.9457","0.01",1],["3720.4985","0.201",1],["3720.3595","0.0804232",1],["3720.3271","0.27",1],["3720.1395","0.03932019",1],["3719.9251","2",1],["3719.5346","0.001",1],["3719.2028","0.0532772",1],["3719.0224","0.07864039",1],["3719.0005","0.002",1],["3718.75","0.04",2],["3718.6005","0.002",1],["3718.2005","0.002",1],["3718.0313","0.01075838",1],["3717.9567","0.01",1],["3717.847","0.0536",1],["3717.6345","0.4",1],["3717.6005","0.002",1],["3717.4005","0.002",1],["3717.3855","2.14776632",1],["3716.9676","0.2412313",1],["3716.6054","0.4035941",1],["3716.6005","0.004",1],["3715.524","2.14776632",1],["3715.5235","0.23592813",1],["3715.4714","0.43",1],["3715","1",1],["3714.6992","0.2",1],["3714.4306","0.01",1],["3714.1","1.5",1],["3713.3848","0.001",1],["3712.3583","1.82398086",1],["3712.2539","4.41521904",1],["3712","0.48145334",1],["3711.941","0.36183685",1],["3711.7512","0.06",1],["3711.5745","0.5043708",1],["3711.571","0.06",1],["3711.3295","2",1],["3711.3216","0.0537",1],["3710.3658","0.53691926",1],["3710.2681","1.098",1],["3710.1645","5.999",1],["3709.5193","0.95",1],["3708.8853","0.84482482",1],["3708.8034","0.15",1],["3708.8033","0.03",1],["3708.56","0.3",1],["3708.2674","0.88548033",1],["3708.23","0.3",1],["3708.19","0.3",1],["3708.17","0.6",2],["3707.9","0.3",1],["3707.7858","2",1],["3707.5","3.32",1],["3706.8961","0.77275267",1],["3706.7991","1.035",1],["3706.6546","1.8",1],["3706.5975","0.00814839",1],["3705.6942","1",1],["3702.7854","1.915",1],["3701.5136","1.01605148",1],["3701.3995","0.001",1],["3700.7777","0.07",1],["3700.0319","5.3",1],["3700","2.55429563",7],["3698.9055","0.06",1],["3698.5018","18",1],["3698","0.0271",1],["3697.0997","0.04",1],["3697","0.15586658",2],["3695.9055","0.06",1],["3695.79","0.13556019",1],["3695.0096","5.9",1],["3695","0.15",1],["3694.7528","0.26850834",1],["3692.9055","0.06",1],["3691.0001","0.08127877",1],["3691","1",1],["3690","0.98776967",6],["3689.9055","0.06",1],["3689.0008","2",1],["3688.2351","0.55239286",1],["3688.235","6.2",1],["3688","3.53150146",3],["3686.9055","0.06",1],["3685","1",1],["3683.9055","0.06",1],["3683","0.2",1],["3682","0.00604001",1],["3680.71","0.00506967",1],["3680.0001","0.00916",2],["3680","6.32790092",10],["3678.3746","0.01",1],["3678","0.15732536",2],["3677.9055","0.06",1],["3676.1","0.0015",1],["3675.1454","0.00272098",1],["3675","0.10553855",2],["3674.9055","0.06",1],["3674.6529","20",1],["3673","0.381",2],["3672.3219","0.0109",1],["3672","1.001",1],["3671.9055","0.06",1],["3671.86","1.06774074",1],["3670.0001","0.01205177",2],["3670","0.76576791",8],["3668.9055","0.06",1],["3668","0.32121046",1],["3667.7725","0.0406451",1],["3666.6692","0.00654389",1],["3666.6666","0.00635484",1],["3666.1717","0.12544606",1],["3666","1.61573531",6],["3665.9055","0.06",1],["3665.595","0.04051184",1],["3665.2453","0.0109",1],["3665","0.74374679",2],["3664.12","0.00545832",2],["3664.0351","0.00203817",1],["3663","0.12968291",1],["3660.8599","0.024173",1],["3660.5587","0.01121498",1],["3660.0001","0.01208196",2],["3660","33.74959941",10],["3659.9996","0.02",1],["3659","0.44007989",3],["3658.5558","0.1015",1],["3658.1688","0.0109",1],["3658","0.80036817",3],["3656","0.301",2],["3655.971","0.13",1],["3655","0.2396528",3],["3653.1276","0.00273738",1],["3652","0.32638281",1],["3651.0922","0.011",1],["3651.04","1.09557824",1],["3651","0.11599767",2],["3650.6263","0.20112031",1],["3650.09","13.07",1],["3650.01","0.003",1],["3650.0001","0.01348219",2],["3650","77.76608707",35],["3649","0.0311316",1],["3648","1.6447",1],["3647.4819","9",1],["3646","0.001",1],["3644.0157","0.0137",1],["3643","4.48672926",2],["3642.1683","0.00274561",1],["3641.2","0.001",1],["3640.3402","0.00256332",1],["3640.0001","0.01351648",2],["3640","0.57724533",5],["3639","0.70794078",1],["3638.9055","0.05686658",1]],
	"timestamp":"2019-03-05T01:16:17.815Z","checksum":1206267559}]}`
	update := `{"table":"spot/depth","action":"update","data":[{"instrument_id":"BTC-USDT","asks":[["3725.3867","0.125",1],["3725.3992","0.6",1],["3725.3993","0.08698038",1],["3725.3994","0",0],["3725.5867","0",0],["3725.5993","0",0],["3725.5994","0",0],["3726.5994","0",0],["3726.6696","0.1999",1],["3727.4745","0.015",1],["3727.4879","0",0],["3728.0883","0.04",1],["3728.5691","0.053",1],["3729.0534","0",0],["3729.2441","0.026",1],["3734.21","0.3",1],["3734.23","1.8",6],["3858.888","0",0],["3859","0",0]],"bids":[["3724.736","0.05240613",1],["3724.5289","0",0],["3724.2054","0",0],["3723.1023","0.74702798",1],["3723.0776","1.50416839",2],["3722.407","0",0],["3720.9457","0",0],["3719.2028","0",0],["3716.6054","0",0],["3710.3658","0",0],["3638","0.5",1],["3636.9391","0.0137",1],["3636","0.003",1],["3635.8888","1.04202103",1],["3633.85","0.04951211",1],["3633.6139","0.03616108",1],["3633","2.61264584",2]],"timestamp":"2019-03-05T01:16:18.309Z","checksum":-1211017441}]}`
	t.Log(update)
	var dataResponse okgroup.WebsocketDataResponse
	err := common.JSONDecode([]byte(original), &dataResponse)
	if err != nil {
		t.Error(err)
	}
	calculatedChecksum := o.WsCalculateOrderBookChecksum(dataResponse.Data[0])

	if calculatedChecksum != dataResponse.Data[0].Checksum {
		t.Errorf("Expected %v, Receieved %v", dataResponse.Data[0].Checksum, calculatedChecksum)
	}
}

// TestOrderBookPartialChecksumCalculator logic test
func TestOrderBookPartialChecksumCalculator(t *testing.T) {
	orderbookPartialJSON := `{"table":"spot/depth","action":"partial","data":[{"instrument_id":"EOS-USDT","asks":[["3.5196","0.1077",1],["3.5198","21.71",1],["3.5199","51.1805",1],["3.5208","75.09",1],["3.521","196.3333",1],["3.5213","0.1",1],["3.5218","39.276",2],["3.5219","395.6334",1],["3.522","27.956",1],["3.5222","404.9595",1],["3.5225","300",1],["3.5227","143.5442",2],["3.523","42.4746",1],["3.5231","852.64",2],["3.5235","34.9602",1],["3.5237","442.0918",2],["3.5238","352.8404",2],["3.5239","341.6759",2],["3.524","84.9493",1],["3.5241","148.4882",1],["3.5242","261.64",1],["3.5243","142.045",1],["3.5246","10",1],["3.5247","284.0788",1],["3.5248","720",1],["3.5249","89.2518",2],["3.5251","1201.8965",2],["3.5254","426.2938",1],["3.5255","213.0863",1],["3.5257","568.1576",1],["3.5258","0.3",1],["3.5259","34.4602",1],["3.526","0.1",1],["3.5263","850.771",1],["3.5265","5.9",1],["3.5268","10.5064",2],["3.5272","1136.8965",1],["3.5274","255.1481",1],["3.5276","29.5374",1],["3.5278","50",1],["3.5282","284.1797",1],["3.5283","1136.8965",1],["3.5284","0.4275",1],["3.5285","100",1],["3.5292","90.9",1],["3.5298","0.2",1],["3.5303","568.1576",1],["3.5305","279.9999",1],["3.532","0.409",1],["3.5321","568.1576",1],["3.5326","6016.8756",1],["3.5328","4.9849",1],["3.533","92.88",2],["3.5343","1200.2383",2],["3.5344","100",1],["3.535","359.7047",1],["3.5354","100",1],["3.5355","100",1],["3.5356","10",1],["3.5358","200",2],["3.5362","435.139",1],["3.5365","2152",1],["3.5366","284.1756",1],["3.5367","568.4644",1],["3.5369","33.9878",1],["3.537","337.1191",2],["3.5373","0.4045",1],["3.5383","1136.7188",1],["3.5386","12.1614",1],["3.5387","90.89",1],["3.54","4.54",1],["3.5423","90.8",1],["3.5436","0.1",1],["3.5454","853.4156",1],["3.5468","142.0656",1],["3.5491","0.0008",1],["3.55","14478.8206",6],["3.5537","21521",1],["3.5555","11.53",1],["3.5573","50.6001",1],["3.5599","4591.4221",1],["3.56","1227.0002",4],["3.5603","2670",1],["3.5608","58.6638",1],["3.5613","0.1",1],["3.5621","45.9473",1],["3.57","2141.7274",3],["3.5712","2956.9816",1],["3.5717","27.9978",1],["3.5718","0.9285",1],["3.5739","299.73",1],["3.5761","864",1],["3.579","22.5225",1],["3.5791","38.26",2],["3.58","7618.4634",5],["3.5801","457.2184",1],["3.582","24.5",1],["3.5822","1572.6425",1],["3.5845","14.1438",1],["3.585","527.169",1],["3.5865","20",1],["3.5867","4490",1],["3.5876","39.0493",1],["3.5879","392.9083",1],["3.5888","436.42",2],["3.5896","50",1],["3.59","2608.9128",8],["3.5913","19.5246",1],["3.5938","7082",1],["3.597","0.1",1],["3.5979","399",1],["3.5995","315.1509",1],["3.5999","2566.2648",1],["3.6","18511.2292",35],["3.603","22.3379",2],["3.605","499.5",1],["3.6055","100",1],["3.6058","499.5",1],["3.608","1021.1485",1],["3.61","11755.4596",13],["3.611","42.8571",1],["3.6131","6690",1],["3.6157","19.5247",1],["3.618","2500",1],["3.6197","525.7146",1],["3.6198","0.4455",1],["3.62","6440.6295",8],["3.6219","0.4175",1],["3.6237","168",1],["3.6265","0.1001",1],["3.628","64.9345",1],["3.63","4435.4985",6],["3.6308","1.7815",1],["3.6331","0.1",1],["3.6338","355.527",2],["3.6358","50",1],["3.6363","2074.7096",1],["3.6376","4000",1],["3.6396","11090",1],["3.6399","0.4055",1],["3.64","4161.9805",4],["3.6437","117.6524",1],["3.648","190",1],["3.6488","200",1],["3.65","11740.5045",25],["3.6512","0.1",1],["3.6521","728",1],["3.6555","100",1],["3.6598","36.6914",1],["3.66","4331.2148",6],["3.6638","200",1],["3.6673","100",1],["3.6679","38",1],["3.6688","2",1],["3.6695","0.1",1],["3.67","7984.698",6],["3.672","300",1],["3.6777","257.8247",1],["3.6789","393.4217",2],["3.68","9202.3222",11],["3.6818","500",1],["3.6823","299.7",1],["3.6839","422.3748",1],["3.685","100",1],["3.6878","0.1",1],["3.6888","72.0958",2],["3.6889","2876",1],["3.689","28",1],["3.6891","28",1],["3.6892","28",1],["3.6895","28",1],["3.6898","28",1],["3.69","643.96",7],["3.6908","118",2],["3.691","28",1],["3.6916","28",1],["3.6918","28",1],["3.6926","28",1],["3.6928","28",1],["3.6932","28",1],["3.6933","200",1],["3.6935","28",1],["3.6936","28",1],["3.6938","28",1],["3.694","28",1],["3.698","1498.5",1],["3.6988","2014.2004",2],["3.7","21904.2689",22],["3.7029","71.95",1],["3.704","3690.1362",1],["3.7055","100",1],["3.7063","0.1",1],["3.71","4421.3468",4],["3.719","17.3491",1],["3.72","1304.5995",3],["3.7211","10",1],["3.7248","0.1",1],["3.725","1900",1],["3.73","31.1785",2],["3.7375","38",1]],"bids":[["3.5182","151.5343",6],["3.5181","0.3691",1],["3.518","271.3967",2],["3.5179","257.8352",1],["3.5178","12.3811",1],["3.5173","34.1921",2],["3.5171","1013.8256",2],["3.517","272.1119",2],["3.5168","395.3376",1],["3.5166","317.1756",2],["3.5165","348.302",3],["3.5164","142.0414",1],["3.5163","96.8933",2],["3.516","600.1034",3],["3.5159","27.481",1],["3.5158","27.33",1],["3.5157","583.1898",2],["3.5156","24.6819",2],["3.5154","25",1],["3.5153","0.429",1],["3.5152","453.9204",3],["3.5151","2131.592",4],["3.515","335",3],["3.5149","37.1586",1],["3.5147","41.6759",1],["3.5146","54.569",1],["3.5145","70.3515",1],["3.5143","68.206",3],["3.5142","359.4538",2],["3.5139","45.4123",2],["3.5137","71.673",2],["3.5136","25",1],["3.5135","300",1],["3.5134","442.57",2],["3.5132","83.3518",1],["3.513","1245.2529",3],["3.5127","20",1],["3.512","284.1353",1],["3.5119","1136.8319",1],["3.5113","56.9351",1],["3.5111","588.1898",2],["3.5109","255.0946",1],["3.5105","48.65",1],["3.5103","50.2",1],["3.5098","720",1],["3.5096","148.95",1],["3.5094","570.5758",2],["3.509","2.386",1],["3.5089","0.4065",1],["3.5087","282.3859",2],["3.5086","145.036",2],["3.5084","2.386",1],["3.5082","90.98",1],["3.5081","2.386",1],["3.5079","2.386",1],["3.5078","857.6229",2],["3.5075","2.386",1],["3.5074","284.1877",1],["3.5073","100",1],["3.5071","100",1],["3.507","768.4159",3],["3.5069","313.0863",2],["3.5068","426.2938",1],["3.5066","568.3594",1],["3.5063","1136.6865",1],["3.5059","0.3",1],["3.5054","9.9999",1],["3.5053","0.2",1],["3.5051","392.428",1],["3.505","13.79",1],["3.5048","99.5497",2],["3.5047","78.5331",2],["3.5046","2153",1],["3.5041","5983.999",1],["3.5037","668.5682",1],["3.5036","160.5948",1],["3.5024","534.8075",1],["3.5014","28.5604",1],["3.5011","91",1],["3.5","1058.8771",2],["3.4997","50.2",1],["3.4985","3430.0414",1],["3.4949","232.0591",1],["3.4942","21521",1],["3.493","2",1],["3.4928","2",1],["3.4925","0.44",1],["3.4917","142.0656",1],["3.49","2051.8826",4],["3.488","280.7459",1],["3.4852","643.4038",1],["3.4851","86.0807",1],["3.485","213.2436",1],["3.484","0.1",1],["3.4811","144.3399",1],["3.4808","89",1],["3.4803","12.1999",1],["3.4801","2390",1],["3.48","930.8453",9],["3.4791","310",1],["3.4768","206",1],["3.4767","0.9415",1],["3.4754","1.4387",1],["3.4728","20",1],["3.4701","1219.2873",1],["3.47","1904.3139",7],["3.468","0.4035",1],["3.4667","0.1",1],["3.4666","3020.0101",1],["3.465","10",1],["3.464","0.4485",1],["3.462","2119.6556",1],["3.46","1305.6113",8],["3.4589","8.0228",1],["3.457","100",1],["3.456","70.3859",2],["3.4538","20",1],["3.4536","4323.9486",2],["3.4531","827.0427",1],["3.4528","0.439",1],["3.4522","8.0381",1],["3.4513","441.1873",1],["3.4512","50.707",1],["3.451","87.0902",1],["3.4509","200",1],["3.4506","100",1],["3.4505","86.4045",2],["3.45","12409.4595",28],["3.4494","0.5365",2],["3.449","10761",1],["3.4482","8.0476",1],["3.4469","0.449",1],["3.445","2000",1],["3.4427","14",1],["3.4421","100",1],["3.4416","8.0631",1],["3.4404","1",1],["3.44","4580.733",11],["3.4388","1868.2085",1],["3.438","937.7246",2],["3.4367","1500",1],["3.4366","62",1],["3.436","29.8743",1],["3.4356","25.4801",1],["3.4349","4.3086",1],["3.4343","43.2402",1],["3.433","2.0688",1],["3.4322","2.7335",2],["3.432","93.3233",1],["3.4302","328.8301",2],["3.43","4440.8158",11],["3.4288","754.574",2],["3.4283","125.7043",2],["3.428","744.3154",2],["3.4273","5460",1],["3.4258","50",1],["3.4255","109.005",1],["3.4248","100",1],["3.4241","129.2048",2],["3.4233","5.3598",1],["3.4228","4498.866",1],["3.4222","3.5435",1],["3.4217","404.3252",2],["3.4211","1000",1],["3.4208","31",1],["3.42","1834.024",9],["3.4175","300",1],["3.4162","400",1],["3.4152","0.1",1],["3.4151","4.3336",1],["3.415","1.5974",1],["3.414","1146",1],["3.4134","306.4246",1],["3.4129","7.5556",1],["3.4111","198.5188",1],["3.4109","500",1],["3.4106","4305",1],["3.41","2150.7635",13],["3.4085","4.342",1],["3.4054","5.6985",1],["3.4019","5.438",1],["3.4015","1010.846",1],["3.4009","8610",1],["3.4005","1.9122",1],["3.4004","1",1],["3.4","27081.1806",67],["3.3955","3.2682",1],["3.3953","5.4486",1],["3.3937","1591.3805",1],["3.39","3221.4155",8],["3.3899","3.2736",1],["3.3888","1500",2],["3.3887","5.4592",1],["3.385","117.0969",2],["3.3821","5.4699",1],["3.382","100.0529",1],["3.3818","172.0164",1],["3.3815","165.6288",1],["3.381","887.3115",1],["3.3808","100",1]],"timestamp":"2019-03-04T00:15:04.155Z","checksum":-2036653089}]}`
	var dataResponse okgroup.WebsocketDataResponse
	err := common.JSONDecode([]byte(orderbookPartialJSON), &dataResponse)
	if err != nil {
		t.Error(err)
	}
	calculatedChecksum := o.WsCalculateOrderBookChecksum(dataResponse.Data[0])

	if calculatedChecksum != dataResponse.Data[0].Checksum {
		t.Errorf("Expected %v, Receieved %v", dataResponse.Data[0].Checksum, calculatedChecksum)
	}
}

// Function tests ----------------------------------------------------------------------------------------------
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
	if resp, err := o.GetFee(feeBuilder); resp != float64(0.0005) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.0005), resp)
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
	if resp, err := o.GetFee(feeBuilder); resp != float64(0.2) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(0.2), resp)
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
	if resp, err := o.GetFee(feeBuilder); resp != float64(15) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Received: %f", float64(15), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	o.SetDefaults()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.WithdrawFiatViaWebsiteOnlyText
	withdrawPermissions := o.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []pair.CurrencyPair{pair.NewCurrencyPair(symbol.LTC, symbol.BTC)},
	}

	_, err := o.GetActiveOrders(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)

	var getOrdersRequest = exchange.GetOrdersRequest{
		OrderType:  exchange.AnyOrderType,
		Currencies: []pair.CurrencyPair{pair.NewCurrencyPair(symbol.LTC, symbol.BTC)},
	}

	_, err := o.GetOrderHistory(getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
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

	resp, err := o.CancelAllOrders(orderCancellation)

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

func TestModifyOrder(t *testing.T) {
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
