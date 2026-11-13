package cryptodotcom

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// marketDataFile holds market data frames recorded live, by the qualified channel whose subscriptions they answer: a
// set of frames for each subscription in turn, the first frame the subscription's answer, carrying its request's ID,
// and the rest the channel's pushes. The settlement push follows Crypto.com's documented example, as the channel only
// pushes when a future expires
const marketDataFile = "testdata/wsPublic.json"

// marketDataFrames returns the frames recorded in testdata/wsPublic.json, loaded once
var marketDataFrames = recordedFrames(marketDataFile)

// recordedFrames returns a function returning the frames a fixture file records by channel, laid out as
// testdata/wsPublic.json is, which it loads once
func recordedFrames(file string) func() (map[string][][]json.RawMessage, error) {
	return sync.OnceValues(func() (map[string][][]json.RawMessage, error) {
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var frames map[string][][]json.RawMessage
		if err := json.Unmarshal(raw, &frames); err != nil {
			return nil, err
		}
		return frames, nil
	})
}

// recordedFrame returns the frame at index i of the frames answering a channel's first subscription, of those frames
// returns
func recordedFrame(t *testing.T, frames func() (map[string][][]json.RawMessage, error), channel string, i int) []byte {
	t.Helper()
	recorded, err := frames()
	require.NoError(t, err, "the recorded frames must load")
	require.NotEmptyf(t, recorded[channel], "a subscription of %s must be recorded", channel)
	require.Greaterf(t, len(recorded[channel][0]), i, "frame %d of the first subscription of %s must be recorded", i, channel)
	return bytes.Clone(recorded[channel][0][i])
}

// marketDataFrame returns the recorded frame at index i of the frames answering a market data channel's first
// subscription
func marketDataFrame(t *testing.T, channel string, i int) []byte {
	t.Helper()
	return recordedFrame(t, marketDataFrames, channel, i)
}

// The pairs of the instruments the test configuration enables that the market data tests use
var (
	spotBTCUSD   = currency.NewBTCUSD().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	spotBTCUSDT  = currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	spotETHUSD   = currency.NewPair(currency.ETH, currency.USD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	spotCROUSD   = currency.NewPair(currency.CRO, currency.USD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	perpBTCUSD   = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.DashDelimiter)
	perpETHUSD   = currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.DashDelimiter)
	futureBTCUSD = currency.NewPairWithDelimiter("BTCUSD", "261030", currency.DashDelimiter)
)

// The recorded book channels, of deltas and of snapshots, the update of the delta channel's first snapshot and those
// the recorded deltas take its books to after its first subscription and after it is subscribed again, and a delta
// channel recorded until Crypto.com pushed a delta without levels, as it does when a book has not changed, at the frame
// given
const (
	recordedBookSnapshotUpdateID  int64 = 382097463058304
	recordedBookUpdateID          int64 = 382097464465088
	resubscribedBookUpdateID      int64 = 382097469896256
	recordedBookChannel                 = "book.BTC_USD.10"
	recordedSnapshotBookChannel         = "book.ETH_USD.10"
	recordedEmptyDeltaBookChannel       = "book.BTC_USDT.10"
	recordedEmptyDeltaFrame             = 19
)

// enableTradeFeed has an exchange relay the trades its websocket receives to its data handler
func enableTradeFeed(ex *Exchange) {
	setTradeFeed(ex, true)
}

// setTradeFeed sets whether an exchange relays the trades its websocket receives to its data handler
func setTradeFeed(ex *Exchange, enabled bool) {
	ex.SetTradeFeedStatus(enabled)
	ex.Websocket.Trade.Setup(enabled, ex.Websocket.DataHandler)
}

// relayedData returns what an exchange's websocket has relayed to its data handler so far
func relayedData(ex *Exchange) []any {
	var data []any
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			data = append(data, p.Data)
		default:
			return data
		}
	}
}

// answerBookSnapshotFirst has a mock answer the recorded delta book channel's first subscription with its
// acknowledgement and snapshot, holding back the deltas recorded after them, which a test can then push in turn, and
// a later subscription with the frames recorded for it
func answerBookSnapshotFirst(t *testing.T, venue *websocketVenue) {
	t.Helper()
	recorded, err := marketDataFrames()
	require.NoError(t, err, "the recorded frames must load")
	sets := recorded[recordedBookChannel]
	require.Len(t, sets, 2, "the recorded delta book channel must be recorded for two subscriptions")
	venue.answerSubscriptions(recordedBookChannel, sets[0][:2], sets[1])
}

// requireBooksAt waits for the books of a pair of the assets given, else of spot and margin, to reach an update
func requireBooksAt(t *testing.T, ex *Exchange, pair currency.Pair, updateID int64, assets ...asset.Item) {
	t.Helper()
	if len(assets) == 0 {
		assets = asset.Items{asset.Spot, asset.Margin}
	}
	require.Eventuallyf(t, func() bool {
		for _, a := range assets {
			if id, err := ex.Websocket.Orderbook.LastUpdateID(pair, a); err != nil || id != updateID {
				return false
			}
		}
		return true
	}, 10*time.Second, 10*time.Millisecond, "the %s books of %s must reach update %d", assets, pair, updateID)
}

// recordedBookChannelSubscription returns a subscription to the recorded book channel, book.BTC_USD.10
func recordedBookChannelSubscription() subscription.List {
	return subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Pairs: currency.Pairs{spotBTCUSD}}}
}

// recordedSnapshotBook returns an asset's book as the last recorded snapshot of the snapshot book channel loads it
func recordedSnapshotBook(exchangeName string, a asset.Item) *orderbook.Book {
	return &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 2359.74, Amount: 6.7928, OrderCount: 7}, {Price: 2359.73, Amount: 1.4923, OrderCount: 1}, {Price: 2359.69, Amount: 0.2465, OrderCount: 1}, {Price: 2359.61, Amount: 0.0042, OrderCount: 1}, {Price: 2359.6, Amount: 0.8118, OrderCount: 1}, {Price: 2359.59, Amount: 3.2354, OrderCount: 3}, {Price: 2359.56, Amount: 1.3625, OrderCount: 2}, {Price: 2359.54, Amount: 2.6086, OrderCount: 2}, {Price: 2359.49, Amount: 0.7426, OrderCount: 1}, {Price: 2359.48, Amount: 0.4303, OrderCount: 1}},
		Asks:              orderbook.Levels{{Price: 2359.75, Amount: 3.5136, OrderCount: 7}, {Price: 2359.83, Amount: 1.6505, OrderCount: 1}, {Price: 2359.91, Amount: 0.2428, OrderCount: 1}, {Price: 2359.94, Amount: 1.0188, OrderCount: 2}, {Price: 2359.97, Amount: 0.2891, OrderCount: 1}, {Price: 2359.99, Amount: 0.4147, OrderCount: 1}, {Price: 2360, Amount: 0.086, OrderCount: 1}, {Price: 2360.02, Amount: 0.2889, OrderCount: 1}, {Price: 2360.03, Amount: 0.8505, OrderCount: 4}, {Price: 2360.04, Amount: 0.0626, OrderCount: 1}},
		Exchange:          exchangeName,
		Pair:              spotETHUSD,
		Asset:             a,
		LastUpdated:       time.UnixMilli(1792893036530),
		LastPushed:        time.UnixMilli(1792893036570),
		LastUpdateID:      390670048243712,
		ValidateOrderbook: true,
	}
}

// recordedDeltaBook returns an asset's book as the recorded delta book channel's first snapshot and its recorded deltas
// leave it
func recordedDeltaBook(exchangeName string, a asset.Item) *orderbook.Book {
	return &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 80775.53, Amount: 0.32012, OrderCount: 12}, {Price: 80775.52, Amount: 0.0028, OrderCount: 1}, {Price: 80775, Amount: 0.003, OrderCount: 1}, {Price: 80774.47, Amount: 0.00495, OrderCount: 1}, {Price: 80774.18, Amount: 0.0033, OrderCount: 1}, {Price: 80774.17, Amount: 0.00018, OrderCount: 1}, {Price: 80773.45, Amount: 0.0035, OrderCount: 1}, {Price: 80773.23, Amount: 0.02144, OrderCount: 1}, {Price: 80773.22, Amount: 0.06191, OrderCount: 1}, {Price: 80772.86, Amount: 0.0099, OrderCount: 1}},
		Asks:              orderbook.Levels{{Price: 80775.54, Amount: 0.82815, OrderCount: 11}, {Price: 80776.11, Amount: 0.03095, OrderCount: 1}, {Price: 80779.95, Amount: 0.04378, OrderCount: 1}, {Price: 80780.58, Amount: 0.14625, OrderCount: 1}, {Price: 80780.6, Amount: 0.0025, OrderCount: 1}, {Price: 80781.32, Amount: 0.0028, OrderCount: 1}, {Price: 80781.33, Amount: 0.0028, OrderCount: 1}, {Price: 80782.21, Amount: 0.00537, OrderCount: 1}, {Price: 80782.71, Amount: 0.08568, OrderCount: 3}, {Price: 80783.44, Amount: 0.1184, OrderCount: 1}},
		Exchange:          exchangeName,
		Pair:              spotBTCUSD,
		Asset:             a,
		LastUpdated:       time.UnixMilli(1792893127261),
		LastPushed:        time.UnixMilli(1792893127306),
		LastUpdateID:      recordedBookUpdateID,
		ValidateOrderbook: true,
	}
}

// requireBooksLoaded waits for an exchange's spot and margin BTC_USD books to be loaded, and returns when the later of
// them was
func requireBooksLoaded(t *testing.T, ex *Exchange) time.Time {
	t.Helper()
	assets := []asset.Item{asset.Spot, asset.Margin}
	require.Eventually(t, func() bool {
		for _, a := range assets {
			if _, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a); err != nil {
				return false
			}
		}
		return true
	}, 30*time.Second, 10*time.Millisecond, "the spot and margin BTC_USD books must be loaded")
	var inserted time.Time
	for _, a := range assets {
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		if book.InsertedAt.After(inserted) {
			inserted = book.InsertedAt
		}
	}
	return inserted
}

// requireBooksReloaded waits for an exchange's spot and margin BTC_USD books to load a snapshot after one loaded when
// given
func requireBooksReloaded(t *testing.T, ex *Exchange, inserted time.Time) {
	t.Helper()
	require.Eventually(t, func() bool {
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			if book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a); err != nil || !book.InsertedAt.After(inserted) {
				return false
			}
		}
		return true
	}, 30*time.Second, 10*time.Millisecond, "the spot and margin BTC_USD books must load a new snapshot")
}

// mockMarketDataVenue returns a mock of Crypto.com's market data websocket answering each channel's subscriptions with
// the frames recorded for it, for a scenario only the mock can produce; it skips the test in live mode
func mockMarketDataVenue(t *testing.T) *websocketVenue {
	t.Helper()
	venue := newWebsocketVenue(t)
	recorded, err := marketDataFrames()
	require.NoError(t, err, "the recorded frames must load")
	for channel, sets := range recorded {
		venue.answerSubscriptions(channel, sets...)
	}
	return venue
}

func TestWebsocketCandleTimeframe(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for timeframe, interval := range map[string]kline.Interval{
		"1m":  kline.OneMin,
		"5m":  kline.FiveMin,
		"15m": kline.FifteenMin,
		"30m": kline.ThirtyMin,
		"1h":  kline.OneHour,
		"2h":  kline.TwoHour,
		"4h":  kline.FourHour,
		"12h": kline.TwelveHour,
		"1D":  kline.OneDay,
		"7D":  kline.SevenDay,
		"14D": kline.TwoWeek,
		"1M":  kline.OneMonth,
	} {
		t.Run(timeframe, func(t *testing.T) {
			got, err := websocketCandleTimeframe(interval)
			require.NoErrorf(t, err, "websocketCandleTimeframe must not error for %s", interval)
			assert.Equalf(t, timeframe, got, "websocketCandleTimeframe should name %s by its timeframe", interval)
			gotInterval, err := websocketCandleInterval(timeframe)
			require.NoErrorf(t, err, "websocketCandleInterval must not error for %s", timeframe)
			assert.Equalf(t, interval, gotInterval, "websocketCandleInterval should give the interval of %s", timeframe)
		})
	}
	_, err := websocketCandleTimeframe(kline.ThreeMin)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "websocketCandleTimeframe should refuse an interval the candlestick channel does not serve")
	_, err = websocketCandleInterval("1d")
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "websocketCandleInterval should refuse a timeframe Crypto.com does not name")
}

func TestInstrumentAssets(t *testing.T) { //nolint:tparallel // the cases run in order on e, whose available pairs instrumentAssets reads
	t.Parallel()
	_, err := e.instrumentAssets("")
	require.ErrorIs(t, err, errInstrumentNameRequired, "instrumentAssets must require an instrument name")
	// The cases run in order on e, whose available pairs instrumentAssets reads, before the instruments below
	for _, name := range []string{"NOPE_USD", "BTC-USD", "btc_usd", "BTCUSD_PERP", "BTCUSD-INDEX"} {
		t.Run(name, func(t *testing.T) {
			_, err := e.instrumentAssets(name)
			assert.ErrorIsf(t, err, currency.ErrPairNotFound, "instrumentAssets should find no asset trading %s", name)
		})
	}
	oneInchUSD := currency.NewPair(currency.NewCode("1INCH"), currency.USD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	futureBTCUSD261127 := currency.NewPairWithDelimiter("BTCUSD", "261127", currency.DashDelimiter)
	// The cases run in order on e, whose available pairs instrumentAssets reads
	for name, exp := range map[string][]instrumentAsset{
		"BTC_USD":       {{Asset: asset.Spot, Pair: spotBTCUSD}, {Asset: asset.Margin, Pair: spotBTCUSD}},
		"BTC_USDT":      {{Asset: asset.Spot, Pair: spotBTCUSDT}, {Asset: asset.Margin, Pair: spotBTCUSDT}},
		"1INCH_USD":     {{Asset: asset.Spot, Pair: oneInchUSD}},
		"BTCUSD-PERP":   {{Asset: asset.PerpetualSwap, Pair: perpBTCUSD}},
		"BTCUSD-261127": {{Asset: asset.Futures, Pair: futureBTCUSD261127}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := e.instrumentAssets(name)
			require.NoErrorf(t, err, "instrumentAssets must not error for %s", name)
			assert.Equalf(t, exp, got, "instrumentAssets should find each asset %s trades as among the available pairs", name)
		})
	}
}

func TestEnabledInstrumentAssets(t *testing.T) { //nolint:tparallel // the cases run in order on e, whose enabled pairs enabledInstrumentAssets reads
	t.Parallel()
	_, err := e.enabledInstrumentAssets("NOPE_USD")
	require.ErrorIs(t, err, currency.ErrPairNotFound, "enabledInstrumentAssets must refuse an instrument no asset trades")
	// The cases run in order on e, whose enabled pairs enabledInstrumentAssets reads
	for name, exp := range map[string][]instrumentAsset{
		"BTC_USD":       {{Asset: asset.Spot, Pair: spotBTCUSD}, {Asset: asset.Margin, Pair: spotBTCUSD}},
		"BTC_USDT":      {{Asset: asset.Spot, Pair: spotBTCUSDT}},
		"1INCH_USD":     {},
		"BTCUSD-261030": {{Asset: asset.Futures, Pair: futureBTCUSD}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := e.enabledInstrumentAssets(name)
			require.NoErrorf(t, err, "enabledInstrumentAssets must not error for %s", name)
			assert.Equalf(t, exp, got, "enabledInstrumentAssets should find each asset whose enabled pairs hold %s", name)
		})
	}
}

func TestWebsocketBookLevelsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var levels WebsocketBookLevels
	require.NoError(t, json.Unmarshal([]byte(`[["80775.54","0.82815","11"],["80779.94","0","0"]]`), &levels), "Unmarshal must not error")
	exp := WebsocketBookLevels{{Price: 80775.54, Amount: 0.82815, OrderCount: 11}, {Price: 80779.94}}
	assert.Equal(t, exp, levels, "Unmarshal should decode each level's price, quantity and order count")
	assert.Error(t, json.Unmarshal([]byte(`[["x","1","1"]]`), &levels), "Unmarshal should error for a level it cannot decode")
	assert.Error(t, json.Unmarshal([]byte(`{}`), &levels), "Unmarshal should error for levels that are not an array")
}

