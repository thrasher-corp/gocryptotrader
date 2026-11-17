package bingx

import (
	"cmp"
	"context"
	"slices"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketHandleCoinMarginedData(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	enableTestTradeFeed(ex)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, true)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, testOrderbookSubscription(testCoinMarginedPair, asset.CoinMarginedFutures, 20)), "wsClaimBook must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsCoinMargined.jsonl", func(ctx context.Context, frame []byte) error {
		return ex.wsHandleCoinMarginedData(ctx, conn, frame)
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
		case *WebsocketCoinMarginedLastPrice:
			counts["last prices"]++
		case *WebsocketMarkPrice:
			counts["mark prices"]++
		case *WebsocketCoinMarginedBookTicker:
			counts["book tickers"]++
		default:
			assert.Failf(t, "unexpected payload", "%T relayed", p)
		}
	}
	assert.Equal(t, map[string]int{"trades": 2, "candles": 2, "books": 2, "tickers": 1, "last prices": 1, "mark prices": 1, "book tickers": 1}, counts, "every fixture frame should be relayed")
	assert.Equal(t, []testWebsocketMessage{{limit: websocketHeartbeatRate, message: `{"pong":"0c6bbddcbc384117842b04e335c8e092","time":"2026-11-11T17:52:45.233+0800"}`}}, conn.messages(), "the heartbeat should be answered")

	state.mu.Lock()
	state.inFlightID, state.inFlightDataType = "req-1", "NOPE-USD@trade"
	state.mu.Unlock()
	require.NoError(t, ex.wsHandleCoinMarginedData(t.Context(), conn, []byte(`{"code":100400,"msg":"code:100400: symbol is not found: NOPE_USD","timestamp":1794390798083}`)), "an error naming the request in flight must be routed to it")
	assert.Equal(t, []any{"req-1"}, conn.matches(), "the error should be matched to the request in flight")
	assert.ErrorIs(t, ex.wsHandleCoinMarginedData(t.Context(), conn, []byte(`{"code":100400,"msg":"code:100400:nonsense dataType is not supported","timestamp":1794390762853}`)), errWebsocketErrorMessage, "an error naming no request in flight should be reported")
	assert.ErrorIs(t, ex.wsHandleCoinMarginedData(t.Context(), conn, []byte(`{"code":0,"dataType":"BTC-USD@incrDepth","data":{}}`)), errWebsocketUnhandledMessage, "an unknown stream should be reported")
	assert.ErrorIs(t, ex.wsHandleCoinMarginedData(t.Context(), conn, []byte(`{"code":0,"data":{}}`)), errWebsocketUnhandledMessage, "a frame naming no stream should be reported")
}

func TestWebsocketProcessCoinMarginedTrade(t *testing.T) {
	t.Parallel()
	data := fixtureData(t, "testdata/wsCoinMargined.jsonl", "@trade")
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessCoinMarginedTrade(data), "wsProcessCoinMarginedTrade must not error without a trade feed")
	assert.Empty(t, websocketPayloads(ex), "trades should not be relayed without a trade feed")

	enableTestTradeFeed(ex)
	require.NoError(t, ex.wsProcessCoinMarginedTrade(data), "wsProcessCoinMarginedTrade must not error")
	exp := []trade.Data{{
		TID:          "178610442",
		Exchange:     ex.Name,
		CurrencyPair: testCoinMarginedPair,
		AssetType:    asset.CoinMarginedFutures,
		Side:         order.Buy,
		Price:        73577.7,
		Amount:       4,
		Timestamp:    time.UnixMilli(1794390762788),
	}}
	assert.Equal(t, exp, waitForWebsocketPayload[[]trade.Data](t, ex, nil), "the trade should be relayed in contracts")

	require.NoError(t, ex.wsProcessCoinMarginedTrade([]byte(`{"s":"ETH-USD","t":"1","p":"1","q":"1","T":1794390762788,"m":true}`)), "wsProcessCoinMarginedTrade must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's trade should not be relayed")
	assert.ErrorIs(t, ex.wsProcessCoinMarginedTrade([]byte(`{"s":"DOGE-USD"}`)), currency.ErrPairNotFound, "wsProcessCoinMarginedTrade should error for an unknown pair")
	assert.Error(t, ex.wsProcessCoinMarginedTrade([]byte(`{"m":"x"}`)), "wsProcessCoinMarginedTrade should error for a trade that does not decode")
}

