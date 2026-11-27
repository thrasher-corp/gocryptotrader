package okx

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	websocketFuturesPair = currency.NewPairWithDelimiter("BTC", "USD-261127", currency.DashDelimiter)
	websocketOptionPair  = currency.NewPairWithDelimiter("BTC", "USD-261127-71000-P", currency.DashDelimiter)
)

// newWebsocketHandlerExchange returns an exchange with the instruments the
// websocket fixtures push enabled, so pushes map to their assets
func newWebsocketHandlerExchange(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	enableWebsocketTestPair(t, ex, asset.PerpetualSwap, perpetualSwapPair)
	enableWebsocketTestPair(t, ex, asset.Futures, websocketFuturesPair)
	enableWebsocketTestPair(t, ex, asset.Options, websocketOptionPair)
	return ex
}

// enableWebsocketTestPair makes a pair available and enabled for an asset
func enableWebsocketTestPair(t *testing.T, ex *Exchange, a asset.Item, p currency.Pair) {
	t.Helper()
	available, err := ex.CurrencyPairs.GetPairs(a, false)
	require.NoError(t, err, "GetPairs must not error")
	require.NoError(t, ex.CurrencyPairs.StorePairs(a, available.Add(p), false), "StorePairs must not error")
	require.NoError(t, ex.CurrencyPairs.EnablePair(a, p), "EnablePair must not error")
}

// relayed drains and returns what the exchange relayed to its data handler
func relayed(ex *Exchange) []any {
	var got []any
	for len(ex.Websocket.DataHandler.C) > 0 {
		got = append(got, (<-ex.Websocket.DataHandler.C).Data)
	}
	return got
}

// relayPauseContext holds up the first read of its values once ready reports
// true, until it is resumed or ends: a relay reads its context's values as it
// sends, so a push's handling stops at its relay once its books are loaded
type relayPauseContext struct {
	context.Context //nolint:containedctx // relayPauseContext implements context.Context
	ready           func() bool
	paused          chan struct{}
	release         chan struct{}
	pause           sync.Once
	resumed         sync.Once
}

func (c *relayPauseContext) Value(key any) any {
	if c.ready() {
		c.pause.Do(func() {
			close(c.paused)
			select {
			case <-c.release:
			case <-c.Done():
			}
		})
	}
	return c.Context.Value(key)
}

// resume releases the read held up
func (c *relayPauseContext) resume() {
	c.resumed.Do(func() { close(c.release) })
}

// handleFixture feeds a fixture's frames through the frame router
func handleFixture(t *testing.T, ex *Exchange, fixture string) {
	t.Helper()
	testexch.FixtureToDataHandler(t, "testdata/websocket/"+fixture+".jsonl", func(ctx context.Context, b []byte) error {
		return ex.wsHandleData(ctx, nil, b)
	})
}

func TestPushAssets(t *testing.T) {
	t.Parallel()
	ex := newWebsocketHandlerExchange(t)
	for _, tc := range []struct {
		name   string
		arg    SubscriptionArgument
		exp    []asset.Item
		expErr error
	}{
		{name: "instrument type", arg: SubscriptionArgument{InstrumentType: instTypeSwap, InstrumentID: mainPair}, exp: []asset.Item{asset.PerpetualSwap}},
		{name: "instrument enabled on spot and margin", arg: SubscriptionArgument{InstrumentID: mainPair}, exp: []asset.Item{asset.Spot, asset.Margin}},
		{name: "unknown instrument type", arg: SubscriptionArgument{InstrumentType: "EVENTS"}, expErr: asset.ErrNotSupported},
		{name: "instrument not enabled", arg: SubscriptionArgument{InstrumentID: currency.NewPairWithDelimiter("XRP", "USDT", currency.DashDelimiter)}, expErr: asset.ErrNotEnabled},
		{name: "no instrument", arg: SubscriptionArgument{}, expErr: errMissingInstrumentID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ex.pushAssets(&tc.arg)
			require.ErrorIs(t, err, tc.expErr, "pushAssets must return the expected error")
			assert.Equal(t, tc.exp, got, "pushAssets should return the push's assets")
		})
	}
}

func TestWebsocketProcessTickers(t *testing.T) {
	t.Parallel()

	t.Run("live captures", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketHandlerExchange(t)
		handleFixture(t, ex, "tickers")
		spot := ticker.Price{
			Last: 61507.1, High: 61964.8, Low: 58768.1, Bid: 61507.7, BidSize: 0.75724859, Ask: 61507.8, AskSize: 0.30454134,
			BaseVolume: 8856.83260534, QuoteVolume: 536582858.8106283, Open: 59218.4, Pair: mainPair, ExchangeName: ex.Name,
			AssetType: asset.Spot, LastUpdated: time.UnixMilli(1795405194074),
		}
		margin := spot
		margin.AssetType = asset.Margin
		exp := []any{
			[]ticker.Price{spot, margin},
			[]ticker.Price{{
				Last: 61479.8, High: 61943, Low: 58731, Bid: 61479.8, BidSize: 753.95, Ask: 61479.9, AskSize: 385.53, BaseVolume: 123998.2135,
				Open: 59187.4, Pair: perpetualSwapPair, ExchangeName: ex.Name, AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1795405194074),
			}},
			[]ticker.Price{{
				Last: 61380.1, High: 61840.6, Low: 58603.8, Bid: 61383.6, BidSize: 383.7, Ask: 61383.7, AskSize: 2.9, BaseVolume: 7434.4158,
				Open: 59045.2, Pair: currency.NewPairWithDelimiter("BTC", "USD-SWAP", currency.DashDelimiter), ExchangeName: ex.Name,
				AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1795405194070),
			}},
			[]ticker.Price{{
				Last: 61516.1, High: 61837.1, Low: 58673.9, Bid: 61417.1, BidSize: 5.5, Ask: 61444.3, AskSize: 0.2, BaseVolume: 29.2631,
				Open: 59139.7, Pair: websocketFuturesPair, ExchangeName: ex.Name, AssetType: asset.Futures, LastUpdated: time.UnixMilli(1795405194072),
			}},
		}
		assert.Equal(t, exp, relayed(ex), "each ticker push should relay its processed tickers")
	})

	t.Run("instrument type on the argument", func(t *testing.T) {
		t.Parallel()
		// The fixture's third frame is constructed: OKX does not send instType on
		// this argument, and it selects the asset without the instrument lookup.
		// The swap's vol24h of 1170000 contracts is 1500 BTC at ctVal 100 USD, so
		// it carries a base volume and no quote volume, where spot and margin
		// carry both.
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		testexch.FixtureToDataHandler(t, "testdata/wsTickers.jsonl", func(ctx context.Context, b []byte) error { return ex.wsProcessTickers(ctx, b) })
		spot := ticker.Price{
			Last: 78000, High: 79000, Low: 76000, Bid: 77999, BidSize: 2, Ask: 78001, AskSize: 1, Open: 77000, BaseVolume: 5000,
			QuoteVolume: 400000000, Pair: mainPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1740394561685),
		}
		margin := spot
		margin.AssetType = asset.Margin
		constructed := spot
		constructed.LastUpdated = time.UnixMilli(1740394561687)
		exp := []any{
			[]ticker.Price{spot, margin},
			[]ticker.Price{{
				Last: 78000, High: 79000, Low: 76000, Bid: 77999, BidSize: 2, Ask: 78001, AskSize: 1, Open: 77000, BaseVolume: 1500,
				Pair: currency.NewPairWithDelimiter("BTC", "USD-SWAP", currency.DashDelimiter), ExchangeName: ex.Name,
				AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1740394561686),
			}},
			[]ticker.Price{constructed},
		}
		assert.Equal(t, exp, relayed(ex), "each ticker should map its volumes and 24 hour prices for its assets")
	})

	t.Run("empty data relays nothing", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		for _, data := range []string{`[]`, `null`} {
			require.NoError(t, ex.wsProcessTickers(t.Context(), []byte(`{"arg":{"channel":"tickers","instType":"SPOT"},"data":`+data+`}`)), "empty ticker data must not error")
			assert.Empty(t, ex.Websocket.DataHandler.C, "empty ticker data should not be relayed")
		}
	})

	t.Run("relays the tickers of a batch which processed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.wsProcessTickers(t.Context(), []byte(`{"arg":{"channel":"tickers","instType":"SPOT"},"data":[{"instId":"BTC-USDT","askPx":"2","bidPx":"1"},{"instId":"ETH-USDT","askPx":"1","bidPx":"2"}]}`))
		require.Error(t, err, "a batch with a crossed ticker must error")
		got := relayed(ex)
		require.Len(t, got, 1, "the batch must be relayed once")
		prices, ok := got[0].([]ticker.Price)
		require.True(t, ok, "the batch must be relayed as processed tickers")
		require.Len(t, prices, 1, "only the ticker which processed must be relayed")
		// A ticker without a time is stamped when it is processed
		assert.WithinDuration(t, time.Now(), prices[0].LastUpdated, time.Minute, "the ticker should be stamped when processed")
		prices[0].LastUpdated = time.Time{}
		assert.Equal(t, ticker.Price{Bid: 1, Ask: 2, Pair: mainPair, ExchangeName: ex.Name, AssetType: asset.Spot}, prices[0], "only the ticker which processed should be relayed")
	})

	t.Run("unknown instrument type", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		err := ex.wsProcessTickers(t.Context(), []byte(`{"arg":{"channel":"tickers","instType":"EVENTS"},"data":[{"instId":"BTC-USDT"}]}`))
		require.ErrorIs(t, err, asset.ErrNotSupported, "an unknown instrument type must error")
		assert.Empty(t, ex.Websocket.DataHandler.C, "nothing should be relayed")
	})

	t.Run("pre-open instrument", func(t *testing.T) {
		t.Parallel()
		instrumentsPush := func(state string) string {
			return `{"arg":{"channel":"instruments","instType":"SPOT"},"data":[{"instType":"SPOT","instId":"BTC-USDT","instIdCode":31,"state":"` + state + `"}]}`
		}
		for _, tc := range []struct {
			name     string
			states   []string
			bid, ask float64
			expErr   error
		}{
			{name: "a crossed ticker is stored and relayed", states: []string{"preopen"}, bid: 101, ask: 99},
			{name: "a locked ticker is stored and relayed", states: []string{"preopen"}, bid: 100, ask: 100},
			{name: "a live instrument's locked ticker is refused", states: []string{"live"}, bid: 100, ask: 100, expErr: ticker.ErrBidEqualsAsk},
			{name: "an instrument which went live is refused a locked ticker", states: []string{"preopen", "live"}, bid: 100, ask: 100, expErr: ticker.ErrBidEqualsAsk},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Name = t.Name()
				for _, state := range tc.states {
					require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(instrumentsPush(state))), "the instruments push must not error")
				}
				relayed(ex)
				err := ex.wsProcessTickers(t.Context(), fmt.Appendf(nil, `{"arg":{"channel":"tickers","instId":"BTC-USDT"},"data":[{"instId":"BTC-USDT","bidPx":"%v","askPx":"%v","ts":"1700000001000"}]}`, tc.bid, tc.ask))
				require.ErrorIs(t, err, tc.expErr, "wsProcessTickers must return the expected error")
				if tc.expErr != nil {
					assert.Empty(t, relayed(ex), "a refused ticker should not be relayed")
					return
				}
				spot := ticker.Price{Bid: tc.bid, Ask: tc.ask, Pair: mainPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1700000001000), PreOpen: true}
				margin := spot
				margin.AssetType = asset.Margin
				assert.Equal(t, []any{[]ticker.Price{spot, margin}}, relayed(ex), "the pre-open ticker should be relayed for each asset")
				stored, err := ticker.GetTicker(ex.Name, mainPair, asset.Spot)
				require.NoError(t, err, "GetTicker must not error")
				assert.Equal(t, &spot, stored, "the pre-open ticker should be stored")
			})
		}
	})

	t.Run("pre-open at the ticker's time", func(t *testing.T) {
		t.Parallel()
		const switchTime int64 = 1700000001000
		instrumentsPush := func(state string) string {
			return fmt.Sprintf(`{"arg":{"channel":"instruments","instType":"SPOT"},"data":[{"instType":"SPOT","instId":"BTC-USDT","instIdCode":31,"state":%q,"openType":"call_auction","listTime":"1700000000000","contTdSwTime":"%d"}]}`, state, switchTime)
		}
		for _, tc := range []struct {
			name       string
			states     []string
			at         int64
			bid, ask   float64
			expPreOpen bool
			expErr     error
		}{
			{name: "an auction quote arriving after the live push is pre-open", states: []string{"preopen", "live"}, at: switchTime - 1, bid: 101, ask: 99, expPreOpen: true},
			{name: "a quote after the switch is live while the last push is pre-open", states: []string{"preopen"}, at: switchTime + 1, bid: 100, ask: 101},
			{name: "a locked quote after the switch is refused while the last push is pre-open", states: []string{"preopen"}, at: switchTime + 1, bid: 100, ask: 100, expErr: ticker.ErrBidEqualsAsk},
			{name: "a locked quote at the switch is refused", states: []string{"preopen"}, at: switchTime, bid: 100, ask: 100, expErr: ticker.ErrBidEqualsAsk},
			{name: "a locked quote before the switch is pre-open", states: []string{"preopen"}, at: switchTime - 1, bid: 100, ask: 100, expPreOpen: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Name = t.Name()
				for _, state := range tc.states {
					require.NoError(t, ex.wsHandleData(t.Context(), nil, []byte(instrumentsPush(state))), "the instruments push must not error")
				}
				relayed(ex)
				err := ex.wsProcessTickers(t.Context(), fmt.Appendf(nil, `{"arg":{"channel":"tickers","instId":"BTC-USDT"},"data":[{"instId":"BTC-USDT","bidPx":"%v","askPx":"%v","ts":"%d"}]}`, tc.bid, tc.ask, tc.at))
				require.ErrorIs(t, err, tc.expErr, "wsProcessTickers must return the expected error")
				if tc.expErr != nil {
					assert.Empty(t, relayed(ex), "a refused ticker should not be relayed")
					return
				}
				spot := ticker.Price{Bid: tc.bid, Ask: tc.ask, Pair: mainPair, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(tc.at), PreOpen: tc.expPreOpen}
				margin := spot
				margin.AssetType = asset.Margin
				assert.Equal(t, []any{[]ticker.Price{spot, margin}}, relayed(ex), "the ticker should be relayed for each asset with the state at its time")
			})
		}
	})

	t.Run("pre-open until the listing time of an instrument without a continuous trading switch", func(t *testing.T) {
		t.Parallel()
		const listTime int64 = 1700000001000
		for _, tc := range []struct {
			name       string
			instType   string
			pair       currency.Pair
			assets     []asset.Item
			states     []string
			unlisted   bool
			at         int64
			bid, ask   float64
			expPreOpen bool
			expErr     error
		}{
			{name: "a crossed futures quote is pre-open while the last listing is, without a listing time", instType: instTypeFutures, pair: websocketFuturesPair, assets: []asset.Item{asset.Futures}, states: []string{"preopen"}, unlisted: true, at: listTime + 1, bid: 101, ask: 99, expPreOpen: true},
			{name: "a futures quote at its listing time is live while the last listing is pre-open", instType: instTypeFutures, pair: websocketFuturesPair, assets: []asset.Item{asset.Futures}, states: []string{"preopen"}, at: listTime, bid: 99, ask: 101},
			{name: "a locked futures quote after its listing time is refused while the last listing is pre-open", instType: instTypeFutures, pair: websocketFuturesPair, assets: []asset.Item{asset.Futures}, states: []string{"preopen"}, at: listTime + 1, bid: 100, ask: 100, expErr: ticker.ErrBidEqualsAsk},
			{name: "a crossed futures quote before its listing time arriving after the live push is pre-open", instType: instTypeFutures, pair: websocketFuturesPair, assets: []asset.Item{asset.Futures}, states: []string{"preopen", "live"}, at: listTime - 1, bid: 101, ask: 99, expPreOpen: true},
			{name: "a crossed swap quote before its listing time is pre-open", instType: instTypeSwap, pair: perpetualSwapPair, assets: []asset.Item{asset.PerpetualSwap}, states: []string{"live"}, at: listTime - 1, bid: 101, ask: 99, expPreOpen: true},
			{name: "a crossed spot quote before its listing time is pre-open", instType: instTypeSpot, pair: mainPair, assets: []asset.Item{asset.Spot, asset.Margin}, states: []string{"live"}, at: listTime - 1, bid: 101, ask: 99, expPreOpen: true},
			{name: "a crossed option quote after its listing time is pre-open while the last listing is", instType: instTypeOption, pair: websocketOptionPair, assets: []asset.Item{asset.Options}, states: []string{"preopen"}, at: listTime + 1, bid: 101, ask: 99, expPreOpen: true},
			{name: "a locked option quote before its listing time is refused once the option is live", instType: instTypeOption, pair: websocketOptionPair, assets: []asset.Item{asset.Options}, states: []string{"preopen", "live"}, at: listTime - 1, bid: 100, ask: 100, expErr: ticker.ErrBidEqualsAsk},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketHandlerExchange(t)
				ex.Name = t.Name()
				listed := listTime
				if tc.unlisted {
					listed = 0
				}
				for _, state := range tc.states {
					push := fmt.Appendf(nil, `{"arg":{"channel":"instruments","instType":%q},"data":[{"instType":%q,"instId":%q,"instIdCode":31,"state":%q,"listTime":"%d","contTdSwTime":""}]}`, tc.instType, tc.instType, tc.pair, state, listed)
					require.NoError(t, ex.wsHandleData(t.Context(), nil, push), "the instruments push must not error")
				}
				relayed(ex)
				err := ex.wsProcessTickers(t.Context(), fmt.Appendf(nil, `{"arg":{"channel":"tickers","instId":%q},"data":[{"instId":%q,"instType":%q,"bidPx":"%v","askPx":"%v","ts":"%d"}]}`, tc.pair, tc.pair, tc.instType, tc.bid, tc.ask, tc.at))
				require.ErrorIs(t, err, tc.expErr, "wsProcessTickers must return the expected error")
				if tc.expErr != nil {
					assert.Empty(t, relayed(ex), "a refused ticker should not be relayed")
					return
				}
				exp := make([]ticker.Price, 0, len(tc.assets))
				for _, a := range tc.assets {
					exp = append(exp, ticker.Price{Bid: tc.bid, Ask: tc.ask, Pair: tc.pair, ExchangeName: ex.Name, AssetType: a, LastUpdated: time.UnixMilli(tc.at), PreOpen: tc.expPreOpen})
				}
				assert.Equal(t, []any{exp}, relayed(ex), "the ticker should be relayed with the state at its time")
			})
		}
	})
}

