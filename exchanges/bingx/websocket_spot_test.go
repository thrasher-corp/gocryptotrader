package bingx

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// enableTestTradeFeed relays trades to the data handler
func enableTestTradeFeed(ex *Exchange) {
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
}

// fixtureData returns the data member of the first line of a fixture containing substr
func fixtureData(tb testing.TB, path, substr string) []byte {
	tb.Helper()
	data, _, _, err := jsonparser.Get(fixtureLine(tb, path, substr), "data")
	require.NoError(tb, err, "the fixture line must have data")
	return data
}

func TestWebsocketHandleSpotData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	enableTestTradeFeed(ex)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, testOrderbookSubscription(testSpotPair, asset.Spot, 20)), "wsClaimBook must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsSpot.jsonl", func(ctx context.Context, frame []byte) error {
		return ex.wsHandleSpotData(ctx, conn, frame)
	})
	counts := make(map[string]int)
	for _, p := range websocketPayloads(ex) {
		switch p.(type) {
		case []trade.Data:
			counts["trades"]++
		case kline.Item:
			counts["candles"]++
		case *orderbook.Depth:
			counts["books"]++
		case *ticker.Price:
			counts["tickers"]++
		case *WebsocketSpotLastPrice:
			counts["last prices"]++
		case *WebsocketSpotBookTicker:
			counts["book tickers"]++
		default:
			assert.Failf(t, "unexpected payload", "%T relayed", p)
		}
	}
	assert.Equal(t, map[string]int{"trades": 2, "candles": 3, "books": 2, "tickers": 1, "last prices": 1, "book tickers": 1}, counts, "every fixture frame should be relayed")
	assert.Equal(t, []testWebsocketMessage{{limit: websocketHeartbeatRate, message: `{"pong":"dfa87f932a004ec5a03e46e7a31171c0","time":"2026-11-11T17:53:38.549+0800"}`}}, conn.messages(), "the heartbeat should be answered")

	for _, tc := range []struct {
		name, frame string
		err         error
	}{
		{"error frame", `{"code":100400,"id":"","msg":"dataType is error"}`, websocket.ErrSignatureNotMatched},
		{"error without an id", `{"code":100400,"msg":"request parse error, check docs."}`, errWebsocketErrorMessage},
		{"incremental depth", `{"code":0,"timestamp":1794390762676,"dataType":"BTC-USDT@incrDepth","data":{"action":"all"}}`, errWebsocketUnhandledMessage},
		{"unknown frame", `{"code":0,"success":true}`, errWebsocketUnhandledMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			conn := &testWebsocketConnection{matchErr: websocket.ErrSignatureNotMatched}
			assert.ErrorIs(t, ex.wsHandleSpotData(t.Context(), conn, []byte(tc.frame)), tc.err, "wsHandleSpotData should return the expected error")
		})
	}
}

func TestWebsocketProcessSpotTrade(t *testing.T) {
	t.Parallel()
	data := fixtureData(t, "testdata/wsSpot.jsonl", "@trade")
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSpotTrade(data), "wsProcessSpotTrade must not error without a trade feed")
	assert.Empty(t, websocketPayloads(ex), "trades should not be relayed without a trade feed")

	enableTestTradeFeed(ex)
	require.NoError(t, ex.wsProcessSpotTrade(data), "wsProcessSpotTrade must not error")
	exp := []trade.Data{{
		TID:          "247081359",
		Exchange:     ex.Name,
		CurrencyPair: testSpotPair,
		AssetType:    asset.Spot,
		Side:         order.Buy,
		Price:        73694,
		Amount:       0.006159,
		Timestamp:    time.UnixMilli(1794390762862),
	}}
	assert.Equal(t, exp, waitForWebsocketPayload[[]trade.Data](t, ex, nil), "the trade should be relayed")

	require.NoError(t, ex.wsProcessSpotTrade([]byte(`{"s":"ETH-USDT","t":"1","p":"1","q":"1","T":1794390762862,"m":true}`)), "wsProcessSpotTrade must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's trade should not be relayed")
	assert.ErrorIs(t, ex.wsProcessSpotTrade([]byte(`{"s":"DOGE-USDT","t":"1"}`)), currency.ErrPairNotFound, "wsProcessSpotTrade should error for an unknown pair")
	assert.Error(t, ex.wsProcessSpotTrade([]byte(`{"t":true}`)), "wsProcessSpotTrade should error for a trade that does not decode")
}

func TestWebsocketProcessSpotTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSpotTicker(t.Context(), fixtureData(t, "testdata/wsSpot.jsonl", "@ticker")), "wsProcessSpotTicker must not error")
	exp := &ticker.Price{
		Last:                73694.04,
		High:                74830,
		Low:                 72735.17,
		Bid:                 73694,
		BidSize:             0.002081,
		Ask:                 73694.01,
		AskSize:             0.005378,
		BaseVolume:          2244.55,
		QuoteVolume:         165407519.39,
		Open:                74468.05,
		PercentChange24Hour: -1.04,
		Pair:                testSpotPair,
		ExchangeName:        ex.Name,
		AssetType:           asset.Spot,
		LastUpdated:         time.UnixMilli(1794390762715),
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*ticker.Price](t, ex, nil), "the ticker should be relayed")
	stored, err := ticker.GetTicker(ex.Name, testSpotPair, asset.Spot)
	require.NoError(t, err, "the ticker must be stored")
	assert.InDelta(t, exp.Last, stored.Last, 1e-9, "the stored ticker should be the one relayed")

	require.NoError(t, ex.wsProcessSpotTicker(t.Context(), []byte(`{"s":"ETH-USDT","c":1}`)), "wsProcessSpotTicker must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's ticker should not be relayed")
	badPercentage := []byte(`{"s":"BTC-USDT","P":"x%"}`)
	assert.Error(t, ex.wsProcessSpotTicker(t.Context(), badPercentage), "wsProcessSpotTicker should error for a ticker that does not decode")
}

func TestWebsocketProcessSpotKline(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSpotKline(t.Context(), fixtureData(t, "testdata/wsSpot.jsonl", "@kline_1min")), "wsProcessSpotKline must not error")
	exp := kline.Item{
		Exchange: ex.Name,
		Pair:     testSpotPair,
		Asset:    asset.Spot,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:             time.UnixMilli(1794390720000),
			Open:             73706.03,
			High:             73706.07,
			Low:              73694.006,
			Close:            73694.006,
			Volume:           0.501372,
			QuoteVolume:      36950.109853,
			ValidationIssues: kline.PartialCandle,
		}},
	}
	assert.Equal(t, exp, waitForWebsocketPayload[kline.Item](t, ex, nil), "the candle should be relayed as partial while the event precedes its end")

	require.NoError(t, ex.wsProcessSpotKline(t.Context(), fixtureData(t, "testdata/wsSpot.jsonl", "@kline_1mon")), "wsProcessSpotKline must not error")
	month := waitForWebsocketPayload[kline.Item](t, ex, nil)
	assert.Equal(t, kline.OneMonth, month.Interval, "the month interval should be read")
	assert.Equal(t, kline.PartialCandle, month.Candles[0].ValidationIssues, "a month candle should be partial although its end time is a minute after its start")

	closed := []byte(`{"E":1794390780001,"K":{"t":1794390720000,"T":1794390779999,"s":"BTC-USDT","i":"1min","o":"1","c":"1","h":"1","l":"1","v":"1","q":"1","n":1}}`)
	require.NoError(t, ex.wsProcessSpotKline(t.Context(), closed), "wsProcessSpotKline must not error")
	assert.Empty(t, waitForWebsocketPayload[kline.Item](t, ex, nil).Candles[0].ValidationIssues, "a candle pushed after its end should not be partial")

	require.NoError(t, ex.wsProcessSpotKline(t.Context(), []byte(`{"K":{"s":"ETH-USDT","i":"1min"}}`)), "wsProcessSpotKline must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's candle should not be relayed")
	assert.ErrorIs(t, ex.wsProcessSpotKline(t.Context(), []byte(`{"K":{"s":"BTC-USDT","i":"1m"}}`)), kline.ErrInvalidInterval, "wsProcessSpotKline should reject another product's interval")
	assert.Error(t, ex.wsProcessSpotKline(t.Context(), []byte(`{"K":{"n":"x"}}`)), "wsProcessSpotKline should error for a candle that does not decode")
}

func TestWebsocketProcessSpotOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, testOrderbookSubscription(testSpotPair, asset.Spot, 20)), "wsClaimBook must not error")
	frames := make([][]byte, 0, 2)
	for _, line := range fixtureLines(t, "testdata/wsSpot.jsonl") {
		if dataType, _ := jsonparser.GetString(line, "dataType"); dataType == "BTC-USDT@depth20" {
			frames = append(frames, line)
		}
	}
	require.Len(t, frames, 2, "the fixture must hold two pushes")
	for i, frame := range frames {
		data, _, _, err := jsonparser.Get(frame, "data")
		require.NoError(t, err, "the push must have data")
		require.NoErrorf(t, ex.wsProcessSpotOrderbook(t.Context(), conn, "BTC-USDT@depth20", frame, data), "wsProcessSpotOrderbook must not error for push %d", i)
		book, err := ex.Websocket.Orderbook.GetOrderbook(testSpotPair, asset.Spot)
		require.NoErrorf(t, err, "push %d must load the book", i)
		var push WebsocketSpotOrderbook
		require.NoError(t, json.Unmarshal(data, &push), "the push must decode")
		assert.Equalf(t, push.LastUpdateID, book.LastUpdateID, "push %d should replace the book's update ID", i)
		pushed, err := jsonparser.GetInt(frame, "timestamp")
		require.NoError(t, err, "the push must have a timestamp")
		assert.Equalf(t, time.UnixMilli(pushed), book.LastUpdated, "push %d should date the book", i)
		require.Lenf(t, book.Asks, 20, "push %d must load every ask", i)
		require.Lenf(t, book.Bids, 20, "push %d must load every bid", i)
		assert.InDeltaf(t, push.Asks[len(push.Asks)-1].Price, book.Asks[0].Price, 1e-9, "push %d's asks arrive highest first and should be sorted lowest first", i)
		assert.InDeltaf(t, push.Asks[0].Price, book.Asks[19].Price, 1e-9, "push %d's highest ask should be last", i)
		assert.InDeltaf(t, push.Bids[0].Price, book.Bids[0].Price, 1e-9, "push %d's bids should be highest first", i)
	}

	stale := []byte(`{"code":0,"data":{"asks":[["2","1"]],"bids":[["1","1"]],"lastUpdateId":1},"dataType":"BTC-USDT@depth5","success":true,"timestamp":1794390763285}`)
	data, _, _, err := jsonparser.Get(stale, "data")
	require.NoError(t, err, "the push must have data")
	require.NoError(t, ex.wsProcessSpotOrderbook(t.Context(), conn, "BTC-USDT@depth5", stale, data), "wsProcessSpotOrderbook must not error for another stream")
	book, err := ex.Websocket.Orderbook.GetOrderbook(testSpotPair, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Len(t, book.Asks, 20, "a push of a stream not feeding the book should not replace it")

	require.NoError(t, ex.wsProcessSpotOrderbook(t.Context(), conn, "ETH-USDT@depth20", nil, nil), "wsProcessSpotOrderbook must not error for a disabled pair")
	assert.Error(t, ex.wsProcessSpotOrderbook(t.Context(), conn, "BTC-USDT@depth20", nil, []byte(`{"bids":[["x"]]}`)), "wsProcessSpotOrderbook should error for a push that does not decode")
	assert.ErrorIs(t, ex.wsProcessSpotOrderbook(t.Context(), conn, "DOGE-USDT@depth20", nil, nil), currency.ErrPairNotFound, "wsProcessSpotOrderbook should error for an unknown pair")
}

