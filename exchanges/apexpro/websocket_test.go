package apexpro

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func newWsTestExchange(t *testing.T) *Exchange {
	t.Helper()
	x := new(Exchange)
	require.NoError(t, testexch.Setup(x), "Setup must not error")
	return x
}

func TestGeneratePingMessage(t *testing.T) {
	t.Parallel()
	msg, err := generatePingMessage()
	require.NoError(t, err, "generatePingMessage must not error")
	var m WsMessage
	require.NoError(t, json.Unmarshal(msg, &m), "ping message must unmarshal")
	assert.Equal(t, "ping", m.Operation, "ping operation should be set")
	require.Len(t, m.Args, 1, "ping must carry a single timestamp argument")
}

func TestGenerateDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	subs, err := e.GenerateDefaultSubscriptions()
	require.NoError(t, err, "GenerateDefaultSubscriptions must not error")
	require.NotEmpty(t, subs, "GenerateDefaultSubscriptions must return subscriptions")
	channels := make(map[string]bool, len(subs))
	for _, s := range subs {
		channels[s.Channel] = true
	}
	for _, ch := range defaultChannels {
		assert.Truef(t, channels[ch], "default subscriptions should include the %s channel", ch)
	}
}

func TestHandleSubscriptionPayload(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	subs := subscription.List{
		{Channel: chOrderbook, Pairs: currency.Pairs{pair}, Levels: 200},
		{Channel: chTrade, Pairs: currency.Pairs{pair}},
		{Channel: chTicker, Pairs: currency.Pairs{pair}},
		{Channel: chCandlestick, Pairs: currency.Pairs{pair}, Interval: kline.FiveMin},
		{Channel: chAllTickers},
	}
	payload, err := e.handleSubscriptionPayload("subscribe", subs)
	require.NoError(t, err, "handleSubscriptionPayload must not error")
	assert.Equal(t, "subscribe", payload.Operation, "operation should be propagated")
	assert.ElementsMatch(t, []string{
		"orderBook200.H.BTCUSDT",
		"recentlyTrade.H.BTCUSDT",
		"instrumentInfo.H.BTCUSDT",
		"candle.5.BTCUSDT",
		"instrumentInfo.all",
	}, payload.Args, "subscription args should match the documented topic format")
}

func TestHandleSubscriptionPayloadErrors(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	_, err := e.handleSubscriptionPayload("subscribe", subscription.List{{Channel: chOrderbook, Pairs: currency.Pairs{pair}}})
	require.ErrorIs(t, err, errOrderbookLevelIsRequired, "orderbook subscription without a level must error")

	_, err = e.handleSubscriptionPayload("subscribe", subscription.List{{Channel: chCandlestick, Pairs: currency.Pairs{pair}}})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "candlestick subscription without an interval must error")
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	x := newWsTestExchange(t)
	fixtures := map[string]string{
		"orderbook snapshot": `{"topic":"orderBook200.H.BTCUSDT","type":"snapshot","data":{"s":"BTC-USDT","b":[["99","2"],["98","3"]],"a":[["101","1"],["102","4"]],"u":12345},"cs":1,"ts":1700000000000}`,
		"trade":              `{"topic":"recentlyTrade.H.BTCUSDT","type":"snapshot","data":[{"T":1700000000000,"s":"BTC-USDT","S":"BUY","v":"1.5","p":"100.5","L":"PlusTick","i":"trade-1"}],"cs":1,"ts":1700000000000}`,
		"ticker":             `{"topic":"instrumentInfo.H.BTCUSDT","type":"snapshot","data":{"symbol":"BTC-USDT","lastPrice":"100","price24hPcnt":"0.1","highPrice24h":"110","lowPrice24h":"90","turnover24h":"1000","volume24h":"50","oraclePrice":"100.2","indexPrice":"100.1","openInterest":"5","tradeCount":"10","fundingRate":"0.0001","predictedFundingRate":"0.0002"},"cs":1,"ts":1700000000000}`,
		"candle":             `{"topic":"candle.5.BTCUSDT","type":"snapshot","data":[{"start":1700000000000,"symbol":"BTC-USDT","interval":"5","low":"90","high":"110","open":"95","close":"105","volume":"50","turnover":"5000"}],"ts":1700000000000}`,
		"all tickers":        `{"topic":"instrumentInfo.all","type":"snapshot","data":[{"s":"BTC-USDT","p":"100","pr":"0.1","h":"110","l":"90","op":"95","xp":"100.1","to":"1000","v":"50","fr":"0.0001","o":"5","tc":"10","mp":"100.2"}],"ts":1700000000000}`,
		"pong":               `{"op":"pong","args":["1700000000000"]}`,
	}
	for name, msg := range fixtures {
		assert.NoErrorf(t, x.wsHandleData(t.Context(), []byte(msg)), "wsHandleData should not error for %s", name)
	}
}

