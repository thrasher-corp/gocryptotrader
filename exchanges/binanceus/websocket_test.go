package binanceus

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// streamFrames holds the recorded stream frames of testdata/websocket_streams.jsonl, untouched live captures
var streamFrames = sync.OnceValues(func() ([][]byte, error) {
	f, err := os.Open("testdata/websocket_streams.jsonl")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var frames [][]byte
	s := bufio.NewScanner(f)
	for s.Scan() {
		frames = append(frames, append([]byte(nil), s.Bytes()...))
	}
	return frames, s.Err()
})

// streamFramesOf returns the recorded frames of a stream, in the order they arrived
func streamFramesOf(tb testing.TB, stream string) [][]byte {
	tb.Helper()
	frames, err := streamFrames()
	require.NoError(tb, err, "streamFrames must not error")
	var of [][]byte
	for _, frame := range frames {
		if name, err := jsonparser.GetString(frame, "stream"); err == nil && name == stream {
			of = append(of, frame)
		}
	}
	require.NotEmptyf(tb, of, "the fixture must hold %s frames", stream)
	return of
}

// streamData returns a recorded frame's payload
func streamData(tb testing.TB, frame []byte) []byte {
	tb.Helper()
	data, _, _, err := jsonparser.Get(frame, "data")
	require.NoError(tb, err, "the frame must carry data")
	return data
}

// newStreamTestExchange returns an exchange set up for the market data stream connection, without connecting, with a
// trade feed and an order book synchronisation taking its snapshots from the recorded partial depth without delay
func newStreamTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := newWebsocketTestExchange(tb)
	ex.Features.Enabled.TradeFeed = true
	useWebsocket(tb, ex, ex.streamConnectionSetup(ex.Config))
	ex.orderbookSync = ex.newOrderbookSync(recordedSnapshot(tb, ex), 0, 30*time.Second)
	return ex
}

// recordedSnapshot returns a snapshot fetch answering with the recorded partial depth payload, a complete top of the
// book at its update ID
func recordedSnapshot(tb testing.TB, ex *Exchange) func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
	tb.Helper()
	var partial WebsocketPartialDepth
	require.NoError(tb, json.Unmarshal(streamData(tb, streamFramesOf(tb, "btcusdt@depth5@100ms")[0]), &partial), "the partial depth frame must decode")
	return func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		return &orderbook.Book{
			Exchange: ex.Name, Pair: p, Asset: a, Bids: partial.Bids.Levels(), Asks: partial.Asks.Levels(),
			LastUpdateID: partial.LastUpdateID, LastUpdated: time.Now(), ValidateOrderbook: true,
		}, nil
	}
}

// streamsMock answers stream subscription requests as Binance.US does, with a null result, or with an error object for
// streams named "fail"; one for streams named "noresult" is answered without a result, and one for streams named
// "silent" is not answered
func streamsMock(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req WebsocketPayloadRequest
	require.NoError(tb, json.Unmarshal(msg, &req), "request must decode")
	result := "null"
	for _, p := range req.Params {
		switch {
		case strings.Contains(p, "fail"):
			result = `{"code":2,"msg":"Invalid request"}`
		case strings.Contains(p, "noresult"):
			return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"id":%q}`, req.ID))
		case strings.Contains(p, "silent"):
			return nil
		}
	}
	return w.WriteMessage(gws.TextMessage, fmt.Appendf(nil, `{"result":%s,"id":%q}`, result, req.ID))
}

func TestWebsocketManagerSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	exch := &config.Exchange{Name: "test"}
	setup := ex.websocketManagerSetup(exch)
	assert.Same(t, exch, setup.ExchangeConfig, "ExchangeConfig should be the exchange's configuration")
	assert.Same(t, &ex.Features.Supports.WebsocketCapabilities, setup.Features, "Features should be the exchange's websocket capabilities")
	assert.True(t, setup.UseMultiConnectionManagement, "UseMultiConnectionManagement should be set")
	assert.Equal(t, maxStreamsPerConnection, setup.MaxWebsocketSubscriptionsPerConnection, "MaxWebsocketSubscriptionsPerConnection should be the most streams a connection carries")
	assert.True(t, setup.InvalidateOrderbooksOnShutdown, "InvalidateOrderbooksOnShutdown should be set, as UpdateOrderbook defers to the books the streams load")

	t.Run("shutdown invalidates the books the streams loaded", func(t *testing.T) {
		t.Parallel()
		ex, _ := newSubscriptionTestExchange(t)
		p := currency.NewBTCUSDT()
		require.NoError(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{Exchange: ex.Name, Pair: p, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now()}), "LoadSnapshot must not error")
		require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
		_, err := orderbook.Get(ex.Name, p, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "Shutdown should invalidate a book the streams loaded")
	})
}

func TestStreamConnectionSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	setup := ex.streamConnectionSetup(&config.Exchange{WebsocketResponseCheckTimeout: time.Second, WebsocketResponseMaxLimit: 2 * time.Second})
	assert.Equal(t, websocketStreamsURL, setup.URL, "URL should be the combined streams endpoint")
	assert.Equal(t, asset.Spot, setup.MessageFilter, "MessageFilter should be spot")
	assert.Equal(t, time.Second, setup.ResponseCheckTimeout, "ResponseCheckTimeout should be the configured one")
	assert.Equal(t, 2*time.Second, setup.ResponseMaxLimit, "ResponseMaxLimit should be the configured one")
	assert.NotNil(t, setup.RateLimit, "RateLimit should hold the connection to its message rate")
	for name, f := range map[string]any{"Connector": setup.Connector, "Subscriber": setup.Subscriber, "Unsubscriber": setup.Unsubscriber, "Handler": setup.Handler, "GenerateSubscriptions": setup.GenerateSubscriptions} {
		assert.NotNilf(t, f, "%s should be set", name)
	}
}

func TestNewOrderbookSync(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	assert.NotNil(t, ex.newOrderbookSync(ex.fetchOrderbookSnapshot, time.Second, time.Minute), "newOrderbookSync should return the synchronisation")
}

func TestWebsocketConnect(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the market data streams from a local server")
		}
		ex, _ := newLiveStreamTestExchange(t)
		assert.True(t, ex.Websocket.IsConnected(), "the market data streams should connect")
	})

	ex := newWebsocketTestExchange(t)
	ex.Features.Subscriptions = subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}
	setup := ex.streamConnectionSetup(ex.Config)
	useMockWebsocket(t, ex, setup, streamsMock)
	generation := ex.orderbookStreamGeneration.Load()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	assert.True(t, ex.Websocket.IsConnected(), "the market data streams should connect")
	assert.Len(t, ex.Websocket.GetSubscriptions(), 1, "the enabled pair's ticker should be subscribed")
	assert.Equal(t, generation+1, ex.orderbookStreamGeneration.Load(), "WebsocketConnect should count the connection in orderbookStreamGeneration")

	err := ex.WebsocketConnect(t.Context(), &fakeDialConnection{fakeConnection: newFakeConnection(), dialErr: errDialFailed})
	assert.ErrorIs(t, err, errDialFailed, "WebsocketConnect should return a failed dial")
}

func TestWebsocketHandleData(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	conn := newFakeConnection()
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"result":null,"id":"7"}`)), "a subscription response must not error")
	assert.Contains(t, conn.matched, "7", "a subscription response should be matched to its request")

	require.NoError(t, ex.wsHandleData(t.Context(), conn, streamFramesOf(t, "btcusdt@trade")[0]), "a trade frame must not error")
	_, ok := relayed(t, ex).([]trade.Data)
	assert.True(t, ok, "a trade frame should relay trades")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, streamFramesOf(t, "btcusdt@ticker")[0]), "a ticker frame must not error")
	_, ok = relayed(t, ex).(*ticker.Price)
	assert.True(t, ok, "a ticker frame should relay a ticker")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, streamFramesOf(t, "btcusdt@kline_1m")[0]), "a kline frame must not error")
	_, ok = relayed(t, ex).(kline.Item)
	assert.True(t, ok, "a kline frame should relay candles")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, streamFramesOf(t, "btcusdt@depth5@100ms")[0]), "a partial depth frame must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "a partial depth frame must load the book")
	assert.Equal(t, int64(3417876290), book.LastUpdateID, "a partial depth frame should load its update ID")
	diff := streamFramesOf(t, "btcusdt@depth@100ms")[1]
	require.NoError(t, ex.wsHandleData(t.Context(), conn, diff), "a diff depth frame must not error")
	updateID, err := jsonparser.GetInt(diff, "data", "u")
	require.NoError(t, err, "the diff depth frame must carry its last update ID")
	assert.Equal(t, updateID, ex.depthUpdateIDs.last(key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}), "a diff depth frame should be passed to the order book synchronisation")

	for name, frame := range map[string]string{
		"no stream":        `{"data":{}}`,
		"no data":          `{"stream":"btcusdt@trade"}`,
		"unknown stream":   `{"stream":"btcusdt@bookTicker","data":{}}`,
		"not an object":    `[]`,
		"unknown channels": `{"stream":"btcusdt@aggTrade","data":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIsf(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), errUnhandledMessage, "a frame with %s should be unhandled", name)
		})
	}

	for name, frame := range map[string]string{
		"raw":      `{"e":"serverShutdown","E":1794466395877}`,
		"combined": `{"stream":"!serverShutdown","data":{"e":"serverShutdown","E":1794466395877}}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			shutdown := newStreamTestExchange(t)
			require.NoErrorf(t, shutdown.wsHandleData(t.Context(), newFakeConnection(), []byte(frame)), "a %s serverShutdown event must not error", name)
		})
	}
}