func TestWebsocketHandleSpotAccountData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	conn := &testWebsocketConnection{}
	_, session := trackTestConnection(t, ex, conn, false)
	testexch.FixtureToDataHandler(t, "testdata/wsSpotAccount.jsonl", func(ctx context.Context, frame []byte) error {
		return ex.wsHandleSpotAccountData(ctx, conn, frame)
	})
	payloads := websocketPayloads(ex)
	require.Len(t, payloads, 2, "each documented event must be relayed")
	assert.IsType(t, &order.Detail{}, payloads[0], "the order event should be relayed as an order detail")
	assert.IsType(t, accounts.SubAccounts{}, payloads[1], "the balance event should be relayed as balances")

	require.NoError(t, ex.wsHandleSpotAccountData(t.Context(), conn, []byte(`{"ping":"a","time":"b"}`)), "a heartbeat must be answered")
	assert.Equal(t, []testWebsocketMessage{{limit: websocketHeartbeatRate, message: `{"pong":"a","time":"b"}`}}, conn.messages(), "the heartbeat should be answered")
	require.NoError(t, ex.wsHandleSpotAccountData(t.Context(), conn, []byte(`{"e":"listenKeyExpired","E":1794390762862}`)), "an expired key must be handled")
	ex.wsState.mu.Lock()
	assert.Equal(t, session.token(), ex.wsState.recovering, "an expired key should reconnect the session")
	ex.wsState.mu.Unlock()
	testWaitBackground(t, ex)
	assert.ErrorIs(t, ex.wsHandleSpotAccountData(t.Context(), conn, []byte(`{"e":"outboundAccountPosition"}`)), errWebsocketUnhandledEvent, "an unknown event should be reported")
	assert.ErrorIs(t, ex.wsHandleSpotAccountData(t.Context(), conn, []byte(`{"code":100400,"msg":"bad"}`)), errWebsocketErrorMessage, "an error frame should be reported")

	t.Run("credentials of the listen key", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		first, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := &testWebsocketConnection{}
		ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, first)
		second := &accounts.Credentials{Key: "second-key", Secret: "second-secret"}
		ex.SetCredentials(second)
		event := fixtureLine(t, "testdata/wsSpotAccount.jsonl", "ACCOUNT_UPDATE")
		require.NoError(t, ex.wsHandleSpotAccountData(t.Context(), conn, event), "wsHandleSpotAccountData must not error")
		_, err = ex.Accounts.GetBalance("", first, asset.Spot, currency.USDT)
		assert.NoError(t, err, "the balances should be stored for the account the connection's listen key was created for")
		_, err = ex.Accounts.GetBalance("", second, asset.Spot, currency.USDT)
		assert.ErrorIs(t, err, accounts.ErrNoBalances, "no balances should be stored for the exchange's new default credentials")
		assert.ErrorIs(t, ex.wsHandleSpotAccountData(t.Context(), &testWebsocketConnection{}, event), errWebsocketConnectionUnknown, "the balance event of a connection no longer tracked should not be stored")
	})
}

func TestWebsocketProcessSpotOrderUpdate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), fixtureData(t, "testdata/wsSpotAccount.jsonl", "executionReport")), "wsProcessSpotOrderUpdate must not error")
	exp := &order.Detail{
		Price:                3.4649,
		Amount:               5,
		QuoteAmount:          17.5845,
		ExecutedAmount:       5,
		Cost:                 17.3235,
		AverageExecutedPrice: 3.4647,
		FeeAsset:             currency.NewCode("SUI"),
		Exchange:             ex.Name,
		OrderID:              "1976495057808654300",
		Type:                 order.Market,
		Side:                 order.Buy,
		Status:               order.Filled,
		AssetType:            asset.Spot,
		Date:                 time.UnixMilli(1760068117941),
		LastUpdated:          time.UnixMilli(1760068118150),
		Pair:                 currency.NewPairWithDelimiter("SUI", "USDT", currency.DashDelimiter),
		Trades: []order.TradeHistory{{
			Price:     3.4647,
			Amount:    5,
			Fee:       0.005,
			Exchange:  ex.Name,
			TID:       "83956769",
			Type:      order.Market,
			Side:      order.Buy,
			Timestamp: time.UnixMilli(1760068118035),
			FeeAsset:  "SUI",
			Total:     17.3235,
		}},
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*order.Detail](t, ex, nil), "the order event should be relayed as an order detail with its fill")

	newOrder := []byte(`{"e":"executionReport","E":1794390762863,"s":"BTC-USDT","S":"SELL","o":"LIMIT","q":"2","p":"100","x":"NEW","X":"NEW","i":7,"z":"0","Z":"0","C":"mine","O":1794390762862}`)
	require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), newOrder), "wsProcessSpotOrderUpdate must not error")
	got := waitForWebsocketPayload[*order.Detail](t, ex, nil)
	assert.Empty(t, got.Trades, "an event without a fill should carry no trade")
	assert.Zero(t, got.AverageExecutedPrice, "an order without fills should have no average price")
	assert.InDelta(t, 2, got.RemainingAmount, 1e-9, "an order without fills should have its whole quantity remaining")
	assert.Equal(t, "mine", got.ClientOrderID, "the client order ID should be relayed")

	for _, tc := range []struct {
		name, event string
		err         error
	}{
		{"status", `{"s":"BTC-USDT","S":"BUY","o":"LIMIT","X":"WAITING","i":1}`, order.ErrUnsupportedStatusType},
		{"type", `{"s":"BTC-USDT","S":"BUY","o":"ICEBERG","X":"NEW","i":1}`, order.ErrUnrecognisedOrderType},
		{"side", `{"s":"BTC-USDT","S":"BOTH","o":"LIMIT","X":"NEW","i":1}`, order.ErrSideIsInvalid},
		{"symbol", `{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"NEW","i":1}`, errWebsocketInvalidSymbol},
	} {
		require.NoErrorf(t, ex.wsProcessSpotOrderUpdate(t.Context(), []byte(tc.event)), "wsProcessSpotOrderUpdate must relay an unclassified %s", tc.name)
		classification := waitForWebsocketPayload[order.ClassificationError](t, ex, nil)
		assert.ErrorIsf(t, classification.Err, tc.err, "an unknown %s should be relayed as a classification error", tc.name)
		assert.Equal(t, "1", classification.OrderID, "the classification error should name the order")
	}
	assert.Error(t, ex.wsProcessSpotOrderUpdate(t.Context(), []byte(`{"i":"x"}`)), "wsProcessSpotOrderUpdate should error for an event that does not decode")
}

