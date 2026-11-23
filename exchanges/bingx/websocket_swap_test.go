package bingx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

func TestWebsocketHandleSwapData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	enableTestTradeFeed(ex)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, testOrderbookSubscription(testSwapPair, asset.USDTMarginedFutures, 5)), "wsClaimBook must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsSwap.jsonl", func(ctx context.Context, frame []byte) error {
		return ex.wsHandleSwapData(ctx, conn, frame)
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
		case *WebsocketSwapLastPrice:
			counts["last prices"]++
		case *WebsocketMarkPrice:
			counts["mark prices"]++
		case *WebsocketSwapBookTicker:
			counts["book tickers"]++
		default:
			assert.Failf(t, "unexpected payload", "%T relayed", p)
		}
	}
	assert.Equal(t, map[string]int{"trades": 2, "candles": 2, "books": 1, "tickers": 1, "last prices": 1, "mark prices": 1, "book tickers": 1}, counts, "every fixture frame should be relayed, the books of the stream the connection feeds")

	require.NoError(t, ex.wsHandleSwapData(t.Context(), conn, []byte(websocketSwapPing)), "a heartbeat must be answered")
	assert.Equal(t, []testWebsocketMessage{{limit: websocketHeartbeatRate, message: websocketSwapPong}}, conn.messages(), "the heartbeat should be answered")
	for _, tc := range []struct {
		name, frame string
		err         error
	}{
		{"error without an id", `{"code":80015,"msg":"dataType not support","dataType":"","data":null}`, errWebsocketErrorMessage},
		{"incremental depth", `{"code":0,"dataType":"BTC-USDT@incrDepth","data":{"action":"all"}}`, errWebsocketUnhandledMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, ex.wsHandleSwapData(t.Context(), conn, []byte(tc.frame)), tc.err, "wsHandleSwapData should return the expected error for %s", tc.name)
		})
	}
}

func TestWebsocketProcessSwapTrades(t *testing.T) {
	t.Parallel()
	data := fixtureData(t, "testdata/wsSwap.jsonl", "@trade")
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSwapTrades(data), "wsProcessSwapTrades must not error without a trade feed")
	assert.Empty(t, websocketPayloads(ex), "trades should not be relayed without a trade feed")

	enableTestTradeFeed(ex)
	require.NoError(t, ex.wsProcessSwapTrades(data), "wsProcessSwapTrades must not error")
	exp := []trade.Data{{
		TID:          "759060171",
		Exchange:     ex.Name,
		CurrencyPair: testSwapPair,
		AssetType:    asset.USDTMarginedFutures,
		Side:         order.Sell,
		Price:        73657.5,
		Amount:       0.0006,
		Timestamp:    time.UnixMilli(1794390762865),
	}}
	assert.Equal(t, exp, waitForWebsocketPayload[[]trade.Data](t, ex, nil), "the trades should be relayed")

	mixed := []byte(`[{"t":"1","q":"1","p":"1","T":1794390762865,"m":false,"s":"ETH-USDT"},{"t":"2","q":"2","p":"2","T":1794390762866,"m":"false","s":"BTC-USDT"}]`)
	require.NoError(t, ex.wsProcessSwapTrades(mixed), "wsProcessSwapTrades must not error")
	got := waitForWebsocketPayload[[]trade.Data](t, ex, nil)
	require.Len(t, got, 1, "only the enabled pair's trade must be relayed")
	assert.Equal(t, "2", got[0].TID, "the enabled pair's trade should be relayed, its buyer flag decoded from a string")
	assert.Equal(t, order.Buy, got[0].Side, "a buyer that was the taker should make a buy")
	assert.ErrorIs(t, ex.wsProcessSwapTrades([]byte(`[{"s":"DOGE-USDT"}]`)), currency.ErrPairNotFound, "wsProcessSwapTrades should error for an unknown pair")
	assert.Error(t, ex.wsProcessSwapTrades([]byte(`{"t":"1"}`)), "wsProcessSwapTrades should error for a push that is not an array")
}

func TestWebsocketProcessSwapTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSwapTicker(t.Context(), fixtureData(t, "testdata/wsSwap.jsonl", "@ticker")), "wsProcessSwapTicker must not error")
	exp := &ticker.Price{
		Last:                73657.5,
		LastSize:            0.0002,
		High:                74813.4,
		Low:                 72705.9,
		Bid:                 73657.5,
		BidSize:             11.9156,
		Ask:                 73665.1,
		AskSize:             49.4739,
		BaseVolume:          15427.51,
		QuoteVolume:         1136169848.16,
		Open:                73670,
		PercentChange24Hour: -0.02,
		Pair:                testSwapPair,
		ExchangeName:        ex.Name,
		AssetType:           asset.USDTMarginedFutures,
		LastUpdated:         time.UnixMilli(1794390763663),
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*ticker.Price](t, ex, nil), "the ticker should be relayed")
	_, err := ticker.GetTicker(ex.Name, testSwapPair, asset.USDTMarginedFutures)
	assert.NoError(t, err, "the ticker should be stored")

	require.NoError(t, ex.wsProcessSwapTicker(t.Context(), []byte(`{"s":"ETH-USDT","c":"1"}`)), "wsProcessSwapTicker must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's ticker should not be relayed")
	assert.Error(t, ex.wsProcessSwapTicker(t.Context(), []byte(`{"c":true}`)), "wsProcessSwapTicker should error for a ticker that does not decode")
}

func TestWebsocketProcessSwapKlines(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessSwapKlines(t.Context(), "BTC-USDT", "1m", fixtureData(t, "testdata/wsSwap.jsonl", "@kline_1m")), "wsProcessSwapKlines must not error")
	exp := kline.Item{
		Exchange: ex.Name,
		Pair:     testSwapPair,
		Asset:    asset.USDTMarginedFutures,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:   time.UnixMilli(1794390720000),
			Open:   73675.6,
			High:   73680.3,
			Low:    73657.5,
			Close:  73657.5,
			Volume: 2.1246,
		}},
	}
	assert.Equal(t, exp, waitForWebsocketPayload[kline.Item](t, ex, nil), "a candle received after its end should be relayed complete")

	open := []byte(`[{"c":"1","o":"1","h":"1","l":"1","v":"1","T":` + strconv.FormatInt(time.Now().Truncate(time.Hour).UnixMilli(), 10) + `}]`)
	require.NoError(t, ex.wsProcessSwapKlines(t.Context(), "BTC-USDT", "1h", open), "wsProcessSwapKlines must not error")
	assert.Equal(t, kline.PartialCandle, waitForWebsocketPayload[kline.Item](t, ex, nil).Candles[0].ValidationIssues, "a candle received before its end should be partial")

	require.NoError(t, ex.wsProcessSwapKlines(t.Context(), "BTC-USDT", "1m", []byte(`[]`)), "wsProcessSwapKlines must not error for an empty push")
	require.NoError(t, ex.wsProcessSwapKlines(t.Context(), "ETH-USDT", "1m", nil), "wsProcessSwapKlines must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "an empty push or a disabled pair should relay nothing")
	assert.ErrorIs(t, ex.wsProcessSwapKlines(t.Context(), "BTC-USDT", "1min", nil), kline.ErrInvalidInterval, "wsProcessSwapKlines should reject another product's interval")
	assert.Error(t, ex.wsProcessSwapKlines(t.Context(), "BTC-USDT", "1m", []byte(`{}`)), "wsProcessSwapKlines should error for a push that is not an array")
}

func TestWebsocketProcessSwapOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, testOrderbookSubscription(testSwapPair, asset.USDTMarginedFutures, 5)), "wsClaimBook must not error")
	frame := fixtureLine(t, "testdata/wsSwap.jsonl", "@depth5")
	data, _, _, err := jsonparser.Get(frame, "data")
	require.NoError(t, err, "the push must have data")
	require.NoError(t, ex.wsProcessSwapOrderbook(t.Context(), conn, "BTC-USDT@depth5", frame, data), "wsProcessSwapOrderbook must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(testSwapPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "the push must load the book")
	assert.Equal(t, orderbook.Levels{{Price: 73657.5, Amount: 4.6893}, {Price: 73657.4, Amount: 0.0239}, {Price: 73656.9, Amount: 0.0127}, {Price: 73656.7, Amount: 0.0134}, {Price: 73656.2, Amount: 0.7782}}, book.Bids, "the bids should be loaded highest first")
	assert.Equal(t, orderbook.Levels{{Price: 73657.6, Amount: 48.512}, {Price: 73657.9, Amount: 0.2912}, {Price: 73658, Amount: 0.0063}, {Price: 73658.3, Amount: 0.9777}, {Price: 73658.8, Amount: 0.0095}}, book.Asks, "the asks should be loaded lowest first")
	assert.Equal(t, time.UnixMilli(1794390762775), book.LastUpdated, "the book should be dated by ts")

	replacement := []byte(`{"code":0,"dataType":"BTC-USDT@depth5","ts":1794390763276,"data":{"bids":[["73657.5","9.4073"]],"asks":[["73657.6","61.0548"]]}}`)
	data, _, _, err = jsonparser.Get(replacement, "data")
	require.NoError(t, err, "the push must have data")
	require.NoError(t, ex.wsProcessSwapOrderbook(t.Context(), conn, "BTC-USDT@depth5", replacement, data), "wsProcessSwapOrderbook must not error")
	book, err = ex.Websocket.Orderbook.GetOrderbook(testSwapPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "the push must load the book")
	assert.Equal(t, orderbook.Levels{{Price: 73657.5, Amount: 9.4073}}, book.Bids, "a push should replace the book's bids")
	assert.Equal(t, orderbook.Levels{{Price: 73657.6, Amount: 61.0548}}, book.Asks, "a push should replace the book's asks")

	require.NoError(t, ex.wsProcessSwapOrderbook(t.Context(), conn, "ETH-USDT@depth5", nil, nil), "wsProcessSwapOrderbook must not error for a disabled pair")
	assert.Error(t, ex.wsProcessSwapOrderbook(t.Context(), conn, "BTC-USDT@depth5", nil, []byte(`{"bids":"x"}`)), "wsProcessSwapOrderbook should error for a push that does not decode")
}

func TestWebsocketHandleSwapAccountData(t *testing.T) {
	t.Parallel()
	server := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, server.URL)
	conn := &testWebsocketConnection{}
	trackTestConnection(t, ex, conn, false)
	testexch.FixtureToDataHandler(t, "testdata/wsSwapAccount.jsonl", func(ctx context.Context, frame []byte) error {
		return ex.wsHandleSwapAccountData(ctx, conn, frame)
	})
	payloads := websocketPayloads(ex)
	require.Len(t, payloads, 4, "each documented event must be relayed")
	assert.IsType(t, &order.Detail{}, payloads[0], "the order event should be relayed as an order detail")
	assert.IsType(t, []WebsocketPerpetualBalance{}, payloads[1], "the account event's balances should be relayed as reported")
	testWaitBackground(t, ex)
	assert.Equal(t, 1, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "the account event should refresh the balances over REST")
	assert.IsType(t, []futures.Position{}, payloads[2], "the account event's positions should be relayed")
	assert.IsType(t, &WebsocketPerpetualConfigUpdate{}, payloads[3], "the configuration event should be relayed as it is")
	for _, p := range payloads {
		if d, ok := p.(*order.Detail); ok {
			assert.Equal(t, asset.USDTMarginedFutures, d.AssetType, "the order should be USDT-M's")
		}
	}
}

func TestWebsocketHandlePerpetualAccountData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	conn := &testWebsocketConnection{}
	_, session := trackTestConnection(t, ex, conn, false)
	enveloped := []byte(`{"code":0,"dataType":"ORDER_TRADE_UPDATE","data":{"e":"ORDER_TRADE_UPDATE","E":1758608899350,"o":{"s":"BTC-USD","i":9,"S":"BUY","o":"LIMIT","q":"1","p":"1","x":"NEW","X":"NEW"}}}`)
	require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, enveloped, asset.CoinMarginedFutures), "an enveloped order event must be handled")
	d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
	assert.Equal(t, "9", d.OrderID, "the enveloped order event should be relayed")
	assert.Equal(t, asset.CoinMarginedFutures, d.AssetType, "the order should be of the connection's asset")

	require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, []byte(`{"ping":"a","time":"b"}`), asset.CoinMarginedFutures), "a heartbeat must be answered")
	assert.Len(t, conn.messages(), 1, "the heartbeat should be answered")
	require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, []byte(`{"e":"listenKeyExpired","E":1794390762862}`), asset.USDTMarginedFutures), "an expired key must be handled")
	ex.wsState.mu.Lock()
	assert.Equal(t, session.token(), ex.wsState.recovering, "an expired key should reconnect the session")
	ex.wsState.mu.Unlock()
	testWaitBackground(t, ex)
	assert.ErrorIs(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, []byte(`{"e":"MARGIN_CALL"}`), asset.USDTMarginedFutures), errWebsocketUnhandledEvent, "an unknown event should be reported")
	assert.ErrorIs(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, []byte(`{"code":100400,"msg":"bad"}`), asset.USDTMarginedFutures), errWebsocketErrorMessage, "an error frame should be reported")
}