func TestHandleStreamsServerShutdown(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	apiConn := newFakeConnection()
	ex.wsAPI.set(apiConn, 10*time.Millisecond)
	down := newFakeConnection()
	down.connected = false
	require.NoError(t, ex.handleStreamsServerShutdown(down, []byte(`{"e":"serverShutdown","E":1794466395877}`)), "handleStreamsServerShutdown must not error")
	assert.Nil(t, ex.wsAPI.get(), "the stream server's shutdown should retire the WebSocket API connection, which the websocket's shutdown closes")
	assert.Error(t, ex.handleStreamsServerShutdown(down, []byte(`{"E":"x"}`)), "handleStreamsServerShutdown should error for an event that does not decode")
}

func TestShutdownWebsocketFor(t *testing.T) {
	t.Parallel()
	ex, conn := newSubscriptionTestExchange(t)
	down := newFakeConnection()
	down.connected = false
	require.NoError(t, ex.shutdownWebsocketFor(down)(), "shutdownWebsocketFor must not error for a closed connection")
	assert.True(t, ex.Websocket.IsConnected(), "shutdownWebsocketFor should leave a websocket that closed the connection alone")
	require.NoError(t, ex.shutdownWebsocketFor(conn)(), "shutdownWebsocketFor must not error")
	assert.False(t, ex.Websocket.IsConnected(), "shutdownWebsocketFor should shut the websocket down for the connection monitor to connect it again")

	t.Run("a connection's close left until after the websocket reconnected leaves the new connection connected", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketAPITestExchange(t, false)
		old := ex.wsAPI.get()
		closeOld := ex.shutdownWebsocketFor(old)
		require.NoError(t, ex.Websocket.Shutdown(), "the websocket must shut down")
		require.NoError(t, ex.Websocket.Connect(t.Context()), "the websocket must connect again")
		replacement := ex.wsAPI.get()
		require.NotSame(t, old, replacement, "connecting again must give a new WebSocket API connection")
		require.NoError(t, closeOld(), "the old connection's close must not error")
		assert.True(t, isConnected(replacement), "the old connection's close should leave the new connection connected")
	})
}

func TestHandleServerShutdown(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	apiConn := newFakeConnection()
	ex.wsAPI.set(apiConn, time.Hour)
	_, release := ex.wsAPI.acquire()
	closed := make(chan struct{})
	require.NoError(t, ex.handleServerShutdown([]byte(`{"e":"serverShutdown","E":1794466395877}`), apiConn, func() error {
		close(closed)
		return nil
	}), "handleServerShutdown must not error")
	assert.Nil(t, ex.wsAPI.get(), "handleServerShutdown should stop the connection taking requests")
	select {
	case <-closed:
		require.Fail(t, "handleServerShutdown must wait for the request in flight before closing")
	case <-time.After(50 * time.Millisecond):
	}
	release()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		require.Fail(t, "handleServerShutdown must close the connection once the request in flight returns")
	}
}

func TestProcessTrade(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	require.NoError(t, ex.processTrade(streamData(t, streamFramesOf(t, "btcusdt@trade")[0])), "processTrade must not error")
	trades, ok := relayed(t, ex).([]trade.Data)
	require.True(t, ok, "processTrade must relay trades")
	require.Len(t, trades, 1, "processTrade must relay the trade")
	assert.Equal(t, "32183726", trades[0].TID, "processTrade should relay the trade ID")
	assert.True(t, trades[0].CurrencyPair.Equal(currency.NewBTCUSDT()), "processTrade should relay the stored pair")
	assert.NotZero(t, trades[0].Price, "processTrade should relay the price")

	require.NoError(t, ex.processTrade([]byte(`{"s":"ETHUSDT","t":1}`)), "processTrade must not error for a pair that is not enabled")
	assert.Empty(t, ex.Websocket.DataHandler.C, "a pair that is not enabled should relay nothing")
	assert.ErrorIs(t, ex.processTrade([]byte(`{"s":"NOTAPAIR","t":1}`)), currency.ErrPairNotFound, "processTrade should error for an unknown symbol")
	assert.Error(t, ex.processTrade([]byte(`{"t":"x"}`)), "processTrade should error for a payload that does not decode")

	require.NoError(t, ex.processTrade([]byte(`{"e":"trade","E":1794465804511,"s":"BTCUSDT","t":32183727,"p":"73564.09","q":"0.001","T":1794465804510,"m":true}`)), "processTrade must not error for a trade the buyer made")
	trades, ok = relayed(t, ex).([]trade.Data)
	require.True(t, ok, "processTrade must relay trades")
	require.Len(t, trades, 1, "processTrade must relay the trade")
	assert.Equal(t, order.Sell, trades[0].Side, "a trade whose buyer made the market should be a sell")

	ex.Features.Enabled.TradeFeed = false
	require.NoError(t, ex.processTrade([]byte(`{"t":"x"}`)), "processTrade must ignore trades without a trade feed or trade saving")
}