func TestWebsocketProcessSpotAccountUpdate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	require.NoError(t, ex.wsProcessSpotAccountUpdate(t.Context(), fixtureLine(t, "testdata/wsSpotAccount.jsonl", "ACCOUNT_UPDATE")), "wsProcessSpotAccountUpdate must not error")
	relayed := waitForWebsocketPayload[accounts.SubAccounts](t, ex, nil)
	require.Len(t, relayed, 1, "the balances must be relayed as one sub-account")
	assert.Equal(t, asset.Spot, relayed[0].AssetType, "the balances should be spot's")
	got := relayed[0].Balances[currency.USDT]
	exp := accounts.Balance{Currency: currency.USDT, Total: 0.00000000999999, Free: 0.00000000999999, UpdatedAt: got.UpdatedAt}
	assert.Equal(t, exp, got, "wb should be the total, lk held and the rest free")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "the balance must be stored")
	assert.Equal(t, got, stored, "the stored balance should be the one relayed")
	spot := testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
	assert.Equal(t, uint64(1), ex.balances.generations[spot], "the stream write should be counted, so a REST snapshot requested before it is not stored over it")

	frozen := []byte(`{"e":"ACCOUNT_UPDATE","a":{"B":[{"a":"BTC","wb":"1","lk":"1.5"}],"m":"ORDER"},"E":1794390762862}`)
	require.NoError(t, ex.wsProcessSpotAccountUpdate(t.Context(), frozen), "wsProcessSpotAccountUpdate must not error")
	btc := waitForWebsocketPayload[accounts.SubAccounts](t, ex, nil)[0].Balances[currency.BTC]
	assert.Zero(t, btc.Free, "the free balance should not go below zero")
	_, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	assert.NoError(t, err, "an event should leave the currencies it does not carry")

	require.NoError(t, ex.wsProcessSpotAccountUpdate(t.Context(), []byte(`{"e":"ACCOUNT_UPDATE","a":{"B":[]}}`)), "wsProcessSpotAccountUpdate must not error without balances")
	assert.Empty(t, websocketPayloads(ex), "an event without balances should relay nothing")
	assert.Equal(t, uint64(2), ex.balances.generations[spot], "an event without balances should write nothing to count")
	assert.Error(t, ex.wsProcessSpotAccountUpdate(t.Context(), []byte(`{"a":{"B":{}}}`)), "wsProcessSpotAccountUpdate should error for an event that does not decode")
	ex.SetCredentials(&accounts.Credentials{})
	assert.Error(t, ex.wsProcessSpotAccountUpdate(t.Context(), frozen), "wsProcessSpotAccountUpdate should error without credentials")
}

