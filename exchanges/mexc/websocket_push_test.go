package mexc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mexc/mexc_proto_types"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// drainData returns everything relayed to the data handler so far, whatever its concrete type.
func drainData(tb testing.TB) []any {
	tb.Helper()
	var out []any
	for {
		select {
		case p := <-e.Websocket.DataHandler.C:
			out = append(out, p.Data)
		default:
			return out
		}
	}
}

// requireOneOf drains the data handler and asserts a single payload of the requested type arrived.
func requireOneOf[T any](tb testing.TB) T {
	tb.Helper()
	got := drainData(tb)
	require.Lenf(tb, got, 1, "exactly one payload must be relayed, got %#v", got)
	typed, ok := got[0].(T)
	require.Truef(tb, ok, "payload must be %T, got %T", *new(T), got[0])
	return typed
}

// TestWsHandleAggreDeals asserts public trades are decoded with the side derived from tradeType:
// MEXC uses 1 for a buy and anything else for a sell.
func TestWsHandleAggreDeals(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelAggreDealsV3+"@100ms@BTCUSDT", 1736409765052,
		&mexc_proto_types.PublicAggreDealsV3Api{
			Deals: []*mexc_proto_types.PublicAggreDealsV3ApiItem{
				{Price: "93220.00", Quantity: "0.04438243", TradeType: 1, Time: 1736409765051},
				{Price: "93221.50", Quantity: "1.5", TradeType: 2, Time: 1736409765099},
			},
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	trades := requireOneOf[[]trade.Data](t)
	require.Len(t, trades, 2, "both deals must be relayed")
	assert.Equal(t, 93220.00, trades[0].Price, "Price should be correct")
	assert.Equal(t, 0.04438243, trades[0].Amount, "Amount should be correct")
	assert.Equal(t, order.Buy, trades[0].Side, "tradeType 1 should map to Buy")
	assert.Equal(t, int64(1736409765051), trades[0].Timestamp.UnixMilli(), "Timestamp should come from the deal time")
	assert.Equal(t, asset.Spot, trades[0].AssetType, "AssetType should be correct")
	assert.Equal(t, e.Name, trades[0].Exchange, "Exchange should be correct")
	assert.Equal(t, order.Sell, trades[1].Side, "any other tradeType should map to Sell")
	assert.Equal(t, 1.5, trades[1].Amount, "Amount should be correct")
}

// TestWsHandleKline asserts the candle is decoded, including that the candle volume is taken from
// the protobuf `amount` field rather than `volume`.
func TestWsHandleKline(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelKlineV3+"@BTCUSDT@Min15", 1736410707571,
		&mexc_proto_types.PublicSpotKlineV3Api{
			Interval: "Min15", WindowStart: 1736410500, WindowEnd: 1736411400,
			OpeningPrice: "92925", ClosingPrice: "93158.47",
			HighestPrice: "93158.47", LowestPrice: "92800",
			Volume: "36.83803224", Amount: "3424811.05",
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	item := requireOneOf[*kline.Item](t)
	assert.Equal(t, kline.FifteenMin, item.Interval, "Interval should be decoded from the exchange string")
	assert.Equal(t, asset.Spot, item.Asset, "Asset should be correct")
	assert.Equal(t, e.Name, item.Exchange, "Exchange should be correct")
	require.Len(t, item.Candles, 1, "exactly one candle must be relayed")
	c := item.Candles[0]
	assert.Equal(t, 92925.0, c.Open, "Open should be correct")
	assert.Equal(t, 93158.47, c.Close, "Close should be correct")
	assert.Equal(t, 93158.47, c.High, "High should be correct")
	assert.Equal(t, 92800.0, c.Low, "Low should be correct")
	assert.Equal(t, 3424811.05, c.Volume, "Volume should come from the amount field")
	assert.Equal(t, time.UnixMilli(1736411400), c.Time, "Time should come from the window end")
}

// TestWsHandleKlineUnknownInterval asserts an interval the exchange has not documented is reported
// rather than silently producing a zero-interval candle.
func TestWsHandleKlineUnknownInterval(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelKlineV3+"@BTCUSDT@Min7", 1736410707571,
		&mexc_proto_types.PublicSpotKlineV3Api{
			Interval: "Min7", OpeningPrice: "1", ClosingPrice: "1",
			HighestPrice: "1", LowestPrice: "1", Volume: "1", Amount: "1",
		})
	assert.Error(t, e.WsHandleData(t.Context(), nil, raw), "an unknown interval should be reported")
}

// TestWsHandleLimitDepth asserts the limit depth channel loads a full orderbook snapshot with the
// bids and asks on the correct sides.
func TestWsHandleLimitDepth(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelLimitDepthV3+"@BTCUSDT@5", 1736411838730,
		&mexc_proto_types.PublicLimitDepthsV3Api{
			Asks: []*mexc_proto_types.PublicLimitDepthV3ApiItem{{Price: "93180.18", Quantity: "0.21976424"}},
			Bids: []*mexc_proto_types.PublicLimitDepthV3ApiItem{{Price: "93179.98", Quantity: "2.82651000"}},
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	book, err := orderbook.Get(e.Name, spotTradablePair, asset.Spot)
	require.NoError(t, err, "the snapshot must be retrievable")
	require.Len(t, book.Asks, 1, "the ask side must hold the pushed level")
	require.Len(t, book.Bids, 1, "the bid side must hold the pushed level")
	assert.Equal(t, 93180.18, book.Asks[0].Price, "ask price should be correct")
	assert.Equal(t, 0.21976424, book.Asks[0].Amount, "ask amount should be correct")
	assert.Equal(t, 93179.98, book.Bids[0].Price, "bid price should be correct")
	assert.Equal(t, 2.82651, book.Bids[0].Amount, "bid amount should be correct")
}

// TestWsHandleAggreDepth asserts the aggregated depth channel is accepted and reaches the book.
func TestWsHandleAggreDepth(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelAggregateDepthV3+"@100ms@BTCUSDT", 1736411507002,
		&mexc_proto_types.PublicAggreDepthsV3Api{
			Asks: []*mexc_proto_types.PublicAggreDepthV3ApiItem{{Price: "92878.10", Quantity: "1.25"}},
			Bids: []*mexc_proto_types.PublicAggreDepthV3ApiItem{{Price: "92877.58", Quantity: "3.5"}},
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")
}

// TestWsHandleAggreDepthBadPrice asserts an unparsable level is reported rather than stored as zero.
func TestWsHandleAggreDepthBadPrice(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelAggregateDepthV3+"@100ms@BTCUSDT", 1736411507002,
		&mexc_proto_types.PublicAggreDepthsV3Api{
			Bids: []*mexc_proto_types.PublicAggreDepthV3ApiItem{{Price: "not-a-price", Quantity: "3.5"}},
		})
	assert.Error(t, e.WsHandleData(t.Context(), nil, raw), "an unparsable price should be reported")
}

// TestWsHandleIncreaseDepthBatch asserts every book in a batched depth frame is applied.
func TestWsHandleIncreaseDepthBatch(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelIncreaseDepthBatchV3+"@BTCUSDT", 1739502064578,
		&mexc_proto_types.PublicIncreaseDepthsBatchV3Api{
			Items: []*mexc_proto_types.PublicIncreaseDepthsV3Api{
				{Bids: []*mexc_proto_types.PublicIncreaseDepthV3ApiItem{{Price: "96578.48", Quantity: "0.00000000"}}, Version: "39003145507"},
				{Asks: []*mexc_proto_types.PublicIncreaseDepthV3ApiItem{{Price: "96579.31", Quantity: "4.88725694"}}, Version: "39003145509"},
			},
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")
}

// TestWsHandleBookTickerBatch asserts a batched book ticker frame relays one ticker per item.
func TestWsHandleBookTickerBatch(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelBookTickerBatch+"@BTCUSDT", 1739503249114,
		&mexc_proto_types.PublicBookTickerBatchV3Api{
			Items: []*mexc_proto_types.PublicBookTickerV3Api{
				{BidPrice: "96567.37", BidQuantity: "3.362925", AskPrice: "96567.38", AskQuantity: "1.545255"},
			},
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	tickers := requireOneOf[[]ticker.Price](t)
	require.Len(t, tickers, 1, "one ticker per item must be relayed")
	assert.Equal(t, 96567.37, tickers[0].Bid, "Bid should be correct")
	assert.Equal(t, 3.362925, tickers[0].BidSize, "BidSize should be correct")
	assert.Equal(t, 96567.38, tickers[0].Ask, "Ask should be correct")
	assert.Equal(t, 1.545255, tickers[0].AskSize, "AskSize should be correct")
	assert.Equal(t, asset.Spot, tickers[0].AssetType, "AssetType should be correct")
	assert.Equal(t, e.Name, tickers[0].ExchangeName, "ExchangeName should be correct")
}

// TestWsHandleUnknownChannel asserts an unrecognised channel is surfaced instead of dropped.
func TestWsHandleUnknownChannel(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@public.not.a.real.channel@100ms@BTCUSDT", 1,
		&mexc_proto_types.PublicAggreBookTickerV3Api{BidPrice: "1", BidQuantity: "1", AskPrice: "1", AskQuantity: "1"})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	warning := requireOneOf[websocket.UnhandledMessageWarning](t)
	assert.Contains(t, warning.Message, websocket.UnhandledMessage, "the warning should mark the message as unhandled")
}

// TestWsHandlePong asserts the acknowledgement of {"method":"PING"} is consumed silently: its id is
// 0, which matches no pending request, so without the check it would be reported as unhandled.
func TestWsHandlePong(t *testing.T) {
	drainData(t)
	require.NoError(t, e.WsHandleData(t.Context(), nil, []byte(`{"id":0,"code":0,"msg":"PONG"}`)), "a pong must not error")
	assert.Empty(t, drainData(t), "a pong should not be relayed to the data handler")
}

// TestWsHandleJSONWithoutID asserts a JSON control frame carrying no id is ignored quietly.
func TestWsHandleJSONWithoutID(t *testing.T) {
	drainData(t)
	require.NoError(t, e.WsHandleData(t.Context(), nil, []byte(`{"code":0,"msg":"no id here"}`)), "an id-less frame must not error")
	assert.Empty(t, drainData(t), "an id-less JSON frame should not be relayed")
}

// TestWsChannelName asserts the channel name is taken from the decoded channel field. A private
// channel is qualified as "spot@<name>" with nothing after the name, so the name has to survive
// being the last element.
func TestWsChannelName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ qualified, expected string }{
		{"spot@" + channelBookTiker + "@100ms@BTCUSDT", channelBookTiker},
		{"spot@" + channelKlineV3 + "@BTCUSDT@Min15", channelKlineV3},
		{"spot@" + channelLimitDepthV3 + "@BTCUSDT@5", channelLimitDepthV3},
		{"spot@" + channelMiniTickerV3 + "@BTCUSDT@UTC+8", channelMiniTickerV3},
		{"spot@" + channelAccountV3, channelAccountV3},
		{"spot@" + channelPrivateDealsV3, channelPrivateDealsV3},
		{"spot@" + channelPrivateOrdersAPI, channelPrivateOrdersAPI},
		{"spot", ""},
		{"", ""},
	} {
		assert.Equalf(t, tc.expected, wsChannelName(tc.qualified), "wsChannelName should return the channel name of %q", tc.qualified)
	}
}

// TestWsHandlePrivateAccount asserts a private account frame is routed and decoded. The private
// channels carry no symbol after the channel name, so routing on the raw bytes never reached them.
func TestWsHandlePrivateAccount(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelAccountV3, 1736409765052,
		&mexc_proto_types.PrivateAccountV3Api{VcoinName: "USDT", BalanceAmount: "100.5", FrozenAmount: "0.5"})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	change := requireOneOf[accounts.Change](t)
	assert.Equal(t, currency.USDT, change.Balance.Currency, "Currency should be correct")
	assert.Equal(t, 100.5, change.Balance.Total, "Total should be the balance amount")
	assert.Equal(t, 0.5, change.Balance.Hold, "Hold should be the frozen amount")
	assert.Equal(t, 100.0, change.Balance.Free, "Free should be the balance less the frozen amount")
	assert.Equal(t, asset.Spot, change.AssetType, "AssetType should be correct")
}