func TestProcessTicker(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	require.NoError(t, ex.processTicker(t.Context(), streamData(t, streamFramesOf(t, "btcusdt@ticker")[0])), "processTicker must not error")
	tick, ok := relayed(t, ex).(*ticker.Price)
	require.True(t, ok, "processTicker must relay a ticker")
	assert.Equal(t, 73711.61, tick.Close, "processTicker should report the previous close as the close, as the REST ticker does")
	stored, err := ticker.GetTicker(ex.Name, currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "processTicker must store the ticker before relaying it")
	assert.Equal(t, tick.Last, stored.Last, "processTicker should store the ticker it relays")

	require.NoError(t, ex.processTicker(t.Context(), []byte(`{"s":"ETHUSDT"}`)), "processTicker must not error for a pair that is not enabled")
	assert.Empty(t, ex.Websocket.DataHandler.C, "a pair that is not enabled should relay nothing")
	assert.Error(t, ex.processTicker(t.Context(), []byte(`{"s":1}`)), "processTicker should error for a payload that does not decode")
}

func TestProcessKline(t *testing.T) {
	t.Parallel()
	frames := streamFramesOf(t, "btcusdt@kline_1m")
	require.Len(t, frames, 12, "the fixture must hold a closed candle and the open one after it, then ten minute candles of a later recording")
	for i, tc := range []struct {
		name    string
		partial string
	}{
		{name: "closed candle"},
		{name: "open candle", partial: kline.PartialCandle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newStreamTestExchange(t)
			require.NoErrorf(t, ex.processKline(t.Context(), streamData(t, frames[i])), "processKline must not error for frame %d", i)
			item, ok := relayed(t, ex).(kline.Item)
			require.Truef(t, ok, "processKline must relay candles for frame %d", i)
			require.Lenf(t, item.Candles, 1, "processKline must relay one candle for frame %d", i)
			assert.Equalf(t, tc.partial, item.Candles[0].ValidationIssues, "processKline should flag an open candle as partial for frame %d", i)
			assert.Equalf(t, kline.OneMin, item.Interval, "processKline should relay the interval for frame %d", i)
			assert.Emptyf(t, ex.Websocket.DataHandler.C, "processKline should relay nothing more for frame %d", i)
		})
	}
	t.Run("the recorded frames in turn on one exchange", func(t *testing.T) {
		t.Parallel()
		ex := newStreamTestExchange(t)
		require.NoError(t, ex.processKline(t.Context(), streamData(t, frames[0])), "processKline must not error for the closed candle")
		closed, ok := relayed(t, ex).(kline.Item)
		require.True(t, ok, "processKline must relay the closed candle")
		closedCandles := slices.Clone(closed.Candles)
		require.NoError(t, ex.processKline(t.Context(), streamData(t, frames[1])), "processKline must not error for the open candle after the closed one")
		open, ok := relayed(t, ex).(kline.Item)
		require.True(t, ok, "processKline must relay the open candle after the closed one")
		require.Len(t, open.Candles, 1, "processKline must relay one candle for the open candle after the closed one")
		assert.Equal(t, time.UnixMilli(1794465840000), open.Candles[0].Time, "the open candle after the closed one should be relayed with its own start")
		assert.Equal(t, kline.PartialCandle, open.Candles[0].ValidationIssues, "the open candle after the closed one should be flagged as partial")
		assert.Equal(t, closedCandles, closed.Candles, "the closed candle relayed first should be left as it was relayed")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processKline should relay each frame once")
		assert.ErrorIs(t, ex.processKline(t.Context(), []byte(`{"s":"BTCUSDT","k":{"i":"1s"}}`)), kline.ErrInvalidInterval, "processKline should error for an unknown interval after the recorded frames")
	})
	t.Run("candles of one pair at several intervals and of two pairs at one interval in turn on one exchange", func(t *testing.T) {
		t.Parallel()
		ex := newStreamTestExchange(t)
		ethUSDT := currency.NewPair(currency.ETH, currency.USDT)
		ethBTC := currency.NewPair(currency.ETH, currency.BTC)
		pairs := currency.Pairs{currency.NewBTCUSDT(), ethUSDT, currency.NewBTCUSD(), ethBTC}
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs, false), "StorePairs must not error")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, pairs, true), "StorePairs must not error")
		fiveMinutes := streamFramesOf(t, "btcusdt@kline_5m")
		etherMinutes := streamFramesOf(t, "ethusdt@kline_1m")
		dollarMinutes := streamFramesOf(t, "btcusd@kline_1m")
		var names []string
		var relays, expected []kline.Item
		relay := func(name string, frame []byte, pair currency.Pair, interval kline.Interval, candle kline.Candle) {
			t.Helper()
			require.NoErrorf(t, ex.processKline(t.Context(), streamData(t, frame)), "processKline must not error for the %s", name)
			item, ok := relayed(t, ex).(kline.Item)
			require.Truef(t, ok, "processKline must relay the %s", name)
			exp := kline.Item{Exchange: ex.Name, Pair: pair, Asset: asset.Spot, Interval: interval, Candles: []kline.Candle{candle}}
			assert.Equalf(t, exp, item, "the %s should be relayed as recorded", name)
			names, relays, expected = append(names, name), append(relays, item), append(expected, exp)
		}
		minute := kline.Candle{Time: time.UnixMilli(1795081020000), Open: 63879.99, Close: 63867.04, High: 63886, Low: 63867.04, Volume: 0.021, ValidationIssues: kline.PartialCandle}
		relay("open minute candle", frames[2], currency.NewBTCUSDT(), kline.OneMin, minute)
		relay("open five minute candle", fiveMinutes[0], currency.NewBTCUSDT(), kline.FiveMin, kline.Candle{
			Time: time.UnixMilli(1795080900000), Open: 63859.81, Close: 63867.04, High: 63886, Low: 63854.09, Volume: 0.03938, ValidationIssues: kline.PartialCandle,
		})
		minute.ValidationIssues = ""
		relay("open minute candle's close", frames[3], currency.NewBTCUSDT(), kline.OneMin, minute)
		relay("close of ETHUSDT's minute without trades", etherMinutes[0], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081020000), Open: 1806.56, Close: 1806.56, High: 1806.56, Low: 1806.56,
		})
		relay("next minute candle before its first trade", frames[4], currency.NewBTCUSDT(), kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081080000), Open: 63867.04, Close: 63867.04, High: 63867.04, Low: 63867.04, ValidationIssues: kline.PartialCandle,
		})
		relay("ETHUSDT's next minute candle before its first trade", etherMinutes[1], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081080000), Open: 1806.56, Close: 1806.56, High: 1806.56, Low: 1806.56, ValidationIssues: kline.PartialCandle,
		})
		relay("next minute candle after its first trades", frames[5], currency.NewBTCUSDT(), kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081080000), Open: 63867.81, Close: 63867.88, High: 63867.88, Low: 63867.81, Volume: 0.00236, ValidationIssues: kline.PartialCandle,
		})
		relay("five minute candle after more trades", fiveMinutes[1], currency.NewBTCUSDT(), kline.FiveMin, kline.Candle{
			Time: time.UnixMilli(1795080900000), Open: 63859.81, Close: 63867.88, High: 63886, Low: 63854.09, Volume: 0.04174, ValidationIssues: kline.PartialCandle,
		})
		relay("next minute candle after more trades", frames[6], currency.NewBTCUSDT(), kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081080000), Open: 63867.81, Close: 63880.18, High: 63880.18, Low: 63867.81, Volume: 0.00472, ValidationIssues: kline.PartialCandle,
		})
		relay("ETHUSDT's later minute candle before its first trade", etherMinutes[2], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081140000), Open: 1806.88, Close: 1806.88, High: 1806.88, Low: 1806.88, ValidationIssues: kline.PartialCandle,
		})
		relay("ETHUSDT's later minute candle after its first trades, lower than the price before them", etherMinutes[3], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081140000), Open: 1806.61, Close: 1806.56, High: 1806.61, Low: 1806.56, Volume: 0.1192, ValidationIssues: kline.PartialCandle,
		})
		relay("close of a later minute candle", frames[7], currency.NewBTCUSDT(), kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081140000), Open: 63855.66, Close: 63832.43, High: 63864.57, Low: 63814.98, Volume: 0.03628,
		})
		relay("five minute candle's close", fiveMinutes[2], currency.NewBTCUSDT(), kline.FiveMin, kline.Candle{
			Time: time.UnixMilli(1795080900000), Open: 63859.81, Close: 63832.43, High: 63886.4, Low: 63814.98, Volume: 0.08878,
		})
		relay("close of ETHUSDT's later minute candle", etherMinutes[4], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081140000), Open: 1806.61, Close: 1805.97, High: 1806.61, Low: 1805.97, Volume: 0.5168,
		})
		opening := kline.Candle{Time: time.UnixMilli(1795081200000), Open: 63832.43, Close: 63832.43, High: 63832.43, Low: 63832.43, ValidationIssues: kline.PartialCandle}
		relay("minute candle opening with the next five minute candle", frames[8], currency.NewBTCUSDT(), kline.OneMin, opening)
		relay("five minute candle opening with the minute candle", fiveMinutes[3], currency.NewBTCUSDT(), kline.FiveMin, opening)
		relay("ETHUSDT's minute candle opening with them", etherMinutes[5], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081200000), Open: 1805.97, Close: 1805.97, High: 1805.97, Low: 1805.97, ValidationIssues: kline.PartialCandle,
		})
		relay("ETHUSDT's minute candle after its first trade", etherMinutes[6], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081200000), Open: 1806.1, Close: 1806.1, High: 1806.1, Low: 1806.1, Volume: 0.02, ValidationIssues: kline.PartialCandle,
		})
		require.NoError(t, ex.CurrencyPairs.DisablePair(asset.Spot, ethUSDT), "DisablePair must not error")
		require.NoError(t, ex.processKline(t.Context(), streamData(t, etherMinutes[7])), "processKline must not error for a pair disabled after its earlier frames")
		assert.Empty(t, ex.Websocket.DataHandler.C, "processKline should relay nothing for a pair disabled after its earlier frames")
		require.NoError(t, ex.CurrencyPairs.EnablePair(asset.Spot, ethUSDT), "EnablePair must not error")
		relay("ETHUSDT's minute candle once the pair is enabled again", etherMinutes[8], ethUSDT, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795081260000), Open: 1807.08, Close: 1807.65, High: 1807.65, Low: 1807.08, Volume: 0.0895, ValidationIssues: kline.PartialCandle,
		})
		lastMinute := kline.Candle{Time: time.UnixMilli(1795081440000), Open: 63897.57, Close: 63897.57, High: 63897.57, Low: 63897.57, ValidationIssues: kline.PartialCandle}
		relay("last minute candle before its first trade", frames[9], currency.NewBTCUSDT(), kline.OneMin, lastMinute)
		lastMinute.Volume = 0.00117
		relay("last minute candle after a first trade at the same price", frames[10], currency.NewBTCUSDT(), kline.OneMin, lastMinute)
		lastMinute.Volume = 0.01116
		relay("last minute candle after more trades at the same price", frames[11], currency.NewBTCUSDT(), kline.OneMin, lastMinute)
		relay("weekly candle", streamFramesOf(t, "btcusdt@kline_1w")[0], currency.NewBTCUSDT(), kline.OneWeek, kline.Candle{
			Time: time.UnixMilli(1794787200000), Open: 65350.8, Close: 63820.99, High: 65711.35, Low: 62071.04, Volume: 256.30226, ValidationIssues: kline.PartialCandle,
		})
		relay("monthly candle", streamFramesOf(t, "btcusdt@kline_1M")[0], currency.NewBTCUSDT(), kline.OneMonth, kline.Candle{
			Time: time.UnixMilli(1793491200000), Open: 77565.14, Close: 63820.99, High: 81195.76, Low: 62026.59, Volume: 1182.37844, ValidationIssues: kline.PartialCandle,
		})
		relay("three day candle", streamFramesOf(t, "btcusdt@kline_3d")[0], currency.NewBTCUSDT(), kline.ThreeDay, kline.Candle{
			Time: time.UnixMilli(1794873600000), Open: 64898.05, Close: 63848.4, High: 65423.28, Low: 62071.04, Volume: 196.80223, ValidationIssues: kline.PartialCandle,
		})
		quiet := kline.Candle{Time: time.UnixMilli(1795085880000), Open: 63771.5, Close: 63771.5, High: 63771.5, Low: 63771.5}
		relay("close of BTCUSD's minute without trades", dollarMinutes[0], currency.NewBTCUSD(), kline.OneMin, quiet)
		quiet.Time, quiet.ValidationIssues = time.UnixMilli(1795085940000), kline.PartialCandle
		relay("BTCUSD's next minute candle", dollarMinutes[1], currency.NewBTCUSD(), kline.OneMin, quiet)
		quiet.ValidationIssues = ""
		relay("close of BTCUSD's next minute, without trades too", dollarMinutes[2], currency.NewBTCUSD(), kline.OneMin, quiet)
		relay("close of ETHBTC's minute without trades", streamFramesOf(t, "ethbtc@kline_1m")[0], ethBTC, kline.OneMin, kline.Candle{
			Time: time.UnixMilli(1795088220000), Open: 0.02832, Close: 0.02832, High: 0.02832, Low: 0.02832,
		})
		for i, item := range relays {
			assert.Equalf(t, expected[i], item, "the %s should be left as it was relayed after the later frames", names[i])
		}
		assert.Empty(t, ex.Websocket.DataHandler.C, "processKline should relay each frame once")
	})
	t.Run("exchanges sharing a name each relay a frame both receive", func(t *testing.T) {
		t.Parallel()
		first, second := newStreamTestExchange(t), newStreamTestExchange(t)
		require.Equal(t, first.Name, second.Name, "both exchanges must share a name")
		relay := func(which string, ex *Exchange) {
			t.Helper()
			require.NoErrorf(t, ex.processKline(t.Context(), streamData(t, frames[2])), "processKline must not error on the %s exchange", which)
			item, ok := relayed(t, ex).(kline.Item)
			require.Truef(t, ok, "processKline must relay the frame on the %s exchange", which)
			require.Lenf(t, item.Candles, 1, "processKline must relay one candle on the %s exchange", which)
			assert.Equalf(t, time.UnixMilli(1795081020000), item.Candles[0].Time, "the %s exchange should relay the frame's candle", which)
		}
		relay("first", first)
		relay("second", second)
	})
	ex := newStreamTestExchange(t)
	require.NoError(t, ex.processKline(t.Context(), []byte(`{"s":"ETHUSDT","k":{"i":"1m"}}`)), "processKline must not error for a pair that is not enabled")
	assert.ErrorIs(t, ex.processKline(t.Context(), []byte(`{"s":"BTCUSDT","k":{"i":"1s"}}`)), kline.ErrInvalidInterval, "processKline should error for an unknown interval")
	assert.Error(t, ex.processKline(t.Context(), []byte(`{"s":1}`)), "processKline should error for a payload that does not decode")
}