func TestWebsocketProcessPerpetualOrderUpdate(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), state, asset.USDTMarginedFutures, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "TRADE_UPDATE")), "wsProcessPerpetualOrderUpdate must not error")
	exp := &order.Detail{
		Price:                112848.1,
		Amount:               0.001,
		AverageExecutedPrice: 112848.1,
		ExecutedAmount:       0.001,
		Cost:                 112.8481,
		Fee:                  0.05642405,
		FeeAsset:             currency.USDT,
		Exchange:             ex.Name,
		OrderID:              "1970374651259388000",
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		AssetType:            asset.USDTMarginedFutures,
		Date:                 time.UnixMilli(1758608899316),
		LastUpdated:          time.UnixMilli(1758608899316),
		Pair:                 testSwapPair,
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*order.Detail](t, ex, nil), "the USDT-M order event should be relayed as an order detail")

	require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), state, asset.CoinMarginedFutures, fixtureLine(t, "testdata/wsCoinMarginedAccount.jsonl", "ORDER_TRADE_UPDATE")), "wsProcessPerpetualOrderUpdate must not error")
	exp = &order.Detail{
		Price:                3.436799,
		Amount:               10,
		AverageExecutedPrice: 3.4367,
		ExecutedAmount:       10,
		Fee:                  0.00145488,
		FeeAsset:             currency.USD,
		Exchange:             ex.Name,
		OrderID:              "1976541385109553200",
		Type:                 order.Market,
		Side:                 order.Sell,
		Status:               order.Filled,
		AssetType:            asset.CoinMarginedFutures,
		Date:                 time.UnixMilli(1760079163296),
		LastUpdated:          time.UnixMilli(1760079163296),
		Pair:                 currency.NewPairWithDelimiter("SUI", "USD", currency.DashDelimiter),
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*order.Detail](t, ex, nil), "the coin-M order event should be relayed filled, as its executed quantity is its whole quantity, without a cost, dated by its event without a trade time")

	reduceOnly := []byte(`{"e":"ORDER_TRADE_UPDATE","E":1758608899350,"o":{"s":"BTC-USDT","i":3,"S":"BUY","o":"STOP","q":"2","p":"10","sp":"9","x":"NEW","X":"NEW","ro":"true","n":"-1"}}`)
	require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), state, asset.USDTMarginedFutures, reduceOnly), "wsProcessPerpetualOrderUpdate must not error")
	got := waitForWebsocketPayload[*order.Detail](t, ex, nil)
	assert.True(t, got.ReduceOnly, "a reduce only order should be relayed as one")
	assert.Zero(t, got.Fee, "an event without a fill should carry no fee")
	assert.InDelta(t, 2, got.RemainingAmount, 1e-9, "an order without fills should have its whole quantity remaining")
	assert.InDelta(t, 9, got.TriggerPrice, 1e-9, "the trigger price should be relayed")
	assert.Equal(t, order.StopLimit, got.Type, "STOP should be a stop limit order")
	assert.Error(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), state, asset.USDTMarginedFutures, []byte(`{"o":{"sg":1}}`)), "wsProcessPerpetualOrderUpdate should error for an event that does not decode")

	for _, tc := range []struct {
		name, event string
		err         error
	}{
		{"status", `{"o":{"s":"BTC-USDT","S":"BUY","o":"LIMIT","X":"WAITING","i":1}}`, order.ErrUnsupportedStatusType},
		{"type", `{"o":{"s":"BTC-USDT","S":"BUY","o":"ICEBERG","X":"NEW","i":1}}`, order.ErrUnrecognisedOrderType},
		{"side", `{"o":{"s":"BTC-USDT","S":"BOTH","o":"LIMIT","X":"NEW","i":1}}`, order.ErrSideIsInvalid},
		{"symbol", `{"o":{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"NEW","i":1}}`, errWebsocketInvalidSymbol},
	} {
		t.Run("unclassified "+tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			conn := &testWebsocketConnection{}
			state, _ := trackTestConnection(t, ex, conn, false)
			require.NoErrorf(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), state, asset.USDTMarginedFutures, []byte(tc.event)), "wsProcessPerpetualOrderUpdate must relay an unclassified %s", tc.name)
			classification := waitForWebsocketPayload[order.ClassificationError](t, ex, nil)
			assert.ErrorIsf(t, classification.Err, tc.err, "an unknown %s should be relayed as a classification error", tc.name)
			assert.Equal(t, "1", classification.OrderID, "the classification error should name the order")
		})
	}

	t.Run("coin-M statuses in the order store", func(t *testing.T) {
		t.Parallel()
		event := func(execution, status, quantity, executed string) []byte {
			return []byte(`{"e":"ORDER_TRADE_UPDATE","E":1760079163296,"o":{"s":"SUI-USD","i":43,"S":"SELL","o":"LIMIT","q":"` + quantity + `","p":"3.4","ap":"3.4","x":"` + execution + `","X":"` + status + `","N":"SUI","n":"-0.001","z":"` + executed + `"}}`)
		}
		for _, tc := range []struct {
			name, orderID string
			events        [][]byte
			exp           order.Status
		}{
			{name: "the documented fill of a stored new order", orderID: "1976541385109553200", events: [][]byte{fixtureLine(t, "testdata/wsCoinMarginedAccount.jsonl", "ORDER_TRADE_UPDATE")}, exp: order.Filled},
			{name: "a part fill reporting Unknown after the order's cancellation", orderID: "43", events: [][]byte{event("CANCELED", "CANCELLED", "10", "3"), event("TRADE", "Unknown", "10", "3")}, exp: order.Cancelled},
			{name: "a part fill reporting Unknown", orderID: "43", events: [][]byte{event("TRADE", "Unknown", "10", "3")}, exp: order.New},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t)
				state, _ := trackTestConnection(t, ex, &testWebsocketConnection{}, false)
				stored := &order.Detail{Exchange: ex.Name, OrderID: tc.orderID, AssetType: asset.CoinMarginedFutures, Amount: 10, Status: order.New}
				for _, data := range tc.events {
					require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), state, asset.CoinMarginedFutures, data), "wsProcessPerpetualOrderUpdate must not error")
					relayed := waitForWebsocketPayload[*order.Detail](t, ex, nil)
					require.Equal(t, tc.orderID, relayed.OrderID, "the event's order must be relayed")
					require.NoError(t, stored.UpdateOrderFromDetail(relayed), "the order store must take the relayed order")
				}
				assert.Equal(t, tc.exp, stored.Status, "the order store should hold the status the coin-M events show")
			})
		}
	})

	t.Run("connection not tracked", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		event := []byte(`{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"1","x":"TRADE","X":"PARTIALLY_FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`)
		require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), nil, asset.USDTMarginedFutures, event), "wsProcessPerpetualOrderUpdate must not error on a connection not tracked")
		d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
		assert.InDelta(t, 1, d.ExecutedAmount, 1e-12, "the order should be relayed")
		assert.Zero(t, d.Fee, "an event on a connection keeping no totals should carry no fee")
	})

	t.Run("fees over several fills", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conns := map[asset.Item]*testWebsocketConnection{asset.USDTMarginedFutures: {}, asset.CoinMarginedFutures: {}}
		states := make(map[asset.Item]*websocketConnection, len(conns))
		for a, conn := range conns {
			states[a], _ = trackTestConnection(t, ex, conn, false)
		}
		stored := map[asset.Item]*order.Detail{
			asset.USDTMarginedFutures: {Exchange: ex.Name, OrderID: "42", AssetType: asset.USDTMarginedFutures, Amount: 3, Status: order.New},
			asset.CoinMarginedFutures: {Exchange: ex.Name, OrderID: "42", AssetType: asset.CoinMarginedFutures, Amount: 3, Status: order.New},
		}
		for i, step := range []struct {
			a                                        asset.Item
			orderID                                  uint64
			executionType, status, executed, feeSent string
			fee                                      float64
		}{
			{asset.USDTMarginedFutures, 42, "NEW", "NEW", "0", "0", 0},
			{asset.USDTMarginedFutures, 42, "TRADE", "PARTIALLY_FILLED", "1", "-0.1", 0.1},
			{asset.CoinMarginedFutures, 42, "TRADE", "PARTIALLY_FILLED", "1", "-0.01", 0.01},
			{asset.USDTMarginedFutures, 43, "TRADE", "PARTIALLY_FILLED", "1", "-5", 5},
			{asset.USDTMarginedFutures, 42, "NEW", "NEW", "0", "0", 0},
			{asset.USDTMarginedFutures, 42, "TRADE", "PARTIALLY_FILLED", "2", "0.05", 0.05},
			{asset.USDTMarginedFutures, 42, "TRADE", "PARTIALLY_FILLED", "2", "0.05", 0.05},
			{asset.CoinMarginedFutures, 42, "TRADE", "PARTIALLY_FILLED", "2", "-0.02", 0.03},
			{asset.USDTMarginedFutures, 42, "CALCULATED", "FILLED", "3", "-0.3", 0.35},
			{asset.CoinMarginedFutures, 42, "TRADE", "Unknown", "3", "-0.03", 0.06},
			{asset.USDTMarginedFutures, 43, "CANCELED", "CANCELED", "1", "0", 0},
			{asset.USDTMarginedFutures, 44, "TRADE", "PARTIALLY_FILLED", "1", "-0.4", 0.4},
			{asset.USDTMarginedFutures, 44, "EXPIRED", "EXPIRED", "1", "0", 0},
		} {
			// The steps run in order, as each event's fee adds to those of the events before it
			t.Run(fmt.Sprintf("%d %s order %d %s at %s", i, step.a, step.orderID, step.executionType, step.executed), func(t *testing.T) {
				symbol := "BTC-USDT"
				if step.a == asset.CoinMarginedFutures {
					symbol = "BTC-USD"
				}
				event := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":%q,"i":%d,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":%q,"X":%q,"N":"USDT","n":%q,"T":1758608899001}}`, symbol, step.orderID, step.executed, step.executionType, step.status, step.feeSent)
				require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), states[step.a], step.a, event), "wsProcessPerpetualOrderUpdate must not error")
				d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
				assert.InDelta(t, step.fee, d.Fee, 1e-12, "the event should carry the order's fee so far")
				if step.orderID == 42 {
					require.NoError(t, stored[step.a].UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
				}
			})
		}
		assert.InDelta(t, 0.35, stored[asset.USDTMarginedFutures].Fee, 1e-12, "the stored USDT-M order should hold the fees of all its fills, less the rebate")
		assert.InDelta(t, 0.06, stored[asset.CoinMarginedFutures].Fee, 1e-12, "the stored coin-M order should hold the fees of all its fills")
		for a, state := range states {
			state.fees.mu.Lock()
			assert.Emptyf(t, state.fees.orders, "the %s connection should forget the orders no longer open", a)
			state.fees.mu.Unlock()
		}
	})

	t.Run("fees of an order read over REST", func(t *testing.T) {
		t.Parallel()
		type feeRow struct {
			name     string
			a        asset.Item
			streamed bool
			// quantity sends a coin-M order's size as its quantity rather than its origQty
			quantity bool
			steps    []perpetualFeeStep
			stored   float64
			totals   int
		}
		// A sequence runs for each perpetual asset and each of its reads, which take the place of its steps reading "read"
		sequences := []struct {
			name   string
			reads  []string
			steps  []perpetualFeeStep
			stored float64
			totals int
		}{
			{"two thirds filled, then its last fill", []string{"list", "get", "connect"}, []perpetualFeeStep{
				{read: "read", executed: "2", fee: "-0.23", status: "PARTIALLY_FILLED"},
				{executed: "3", fee: "-0.1", status: "FILLED", relayed: 0.33},
			}, 0.33, 0},
			{"a third filled, then two fills", []string{"list", "get", "connect"}, []perpetualFeeStep{
				{read: "read", executed: "1", fee: "-0.1", status: "PARTIALLY_FILLED"},
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.3},
				{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6},
			}, 0.6, 0},
			{"a fill the read counted arriving after it", []string{"list", "get"}, []perpetualFeeStep{
				{read: "read", executed: "2", fee: "-0.23", status: "PARTIALLY_FILLED"},
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.23},
				{executed: "3", fee: "-0.1", status: "FILLED", relayed: 0.33},
			}, 0.33, 0},
			{"a read as far as a total missing its first fills", []string{"list", "get"}, []perpetualFeeStep{
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.2},
				{read: "read", executed: "2", fee: "-0.3", status: "PARTIALLY_FILLED"},
				{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6},
			}, 0.6, 0},
			{"a read behind a total missing its first fills", []string{"list", "get"}, []perpetualFeeStep{
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.2},
				{read: "read", executed: "1", fee: "-0.1", status: "PARTIALLY_FILLED"},
				{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6},
			}, 0.6, 0},
			{"a read behind a total started at a NEW event reporting fills", []string{"list", "get"}, []perpetualFeeStep{
				{executionType: "NEW", executed: "1", fee: "0", status: "PARTIALLY_FILLED"},
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.2},
				{read: "read", executed: "1", fee: "-0.1", status: "PARTIALLY_FILLED"},
				{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6},
			}, 0.6, 0},
			{"a read behind a total counting every fill", []string{"list", "get"}, []perpetualFeeStep{
				{executionType: "NEW", executed: "0", fee: "0", status: "NEW"},
				{executed: "1", fee: "-0.1", status: "PARTIALLY_FILLED", relayed: 0.1},
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.3},
				{read: "read", executed: "1", fee: "-0.15", status: "PARTIALLY_FILLED"},
				{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6},
			}, 0.6, 0},
			{"a rebate the read reported", []string{"list", "get"}, []perpetualFeeStep{
				{read: "read", executed: "1", fee: "0.05", status: "PARTIALLY_FILLED"},
				{executed: "2", fee: "-0.1", status: "PARTIALLY_FILLED", relayed: 0.05},
			}, 0.05, 1},
			{"a closed order read before its closing event", []string{"get"}, []perpetualFeeStep{
				{executed: "1", fee: "-0.1", status: "PARTIALLY_FILLED", relayed: 0.1},
				{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.3},
				{read: "read", executed: "3", fee: "-0.6", status: "FILLED"},
				{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6},
			}, 0.6, 0},
			{"orders read without an account stream", nil, []perpetualFeeStep{
				{read: "list", executed: "2", fee: "-0.23", status: "PARTIALLY_FILLED"},
				{read: "get", executed: "2", fee: "-0.23", status: "PARTIALLY_FILLED"},
				{read: "get", executed: "3", fee: "-0.33", status: "FILLED"},
			}, 0.33, 0},
		}
		var rows []feeRow
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, seq := range sequences {
				if seq.reads == nil {
					rows = append(rows, feeRow{name: seq.name + ", " + a.String(), a: a, steps: seq.steps, stored: seq.stored, totals: seq.totals})
					continue
				}
				for _, read := range seq.reads {
					steps := slices.Clone(seq.steps)
					for i := range steps {
						if steps[i].read == "read" {
							steps[i].read = read
						}
					}
					rows = append(rows, feeRow{name: seq.name + ", " + a.String() + " " + read, a: a, streamed: true, steps: steps, stored: seq.stored, totals: seq.totals})
				}
			}
		}
		rows = append(rows, feeRow{"coin-M order listed with its quantity", asset.CoinMarginedFutures, true, true, []perpetualFeeStep{
			{read: "list", executed: "2", fee: "-0.0023", status: "PARTIALLY_FILLED"},
			{executed: "3", fee: "-0.001", status: "FILLED", relayed: 0.0033},
		}, 0.0033, 0})
		for _, tc := range rows {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestPerpetualOrderServer(t, ex, tc.a, tc.quantity)
				conn := &testWebsocketConnection{}
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				account := tc.a
				if !tc.streamed {
					account = asset.Spot
				}
				state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, account, creds)
				stored := &order.Detail{Exchange: ex.Name, OrderID: "42", AssetType: tc.a, Pair: server.pair, Amount: 3, Status: order.New}
				for i, step := range tc.steps {
					// The steps run in order, as each read or event builds on the total the steps before it left
					t.Run(strconv.Itoa(i)+" "+cmp.Or(step.read, "event")+" at "+step.executed+" executed", func(t *testing.T) {
						server.setOrder(step.executed, step.fee, step.status)
						if d := server.step(t, ex, conn, &step); d != nil {
							require.NoError(t, stored.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
						}
					})
				}
				assert.InDelta(t, tc.stored, stored.Fee, 1e-12, "the stored order should hold the fees of all its fills")
				state.fees.mu.Lock()
				assert.Len(t, state.fees.orders, tc.totals, "only an open order's total should be kept, and only on its account's connection")
				state.fees.mu.Unlock()
			})
		}
	})

	t.Run("credentials changing during a read", func(t *testing.T) {
		t.Parallel()
		other := &accounts.Credentials{Key: "other-key", Secret: "other-secret"}
		type credentialsRow struct {
			name string
			a    asset.Item
			read string
			// inContext passes the read's credentials in its context rather than as the exchange's
			inContext bool
			// resolved changes the exchange's credentials as soon as the read has resolved its own, rather than once its
			// request arrives
			resolved bool
		}
		rows := make([]credentialsRow, 0, 16)
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, read := range []string{"list", "get"} {
				for _, inContext := range []bool{false, true} {
					for _, resolved := range []bool{false, true} {
						name := fmt.Sprintf("%s %s, credentials in the context %t, changed once resolved %t", a, read, inContext, resolved)
						rows = append(rows, credentialsRow{name, a, read, inContext, resolved})
					}
				}
			}
		}
		for _, tc := range rows {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestPerpetualOrderServer(t, ex, tc.a, false)
				readCreds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				ctx := t.Context()
				if tc.inContext {
					readCreds = &accounts.Credentials{Key: "context-key", Secret: "context-secret"}
					ctx = accounts.DeployCredentialsToContext(ctx, readCreds)
				}
				session := newTestSession(t, ex).ch
				readConn, otherConn := &testWebsocketConnection{}, &testWebsocketConnection{}
				readState := ex.wsTrackConnection(readConn, session, false, tc.a, readCreds)
				otherState := ex.wsTrackConnection(otherConn, session, false, tc.a, other)
				server.setOrder("2", "-0.23", "PARTIALLY_FILLED")
				if tc.resolved {
					ctx = &testCredentialsChangingContext{Context: ctx, change: func() { ex.SetCredentials(other) }}
					require.NoError(t, server.readError(ctx, ex, tc.read), "the read must not error")
				} else {
					arrived, release := server.holdNext()
					readErr := make(chan error, 1)
					go func() { readErr <- server.readError(ctx, ex, tc.read) }()
					select {
					case <-arrived:
					case <-time.After(5 * time.Second):
						require.FailNow(t, "the read's request must arrive")
					}
					ex.SetCredentials(other)
					close(release)
					select {
					case err := <-readErr:
						require.NoError(t, err, "the read must not error")
					case <-time.After(5 * time.Second):
						require.FailNow(t, "the read must return")
					}
				}
				assert.Equal(t, []string{readCreds.Key}, server.requestKeys(), "the read should be sent with its own account's key")
				for _, s := range []struct {
					name   string
					state  *websocketConnection
					totals int
				}{{"the read's account", readState, 1}, {"the account the default credentials changed to", otherState, 0}} {
					t.Run(s.name, func(t *testing.T) {
						s.state.fees.mu.Lock()
						defer s.state.fees.mu.Unlock()
						assert.Lenf(t, s.state.fees.orders, s.totals, "the connection should keep %d totals", s.totals)
					})
				}
				event := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":%q,"i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, server.symbol)
				require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), readState, tc.a, event), "wsProcessPerpetualOrderUpdate must not error")
				assert.InDelta(t, 0.33, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, 1e-12, "the read's account's event should carry the fees of all the order's fills")
			})
		}
	})

	t.Run("an order closing while a read is in flight", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			a    asset.Item
			read string
		}{
			{asset.USDTMarginedFutures, "list"},
			{asset.USDTMarginedFutures, "get"},
			{asset.CoinMarginedFutures, "list"},
			{asset.CoinMarginedFutures, "get"},
		} {
			t.Run(tc.a.String()+" "+tc.read, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestPerpetualOrderServer(t, ex, tc.a, false)
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				conn := &testWebsocketConnection{}
				state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, tc.a, creds)
				for i, step := range []perpetualFeeStep{
					{executionType: "NEW", executed: "0", fee: "0", status: "NEW"},
					{executed: "1", fee: "-0.1", status: "PARTIALLY_FILLED", relayed: 0.1},
					{executed: "2", fee: "-0.2", status: "PARTIALLY_FILLED", relayed: 0.3},
				} {
					// The steps run in order, as each event builds on the total the events before it left
					t.Run(strconv.Itoa(i)+" event at "+step.executed+" executed", func(t *testing.T) {
						server.step(t, ex, conn, &step)
					})
				}
				server.setOrder("2", "-0.3", "PARTIALLY_FILLED")
				arrived, release := server.holdNext()
				readErr := make(chan error, 1)
				go func() { readErr <- server.readError(t.Context(), ex, tc.read) }()
				select {
				case <-arrived:
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the read's request must arrive")
				}
				server.step(t, ex, conn, &perpetualFeeStep{executed: "3", fee: "-0.3", status: "FILLED", relayed: 0.6})
				close(release)
				select {
				case err := <-readErr:
					require.NoError(t, err, "the read must not error")
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the read must return")
				}
				state.fees.mu.Lock()
				assert.Empty(t, state.fees.orders, "a read begun before the order closed should not start its total again")
				assert.Empty(t, state.fees.reads, "the read should end")
				assert.Empty(t, state.fees.closed, "the close should be dropped once no read begun before it is in flight")
				state.fees.mu.Unlock()
			})
		}
	})

	t.Run("several connections streaming the account", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			for _, read := range []string{"list", "get"} {
				t.Run(a.String()+" "+read, func(t *testing.T) {
					t.Parallel()
					ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
					server := newTestPerpetualOrderServer(t, ex, a, false)
					creds, err := ex.GetCredentials(t.Context())
					require.NoError(t, err, "GetCredentials must not error")
					session := newTestSession(t, ex).ch
					first, second, down := &testWebsocketConnection{}, &testWebsocketConnection{}, &testWebsocketConnection{}
					states := make([]*websocketConnection, 0, 2)
					for _, conn := range []*testWebsocketConnection{first, second} {
						states = append(states, ex.wsTrackConnection(conn, session, false, a, creds))
					}
					downState := ex.wsTrackConnection(down, session, false, a, creds)
					down.down.Store(true)
					server.setOrder("2", "-0.23", "PARTIALLY_FILLED")
					server.read(t.Context(), t, ex, read)
					for i, name := range []string{"first", "second"} {
						// The connections share the exchange whose relayed order each case reads
						t.Run(name, func(t *testing.T) {
							server.step(t, ex, []*testWebsocketConnection{first, second}[i], &perpetualFeeStep{executed: "3", fee: "-0.1", status: "FILLED", relayed: 0.33})
							states[i].fees.mu.Lock()
							assert.Empty(t, states[i].fees.orders, "the connection should forget the order once it closes")
							states[i].fees.mu.Unlock()
						})
					}
					downState.fees.mu.Lock()
					assert.Empty(t, downState.fees.orders, "a connection that is down should get no total")
					downState.fees.mu.Unlock()
				})
			}
		}
	})
}

func TestWebsocketPerpetualOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name               string
		a                  asset.Item
		execution          ExecutionType
		status             string
		quantity, executed types.Number
		exp                order.Status
		err                error
	}{
		{name: "coin-M Pending", a: asset.CoinMarginedFutures, execution: ExecutionTypeNew, status: "Pending", quantity: 10, executed: 0, exp: order.Pending},
		{name: "coin-M filled", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "filled", quantity: 10, executed: 10, exp: order.Filled},
		{name: "coin-M Cancelled", a: asset.CoinMarginedFutures, execution: ExecutionTypeCancelled, status: "Cancelled", quantity: 10, executed: 3, exp: order.Cancelled},
		{name: "coin-M PARTIALLY_FILLED", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "PARTIALLY_FILLED", quantity: 10, executed: 3, exp: order.PartiallyFilled},
		{name: "coin-M Pending with no quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Pending", quantity: 0, executed: 0, exp: order.Pending},
		{name: "coin-M Unknown fill of the whole quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: 10, executed: 10, exp: order.Filled},
		{name: "coin-M UNKNOWN fill of the whole quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "UNKNOWN", quantity: 10, executed: 10, exp: order.Filled},
		{name: "coin-M Unknown calculated fill of the whole quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeCalculated, status: "Unknown", quantity: 10, executed: 10, exp: order.Filled},
		{name: "coin-M Unknown part fill", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: 10, executed: 3, exp: order.UnknownStatus},
		{name: "coin-M Unknown fill beyond the quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: 1, executed: 2, exp: order.UnknownStatus},
		{name: "coin-M Unknown fill without a quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: 0, executed: 0, exp: order.UnknownStatus},
		{name: "coin-M Unknown fill of a quantity that is not a number", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: types.Number(math.NaN()), executed: types.Number(math.NaN()), exp: order.UnknownStatus},
		{name: "coin-M Unknown fill of an executed quantity that is not a number", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: 10, executed: types.Number(math.NaN()), exp: order.UnknownStatus},
		{name: "coin-M Unknown new event at the whole quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeNew, status: "Unknown", quantity: 10, executed: 10, exp: order.UnknownStatus},
		{name: "coin-M Unknown cancellation at the whole quantity", a: asset.CoinMarginedFutures, execution: ExecutionTypeCancelled, status: "Unknown", quantity: 10, executed: 10, exp: order.UnknownStatus},
		{name: "coin-M status that cannot be converted", a: asset.CoinMarginedFutures, execution: ExecutionTypeTrade, status: "Waiting", quantity: 10, executed: 10, exp: order.UnknownStatus, err: order.ErrUnsupportedStatusType},
		{name: "USDT-M Unknown fill of the whole quantity", a: asset.USDTMarginedFutures, execution: ExecutionTypeTrade, status: "Unknown", quantity: 10, executed: 10, exp: order.UnknownStatus},
		{name: "USDT-M FILLED", a: asset.USDTMarginedFutures, execution: ExecutionTypeTrade, status: "FILLED", quantity: 10, executed: 10, exp: order.Filled},
		{name: "USDT-M Pending", a: asset.USDTMarginedFutures, execution: ExecutionTypeNew, status: "Pending", quantity: 10, executed: 0, exp: order.UnknownStatus, err: order.ErrUnsupportedStatusType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status, err := wsPerpetualOrderStatus(tc.a, &WebsocketPerpetualOrder{ExecutionType: tc.execution, OrderStatus: tc.status, Quantity: tc.quantity, CumulativeFilledQuantity: tc.executed})
			require.ErrorIs(t, err, tc.err, "wsPerpetualOrderStatus must return the expected error")
			assert.Equal(t, tc.exp, status, "wsPerpetualOrderStatus should convert the event's status")
		})
	}
}

// perpetualFeeStep is a REST read of a perpetual order or an order event in a fee total test
type perpetualFeeStep struct {
	// read is list, get or connect for a read through GetActiveOrders, GetOrderInfo or the account connection's read of
	// its open orders, and empty for an order event
	read string
	// executionType is an order event's x, TRADE when empty
	executionType string
	// fee is the commission a read reports, or the fee of an event's fill
	executed, fee, status string
	// relayed is the fee an event should carry
	relayed float64
}

// testPerpetualOrderServer answers a perpetual product's open order listing and order query with one order, recording
// the API key and route of each request, and holds a reply when asked. The query reports the order under the ID it is
// asked for, or the filled order set for that ID, and leaves the IDs set to hang unanswered; the listing leaves the
// order out while it is unlisted. A query reply set answers every query in its place
type testPerpetualOrderServer struct {
	a                         asset.Item
	pair                      currency.Pair
	symbol, listPath, getPath string
	quantity                  bool

	mu        sync.Mutex
	order     string
	unlisted  bool
	getReply  string
	getOrders map[string]string
	hang      map[string]bool
	keys      []string
	routes    []string
	arrived   chan struct{}
	release   chan struct{}
}

// newTestPerpetualOrderServer serves asset a's order endpoints to ex from an in-memory server, which a test can also run
// in a synctest bubble. quantity sends a coin-M order's size as its quantity rather than its origQty
func newTestPerpetualOrderServer(tb testing.TB, ex *Exchange, a asset.Item, quantity bool) *testPerpetualOrderServer {
	tb.Helper()
	s := &testPerpetualOrderServer{a: a, pair: currency.NewBTCUSDT(), symbol: "BTC-USDT", listPath: "/openApi/swap/v2/trade/openOrders", getPath: "/openApi/swap/v2/trade/order", quantity: quantity}
	if a == asset.CoinMarginedFutures {
		s.pair, s.symbol, s.listPath, s.getPath = currency.NewBTCUSD(), "BTC-USD", "/openApi/cswap/v1/trade/openOrders", "/openApi/cswap/v1/trade/orderDetail"
	}
	server := httptest.NewTestServer(tb, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := r.URL.Path
		if route == s.getPath {
			route += "?orderId=" + r.URL.Query().Get("orderId")
		}
		s.mu.Lock()
		s.keys = append(s.keys, r.Header.Get(keyHeader))
		s.routes = append(s.routes, route)
		reply, unlisted, getReply, arrived, release := s.order, s.unlisted, s.getReply, s.arrived, s.release
		orderID := r.URL.Query().Get("orderId")
		getOrder, hang := s.getOrders[orderID], route != s.listPath && s.hang[orderID]
		s.arrived, s.release = nil, nil
		s.mu.Unlock()
		if hang {
			<-r.Context().Done()
			return
		}
		if arrived != nil {
			close(arrived)
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		switch r.URL.Path {
		case s.listPath:
			if unlisted {
				reply = ""
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"orders":[`+reply+`]}}`)
		case s.getPath:
			if getReply != "" {
				_, _ = io.WriteString(w, getReply)
				return
			}
			if getOrder != "" {
				reply = getOrder
			} else if id, err := strconv.ParseUint(orderID, 10, 64); err == nil {
				reply = strings.Replace(reply, `"orderId":42`, `"orderId":`+strconv.FormatUint(id, 10), 1)
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"order":`+reply+`}}`)
		default:
			assert.Failf(tb, "unexpected request", "%s should not be requested", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	require.NoError(tb, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	testRESTEndpoints(tb, ex, server.URL)
	require.NoError(tb, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	return s
}

// setUnlisted makes the listing leave the order out, as BingX does once the order has closed
func (s *testPerpetualOrderServer) setUnlisted(unlisted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unlisted = unlisted
}

// setFilledOrder makes the query for an order ID report that order filled, with its commission
func (s *testPerpetualOrderServer) setFilledOrder(orderID uint64, commission string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.getOrders == nil {
		s.getOrders = make(map[string]string)
	}
	s.getOrders[strconv.FormatUint(orderID, 10)] = fmt.Sprintf(`{"symbol":%q,"orderId":%d,"side":"BUY","type":"LIMIT","origQty":"3","price":"10","executedQty":"3","avgPrice":"10","commission":%q,"status":"FILLED"}`, s.symbol, orderID, commission)
}

// hangGet leaves the queries for an order ID unanswered until they are given up
func (s *testPerpetualOrderServer) hangGet(orderID uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.hang == nil {
		s.hang = make(map[string]bool)
	}
	s.hang[strconv.FormatUint(orderID, 10)] = true
}

// setGetReply makes the query answer with reply instead of the order, or with the order again when reply is empty
func (s *testPerpetualOrderServer) setGetReply(reply string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getReply = reply
}

// requestRoutes returns the route of each request the server received, a query's with the order ID it asked for
func (s *testPerpetualOrderServer) requestRoutes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.routes)
}

// setOrder sets the order the server reports, of three contracts or coins, with its executed quantity, commission and
// status
func (s *testPerpetualOrderServer) setOrder(executed, commission, status string) {
	size := `"origQty":"3"`
	if s.quantity {
		size = `"quantity":3`
	}
	s.mu.Lock()
	s.order = fmt.Sprintf(`{"symbol":%q,"orderId":42,"side":"BUY","type":"LIMIT",%s,"price":"10","executedQty":%q,"avgPrice":"10","commission":%q,"status":%q}`, s.symbol, size, executed, commission, status)
	s.mu.Unlock()
}

// holdNext holds the next request's reply until release is closed, closing arrived once the request arrives
func (s *testPerpetualOrderServer) holdNext() (arrived, release chan struct{}) {
	arrived, release = make(chan struct{}), make(chan struct{})
	s.mu.Lock()
	s.arrived, s.release = arrived, release
	s.mu.Unlock()
	return arrived, release
}

// requestKeys returns the API key of each request the server received
func (s *testPerpetualOrderServer) requestKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.keys)
}

// read reads the order through GetActiveOrders, list, or GetOrderInfo, returning the order read
func (s *testPerpetualOrderServer) read(ctx context.Context, tb testing.TB, ex *Exchange, kind string) *order.Detail {
	tb.Helper()
	if kind == "list" {
		open, err := ex.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: s.a, Pairs: currency.Pairs{s.pair}, Side: order.AnySide, Type: order.AnyType})
		require.NoError(tb, err, "GetActiveOrders must not error")
		require.Len(tb, open, 1, "GetActiveOrders must list the order")
		return &open[0]
	}
	d, err := ex.GetOrderInfo(ctx, "42", s.pair, s.a)
	require.NoError(tb, err, "GetOrderInfo must not error")
	return d
}

// readError reads the order through GetActiveOrders or GetOrderInfo, returning the read's error
func (s *testPerpetualOrderServer) readError(ctx context.Context, ex *Exchange, kind string) error {
	if kind == "list" {
		_, err := ex.GetActiveOrders(ctx, &order.MultiOrderRequest{AssetType: s.a, Pairs: currency.Pairs{s.pair}, Side: order.AnySide, Type: order.AnyType})
		return err
	}
	_, err := ex.GetOrderInfo(ctx, "42", s.pair, s.a)
	return err
}

// step runs a fee total test step, returning the order a read or event reports, or nil for the account connection's
// read of its open orders
func (s *testPerpetualOrderServer) step(tb testing.TB, ex *Exchange, conn *testWebsocketConnection, step *perpetualFeeStep) *order.Detail {
	tb.Helper()
	switch step.read {
	case "":
	case "connect":
		state, err := ex.wsConnection(conn)
		require.NoError(tb, err, "the connection must be tracked")
		ex.wsReadOpenOrderFees(tb.Context(), state)
		testWaitBackground(tb, ex)
		return nil
	default:
		return s.read(tb.Context(), tb, ex, step.read)
	}
	event := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":%q,"i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":%q,"X":%q,"N":"USDT","n":%q,"T":1758608899001}}`, s.symbol, step.executed, cmp.Or(step.executionType, "TRADE"), step.status, step.fee)
	state, err := ex.wsConnection(conn)
	require.NoError(tb, err, "the connection must be tracked")
	require.NoError(tb, ex.wsProcessPerpetualOrderUpdate(tb.Context(), state, s.a, event), "wsProcessPerpetualOrderUpdate must not error")
	d := waitForWebsocketPayload[*order.Detail](tb, ex, nil)
	assert.InDelta(tb, step.relayed, d.Fee, 1e-12, "the event should carry the fees of the order's fills so far")
	return d
}