// TestWsHandlePrivateDeals asserts a private trade frame is routed and decoded.
func TestWsHandlePrivateDeals(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelPrivateDealsV3, 1736409765052,
		&mexc_proto_types.PrivateDealsV3Api{
			Price: "93220.00", Amount: "0.044", OrderId: "o-1", TradeType: 1, Time: 1736409765051,
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	trades := requireOneOf[[]trade.Data](t)
	require.Len(t, trades, 1, "one trade must be relayed")
	assert.Equal(t, "o-1", trades[0].TID, "TID should be the order id")
	assert.Equal(t, 93220.00, trades[0].Price, "Price should be correct")
	assert.Equal(t, 0.044, trades[0].Amount, "Amount should be correct")
	assert.Equal(t, order.Buy, trades[0].Side, "tradeType 1 should map to Buy")
	assert.Equal(t, int64(1736409765051), trades[0].Timestamp.UnixMilli(), "Timestamp should come from the deal time")
}

// TestWsHandlePrivateOrders asserts a private order frame is routed and that the base and quote
// figures land in the matching order.Detail fields: MEXC sends quantity/remainQuantity/
// cumulativeQuantity in base terms and amount/remainAmount/cumulativeAmount in quote terms.
func TestWsHandlePrivateOrders(t *testing.T) {
	drainData(t)
	raw := wsPushFrame(t, "spot@"+channelPrivateOrdersAPI, 1736409765052,
		&mexc_proto_types.PrivateOrdersV3Api{
			Id: "o-2", ClientId: "c-2", Price: "100", AvgPrice: "101",
			Quantity: "10", RemainQuantity: "6", CumulativeQuantity: "4",
			Amount: "1000", RemainAmount: "600", CumulativeAmount: "404",
			OrderType: 1, TradeType: 1, Status: 3, CreateTime: 1736409765000,
		})
	require.NoError(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error")

	detail := requireOneOf[*order.Detail](t)
	assert.Equal(t, "o-2", detail.OrderID, "OrderID should be correct")
	assert.Equal(t, "c-2", detail.ClientID, "ClientID should be correct")
	assert.Equal(t, 100.0, detail.Price, "Price should be correct")
	assert.Equal(t, 101.0, detail.AverageExecutedPrice, "AverageExecutedPrice should be correct")
	assert.Equal(t, 10.0, detail.Amount, "Amount should be the base quantity")
	assert.Equal(t, 1000.0, detail.QuoteAmount, "QuoteAmount should be the quote amount")
	assert.Equal(t, 4.0, detail.ExecutedAmount, "ExecutedAmount should be the base cumulative quantity")
	assert.Equal(t, 6.0, detail.RemainingAmount, "RemainingAmount should be the base remaining quantity")
	assert.Equal(t, order.Limit, detail.Type, "orderType 1 should map to a limit order")
	assert.Equal(t, order.GoodTillCancel, detail.TimeInForce, "orderType 1 should map to GoodTillCancel")
	assert.Equal(t, order.PartiallyFilled, detail.Status, "status 3 should map to PartiallyFilled")
	assert.Equal(t, order.Buy, detail.Side, "tradeType 1 should map to Buy")
	assert.Equal(t, int64(1736409765000), detail.LastUpdated.UnixMilli(), "LastUpdated should come from createTime")
	assert.Equal(t, asset.Spot, detail.AssetType, "AssetType should be correct")
}

// TestWsBookTickerSnapshotIsPerPair asserts each pair gets its own orderbook snapshot. The
// aggregated book ticker is qualified as "spot@<name>@<interval>@<symbol>", so keying the
// "snapshot already loaded" map by the channel's third element keyed it by the interval, a value
// shared by every pair, which left the second pair updating a book that was never loaded.
func TestWsBookTickerSnapshotIsPerPair(t *testing.T) {
	drainData(t)
	second := currency.NewPair(currency.ETH, currency.USDT)
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotTradablePair, second}, false), "StorePairs must not error")
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotTradablePair, second}, true), "StorePairs must not error")
	t.Cleanup(func() {
		require.NoError(t, e.setEnabledPairs(spotTradablePair), "restoring the enabled pairs must not error")
	})

	syncOrderbookPairsLock.Lock()
	clear(orderbookSnapshotLoadedPairs)
	syncOrderbookPairsLock.Unlock()

	for _, symbol := range []string{wsTestSymbol, "ETHUSDT"} {
		raw := wsPushFrameForSymbol(t, symbol, "spot@"+channelBookTiker+"@100ms@"+symbol, 1736409765052,
			&mexc_proto_types.PublicAggreBookTickerV3Api{BidPrice: "1", BidQuantity: "2", AskPrice: "3", AskQuantity: "4"})
		require.NoErrorf(t, e.WsHandleData(t.Context(), nil, raw), "WsHandleData must not error for %s", symbol)
	}

	for _, pair := range []currency.Pair{spotTradablePair, second} {
		book, err := orderbook.Get(e.Name, pair, asset.Spot)
		require.NoErrorf(t, err, "each pair must have its own orderbook, %s does not", pair)
		require.NotEmptyf(t, book.Bids, "%s must carry a bid", pair)
		assert.Equalf(t, 1.0, book.Bids[0].Price, "%s bid price should be correct", pair)
	}
}
