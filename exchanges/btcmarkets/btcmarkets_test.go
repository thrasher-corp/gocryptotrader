package btcmarkets

import (
	"encoding/base64"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
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

var e *Exchange

// Please supply your own keys here to do better tests
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var spotTestPair = currency.NewPair(currency.BTC, currency.AUD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter})

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("BTCMarkets Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	os.Exit(m.Run())
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkets(t.Context())
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), spotTestPair.String())
	assert.NoError(t, err, "GetTicker should not error")
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetTrades(t.Context(), spotTestPair.String(), 0, 0, 5)
	assert.NoError(t, err, "GetTrades should not error")
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), spotTestPair.String(), 2)
	assert.NoError(t, err, "GetOrderbook should not error")
}

func TestGetMarketCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketCandles(t.Context(), spotTestPair.String(), "1h", time.Now().UTC().Add(-time.Hour*24), time.Now().UTC(), -1, -1, -1)
	assert.NoError(t, err, "GetMarketCandles should not error")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	pairs := currency.Pairs{spotTestPair, currency.NewPair(currency.LTC, currency.AUD)}
	_, err := e.GetTickers(t.Context(), pairs)
	assert.NoError(t, err, "GetTickers should not error")
}

func TestGetMultipleOrderbooks(t *testing.T) {
	t.Parallel()
	marketIDs := []string{spotTestPair.String(), "LTC-AUD", "ETH-AUD"}
	_, err := e.GetMultipleOrderbooks(t.Context(), marketIDs)
	assert.NoError(t, err, "GetMultipleOrderbooks should not error")
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentServerTime(t.Context())
	assert.NoError(t, err, "GetCurrentServerTime should not error")
}

