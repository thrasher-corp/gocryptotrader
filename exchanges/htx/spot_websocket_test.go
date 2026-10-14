package htx

import (
	"context"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

func TestWSHandleCandleMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.kline.1min", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.CandlesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsCandles.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	cAny := <-e.Websocket.DataHandler.C
	c, ok := cAny.Data.(kline.Item)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.Len(t, c.Candles, 1, "Candles must contain a single candle")
	exp := kline.Item{
		Exchange: e.Name,
		Asset:    asset.Spot,
		Pair:     btcusdtPair,
		Interval: 0,
		Candles: []kline.Candle{{
			Time:   time.Unix(1489464480, 0).UTC(),
			Open:   7962.62,
			Close:  8014.56,
			High:   14962.77,
			Low:    5110.14,
			Volume: 4.4,
		}},
	}
	assert.Equal(t, exp, c)
}

func TestWSHandleOrderbookMsg(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		raw  string
		err  error
	}{
		{name: "empty bid", raw: `{"tick":{"bids":[[]]}}`, err: errInvalidBidData},
		{name: "short bid", raw: `{"tick":{"bids":[[100]]}}`, err: errInvalidBidData},
		{name: "long bid", raw: `{"tick":{"bids":[[100,2,3]]}}`, err: errInvalidBidData},
		{name: "empty ask", raw: `{"tick":{"asks":[[]]}}`, err: errInvalidAskData},
		{name: "short ask", raw: `{"tick":{"asks":[[100]]}}`, err: errInvalidAskData},
		{name: "long ask", raw: `{"tick":{"asks":[[100,2,3]]}}`, err: errInvalidAskData},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
			require.ErrorIs(t, h.wsHandleOrderbookMsg(t.Context(), &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}}, []byte(tc.raw)), tc.err, "malformed tuple must be rejected")
			assert.Empty(t, h.Websocket.DataHandler.C, "invalid book should not dispatch")
		})
	}

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.depth.step0", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.OrderbookChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsOrderbook.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	dAny := <-e.Websocket.DataHandler.C
	d, ok := dAny.Data.(*orderbook.Depth)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.NotNil(t, d)
	l, err := d.GetAskLength()
	require.NoError(t, err, "GetAskLength must not error")
	assert.Equal(t, 2, l, "Ask length should be correct")
	liq, _, err := d.TotalAskAmounts()
	require.NoError(t, err, "TotalAskAmount must not error")
	assert.Equal(t, 0.502591, liq, "Ask Liquidity should be correct")
	l, err = d.GetBidLength()
	require.NoError(t, err, "GetBidLength must not error")
	assert.Equal(t, 2, l, "Bid length should be correct")
	liq, _, err = d.TotalBidAmounts()
	require.NoError(t, err, "TotalBidAmount must not error")
	assert.Equal(t, 0.56281, liq, "Bid Liquidity should be correct")
}

// TestWSHandleAllTradesMsg ensures wsHandleAllTrades sends trade.Data to the ws.DataHandler
func TestWSHandleAllTradesMsg(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.trade.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.AllTradesChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	e.SetSaveTradeDataStatus(true)
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	exp := []trade.Data{
		{
			Exchange:     e.Name,
			CurrencyPair: btcusdtPair,
			Timestamp:    time.UnixMilli(1630994963173).UTC(),
			Price:        52648.62,
			Amount:       0.006754,
			Side:         order.Buy,
			TID:          "102523573486",
			AssetType:    asset.Spot,
		},
		{
			Exchange:     e.Name,
			CurrencyPair: btcusdtPair,
			Timestamp:    time.UnixMilli(1630994963184).UTC(),
			Price:        52648.73,
			Amount:       0.006755,
			Side:         order.Sell,
			TID:          "102523573487",
			AssetType:    asset.Spot,
		},
	}
	require.Len(t, e.Websocket.DataHandler.C, 2, "Must see correct number of trades")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
			require.Equalf(t, exp[i], v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T(%s)", v, v)
		}
	}
	require.Empty(t, e.Websocket.DataHandler.C, "Must not see any errors going to datahandler")
}

