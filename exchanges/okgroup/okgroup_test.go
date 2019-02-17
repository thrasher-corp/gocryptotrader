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
	if o.Name != OKGroupExchange {
		o.SetDefaults()
	}
	if o.GetName() != OKGroupExchange {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
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
		t.Errorf("Error recieving wallet information for currency: %v", symbol.BTC)
	}
}

// TestTransferAccountFunds API endpoint test
func TestTransferAccountFunds(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := TransferAccountFundsRequest{
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
	request := AccountWithdrawRequest{
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
	_, err := o.GetAccountBillDetails(GetAccountBillDetailsRequest{})
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

// TestPlaceSpotOrderLimit API endpoint test
func TestPlaceSpotOrderLimit(t *testing.T) {
	TestSetRealOrderDefaults(t)
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

// TestPlaceSpotOrderMarket API endpoint test
func TestPlaceSpotOrderMarket(t *testing.T) {
	TestSetRealOrderDefaults(t)
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

// TestPlaceMultipleSpotOrders API endpoint test
func TestPlaceMultipleSpotOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
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

// TestPlaceMultipleSpotOrdersOverCurrencyLimits API logic test
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

// TestPlaceMultipleSpotOrdersOverPairLimits API logic test
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

// TestCancelSpotOrder API endpoint test
func TestCancelSpotOrder(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := CancelSpotOrderRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderID:      1234,
	}

	_, err := o.CancelSpotOrder(request)
	testStandardErrorHandling(t, err)
}

// TestCancelMultipleSpotOrders API endpoint test
func TestCancelMultipleSpotOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4},
	}

	_, errs := o.CancelMultipleSpotOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

// TestCancelMultipleSpotOrdersOverCurrencyLimits API logic test
func TestCancelMultipleSpotOrdersOverCurrencyLimits(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderIDs:     []int64{1, 2, 3, 4, 5},
	}

	_, errs := o.CancelMultipleSpotOrders(request)
	if errs[0].Error() != "maximum 4 order cancellations for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", errs[0])
	}
}

// TestGetSpotOrders API endpoint test
func TestGetSpotOrders(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Status:       "all",
	}
	_, err := o.GetSpotOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotOpenOrders API endpoint test
