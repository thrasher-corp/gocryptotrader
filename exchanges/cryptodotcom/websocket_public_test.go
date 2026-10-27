package cryptodotcom

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
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
	ex.SetTradeFeedStatus(true)
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
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

func TestWebsocketCandleTimeframe(t *testing.T) {
	t.Parallel()
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
		got, err := websocketCandleTimeframe(interval)
		require.NoErrorf(t, err, "websocketCandleTimeframe must not error for %s", interval)
		assert.Equalf(t, timeframe, got, "websocketCandleTimeframe should name %s by its timeframe", interval)
		gotInterval, err := websocketCandleInterval(timeframe)
		require.NoErrorf(t, err, "websocketCandleInterval must not error for %s", timeframe)
		assert.Equalf(t, interval, gotInterval, "websocketCandleInterval should give the interval of %s", timeframe)
	}
	_, err := websocketCandleTimeframe(kline.ThreeMin)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "websocketCandleTimeframe should refuse an interval the candlestick channel does not serve")
	_, err = websocketCandleInterval("1d")
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "websocketCandleInterval should refuse a timeframe Crypto.com does not name")
}

func TestInstrumentAssets(t *testing.T) {
	t.Parallel()
	_, err := e.instrumentAssets("")
	require.ErrorIs(t, err, errInstrumentNameRequired, "instrumentAssets must require an instrument name")
	for _, name := range []string{"NOPE_USD", "BTC-USD", "btc_usd", "BTCUSD_PERP", "BTCUSD-INDEX"} {
		_, err := e.instrumentAssets(name)
		assert.ErrorIsf(t, err, currency.ErrPairNotFound, "instrumentAssets should find no asset trading %s", name)
	}
	oneInchUSD := currency.NewPair(currency.NewCode("1INCH"), currency.USD).Format(currency.PairFormat{Uppercase: true, Delimiter: currency.UnderscoreDelimiter})
	futureBTCUSD261127 := currency.NewPairWithDelimiter("BTCUSD", "261127", currency.DashDelimiter)
	for name, exp := range map[string][]instrumentAsset{
		"BTC_USD":       {{Asset: asset.Spot, Pair: spotBTCUSD}, {Asset: asset.Margin, Pair: spotBTCUSD}},
		"BTC_USDT":      {{Asset: asset.Spot, Pair: spotBTCUSDT}, {Asset: asset.Margin, Pair: spotBTCUSDT}},
		"1INCH_USD":     {{Asset: asset.Spot, Pair: oneInchUSD}},
		"BTCUSD-PERP":   {{Asset: asset.PerpetualSwap, Pair: perpBTCUSD}},
		"BTCUSD-261127": {{Asset: asset.Futures, Pair: futureBTCUSD261127}},
	} {
		got, err := e.instrumentAssets(name)
		require.NoErrorf(t, err, "instrumentAssets must not error for %s", name)
		assert.Equalf(t, exp, got, "instrumentAssets should find each asset %s trades as among the available pairs", name)
	}
}