func TestIntegerUnmarshalJSON(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order before the later calls of the same function, so each also checks that a call keeps
	// nothing from the last
	for _, tc := range []struct {
		name string
		data string
		exp  Integer
	}{
		{"a number", `542048017824`, 542048017824},
		{"a quoted number", `"542048017824"`, 542048017824},
		{"a negative number", `-1`, -1},
		// 2^53 + 1, which a float cannot hold
		{"a number a float cannot hold", `9007199254740993`, 9007199254740993},
		{"a quoted number a float cannot hold", `"9007199254740993"`, 9007199254740993},
		// A JSON string's escapes are decoded first
		{"a quoted number in escapes", `"\u002d\u0031"`, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var i Integer
			require.NoErrorf(t, json.Unmarshal([]byte(tc.data), &i), "Unmarshal must not error for %s", tc.data)
			assert.Equalf(t, tc.exp, i, "Unmarshal should decode %s", tc.data)
		})
	}
	i := Integer(7)
	require.NoError(t, json.Unmarshal([]byte(`null`), &i), "Unmarshal must not error for null")
	assert.Equal(t, Integer(7), i, "Unmarshal should leave an integer unchanged for null")
	// The cases decode in order into the i above, which the calls below decode into too
	for _, tc := range []struct{ name, data string }{
		{"a string that is no number", `"x"`},
		{"a fraction", `1.5`},
		{"a quoted fraction", `"1.5"`},
		{"an empty string", `""`},
		{"a boolean", `true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, json.Unmarshal([]byte(tc.data), &i), errInvalidInteger, "Unmarshal should refuse %s", tc.data)
		})
	}
	assert.ErrorIs(t, i.UnmarshalJSON([]byte(`"6498`)), errInvalidInteger, "UnmarshalJSON should refuse a string it cannot decode")
	// Sonic keeps a string it has read when it refuses what follows it, so the string must not then be parsed
	assert.ErrorIs(t, i.UnmarshalJSON([]byte(`"6498"x`)), errInvalidInteger, "UnmarshalJSON should refuse a string followed by other data")
}

func TestWebsocketBook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for i := 1; i <= 2; i++ {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, recordedSnapshotBookChannel, i)), "wsHandleChannelData must not error for snapshot %d", i)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotETHUSD, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		assert.NotZerof(t, book.InsertedAt, "the %s book should record when it was loaded", a)
		book.InsertedAt = time.Time{}
		assert.Equalf(t, recordedSnapshotBook(ex.Name, a), book, "the %s book should be the last snapshot", a)
	}
	relayed := relayedData(ex)
	assert.Len(t, relayed, 4, "each snapshot should relay the book of each asset its instrument trades as")
	for _, data := range relayed {
		assert.IsType(t, &orderbook.Depth{}, data, "a snapshot should relay the book it loads")
	}
}

func TestWebsocketBookUnmarshalJSON(t *testing.T) {
	t.Parallel()
	// Crypto.com's schema quotes a book's update IDs and times, and gives a delta no update time; its pushes leave them
	// bare and give deltas one
	var book WebsocketBook
	require.NoError(t, json.Unmarshal([]byte(`{"update":{"asks":[],"bids":[]},"t":"1792893127459","u":"382097464750000","pu":"382097464465088","cs":"-12"}`), &book), "Unmarshal must not error for a delta in the schema's form")
	exp := WebsocketBook{
		Update:           &WebsocketBookUpdate{Asks: WebsocketBookLevels{}, Bids: WebsocketBookLevels{}},
		PublishTime:      types.Time(time.UnixMilli(1792893127459)),
		UpdateID:         382097464750000,
		PreviousUpdateID: 382097464465088,
		Checksum:         -12,
	}
	assert.Equal(t, exp, book, "Unmarshal should decode a delta in the schema's form")
	assert.ErrorIs(t, json.Unmarshal([]byte(`{"u":"x"}`), &book), errInvalidInteger, "Unmarshal should refuse an update ID that is no integer")
}

func TestBookUpdateTime(t *testing.T) {
	t.Parallel()
	published, updated := types.Time(time.UnixMilli(1792893127459)), types.Time(time.UnixMilli(1792893127450))
	assert.Equal(t, updated.Time(), bookUpdateTime(&WebsocketBook{PublishTime: published, UpdateTime: updated}), "bookUpdateTime should give when a book was last updated")
	assert.Equal(t, published.Time(), bookUpdateTime(&WebsocketBook{PublishTime: published}), "bookUpdateTime should give when a book was published without its update time")
}

func TestWebsocketBookUpdate(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	for i := 1; i <= 4; i++ {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, recordedBookChannel, i)), "wsHandleChannelData must not error for frame %d", i)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		book.InsertedAt = time.Time{}
		assert.Equalf(t, recordedDeltaBook(ex.Name, a), book, "the %s book should be the snapshot with each delta applied", a)
	}
	assert.Len(t, relayedData(ex), 8, "the snapshot and each delta should relay the book of each asset its instrument trades as")

	// A delta changing a level the book holds gives it the delta's order count as well as its amount
	countDelta := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"book.BTC_USD.10","channel":"book.update","depth":10,"data":[{"update":{"asks":[],"bids":[["80775.53","0.5","13"]]},"t":1792893127409,"tt":1792893127400,"u":382097464465089,"pu":382097464465088,"cs":0}]}}`
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(countDelta)), "wsHandleChannelData must not error for a delta changing a level")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		require.NotEmptyf(t, book.Bids, "the %s book must hold bids", a)
		assert.Equalf(t, orderbook.Level{Price: 80775.53, Amount: 0.5, OrderCount: 13}, book.Bids[0], "the %s book's changed level should take the delta's amount and order count", a)
	}

	// A delta without levels, which Crypto.com pushes when a book has not changed, moves the book on to its update alone
	for i := 1; i < recordedEmptyDeltaFrame; i++ {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, recordedEmptyDeltaBookChannel, i)), "wsHandleChannelData must not error for frame %d", i)
	}
	before, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSDT, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, recordedEmptyDeltaBookChannel, recordedEmptyDeltaFrame)), "wsHandleChannelData must not error for a delta without levels")
	after, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSDT, asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error")
	expEmpty := *before
	expEmpty.LastUpdated, expEmpty.LastPushed, expEmpty.LastUpdateID = time.UnixMilli(1792908878459), time.UnixMilli(1792908878463), 382144301981024
	expEmpty.InsertedAt, after.InsertedAt = time.Time{}, time.Time{}
	assert.Equal(t, &expEmpty, after, "a delta without levels should move the book on to its update, leaving its levels")

	noUpdate := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"book.BTC_USD.10","channel":"book.update","depth":10,"data":[{"t":1792893127509,"tt":1792893127500,"u":382097464800000,"pu":382097464700000,"cs":0}]}}`
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, []byte(noUpdate)), errBookDeltaHasNoUpdate, "wsHandleChannelData should refuse a delta without its update")
}