func TestProcessDepthUpdate(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	for i, frame := range streamFramesOf(t, "btcusdt@depth@100ms") {
		require.NoErrorf(t, ex.processDepthUpdate(t.Context(), streamData(t, frame)), "processDepthUpdate must not error for frame %d", i)
	}
	assert.Eventually(t, func() bool {
		id, err := ex.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
		return err == nil && id == 3417876293
	}, 5*time.Second, 10*time.Millisecond, "the diff events following the snapshot should bring the book to the last event's update ID")
	require.NoError(t, ex.processDepthUpdate(t.Context(), []byte(`{"e":"depthUpdate","E":1794465804911,"s":"BTCUSDT","U":3417876294,"u":3417876295,"b":[],"a":[]}`)), "an event without level changes must not error")
	assert.Eventually(t, func() bool {
		id, err := ex.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
		return err == nil && id == 3417876295
	}, 5*time.Second, 10*time.Millisecond, "an event without level changes should still move the book's update ID on")

	require.NoError(t, ex.processDepthUpdate(t.Context(), []byte(`{"s":"ETHUSDT","U":1,"u":1}`)), "processDepthUpdate must not error for a pair that is not enabled")
	assert.Error(t, ex.processDepthUpdate(t.Context(), []byte(`{"s":1}`)), "processDepthUpdate should error for a payload that does not decode")
}