func TestEnabledInstrumentAssets(t *testing.T) {
	t.Parallel()
	_, err := e.enabledInstrumentAssets("NOPE_USD")
	require.ErrorIs(t, err, currency.ErrPairNotFound, "enabledInstrumentAssets must refuse an instrument no asset trades")
	for name, exp := range map[string][]instrumentAsset{
		"BTC_USD":       {{Asset: asset.Spot, Pair: spotBTCUSD}, {Asset: asset.Margin, Pair: spotBTCUSD}},
		"BTC_USDT":      {{Asset: asset.Spot, Pair: spotBTCUSDT}},
		"1INCH_USD":     {},
		"BTCUSD-261030": {{Asset: asset.Futures, Pair: futureBTCUSD}},
	} {
		got, err := e.enabledInstrumentAssets(name)
		require.NoErrorf(t, err, "enabledInstrumentAssets must not error for %s", name)
		assert.Equalf(t, exp, got, "enabledInstrumentAssets should find each asset whose enabled pairs hold %s", name)
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

func TestIntegerUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for data, exp := range map[string]Integer{
		`542048017824`:   542048017824,
		`"542048017824"`: 542048017824,
		`-1`:             -1,
		// 2^53 + 1, which a float cannot hold
		`9007199254740993`:   9007199254740993,
		`"9007199254740993"`: 9007199254740993,
		// A JSON string's escapes are decoded first
		`"\u002d\u0031"`: -1,
	} {
		var i Integer
		require.NoErrorf(t, json.Unmarshal([]byte(data), &i), "Unmarshal must not error for %s", data)
		assert.Equalf(t, exp, i, "Unmarshal should decode %s", data)
	}
	i := Integer(7)
	require.NoError(t, json.Unmarshal([]byte(`null`), &i), "Unmarshal must not error for null")
	assert.Equal(t, Integer(7), i, "Unmarshal should leave an integer unchanged for null")
	for _, data := range []string{`"x"`, `1.5`, `"1.5"`, `""`, `true`} {
		assert.ErrorIsf(t, json.Unmarshal([]byte(data), &i), errInvalidInteger, "Unmarshal should refuse %s", data)
	}
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
		exp := &orderbook.Book{
			Bids:              orderbook.Levels{{Price: 2359.74, Amount: 6.7928, OrderCount: 7}, {Price: 2359.73, Amount: 1.4923, OrderCount: 1}, {Price: 2359.69, Amount: 0.2465, OrderCount: 1}, {Price: 2359.61, Amount: 0.0042, OrderCount: 1}, {Price: 2359.6, Amount: 0.8118, OrderCount: 1}, {Price: 2359.59, Amount: 3.2354, OrderCount: 3}, {Price: 2359.56, Amount: 1.3625, OrderCount: 2}, {Price: 2359.54, Amount: 2.6086, OrderCount: 2}, {Price: 2359.49, Amount: 0.7426, OrderCount: 1}, {Price: 2359.48, Amount: 0.4303, OrderCount: 1}},
			Asks:              orderbook.Levels{{Price: 2359.75, Amount: 3.5136, OrderCount: 7}, {Price: 2359.83, Amount: 1.6505, OrderCount: 1}, {Price: 2359.91, Amount: 0.2428, OrderCount: 1}, {Price: 2359.94, Amount: 1.0188, OrderCount: 2}, {Price: 2359.97, Amount: 0.2891, OrderCount: 1}, {Price: 2359.99, Amount: 0.4147, OrderCount: 1}, {Price: 2360, Amount: 0.086, OrderCount: 1}, {Price: 2360.02, Amount: 0.2889, OrderCount: 1}, {Price: 2360.03, Amount: 0.8505, OrderCount: 4}, {Price: 2360.04, Amount: 0.0626, OrderCount: 1}},
			Exchange:          ex.Name,
			Pair:              spotETHUSD,
			Asset:             a,
			LastUpdated:       time.UnixMilli(1792893036530),
			LastPushed:        time.UnixMilli(1792893036570),
			LastUpdateID:      390670048243712,
			ValidateOrderbook: true,
		}
		assert.Equalf(t, exp, book, "the %s book should be the last snapshot", a)
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
	exp := &orderbook.Book{
		Bids:              orderbook.Levels{{Price: 80775.53, Amount: 0.32012, OrderCount: 12}, {Price: 80775.52, Amount: 0.0028, OrderCount: 1}, {Price: 80775, Amount: 0.003, OrderCount: 1}, {Price: 80774.47, Amount: 0.00495, OrderCount: 1}, {Price: 80774.18, Amount: 0.0033, OrderCount: 1}, {Price: 80774.17, Amount: 0.00018, OrderCount: 1}, {Price: 80773.45, Amount: 0.0035, OrderCount: 1}, {Price: 80773.23, Amount: 0.02144, OrderCount: 1}, {Price: 80773.22, Amount: 0.06191, OrderCount: 1}, {Price: 80772.86, Amount: 0.0099, OrderCount: 1}},
		Asks:              orderbook.Levels{{Price: 80775.54, Amount: 0.82815, OrderCount: 11}, {Price: 80776.11, Amount: 0.03095, OrderCount: 1}, {Price: 80779.95, Amount: 0.04378, OrderCount: 1}, {Price: 80780.58, Amount: 0.14625, OrderCount: 1}, {Price: 80780.6, Amount: 0.0025, OrderCount: 1}, {Price: 80781.32, Amount: 0.0028, OrderCount: 1}, {Price: 80781.33, Amount: 0.0028, OrderCount: 1}, {Price: 80782.21, Amount: 0.00537, OrderCount: 1}, {Price: 80782.71, Amount: 0.08568, OrderCount: 3}, {Price: 80783.44, Amount: 0.1184, OrderCount: 1}},
		Exchange:          ex.Name,
		Pair:              spotBTCUSD,
		LastUpdated:       time.UnixMilli(1792893127261),
		LastPushed:        time.UnixMilli(1792893127306),
		LastUpdateID:      recordedBookUpdateID,
		ValidateOrderbook: true,
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		book, err := ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
		require.NoErrorf(t, err, "GetOrderbook must not error for %s", a)
		book.InsertedAt = time.Time{}
		exp.Asset = a
		assert.Equalf(t, exp, book, "the %s book should be the snapshot with each delta applied", a)
	}
	assert.Len(t, relayedData(ex), 8, "the snapshot and each delta should relay the book of each asset its instrument trades as")

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

func TestWebsocketBookResubscribesOutOfSequence(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can send deltas out of sequence")
	}
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
}

func TestWebsocketBookResubscribesOutOfStep(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can send deltas out of sequence")
	}
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
}