func TestWebsocketBookResubscription(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can send deltas out of sequence or that break a book, and fail an unsubscription")
	}
	t.Run("REST refresh during a live book", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		answerBookSnapshotFirst(t, venue)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		restBook, err := ex.UpdateOrderbook(t.Context(), spotBTCUSD, asset.Spot)
		require.NoError(t, err, "UpdateOrderbook must not error during a live book")
		assert.True(t, restBook.RestSnapshot, "UpdateOrderbook should return the REST snapshot")
		assert.Equal(t, testOrderbooks[asset.Spot].Bids, restBook.Bids, "UpdateOrderbook should return the REST snapshot's bids")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		for i := 2; i <= 4; i++ {
			require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, i)), "delta %d must extend the live book", i)
		}
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		assert.Empty(t, venue.received(websocketMethodUnsubscribe), "the REST refresh should leave the live book subscribed")
	})
	t.Run("REST refresh when the websocket book is unavailable", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name        string
			stop        func(*testing.T, *Exchange, websocket.Connection, *websocketVenue)
			invalidated []asset.Item
		}{
			{name: "invalidated", stop: func(t *testing.T, ex *Exchange, _ websocket.Connection, _ *websocketVenue) {
				t.Helper()
				require.NoError(t, ex.Websocket.Orderbook.InvalidateOrderbook(spotBTCUSD, asset.Spot), "InvalidateOrderbook must not error")
			}, invalidated: []asset.Item{asset.Spot}},
			{name: "unsubscribed", stop: func(t *testing.T, ex *Exchange, conn websocket.Connection, _ *websocketVenue) {
				t.Helper()
				s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
				require.NotNil(t, s, "the book channel must be stored against its connection")
				require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subscription.List{s}), "UnsubscribeChannels must not error")
			}, invalidated: []asset.Item{asset.Spot, asset.Margin}},
			{name: "unsubscription unanswered", stop: func(t *testing.T, ex *Exchange, conn websocket.Connection, venue *websocketVenue) {
				t.Helper()
				s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
				require.NotNil(t, s, "the book channel must be stored against its connection")
				venue.loseAnswers(websocketMethodUnsubscribe, 1)
				require.ErrorIs(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subscription.List{s}), websocket.ErrSignatureTimeout, "UnsubscribeChannels must return the unanswered request's timeout")
			}, invalidated: []asset.Item{asset.Spot, asset.Margin}},
			{name: "shut down", stop: func(t *testing.T, ex *Exchange, _ websocket.Connection, _ *websocketVenue) {
				t.Helper()
				// Disabled first, so the connection monitor cannot reconnect and reload the books before the REST refresh
				require.NoError(t, ex.Websocket.Disable(), "Disable must not error")
				if err := ex.Websocket.Shutdown(); err != nil {
					require.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown must only fail when the connection monitor has shut the connection down first")
				}
			}, invalidated: []asset.Item{asset.Spot, asset.Margin}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				venue := newMarketDataVenue(t)
				answerBookSnapshotFirst(t, venue)
				ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), withTestResponseMaxLimit(t, venue))
				require.NoError(t, err, "Connect must not error")
				conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
				require.NoError(t, err, "GetConnection must not error")
				requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
				tc.stop(t, ex, conn, venue)
				for _, a := range tc.invalidated {
					_, err := orderbook.Get(ex.Name, spotBTCUSD, a)
					assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "the %s book should be invalid once the websocket no longer maintains it", a)
				}
				restBook, err := ex.UpdateOrderbook(t.Context(), spotBTCUSD, asset.Spot)
				require.NoError(t, err, "UpdateOrderbook must not error")
				stored, err := orderbook.Get(ex.Name, spotBTCUSD, asset.Spot)
				require.NoError(t, err, "orderbook.Get must return the stored REST snapshot")
				assert.True(t, stored.RestSnapshot, "the REST snapshot should replace the websocket book")
				assert.Equal(t, restBook.Bids, stored.Bids, "the store should hold the REST snapshot's bids")
				if !ex.Websocket.IsEnabled() {
					// The cleanup disables the websocket, which it must find enabled
					require.NoError(t, ex.Websocket.Enable(t.Context()), "Enable must not error")
				}
			})
		}
	})
	t.Run("a delta out of sequence", func(t *testing.T) {
		t.Parallel()
		t.Run("a delta repeated", func(t *testing.T) {
			t.Parallel()
			venue := newMarketDataVenue(t)
			ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)

			// The second recorded delta again, which follows an update before the books' last
			err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
			require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta out of sequence")
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)

			unsubscribes := venue.received(websocketMethodUnsubscribe)
			require.Len(t, unsubscribes, 1, "the book channel must be unsubscribed once")
			assert.JSONEq(t, `{"channels":["book.BTC_USD.10"]}`, string(unsubscribes[0].Params), "the book channel should be unsubscribed")
			subscribes := venue.received(websocketMethodSubscribe)
			require.Len(t, subscribes, 2, "the book channel must be subscribed again")
			assert.JSONEq(t, `{"channels":["book.BTC_USD.10"],"book_subscription_type":"SNAPSHOT_AND_UPDATE","book_update_frequency":100}`, string(subscribes[1].Params), "the book channel should be subscribed again for deltas")
			// The books take the snapshot the acknowledgement carries before the resubscription stores the channel again
			assert.Eventually(t, func() bool {
				s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
				return s != nil && s.State() == subscription.SubscribedState
			}, 10*time.Second, 10*time.Millisecond, "the book channel should be stored against its connection again, subscribed")
		})

		t.Run("a delta following a missed one", func(t *testing.T) {
			t.Parallel()
			venue := newMarketDataVenue(t)
			answerBookSnapshotFirst(t, venue)
			ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)

			// The second recorded delta, which follows the first, never taken
			err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
			require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta following an update the books never took")
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
			assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "the book channel should be unsubscribed once")
		})

		t.Run("a delta of a pair disabled since it subscribed", func(t *testing.T) {
			t.Parallel()
			venue := newMarketDataVenue(t)
			ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
			for _, a := range []asset.Item{asset.Spot, asset.Margin} {
				require.NoErrorf(t, ex.CurrencyPairs.DisablePair(a, spotBTCUSD), "DisablePair must not error for %s", a)
			}

			// A delta out of sequence, which the books of an enabled pair are resubscribed for
			require.NoError(t, ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3)), "wsHandleData must drop a delta of a pair disabled since it subscribed")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
			assert.Never(t, func() bool { return len(venue.received(websocketMethodUnsubscribe)) != 0 }, 100*time.Millisecond, 10*time.Millisecond, "a delta of a disabled pair should not resubscribe its channel")
		})
	})

	t.Run("books out of step", func(t *testing.T) {
		t.Parallel()
		// Each book's last update is the snapshot's, which the first recorded delta follows, so that only the books being out
		// of step can resynchronise them
		t.Run("a book missing from an asset enabled since the snapshot", func(t *testing.T) {
			t.Parallel()
			venue := newMarketDataVenue(t)
			answerBookSnapshotFirst(t, venue)
			ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), func(ex *Exchange) {
				require.NoError(t, ex.CurrencyPairs.DisablePair(asset.Margin, spotBTCUSD), "DisablePair must not error")
			})
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID, asset.Spot)
			require.NoError(t, ex.CurrencyPairs.EnablePair(asset.Margin, spotBTCUSD), "EnablePair must not error")

			err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 2))
			require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report books out of step")
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
			assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "the book channel should be unsubscribed once")
		})
		t.Run("a book invalidated", func(t *testing.T) {
			t.Parallel()
			venue := newMarketDataVenue(t)
			answerBookSnapshotFirst(t, venue)
			ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
			require.NoError(t, ex.Websocket.Orderbook.InvalidateOrderbook(spotBTCUSD, asset.Margin), "InvalidateOrderbook must not error")
			// The mock holds back the resubscription, whose snapshot would make the books valid again, until they are checked
			checked := make(chan struct{})
			release := sync.OnceFunc(func() { close(checked) })
			t.Cleanup(release)
			venue.beforeAnswering(websocketMethodUnsubscribe, func() { <-checked })

			err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 2))
			require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report books out of step")
			_, err = ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, asset.Spot)
			assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the spot book should be invalidated with the margin book")
			release()
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
			assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "the book channel should be unsubscribed once")
		})
	})

	t.Run("a delta that breaks a book", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)

		broken := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"book.BTC_USD.10","channel":"book.update","depth":10,"data":[{"update":{"asks":[],"bids":[["0","1","1"]]},"t":1792893127409,"tt":1792893127400,"u":382097464700000,"pu":382097464465088,"cs":0}]}}`
		err = ex.wsHandleData(t.Context(), conn, []byte(broken))
		require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "wsHandleData must report a delta that breaks a book")
		requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
		assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "the book channel should be unsubscribed once")
	})

	t.Run("retried by a later delta", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		s := conn.Subscriptions().Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}})
		require.NotNil(t, s, "the book channel must be stored against its connection")

		// A delta arriving while the channel resubscribes or unsubscribes starts no resubscription of its own
		require.NoError(t, s.SetState(subscription.ResubscribingState), "SetState must not error")
		err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
		require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta out of sequence")
		require.NoError(t, s.SetState(subscription.UnsubscribingState), "SetState must not error")
		require.NoError(t, ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 4)), "wsHandleData must drop a delta while its books await a snapshot")
		assert.Never(t, func() bool { return len(venue.received(websocketMethodUnsubscribe)) != 0 }, 100*time.Millisecond, 10*time.Millisecond, "a delta should not resubscribe a channel resubscribing or unsubscribing")

		// A failed unsubscription leaves the channel subscribed, for a later delta to retry
		require.NoError(t, s.SetState(subscription.SubscribedState), "SetState must not error")
		venue.fail(websocketMethodUnsubscribe, &APIError{Code: 50001, Message: "ERR_INTERNAL"})
		require.NoError(t, ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 4)), "wsHandleData must drop a delta while its books await a snapshot")
		require.Eventually(t, func() bool { return len(venue.received(websocketMethodUnsubscribe)) == 1 }, 10*time.Second, 10*time.Millisecond, "the book channel must be unsubscribed")
		require.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 10*time.Second, 10*time.Millisecond, "a failed unsubscription must leave the book channel subscribed")
		assert.Len(t, venue.received(websocketMethodSubscribe), 1, "the book channel should not be subscribed again after its unsubscription failed")

		venue.fail(websocketMethodUnsubscribe, nil)
		require.NoError(t, ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 4)), "wsHandleData must drop a delta while its books await a snapshot")
		requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
		assert.Len(t, venue.received(websocketMethodUnsubscribe), 2, "the book channel should be unsubscribed again")
		assert.Len(t, venue.received(websocketMethodSubscribe), 2, "the book channel should be subscribed again")
	})

	// An unanswered unsubscription may still have stopped the channel's deltas, which would then never ask for the
	// snapshot the books await. The mock refuses to subscribe a channel its connection holds, so that the channel is
	// subscribed again only once it is unsubscribed
	for _, tc := range []struct {
		name string
		lose func(v *websocketVenue, method string, n int)
	}{
		{name: "unsubscribe applied without an acknowledgement", lose: (*websocketVenue).loseAnswers},
		{name: "unsubscribe unanswered before being applied", lose: (*websocketVenue).loseRequests},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			venue := newMarketDataVenue(t)
			venue.trackSubscriptions()
			// The trade connection is authenticated too, as a recovery connecting the websocket again would drop it and with
			// it any orders it placed under cancel on disconnect
			ex, err := newUserDataTestExchange(t, venue, recordedBookChannelSubscription(), withTestResponseMaxLimit(t, venue))
			require.NoError(t, err, "Connect must not error")
			conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
			require.NoError(t, err, "GetConnection must not error")
			tradeConn, err := ex.Websocket.GetConnection(websocketTradeConnection)
			require.NoError(t, err, "GetConnection must not error")
			requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)

			tc.lose(venue, websocketMethodUnsubscribe, 1)
			err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
			require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta out of sequence")
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
			unsubscribes := venue.received(websocketMethodUnsubscribe)
			require.Len(t, unsubscribes, 2, "the book channel must be unsubscribed again once its unsubscription goes unanswered")
			assert.Equal(t, unsubscribes[0].Connection, unsubscribes[1].Connection, "the book channel should be unsubscribed again on its connection")
			assert.Len(t, venue.received(websocketMethodSubscribe), 2, "the book channel should be subscribed again once")

			current, err := ex.Websocket.GetConnection(websocketTradeConnection)
			require.NoError(t, err, "GetConnection must not error")
			assert.Same(t, tradeConn, current, "the trade connection should be kept")
			_, err = ex.WebsocketAPI().GetCancelOnDisconnect(t.Context())
			require.NoError(t, err, "the trade connection must stay usable")
			auths := venue.received(websocketMethodAuth)
			require.Len(t, auths, 1, "the trade connection must be authenticated once, not again on a new connection")
			requests := venue.received("private/get-cancel-on-disconnect")
			require.Len(t, requests, 1, "GetCancelOnDisconnect must send one request")
			assert.Equal(t, auths[0].Connection, requests[0].Connection, "the trade connection should be the one authenticated when the websocket connected")
		})
	}
}

func TestInvalidateUnsubscribedBooks(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	unlisted := currency.NewPair(currency.NewCode("ZZZ"), currency.USD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	load := func(p currency.Pair, a asset.Item) {
		t.Helper()
		require.NoErrorf(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{Exchange: ex.Name, Pair: p, Asset: a, Bids: orderbook.Levels{{Price: 1, Amount: 1}}, Asks: orderbook.Levels{{Price: 2, Amount: 1}}, LastUpdated: time.Now()}), "LoadSnapshot must load the %s %s book", a, p)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		load(spotBTCUSD, a)
		load(spotETHUSD, a)
	}
	load(unlisted, asset.Spot)
	require.NoError(t, (&orderbook.Book{Exchange: ex.Name, Pair: spotETHUSD, Asset: asset.Margin, Bids: orderbook.Levels{{Price: 3, Amount: 1}}, LastUpdated: time.Now()}).Process(), "Process must store a REST snapshot over the margin ETH_USD book")

	ex.invalidateUnsubscribedBooks(&subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD, spotETHUSD, unlisted}})
	for _, tc := range []struct {
		pair    currency.Pair
		asset   asset.Item
		invalid bool
	}{
		{pair: spotBTCUSD, asset: asset.Spot, invalid: true},
		{pair: spotBTCUSD, asset: asset.Margin, invalid: true},
		{pair: spotETHUSD, asset: asset.Spot, invalid: true},
		{pair: spotETHUSD, asset: asset.Margin},
		{pair: unlisted, asset: asset.Spot, invalid: true},
	} {
		_, err := orderbook.Get(ex.Name, tc.pair, tc.asset)
		if tc.invalid {
			assert.ErrorIsf(t, err, errBookUnsubscribed, "invalidateUnsubscribedBooks should invalidate the %s %s book the channel loaded", tc.asset, tc.pair)
		} else {
			assert.NoErrorf(t, err, "invalidateUnsubscribedBooks should leave the %s %s book a REST snapshot replaced", tc.asset, tc.pair)
		}
	}
	assert.NotPanics(t, func() {
		ex.invalidateUnsubscribedBooks(&subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotCROUSD}})
	}, "invalidateUnsubscribedBooks should pass over a pair without a book")
}

func TestWebsocketBookSubscriptionRetry(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can refuse a subscription")
	}
	t.Run("a refused resubscription retried", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		key := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}}
		s := conn.Subscriptions().Get(key)
		require.NotNil(t, s, "the book channel must be stored against its connection")

		// The channel is unsubscribed for a new snapshot, then refused when subscribed again, leaving no delta to retry it. The
		// mock clears the refusal as it answers that subscription, so that the retry is accepted whatever the test's timing
		venue.rejectSubscriptions(recordedBookChannel, &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"})
		venue.beforeAnswering(websocketMethodSubscribe, func() { venue.rejectSubscriptions(recordedBookChannel, nil) })
		err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
		require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta out of sequence")
		require.Eventually(t, func() bool { return len(venue.received(websocketMethodSubscribe)) >= 2 }, 10*time.Second, 10*time.Millisecond, "the book channel must be subscribed again")
		requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
		require.Eventually(t, func() bool {
			stored := conn.Subscriptions().Get(key)
			return stored != nil && stored.State() == subscription.SubscribedState
		}, 10*time.Second, 10*time.Millisecond, "the retried book channel must be stored against its connection again, subscribed")
		assert.Len(t, venue.received(websocketMethodSubscribe), 3, "the refused book channel should be subscribed once more")
		assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "the refused book channel should not be unsubscribed again")

		// TestRetryBookSubscription checks how soon the retry stops on a synctest bubble's clock, which a stall of the test
		// cannot move
		ex.retryBookSubscription(t.Context(), conn, s)
		assert.Len(t, venue.received(websocketMethodSubscribe), 3, "retryBookSubscription should not subscribe a book channel held again")
	})

	t.Run("a channel no longer configured left out", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		tickers := subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}}
		ex, err := newMarketDataTestExchange(t, venue, append(recordedBookChannelSubscription(), tickers...))
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		key := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}}
		s := conn.Subscriptions().Get(key)
		require.NotNil(t, s, "the book channel must be stored against its connection")
		venue.rejectSubscriptions(recordedBookChannel, &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"})
		require.Error(t, ex.Websocket.ResubscribeFromConnection(t.Context(), conn, subscription.List{s}), "ResubscribeFromConnection must fail for a refused subscription")

		// The book channel is no longer configured by the time it is retried, as a flush applying the change makes it
		ex.Features.Subscriptions = tickers
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "FlushChannels must not error")
		venue.rejectSubscriptions(recordedBookChannel, nil)
		ex.retryBookSubscription(t.Context(), conn, s)
		assert.Nil(t, ex.Websocket.GetSubscription(key), "retryBookSubscription should not subscribe a book channel no longer configured")
		assert.Len(t, venue.received(websocketMethodSubscribe), 3, "retryBookSubscription should not subscribe the book channel again")
	})

	t.Run("given up once its attempts run out", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), func(ex *Exchange) {
			ex.bookResubscribeDelay = 10 * time.Millisecond
		})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)

		venue.rejectSubscriptions(recordedBookChannel, &APIError{Code: 40003, Message: "BAD_REQUEST"})
		start := time.Now()
		err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
		require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta out of sequence")
		// The first subscription, the resubscription and each retry
		subscribes := 2 + bookResubscribeAttempts
		require.Eventuallyf(t, func() bool { return len(venue.received(websocketMethodSubscribe)) == subscribes }, 10*time.Second, 10*time.Millisecond, "the book channel must be subscribed %d times", subscribes)
		// The retries wait 1, 2, 4, 8 and 16 times the first delay
		assert.GreaterOrEqual(t, time.Since(start), 31*ex.bookResubscribeDelay, "each retry should wait twice as long as the last")
		assert.Never(t, func() bool { return len(venue.received(websocketMethodSubscribe)) > subscribes }, 500*time.Millisecond, 10*time.Millisecond, "the book channel should not be retried once its attempts run out")
		assert.Nil(t, ex.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}}), "a book channel whose attempts run out should be left unsubscribed")
	})
}

// bookRetryConnection stands in for a connection to the mock websocket in the synctest bubbles timing
// retryBookSubscription, as the reader of a connection to the mock blocks on its socket, which would keep a bubble's
// clock from moving; it counts how often it is asked whether it is connected, and holds the subscriptions stored
// against it
type bookRetryConnection struct {
	websocket.Connection
	connected     bool
	connectChecks uint64
	subscriptions *subscription.Store
}

// IsConnected reports whether the connection is connected, counting each check
func (c *bookRetryConnection) IsConnected() bool {
	c.connectChecks++
	return c.connected
}

// Subscriptions returns the subscriptions stored against the connection
func (c *bookRetryConnection) Subscriptions() *subscription.Store {
	return c.subscriptions
}

func TestRetryBookSubscription(t *testing.T) {
	t.Parallel()
	t.Run("a book channel subscribed again over a connection", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), func(ex *Exchange) {
			ex.bookResubscribeDelay = 10 * time.Millisecond
		})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		inserted := requireBooksLoaded(t, ex)
		key := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}}
		s := conn.Subscriptions().Get(key)
		require.NotNil(t, s, "the book channel must be stored against its connection")
		// A resubscription refused once its unsubscription is answered leaves the channel so: unsubscribed, and off its
		// connection
		require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subscription.List{s}), "UnsubscribeChannels must not error")
		require.NoError(t, conn.Subscriptions().Remove(s), "Remove must not error")

		ex.retryBookSubscription(t.Context(), conn, s)
		assert.Same(t, s, ex.Websocket.GetSubscription(key), "retryBookSubscription should subscribe the book channel again")
		assert.Same(t, s, conn.Subscriptions().Get(key), "retryBookSubscription should store the book channel against its connection again")
		assert.Equal(t, subscription.SubscribedState, s.State(), "retryBookSubscription should leave the book channel subscribed")
		requireBooksReloaded(t, ex, inserted)
		if mockTests {
			assert.Len(t, venue.received(websocketMethodSubscribe), 2, "retryBookSubscription should subscribe the book channel once more")
		}
	})

	const delay = 50 * time.Millisecond
	refuse := func(*Exchange, *bookRetryConnection) error {
		return &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"}
	}
	for _, tc := range []struct {
		name      string
		done      bool
		doneAfter time.Duration
		connected bool
		// held has the book channel held again, and storedBefore stored against its connection already
		held, storedBefore bool
		// answer answers each attempt's subscription, and can change what the retry checks before its next attempt
		answer        func(*Exchange, *bookRetryConnection) error
		elapsed       time.Duration
		connectChecks uint64
		// subscribed holds when each attempt subscribed, from the retry's start
		subscribed []time.Duration
		stored     bool
	}{
		{name: "context done", done: true, connected: true, answer: refuse},
		{name: "connection lost", answer: refuse, elapsed: delay, connectChecks: 1},
		{name: "book channel held again", connected: true, held: true, answer: refuse, elapsed: delay, connectChecks: 1},
		{name: "context done while waiting", doneAfter: delay / 2, connected: true, answer: refuse, elapsed: delay / 2},
		// The attempts wait 1, 2, 4, 8 and 16 times the first delay
		{name: "attempts run out", connected: true, answer: refuse, elapsed: 31 * delay, connectChecks: bookResubscribeAttempts, subscribed: []time.Duration{delay, 3 * delay, 7 * delay, 15 * delay, 31 * delay}},
		{name: "subscribed", connected: true, answer: func(*Exchange, *bookRetryConnection) error { return nil }, elapsed: delay, connectChecks: 1, subscribed: []time.Duration{delay}, stored: true},
		{name: "subscribed while stored against its connection already", connected: true, storedBefore: true, answer: func(*Exchange, *bookRetryConnection) error { return nil }, elapsed: delay, connectChecks: 1, subscribed: []time.Duration{delay}, stored: true},
		{name: "configuration removed after a refusal", connected: true, answer: func(ex *Exchange, conn *bookRetryConnection) error {
			ex.Features.Subscriptions = nil
			return refuse(ex, conn)
		}, elapsed: 3 * delay, connectChecks: 2, subscribed: []time.Duration{delay}},
		{name: "connection lost after a refusal", connected: true, answer: func(ex *Exchange, conn *bookRetryConnection) error {
			conn.connected = false
			return refuse(ex, conn)
		}, elapsed: 3 * delay, connectChecks: 2, subscribed: []time.Duration{delay}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Features.Subscriptions = recordedBookChannelSubscription()
			ex.bookResubscribeDelay = delay
			// The retry waits on a synctest bubble's clock, which moves only while everything in the bubble waits, so a
			// stall of the test adds nothing to the time measured. The manager is made in the bubble, as a channel made
			// outside it that the retry waited on would keep the clock from moving
			synctest.Test(t, func(t *testing.T) {
				ex.Websocket = websocket.NewManager()
				conn := &bookRetryConnection{connected: tc.connected, subscriptions: subscription.NewStore()}
				start := time.Now()
				var subscribed []time.Duration
				ex.Websocket.Subscriber = func(subscription.List) error {
					subscribed = append(subscribed, time.Since(start))
					return tc.answer(ex, conn)
				}
				s := &subscription.Subscription{QualifiedChannel: recordedBookChannel}
				require.True(t, ex.isConfiguredMarketDataChannel(s.QualifiedChannel), "the book channel must be configured, so that only the stop under test can end the retry")
				if tc.held {
					require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, s), "AddSuccessfulSubscriptions must not error")
				}
				if tc.storedBefore {
					require.NoError(t, conn.subscriptions.Add(s), "Add must not error")
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if tc.done {
					cancel()
				}
				if tc.doneAfter != 0 {
					time.AfterFunc(tc.doneAfter, cancel)
				}
				ex.retryBookSubscription(ctx, conn, s)
				assert.Equal(t, tc.elapsed, time.Since(start), "retryBookSubscription should stop after waiting as long as expected")
				assert.Equal(t, tc.connectChecks, conn.connectChecks, "retryBookSubscription should check its connection as often as expected")
				assert.Equal(t, tc.subscribed, subscribed, "retryBookSubscription should subscribe the book channel when expected")
				assert.Equal(t, tc.stored, conn.subscriptions.Get(s) != nil, "retryBookSubscription should store only a book channel it subscribed against its connection")
			})
		})
	}

	t.Run("a connection that cannot say whether it is connected", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		ex.Features.Subscriptions = recordedBookChannelSubscription()
		ex.bookResubscribeDelay = delay
		synctest.Test(t, func(t *testing.T) {
			ex.Websocket = websocket.NewManager()
			conn := newScriptedConnection()
			start := time.Now()
			var subscribed []time.Duration
			ex.Websocket.Subscriber = func(subscription.List) error {
				subscribed = append(subscribed, time.Since(start))
				return nil
			}
			s := &subscription.Subscription{QualifiedChannel: recordedBookChannel}
			ex.retryBookSubscription(t.Context(), conn, s)
			assert.Equal(t, []time.Duration{delay}, subscribed, "retryBookSubscription should subscribe the book channel over a connection that cannot say whether it is connected")
			assert.Same(t, s, conn.subscriptions.Get(s), "retryBookSubscription should store the book channel it subscribed against its connection")
		})
	})

	t.Run("connection to the mock lost", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("a lost connection is tested against the mock websocket")
		}
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		// The mock can still be pushing the book channel's recorded frames, which it cannot write once the connection is shut
		venue.closing()
		if err := ex.Websocket.Shutdown(); err != nil {
			require.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown must only fail when the traffic monitor has shut the connection down first")
		}
		// retryBookSubscription asks its connection whether it is connected, as bookRetryConnection answers above
		c, ok := conn.(interface{ IsConnected() bool })
		require.True(t, ok, "the market data connection must report whether it is connected")
		assert.False(t, c.IsConnected(), "the market data connection should report it is not connected once shut")
	})
}

func TestQualifiedChannelKeyMatch(t *testing.T) {
	t.Parallel()
	k := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ticker.BTC_USD"}}
	for _, tc := range []struct {
		name string
		key  subscription.MatchableKey
		exp  bool
	}{
		{"a subscription with the key's qualified channel", &subscription.ExactKey{Subscription: &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker.BTC_USD"}}, true},
		{"a subscription with the key's qualified channel, whatever else it names", &subscription.ExactKey{Subscription: &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Margin, Pairs: currency.Pairs{spotBTCUSD}, QualifiedChannel: "ticker.BTC_USD"}}, true},
		{"another key of the same qualified channel", qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ticker.BTC_USD"}}, true},
		{"a subscription with another qualified channel", &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "ticker.BTC_USDT"}}, false},
		{"a qualified channel differing only in case", &subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "ticker.btc_usd"}}, false},
		{"a key without a subscription", &subscription.ExactKey{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, k.Match(tc.key), "Match should report whether a key's subscription has the key's qualified channel")
		})
	}

	t.Run("a subscription found in a store", func(t *testing.T) {
		t.Parallel()
		store := subscription.NewStore()
		s := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, QualifiedChannel: "ticker.BTC_USD"}
		require.NoError(t, store.Add(s), "Add must not error")
		assert.Same(t, s, store.Get(k), "a store should find the subscription with the key's qualified channel")
		assert.Nil(t, store.Get(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ticker.ETH_USD"}}), "a store should find no subscription without the key's qualified channel")
	})
}

func TestQualifiedChannelKeyGetSubscription(t *testing.T) {
	t.Parallel()
	s := &subscription.Subscription{QualifiedChannel: "ticker.BTC_USD"}
	assert.Same(t, s, qualifiedChannelKey{s}.GetSubscription(), "GetSubscription should return the subscription holding the qualified channel")
	assert.Nil(t, qualifiedChannelKey{}.GetSubscription(), "GetSubscription should return no subscription for a key without one")
}

func TestQualifiedChannelKeyString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "ticker.BTC_USD", qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ticker.BTC_USD"}}.String(), "String should give the qualified channel")
	assert.Empty(t, qualifiedChannelKey{&subscription.Subscription{}}.String(), "String should give no channel for a subscription without a qualified channel")
}

func TestWebsocketTicker(t *testing.T) { //nolint:tparallel // the cases share one exchange and its data handler in order
	t.Parallel()
	ex := newTestExchange(t)
	btcUSDChange, btcUSDTChange, perpChange := -0.02, 0.0, -0.13
	// The cases share ex and drain its data handler in order, before the checks below on the same exchange
	for channel, exp := range map[string][]ticker.Price{
		"ticker.BTC_USD": {
			{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.05028, Ask: 80787.85, AskSize: 0.5319, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: btcUSDChange, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1792892978711)},
			{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.05028, Ask: 80787.85, AskSize: 0.5319, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: btcUSDChange, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.UnixMilli(1792892978711)},
		},
		// The volume value is in USD, which is not this pair's quote currency
		"ticker.BTC_USDT": {
			{Last: 80956.49, High: 81099.29, Low: 79232.39, Bid: 80951.66, BidSize: 0.01872, Ask: 80951.67, AskSize: 0.20126, BaseVolume: 1071.3429, PercentChange24Hour: btcUSDTChange, Pair: spotBTCUSDT, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1792894311364)},
		},
		"ticker.BTCUSD-PERP": {
			{Last: 80750.3, High: 81004.7, Low: 79117.7, Bid: 80750.3, BidSize: 0.0087, Ask: 80750.4, AskSize: 0.0331, BaseVolume: 2512.7932, QuoteVolume: 201559679.09, PercentChange24Hour: perpChange, OpenInterest: 6179.573, Pair: perpBTCUSD, ExchangeName: ex.Name, AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1792893199728)},
		},
	} {
		t.Run(channel, func(t *testing.T) {
			require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, channel, 0)), "wsHandleChannelData must not error for %s", channel)
			relayed := relayedData(ex)
			require.Lenf(t, relayed, 1, "the %s tickers must be relayed once", channel)
			assert.Equalf(t, exp, relayed[0], "%s should relay a ticker for each asset its instrument trades as", channel)
			for i := range exp {
				stored, err := ticker.GetTicker(ex.Name, exp[i].Pair, exp[i].AssetType)
				require.NoErrorf(t, err, "the %s %s ticker must be stored", channel, exp[i].AssetType)
				assert.Equalf(t, &exp[i], stored, "the %s %s ticker should be stored as relayed", channel, exp[i].AssetType)
			}
		})
	}

	// A ticker that cannot be stored is not relayed
	crossed := bytes.Replace(marketDataFrame(t, "ticker.BTC_USD", 1), []byte(`"b":"80787.84"`), []byte(`"b":"80787.85"`), 1)
	err := ex.wsHandleChannelData(t.Context(), nil, crossed)
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "wsHandleChannelData should report a ticker it cannot store")
	assert.Empty(t, relayedData(ex), "a ticker that cannot be stored should not be relayed")
	stored, err := ticker.GetTicker(ex.Name, spotBTCUSD, asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, time.UnixMilli(1792892978711), stored.LastUpdated, "the stored ticker should not be replaced by one that cannot be stored")

	// Of a batch, the tickers stored are relayed and the others reported
	valid := `{"h":"81007.04","l":"79128.21","a":"80787.84","c":"-0.0002","b":"80787.84","bs":"0.05030","k":"80787.85","ks":"0.54183","i":"BTC_USD","v":"2215.1469","vv":"177736579.59","oi":"0","t":1792892978822}`
	invalid := `{"h":"81007.04","l":"79128.21","a":"80787.84","c":"-0.0002","b":"80787.85","bs":"0.05030","k":"80787.85","ks":"0.54183","i":"BTC_USD","v":"2215.1469","vv":"177736579.59","oi":"0","t":1792892978923}`
	batch := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"ticker.BTC_USD","channel":"ticker","data":[` + valid + `,` + invalid + `]}}`
	err = ex.wsHandleChannelData(t.Context(), nil, []byte(batch))
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "wsHandleChannelData should report the tickers of a batch it cannot store")
	exp := []any{[]ticker.Price{
		{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.0503, Ask: 80787.85, AskSize: 0.54183, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: btcUSDChange, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1792892978822)},
		{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.0503, Ask: 80787.85, AskSize: 0.54183, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: btcUSDChange, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.UnixMilli(1792892978822)},
	}}
	assert.Equal(t, exp, relayedData(ex), "the tickers of a batch that are stored should be relayed alone")

	// The change is taken as a percentage in decimal arithmetic, as the REST ticker takes it
	changed := bytes.Replace(marketDataFrame(t, "ticker.BTC_USD", 1), []byte(`"c":"-0.0002"`), []byte(`"c":"-0.0078"`), 1)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, changed), "wsHandleChannelData must not error for a ticker's change")
	exp = []any{[]ticker.Price{
		{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.0503, Ask: 80787.85, AskSize: 0.54183, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: -0.78, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1792892978822)},
		{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.0503, Ask: 80787.85, AskSize: 0.54183, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: -0.78, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: asset.Margin, LastUpdated: time.UnixMilli(1792892978822)},
	}}
	assert.Equal(t, exp, relayedData(ex), "a ticker's change of -0.0078 should be relayed as -0.78")
}

