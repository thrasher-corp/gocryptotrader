package bingx

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
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
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
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

func TestWebsocketProcessSpotOrderbook(t *testing.T) { //nolint:tparallel // the pushes load one book in turn for the checks that follow
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
		t.Run("push "+strconv.Itoa(i), func(t *testing.T) {
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
		})
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
		ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.Spot, first)
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
	conn := &testWebsocketConnection{}
	trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, fixtureData(t, "testdata/wsSpotAccount.jsonl", "executionReport")), "wsProcessSpotOrderUpdate must not error")
	exp := &order.Detail{
		Price:                3.4649,
		Amount:               5,
		QuoteAmount:          17.5845,
		ExecutedAmount:       5,
		Cost:                 17.3235,
		AverageExecutedPrice: 3.4647,
		Fee:                  0.005,
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
	assert.Equal(t, exp, waitForWebsocketPayload[*order.Detail](t, ex, nil), "the order event should be relayed as an order detail with its fill, its only fill's fee being the order's")

	newOrder := []byte(`{"e":"executionReport","E":1794390762863,"s":"BTC-USDT","S":"SELL","o":"LIMIT","q":"2","p":"100","x":"NEW","X":"NEW","i":7,"z":"0","Z":"0","C":"mine","O":1794390762862}`)
	require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, newOrder), "wsProcessSpotOrderUpdate must not error")
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
		// The cases take their payloads from ex's data handler in turn
		t.Run(tc.name, func(t *testing.T) {
			require.NoErrorf(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, []byte(tc.event)), "wsProcessSpotOrderUpdate must relay an unclassified %s", tc.name)
			classification := waitForWebsocketPayload[order.ClassificationError](t, ex, nil)
			assert.ErrorIsf(t, classification.Err, tc.err, "an unknown %s should be relayed as a classification error", tc.name)
			assert.Equal(t, "1", classification.OrderID, "the classification error should name the order")
		})
	}
	assert.Error(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, []byte(`{"i":"x"}`)), "wsProcessSpotOrderUpdate should error for an event that does not decode")

	t.Run("quote-sized order executed past its quantity", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		quoteSized := []byte(`{"e":"executionReport","E":1700000001000,"s":"BTC-USDT","S":"BUY","o":"MARKET","q":"0","p":"0","x":"TRADE","X":"FILLED","i":44,"l":"2","z":"2","L":"10","n":"-0.002","N":"BTC","T":1700000001000,"t":301,"Z":"20","Y":"20","Q":"20","O":1700000000000}`)
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), &testWebsocketConnection{}, quoteSized), "wsProcessSpotOrderUpdate must not error")
		assert.Zero(t, waitForWebsocketPayload[*order.Detail](t, ex, nil).RemainingAmount, "an order executed past its quantity should have nothing remaining")
	})

	t.Run("connection not tracked", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), &testWebsocketConnection{}, fixtureData(t, "testdata/wsSpotAccount.jsonl", "executionReport")), "wsProcessSpotOrderUpdate must not error on a connection not tracked")
		d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
		require.Len(t, d.Trades, 1, "the fill must be relayed")
		assert.Equal(t, 0.005, d.Trades[0].Fee, "the fill should carry its fee")
		assert.Zero(t, d.Fee, "an event on a connection keeping no totals should carry no order fee")
	})

	t.Run("order fees in the order store", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name    string
			stored  order.Detail
			lookups []string
			events  [][]byte
			fees    []float64
			exp     float64
			queries int
			records int
		}{
			{
				name:   "the documented fill",
				events: [][]byte{fixtureData(t, "testdata/wsSpotAccount.jsonl", "executionReport")},
				fees:   []float64{0.005},
				exp:    0.005,
			},
			{
				name:    "the last fills of an order the sweep stored part filled",
				stored:  order.Detail{ExecutedAmount: 1, Fee: 0.1, Status: order.PartiallyFilled},
				lookups: []string{testSpotOrderReply("FILLED", "3", "-0.3")},
				events:  [][]byte{testSpotOrderEvent("TRADE", "FILLED", "2", "3", "-0.2", 102)},
				fees:    []float64{0, 0.3},
				exp:     0.3,
				queries: 1,
			},
			{
				name:    "the last fills of an order the sweep stored part filled, looked up behind them",
				stored:  order.Detail{ExecutedAmount: 1, Fee: 0.1, Status: order.PartiallyFilled},
				lookups: []string{testSpotOrderReply("PARTIALLY_FILLED", "1", "-0.1")},
				events:  [][]byte{testSpotOrderEvent("TRADE", "FILLED", "2", "3", "-0.2", 102)},
				fees:    []float64{0, 0.3},
				exp:     0.3,
				queries: 1,
			},
			{
				name:   "a first fill and the order's cancellation",
				events: [][]byte{testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "1", "1", "-0.1", 101), testSpotOrderEvent("CANCELED", "CANCELED", "0", "1", "0", 0)},
				fees:   []float64{0.1, 0.1},
				exp:    0.1,
			},
			{
				name:   "fills in turn",
				events: [][]byte{testSpotOrderEvent("NEW", "NEW", "0", "0", "0", 0), testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "1", "1", "-0.1", 101), testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "1.5", "2.5", "-0.15", 102), testSpotOrderEvent("TRADE", "FILLED", "0.5", "3", "-0.05", 103)},
				fees:   []float64{0, 0.1, 0.25, 0.3},
				exp:    0.3,
			},
			{
				name:    "a cancellation after a fill the connection did not stream",
				stored:  order.Detail{Status: order.New},
				lookups: []string{testSpotOrderReply("CANCELED", "1", "-0.1")},
				events:  [][]byte{testSpotOrderEvent("CANCELED", "CANCELED", "0", "1", "0", 0)},
				fees:    []float64{0, 0.1},
				exp:     0.1,
				queries: 1,
			},
			{
				name:    "a fill repeated",
				events:  [][]byte{testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "1", "1", "-0.1", 101), testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "1", "1", "-0.1", 101)},
				fees:    []float64{0.1, 0.1},
				exp:     0.1,
				records: 1,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestSpotOrderServer(t, ex)
				server.queue(tc.lookups...)
				conn := &testWebsocketConnection{}
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.Spot, creds)
				stored := tc.stored
				stored.Exchange, stored.AssetType, stored.Amount = ex.Name, asset.Spot, 3
				var fees []float64
				for _, data := range tc.events {
					require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, data), "wsProcessSpotOrderUpdate must not error")
					d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
					if stored.OrderID == "" {
						stored.OrderID = d.OrderID
					}
					fees = append(fees, d.Fee)
					require.NoError(t, stored.UpdateOrderFromDetail(d), "the order store must take the event")
				}
				testWaitBackground(t, ex)
				for _, p := range websocketPayloads(ex) {
					d, ok := p.(*order.Detail)
					require.Truef(t, ok, "only order details must be relayed, got %T", p)
					fees = append(fees, d.Fee)
					require.NoError(t, stored.UpdateOrderFromDetail(d), "the order store must take the looked up fee")
				}
				assert.Equal(t, tc.fees, fees, "each relayed detail should carry the order's fee as far as it is known")
				assert.InDelta(t, tc.exp, stored.Fee, 1e-12, "the stored order should hold the fees of all its fills")
				assert.Len(t, server.queried(), tc.queries, "only an order whose earlier fills the connection did not stream should be looked up")
				state.spotFees.mu.Lock()
				assert.Len(t, state.spotFees.orders, tc.records, "only an open order's record should be kept")
				state.spotFees.mu.Unlock()
			})
		}
	})
}

