package huobi

import (
	"log"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply you own test keys here for due diligence testing.
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	testSymbol              = "btcusdt"
)

var h HUOBI
var wsSetupRan bool

func TestMain(m *testing.M) {
	h.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Huobi load config error", err)
	}
	hConfig, err := cfg.GetExchangeConfig("Huobi")
	if err != nil {
		log.Fatal("Huobi Setup() init error")
	}
	hConfig.API.AuthenticatedSupport = true
	hConfig.API.AuthenticatedWebsocketSupport = true
	hConfig.API.Credentials.Key = apiKey
	hConfig.API.Credentials.Secret = apiSecret
	h.Websocket = sharedtestvalues.NewTestWebsocket()
	err = h.Setup(hConfig)
	if err != nil {
		log.Fatal("Huobi setup error", err)
	}
	os.Exit(m.Run())
}

func setupWsTests(t *testing.T) {
	if wsSetupRan {
		return
	}
	if !h.Websocket.IsEnabled() && !h.API.AuthenticatedWebsocketSupport || !areTestAPIKeysSet() {
		t.Skip(stream.WebsocketNotEnabled)
	}
	comms = make(chan WsMessage, sharedtestvalues.WebsocketChannelOverrideCapacity)
	go h.wsReadData()
	var dialer websocket.Dialer
	err := h.wsAuthenticatedDial(&dialer)
	if err != nil {
		t.Fatal(err)
	}
	err = h.wsLogin()
	if err != nil {
		t.Fatal(err)
	}

	wsSetupRan = true
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := h.GetSpotKline(KlinesRequestParams{
		Symbol: testSymbol,
		Period: "1min",
		Size:   0,
	})
	if err != nil {
		t.Errorf("Huobi TestGetSpotKline: %s", err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 1)
	_, err = h.GetHistoricCandles(currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}

	_, err = h.GetHistoricCandles(currencyPair, asset.Spot, startTime.AddDate(0, 0, -7), time.Now(), kline.OneDay)
	if err != nil {
		t.Fatal(err)
	}

	_, err = h.GetHistoricCandles(currencyPair, asset.Spot, startTime, time.Now(), kline.Interval(time.Hour*7))
	if err == nil {
		t.Fatal("unexpected result")
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	currencyPair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 1)
	_, err = h.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, time.Now(), kline.OneMin)
	if err != nil {
		t.Fatal(err)
	}

	_, err = h.GetHistoricCandlesExtended(currencyPair, asset.Spot, startTime, time.Now(), kline.Interval(time.Hour*7))
	if err == nil {
		t.Fatal("unexpected result")
	}
}

func TestGetMarketDetailMerged(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetailMerged(testSymbol)
	if err != nil {
		t.Errorf("Huobi TestGetMarketDetailMerged: %s", err)
	}
}