func TestQuotedInUSD(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		instrumentAsset
		exp bool
	}{
		{instrumentAsset{Asset: asset.Spot, Pair: spotBTCUSD}, true},
		{instrumentAsset{Asset: asset.Margin, Pair: spotETHUSD}, true},
		{instrumentAsset{Asset: asset.Spot, Pair: spotBTCUSDT}, false},
		{instrumentAsset{Asset: asset.PerpetualSwap, Pair: perpBTCUSD}, true},
		{instrumentAsset{Asset: asset.Futures, Pair: futureBTCUSD}, true},
		{instrumentAsset{Asset: asset.PerpetualSwap, Pair: currency.NewPair(currency.NewCode("BTCUSDT"), currency.NewCode("PERP"))}, false},
	} {
		t.Run(tc.Asset.String()+" "+tc.Pair.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, quotedInUSD(tc.instrumentAsset), "quotedInUSD should report whether %s %s is quoted in USD", tc.Asset, tc.Pair)
		})
	}
}

func TestWebsocketTrades(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	frame := marketDataFrame(t, "trade.BTC_USD", 0)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, frame), "wsHandleChannelData must not error without a trade feed")
	assert.Empty(t, relayedData(ex), "trades should not be relayed without a trade feed")
	undecoded := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"trade.BTC_USD","channel":"trade","data":{}}}`
	assert.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(undecoded)), "wsHandleChannelData should not decode trades without a trade feed")

	enableTradeFeed(ex)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, frame), "wsHandleChannelData must not error")
	relayed := relayedData(ex)
	require.Len(t, relayed, 1, "the trades must be relayed once")
	trades := []struct {
		id     string
		amount float64
	}{
		{"1792893186501857915", 0.04871},
		{"1792893186501857914", 0.13482},
		{"1792893186501857913", 0.03096},
	}
	exp := make([]trade.Data, 0, 2*len(trades))
	for _, tr := range trades {
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			exp = append(exp, trade.Data{TID: tr.id, Exchange: ex.Name, CurrencyPair: spotBTCUSD, AssetType: a, Side: order.Sell, Price: 80750.57, Amount: tr.amount, Timestamp: time.UnixMilli(1792893186501)})
		}
	}
	assert.Equal(t, exp, relayed[0], "the trades should be relayed for each asset their instrument trades as")

	// A trade whose side cannot be read is reported, and the others relayed
	badSide := bytes.Replace(frame, []byte(`"s":"SELL"`), []byte(`"s":"NOPE"`), 1)
	err := ex.wsHandleChannelData(t.Context(), nil, badSide)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "wsHandleChannelData should report a trade whose side cannot be read")
	relayed = relayedData(ex)
	require.Len(t, relayed, 1, "the other trades must be relayed once")
	assert.Equal(t, exp[2:], relayed[0], "the trades whose side can be read should be relayed")
}

func TestWebsocketCandles(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "candlestick.1m.BTC_USD", 0)), "wsHandleChannelData must not error")
	candles := []kline.Candle{
		{Time: time.UnixMilli(1792893060000), Open: 80776.63, High: 80786.75, Low: 80776.63, Close: 80782.7, Volume: 0.51772},
		{Time: time.UnixMilli(1792893120000), Open: 80775.53, High: 80775.53, Low: 80755.6, Close: 80755.6, Volume: 0.30336},
		{Time: time.UnixMilli(1792893180000), Open: 80750.57, High: 80750.57, Low: 80750.57, Close: 80750.57, Volume: 0.24306},
	}
	exp := []any{
		kline.Item{Exchange: ex.Name, Pair: spotBTCUSD, Asset: asset.Spot, Interval: kline.OneMin, Candles: candles},
		kline.Item{Exchange: ex.Name, Pair: spotBTCUSD, Asset: asset.Margin, Interval: kline.OneMin, Candles: candles},
	}
	relayed := relayedData(ex)
	assert.Equal(t, exp, relayed, "the candles should be relayed for each asset their instrument trades as")
	require.Len(t, relayed, 2, "the candles of each asset must be relayed")
	spotCandles, ok := relayed[0].(kline.Item)
	require.True(t, ok, "the spot candles must be relayed as a kline item")
	spotCandles.Candles[0].Open = 0
	assert.Equal(t, exp[1], relayed[1], "the margin candles should not change with the spot candles")

	unknown := bytes.Replace(marketDataFrame(t, "candlestick.1m.BTC_USD", 1), []byte(`"interval":"1m"`), []byte(`"interval":"3m"`), 1)
	assert.ErrorIs(t, ex.wsHandleChannelData(t.Context(), nil, unknown), kline.ErrUnsupportedInterval, "wsHandleChannelData should refuse candles of an interval it does not know")
	empty := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"candlestick.1m.BTC_USD","channel":"candlestick","interval":"1m","data":[]}}`
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(empty)), "wsHandleChannelData must not error without candles")
	assert.Empty(t, relayedData(ex), "no candles should be relayed without candles")
}

func TestWebsocketIndexPrice(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "index.BTCUSD-INDEX", 0)), "wsHandleChannelData must not error")
	exp := []any{&WebsocketIndexPrice{IndexName: "BTCUSD-INDEX", IndexPrice: 80784.28, Time: types.Time(time.UnixMilli(1792892978000))}}
	assert.Equal(t, exp, relayedData(ex), "the index price should be relayed")
}

func TestWebsocketMarkPrice(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "mark.BTC_USD", 0)), "wsHandleChannelData must not error for a spot instrument")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "mark.BTCUSD-PERP", 0)), "wsHandleChannelData must not error for a perpetual swap")
	exp := []any{
		&WebsocketMarkPrice{Pair: spotBTCUSD, AssetType: asset.Spot, MarkPrice: 80783.17, Time: types.Time(time.UnixMilli(1792893005000))},
		&WebsocketMarkPrice{Pair: spotBTCUSD, AssetType: asset.Margin, MarkPrice: 80783.17, Time: types.Time(time.UnixMilli(1792893005000))},
		&WebsocketMarkPrice{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, MarkPrice: 80784, Time: types.Time(time.UnixMilli(1792892978000))},
	}
	assert.Equal(t, exp, relayedData(ex), "the mark prices should be relayed for each asset their instrument trades as")
}

func TestWebsocketSettlementPrice(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "settlement.BTCUSD-261030", 0)), "wsHandleChannelData must not error for the acknowledgement")
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "settlement.BTCUSD-261030", 1)), "wsHandleChannelData must not error")
	exp := []any{&WebsocketSettlementPrice{Pair: futureBTCUSD, AssetType: asset.Futures, SettlementPrice: 80860.9, Time: types.Time(time.UnixMilli(1793347200000))}}
	assert.Equal(t, exp, relayedData(ex), "the settlement price should be relayed")
}

func TestWebsocketFundingRate(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "funding.BTCUSD-PERP", 0)), "wsHandleChannelData must not error")
	exp := []any{&WebsocketFundingRate{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, FundingRate: 0.000009839, Time: types.Time(time.UnixMilli(1792893185000))}}
	assert.Equal(t, exp, relayedData(ex), "the funding rate should be relayed")
}

func TestWebsocketEstimatedFundingRate(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, "estimatedfunding.BTCUSD-PERP", 0)), "wsHandleChannelData must not error")
	exp := []any{&WebsocketEstimatedFundingRate{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, EstimatedFundingRate: 0.000003673, Time: types.Time(time.UnixMilli(1792893179000))}}
	assert.Equal(t, exp, relayedData(ex), "the estimated funding rate should be relayed")
}

func TestWebsocketHandleChannelData(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	enableTradeFeed(ex)
	// The cases share ex in order, and the check after them covers what they relayed
	for _, tc := range []struct{ name, frame string }{
		{"an acknowledgement", `{"id":1,"method":"subscribe","code":0,"channel":"book.BTC_USD.10"}`},
		{"a null result", `{"id":-1,"method":"subscribe","code":0,"result":null}`},
		{"an instrument not enabled", string(marketDataFrame(t, "ticker.1INCH_USD", 0))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(tc.frame)), "wsHandleChannelData must not error for %s", tc.frame)
		})
	}
	assert.Empty(t, relayedData(ex), "nothing should be relayed for an acknowledgement or an instrument not enabled")

	// The cases share ex in order, before the calls below on the same exchange
	for _, tc := range []struct{ name, frame string }{
		{"a truncated message", `{"result":`},
		{"a string result", `{"result":"x"}`},
		{"an array result", `{"result":[1]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(tc.frame)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse %s", tc.frame)
		})
	}

	unhandled := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"nope.BTC_USD","channel":"nope","data":[]}}`
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(unhandled)), "wsHandleChannelData must not error for a channel it does not know")
	assert.Equal(t, []any{websocket.UnhandledMessageWarning{Message: ex.Name + websocket.UnhandledMessage + unhandled}}, relayedData(ex), "a channel wsHandleChannelData does not know should be relayed as unhandled")

	channels := []string{channelBook, channelBookUpdate, channelTicker, channelTrade, channelCandlestick, IndexPriceChannel, MarkPriceChannel, SettlementPriceChannel, FundingRateChannel, EstimatedFundingRateChannel}
	// The cases share ex in order, and the check after them covers what they relayed
	for _, channel := range channels {
		t.Run(channel, func(t *testing.T) {
			frame := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"x","channel":"` + channel + `","interval":"1m","data":{}}}`
			assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse %s data it cannot decode", channel)
			if channel == IndexPriceChannel {
				// An index is named for itself rather than an instrument
				return
			}
			frame = `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"NOPE_USD","subscription":"x","channel":"` + channel + `","interval":"1m","data":[]}}`
			assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), currency.ErrPairNotFound, "wsHandleChannelData should refuse %s data for an instrument no asset trades", channel)
		})
	}
	assert.Empty(t, relayedData(ex), "nothing should be relayed for data that cannot be decoded or routed")

	t.Run("undecodable or unknown user channels", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		// The cases share ex in order, and the check after them covers what they relayed
		for _, channel := range []string{TransactionsChannel, PositionsChannel, PositionBalanceChannel, OTCQuoteRequestsChannel, OTCQuotesChannel, OTCDealsChannel} {
			t.Run(channel, func(t *testing.T) {
				frame := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"` + channel + `","channel":"` + channel + `","data":{}}}`
				assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse %s data it cannot decode", channel)
			})
		}
		assert.Empty(t, relayedData(ex), "nothing should be relayed for data that cannot be decoded")

		unknown := `{"id":-1,"method":"subscribe","code":0,"result":{"subscription":"user.nope","channel":"user.nope","data":[]}}`
		require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(unknown)), "wsHandleChannelData must not error for a user channel it does not know")
		exp := []any{websocket.UnhandledMessageWarning{Message: ex.Name + websocket.UnhandledMessage + unknown}}
		assert.Equal(t, exp, relayedData(ex), "a user channel wsHandleChannelData does not know should be relayed as unhandled")
	})
}