func TestWebsocketProcessCandles(t *testing.T) {
	t.Parallel()

	t.Run("live captures", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			fixture  string
			interval kline.Interval
			candles  []kline.Candle
		}{
			{"candle1m", kline.OneMin, []kline.Candle{
				{Time: time.UnixMilli(1795405200000), Open: 61502.4, High: 61502.4, Low: 61489.3, Close: 61491.4, Volume: 0.00161768},
				{Time: time.UnixMilli(1795405200000), Open: 61502.4, High: 61502.4, Low: 61400.1, Close: 61400.1, Volume: 9.40866285},
			}},
			{"candle1D", kline.OneDay, []kline.Candle{{Time: time.UnixMilli(1795363200000), Open: 60890.5, High: 61964.8, Low: 60380.8, Close: 61491.4, Volume: 3649.08289809}}},
			{"candle1s", kline.ThousandMilliseconds, []kline.Candle{{Time: time.UnixMilli(1795405496000), Open: 61294.1, High: 61294.1, Low: 61289.6, Close: 61289.7, Volume: 0.02483911}}},
		} {
			t.Run(tc.fixture, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketHandlerExchange(t)
				handleFixture(t, ex, tc.fixture)
				var exp []any
				for _, c := range tc.candles {
					for _, a := range []asset.Item{asset.Spot, asset.Margin} {
						exp = append(exp, kline.Item{Exchange: ex.Name, Pair: mainPair, Asset: a, Interval: tc.interval, Candles: []kline.Candle{c}})
					}
				}
				assert.Equal(t, exp, relayed(ex), "each candlestick should be relayed for every asset its instrument is enabled for")
			})
		}
	})

	t.Run("instrument type on the argument", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.wsProcessCandles(t.Context(), []byte(`{"arg":{"channel":"candle1D","instType":"SPOT","instId":"BTC-USDT"},"data":[["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`)), "wsProcessCandles must not error")
		exp := []any{kline.Item{
			Pair:     mainPair,
			Asset:    asset.Spot,
			Exchange: ex.Name,
			Interval: kline.OneDay,
			Candles:  []kline.Candle{{Time: time.UnixMilli(1597026383085), Open: 8533.02, Close: 8548.26, High: 8553.74, Low: 8527.17, Volume: 45247}},
		}}
		assert.Equal(t, exp, relayed(ex), "the candlestick should be relayed for the argument's instrument type")
	})

	t.Run("a derivative candlestick reports its base currency volume", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.NoError(t, ex.wsProcessCandles(t.Context(), []byte(`{"arg":{"channel":"candle1H","instType":"SWAP","instId":"BTC-USD-SWAP"},"data":[["1795410000000","61126.5","61717.8","60989","61563.7","172262.8","280.4185","17226280","1"]]}`)), "wsProcessCandles must not error")
		exp := []any{kline.Item{
			Pair:     btcUSDSwapPair,
			Asset:    asset.PerpetualSwap,
			Exchange: ex.Name,
			Interval: kline.OneHour,
			Candles:  []kline.Candle{{Time: time.UnixMilli(1795410000000), Open: 61126.5, High: 61717.8, Low: 60989, Close: 61563.7, Volume: 280.4185}},
		}}
		assert.Equal(t, exp, relayed(ex), "the candlestick should report volCcy, the base currency volume, rather than its contract count")
	})

	for _, tc := range []struct {
		name   string
		frame  string
		expErr error
	}{
		{"empty data", `{"arg":{"channel":"candle1D","instId":"BTC-USDT"},"data":[]}`, kline.ErrNoTimeSeriesDataToConvert},
		{"undocumented interval", `{"arg":{"channel":"candle1Y","instId":"BTC-USDT"},"data":[["1597026383085","1","1","1","1","1","1","1","0"]]}`, kline.ErrInvalidInterval},
		{"instrument not enabled", `{"arg":{"channel":"candle1D","instId":"XRP-USDT"},"data":[["1597026383085","1","1","1","1","1","1","1","0"]]}`, asset.ErrNotEnabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.ErrorIs(t, ex.wsProcessCandles(t.Context(), []byte(tc.frame)), tc.expErr, "wsProcessCandles must return the expected error")
			assert.Empty(t, ex.Websocket.DataHandler.C, "nothing should be relayed")
		})
	}
}

func TestRelayTrades(t *testing.T) {
	t.Parallel()
	trades := []trade.Data{{TID: "1", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Buy, Price: 1, Amount: 1, Timestamp: time.UnixMilli(1795405194320)}}
	// The zero amount trade is rejected by the trade buffer, which shows the buffer was reached
	unsaveable := []trade.Data{{TID: "2", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Buy, Price: 1, Timestamp: time.UnixMilli(1795405194320)}}
	// The trade buffer normalises a bid to a buy and gives the trade an ID in place
	unnormalised := []trade.Data{{TID: "3", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Bid, Price: 1, Amount: 1, Timestamp: time.UnixMilli(1795405194320)}}
	for _, tc := range []struct {
		name           string
		feed, save     bool
		fullRelay      bool
		relayRoom      uint
		trades         []trade.Data
		expRelayed     []any
		expErrContains []string
		expBuffered    []bool
	}{
		{name: "trade feed", feed: true, trades: trades, expRelayed: []any{trades[0]}, expBuffered: []bool{false}},
		{name: "neither feed nor saving", trades: trades, expBuffered: []bool{false}},
		{name: "saving", save: true, trades: unsaveable, expErrContains: []string{"received invalid trade data"}, expBuffered: []bool{false}},
		{name: "trade feed and saving relay the trades as received", feed: true, save: true, trades: unnormalised, expRelayed: []any{unnormalised[0]}, expBuffered: []bool{true}},
		{
			name: "a relay filling during the trades still saves them all", feed: true, save: true, relayRoom: 1,
			trades:         []trade.Data{{TID: "5", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Buy, Price: 1, Amount: 1, Timestamp: time.UnixMilli(1795405194320)}, {TID: "6", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Sell, Price: 2, Amount: 1, Timestamp: time.UnixMilli(1795405194321)}},
			expRelayed:     []any{trade.Data{TID: "5", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Buy, Price: 1, Amount: 1, Timestamp: time.UnixMilli(1795405194320)}},
			expErrContains: []string{"channel buffer is full"},
			expBuffered:    []bool{true, true},
		},
		{
			name: "a full relay still saves", feed: true, save: true, fullRelay: true,
			trades:         []trade.Data{{TID: "4", Exchange: "Okx", CurrencyPair: mainPair, AssetType: asset.Spot, Side: order.Buy, Price: 1, Amount: 1, Timestamp: time.UnixMilli(1795405194320)}, unsaveable[0]},
			expRelayed:     []any{struct{}{}},
			expErrContains: []string{"channel buffer is full", "received invalid trade data"},
			expBuffered:    []bool{true, false},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.SetTradeFeedStatus(tc.feed)
			ex.SetSaveTradeDataStatus(tc.save)
			if tc.fullRelay {
				ex.Websocket.DataHandler = stream.NewRelay(1)
				require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), struct{}{}), "filling the relay must not error")
			}
			if tc.relayRoom != 0 {
				ex.Websocket.DataHandler = stream.NewRelay(tc.relayRoom)
			}
			err := ex.relayTrades(t.Context(), tc.trades)
			if len(tc.expErrContains) == 0 {
				require.NoError(t, err, "relayTrades must not error")
			}
			for _, s := range tc.expErrContains {
				assert.ErrorContains(t, err, s, "relayTrades should report the failure of both relaying and saving")
			}
			assert.Equal(t, tc.expRelayed, relayed(ex), "relayTrades should relay trades only to an enabled trade feed")
			saved := trade.BufferedTrades()
			buffered := make([]bool, len(tc.trades))
			for i := range tc.trades {
				buffered[i] = slices.Contains(saved, tc.trades[i])
			}
			assert.Equal(t, tc.expBuffered, buffered, "relayTrades should buffer each valid trade when saving")
		})
	}
}

func TestWebsocketProcessTrades(t *testing.T) {
	t.Parallel()

	t.Run("live captures", func(t *testing.T) {
		t.Parallel()
		for fixture, rows := range map[string][]trade.Data{
			"trades": {
				{TID: "1089297792", Side: order.Buy, Price: 61512.6, Amount: 0.00105628, Timestamp: time.UnixMilli(1795405194320).UTC()},
				{TID: "1089297793", Side: order.Sell, Price: 61513, Amount: 0.00001, Timestamp: time.UnixMilli(1795405195366).UTC()},
			},
			"trades-all": {
				{TID: "1089297835", Side: order.Sell, Price: 61492.6, Amount: 0.00001, Timestamp: time.UnixMilli(1795405202317).UTC()},
				{TID: "1089297836", Side: order.Sell, Price: 61486.3, Amount: 0.00001, Timestamp: time.UnixMilli(1795405203520).UTC()},
			},
		} {
			t.Run(fixture, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketHandlerExchange(t)
				ex.SetTradeFeedStatus(true)
				handleFixture(t, ex, fixture)
				var exp []any
				for _, row := range rows {
					for _, a := range []asset.Item{asset.Spot, asset.Margin} {
						row.Exchange, row.CurrencyPair, row.AssetType = ex.Name, mainPair, a
						exp = append(exp, row)
					}
				}
				assert.Equal(t, exp, relayed(ex), "each trade should be relayed for every asset its instrument is enabled for")
			})
		}
	})

	t.Run("trades for spot and margin", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
		ex.SetTradeFeedStatus(true)
		testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.jsonl", func(ctx context.Context, b []byte) error { return ex.wsHandleData(ctx, nil, b) })
		exp := make([]any, 0, 4)
		for _, row := range []trade.Data{
			{Timestamp: time.UnixMilli(1740394561685).UTC(), Price: 95634.9, Amount: 0.00011186, Side: order.Buy, TID: "674510826"},
			{Timestamp: time.UnixMilli(1740394561686).UTC(), Price: 95635.3, Amount: 0.00011194, Side: order.Sell, TID: "674510827"},
		} {
			for _, a := range []asset.Item{asset.Spot, asset.Margin} {
				row.Exchange, row.CurrencyPair, row.AssetType = ex.Name, mainPair, a
				exp = append(exp, row)
			}
		}
		assert.Equal(t, exp, relayed(ex), "every trade should be relayed for spot and margin")
	})

	t.Run("feeds off", func(t *testing.T) {
		t.Parallel()
		ex := newWebsocketHandlerExchange(t)
		ex.SetTradeFeedStatus(false)
		ex.SetSaveTradeDataStatus(false)
		handleFixture(t, ex, "trades")
		assert.Empty(t, ex.Websocket.DataHandler.C, "no trade should be relayed with the trade feed off")
	})

	t.Run("instrument not enabled", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.SetTradeFeedStatus(true)
		err := ex.wsProcessTrades(t.Context(), []byte(`{"arg":{"channel":"trades","instId":"XRP-USDT"},"data":[{"instId":"XRP-USDT","tradeId":"1","px":"1","sz":"1","side":"buy","ts":"1795405194320"}]}`))
		require.ErrorIs(t, err, asset.ErrNotEnabled, "a trade for an instrument which is not enabled must error")
	})
}

func TestWebsocketProcessOptionTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketHandlerExchange(t)
	ex.SetTradeFeedStatus(true)
	handleFixture(t, ex, "option-trades")
	exp := []any{trade.Data{
		TID: "116", Exchange: ex.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "USD-261225-60000-C", currency.DashDelimiter),
		AssetType: asset.Options, Side: order.Sell, Price: 0.072, Amount: 220, Timestamp: time.UnixMilli(1795405205844),
	}}
	assert.Equal(t, exp, relayed(ex), "the option trade should be relayed")

	ex.SetTradeFeedStatus(false)
	handleFixture(t, ex, "option-trades")
	assert.Empty(t, ex.Websocket.DataHandler.C, "no trade should be relayed with the trade feed off")
}

func TestWebsocketProcessPublicBlockTrades(t *testing.T) {
	t.Parallel()
	ex := newWebsocketHandlerExchange(t)
	ex.SetTradeFeedStatus(true)
	handleFixture(t, ex, "public-block-trades")
	exp := []any{trade.Data{
		TID: "4113940107896115203", Exchange: ex.Name, CurrencyPair: websocketOptionPair, AssetType: asset.Options,
		Side: order.Sell, Price: 0.1281, Amount: 20, Timestamp: time.UnixMilli(1795107369390),
	}}
	assert.Equal(t, exp, relayed(ex), "the block trade leg should be relayed as its instrument's trade")

	err := ex.wsProcessPublicBlockTrades(t.Context(), []byte(`{"arg":{"channel":"public-block-trades","instId":"XRP-USDT"},"data":[]}`))
	require.ErrorIs(t, err, asset.ErrNotEnabled, "a push for an instrument which is not enabled must error")

	ex.SetTradeFeedStatus(false)
	handleFixture(t, ex, "public-block-trades")
	assert.Empty(t, ex.Websocket.DataHandler.C, "no trade should be relayed with the trade feed off")
}

func TestOrderBookLevels(t *testing.T) {
	t.Parallel()
	assert.Equal(t, orderbook.Levels{{Price: 61506.2, Amount: 0.30039379, OrderCount: 5}}, orderBookLevels([][4]types.Number{{61506.2, 0.30039379, 0, 5}}, true), "a snapshot level should take price, size and order count")
	assert.Equal(t, orderbook.Levels{{Price: 61506.2, Amount: 0.30039379}}, orderBookLevels([][4]types.Number{{61506.2, 0.30039379, 0, 5}}, false), "an incremental level should leave the order count out")
	assert.Empty(t, orderBookLevels(nil, true), "no levels should convert to none")
}

func TestSpreadOrderBookLevels(t *testing.T) {
	t.Parallel()
	var levels [][3]types.PreciseNumber
	require.NoError(t, json.Unmarshal([]byte(`[["-8.7","2.7299","1"],["0.50","1.000","2"]]`), &levels), "the levels must decode")
	exp := orderbook.Levels{
		{Price: -8.7, StrPrice: "-8.7", Amount: 2.7299, StrAmount: "2.7299", OrderCount: 1},
		{Price: 0.5, StrPrice: "0.50", Amount: 1, StrAmount: "1.000", OrderCount: 2},
	}
	assert.Equal(t, exp, spreadOrderBookLevels(levels, true), "a snapshot level should take price, size and order count, keeping the price and size as sent")
	exp[0].OrderCount, exp[1].OrderCount = 0, 0
	assert.Equal(t, exp, spreadOrderBookLevels(levels, false), "an incremental level should leave the order count out")
	assert.Empty(t, spreadOrderBookLevels(nil, true), "no levels should convert to none")
}