// testCredentialsChangingContext changes the exchange's credentials once, when a credentials lookup through it asks for
// the sub-account override, which the lookup reads after the exchange's credentials
type testCredentialsChangingContext struct {
	context.Context //nolint:containedctx // the type is a context wrapping another, to act on the lookups made through it
	once            sync.Once
	change          func()
}

func (c *testCredentialsChangingContext) Value(key any) any {
	if key == accounts.ContextSubAccountFlag {
		c.once.Do(c.change)
	}
	return c.Context.Value(key)
}

func TestPerpetualOrderFeeTotalsAdd(t *testing.T) { //nolint:tparallel // each step adds to the totals the steps before it left
	t.Parallel()
	var totals perpetualOrderFeeTotals
	for i, step := range []struct {
		name, orderID, executed, fee, want string
	}{
		{"nothing executed", "1", "0", "0.5", "0"},
		{"first fill", "1", "1", "0.1", "0.1"},
		{"repeated event", "1", "1", "0.1", "0.1"},
		{"another order", "2", "1", "0.01", "0.01"},
		{"less executed than before", "1", "0.5", "0.3", "0.1"},
		{"rebate", "1", "2", "-0.05", "0.05"},
		{"later fill", "1", "3", "0.2", "0.25"},
	} {
		t.Run(strconv.Itoa(i)+" "+step.name, func(t *testing.T) {
			got := totals.add(step.orderID, decimal.MustFromString(step.executed), decimal.MustFromString(step.fee))
			assert.Truef(t, decimal.MustFromString(step.want).Equal(got), "add should return %s, not %s", step.want, got)
			if i == 0 {
				assert.Empty(t, totals.orders, "add should start no total for an event with nothing executed")
			}
		})
	}
	totals.mu.Lock()
	defer totals.mu.Unlock()
	require.Contains(t, totals.orders, "1", "add must start a total at an order's first fill")
	assert.False(t, totals.orders["1"].complete, "a total started at a fill should not count the fills before it")
	assert.Len(t, totals.orders["1"].fills, 3, "a total missing its first fills should keep each fill it counted")
}