func TestWebsocketMarketDataConnect(t *testing.T) {
	t.Parallel()
	t.Run("public data works while the private socket is down", func(t *testing.T) {
		t.Parallel()
		if !mockTests {
			t.Skip("separate endpoints are tested against mock servers")
		}
		venue := newMarketDataVenue(t)
		var privateCalls atomic.Int32
		private := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			privateCalls.Add(1)
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		private.Start()
		privateURL := "ws" + strings.TrimPrefix(private.URL, "http")
		ex := newTestExchange(t)
		ex.Config.API.AuthenticatedWebsocketSupport = false
		ex.Features.Subscriptions = subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}},
		}
		require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.WebsocketSpot.String(), venue.url), "the public endpoint must be configured")
		for _, endpoint := range []exchange.URL{exchange.WebsocketPrivate, exchange.WebsocketTrade} {
			require.NoError(t, ex.API.Endpoints.SetRunningURL(endpoint.String(), privateURL), "the private endpoint must be configured")
		}
		ex.Websocket = websocket.NewManager()
		require.NoError(t, ex.setupWebsocket(ex.Config), "websocket setup must succeed")
		t.Cleanup(func() {
			assert.NoError(t, ex.Websocket.Disable(), "the websocket should be disabled during cleanup")
			if err := ex.Websocket.Shutdown(); err != nil {
				assert.ErrorIs(t, err, websocket.ErrNotConnected, "only an already-disconnected socket should fail shutdown")
			}
		})
		t.Cleanup(venue.closing)
		err := ex.Websocket.Connect(t.Context())
		assert.Zero(t, privateCalls.Load(), "public-only mode should leave the private endpoint unused")
		require.NoError(t, err, "public market data must connect")
		require.Len(t, ex.Websocket.GetSubscriptions(), 1, "the ticker subscription must survive")
		err = ex.SendWebsocketRequest(t.Context(), "private/get-cancel-on-disconnect", nil, nil)
		assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "SendWebsocketRequest should report the missing authentication, not the missing trade connection")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		err = ex.SendWebsocketRequest(t.Context(), "private/get-cancel-on-disconnect", nil, nil)
		assert.ErrorIs(t, err, websocket.ErrNotConnected, "SendWebsocketRequest should report the trade connection a public-only connect left out")
		assert.Zero(t, privateCalls.Load(), "SendWebsocketRequest should not dial the private endpoint")
	})
	subs := subscription.List{
		{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Pairs: currency.Pairs{spotBTCUSD}},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{spotBTCUSD}},
		{Enabled: true, Asset: asset.Spot, Channel: subscription.CandlesChannel, Interval: kline.OneMin, Pairs: currency.Pairs{spotBTCUSD}},
		{Enabled: true, Asset: asset.Spot, Channel: MarkPriceChannel, Pairs: currency.Pairs{spotBTCUSD}},
		{Enabled: true, Asset: asset.PerpetualSwap, Channel: subscription.TickerChannel, Pairs: currency.Pairs{perpBTCUSD}},
		{Enabled: true, Asset: asset.PerpetualSwap, Channel: IndexPriceChannel, Pairs: currency.Pairs{perpBTCUSD}},
		{Enabled: true, Asset: asset.PerpetualSwap, Channel: MarkPriceChannel, Pairs: currency.Pairs{perpBTCUSD}},
		{Enabled: true, Asset: asset.PerpetualSwap, Channel: FundingRateChannel, Pairs: currency.Pairs{perpBTCUSD}},
		{Enabled: true, Asset: asset.PerpetualSwap, Channel: EstimatedFundingRateChannel, Pairs: currency.Pairs{perpBTCUSD}},
	}
	expRelayed := []any{[]ticker.Price{}, &orderbook.Depth{}, []trade.Data{}, kline.Item{}, &WebsocketMarkPrice{}, &WebsocketIndexPrice{}, &WebsocketFundingRate{}, &WebsocketEstimatedFundingRate{}}
	if mockTests {
		// The settlement channel pushes only when a future expires, which the mock does on subscribing
		subs = append(subs, &subscription.Subscription{Enabled: true, Asset: asset.Futures, Channel: SettlementPriceChannel, Pairs: currency.Pairs{futureBTCUSD}})
		expRelayed = append(expRelayed, &WebsocketSettlementPrice{})
	}
	ex, err := newMarketDataTestExchange(t, newMarketDataVenue(t), subs)
	require.NoError(t, err, "Connect must subscribe to each channel")

	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, len(subs), "each subscription must be stored")
	for _, s := range stored {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
	}

	seen := make(map[string]bool)
	require.Eventually(t, func() bool {
		for _, data := range relayedData(ex) {
			seen[typeName(data)] = true
		}
		for _, data := range expRelayed {
			if !seen[typeName(data)] {
				return false
			}
		}
		return true
	}, 30*time.Second, 10*time.Millisecond, "each channel's data must be relayed")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err := ticker.GetTicker(ex.Name, spotBTCUSD, a)
		assert.NoErrorf(t, err, "the %s ticker should be stored", a)
		// A book is relayed as it loads, ahead of the next asset's
		assert.Eventuallyf(t, func() bool {
			_, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
			return err == nil
		}, 30*time.Second, 10*time.Millisecond, "the %s book should be loaded", a)
	}
	_, err = ticker.GetTicker(ex.Name, perpBTCUSD, asset.PerpetualSwap)
	assert.NoError(t, err, "the perpetual swap ticker should be stored")
}

// typeName returns the name of a value's type
func typeName(v any) string {
	return fmt.Sprintf("%T", v)
}

func TestWebsocketDefaultSubscriptions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the default subscriptions are tested against the mock websocket, as the test configuration's future expires")
	}
	venue := newMarketDataVenue(t)
	ex, err := newMarketDataTestExchange(t, venue, defaultSubscriptions.Clone())
	require.NoError(t, err, "Connect must subscribe to the default subscriptions")
	stored := ex.Websocket.GetSubscriptions()
	require.Len(t, stored, 28, "each enabled instrument must be subscribed to each default channel once")
	for _, s := range stored {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
	}
	requests := venue.received(websocketMethodSubscribe)
	require.Len(t, requests, 2, "the books must be subscribed apart from the other channels")
	assert.JSONEq(t, `{"channels":[
		"ticker.BTC_USD","ticker.BTC_USDT","ticker.CRO_USD","ticker.ETH_USD","ticker.BTCUSD-261030","ticker.BTCUSD-PERP","ticker.ETHUSD-PERP",
		"trade.BTC_USD","trade.BTC_USDT","trade.CRO_USD","trade.ETH_USD","trade.BTCUSD-261030","trade.BTCUSD-PERP","trade.ETHUSD-PERP",
		"candlestick.1m.BTC_USD","candlestick.1m.BTC_USDT","candlestick.1m.CRO_USD","candlestick.1m.ETH_USD","candlestick.1m.BTCUSD-261030","candlestick.1m.BTCUSD-PERP","candlestick.1m.ETHUSD-PERP"
	]}`, string(requests[0].Params), "the default tickers, trades and candles should be subscribed together")
	assert.JSONEq(t, `{"channels":[
		"book.BTC_USD.50","book.BTC_USDT.50","book.CRO_USD.50","book.ETH_USD.50","book.BTCUSD-261030.50","book.BTCUSD-PERP.50","book.ETHUSD-PERP.50"
	],"book_subscription_type":"SNAPSHOT_AND_UPDATE","book_update_frequency":100}`, string(requests[1].Params), "the default books should be subscribed for deltas every 100 milliseconds")
}

func TestWebsocketSubscribe(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("subscribing is tested against the mock websocket, which can reject a subscription, refuse a request or send a malformed acknowledgement, and records the requests")
	}
	t.Run("a subscription rejected", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		venue.rejectSubscriptions("ticker.BTC_USDT", &APIError{Code: 40003, Message: "Unknown symbol"})
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD, spotBTCUSDT}}})
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "Connect must report the rejected subscription")
		assert.Equal(t, &APIError{Code: 40003, Message: "Unknown symbol"}, apiErr, "Connect should report why the subscription was rejected")
		stored := ex.Websocket.GetSubscriptions()
		require.Len(t, stored, 1, "only the accepted subscription must be stored")
		assert.Equal(t, "ticker.BTC_USD", stored[0].QualifiedChannel, "the accepted subscription should be stored")
		assert.Equal(t, subscription.SubscribedState, stored[0].State(), "the accepted subscription should be subscribed")
	})

	t.Run("a subscribe request refused", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		venue.fail(websocketMethodSubscribe, &APIError{Code: 40001, Message: "BAD_REQUEST"})
		start := time.Now()
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD, spotBTCUSDT}}})
		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr, "Connect must report the refused subscribe request")
		assert.Equal(t, &APIError{Code: 40001, Message: "BAD_REQUEST"}, apiErr, "Connect should report why the subscribe request was refused")
		assert.Less(t, time.Since(start), ex.Config.WebsocketResponseMaxLimit, "Connect should not wait for answers to a subscribe request refused as a whole")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "no subscription should be stored")
	})

	t.Run("an acknowledgement naming no channel", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		venue.answerSubscriptions("ticker.BTC_USD", []json.RawMessage{json.RawMessage(`{"id":1,"method":"subscribe","code":0}`)})
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}})
		require.ErrorIs(t, err, errMalformedWebsocketMessage, "Connect must report an acknowledgement naming no channel")
		assert.Empty(t, ex.Websocket.GetSubscriptions(), "no subscription should be stored")
	})

	t.Run("book update frequencies", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, subscription.List{
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Levels: 10, Interval: kline.TenMilliseconds, Pairs: currency.Pairs{spotBTCUSD}},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: currency.Pairs{spotETHUSD}},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: bookSnapshotInterval, Pairs: currency.Pairs{spotBTCUSDT}},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds, Pairs: currency.Pairs{spotCROUSD}},
			{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}},
		})
		require.NoError(t, err, "Connect must not error")
		requests := venue.received(websocketMethodSubscribe)
		require.Len(t, requests, 4, "a subscribe request must be sent for each book update frequency and one for the other channels")
		for i, exp := range []string{
			`{"channels":["book.BTC_USD.10"],"book_subscription_type":"SNAPSHOT_AND_UPDATE","book_update_frequency":10}`,
			`{"channels":["book.ETH_USD.50","book.CRO_USD.50"],"book_subscription_type":"SNAPSHOT_AND_UPDATE","book_update_frequency":100}`,
			`{"channels":["book.BTC_USDT.50"],"book_subscription_type":"SNAPSHOT","book_update_frequency":500}`,
			`{"channels":["ticker.BTC_USD"]}`,
		} {
			assert.JSONEqf(t, exp, string(requests[i].Params), "subscribe request %d should name its channels and their book updates", i)
		}

		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		invalid := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotCROUSD}, Interval: kline.OneMin, QualifiedChannel: "book.CRO_USD.10"}
		valid := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotETHUSD}, QualifiedChannel: "ticker.ETH_USD"}
		err = ex.wsSubscribe(t.Context(), conn, subscription.List{invalid, valid})
		assert.ErrorIs(t, err, subscription.ErrInvalidInterval, "wsSubscribe should refuse a book subscription to updates Crypto.com does not send")
		assert.Equal(t, subscription.SubscribedState, valid.State(), "wsSubscribe should subscribe the other channels")
		requests = venue.received(websocketMethodSubscribe)
		require.Len(t, requests, 5, "wsSubscribe must send one subscribe request")
		assert.JSONEq(t, `{"channels":["ticker.ETH_USD"]}`, string(requests[4].Params), "wsSubscribe should leave out the refused book subscription")
	})

	t.Run("one channel is acknowledged before the batch times out", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		accepted := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, QualifiedChannel: "ticker.BTC_USD"}
		missing := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSDT}, QualifiedChannel: "ticker.BTC_USDT"}
		venue.answerSubscriptions(missing.QualifiedChannel, []json.RawMessage{marketDataFrame(t, missing.QualifiedChannel, 1)})
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		err = ex.wsSubscribe(ctx, conn, subscription.List{accepted, missing})
		require.ErrorIs(t, err, context.DeadlineExceeded, "the unanswered channel must be reported")
		assert.Same(t, accepted, ex.Websocket.GetSubscription(accepted), "the acknowledged channel should remain tracked")
		assert.Equal(t, subscription.SubscribedState, accepted.State(), "the acknowledged channel should be subscribed")
		assert.Nil(t, ex.Websocket.GetSubscription(missing), "the unanswered channel should stay unconfirmed")
	})
}

// marketDataPush returns a market data channel's push of data for an instrument
func marketDataPush(channel, instrument, data string) *websocketChannelPush {
	return &websocketChannelPush{InstrumentName: instrument, Subscription: channel + "." + instrument, Channel: channel, Data: json.RawMessage(data)}
}

// oneInchUSD is the pair of 1INCH_USD, a spot instrument the test configuration makes available without enabling it
var oneInchUSD = currency.NewPair(currency.NewCode("1INCH"), currency.USD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})

func TestWebsocketCandleInterval(t *testing.T) { //nolint:tparallel // the cases run in order before the later calls of the same function
	t.Parallel()
	// The cases run in order, the refusals among them, so each also checks that a call keeps nothing from the last
	for _, tc := range []struct {
		timeframe string
		exp       kline.Interval
		err       error
	}{
		{"1m", kline.OneMin, nil},
		{"5m", kline.FiveMin, nil},
		{"15m", kline.FifteenMin, nil},
		{"30m", kline.ThirtyMin, nil},
		{"1h", kline.OneHour, nil},
		{"2h", kline.TwoHour, nil},
		{"4h", kline.FourHour, nil},
		{"12h", kline.TwelveHour, nil},
		{"1D", kline.OneDay, nil},
		{"7D", kline.SevenDay, nil},
		{"14D", kline.TwoWeek, nil},
		{"1M", kline.OneMonth, nil},
		{"1d", 0, kline.ErrUnsupportedInterval},
		{"3m", 0, kline.ErrUnsupportedInterval},
		{"1m ", 0, kline.ErrUnsupportedInterval},
		{"", 0, kline.ErrUnsupportedInterval},
		{"1h", kline.OneHour, nil},
	} {
		t.Run(subtestName(tc.timeframe), func(t *testing.T) {
			got, err := websocketCandleInterval(tc.timeframe)
			require.ErrorIsf(t, err, tc.err, "websocketCandleInterval must return the expected error for %q", tc.timeframe)
			assert.Equalf(t, tc.exp, got, "websocketCandleInterval should give the interval %q names", tc.timeframe)
		})
	}
}