// bookTop returns a book's sequence, time and best levels
func bookTop(t *testing.T, ex *Exchange, pair currency.Pair, a asset.Item, levels int) *orderbook.Book {
	t.Helper()
	book, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
	require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
	return &orderbook.Book{
		LastUpdateID: book.LastUpdateID,
		LastUpdated:  book.LastUpdated,
		Asks:         book.Asks[:min(levels, len(book.Asks))],
		Bids:         book.Bids[:min(levels, len(book.Bids))],
	}
}

// signedCRC32 returns the CRC32 of a checksum string as the signed integer OKX
// sends
func signedCRC32(s string) int64 {
	c := int64(crc32.ChecksumIEEE([]byte(s)))
	if c > math.MaxInt32 {
		c -= 1 << 32
	}
	return c
}

func TestOrderBookChecksum(t *testing.T) {
	t.Parallel()
	sides := func(n int) (bids, asks orderbook.Levels, pairs [][2]string) {
		for i := range n {
			bid := orderbook.Level{StrPrice: fmt.Sprintf("-%d.10", i), StrAmount: strconv.Itoa(i + 1)}
			ask := orderbook.Level{StrPrice: fmt.Sprintf("%d.20", i), StrAmount: fmt.Sprintf("%d.0", i+2)}
			bids, asks = append(bids, bid), append(asks, ask)
			pairs = append(pairs, [2]string{bid.StrPrice + ":" + bid.StrAmount, ask.StrPrice + ":" + ask.StrAmount})
		}
		return bids, asks, pairs
	}
	twentySixBids, twentySixAsks, twentySix := sides(26)
	firstTwentyFive := make([]string, 0, 50)
	for _, p := range twentySix[:25] {
		firstTwentyFive = append(firstTwentyFive, p[0], p[1])
	}
	for _, tc := range []struct {
		name string
		book *orderbook.Book
		exp  string
	}{
		{name: "no levels", book: &orderbook.Book{}},
		{name: "one level a side", book: &orderbook.Book{Bids: orderbook.Levels{{StrPrice: "-2", StrAmount: "3"}}, Asks: orderbook.Levels{{StrPrice: "0", StrAmount: "1"}}}, exp: "-2:3:0:1"},
		{
			name: "more bids than asks",
			book: &orderbook.Book{Bids: orderbook.Levels{{StrPrice: "3", StrAmount: "1"}, {StrPrice: "2", StrAmount: "2"}, {StrPrice: "1", StrAmount: "3"}}, Asks: orderbook.Levels{{StrPrice: "4", StrAmount: "4"}}},
			exp:  "3:1:4:4:2:2:1:3",
		},
		{name: "asks alone", book: &orderbook.Book{Asks: orderbook.Levels{{StrPrice: "4", StrAmount: "4"}, {StrPrice: "5.0", StrAmount: "0.10"}}}, exp: "4:4:5.0:0.10"},
		{name: "25 levels a side", book: &orderbook.Book{Bids: twentySixBids[:25], Asks: twentySixAsks[:25]}, exp: strings.Join(firstTwentyFive, ":")},
		{name: "the levels past 25 a side are left out", book: &orderbook.Book{Bids: twentySixBids, Asks: twentySixAsks}, exp: strings.Join(firstTwentyFive, ":")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, crc32.ChecksumIEEE([]byte(tc.exp)), orderBookChecksum(tc.book), "orderBookChecksum should checksum the levels' prices and sizes as sent, alternating from the best bid")
		})
	}
}

func TestOrderBookPushHeldBy(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                   string
		previous, sequence, at int64
		exp                    bool
	}{
		{name: "stale update", previous: 9, sequence: 10, at: 10, exp: true},
		{name: "keep-alive", previous: 10, sequence: 10, at: 10, exp: true},
		{name: "contiguous update", previous: 10, sequence: 11, at: 10},
		{name: "sequence reset", previous: 10, sequence: 3, at: 10},
		{name: "keep-alive after a missed update", previous: 11, sequence: 11, at: 10},
		{name: "older update", previous: 8, sequence: 9, at: 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			push := &orderBookPush{previousSequenceID: tc.previous, sequenceID: tc.sequence}
			assert.Equal(t, tc.exp, push.heldBy(tc.at), "heldBy should report whether the book already holds the push")
		})
	}
}

