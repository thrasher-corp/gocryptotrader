package lbank

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/asset"
	"github.com/thrasher-corp/gocryptotrader/exchange/kline"
	"github.com/thrasher-corp/gocryptotrader/exchange/order"
	"github.com/thrasher-corp/gocryptotrader/exchange/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchange/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchange/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestWebsocketPublicHandlers(t *testing.T) {
	t.Parallel()
	ex := newChannelTestExchange(t)
	conn := &recordingConnection{}
	testexch.FixtureToDataHandler(t, "testdata/wsPublic.jsonl", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, conn, b)
	})
	pair, err := ex.MatchSymbolWithAvailablePairs("btc_usdt", asset.Spot, true)
	require.NoError(t, err, "MatchSymbolWithAvailablePairs must find btc_usdt")

	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 4, "the handlers must relay the ticker, both trades and the candle")
	expTicker := &ticker.Price{
		ExchangeName: ex.Name, Pair: pair, AssetType: asset.Spot,
		Last: 80802.71, High: 81092.52, Low: 79270.99, BaseVolume: 2980.851, QuoteVolume: 2.3975039315e8, PercentChange24Hour: 0.1,
		LastUpdated: time.Date(2026, 10, 25, 8, 23, 20, 879000000, beijingTime),
	}
	assert.Equal(t, expTicker, payloads[0], "the ticker handler should relay the ticker")
	stored, err := ticker.GetTicker(ex.Name, pair, asset.Spot)
	require.NoError(t, err, "the ticker handler must store the ticker")
	assert.Equal(t, expTicker.Last, stored.Last, "the ticker handler should store the ticker it relays")

	book, err := ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	require.NoError(t, err, "the depth handler must load the order book")
	expBook := &orderbook.Book{
		Exchange: ex.Name, Pair: pair, Asset: asset.Spot,
		Asks: orderbook.Levels{
			{Price: 80873.08, Amount: 6.31598},
			{Price: 80873.09, Amount: 0.00054},
			{Price: 80873.25, Amount: 0.00447},
			{Price: 80873.8, Amount: 0.00016},
			{Price: 80873.81, Amount: 0.05055},
			{Price: 80874.37, Amount: 0.00016},
			{Price: 80874.38, Amount: 0.16577},
		},
		Bids: orderbook.Levels{
			{Price: 80873.07, Amount: 2.57801},
			{Price: 80873.06, Amount: 0.00086},
			{Price: 80872.88, Amount: 0.00033},
			{Price: 80872.87, Amount: 0.04936},
			{Price: 80872.44, Amount: 0.00017},
			{Price: 80872.43, Amount: 0.15375},
			{Price: 80872.32, Amount: 0.00009},
		},
		LastUpdated:       time.UnixMilli(1792888874775),
		LastPushed:        time.Date(2026, 10, 25, 8, 41, 14, 979000000, beijingTime),
		ValidateOrderbook: true,
	}
	require.NotZero(t, book.InsertedAt, "the depth handler must record when the book was loaded")
	book.InsertedAt = time.Time{}
	assert.Equal(t, expBook, book, "the depth handler should load the book")

	expTrades := []any{
		[]trade.Data{{
			Exchange: ex.Name, CurrencyPair: pair, AssetType: asset.Spot, Side: order.Sell, Price: 80802.71, Amount: 0.0024,
			Timestamp: time.Date(2026, 10, 25, 8, 23, 20, 872000000, beijingTime),
		}},
		[]trade.Data{{
			Exchange: ex.Name, CurrencyPair: pair, AssetType: asset.Spot, Side: order.Buy, Price: 80802.72, Amount: 0.00003,
			Timestamp: time.Date(2026, 10, 25, 8, 23, 22, 701000000, beijingTime),
		}},
	}
	assert.Equal(t, expTrades, payloads[1:3], "the trade handler should relay each trade")

	expKline := &kline.Item{
		Exchange: ex.Name, Pair: pair, Asset: asset.Spot, Interval: kline.OneMin,
		Candles: []kline.Candle{{Time: time.Date(2026, 10, 25, 8, 23, 0, 0, beijingTime), Open: 80797.03, High: 80802.72, Low: 80797.03, Close: 80802.71, Volume: 0.087}},
	}
	assert.Equal(t, expKline, payloads[3], "the candle handler should relay the candle")
	assert.Empty(t, conn.messages(), "the handlers should send nothing back")
}

func TestIntervalFromKlineSlot(t *testing.T) {
	t.Parallel()
	for interval, slot := range websocketKlineSlots {
		t.Run(slot, func(t *testing.T) {
			t.Parallel()
			got, err := intervalFromKlineSlot(slot)
			require.NoErrorf(t, err, "intervalFromKlineSlot must not error for %s", slot)
			assert.Equalf(t, interval, got, "intervalFromKlineSlot should map %s back to its interval", slot)
		})
	}
	t.Run("slot without an interval", func(t *testing.T) {
		t.Parallel()
		_, err := intervalFromKlineSlot("year")
		assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "intervalFromKlineSlot should reject a slot without an interval")
	})
}