func TestWebsocketHandleBook(t *testing.T) { //nolint:tparallel // the cases change one exchange's books in order
	t.Parallel()
	ex := newTestExchange(t)
	// bookState is what the ETH_USD books hold once a snapshot loads them
	type bookState struct {
		bids, asks                   orderbook.Levels
		updated, published, updateID int64
	}
	recorded := recordedSnapshotBook(ex.Name, asset.Spot)
	snapshot := &bookState{bids: recorded.Bids, asks: recorded.Asks, updated: recorded.LastUpdated.UnixMilli(), published: recorded.LastPushed.UnixMilli(), updateID: recorded.LastUpdateID}
	replaced := &bookState{bids: orderbook.Levels{{Price: 2399, Amount: 2.5, OrderCount: 3}}, asks: orderbook.Levels{{Price: 2400, Amount: 1.5, OrderCount: 2}}, updated: 1792893039990, published: 1792893040000, updateID: 390670048300000}
	untimed := &bookState{bids: orderbook.Levels{{Price: 2398, Amount: 2.5, OrderCount: 3}}, asks: orderbook.Levels{{Price: 2401, Amount: 1.5, OrderCount: 2}}, updated: 1792893041000, published: 1792893041000, updateID: 390670048300001}
	last := &bookState{bids: orderbook.Levels{{Price: 2396, Amount: 2, OrderCount: 2}}, asks: orderbook.Levels{{Price: 2403, Amount: 2, OrderCount: 2}}, updated: 1792893042990, published: 1792893043000, updateID: 390670048300003}
	var state *bookState
	// The cases change the books ex holds in order
	for _, tc := range []struct {
		name string
		push *websocketChannelPush
		// loaded is what the books hold after, as before when nil, and relayed how many books are relayed
		loaded  *bookState
		relayed int
		err     error
	}{
		{name: "a snapshot loading the book of each asset its instrument trades as", push: channelPush(t, marketDataFrame(t, recordedSnapshotBookChannel, 2)), loaded: snapshot, relayed: 2},
		{
			name:    "a snapshot replacing the books",
			push:    marketDataPush(channelBook, "ETH_USD", `[{"asks":[["2400.00","1.5","2"]],"bids":[["2399.00","2.5","3"]],"t":1792893040000,"tt":1792893039990,"u":390670048300000}]`),
			loaded:  replaced,
			relayed: 2,
		},
		{
			name:    "a snapshot without an update time, updated when it was published",
			push:    marketDataPush(channelBook, "ETH_USD", `[{"asks":[["2401.00","1.5","2"]],"bids":[["2398.00","2.5","3"]],"t":1792893041000,"u":390670048300001}]`),
			loaded:  untimed,
			relayed: 2,
		},
		{
			name:    "several snapshots in one push, the last kept",
			push:    marketDataPush(channelBook, "ETH_USD", `[{"asks":[["2402.00","1","1"]],"bids":[["2397.00","1","1"]],"t":1792893042000,"tt":1792893041990,"u":390670048300002},{"asks":[["2403.00","2","2"]],"bids":[["2396.00","2","2"]],"t":1792893043000,"tt":1792893042990,"u":390670048300003}]`),
			loaded:  last,
			relayed: 4,
		},
		{name: "a snapshot of an instrument not enabled", push: marketDataPush(channelBook, "1INCH_USD", `[{"asks":[["0.09","10","1"]],"bids":[["0.08","10","1"]],"t":1792909196186,"u":1}]`)},
		{name: "a snapshot of an instrument no asset trades", push: marketDataPush(channelBook, "NOPE_USD", `[]`), err: currency.ErrPairNotFound},
		{name: "a snapshot that cannot be decoded", push: marketDataPush(channelBook, "ETH_USD", `{}`), err: errMalformedWebsocketMessage},
		{name: "a snapshot without a time", push: marketDataPush(channelBook, "ETH_USD", `[{"asks":[["2404.00","1","1"]],"bids":[["2395.00","1","1"]],"u":390670048300004}]`), err: orderbook.ErrLastUpdatedNotSet},
		{name: "a snapshot of a level without a price", push: marketDataPush(channelBook, "ETH_USD", `[{"asks":[["2404.00","1","1"]],"bids":[["0","1","1"]],"t":1792893044000,"u":390670048300004}]`), err: orderbook.ErrPriceZero},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, ex.wsHandleBook(t.Context(), tc.push), tc.err, "wsHandleBook must return the expected error")
			if tc.loaded != nil {
				state = tc.loaded
			}
			for _, a := range []asset.Item{asset.Spot, asset.Margin} {
				book, err := ex.Websocket.Orderbook.GetOrderbook(spotETHUSD, a)
				require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
				assert.NotZerof(t, book.InsertedAt, "the %s book should record when it was loaded", a)
				book.InsertedAt = time.Time{}
				exp := &orderbook.Book{
					Bids:              state.bids,
					Asks:              state.asks,
					Exchange:          ex.Name,
					Pair:              spotETHUSD,
					Asset:             a,
					LastUpdated:       time.UnixMilli(state.updated),
					LastPushed:        time.UnixMilli(state.published),
					LastUpdateID:      state.updateID,
					ValidateOrderbook: true,
				}
				assert.Equalf(t, exp, book, "the %s book should hold the last snapshot loaded", a)
			}
			relayed := relayedData(ex)
			assert.Len(t, relayed, tc.relayed, "wsHandleBook should relay each book it loads")
			for _, data := range relayed {
				assert.IsType(t, &orderbook.Depth{}, data, "wsHandleBook should relay the books it loads")
			}
		})
	}
	_, err := ex.Websocket.Orderbook.GetOrderbook(oneInchUSD, asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "wsHandleBook should load no book of an instrument not enabled")

	t.Run("snapshots that cannot be relayed", func(t *testing.T) {
		full := fillDataHandler(t, ex)
		err := ex.wsHandleBook(t.Context(), marketDataPush(channelBook, "ETH_USD", `[{"asks":[["2405.00","1","1"]],"bids":[["2394.00","1","1"]],"t":1792893045000,"u":390670048300005}]`))
		assert.ErrorIs(t, err, full, "wsHandleBook should report the books it cannot relay")
		requireBooksAt(t, ex, spotETHUSD, 390670048300005)
	})
}

func TestWebsocketHandleBookUpdate(t *testing.T) {
	t.Parallel()
	recordedDelta := func(t *testing.T, frame int) string {
		t.Helper()
		data, _, _, err := jsonparser.Get(marketDataFrame(t, recordedBookChannel, frame), "result", "data", "[0]")
		require.NoErrorf(t, err, "recorded frame %d must hold a delta", frame)
		return string(data)
	}
	deltas := func(instrument string, data ...string) *websocketChannelPush {
		return marketDataPush(channelBookUpdate, instrument, "["+strings.Join(data, ",")+"]")
	}
	// withRecordedSnapshot returns an exchange whose spot and margin BTC_USD books hold the snapshot the recorded deltas
	// follow
	withRecordedSnapshot := func(t *testing.T) *Exchange {
		t.Helper()
		ex := newTestExchange(t)
		require.NoError(t, ex.wsHandleBook(t.Context(), channelPush(t, marketDataFrame(t, recordedBookChannel, 1))), "wsHandleBook must load the recorded snapshot")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID)
		relayedData(ex)
		return ex
	}
	// loadBook loads an asset's BTC_USD book at an update
	loadBook := func(t *testing.T, ex *Exchange, a asset.Item, updateID int64) {
		t.Helper()
		book := &orderbook.Book{Exchange: ex.Name, Pair: spotBTCUSD, Asset: a, Bids: orderbook.Levels{{Price: 80775.53, Amount: 1}}, Asks: orderbook.Levels{{Price: 80775.54, Amount: 1}}, LastUpdated: time.Now(), LastUpdateID: updateID}
		require.NoErrorf(t, ex.Websocket.Orderbook.LoadSnapshot(t.Context(), book), "LoadSnapshot must load the %s book", a)
	}
	// booksInvalid checks an exchange's BTC_USD books of the assets given have been invalidated
	booksInvalid := func(t *testing.T, ex *Exchange, assets ...asset.Item) {
		t.Helper()
		for _, a := range assets {
			_, err := ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, a)
			assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "wsHandleBookUpdate should invalidate the %s book", a)
		}
	}
	firstDelta, secondDelta, thirdDelta := recordedDelta(t, 2), recordedDelta(t, 3), recordedDelta(t, 4)
	const firstDeltaUpdateID = 382097463890240
	unchangedDelta := `{"update":{"asks":[],"bids":[]},"t":1792893127150,"tt":1792893127140,"u":382097463058305,"pu":382097463058304,"cs":0}`
	noUpdateDelta := `{"t":1792893127150,"tt":1792893127140,"u":382097463058305,"pu":382097463058304,"cs":0}`
	breakingDelta := `{"update":{"asks":[],"bids":[["0","1","1"]]},"t":1792893127150,"tt":1792893127140,"u":382097463058305,"pu":382097463058304,"cs":0}`

	for _, tc := range []struct {
		name   string
		pushes []*websocketChannelPush
	}{
		{"deltas following the books' last update applied in turn", []*websocketChannelPush{deltas("BTC_USD", firstDelta), deltas("BTC_USD", secondDelta), deltas("BTC_USD", thirdDelta)}},
		{"deltas in one push applied in turn", []*websocketChannelPush{deltas("BTC_USD", firstDelta, secondDelta, thirdDelta)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := withRecordedSnapshot(t)
			conn := newScriptedConnection()
			for i, push := range tc.pushes {
				require.NoErrorf(t, ex.wsHandleBookUpdate(t.Context(), conn, push), "wsHandleBookUpdate must apply push %d", i)
			}
			for _, a := range []asset.Item{asset.Spot, asset.Margin} {
				book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
				require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
				book.InsertedAt = time.Time{}
				assert.Equalf(t, recordedDeltaBook(ex.Name, a), book, "wsHandleBookUpdate should apply each delta to the %s book", a)
			}
			assert.Len(t, relayedData(ex), 6, "wsHandleBookUpdate should relay each asset's book as each delta updates it")
			assert.Zero(t, conn.recorded().lookups, "wsHandleBookUpdate should not resubscribe a book channel whose deltas follow on")
		})
	}

	t.Run("a delta without levels moving the books on", func(t *testing.T) {
		t.Parallel()
		ex := withRecordedSnapshot(t)
		conn := newScriptedConnection()
		before, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, asset.Spot)
		require.NoError(t, err, "GetOrderbook must not error")
		require.NoError(t, ex.wsHandleBookUpdate(t.Context(), conn, deltas("BTC_USD", unchangedDelta)), "wsHandleBookUpdate must not error for a delta without levels")
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
			require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
			exp := *before
			exp.Asset, exp.LastUpdated, exp.LastPushed, exp.LastUpdateID = a, time.UnixMilli(1792893127140), time.UnixMilli(1792893127150), 382097463058305
			exp.InsertedAt, book.InsertedAt = time.Time{}, time.Time{}
			assert.Equalf(t, &exp, book, "wsHandleBookUpdate should move the %s book on to the delta's update, leaving its levels", a)
		}
		assert.Zero(t, conn.recorded().lookups, "wsHandleBookUpdate should not resubscribe a book channel whose deltas follow on")
	})

	for _, tc := range []struct {
		name string
		push *websocketChannelPush
		err  error
	}{
		{"a delta of an instrument not enabled, which is dropped", deltas("1INCH_USD", firstDelta), nil},
		{"a delta of an instrument no asset trades", deltas("NOPE_USD", firstDelta), currency.ErrPairNotFound},
		{"deltas that cannot be decoded", marketDataPush(channelBookUpdate, "BTC_USD", `{}`), errMalformedWebsocketMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			conn := newScriptedConnection()
			assert.ErrorIs(t, ex.wsHandleBookUpdate(t.Context(), conn, tc.push), tc.err, "wsHandleBookUpdate should return the expected error")
			assert.Empty(t, relayedData(ex), "wsHandleBookUpdate should update no book")
			assert.Zero(t, conn.recorded().lookups, "wsHandleBookUpdate should not resubscribe the book channel")
		})
	}

	for _, tc := range []struct {
		name   string
		push   *websocketChannelPush
		update int64
	}{
		{"a delta without its update", deltas("BTC_USD", noUpdateDelta), recordedBookSnapshotUpdateID},
		{"a delta without its update after one applied", deltas("BTC_USD", firstDelta, noUpdateDelta), firstDeltaUpdateID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := withRecordedSnapshot(t)
			conn := newScriptedConnection()
			assert.ErrorIs(t, ex.wsHandleBookUpdate(t.Context(), conn, tc.push), errBookDeltaHasNoUpdate, "wsHandleBookUpdate should refuse a delta without its update")
			requireBooksAt(t, ex, spotBTCUSD, tc.update)
			assert.Zero(t, conn.recorded().lookups, "wsHandleBookUpdate should not resubscribe the book channel")
		})
	}

	t.Run("a delta while the books await a snapshot, asking for one again", func(t *testing.T) {
		t.Parallel()
		ex := withRecordedSnapshot(t)
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			require.NoErrorf(t, ex.Websocket.Orderbook.InvalidateOrderbook(spotBTCUSD, a), "InvalidateOrderbook must not error for %s", a)
		}
		conn := newScriptedConnection()
		require.NoError(t, ex.wsHandleBookUpdate(t.Context(), conn, deltas("BTC_USD", firstDelta)), "wsHandleBookUpdate must drop a delta while the books await a snapshot")
		booksInvalid(t, ex, asset.Spot, asset.Margin)
		assert.Equal(t, 1, conn.recorded().lookups, "wsHandleBookUpdate should ask for a snapshot again")
	})

	t.Run("a delta before the books are loaded, asking for a snapshot", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		conn := newScriptedConnection()
		require.NoError(t, ex.wsHandleBookUpdate(t.Context(), conn, deltas("BTC_USD", firstDelta)), "wsHandleBookUpdate must drop a delta before the books are loaded")
		_, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "wsHandleBookUpdate should load no book")
		assert.Equal(t, 1, conn.recorded().lookups, "wsHandleBookUpdate should ask for a snapshot")
	})

	for _, tc := range []struct {
		name string
		// books returns an exchange holding the books the delta finds, and invalidated are those it invalidates
		books       func(*testing.T) *Exchange
		push        *websocketChannelPush
		invalidated []asset.Item
		err         error
	}{
		{
			name:        "a delta following an update the books did not take",
			books:       withRecordedSnapshot,
			push:        deltas("BTC_USD", secondDelta),
			invalidated: []asset.Item{asset.Spot, asset.Margin},
			err:         errBookOutOfSync,
		},
		{
			name: "a book missing from an asset",
			books: func(t *testing.T) *Exchange {
				t.Helper()
				ex := newTestExchange(t)
				loadBook(t, ex, asset.Spot, recordedBookSnapshotUpdateID)
				return ex
			},
			push:        deltas("BTC_USD", firstDelta),
			invalidated: []asset.Item{asset.Spot},
			err:         errBookOutOfSync,
		},
		{
			name: "books at different updates",
			books: func(t *testing.T) *Exchange {
				t.Helper()
				ex := newTestExchange(t)
				loadBook(t, ex, asset.Spot, recordedBookSnapshotUpdateID)
				loadBook(t, ex, asset.Margin, recordedBookSnapshotUpdateID+1)
				return ex
			},
			push:        deltas("BTC_USD", firstDelta),
			invalidated: []asset.Item{asset.Spot, asset.Margin},
			err:         errBookOutOfSync,
		},
		{
			name:        "a delta that breaks a book",
			books:       withRecordedSnapshot,
			push:        deltas("BTC_USD", breakingDelta),
			invalidated: []asset.Item{asset.Spot, asset.Margin},
			err:         orderbook.ErrOrderbookInvalid,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := tc.books(t)
			conn := newScriptedConnection()
			assert.ErrorIs(t, ex.wsHandleBookUpdate(t.Context(), conn, tc.push), tc.err, "wsHandleBookUpdate should report books it can no longer update")
			booksInvalid(t, ex, tc.invalidated...)
			assert.Equal(t, 1, conn.recorded().lookups, "wsHandleBookUpdate should resubscribe the book channel for a new snapshot")
		})
	}

	t.Run("deltas that cannot be relayed reported, every book updated", func(t *testing.T) {
		t.Parallel()
		ex := withRecordedSnapshot(t)
		conn := newScriptedConnection()
		full := fillDataHandler(t, ex)
		assert.ErrorIs(t, ex.wsHandleBookUpdate(t.Context(), conn, deltas("BTC_USD", firstDelta)), full, "wsHandleBookUpdate should report the books it cannot relay")
		requireBooksAt(t, ex, spotBTCUSD, firstDeltaUpdateID)
		assert.Zero(t, conn.recorded().lookups, "wsHandleBookUpdate should not resubscribe a book channel whose books it cannot relay")
	})
}

func TestResynchroniseBooks(t *testing.T) {
	t.Parallel()
	t.Run("a new snapshot over a connection", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		inserted := requireBooksLoaded(t, ex)
		// The mock holds back the unsubscription, and with it the new snapshot, until the books are checked
		checked := make(chan struct{})
		release := sync.OnceFunc(func() { close(checked) })
		t.Cleanup(release)
		venue.beforeAnswering(websocketMethodUnsubscribe, func() { <-checked })
		assets, err := ex.enabledInstrumentAssets("BTC_USD")
		require.NoError(t, err, "enabledInstrumentAssets must not error")

		ex.resynchroniseBooks(t.Context(), conn, recordedBookChannel, assets)
		if mockTests {
			for _, a := range assets {
				_, err := ex.Websocket.Orderbook.LastUpdateID(a.Pair, a.Asset)
				assert.ErrorIsf(t, err, orderbook.ErrOrderbookInvalid, "resynchroniseBooks should invalidate the %s book", a.Asset)
			}
		}
		release()
		requireBooksReloaded(t, ex, inserted)
		if mockTests {
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
			assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "resynchroniseBooks should unsubscribe the book channel once")
			assert.Len(t, venue.received(websocketMethodSubscribe), 2, "resynchroniseBooks should subscribe the book channel again once")
		}
	})

	t.Run("the books named invalidated and a snapshot asked for", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		require.NoError(t, ex.wsHandleBook(t.Context(), channelPush(t, marketDataFrame(t, recordedBookChannel, 1))), "wsHandleBook must load the recorded snapshot")
		conn := newScriptedConnection()
		ex.resynchroniseBooks(t.Context(), conn, recordedBookChannel, []instrumentAsset{{Asset: asset.Spot, Pair: spotBTCUSD}})
		_, err := ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "resynchroniseBooks should invalidate the books named")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookSnapshotUpdateID, asset.Margin)
		assert.Equal(t, scriptedRecord{lookups: 1}, conn.recorded(), "resynchroniseBooks should ask for a new snapshot once")
	})

	t.Run("books never loaded", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		conn := newScriptedConnection()
		ex.resynchroniseBooks(t.Context(), conn, recordedBookChannel, []instrumentAsset{{Asset: asset.Spot, Pair: spotBTCUSD}, {Asset: asset.Margin, Pair: spotBTCUSD}})
		_, err := ex.Websocket.Orderbook.LastUpdateID(spotBTCUSD, asset.Spot)
		assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "resynchroniseBooks should load no book")
		assert.Equal(t, scriptedRecord{lookups: 1}, conn.recorded(), "resynchroniseBooks should ask for a snapshot of books never loaded")
	})
}