func TestQualifiedChannelKey(t *testing.T) {
	t.Parallel()
	argument := `{"channel":"books","instId":"BTC-USDT"}`
	key := qualifiedChannelKey(argument)
	assert.Equal(t, argument, key.String(), "String should return the argument")
	assert.Equal(t, argument, key.GetQualifiedChannel(), "GetQualifiedChannel should return the argument")
	assert.Equal(t, &subscription.Subscription{QualifiedChannel: argument}, key.GetSubscription(), "GetSubscription should hold the argument")
	books := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: argument}
	bbo := &subscription.Subscription{Channel: channelBBOTBT, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"bbo-tbt","instId":"BTC-USDT"}`}
	assert.True(t, key.Match(subscription.ExactKey{Subscription: books}), "Match should accept a subscription to the argument")
	assert.False(t, key.Match(subscription.ExactKey{Subscription: bbo}), "Match should refuse a subscription to another argument")
	assert.False(t, key.Match(subscription.ExactKey{}), "Match should refuse a key without a subscription")
	store, err := subscription.NewStoreFromList(subscription.List{bbo, books})
	require.NoError(t, err, "NewStoreFromList must not error")
	assert.Same(t, books, store.Get(key), "a store should find the subscription to the argument")
	assert.Nil(t, store.Get(qualifiedChannelKey(`{"channel":"books5","instId":"BTC-USDT"}`)), "a store should find no subscription to an argument nobody subscribed")
}

func TestKeepsOrderBook(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		a             asset.Item
		id, otherID   currency.Pair
		idField       string
		depthChannels []string
		ranked        []string
	}{
		{
			name: "instrument", a: asset.Spot, id: mainPair, otherID: currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), idField: "instId", depthChannels: orderBookDepthChannels,
			ranked: []string{channelOrderBooksTBT, channelOrderBooks, channelOrderBooksRPI, channelOrderBooks50TBT, channelOrderBooks5, channelBBOTBT},
		},
		{
			name: "spread", a: asset.Spread, id: spreadPair, otherID: currency.NewPairWithDelimiter("ETH", "USDT_ETH-USDT-SWAP", currency.DashDelimiter), idField: "sprdId", depthChannels: spreadOrderBookDepthChannels,
			ranked: []string{channelSpreadOrderBooksTBT, channelSpreadOrderBooks5, channelSpreadBBOTBT},
		},
	} {
		for i, subscribed := range tc.ranked {
			t.Run(tc.name+" subscribed to "+subscribed, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				require.NoError(t, ex.Websocket.TrackTestConnection(channelConnection(subscribed), conn), "TrackTestConnection must not error")
				// The subscription carries the argument channelArgument generates, which keepsOrderBook must match
				qualified, err := ex.channelArgument(&subscription.Subscription{Channel: subscribed}, subscribed, tc.a, tc.id)
				require.NoError(t, err, "channelArgument must not error")
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, &subscription.Subscription{Channel: subscribed, Asset: tc.a, Pairs: currency.Pairs{tc.id}, QualifiedChannel: qualified}), "AddSuccessfulSubscriptions must not error")
				for j, channel := range tc.ranked {
					assert.Equalf(t, j <= i, ex.keepsOrderBook(channel, tc.depthChannels, tc.idField, tc.id), "%s should keep the book only if it is %s or deeper", channel, subscribed)
					assert.Truef(t, ex.keepsOrderBook(channel, tc.depthChannels, tc.idField, tc.otherID), "%s should keep the book of another instrument", channel)
				}
			})
			t.Run(tc.name+" being subscribed to "+subscribed, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				qualified, err := ex.channelArgument(&subscription.Subscription{Channel: subscribed}, subscribed, tc.a, tc.id)
				require.NoError(t, err, "channelArgument must not error")
				ex.subscriptionRequests.request([]string{qualified})
				for j, channel := range tc.ranked {
					assert.Equalf(t, j <= i, ex.keepsOrderBook(channel, tc.depthChannels, tc.idField, tc.id), "%s should keep the book only if it is %s or deeper", channel, subscribed)
					assert.Truef(t, ex.keepsOrderBook(channel, tc.depthChannels, tc.idField, tc.otherID), "%s should keep the book of another instrument", channel)
				}
				ex.subscriptionRequests.release([]string{qualified})
				for _, channel := range tc.ranked {
					assert.Truef(t, ex.keepsOrderBook(channel, tc.depthChannels, tc.idField, tc.id), "%s should keep the book once %s is no longer being subscribed", channel, subscribed)
				}
			})
		}
	}
}

func TestOrderBookArgument(t *testing.T) {
	t.Parallel()
	assert.JSONEq(t, `{"channel":"books","instId":"BTC-USDT"}`, orderBookArgument(channelOrderBooks, mainPair), "an instrument's order book argument should name its instrument")
	assert.JSONEq(t, `{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`, orderBookArgument(channelSpreadOrderBooksTBT, spreadPair), "a spread's order book argument should name its spread")
	for _, sub := range []*subscription.Subscription{
		{Channel: channelOrderBooksRPI, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}},
		{Channel: channelSpreadOrderBooks5, Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}},
	} {
		qualified, err := e.channelArgument(sub, sub.Channel, sub.Asset, sub.Pairs[0])
		require.NoError(t, err, "channelArgument must not error")
		assert.Equalf(t, qualified, orderBookArgument(sub.Channel, sub.Pairs[0]), "the %s argument should be built as channelArgument marshals it", sub.Channel)
	}
}

func TestOrderBookStatesLock(t *testing.T) {
	t.Parallel()
	var s orderBookStates
	btc := s.lock("BTC-USDT")
	eth := s.lock("ETH-USDT")
	assert.NotSame(t, btc, eth, "each ID should have a lock of its own")
	assert.False(t, btc.TryLock(), "lock should return the ID's lock held")
	eth.Unlock()
	btc.Unlock()
	requireWaitsForLock(t, btc, func() { s.lock("BTC-USDT").Unlock() })
}

func TestOrderBookStatesLoad(t *testing.T) {
	t.Parallel()
	var s orderBookStates
	s.deferRecovery(`{"channel":"books","instId":"BTC-USDT"}`)
	s.deferRecovery(`{"channel":"books","instId":"ETH-USDT"}`)
	s.load("BTC-USDT", channelOrderBooks, `{"channel":"books","instId":"BTC-USDT"}`)
	s.load("BTC-USDT", channelOrderBooksRPI, `{"channel":"books-rpi","instId":"BTC-USDT"}`)
	assert.Equal(t, map[string]string{"BTC-USDT": channelOrderBooksRPI}, s.sources, "load should record the channel whose snapshot the book holds")
	assert.Equal(t, map[string]bool{`{"channel":"books","instId":"ETH-USDT"}`: true}, s.deferred, "load should drop the loaded argument's deferred recovery alone")
}

func TestOrderBookStatesHandOver(t *testing.T) {
	t.Parallel()
	var s orderBookStates
	source, changed := s.handOver("BTC-USDT", channelOrderBooks)
	assert.False(t, changed, "a book loaded from no channel should not be handed over")
	assert.Empty(t, source, "a book loaded from no channel should have no source")
	s.load("BTC-USDT", channelOrderBooks, "")
	source, changed = s.handOver("BTC-USDT", channelOrderBooks)
	assert.False(t, changed, "an update from the channel the book was loaded from should not hand it over")
	assert.Empty(t, source, "an update from the book's own channel should report no other source")
	source, changed = s.handOver("BTC-USDT", channelOrderBooksRPI)
	assert.True(t, changed, "an update from another channel should hand the book over")
	assert.Equal(t, channelOrderBooks, source, "the handover should report the channel the book was loaded from")
	assert.Equal(t, map[string]string{"BTC-USDT": channelOrderBooksRPI}, s.sources, "the handover should record the updating channel as the book's")
}

func TestOrderBookStatesDeferRecovery(t *testing.T) {
	t.Parallel()
	var s orderBookStates
	s.deferRecovery("a")
	s.deferRecovery("a")
	s.deferRecovery("b")
	assert.Equal(t, map[string]bool{"a": true, "b": true}, s.deferred, "deferRecovery should record each argument's deferred recovery once")
}

func TestOrderBookStatesTakeDeferredRecovery(t *testing.T) {
	t.Parallel()
	var s orderBookStates
	assert.False(t, s.takeDeferredRecovery("a"), "an argument without a deferred recovery should have none to take")
	s.deferRecovery("a")
	s.deferRecovery("b")
	assert.True(t, s.takeDeferredRecovery("a"), "a deferred recovery should be taken")
	assert.False(t, s.takeDeferredRecovery("a"), "a deferred recovery should be taken once")
	assert.Equal(t, map[string]bool{"b": true}, s.deferred, "taking a deferred recovery should leave the others")
}

func TestRecoverOrderBook(t *testing.T) {
	t.Parallel()
	const argument = `{"channel":"books","instId":"BTC-USDT"}`
	books := func(a asset.Item) *subscription.Subscription {
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: a, Pairs: currency.Pairs{mainPair}, QualifiedChannel: argument}
	}
	t.Run("resubscribes the channel's subscriptions on the connection", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		spot, margin := books(asset.Spot), books(asset.Margin)
		tickers := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, spot, margin, tickers), "AddSuccessfulSubscriptions must not error")
		for _, s := range []*subscription.Subscription{spot, margin, tickers} {
			require.NoError(t, conn.Subscriptions().Add(s), "the connection must hold the subscription")
		}
		require.NoError(t, ex.recoverOrderBook(t.Context(), conn, channelOrderBooks, mainPair, errInvalidOrderbookSequence), "recoverOrderBook must not error")
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "recoverOrderBook must unsubscribe and subscribe the channel")
		for _, req := range conn.Requests() {
			assert.ElementsMatch(t, subscription.List{spot, margin}, req.subs, "recoverOrderBook should renew the channel's subscriptions alone")
		}
	})

	t.Run("a resubscription satisfies the recovery deferred for its argument", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		spot := books(asset.Spot)
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, spot), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(spot), "the connection must hold the subscription")
		ex.orderBookStates.deferRecovery(argument)
		require.NoError(t, ex.recoverOrderBook(t.Context(), conn, channelOrderBooks, mainPair, errOrderbookSourceChanged), "recoverOrderBook must not error")
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "recoverOrderBook must unsubscribe and subscribe the channel")
		assert.False(t, ex.orderBookStates.takeDeferredRecovery(argument), "the resubscription should leave no recovery deferred for its argument")
	})

	for _, tc := range []struct {
		name      string
		requested bool
		tracked   bool
		noConn    bool
		expErr    error
		expDefer  bool
	}{
		{name: "a subscription being subscribed defers the recovery", requested: true, expDefer: true},
		{name: "a subscription tracked but not yet on the connection defers the recovery", tracked: true, expDefer: true},
		{name: "no subscription to recover", expErr: subscription.ErrNotFound},
		{name: "no connection", noConn: true, expErr: subscription.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
			require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
			if tc.requested {
				ex.subscriptionRequests.send(conn, &WebsocketSubscriptionRequest{ID: "1", Operation: operationSubscribe, subs: subscription.List{books(asset.Spot)}})
			}
			if tc.tracked {
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, books(asset.Spot)), "AddSuccessfulSubscriptions must not error")
			}
			var c websocket.Connection = conn
			if tc.noConn {
				c = nil
			}
			require.ErrorIs(t, ex.recoverOrderBook(t.Context(), c, channelOrderBooks, mainPair, errInvalidOrderbookSequence), tc.expErr, "recoverOrderBook must return the expected error")
			assert.Equal(t, tc.expDefer, ex.orderBookStates.takeDeferredRecovery(argument), "recoverOrderBook should defer the recovery only for a subscription not yet on the connection")
			assert.Empty(t, conn.Requests(), "nothing should be resubscribed")
		})
	}
}

func TestInvalidateOrderbooks(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	pair := currency.NewPairWithDelimiter("INVALID", "USDT", currency.DashDelimiter)
	require.NoError(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
		Exchange:     ex.Name,
		Pair:         pair,
		Asset:        asset.Spot,
		LastUpdateID: 1,
		LastUpdated:  time.UnixMilli(1659792392540),
		Bids:         orderbook.Levels{{Price: 1, Amount: 1}},
		Asks:         orderbook.Levels{{Price: 2, Amount: 1}},
	}), "LoadSnapshot must not error")
	cause := errors.New("cause")
	err := ex.invalidateOrderbooks(pair, []asset.Item{asset.Margin, asset.Spot}, cause)
	assert.Same(t, cause, err, "invalidateOrderbooks should return the cause alone when every book which exists invalidates")
	_, err = ex.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the spot book should be invalidated")
	_, err = ex.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "the missing margin book should stay missing")
	assert.NoError(t, ex.invalidateOrderbooks(pair, []asset.Item{asset.Margin}, nil), "invalidateOrderbooks should not report a missing book")
}

func TestWebsocketProcessOrderBooks(t *testing.T) {
	t.Parallel()
	rpiSnapshot := func(sequence int) []byte {
		return fmt.Appendf(nil, `{"arg":{"channel":"books-rpi","instId":"BTC-USDT"},"action":"snapshot","data":[{"bids":[["101","4","1","1"],["100","2","2","1"],["99","3","0","1"]],"asks":[["102","1","1","1"],["103","3","3","1"]],"ts":"1700000000000","prevSeqId":-1,"seqId":%d}]}`, sequence)
	}
	rpiUpdate := func(previous, sequence int) []byte {
		return fmt.Appendf(nil, `{"arg":{"channel":"books-rpi","instId":"BTC-USDT"},"action":"update","data":[{"bids":[],"asks":[["102","2","2","1"]],"ts":"1700000001000","prevSeqId":%d,"seqId":%d}]}`, previous, sequence)
	}

	t.Run("live captures", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			fixture string
			exp     *orderbook.Book
		}{
			{"books", &orderbook.Book{
				LastUpdateID: 84749421744, LastUpdated: time.UnixMilli(1795405196109),
				Asks: orderbook.Levels{{Price: 61506.2, Amount: 0.30036128}, {Price: 61506.3, Amount: 0.0000102}, {Price: 61507.6, Amount: 0.08055589}},
				Bids: orderbook.Levels{{Price: 61506.1, Amount: 0.78661596}, {Price: 61501.3, Amount: 0.00131896}, {Price: 61501.2, Amount: 0.00208527}},
			}},
			{"books-rpi", &orderbook.Book{
				LastUpdateID: 84749422618, LastUpdated: time.UnixMilli(1795405196909),
				Asks: orderbook.Levels{{Price: 61503.9, Amount: 5.75376542}, {Price: 61504.2, Amount: 0.0040704}, {Price: 61504.3, Amount: 0.27417959}},
				Bids: orderbook.Levels{{Price: 61503.8, Amount: 0.23088859}, {Price: 61502.4, Amount: 0.00017}, {Price: 61501.2, Amount: 0.00208527}},
			}},
			{"books5", &orderbook.Book{
				LastUpdateID: 84749417760, LastUpdated: time.UnixMilli(1795405194309),
				Asks: orderbook.Levels{{Price: 61512.6, Amount: 0.22258228, OrderCount: 5}, {Price: 61513.1, Amount: 0.00094109, OrderCount: 1}, {Price: 61513.4, Amount: 0.28634567, OrderCount: 1}},
				Bids: orderbook.Levels{{Price: 61512.5, Amount: 0.46390398, OrderCount: 7}, {Price: 61509.8, Amount: 0.10590696, OrderCount: 2}, {Price: 61509.4, Amount: 0.00057618, OrderCount: 1}},
			}},
			{"bbo-tbt", &orderbook.Book{
				LastUpdateID: 84749417760, LastUpdated: time.UnixMilli(1795405194309),
				Asks: orderbook.Levels{{Price: 61512.6, Amount: 0.22258228, OrderCount: 5}},
				Bids: orderbook.Levels{{Price: 61512.5, Amount: 0.46390398, OrderCount: 7}},
			}},
		} {
			t.Run(tc.fixture, func(t *testing.T) {
				t.Parallel()
				ex := newWebsocketHandlerExchange(t)
				ex.Name = t.Name()
				handleFixture(t, ex, tc.fixture)
				for _, a := range []asset.Item{asset.Spot, asset.Margin} {
					assert.Equalf(t, tc.exp, bookTop(t, ex, mainPair, a, 3), "the %s book should hold the snapshot with every update applied", a)
				}
			})
		}
	})

	t.Run("books5 snapshot replicates to margin", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.wsProcessOrderBooks(t.Context(), nil, []byte(`{"arg":{"channel":"books5","instId":"OKB-USDT"},"data":[{"asks":[["0.0000007465","2290075956","0","4"],["0.0000007466","1747284705","0","4"],["0.0000007467","1338861655","0","3"],["0.0000007468","1661668387","0","6"],["0.0000007469","2715477116","0","5"]],"bids":[["0.0000007464","15693119","0","1"],["0.0000007463","2330835024","0","4"],["0.0000007462","1182926517","0","2"],["0.0000007461","3818684357","0","4"],["0.000000746","6021641435","0","7"]],"instId":"OKB-USDT","ts":"1695864901807","seqId":4826378794}]}`)), "wsProcessOrderBooks must not error")
		pair := currency.NewPairWithDelimiter("OKB", "USDT", "-")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			got, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
			require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
			assert.Lenf(t, got.Asks, 5, "the %s book should hold the five asks", a)
			assert.Lenf(t, got.Bids, 5, "the %s book should hold the five bids", a)
		}
	})

	t.Run("recover from sequence gap", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		tracked.ValidateOrderbook = true
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		subs := subscription.List{
			{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`, Levels: 400},
			{Asset: asset.Margin, Pairs: currency.Pairs{mainPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`, Levels: 400},
		}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(conn, subs...), "AddSuccessfulSubscriptions must not error")
		for _, sub := range subs {
			require.NoError(t, conn.Subscriptions().Add(sub), "Connection subscription tracking must not error")
		}

		snapshot := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["101","1","0","1"]],"bids":[["100","1","0","1"]],"ts":"1659792392540","checksum":0,"prevSeqId":-1,"seqId":10}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, snapshot), "wsProcessOrderBooks must load snapshots without checksum validation")

		gap := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[],"bids":[],"ts":"1659792392640","checksum":0,"prevSeqId":12,"seqId":13}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, gap), "wsProcessOrderBooks must recover from a sequence gap")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := tracked.Websocket.Orderbook.GetOrderbook(mainPair, a)
			require.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "The %s orderbook must remain invalid until a new snapshot arrives", a)
		}
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "Recovery must send an unsubscribe and subscribe request")
		requests := conn.Requests()
		assert.Equal(t, operationUnsubscribe, requests[0].Operation, "Recovery should unsubscribe first")
		require.Len(t, requests[0].Arguments, 1, "Recovery must deduplicate equivalent unsubscribe arguments")
		assert.Equal(t, operationSubscribe, requests[1].Operation, "Recovery should resubscribe second")
		require.Len(t, requests[1].Arguments, 1, "Recovery must deduplicate equivalent subscribe arguments")
		require.Len(t, requests[1].subs, 2, "Recovery must resubscribe both tracked subscriptions")
		for _, sub := range requests[1].subs {
			assert.True(t, slices.Contains(subs, sub), "Recovery should resubscribe the tracked subscription itself")
		}
		recovered := slices.Concat(requests[0].subs, requests[1].subs)
		levels := make([]int, 0, len(recovered))
		for _, sub := range recovered {
			levels = append(levels, sub.Levels)
		}
		assert.Equal(t, []int{400, 400, 400, 400}, levels, "Recovery should keep the tracked subscriptions' levels")
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, gap), "wsProcessOrderBooks must ignore updates while awaiting a replacement snapshot")
		require.Never(t, func() bool { return len(conn.Requests()) > 2 }, 100*time.Millisecond, 10*time.Millisecond, "Updates received before the replacement snapshot must not trigger another resubscription")

		freshSnapshot := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["101","1","0","1"]],"bids":[["100","1","0","1"]],"ts":"1659792392740","checksum":0,"prevSeqId":-1,"seqId":50}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, freshSnapshot), "wsProcessOrderBooks must load a fresh snapshot after recovery")
		validUpdate := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[],"bids":[["100","2","0","1"]],"ts":"1659792392840","checksum":0,"prevSeqId":50,"seqId":51}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, validUpdate), "wsProcessOrderBooks must process updates after recovery")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			book, err := tracked.Websocket.Orderbook.GetOrderbook(mainPair, a)
			require.NoErrorf(t, err, "GetOrderbook must return the recovered %s orderbook", a)
			assert.Equalf(t, int64(51), book.LastUpdateID, "The recovered %s orderbook should accept subsequent updates", a)
		}

		invalidUpdate := []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[],"bids":[["0","1","0","1"]],"ts":"1659792392940","checksum":0,"prevSeqId":51,"seqId":52}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, invalidUpdate), "wsProcessOrderBooks must recover from an invalid orderbook update")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := tracked.Websocket.Orderbook.GetOrderbook(mainPair, a)
			require.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "The %s orderbook must be invalidated after an invalid update", a)
		}
		require.Eventually(t, func() bool { return len(conn.Requests()) == 4 }, time.Second, 10*time.Millisecond, "Invalid update recovery must send another unsubscribe and subscribe request")
		requests = conn.Requests()
		assert.Equal(t, operationUnsubscribe, requests[2].Operation, "Invalid update recovery should unsubscribe first")
		assert.Equal(t, operationSubscribe, requests[3].Operation, "Invalid update recovery should resubscribe second")
	})

	t.Run("exchange-specific channel", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.TrackTestConnection(publicLoginConnection, conn), "TrackTestConnection must not error")
		sub := &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: channelOrderBooksTBT, QualifiedChannel: `{"channel":"books-l2-tbt","instId":"BTC-USDT"}`}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(sub), "Connection subscription tracking must not error")

		snapshot := []byte(`{"arg":{"channel":"books-l2-tbt","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["101","1","0","1"]],"bids":[["100","1","0","1"]],"ts":"1659792392540","prevSeqId":-1,"seqId":10}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, snapshot), "wsProcessOrderBooks must load the exchange-specific channel snapshot")
		gap := []byte(`{"arg":{"channel":"books-l2-tbt","instId":"BTC-USDT"},"action":"update","data":[{"asks":[],"bids":[],"ts":"1659792392640","prevSeqId":12,"seqId":13}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), conn, gap), "wsProcessOrderBooks must recover the exchange-specific channel")
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "Recovery must send one unsubscribe and one subscribe request")
		requests := conn.Requests()
		require.Len(t, requests[0].Arguments, 1, "Unsubscribe must contain the exchange-specific channel")
		assert.Equal(t, channelOrderBooksTBT, requests[0].Arguments[0].Channel, "Unsubscribe should use the exchange-specific channel")
		require.Len(t, requests[1].Arguments, 1, "Subscribe must contain the exchange-specific channel")
		assert.Equal(t, channelOrderBooksTBT, requests[1].Arguments[0].Channel, "Subscribe should use the exchange-specific channel")
	})

	t.Run("a shallower channel's push before the deeper channel's subscription is answered is relayed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		books := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		bbo := &subscription.Subscription{Channel: channelBBOTBT, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"bbo-tbt","instId":"BTC-USDT"}`}
		tickers := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"tickers","instId":"BTC-USDT"}`}
		bboPush := `{"arg":{"channel":"bbo-tbt","instId":"BTC-USDT"},"data":[{"asks":[["102","2","0","1"]],"bids":[["101","1","0","1"]],"ts":"1700000001000","seqId":11}]}`
		conn := dialWebsocketTestServer(t, ex, publicConnection, func(id, operation string, arguments []json.RawMessage) []string {
			return []string{
				subscriptionReply(id, operation, arguments[0]),
				subscriptionReply(id, operation, arguments[1]),
				`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["102","1","0","1"],["103","3","0","1"]],"bids":[["101","1","0","1"],["100","2","0","1"]],"ts":"1700000000000","prevSeqId":-1,"seqId":10}]}`,
				bboPush,
				subscriptionReply(id, operation, arguments[2]),
			}
		})
		require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{books, bbo, tickers}), "Subscribe must not error")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[["102","2","0","1"]],"bids":[],"ts":"1700000001000","prevSeqId":10,"seqId":11}]}`)), "the books update must apply")
		exp := &orderbook.Book{
			LastUpdateID: 11, LastUpdated: time.UnixMilli(1700000001000),
			Asks: orderbook.Levels{{Price: 102, Amount: 2}, {Price: 103, Amount: 3}},
			Bids: orderbook.Levels{{Price: 101, Amount: 1}, {Price: 100, Amount: 2}},
		}
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			assert.Equalf(t, exp, bookTop(t, ex, mainPair, a, 5), "the %s book should hold the books levels", a)
		}
		expPush := new(WebsocketOrderBookPush)
		require.NoError(t, json.Unmarshal([]byte(bboPush), expPush), "the bbo-tbt push must decode")
		var pushes []any
		for _, r := range relayed(ex) {
			if _, isDepth := r.(*orderbook.Depth); !isDepth {
				pushes = append(pushes, r)
			}
		}
		assert.Equal(t, []any{expPush}, pushes, "the bbo-tbt push should be relayed as it arrived")
	})

	t.Run("an update from a channel taking over the book is not applied to the other channel's snapshot", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		books := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		rpi := &subscription.Subscription{Channel: channelOrderBooksRPI, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books-rpi","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, books, rpi), "AddSuccessfulSubscriptions must not error")
		for _, s := range []*subscription.Subscription{books, rpi} {
			require.NoError(t, conn.Subscriptions().Add(s), "the connection must hold the subscription")
		}
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["102","1","0","1"],["103","3","0","1"]],"bids":[["101","1","0","1"],["100","2","0","1"]],"ts":"1700000000000","prevSeqId":-1,"seqId":10}]}`)), "the books snapshot must load")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiSnapshot(10)), "the books-rpi snapshot must be relayed")
		require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, books), "RemoveSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Remove(books), "the connection must release the books subscription")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiUpdate(10, 11)), "the books-rpi update must start recovery")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err := ex.Websocket.Orderbook.GetOrderbook(mainPair, a)
			require.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "the %s book must not serve the books snapshot updated by books-rpi", a)
		}
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "the books-rpi subscription must be renewed for its snapshot")
		renewal := make([]string, 0, len(conn.Requests()))
		for _, req := range conn.Requests() {
			renewal = append(renewal, req.Operation+" "+req.subs[0].QualifiedChannel)
		}
		assert.Equal(t, []string{operationUnsubscribe + " " + rpi.QualifiedChannel, operationSubscribe + " " + rpi.QualifiedChannel}, renewal, "the books-rpi subscription should be renewed")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiUpdate(11, 12)), "an update awaiting the books-rpi snapshot must be skipped")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiSnapshot(20)), "the renewed books-rpi snapshot must load")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiUpdate(20, 21)), "the books-rpi update must apply")
		exp := &orderbook.Book{
			LastUpdateID: 21, LastUpdated: time.UnixMilli(1700000001000),
			Asks: orderbook.Levels{{Price: 102, Amount: 2}, {Price: 103, Amount: 3}},
			Bids: orderbook.Levels{{Price: 101, Amount: 4}, {Price: 100, Amount: 2}, {Price: 99, Amount: 3}},
		}
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			assert.Equalf(t, exp, bookTop(t, ex, mainPair, a, 5), "the %s book should hold the books-rpi levels alone", a)
		}
		assert.Len(t, conn.Requests(), 2, "the book should be renewed once")
	})

	t.Run("a channel whose snapshot was relayed while a deeper one was requested renews it once it keeps the book", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		rpi := &subscription.Subscription{Channel: channelOrderBooksRPI, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books-rpi","instId":"BTC-USDT"}`}
		requested := []string{`{"channel":"books","instId":"BTC-USDT"}`, rpi.QualifiedChannel}
		ex.subscriptionRequests.request(requested)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiSnapshot(10)), "the books-rpi snapshot must be relayed while books is requested")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, rpi), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(rpi), "the connection must hold the books-rpi subscription")
		ex.subscriptionRequests.release(requested)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiUpdate(10, 11)), "the books-rpi update must renew the snapshot it lacks")
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 5*time.Millisecond, "the books-rpi subscription must be renewed for its snapshot")
		renewal := make([]string, 0, len(conn.Requests()))
		for _, req := range conn.Requests() {
			renewal = append(renewal, req.Operation+" "+req.subs[0].QualifiedChannel)
		}
		assert.Equal(t, []string{operationUnsubscribe + " " + rpi.QualifiedChannel, operationSubscribe + " " + rpi.QualifiedChannel}, renewal, "the books-rpi subscription should be renewed")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiSnapshot(20)), "the renewed books-rpi snapshot must load")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, rpiUpdate(20, 21)), "the books-rpi update must apply")
		exp := &orderbook.Book{
			LastUpdateID: 21, LastUpdated: time.UnixMilli(1700000001000),
			Asks: orderbook.Levels{{Price: 102, Amount: 2}, {Price: 103, Amount: 3}},
			Bids: orderbook.Levels{{Price: 101, Amount: 4}, {Price: 100, Amount: 2}, {Price: 99, Amount: 3}},
		}
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			assert.Equalf(t, exp, bookTop(t, ex, mainPair, a, 5), "the %s book should hold the renewed books-rpi levels", a)
		}
		assert.Len(t, conn.Requests(), 2, "the book should be renewed once")
	})

	t.Run("a snapshot loaded on another connection while one is relayed keeps the book", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		rpi, tbt := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}, &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, rpi), "TrackTestConnection must not error")
		require.NoError(t, ex.Websocket.TrackTestConnection(publicLoginConnection, tbt), "TrackTestConnection must not error")
		subscribe := func(conn *subscriptionRecorderConnection, channel string) {
			sub := &subscription.Subscription{Channel: channel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: orderBookArgument(channel, mainPair)}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
			require.NoError(t, conn.Subscriptions().Add(sub), "the connection must hold the subscription")
		}
		subscribe(rpi, channelOrderBooksRPI)
		ctx := &relayPauseContext{Context: t.Context(), paused: make(chan struct{}), release: make(chan struct{}), ready: func() bool {
			for _, a := range []asset.Item{asset.Spot, asset.Margin} {
				if b, err := ex.Websocket.Orderbook.GetOrderbook(mainPair, a); err != nil || len(b.Bids) != 3 {
					return false
				}
			}
			return true
		}}
		t.Cleanup(ctx.resume)
		rpiHandled := make(chan error, 1)
		go func() { rpiHandled <- ex.wsHandleData(ctx, rpi, rpiSnapshot(10)) }()
		select {
		case <-ctx.paused:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the books-rpi snapshot must reach its relay")
		}
		subscribe(tbt, channelOrderBooksTBT)
		tbtHandled := make(chan error, 1)
		go func() {
			tbtHandled <- ex.wsHandleData(t.Context(), tbt, []byte(`{"arg":{"channel":"books-l2-tbt","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["102","1","0","1"],["103","3","0","1"]],"bids":[["101","1","0","1"],["100","2","0","1"]],"ts":"1700000000000","prevSeqId":-1,"seqId":10}]}`))
		}()
		assert.Never(t, func() bool { return len(tbtHandled) != 0 }, 50*time.Millisecond, 5*time.Millisecond, "the books-l2-tbt snapshot should wait for the books-rpi snapshot's application")
		ctx.resume()
		for _, handled := range []chan error{rpiHandled, tbtHandled} {
			select {
			case err := <-handled:
				require.NoError(t, err, "each snapshot must be handled")
			case <-time.After(5 * time.Second):
				require.FailNow(t, "each snapshot must be handled")
			}
		}
		require.NoError(t, ex.wsHandleData(t.Context(), tbt, []byte(`{"arg":{"channel":"books-l2-tbt","instId":"BTC-USDT"},"action":"update","data":[{"asks":[["102","2","0","1"]],"bids":[],"ts":"1700000001000","prevSeqId":10,"seqId":11}]}`)), "the books-l2-tbt update must apply")
		exp := &orderbook.Book{
			LastUpdateID: 11, LastUpdated: time.UnixMilli(1700000001000),
			Asks: orderbook.Levels{{Price: 102, Amount: 2}, {Price: 103, Amount: 3}},
			Bids: orderbook.Levels{{Price: 101, Amount: 1}, {Price: 100, Amount: 2}},
		}
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			assert.Equalf(t, exp, bookTop(t, ex, mainPair, a, 5), "the %s book should hold the books-l2-tbt levels with its update", a)
		}
		assert.Empty(t, tbt.Requests(), "the books-l2-tbt book should not be renewed")
	})

	t.Run("a push waiting for its instrument decides which channel keeps the book once it holds it", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		instrument := ex.orderBookStates.lock(mainPair.String())
		handled := make(chan error, 1)
		go func() { handled <- ex.wsHandleData(t.Context(), conn, rpiSnapshot(10)) }()
		assert.Never(t, func() bool { return len(handled) != 0 }, 50*time.Millisecond, 5*time.Millisecond, "the books-rpi snapshot should wait for the instrument")
		books := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, books), "AddSuccessfulSubscriptions must not error")
		instrument.Unlock()
		select {
		case err := <-handled:
			require.NoError(t, err, "the books-rpi snapshot must be relayed")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the books-rpi snapshot must be handled once the instrument is free")
		}
		_, err := ex.Websocket.Orderbook.GetOrderbook(mainPair, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "the books-rpi snapshot should not load once books keeps the book")
		expPush := new(WebsocketOrderBookPush)
		require.NoError(t, json.Unmarshal(rpiSnapshot(10), expPush), "the books-rpi snapshot must decode")
		assert.Equal(t, []any{expPush}, relayed(ex), "the books-rpi snapshot should be relayed as it arrived")
	})

	t.Run("bbo-tbt push is a snapshot", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		push := []byte(`{"arg":{"channel":"bbo-tbt","instId":"BTC-USDT"},"data":[{"asks":[["101","1","0","1"]],"bids":[["100","1","0","1"]],"ts":"1659792392540","prevSeqId":-1,"seqId":10}]}`)
		require.NoError(t, tracked.wsProcessOrderBooks(t.Context(), nil, push), "wsProcessOrderBooks must load bbo-tbt pushes as snapshots")
		book, err := tracked.Websocket.Orderbook.GetOrderbook(mainPair, asset.Spot)
		require.NoError(t, err, "GetOrderbook must return the bbo-tbt snapshot")
		assert.Equal(t, int64(10), book.LastUpdateID, "The bbo-tbt snapshot should set the sequence ID")
	})

	t.Run("the deepest subscribed channel keeps the book", func(t *testing.T) {
		t.Parallel()
		type framedPush struct {
			connection, frame string
		}
		booksSnapshot := `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["102","1","0","1"],["103","3","0","1"]],"bids":[["101","1","0","1"],["100","2","0","1"]],"ts":"1700000000000","prevSeqId":-1,"seqId":10}]}`
		bboPush := `{"arg":{"channel":"bbo-tbt","instId":"BTC-USDT"},"data":[{"asks":[["102","2","0","1"]],"bids":[["101","1","0","1"]],"ts":"1700000001000","seqId":11}]}`
		bboRelay := &WebsocketOrderBookPush{
			Argument: SubscriptionArgument{Channel: channelBBOTBT, InstrumentID: mainPair},
			Data:     []WebsocketOrderBook{{Asks: [][4]types.Number{{102, 2, 0, 1}}, Bids: [][4]types.Number{{101, 1, 0, 1}}, Timestamp: types.Time(time.UnixMilli(1700000001000)), SequenceID: 11}},
		}
		for _, tc := range []struct {
			name       string
			subscribed map[string][]string
			pushes     []framedPush
			exp        *orderbook.Book
			expRelayed []any
		}{
			{
				name:       "books over bbo-tbt",
				subscribed: map[string][]string{publicConnection: {subscription.OrderbookChannel, channelBBOTBT}},
				pushes: []framedPush{
					{publicConnection, booksSnapshot},
					{publicConnection, bboPush},
					{publicConnection, `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[["102","2","0","1"]],"bids":[],"ts":"1700000001000","prevSeqId":10,"seqId":11}]}`},
				},
				exp: &orderbook.Book{
					LastUpdateID: 11, LastUpdated: time.UnixMilli(1700000001000),
					Asks: orderbook.Levels{{Price: 102, Amount: 2}, {Price: 103, Amount: 3}},
					Bids: orderbook.Levels{{Price: 101, Amount: 1}, {Price: 100, Amount: 2}},
				},
				expRelayed: []any{bboRelay},
			},
			{
				name:       "books-l2-tbt on the login connection over books5",
				subscribed: map[string][]string{publicLoginConnection: {channelOrderBooksTBT}, publicConnection: {channelOrderBooks5}},
				pushes: []framedPush{
					{publicLoginConnection, `{"arg":{"channel":"books-l2-tbt","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["102","1","0","1"],["103","3","0","1"]],"bids":[["101","1","0","1"],["100","2","0","1"]],"ts":"1700000000000","prevSeqId":-1,"seqId":20}]}`},
					{publicConnection, `{"arg":{"channel":"books5","instId":"BTC-USDT"},"data":[{"asks":[["102","4","0","2"]],"bids":[["101","5","0","2"]],"ts":"1700000001000","seqId":21}]}`},
				},
				exp: &orderbook.Book{
					LastUpdateID: 20, LastUpdated: time.UnixMilli(1700000000000),
					Asks: orderbook.Levels{{Price: 102, Amount: 1}, {Price: 103, Amount: 3}},
					Bids: orderbook.Levels{{Price: 101, Amount: 1}, {Price: 100, Amount: 2}},
				},
				expRelayed: []any{&WebsocketOrderBookPush{
					Argument: SubscriptionArgument{Channel: channelOrderBooks5, InstrumentID: mainPair},
					Data:     []WebsocketOrderBook{{Asks: [][4]types.Number{{102, 4, 0, 2}}, Bids: [][4]types.Number{{101, 5, 0, 2}}, Timestamp: types.Time(time.UnixMilli(1700000001000)), SequenceID: 21}},
				}},
			},
			{
				name:       "bbo-tbt alone",
				subscribed: map[string][]string{publicConnection: {channelBBOTBT}},
				pushes:     []framedPush{{publicConnection, bboPush}},
				exp: &orderbook.Book{
					LastUpdateID: 11, LastUpdated: time.UnixMilli(1700000001000),
					Asks: orderbook.Levels{{Price: 102, Amount: 2, OrderCount: 1}},
					Bids: orderbook.Levels{{Price: 101, Amount: 1, OrderCount: 1}},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Name = t.Name()
				conns := make(map[string]*subscriptionRecorderConnection, len(tc.subscribed))
				for connection, channels := range tc.subscribed {
					conns[connection] = &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
					require.NoError(t, ex.Websocket.TrackTestConnection(connection, conns[connection]), "TrackTestConnection must not error")
					for _, channel := range channels {
						sub := &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: channel}
						name, err := channelName(sub, asset.Spot)
						require.NoError(t, err, "channelName must not error")
						sub.QualifiedChannel = `{"channel":"` + name + `","instId":"BTC-USDT"}`
						require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conns[connection], sub), "AddSuccessfulSubscriptions must not error")
					}
				}
				for _, p := range tc.pushes {
					require.NoErrorf(t, ex.wsHandleData(t.Context(), conns[p.connection], []byte(p.frame)), "wsHandleData must not error for %s", p.frame)
				}
				for _, a := range []asset.Item{asset.Spot, asset.Margin} {
					assert.Equalf(t, tc.exp, bookTop(t, ex, mainPair, a, 5), "the %s book should hold the deepest channel's levels", a)
				}
				var pushes []any
				for _, r := range relayed(ex) {
					if _, isDepth := r.(*orderbook.Depth); !isDepth {
						pushes = append(pushes, r)
					}
				}
				assert.Equal(t, tc.expRelayed, pushes, "the pushes of a shallower channel should be relayed as they arrived")
			})
		}
	})

	for _, tc := range []struct {
		name   string
		frame  string
		expErr error
	}{
		{"unknown action", `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"partial","data":[]}`, orderbook.ErrInvalidAction},
		{"action on a snapshot channel", `{"arg":{"channel":"books5","instId":"BTC-USDT"},"action":"update","data":[]}`, orderbook.ErrInvalidAction},
		{"no instrument", `{"arg":{"channel":"books"},"action":"snapshot","data":[]}`, currency.ErrCurrencyPairsEmpty},
		{"instrument not enabled", `{"arg":{"channel":"books","instId":"XRP-USDT"},"action":"snapshot","data":[]}`, asset.ErrNotEnabled},
		{"no subscription to resubscribe", `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[],"bids":[],"ts":"1659792392640","prevSeqId":12,"seqId":13}]}`, subscription.ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			ex.Name = t.Name()
			conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
			if errors.Is(tc.expErr, subscription.ErrNotFound) {
				require.NoError(t, ex.wsProcessOrderBooks(t.Context(), conn, []byte(`{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["101","1","0","1"]],"bids":[["100","1","0","1"]],"ts":"1659792392540","prevSeqId":-1,"seqId":10}]}`)), "the snapshot must load")
			}
			require.ErrorIs(t, ex.wsProcessOrderBooks(t.Context(), conn, []byte(tc.frame)), tc.expErr, "wsProcessOrderBooks must return the expected error")
		})
	}
}