func TestWebsocketLookUpSpotOrderFee(t *testing.T) {
	t.Parallel()
	// spotFeeSetup tracks a spot account connection with the exchange's credentials and serves its order queries
	spotFeeSetup := func(t *testing.T) (*Exchange, *testSpotOrderServer, *testWebsocketConnection, *testSession) {
		t.Helper()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := newTestSpotOrderServer(t, ex)
		conn := &testWebsocketConnection{}
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		session := newTestSession(t, ex)
		ex.wsTrackConnection(conn, session.ch, false, asset.Spot, creds)
		return ex, server, conn, session
	}
	gapFill := testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "1", "2", "-0.1", 102)

	t.Run("lookup refused", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(`{"code":100400,"msg":"refused"}`, testSpotOrderReply("FILLED", "3", "-0.3"))
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, gapFill), "wsProcessSpotOrderUpdate must not error")
		d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
		assert.Equal(t, order.PartiallyFilled, d.Status, "the event's status should be relayed whatever the lookup's outcome")
		assert.Len(t, d.Trades, 1, "the event's fill should be relayed whatever the lookup's outcome")
		testWaitBackground(t, ex)
		assert.Empty(t, websocketPayloads(ex), "a refused lookup should relay nothing")
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, testSpotOrderEvent("TRADE", "FILLED", "1", "3", "-0.1", 103)), "wsProcessSpotOrderUpdate must not error")
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the next event's lookup must relay the order's fee")
		assert.Equal(t, 0.3, got[0].Fee, "the next event's lookup should relay the order's fee")
		assert.Equal(t, []string{"42", "42"}, server.queried(), "an order should be looked up again after a refused lookup")
	})

	t.Run("fills streamed while the lookup is in flight", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("PARTIALLY_FILLED", "2", "-0.2"))
		release, arrived := server.hold()
		testSpotOrderEventLookedUp(t, ex, conn, gapFill, arrived)
		assert.Zero(t, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, "an event whose earlier fills were not streamed should carry no order fee")
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, testSpotOrderEvent("TRADE", "FILLED", "1", "3", "-0.05", 103)), "wsProcessSpotOrderUpdate must not error while the lookup is in flight")
		assert.Zero(t, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, "an event whose earlier fills are being looked up should carry no order fee")
		close(release)
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the lookup must relay the order's fee")
		d := got[0]
		assert.InDelta(t, 0.25, d.Fee, 1e-12, "the lookup's fee should add the fills streamed beyond it")
		assert.Equal(t, currency.USDT, d.FeeAsset, "the fee's currency should be relayed")
		assert.Equal(t, order.UnknownStatus, d.Status, "the looked up fee should be relayed without a status, which the events report")
		assert.Zero(t, d.ExecutedAmount, "the looked up fee should be relayed without an executed quantity, which the events report")
		assert.Equal(t, "42", d.OrderID, "the looked up fee should name its order")
		assert.Equal(t, 3.0, d.Amount, "the looked up fee should carry the order's quantity")
		assert.Equal(t, 30.0, d.QuoteAmount, "the looked up fee should carry the order's quote amount")
		assert.Equal(t, time.UnixMilli(1700000000000), d.Date, "the looked up fee should carry the order's creation time")
		assert.Equal(t, 10.0, d.Price, "the looked up fee should carry the order's price")
		assert.Equal(t, time.UnixMilli(1700000002000), d.LastUpdated, "the looked up fee should be dated by the order's update time BingX reports")
		assert.Equal(t, []string{"42"}, server.queried(), "an order should be looked up once while a lookup of it is in flight")
	})

	t.Run("reply ahead of the stream", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("FILLED", "3", "-0.3"))
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, gapFill), "wsProcessSpotOrderUpdate must not error")
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the lookup must relay the order's fee")
		assert.Equal(t, 0.3, got[0].Fee, "the lookup should relay the fee of every fill it reports")
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, testSpotOrderEvent("TRADE", "FILLED", "1", "3", "-0.1", 103)), "wsProcessSpotOrderUpdate must not error")
		assert.Equal(t, 0.3, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, "a fill the lookup already counted should carry the order's fee once")
	})

	t.Run("reply behind a fill the connection did not stream", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("PARTIALLY_FILLED", "1", "-0.1"), testSpotOrderReply("FILLED", "3", "-0.3"))
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, testSpotOrderEvent("TRADE", "PARTIALLY_FILLED", "0.5", "2.5", "-0.05", 102)), "wsProcessSpotOrderUpdate must not error")
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		assert.Empty(t, websocketPayloads(ex), "a lookup leaving fills unknown should relay nothing")
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, testSpotOrderEvent("TRADE", "FILLED", "0.5", "3", "-0.05", 103)), "wsProcessSpotOrderUpdate must not error")
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the next event's lookup must relay the order's fee")
		assert.Equal(t, 0.3, got[0].Fee, "the next event's lookup should relay the order's fee")
	})

	t.Run("connection down before the reply", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("FILLED", "3", "-0.3"))
		release, arrived := server.hold()
		testSpotOrderEventLookedUp(t, ex, conn, gapFill, arrived)
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		conn.down.Store(true)
		close(release)
		testWaitBackground(t, ex)
		assert.Empty(t, websocketPayloads(ex), "a lookup answered after its connection went down should relay nothing")
	})

	t.Run("session ended during the lookup", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, session := spotFeeSetup(t)
		ex.wsState.mu.Lock()
		ex.wsState.openOrderReadTimeout = time.Hour
		ex.wsState.mu.Unlock()
		server.queue(testSpotOrderReply("FILLED", "3", "-0.3"))
		_, arrived := server.hold()
		testSpotOrderEventLookedUp(t, ex, conn, gapFill, arrived)
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		session.end()
		testWaitBackground(t, ex)
		assert.Empty(t, websocketPayloads(ex), "a lookup ended with its session should relay nothing")
	})

	t.Run("lookup unanswered by its deadline", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		ex.wsState.mu.Lock()
		ex.wsState.openOrderReadTimeout = 50 * time.Millisecond
		ex.wsState.mu.Unlock()
		server.queue(testSpotOrderReply("FILLED", "3", "-0.3"))
		_, arrived := server.hold()
		testSpotOrderEventLookedUp(t, ex, conn, gapFill, arrived)
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		assert.Empty(t, websocketPayloads(ex), "a lookup unanswered by its deadline should relay nothing")
		assert.Len(t, server.queried(), 1, "the lookup should be sent once")
	})

	t.Run("lookup refused for its rate", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(wrapperStatusPrefix+`429 {"code":100410,"msg":"rate limited"}`, testSpotOrderReply("FILLED", "3", "-0.3"))
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, gapFill), "wsProcessSpotOrderUpdate must not error")
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		assert.Empty(t, websocketPayloads(ex), "a lookup refused for its rate should relay nothing")
		assert.Len(t, server.queried(), 1, "the lookup should be sent once")
	})

	t.Run("cancellation without a fee currency", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("CANCELED", "1", "-0.1"))
		cancelled := []byte(`{"e":"executionReport","E":1700000002000,"s":"BTC-USDT","S":"BUY","o":"LIMIT","q":"3","p":"10","x":"CANCELED","X":"CANCELED","i":42,"z":"1","Z":"10","O":1700000000000}`)
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, cancelled), "wsProcessSpotOrderUpdate must not error")
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the lookup must relay the order's fee")
		assert.Equal(t, 0.1, got[0].Fee, "the lookup should relay the order's fee")
		assert.Equal(t, currency.USDT, got[0].FeeAsset, "the lookup should relay the fee's currency BingX reports for the order")
	})

	t.Run("cancellation repeating its last fill's quantity", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("CANCELED", "1", "-0.1"))
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, testSpotOrderEvent("CANCELED", "CANCELED", "1", "1", "0", 0)), "wsProcessSpotOrderUpdate must not error")
		assert.Zero(t, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, "an event that reports no fill should carry no order fee its fills were not streamed for")
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the lookup must relay the order's fee")
		assert.Equal(t, 0.1, got[0].Fee, "the lookup should relay the fee of the fill the connection did not stream")
	})

	t.Run("event refused by the data handler", func(t *testing.T) {
		t.Parallel()
		for _, event := range [][]byte{gapFill, testSpotOrderEvent("TRADE", "FILLED", "2", "3", "-0.2", 103)} {
			ex, server, conn, _ := spotFeeSetup(t)
			ex.Websocket.DataHandler = stream.NewRelay(1)
			require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "full"), "Send must fill the data handler")
			assert.Error(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, event), "wsProcessSpotOrderUpdate should return the data handler's refusal")
			testWaitBackground(t, ex)
			assert.Empty(t, server.queried(), "an order whose event was not relayed should not be looked up, open or closed")
		}
	})

	t.Run("looked up fee refused by the data handler", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		ex.Websocket.DataHandler = stream.NewRelay(1)
		server.queue(testSpotOrderReply("FILLED", "3", "-0.3"))
		require.NoError(t, ex.wsProcessSpotOrderUpdate(t.Context(), conn, gapFill), "wsProcessSpotOrderUpdate must not error")
		testWaitBackground(t, ex)
		got := testOrderDetails(t, ex)
		require.Len(t, got, 1, "the data handler must hold only the event, being full when the lookup relays")
		assert.Zero(t, got[0].Fee, "the event should carry no order fee")
		assert.Len(t, server.queried(), 1, "the order should be looked up")
	})

	t.Run("credentials changing during the lookup", func(t *testing.T) {
		t.Parallel()
		ex, server, conn, _ := spotFeeSetup(t)
		server.queue(testSpotOrderReply("FILLED", "3", "-0.3"))
		release, arrived := server.hold()
		ex.SetCredentials(&accounts.Credentials{Key: "other-key", Secret: "other-secret"})
		testSpotOrderEventLookedUp(t, ex, conn, gapFill, arrived)
		waitForWebsocketPayload[*order.Detail](t, ex, nil)
		close(release)
		testWaitBackground(t, ex)
		server.mu.Lock()
		assert.Equal(t, []string{"test-api-key"}, server.keys, "the lookup should be sent with the credentials of the connection's listen key")
		server.mu.Unlock()
	})
}