func TestWebsocketStoreSpotBalances(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	subAccount, err := ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 5, FrozenAssets: 2}, {Asset: currency.BTC, WalletBalance: 1, FrozenAssets: 1.5}})
	require.NoError(t, err, "wsStoreSpotBalances must not error")
	assert.Equal(t, asset.Spot, subAccount.AssetType, "wsStoreSpotBalances should return spot balances")
	usdt := subAccount.Balances[currency.USDT]
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 5, Hold: 2, Free: 3, UpdatedAt: usdt.UpdatedAt}, usdt, "wsStoreSpotBalances should take wb as the total and lk as held")
	assert.Zero(t, subAccount.Balances[currency.BTC].Free, "wsStoreSpotBalances should not take the free balance below zero")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "GetBalance must not error")
	assert.Equal(t, usdt, stored, "wsStoreSpotBalances should store the balances it returns")
	spot := testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
	assert.Equal(t, uint64(1), ex.balances.generations[spot], "wsStoreSpotBalances should count the write")

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_, err := ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 1}})
			assert.NoError(t, err, "wsStoreSpotBalances should not error")
		})
	}
	wg.Wait()
	assert.Equal(t, uint64(11), ex.balances.generations[spot], "wsStoreSpotBalances should count every write")
	assert.Zero(t, ex.balances.generations[testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures)], "wsStoreSpotBalances should count spot's writes alone")

	ex.SetCredentials(&accounts.Credentials{})
	_, err = ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 1}})
	assert.Error(t, err, "wsStoreSpotBalances should error without credentials")

	t.Run("two credential sets", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		otherCreds := &accounts.Credentials{Key: "other-key", Secret: "other-secret"}
		otherCtx := accounts.DeployCredentialsToContext(t.Context(), otherCreds)
		_, err := ex.wsStoreSpotBalances(otherCtx, []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 2}})
		require.NoError(t, err, "wsStoreSpotBalances must not error for context credentials")
		stored, err := ex.Accounts.GetBalance("", otherCreds, asset.Spot, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the context's account")
		assert.InDelta(t, 2, stored.Free, 1e-9, "wsStoreSpotBalances should store the context account's balances")
		assert.Equal(t, uint64(1), ex.balances.generations[testAccountBalanceKey(otherCtx, t, ex, asset.Spot)], "wsStoreSpotBalances should count the write for the context's account")
		assert.Zero(t, ex.balances.generations[testAccountBalanceKey(t.Context(), t, ex, asset.Spot)], "wsStoreSpotBalances should count each account's writes apart")
	})

	t.Run("accounts store error", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		_, err := ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{WalletBalance: 1}})
		assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "wsStoreSpotBalances should return the accounts store's error")
	})

	t.Run("default credentials change during the write", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		before, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		ctx := &testDefaultCredentialsReadContext{Context: t.Context(), read: make(chan struct{})}
		// Holding the write lock stops the write after it has read the default credentials and before it stores anything
		ex.balances.writes.Lock()
		done := make(chan error, 1)
		go func() {
			_, err := ex.wsStoreSpotBalances(ctx, []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 4}})
			done <- err
		}()
		select {
		case <-ctx.read:
		case <-time.After(5 * time.Second):
			ex.balances.writes.Unlock()
			require.FailNow(t, "wsStoreSpotBalances must read the default credentials")
		}
		after := &accounts.Credentials{Key: "after-key", Secret: "after-secret"}
		ex.SetCredentials(after)
		ex.balances.writes.Unlock()
		select {
		case err := <-done:
			require.NoError(t, err, "wsStoreSpotBalances must not error")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "wsStoreSpotBalances must return")
		}
		stored, err := ex.Accounts.GetBalance("", before, asset.Spot, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the credentials the write started with")
		assert.InDelta(t, 4, stored.Total, 1e-9, "wsStoreSpotBalances should store the balances under the credentials it started with")
		_, err = ex.Accounts.GetBalance("", after, asset.Spot, currency.USDT)
		assert.ErrorIs(t, err, accounts.ErrNoBalances, "wsStoreSpotBalances should store nothing under the credentials set during the write")
		assert.Equal(t, uint64(1), ex.balances.generations[testAccountBalanceKey(accounts.DeployCredentialsToContext(t.Context(), before), t, ex, asset.Spot)], "wsStoreSpotBalances should count the write for the account whose balances it stores")
	})
}