func TestWebsocketProcessCoinMarginedTicker(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	require.NoError(t, ex.wsProcessCoinMarginedTicker(t.Context(), fixtureData(t, "testdata/wsCoinMargined.jsonl", "@ticker")), "wsProcessCoinMarginedTicker must not error")
	exp := &ticker.Price{
		Last:                73577.7,
		LastSize:            4,
		High:                74731,
		Low:                 72634.3,
		Bid:                 73577.7,
		BidSize:             577,
		Ask:                 73577.8,
		AskSize:             14110,
		BaseVolume:          1219.67,
		QuoteVolume:         89679300,
		Open:                74343.8,
		PercentChange24Hour: -1.02,
		Pair:                testCoinMarginedPair,
		ExchangeName:        ex.Name,
		AssetType:           asset.CoinMarginedFutures,
		LastUpdated:         time.UnixMilli(1794390761000),
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*ticker.Price](t, ex, nil), "the ticker should be relayed with its volumes in the base coin and USD")
	_, err := ticker.GetTicker(ex.Name, testCoinMarginedPair, asset.CoinMarginedFutures)
	assert.NoError(t, err, "the ticker should be stored")

	require.NoError(t, ex.wsProcessCoinMarginedTicker(t.Context(), []byte(`{"s":"ETH-USD","c":"1"}`)), "wsProcessCoinMarginedTicker must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's ticker should not be relayed")
	badPercentage := []byte(`{"P":"%x"}`)
	assert.Error(t, ex.wsProcessCoinMarginedTicker(t.Context(), badPercentage), "wsProcessCoinMarginedTicker should error for a ticker that does not decode")
}

func TestWebsocketProcessCoinMarginedKline(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	frame := fixtureLine(t, "testdata/wsCoinMargined.jsonl", "@kline_1m")
	data, _, _, err := jsonparser.Get(frame, "data")
	require.NoError(t, err, "the push must have data")
	require.NoError(t, ex.wsProcessCoinMarginedKline(t.Context(), "1m", frame, data), "wsProcessCoinMarginedKline must not error")
	exp := kline.Item{
		Exchange: ex.Name,
		Pair:     testCoinMarginedPair,
		Asset:    asset.CoinMarginedFutures,
		Interval: kline.OneMin,
		Candles: []kline.Candle{{
			Time:             time.UnixMilli(1794390720000),
			Open:             73594.701,
			High:             73597.299,
			Low:              73577.701,
			Close:            73577.799,
			Volume:           182,
			ValidationIssues: kline.PartialCandle,
		}},
	}
	assert.Equal(t, exp, waitForWebsocketPayload[kline.Item](t, ex, nil), "the candle should be relayed in contracts, partial while the frame's timestamp precedes its end")

	week := fixtureLine(t, "testdata/wsCoinMargined.jsonl", "@kline_1w")
	weekData, _, _, err := jsonparser.Get(week, "data")
	require.NoError(t, err, "the push must have data")
	require.NoError(t, ex.wsProcessCoinMarginedKline(t.Context(), "1w", week, weekData), "wsProcessCoinMarginedKline must not error")
	weekly := waitForWebsocketPayload[kline.Item](t, ex, nil)
	assert.Equal(t, time.UnixMilli(1794153600000), weekly.Candles[0].Time, "a candle stamped with the current minute should be dated by the start of its UTC+8 week")
	assert.Equal(t, kline.PartialCandle, weekly.Candles[0].ValidationIssues, "a candle within its week should be partial")

	require.NoError(t, ex.wsProcessCoinMarginedKline(t.Context(), "1m", nil, data), "wsProcessCoinMarginedKline must not error without a frame timestamp")
	assert.Empty(t, waitForWebsocketPayload[kline.Item](t, ex, nil).Candles[0].ValidationIssues, "a candle received after its end should not be partial")

	require.NoError(t, ex.wsProcessCoinMarginedKline(t.Context(), "1m", nil, []byte(`{"s":"ETH-USD"}`)), "wsProcessCoinMarginedKline must not error for a disabled pair")
	assert.Empty(t, websocketPayloads(ex), "a disabled pair's candle should not be relayed")
	assert.ErrorIs(t, ex.wsProcessCoinMarginedKline(t.Context(), "1min", nil, []byte(`{"s":"BTC-USD"}`)), kline.ErrInvalidInterval, "wsProcessCoinMarginedKline should reject another product's interval")
	assert.Error(t, ex.wsProcessCoinMarginedKline(t.Context(), "1m", nil, []byte(`{"u":"x"}`)), "wsProcessCoinMarginedKline should error for a candle that does not decode")
}