func TestSpotOrderFeesRelay(t *testing.T) {
	t.Parallel()
	type event struct {
		executed, from, fee string
		fill, closes        bool
	}
	for _, tc := range []struct {
		name    string
		events  []event
		fees    []float64
		lookups []bool
		known   bool
		open    bool
		latest  string
	}{
		{name: "new order", events: []event{{executed: "0"}}, fees: []float64{0}, lookups: []bool{false}},
		{name: "new order, then its first fill", events: []event{{executed: "0"}, {executed: "1", from: "0", fee: "0.1", fill: true}}, fees: []float64{0, 0.1}, lookups: []bool{false, false}, known: true, open: true, latest: "1"},
		{name: "first fill", events: []event{{executed: "1", from: "0", fee: "0.1", fill: true}}, fees: []float64{0.1}, lookups: []bool{false}, known: true, open: true},
		{name: "fills in turn", events: []event{{executed: "1", from: "0", fee: "0.1", fill: true}, {executed: "3", from: "1", fee: "0.2", fill: true, closes: true}}, fees: []float64{0.1, 0.3}, lookups: []bool{false, false}},
		{name: "fill after a missed one", events: []event{{executed: "1", from: "0", fee: "0.1", fill: true}, {executed: "3", from: "2", fee: "0.1", fill: true}}, fees: []float64{0.1, 0}, lookups: []bool{false, true}, known: true, open: true},
		{name: "fill after earlier ones not streamed", events: []event{{executed: "3", from: "1", fee: "0.2", fill: true}}, fees: []float64{0}, lookups: []bool{true}, open: true},
		{name: "cancellation after a fill not streamed", events: []event{{executed: "1", closes: true}}, fees: []float64{0}, lookups: []bool{true}, open: true},
		{name: "cancellation after a streamed fill", events: []event{{executed: "1", from: "0", fee: "0.1", fill: true}, {executed: "1", closes: true}}, fees: []float64{0.1, 0.1}, lookups: []bool{false, false}},
		{name: "cancellation after a streamed fill and one not streamed", events: []event{{executed: "1", from: "0", fee: "0.1", fill: true}, {executed: "2", closes: true}}, fees: []float64{0.1, 0}, lookups: []bool{false, true}, known: true, open: true, latest: "2"},
		{name: "cancellation with nothing executed", events: []event{{executed: "0", closes: true}}, fees: []float64{0}, lookups: []bool{false}},
		{name: "fill repeated", events: []event{{executed: "1", from: "0", fee: "0.1", fill: true}, {executed: "1", from: "0", fee: "0.1", fill: true}}, fees: []float64{0.1, 0.1}, lookups: []bool{false, false}, known: true, open: true},
		{name: "fills not streamed, one lookup", events: []event{{executed: "2", from: "1", fee: "0.1", fill: true}, {executed: "3", from: "2", fee: "0.1", fill: true}}, fees: []float64{0, 0}, lookups: []bool{true, false}, open: true},
		{name: "fills out of order", events: []event{{executed: "3", from: "1", fee: "0.2", fill: true}, {executed: "1", from: "0", fee: "0.1", fill: true}}, fees: []float64{0, 0.3}, lookups: []bool{true, false}, known: true, open: true, latest: "3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var f spotOrderFees
			for i, ev := range tc.events {
				var fill *spotOrderFill
				if ev.fill {
					fill = &spotOrderFill{from: testDecimal(t, ev.from), to: testDecimal(t, ev.executed), fee: testDecimal(t, ev.fee)}
				}
				d := &order.Detail{OrderID: "42"}
				var sent int
				lookUp, err := f.relay("42", d, testDecimal(t, ev.executed), fill, ev.closes, func() error {
					sent++
					return nil
				})
				require.NoError(t, err, "relay must not error")
				assert.Equal(t, 1, sent, "relay should send the event")
				assert.InDeltaf(t, tc.fees[i], d.Fee, 1e-12, "event %d should carry the order's fee as far as it is known", i)
				assert.Equalf(t, tc.lookups[i], lookUp, "event %d should look the order up only when fills were not streamed and no lookup is in flight", i)
			}
			o, ok := f.orders["42"]
			require.Equal(t, tc.open, ok, "relay must keep the record of an order still open or being looked up")
			if ok {
				assert.Equal(t, tc.known, o.known, "relay should know the order's fee only from its first fill")
			}
			if tc.latest != "" {
				assert.Equal(t, tc.latest, o.latest.String(), "relay should keep the highest executed quantity the order's events reported")
			}
		})
	}

	t.Run("send failure of an event starting no record", func(t *testing.T) {
		t.Parallel()
		var f spotOrderFees
		lookUp, err := f.relay("42", &order.Detail{}, decimal.Zero, nil, false, func() error { return errTestDial })
		assert.ErrorIs(t, err, errTestDial, "relay should return the send's error")
		assert.False(t, lookUp, "relay should not look up an order whose event reports nothing executed")
		assert.Empty(t, f.orders, "relay should start no record for an event reporting no fill and nothing executed")
	})

	t.Run("send failure", func(t *testing.T) {
		t.Parallel()
		var f spotOrderFees
		lookUp, err := f.relay("42", &order.Detail{}, decimal.NewFromInt(2), &spotOrderFill{from: decimal.NewFromInt(1), to: decimal.NewFromInt(2), fee: testDecimal(t, "0.1")}, false, func() error { return errTestDial })
		assert.ErrorIs(t, err, errTestDial, "relay should return the send's error")
		assert.False(t, lookUp, "relay should not look up an order whose event it could not relay")
		require.Contains(t, f.orders, "42", "relay must keep the order's record")
		assert.Zero(t, f.orders["42"].lookups, "relay should count no lookup for an event it could not relay")
	})
}