func TestWsHandleDataTickerRouting(t *testing.T) {
	t.Parallel()
	x := newWsTestExchange(t)

	single := `{"topic":"instrumentInfo.H.BTCUSDT","type":"snapshot","data":{"symbol":"BTC-USDT","lastPrice":"100","indexPrice":"100.1","oraclePrice":"100.2"},"cs":1,"ts":1700000000000}`
	require.NoError(t, x.wsHandleData(t.Context(), []byte(single)), "single ticker must route without error")
	payload := <-x.Websocket.DataHandler.C
	_, ok := payload.Data.(*ticker.Price)
	assert.True(t, ok, "instrumentInfo topic should route to a single *ticker.Price")

	all := `{"topic":"instrumentInfo.all","type":"snapshot","data":[{"s":"BTC-USDT","p":"100","xp":"100.1","mp":"100.2"}],"ts":1700000000000}`
	require.NoError(t, x.wsHandleData(t.Context(), []byte(all)), "all tickers must route without error")
	payload = <-x.Websocket.DataHandler.C
	_, ok = payload.Data.([]ticker.Price)
	assert.True(t, ok, "instrumentInfo.all topic should route to a []ticker.Price")
}

func TestWsProcessOrderbook(t *testing.T) {
	snapshot := `{"topic":"orderBook200.H.BTCUSDT","type":"snapshot","data":{"s":"BTC-USDT","b":[["99","2"],["98","3"]],"a":[["101","1"],["102","4"]],"u":12345},"cs":1,"ts":1700000000000}`
	delta := `{"topic":"orderBook200.H.BTCUSDT","type":"delta","data":{"s":"BTC-USDT","b":[["99","5"]],"a":[["101","1"]],"u":12346},"cs":2,"ts":1700000000001}`
	require.NoError(t, e.wsHandleData(t.Context(), []byte(snapshot)), "orderbook snapshot must not error")
	require.NoError(t, e.wsHandleData(t.Context(), []byte(delta)), "delta must apply to the snapshot loaded under the same asset")

	ob, err := e.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.PerpetualContract)
	require.NoError(t, err, "orderbook must be retrievable under asset.PerpetualContract")
	require.NotEmpty(t, ob.Bids, "orderbook must contain bids")
	require.NotEmpty(t, ob.Asks, "orderbook must contain asks")
	assert.Equal(t, 99.0, ob.Bids[0].Price, "best bid price should match the delta update")
	assert.Equal(t, 5.0, ob.Bids[0].Amount, "best bid amount should reflect the delta update")
	assert.Equal(t, 101.0, ob.Asks[0].Price, "best ask price should match the delta update")
}

func TestWsProcessTrades(t *testing.T) {
	t.Parallel()
	x := newWsTestExchange(t)
	// Enable the trade feed so processed trades are relayed to the data handler.
	x.SetTradeFeedStatus(true)
	x.Websocket.Trade.Setup(true, x.Websocket.DataHandler)
	msg := `{"topic":"recentlyTrade.H.BTCUSDT","type":"snapshot","data":[{"T":1700000000000,"s":"BTC-USDT","S":"BUY","v":"1.5","p":"100.5","L":"PlusTick","i":"trade-1"}],"cs":1,"ts":1700000000000}`
	require.NoError(t, x.wsHandleData(t.Context(), []byte(msg)), "trade message must route without error")
	payload := <-x.Websocket.DataHandler.C
	trades, ok := payload.Data.([]trade.Data)
	require.True(t, ok, "trade topic must produce []trade.Data")
	require.Len(t, trades, 1, "a single trade must be processed")
	assert.Equal(t, 100.5, trades[0].Price, "trade price should be decoded")
	assert.Equal(t, 1.5, trades[0].Amount, "trade amount should be decoded")
	assert.Equal(t, asset.PerpetualContract, trades[0].AssetType, "trade should be filed under asset.PerpetualContract")
}

func TestProcessAccountOrdersAndFills(t *testing.T) {
	t.Parallel()
	x := newWsTestExchange(t)
	orders := []*OrderDetail{{
		ID: "order-1", Symbol: "BTC-USDT", Side: "BUY", OrderType: "LIMIT", Status: "OPEN",
		TimeInForce: "GTC", Price: types.Number(100), Size: types.Number(2), RemainingSize: types.Number(1),
	}}
	require.NoError(t, x.processAccountOrders(t.Context(), orders), "processAccountOrders must not error")
	payload := <-x.Websocket.DataHandler.C
	gotOrders, ok := payload.Data.([]order.Detail)
	require.True(t, ok, "account orders must produce []order.Detail")
	require.Len(t, gotOrders, 1, "a single order must be processed")
	assert.Equal(t, asset.PerpetualContract, gotOrders[0].AssetType, "order should be filed under asset.PerpetualContract")
	assert.Equal(t, order.Open, gotOrders[0].Status, "order status should map to open")

	fills := []*WsAccountOrderFill{{ID: "fill-1", OrderID: "order-1", Symbol: "BTC-USDT", Side: "BUY", Price: types.Number(100), Size: types.Number(1)}}
	require.NoError(t, x.processAccountFills(t.Context(), fills), "processAccountFills must not error")
	payload = <-x.Websocket.DataHandler.C
	gotFills, ok := payload.Data.([]fill.Data)
	require.True(t, ok, "account fills must produce []fill.Data")
	require.Len(t, gotFills, 1, "a single fill must be processed")
	assert.Equal(t, asset.PerpetualContract, gotFills[0].AssetType, "fill should be filed under asset.PerpetualContract")
}