func TestWebsocketBookResubscribesBrokenBook(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can send deltas that break a book")
	}
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
}

func TestWebsocketBookRetriesResubscription(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can fail an unsubscription")
	}
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
}

func TestWebsocketBookRetriesRefusedSubscription(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can refuse a subscription")
	}
	venue := newMarketDataVenue(t)
	ex, err := newMarketDataTestExchange(t, venue, recordedBookChannelSubscription())
	require.NoError(t, err, "Connect must not error")
	conn, err := ex.Websocket.GetConnection(websocketMarketDataConnection)
	require.NoError(t, err, "GetConnection must not error")
	requireBooksAt(t, ex, spotBTCUSD, recordedBookUpdateID)
	key := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: recordedBookChannel}}
	s := conn.Subscriptions().Get(key)
	require.NotNil(t, s, "the book channel must be stored against its connection")

	// The channel is unsubscribed for a new snapshot, then refused when subscribed again, leaving no delta to retry it
	venue.rejectSubscriptions(recordedBookChannel, &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"})
	err = ex.wsHandleData(t.Context(), conn, marketDataFrame(t, recordedBookChannel, 3))
	require.ErrorIs(t, err, errBookOutOfSync, "wsHandleData must report a delta out of sequence")
	require.Eventually(t, func() bool { return len(venue.received(websocketMethodSubscribe)) == 2 }, 10*time.Second, 10*time.Millisecond, "the book channel must be subscribed again")
	venue.rejectSubscriptions(recordedBookChannel, nil)
	requireBooksAt(t, ex, spotBTCUSD, resubscribedBookUpdateID)
	require.Eventually(t, func() bool {
		stored := conn.Subscriptions().Get(key)
		return stored != nil && stored.State() == subscription.SubscribedState
	}, 10*time.Second, 10*time.Millisecond, "the retried book channel must be stored against its connection again, subscribed")
	assert.Len(t, venue.received(websocketMethodSubscribe), 3, "the refused book channel should be subscribed once more")
	assert.Len(t, venue.received(websocketMethodUnsubscribe), 1, "the refused book channel should not be unsubscribed again")

	// TestWebsocketBookRetryStops checks how soon the retry stops on a synctest bubble's clock, which a stall of the test
	// cannot move
	ex.retryBookSubscription(t.Context(), conn, s)
	assert.Len(t, venue.received(websocketMethodSubscribe), 3, "retryBookSubscription should not subscribe a book channel held again")
}