func TestResubscribeBook(t *testing.T) {
	t.Parallel()
	key := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}}
	t.Run("a subscribed book channel resubscribed for a new snapshot over a connection", func(t *testing.T) {
		t.Parallel()
		venue := newMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		inserted := requireBooksLoaded(t, ex)
		s := conn.Subscriptions().Get(key)
		require.NotNil(t, s, "the book channel must be stored against its connection")

		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		requireBooksReloaded(t, ex, inserted)
		assert.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 10*time.Second, 10*time.Millisecond, "resubscribeBook should leave the book channel subscribed")
		if mockTests {
			requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
			assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "resubscribeBook should unsubscribe the book channel once")
			assert.Len(t, venue.received(websocketMethodSubscribe), 2, "resubscribeBook should subscribe the book channel again once")
		}
	})

	t.Run("a book channel not stored against its connection left alone", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		conn := newScriptedConnection()
		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		assert.Equal(t, scriptedRecord{lookups: 1}, conn.recorded(), "resubscribeBook should look the book channel up and send nothing")
	})

	for _, tc := range []struct {
		name  string
		state subscription.State
	}{
		{"inactive", subscription.InactiveState},
		{"subscribing", subscription.SubscribingState},
		{"resubscribing", subscription.ResubscribingState},
		{"unsubscribing", subscription.UnsubscribingState},
		{"unsubscribed", subscription.UnsubscribedState},
	} {
		t.Run("a book channel "+tc.name+" left alone", func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			conn := newScriptedConnection()
			s := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, Levels: 10, QualifiedChannel: recordedBookChannel}
			if tc.state != subscription.InactiveState {
				require.NoError(t, s.SetState(tc.state), "SetState must not error")
			}
			require.NoError(t, conn.subscriptions.Add(s), "Add must not error")
			ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
			assert.Never(t, func() bool { return s.State() != tc.state }, 100*time.Millisecond, 10*time.Millisecond, "resubscribeBook should leave a book channel it is not subscribed to as it is")
			assert.Equal(t, scriptedRecord{lookups: 1}, conn.recorded(), "resubscribeBook should look the book channel up and send nothing")
		})
	}

	t.Run("a book channel whose resubscription fails before unsubscribing left subscribed", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t)
		conn := newScriptedConnection()
		s := &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{spotBTCUSD}, Levels: 10, QualifiedChannel: recordedBookChannel}
		require.NoError(t, s.SetState(subscription.SubscribedState), "SetState must not error")
		require.NoError(t, conn.subscriptions.Add(s), "Add must not error")
		// The websocket's manager refuses to unsubscribe over a connection it does not manage
		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		require.Eventually(t, func() bool { return conn.recorded().lookups == 2 }, 10*time.Second, 10*time.Millisecond, "the resubscription must look up the connection's subscriptions")
		assert.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 10*time.Second, 10*time.Millisecond, "resubscribeBook should leave the book channel subscribed")
		assert.Empty(t, conn.recorded().requests, "resubscribeBook should send nothing over a connection it cannot unsubscribe from")
	})

	t.Run("a failed unsubscription leaving the book channel subscribed", func(t *testing.T) {
		t.Parallel()
		venue := mockMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		s := conn.Subscriptions().Get(key)
		require.NotNil(t, s, "the book channel must be stored against its connection")
		venue.fail(websocketMethodUnsubscribe, &APIError{Code: 50001, Message: "ERR_INTERNAL"})

		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		require.Eventually(t, func() bool { return len(venue.received(websocketMethodUnsubscribe)) == 1 }, 10*time.Second, 10*time.Millisecond, "the book channel must be unsubscribed")
		assert.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 10*time.Second, 10*time.Millisecond, "resubscribeBook should leave a book channel it failed to unsubscribe subscribed")
		assert.Never(t, func() bool { return len(venue.received(websocketMethodSubscribe)) > 1 }, 200*time.Millisecond, 10*time.Millisecond, "resubscribeBook should not subscribe a book channel it failed to unsubscribe")
	})

	t.Run("a refused subscription retried", func(t *testing.T) {
		t.Parallel()
		venue := mockMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), func(ex *Exchange) {
			ex.bookResubscribeDelay = 10 * time.Millisecond
		})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		// The mock clears the refusal as it answers the resubscription, so that the retry is accepted whatever the timing
		venue.rejectSubscriptions(recordedBookChannel, &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"})
		venue.beforeAnswering(websocketMethodSubscribe, func() { venue.rejectSubscriptions(recordedBookChannel, nil) })

		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		require.Eventually(t, func() bool {
			s := conn.Subscriptions().Get(key)
			return s != nil && s.State() == subscription.SubscribedState && len(venue.received(websocketMethodSubscribe)) == 3
		}, 10*time.Second, 10*time.Millisecond, "the refused book channel must be subscribed again by a retry")
		requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
		assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "resubscribeBook should unsubscribe the book channel once")
	})

	// A flush removing the book channel can unsubscribe it while its resubscription waits for its unsubscription's
	// answer, and the resubscription then finds the channel unsubscribed already
	t.Run("a book channel unsubscribed while it resubscribes left unsubscribed", func(t *testing.T) {
		t.Parallel()
		venue := mockMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription(), func(ex *Exchange) {
			ex.bookResubscribeDelay = 10 * time.Millisecond
		})
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		s := conn.Subscriptions().Get(key)
		require.NotNil(t, s, "the book channel must be stored against its connection")
		venue.beforeAnswering(websocketMethodUnsubscribe, func() { _ = s.SetState(subscription.UnsubscribedState) })

		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		require.Eventually(t, func() bool { return len(venue.received(websocketMethodUnsubscribe)) == 1 }, 10*time.Second, 10*time.Millisecond, "the book channel must be unsubscribed")
		assert.Never(t, func() bool { return len(venue.received(websocketMethodSubscribe)) > 1 }, 300*time.Millisecond, 10*time.Millisecond, "resubscribeBook should not subscribe a book channel unsubscribed while it resubscribed")
		assert.Equal(t, subscription.UnsubscribedState, s.State(), "resubscribeBook should leave the book channel unsubscribed")
		assert.Nil(t, ex.Websocket.GetSubscription(key), "resubscribeBook should leave the book channel removed")
	})

	t.Run("a resubscription started twice sent once", func(t *testing.T) {
		t.Parallel()
		venue := mockMarketDataVenue(t)
		ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
		require.NoError(t, err, "Connect must not error")
		conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
		require.NoError(t, err, "GetConnection must not error")
		requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
		// The mock holds back the unsubscription until both resubscriptions have started
		started := make(chan struct{})
		release := sync.OnceFunc(func() { close(started) })
		t.Cleanup(release)
		venue.beforeAnswering(websocketMethodUnsubscribe, func() { <-started })

		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		ex.resubscribeBook(t.Context(), conn, recordedBookChannel)
		release()
		requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
		assert.Never(t, func() bool { return len(venue.received(websocketMethodUnsubscribe)) > 1 }, 200*time.Millisecond, 10*time.Millisecond, "resubscribeBook should resubscribe a book channel once while it resubscribes")
		assert.Len(t, venue.received(websocketMethodSubscribe), 2, "resubscribeBook should subscribe the book channel again once")
	})
}

func TestIsConfiguredMarketDataChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		subs    subscription.List
		channel string
		exp     bool
	}{
		{"a channel the configuration names", recordedBookChannelSubscription(), recordedBookChannel, true},
		{"a channel the configuration does not name", recordedBookChannelSubscription(), "book.ETH_USD.10", false},
		{"a channel of margin's, named as spot's", subscription.List{{Enabled: true, Asset: asset.Margin, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD}}}, "ticker.BTC_USD", true},
		{"a user channel, which no market data connection carries", subscription.List{{Enabled: true, Channel: subscription.MyOrdersChannel, Authenticated: true}}, "user.order", false},
		{"a channel of a configuration that cannot be told", append(recordedBookChannelSubscription(), &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: "nope"}), recordedBookChannel, false},
		{"no configuration", nil, recordedBookChannel, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestExchange(t)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			ex.Features.Subscriptions = tc.subs
			assert.Equal(t, tc.exp, ex.isConfiguredMarketDataChannel(tc.channel), "isConfiguredMarketDataChannel should report whether the configured market data subscriptions name the channel")
		})
	}
}

func TestWebsocketHandleTicker(t *testing.T) { //nolint:tparallel // the cases share one exchange, whose tickers they store, and its data handler in order
	t.Parallel()
	ex := newTestExchange(t)
	btcUSD := func(a asset.Item) ticker.Price {
		return ticker.Price{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.05028, Ask: 80787.85, AskSize: 0.5319, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: -0.02, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: a, LastUpdated: time.UnixMilli(1792892978711)}
	}
	laterBTCUSD := func(a asset.Item) ticker.Price {
		return ticker.Price{Last: 80787.84, High: 81007.04, Low: 79128.21, Bid: 80787.84, BidSize: 0.0503, Ask: 80787.85, AskSize: 0.54183, BaseVolume: 2215.1469, QuoteVolume: 177736579.59, PercentChange24Hour: -0.02, Pair: spotBTCUSD, ExchangeName: ex.Name, AssetType: a, LastUpdated: time.UnixMilli(1792892978822)}
	}
	firstSpotBTCUSD := btcUSD(asset.Spot)
	later := `{"h":"81007.04","l":"79128.21","a":"80787.84","c":"-0.0002","b":"80787.84","bs":"0.05030","k":"80787.85","ks":"0.54183","i":"BTC_USD","v":"2215.1469","vv":"177736579.59","oi":"0","t":1792892978822}`
	crossed := `{"h":"81007.04","l":"79128.21","a":"80787.84","c":"-0.0002","b":"80787.85","bs":"0.05030","k":"80787.85","ks":"0.54183","i":"BTC_USD","v":"2215.1469","vv":"177736579.59","oi":"0","t":1792892978923}`
	// The cases share ex, whose tickers they store, and drain its data handler in order
	for _, tc := range []struct {
		name    string
		push    *websocketChannelPush
		relayed []ticker.Price
		// kept is a ticker stored before that the case leaves stored
		kept *ticker.Price
		err  error
	}{
		{name: "a spot instrument's ticker stored and relayed for spot and margin", push: channelPush(t, marketDataFrame(t, "ticker.BTC_USD", 0)), relayed: []ticker.Price{btcUSD(asset.Spot), btcUSD(asset.Margin)}},
		{
			name: "a ticker quoted in a currency other than its volume value's", push: channelPush(t, marketDataFrame(t, "ticker.BTC_USDT", 0)),
			relayed: []ticker.Price{{Last: 80956.49, High: 81099.29, Low: 79232.39, Bid: 80951.66, BidSize: 0.01872, Ask: 80951.67, AskSize: 0.20126, BaseVolume: 1071.3429, Pair: spotBTCUSDT, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(1792894311364)}},
		},
		{
			name: "a perpetual swap's ticker", push: channelPush(t, marketDataFrame(t, "ticker.BTCUSD-PERP", 0)),
			relayed: []ticker.Price{{Last: 80750.3, High: 81004.7, Low: 79117.7, Bid: 80750.3, BidSize: 0.0087, Ask: 80750.4, AskSize: 0.0331, BaseVolume: 2512.7932, QuoteVolume: 201559679.09, PercentChange24Hour: -0.13, OpenInterest: 6179.573, Pair: perpBTCUSD, ExchangeName: ex.Name, AssetType: asset.PerpetualSwap, LastUpdated: time.UnixMilli(1792893199728)}},
		},
		{name: "a ticker of an instrument not enabled", push: channelPush(t, marketDataFrame(t, "ticker.1INCH_USD", 0))},
		{name: "a ticker of an instrument no asset trades", push: marketDataPush(channelTicker, "NOPE_USD", `[]`), err: currency.ErrPairNotFound},
		{name: "tickers that cannot be decoded", push: marketDataPush(channelTicker, "BTC_USD", `{}`), err: errMalformedWebsocketMessage},
		{name: "no tickers", push: marketDataPush(channelTicker, "BTC_USD", `[]`)},
		{name: "a ticker that cannot be stored, which is not relayed", push: marketDataPush(channelTicker, "BTC_USD", `[`+crossed+`]`), kept: &firstSpotBTCUSD, err: ticker.ErrBidEqualsAsk},
		{name: "the tickers of a batch stored relayed, the others reported", push: marketDataPush(channelTicker, "BTC_USD", `[`+later+`,`+crossed+`]`), relayed: []ticker.Price{laterBTCUSD(asset.Spot), laterBTCUSD(asset.Margin)}, err: ticker.ErrBidEqualsAsk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, ex.wsHandleTicker(t.Context(), tc.push), tc.err, "wsHandleTicker must return the expected error")
			var exp []any
			if len(tc.relayed) != 0 {
				exp = []any{tc.relayed}
			}
			assert.Equal(t, exp, relayedData(ex), "wsHandleTicker should relay the tickers it stores, for each asset their instrument trades as")
			for i := range tc.relayed {
				stored, err := ticker.GetTicker(ex.Name, tc.relayed[i].Pair, tc.relayed[i].AssetType)
				require.NoErrorf(t, err, "GetTicker must not error for the %s ticker relayed", tc.relayed[i].AssetType)
				assert.Equalf(t, &tc.relayed[i], stored, "wsHandleTicker should store the %s ticker it relays", tc.relayed[i].AssetType)
			}
			if tc.kept != nil {
				stored, err := ticker.GetTicker(ex.Name, tc.kept.Pair, tc.kept.AssetType)
				require.NoError(t, err, "GetTicker must not error for the ticker stored before")
				assert.Equal(t, tc.kept, stored, "wsHandleTicker should keep the ticker stored before one it cannot store")
			}
		})
	}
	_, err := ticker.GetTicker(ex.Name, oneInchUSD, asset.Spot)
	assert.Error(t, err, "wsHandleTicker should store no ticker of an instrument not enabled")

	t.Run("tickers that cannot be relayed", func(t *testing.T) {
		full := fillDataHandler(t, ex)
		assert.ErrorIs(t, ex.wsHandleTicker(t.Context(), channelPush(t, marketDataFrame(t, "ticker.BTC_USD", 0))), full, "wsHandleTicker should report the tickers it cannot relay")
		stored, err := ticker.GetTicker(ex.Name, spotBTCUSD, asset.Spot)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equal(t, &firstSpotBTCUSD, stored, "wsHandleTicker should store the tickers it cannot relay")
	})
}

func TestWebsocketHandleTrades(t *testing.T) { //nolint:tparallel // the cases share one exchange, its trade feed and its data handler in order
	t.Parallel()
	ex := newTestExchange(t)
	frame := marketDataFrame(t, "trade.BTC_USD", 0)
	recordedTrades := make([]trade.Data, 0, 6)
	for _, recorded := range []struct {
		id     string
		amount float64
	}{{"1792893186501857915", 0.04871}, {"1792893186501857914", 0.13482}, {"1792893186501857913", 0.03096}} {
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			recordedTrades = append(recordedTrades, trade.Data{TID: recorded.id, Exchange: ex.Name, CurrencyPair: spotBTCUSD, AssetType: a, Side: order.Sell, Price: 80750.57, Amount: recorded.amount, Timestamp: time.UnixMilli(1792893186501)})
		}
	}
	unreadSide := bytes.Replace(frame, []byte(`"s":"SELL"`), []byte(`"s":"NOPE"`), 1)
	// The cases share ex, setting its trade feed and trade saving in turn, and drain its data handler in order
	for _, tc := range []struct {
		name            string
		tradeFeed, save bool
		push            *websocketChannelPush
		relayed         []any
		err             error
	}{
		{name: "trades neither relayed nor saved, which are not decoded", push: marketDataPush(channelTrade, "BTC_USD", `{}`)},
		{name: "trades saved without a trade feed", save: true, push: channelPush(t, frame)},
		{name: "trades relayed through the trade feed for each asset their instrument trades as", tradeFeed: true, push: channelPush(t, frame), relayed: []any{recordedTrades}},
		{name: "trades relayed and saved", tradeFeed: true, save: true, push: channelPush(t, frame), relayed: []any{recordedTrades}},
		{name: "a trade whose side cannot be read reported, the others relayed", tradeFeed: true, push: channelPush(t, unreadSide), relayed: []any{recordedTrades[2:]}, err: order.ErrSideIsInvalid},
		{name: "trades of an instrument not enabled", tradeFeed: true, push: marketDataPush(channelTrade, "1INCH_USD", `[{"d":"1","t":1792909196186,"p":"0.08901","q":"10","s":"BUY","i":"1INCH_USD","m":"1"}]`)},
		{name: "trades of an instrument no asset trades", tradeFeed: true, push: marketDataPush(channelTrade, "NOPE_USD", `[]`), err: currency.ErrPairNotFound},
		{name: "trades that cannot be decoded", tradeFeed: true, push: marketDataPush(channelTrade, "BTC_USD", `{}`), err: errMalformedWebsocketMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setTradeFeed(ex, tc.tradeFeed)
			ex.SetSaveTradeDataStatus(tc.save)
			require.ErrorIs(t, ex.wsHandleTrades(tc.push), tc.err, "wsHandleTrades must return the expected error")
			relayed := relayedData(ex)
			if tc.save {
				// Saving trades gives them IDs, which it writes into the trades the trade feed has relayed already
				for _, data := range relayed {
					if trades, ok := data.([]trade.Data); ok {
						for i := range trades {
							trades[i].ID = uuid.UUID{}
						}
					}
				}
			}
			assert.Equal(t, tc.relayed, relayed, "wsHandleTrades should relay the trades it can read through the trade feed")
		})
	}

	t.Run("trades that cannot be relayed", func(t *testing.T) {
		setTradeFeed(ex, true)
		ex.SetSaveTradeDataStatus(false)
		full := fillDataHandler(t, ex)
		assert.ErrorIs(t, ex.wsHandleTrades(channelPush(t, frame)), full, "wsHandleTrades should report the trades it cannot relay")
	})
}

