package btcmarkets

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
)

var b = &BTCMarkets{}

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
	BTCAUD                  = "BTC-AUD"
	LTCAUD                  = "LTC-AUD"
	ETHAUD                  = "ETH-AUD"
	fakePair                = "Fake-USDT"
	bid                     = "bid"
)

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	bConfig, err := cfg.GetExchangeConfig("BTC Markets")
	if err != nil {
		log.Fatal(err)
	}
	bConfig.API.Credentials.Key = apiKey
	bConfig.API.Credentials.Secret = apiSecret
	bConfig.API.AuthenticatedSupport = true
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(bConfig)
	if err != nil {
		log.Fatal(err)
	}
	err = b.ValidateAPICredentials(context.Background(), asset.Spot)
	if err != nil {
		fmt.Println("API credentials are invalid:", err)
		b.API.AuthenticatedSupport = false
		b.API.AuthenticatedWebsocketSupport = false
	}
	os.Exit(m.Run())
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets(t.Context())
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(t.Context(), BTCAUD)
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades(t.Context(), BTCAUD, 0, 0, 5)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(t.Context(), BTCAUD, 2)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetMarketCandles(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketCandles(t.Context(),
		BTCAUD, "1h", time.Now().UTC().Add(-time.Hour*24), time.Now().UTC(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	temp, err := currency.NewPairsFromStrings([]string{LTCAUD, BTCAUD})
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetTickers(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMultipleOrderbooks(t *testing.T) {
	t.Parallel()
	temp := []string{BTCAUD, LTCAUD, ETHAUD}
	_, err := b.GetMultipleOrderbooks(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetCurrentServerTime(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := b.GetServerTime(t.Context(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetAccountBalance(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTradingFees(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTradeHistory(t.Context(), ETHAUD, "", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTradeHistory(t.Context(), BTCAUD, "", -1, -1, 1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTradeHistory(t.Context(), fakePair, "", -1, -1, -1)
	if err == nil {
		t.Error("expected an error due to invalid trading pair")
	}
}

func TestGetTradeByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetTradeByID(t.Context(), "4712043732")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := b.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  b.Name,
		Price:     100,
		Amount:    1,
		Type:      order.TrailingStop,
		AssetType: asset.Spot,
		Side:      order.Bid,
		Pair:      currency.NewPair(currency.BTC, currency.AUD),
		PostOnly:  true,
	})
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, order.ErrTypeIsInvalid)
	}
	_, err = b.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  b.Name,
		Price:     100,
		Amount:    1,
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.AnySide,
		Pair:      currency.NewPair(currency.BTC, currency.AUD),
		PostOnly:  true,
	})
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, order.ErrSideIsInvalid)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err = b.SubmitOrder(t.Context(), &order.Submit{
		Exchange:  b.Name,
		Price:     100,
		Amount:    1,
		Type:      order.Limit,
		AssetType: asset.Spot,
		Side:      order.Bid,
		Pair:      currency.NewPair(currency.BTC, currency.AUD),
		PostOnly:  true,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.NewOrder(t.Context(), 100, 1, 0, 0, BTCAUD, limit, bidSide, "", "", "", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetOrders(t.Context(), "", -1, -1, 2, false)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrders(t.Context(), LTCAUD, -1, -1, -1, true)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	temp := []string{BTCAUD, LTCAUD}
	_, err := b.CancelAllOpenOrdersByPairs(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
	temp = []string{BTCAUD, fakePair}
	_, err = b.CancelAllOpenOrdersByPairs(t.Context(), temp)
	if err == nil {
		t.Error("expected an error due to invalid marketID")
	}
}

func TestFetchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.FetchOrder(t.Context(), "4477045999")
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchOrder(t.Context(), "696969")
	if err == nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.RemoveOrder(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestListWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.ListWithdrawals(t.Context(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetWithdrawal(t.Context(), "4477381751")
	if err != nil {
		t.Error(err)
	}
}

func TestListDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.ListDeposits(t.Context(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeposit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetDeposit(t.Context(), "4476769607")
	if err != nil {
		t.Error(err)
	}
}

func TestListTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.ListTransfers(t.Context(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetTransfer(t.Context(), "4476769607")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTransfer(t.Context(), "6969696")
	if err == nil {
		t.Error("expected an error due to invalid transferID")
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.FetchDepositAddress(t.Context(), currency.XRP, -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchDepositAddress(t.Context(), currency.NewCode("MOOCOW"), -1, -1, -1)
	if err != nil {
		t.Error("expected an error due to invalid assetID")
	}
}

func TestGetWithdrawalFees(t *testing.T) {
	t.Parallel()
	_, err := b.GetWithdrawalFees(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.ListAssets(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetTransactions(t.Context(), "", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewReport(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.CreateNewReport(t.Context(), "TransactionReport", "json")
	if err != nil {
		t.Error(err)
	}
}

func TestGetReport(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetReport(t.Context(), "1kv38epne5v7lek9f18m60idg6")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestWithdaw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err := b.RequestWithdraw(t.Context(), "BTC", 1, "sdjflajdslfjld", "", "", "", "")
	if err == nil {
		t.Error("expected an error due to invalid toAddress")
	}
}

func TestBatchPlaceCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	var temp []PlaceBatch
	o := PlaceBatch{
		MarketID:  BTCAUD,
		Amount:    11000,
		Price:     1,
		OrderType: order.Limit.String(),
		Side:      bid,
	}
	_, err := b.BatchPlaceCancelOrders(t.Context(), nil, append(temp, o))
	if err != nil {
		t.Error(err)
	}
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	temp := []string{"4477045999", "4477381751", "4476769607"}
	_, err := b.GetBatchTrades(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	temp := []string{"4477045999", "4477381751", "4477381751"}
	_, err := b.CancelBatch(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
		Side:      order.Buy,
		AssetType: asset.Spot,
		Type:      order.AnyType,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := b.UpdateOrderbook(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := b.UpdateTicker(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := b.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)

	_, err := b.GetActiveOrders(t.Context(),
		&order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWsTicker(t *testing.T) {
	pressXToJSON := []byte(`{ "marketId": "BTC-AUD",
    "timestamp": "2019-04-08T18:56:17.405Z",
    "bestBid": "7309.12",
    "bestAsk": "7326.88",
    "lastPrice": "7316.81",
    "volume24h": "299.12936654",
    "messageType": "tick"
  }`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSTrade(t *testing.T) {
	t.Parallel()

	b := new(BTCMarkets) //nolint:govet // Intentional shadow to avoid future copy/paste mistakes
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	fErrs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsAllTrades.json", b.wsHandleData)
	require.Equal(t, 2, len(fErrs), "Must get correct number of errors from wsHandleData")
	assert.ErrorIs(t, fErrs[0].Err, order.ErrSideIsInvalid, "Side.UnmarshalJSON errors should propagate correctly")
	assert.ErrorContains(t, fErrs[0].Err, "WRONG", "Side.UnmarshalJSON errors should propagate correctly")
	assert.ErrorIs(t, fErrs[1].Err, order.ErrSideIsInvalid, "wsHandleData errors should propagate correctly")
	assert.ErrorContains(t, fErrs[1].Err, "ANY", "wsHandleData errors should propagate correctly")
	close(b.Websocket.DataHandler)

	exp := []trade.Data{
		{
			Exchange:     b.Name,
			CurrencyPair: currency.NewPairWithDelimiter("BTC", "AUD", currency.DashDelimiter),
			Timestamp:    time.Date(2025, 3, 13, 8, 27, 55, 691000000, time.UTC),
			Price:        131200.34,
			Amount:       0.00151228,
			Side:         order.Buy,
			TID:          "7006384466",
			AssetType:    asset.Spot,
		},
		{
			Exchange:     b.Name,
			CurrencyPair: currency.NewPairWithDelimiter("BTC", "AUD", currency.DashDelimiter),
			Timestamp:    time.Date(2025, 3, 13, 8, 28, 2, 273000000, time.UTC),
			Price:        131065.01,
			Amount:       0.05,
			Side:         order.Sell,
			TID:          "7006384467",
			AssetType:    asset.Spot,
		},
	}
	require.Len(t, b.Websocket.DataHandler, 2, "Must see correct number of trades")

	for resp := range b.Websocket.DataHandler {
		switch v := resp.(type) {
		case trade.Data:
			i := 1 - len(b.Websocket.DataHandler)
			require.Equalf(t, exp[i], v, "Trade[%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T(%s)", v, v)
		}
	}
}

func TestWsFundChange(t *testing.T) {
	pressXToJSON := []byte(`{
  "fundtransferId": 276811,
  "type": "Deposit",
  "status": "Complete",
  "timestamp": "2019-04-16T01:38:02.931Z",
  "amount": "0.001",
  "currency": "BTC",
  "fee": "0",
  "messageType": "fundChange"
}`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsOrderbookUpdate(t *testing.T) {
	pressXToJSON := []byte(`{ "marketId": "LTC-AUD",
    "snapshot": true,
    "timestamp": "2020-01-08T19:47:13.986Z",
    "snapshotId": 1578512833978000,
      "bids":
      [ [ "99.57", "0.55", 1 ],
        [ "97.62", "3.20", 2 ],
        [ "97.07", "0.9", 1 ],
        [ "96.7", "1.9", 1 ],
        [ "95.8", "7.0", 1 ] ],
      "asks":
        [ [ "100", "3.79", 3 ],
          [ "101", "6.32", 2 ] ],
      "messageType": "orderbookUpdate"
  }`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`  { "marketId": "LTC-AUD",
    "timestamp": "2020-01-08T19:47:24.054Z",
    "snapshotId": 1578512844045000,
    "bids":  [ ["99.81", "1.2", 1 ], ["95.8", "0", 0 ]],
    "asks": [ ["100", "3.2", 2 ] ],
    "messageType": "orderbookUpdate",
	"checksum": "2513007604"
  }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWsHeartbeats(t *testing.T) {
	pressXToJSON := []byte(`{
  "messageType": "error",
  "code": 3,
  "message": "invalid channel names"
}`)
	err := b.wsHandleData(pressXToJSON)
	if err == nil {
		t.Error("expected error")
	}

	pressXToJSON = []byte(`{ 
"messageType": "error",
"code": 3,
"message": "invalid marketIds"
}`)
	err = b.wsHandleData(pressXToJSON)
	if err == nil {
		t.Error("expected error")
	}

	pressXToJSON = []byte(`{ 
"messageType": "error",
"code": 1,
"message": "authentication failed. invalid key"
}`)
	err = b.wsHandleData(pressXToJSON)
	if err == nil {
		t.Error("expected error")
	}
}

func TestWsOrders(t *testing.T) {
	pressXToJSON := []byte(`{ 
	"orderId": 79003,
    "marketId": "BTC-AUD",
    "side": "Bid",
    "type": "Limit",
    "openVolume": "1",
    "status": "Placed",
    "triggerStatus": "",
    "trades": [],
    "timestamp": "2019-04-08T20:41:19.339Z",
    "messageType": "orderChange"
  }`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(` { 
	"orderId": 79033,
    "marketId": "BTC-AUD",
    "side": "Bid",
    "type": "Limit",
    "openVolume": "0",
    "status": "Fully Matched",
    "triggerStatus": "",
    "trades": [{
               "tradeId":31727,
               "price":"0.1634",
               "volume":"10",
               "fee":"0.001",
               "liquidityType":"Taker"
             }],
    "timestamp": "2019-04-08T20:50:39.658Z",
    "messageType": "orderChange"
  }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(` { 
	"orderId": 79003,
    "marketId": "BTC-AUD",
    "side": "Bid",
    "type": "Limit",
    "openVolume": "1",
    "status": "Cancelled",
    "triggerStatus": "",
    "trades": [],
    "timestamp": "2019-04-08T20:41:41.857Z",
    "messageType": "orderChange"
  }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(`  { 
	"orderId": 79003,
    "marketId": "BTC-AUD",
    "side": "Bid",
    "type": "Limit",
    "openVolume": "1",
    "status": "Partially Matched",
    "triggerStatus": "",
    "trades": [{
               "tradeId":31927,
               "price":"0.1634",
               "volume":"5",
               "fee":"0.001",
               "liquidityType":"Taker"
             }],
	"timestamp": "2019-04-08T20:41:41.857Z",
    "messageType": "orderChange"
  }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}

	pressXToJSON = []byte(` { 
	"orderId": 7903,
    "marketId": "BTC-AUD",
    "side": "Bid",
    "type": "Limit",
    "openVolume": "1.2",
    "status": "Placed",
    "triggerStatus": "Triggered",
    "trades": [],
    "timestamp": "2019-04-08T20:41:41.857Z",
    "messageType": "orderChange"
  }`)
	err = b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestBTCMarkets_GetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair, err := currency.NewPairFromString(BTCAUD)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(t.Context(), pair, asset.Spot, kline.OneHour, time.Now().Add(-time.Hour*24).UTC(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandles(t.Context(), pair, asset.Spot, kline.FifteenMin, time.Now().Add(-time.Hour*24).UTC(), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
}

func TestBTCMarkets_GetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.Now().AddDate(0, 0, -1)
	end := time.Now()
	pair, err := currency.NewPairFromString(BTCAUD)
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.GetHistoricCandlesExtended(t.Context(), pair, asset.Spot, kline.OneHour, start, end)
	if err != nil {
		t.Fatal(err)
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
			"1m",
		},
		{
			"OneDay",
			kline.OneDay,
			"1d",
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			ret := b.FormatExchangeKlineInterval(test.interval)

			if ret != test.output {
				t.Fatalf("unexpected result return expected: %v received: %v", test.output, ret)
			}
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC-AUD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetRecentTrades(t.Context(), currencyPair, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	currencyPair, err := currency.NewPairFromString("BTC-AUD")
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricTrades(t.Context(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestChecksum(t *testing.T) {
	b := &orderbook.Base{
		Asks: []orderbook.Tranche{
			{Price: 0.3965, Amount: 44149.815},
			{Price: 0.3967, Amount: 16000.0},
		},
		Bids: []orderbook.Tranche{
			{Price: 0.396, Amount: 51.0},
			{Price: 0.396, Amount: 25.0},
			{Price: 0.3958, Amount: 18570.0},
		},
	}

	expecting := uint32(3802968298)
	err := checksum(b, expecting)
	if err != nil {
		t.Fatal(err)
	}
	err = checksum(b, uint32(1223123))
	if !errors.Is(err, errChecksumFailure) {
		t.Errorf("received '%v', expected '%v'", err, errChecksumFailure)
	}
}

func TestTrim(t *testing.T) {
	testCases := []struct {
		Value    float64
		Expected string
	}{
		{Value: 0.1234, Expected: "1234"},
		{Value: 0.00001234, Expected: "1234"},
		{Value: 32.00001234, Expected: "3200001234"},
		{Value: 0, Expected: ""},
		{Value: 0.0, Expected: ""},
		{Value: 1.0, Expected: "1"},
		{Value: 0.3965, Expected: "3965"},
		{Value: 16000.0, Expected: "16000"},
		{Value: 0.0019, Expected: "19"},
		{Value: 1.01, Expected: "101"},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			received := trim(tt.Value)
			if received != tt.Expected {
				t.Fatalf("received: %v but expected: %v", received, tt.Expected)
			}
		})
	}
}

func TestFormatOrderType(t *testing.T) {
	t.Parallel()
	_, err := b.formatOrderType(0)
	if !errors.Is(err, order.ErrTypeIsInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, order.ErrTypeIsInvalid)
	}

	r, err := b.formatOrderType(order.Limit)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r != limit {
		t.Fatal("unexpected value")
	}

	r, err = b.formatOrderType(order.Market)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r != market {
		t.Fatal("unexpected value")
	}

	r, err = b.formatOrderType(order.StopLimit)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r != stopLimit {
		t.Fatal("unexpected value")
	}

	r, err = b.formatOrderType(order.Stop)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r != stop {
		t.Fatal("unexpected value")
	}

	r, err = b.formatOrderType(order.TakeProfit)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if r != takeProfit {
		t.Fatal("unexpected value")
	}
}

func TestFormatOrderSide(t *testing.T) {
	t.Parallel()
	_, err := b.formatOrderSide(255)
	if !errors.Is(err, order.ErrSideIsInvalid) {
		t.Fatalf("received: '%v' but expected: '%v'", err, order.ErrSideIsInvalid)
	}

	f, err := b.formatOrderSide(order.Bid)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if f != bidSide {
		t.Fatal("unexpected value")
	}

	f, err = b.formatOrderSide(order.Ask)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if f != askSide {
		t.Fatal("unexpected value")
	}
}

func TestGetTimeInForce(t *testing.T) {
	t.Parallel()
	f := b.getTimeInForce(&order.Submit{})
	if f != "" {
		t.Fatal("unexpected value")
	}

	f = b.getTimeInForce(&order.Submit{ImmediateOrCancel: true})
	if f != immediateOrCancel {
		t.Fatalf("received: '%v' but expected: '%v'", f, immediateOrCancel)
	}

	f = b.getTimeInForce(&order.Submit{FillOrKill: true})
	if f != fillOrKill {
		t.Fatalf("received: '%v' but expected: '%v'", f, fillOrKill)
	}
}

func TestReplaceOrder(t *testing.T) {
	t.Parallel()
	_, err := b.ReplaceOrder(t.Context(), "", "bro", 0, 0)
	if !errors.Is(err, errInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAmount)
	}

	_, err = b.ReplaceOrder(t.Context(), "", "bro", 1, 0)
	if !errors.Is(err, errInvalidAmount) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errInvalidAmount)
	}

	_, err = b.ReplaceOrder(t.Context(), "", "bro", 1, 1)
	if !errors.Is(err, errIDRequired) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errIDRequired)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	_, err = b.ReplaceOrder(t.Context(), "8207096301", "bruh", 100000, 0.001)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}
}

func TestWrapperModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := b.ModifyOrder(t.Context(), &order.Modify{})
	if !errors.Is(err, order.ErrPairIsEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, order.ErrPairIsEmpty)
	}

	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)

	mo, err := b.ModifyOrder(t.Context(), &order.Modify{
		Pair:          currency.NewPair(currency.BTC, currency.AUD),
		AssetType:     asset.Spot,
		Price:         100000,
		Amount:        0.001,
		OrderID:       "8207123461",
		ClientOrderID: "bruh3",
	})
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if mo == nil {
		t.Fatal("expected data return")
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := b.UpdateOrderExecutionLimits(t.Context(), asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	err = b.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	lim, err := b.ExecutionLimits.GetOrderExecutionLimits(asset.Spot, currency.NewPair(currency.BTC, currency.AUD))
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if lim == (order.MinMaxLevel{}) {
		t.Fatal("expected value return")
	}
}

func TestConvertToKlineCandle(t *testing.T) {
	t.Parallel()

	_, err := convertToKlineCandle(nil)
	if !errors.Is(err, errFailedToConvertToCandle) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errFailedToConvertToCandle)
	}

	data := [6]string{time.RFC3339[:len(time.RFC3339)-5], "1.0", "2", "3", "4", "5"}

	candle, err := convertToKlineCandle(&data)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if candle.Time.IsZero() {
		t.Fatal("time unset")
	}

	if candle.Open != 1 {
		t.Fatalf("received: '%v' but expected: '%v'", candle.Open, 1)
	}

	if candle.High != 2 {
		t.Fatalf("received: '%v' but expected: '%v'", candle.High, 2)
	}

	if candle.Low != 3 {
		t.Fatalf("received: '%v' but expected: '%v'", candle.Low, 3)
	}

	if candle.Close != 4 {
		t.Fatalf("received: '%v' but expected: '%v'", candle.Close, 4)
	}

	if candle.Volume != 5 {
		t.Fatalf("received: '%v' but expected: '%v'", candle.Volume, 5)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b)
	_, err := b.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, b, canManipulateRealOrders)
	_, err := b.CancelBatchOrders(t.Context(), []order.Cancel{
		{
			OrderID:   "1234",
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.BTC, currency.AUD),
		},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, b)
	for _, a := range b.GetAssetTypes(false) {
		pairs, err := b.CurrencyPairs.GetPairs(a, false)
		require.NoError(t, err, "cannot get pairs for %s", a)
		require.NotEmpty(t, pairs, "no pairs for %s", a)
		resp, err := b.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoError(t, err)
		assert.NotEmpty(t, resp)
	}
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	b := new(BTCMarkets)
	require.NoError(t, testexch.Setup(b), "Test instance Setup must not error")
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "_"), currency.NewPairWithDelimiter("ETH", "BTC", "_")}
	require.NoError(t, b.CurrencyPairs.StorePairs(asset.Spot, p, false))
	require.NoError(t, b.CurrencyPairs.StorePairs(asset.Spot, p, true))
	b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, b.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")
	subs, err := b.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	pairs, err := b.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	exp := subscription.List{}
	for _, baseSub := range b.Features.Subscriptions {
		s := baseSub.Clone()
		if !s.Authenticated && s.Channel != subscription.HeartbeatChannel {
			s.Pairs = pairs
		}
		s.QualifiedChannel = channelName(s)
		exp = append(exp, s)
	}
	testsubs.EqualLists(t, exp, subs)
	assert.PanicsWithError(t,
		"subscription channel not supported: wibble",
		func() { channelName(&subscription.Subscription{Channel: "wibble"}) },
		"should panic on invalid channel",
	)
}