func TestProcessOrderbookUpdate(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	ex.orderbookSync = nil
	err := ex.processOrderbookUpdate(t.Context(), 1, &orderbook.Update{Pair: currency.NewBTCUSDT(), Asset: asset.Spot, UpdateID: 1})
	require.ErrorIs(t, err, errOrderbookSyncNotSetUp, "processOrderbookUpdate must error without a synchronisation")

	ex.orderbookSync = ex.newOrderbookSync(func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
		return &orderbook.Book{
			Exchange: ex.Name, Pair: p, Asset: a, LastUpdateID: 100, LastUpdated: time.Now(),
			Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}},
		}, nil
	}, 0, 30*time.Second)
	lastUpdateID := func() int64 {
		id, err := ex.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
		if err != nil {
			return 0
		}
		return id
	}
	update := func(updateID int64) *orderbook.Update {
		return &orderbook.Update{Pair: currency.NewBTCUSDT(), Asset: asset.Spot, UpdateID: updateID, UpdateTime: time.Now(), AllowEmpty: true}
	}
	require.NoError(t, ex.processOrderbookUpdate(t.Context(), 101, update(101)), "the event following the snapshot must not error")
	require.Eventually(t, func() bool { return lastUpdateID() == 101 }, 5*time.Second, 10*time.Millisecond, "the event following the snapshot must be applied")
	require.NoError(t, ex.processOrderbookUpdate(t.Context(), 90, update(95)), "a late event must not error")
	assert.Equal(t, int64(101), lastUpdateID(), "an event the book has moved past should be dropped")
	require.NoError(t, ex.processOrderbookUpdate(t.Context(), 99, update(103)), "an overlapping event must not error")
	assert.Eventually(t, func() bool { return lastUpdateID() == 103 }, 5*time.Second, 10*time.Millisecond, "an event overlapping the book should be applied from the book's next update")

	t.Run("events overlapping each other while a snapshot is fetched", func(t *testing.T) {
		t.Parallel()
		ex := newStreamTestExchange(t)
		release := make(chan struct{})
		var fetches atomic.Int32
		ex.orderbookSync = ex.newOrderbookSync(func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
			fetches.Add(1)
			<-release
			return &orderbook.Book{
				Exchange: ex.Name, Pair: p, Asset: a, LastUpdateID: 103, LastUpdated: time.Now(),
				Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}},
			}, nil
		}, 0, 30*time.Second)
		// Events overlap the ones before them while the snapshot is fetched, as the local order book procedure allows
		for _, ids := range [][2]int64{{101, 103}, {103, 106}, {105, 108}, {106, 108}} {
			update := &orderbook.Update{Pair: currency.NewBTCUSDT(), Asset: asset.Spot, UpdateID: ids[1], UpdateTime: time.Now(), AllowEmpty: true}
			require.NoErrorf(t, ex.processOrderbookUpdate(t.Context(), ids[0], update), "processOrderbookUpdate must not error for event %d to %d", ids[0], ids[1])
		}
		close(release)
		assert.Eventually(t, func() bool {
			id, err := ex.Websocket.Orderbook.LastUpdateID(currency.NewBTCUSDT(), asset.Spot)
			return err == nil && id == 108
		}, 5*time.Second, 10*time.Millisecond, "events overlapping each other while a snapshot is fetched should bring the book to the last event's update ID")
		assert.Equal(t, int32(1), fetches.Load(), "events overlapping each other should not need another snapshot")
	})
}

func TestDepthUpdateIDsLast(t *testing.T) {
	t.Parallel()
	var d depthUpdateIDs
	k := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}
	assert.Zero(t, d.last(k), "last should be 0 before any event")
	d.record(k, 5)
	assert.Equal(t, int64(5), d.last(k), "last should return the recorded update ID")
}

func TestDepthUpdateIDsRecord(t *testing.T) {
	t.Parallel()
	var d depthUpdateIDs
	k := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}
	d.record(k, 7)
	d.record(k, 6)
	assert.Equal(t, int64(7), d.last(k), "record should keep the latest update ID")
}