func TestGetDepth(t *testing.T) {
	t.Parallel()
	_, err := h.GetDepth(OrderBookDataRequestParams{
		Symbol: testSymbol,
		Type:   OrderBookDataRequestParamsTypeStep1,
	})

	if err != nil {
		t.Errorf("Huobi TestGetDepth: %s", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := h.GetTrades(testSymbol)
	if err != nil {
		t.Errorf("Huobi TestGetTrades: %s", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := h.GetLatestSpotPrice(testSymbol)
	if err != nil {
		t.Errorf("Huobi GetLatestSpotPrice: %s", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := h.GetTradeHistory(testSymbol, 50)
	if err != nil {
		t.Errorf("Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetMarketDetail(t *testing.T) {
	t.Parallel()
	_, err := h.GetMarketDetail(testSymbol)
	if err != nil {
		t.Errorf("Huobi TestGetTradeHistory: %s", err)
	}
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := h.GetSymbols()
	if err != nil {
		t.Errorf("Huobi TestGetSymbols: %s", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := h.GetCurrencies()
	if err != nil {
		t.Errorf("Huobi TestGetCurrencies: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	_, err := h.GetTickers()
	if err != nil {
		t.Error(err)
	}
}

func TestGetTimestamp(t *testing.T) {
	t.Parallel()
	_, err := h.GetTimestamp()
	if err != nil {
		t.Errorf("Huobi TestGetTimestamp: %s", err)
	}
}

func TestGetAccounts(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() || !canManipulateRealOrders {
		t.Skip()
	}

	_, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Huobi GetAccounts: %s", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() || !canManipulateRealOrders {
		t.Skip()
	}

	result, err := h.GetAccounts()
	if err != nil {
		t.Errorf("Huobi GetAccounts: %s", err)
	}

	userID := strconv.FormatInt(result[0].ID, 10)
	_, err = h.GetAccountBalance(userID)
	if err != nil {
		t.Errorf("Huobi GetAccountBalance: %s", err)
	}
}

func TestGetAggregatedBalance(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetAggregatedBalance()
	if err != nil {
		t.Errorf("Huobi GetAggregatedBalance: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() || !canManipulateRealOrders {
		t.Skip()
	}

	arg := SpotNewOrderRequestParams{
		Symbol:    testSymbol,
		AccountID: 1,
		Amount:    0.01,
		Price:     10.1,
		Type:      SpotNewOrderRequestTypeBuyLimit,
	}

	_, err := h.SpotNewOrder(arg)
	if err != nil {
		t.Errorf("Huobi SpotNewOrder: %s", err)
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() || !canManipulateRealOrders {
		t.Skip()
	}
	_, err := h.CancelExistingOrder(1337)
	if err == nil {
		t.Error("Huobi TestCancelExistingOrder Expected error")
	}
}

func TestGetOrder(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() || !canManipulateRealOrders {
		t.Skip()
	}
	_, err := h.GetOrder(1337)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMarginLoanOrders(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetMarginLoanOrders(testSymbol, "", "", "", "", "", "", "")
	if err != nil {
		t.Errorf("Huobi TestGetMarginLoanOrders: %s", err)
	}
}

func TestGetMarginAccountBalance(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	_, err := h.GetMarginAccountBalance(testSymbol)
	if err != nil {
		t.Errorf("Huobi TestGetMarginAccountBalance: %s", err)
	}
}

func TestCancelWithdraw(t *testing.T) {
	t.Parallel()
	if !h.ValidateAPICredentials() || !canManipulateRealOrders {
		t.Skip()
	}
	_, err := h.CancelWithdraw(1337)
	if err == nil {
		t.Error("Huobi TestCancelWithdraw Expected error")
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.LTC.String(),
			"_"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	h.GetFeeByType(feeBuilder)
	if !areTestAPIKeysSet() {
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
	t.Parallel()
	var feeBuilder = setFeeBuilder()
	// CryptocurrencyTradeFee Basic
	if resp, err := h.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Error(err)
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
	}

	// CryptocurrencyTradeFee High quantity
	feeBuilder = setFeeBuilder()
	feeBuilder.Amount = 1000
	feeBuilder.PurchasePrice = 1000
	if resp, err := h.GetFee(feeBuilder); resp != float64(2000) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(2000), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee IsMaker
	feeBuilder = setFeeBuilder()
	feeBuilder.IsMaker = true
	if resp, err := h.GetFee(feeBuilder); resp != float64(0.002) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0.002), resp)
		t.Error(err)
	}

	// CryptocurrencyTradeFee Negative purchase price
	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
	// CryptocurrencyWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CryptocurrencyWithdrawalFee Invalid currency
	feeBuilder = setFeeBuilder()
	feeBuilder.Pair.Base = currency.NewCode("hello")
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// CyptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CyptocurrencyDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if resp, err := h.GetFee(feeBuilder); resp != float64(0) || err != nil {
		t.Errorf("GetFee() error. Expected: %f, Received: %f", float64(0), resp)
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoWithSetupText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := h.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:  order.AnyType,
		Pairs: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
	}

	_, err := h.GetActiveOrders(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	var getOrdersRequest = order.GetOrdersRequest{
		Type:  order.AnyType,
		Pairs: []currency.Pair{currency.NewPair(currency.BTC, currency.USDT)},
	}

	_, err := h.GetOrderHistory(&getOrdersRequest)
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func areTestAPIKeysSet() bool {
	return h.ValidateAPICredentials()
}

func TestSubmitOrder(t *testing.T) {
	if !h.ValidateAPICredentials() {
		t.Skip()
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	accounts, err := h.GetAccounts()
	if err != nil {
		t.Fatalf("Failed to get accounts. Err: %s", err)
	}

	var orderSubmission = &order.Submit{
		Pair: currency.Pair{
			Base:  currency.BTC,
			Quote: currency.USDT,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  strconv.FormatInt(accounts[0].ID, 10),
		AssetType: asset.Spot,
	}
	response, err := h.SubmitOrder(orderSubmission)
	if areTestAPIKeysSet() && (err != nil || !response.IsOrderPlaced) {
		t.Errorf("Order failed to be placed: %v", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := h.CancelOrder(orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = order.Cancel{
		ID:            "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
	}

	resp, err := h.CancelAllOrders(&orderCancellation)

	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestGetAccountInfo(t *testing.T) {
	if !areTestAPIKeysSet() {
		_, err := h.UpdateAccountInfo()
		if err == nil {
			t.Error("GetAccountInfo() Expected error")
		}
	} else {
		_, err := h.UpdateAccountInfo()
		if err != nil {
			t.Error("GetAccountInfo() error", err)
		}
	}
}

func TestModifyOrder(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}
	_, err := h.ModifyOrder(&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	withdrawCryptoRequest := withdraw.Request{
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	_, err := h.WithdrawCryptocurrencyFunds(&withdrawCryptoRequest)
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFunds(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	if areTestAPIKeysSet() && !canManipulateRealOrders {
		t.Skip("API keys set, canManipulateRealOrders false, skipping test")
	}

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFundsToInternationalBank(&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestQueryDepositAddress(t *testing.T) {
	_, err := h.QueryDepositAddress(currency.BTC.Lower().String())
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	}
}

func TestQueryWithdrawQuota(t *testing.T) {
	_, err := h.QueryWithdrawQuotas(currency.BTC.Lower().String())
	if !areTestAPIKeysSet() && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if areTestAPIKeysSet() && err != nil {
		t.Error(err)
	}
}

// TestWsGetAccountsList connects to WS, logs in, gets account list
func TestWsGetAccountsList(t *testing.T) {
	setupWsTests(t)
	_, err := h.wsGetAccountsList()
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetOrderList connects to WS, logs in, gets order list
func TestWsGetOrderList(t *testing.T) {
	setupWsTests(t)
	p, err := currency.NewPairFromString("ethbtc")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.wsGetOrdersList(1, p)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetOrderDetails connects to WS, logs in, gets order details
func TestWsGetOrderDetails(t *testing.T) {
	setupWsTests(t)
	orderID := "123"
	_, err := h.wsGetOrderDetails(orderID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsSubResponse(t *testing.T) {
	pressXToJSON := []byte(`{
  "op": "sub",
  "cid": "123",
  "err-code": 0,
  "ts": 1489474081631,
  "topic": "accounts"
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKline(t *testing.T) {
	pressXToJSON := []byte(`{
  "ch": "market.btcusdt.kline.1min",
  "ts": 1489474082831,
  "tick": {
    "id": 1489464480,
    "amount": 0.0,
    "count": 0,
    "open": 7962.62,
    "close": 7962.62,
    "low": 7962.62,
    "high": 7962.62,
    "vol": 0.0
  }
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsUnsubscribe(t *testing.T) {
	pressXToJSON := []byte(`{
  "id": "id4",
  "status": "ok",
  "unsubbed": "market.btcusdt.trade.detail",
  "ts": 1494326028889
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsKlineArray(t *testing.T) {
	pressXToJSON := []byte(`{
  "status": "ok",
  "rep": "market.btcusdt.kline.1min",
  "data": [
    {
      "amount": 1.6206,
      "count":  3,
      "id":     1494465840,
      "open":   9887.00,
      "close":  9885.00,
      "low":    9885.00,
      "high":   9887.00,
      "vol":    16021.632026
    },
    {
      "amount": 2.2124,
      "count":  6,
      "id":     1494465900,
      "open":   9885.00,
      "close":  9880.00,
      "low":    9880.00,
      "high":   9885.00,
      "vol":    21859.023500
    }
  ]
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsMarketDepth(t *testing.T) {
	pressXToJSON := []byte(`{
  "ch": "market.htusdt.depth.step0",
  "ts": 1572362902027,
  "tick": {
    "bids": [
      [3.7721, 344.86],
      [3.7709, 46.66]
    ],
    "asks": [
      [3.7745, 15.44],
      [3.7746, 70.52]
    ],
    "version": 100434317651,
    "ts": 1572362902012
  }
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsBestBidOffer(t *testing.T) {
	pressXToJSON := []byte(`{
	  "ch": "market.btcusdt.bbo",
	  "ts": 1489474082831,
	  "tick": {
		"symbol": "btcusdt",
		"quoteTime": "1489474082811",
		"bid": "10008.31",
		"bidSize": "0.01",
		"ask": "10009.54",
		"askSize": "0.3"
	  }
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTradeDetail(t *testing.T) {
	pressXToJSON := []byte(`{
	  "ch": "market.btcusdt.trade.detail",
	  "ts": 1489474082831,
	  "tick": {
			"id": 14650745135,
			"ts": 1533265950234,
			"data": [
				{
					"amount": 0.0099,
					"ts": 1533265950234,
					"id": 146507451359183894799,
					"tradeId": 102043495674,
					"price": 401.74,
					"direction": "buy"
				}
			]
	  }
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
  "rep": "market.btcusdt.detail",
  "id": "id11",
	"data":{
		"amount": 12224.2922,
		"open":   9790.52,
		"close":  10195.00,
		"high":   10300.00,
		"ts":     1494496390000,
		"id":     1494496390,
		"count":  15195,
		"low":    9657.00,
		"vol":    121906001.754751
	  }
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsAccountUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
	  "op": "notify",
	  "ts": 1522856623232,
	  "topic": "accounts",
	  "data": {
		"event": "order.place",
		"list": [
		  {
			"account-id": 419013,
			"currency": "usdt",
			"type": "trade",
			"balance": "500009195917.4362872650"
		  }
		]
	  }
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
  "op": "notify",
  "topic": "orders.htusdt",
  "ts": 1522856623232,
  "data": {
    "seq-id": 94984,
    "order-id": 2039498445,
    "symbol": "btcusdt",
    "account-id": 100077,
    "order-amount": "5000.000000000000000000",
    "order-price": "1.662100000000000000",
    "created-at": 1522858623622,
    "order-type": "buy-limit",
    "order-source": "api",
    "order-state": "filled",
    "role": "taker",
    "price": "1.662100000000000000",
    "filled-amount": "5000.000000000000000000",
    "unfilled-amount": "0.000000000000000000",
    "filled-cash-amount": "8301.357280000000000000",
    "filled-fees": "8.000000000000000000"
  }
}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubsbOp(t *testing.T) {
	pressXToJSON := []byte(`{
	  "op": "unsub",
	  "topic": "accounts",
	  "cid": "123"
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
	  "op": "sub",
	  "cid": "123",
	  "err-code": 0,
	  "ts": 1489474081631,
	  "topic": "accounts"
	}`)
	err = h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsMarketByPrice(t *testing.T) {
	pressXToJSON := []byte(`{
		"ch": "market.btcusdt.mbp.150",
		"ts": 1573199608679,
		"tick": {
			"seqNum": 100020146795,
			"prevSeqNum": 100020146794,
			"bids": [],
			"asks": [
				[645.140000000000000000, 26.755973959140651643]
			]
		}
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
	pressXToJSON = []byte(`{
		"id": "id2",
		"rep": "market.btcusdt.mbp.150",
		"status": "ok",
		"data": {
			"seqNum": 100020142010,
			"bids": [
				[618.37, 71.594],
				[423.33, 77.726],
				[223.18, 47.997],
				[219.34, 24.82],
				[210.34, 94.463]
		],
			"asks": [
				[650.59, 14.909733438479636],
				[650.63, 97.996],
				[650.77, 97.465],
				[651.23, 83.973],
				[651.42, 34.465]
			]
		}
	}`)
	err = h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrdersUpdate(t *testing.T) {
	pressXToJSON := []byte(`{
		"op": "notify",
		"ts": 1522856623232,
		"topic": "orders.btcusdt.update",
		"data": {
		"unfilled-amount": "0.000000000000000000",
			"filled-amount": "5000.000000000000000000",
			"price": "1.662100000000000000",
			"order-id": 2039498445,
			"symbol": "btcusdt",
			"match-id": 94984,
			"filled-cash-amount": "8301.357280000000000000",
			"role": "taker|maker",
			"order-state": "filled",
			"client-order-id": "a0001",
			"order-type": "buy-limit"
	}
	}`)
	err := h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestStringToOrderStatus(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Status
	}
	testCases := []TestCases{
		{Case: "submitted", Result: order.New},
		{Case: "canceled", Result: order.Cancelled},
		{Case: "partial-filled", Result: order.PartiallyFilled},
		{Case: "partial-canceled", Result: order.PartiallyCancelled},
		{Case: "LOL", Result: order.UnknownStatus},
	}
	for i := range testCases {
		result, _ := stringToOrderStatus(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestStringToOrderSide(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Side
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Buy},
		{Case: "sell-limit", Result: order.Sell},
		{Case: "woah-nelly", Result: order.UnknownSide},
	}
	for i := range testCases {
		result, _ := stringToOrderSide(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func TestStringToOrderType(t *testing.T) {
	type TestCases struct {
		Case   string
		Result order.Type
	}
	testCases := []TestCases{
		{Case: "buy-limit", Result: order.Limit},
		{Case: "sell-market", Result: order.Market},
		{Case: "woah-nelly", Result: order.UnknownType},
	}
	for i := range testCases {
		result, _ := stringToOrderType(testCases[i].Case)
		if result != testCases[i].Result {
			t.Errorf("Exepcted: %v, received: %v", testCases[i].Result, result)
		}
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"1min",
		},
		{
			"FourHour",
			kline.FourHour,
			"4hour",
		},
		{
			"OneDay",
			kline.OneDay,
			"1day",
		},
		{
			"OneWeek",
			kline.OneWeek,
			"1week",
		},
		{
			"OneMonth",
			kline.OneMonth,
			"1mon",
		},
		{
			"OneYear",
			kline.OneYear,
			"1year",
		},
		{
			"AllOthers",
			kline.TwoWeek,
			"",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := h.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC-USDT")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.GetRecentTrades(currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString(testSymbol)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.GetHistoricTrades(currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}