func TestWSHandleTickerMsg(t *testing.T) {
	t.Parallel()
	t.Run("derivative volume units", func(t *testing.T) {
		t.Parallel()
		e := new(Exchange)
		require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
		e.Name = t.Name()
		sub := &subscription.Subscription{Key: "market.BTC-USD.detail", Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{currency.NewBTCUSD()}, Channel: subscription.TickerChannel}
		require.NoError(t, e.Websocket.AddSubscriptions(e.Websocket.Conn, sub), "AddSubscriptions must not error")

		// Trimmed from the coin margined swap detail channel, where vol is a count of contracts
		payload := []byte(`{"ch":"market.BTC-USD.detail","ts":1630998026649,"tick":{"low":51000,"high":52924.14,"open":51823.62,"close":52379.99,"vol":727676,"amount":13991.028076056185}}`)
		require.NoError(t, e.wsHandleTickerMsg(t.Context(), sub, payload), "wsHandleData must not error")

		e.Websocket.DataHandler.Close()
		require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
		tickAny := <-e.Websocket.DataHandler.C
		tick, ok := tickAny.Data.(*ticker.Price)
		require.True(t, ok, "Must get the correct type from DataHandler")
		assert.Equal(t, 13991.028076056185, tick.BaseVolume, "amount should be recorded as the base volume")
		assert.Zero(t, tick.QuoteVolume, "vol counts contracts here, so no quote volume should be recorded")
		stored, err := ticker.GetTicker(e.Name, tick.Pair, tick.AssetType)
		require.NoError(t, err, "websocket ticker must reach the shared store")
		assert.Equal(t, tick, stored, "stored ticker should match the dispatched ticker")
	})
	for _, tc := range []struct {
		name  string
		asset asset.Item
		raw   string
		quote float64
		err   error
	}{
		{name: "spot quotes", asset: asset.Spot, raw: `{"tick":{"bid":[100,2],"ask":[101,3],"vol":40,"trade_turnover":50}}`, quote: 40},
		{name: "delivery volume", asset: asset.Futures, raw: `{"tick":{"bid":[100,2],"ask":[101,3],"amount":2,"vol":40}}`, quote: 0},
		{name: "coin volume", asset: asset.CoinMarginedFutures, raw: `{"tick":{"bid":[100,2],"ask":[101,3],"amount":2,"vol":40}}`, quote: 0},
		{name: "USDT turnover", asset: asset.USDTMarginedFutures, raw: `{"tick":{"bid":[100,2],"ask":[101,3],"vol":40,"trade_turnover":50}}`, quote: 50},
		{name: "short bid", asset: asset.Spot, raw: `{"tick":{"bid":[100]}}`, err: errInvalidBidData},
		{name: "long ask", asset: asset.Spot, raw: `{"tick":{"ask":[101,3,4]}}`, err: errInvalidAskData},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := new(Exchange)
			require.NoError(t, testexch.Setup(h), "exchange setup must succeed")
			h.Name = t.Name()
			err := h.wsHandleTickerMsg(t.Context(), &subscription.Subscription{Asset: tc.asset, Pairs: currency.Pairs{btcusdtPair}}, []byte(tc.raw))
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "invalid quote tuple must be rejected")
				assert.Empty(t, h.Websocket.DataHandler.C, "invalid tick should not dispatch")
				return
			}
			require.NoError(t, err, "ticker update must succeed")
			stored, err := ticker.GetTicker(h.Name, btcusdtPair, tc.asset)
			require.NoError(t, err, "ticker must reach shared store")
			assert.Equal(t, 100.0, stored.Bid)
			assert.Equal(t, 2.0, stored.BidSize)
			assert.Equal(t, 101.0, stored.Ask)
			assert.Equal(t, 3.0, stored.AskSize)
			assert.Equal(t, tc.quote, stored.QuoteVolume)
		})
	}

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	e.Name = t.Name()
	err := e.Websocket.AddSubscriptions(e.Websocket.Conn, &subscription.Subscription{Key: "market.btcusdt.detail", Asset: asset.Spot, Pairs: currency.Pairs{btcusdtPair}, Channel: subscription.TickerChannel})
	require.NoError(t, err, "AddSubscriptions must not error")
	testexch.FixtureToDataHandler(t, "testdata/wsTicker.json", func(ctx context.Context, data []byte) error { return e.wsHandleData(ctx, e.Websocket.Conn, data) })
	e.Websocket.DataHandler.Close()
	require.Len(t, e.Websocket.DataHandler.C, 1, "Must see correct number of records")
	tickAny := <-e.Websocket.DataHandler.C
	tick, ok := tickAny.Data.(*ticker.Price)
	require.True(t, ok, "Must get the correct type from DataHandler")
	require.NotNil(t, tick)
	exp := &ticker.Price{
		High:         52924.14,
		Low:          51000,
		Bid:          0,
		BaseVolume:   13991.028076056185,
		QuoteVolume:  7.27676440200527e+08,
		Open:         51823.62,
		Last:         52379.99,
		Close:        52379.99,
		Pair:         btcusdtPair,
		ExchangeName: e.Name,
		AssetType:    asset.Spot,
		LastUpdated:  time.UnixMilli(1630998026649),
	}
	assert.Equal(t, exp, tick)
	stored, err := ticker.GetTicker(e.Name, tick.Pair, tick.AssetType)
	require.NoError(t, err, "websocket ticker must reach the shared store")
	assert.Equal(t, tick, stored, "stored ticker should match the dispatched ticker")
}

func TestWSHandleV1Ping(t *testing.T) {
	t.Parallel()
	messages := make(chan []byte, 1)
	h := testexch.MockWsInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, func(_ testing.TB, msg []byte, _ *gws.Conn) error {
		messages <- msg
		return nil
	}))
	conn, err := h.Websocket.GetConnection(exchange.WebsocketSpot)
	require.NoError(t, err, "spot websocket connection must be available")
	require.NoError(t, h.wsHandleV1ping(t.Context(), conn, 123), "V1 ping handler must not error")
	select {
	case msg := <-messages:
		assert.JSONEq(t, `{"pong":123}`, string(msg), "V1 ping handler should send the matching pong")
	case <-time.After(time.Second):
		require.Fail(t, "V1 pong must be sent")
	}
	require.ErrorIs(t, h.wsHandleV1ping(t.Context(), nil, 123), common.ErrNilPointer, "V1 ping handler must reject a nil connection")
}