func TestFetchOrderbookSnapshot(t *testing.T) {
	t.Parallel()
	p := currency.NewBTCUSDT()
	stream := func() *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{p}}
	}
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the order book snapshots from a local server")
		}
		live := newWebsocketTestExchange(t)
		require.NoError(t, live.Websocket.AddSuccessfulSubscriptions(nil, stream()), "AddSuccessfulSubscriptions must not error")
		book, err := live.fetchOrderbookSnapshot(t.Context(), p, asset.Spot)
		require.NoError(t, err, "fetchOrderbookSnapshot must not error")
		assert.Positive(t, book.LastUpdateID, "fetchOrderbookSnapshot should return the snapshot's update ID")
		assert.NotEmpty(t, book.Bids, "fetchOrderbookSnapshot should return the bids")
	})

	ex := routedTestExchange(t, map[string]http.HandlerFunc{
		"GET /api/v3/depth": reply(t, url.Values{"symbol": {"BTCUSDT"}, "limit": {"1000"}}, `{"lastUpdateId":42,"bids":[["1","2"]],"asks":[["3","4"]]}`),
	})
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, stream()), "AddSuccessfulSubscriptions must not error")
	book, err := ex.fetchOrderbookSnapshot(t.Context(), p, asset.Spot)
	require.NoError(t, err, "fetchOrderbookSnapshot must not error")
	assert.Equal(t, int64(42), book.LastUpdateID, "fetchOrderbookSnapshot should return the snapshot's update ID")
	assert.Equal(t, orderbook.Levels{{Price: 3, Amount: 4}}, book.Asks, "fetchOrderbookSnapshot should return the asks")
	assert.NotZero(t, book.LastUpdated, "fetchOrderbookSnapshot should stamp the snapshot")

	failing := routedTestExchange(t, map[string]http.HandlerFunc{"GET /api/v3/depth": replyError(t, -1121, "Invalid symbol.")})
	_, err = failing.fetchOrderbookSnapshot(t.Context(), p, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrPairNotFound, "fetchOrderbookSnapshot should return a failed request")

	unstreamed := newWebsocketTestExchange(t)
	require.NoError(t, unstreamed.SetHTTPClient(orderBookClient(42, nil)), "SetHTTPClient must not error")
	_, err = unstreamed.fetchOrderbookSnapshot(t.Context(), p, asset.Spot)
	assert.ErrorIs(t, err, errOrderbookStreamStopped, "fetchOrderbookSnapshot should discard a snapshot for a pair no subscribed order book stream maintains")

	restarted := newWebsocketTestExchange(t)
	require.NoError(t, restarted.Websocket.AddSuccessfulSubscriptions(nil, stream()), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, restarted.SetHTTPClient(orderBookClient(42, func() {
		// The market data streams connect again while the snapshot is fetched
		restarted.orderbookStreamGeneration.Add(1)
	})), "SetHTTPClient must not error")
	_, err = restarted.fetchOrderbookSnapshot(t.Context(), p, asset.Spot)
	assert.ErrorIs(t, err, errOrderbookStreamStopped, "fetchOrderbookSnapshot should discard a snapshot fetched while the streams connected again")

	subscribing := newWebsocketTestExchange(t)
	require.NoError(t, subscribing.Websocket.AddSubscriptions(nil, stream()), "AddSubscriptions must not error")
	require.NoError(t, subscribing.SetHTTPClient(orderBookClient(42, nil)), "SetHTTPClient must not error")
	book, err = subscribing.fetchOrderbookSnapshot(t.Context(), p, asset.Spot)
	require.NoError(t, err, "fetchOrderbookSnapshot must return the snapshot for a stream whose subscription is not yet answered")
	assert.Equal(t, int64(42), book.LastUpdateID, "fetchOrderbookSnapshot should return the snapshot for a stream subscribing")
}

func TestCheckPendingUpdate(t *testing.T) {
	t.Parallel()
	update := &orderbook.Update{Pair: currency.NewBTCUSDT(), Asset: asset.Spot, UpdateID: 100}
	skip, err := checkPendingUpdate(100, 90, update)
	require.NoError(t, err, "checkPendingUpdate must not error for an event the snapshot covers")
	assert.True(t, skip, "an event the snapshot covers should be skipped")
	update.UpdateID = 105
	skip, err = checkPendingUpdate(100, 101, update)
	require.NoError(t, err, "checkPendingUpdate must not error for the event following the snapshot")
	assert.False(t, skip, "the event following the snapshot should be applied")
	skip, err = checkPendingUpdate(100, 95, update)
	require.NoError(t, err, "checkPendingUpdate must not error for an event spanning the snapshot")
	assert.False(t, skip, "an event spanning the snapshot should be applied")
	_, err = checkPendingUpdate(100, 102, update)
	assert.ErrorIs(t, err, orderbookmanager.ErrOrderbookSnapshotOutdated, "an event starting after the snapshot's next update should need a new snapshot")
}

func TestProcessPartialDepth(t *testing.T) {
	t.Parallel()
	ex := newStreamTestExchange(t)
	require.NoError(t, ex.processPartialDepth(t.Context(), "btcusdt", streamData(t, streamFramesOf(t, "btcusdt@depth5@100ms")[0])), "processPartialDepth must not error")
	book, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err, "processPartialDepth must load the book")
	assert.Len(t, book.Bids, 5, "processPartialDepth should load every level")
	require.NoError(t, ex.processPartialDepth(t.Context(), "ethusdt", []byte(`{}`)), "processPartialDepth must not error for a pair that is not enabled")
	assert.ErrorIs(t, ex.processPartialDepth(t.Context(), "notapair", []byte(`{}`)), currency.ErrPairNotFound, "processPartialDepth should error for an unknown symbol")
	assert.Error(t, ex.processPartialDepth(t.Context(), "btcusdt", []byte(`{"bids":1}`)), "processPartialDepth should error for a payload that does not decode")
}

var orderStatuses = map[string]order.Status{
	"NEW": order.New, "PENDING_NEW": order.Pending, "PARTIALLY_FILLED": order.PartiallyFilled, "FILLED": order.Filled,
	"CANCELED": order.Cancelled, "PENDING_CANCEL": order.PendingCancel, "REJECTED": order.Rejected,
	"EXPIRED": order.Expired, "EXPIRED_IN_MATCH": order.Expired,
}

func TestStringToOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range orderStatuses {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			got, err := stringToOrderStatus(status)
			require.NoErrorf(t, err, "stringToOrderStatus must not error for %s", status)
			assert.Equalf(t, exp, got, "stringToOrderStatus should convert %s", status)
		})
	}
	_, err := stringToOrderStatus("UNKNOWN")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "stringToOrderStatus should reject an unknown status")
}

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Features.Subscriptions = subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.Futures, Channel: subscription.TickerChannel},
	}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	names := make([]string, len(subs))
	for i, s := range subs {
		names[i] = s.QualifiedChannel
	}
	assert.ElementsMatch(t, []string{"btcusdt@ticker", "btcusdt@kline_1m", "btcusdt@depth@100ms"}, names, "generateSubscriptions should expand each spot subscription once for the enabled pairs")

	ex.Features.Subscriptions = append(ex.Features.Subscriptions, &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 5})
	_, err = ex.generateSubscriptions()
	assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateSubscriptions should report an order book subscription left out as partial generation")
	assert.ErrorIs(t, err, errOrderbookSubscriptionConflict, "generateSubscriptions should say why a subscription was left out")

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error without spot enabled")
	assert.Empty(t, subs, "generateSubscriptions should generate nothing without spot enabled")
}