func TestWebsocketBookRetryLeavesOutUnconfiguredChannel(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can refuse a subscription")
	}
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
}

// bookRetryConnection stands in for a connection to the mock websocket in the synctest bubbles timing
// retryBookSubscription, as the reader of a connection to the mock blocks on its socket, which would keep a bubble's
// clock from moving; it counts how often it is asked whether it is connected
type bookRetryConnection struct {
	websocket.Connection
	connected     bool
	connectChecks uint64
}

// IsConnected reports whether the connection is connected, counting each check
func (c *bookRetryConnection) IsConnected() bool {
	c.connectChecks++
	return c.connected
}

func TestWebsocketBookRetryStops(t *testing.T) {
	t.Parallel()
	const delay = 50 * time.Millisecond
	for _, tc := range []struct {
		name          string
		done          bool
		connected     bool
		held          bool
		elapsed       time.Duration
		connectChecks uint64
	}{
		{name: "context done", done: true, connected: true},
		{name: "connection lost", elapsed: delay, connectChecks: 1},
		{name: "book channel held again", connected: true, held: true, elapsed: delay, connectChecks: 1},
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
				var subscribes uint64
				ex.Websocket.Subscriber = func(subscription.List) error {
					subscribes++
					return &APIError{Code: 42901, Message: "TOO_MANY_REQUESTS"}
				}
				s := &subscription.Subscription{QualifiedChannel: recordedBookChannel}
				require.True(t, ex.isConfiguredMarketDataChannel(s.QualifiedChannel), "the book channel must be configured, so that only the stop under test can end the retry")
				if tc.held {
					require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(nil, s), "AddSuccessfulSubscriptions must not error")
				}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				if tc.done {
					cancel()
				}
				conn := &bookRetryConnection{connected: tc.connected}
				start := time.Now()
				ex.retryBookSubscription(ctx, conn, s)
				assert.Equal(t, tc.elapsed, time.Since(start), "retryBookSubscription should stop after waiting as long as expected")
				assert.Equal(t, tc.connectChecks, conn.connectChecks, "retryBookSubscription should check its connection as often as expected")
				assert.Zero(t, subscribes, "retryBookSubscription should not subscribe the book channel")
			})
		})
	}

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

func TestWebsocketBookGivesUpRefusedSubscription(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("book resynchronisation is tested against the mock websocket, which can refuse a subscription")
	}
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
}

func TestQualifiedChannelKey(t *testing.T) {
	t.Parallel()
	k := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "ticker.BTC_USD"}}
	assert.True(t, k.Match(&subscription.ExactKey{Subscription: &subscription.Subscription{Channel: subscription.TickerChannel, QualifiedChannel: "ticker.BTC_USD"}}), "Match should match a subscription with the key's qualified channel")
	assert.False(t, k.Match(&subscription.ExactKey{Subscription: &subscription.Subscription{QualifiedChannel: "ticker.BTC_USDT"}}), "Match should not match a subscription with another qualified channel")
	assert.False(t, k.Match(&subscription.ExactKey{}), "Match should not match a key without a subscription")
	assert.Equal(t, k.Subscription, k.GetSubscription(), "GetSubscription should return the key's subscription")
	assert.Equal(t, "ticker.BTC_USD", k.String(), "String should give the qualified channel")
}

func TestWebsocketTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	btcUSDChange, btcUSDTChange, perpChange := -0.02, 0.0, -0.13
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
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, marketDataFrame(t, channel, 0)), "wsHandleChannelData must not error for %s", channel)
		relayed := relayedData(ex)
		require.Lenf(t, relayed, 1, "the %s tickers must be relayed once", channel)
		assert.Equalf(t, exp, relayed[0], "%s should relay a ticker for each asset its instrument trades as", channel)
		for i := range exp {
			stored, err := ticker.GetTicker(ex.Name, exp[i].Pair, exp[i].AssetType)
			require.NoErrorf(t, err, "the %s %s ticker must be stored", channel, exp[i].AssetType)
			assert.Equalf(t, &exp[i], stored, "the %s %s ticker should be stored as relayed", channel, exp[i].AssetType)
		}
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
		assert.Equalf(t, tc.exp, quotedInUSD(tc.instrumentAsset), "quotedInUSD should report whether %s %s is quoted in USD", tc.Asset, tc.Pair)
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
	for _, frame := range []string{
		`{"id":1,"method":"subscribe","code":0,"channel":"book.BTC_USD.10"}`,
		`{"id":-1,"method":"subscribe","code":0,"result":null}`,
		string(marketDataFrame(t, "ticker.1INCH_USD", 0)),
	} {
		require.NoErrorf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), "wsHandleChannelData must not error for %s", frame)
	}
	assert.Empty(t, relayedData(ex), "nothing should be relayed for an acknowledgement or an instrument not enabled")

	for _, frame := range []string{`{"result":`, `{"result":"x"}`, `{"result":[1]}`} {
		assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse %s", frame)
	}

	unhandled := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"nope.BTC_USD","channel":"nope","data":[]}}`
	require.NoError(t, ex.wsHandleChannelData(t.Context(), nil, []byte(unhandled)), "wsHandleChannelData must not error for a channel it does not know")
	assert.Equal(t, []any{websocket.UnhandledMessageWarning{Message: ex.Name + websocket.UnhandledMessage + unhandled}}, relayedData(ex), "a channel wsHandleChannelData does not know should be relayed as unhandled")

	channels := []string{channelBook, channelBookUpdate, channelTicker, channelTrade, channelCandlestick, IndexPriceChannel, MarkPriceChannel, SettlementPriceChannel, FundingRateChannel, EstimatedFundingRateChannel}
	for _, channel := range channels {
		frame := `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"BTC_USD","subscription":"x","channel":"` + channel + `","interval":"1m","data":{}}}`
		assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), errMalformedWebsocketMessage, "wsHandleChannelData should refuse %s data it cannot decode", channel)
		if channel == IndexPriceChannel {
			// An index is named for itself rather than an instrument
			continue
		}
		frame = `{"id":-1,"method":"subscribe","code":0,"result":{"instrument_name":"NOPE_USD","subscription":"x","channel":"` + channel + `","interval":"1m","data":[]}}`
		assert.ErrorIsf(t, ex.wsHandleChannelData(t.Context(), nil, []byte(frame)), currency.ErrPairNotFound, "wsHandleChannelData should refuse %s data for an instrument no asset trades", channel)
	}
	assert.Empty(t, relayedData(ex), "nothing should be relayed for data that cannot be decoded or routed")
}

func TestWebsocketMarketDataConnect(t *testing.T) {
	t.Parallel()
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
		_, err = ex.Websocket.Orderbook.GetOrderbook(spotBTCUSD, a)
		assert.NoErrorf(t, err, "the %s book should be loaded", a)
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

func TestWebsocketSubscribeRejected(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("rejected subscriptions are tested against the mock websocket")
	}
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
}

func TestWebsocketSubscribeRefused(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("refused subscribe requests are tested against the mock websocket")
	}
	venue := newMarketDataVenue(t)
	venue.fail(websocketMethodSubscribe, &APIError{Code: 40001, Message: "BAD_REQUEST"})
	start := time.Now()
	ex, err := newMarketDataTestExchange(t, venue, subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: currency.Pairs{spotBTCUSD, spotBTCUSDT}}})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "Connect must report the refused subscribe request")
	assert.Equal(t, &APIError{Code: 40001, Message: "BAD_REQUEST"}, apiErr, "Connect should report why the subscribe request was refused")
	assert.Less(t, time.Since(start), ex.Config.WebsocketResponseMaxLimit, "Connect should not wait for answers to a subscribe request refused as a whole")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "no subscription should be stored")
}

func TestWebsocketSubscribeBookUpdates(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("the subscribe requests are tested against the mock websocket, which records them")
	}
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
}