func TestWebsocketHandleCandles(t *testing.T) { //nolint:tparallel // the cases share one exchange and its data handler in order
	t.Parallel()
	ex := newTestExchange(t)
	candles := []kline.Candle{
		{Time: time.UnixMilli(1792893060000), Open: 80776.63, High: 80786.75, Low: 80776.63, Close: 80782.7, Volume: 0.51772},
		{Time: time.UnixMilli(1792893120000), Open: 80775.53, High: 80775.53, Low: 80755.6, Close: 80755.6, Volume: 0.30336},
		{Time: time.UnixMilli(1792893180000), Open: 80750.57, High: 80750.57, Low: 80750.57, Close: 80750.57, Volume: 0.24306},
	}
	candlesPush := func(instrument, timeframe, data string) *websocketChannelPush {
		push := marketDataPush(channelCandlestick, instrument, data)
		push.Interval = timeframe
		return push
	}
	hourCandle := `[{"o":"80750.5","h":"80800.0","l":"80700.0","c":"80790.0","v":"12.5","t":1792890000000,"ut":1792893599000}]`
	// The cases share ex and drain its data handler in order
	for _, tc := range []struct {
		name    string
		push    *websocketChannelPush
		relayed []any
		err     error
	}{
		{
			name: "candles relayed for each asset their instrument trades as", push: channelPush(t, marketDataFrame(t, "candlestick.1m.BTC_USD", 0)),
			relayed: []any{
				kline.Item{Exchange: ex.Name, Pair: spotBTCUSD, Asset: asset.Spot, Interval: kline.OneMin, Candles: candles},
				kline.Item{Exchange: ex.Name, Pair: spotBTCUSD, Asset: asset.Margin, Interval: kline.OneMin, Candles: candles},
			},
		},
		{
			name: "a perpetual swap's candles", push: candlesPush("BTCUSD-PERP", "1h", hourCandle),
			relayed: []any{kline.Item{Exchange: ex.Name, Pair: perpBTCUSD, Asset: asset.PerpetualSwap, Interval: kline.OneHour, Candles: []kline.Candle{{Time: time.UnixMilli(1792890000000), Open: 80750.5, High: 80800, Low: 80700, Close: 80790, Volume: 12.5}}}},
		},
		{name: "candles of a timeframe Crypto.com does not name, refused before they are decoded", push: candlesPush("BTC_USD", "3m", `{}`), err: kline.ErrUnsupportedInterval},
		{name: "candles that cannot be decoded", push: candlesPush("BTC_USD", "1m", `{}`), err: errMalformedWebsocketMessage},
		{name: "no candles", push: candlesPush("BTC_USD", "1m", `[]`)},
		{name: "no candles of an instrument no asset trades", push: candlesPush("NOPE_USD", "1m", `[]`), err: currency.ErrPairNotFound},
		{name: "candles of an instrument not enabled", push: candlesPush("1INCH_USD", "1h", hourCandle)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, ex.wsHandleCandles(t.Context(), tc.push), tc.err, "wsHandleCandles must return the expected error")
			assert.Equal(t, tc.relayed, relayedData(ex), "wsHandleCandles should relay the candles for each asset their instrument trades as")
		})
	}

	t.Run("each asset's candles its own", func(t *testing.T) {
		require.NoError(t, ex.wsHandleCandles(t.Context(), channelPush(t, marketDataFrame(t, "candlestick.1m.BTC_USD", 0))), "wsHandleCandles must not error")
		relayed := relayedData(ex)
		require.Len(t, relayed, 2, "wsHandleCandles must relay the candles of spot and margin")
		spotCandles, ok := relayed[0].(kline.Item)
		require.True(t, ok, "the spot candles must be relayed as a kline item")
		marginCandles, ok := relayed[1].(kline.Item)
		require.True(t, ok, "the margin candles must be relayed as a kline item")
		spotCandles.Candles[0].Open = 0
		assert.Equal(t, candles, marginCandles.Candles, "wsHandleCandles should give each asset candles of its own, which changing another's leaves")
	})

	t.Run("candles that cannot be relayed", func(t *testing.T) {
		full := fillDataHandler(t, ex)
		assert.ErrorIs(t, ex.wsHandleCandles(t.Context(), channelPush(t, marketDataFrame(t, "candlestick.1m.BTC_USD", 0))), full, "wsHandleCandles should report the candles it cannot relay")
	})
}

func TestWebsocketHandleIndexPrice(t *testing.T) { //nolint:tparallel // the cases share one exchange and its data handler in order
	t.Parallel()
	ex := newTestExchange(t)
	// The cases share ex and drain its data handler in order
	for _, tc := range []struct {
		name    string
		push    *websocketChannelPush
		relayed []any
		err     error
	}{
		{
			name: "an index's price relayed with the index's name", push: channelPush(t, marketDataFrame(t, "index.BTCUSD-INDEX", 0)),
			relayed: []any{&WebsocketIndexPrice{IndexName: "BTCUSD-INDEX", IndexPrice: 80784.28, Time: types.Time(time.UnixMilli(1792892978000))}},
		},
		{
			name: "an index's prices relayed in turn", push: marketDataPush(IndexPriceChannel, "ETHUSD-INDEX", `[{"v":"2359.1","t":1792892978000},{"v":"2359.2","t":1792892979000}]`),
			relayed: []any{
				&WebsocketIndexPrice{IndexName: "ETHUSD-INDEX", IndexPrice: 2359.1, Time: types.Time(time.UnixMilli(1792892978000))},
				&WebsocketIndexPrice{IndexName: "ETHUSD-INDEX", IndexPrice: 2359.2, Time: types.Time(time.UnixMilli(1792892979000))},
			},
		},
		{name: "no prices", push: marketDataPush(IndexPriceChannel, "BTCUSD-INDEX", `[]`)},
		{name: "prices that cannot be decoded", push: marketDataPush(IndexPriceChannel, "BTCUSD-INDEX", `{}`), err: errMalformedWebsocketMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, ex.wsHandleIndexPrice(t.Context(), tc.push), tc.err, "wsHandleIndexPrice must return the expected error")
			assert.Equal(t, tc.relayed, relayedData(ex), "wsHandleIndexPrice should relay each of the index's prices with its name")
		})
	}

	t.Run("prices that cannot be relayed", func(t *testing.T) {
		full := fillDataHandler(t, ex)
		assert.ErrorIs(t, ex.wsHandleIndexPrice(t.Context(), channelPush(t, marketDataFrame(t, "index.BTCUSD-INDEX", 0))), full, "wsHandleIndexPrice should report the prices it cannot relay")
	})
}

func TestRelayInstrumentValues(t *testing.T) { //nolint:tparallel // the cases share one exchange and its data handler in order
	t.Parallel()
	ex := newTestExchange(t)
	markPrices := func(ctx context.Context, push *websocketChannelPush) error {
		return relayInstrumentValues[WebsocketMarkPrice](ctx, ex, push)
	}
	settlementPrices := func(ctx context.Context, push *websocketChannelPush) error {
		return relayInstrumentValues[WebsocketSettlementPrice](ctx, ex, push)
	}
	fundingRates := func(ctx context.Context, push *websocketChannelPush) error {
		return relayInstrumentValues[WebsocketFundingRate](ctx, ex, push)
	}
	estimatedFundingRates := func(ctx context.Context, push *websocketChannelPush) error {
		return relayInstrumentValues[WebsocketEstimatedFundingRate](ctx, ex, push)
	}
	markPrice := func(a asset.Item, price float64, at int64) *WebsocketMarkPrice {
		return &WebsocketMarkPrice{Pair: spotBTCUSD, AssetType: a, MarkPrice: types.Number(price), Time: types.Time(time.UnixMilli(at))}
	}
	// The cases share ex and drain its data handler in order
	for _, tc := range []struct {
		name    string
		relay   func(context.Context, *websocketChannelPush) error
		push    *websocketChannelPush
		relayed []any
		err     error
	}{
		{
			name: "a spot instrument's mark price, relayed for spot and margin", relay: markPrices, push: channelPush(t, marketDataFrame(t, "mark.BTC_USD", 0)),
			relayed: []any{markPrice(asset.Spot, 80783.17, 1792893005000), markPrice(asset.Margin, 80783.17, 1792893005000)},
		},
		{
			name: "a perpetual swap's mark price", relay: markPrices, push: channelPush(t, marketDataFrame(t, "mark.BTCUSD-PERP", 0)),
			relayed: []any{&WebsocketMarkPrice{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, MarkPrice: 80784, Time: types.Time(time.UnixMilli(1792892978000))}},
		},
		{
			name: "a future's settlement price", relay: settlementPrices, push: channelPush(t, marketDataFrame(t, "settlement.BTCUSD-261030", 1)),
			relayed: []any{&WebsocketSettlementPrice{Pair: futureBTCUSD, AssetType: asset.Futures, SettlementPrice: 80860.9, Time: types.Time(time.UnixMilli(1793347200000))}},
		},
		{
			name: "a perpetual swap's funding rate", relay: fundingRates, push: channelPush(t, marketDataFrame(t, "funding.BTCUSD-PERP", 0)),
			relayed: []any{&WebsocketFundingRate{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, FundingRate: 0.000009839, Time: types.Time(time.UnixMilli(1792893185000))}},
		},
		{
			name: "a perpetual swap's estimated funding rate", relay: estimatedFundingRates, push: channelPush(t, marketDataFrame(t, "estimatedfunding.BTCUSD-PERP", 0)),
			relayed: []any{&WebsocketEstimatedFundingRate{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, EstimatedFundingRate: 0.000003673, Time: types.Time(time.UnixMilli(1792893179000))}},
		},
		{
			name: "several values, each relayed for each asset in turn", relay: markPrices, push: marketDataPush(MarkPriceChannel, "BTC_USD", `[{"v":"80783.17","t":1792893005000},{"v":"80783.15","t":1792893006000}]`),
			relayed: []any{markPrice(asset.Spot, 80783.17, 1792893005000), markPrice(asset.Margin, 80783.17, 1792893005000), markPrice(asset.Spot, 80783.15, 1792893006000), markPrice(asset.Margin, 80783.15, 1792893006000)},
		},
		{name: "values of an instrument not enabled", relay: markPrices, push: marketDataPush(MarkPriceChannel, "1INCH_USD", `[{"v":"0.08901","t":1792909196186}]`)},
		{name: "values of an instrument no asset trades", relay: markPrices, push: marketDataPush(MarkPriceChannel, "NOPE_USD", `[]`), err: currency.ErrPairNotFound},
		{name: "values that cannot be decoded", relay: markPrices, push: marketDataPush(MarkPriceChannel, "BTC_USD", `{}`), err: errMalformedWebsocketMessage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorIs(t, tc.relay(t.Context(), tc.push), tc.err, "relayInstrumentValues must return the expected error")
			assert.Equal(t, tc.relayed, relayedData(ex), "relayInstrumentValues should relay each value for each asset its instrument trades as")
		})
	}

	t.Run("each asset's value its own", func(t *testing.T) {
		require.NoError(t, markPrices(t.Context(), channelPush(t, marketDataFrame(t, "mark.BTC_USD", 0))), "relayInstrumentValues must not error")
		relayed := relayedData(ex)
		require.Len(t, relayed, 2, "relayInstrumentValues must relay the mark price of spot and margin")
		spotPrice, ok := relayed[0].(*WebsocketMarkPrice)
		require.True(t, ok, "the spot mark price must be relayed as a mark price")
		spotPrice.MarkPrice = 0
		assert.Equal(t, markPrice(asset.Margin, 80783.17, 1792893005000), relayed[1], "relayInstrumentValues should give each asset a value of its own, which changing another's leaves")
	})

	t.Run("values that cannot be relayed", func(t *testing.T) {
		full := fillDataHandler(t, ex)
		assert.ErrorIs(t, markPrices(t.Context(), channelPush(t, marketDataFrame(t, "mark.BTC_USD", 0))), full, "relayInstrumentValues should report the values it cannot relay")
	})
}

func TestWebsocketMarkPriceSetInstrumentAsset(t *testing.T) {
	t.Parallel()
	price := &WebsocketMarkPrice{MarkPrice: 80783.17, Time: types.Time(time.UnixMilli(1792893005000))}
	price.setInstrumentAsset(instrumentAsset{Asset: asset.Spot, Pair: spotBTCUSD})
	assert.Equal(t, &WebsocketMarkPrice{Pair: spotBTCUSD, AssetType: asset.Spot, MarkPrice: 80783.17, Time: types.Time(time.UnixMilli(1792893005000))}, price, "setInstrumentAsset should set the pair and asset, leaving the price")
	price.setInstrumentAsset(instrumentAsset{Asset: asset.PerpetualSwap, Pair: perpBTCUSD})
	assert.Equal(t, &WebsocketMarkPrice{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, MarkPrice: 80783.17, Time: types.Time(time.UnixMilli(1792893005000))}, price, "setInstrumentAsset should replace the pair and asset set before")
}

func TestWebsocketSettlementPriceSetInstrumentAsset(t *testing.T) {
	t.Parallel()
	price := &WebsocketSettlementPrice{SettlementPrice: 80860.9, Time: types.Time(time.UnixMilli(1793347200000))}
	price.setInstrumentAsset(instrumentAsset{Asset: asset.Futures, Pair: futureBTCUSD})
	assert.Equal(t, &WebsocketSettlementPrice{Pair: futureBTCUSD, AssetType: asset.Futures, SettlementPrice: 80860.9, Time: types.Time(time.UnixMilli(1793347200000))}, price, "setInstrumentAsset should set the pair and asset, leaving the price")
	price.setInstrumentAsset(instrumentAsset{Asset: asset.Spot, Pair: spotBTCUSD})
	assert.Equal(t, &WebsocketSettlementPrice{Pair: spotBTCUSD, AssetType: asset.Spot, SettlementPrice: 80860.9, Time: types.Time(time.UnixMilli(1793347200000))}, price, "setInstrumentAsset should replace the pair and asset set before")
}

func TestWebsocketFundingRateSetInstrumentAsset(t *testing.T) {
	t.Parallel()
	rate := &WebsocketFundingRate{FundingRate: 0.000009839, Time: types.Time(time.UnixMilli(1792893185000))}
	rate.setInstrumentAsset(instrumentAsset{Asset: asset.PerpetualSwap, Pair: perpBTCUSD})
	assert.Equal(t, &WebsocketFundingRate{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, FundingRate: 0.000009839, Time: types.Time(time.UnixMilli(1792893185000))}, rate, "setInstrumentAsset should set the pair and asset, leaving the rate")
	rate.setInstrumentAsset(instrumentAsset{Asset: asset.PerpetualSwap, Pair: perpETHUSD})
	assert.Equal(t, &WebsocketFundingRate{Pair: perpETHUSD, AssetType: asset.PerpetualSwap, FundingRate: 0.000009839, Time: types.Time(time.UnixMilli(1792893185000))}, rate, "setInstrumentAsset should replace the pair set before")
}

func TestWebsocketEstimatedFundingRateSetInstrumentAsset(t *testing.T) {
	t.Parallel()
	rate := &WebsocketEstimatedFundingRate{EstimatedFundingRate: 0.000003673, Time: types.Time(time.UnixMilli(1792893179000))}
	rate.setInstrumentAsset(instrumentAsset{Asset: asset.PerpetualSwap, Pair: perpBTCUSD})
	assert.Equal(t, &WebsocketEstimatedFundingRate{Pair: perpBTCUSD, AssetType: asset.PerpetualSwap, EstimatedFundingRate: 0.000003673, Time: types.Time(time.UnixMilli(1792893179000))}, rate, "setInstrumentAsset should set the pair and asset, leaving the rate")
	rate.setInstrumentAsset(instrumentAsset{Asset: asset.Futures, Pair: futureBTCUSD})
	assert.Equal(t, &WebsocketEstimatedFundingRate{Pair: futureBTCUSD, AssetType: asset.Futures, EstimatedFundingRate: 0.000003673, Time: types.Time(time.UnixMilli(1792893179000))}, rate, "setInstrumentAsset should replace the pair and asset set before")
}