func TestWebsocketProcessCoinMarginedOrderbook(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	conn := &testWebsocketConnection{}
	state, _ := trackTestConnection(t, ex, conn, true)
	require.NoError(t, ex.wsClaimBook(conn, state.generation, testOrderbookSubscription(testCoinMarginedPair, asset.CoinMarginedFutures, 20)), "wsClaimBook must not error")
	var pushes int
	for _, line := range fixtureLines(t, "testdata/wsCoinMargined.jsonl") {
		if dataType, _ := jsonparser.GetString(line, "dataType"); dataType != "BTC-USD@depth20" {
			continue
		}
		pushes++
		data, _, _, err := jsonparser.Get(line, "data")
		require.NoError(t, err, "the push must have data")
		require.NoError(t, ex.wsProcessCoinMarginedOrderbook(t.Context(), conn, "BTC-USD@depth20", data), "wsProcessCoinMarginedOrderbook must not error")
		var push WebsocketCoinMarginedOrderbook
		require.NoError(t, json.Unmarshal(data, &push), "the push must decode")
		expBids, expAsks := make(orderbook.Levels, len(push.Bids)), make(orderbook.Levels, len(push.Asks))
		for i := range push.Bids {
			expBids[i] = orderbook.Level{Price: push.Bids[i].Price.Float64(), Amount: push.Bids[i].Volume.Float64()}
		}
		for i := range push.Asks {
			expAsks[i] = orderbook.Level{Price: push.Asks[i].Price.Float64(), Amount: push.Asks[i].Volume.Float64()}
		}
		slices.SortFunc(expBids, func(a, b orderbook.Level) int { return cmp.Compare(b.Price, a.Price) })
		slices.SortFunc(expAsks, func(a, b orderbook.Level) int { return cmp.Compare(a.Price, b.Price) })
		book, err := ex.Websocket.Orderbook.GetOrderbook(testCoinMarginedPair, asset.CoinMarginedFutures)
		require.NoError(t, err, "the push must load the book")
		assert.Equal(t, expBids, book.Bids, "the bids should be loaded highest first in contracts")
		assert.Equal(t, expAsks, book.Asks, "the asks, which arrive highest first, should be loaded lowest first in contracts")
		assert.Equal(t, push.Timestamp.Time(), book.LastUpdated, "the book should be dated by the push's timestamp")
	}
	require.Equal(t, 2, pushes, "the fixture must hold two pushes")

	require.NoError(t, ex.wsProcessCoinMarginedOrderbook(t.Context(), conn, "BTC-USD@depth20", []byte(`{"symbol":"BTC-USD","bids":[{"p":"1","a":"0.1","v":"10"}],"asks":[{"p":"2","a":"0.1","v":"20"}]}`)), "wsProcessCoinMarginedOrderbook must not error without a timestamp")
	book, err := ex.Websocket.Orderbook.GetOrderbook(testCoinMarginedPair, asset.CoinMarginedFutures)
	require.NoError(t, err, "the push must load the book")
	assert.WithinDuration(t, time.Now(), book.LastUpdated, time.Minute, "a push without a timestamp should be dated by its receipt")

	require.NoError(t, ex.wsProcessCoinMarginedOrderbook(t.Context(), conn, "BTC-USD@depth20", []byte(`{"symbol":"ETH-USD"}`)), "wsProcessCoinMarginedOrderbook must not error for a disabled pair")
	assert.ErrorIs(t, ex.wsProcessCoinMarginedOrderbook(t.Context(), conn, "BTC-USD@depth20", []byte(`{"symbol":"DOGE-USD"}`)), currency.ErrPairNotFound, "wsProcessCoinMarginedOrderbook should error for an unknown pair")
	assert.Error(t, ex.wsProcessCoinMarginedOrderbook(t.Context(), conn, "BTC-USD@depth20", []byte(`{"bids":{}}`)), "wsProcessCoinMarginedOrderbook should error for a push that does not decode")
}

func TestWebsocketCoinMarginedLevels(t *testing.T) {
	t.Parallel()
	levels := wsCoinMarginedLevels([]WebsocketCoinMarginedOrderbookLevel{{Price: 73577.7, Amount: 0.595289, Volume: 438}, {Price: 73577.6, Amount: 0.001359, Volume: 1}})
	assert.Equal(t, orderbook.Levels{{Price: 73577.7, Amount: 438}, {Price: 73577.6, Amount: 1}}, levels, "levels should be sized in contracts")
	assert.Empty(t, wsCoinMarginedLevels(nil), "no levels should give none")
}