func TestSpotOrderFeesSettle(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		known           bool
		executed, fee   string
		fills           []spotOrderFill
		latest          string
		reply, replyFee string
		ok, closed      bool
		relayed         string
		records         int
	}{
		{name: "reply covering the fills", fills: []spotOrderFill{{from: decimal.NewFromInt(1), to: decimal.NewFromInt(3), fee: testDecimal(t, "0.2")}}, latest: "3", reply: "3", replyFee: "0.3", ok: true, relayed: "0.3", records: 1},
		{name: "reply behind the fills", fills: []spotOrderFill{{from: decimal.NewFromInt(1), to: decimal.NewFromInt(3), fee: testDecimal(t, "0.2")}}, latest: "3", reply: "1", replyFee: "0.1", ok: true, relayed: "0.3", records: 1},
		{name: "reply leaving a gap", fills: []spotOrderFill{{from: decimal.NewFromInt(2), to: decimal.NewFromInt(3), fee: testDecimal(t, "0.1")}}, latest: "3", reply: "1", replyFee: "0.1", ok: true, records: 1},
		{name: "failed lookup", fills: []spotOrderFill{{from: decimal.NewFromInt(1), to: decimal.NewFromInt(3), fee: testDecimal(t, "0.2")}}, latest: "3", reply: "3", replyFee: "0.3", records: 1},
		{name: "reply no further than known", known: true, executed: "3", fee: "0.3", latest: "3", reply: "2", replyFee: "0.2", ok: true, records: 1},
		{name: "reply at the known quantity", known: true, executed: "3", fee: "0.3", latest: "3", reply: "3", replyFee: "0.25", ok: true, records: 1},
		{name: "reply on a closed order", fills: []spotOrderFill{{from: decimal.NewFromInt(1), to: decimal.NewFromInt(3), fee: testDecimal(t, "0.2")}}, latest: "3", reply: "3", replyFee: "0.3", ok: true, closed: true, relayed: "0.3"},
		{name: "failed lookup on a closed order", latest: "1", reply: "1", replyFee: "0.1", closed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := &spotOrderFee{known: tc.known, fills: tc.fills, latest: testDecimal(t, tc.latest), lookups: 1, closed: tc.closed}
			if tc.known {
				o.executed, o.fee = testDecimal(t, tc.executed), testDecimal(t, tc.fee)
			}
			f := spotOrderFees{orders: map[string]*spotOrderFee{"42": o}}
			var relayed []string
			err := f.settle("42", testDecimal(t, tc.reply), testDecimal(t, tc.replyFee), tc.ok, func(fee decimal.Decimal) error {
				relayed = append(relayed, fee.String())
				return nil
			})
			require.NoError(t, err, "settle must not error")
			var exp []string
			if tc.relayed != "" {
				exp = []string{tc.relayed}
			}
			assert.Equal(t, exp, relayed, "settle should relay the order's fee once it covers every fill reported")
			if tc.known {
				assert.Equal(t, tc.executed, o.executed.String(), "settle should keep a known total a reply does not pass")
				assert.Equal(t, tc.fee, o.fee.String(), "settle should keep the fee of a known total a reply does not pass")
			}
			assert.Zero(t, o.lookups, "settle should end the lookup")
			assert.Len(t, f.orders, tc.records, "settle should drop a closed order's record")
		})
	}

	t.Run("send failure", func(t *testing.T) {
		t.Parallel()
		f := spotOrderFees{orders: map[string]*spotOrderFee{"42": {latest: decimal.NewFromInt(1), lookups: 1}}}
		err := f.settle("42", decimal.NewFromInt(1), testDecimal(t, "0.1"), true, func(decimal.Decimal) error { return errTestDial })
		assert.ErrorIs(t, err, errTestDial, "settle should return the send's error")
	})
}

