package bingx

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
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
		assert.ErrorIsf(t, ex.wsHandleSwapData(t.Context(), conn, []byte(tc.frame)), tc.err, "wsHandleSwapData should return the expected error for %s", tc.name)
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
	require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), asset.USDTMarginedFutures, fixtureLine(t, "testdata/wsSwapAccount.jsonl", "TRADE_UPDATE")), "wsProcessPerpetualOrderUpdate must not error")
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

	require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), asset.CoinMarginedFutures, fixtureLine(t, "testdata/wsCoinMarginedAccount.jsonl", "ORDER_TRADE_UPDATE")), "wsProcessPerpetualOrderUpdate must not error")
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
		Status:               order.UnknownStatus,
		AssetType:            asset.CoinMarginedFutures,
		Date:                 time.UnixMilli(1760079163296),
		LastUpdated:          time.UnixMilli(1760079163296),
		Pair:                 currency.NewPairWithDelimiter("SUI", "USD", currency.DashDelimiter),
	}
	assert.Equal(t, exp, waitForWebsocketPayload[*order.Detail](t, ex, nil), "the coin-M order event should be relayed without a cost, dated by its event without a trade time")

	reduceOnly := []byte(`{"e":"ORDER_TRADE_UPDATE","E":1758608899350,"o":{"s":"BTC-USDT","i":3,"S":"BUY","o":"STOP","q":"2","p":"10","sp":"9","x":"NEW","X":"NEW","ro":"true","n":"-1"}}`)
	require.NoError(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), asset.USDTMarginedFutures, reduceOnly), "wsProcessPerpetualOrderUpdate must not error")
	got := waitForWebsocketPayload[*order.Detail](t, ex, nil)
	assert.True(t, got.ReduceOnly, "a reduce only order should be relayed as one")
	assert.Zero(t, got.Fee, "an event without a fill should carry no fee")
	assert.InDelta(t, 2, got.RemainingAmount, 1e-9, "an order without fills should have its whole quantity remaining")
	assert.InDelta(t, 9, got.TriggerPrice, 1e-9, "the trigger price should be relayed")
	assert.Equal(t, order.StopLimit, got.Type, "STOP should be a stop limit order")

	for _, tc := range []struct {
		name, event string
		err         error
	}{
		{"status", `{"o":{"s":"BTC-USDT","S":"BUY","o":"LIMIT","X":"WAITING","i":1}}`, order.ErrUnsupportedStatusType},
		{"type", `{"o":{"s":"BTC-USDT","S":"BUY","o":"ICEBERG","X":"NEW","i":1}}`, order.ErrUnrecognisedOrderType},
		{"side", `{"o":{"s":"BTC-USDT","S":"BOTH","o":"LIMIT","X":"NEW","i":1}}`, order.ErrSideIsInvalid},
		{"symbol", `{"o":{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"NEW","i":1}}`, errWebsocketInvalidSymbol},
	} {
		require.NoErrorf(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), asset.USDTMarginedFutures, []byte(tc.event)), "wsProcessPerpetualOrderUpdate must relay an unclassified %s", tc.name)
		classification := waitForWebsocketPayload[order.ClassificationError](t, ex, nil)
		assert.ErrorIsf(t, classification.Err, tc.err, "an unknown %s should be relayed as a classification error", tc.name)
		assert.Equal(t, "1", classification.OrderID, "the classification error should name the order")
	}
	assert.Error(t, ex.wsProcessPerpetualOrderUpdate(t.Context(), asset.USDTMarginedFutures, []byte(`{"o":{"sg":1}}`)), "wsProcessPerpetualOrderUpdate should error for an event that does not decode")
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
		ex.wsTrackConnection(conn, newTestSession(t, ex).ch, false, first)
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
		assert.Zerof(t, servers.ws[a].marketDials(), "disabled %s should dial no market connection", a)
		assert.Emptyf(t, servers.ws[a].accountKeys(), "disabled %s should dial no account connection", a)
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