func TestDistinctSubscriptions(t *testing.T) {
	t.Parallel()
	btc := currency.NewBTCUSDT()
	subs := subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{btc}},
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPairWithDelimiter("btc", "usdt", "-")}},
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}},
	}
	assert.Equal(t, subscription.List{subs[0], subs[2]}, distinctSubscriptions(subs), "distinctSubscriptions should keep the first subscription to each stream, whatever its pairs' case or delimiter")
}

func TestOneOrderbookSubscription(t *testing.T) {
	t.Parallel()
	btc := currency.Pairs{currency.NewBTCUSDT()}
	partial := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btc, Levels: 5}
	diff := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: btc}
	tickerSub := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: btc}
	kept, err := oneOrderbookSubscription(subscription.List{partial, diff, tickerSub})
	assert.ErrorIs(t, err, errOrderbookSubscriptionConflict, "oneOrderbookSubscription should report the subscription left out")
	assert.Equal(t, subscription.List{diff, tickerSub}, kept, "oneOrderbookSubscription should keep a pair's diff depth subscription over partial depth")
	kept, err = oneOrderbookSubscription(subscription.List{partial, tickerSub})
	require.NoError(t, err, "oneOrderbookSubscription must not error without a conflict")
	assert.Equal(t, subscription.List{partial, tickerSub}, kept, "oneOrderbookSubscription should keep a pair's only partial depth subscription")
}

func TestGetSubscriptionTemplate(t *testing.T) {
	t.Parallel()
	tpl, err := new(Exchange).GetSubscriptionTemplate(nil)
	require.NoError(t, err, "GetSubscriptionTemplate must not error")
	assert.NotNil(t, tpl, "GetSubscriptionTemplate should return the template")
}

func TestChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		s    *subscription.Subscription
		exp  string
		err  error
	}{
		{name: "ticker", s: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "ticker"},
		{name: "trades", s: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "trade"},
		{name: "candles every 15 minutes", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FifteenMin}, exp: "kline_15m"},
		{name: "candles every 10 minutes", s: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.TenMin}, err: kline.ErrUnsupportedInterval},
		{name: "order book", s: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "depth"},
		{name: "order book every second", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.ThousandMilliseconds}, exp: "depth"},
		{name: "order book every 100 milliseconds", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, exp: "depth@100ms"},
		{name: "order book of 10 levels every 100 milliseconds", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 10, Interval: kline.HundredMilliseconds}, exp: "depth10@100ms"},
		{name: "order book of 15 levels", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 15}, err: errUnsupportedLevels},
		{name: "order book every minute", s: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMin}, err: errUnsupportedFrequency},
		{name: "my orders", s: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, err: errUnsupportedChannel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := channelName(tc.s)
			require.ErrorIsf(t, err, tc.err, "channelName must return the expected error for %s %s levels %d", tc.s.Channel, tc.s.Interval, tc.s.Levels)
			assert.Equalf(t, tc.exp, got, "channelName should name %s %s levels %d", tc.s.Channel, tc.s.Interval, tc.s.Levels)
		})
	}
}

// newSubscriptionTestExchange returns an exchange connected to a local stream server answering subscriptions with
// streamsMock, without subscriptions, its stream connection's setup changed by each of modify
func newSubscriptionTestExchange(t *testing.T, modify ...func(*websocket.ConnectionSetup)) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newWebsocketTestExchange(t)
	ex.Features.Subscriptions = subscription.List{}
	setup := ex.streamConnectionSetup(ex.Config)
	setup.GenerateSubscriptions = func() (subscription.List, error) {
		return subscription.List{{Channel: "keepalive", QualifiedChannel: "btcusdt@trade", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}, nil
	}
	for _, m := range modify {
		m(setup)
	}
	useMockWebsocket(t, ex, setup, streamsMock)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")
	return ex, conn
}

// newLiveStreamTestExchange returns an exchange connected to Binance.US's market data streams with one trade stream
func newLiveStreamTestExchange(t *testing.T) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newWebsocketTestExchange(t)
	ex.Features.Subscriptions = subscription.List{}
	setup := ex.streamConnectionSetup(ex.Config)
	setup.GenerateSubscriptions = func() (subscription.List, error) {
		return subscription.List{{Channel: subscription.AllTradesChannel, QualifiedChannel: "btcusdt@trade", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}}, nil
	}
	useWebsocket(t, ex, setup)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	conn, err := ex.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")
	require.Equal(t, websocketStreamsURL, conn.GetURL(), "the market data streams must be Binance.US's")
	return ex, conn
}

func TestSubscribe(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the market data streams from a local server")
		}
		ex, conn := newLiveStreamTestExchange(t)
		sub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "btcusdt@ticker", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
		require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{sub}), "Subscribe must not error")
		assert.Equal(t, subscription.SubscribedState, sub.State(), "Subscribe should mark the subscription subscribed")
	})

	ex, conn := newSubscriptionTestExchange(t)
	sub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "btcusdt@ticker", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{sub}), "Subscribe must not error")
	assert.Equal(t, subscription.SubscribedState, sub.State(), "Subscribe should mark the subscription subscribed")
}

func TestUnsubscribe(t *testing.T) {
	t.Parallel()
	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("mock tests serve the market data streams from a local server")
		}
		ex, conn := newLiveStreamTestExchange(t)
		sub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "btcusdt@ticker", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
		require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{sub}), "Subscribe must not error")
		require.NoError(t, ex.Unsubscribe(t.Context(), conn, subscription.List{sub}), "Unsubscribe must not error")
		assert.Nil(t, ex.Websocket.GetSubscription(sub), "Unsubscribe should remove the subscription")
	})

	ex, conn := newSubscriptionTestExchange(t)
	sub := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "btcusdt@ticker", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{sub}), "Subscribe must not error")
	require.NoError(t, ex.Unsubscribe(t.Context(), conn, subscription.List{sub}), "Unsubscribe must not error")
	assert.Nil(t, ex.Websocket.GetSubscription(sub), "Unsubscribe should remove the subscription")
}