func TestWebsocketProcessSpreadOrderBooks(t *testing.T) {
	t.Parallel()

	t.Run("live captures", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			fixture string
			exp     *orderbook.Book
		}{
			{"sprd-books5", &orderbook.Book{
				LastUpdateID: 1794451134517825, LastUpdated: time.UnixMilli(1795405202903),
				Asks: orderbook.Levels{{Price: -8.7, StrPrice: "-8.7", Amount: 2.7299, StrAmount: "2.7299", OrderCount: 1}}, Bids: orderbook.Levels{{Price: -64.2, StrPrice: "-64.2", Amount: 0.2297, StrAmount: "0.2297", OrderCount: 1}},
			}},
			{"sprd-bbo-tbt", &orderbook.Book{
				LastUpdateID: 1794451134517825, LastUpdated: time.UnixMilli(1795405202954),
				Asks: orderbook.Levels{{Price: -8.7, StrPrice: "-8.7", Amount: 2.7299, StrAmount: "2.7299", OrderCount: 1}}, Bids: orderbook.Levels{{Price: -64.2, StrPrice: "-64.2", Amount: 0.2297, StrAmount: "0.2297", OrderCount: 1}},
			}},
			// The two updates after the snapshot are keep-alive messages OKX pushes when the book has not changed, so
			// the book keeps the snapshot's time
			{"sprd-books-l2-tbt", &orderbook.Book{
				LastUpdateID: 1794451134581245, LastUpdated: time.UnixMilli(1795406820638),
				Asks: orderbook.Levels{{Price: -8.7, StrPrice: "-8.7", Amount: 2.749, StrAmount: "2.749"}}, Bids: orderbook.Levels{{Price: -63.9, StrPrice: "-63.9", Amount: 0.2297, StrAmount: "0.2297"}},
			}},
		} {
			t.Run(tc.fixture, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Name = t.Name()
				handleFixture(t, ex, tc.fixture)
				assert.Equal(t, tc.exp, bookTop(t, ex, spreadPair, asset.Spread, 3), "the spread book should hold the snapshot with every update applied")
			})
		}
	})

	t.Run("a push waits for the pushes applied to its spread", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		spread := ex.orderBookStates.lock("BTC-USDT_BTC-USDT-SWAP")
		spread.Unlock()
		requireWaitsForLock(t, spread, func() {
			assert.NoError(t, ex.wsProcessSpreadOrderBooks(t.Context(), nil, []byte(`{"arg":{"channel":"sprd-books5","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[{"asks":[["0.5","2","1"]],"bids":[["-1.5","3","2"]],"ts":"1700000000000"}]}`)), "wsProcessSpreadOrderBooks should not error")
		})
		exp := &orderbook.Book{
			LastUpdated: time.UnixMilli(1700000000000),
			Asks:        orderbook.Levels{{Price: 0.5, StrPrice: "0.5", Amount: 2, StrAmount: "2", OrderCount: 1}},
			Bids:        orderbook.Levels{{Price: -1.5, StrPrice: "-1.5", Amount: 3, StrAmount: "3", OrderCount: 2}},
		}
		assert.Equal(t, exp, bookTop(t, ex, spreadPair, asset.Spread, 3), "the spread book should load once the spread is free")
	})

	t.Run("zero-priced levels", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.wsProcessSpreadOrderBooks(t.Context(), nil, []byte(`{"arg":{"channel":"sprd-books5","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[{"asks":[["0.5","2","1"]],"bids":[["0","1","1"],["-1.5","3","2"]],"ts":"1700000000000"}]}`)), "wsProcessSpreadOrderBooks must not error")
		exp := &orderbook.Book{
			LastUpdated: time.UnixMilli(1700000000000),
			Asks:        orderbook.Levels{{Price: 0.5, StrPrice: "0.5", Amount: 2, StrAmount: "2", OrderCount: 1}},
			Bids:        orderbook.Levels{{Price: 0, StrPrice: "0", Amount: 1, StrAmount: "1", OrderCount: 1}, {Price: -1.5, StrPrice: "-1.5", Amount: 3, StrAmount: "3", OrderCount: 2}},
		}
		assert.Equal(t, exp, bookTop(t, ex, spreadPair, asset.Spread, 3), "the spread book should hold its zero-priced bid")
	})

	t.Run("documented push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.wsProcessSpreadOrderBooks(t.Context(), nil, []byte(`{"arg":{"channel":"sprd-books5", "sprdId": "BTC-USDT_BTC-USDT-SWAP" }, "data": [ { "asks": [ ["111.06","55154","2"], ["111.07","53276","2"], ["111.08","72435","2"], ["111.09","70312","2"], ["111.1","67272","2"]], "bids": [ ["111.05","57745","2"], ["111.04","57109","2"], ["111.03","69563","2"], ["111.02","71248","2"], ["111.01","65090","2"]], "ts": "1670324386802"}]}`)), "wsProcessSpreadOrderBooks must not error")
		book, err := ex.Websocket.Orderbook.GetOrderbook(spreadPair, asset.Spread)
		require.NoError(t, err, "GetOrderbook must not error")
		assert.Len(t, book.Asks, 5, "the spread book should hold the five asks")
		assert.Len(t, book.Bids, 5, "the spread book should hold the five bids")
	})

	t.Run("the deepest subscribed channel keeps the book", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(businessConnection, conn), "TrackTestConnection must not error")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn,
			&subscription.Subscription{Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"sprd-books5","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`},
			&subscription.Subscription{Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}, Channel: channelSpreadOrderBooksTBT, QualifiedChannel: `{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`},
		), "AddSuccessfulSubscriptions must not error")
		books5 := []byte(`{"arg":{"channel":"sprd-books5","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[{"asks":[["-8.6","7","2"]],"bids":[["-8.8","6","2"]],"ts":"1700000001000","seqId":6}]}`)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"snapshot","data":[{"asks":[["-8.7","2","1"],["-8.5","3","1"]],"bids":[["-9","1","1"],["-9.5","4","1"]],"ts":"1700000000000","checksum":0,"prevSeqId":-1,"seqId":5}]}`)), "the sprd-books-l2-tbt snapshot must load")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, books5), "the sprd-books5 push must not error")
		exp := &orderbook.Book{
			LastUpdateID: 5, LastUpdated: time.UnixMilli(1700000000000),
			Asks: orderbook.Levels{{Price: -8.7, StrPrice: "-8.7", Amount: 2, StrAmount: "2"}, {Price: -8.5, StrPrice: "-8.5", Amount: 3, StrAmount: "3"}},
			Bids: orderbook.Levels{{Price: -9, StrPrice: "-9", Amount: 1, StrAmount: "1"}, {Price: -9.5, StrPrice: "-9.5", Amount: 4, StrAmount: "4"}},
		}
		assert.Equal(t, exp, bookTop(t, ex, spreadPair, asset.Spread, 5), "the spread book should hold the sprd-books-l2-tbt levels")
		expPush := new(WebsocketSpreadOrderBookPush)
		require.NoError(t, json.Unmarshal(books5, expPush), "the sprd-books5 push must decode")
		var pushes []any
		for _, r := range relayed(ex) {
			if _, isDepth := r.(*orderbook.Depth); !isDepth {
				pushes = append(pushes, r)
			}
		}
		assert.Equal(t, []any{expPush}, pushes, "the sprd-books5 push should be relayed as it arrived")
	})

	t.Run("checksum", func(t *testing.T) {
		t.Parallel()
		snapshot := func(checksum int64) string {
			return fmt.Sprintf(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"snapshot","data":[{"asks":[["0","1","1"]],"bids":[["-2.50","3","1"]],"ts":"1700000000000","checksum":%d,"prevSeqId":-1,"seqId":1}]}`, checksum)
		}
		update := func(checksum int64) string {
			return fmt.Sprintf(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"update","data":[{"asks":[["0","9","1"]],"bids":[],"ts":"1700000001000","checksum":%d,"prevSeqId":1,"seqId":2}]}`, checksum)
		}
		updated := &orderbook.Book{
			LastUpdateID: 2, LastUpdated: time.UnixMilli(1700000001000),
			Asks: orderbook.Levels{{Price: 0, StrPrice: "0", Amount: 9, StrAmount: "9"}},
			Bids: orderbook.Levels{{Price: -2.5, StrPrice: "-2.50", Amount: 3, StrAmount: "3"}},
		}
		for _, tc := range []struct {
			name        string
			unvalidated bool
			frames      []string
			exp         *orderbook.Book
			expErr      error
		}{
			{name: "an update matching its checksum applies", frames: []string{snapshot(signedCRC32("-2.50:3:0:1")), update(signedCRC32("-2.50:3:0:9"))}, exp: updated},
			{name: "an update failing its checksum invalidates the book", frames: []string{snapshot(signedCRC32("-2.50:3:0:1")), update(signedCRC32("-2.50:3:0:2"))}, expErr: orderbook.ErrOrderbookInvalid},
			{name: "a snapshot failing its checksum is not loaded", frames: []string{snapshot(signedCRC32("-2.5:3:0:1"))}, expErr: orderbook.ErrDepthNotFound},
			{name: "a zero checksum is not verified", frames: []string{snapshot(0), update(0)}, exp: updated},
			{name: "order book validation off skips the checksums", unvalidated: true, frames: []string{snapshot(signedCRC32("-2.5:3:0:1")), update(signedCRC32("-2.50:3:0:2"))}, exp: updated},
			{
				name: "an update respelling a level's price is checked as sent",
				frames: []string{
					fmt.Sprintf(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"snapshot","data":[{"asks":[["0.010","1","1"]],"bids":[["-2.50","3","1"]],"ts":"1700000000000","checksum":%d,"prevSeqId":-1,"seqId":1}]}`, signedCRC32("-2.50:3:0.010:1")),
					fmt.Sprintf(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"update","data":[{"asks":[["0.01","2.00","1"]],"bids":[],"ts":"1700000001000","checksum":%d,"prevSeqId":1,"seqId":2}]}`, signedCRC32("-2.50:3:0.01:2.00")),
				},
				exp: &orderbook.Book{
					LastUpdateID: 2, LastUpdated: time.UnixMilli(1700000001000),
					Asks: orderbook.Levels{{Price: 0.01, StrPrice: "0.01", Amount: 2, StrAmount: "2.00"}},
					Bids: orderbook.Levels{{Price: -2.5, StrPrice: "-2.50", Amount: 3, StrAmount: "3"}},
				},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Setup must not error")
				ex.Name = t.Name()
				ex.ValidateOrderbook = !tc.unvalidated
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				require.NoError(t, ex.Websocket.TrackTestConnection(businessConnection, conn), "TrackTestConnection must not error")
				sub := &subscription.Subscription{Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}, Channel: channelSpreadOrderBooksTBT, QualifiedChannel: `{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`}
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
				require.NoError(t, conn.Subscriptions().Add(sub), "Connection subscription tracking must not error")
				for _, frame := range tc.frames {
					require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(frame)), "wsHandleData must not error for %s", frame)
				}
				if tc.expErr != nil {
					_, err := ex.Websocket.Orderbook.GetOrderbook(spreadPair, asset.Spread)
					require.ErrorIs(t, err, tc.expErr, "a book failing its checksum must not be in service")
					assert.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "a checksum failure should resubscribe the channel")
					return
				}
				assert.Equal(t, tc.exp, bookTop(t, ex, spreadPair, asset.Spread, 5), "the spread book should hold the snapshot with the update applied")
				assert.Never(t, func() bool { return len(conn.Requests()) != 0 }, 50*time.Millisecond, 10*time.Millisecond, "a verified book should not resubscribe")
			})
		}
	})

	t.Run("a snapshot failing its checksum before its subscription is tracked recovers once it is", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		ex.ValidateOrderbook = true
		books := &subscription.Subscription{Channel: channelSpreadOrderBooksTBT, Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}, QualifiedChannel: `{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`}
		tickers := &subscription.Subscription{Channel: channelSpreadTickers, Asset: asset.Spread, Pairs: currency.Pairs{spreadPair}, QualifiedChannel: `{"channel":"sprd-tickers","sprdId":"BTC-USDT_BTC-USDT-SWAP"}`}
		snapshot := func(checksum int64) string {
			return fmt.Sprintf(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"snapshot","data":[{"asks":[["0","1","1"]],"bids":[["-2.50","3","1"]],"ts":"1700000000000","checksum":%d,"prevSeqId":-1,"seqId":1}]}`, checksum)
		}
		synctest.Test(t, func(t *testing.T) {
			var mu sync.Mutex
			var received []string
			conn := dialWebsocketTestServer(t, ex, businessConnection, func(id, operation string, arguments []json.RawMessage) []string {
				mu.Lock()
				defer mu.Unlock()
				for _, a := range arguments {
					received = append(received, operation+" "+string(a))
				}
				switch {
				case operation == operationSubscribe && len(arguments) == 2:
					return []string{subscriptionReply(id, operation, arguments[0]), snapshot(signedCRC32("-2.50:3:0:9")), subscriptionReply(id, operation, arguments[1])}
				case operation == operationSubscribe:
					return []string{subscriptionReply(id, operation, arguments[0]), snapshot(signedCRC32("-2.50:3:0:1"))}
				}
				return []string{subscriptionReply(id, operation, arguments[0])}
			})
			require.NoError(t, ex.Websocket.TrackTestConnection(businessConnection, conn), "TrackTestConnection must not error")
			require.NoError(t, ex.Subscribe(t.Context(), conn, subscription.List{books, tickers}), "Subscribe must not error")
			for _, s := range []*subscription.Subscription{books, tickers} {
				require.NoError(t, conn.Subscriptions().Add(s), "the connection must hold the subscription, as the websocket manager records it")
			}
			_, err := ex.Websocket.Orderbook.GetOrderbook(spreadPair, asset.Spread)
			require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "the snapshot failing its checksum must not load")
			require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"update","data":[{"asks":[],"bids":[],"ts":"1700000001000","checksum":0,"prevSeqId":1,"seqId":1}]}`)), "the next push must start the recovery")
			synctest.Wait()
			mu.Lock()
			defer mu.Unlock()
			exp := []string{
				operationSubscribe + " " + books.QualifiedChannel,
				operationSubscribe + " " + tickers.QualifiedChannel,
				operationUnsubscribe + " " + books.QualifiedChannel,
				operationSubscribe + " " + books.QualifiedChannel,
			}
			assert.Equal(t, exp, received, "the book's subscription should be renewed once it is tracked")
			expBook := &orderbook.Book{
				LastUpdateID: 1, LastUpdated: time.UnixMilli(1700000000000),
				Asks: orderbook.Levels{{Price: 0, StrPrice: "0", Amount: 1, StrAmount: "1"}},
				Bids: orderbook.Levels{{Price: -2.5, StrPrice: "-2.50", Amount: 3, StrAmount: "3"}},
			}
			assert.Equal(t, expBook, bookTop(t, ex, spreadPair, asset.Spread, 5), "the renewed snapshot should load")
		})
	})

	for _, tc := range []struct {
		name   string
		frame  string
		expErr error
	}{
		{"unknown action", `{"arg":{"channel":"sprd-books-l2-tbt","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"partial","data":[]}`, orderbook.ErrInvalidAction},
		{"action on a snapshot channel", `{"arg":{"channel":"sprd-books5","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"action":"update","data":[]}`, orderbook.ErrInvalidAction},
		{"no spread", `{"arg":{"channel":"sprd-books5"},"data":[]}`, currency.ErrCurrencyPairsEmpty},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			require.ErrorIs(t, ex.wsProcessSpreadOrderBooks(t.Context(), nil, []byte(tc.frame)), tc.expErr, "wsProcessSpreadOrderBooks must return the expected error")
		})
	}
}