func TestSpotOrderFeeAdd(t *testing.T) {
	t.Parallel()
	fill := func(from, to int64, fee string) spotOrderFill {
		return spotOrderFill{from: decimal.NewFromInt(from), to: decimal.NewFromInt(to), fee: testDecimal(t, fee)}
	}
	for _, tc := range []struct {
		name          string
		start         spotOrderFee
		fill          spotOrderFill
		known         bool
		executed, fee string
		kept          int
	}{
		{name: "first fill", fill: fill(0, 1, "0.1"), known: true, executed: "1", fee: "0.1"},
		{name: "next fill", start: spotOrderFee{known: true, executed: decimal.NewFromInt(1), fee: testDecimal(t, "0.1")}, fill: fill(1, 3, "0.2"), known: true, executed: "3", fee: "0.3"},
		{name: "fill counted", start: spotOrderFee{known: true, executed: decimal.NewFromInt(3), fee: testDecimal(t, "0.3")}, fill: fill(1, 3, "0.2"), known: true, executed: "3", fee: "0.3"},
		{name: "fill after a gap", start: spotOrderFee{known: true, executed: decimal.NewFromInt(1), fee: testDecimal(t, "0.1")}, fill: fill(2, 3, "0.1"), known: true, executed: "1", fee: "0.1", kept: 1},
		{name: "fill without the first", fill: fill(1, 2, "0.1"), executed: "0", fee: "0", kept: 1},
		{name: "fill kept already", start: spotOrderFee{fills: []spotOrderFill{fill(1, 2, "0.1")}}, fill: fill(1, 2, "0.1"), executed: "0", fee: "0", kept: 1},
		{name: "first fill joining a kept one", start: spotOrderFee{fills: []spotOrderFill{fill(1, 2, "0.1")}}, fill: fill(0, 1, "0.1"), known: true, executed: "2", fee: "0.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := tc.start
			o.add(&tc.fill)
			assert.Equal(t, tc.known, o.known, "add should know the fee from the order's first fill")
			assert.Equal(t, tc.executed, o.executed.String(), "add should count the fills that continue what is known")
			assert.Equal(t, tc.fee, o.fee.String(), "add should total the fees of the fills it counts")
			assert.Len(t, o.fills, tc.kept, "add should keep the fills that do not continue what is known")
		})
	}
}