func TestGetSpotOpenOrders(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOpenOrdersRequest{}
	_, err := o.GetSpotOpenOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotOrder API endpoint test
func TestGetSpotOrder(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOrderRequest{
		OrderID:      -1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Upper().String(),
	}
	_, err := o.GetSpotOrder(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotTransactionDetails API endpoint test
func TestGetSpotTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotTransactionDetailsRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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
	request := GetSpotOrderBookRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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
	_, err := o.GetSpotAllTokenPairsInformationForCurrency(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

// TestGetSpotFilledOrdersInformation API endpoint test
func TestGetSpotFilledOrdersInformation(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotFilledOrdersInformationRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
	}
	_, err := o.GetSpotFilledOrdersInformation(request)
	testStandardErrorHandling(t, err)
}

// TestGetSpotMarketData API endpoint test
func TestGetSpotMarketData(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotMarketDataRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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
	_, err := o.GetMarginTradingAccountsForCurrency(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

// TestGetMarginBillDetails API endpoint test
func TestGetMarginBillDetails(t *testing.T) {
	TestSetDefaults(t)
	request := GetAccountBillDetailsRequest{
		Currency: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Limit:    100,
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
	_, err := o.GetMarginAccountSettings(pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String())
	testStandardErrorHandling(t, err)
}

// TestOpenMarginLoan API endpoint test
func TestOpenMarginLoan(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := OpenMarginLoanRequest{
		Amount:        100,
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		QuoteCurrency: symbol.USDT,
	}

	_, err := o.OpenMarginLoan(request)
	testStandardErrorHandling(t, err)
}

// TestRepayMarginLoan API endpoint test
func TestRepayMarginLoan(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := RepayMarginLoanRequest{
		Amount:        100,
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		QuoteCurrency: symbol.USDT,
		BorrowID:      1,
	}

	_, err := o.RepayMarginLoan(request)
	testStandardErrorHandling(t, err)
}

// TestPlaceMarginOrderLimit API endpoint test
func TestPlaceMarginOrderLimit(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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
	request := PlaceSpotOrderRequest{
		InstrumentID:  pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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

	_, errs := o.PlaceMultipleMarginOrders(request)
	if len(errs) > 0 {
		testStandardErrorHandling(t, errs[0])
	}
}

// TestPlaceMultipleMarginOrdersOverCurrencyLimits API logic test

func TestPlaceMultipleMarginOrdersOverCurrencyLimits(t *testing.T) {
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

	_, errs := o.PlaceMultipleMarginOrders(request)
	if errs[0].Error() != "maximum 4 orders for each pair" {
		t.Error("Expecting an error when more than 4 orders for a pair supplied", errs[0])
	}
}

// TestPlaceMultipleMarginOrdersOverPairLimits API logic test
func TestPlaceMultipleMarginOrdersOverPairLimits(t *testing.T) {
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

	_, errs := o.PlaceMultipleMarginOrders(request)
	if errs[0].Error() != "up to 4 trading pairs" {
		t.Error("Expecting an error when more than 4 trading pairs supplied", errs[0])
	}
}

// TestCancelMarginOrder API endpoint test
func TestCancelMarginOrder(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := CancelSpotOrderRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		OrderID:      1234,
	}

	_, err := o.CancelMarginOrder(request)
	testStandardErrorHandling(t, err)
}

// TestCancelMultipleMarginOrders API endpoint test
func TestCancelMultipleMarginOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
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
	request := GetSpotOrdersRequest{
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
		Status:       "all",
	}
	_, err := o.GetMarginOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginOpenOrders API endpoint test
func TestGetMarginOpenOrders(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOpenOrdersRequest{}
	_, err := o.GetMarginOpenOrders(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginOrder API endpoint test
func TestGetMarginOrder(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotOrderRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Upper().String(),
	}
	_, err := o.GetMarginOrder(request)
	testStandardErrorHandling(t, err)
}

// TestGetMarginTransactionDetails API endpoint test
func TestGetMarginTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	request := GetSpotTransactionDetailsRequest{
		OrderID:      1234,
		InstrumentID: pair.NewCurrencyPairWithDelimiter(symbol.BTC, symbol.USDT, "-").Pair().Lower().String(),
	}
	_, err := o.GetMarginTransactionDetails(request)
	testStandardErrorHandling(t, err)
}

var genericFutureInstrumentID string

// getFutureInstrumentID Future contract ids are date based without an easy way to calculate the closest valid date
// This retrieves the value and stores it if running all tests so only one call is made
func getFutureInstrumentID() string {
	if genericFutureInstrumentID != "" {
		return genericFutureInstrumentID
	}
	resp, err := o.GetFuturesContractInformation()
	if err != nil {
		// No error handling here because we're not testing this
		return err.Error()
	}
	genericFutureInstrumentID = resp[0].InstrumentID
	return genericFutureInstrumentID
}

// TestGetFuturesPostions API endpoint test
func TestGetFuturesPostions(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesPostions()
	testStandardErrorHandling(t, err)
}

// TestGetFuturesPostionsForCurrency API endpoint test
func TestGetFuturesPostionsForCurrency(t *testing.T) {
	TestSetDefaults(t)
	currencyContract := getFutureInstrumentID()
	_, err := o.GetFuturesPostionsForCurrency(currencyContract)
	testStandardErrorHandling(t, err)
}

// TestGetFuturesAccountOfAllCurrencies API endpoint test
func TestGetFuturesAccountOfAllCurrencies(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesAccountOfAllCurrencies()
	testStandardErrorHandling(t, err)
}

// TestGetFuturesAccountOfACurrency API endpoint test
func TestGetFuturesAccountOfACurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesAccountOfACurrency(symbol.BTC)
	testStandardErrorHandling(t, err)
}

// TestGetFuturesLeverage API endpoint test
func TestGetFuturesLeverage(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesLeverage(symbol.BTC)
	testStandardErrorHandling(t, err)
}

// TestSetFuturesLeverage API endpoint test
func TestSetFuturesLeverage(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := SetFuturesLeverageRequest{
		Currency:     symbol.BTC,
		InstrumentID: getFutureInstrumentID(),
		Leverage:     10,
		Direction:    "Long",
	}
	_, err := o.SetFuturesLeverage(request)
	testStandardErrorHandling(t, err)
}

// TestGetFuturesBillDetails API endpoint test
func TestGetFuturesBillDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesBillDetails(GetSpotBillDetailsForCurrencyRequest{
		Currency: symbol.BTC,
	})
	testStandardErrorHandling(t, err)
}

// TestPlaceFuturesOrder API endpoint test
func TestPlaceFuturesOrder(t *testing.T) {
	TestSetRealOrderDefaults(t)
	_, err := o.PlaceFuturesOrder(PlaceFuturesOrderRequest{
		InstrumentID: getFutureInstrumentID(),
		Leverage:     10,
		Type:         1,
		Size:         2,
		Price:        432.11,
		ClientOid:    "12233456",
	})
	testStandardErrorHandling(t, err)
}

// TestPlaceFuturesOrderBatch API endpoint test
func TestPlaceFuturesOrderBatch(t *testing.T) {
	TestSetRealOrderDefaults(t)
	_, err := o.PlaceFuturesOrderBatch(PlaceFuturesOrderBatchRequest{
		InstrumentID: getFutureInstrumentID(),
		Leverage:     10,
		OrdersData: []PlaceFuturesOrderBatchRequestDetails{
			PlaceFuturesOrderBatchRequestDetails{
				ClientOid:  "1",
				MatchPrice: "0",
				Price:      "100",
				Size:       "100",
				Type:       "1",
			},
		},
	})
	testStandardErrorHandling(t, err)
}

// TestCancelFuturesOrder API endpoint test
func TestCancelFuturesOrder(t *testing.T) {
	TestSetRealOrderDefaults(t)
	_, err := o.CancelFuturesOrder(CancelFuturesOrderRequest{
		InstrumentID: getFutureInstrumentID(),
		OrderID:      "1",
	})
	testStandardErrorHandling(t, err)
}

// TestCancelMultipleSpotOrders API endpoint test
func TestCancelMultipleFuturesOrders(t *testing.T) {
	TestSetRealOrderDefaults(t)
	request := CancelMultipleSpotOrdersRequest{
		InstrumentID: getFutureInstrumentID(),
		OrderIDs:     []int64{1, 2, 3, 4},
	}

	_, err := o.CancelFuturesOrderBatch(request)
	testStandardErrorHandling(t, err)
}

// TestGetFuturesOrderList API endpoint test
func TestGetFuturesOrderList(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesOrderList(GetFuturesOrdersListRequest{
		InstrumentID: getFutureInstrumentID(),
		Status:       6,
	})
	testStandardErrorHandling(t, err)
}

// TestGetFuturesOrderDetails API endpoint test
func TestGetFuturesOrderDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesOrderDetails(GetFuturesOrderDetailsRequest{
		InstrumentID: getFutureInstrumentID(),
		OrderID:      1,
	})
	testStandardErrorHandling(t, err)
}

// TestGetFuturesTransactionDetails API endpoint test
func TestGetFuturesTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesTransactionDetails(GetFuturesTransactionDetailsRequest{
		InstrumentID: getFutureInstrumentID(),
		OrderID:      1,
	})
	testStandardErrorHandling(t, err)
}

// TestGetFuturesContractInformation API endpoint test
func TestGetFuturesContractInformation(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesContractInformation()
	if err != nil {

	}
}

// TestGetFuturesContractInformation API endpoint test
func TestGetFuturesOrderBook(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesOrderBook(GetFuturesOrderBookRequest{
		InstrumentID: getFutureInstrumentID(),
		Size:         10,
	})
	testStandardErrorHandling(t, err)
}

// TestGetAllFuturesTokenInfo API endpoint test
func TestGetAllFuturesTokenInfo(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAllFuturesTokenInfo()
	testStandardErrorHandling(t, err)
}

// TestGetAllFuturesTokenInfo API endpoint test
func TestGetFuturesTokenInfoForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesTokenInfoForCurrency(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesFilledOrder API endpoint test
func TestGetFuturesFilledOrder(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesFilledOrder(GetFuturesFilledOrderRequest{
		InstrumentID: getFutureInstrumentID(),
	})
	testStandardErrorHandling(t, err)
}

// TestGetFuturesHoldAmount API endpoint test
func TestGetFuturesHoldAmount(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesHoldAmount(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesHoldAmount API endpoint test
func TestGetFuturesIndices(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesIndices(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesHoldAmount API endpoint test
func TestGetFuturesExchangeRates(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesExchangeRates()
	if err != nil {
		t.Errorf("Encountered error: %v", err)
	}
}

// TestGetFuturesHoldAmount API endpoint test
func TestGetFuturesEstimatedDeliveryPrice(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesEstimatedDeliveryPrice(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesOpenInterests API endpoint test
func TestGetFuturesOpenInterests(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesOpenInterests(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesOpenInterests API endpoint test
func TestGetFuturesCurrentPriceLimit(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesCurrentPriceLimit(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesCurrentMarkPrice API endpoint test
func TestGetFuturesCurrentMarkPrice(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesCurrentMarkPrice(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// TestGetFuturesForceLiquidatedOrders API endpoint test
func TestGetFuturesForceLiquidatedOrders(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesForceLiquidatedOrders(GetFuturesForceLiquidatedOrdersRequest{
		InstrumentID: getFutureInstrumentID(),
		Status:       "1",
	})
	testStandardErrorHandling(t, err)
}

// TestGetFuturesTagPrice API endpoint test
func TestGetFuturesTagPrice(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetFuturesTagPrice(getFutureInstrumentID())
	testStandardErrorHandling(t, err)
}

// -------------------------------------------------------------------------------------------------------

// TestGetSwapPostions API endpoint test
func TestGetSwapPostions(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapPostions()
	testStandardErrorHandling(t, err)
}

func TestGetSwapPostionsForContract(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapPostionsForContract(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapAccountOfAllCurrency API endpoint test
func TestGetSwapAccountOfAllCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapAccountOfAllCurrency()
	testStandardErrorHandling(t, err)
}

// TestGetSwapAccountSettingsOfAContract API endpoint test
func TestGetSwapAccountSettingsOfAContract(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapAccountSettingsOfAContract(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestSetSwapLeverageLevelOfAContract API endpoint test
func TestSetSwapLeverageLevelOfAContract(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.SetSwapLeverageLevelOfAContract(SetSwapLeverageLevelOfAContractRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Leverage:     10,
		Side:         1,
	})

	testStandardErrorHandling(t, err)
}

// TestGetSwapAccountSettingsOfAContract API endpoint test
func TestGetSwapBillDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapBillDetails(GetSpotBillDetailsForCurrencyRequest{
		Currency: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Limit:    100,
	})
	testStandardErrorHandling(t, err)
}

// TestPlaceSwapOrder API endpoint test
func TestPlaceSwapOrder(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.PlaceSwapOrder(PlaceSwapOrderRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Size:         1,
		Type:         1,
		Price:        1,
	})
	testStandardErrorHandling(t, err)
}

// TestPlaceMultipleSwapOrders API endpoint test
func TestPlaceMultipleSwapOrders(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.PlaceMultipleSwapOrders(PlaceMultipleSwapOrdersRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Leverage:     10,
		OrdersData: []PlaceMultipleSwapOrderData{
			PlaceMultipleSwapOrderData{
				ClientOID:  "hello",
				MatchPrice: "0",
				Price:      "10",
				Size:       "1",
				Type:       "1",
			}, PlaceMultipleSwapOrderData{
				ClientOID:  "hello2",
				MatchPrice: "0",
				Price:      "10",
				Size:       "1",
				Type:       "1",
			}},
	})
	testStandardErrorHandling(t, err)
}

// TestCancelSwapOrder API endpoint test
func TestCancelSwapOrder(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.CancelSwapOrder(CancelSwapOrderRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		OrderID:      "64-2a-26132f931-3",
	})
	testStandardErrorHandling(t, err)
}

// TestCancelMultipleSwapOrders API endpoint test
func TestCancelMultipleSwapOrders(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.CancelMultipleSwapOrders(CancelMultipleSwapOrdersRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		OrderIDs:     []int64{1, 2, 3, 4},
	})
	testStandardErrorHandling(t, err)
}

// TestGetSwapOrderList API endpoint test
func TestGetSwapOrderList(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapOrderList(GetSwapOrderListRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Status:       6,
	})
	testStandardErrorHandling(t, err)
}

// TestGetSwapOrderDetails API endpoint test
func TestGetSwapOrderDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapOrderDetails(GetSwapOrderDetailsRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		OrderID:      "64-2a-26132f931-3",
	})
	testStandardErrorHandling(t, err)
}

// TestGetSwapTransactionDetails API endpoint test
func TestGetSwapTransactionDetails(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapTransactionDetails(GetSwapTransactionDetailsRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		OrderID:      "64-2a-26132f931-3",
	})
	testStandardErrorHandling(t, err)
}

// TestGetSwapContractInformation API endpoint test
func TestGetSwapContractInformation(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapContractInformation()
	testStandardErrorHandling(t, err)
}

// TestGetSwapOrderBook API endpoint test
func TestGetSwapOrderBook(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapOrderBook(GetSwapOrderBookRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Size:         200,
	})

	testStandardErrorHandling(t, err)
}

// TestGetAllSwapTokensInformation API endpoint test
func TestGetAllSwapTokensInformation(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetAllSwapTokensInformation()
	testStandardErrorHandling(t, err)
}

// TestGetSwapTokensInformationForCurrency API endpoint test
func TestGetSwapTokensInformationForCurrency(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapTokensInformationForCurrency(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapFilledOrdersData API endpoint test
func TestGetSwapFilledOrdersData(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapFilledOrdersData(&GetSwapFilledOrdersDataRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Limit:        100,
	})
	testStandardErrorHandling(t, err)
}

// TestGetSwapMarketData API endpoint test
func TestGetSwapMarketData(t *testing.T) {
	TestSetDefaults(t)
	request := GetSwapMarketDataRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Granularity:  604800,
	}
	_, err := o.GetSwapMarketData(request)
	testStandardErrorHandling(t, err)
}

// TestGetSwapIndeces API endpoint test
func TestGetSwapIndeces(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapIndeces(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapExchangeRates API endpoint test
func TestGetSwapExchangeRates(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapExchangeRates()
	testStandardErrorHandling(t, err)
}

// TestGetSwapOpenInterest API endpoint test
func TestGetSwapOpenInterest(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapOpenInterest(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapCurrentPriceLimits API endpoint test
func TestGetSwapCurrentPriceLimits(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapCurrentPriceLimits(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapForceLiquidatedOrders API endpoint test
func TestGetSwapForceLiquidatedOrders(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapForceLiquidatedOrders(GetSwapForceLiquidatedOrdersRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Status:       "0",
	})
	testStandardErrorHandling(t, err)
}

// TestGetSwapOnHoldAmountForOpenOrders API endpoint test
func TestGetSwapOnHoldAmountForOpenOrders(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapOnHoldAmountForOpenOrders(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapNextSettlementTime API endpoint test
func TestGetSwapNextSettlementTime(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapNextSettlementTime(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapMarkPrice API endpoint test
func TestGetSwapMarkPrice(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapMarkPrice(fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD))
	testStandardErrorHandling(t, err)
}

// TestGetSwapFundingRateHistory API endpoint test
func TestGetSwapFundingRateHistory(t *testing.T) {
	TestSetDefaults(t)
	_, err := o.GetSwapFundingRateHistory(GetSwapFundingRateHistoryRequest{
		InstrumentID: fmt.Sprintf("%v-%v-SWAP", symbol.BTC, symbol.USD),
		Limit:        100,
	})
	testStandardErrorHandling(t, err)
}

// -------------------------------------------------------------------------------------------------------

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

// TestGetFee fee calcuation test
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

// TestFormatWithdrawPermissions helper test
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

// TestSubmitOrder Wrapper test
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

// TestCancelExchangeOrder Wrapper test
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

// TestCancelAllExchangeOrders Wrapper test
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

// TestGetAccountInfo Wrapper test
func TestGetAccountInfo(t *testing.T) {
	_, err := o.GetAccountInfo()
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

// TestModifyOrder Wrapper test
func TestModifyOrder(t *testing.T) {
	o.SetDefaults()
	TestSetup(t)
	_, err := o.ModifyOrder(exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyOrder() error")
	}
}

// TestWithdraw Wrapper test
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

// TestWithdrawFiat Wrapper test
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

// TestSubmitOrder Wrapper test
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
