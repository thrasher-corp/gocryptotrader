package hitbtc

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var (
	e          *Exchange
	wsSetupRan bool
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var spotPair = currency.NewBTCUSD().Format(currency.PairFormat{Uppercase: true})

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("HitBTC Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		log.Fatalf("HitBTC UpdateTradablePairs error: %s", err)
	}

	os.Exit(m.Run())
}

func TestGetOrderbook(t *testing.T) {
	_, err := e.GetOrderbook(t.Context(), spotPair.String(), 50)
	assert.NoError(t, err, "GetOrderbook should not error")
}

func TestGetTrades(t *testing.T) {
	_, err := e.GetTrades(t.Context(), spotPair.String(), "", "", 0, 0, 0, 0)
	assert.NoError(t, err, "GetTrades should not error")
}

func TestGetChartCandles(t *testing.T) {
	_, err := e.GetCandles(t.Context(), spotPair.String(), "", "D1", time.Now().Add(-24*time.Hour), time.Now())
	assert.NoError(t, err, "GetCandles should not error")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	startTime := time.Now().Add(-time.Hour * 6)
	end := time.Now()
	_, err := e.GetHistoricCandles(t.Context(), spotPair, asset.Spot, kline.OneMin, startTime, end)
	assert.NoError(t, err, "GetHistoricCandles should not error")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()

	startTime := time.Unix(1546300800, 0)
	end := time.Unix(1577836799, 0)
	_, err := e.GetHistoricCandlesExtended(t.Context(), spotPair, asset.Spot, kline.OneHour, startTime, end)
	assert.NoError(t, err, "GetHistoricCandlesExtended should not error")
}