func TestSpotOrderFeeAbsorb(t *testing.T) {
	t.Parallel()
	o := &spotOrderFee{known: true, executed: decimal.NewFromInt(1), fee: testDecimal(t, "0.1"), fills: []spotOrderFill{
		{from: decimal.NewFromInt(4), to: decimal.NewFromInt(5), fee: testDecimal(t, "0.1")},
		{from: decimal.NewFromInt(2), to: decimal.NewFromInt(3), fee: testDecimal(t, "0.1")},
		{from: decimal.NewFromInt(1), to: decimal.NewFromInt(2), fee: testDecimal(t, "0.1")},
		{from: decimal.NewFromInt(0), to: decimal.NewFromInt(1), fee: testDecimal(t, "0.1")},
	}}
	o.absorb()
	assert.Equal(t, "3", o.executed.String(), "absorb should count the kept fills continuing what is known, in order")
	assert.Equal(t, "0.3", o.fee.String(), "absorb should total the fees of the fills it counts")
	require.Len(t, o.fills, 1, "absorb must keep the fills beyond a gap")
	assert.Equal(t, "4", o.fills[0].from.String(), "absorb should keep the fill beyond the gap")
	o.executed = decimal.NewFromInt(4)
	o.absorb()
	assert.Equal(t, "5", o.executed.String(), "absorb should count the last kept fill")
	assert.Nil(t, o.fills, "absorb should let go of the fills once it counts them all")
}