func TestPerpetualOrderFeeTotalsStart(t *testing.T) {
	t.Parallel()
	var totals perpetualOrderFeeTotals
	totals.start("1")
	totals.add("1", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
	totals.start("1")
	totals.add("2", decimal.NewFromInt(1), decimal.MustFromString("0.2"))
	totals.start("2")
	totals.mu.Lock()
	defer totals.mu.Unlock()
	require.Contains(t, totals.orders, "1", "start must give an order a total")
	assert.True(t, totals.orders["1"].complete, "an order started at its NEW event should have every fill counted")
	assert.Empty(t, totals.orders["1"].fills, "a total counting every fill should keep no fills")
	assert.True(t, decimal.MustFromString("0.1").Equal(totals.orders["1"].fee), "a later NEW event should leave the order's total alone")
	require.Contains(t, totals.orders, "2", "add must give an order a total")
	assert.False(t, totals.orders["2"].complete, "a NEW event after a fill should not mark the order's fills counted")
}

func TestPerpetualOrderFeeTotalsCounted(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		start   func(*perpetualOrderFeeTotals)
		counted bool
	}{
		{"no total", func(*perpetualOrderFeeTotals) {}, false},
		{"a total started from a fill", func(totals *perpetualOrderFeeTotals) {
			totals.add("42", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
		}, false},
		{"a total a read started", func(totals *perpetualOrderFeeTotals) {
			totals.seed("42", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
		}, true},
		{"a total a read completed", func(totals *perpetualOrderFeeTotals) {
			totals.add("42", decimal.NewFromInt(2), decimal.MustFromString("0.2"))
			totals.seed("42", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
		}, true},
		{"a total started from the order's NEW event", func(totals *perpetualOrderFeeTotals) { totals.start("42") }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var totals perpetualOrderFeeTotals
			tc.start(&totals)
			assert.Equal(t, tc.counted, totals.counted("42"), "counted should report whether the order's total counts every fill")
		})
	}
}

func TestPerpetualOrderFeeTotalsSeed(t *testing.T) {
	t.Parallel()
	type fill struct{ executed, fee string }
	for _, tc := range []struct {
		name string
		// started marks the order's NEW event before its fills
		started bool
		fills   []fill
		// reading begins a read and closes the order before the seed
		closed                bool
		executed, fee         string
		wantExecuted, wantFee string
		wantFills             int
	}{
		{name: "no total", executed: "2", fee: "0.23", wantExecuted: "2", wantFee: "0.23"},
		{name: "nothing executed", executed: "0", fee: "0"},
		{name: "a rebate", executed: "1", fee: "-0.05", wantExecuted: "1", wantFee: "-0.05"},
		{name: "a total counting every fill, behind the read", started: true, fills: []fill{{"1", "0.1"}}, executed: "2", fee: "0.23", wantExecuted: "2", wantFee: "0.23"},
		{name: "a total counting every fill, as far as the read", started: true, fills: []fill{{"1", "0.1"}, {"2", "0.2"}}, executed: "2", fee: "0.25", wantExecuted: "2", wantFee: "0.25"},
		{name: "a total counting every fill, beyond the read", started: true, fills: []fill{{"1", "0.1"}, {"2", "0.2"}, {"3", "0.3"}}, executed: "2", fee: "0.23", wantExecuted: "3", wantFee: "0.6"},
		{name: "a total missing its first fills, behind the read", fills: []fill{{"1", "0.1"}}, executed: "2", fee: "0.23", wantExecuted: "2", wantFee: "0.23"},
		{name: "a total missing its first fills, as far as the read", fills: []fill{{"2", "0.2"}}, executed: "2", fee: "0.3", wantExecuted: "2", wantFee: "0.3"},
		{name: "a total missing its first fills, beyond the read", fills: []fill{{"2", "0.2"}, {"3", "0.3"}}, executed: "1", fee: "0.1", wantExecuted: "3", wantFee: "0.6"},
		{name: "an order closed during the read", fills: []fill{{"1", "0.1"}}, closed: true, executed: "2", fee: "0.23"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var totals perpetualOrderFeeTotals
			totals.add("other", decimal.NewFromInt(5), decimal.MustFromString("0.5"))
			if tc.started {
				totals.start("1")
			}
			for _, f := range tc.fills {
				totals.add("1", decimal.MustFromString(f.executed), decimal.MustFromString(f.fee))
			}
			if tc.closed {
				totals.beginRead()
				totals.forget("1")
			}
			totals.seed("1", decimal.MustFromString(tc.executed), decimal.MustFromString(tc.fee))
			totals.mu.Lock()
			defer totals.mu.Unlock()
			got, ok := totals.orders["1"]
			if tc.wantExecuted == "" {
				assert.False(t, ok, "seed should start no total")
			} else {
				require.True(t, ok, "seed must leave the order a total")
				assert.Truef(t, decimal.MustFromString(tc.wantExecuted).Equal(got.executed), "the total should count %s executed, not %s", tc.wantExecuted, got.executed)
				assert.Truef(t, decimal.MustFromString(tc.wantFee).Equal(got.fee), "the total should hold a fee of %s, not %s", tc.wantFee, got.fee)
				assert.True(t, got.complete, "a total a read started or completed should count every fill")
				assert.Empty(t, got.fills, "a total counting every fill should keep no fills")
			}
			assert.True(t, decimal.MustFromString("0.5").Equal(totals.orders["other"].fee), "seed should leave another order's total alone")
		})
	}
}

func TestPerpetualOrderFeeTotalsForget(t *testing.T) {
	t.Parallel()
	var totals perpetualOrderFeeTotals
	totals.forget("1")
	assert.Empty(t, totals.orders, "forget should leave totals that hold nothing empty")
	totals.add("1", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
	totals.add("2", decimal.NewFromInt(1), decimal.MustFromString("0.2"))
	totals.forget("1")
	assert.Equal(t, []string{"2"}, slices.Collect(maps.Keys(totals.orders)), "forget should drop the order's total alone")
	assert.Empty(t, totals.closed, "a close with no read in flight should not be noted")
	got := totals.add("1", decimal.NewFromInt(1), decimal.MustFromString("0.3"))
	assert.Truef(t, decimal.MustFromString("0.3").Equal(got), "add should start a forgotten order's total afresh, not return %s", got)
}

func TestPerpetualOrderFeeTotalsTotal(t *testing.T) {
	t.Parallel()
	var totals perpetualOrderFeeTotals
	total := totals.total("1")
	require.NotNil(t, total, "total must start a total for an order without one")
	total.fee = decimal.MustFromString("0.1")
	assert.Same(t, total, totals.total("1"), "total should return the order's total once it has one")
	assert.NotSame(t, total, totals.total("2"), "total should start another order's total apart")
}

func TestPerpetualOrderFeeTotalsBeginRead(t *testing.T) {
	t.Parallel()
	var totals perpetualOrderFeeTotals
	first, second := totals.beginRead(), totals.beginRead()
	assert.Less(t, first, second, "each read should be numbered after the one before it")
	assert.Equal(t, map[uint64]struct{}{first: {}, second: {}}, totals.reads, "each read should be recorded as in flight")
	totals.forget("1")
	third := totals.beginRead()
	assert.Less(t, totals.closed["1"], third, "a read begun after a close should be numbered after it")
}

func TestPerpetualOrderFeeTotalsEndRead(t *testing.T) {
	t.Parallel()
	var totals perpetualOrderFeeTotals
	first := totals.beginRead()
	totals.forget("1")
	second := totals.beginRead()
	totals.forget("2")
	totals.seed("1", decimal.NewFromInt(2), decimal.MustFromString("0.3"))
	totals.seed("2", decimal.NewFromInt(2), decimal.MustFromString("0.3"))
	assert.Empty(t, totals.orders, "a read should start no total for an order closed while a read was in flight")
	totals.endRead(first)
	assert.NotContains(t, totals.closed, "1", "a close before every read still in flight began should be dropped")
	assert.Contains(t, totals.closed, "2", "a close after a read still in flight began should be kept")
	assert.Equal(t, map[uint64]struct{}{second: {}}, totals.reads, "the read should no longer be in flight")
	totals.endRead(second)
	assert.Empty(t, totals.closed, "every close should be dropped once no read is in flight")
	assert.Empty(t, totals.reads, "every read should have ended")
}

func TestBeginPerpetualOrderFeeRead(t *testing.T) {
	t.Parallel()
	t.Run("without credentials", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		ctx, read := ex.beginPerpetualOrderFeeRead(t.Context(), asset.USDTMarginedFutures)
		assert.Equal(t, t.Context(), ctx, "the context should be returned unchanged")
		assert.Nil(t, read, "a read without credentials should start no totals")
	})

	t.Run("with credentials", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		session := newTestSession(t, ex).ch
		streaming := make([]*websocketConnection, 0, 2)
		for range 2 {
			streaming = append(streaming, ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.USDTMarginedFutures, creds))
		}
		down := &testWebsocketConnection{}
		down.down.Store(true)
		ex.wsTrackConnection(down, session, false, asset.USDTMarginedFutures, creds)
		ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.CoinMarginedFutures, creds)
		ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.USDTMarginedFutures, &accounts.Credentials{Key: "other-key", Secret: "other-secret"})
		ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.Empty, nil)
		// A connection holding order events is no recipient of a read that is not its own
		holdingDown := &testWebsocketConnection{}
		holdingDown.down.Store(true)
		ex.wsState.orderRelay.hold(ex.wsTrackConnection(holdingDown, session, false, asset.USDTMarginedFutures, creds), "42")
		ctx, read := ex.beginPerpetualOrderFeeRead(t.Context(), asset.USDTMarginedFutures)
		ex.SetCredentials(&accounts.Credentials{Key: "changed-key", Secret: "changed-secret"})
		got, err := ex.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		assert.Equal(t, creds.Key, got.Key, "the context should carry the read's credentials")
		require.NotNil(t, read, "a read with credentials must be returned")
		assert.ElementsMatch(t, []*perpetualOrderFeeTotals{&streaming[0].fees, &streaming[1].fees}, read.totals, "the read should start the totals of each connection up for its account and product")
		for i, state := range streaming {
			assert.Containsf(t, state.fees.reads, read.reads[slices.Index(read.totals, &state.fees)], "connection %d should have the read in flight", i)
		}
	})

	t.Run("the connection whose own read it is", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		session := newTestSession(t, ex).ch
		up := ex.wsTrackConnection(&testWebsocketConnection{}, session, false, asset.USDTMarginedFutures, creds)
		downConn := &testWebsocketConnection{}
		downConn.down.Store(true)
		down := ex.wsTrackConnection(downConn, session, false, asset.USDTMarginedFutures, creds)
		// The cases read through one exchange's connections, so they run in turn
		for _, tc := range []struct {
			name  string
			owner *websocketConnection
			exp   []*perpetualOrderFeeTotals
		}{
			{"a connection no longer up", down, []*perpetualOrderFeeTotals{&up.fees, &down.fees}},
			{"a connection up", up, []*perpetualOrderFeeTotals{&up.fees}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, read := ex.beginPerpetualOrderFeeRead(context.WithValue(t.Context(), perpetualOrderFeeReadOwnerKey{}, tc.owner), asset.USDTMarginedFutures)
				require.NotNil(t, read, "a read with credentials must be returned")
				defer read.end()
				assert.ElementsMatch(t, tc.exp, read.totals, "the read should start the totals of the connection whose own read it is, once")
			})
		}
	})
}

func TestPerpetualOrderFeeReadSeed(t *testing.T) {
	t.Parallel()
	var read *perpetualOrderFeeRead
	read.seed(&order.Detail{OrderID: "42", Amount: 3, ExecutedAmount: 2, Status: order.PartiallyFilled}, 2, -0.23)
	var first, second perpetualOrderFeeTotals
	read = &perpetualOrderFeeRead{totals: []*perpetualOrderFeeTotals{&first, &second}, reads: []uint64{first.beginRead(), second.beginRead()}}
	for _, d := range []*order.Detail{
		{OrderID: "1", Amount: 3, ExecutedAmount: 3, Status: order.Filled},
		{OrderID: "2", Amount: 3, Status: order.New},
		{OrderID: "3", Amount: 3, ExecutedAmount: 2, Status: order.PartiallyFilled},
		{OrderID: "4", Amount: 3, ExecutedAmount: 1, Status: order.PartiallyFilled},
	} {
		commission := types.Number(-0.23)
		if d.OrderID == "4" {
			commission = 0.05
		}
		read.seed(d, types.Number(d.ExecutedAmount), commission)
	}
	for _, tc := range []struct {
		name   string
		totals *perpetualOrderFeeTotals
	}{{"first connection", &first}, {"second connection", &second}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, []string{"3", "4"}, slices.Collect(maps.Keys(tc.totals.orders)), "the read should start a total for each open order with fills alone")
			if total, ok := tc.totals.orders["3"]; assert.True(t, ok, "the read should start the open order's total") {
				assert.Truef(t, decimal.MustFromString("0.23").Equal(total.fee), "the total should hold the commission negated, not %s", total.fee)
			}
			if total, ok := tc.totals.orders["4"]; assert.True(t, ok, "the read should start the rebated order's total") {
				assert.Truef(t, decimal.MustFromString("-0.05").Equal(total.fee), "the total should hold a rebate as a negative fee, not %s", total.fee)
			}
		})
	}
}

func TestPerpetualOrderFeeReadEnd(t *testing.T) {
	t.Parallel()
	var read *perpetualOrderFeeRead
	read.end()
	var first, second perpetualOrderFeeTotals
	read = &perpetualOrderFeeRead{totals: []*perpetualOrderFeeTotals{&first, &second}, reads: []uint64{first.beginRead(), second.beginRead()}}
	other := second.beginRead()
	first.forget("1")
	second.forget("1")
	read.end()
	assert.Empty(t, first.reads, "end should end the read on every total it began on")
	assert.Empty(t, first.closed, "end should drop the closes no read in flight began before")
	assert.Equal(t, map[uint64]struct{}{other: {}}, second.reads, "end should leave another read in flight")
	assert.Contains(t, second.closed, "1", "a close another read in flight began before should be kept")
}

