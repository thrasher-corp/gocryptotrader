package btcmarkets

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var b BTCMarkets

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
	err = b.ValidateCredentials(context.Background(), asset.Spot)
	if err != nil {
		fmt.Println("API credentials are invalid:", err)
		b.API.AuthenticatedSupport = false
		b.API.AuthenticatedWebsocketSupport = false
	}
	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return b.ValidateAPICredentials(b.GetDefaultCredentials()) == nil
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := b.Start(nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = b.Start(&testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetMarkets(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarkets(context.Background())
	if err != nil {
		t.Error("GetTicker() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker(context.Background(), BTCAUD)
	if err != nil {
		t.Error("GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetTrades(context.Background(), BTCAUD, 0, 0, 5)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook(context.Background(), BTCAUD, 2)
	if err != nil {
		t.Error("GetTrades() error", err)
	}
}

func TestGetMarketCandles(t *testing.T) {
	t.Parallel()
	_, err := b.GetMarketCandles(context.Background(),
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
	_, err = b.GetTickers(context.Background(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetMultipleOrderbooks(t *testing.T) {
	t.Parallel()
	temp := []string{BTCAUD, LTCAUD, ETHAUD}
	_, err := b.GetMultipleOrderbooks(context.Background(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := b.GetServerTime(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetAccountBalance(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradingFees(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTradingFees(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTradeHistory(context.Background(), ETHAUD, "", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTradeHistory(context.Background(), BTCAUD, "", -1, -1, 1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTradeHistory(context.Background(), fakePair, "", -1, -1, -1)
	if err == nil {
		t.Error("expected an error due to invalid trading pair")
	}
}

func TestGetTradeByID(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTradeByID(context.Background(), "4712043732")
	if err != nil {
		t.Error(err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := b.SubmitOrder(context.Background(), &order.Submit{
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
	_, err = b.SubmitOrder(context.Background(), &order.Submit{
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

	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err = b.SubmitOrder(context.Background(), &order.Submit{
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
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.NewOrder(context.Background(), 100, 1, 0, 0, BTCAUD, limit, bidSide, "", "", "", true)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetOrders(context.Background(), "", -1, -1, 2, false)
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetOrders(context.Background(), LTCAUD, -1, -1, -1, true)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOpenOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	temp := []string{BTCAUD, LTCAUD}
	_, err := b.CancelAllOpenOrdersByPairs(context.Background(), temp)
	if err != nil {
		t.Error(err)
	}
	temp = []string{BTCAUD, fakePair}
	_, err = b.CancelAllOpenOrdersByPairs(context.Background(), temp)
	if err == nil {
		t.Error("expected an error due to invalid marketID")
	}
}

func TestFetchOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.FetchOrder(context.Background(), "4477045999")
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchOrder(context.Background(), "696969")
	if err == nil {
		t.Error(err)
	}
}

func TestRemoveOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.RemoveOrder(context.Background(), "")
	if err != nil {
		t.Error(err)
	}
}

func TestListWithdrawals(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListWithdrawals(context.Background(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetWithdrawal(context.Background(), "4477381751")
	if err != nil {
		t.Error(err)
	}
}

func TestListDeposits(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListDeposits(context.Background(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetDeposit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetDeposit(context.Background(), "4476769607")
	if err != nil {
		t.Error(err)
	}
}

func TestListTransfers(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListTransfers(context.Background(), -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransfer(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTransfer(context.Background(), "4476769607")
	if err != nil {
		t.Error(err)
	}
	_, err = b.GetTransfer(context.Background(), "6969696")
	if err == nil {
		t.Error("expected an error due to invalid transferID")
	}
}

func TestFetchDepositAddress(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.FetchDepositAddress(context.Background(), currency.XRP, -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
	_, err = b.FetchDepositAddress(context.Background(), currency.NewCode("MOOCOW"), -1, -1, -1)
	if err != nil {
		t.Error("expected an error due to invalid assetID")
	}
}

func TestGetWithdrawalFees(t *testing.T) {
	t.Parallel()
	_, err := b.GetWithdrawalFees(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestListAssets(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.ListAssets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetTransactions(context.Background(), "", -1, -1, -1)
	if err != nil {
		t.Error(err)
	}
}

func TestCreateNewReport(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.CreateNewReport(context.Background(), "TransactionReport", "json")
	if err != nil {
		t.Error(err)
	}
}

func TestGetReport(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.GetReport(context.Background(), "1kv38epne5v7lek9f18m60idg6")
	if err != nil {
		t.Error(err)
	}
}

func TestRequestWithdaw(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	_, err := b.RequestWithdraw(context.Background(), "BTC", 1, "sdjflajdslfjld", "", "", "", "")
	if err == nil {
		t.Error("expected an error due to invalid toAddress")
	}
}

func TestBatchPlaceCancelOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	var temp []PlaceBatch
	o := PlaceBatch{
		MarketID:  BTCAUD,
		Amount:    11000,
		Price:     1,
		OrderType: order.Limit.String(),
		Side:      bid,
	}
	_, err := b.BatchPlaceCancelOrders(context.Background(), nil, append(temp, o))
	if err != nil {
		t.Error(err)
	}
}

func TestGetBatchTrades(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	temp := []string{"4477045999", "4477381751", "4476769607"}
	_, err := b.GetBatchTrades(context.Background(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelBatch(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.Skip("skipping test, either api keys or manipulaterealorders isnt set correctly")
	}
	temp := []string{"4477045999", "4477381751", "4477381751"}
	_, err := b.CancelBatch(context.Background(), temp)
	if err != nil {
		t.Error(err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}
	_, err := b.FetchAccountInfo(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}

	_, err := b.GetOrderHistory(context.Background(), &order.GetOrdersRequest{
		Side:      order.Buy,
		AssetType: asset.Spot,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := b.UpdateOrderbook(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	cp := currency.NewPairWithDelimiter(currency.BTC.String(), currency.AUD.String(), "-")
	_, err := b.UpdateTicker(context.Background(), cp, asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := b.UpdateTickers(context.Background(), asset.Spot)
	if err != nil {
		t.Error(err)
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.Skip("API keys required but not set, skipping test")
	}

	_, err := b.GetActiveOrders(context.Background(),
		&order.GetOrdersRequest{AssetType: asset.Spot})
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

func TestWsTrade(t *testing.T) {
	pressXToJSON := []byte(` { "marketId": "BTC-AUD",
    "timestamp": "2019-04-08T20:54:27.632Z",
    "tradeId": 3153171493,
    "price": "7370.11",
    "volume": "0.10901605",
    "side": "Ask",
    "messageType": "trade"
  }`)
	err := b.wsHandleData(pressXToJSON)
	if err != nil {
		t.Error(err)
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
	p, err := currency.NewPairFromString(BTCAUD)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricCandles(context.Background(),
		p,
		asset.Spot,
		time.Now().Add(-time.Hour*24).UTC(),
		time.Now().UTC(),
		kline.OneHour)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricCandles(context.Background(),
		p,
		asset.Spot,
		time.Now().Add(-time.Hour*24).UTC(),
		time.Now().UTC(),
		kline.FifteenMin)
	if err != nil {
		if err.Error() != "interval not supported" {
			t.Fatal(err)
		}
	}
}

func TestBTCMarkets_GetHistoricCandlesExtended(t *testing.T) {
	start := time.Now().AddDate(0, 0, -2)
	end := time.Now()
	p, err := currency.NewPairFromString(BTCAUD)
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.GetHistoricCandlesExtended(context.Background(),
		p, asset.Spot, start, end, kline.OneDay)
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
	_, err = b.GetRecentTrades(context.Background(), currencyPair, asset.Spot)
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
	_, err = b.GetHistoricTrades(context.Background(),
		currencyPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	if err != nil && err != common.ErrFunctionNotSupported {
		t.Error(err)
	}
}

func TestChecksum(t *testing.T) {
	b := &orderbook.Base{
		Asks: []orderbook.Item{
			{Price: 0.3965, Amount: 44149.815},
			{Price: 0.3967, Amount: 16000.0},
		},
		Bids: []orderbook.Item{
			{Price: 0.396, Amount: 51.0},
			{Price: 0.396, Amount: 25.0},
			{Price: 0.3958, Amount: 18570.0},
		},
	}

	expecting := 3802968298
	err := checksum(b, uint32(expecting))
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

	for x := range testCases {
		tt := testCases[x]
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
	_, err := b.formatOrderType(order.Type("SWOOON"))
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
	_, err := b.formatOrderSide("invalid")
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