func TestSpotOrderFee(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, reply   string
		executed, fee string
		err           error
	}{
		{name: "charged fee reported negative", reply: testSpotOrderReply("FILLED", "3", "-0.3"), executed: "3", fee: "0.3"},
		{name: "charged fee reported positive", reply: testSpotOrderReply("FILLED", "3", "0.3"), executed: "3", fee: "0.3"},
		{name: "executed quantity not a number", reply: testSpotOrderReply("FILLED", "NaN", "-0.3"), err: errSpotOrderFeeNotFinite},
		{name: "infinite fee", reply: testSpotOrderReply("FILLED", "3", "Infinity"), err: errSpotOrderFeeNotFinite},
		{name: "refused", reply: `{"code":100400,"msg":"refused"}`, err: errAPIResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
			newTestSpotOrderServer(t, ex).queue(tc.reply)
			executed, fee, feeAsset, updated, err := ex.spotOrderFee(t.Context(), currency.NewBTCUSDT(), 42)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "spotOrderFee must return the lookup's error")
				return
			}
			require.NoError(t, err, "spotOrderFee must not error")
			assert.Equal(t, tc.executed, executed.String(), "spotOrderFee should return the executed quantity")
			assert.Equal(t, tc.fee, fee.String(), "spotOrderFee should return the fee as paid")
			assert.Equal(t, currency.USDT, feeAsset, "spotOrderFee should return the fee's currency")
			assert.Equal(t, time.UnixMilli(1700000002000), updated, "spotOrderFee should return the order's update time")
		})
	}
}

func TestFiniteDecimal(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		n   types.Number
		exp string
		ok  bool
	}{
		{1.5, "1.5", true},
		{types.Number(math.NaN()), "0", false},
		{types.Number(math.Inf(1)), "0", false},
		{types.Number(math.Inf(-1)), "0", false},
	} {
		t.Run(tc.n.String(), func(t *testing.T) {
			t.Parallel()
			d, ok := finiteDecimal(tc.n)
			assert.Equal(t, tc.ok, ok, "finiteDecimal should report whether the number is finite")
			assert.Equal(t, tc.exp, d.String(), "finiteDecimal should return the number as a decimal, or zero")
		})
	}
}

// testSpotOrderEventLookedUp hands an event to wsProcessSpotOrderUpdate, which must return while the lookup it starts
// is held, and waits for the lookup to arrive
func testSpotOrderEventLookedUp(tb testing.TB, ex *Exchange, conn websocket.Connection, event []byte, arrived <-chan struct{}) {
	tb.Helper()
	returned := make(chan error, 1)
	go func() { returned <- ex.wsProcessSpotOrderUpdate(context.Background(), conn, event) }()
	select {
	case err := <-returned:
		require.NoError(tb, err, "wsProcessSpotOrderUpdate must not error")
	case <-time.After(5 * time.Second):
		require.FailNow(tb, "wsProcessSpotOrderUpdate must not wait for the lookup")
	}
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		require.FailNow(tb, "the order must be looked up")
	}
}

// testOrderDetails returns the order details relayed so far, failing the test on any other payload
func testOrderDetails(tb testing.TB, ex *Exchange) []*order.Detail {
	tb.Helper()
	payloads := websocketPayloads(ex)
	details := make([]*order.Detail, 0, len(payloads))
	for _, p := range payloads {
		d, ok := p.(*order.Detail)
		require.Truef(tb, ok, "only order details must be relayed, got %T", p)
		details = append(details, d)
	}
	return details
}

