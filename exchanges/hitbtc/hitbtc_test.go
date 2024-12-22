package hitbtc

import (
	"context"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var (
	h          = &HitBTC{}
	wsSetupRan bool
)

// Please supply your own APIKEYS here for due diligence testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

func TestMain(m *testing.M) {
	h.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("HitBTC load config error", err)
	}
	hitbtcConfig, err := cfg.GetExchangeConfig("HitBTC")
	if err != nil {
		log.Fatal("HitBTC Setup() init error")
	}
	hitbtcConfig.API.AuthenticatedSupport = true
	hitbtcConfig.API.AuthenticatedWebsocketSupport = true
	hitbtcConfig.API.Credentials.Key = apiKey
	hitbtcConfig.API.Credentials.Secret = apiSecret
	h.Websocket = sharedtestvalues.NewTestWebsocket()
	err = h.Setup(hitbtcConfig)
	if err != nil {
		log.Fatal("HitBTC setup error", err)
	}

	err = h.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		log.Fatal("HitBTC setup error", err)
	}

	os.Exit(m.Run())
}

func TestGetOrderbook(t *testing.T) {
	_, err := h.GetOrderbook(context.Background(), "BTCUSD", 50)
	if err != nil {
		t.Error("Test failed - HitBTC GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := h.GetTrades(context.Background(), "BTCUSD", "", "", 0, 0, 0, 0)
	if err != nil {
		t.Error("Test failed - HitBTC GetTradeHistory() error", err)
	}
}

func TestGetChartCandles(t *testing.T) {
	_, err := h.GetCandles(context.Background(),
		"BTCUSD", "", "D1", time.Now().Add(-24*time.Hour), time.Now())
	if err != nil {
		t.Error("Test failed - HitBTC GetChartData() error", err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()

	pair, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Now().Add(-time.Hour * 6)
	end := time.Now()
	_, err = h.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneMin, startTime, end)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Fatal(err)
	}
	startTime := time.Unix(1546300800, 0)
	end := time.Unix(1577836799, 0)
	_, err = h.GetHistoricCandlesExtended(context.Background(), pair, asset.Spot, kline.OneHour, startTime, end)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := h.GetCurrencies(context.Background())
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

// TestGetFeeByTypeOfflineTradeFee logic test
func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	_, err := h.GetFeeByType(context.Background(), feeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	if !sharedtestvalues.AreAPICredentialsSet(h) {
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
	err = h.CurrencyPairs.StorePairs(asset.Spot, pairs, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.UpdateTicker(context.Background(), pairs[0], asset.Spot)
	if err != nil {
		t.Error(err)
	}
	_, err = h.FetchTicker(context.Background(), pairs[1], asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	err := h.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}

	enabled, err := h.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	for j := range enabled {
		_, err = h.FetchTicker(context.Background(), enabled[j], asset.Spot)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestGetAllTickers(t *testing.T) {
	_, err := h.GetTickers(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetSingularTicker(t *testing.T) {
	_, err := h.GetTicker(context.Background(), "BTCUSD")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFee(t *testing.T) {
	var feeBuilder = setFeeBuilder()
	if sharedtestvalues.AreAPICredentialsSet(h) {
		// CryptocurrencyTradeFee Basic
		if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}

		// CryptocurrencyTradeFee High quantity
		feeBuilder = setFeeBuilder()
		feeBuilder.Amount = 1000
		feeBuilder.PurchasePrice = 1000
		if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyTradeFee IsMaker
		feeBuilder = setFeeBuilder()
		feeBuilder.IsMaker = true
		if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyTradeFee Negative purchase price
		feeBuilder = setFeeBuilder()
		feeBuilder.PurchasePrice = -1000
		if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyWithdrawalFee Basic
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
		// CryptocurrencyWithdrawalFee Invalid currency
		feeBuilder = setFeeBuilder()
		feeBuilder.Pair.Base = currency.NewCode("hello")
		feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
		if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
			t.Error(err)
		}
	}

	// CryptocurrencyDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyDepositFee
	feeBuilder.Pair.Base = currency.BTC
	feeBuilder.Pair.Quote = currency.LTC
	if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankDepositFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankDepositFee
	if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}

	// InternationalBankWithdrawalFee Basic
	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.InternationalBankWithdrawalFee
	feeBuilder.FiatCurrency = currency.USD
	if _, err := h.GetFee(context.Background(), feeBuilder); err != nil {
		t.Error(err)
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := h.FormatWithdrawPermissions()
	if withdrawPermissions != expectedResult {
		t.Errorf("Expected: %s, Received: %s", expectedResult, withdrawPermissions)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		Pairs:     []currency.Pair{currency.NewPair(currency.ETH, currency.BTC)},
		AssetType: asset.Spot,
		Side:      order.AnySide,
	}

	_, err := h.GetActiveOrders(context.Background(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Could not get open orders: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Pairs:     []currency.Pair{currency.NewPair(currency.ETH, currency.BTC)},
		Side:      order.AnySide,
	}

	_, err := h.GetOrderHistory(context.Background(), &getOrdersRequest)
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Could not get order history: %s", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

// Any tests below this line have the ability to impact your orders on the exchange. Enable canManipulateRealOrders to run them
// ----------------------------------------------------------------------------------------------------------------------------
func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	var orderSubmission = &order.Submit{
		Exchange: h.Name,
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
	response, err := h.SubmitOrder(context.Background(), orderSubmission)
	if sharedtestvalues.AreAPICredentialsSet(h) && (err != nil || response.Status != order.New) {
		t.Errorf("Order failed to be placed: %v", err)
	} else if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	err := h.CancelOrder(context.Background(), orderCancellation)
	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	currencyPair := currency.NewPair(currency.LTC, currency.BTC)
	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currencyPair,
		AssetType:     asset.Spot,
	}

	resp, err := h.CancelAllOrders(context.Background(), orderCancellation)

	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Could not cancel orders: %v", err)
	}

	if len(resp.Status) > 0 {
		t.Errorf("%v orders failed to cancel", len(resp.Status))
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	_, err := h.ModifyOrder(context.Background(),
		&order.Modify{AssetType: asset.Spot})
	if err == nil {
		t.Error("ModifyOrder() Expected error")
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange:    h.Name,
		Amount:      -1,
		Currency:    currency.BTC,
		Description: "WITHDRAW IT ALL",
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}

	_, err := h.WithdrawCryptocurrencyFunds(context.Background(),
		&withdrawCryptoRequest)
	if !sharedtestvalues.AreAPICredentialsSet(h) && err == nil {
		t.Error("Expecting an error when no keys are set")
	}
	if sharedtestvalues.AreAPICredentialsSet(h) && err != nil {
		t.Errorf("Withdraw failed to be placed: %v", err)
	}
}

func TestWithdrawFiat(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFunds(context.Background(), &withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, h, canManipulateRealOrders)

	var withdrawFiatRequest = withdraw.Request{}
	_, err := h.WithdrawFiatFundsToInternationalBank(context.Background(),
		&withdrawFiatRequest)
	if err != common.ErrFunctionNotSupported {
		t.Errorf("Expected '%v', received: '%v'", common.ErrFunctionNotSupported, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	if sharedtestvalues.AreAPICredentialsSet(h) {
		_, err := h.GetDepositAddress(context.Background(), currency.XRP, "", "")
		if err != nil {
			t.Error("GetDepositAddress() error", err)
		}
	} else {
		_, err := h.GetDepositAddress(context.Background(), currency.BTC, "", "")
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
	if !h.Websocket.IsEnabled() && !h.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(h) {
		t.Skip(stream.ErrWebsocketNotEnabled.Error())
	}

	var dialer websocket.Dialer
	err := h.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	go h.wsReadData()
	err = h.wsLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	timer := time.NewTimer(time.Second)
	select {
	case loginError := <-h.Websocket.DataHandler:
		t.Fatal(loginError)
	case <-timer.C:
	}
	timer.Stop()
	wsSetupRan = true
}

// TestWsCancelOrder dials websocket, sends cancel request.
func TestWsCancelOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := h.wsCancelOrder("ImNotARealOrderID")
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsPlaceOrder dials websocket, sends order submission.
func TestWsPlaceOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := h.wsPlaceOrder(currency.NewPair(currency.LTC, currency.BTC),
		order.Buy.String(),
		1,
		1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsReplaceOrder dials websocket, sends replace order request.
func TestWsReplaceOrder(t *testing.T) {
	setupWsAuth(t)
	if !canManipulateRealOrders {
		t.Skip("canManipulateRealOrders false, skipping test")
	}
	_, err := h.wsReplaceOrder("ImNotARealOrderID", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetActiveOrders dials websocket, sends get active orders request.
func TestWsGetActiveOrders(t *testing.T) {
	setupWsAuth(t)
	if _, err := h.wsGetActiveOrders(); err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestWsGetTradingBalance(t *testing.T) {
	setupWsAuth(t)
	if _, err := h.wsGetTradingBalance(); err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestWsGetTrades(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetTrades(currency.NewPair(currency.ETH, currency.BTC), 1000, "ASC", "id")
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetTradingBalance dials websocket, sends get trading balance request.
func TestWsGetSymbols(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetSymbols(currency.NewPair(currency.ETH, currency.BTC))
	if err != nil {
		t.Fatal(err)
	}
}

// TestWsGetCurrencies dials websocket, sends get trading balance request.
func TestWsGetCurrencies(t *testing.T) {
	setupWsAuth(t)
	_, err := h.wsGetCurrencies(currency.BTC)
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
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
	err := h.wsHandleData(pressXToJSON)
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
	err := h.wsHandleData(pressXToJSON)
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
	err = h.wsHandleData(pressXToJSON)
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
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
  "id": 123
}`)
	err := h.wsHandleData(pressXToJSON)
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
	err := h.wsHandleData(pressXToJSON)
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
	err = h.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func Test_FormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		interval kline.Interval
		output   string
	}{
		{
			"OneMin",
			kline.OneMin,
			"M1",
		},
		{
			"OneDay",
			kline.OneDay,
			"D1",
		},
		{
			"SevenDay",
			kline.SevenDay,
			"D7",
		},
		{
			"OneMonth",
			kline.OneMonth,
			"1M",
		},
	}

	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ret, err := formatExchangeKlineInterval(test.interval)
			if err != nil {
				t.Fatal(err)
			}
			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTCUSD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = h.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
	// longer term
	_, err = h.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot,
		time.Now().Add(-time.Minute*60*200),
		time.Now().Add(-time.Minute*60*199))
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.GetActiveOrderByClientOrderID(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, h)

	_, err := h.GetOrderInfo(context.Background(), "1234", currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	if _, err := h.FetchTradablePairs(context.Background(), asset.Spot); err != nil {
		t.Fatal(err)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, h)
	for _, a := range h.GetAssetTypes(false) {
		pairs, err := h.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := h.GetCurrencyTradeURL(context.Background(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	h := new(HitBTC)
	require.NoError(t, testexch.Setup(h), "Test instance Setup must not error")

	h.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, h.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")
	subs, err := h.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	exp := subscription.List{}
	pairs, err := h.GetEnabledPairs(asset.Spot)
	require.NoErrorf(t, err, "GetEnabledPairs must not error")
	for _, s := range h.Features.Subscriptions {
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