func TestWebsocketReadOpenOrderFees(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a     asset.Item
		event string
	}{
		{asset.USDTMarginedFutures, "ORDER_TRADE_UPDATE"},
		{asset.USDTMarginedFutures, "TRADE_UPDATE"},
		{asset.CoinMarginedFutures, "ORDER_TRADE_UPDATE"},
		{asset.CoinMarginedFutures, "TRADE_UPDATE"},
	} {
		a := tc.a
		t.Run(a.String()+" "+tc.event, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
			server := newTestPerpetualOrderServer(t, ex, a, false)
			creds, err := ex.GetCredentials(t.Context())
			require.NoError(t, err, "GetCredentials must not error")
			conn := &testWebsocketConnection{}
			state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, a, creds)
			server.setOrder("2", "-0.23", "PARTIALLY_FILLED")
			arrived, release := server.holdNext()
			var releaseOnce sync.Once
			t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
			waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
			fill := fmt.Appendf(nil, `{"e":%q,"E":1758608899001,"o":{"s":%q,"i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, tc.event, server.symbol)
			require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, fill, a), "an order event must be handled while the read is in flight")
			require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, []byte(websocketSwapPing), a), "a heartbeat must be handled while the read is in flight")
			replies := conn.messages()
			require.Len(t, replies, 1, "a heartbeat must be answered while the read is in flight")
			assert.Equal(t, websocketSwapPong, replies[0].message, "the heartbeat should be answered with a pong")
			assert.Empty(t, websocketPayloads(ex), "the order event should be held while the read is in flight")
			releaseOnce.Do(func() { close(release) })
			testWaitBackground(t, ex)
			assert.InDelta(t, 0.33, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, 1e-12, "the held event should carry the fees the read found and its own fill's")
			assert.Equal(t, []string{creds.Key}, server.requestKeys(), "the open orders should be read once, with the context's credentials")
			state.fees.mu.Lock()
			assert.Empty(t, state.fees.orders, "the order should be forgotten once its held closing event is handled")
			state.fees.mu.Unlock()
		})
	}

	t.Run("an order the read does not list", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			a        asset.Item
			listed   bool
			order    [3]string // executed, commission and status the order query reports
			getReply string
			held     [][4]string // execution type, status, executed and fee of each held event
			fees     []float64
			lookedUp bool
		}{
			{"USDT-M closed by its last fill", asset.USDTMarginedFutures, false, [3]string{"3", "-0.33", "FILLED"}, "", [][4]string{{"TRADE", "FILLED", "3", "-0.1"}}, []float64{0.33}, true},
			{"USDT-M closed by its last two fills", asset.USDTMarginedFutures, false, [3]string{"3", "-0.6", "FILLED"}, "", [][4]string{{"TRADE", "PARTIALLY_FILLED", "2", "-0.2"}, {"TRADE", "FILLED", "3", "-0.3"}}, []float64{0.6, 0.6}, true},
			{"coin-M closed by its last fill", asset.CoinMarginedFutures, false, [3]string{"3", "-0.0033", "FILLED"}, "", [][4]string{{"TRADE", "FILLED", "3", "-0.001"}}, []float64{0.0033}, true},
			{"looked up before its held fill", asset.USDTMarginedFutures, false, [3]string{"2", "-0.23", "PARTIALLY_FILLED"}, "", [][4]string{{"TRADE", "FILLED", "3", "-0.1"}}, []float64{0.33}, true},
			{"amended before its held fill", asset.USDTMarginedFutures, false, [3]string{"2", "-0.3", "PARTIALLY_FILLED"}, "", [][4]string{{"NEW", "PARTIALLY_FILLED", "1", "0"}, {"TRADE", "PARTIALLY_FILLED", "2", "-0.2"}}, []float64{0, 0.3}, true},
			{"a lookup failing", asset.USDTMarginedFutures, false, [3]string{"3", "-0.33", "FILLED"}, `{"code":100500,"msg":"internal error"}`, [][4]string{{"TRADE", "FILLED", "3", "-0.1"}}, []float64{0.1}, true},
			{"placed during the read", asset.USDTMarginedFutures, false, [3]string{"1", "-0.1", "PARTIALLY_FILLED"}, "", [][4]string{{"NEW", "NEW", "0", "0"}, {"TRADE", "PARTIALLY_FILLED", "1", "-0.1"}}, []float64{0, 0.1}, false},
			{"listed with nothing executed", asset.USDTMarginedFutures, true, [3]string{"0", "0", "NEW"}, "", [][4]string{{"TRADE", "PARTIALLY_FILLED", "1", "-0.1"}}, []float64{0.1}, false},
			{"cancelled without a held fill", asset.USDTMarginedFutures, false, [3]string{"2", "-0.23", "CANCELED"}, "", [][4]string{{"CANCELED", "CANCELED", "2", "0"}}, []float64{0}, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestPerpetualOrderServer(t, ex, tc.a, false)
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				conn := &testWebsocketConnection{}
				state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, tc.a, creds)
				server.setOrder(tc.order[0], tc.order[1], tc.order[2])
				server.setUnlisted(!tc.listed)
				server.setGetReply(tc.getReply)
				arrived, release := server.holdNext()
				var releaseOnce sync.Once
				t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
				waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
				for _, h := range tc.held {
					event := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":%q,"i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":%q,"X":%q,"N":"USDT","n":%q,"T":1758608899001}}`, server.symbol, h[2], h[0], h[1], h[3])
					require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, event, tc.a), "an order event must be handled while the read is in flight")
				}
				assert.Empty(t, websocketPayloads(ex), "the order events should be held while the read is in flight")
				releaseOnce.Do(func() { close(release) })
				testWaitBackground(t, ex)
				fees := make([]float64, 0, len(tc.held))
				for range tc.held {
					fees = append(fees, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee)
				}
				assert.InDeltaSlice(t, tc.fees, fees, 1e-12, "the held events should carry the order's fees")
				routes := []string{server.listPath}
				if tc.lookedUp {
					routes = append(routes, server.getPath+"?orderId=42")
				}
				assert.Equal(t, routes, server.requestRoutes(), "an order the read does not list should be looked up once, when a held fill would start its total without its earlier fees")
				assert.Equal(t, slices.Repeat([]string{creds.Key}, len(routes)), server.requestKeys(), "the read and the lookup should be sent for the connection's account")
			})
		}
	})

	t.Run("a fill of another order held during a lookup", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := newTestPerpetualOrderServer(t, ex, asset.USDTMarginedFutures, false)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
		server.setOrder("3", "-0.33", "FILLED")
		server.setUnlisted(true)
		closingFill := func(orderID int) []byte {
			return fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":%d,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, orderID)
		}
		readArrived, readRelease := server.holdNext()
		var readOnce, lookupOnce sync.Once
		t.Cleanup(func() { readOnce.Do(func() { close(readRelease) }) })
		waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, readArrived)
		require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, closingFill(42), asset.USDTMarginedFutures), "the first order's fill must be handled while the read is in flight")
		lookupArrived, lookupRelease := server.holdNext()
		t.Cleanup(func() { lookupOnce.Do(func() { close(lookupRelease) }) })
		readOnce.Do(func() { close(readRelease) })
		select {
		case <-lookupArrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first order must be looked up")
		}
		require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, closingFill(43), asset.USDTMarginedFutures), "the second order's fill must be handled while the first is looked up")
		lookupOnce.Do(func() { close(lookupRelease) })
		testWaitBackground(t, ex)
		exp := []any{testRelayedOrder{"42", 3, order.Filled, 0.33}, testRelayedOrder{"43", 3, order.Filled, 0.33}}
		assert.Equal(t, exp, testRelayedOrders(ex), "each held fill should carry its order's looked up fees")
		assert.Equal(t, []string{server.listPath, server.getPath + "?orderId=42", server.getPath + "?orderId=43"}, server.requestRoutes(), "an order whose fill is held during another's lookup should be looked up before the events are handled")
	})

	t.Run("two orders the read does not list", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := newTestPerpetualOrderServer(t, ex, asset.USDTMarginedFutures, false)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
		server.setUnlisted(true)
		server.setFilledOrder(42, "-0.33")
		server.setFilledOrder(43, "-0.87")
		arrived, release := server.holdNext()
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
		for _, orderID := range []int{42, 43} {
			fill := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":%d,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, orderID)
			require.NoErrorf(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, fill, asset.USDTMarginedFutures), "order %d's fill must be handled while the read is in flight", orderID)
		}
		releaseOnce.Do(func() { close(release) })
		testWaitBackground(t, ex)
		exp := []any{testRelayedOrder{"42", 3, order.Filled, 0.33}, testRelayedOrder{"43", 3, order.Filled, 0.87}}
		assert.Equal(t, exp, testRelayedOrders(ex), "each held fill should carry its own order's looked up fees")
		routes := server.requestRoutes()
		slices.Sort(routes)
		assert.Equal(t, []string{server.listPath, server.getPath + "?orderId=42", server.getPath + "?orderId=43"}, routes, "each order should be looked up once")
	})

	t.Run("a read or lookup left unanswered", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			readTime time.Duration // how long the read takes to be answered, the listing leaving the orders out
			hang     []uint64      // the orders whose lookups are left unanswered
			relayed  []testRelayedOrder
		}{
			{"the read", -1, nil, []testRelayedOrder{{"42", 3, order.Filled, 0.1}}},
			{"a lookup", 0, []uint64{42}, []testRelayedOrder{{"42", 3, order.Filled, 0.1}}},
			{"a lookup after a slow read", 4 * time.Second, []uint64{42}, []testRelayedOrder{{"42", 3, order.Filled, 0.1}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				synctest.Test(t, func(t *testing.T) {
					ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
					server := newTestPerpetualOrderServer(t, ex, asset.USDTMarginedFutures, false)
					creds, err := ex.GetCredentials(t.Context())
					require.NoError(t, err, "GetCredentials must not error")
					conn := &testWebsocketConnection{}
					state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
					server.setOrder("2", "-0.23", "PARTIALLY_FILLED")
					server.setUnlisted(true)
					server.setFilledOrder(43, "-0.87")
					for _, orderID := range tc.hang {
						server.hangGet(orderID)
					}
					arrived, release := server.holdNext()
					var releaseOnce sync.Once
					t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
					start := time.Now()
					waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
					exp := make([]any, 0, len(tc.relayed))
					for _, r := range tc.relayed {
						exp = append(exp, r)
						fill := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":%s,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, r.orderID)
						require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, fill, asset.USDTMarginedFutures), "an order event must be handled while the read is in flight")
					}
					if tc.readTime >= 0 {
						time.Sleep(tc.readTime)
						releaseOnce.Do(func() { close(release) })
					}
					testWaitBackground(t, ex)
					assert.Equal(t, 5*time.Second, time.Since(start), "the held events should be handled five seconds after the read began")
					assert.Equal(t, exp, testRelayedOrders(ex), "the held fills should be relayed with the fees their lookups found, or their connection counted")
					routes := []string{server.listPath}
					if tc.readTime >= 0 {
						for _, r := range tc.relayed {
							routes = append(routes, server.getPath+"?orderId="+r.orderID)
						}
					}
					got := server.requestRoutes()
					slices.Sort(got)
					assert.Equal(t, routes, got, "nothing left unanswered should be sent again")
				})
			})
		}
	})

	// The requester's lock, which a request holds for up to 50 milliseconds, keeps a second lookup out of a synctest
	// bubble while the first is unanswered, so this case runs on the clock
	t.Run("one of three lookups left unanswered", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := newTestPerpetualOrderServer(t, ex, asset.USDTMarginedFutures, false)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
		server.setUnlisted(true)
		server.setFilledOrder(43, "-0.87")
		server.setFilledOrder(44, "-0.64")
		server.hangGet(42)
		arrived, release := server.holdNext()
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
		for _, orderID := range []int{42, 43, 44} {
			fill := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":%d,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, orderID)
			require.NoErrorf(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, fill, asset.USDTMarginedFutures), "order %d's fill must be handled while the read is in flight", orderID)
		}
		releaseOnce.Do(func() { close(release) })
		testWaitBackground(t, ex)
		exp := []any{testRelayedOrder{"42", 3, order.Filled, 0.1}, testRelayedOrder{"43", 3, order.Filled, 0.87}, testRelayedOrder{"44", 3, order.Filled, 0.64}}
		assert.Equal(t, exp, testRelayedOrders(ex), "an unanswered lookup should leave the others their time, and its own order the fees its connection counted")
		routes := server.requestRoutes()
		slices.Sort(routes)
		assert.Equal(t, []string{server.listPath, server.getPath + "?orderId=42", server.getPath + "?orderId=43", server.getPath + "?orderId=44"}, routes, "nothing left unanswered should be sent again")
	})

	t.Run("a read refused for its rate", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) {
			var requests atomic.Int32
			arrived, release := make(chan struct{}), make(chan struct{})
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					close(arrived)
					select {
					case <-release:
					case <-r.Context().Done():
					}
				}
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
			require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
			testRESTEndpoints(t, ex, server.URL)
			require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
			creds, err := ex.GetCredentials(t.Context())
			require.NoError(t, err, "GetCredentials must not error")
			conn := &testWebsocketConnection{}
			state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
			var releaseOnce sync.Once
			t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
			start := time.Now()
			waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
			fill := []byte(`{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`)
			require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), conn, fill, asset.USDTMarginedFutures), "an order event must be handled while the read is in flight")
			releaseOnce.Do(func() { close(release) })
			testWaitBackground(t, ex)
			assert.Zero(t, time.Since(start), "a refused read and lookup should end at once rather than wait to be sent again")
			assert.Equal(t, int32(2), requests.Load(), "the read and the held order's lookup should each be sent once")
			assert.Equal(t, []any{testRelayedOrder{"42", 3, order.Filled, 0.1}}, testRelayedOrders(ex), "the held fill should be relayed with the fees its connection counted")
		})
	})

	t.Run("read failing", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"code":100500,"msg":"internal error"}`)
		}))
		server.Start()
		require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
		testRESTEndpoints(t, ex, server.URL)
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
		ex.wsReadOpenOrderFees(t.Context(), state)
		testWaitBackground(t, ex)
		assert.Empty(t, state.fees.orders, "a failed read should start no totals")
		assert.Empty(t, state.fees.reads, "a failed read should end")
		assert.False(t, state.holding, "a failed read should stop holding order events")
	})

	t.Run("another connection of the account dialled during the read", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := newTestPerpetualOrderServer(t, ex, asset.USDTMarginedFutures, false)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		session := newTestSession(t, ex)
		holder := &testWebsocketConnection{}
		state := ex.wsTrackConnection(holder, session.ch, false, asset.USDTMarginedFutures, creds)
		update := func(execution, status, executed, fee string) []byte {
			return fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":%q,"X":%q,"N":"USDT","n":%q,"T":1758608899001}}`, executed, execution, status, fee)
		}
		require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), holder, update("NEW", "NEW", "0", "0"), asset.USDTMarginedFutures), "the order's NEW event must be handled")
		require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), holder, update("TRADE", "PARTIALLY_FILLED", "1", "-0.1"), asset.USDTMarginedFutures), "the order's first fill must be handled")
		assert.Len(t, websocketPayloads(ex), 2, "both events should be relayed before the read")
		server.setOrder("1", "-0.1", "PARTIALLY_FILLED")
		arrived, release := server.holdNext()
		var releaseOnce sync.Once
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
		// A connection dialled after the read began holds no total for the order
		ex.wsTrackConnection(&testWebsocketConnection{}, session.ch, false, asset.USDTMarginedFutures, creds)
		require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), holder, update("TRADE", "PARTIALLY_FILLED", "2", "-0.2"), asset.USDTMarginedFutures), "the second fill must be handled")
		releaseOnce.Do(func() { close(release) })
		testWaitBackground(t, ex)
		assert.InDelta(t, 0.3, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, 1e-12, "the held fill should add to the total of the connection that held it")
	})

	t.Run("a connection closed before its read begins", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestPerpetualOrderServer(t, ex, a, false)
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				conn := &testWebsocketConnection{}
				state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, a, creds)
				require.NoError(t, conn.Shutdown(), "Shutdown must not error")
				server.setOrder("1", "-0.1", "PARTIALLY_FILLED")
				arrived, release := server.holdNext()
				var releaseOnce sync.Once
				t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
				waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
				// Its reader hands over a frame it read before the connection closed
				fill := fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":%q,"i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"2","x":"TRADE","X":"PARTIALLY_FILLED","N":"USDT","n":"-0.2","T":1758608899001}}`, server.symbol)
				require.True(t, ex.wsHoldOrderEvent(conn, fill), "wsHoldOrderEvent must hold the event while the read is in flight")
				releaseOnce.Do(func() { close(release) })
				testWaitBackground(t, ex)
				assert.InDelta(t, 0.3, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, 1e-12, "the held fill should add to the fees the connection's own read found")
			})
		}
	})

	t.Run("session ending during the read", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		server := newTestPerpetualOrderServer(t, ex, asset.USDTMarginedFutures, false)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		session := newTestSession(t, ex)
		// The session counts as watched, so no watcher of its end joins the manager's wait, which then covers the read alone
		ex.wsState.mu.Lock()
		ex.wsState.connections = make(map[websocket.Connection]*websocketConnection)
		ex.wsState.sessions = map[<-chan struct{}]bool{session.ch: true}
		ex.wsState.mu.Unlock()
		t.Cleanup(func() {
			ex.wsState.mu.Lock()
			delete(ex.wsState.sessions, session.ch)
			ex.wsState.mu.Unlock()
		})
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, session.ch, false, asset.USDTMarginedFutures, creds)
		server.setOrder("2", "-0.23", "PARTIALLY_FILLED")
		arrived, release := server.holdNext()
		t.Cleanup(func() { close(release) })
		waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
		fill := []byte(`{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":"3","x":"TRADE","X":"FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`)
		require.True(t, ex.wsHoldOrderEvent(conn, fill), "wsHoldOrderEvent must hold the event while the read is in flight")
		// Holding the relay back stops the release inside it, holding the connection's held events, which shows the manager's
		// wait covers the release, not only the read
		relay := &ex.wsState.orderRelay
		relay.mu.Lock()
		session.end()
		releasing := assert.Eventually(t, func() bool {
			if state.heldOrders.TryLock() {
				state.heldOrders.Unlock()
				return false
			}
			return true
		}, 5*time.Second, time.Millisecond, "the session's end should end the read and start its release")
		if !releasing {
			relay.mu.Unlock()
			t.FailNow()
		}
		waited := make(chan struct{})
		go func() {
			ex.Websocket.Wg.Wait()
			close(waited)
		}()
		select {
		case <-waited:
			relay.mu.Unlock()
			require.FailNow(t, "the manager's wait must not end before the read's release")
		case <-time.After(500 * time.Millisecond):
		}
		relay.mu.Unlock()
		select {
		case <-waited:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the manager's wait must end once the read's release is done")
		}
		state.heldOrders.Lock()
		assert.False(t, state.holding, "the manager's wait for its session's work should cover the read's release")
		state.heldOrders.Unlock()
		assert.NotEmpty(t, websocketPayloads(ex), "the held event should be relayed before the manager's wait ends")
		testWaitBackground(t, ex)
		assert.Empty(t, state.fees.orders, "a read the session's end cut off should start no totals")
	})
}

// waitForTestRead starts an account connection's read of its open orders, failing the test unless start returns while
// the read's request is in flight and the request arrives
func waitForTestRead(tb testing.TB, start func(), arrived <-chan struct{}) {
	tb.Helper()
	returned := make(chan struct{})
	go func() {
		start()
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		require.FailNow(tb, "the read must not hold its caller up")
	}
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		require.FailNow(tb, "the open orders must be read")
	}
}

func TestWebsocketHoldOrderEvent(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	data := []byte(`{"o":{"i":42}}`)
	assert.False(t, ex.wsHoldOrderEvent(&testWebsocketConnection{}, data), "wsHoldOrderEvent should not hold an event of a connection not tracked")
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, false)
	assert.False(t, ex.wsHoldOrderEvent(conn, data), "wsHoldOrderEvent should not hold an event while no read is in flight")
	state.holding = true
	require.True(t, ex.wsHoldOrderEvent(conn, data), "wsHoldOrderEvent must hold an event while the read is in flight")
	require.True(t, ex.wsHoldOrderEvent(conn, []byte(`{"o":{"sg":1}}`)), "wsHoldOrderEvent must hold an event that does not decode")
	data[0] = 'x'
	assert.Equal(t, [][]byte{[]byte(`{"o":{"i":42}}`), []byte(`{"o":{"sg":1}}`)}, state.held, "wsHoldOrderEvent should keep a copy of each event")
	assert.Equal(t, map[*websocketConnection]map[perpetualOrderKey]struct{}{state: {{asset: state.account, orderID: "42"}: {}}}, ex.wsState.orderRelay.holders, "wsHoldOrderEvent should note the order of each event it holds that decodes")

	for _, tc := range []struct {
		name, event, orderID string
	}{
		{"an event with a trade and a client order ID", `{"e":"ORDER_TRADE_UPDATE","E":1758608899350,"o":{"s":"BTC-USDT","c":"desk-a","i":1970374651259388000,"S":"SELL","o":"MARKET","q":"3","z":"1","x":"TRADE","X":"PARTIALLY_FILLED","td":511178071}}`, "1970374651259388000"},
		{"an event named as the USDT-M docs example names it", `{"e":"TRADE_UPDATE","E":1758608899350,"o":{"s":"BTC-USDT","i":43,"S":"SELL","o":"MARKET","q":"3","z":"1","x":"TRADE","X":"PARTIALLY_FILLED"}}`, "43"},
		{"an order ID sent as a string", `{"e":"ORDER_TRADE_UPDATE","E":1758608899350,"o":{"s":"BTC-USDT","i":"44","S":"SELL","o":"MARKET","q":"3","z":"1","x":"TRADE","X":"PARTIALLY_FILLED"}}`, "44"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			conn := &testWebsocketConnection{}
			state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, nil)
			state.holding = true
			require.True(t, ex.wsHoldOrderEvent(conn, []byte(tc.event)), "wsHoldOrderEvent must hold the event while the read is in flight")
			exp := map[*websocketConnection]map[perpetualOrderKey]struct{}{state: {{asset: asset.USDTMarginedFutures, orderID: tc.orderID}: {}}}
			assert.Equal(t, exp, ex.wsState.orderRelay.holders, "wsHoldOrderEvent should note the event's order by its order ID")
		})
	}
}

func TestWebsocketReleaseOrderEvents(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	creds := &accounts.Credentials{Key: "account-key", Secret: "account-secret"}
	conn := &testWebsocketConnection{}
	state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
	state.holding = true
	event := func(executed string) []byte {
		return fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":"TRADE","X":"PARTIALLY_FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, executed)
	}
	// An event that fails comes first, so the events after it are handled whatever its outcome
	require.True(t, ex.wsHoldOrderEvent(conn, []byte(`{"o":{"sg":1}}`)), "wsHoldOrderEvent must hold an event that does not decode")
	require.True(t, ex.wsHoldOrderEvent(conn, event("1")), "wsHoldOrderEvent must hold the first event")
	require.True(t, ex.wsHoldOrderEvent(conn, event("2")), "wsHoldOrderEvent must hold the second event")
	ex.wsReleaseOrderEvents(t.Context(), state, nil, false)
	fees := []float64{waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee}
	assert.InDeltaSlice(t, []float64{0.1, 0.2}, fees, 1e-12, "the held events should be handled in the order they arrived")
	assert.Empty(t, websocketPayloads(ex), "an event that does not decode should be logged and dropped")
	assert.False(t, state.holding, "wsReleaseOrderEvents should stop holding order events")
	assert.Empty(t, state.held, "wsReleaseOrderEvents should drop the events it handled")
	assert.False(t, ex.wsHoldOrderEvent(conn, []byte(`{}`)), "an event after the release should not be held")

	t.Run("an order to look up first", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
		state.holding = true
		require.True(t, ex.wsHoldOrderEvent(conn, event("1")), "wsHoldOrderEvent must hold the event")
		lookups := ex.wsReleaseOrderEvents(t.Context(), state, map[string]bool{}, true)
		assert.Equal(t, []perpetualOrderLookup{{orderID: "42", symbol: "BTC-USDT"}}, lookups, "wsReleaseOrderEvents should return the order whose held fill lacks its earlier fees")
		assert.True(t, state.holding, "wsReleaseOrderEvents should keep holding order events until the order is looked up")
		assert.Len(t, state.held, 1, "wsReleaseOrderEvents should keep the held event")
		assert.Empty(t, websocketPayloads(ex), "nothing should be relayed before the order is looked up")
		assert.Empty(t, ex.wsReleaseOrderEvents(t.Context(), state, map[string]bool{"42": true}, true), "wsReleaseOrderEvents should not return an order already looked up")
		assert.False(t, state.holding, "wsReleaseOrderEvents should stop holding order events once no order is left to look up")
		assert.InDelta(t, 0.1, waitForWebsocketPayload[*order.Detail](t, ex, nil).Fee, 1e-12, "the held event should be relayed once no order is left to look up")
	})

	t.Run("an order event arriving during the release", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t)
		conn := &testWebsocketConnection{}
		state := ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, creds)
		state.holding = true
		fill := func(executed, fee string) []byte {
			return fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"BTC-USDT","i":42,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":"TRADE","X":"PARTIALLY_FILLED","N":"USDT","n":%q,"T":1758608899001}}`, executed, fee)
		}
		require.True(t, ex.wsHoldOrderEvent(conn, fill("1", "-0.1")), "wsHoldOrderEvent must hold the first fill")
		require.True(t, ex.wsHoldOrderEvent(conn, fill("2", "-0.2")), "wsHoldOrderEvent must hold the second fill")
		ctx := &testPausingContext{Context: t.Context(), paused: make(chan struct{}), resume: make(chan struct{})}
		var resumeOnce sync.Once
		resume := func() { resumeOnce.Do(func() { close(ctx.resume) }) }
		// A release reaching its context after the test failed would otherwise wait for resume for good
		t.Cleanup(resume)
		released := make(chan struct{})
		go func() {
			ex.wsReleaseOrderEvents(ctx, state, nil, false)
			close(released)
		}()
		select {
		case <-ctx.paused:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the release must look its context up")
		}
		handled := make(chan error, 1)
		go func() {
			handled <- ex.wsHandlePerpetualAccountData(t.Context(), conn, fill("3", "-0.3"), asset.USDTMarginedFutures)
		}()
		// The later event gets the time to overtake the held ones, which it must not take
		time.Sleep(50 * time.Millisecond)
		resume()
		select {
		case <-released:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the release must end")
		}
		select {
		case err := <-handled:
			require.NoError(t, err, "the later event must be handled")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the later event must be handled once the release ends")
		}
		executed, fees := make([]float64, 0, 3), make([]float64, 0, 3)
		for range 3 {
			d := waitForWebsocketPayload[*order.Detail](t, ex, nil)
			executed = append(executed, d.ExecutedAmount)
			fees = append(fees, d.Fee)
		}
		assert.InDeltaSlice(t, []float64{1, 2, 3}, executed, 1e-12, "the relays should carry the fill events in the order they arrived")
		assert.InDeltaSlice(t, []float64{0.1, 0.3, 0.6}, fees, 1e-12, "the relays should carry the fees of the fills so far")
	})

	t.Run("a connection retired during its read", func(t *testing.T) {
		t.Parallel()
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			t.Run(a.String(), func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
				server := newTestPerpetualOrderServer(t, ex, a, false)
				creds, err := ex.GetCredentials(t.Context())
				require.NoError(t, err, "GetCredentials must not error")
				session := newTestSession(t, ex)
				retired := &testWebsocketConnection{}
				state := ex.wsTrackConnection(retired, session.ch, false, a, creds)
				event := func(orderID int, execution, status, executed, fee string) []byte {
					return fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":%q,"i":%d,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":%q,"X":%q,"N":"USDT","n":%q,"T":1758608899001}}`, server.symbol, orderID, executed, execution, status, fee)
				}
				server.setOrder("1", "-0.1", "PARTIALLY_FILLED")
				arrived, release := server.holdNext()
				var releaseOnce sync.Once
				t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
				waitForTestRead(t, func() { ex.wsReadOpenOrderFees(t.Context(), state) }, arrived)
				for _, step := range []struct {
					name  string
					event []byte
				}{
					{"order 42 filled to 2", event(42, "TRADE", "PARTIALLY_FILLED", "2", "-0.2")},
					{"order 43 placed", event(43, "NEW", "NEW", "0", "0")},
					{"order 43 filled to 1", event(43, "TRADE", "PARTIALLY_FILLED", "1", "-0.4")},
					{"order 43 amended", event(43, "NEW", "PARTIALLY_FILLED", "1", "0")},
				} {
					// The events are held in the order they arrive
					t.Run(step.name, func(t *testing.T) {
						require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), retired, step.event, a), "an order event must be handled while the read is in flight")
					})
				}
				require.NoError(t, retired.Shutdown(), "Shutdown must not error")
				// The connection dialled in its place forgets the retired one, and streams order 43's cancellation before the read ends
				replacement := &testWebsocketConnection{}
				ex.wsTrackConnection(replacement, session.ch, false, a, creds)
				_, err = ex.wsConnection(retired)
				require.ErrorIs(t, err, errWebsocketConnectionUnknown, "the retired connection must be forgotten")
				require.NoError(t, ex.wsHandlePerpetualAccountData(t.Context(), replacement, event(43, "CANCELED", "CANCELED", "1", "0"), a), "the replacement's event must be handled")
				assert.Equal(t, []any{testRelayedOrder{"43", 1, order.Cancelled, 0}}, testRelayedOrders(ex), "the replacement's event should be relayed while the retired connection holds its own")
				releaseOnce.Do(func() { close(release) })
				testWaitBackground(t, ex)
				assert.Equal(t, []any{testRelayedOrder{"42", 2, order.PartiallyFilled, 0.3}}, testRelayedOrders(ex), "the retired connection's events behind the replacement's should be dropped, and its others relayed with its own fee totals")
				relay := &ex.wsState.orderRelay
				relay.mu.Lock()
				defer relay.mu.Unlock()
				assert.Empty(t, relay.holders, "the release should end the connection's hold")
				assert.Empty(t, relay.relayed, "how far the orders went should be forgotten once no connection holds events of them")
			})
		}
	})
}

func TestHeldOrdersMissingFees(t *testing.T) {
	t.Parallel()
	event := func(orderID int, execution, executed string) []byte {
		return fmt.Appendf(nil, `{"e":"ORDER_TRADE_UPDATE","E":1758608899001,"o":{"s":"ETH-USDT","i":%d,"S":"BUY","o":"LIMIT","q":"3","p":"10","ap":"10","z":%q,"x":%q,"X":"PARTIALLY_FILLED","N":"USDT","n":"-0.1","T":1758608899001}}`, orderID, executed, execution)
	}
	missing := func(orderIDs ...string) []perpetualOrderLookup {
		if len(orderIDs) == 0 {
			return nil
		}
		lookups := make([]perpetualOrderLookup, 0, len(orderIDs))
		for _, orderID := range orderIDs {
			lookups = append(lookups, perpetualOrderLookup{orderID: orderID, symbol: "ETH-USDT"})
		}
		return lookups
	}
	for _, tc := range []struct {
		name    string
		held    [][]byte
		settled map[string]bool
		total   string // how order 42's total was started before the events: "", "read" or "fill"
		exp     []perpetualOrderLookup
	}{
		{"a fill", [][]byte{event(42, "TRADE", "1")}, nil, "", missing("42")},
		{"a fill reported as calculated", [][]byte{event(42, "CALCULATED", "1")}, nil, "", missing("42")},
		{"no fill", [][]byte{event(42, "CANCELED", "1")}, nil, "", missing()},
		{"a NEW event reporting nothing executed first", [][]byte{event(42, "NEW", "0"), event(42, "TRADE", "1")}, nil, "", missing()},
		{"a NEW event reporting fills first", [][]byte{event(42, "NEW", "1"), event(42, "TRADE", "2")}, nil, "", missing("42")},
		{"a fill then an amendment", [][]byte{event(42, "TRADE", "1"), event(42, "NEW", "1")}, nil, "", missing("42")},
		{"an order settled", [][]byte{event(42, "TRADE", "1")}, map[string]bool{"42": true}, "", missing()},
		{"a total counting every fill", [][]byte{event(42, "TRADE", "2")}, nil, "read", missing()},
		{"a total missing its first fills", [][]byte{event(42, "TRADE", "2")}, nil, "fill", missing("42")},
		{"an event that does not decode", [][]byte{[]byte(`{"o":{"sg":1}}`), event(42, "TRADE", "1")}, nil, "", missing("42")},
		{"several orders", [][]byte{
			event(47, "TRADE", "1"), event(41, "TRADE", "1"), event(44, "NEW", "0"), event(45, "TRADE", "1"), event(47, "TRADE", "2"),
			event(43, "TRADE", "1"), event(44, "TRADE", "1"), event(49, "TRADE", "1"), event(42, "TRADE", "1"), event(48, "TRADE", "1"),
			event(46, "TRADE", "1"),
		}, nil, "", missing("47", "41", "45", "43", "49", "42", "48", "46")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := &websocketConnection{account: asset.USDTMarginedFutures, held: tc.held}
			switch tc.total {
			case "read":
				state.fees.seed("42", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
			case "fill":
				state.fees.add("42", decimal.NewFromInt(1), decimal.MustFromString("0.1"))
			}
			assert.Equal(t, tc.exp, heldOrdersMissingFees(state, tc.settled), "heldOrdersMissingFees should return the orders whose held fills would start a total without their earlier fees")
		})
	}
}

func TestWebsocketLookUpOrderFees(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
			server := newTestPerpetualOrderServer(t, ex, a, false)
			server.setOrder("3", "-0.33", "FILLED")
			creds, err := ex.GetCredentials(t.Context())
			require.NoError(t, err, "GetCredentials must not error")
			state := ex.wsTrackConnection(&testWebsocketConnection{}, newTestSession(t, ex).ch, false, a, creds)
			settled := map[string]bool{"41": true}
			ex.wsLookUpOrderFees(accounts.DeployCredentialsToContext(t.Context(), creds), state, []perpetualOrderLookup{{orderID: "42", symbol: server.symbol}, {orderID: "x", symbol: server.symbol}}, settled)
			assert.Equal(t, map[string]bool{"41": true, "42": true, "x": true}, settled, "each order should be settled, whether or not its lookup succeeds")
			assert.Equal(t, []string{server.getPath + "?orderId=42"}, server.requestRoutes(), "an order should be looked up by its ID, and one whose ID does not parse not at all")
			state.fees.mu.Lock()
			defer state.fees.mu.Unlock()
			require.Contains(t, state.fees.orders, "42", "a closed order's lookup must start its total")
			total := state.fees.orders["42"]
			assert.True(t, total.complete, "a looked up total should count every fill")
			assert.True(t, decimal.NewFromInt(3).Equal(total.executed), "a looked up total should take the executed quantity BingX reports")
			assert.True(t, decimal.MustFromString("0.33").Equal(total.fee), "a looked up total should take the commission BingX reports as a fee paid")
		})
	}
}

func TestPerpetualOrderFees(t *testing.T) {
	t.Parallel()
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
			server := newTestPerpetualOrderServer(t, ex, a, false)
			server.setOrder("2", "-0.23", "PARTIALLY_FILLED")
			executed, commission, err := ex.perpetualOrderFees(t.Context(), a, perpetualOrderLookup{orderID: "42", symbol: server.symbol})
			require.NoError(t, err, "perpetualOrderFees must not error")
			assert.Equal(t, 2.0, executed.Float64(), "perpetualOrderFees should return the executed quantity BingX reports")
			assert.Equal(t, -0.23, commission.Float64(), "perpetualOrderFees should return the commission BingX reports")
			server.setGetReply(`{"code":100500,"msg":"internal error"}`)
			_, _, err = ex.perpetualOrderFees(t.Context(), a, perpetualOrderLookup{orderID: "42", symbol: server.symbol})
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr, "perpetualOrderFees must return BingX's refusal")
			assert.Equal(t, int64(100500), apiErr.Code, "the refusal should carry BingX's code")
		})
	}

	for _, tc := range []struct {
		name   string
		a      asset.Item
		lookup perpetualOrderLookup
		err    error
	}{
		{"a symbol that does not parse", asset.USDTMarginedFutures, perpetualOrderLookup{orderID: "42", symbol: "BTCUSDT"}, errWebsocketInvalidSymbol},
		{"an order ID that does not parse", asset.CoinMarginedFutures, perpetualOrderLookup{orderID: "x", symbol: "BTC-USD"}, errInvalidOrderID},
		{"spot", asset.Spot, perpetualOrderLookup{orderID: "42", symbol: "BTC-USDT"}, asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newWebsocketTestExchange(t)
			_, _, err := ex.perpetualOrderFees(t.Context(), tc.a, tc.lookup)
			assert.ErrorIs(t, err, tc.err, "perpetualOrderFees should refuse a lookup it cannot send")
		})
	}
}

// testRelayedOrder is what a relayed order detail reports of an order's progress
type testRelayedOrder struct {
	orderID  string
	executed float64
	status   order.Status
	fee      float64
}

// testRelayedOrders drains what has been relayed so far, with each order detail as a testRelayedOrder
func testRelayedOrders(ex *Exchange) []any {
	payloads := websocketPayloads(ex)
	got := make([]any, 0, len(payloads))
	for _, p := range payloads {
		if d, ok := p.(*order.Detail); ok {
			p = testRelayedOrder{d.OrderID, d.ExecutedAmount, d.Status, d.Fee}
		}
		got = append(got, p)
	}
	return got
}

// testPausingContext holds up the first lookup made through it, closing paused, until resume is closed
type testPausingContext struct {
	context.Context //nolint:containedctx // the type is a context wrapping another, to pause the first lookup made through it
	once            sync.Once
	paused, resume  chan struct{}
}

func (c *testPausingContext) Value(key any) any {
	c.once.Do(func() {
		close(c.paused)
		<-c.resume
	})
	return c.Context.Value(key)
}

func TestPerpetualOrderRelayHold(t *testing.T) {
	t.Parallel()
	var r perpetualOrderRelay
	usdt, coin := &websocketConnection{account: asset.USDTMarginedFutures}, &websocketConnection{account: asset.CoinMarginedFutures}
	r.hold(usdt, "42")
	r.hold(usdt, "43")
	r.hold(usdt, "42")
	r.hold(coin, "42")
	key := func(a asset.Item, orderID string) perpetualOrderKey {
		return perpetualOrderKey{asset: a, orderID: orderID}
	}
	assert.Equal(t, map[*websocketConnection]map[perpetualOrderKey]struct{}{
		usdt: {key(asset.USDTMarginedFutures, "42"): {}, key(asset.USDTMarginedFutures, "43"): {}},
		coin: {key(asset.CoinMarginedFutures, "42"): {}},
	}, r.holders, "hold should note each connection holding events of its product's orders")
	r.hold(&websocketConnection{account: asset.USDTMarginedFutures}, "42")
	assert.Equal(t, map[perpetualOrderKey]int{
		key(asset.USDTMarginedFutures, "42"): 2,
		key(asset.USDTMarginedFutures, "43"): 1,
		key(asset.CoinMarginedFutures, "42"): 1,
	}, r.heldBy, "hold should count the connections holding events of each order")

	relayed := func(executed int64, status order.Status) bool {
		var sent bool
		d := &order.Detail{OrderID: "42", AssetType: asset.USDTMarginedFutures, Amount: 3, ExecutedAmount: float64(executed), Status: status}
		require.NoError(t, r.relay(d, decimal.NewFromInt(executed), func() error {
			sent = true
			return nil
		}), "relay must not error")
		return sent
	}
	require.True(t, relayed(3, order.Filled), "the fill must be relayed")
	r.hold(&websocketConnection{account: asset.CoinMarginedFutures}, "44")
	assert.False(t, relayed(1, order.PartiallyFilled), "hold should keep how far the orders other connections hold events of went")
}

func TestPerpetualOrderRelayUnhold(t *testing.T) {
	t.Parallel()
	var r perpetualOrderRelay
	older, newer := &websocketConnection{account: asset.USDTMarginedFutures}, &websocketConnection{account: asset.USDTMarginedFutures}
	r.hold(older, "42")
	r.hold(older, "43")
	r.hold(newer, "42")
	relayed := func(orderID string, executed int64, status order.Status) bool {
		var sent bool
		d := &order.Detail{OrderID: orderID, AssetType: asset.USDTMarginedFutures, Amount: 3, ExecutedAmount: float64(executed), Status: status}
		require.NoError(t, r.relay(d, decimal.NewFromInt(executed), func() error {
			sent = true
			return nil
		}), "relay must not error")
		return sent
	}
	require.True(t, relayed("42", 3, order.Filled), "the fill must be relayed")
	require.True(t, relayed("43", 2, order.Filled), "the fill must be relayed")
	r.unhold(older)
	r.unhold(&websocketConnection{account: asset.USDTMarginedFutures})
	assert.Equal(t, []*websocketConnection{newer}, slices.Collect(maps.Keys(r.holders)), "unhold should end only the hold of the connection given")
	assert.False(t, relayed("42", 1, order.PartiallyFilled), "unhold should keep how far an order went while another connection holds events of it")
	later := &websocketConnection{account: asset.USDTMarginedFutures}
	r.hold(later, "43")
	assert.True(t, relayed("43", 1, order.PartiallyFilled), "unhold should forget how far an order went once no connection holds events of it")
	r.unhold(newer)
	r.unhold(later)
	assert.Nil(t, r.holders, "unhold should drop the holds once none is left")
	assert.Nil(t, r.heldBy, "unhold should drop which orders were held once no hold is left")
	assert.Nil(t, r.relayed, "unhold should drop how far the orders went once no hold is left")
	assert.True(t, relayed("42", 1, order.PartiallyFilled), "an event of an order no connection holds events of should be relayed")
}

func TestPerpetualOrderRelayRelay(t *testing.T) {
	t.Parallel()
	type event struct {
		a                asset.Item
		orderID          string
		amount, executed string
		status           order.Status
	}
	usdt := func(executed string, status order.Status) event {
		return event{asset.USDTMarginedFutures, "42", "3", executed, status}
	}
	usdtOrder := []perpetualOrderKey{{asset: asset.USDTMarginedFutures, orderID: "42"}}
	for _, tc := range []struct {
		name string
		// held holds the orders connections hold events of
		held   []perpetualOrderKey
		events []event
		// relayed holds the indexes of the events relayed
		relayed []int
	}{
		{name: "an order no connection holds events of", events: []event{usdt("2", order.Cancelled), usdt("1", order.PartiallyFilled), usdt("1", order.New)}, relayed: []int{0, 1, 2}},
		{
			name:    "an order no connection holds events of, while another order is held",
			held:    []perpetualOrderKey{{asset: asset.USDTMarginedFutures, orderID: "43"}},
			events:  []event{usdt("2", order.Cancelled), usdt("1", order.PartiallyFilled), usdt("1", order.New)},
			relayed: []int{0, 1, 2},
		},
		{name: "events taking the order further", held: usdtOrder, events: []event{usdt("0", order.New), usdt("1", order.PartiallyFilled), usdt("3", order.Filled)}, relayed: []int{0, 1, 2}},
		{name: "an event at a lower executed quantity", held: usdtOrder, events: []event{usdt("2", order.PartiallyFilled), usdt("1", order.PartiallyFilled)}, relayed: []int{0}},
		{name: "an event of nothing executed after a fill", held: usdtOrder, events: []event{usdt("2", order.PartiallyFilled), usdt("0", order.New)}, relayed: []int{0}},
		{
			name:    "an event a fraction behind",
			held:    usdtOrder,
			events:  []event{{asset.USDTMarginedFutures, "42", "0.005", "0.002", order.PartiallyFilled}, {asset.USDTMarginedFutures, "42", "0.005", "0.001", order.PartiallyFilled}},
			relayed: []int{0},
		},
		{name: "an event behind a fill finishing the order", held: usdtOrder, events: []event{usdt("1", order.PartiallyFilled), usdt("3", order.Filled), usdt("2", order.PartiallyFilled)}, relayed: []int{0, 1}},
		{name: "an active event at the quantity the order was cancelled at", held: usdtOrder, events: []event{usdt("2", order.Cancelled), usdt("2", order.PartiallyFilled)}, relayed: []int{0}},
		{name: "an active event at the quantity the order expired at", held: usdtOrder, events: []event{usdt("1", order.Expired), usdt("1", order.PartiallyFilled)}, relayed: []int{0}},
		{name: "an active event at the quantity the order was rejected at", held: usdtOrder, events: []event{usdt("0", order.Rejected), usdt("0", order.New)}, relayed: []int{0}},
		{name: "an event of unknown status at the quantity the order was cancelled at", held: usdtOrder, events: []event{usdt("2", order.Cancelled), usdt("2", order.UnknownStatus)}, relayed: []int{0}},
		{name: "an event of unknown status filling the order at the quantity a fill finished it at", held: usdtOrder, events: []event{usdt("3", order.Filled), usdt("3", order.UnknownStatus)}, relayed: []int{0, 1}},
		{name: "an active event at the quantity of an active one", held: usdtOrder, events: []event{usdt("2", order.PartiallyFilled), usdt("2", order.PartiallyFilled)}, relayed: []int{0, 1}},
		{
			name:    "an amendment lowering the order's quantity at the same executed quantity",
			held:    usdtOrder,
			events:  []event{{asset.USDTMarginedFutures, "42", "3", "1", order.PartiallyFilled}, {asset.USDTMarginedFutures, "42", "2", "1", order.PartiallyFilled}},
			relayed: []int{0, 1},
		},
		{name: "a close at the quantity of an active event", held: usdtOrder, events: []event{usdt("2", order.PartiallyFilled), usdt("2", order.Cancelled), usdt("2", order.PartiallyFilled)}, relayed: []int{0, 1}},
		{name: "a close repeated", held: usdtOrder, events: []event{usdt("2", order.Cancelled), usdt("2", order.Cancelled)}, relayed: []int{0, 1}},
		{
			name: "an order of no quantity, finished only by its status",
			held: usdtOrder,
			events: []event{
				{asset.USDTMarginedFutures, "42", "0", "0", order.New},
				{asset.USDTMarginedFutures, "42", "0", "0", order.New},
				{asset.USDTMarginedFutures, "42", "0", "0", order.Cancelled},
				{asset.USDTMarginedFutures, "42", "0", "0", order.New},
			},
			relayed: []int{0, 1, 2},
		},
		{
			name:    "a coin-M order",
			held:    []perpetualOrderKey{{asset: asset.CoinMarginedFutures, orderID: "42"}},
			events:  []event{{asset.CoinMarginedFutures, "42", "3", "3", order.PartiallyFilled}, {asset.CoinMarginedFutures, "42", "3", "1", order.PartiallyFilled}},
			relayed: []int{0},
		},
		{
			name:    "the same order ID on two products",
			held:    []perpetualOrderKey{{asset: asset.USDTMarginedFutures, orderID: "42"}, {asset: asset.CoinMarginedFutures, orderID: "42"}},
			events:  []event{usdt("3", order.Filled), {asset.CoinMarginedFutures, "42", "3", "1", order.PartiallyFilled}},
			relayed: []int{0, 1},
		},
		{
			name:    "adjacent 19 digit order IDs",
			held:    []perpetualOrderKey{{asset: asset.USDTMarginedFutures, orderID: "2000000000000000000"}, {asset: asset.USDTMarginedFutures, orderID: "2000000000000000001"}},
			events:  []event{{asset.USDTMarginedFutures, "2000000000000000000", "3", "3", order.Filled}, {asset.USDTMarginedFutures, "2000000000000000001", "3", "1", order.PartiallyFilled}},
			relayed: []int{0, 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var r perpetualOrderRelay
			for _, k := range tc.held {
				r.hold(&websocketConnection{account: k.asset}, k.orderID)
			}
			var got []int
			var errs error
			for i, e := range tc.events {
				executed := decimal.MustFromString(e.executed)
				d := &order.Detail{OrderID: e.orderID, AssetType: e.a, Amount: decimal.MustFromString(e.amount).InexactFloat64(), ExecutedAmount: executed.InexactFloat64(), Status: e.status}
				errs = errors.Join(errs, r.relay(d, executed, func() error {
					got = append(got, i)
					return nil
				}))
			}
			require.NoError(t, errs, "relay must not error")
			assert.Equal(t, tc.relayed, got, "relay should relay the events not behind those relayed while a connection holds events of their order")
			kept := slices.Collect(maps.Keys(r.relayed))
			assert.Subset(t, tc.held, kept, "relay should keep nothing of an order no connection holds events of")
		})
	}

	// newHeld returns a func relaying an event of USDT-M order 42, which a connection holds events of, with a send
	// returning sendErr, reporting whether the event was sent
	newHeld := func() func(executed int64, status order.Status, sendErr error) (bool, error) {
		r := new(perpetualOrderRelay)
		r.hold(&websocketConnection{account: asset.USDTMarginedFutures}, "42")
		return func(executed int64, status order.Status, sendErr error) (bool, error) {
			var sent bool
			d := &order.Detail{OrderID: "42", AssetType: asset.USDTMarginedFutures, Amount: 3, ExecutedAmount: float64(executed), Status: status}
			err := r.relay(d, decimal.NewFromInt(executed), func() error {
				sent = true
				return sendErr
			})
			return sent && err == nil, err
		}
	}
	errRefused := errors.New("refused")

	t.Run("an event refused after one relayed", func(t *testing.T) {
		t.Parallel()
		relay := newHeld()
		_, err := relay(2, order.PartiallyFilled, nil)
		require.NoError(t, err, "relay must not error")
		_, err = relay(3, order.Filled, errRefused)
		require.ErrorIs(t, err, errRefused, "relay must return the refusal")
		sent, err := relay(1, order.PartiallyFilled, nil)
		require.NoError(t, err, "relay must not error")
		assert.False(t, sent, "an event behind one relayed before a refused one should be dropped")
		sent, err = relay(2, order.PartiallyFilled, nil)
		require.NoError(t, err, "relay must not error")
		assert.True(t, sent, "an event at the quantity of one relayed before a refused one should be relayed")
	})

	t.Run("a first event refused", func(t *testing.T) {
		t.Parallel()
		relay := newHeld()
		_, err := relay(3, order.Filled, errRefused)
		require.ErrorIs(t, err, errRefused, "relay must return the refusal")
		sent, err := relay(1, order.PartiallyFilled, nil)
		require.NoError(t, err, "relay must not error")
		assert.True(t, sent, "an event behind only a refused one should be relayed")
	})

	t.Run("a close refused at the quantity of an event relayed", func(t *testing.T) {
		t.Parallel()
		relay := newHeld()
		_, err := relay(1, order.PartiallyFilled, nil)
		require.NoError(t, err, "relay must not error")
		_, err = relay(1, order.Cancelled, errRefused)
		require.ErrorIs(t, err, errRefused, "relay must return the refusal")
		sent, err := relay(1, order.PartiallyFilled, nil)
		require.NoError(t, err, "relay must not error")
		assert.True(t, sent, "an active event at the quantity of a refused close should be relayed")
	})

	t.Run("the lock while an event is sent", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			// held holds the orders a connection holds events of
			held []string
		}{
			{"an order a connection holds events of", []string{"42"}},
			{"an order no connection holds events of, while another is held", []string{"90"}},
			{"no order held", nil},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var r perpetualOrderRelay
				for _, orderID := range tc.held {
					r.hold(&websocketConnection{account: asset.USDTMarginedFutures}, orderID)
				}
				sending, resume := make(chan struct{}), make(chan struct{})
				returned := make(chan error, 1)
				go func() {
					d := &order.Detail{OrderID: "42", AssetType: asset.USDTMarginedFutures, Amount: 3, ExecutedAmount: 1, Status: order.PartiallyFilled}
					returned <- r.relay(d, decimal.NewFromInt(1), func() error {
						close(sending)
						<-resume
						return nil
					})
				}()
				select {
				case <-sending:
				case <-time.After(5 * time.Second):
					require.FailNow(t, "the event must be sent")
				}
				locked := r.mu.TryLock()
				if locked {
					r.mu.Unlock()
				}
				close(resume)
				assert.False(t, locked, "relay should hold its lock while it sends, so no other event is checked or sent meanwhile")
				select {
				case err := <-returned:
					require.NoError(t, err, "relay must not error")
				case <-time.After(5 * time.Second):
					require.FailNow(t, "relay must return")
				}
			})
		}
	})
}

func TestWebsocketProcessPerpetualAccountUpdate(t *testing.T) {
	t.Parallel()
	server := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, server.URL)
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	// A connection no Connector dialled has no session, so its events start no refresh
	untracked := &testWebsocketConnection{}
	require.NoError(t, ex.wsProcessPerpetualAccountUpdate(t.Context(), untracked, asset.USDTMarginedFutures, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "ACCOUNT_UPDATE")), "wsProcessPerpetualAccountUpdate must not error")
	relayed := waitForWebsocketPayload[[]WebsocketPerpetualBalance](t, ex, nil)
	assert.Equal(t, []WebsocketPerpetualBalance{{Asset: currency.USDT, WalletBalance: 117.80007595, CrossWalletBalance: 95.23045595}}, relayed, "the event's balances should be relayed as it reports them")
	positions := waitForWebsocketPayload[[]futures.Position](t, ex, nil)
	exp := []futures.Position{{
		Exchange:         ex.Name,
		Asset:            asset.USDTMarginedFutures,
		Pair:             testSwapPair,
		PositionMargin:   types.Number(22.56957565).Decimal(),
		RealisedPNL:      types.Number(-0.05642405).Decimal(),
		UnrealisedPNL:    types.Number(-0.00004435).Decimal(),
		Status:           order.Open,
		OpeningPrice:     types.Number(112848.1).Decimal(),
		OpeningDirection: order.Short,
		LatestSize:       types.Number(0.001).Decimal().Abs(),
		LatestDirection:  order.Short,
		LastUpdated:      time.UnixMilli(1758608899352),
	}}
	assert.Equal(t, exp, positions, "the position should be relayed")
	_, err = ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "the event's wallet balance should not be stored as the available balance")

	conn := &testWebsocketConnection{}
	trackTestConnection(t, ex, conn, false)
	for _, tc := range []struct {
		a     asset.Item
		event []byte
		path  string
		code  currency.Code
		exp   accounts.Balance
	}{
		{asset.USDTMarginedFutures, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "ACCOUNT_UPDATE"), "/openApi/swap/v3/user/balance", currency.USDT, accounts.Balance{Currency: currency.USDT, Total: 194.8212, Hold: 1.0602, Free: 193.7609}},
		{asset.CoinMarginedFutures, fixtureLine(t, "testdata/wsCoinMarginedAccount.jsonl", "ACCOUNT_UPDATE"), "/openApi/cswap/v1/user/balance", currency.BTC, accounts.Balance{Currency: currency.BTC, Total: 0.14438227, Hold: 0.00110845, Free: 0.14317271}},
	} {
		// The cases share ex and its server, whose balance requests are counted below, so they run in turn
		t.Run(tc.a.String(), func(t *testing.T) {
			require.NoErrorf(t, ex.wsProcessPerpetualAccountUpdate(t.Context(), conn, tc.a, tc.event), "wsProcessPerpetualAccountUpdate must not error for %s", tc.a)
			waitForWebsocketPayload[[]WebsocketPerpetualBalance](t, ex, nil)
			var stored accounts.Balance
			require.Eventuallyf(t, func() bool {
				stored, err = ex.Accounts.GetBalance("", creds, tc.a, tc.code)
				return err == nil
			}, 5*time.Second, time.Millisecond, "the %s balances must be refreshed over REST", tc.a)
			tc.exp.UpdatedAt = stored.UpdatedAt
			assert.Equalf(t, tc.exp, stored, "the %s balances should be REST's", tc.a)
			assert.Equalf(t, 1, server.balanceRequestCount(tc.path), "a %s balance event should refresh the balances once", tc.a)
		})
	}
	websocketPayloads(ex)

	require.NoError(t, ex.wsProcessPerpetualAccountUpdate(t.Context(), conn, asset.USDTMarginedFutures, []byte(`{"e":"ACCOUNT_UPDATE","a":{"B":[],"P":[{"s":"BTCUSDT","pa":"1"}]}}`)), "wsProcessPerpetualAccountUpdate must not error")
	assert.Empty(t, websocketPayloads(ex), "an event without balances, whose positions cannot be read, should relay nothing")
	assert.Error(t, ex.wsProcessPerpetualAccountUpdate(t.Context(), conn, asset.USDTMarginedFutures, []byte(`{"a":{"B":{}}}`)), "wsProcessPerpetualAccountUpdate should error for an event that does not decode")
	testWaitBackground(t, ex)
	assert.Equal(t, 1, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "an event without balances should not refresh them")

	t.Run("balances the data handler refuses", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		ex.Websocket.DataHandler = stream.NewRelay(1)
		require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "full"), "Send must not error")
		err := ex.wsProcessPerpetualAccountUpdate(t.Context(), &testWebsocketConnection{}, asset.USDTMarginedFutures, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "ACCOUNT_UPDATE"))
		assert.ErrorContains(t, err, "failed to relay <[]bingx.WebsocketPerpetualBalance>", "wsProcessPerpetualAccountUpdate should report balances it cannot relay, before its positions")
	})

	t.Run("credentials of the listen key", func(t *testing.T) {
		t.Parallel()
		server := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, server.URL)
		ex.balances.interval = time.Millisecond
		first, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		conn := &testWebsocketConnection{}
		ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, asset.USDTMarginedFutures, first)
		second := &accounts.Credentials{Key: "second-key", Secret: "second-secret"}
		ex.SetCredentials(second)
		require.NoError(t, ex.wsProcessPerpetualAccountUpdate(t.Context(), conn, asset.USDTMarginedFutures, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "ACCOUNT_UPDATE")), "wsProcessPerpetualAccountUpdate must not error")
		require.Eventually(t, func() bool {
			_, err := ex.Accounts.GetBalance("", first, asset.USDTMarginedFutures, currency.USDT)
			return err == nil
		}, 5*time.Second, time.Millisecond, "the balances must be refreshed for the account the connection's listen key was created for")
		testWaitBackground(t, ex)
		assert.Equal(t, []string{first.Key}, server.requestKeys("GET /openApi/swap/v3/user/balance"), "the refresh should be requested with the credentials of the connection's listen key")
		_, err = ex.Accounts.GetBalance("", second, asset.USDTMarginedFutures, currency.USDT)
		assert.ErrorIs(t, err, accounts.ErrNoBalances, "no balances should be refreshed for the exchange's new default credentials")
	})
}

func TestWebsocketPerpetualPosition(t *testing.T) {
	t.Parallel()
	updated := time.UnixMilli(1758608899352)
	for _, tc := range []struct {
		name   string
		side   PositionSide
		amount types.Number
		exp    order.Side
		status order.Status
	}{
		{"hedge long", PositionSideLong, 1, order.Long, order.Open},
		{"hedge short", PositionSideShort, 1, order.Short, order.Open},
		{"one-way long", PositionSideBoth, 2, order.Long, order.Open},
		{"one-way short", PositionSideBoth, -2, order.Short, order.Open},
		{"closed", PositionSideBoth, 0, order.UnknownSide, order.Closed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := wsPerpetualPosition("BingX", asset.USDTMarginedFutures, testSwapPair, &WebsocketPerpetualPosition{Symbol: "BTC-USDT", PositionAmount: tc.amount, PositionSide: tc.side}, updated)
			assert.Equal(t, tc.exp, p.OpeningDirection, "the position should have its side")
			assert.Equal(t, tc.exp, p.LatestDirection, "the position's latest side should be its side")
			assert.Equal(t, tc.status, p.Status, "the position should have its status")
			assert.True(t, p.LatestSize.Equal(tc.amount.Decimal().Abs()), "the size should be unsigned")
			if tc.status == order.Closed {
				assert.Equal(t, updated, p.CloseDate, "a closed position should be dated")
			} else {
				assert.Zero(t, p.CloseDate, "an open position should have no close date")
			}
		})
	}
}

func TestWebsocketSwapIntegration(t *testing.T) {
	t.Parallel()
	servers := newTestWebsocketServers(t, asset.USDTMarginedFutures)
	ex := newWebsocketIntegrationExchange(t, servers, asset.USDTMarginedFutures, true, func(cfg *config.Exchange) {
		cfg.Features.Enabled.TradeFeed = true
		cfg.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel, Levels: 5},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: WebsocketMarkPriceChannel},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
			{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
		}
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	assert.ElementsMatch(t, []string{"BTC-USDT@ticker", "BTC-USDT@trade", "BTC-USDT@kline_1m", "BTC-USDT@depth5", "BTC-USDT@markPrice", "ORDER_TRADE_UPDATE", "ACCOUNT_UPDATE"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "every subscription should be stored")
	requested := make([]string, 0, 5)
	for _, req := range servers.ws[asset.USDTMarginedFutures].receivedRequests() {
		requested = append(requested, req.DataType)
	}
	assert.ElementsMatch(t, []string{"BTC-USDT@ticker", "BTC-USDT@trade", "BTC-USDT@kline_1m", "BTC-USDT@depth5", "BTC-USDT@markPrice"}, requested, "only the market streams should be requested; the account events are pushed")

	waitForWebsocketPayload(t, ex, func(d *order.Detail) bool { return d.OrderID == "1970374651259388000" })
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	require.Eventually(t, func() bool {
		_, tickerErr := ticker.GetTicker(ex.Name, testSwapPair, asset.USDTMarginedFutures)
		_, bookErr := ex.Websocket.Orderbook.GetOrderbook(testSwapPair, asset.USDTMarginedFutures)
		balance, balanceErr := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
		return tickerErr == nil && bookErr == nil && balanceErr == nil && balance.Free == 193.7609 && len(servers.ws[asset.USDTMarginedFutures].heartbeatReplies()) == 2
	}, 5*time.Second, time.Millisecond, "the ticker, the book, the balance the account event refreshes over REST and both heartbeat replies must arrive")
	assert.Equal(t, []string{websocketSwapPong, websocketSwapPong}, servers.ws[asset.USDTMarginedFutures].heartbeatReplies(), "both connections should answer the text heartbeat")
	assert.Equal(t, []string{testListenKey + "-1"}, servers.ws[asset.USDTMarginedFutures].accountKeys(), "the account connection should be dialled with the listen key")
	for _, a := range []asset.Item{asset.Spot, asset.CoinMarginedFutures} {
		// Serial, as parallel cases would run only after the websocket is shut down below
		t.Run("disabled "+a.String(), func(t *testing.T) {
			assert.Zerof(t, servers.ws[a].marketDials(), "disabled %s should dial no market connection", a)
			assert.Emptyf(t, servers.ws[a].accountKeys(), "disabled %s should dial no account connection", a)
		})
	}

	ex.Features.Subscriptions = slices.DeleteFunc(slices.Clone(ex.Features.Subscriptions), func(s *subscription.Subscription) bool { return s.Authenticated })
	require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
	assert.ElementsMatch(t, []string{"BTC-USDT@ticker", "BTC-USDT@trade", "BTC-USDT@kline_1m", "BTC-USDT@depth5", "BTC-USDT@markPrice"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "the account subscriptions should be removed")
	assert.Len(t, servers.ws[asset.USDTMarginedFutures].receivedRequests(), 5, "removing the account subscriptions should send nothing")
	require.Eventually(t, func() bool { return servers.ws[asset.USDTMarginedFutures].accountCloses() == 1 }, 5*time.Second, time.Millisecond, "removing the last account subscription must close the account connection")
	shutdownTestWebsocket(t, ex)
	assert.ErrorIs(t, bookError(ex, testSwapPair, asset.USDTMarginedFutures), orderbook.ErrOrderbookInvalid, "shutting down should invalidate the book")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		testLiveMarketStreams(t, asset.USDTMarginedFutures, testSwapPair, 20)
	})
}