func TestWebsocketHandleCoinMarginedAccountData(t *testing.T) {
	t.Parallel()
	server := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, server.URL)
	conn := &testWebsocketConnection{}
	trackTestConnection(t, ex, conn, false)
	testexch.FixtureToDataHandler(t, "testdata/wsCoinMarginedAccount.jsonl", func(ctx context.Context, frame []byte) error {
		return ex.wsHandleCoinMarginedAccountData(ctx, conn, frame)
	})
	payloads := websocketPayloads(ex)
	require.Len(t, payloads, 3, "each documented event must be relayed")
	assert.IsType(t, []WebsocketPerpetualBalance{}, payloads[0], "the account event's balances should be relayed as reported")
	testWaitBackground(t, ex)
	assert.Equal(t, 1, server.balanceRequestCount("/openApi/cswap/v1/user/balance"), "the account event should refresh the balances over REST")
	assert.IsType(t, &order.Detail{}, payloads[1], "the order event should be relayed as an order detail")
	assert.IsType(t, &WebsocketPerpetualConfigUpdate{}, payloads[2], "the configuration event should be relayed as it is")
	if d, ok := payloads[1].(*order.Detail); ok {
		assert.Equal(t, asset.CoinMarginedFutures, d.AssetType, "the order should be coin-M's")
	}
	if c, ok := payloads[2].(*WebsocketPerpetualConfigUpdate); ok {
		assert.Equal(t, WebsocketPerpetualConfig{Symbol: "BTC-USD", LongLeverage: 4, ShortLeverage: 20, MarginMode: "Cross"}, c.Configuration, "the configuration should be relayed")
	}
}

func TestWebsocketCoinMarginedIntegration(t *testing.T) {
	t.Parallel()
	servers := newTestWebsocketServers(t, asset.CoinMarginedFutures, "BTC-USD@markPrice")
	ex := newWebsocketIntegrationExchange(t, servers, asset.CoinMarginedFutures, true, func(cfg *config.Exchange) {
		cfg.Features.Enabled.TradeFeed = true
		cfg.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.AllTradesChannel},
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel, Levels: 20},
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: WebsocketMarkPriceChannel},
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true},
			{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyAccountChannel, Authenticated: true},
		}
	})
	err := ex.Websocket.Connect(t.Context())
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "the refused subscription should be reported")
	assert.ErrorIs(t, err, errWebsocketRequestRejected, "the refusal without an id should reach the request that it names")
	require.True(t, ex.Websocket.IsConnected(), "a refused subscription must leave the others connected")
	assert.ElementsMatch(t, []string{"BTC-USD@ticker", "BTC-USD@trade", "BTC-USD@kline_1m", "BTC-USD@depth20", "ORDER_TRADE_UPDATE", "ACCOUNT_UPDATE"}, ex.Websocket.GetSubscriptions().QualifiedChannels(), "every subscription but the refused one should be stored")
	assert.Len(t, servers.ws[asset.CoinMarginedFutures].receivedRequests(), 5, "each market stream should be requested")

	waitForWebsocketPayload(t, ex, func(d *order.Detail) bool { return d.OrderID == "1976541385109553200" })
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	require.Eventually(t, func() bool {
		_, tickerErr := ticker.GetTicker(ex.Name, testCoinMarginedPair, asset.CoinMarginedFutures)
		_, bookErr := ex.Websocket.Orderbook.GetOrderbook(testCoinMarginedPair, asset.CoinMarginedFutures)
		_, balanceErr := ex.Accounts.GetBalance("", creds, asset.CoinMarginedFutures, currency.BTC)
		return tickerErr == nil && bookErr == nil && balanceErr == nil && len(servers.ws[asset.CoinMarginedFutures].heartbeatReplies()) == 2
	}, 5*time.Second, time.Millisecond, "the ticker, the book, the balance the account event refreshes over REST and both heartbeat replies must arrive")
	_, err = ex.Accounts.GetBalance("", creds, asset.CoinMarginedFutures, currency.NewCode("SUI"))
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "the account event's wallet balance should not be stored")
	// The server sends both depth frames at once, so the book may still hold the first
	assert.Eventually(t, func() bool {
		book, err := ex.Websocket.Orderbook.GetOrderbook(testCoinMarginedPair, asset.CoinMarginedFutures)
		return err == nil && len(book.Bids) != 0 && book.Bids[0].Amount == 577
	}, 5*time.Second, time.Millisecond, "the book should take the last depth frame's best bid, counted in contracts")
	assert.Equal(t, []string{testListenKey + "-1"}, servers.ws[asset.CoinMarginedFutures].accountKeys(), "the account connection should be dialled with the listen key")
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
		assert.Zerof(t, servers.ws[a].marketDials(), "disabled %s should dial no market connection", a)
		assert.Emptyf(t, servers.ws[a].accountKeys(), "disabled %s should dial no account connection", a)
	}
	shutdownTestWebsocket(t, ex)
	assert.ErrorIs(t, bookError(ex, testCoinMarginedPair, asset.CoinMarginedFutures), orderbook.ErrOrderbookInvalid, "shutting down should invalidate the book")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		testLiveMarketStreams(t, asset.CoinMarginedFutures, testCoinMarginedPair, 20)
	})
}