func TestGetCurrencies(t *testing.T) {
	_, err := e.GetCurrencies(t.Context())
	if err != nil {
		t.Error("Test failed - HitBTC GetCurrencies() error", err)
	}
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:              1,
		FeeType:             exchange.CryptocurrencyTradeFee,
		Pair:                currency.NewPair(currency.ETH, currency.BTC),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	_, err := e.GetFeeByType(t.Context(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(e) {
		if feeBuilder.FeeType != exchange.OfflineTradeFee {
			t.Errorf("Expected %v, received %v", exchange.OfflineTradeFee, feeBuilder.FeeType)
		}
	} else {
		if feeBuilder.FeeType != exchange.CryptocurrencyTradeFee {
			t.Errorf("Expected %v, received %v", exchange.CryptocurrencyTradeFee, feeBuilder.FeeType)
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	pairs, err := currency.NewPairsFromStrings([]string{"BTC-USD", "XRP-USDT"})
	if err != nil {
		t.Fatal(err)
	}
	err = e.CurrencyPairs.StorePairs(asset.Spot, pairs, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.UpdateTicker(t.Context(), pairs[0], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	require.NoError(t, e.UpdateTickers(t.Context(), asset.Spot))

	enabled, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err)

	for j := range enabled {
		_, err = e.GetCachedTicker(enabled[j], asset.Spot)
		require.NoErrorf(t, err, "GetCached Ticker must not error for pair %q", enabled[j])
	}
}

func TestGetAllTickers(t *testing.T) {
	_, err := e.GetTickers(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetSingularTicker(t *testing.T) {
	_, err := e.GetTicker(t.Context(), spotPair.String())
	assert.NoError(t, err, "GetTicker should not error")
}

func TestGetFee(t *testing.T) {
	feeBuilder := setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(e) {
		// CryptocurrencyTradeFee Basic
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyWithdrawalFee Invalid currency
		feeBuilder = setFeeBuilder()
		feeBuilder.Pair.Base = currency.NewCode("hello")
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	feeBuilder.Pair.Base = currency.BTC
	feeBuilder.Pair.Quote = currency.LTC
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := e.GetFee(t.Context(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.ETH, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := e.GetActiveOrders(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	getOrdersRequest := order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{currency.NewPair(currency.ETH, currency.BTC)},
		Side:      order.AnySide,
	}

	_, err := e.GetOrderHistory(t.Context(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// TestSubmitOrder and below can impact your orders on the exchange. Enable canManipulateRealOrders to run them
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	orderSubmission := &order.Submit{
		Exchange: e.Name,
		Pair: currency.Pair{
			Base:  currency.DGD,
			Quote: currency.BTC,
		},
		Side:      order.Buy,
		Type:      order.Limit,
		Price:     1,
		Amount:    1,
		ClientID:  "meowOrder",
		AssetType: asset.Spot,
	}
	response, err := e.SubmitOrder(t.Context(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(e) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	err := e.CancelOrder(t.Context(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	orderCancellation := &order.Cancel{
		OrderID:   "1",
		AccountID: "1",
		Pair:      currencyPair,
		AssetType: asset.Spot,
	}

	resp, err := e.CancelAllOrders(t.Context(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	_, err := e.ModifyOrder(t.Context(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    e.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := e.WithdrawCryptocurrencyFunds(t.Context(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(e) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(e) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFunds(t.Context(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, e, canManipulateRealOrders)

	withdrawFiatRequest := withdraw.Request{}
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(e) {
		_, err := e.GetDepositAddress(t.Context(), currency.XRP, "", "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := e.GetDepositAddress(t.Context(), currency.BTC, "", "")
		if err == nil {
			t.Error("GetDepositAddress() error cannot be nil")
		}
	}
}

func setupWsAuth(t *testing.T) {
	t.Helper()
	if wsSetupRan {
		return
	}
	if !e.Websocket.IsEnabled() && !e.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(e) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}

	var dialer gws.Dialer
	err := e.Websocket.Conn.Dial(t.Context(), &dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go e.wsReadData(t.Context())
	err = e.wsLogin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(time.Second)
	select {
	case loginError := <-e.Websocket.DataHandler.C:
		t.Fatal(loginError)
	case <-timer.C:
	}
	timer.Stop()
	wsSetupRan = true
}

func TestWsCancelOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := e.wsCancelOrder(t.Context(), "ImNotARealOrderID")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsPlaceOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := e.wsPlaceOrder(t.Context(), currency.NewPair(currency.LTC, currency.BTC), order.Buy.String(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsReplaceOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := e.wsReplaceOrder(t.Context(), "ImNotARealOrderID", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsGetActiveOrders(t *testing.T) {
	setupWsAuth(t)
	if _, err := e.wsGetActiveOrders(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestWsGetTradingBalance(t *testing.T) {
	setupWsAuth(t)
	if _, err := e.wsGetTradingBalance(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestWsGetTrades(t *testing.T) {
	setupWsAuth(t)
	_, err := e.wsGetTrades(t.Context(), currency.NewPair(currency.ETH, currency.BTC), 1000, "ASC", "id")
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsGetSymbols(t *testing.T) {
	setupWsAuth(t)
	_, err := e.wsGetSymbols(t.Context(), currency.NewPair(currency.ETH, currency.BTC))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsGetCurrencies(t *testing.T) {
	setupWsAuth(t)
	_, err := e.wsGetCurrencies(t.Context(), currency.BTC)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsGetActiveOrdersJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "method": "activeOrders",
  "params": [
    {
      "id": "4345613661",
      "clientOrderId": "57d5525562c945448e3cbd559bd068c3",
      "symbol": "BTCUSD",
      "side": "sell",
      "status": "new",
      "type": "limit",
      "timeInForce": "GTC",
      "quantity": "0.013",
      "price": "0.100000",
      "cumQuantity": "0.000",
      "postOnly": false,
      "createdAt": "2017-10-20T12:17:12.245Z",
      "updatedAt": "2017-10-20T12:17:12.245Z",
      "reportType": "status"
    }
  ]
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsGetCurrenciesJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": {
    "id": "ETH",
    "fullName": "Ethereum",
    "crypto": true,
    "payinEnabled": true,
    "payinPaymentId": false,
    "payinConfirmations": 2,
    "payoutEnabled": true,
    "payoutIsPaymentId": false,
    "transferEnabled": true,
    "delisted": false,
    "payoutFee": "0.001"
  },
  "id": "c4ce77f5-1c50-435a-b623-4961191ca129"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsGetSymbolsJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": {
    "id": "ETHBTC",
    "baseCurrency": "ETH",
    "quoteCurrency": "BTC",
    "quantityIncrement": "0.001",
    "tickSize": "0.000001",
    "takeLiquidityRate": "0.001",
    "provideLiquidityRate": "-0.0001",
    "feeCurrency": "BTC"
  },
  "id": "1c847290-b366-412b-b8f5-dc630ed5b147"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "method": "ticker",
  "params": {
    "ask": "0.054464",
    "bid": "0.054463",
    "last": "0.054463",
    "open": "0.057133",
    "low": "0.053615",
    "high": "0.057559",
    "volume": "33068.346",
    "volumeQuote": "1832.687530809",
    "timestamp": "2017-10-19T15:45:44.941Z",
    "symbol": "BTCUSD"
  }
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderbook(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "method": "snapshotOrderbook",
  "params": {
    "ask": [
      {
        "price": "0.054588",
        "size": "0.245"
      },
      {
        "price": "0.054590",
        "size": "1.000"
      },
      {
        "price": "0.054591",
        "size": "2.784"
      }
    ],
    "bid": [
      {
        "price": "0.054558",
        "size": "0.500"
      },
      {
        "price": "0.054557",
        "size": "0.076"
      },
      {
        "price": "0.054524",
        "size": "7.725"
      }
    ],
    "symbol": "BTCUSD",
    "sequence": 8073827,    
    "timestamp": "2018-11-19T05:00:28.193Z"
  }
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "jsonrpc": "2.0",
  "method": "updateOrderbook",
  "params": {    
    "ask": [
      {
        "price": "0.054590",
        "size": "0.000"
      },
      {
        "price": "0.054591",
        "size": "0.000"
      }
    ],
    "bid": [
      {
        "price": "0.054504",
         "size": "0.000"
      }
    ],
    "symbol": "BTCUSD",
    "sequence": 8073830,
    "timestamp": "2018-11-19T05:00:28.700Z"
  }
}`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderNotification(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "method": "report",
  "params": {
    "id": "4345697765",
    "clientOrderId": "53b7cf917963464a811a4af426102c19",
    "symbol": "BTCUSD",
    "side": "sell",
    "status": "filled",
    "type": "limit",
    "timeInForce": "GTC",
    "quantity": "0.001",
    "price": "0.053868",
    "cumQuantity": "0.001",
    "postOnly": false,
    "createdAt": "2017-10-20T12:20:05.952Z",
    "updatedAt": "2017-10-20T12:20:38.708Z",
    "reportType": "trade",
    "tradeQuantity": "0.001",
    "tradePrice": "0.053868",
    "tradeId": 55051694,
    "tradeFee": "-0.000000005"
  }
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsSubmitOrderJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": {
    "id": "4345947689",
    "clientOrderId": "57d5525562c945448e3cbd559bd068c4",
    "symbol": "BTCUSD",
    "side": "sell",
    "status": "new",
    "type": "limit",
    "timeInForce": "GTC",
    "quantity": "0.001",
    "price": "0.093837",
    "cumQuantity": "0.000",
    "postOnly": false,
    "createdAt": "2017-10-20T12:29:43.166Z",
    "updatedAt": "2017-10-20T12:29:43.166Z",
    "reportType": "new"
  },
  "id": "99f55c70-1166-49a7-87e9-3b54a00ad893"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsCancelOrderJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": {
    "id": "4345947689",
    "clientOrderId": "57d5525562c945448e3cbd559bd068c4",
    "symbol": "BTCUSD",
    "side": "sell",
    "status": "canceled",
    "type": "limit",
    "timeInForce": "GTC",
    "quantity": "0.001",
    "price": "0.093837",
    "cumQuantity": "0.000",
    "postOnly": false,
    "createdAt": "2017-10-20T12:29:43.166Z",
    "updatedAt": "2017-10-20T12:31:26.174Z",
    "reportType": "canceled"
  },
  "id": "2ce46937-2770-4453-ac99-ee87939bf5bb"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsCancelReplaceJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": {
    "id": "4346371528",
    "clientOrderId": "9cbe79cb6f864b71a811402a48d4b5b2",
    "symbol": "BTCUSD",
    "side": "sell",
    "status": "new",
    "type": "limit",
    "timeInForce": "GTC",
    "quantity": "0.002",
    "price": "0.083837",
    "cumQuantity": "0.000",
    "postOnly": false,
    "createdAt": "2017-10-20T12:47:07.942Z",
    "updatedAt": "2017-10-20T12:50:34.488Z",
    "reportType": "replaced",
    "originalRequestClientOrderId": "9cbe79cb6f864b71a811402a48d4b5b1"
  },
  "id": "91e925d3-3b95-4e29-8ae7-938fd5006709"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsGetTradesRequestResponse(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": [
    {
      "currency": "BCN",
      "available": "100.000000000",
      "reserved": "0"
    },
    {
      "currency": "BTC",
      "available": "0.013634021",
      "reserved": "0"
    },
    {
      "currency": "ETH",
      "available": "0",
      "reserved": "0.00200000"
    }
  ],
  "id": "4b1f1391-215e-4d12-972c-5cea9d50edf4"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsGetActiveOrdersRequestJSON(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "result": [
    {
      "id": "4346371528",
      "clientOrderId": "9cbe79cb6f864b71a811402a48d4b5b2",
      "symbol": "BTCUSD",
      "side": "sell",
      "status": "new",
      "type": "limit",
      "timeInForce": "GTC",
      "quantity": "0.002",
      "price": "0.083837",
      "cumQuantity": "0.000",
      "postOnly": false,
      "createdAt": "2017-10-20T12:47:07.942Z",
      "updatedAt": "2017-10-20T12:50:34.488Z",
      "reportType": "replaced",
      "originalRequestClientOrderId": "9cbe79cb6f864b71a811402a48d4b5b1"
    }
  ],
  "id": "9e67b440-2eec-445a-be3a-e81f962c8391"
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsTrades(t *testing.T) {
	pressXToJSON := []byte(`{
  "jsonrpc": "2.0",
  "method": "snapshotTrades",
  "params": {
    "data": [
      {
        "id": 54469456,
        "price": "0.054656",
        "quantity": "0.057",
        "side": "buy",
        "timestamp": "2017-10-19T16:33:42.821Z"
      },
      {
        "id": 54469497,
        "price": "0.054656",
        "quantity": "0.092",
        "side": "buy",
        "timestamp": "2017-10-19T16:33:48.754Z"
      },
      {
        "id": 54469697,
        "price": "0.054669",
        "quantity": "0.002",
        "side": "buy",
        "timestamp": "2017-10-19T16:34:13.288Z"
      }
    ],
    "symbol": "BTCUSD"
  }
}`)
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`{
  "jsonrpc": "2.0",
  "method": "updateTrades",
  "params": {
    "data": [
      {
        "id": 54469813,
        "price": "0.054670",
        "quantity": "0.183",
        "side": "buy",
        "timestamp": "2017-10-19T16:34:25.041Z"
      }
    ],
    "symbol": "BTCUSD"
  }
}    `)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		output   string
	}{
		{
			kline.OneMin,
			"M1",
		},
		{
			kline.OneDay,
			"D1",
		},
		{
			kline.SevenDay,
			"D7",
		},
		{
			kline.OneMonth,
			"1M",
		},
	} {
		t.Run(tc.interval.String(), func(t *testing.T) {
			t.Parallel()
			ret, err := formatExchangeKlineInterval(tc.interval)
			require.NoError(t, err)
			assert.Equal(t, tc.output, ret)
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), spotPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), spotPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.NoError(t, err, "GetHistoricTrades should not error")
	// longer term
	_, err = e.GetHistoricTrades(t.Context(), spotPair, asset.Spot, time.Now().Add(-time.Minute*60*200), time.Now().Add(-time.Minute*60*199))
	assert.NoError(t, err, "GetHistoricTrades should not error")
}

func TestGetActiveOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetActiveOrderByClientOrderID(t.Context(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetOrderInfo(t.Context(), "1234", currency.NewBTCUSD(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()

	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	r, err := e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err)
	require.NotEmpty(t, r)
	assert.Contains(t, r, spotPair, "BTC-USD should be in the fetched pairs")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, false)
		require.NoErrorf(t, err, "cannot get pairs for %s", a)
		require.NotEmptyf(t, pairs, "no pairs for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")

	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	pairs, err := e.GetEnabledPairs(asset.Spot)
	require.NoErrorf(t, err, "GetEnabledPairs must not error")
	for _, s := range e.Features.Subscriptions {
		for _, p := range pairs.Format(currency.PairFormat{Uppercase: true}) {
			s = s.Clone()
			s.Pairs = currency.Pairs{p}
			n := subscriptionNames[s.Channel]
			switch s.Channel {
			case subscription.MyAccountChannel:
				s.QualifiedChannel = `{"method":"` + n + `"}`
			case subscription.CandlesChannel:
				s.QualifiedChannel = `{"method":"` + n + `","params":{"symbol":"` + p.String() + `","period":"M30","limit":100}}`
			case subscription.AllTradesChannel:
				s.QualifiedChannel = `{"method":"` + n + `","params":{"symbol":"` + p.String() + `","limit":100}}`
			default:
				s.QualifiedChannel = `{"method":"` + n + `","params":{"symbol":"` + p.String() + `"}}`
			}
			exp = append(exp, s)
		}
	}
	testsubs.EqualLists(t, exp, subs)
}

func TestIsSymbolChannel(t *testing.T) {
	t.Parallel()
	assert.True(t, isSymbolChannel(&subscription.Subscription{Channel: subscription.TickerChannel}))
	assert.False(t, isSymbolChannel(&subscription.Subscription{Channel: subscription.MyAccountChannel}))
}

func TestSubToReq(t *testing.T) {
	t.Parallel()
	p := currency.NewPairWithDelimiter("BTC", "USD", "-")
	r := subToReq(&subscription.Subscription{Channel: subscription.TickerChannel}, p)
	assert.Equal(t, "Ticker", r.Method)
	assert.Equal(t, "BTC-USD", (r.Params.Symbol))

	r = subToReq(&subscription.Subscription{Channel: subscription.CandlesChannel, Levels: 4, Interval: kline.OneHour}, p)
	assert.Equal(t, "Candles", r.Method)
	assert.Equal(t, "H1", r.Params.Period)
	assert.Equal(t, 4, r.Params.Limit)
	assert.Equal(t, "BTC-USD", (r.Params.Symbol))

	r = subToReq(&subscription.Subscription{Channel: subscription.AllTradesChannel, Levels: 150})
	assert.Equal(t, "Trades", r.Method)
	assert.Equal(t, 150, r.Params.Limit)

	assert.PanicsWithError(t,
		"subscription channel not supported: myTrades",
		func() { subToReq(&subscription.Subscription{Channel: subscription.MyTradesChannel}, p) },
		"should panic on invalid channel",
	)
}
