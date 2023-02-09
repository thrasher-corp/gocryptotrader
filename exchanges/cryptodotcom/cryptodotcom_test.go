package cryptodotcom

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false

	credInfoNotProvided                             = "credentials not provided"
	credInfoNotProvidedOrCannotManipulateRealOrders = "credentials must be provided and field canManipulateRealOrders must be enabled"
)

var cr Cryptodotcom

func TestMain(m *testing.M) {
	cr.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	exchCfg, err := cfg.GetExchangeConfig("Cryptodotcom")
	if err != nil {
		log.Fatal(err)
	}
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	if apiKey != "" && apiSecret != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}
	cr.Websocket = sharedtestvalues.NewTestWebsocket()
	err = cr.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	cr.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	cr.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	setupWS()
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return cr.ValidateAPICredentials(cr.GetDefaultCredentials()) == nil
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := cr.GetInstruments(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := cr.GetOrderbook(context.Background(), "BTC_USDT", 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCandlestickDetail(t *testing.T) {
	t.Parallel()
	_, err := cr.GetCandlestickDetail(context.Background(), "BTC_USDT", kline.FiveMin)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := cr.GetTicker(context.Background(), "BTC_USDT")
	if err != nil {
		t.Error(err)
	}
	_, err = cr.GetTicker(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := cr.GetTrades(context.Background(), "BTC_USDT")
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	_, err := cr.WithdrawFunds(context.Background(), currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsCreateWithdrawal(currency.BTC, 10, core.BitcoinDonationAddress, "", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyNetworks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetCurrencyNetworks(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetWithdrawalHistory(context.Background())
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsRetriveWithdrawalHistory()
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetDepositHistory(context.Background(), currency.EMPTYCODE, time.Time{}, time.Time{}, 20, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPersonalDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetPersonalDepositAddress(context.Background(), currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountSummary(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetAccountSummary(context.Background(), currency.USDT)
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsRetriveAccountSummary(currency.BTC)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	arg := &CreateOrderParam{InstrumentName: "BTC_USDT", Side: order.Buy, OrderType: orderTypeToString(order.Limit), Price: 123, Quantity: 12}
	_, err := cr.CreateOrder(context.Background(), arg)
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsPlaceOrder(arg)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	err := cr.CancelExistingOrder(context.Background(), "BTC_USDT", "1232412")
	if err != nil {
		t.Error(err)
	}
	err = cr.WsCancelExistingOrder("BTC_USDT", "1232412")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPrivateTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetPrivateTrades(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsRetrivePrivateTrades("", time.Time{}, time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetOrderDetail(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsRetriveOrderDetail("1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetPersonalOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetPersonalOpenOrders(context.Background(), "", 0, 0)
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsRetrivePersonalOpenOrders("", 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestGetPersonalOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetPersonalOrderHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 20)
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsRetrivePersonalOrderHistory("", time.Time{}, time.Time{}, 0, 20)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrderList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	_, err := cr.CreateOrderList(context.Background(), "LIST", []CreateOrderParam{
		{
			InstrumentName: "BTC_USDT", ClientOrderID: "", TimeInForce: "", Side: order.Buy, OrderType: orderTypeToString(order.Limit), PostOnly: false, TriggerPrice: 0, Price: 123, Quantity: 12, Notional: 0,
		}})
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsCreateOrderList("LIST", []CreateOrderParam{
		{
			InstrumentName: "BTC_USDT", ClientOrderID: "", TimeInForce: "", Side: order.Buy, OrderType: orderTypeToString(order.Limit), PostOnly: false, TriggerPrice: 0, Price: 123, Quantity: 12, Notional: 0,
		}})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrderList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	_, err := cr.CancelOrderList(context.Background(), []CancelOrderParam{
		{InstrumentName: "BTC_USDT", OrderID: "1234567"}, {InstrumentName: "BTC_USDT",
			OrderID: "123450067"}})
	if err != nil {
		t.Error(err)
	}
	_, err = cr.WsCancelOrderList([]CancelOrderParam{
		{InstrumentName: "BTC_USDT", OrderID: "1234567"}, {InstrumentName: "BTC_USDT",
			OrderID: "123450067"}})
	if err != nil {
		t.Error(err)
	}
}

func TestCancelAllPersonalOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	err = cr.CancelAllPersonalOrders(context.Background(), enabledPairs[0].String())
	if err != nil {
		t.Error(err)
	}
	err = cr.WsCancelAllPersonalOrders(enabledPairs[0].String())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetAccounts(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetTransactions(context.Background(), "BTC-USDT", "", time.Time{}, time.Time{}, 20)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateSubAccountTransfer(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	err := cr.CreateSubAccountTransfer(context.Background(), "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfavf", core.BitcoinDonationAddress, currency.USDT, 1232)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOTCUser(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetOTCUser(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetOTCInstruments(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetOTCInstruments(context.Background())
	if err != nil {
		t.Error(err)
	}
}
func TestRequestOTCQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.RequestOTCQuote(context.Background(), currency.NewPair(currency.BTC, currency.USDT), .001, 232, "BUY")
	if err != nil {
		t.Error(err)
	}
}

func TestAcceptOTCQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.AcceptOTCQuote(context.Background(), "12323123", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOTCQuoteHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetOTCQuoteHistory(context.Background(), currency.EMPTYPAIR, time.Time{}, time.Time{}, 0, 10)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOTCTradeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetOTCTradeHistory(context.Background(), currency.NewPair(currency.BTC, currency.USDT), time.Time{}, time.Time{}, 0, 0)
	if err != nil && !strings.Contains(err.Error(), "OTC_USER_NO_PERMISSION") {
		t.Error(err)
	}
}

// wrapper test functions

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := cr.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cr.UpdateTicker(context.Background(), enabledPairs[0], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := cr.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cr.FetchTicker(context.Background(), enabledPairs[0], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	cr.Verbose = true
	_, err = cr.FetchOrderbook(context.Background(), enabledPairs[1], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	cr.Verbose = true
	_, err = cr.UpdateOrderbook(context.Background(), enabledPairs[1], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	if _, err := cr.UpdateAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Cryptodotcom UpdateAccountInfo() error", err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	if _, err := cr.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot); err != nil {
		t.Error("Cryptodotcom GetWithdrawalsHistory() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	if _, err := cr.GetRecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); err != nil {
		t.Error("Cryptodotcom GetRecentTrades() error", err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	if _, err := cr.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, time.Now().Add(-time.Hour*4), time.Now()); err != nil {
		t.Error(err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	if _, err := cr.GetFundingHistory(context.Background()); err != nil {
		t.Error("Cryptodotcom GetFundingHistory() error", err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Minute * 40)
	endTime := time.Now()
	_, err = cr.GetHistoricCandles(context.Background(), enabledPairs[0], asset.Spot, kline.OneDay, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}
	_, err = cr.GetHistoricCandles(context.Background(), enabledPairs[0], asset.Spot, kline.FiveMin, startTime, endTime)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	enabledPairs, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{enabledPairs[0], currency.NewPair(currency.USDT, currency.USD), currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	if _, err := cr.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Cryptodotcom GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	_, err := cr.GetOrderHistory(context.Background(), &getOrdersRequest)
	if err != nil {
		t.Error(err)
	}
	getOrdersRequest.Pairs = []currency.Pair{currency.NewPair(currency.LTC, currency.BTC)}
	if _, err := cr.GetOrderHistory(context.Background(), &getOrdersRequest); err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  cr.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.Spot,
	}
	_, err := cr.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error("Cryptodotcom SubmitOrder() error", err)
	}
	orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.LTC,
			Quote: currency.BTC,
		},
		Exchange:  cr.Name,
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1000000000,
		ClientID:  "myOwnOrder",
		AssetType: asset.Spot,
	}
	_, err = cr.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error("Cryptodotcom SubmitOrder() error", err)
	}
}
func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	var orderCancellation = &order.Cancel{
		OrderID:   "1",
		Pair:      currency.NewPair(currency.LTC, currency.BTC),
		AssetType: asset.Spot,
	}
	if err := cr.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	var orderCancellationParams = []order.Cancel{
		{
			OrderID: "1",
			Pair:    currency.NewPair(currency.LTC, currency.BTC),
		},
		{
			OrderID: "1",
			Pair:    currency.NewPair(currency.LTC, currency.BTC),
		},
	}
	_, err := cr.CancelBatchOrders(context.Background(), orderCancellationParams)
	if err != nil {
		t.Error("Cryptodotcom CancelBatchOrders() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	if _, err := cr.CancelAllOrders(context.Background(), &order.Cancel{}); err != nil {
		t.Errorf("%s CancelAllOrders() error: %v", cr.Name, err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	enabled, err := cr.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error("couldn't find enabled tradable pairs")
	}
	if len(enabled) == 0 {
		t.SkipNow()
	}
	_, err = cr.GetOrderInfo(context.Background(),
		"123", enabled[0], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.GetDepositAddress(context.Background(), currency.ETH, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("skipping test: api keys not set")
	}
	_, err := cr.WithdrawCryptocurrencyFunds(context.Background(), &withdraw.Request{
		Amount:   10,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Chain:      currency.BTC.String(),
			Address:    core.BitcoinDonationAddress,
			AddressTag: "",
		}})
	if err != nil {
		t.Fatal(err)
	}
}

func setupWS() {
	if !cr.Websocket.IsEnabled() {
		return
	}
	if !areTestAPIKeysSet() {
		cr.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := cr.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	_, err := cr.GenerateDefaultSubscriptions()
	if err != nil {
		t.Error(err)
	}
}

func TestWsRetriveCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip(credInfoNotProvided)
	}
	_, err := cr.WsRetriveCancelOnDisconnect()
	if err != nil {
		t.Error(err)
	}
}
func TestWsSetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip(credInfoNotProvidedOrCannotManipulateRealOrders)
	}
	_, err := cr.WsSetCancelOnDisconnect("ACCOUNT")
	if err != nil {
		t.Error(err)
	}
}

func TestGetCreateParamMap(t *testing.T) {
	t.Parallel()
	arg := &CreateOrderParam{InstrumentName: "BTC_USDT", Side: order.Buy, OrderType: orderTypeToString(order.Limit), Price: 123, Quantity: 12}
	_, err := arg.getCreateParamMap()
	if err != nil {
		t.Error(err)
	}
	arg.OrderType = orderTypeToString(order.Market)
	_, err = arg.getCreateParamMap()
	if err != nil {
		t.Error(err)
	}
	arg.OrderType = orderTypeToString(order.TakeProfit)
	arg.Notional = 12
	_, err = arg.getCreateParamMap()
	if !errors.Is(err, errTriggerPriceRequired) {
		t.Errorf("found %v, but expecting %v", err, errTriggerPriceRequired)
	}
	arg.OrderType = orderTypeToString(order.UnknownType)
	_, err = arg.getCreateParamMap()
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Errorf("found %v, but expecting %v", err, order.ErrTypeIsInvalid)
	}
	arg.OrderType = orderTypeToString(order.StopLimit)
	_, err = arg.getCreateParamMap()
	if !errors.Is(err, errTriggerPriceRequired) {
		t.Errorf("found %v, but expecting %v", err, order.ErrTypeIsInvalid)
	}
}

func TestWsConnect(t *testing.T) {
	t.Parallel()
	if err := cr.WsConnect(); err != nil {
		t.Error(err)
	}
}