func TestManageSubs(t *testing.T) {
	t.Parallel()
	ex, conn := newSubscriptionTestExchange(t)
	failing := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "fail@ticker", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	err := ex.manageSubs(t.Context(), conn, wsSubscribeMethod, subscription.List{failing})
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "manageSubs should report a subscription the server refused")
	assert.Nil(t, ex.Websocket.GetSubscription(failing), "manageSubs should remove a refused subscription")
	unanswered := &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "noresult@ticker", Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}
	err = ex.manageSubs(t.Context(), conn, wsSubscribeMethod, subscription.List{unanswered})
	assert.ErrorIs(t, err, jsonparser.KeyPathNotFoundError, "manageSubs should report a response without a result")
	assert.Nil(t, ex.Websocket.GetSubscription(unanswered), "manageSubs should remove a subscription without a result")

	t.Run("order book stream unsubscription", func(t *testing.T) {
		t.Parallel()
		// Connecting subscribed with the connection's one message, so the next waits an hour
		hourLimit := func(s *websocket.ConnectionSetup) { s.RateLimit = request.NewWeightedRateLimitByDuration(time.Hour) }
		for _, tc := range []struct {
			name        string
			stream      string
			setup       func(*websocket.ConnectionSetup)
			ctxTimeout  time.Duration
			cancelled   bool
			err         error
			invalidated bool
		}{
			{name: "answered", stream: "btcusdt@depth@100ms", invalidated: true},
			{name: "refused", stream: "fail@depth@100ms", err: websocket.ErrSubscriptionFailure},
			{name: "unanswered", stream: "silent@depth@100ms", setup: func(s *websocket.ConnectionSetup) { s.ResponseMaxLimit = 2 * time.Second }, err: websocket.ErrSignatureTimeout, invalidated: true},
			{name: "wait for the answer ended", stream: "silent@depth@100ms", setup: func(s *websocket.ConnectionSetup) { s.ResponseMaxLimit = time.Minute }, ctxTimeout: time.Second, err: context.DeadlineExceeded, invalidated: true},
			{name: "wait to send refused for its deadline", stream: "btcusdt@depth@100ms", setup: hourLimit, ctxTimeout: time.Minute, err: context.DeadlineExceeded},
			{name: "cancelled before sending", stream: "btcusdt@depth@100ms", setup: hourLimit, cancelled: true, err: context.Canceled},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var modify []func(*websocket.ConnectionSetup)
				if tc.setup != nil {
					modify = append(modify, tc.setup)
				}
				ex, conn := newSubscriptionTestExchange(t, modify...)
				streamed, other := currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)
				for _, p := range []currency.Pair{streamed, other} {
					require.NoError(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{Exchange: ex.Name, Pair: p, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now()}), "LoadSnapshot must not error")
				}
				subs := subscription.List{
					{Channel: subscription.OrderbookChannel, QualifiedChannel: tc.stream, Asset: asset.Spot, Pairs: currency.Pairs{streamed}},
					{Channel: subscription.TickerChannel, QualifiedChannel: "ethusdt@ticker", Asset: asset.Spot, Pairs: currency.Pairs{other}},
				}
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
				ctx := t.Context()
				if tc.ctxTimeout != 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, tc.ctxTimeout)
					defer cancel()
				}
				if tc.cancelled {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				err := ex.manageSubs(ctx, conn, wsUnsubscribeMethod, subs)
				if tc.err == nil {
					require.NoError(t, err, "manageSubs must not error")
				} else {
					require.ErrorIs(t, err, tc.err, "manageSubs must report why the unsubscription failed")
				}
				_, err = orderbook.Get(ex.Name, streamed, asset.Spot)
				if tc.invalidated {
					assert.ErrorIs(t, err, errOrderbookUnsubscribed, "manageSubs should invalidate the book of an order book stream Binance.US may have stopped")
					assert.False(t, ex.orderbookStreamed(streamed, asset.Spot), "manageSubs should not leave the order book stream subscribed")
				} else {
					assert.NoError(t, err, "manageSubs should leave the book of an order book stream Binance.US still sends")
					assert.True(t, ex.orderbookStreamed(streamed, asset.Spot), "manageSubs should leave subscribed an order book stream Binance.US still sends")
				}
				_, err = orderbook.Get(ex.Name, other, asset.Spot)
				assert.NoError(t, err, "manageSubs should leave the book of a pair whose order book stream it did not unsubscribe")
			})
		}
	})
}

func TestOrderbookStreamed(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	p := currency.NewBTCUSDT()
	assert.False(t, ex.orderbookStreamed(p, asset.Spot), "orderbookStreamed should be false without an order book stream")
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil,
		&subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{p}},
		&subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}},
	), "AddSuccessfulSubscriptions must not error")
	assert.False(t, ex.orderbookStreamed(p, asset.Spot), "orderbookStreamed should be false for another channel's stream or another pair's")
	streamed := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{p}}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, streamed), "AddSuccessfulSubscriptions must not error")
	assert.True(t, ex.orderbookStreamed(p, asset.Spot), "orderbookStreamed should be true for a subscribed order book stream of the pair")
	assert.False(t, ex.orderbookStreamed(p, asset.Margin), "orderbookStreamed should be false for another asset")
	require.NoError(t, streamed.SetState(subscription.UnsubscribingState), "SetState must not error")
	assert.False(t, ex.orderbookStreamed(p, asset.Spot), "orderbookStreamed should be false for a stream unsubscribing")
	require.NoError(t, streamed.SetState(subscription.SubscribingState), "SetState must not error")
	assert.True(t, ex.orderbookStreamed(p, asset.Spot), "orderbookStreamed should be true for a stream subscribing, whose events can arrive before its subscription is answered")
}

func TestInvalidateUnsubscribedOrderbooks(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.Name = t.Name()
	streamed, replaced, other := currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT), currency.NewBTCUSD()
	for _, p := range []currency.Pair{streamed, replaced, other} {
		d, err := orderbook.DeployDepth(ex.Name, p, asset.Spot)
		require.NoError(t, err, "DeployDepth must not error")
		d.AssignOptions(&orderbook.Book{Exchange: ex.Name, Pair: p, Asset: asset.Spot})
		require.NoError(t, d.LoadSnapshot(&orderbook.Book{Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now()}), "LoadSnapshot must load the websocket book")
	}
	require.NoError(t, (&orderbook.Book{Exchange: ex.Name, Pair: replaced, Asset: asset.Spot, Bids: orderbook.Levels{{Price: 3, Amount: 1}}, Asks: orderbook.Levels{{Price: 4, Amount: 1}}, LastUpdated: time.Now()}).Process(), "Process must store the REST snapshot")
	generation := ex.orderbookStreamGeneration.Load()
	ex.invalidateUnsubscribedOrderbooks(subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{streamed, replaced, currency.NewPair(currency.LTC, currency.USDT)}},
		{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{other}},
	})
	assert.Equal(t, generation+1, ex.orderbookStreamGeneration.Load(), "invalidateUnsubscribedOrderbooks should count the order book stream's unsubscription, and only that, in orderbookStreamGeneration")
	_, err := orderbook.Get(ex.Name, streamed, asset.Spot)
	assert.ErrorIs(t, err, errOrderbookUnsubscribed, "invalidateUnsubscribedOrderbooks should invalidate a book an order book stream loaded")
	got, err := orderbook.Get(ex.Name, replaced, asset.Spot)
	require.NoError(t, err, "invalidateUnsubscribedOrderbooks must leave a book a REST snapshot replaced")
	assert.Equal(t, orderbook.Levels{{Price: 3, Amount: 1}}, got.Bids, "invalidateUnsubscribedOrderbooks should leave the REST snapshot's bids")
	_, err = orderbook.Get(ex.Name, other, asset.Spot)
	assert.NoError(t, err, "invalidateUnsubscribedOrderbooks should leave the book of a stream other than an order book stream")
}