func TestWrapperGetServerTime(t *testing.T) {
	t.Parallel()
	st, err := e.GetServerTime(t.Context(), asset.Spot)
	require.NoError(t, err)

	if st.IsZero() {
		t.Fatal("expected a time")
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountBalance(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingFees(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetTradingFees(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetTradeHistory(t.Context(), spotTestPair.String(), "", -1, -1, 1)
	assert.NoError(t, err, "GetTradeHistory should not error")
}

func TestGetTradeByID(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetTradeByID(t.Context(), "4712043732")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:    e.Name,
		Price:       100,
		Amount:      1,
		Type:        order.TrailingStop,
		AssetType:   asset.Spot,
		Side:        order.Bid,
		Pair:        currency.NewPair(currency.BTC, currency.AUD),
		TimeInForce: order.PostOnly,
	})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:    e.Name,
		Price:       100,
		Amount:      1,
		Type:        order.Limit,
		AssetType:   asset.Spot,
		Side:        order.AnySide,
		Pair:        currency.NewPair(currency.BTC, currency.AUD),
		TimeInForce: order.PostOnly,
	})
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err = e.SubmitOrder(t.Context(), &order.Submit{
		Exchange:    e.Name,
		Price:       100,
		Amount:      1,
		Type:        order.Limit,
		AssetType:   asset.Spot,
		Side:        order.Bid,
		Pair:        currency.NewPair(currency.BTC, currency.AUD),
		TimeInForce: order.PostOnly,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.NewOrder(t.Context(), 100, 1, 0, 0, spotTestPair.String(), limit, bidSide, "", "", "", true)
	assert.NoError(t, err, "NewOrder should not error")
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetOrders(t.Context(), "", -1, -1, 2, false)
	assert.NoError(t, err, "GetOrders should not error")
	_, err = e.GetOrders(t.Context(), spotTestPair.String(), -1, -1, -1, true)
	assert.NoError(t, err, "GetOrders should not error")
}

func TestCancelOpenOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	pairs := []string{spotTestPair.String(), spotTestPair.String()}
	_, err := e.CancelAllOpenOrdersByPairs(t.Context(), pairs)
	assert.NoError(t, err, "CancelAllOpenOrdersByPairs should not error")
}

func TestFetchOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.FetchOrder(t.Context(), "4477045999")
	if err != nil {
		t.Error(err)
	}
	_, err = e.FetchOrder(t.Context(), "696969")
	if err == nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err := e.RemoveOrder(t.Context(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestListWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.ListWithdrawals(t.Context(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetWithdrawal(t.Context(), "4477381751")
	if err != nil {
		t.Error(err)
	}
}

func TestListDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.ListDeposits(t.Context(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeposit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetDeposit(t.Context(), "4476769607")
	if err != nil {
		t.Error(err)
	}
}

func TestListTransfers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.ListTransfers(t.Context(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetTransfer(t.Context(), "4476769607")
	if err != nil {
		t.Error(err)
	}
	_, err = e.GetTransfer(t.Context(), "6969696")
	if err == nil {
		t.Error("expected an error due to invalid transferID")
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.FetchDepositAddress(t.Context(), currency.XRP, -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = e.FetchDepositAddress(t.Context(), currency.NewCode("MOOCOW"), -1, -1, -1)
	if err != nil {
		t.Error("expected an error due to invalid assetID")
	}
}

func TestGetWithdrawalFees(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalFees(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.ListAssets(t.Context())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetTransactions(t.Context(), "", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewReport(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.CreateNewReport(t.Context(), "TransactionReport", "json")
	if err != nil {
		t.Error(err)
	}
}

func TestGetReport(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetReport(t.Context(), "1kv38epne5v7lek9f18m60idg6")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestWithdaw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err := e.RequestWithdraw(t.Context(), "BTC", 1, "sdjflajdslfjld", "", "", "", "")
	if err == nil {
		t.Error("expected an error due to invalid toAddress")
	}
}

func TestBatchPlaceCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	var temp []PlaceBatch
	o := PlaceBatch{
		MarketID:  spotTestPair.String(),
		Amount:    11000,
		Price:     1,
		OrderType: order.Limit.String(),
		Side:      order.Bid.String(),
	}
	_, err := e.BatchPlaceCancelOrders(t.Context(), nil, append(temp, o))
	if err != nil {
		t.Error(err)
	}
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	temp := []string{"4477045999", "4477381751", "4476769607"}
	_, err := e.GetBatchTrades(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatch(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	temp := []string{"4477045999", "4477381751", "4477381751"}
	_, err := e.CancelBatch(t.Context(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{
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
	_, err := e.UpdateOrderbook(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := e.UpdateTicker(t.Context(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := e.UpdateTickers(t.Context(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	_, err := e.GetActiveOrders(t.Context(),
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
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestWSTrade(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	fErrs := testexch.FixtureToDataHandlerWithErrors(t, "testdata/wsAllTrades.json", e.wsHandleData)
	require.Equal(t, 2, len(fErrs), "Must get correct number of errors from wsHandleData")
	assert.ErrorIs(t, fErrs[0].Err, order.ErrSideIsInvalid, "Side.UnmarshalJSON errors should propagate correctly")
	assert.ErrorContains(t, fErrs[0].Err, "WRONG", "Side.UnmarshalJSON errors should propagate correctly")
	assert.ErrorIs(t, fErrs[1].Err, order.ErrSideIsInvalid, "wsHandleData errors should propagate correctly")
	assert.ErrorContains(t, fErrs[1].Err, "ANY", "wsHandleData errors should propagate correctly")
	e.Websocket.DataHandler.Close()

	exp := []trade.Data{
		{
			Exchange:     e.Name,
			CurrencyPair: currency.NewPairWithDelimiter("BTC", "AUD", currency.DashDelimiter),
			Timestamp:    time.Date(2025, 3, 13, 8, 27, 55, 691000000, time.UTC),
			Price:        131200.34,
			Amount:       0.00151228,
			Side:         order.Buy,
			TID:          "7006384466",
			AssetType:    asset.Spot,
		},
		{
			Exchange:     e.Name,
			CurrencyPair: currency.NewPairWithDelimiter("BTC", "AUD", currency.DashDelimiter),
			Timestamp:    time.Date(2025, 3, 13, 8, 28, 2, 273000000, time.UTC),
			Price:        131065.01,
			Amount:       0.05,
			Side:         order.Sell,
			TID:          "7006384467",
			AssetType:    asset.Spot,
		},
	}
	require.Len(t, e.Websocket.DataHandler.C, 2, "Must see correct number of trades")

	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
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
	err := e.wsHandleData(t.Context(), pressXToJSON)
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
	err := e.wsHandleData(t.Context(), pressXToJSON)
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
	err = e.wsHandleData(t.Context(), pressXToJSON)
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
	err := e.wsHandleData(t.Context(), pressXToJSON)
	if err == nil {
		t.Error("expected error")
	}

	pressXToJSON = []byte(`{ 
"messageType": "error",
"code": 3,
"message": "invalid marketIds"
}`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err == nil {
		t.Error("expected error")
	}

	pressXToJSON = []byte(`{ 
"messageType": "error",
"code": 1,
"message": "authentication failed. invalid key"
}`)
	err = e.wsHandleData(t.Context(), pressXToJSON)
	if err == nil {
		t.Error("expected error")
	}
}

func TestWsOrders(t *testing.T) {
	ctx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{
		Key:    "testkey",
		Secret: base64.StdEncoding.EncodeToString([]byte("testsecret")),
	})
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
	err := e.wsHandleData(ctx, pressXToJSON)
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
	err = e.wsHandleData(ctx, pressXToJSON)
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
	err = e.wsHandleData(ctx, pressXToJSON)
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
	err = e.wsHandleData(ctx, pressXToJSON)
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
	err = e.wsHandleData(ctx, pressXToJSON)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandles(t.Context(), spotTestPair, asset.Spot, kline.OneHour, time.Now().Add(-time.Hour*24), time.Now())
	assert.NoError(t, err, "GetHistoricCandles should not error")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), spotTestPair, asset.Spot, kline.OneHour, time.Now().AddDate(0, 0, -1), time.Now())
	assert.NoError(t, err, "GetHistoricCandlesExtended should not error")
}

func TestFormatExchangeKlineInterval(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		interval kline.Interval
		output   string
	}{
		{
			kline.OneMin,
			"1m",
		},
		{
			kline.OneDay,
			"1d",
		},
	} {
		t.Run(tc.interval.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.output, e.FormatExchangeKlineInterval(tc.interval))
		})
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), spotTestPair, asset.Spot)
	assert.NoError(t, err, "GetRecentTrades should not error")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), spotTestPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestOrderbookChecksum(t *testing.T) {
	b := &orderbook.Book{
		Asks: orderbook.Levels{
			{Price: 0.3965, Amount: 44149.815},
			{Price: 0.3967, Amount: 16000.0},
		},
		Bids: orderbook.Levels{
			{Price: 0.396, Amount: 51.0},
			{Price: 0.396, Amount: 25.0},
			{Price: 0.3958, Amount: 18570.0},
		},
	}
	require.Equal(t, uint32(3802968298), orderbookChecksum(b))
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
	_, err := e.formatOrderType(0)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid)

	r, err := e.formatOrderType(order.Limit)
	require.NoError(t, err)

	if r != limit {
		t.Fatal("unexpected value")
	}

	r, err = e.formatOrderType(order.Market)
	require.NoError(t, err)

	if r != market {
		t.Fatal("unexpected value")
	}

	r, err = e.formatOrderType(order.StopLimit)
	require.NoError(t, err)

	if r != stopLimit {
		t.Fatal("unexpected value")
	}

	r, err = e.formatOrderType(order.Stop)
	require.NoError(t, err)

	if r != stop {
		t.Fatal("unexpected value")
	}

	r, err = e.formatOrderType(order.TakeProfit)
	require.NoError(t, err)

	if r != takeProfit {
		t.Fatal("unexpected value")
	}
}

func TestFormatOrderSide(t *testing.T) {
	t.Parallel()
	_, err := e.formatOrderSide(255)
	require.ErrorIs(t, err, order.ErrSideIsInvalid)

	f, err := e.formatOrderSide(order.Bid)
	require.NoError(t, err)

	if f != bidSide {
		t.Fatal("unexpected value")
	}

	f, err = e.formatOrderSide(order.Ask)
	require.NoError(t, err)

	if f != askSide {
		t.Fatal("unexpected value")
	}
}

func TestGetTimeInForce(t *testing.T) {
	t.Parallel()
	f := e.getTimeInForce(&order.Submit{})
	require.Empty(t, f)

	f = e.getTimeInForce(&order.Submit{TimeInForce: order.ImmediateOrCancel})
	require.Equal(t, "IOC", f)

	f = e.getTimeInForce(&order.Submit{TimeInForce: order.FillOrKill})
	assert.Equal(t, "FOK", f)
}

func TestReplaceOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ReplaceOrder(t.Context(), "", "bro", 0, 0)
	require.ErrorIs(t, err, errInvalidAmount)

	_, err = e.ReplaceOrder(t.Context(), "", "bro", 1, 0)
	require.ErrorIs(t, err, errInvalidAmount)

	_, err = e.ReplaceOrder(t.Context(), "", "bro", 1, 1)
	require.ErrorIs(t, err, errIDRequired)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	_, err = e.ReplaceOrder(t.Context(), "8207096301", "bruh", 100000, 0.001)
	require.NoError(t, err)
}

func TestWrapperModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{})
	require.ErrorIs(t, err, order.ErrPairIsEmpty)

	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)

	mo, err := e.ModifyOrder(t.Context(), &order.Modify{
		Pair:          currency.NewPair(currency.BTC, currency.AUD),
		AssetType:     asset.Spot,
		Price:         100000,
		Amount:        0.001,
		OrderID:       "8207123461",
		ClientOrderID: "bruh3",
	})
	require.NoError(t, err)

	if mo == nil {
		t.Fatal("expected data return")
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error")
			pairs, err := e.CurrencyPairs.GetPairs(a, false)
			require.NoError(t, err, "GetPairs must not error")
			l, err := e.GetOrderExecutionLimits(a, pairs[0])
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			assert.Positive(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive")
		})
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
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
	p := currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "_"), currency.NewPairWithDelimiter("ETH", "BTC", "_")}
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, false))
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, p, true))
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "CanUseAuthenticatedEndpoints must return true")
	subs, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	pairs, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	exp := subscription.List{}
	for _, baseSub := range e.Features.Subscriptions {
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