// testDefaultCredentialsReadContext closes read once a credential lookup made with it has read the exchange's default
// credentials, which GetCredentials does just before it looks the sub-account override up
type testDefaultCredentialsReadContext struct {
	context.Context //nolint:containedctx // It observes the credential lookups of the calls made with it
	read            chan struct{}
	once            sync.Once
}

// Value returns the parent's value for key, closing read at the first sub-account override lookup
func (c *testDefaultCredentialsReadContext) Value(key any) any {
	if key == accounts.ContextSubAccountFlag {
		c.once.Do(func() { close(c.read) })
	}
	return c.Context.Value(key)
}

func TestWebsocketSpotIntegration(t *testing.T) {
	t.Parallel()
	servers := newTestWebsocketServers(t, asset.Spot)
	ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, true, func(cfg *config.Exchange) {
		cfg.Features.Enabled.TradeFeed = true
		cfg.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 20},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.MyOrdersChannel, Authenticated: true},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.MyAccountChannel, Authenticated: true},
		}
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	assert.ElementsMatch(t, []string{"BTC-USDT@ticker", "BTC-USDT@trade", "BTC-USDT@kline_1min", "BTC-USDT@depth20", "spot.executionReport", "ACCOUNT_UPDATE"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "every subscription should be stored")
	assert.Len(t, servers.ws[asset.Spot].receivedRequests(), 6, "every spot stream, account streams included, should be subscribed with a request")

	waitForWebsocketPayload(t, ex, func(d *order.Detail) bool { return d.OrderID == "1976495057808654300" })
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	require.Eventually(t, func() bool {
		_, tickerErr := ticker.GetTicker(ex.Name, testSpotPair, asset.Spot)
		book, bookErr := ex.Websocket.Orderbook.GetOrderbook(testSpotPair, asset.Spot)
		_, balanceErr := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
		return tickerErr == nil && bookErr == nil && book.LastUpdateID == 16554155664 && balanceErr == nil && len(servers.ws[asset.Spot].heartbeatReplies()) == 2
	}, 5*time.Second, time.Millisecond, "the ticker, the last book push, the balance and both heartbeat replies must arrive")
	assert.Equal(t, 1, servers.ws[asset.Spot].marketDials(), "one market connection should be dialled")
	assert.Equal(t, []string{testListenKey + "-1"}, servers.ws[asset.Spot].accountKeys(), "the account connection should be dialled with the listen key")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		assert.Zerof(t, servers.ws[a].marketDials(), "disabled %s should dial no market connection", a)
		assert.Emptyf(t, servers.ws[a].accountKeys(), "disabled %s should dial no account connection", a)
	}

	ex.wsState.mu.Lock()
	keeper := ex.wsState.keepers[servers.rest.URL]
	ex.wsState.mu.Unlock()
	require.NotNil(t, keeper, "a keeper must keep the listen key alive")
	shutdownTestWebsocket(t, ex)
	assert.ErrorIs(t, bookError(ex, testSpotPair, asset.Spot), orderbook.ErrOrderbookInvalid, "shutting down should invalidate the book")
	waitForKeeper(t, keeper)

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		testLiveMarketStreams(t, asset.Spot, testSpotPair, 20)
	})

	t.Run("capacity", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			pairs, dials int
		}{{50, 1}, {51, 2}} {
			pairs := make(currency.Pairs, tc.pairs)
			for i := range pairs {
				pairs[i] = currency.NewPairWithDelimiter("C"+strconv.Itoa(i), "USDT", currency.DashDelimiter)
			}
			servers := newTestWebsocketServers(t, asset.Spot)
			ex := newWebsocketIntegrationExchange(t, servers, asset.Spot, false, testWebsocketPairs(asset.Spot, pairs), testWebsocketSubscriptions(
				&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
				&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel},
				&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
				&subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 20},
				&subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
			))
			ex.Requester.GetRateLimiterDefinitions()[websocketSubscriptionRate] = request.NewRateLimitWithWeight(0, 0, 1)
			require.NoErrorf(t, ex.Websocket.Connect(t.Context()), "Connect must not error with %d pairs", tc.pairs)
			assert.Lenf(t, ex.Websocket.GetSubscriptions(), tc.pairs*4, "every distinct stream of %d pairs should be subscribed", tc.pairs)
			assert.Equalf(t, tc.dials, servers.ws[asset.Spot].marketDials(), "%d pairs should need %d connections", tc.pairs, tc.dials)
		}
	})
}