func TestApplyOrderBookPush(t *testing.T) { //nolint:tparallel // The failed resubscription case replaces the process-wide log hook, so it runs alone
	push := &orderBookPush{
		asks:       orderbook.Levels{{Price: 101, Amount: 1}},
		bids:       orderbook.Levels{{Price: 100, Amount: 1}},
		updated:    time.UnixMilli(1659792392540),
		sequenceID: 10,
	}

	t.Run("snapshot then update", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.applyOrderBookPush(t.Context(), nil, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, push, true), "the snapshot must load")
		update := &orderBookPush{bids: orderbook.Levels{{Price: 100, Amount: 2}}, updated: time.UnixMilli(1659792392640), previousSequenceID: 10, sequenceID: 11}
		require.NoError(t, ex.applyOrderBookPush(t.Context(), nil, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, update, false), "the update must apply")
		exp := &orderbook.Book{LastUpdateID: 11, LastUpdated: time.UnixMilli(1659792392640), Asks: orderbook.Levels{{Price: 101, Amount: 1}}, Bids: orderbook.Levels{{Price: 100, Amount: 2}}}
		assert.Equal(t, exp, bookTop(t, ex, mainPair, asset.Spot, 3), "the update should apply to the snapshot")
	})

	t.Run("update before a snapshot", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.ErrorIs(t, ex.applyOrderBookPush(t.Context(), nil, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, push, false), orderbook.ErrDepthNotFound, "an update without a book must error")
	})

	t.Run("update while a snapshot is pending", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.applyOrderBookPush(t.Context(), nil, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, push, true), "the snapshot must load")
		require.NoError(t, ex.Websocket.Orderbook.InvalidateOrderbook(mainPair, asset.Spot), "InvalidateOrderbook must not error")
		assert.NoError(t, ex.applyOrderBookPush(t.Context(), nil, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, push, false), "an update while a fresh snapshot is pending should be skipped")
	})

	t.Run("a snapshot which cannot load is reported", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		sub := &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(sub), "Connection subscription tracking must not error")
		untimed := &orderBookPush{asks: push.asks, bids: push.bids, sequenceID: 10}
		err := ex.applyOrderBookPush(t.Context(), conn, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, untimed, true)
		require.ErrorIs(t, err, orderbook.ErrLastUpdatedNotSet, "a snapshot without a time must not load")
		assert.Never(t, func() bool { return len(conn.Requests()) != 0 }, 50*time.Millisecond, 10*time.Millisecond, "a snapshot which cannot load should not resubscribe")
	})

	t.Run("a snapshot which cannot replace another channel's ends the other channel's book", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		ex.ValidateOrderbook = true
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		rpi := &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: channelOrderBooksRPI, QualifiedChannel: `{"channel":"books-rpi","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, rpi), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(rpi), "the connection must hold the books-rpi subscription")
		spot := []asset.Item{asset.Spot}
		require.NoError(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooks, mainPair, spot, push, true), "the books snapshot must load")
		unpriced := &orderBookPush{asks: push.asks, bids: orderbook.Levels{{Price: 0, Amount: 1}}, updated: time.UnixMilli(1659792392640), sequenceID: 10}
		require.ErrorIs(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooksRPI, mainPair, spot, unpriced, true), orderbook.ErrPriceZero, "the books-rpi snapshot which cannot load must be reported")
		_, err := ex.Websocket.Orderbook.GetOrderbook(mainPair, asset.Spot)
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the books snapshot must not stay in service once books-rpi keeps the book")
		require.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 5*time.Millisecond, "the books-rpi subscription must be renewed for its snapshot")
		update := &orderBookPush{asks: orderbook.Levels{{Price: 101, Amount: 2}}, updated: time.UnixMilli(1659792392740), previousSequenceID: 10, sequenceID: 11}
		require.NoError(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooksRPI, mainPair, spot, update, false), "a books-rpi update awaiting its snapshot must be skipped")
		_, err = ex.Websocket.Orderbook.GetOrderbook(mainPair, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the books-rpi update should not apply to the books snapshot")
		assert.Never(t, func() bool { return len(conn.Requests()) != 2 }, 50*time.Millisecond, 10*time.Millisecond, "the book should be renewed once")
	})

	t.Run("a snapshot which loads beside a full relay keeps the book", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		spot := []asset.Item{asset.Spot}
		require.NoError(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooks, mainPair, spot, push, true), "the books snapshot must load")
		for len(ex.Websocket.DataHandler.C) < cap(ex.Websocket.DataHandler.C) {
			require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), struct{}{}), "DataHandler must accept the saturation payload")
		}
		snapshot := &orderBookPush{asks: orderbook.Levels{{Price: 102, Amount: 3}}, bids: orderbook.Levels{{Price: 99, Amount: 4}}, updated: time.UnixMilli(1659792392640), sequenceID: 20}
		require.ErrorContains(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooksRPI, mainPair, spot, snapshot, true), "channel buffer is full", "the books-rpi snapshot's relay failure must be reported")
		update := &orderBookPush{bids: orderbook.Levels{{Price: 99, Amount: 5}}, updated: time.UnixMilli(1659792392740), previousSequenceID: 20, sequenceID: 21}
		require.ErrorContains(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooksRPI, mainPair, spot, update, false), "channel buffer is full", "the books-rpi update's relay failure must be reported")
		exp := &orderbook.Book{LastUpdateID: 21, LastUpdated: time.UnixMilli(1659792392740), Asks: orderbook.Levels{{Price: 102, Amount: 3}}, Bids: orderbook.Levels{{Price: 99, Amount: 5}}}
		assert.Equal(t, exp, bookTop(t, ex, mainPair, asset.Spot, 3), "the books-rpi update should apply to the books-rpi snapshot")
		assert.Empty(t, conn.Requests(), "the book should not be renewed")
	})

	t.Run("a recovery returns the errors relaying the books the update changed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		pair := currency.NewPairWithDelimiter("RELAY", "USDT", currency.DashDelimiter)
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			sub := &subscription.Subscription{Asset: a, Pairs: currency.Pairs{pair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"RELAY-USDT"}`}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
			require.NoError(t, conn.Subscriptions().Add(sub), "the connection must hold the subscription")
			require.NoError(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
				Exchange:          ex.Name,
				Pair:              pair,
				Asset:             a,
				LastUpdateID:      10,
				LastUpdated:       time.UnixMilli(1659792392540),
				Bids:              orderbook.Levels{{Price: 100, Amount: 1}},
				Asks:              orderbook.Levels{{Price: 101, Amount: 1}},
				ValidateOrderbook: a == asset.Margin, // the zero-priced bid applies to spot, then fails margin's validation
			}), "LoadSnapshot must not error")
		}
		for len(ex.Websocket.DataHandler.C) < cap(ex.Websocket.DataHandler.C) {
			require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), struct{}{}), "DataHandler must accept the saturation payload")
		}
		update := &orderBookPush{bids: orderbook.Levels{{Price: 0, Amount: 1}}, updated: time.UnixMilli(1659792392640), previousSequenceID: 10, sequenceID: 11}
		err := ex.applyOrderBookPush(t.Context(), conn, channelOrderBooks, pair, []asset.Item{asset.Spot, asset.Margin}, update, false)
		require.ErrorContains(t, err, "channel buffer is full", "the spot relay failure must be returned beside the recovery")
		assert.NotErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the invalid margin book should be left to the recovery")
		assert.Eventually(t, func() bool { return len(conn.Requests()) == 2 }, time.Second, 10*time.Millisecond, "the invalid book should be recovered")
	})

	t.Run("a failed resubscription is logged", func(t *testing.T) {
		// Not parallel: the log hook and log config are process-wide, and cleanup returns both to the package defaults
		var mu sync.Mutex
		var logged []string
		require.NoError(t, gctlog.SetGlobalLogConfig(gctlog.GenDefaultSettings()), "SetGlobalLogConfig must not error")
		gctlog.SetCustomLogHook(func(_, _ string, a ...any) bool {
			mu.Lock()
			logged = append(logged, fmt.Sprint(a...))
			mu.Unlock()
			return true
		})
		t.Cleanup(func() {
			gctlog.SetCustomLogHook(nil)
			assert.NoError(t, gctlog.SetGlobalLogConfig(&gctlog.Config{}), "SetGlobalLogConfig should not error")
		})
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, ex.Websocket.TrackTestConnection(publicConnection, conn), "TrackTestConnection must not error")
		sub := &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{mainPair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, sub), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, conn.Subscriptions().Add(sub), "Connection subscription tracking must not error")
		// A subscription already resubscribing refuses the recovery's resubscription
		require.NoError(t, sub.SetState(subscription.ResubscribingState), "SetState must not error")
		require.NoError(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, push, true), "the snapshot must load")
		gap := &orderBookPush{updated: time.UnixMilli(1659792392640), previousSequenceID: 12, sequenceID: 13}
		require.NoError(t, ex.applyOrderBookPush(t.Context(), conn, channelOrderBooks, mainPair, []asset.Item{asset.Spot}, gap, false), "a sequence gap must start recovery")
		assert.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return slices.ContainsFunc(logged, func(s string) bool {
				return strings.Contains(s, "Failed to resubscribe") && strings.Contains(s, subscription.ErrInStateAlready.Error())
			})
		}, time.Second, 10*time.Millisecond, "the refused resubscription should be logged")
		assert.Empty(t, conn.Requests(), "the refused resubscription should send nothing")
	})
}