// testDecimal returns s as a decimal
func testDecimal(tb testing.TB, s string) decimal.Decimal {
	tb.Helper()
	d, err := decimal.NewFromString(s)
	require.NoErrorf(tb, err, "decimal %q must parse", s)
	return d
}

// testSpotOrderEvent returns an executionReport event of spot order 42, a LIMIT buy of 3 BTC-USDT at 10, reporting an
// execution of type x, the order's status, the fill's quantity, the order's executed quantity, the fill's fee in USDT
// and its trade ID, at an event time 1700000001000 plus the trade ID's last two digits
func testSpotOrderEvent(x, status, filled, executed, fee string, tradeID uint64) []byte {
	return []byte(`{"e":"executionReport","E":` + strconv.FormatUint(1700000001000+tradeID%100, 10) + `,"s":"BTC-USDT","S":"BUY","o":"LIMIT","q":"3","p":"10","x":"` + x + `","X":"` + status + `","i":42,"l":"` + filled + `","z":"` + executed + `","L":"10","n":"` + fee + `","N":"USDT","T":1700000001000,"t":` + strconv.FormatUint(tradeID, 10) + `,"O":1700000000000,"Q":"30"}`)
}

// testSpotOrderReply returns a spot order query reply for order 42, a LIMIT buy of 3 BTC-USDT at 10, with its status,
// executed quantity and fee in USDT, last updated at 1700000002000
func testSpotOrderReply(status, executed, fee string) string {
	return `{"code":0,"data":{"symbol":"BTC-USDT","orderId":42,"price":"10","origQty":"3","executedQty":"` + executed + `","status":"` + status + `","type":"LIMIT","side":"BUY","fee":"` + fee + `","feeAsset":"USDT","time":1700000000000,"updateTime":1700000002000}}`
}

// testSpotOrderServer answers spot order queries with the replies queued, in turn, recording the order ID and API key
// of each query, and holds the queries while held is set
type testSpotOrderServer struct {
	mu      sync.Mutex
	replies []string
	keys    []string
	ids     []string
	held    chan struct{}
	arrived chan struct{}
}

// newTestSpotOrderServer serves spot order queries to ex from an in-memory server
func newTestSpotOrderServer(tb testing.TB, ex *Exchange) *testSpotOrderServer {
	tb.Helper()
	s := new(testSpotOrderServer)
	stopped := make(chan struct{})
	server := httptest.NewTestServer(tb, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openApi/spot/v1/trade/query" {
			assert.Failf(tb, "unexpected request", "%s should not be requested", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		orderID := r.URL.Query().Get("orderId")
		s.mu.Lock()
		s.keys = append(s.keys, r.Header.Get(keyHeader))
		s.ids = append(s.ids, orderID)
		var reply string
		if len(s.replies) != 0 {
			reply, s.replies = s.replies[0], s.replies[1:]
		}
		held, arrived := s.held, s.arrived
		s.arrived = nil
		s.mu.Unlock()
		if arrived != nil {
			close(arrived)
		}
		if held != nil {
			select {
			case <-held:
			case <-r.Context().Done():
				return
			case <-stopped:
				return
			}
		}
		if reply == "" {
			assert.Failf(tb, "unexpected query", "order %s should not be queried", orderID)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if after, ok := strings.CutPrefix(reply, wrapperStatusPrefix); ok {
			code, body, _ := strings.Cut(after, " ")
			status, err := strconv.Atoi(code)
			assert.NoErrorf(tb, err, "the status of reply %q should be a number", after)
			w.WriteHeader(status)
			reply = body
		}
		_, _ = io.WriteString(w, reply)
	}))
	// Registered after the server's own cleanup, so held queries end before the server closes
	tb.Cleanup(func() { close(stopped) })
	require.NoError(tb, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	testRESTEndpoints(tb, ex, server.URL)
	require.NoError(tb, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	return s
}

// queue queues replies to the queries
func (s *testSpotOrderServer) queue(replies ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replies = append(s.replies, replies...)
}

// hold holds the queries until the channel returned is closed, and returns a channel closed when the next query arrives
func (s *testSpotOrderServer) hold() (release, arrived chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.held, s.arrived = make(chan struct{}), make(chan struct{})
	return s.held, s.arrived
}

// queried returns the order IDs queried so far
func (s *testSpotOrderServer) queried() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.ids)
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
		// Serial, as the websocket is shut down after these checks
		t.Run("disabled "+a.String(), func(t *testing.T) {
			assert.Zerof(t, servers.ws[a].marketDials(), "disabled %s should dial no market connection", a)
			assert.Emptyf(t, servers.ws[a].accountKeys(), "disabled %s should dial no account connection", a)
		})
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
			t.Run(strconv.Itoa(tc.pairs)+" pairs", func(t *testing.T) {
				t.Parallel()
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
			})
		}
	})
}