func TestWebsocketProcessSnapshotOrderBook(t *testing.T) {
	t.Parallel()
	push := &orderBookPush{
		bids:       orderbook.Levels{{Price: 100.5, Amount: 1.25, OrderCount: 1}},
		asks:       orderbook.Levels{{Price: 100.6, Amount: 0.75, OrderCount: 1}},
		updated:    time.UnixMilli(1659792392540),
		sequenceID: 42,
	}
	exp := &orderbook.Book{
		LastUpdateID: 42,
		LastUpdated:  time.UnixMilli(1659792392540),
		Bids:         orderbook.Levels{{Price: 100.5, Amount: 1.25, OrderCount: 1}},
		Asks:         orderbook.Levels{{Price: 100.6, Amount: 0.75, OrderCount: 1}},
	}

	t.Run("loads the snapshot with its sequence ID", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		pair := currency.NewPairWithDelimiter("SNAP", "USDT", "-")
		dispatchErr, err := tracked.wsProcessSnapshotOrderBook(t.Context(), push, pair, []asset.Item{asset.Spot})
		require.NoError(t, err, "wsProcessSnapshotOrderBook must load the snapshot")
		require.NoError(t, dispatchErr, "wsProcessSnapshotOrderBook must relay the book")
		assert.Equal(t, exp, bookTop(t, tracked, pair, asset.Spot, 1), "the snapshot should load with its sequence ID")
	})

	t.Run("a full relay still loads every asset", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		pair := currency.NewPairWithDelimiter("HALF", "USDT", "-")
		for len(tracked.Websocket.DataHandler.C) < cap(tracked.Websocket.DataHandler.C) {
			require.NoError(t, tracked.Websocket.DataHandler.Send(t.Context(), struct{}{}), "DataHandler must accept the saturation payload")
		}
		dispatchErr, err := tracked.wsProcessSnapshotOrderBook(t.Context(), push, pair, []asset.Item{asset.Spot, asset.Margin})
		require.NoError(t, err, "wsProcessSnapshotOrderBook must not report books which loaded as unloaded")
		require.ErrorContains(t, dispatchErr, "channel buffer is full", "wsProcessSnapshotOrderBook must report the relay failure")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			assert.Equalf(t, exp, bookTop(t, tracked, pair, a, 1), "the %s snapshot should load despite the relay failure", a)
		}
	})

	t.Run("a snapshot which does not load leaves every book as it was", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			snapshot *orderBookPush
			expErr   error
		}{
			{name: "levels which do not validate", snapshot: &orderBookPush{bids: orderbook.Levels{{Price: 0, Amount: 1}}, asks: push.asks, updated: time.UnixMilli(1659792392640), sequenceID: 43}, expErr: orderbook.ErrPriceZero},
			{name: "no time", snapshot: &orderBookPush{bids: push.bids, asks: push.asks, sequenceID: 43}, expErr: orderbook.ErrLastUpdatedNotSet},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				tracked := new(Exchange)
				require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
				tracked.Name = t.Name()
				tracked.ValidateOrderbook = true
				pair := currency.NewPairWithDelimiter("KEEP", "USDT", "-")
				assets := []asset.Item{asset.Spot, asset.Margin}
				dispatchErr, err := tracked.wsProcessSnapshotOrderBook(t.Context(), push, pair, assets)
				require.NoError(t, err, "the first snapshot must load")
				require.NoError(t, dispatchErr, "the first snapshot must be relayed")
				relayed(tracked)
				dispatchErr, err = tracked.wsProcessSnapshotOrderBook(t.Context(), tc.snapshot, pair, assets)
				require.ErrorIs(t, err, tc.expErr, "wsProcessSnapshotOrderBook must report the snapshot which does not load")
				assert.NoError(t, dispatchErr, "nothing should be relayed")
				for _, a := range assets {
					assert.Equalf(t, exp, bookTop(t, tracked, pair, a, 1), "the %s book should keep the first snapshot", a)
				}
				assert.Empty(t, relayed(tracked), "no book should be relayed")
			})
		}
	})

	t.Run("a snapshot failing its checksum invalidates the book", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		tracked.ValidateOrderbook = true
		bids, asks := orderbook.Levels{{Price: -2.5, StrPrice: "-2.50", Amount: 3, StrAmount: "3"}}, orderbook.Levels{{Price: 0, StrPrice: "0", Amount: 1, StrAmount: "1"}}
		dispatchErr, err := tracked.wsProcessSnapshotOrderBook(t.Context(), &orderBookPush{bids: bids, asks: asks, updated: time.UnixMilli(1700000000000), sequenceID: 1, checksum: crc32.ChecksumIEEE([]byte("-2.50:3:0:1"))}, spreadPair, []asset.Item{asset.Spread})
		require.NoError(t, err, "a snapshot matching its checksum must load")
		require.NoError(t, dispatchErr, "a snapshot matching its checksum must be relayed")
		dispatchErr, err = tracked.wsProcessSnapshotOrderBook(t.Context(), &orderBookPush{bids: bids, asks: asks, updated: time.UnixMilli(1700000001000), sequenceID: 2, checksum: crc32.ChecksumIEEE([]byte("-2.5:3:0:1"))}, spreadPair, []asset.Item{asset.Spread})
		require.ErrorIs(t, err, errOrderbookChecksumMismatch, "a snapshot failing its checksum must be refused")
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a snapshot failing its checksum must report the book invalid, which starts recovery")
		assert.NoError(t, dispatchErr, "a snapshot failing its checksum should relay nothing")
		_, err = tracked.Websocket.Orderbook.GetOrderbook(spreadPair, asset.Spread)
		assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the book the refused snapshot was to replace should be invalidated")
	})
}

func TestWebsocketProcessUpdateOrderBook(t *testing.T) {
	t.Parallel()

	snapshotTime, pushTime := time.UnixMilli(1659792392540), time.UnixMilli(1659792392640)
	updateBids, updateAsks := orderbook.Levels{{Price: 100.5, Amount: 9}}, orderbook.Levels{{Price: 100.6, Amount: 9}}
	unchanged := &orderbook.Book{LastUpdateID: 10, LastUpdated: snapshotTime, Bids: orderbook.Levels{{Price: 100.5, Amount: 1.25}}, Asks: orderbook.Levels{{Price: 100.6, Amount: 0.75}}}
	testCases := []struct {
		name               string
		instrument         string
		previousSequenceID int64
		sequenceID         int64
		bids, asks         orderbook.Levels
		exp                *orderbook.Book
		expPublished       int
		expectedError      error
		multipleAssets     bool
	}{
		{name: "contiguous update", instrument: "CONT", previousSequenceID: 10, sequenceID: 11, bids: updateBids, asks: updateAsks, exp: &orderbook.Book{LastUpdateID: 11, LastUpdated: pushTime, Bids: updateBids, Asks: updateAsks}, expPublished: 1},
		{name: "documented sequence reset", instrument: "RESET", previousSequenceID: 10, sequenceID: 3, bids: updateBids, asks: updateAsks, exp: &orderbook.Book{LastUpdateID: 3, LastUpdated: pushTime, Bids: updateBids, Asks: updateAsks}, expPublished: 1},
		{name: "stale update", instrument: "STALE", previousSequenceID: 9, sequenceID: 10, bids: updateBids, asks: updateAsks, exp: unchanged},
		{name: "keep-alive", instrument: "KEEPALIVE", previousSequenceID: 10, sequenceID: 10, exp: unchanged},
		{name: "sequence gap invalidates mapped assets", instrument: "GAP", previousSequenceID: 9, sequenceID: 11, bids: updateBids, asks: updateAsks, expectedError: errInvalidOrderbookSequence, multipleAssets: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tracked := new(Exchange)
			require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
			pair := currency.NewPairWithDelimiter(tc.instrument, "USDT", "-")
			assets := []asset.Item{asset.Spot}
			if tc.multipleAssets {
				assets = append(assets, asset.Margin)
			}
			for _, a := range assets {
				require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
					Exchange:     tracked.Name,
					Pair:         pair,
					Asset:        a,
					LastUpdateID: 10,
					LastUpdated:  snapshotTime,
					Bids:         orderbook.Levels{{Price: 100.5, Amount: 1.25}},
					Asks:         orderbook.Levels{{Price: 100.6, Amount: 0.75}},
				}), "LoadSnapshot must not error")
			}
			relayed(tracked)
			dispatchErr, err := tracked.wsProcessUpdateOrderBook(t.Context(), &orderBookPush{
				bids:               tc.bids,
				asks:               tc.asks,
				updated:            pushTime,
				previousSequenceID: tc.previousSequenceID,
				sequenceID:         tc.sequenceID,
			}, pair, assets)
			require.NoError(t, dispatchErr, "wsProcessUpdateOrderBook must relay every book it updates")
			assert.Len(t, relayed(tracked), tc.expPublished, "wsProcessUpdateOrderBook should publish only an applied update")
			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError, "wsProcessUpdateOrderBook must return the expected error")
				for _, a := range assets {
					_, err = tracked.Websocket.Orderbook.GetOrderbook(pair, a)
					require.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "The %s orderbook must be invalidated", a)
				}
				return
			}
			require.NoError(t, err, "wsProcessUpdateOrderBook must not error")
			assert.Equal(t, tc.exp, bookTop(t, tracked, pair, asset.Spot, 5), "the book should hold the update only when it applies")
		})
	}

	t.Run("a gap on any mapped book changes none", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		pair := currency.NewPairWithDelimiter("SEQ", "USDT", "-")
		for a, lastUpdateID := range map[asset.Item]int64{asset.Spot: 10, asset.Margin: 9} {
			require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
				Exchange:     tracked.Name,
				Pair:         pair,
				Asset:        a,
				LastUpdateID: lastUpdateID,
				LastUpdated:  snapshotTime,
				Bids:         orderbook.Levels{{Price: 100.5, Amount: 1.25}},
				Asks:         orderbook.Levels{{Price: 100.6, Amount: 0.75}},
			}), "LoadSnapshot must not error")
		}
		relayed(tracked)
		dispatchErr, err := tracked.wsProcessUpdateOrderBook(t.Context(), &orderBookPush{bids: updateBids, updated: pushTime, previousSequenceID: 10, sequenceID: 11}, pair, []asset.Item{asset.Spot, asset.Margin})
		require.ErrorIs(t, err, errInvalidOrderbookSequence, "the margin gap must be reported")
		assert.NoError(t, dispatchErr, "no book should be relayed when a mapped book has a gap")
		assert.Empty(t, relayed(tracked), "no book should be published when a mapped book has a gap")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err = tracked.Websocket.Orderbook.GetOrderbook(pair, a)
			assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "the %s orderbook should be invalidated", a)
		}
	})

	t.Run("an invalid update keeps the dispatch errors before it", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		pair := currency.NewPairWithDelimiter("UPD", "USDT", "-")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
				Exchange:          tracked.Name,
				Pair:              pair,
				Asset:             a,
				LastUpdateID:      10,
				LastUpdated:       snapshotTime,
				Bids:              orderbook.Levels{{Price: 100.5, Amount: 1.25}},
				Asks:              orderbook.Levels{{Price: 100.6, Amount: 0.75}},
				ValidateOrderbook: a == asset.Margin, // the zero-priced bid applies to spot, then fails margin's validation
			}), "LoadSnapshot must not error")
		}
		for len(tracked.Websocket.DataHandler.C) < cap(tracked.Websocket.DataHandler.C) {
			require.NoError(t, tracked.Websocket.DataHandler.Send(t.Context(), struct{}{}), "DataHandler must accept the saturation payload")
		}
		dispatchErr, err := tracked.wsProcessUpdateOrderBook(t.Context(), &orderBookPush{bids: orderbook.Levels{{Price: 0, Amount: 1}}, updated: pushTime, previousSequenceID: 10, sequenceID: 11}, pair, []asset.Item{asset.Spot, asset.Margin})
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the invalid margin book must be reported")
		assert.NotContains(t, err.Error(), "channel buffer is full", "the spot dispatch failure should be reported apart from the invalid book")
		assert.ErrorContains(t, dispatchErr, "channel buffer is full", "the spot dispatch failure should be reported")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			_, err = tracked.Websocket.Orderbook.GetOrderbook(pair, a)
			assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "the %s orderbook should be invalidated", a)
		}
	})

	t.Run("missing mapped asset invalidates existing depth", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		pair := currency.NewPairWithDelimiter("PART", "USDT", "-")
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange:     tracked.Name,
			Pair:         pair,
			Asset:        asset.Spot,
			LastUpdateID: 20,
			LastUpdated:  time.UnixMilli(1659792392540),
			Bids:         orderbook.Levels{{Price: 100.5, Amount: 1.25}},
			Asks:         orderbook.Levels{{Price: 100.6, Amount: 0.75}},
		}), "LoadSnapshot must not error")
		dispatchErr, err := tracked.wsProcessUpdateOrderBook(t.Context(), &orderBookPush{
			updated:            time.UnixMilli(1659792392640),
			previousSequenceID: 20,
			sequenceID:         21,
		}, pair, []asset.Item{asset.Spot, asset.Margin})
		require.NoError(t, dispatchErr, "no book must be relayed when a mapped book is missing")
		require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "wsProcessUpdateOrderBook must validate every mapped asset")
		assert.NotImplements(t, (*interface{ Unwrap() []error })(nil), err, "wsProcessUpdateOrderBook should report the missing depth alone, not a failure to invalidate it")
		_, err = tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Spot)
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "Spot orderbook must be invalidated on mapped asset error")
		_, err = tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
		require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "Missing margin orderbook must remain missing")
	})

	t.Run("invalid book awaits replacement snapshot", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		pair := currency.NewPairWithDelimiter("PENDING", "USDT", "-")
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange:     tracked.Name,
			Pair:         pair,
			Asset:        asset.Spot,
			LastUpdateID: 20,
			LastUpdated:  time.UnixMilli(1659792392540),
			Bids:         orderbook.Levels{{Price: 100.5, Amount: 1.25}},
			Asks:         orderbook.Levels{{Price: 100.6, Amount: 0.75}},
		}), "LoadSnapshot must not error")
		require.NoError(t, tracked.Websocket.Orderbook.InvalidateOrderbook(pair, asset.Spot), "InvalidateOrderbook must not error")
		dispatchErr, err := tracked.wsProcessUpdateOrderBook(t.Context(), &orderBookPush{
			updated:            time.UnixMilli(1659792392640),
			previousSequenceID: 20,
			sequenceID:         21,
		}, pair, []asset.Item{asset.Spot})
		require.NoError(t, dispatchErr, "no book must be relayed while a replacement snapshot is pending")
		require.ErrorIs(t, err, errOrderbookSnapshotPending, "wsProcessUpdateOrderBook must report that a replacement snapshot is pending")
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsProcessUpdateOrderBook must preserve the invalid orderbook error")
	})

	t.Run("dispatch error preserves mapped depths", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Test instance Setup must not error")
		tracked.Name = t.Name()
		pair := currency.NewPairWithDelimiter("DISPATCH", "USDT", "-")
		assets := []asset.Item{asset.Spot, asset.Margin}
		for _, a := range assets {
			require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
				Exchange:     tracked.Name,
				Pair:         pair,
				Asset:        a,
				LastUpdateID: 10,
				LastUpdated:  time.UnixMilli(1659792392540),
				Bids:         orderbook.Levels{{Price: 100.5, Amount: 1.25}},
				Asks:         orderbook.Levels{{Price: 100.6, Amount: 0.75}},
			}), "LoadSnapshot must not error")
		}
		for len(tracked.Websocket.DataHandler.C) < cap(tracked.Websocket.DataHandler.C) {
			require.NoError(t, tracked.Websocket.DataHandler.Send(t.Context(), struct{}{}), "DataHandler must accept the saturation payload")
		}
		dispatchErr, err := tracked.wsProcessUpdateOrderBook(t.Context(), &orderBookPush{
			bids:               orderbook.Levels{{Price: 100.5, Amount: 2}},
			updated:            time.UnixMilli(1659792392640),
			previousSequenceID: 10,
			sequenceID:         11,
		}, pair, assets)
		require.NoError(t, err, "a dispatch failure must not fail the update")
		require.ErrorContains(t, dispatchErr, "channel buffer is full", "wsProcessUpdateOrderBook must return the dispatch failure")
		for _, a := range assets {
			book, bookErr := tracked.Websocket.Orderbook.GetOrderbook(pair, a)
			require.NoErrorf(t, bookErr, "The %s orderbook must remain valid after a dispatch failure", a)
			assert.Equalf(t, int64(11), book.LastUpdateID, "The %s orderbook should remain aligned after a dispatch failure", a)
		}
	})
}

func TestRefreshEquivalentOrderbookSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("copies the inverse snapshot to the equivalent asset", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Setup must not error")
		tracked.Name = t.Name()
		pair := currency.NewBTCUSDT()
		marginSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{pair}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		spot := &orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Spot,
			LastUpdateID:      123,
			LastUpdated:       time.Unix(123, 0),
			Bids:              orderbook.Levels{{Price: 99, Amount: 2}},
			Asks:              orderbook.Levels{{Price: 100, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), spot), "LoadSnapshot must not error")
		require.NoError(t, tracked.refreshEquivalentOrderbookSnapshot(t.Context(), marginSub), "refreshEquivalentOrderbookSnapshot must not error")
		book, err := tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
		require.NoError(t, err, "GetOrderbook must not error")
		assert.Equal(t, tracked.Name, book.Exchange, "the margin book should keep the exchange")
		assert.Equal(t, pair, book.Pair, "the margin book should keep the pair")
		assert.Equal(t, asset.Margin, book.Asset, "the margin book should be for margin")
		exp := &orderbook.Book{LastUpdateID: 123, LastUpdated: time.Unix(123, 0), Bids: spot.Bids, Asks: spot.Asks}
		assert.Equal(t, exp, bookTop(t, tracked, pair, asset.Margin, 1), "the margin book should take the spot snapshot")
	})

	t.Run("waits for the pushes applied to its instrument", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Setup must not error")
		tracked.Name = t.Name()
		instrument := tracked.orderBookStates.lock("BTC-USDT")
		instrument.Unlock()
		marginSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: subscription.OrderbookChannel, QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`}
		requireWaitsForLock(t, instrument, func() {
			assert.NoError(t, tracked.refreshEquivalentOrderbookSnapshot(t.Context(), marginSub), "refreshEquivalentOrderbookSnapshot should not error")
		})
	})

	t.Run("ignores a missing inverse snapshot", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked), "Setup must not error")
		tracked.Name = t.Name()
		err := tracked.refreshEquivalentOrderbookSnapshot(t.Context(), &subscription.Subscription{
			Asset:            asset.Margin,
			Pairs:            []currency.Pair{currency.NewBTCUSDT()},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`,
		})
		require.NoError(t, err, "a missing inverse snapshot must not error")
		_, err = tracked.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Margin)
		require.ErrorIs(t, err, orderbook.ErrDepthNotFound, "no margin book must be created")
	})

	t.Run("ignores other subscriptions", func(t *testing.T) {
		t.Parallel()
		for _, sub := range []*subscription.Subscription{
			nil,
			{Asset: asset.Spot, Pairs: []currency.Pair{mainPair}, Channel: subscription.TickerChannel},
			{Asset: asset.Spot, Channel: subscription.OrderbookChannel},
			{Asset: asset.PerpetualSwap, Pairs: []currency.Pair{perpetualSwapPair}, Channel: subscription.OrderbookChannel},
		} {
			assert.NoError(t, e.refreshEquivalentOrderbookSnapshot(t.Context(), sub), "a subscription without an equivalent should be ignored")
		}
	})
}

// spreadPublicTradesJSON is a docs push example: no spread trade was pushed while capturing
const spreadPublicTradesJSON = `{"arg":{"channel":"sprd-public-trades", "sprdId": "BTC-USDT_BTC-USDT-SWAP" }, "data": [ { "sprdId": "BTC-USDT_BTC-USDT-SWAP", "tradeId": "2499206329160695808", "px": "-10", "sz": "0.001", "side": "sell", "ts": "1726801105519"}]}`

func TestWebsocketProcessSpreadPublicTrades(t *testing.T) {
	t.Parallel()

	t.Run("public spread trades decode", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.SetTradeFeedStatus(true)
		require.NoError(t, ex.wsProcessSpreadPublicTrades(t.Context(), []byte(spreadPublicTradesJSON)), "wsProcessSpreadPublicTrades must not error")
		exp := []any{trade.Data{
			TID: "2499206329160695808", Exchange: ex.Name, CurrencyPair: spreadPair, AssetType: asset.Spread,
			Side: order.Sell, Price: -10, Amount: 0.001, Timestamp: time.UnixMilli(1726801105519),
		}}
		assert.Equal(t, exp, relayed(ex), "the push should relay its trade with its negative price")
	})

	t.Run("honours the save trade data gate", func(t *testing.T) {
		t.Parallel()
		// The zero size trade is rejected by the trade buffer, which shows
		// whether the buffer was reached. The database is only configured,
		// never connected: the rejected trade never enters the buffer.
		zeroSize := strings.Replace(spreadPublicTradesJSON, `"sz": "0.001"`, `"sz": "0"`, 1)
		require.NotEqual(t, spreadPublicTradesJSON, zeroSize, "the zero size replacement must find its fixture anchor")
		for _, tc := range []struct {
			name       string
			save       bool
			push       string
			expErr     string
			wantAnyErr bool
		}{
			{name: "saving off", save: false, push: zeroSize},
			{name: "saving on", save: true, push: zeroSize, expErr: "received invalid trade data"},
			// The decoder's message differs per JSON backend (stdlib vs sonic),
			// so only the presence of the error is pinned for the malformed push.
			{name: "malformed push with saving on", save: true, push: "{", wantAnyErr: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
				ex.SetTradeFeedStatus(false)
				ex.SetSaveTradeDataStatus(tc.save)
				err := ex.wsProcessSpreadPublicTrades(t.Context(), []byte(tc.push))
				switch {
				case tc.wantAnyErr:
					assert.Error(t, err, "a malformed push should report its decode error")
				case tc.expErr == "":
					assert.NoError(t, err, "a push with trade saving off should not reach the trade buffer")
				default:
					assert.ErrorContains(t, err, tc.expErr, "the push should report the expected error")
				}
			})
		}
	})
}

func TestWebsocketProcessSpreadTickers(t *testing.T) {
	t.Parallel()

	t.Run("live capture", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		handleFixture(t, ex, "sprd-tickers")
		exp := []any{[]ticker.Price{{
			Last: -30.5, High: -30.5, Low: -30.5, Bid: -64.2, BidSize: 0.2297, Ask: -8.7, AskSize: 2.7299, Open: -30.5,
			Pair: spreadPair, ExchangeName: ex.Name, AssetType: asset.Spread, LastUpdated: time.UnixMilli(1795405201579),
		}}}
		assert.Equal(t, exp, relayed(ex), "the spread ticker should be processed and relayed")
	})

	t.Run("maps the 24 hour figures", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.wsProcessSpreadTickers(t.Context(), []byte(`{"arg":{"channel":"sprd-tickers", "sprdId": "BTC-USDT_BTC-USDT-SWAP" }, "data": [ { "sprdId": "BTC-USDT_BTC-USDT-SWAP", "last": "4", "lastSz": "0.01", "askPx": "19.7", "askSz": "5.79", "bidPx": "5.9", "bidSz": "5.79", "open24h": "-7", "high24h": "19.6", "low24h": "-7", "vol24h": "9.87", "ts": "1715247061026"}]}`)), "wsProcessSpreadTickers must not error")
		exp := []any{[]ticker.Price{{
			Last: 4, High: 19.6, Low: -7, Bid: 5.9, BidSize: 5.79, Ask: 19.7, AskSize: 5.79, Open: -7,
			Pair: spreadPair, ExchangeName: ex.Name, AssetType: asset.Spread, LastUpdated: time.UnixMilli(1715247061026),
		}}}
		assert.Equal(t, exp, relayed(ex), "the 24 hour figures should map, without the volume")
	})

	t.Run("empty data relays nothing", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.NoError(t, ex.wsProcessSpreadTickers(t.Context(), []byte(`{"arg":{"channel":"sprd-tickers","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[]}`)), "wsProcessSpreadTickers must not error for empty data")
		assert.Empty(t, ex.Websocket.DataHandler.C, "wsProcessSpreadTickers should not relay an empty batch")
	})

	t.Run("a ticker which fails processing is not relayed", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		ex.Name = t.Name()
		require.Error(t, ex.wsProcessSpreadTickers(t.Context(), []byte(`{"arg":{"channel":"sprd-tickers","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[{"sprdId":"","last":"1"}]}`)), "a ticker without a spread must fail processing")
		assert.Empty(t, ex.Websocket.DataHandler.C, "a ticker which fails processing should not be relayed")
	})
}

func TestWebsocketProcessSpreadCandles(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	handleFixture(t, ex, "sprd-candle1m")
	exp := []any{kline.Item{
		Exchange: ex.Name,
		Pair:     spreadPair,
		Asset:    asset.Spread,
		Interval: kline.OneMin,
		Candles:  []kline.Candle{{Time: time.UnixMilli(1795405200000), Open: -30.5, High: -30.5, Low: -30.5, Close: -30.5}},
	}}
	assert.Equal(t, exp, relayed(ex), "the spread candlestick should be relayed")

	require.ErrorIs(t, ex.wsProcessSpreadCandles(t.Context(), []byte(`{"arg":{"channel":"sprd-candle1m","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[]}`)), kline.ErrNoTimeSeriesDataToConvert, "an empty push must error")
	require.ErrorIs(t, ex.wsProcessSpreadCandles(t.Context(), []byte(`{"arg":{"channel":"sprd-candle1Y","sprdId":"BTC-USDT_BTC-USDT-SWAP"},"data":[["1795405200000","1","1","1","1","1","0"]]}`)), kline.ErrInvalidInterval, "an undocumented interval must error")
}

func TestWebsocketProcessInstruments(t *testing.T) {
	t.Parallel()

	t.Run("live captures cache the instrument ID codes", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		handleFixture(t, ex, "instruments")
		assert.EqualValues(t, 577809, ex.getInstrumentIDCode("BTC-USD_UM-261123-61800-C"), "the option's instrument ID code should be cached")
		assert.EqualValues(t, 577779, ex.getInstrumentIDCode("BTC-UPDOWN-5MIN-261123-1135-1140"), "the event contract's instrument ID code should be cached")
		got := relayed(ex)
		require.Len(t, got, 2, "both pushes must be relayed")
		push, ok := got[0].(*WebsocketInstrumentsPush)
		require.True(t, ok, "the instruments push must be relayed with its type")
		assert.Equal(t, SubscriptionArgument{Channel: channelInstruments, InstrumentType: instTypeOption}, push.Argument, "the push should keep its argument")
		require.NotEmpty(t, push.Data, "the push must carry its instruments")
		assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USD_UM-261123-61800-C", currency.DashDelimiter), push.Data[0].InstrumentID, "the instrument should decode")
	})

	t.Run("instruments push caches ID codes", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
		push := []byte(`{"arg":{"channel":"instruments","instType":"SPOT"},"data":[{"instType":"SPOT","instId":"NEW-USDT","instIdCode":4242}]}`)
		require.NoError(t, ex.wsHandleData(t.Context(), nil, push), "an instruments push must not error")
		assert.EqualValues(t, 4242, ex.getInstrumentIDCode("NEW-USDT"), "the pushed instrument's ID code should be cached")
	})

	t.Run("full relay still updates the code cache", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name string
			code string
			want uint64
		}{
			{name: "new code replaces the old code", code: "4242", want: 4242},
			{name: "null code invalidates the old code", code: "null"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				ex := new(Exchange)
				require.NoError(t, testexch.Setup(ex), "Test instance Setup must not error")
				ex.cacheInstrumentPush([]InstrumentResponse{{InstrumentID: mainPair, InstrumentIDCode: 123}})
				for range cap(ex.Websocket.DataHandler.C) {
					require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), nil), "prefilling the relay must not error")
				}
				push := []byte(`{"arg":{"channel":"instruments","instType":"SPOT"},"data":[{"instType":"SPOT","instId":"BTC-USDT","instIdCode":` + tc.code + `}]}`)
				require.Error(t, ex.wsHandleData(t.Context(), nil, push), "a full relay must report its delivery error")
				assert.Equal(t, tc.want, ex.getInstrumentIDCode("BTC-USDT"), "the cache should reflect the received push even when relay delivery fails")
			})
		}
	})

	t.Run("malformed push", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup must not error")
		require.Error(t, ex.wsProcessInstruments(t.Context(), []byte(`{"data":[{"instIdCode":"x"}]}`)), "a malformed push must error")
		assert.Empty(t, ex.Websocket.DataHandler.C, "a malformed push should not be relayed")
	})
}

func TestWebsocketPublicChannelErrors(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.SetTradeFeedStatus(true)
	for _, tc := range []struct {
		name    string
		handler func(context.Context, []byte) error
	}{
		{"tickers", ex.wsProcessTickers},
		{"candles", ex.wsProcessCandles},
		{"trades", ex.wsProcessTrades},
		{"option trades", ex.wsProcessOptionTrades},
		{"public block trades", ex.wsProcessPublicBlockTrades},
		{"spread public trades", ex.wsProcessSpreadPublicTrades},
		{"spread tickers", ex.wsProcessSpreadTickers},
		{"spread candles", ex.wsProcessSpreadCandles},
		{"order books", func(ctx context.Context, b []byte) error { return ex.wsProcessOrderBooks(ctx, nil, b) }},
		{"spread order books", func(ctx context.Context, b []byte) error { return ex.wsProcessSpreadOrderBooks(ctx, nil, b) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Errorf(t, tc.handler(t.Context(), []byte(`{"data":`)), "%s should report a malformed push", tc.name)
		})
	}
}
