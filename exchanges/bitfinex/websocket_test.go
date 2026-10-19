package bitfinex

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	v18 "github.com/thrasher-corp/gocryptotrader/config/versions/v18"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/nonce"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// wsFixture holds a websocket fixture: the subscribed event confirming each recorded channel, and the channel's
// frames, under its fixture key
type wsFixture struct {
	subscribed map[string][]byte
	frames     map[string][][]byte
}

// wsPublicFixture holds testdata/wsPublic.json, trimmed from live recordings of Bitfinex's public channels: each
// channel's subscribed event, labelled with its fixture key as its subscription ID, then its frames, each book's up to
// a checksum of the book they leave
var wsPublicFixture = sync.OnceValues(func() (*wsFixture, error) {
	f, err := os.Open("testdata/wsPublic.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fixture := &wsFixture{subscribed: map[string][]byte{}, frames: map[string][][]byte{}}
	keys := map[int64]string{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		frame := bytes.Clone(s.Bytes())
		if frame[0] == '[' {
			channelID, err := jsonparser.GetInt(frame, "[0]")
			if err != nil {
				return nil, err
			}
			fixture.frames[keys[channelID]] = append(fixture.frames[keys[channelID]], frame)
			continue
		}
		if event, _ := jsonparser.GetString(frame, "event"); event != "subscribed" {
			continue
		}
		key, err := jsonparser.GetString(frame, "subId")
		if err != nil {
			return nil, err
		}
		channelID, err := jsonparser.GetInt(frame, "chanId")
		if err != nil {
			return nil, err
		}
		keys[channelID] = key
		fixture.subscribed[key] = frame
	}
	return fixture, s.Err()
})

var (
	testFundingUSD = currency.Pair{Base: currency.USD}
	testETHUSD     = currency.NewPair(currency.ETH, currency.USD)
)

// fixtureSubscription returns the subscription carrying a recorded channel
func fixtureSubscription(tb testing.TB, fixtureKey string) *subscription.Subscription {
	tb.Helper()
	switch fixtureKey {
	case "ticker:tBTCUSD":
		return &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}}
	case "ticker:fUSD":
		return &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.MarginFunding, Pairs: currency.Pairs{testFundingUSD}}
	case "trades:tBTCUSD":
		return &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}}
	case "trades:fUSD":
		return &subscription.Subscription{Channel: subscription.AllTradesChannel, Asset: asset.MarginFunding, Pairs: currency.Pairs{testFundingUSD}}
	case "candles:trade:1m:tBTCUSD":
		return &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, Interval: kline.OneMin}
	case "candles:trade:1m:fUSD:p30":
		return &subscription.Subscription{Channel: subscription.CandlesChannel, Asset: asset.MarginFunding, Pairs: currency.Pairs{testFundingUSD}, Interval: kline.OneMin, Params: map[string]any{CandlesPeriodKey: "p30"}}
	case "book:tBTCUSD:R0":
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, Levels: 25, Params: map[string]any{"prec": "R0"}}
	case "book:tETHUSD:P0":
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{testETHUSD}, Levels: 25, Params: map[string]any{"prec": "P0"}}
	case "book:fUSD:P0", "book:fUSD:R0", "book:fUSD:P2":
		return &subscription.Subscription{Channel: subscription.OrderbookChannel, Asset: asset.MarginFunding, Pairs: currency.Pairs{testFundingUSD}, Levels: 25, Params: map[string]any{"prec": fixtureKey[len(fixtureKey)-2:]}}
	case "status:deriv:tBTCF0:USTF0":
		return &subscription.Subscription{Channel: wsStatusChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testDerivativePair}}
	}
	require.Failf(tb, "Fixture channel must be known", "%s has no subscription", fixtureKey)
	return nil
}

// newChannelTestExchange returns an exchange named after the test, with the trade feed on, and a public connection that
// is not connected, for handling recorded frames
func newChannelTestExchange(t *testing.T) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newTestExchange(t)
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	conn, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	return ex, conn
}

// subscribeFixtureChannel subscribes a recorded channel, as its subscribed event confirmed it, returning its
// subscription
func subscribeFixtureChannel(t *testing.T, ex *Exchange, conn websocket.Connection, fixtureKey string) *subscription.Subscription {
	t.Helper()
	fixture, err := wsPublicFixture()
	require.NoError(t, err, "wsPublicFixture must not error")
	require.Containsf(t, fixture.subscribed, fixtureKey, "the fixture must record channel %s", fixtureKey)
	s := fixtureSubscription(t, fixtureKey)
	ex.wsChannels.addPending(fixtureKey, conn, s)
	t.Cleanup(func() { ex.wsChannels.removePending(fixtureKey) })
	_, err = conn.MatchReturnResponses(t.Context(), wsSubscribeSignature(fixtureKey), 1)
	require.NoError(t, err, "MatchReturnResponses must not error")
	require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, fixture.subscribed[fixtureKey]), "wsHandleData must not error for channel %s's subscribed event", fixtureKey)
	return s
}

// replayFixtureChannel subscribes a recorded channel, then handles its frames, returning the errors handling them
func replayFixtureChannel(t *testing.T, ex *Exchange, conn websocket.Connection, fixtureKey string) []error {
	t.Helper()
	subscribeFixtureChannel(t, ex, conn, fixtureKey)
	fixture, err := wsPublicFixture()
	require.NoError(t, err, "wsPublicFixture must not error")
	var errs []error
	for _, frame := range fixture.frames[fixtureKey] {
		if err := ex.wsHandleData(t.Context(), conn, frame); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// relayRecorder keeps the errors relayed by a connected exchange, draining its data handler from before it connects, as
// a busy feed fills the handler's buffer in seconds
type relayRecorder struct {
	mu   sync.Mutex
	errs []error
}

// relayRecorders holds each connected test exchange's recorder
var relayRecorders sync.Map

// startRelayRecorder drains an exchange's data handler until the test ends
func startRelayRecorder(t *testing.T, ex *Exchange) {
	t.Helper()
	r := new(relayRecorder)
	relayRecorders.Store(ex, r)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case p := <-ex.Websocket.DataHandler.C:
				if err, ok := p.Data.(error); ok {
					r.mu.Lock()
					r.errs = append(r.errs, err)
					r.mu.Unlock()
				}
			case <-done:
				return
			}
		}
	}()
	t.Cleanup(func() {
		close(done)
		relayRecorders.Delete(ex)
	})
}

// recordedErrors returns the errors a connected exchange has relayed
func recordedErrors(t *testing.T, ex *Exchange) []error {
	t.Helper()
	stored, ok := relayRecorders.Load(ex)
	require.True(t, ok, "the exchange must be connected with a relay recorder")
	r, ok := stored.(*relayRecorder)
	require.True(t, ok, "the relay recorder must be stored")
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.errs)
}

// matchedResponse returns the one response a request awaiting a match received
func matchedResponse(t *testing.T, responses <-chan websocket.MatchedResponse) []byte {
	t.Helper()
	select {
	case r := <-responses:
		require.NoError(t, r.Err, "the matched response must not error")
		require.Len(t, r.Responses, 1, "one response must be matched")
		return r.Responses[0]
	case <-time.After(time.Second):
		require.Fail(t, "a response must be matched")
		return nil
	}
}

// relayedPayloads drains what has been relayed so far, leaving out order book depth pointers, which the order book
// tests inspect through the stored books instead
func relayedPayloads(ex *Exchange) []any {
	var got []any
	for {
		select {
		case p := <-ex.Websocket.DataHandler.C:
			if _, ok := p.Data.(*orderbook.Depth); !ok {
				got = append(got, p.Data)
			}
		default:
			return got
		}
	}
}

func TestWebsocketFrameData(t *testing.T) {
	t.Parallel()
	for frame, exp := range map[string]string{
		`[1,[2,3],4]`:     `[2,3],4]`,
		`[1, "hb",4]`:     `"hb",4]`,
		`[1]`:             ``,
		`[17082,[[1]],1]`: `[[1]],1]`,
	} {
		assert.Equalf(t, exp, string(wsFrameData([]byte(frame))), "wsFrameData should return %s from its data on", frame)
	}
	for data, exp := range map[string]bool{`[[1,2]]`: true, `[ ]`: true, `[ [1]]`: true, `[1,2]`: false, `"hb"`: false, ``: false, `[`: false} {
		assert.Equalf(t, exp, isSnapshot([]byte(data)), "isSnapshot should report %s correctly", data)
	}
}

func TestWebsocketHandleTicker(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	require.Empty(t, replayFixtureChannel(t, ex, conn, "ticker:tBTCUSD"), "the trading pair ticker's frames must be handled without error")
	got, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, ticker.Price{
		Last: 82187, High: 82641, Low: 79821, Bid: 82191, BidSize: 2.40768183, Ask: 82200, AskSize: 1.29977942, BaseVolume: 530.82146983,
		Open24Hour: 82187 - 1103, PercentChange24Hour: 1.360318, Pair: currency.NewBTCUSD(), ExchangeName: ex.Name, AssetType: asset.Spot,
		LastUpdated: time.UnixMilli(1792128960280),
	}, *got, "the stored ticker should be the last update, as of the time Bitfinex sent it")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 2, "each ticker update must be relayed")
	assert.IsType(t, &ticker.Price{}, relayed[1], "a ticker update should be relayed as a ticker price")

	require.Empty(t, replayFixtureChannel(t, ex, conn, "ticker:fUSD"), "the funding currency ticker's frames must be handled without error")
	got, err = ticker.GetTicker(ex.Name, testFundingUSD, asset.MarginFunding)
	require.NoError(t, err, "GetTicker must not error for the funding currency")
	assert.Equal(t, ticker.Price{
		Last: 0.00013466, High: 0.0002, Low: 0.000001, Bid: 0.0001712, BidSize: 17680416.90222361, Ask: 0.0001292255, AskSize: 73981.58877564,
		BaseVolume: 96352522.29982284, Pair: testFundingUSD, ExchangeName: ex.Name, AssetType: asset.MarginFunding, LastUpdated: time.UnixMilli(1792128955839),
		FlashReturnRate: 0.0003007013698630137, BidPeriod: 120, AskPeriod: 2, FlashReturnRateAmount: 220889680.31768298,
	}, *got, "the stored funding ticker should hold its flash return rate and periods, its best bid rate above its best offer's")
}

func TestWebsocketHandleDerivativeStatus(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	require.Empty(t, replayFixtureChannel(t, ex, conn, "status:deriv:tBTCF0:USTF0"), "the derivative status channel's frames must be handled without error")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 4, "each status update must be relayed as funding data and as the pair's status")
	status := DerivativeStatus{
		Time: types.Time(time.UnixMilli(1792362497000)), DerivativePrice: 81640.14553928, SpotPrice: 81592.5, InsuranceFundBalance: 61230742.63264047,
		NextFundingTime: types.Time(time.UnixMilli(1792368000000)), NextFundingAccrued: 0.0004144, NextFundingStep: 7436, CurrentFunding: 0.00003859,
		MarkPrice: 81589.7751, OpenInterest: 9268.67851682, ClampMinimum: 0.0005, ClampMaximum: 0.0025,
	}
	assert.Equal(t, websocket.FundingData{
		Timestamp: time.UnixMilli(1792362497000), CurrencyPair: testDerivativePair, AssetType: asset.USDTMarginedFutures, Exchange: ex.Name, Rate: 0.00003859,
	}, relayed[0], "a status update should relay the pair's current funding rate")
	assert.Equal(t, &WebsocketDerivativeStatus{Pair: testDerivativePair, AssetType: asset.USDTMarginedFutures, Status: status}, relayed[1], "a status update should relay the pair's status")
}

func TestWebsocketHandleSharedChannels(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	keys := []string{"ticker:tBTCUSD", "trades:tBTCUSD", "candles:trade:1m:tBTCUSD", "book:tBTCUSD:R0"}
	shared := make(map[string]*subscription.Subscription)
	for _, k := range keys {
		s := subscribeFixtureChannel(t, ex, conn, k)
		s.QualifiedChannel = k
		shared[k] = s
	}
	ex.wsSharedChannels.set(shared)
	var relayed []any
	for _, k := range keys {
		for _, frame := range mustPublicFixture(t).frames[k] {
			// The book's recording ends with a checksum, which both its books must match
			require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, frame), "wsHandleData must not error for channel %s's frame %s", k, frame)
			relayed = append(relayed, relayedPayloads(ex)...)
		}
	}
	spotTicker, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetTicker must not error for spot")
	marginTicker, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Margin)
	require.NoError(t, err, "GetTicker must not error for margin")
	spotTicker.AssetType = asset.Margin
	assert.Equal(t, spotTicker, marginTicker, "a channel spot and margin share should store margin's ticker as spot's")

	spotBook, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error for spot")
	marginBook, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Margin)
	require.NoError(t, err, "GetOrderbook must not error for margin")
	assert.Equal(t, spotBook.Bids, marginBook.Bids, "a channel spot and margin share should keep margin's bids as spot's")
	assert.Equal(t, spotBook.Asks, marginBook.Asks, "a channel spot and margin share should keep margin's asks as spot's")

	trades := map[asset.Item]int{}
	candles := map[asset.Item]int{}
	for _, p := range relayed {
		switch v := p.(type) {
		case []trade.Data:
			for i := range v {
				trades[v[i].AssetType]++
			}
		case kline.Item:
			candles[v.Asset] += len(v.Candles)
		}
	}
	assert.NotZero(t, trades[asset.Spot], "the shared channel's trades should be relayed for spot")
	assert.Equal(t, trades[asset.Spot], trades[asset.Margin], "the shared channel's trades should be relayed for margin as well as spot")
	assert.NotZero(t, candles[asset.Spot], "the shared channel's candles should be relayed for spot")
	assert.Equal(t, candles[asset.Spot], candles[asset.Margin], "the shared channel's candles should be relayed for margin as well as spot")
}

func TestWebsocketHandleSharedChannelMarginJoins(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	s := subscribeFixtureChannel(t, ex, conn, "book:tBTCUSD:R0")
	channelID, ok := ex.wsChannels.idOf(conn, s)
	require.True(t, ok, "the book's channel must be confirmed")
	frame := func(data string) []byte { return []byte("[" + strconv.FormatInt(channelID, 10) + "," + data + "]") }
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[[1,100,1],[2,101,-1]],1792128962861`)), "a snapshot must load")
	// Margin enables the pair once the channel has sent its snapshot
	s.QualifiedChannel = "book:tBTCUSD:R0"
	ex.wsSharedChannels.set(map[string]*subscription.Subscription{s.QualifiedChannel: s})
	err := ex.wsHandleData(t.Context(), conn, frame(`[3,99,1],1792128962862`))
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an update for a margin book without a snapshot must call for one")
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "an update for a margin book without a snapshot should report it missing")
	_, err = ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Spot)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "the spot book should await the snapshot margin's book needs")
	waitForRenewals(t, ex)

	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[[1,100,1],[2,101,-1]],1792128962863`)), "a fresh snapshot must load")
	err = ex.wsHandleData(t.Context(), conn, frame(`"cs",12345,1792128962864`))
	assert.ErrorIs(t, err, errChecksumMismatch, "a checksum neither book matches should error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Margin)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a checksum mismatch should invalidate margin's book too")
}

func TestWebsocketSharedBookToggle(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	s := subscribeFixtureChannel(t, ex, conn, "book:tBTCUSD:R0")
	// The request the test config's book subscription generates for the pair
	s.QualifiedChannel = `{"channel":"book","len":"100","prec":"R0","symbol":"tBTCUSD"}`
	channelID, ok := ex.wsChannels.idOf(conn, s)
	require.True(t, ok, "the book's channel must be confirmed")
	frame := func(data string) []byte { return []byte("[" + strconv.FormatInt(channelID, 10) + "," + data + "]") }
	share := func(shared bool) {
		t.Helper()
		if shared {
			require.NoError(t, ex.CurrencyPairs.EnablePair(asset.Margin, currency.NewBTCUSD()), "EnablePair must not error")
		} else {
			require.NoError(t, ex.CurrencyPairs.DisablePair(asset.Margin, currency.NewBTCUSD()), "DisablePair must not error")
		}
		_, err := ex.generateSubscriptions()
		require.NoError(t, err, "generateSubscriptions must not error")
		require.Equal(t, shared, ex.wsSharedChannels.carriesMargin(s), "the book's channel must carry margin's data only while margin enables its pair")
	}

	share(true)
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[[1,100,1],[2,101,-1]],1792128962861`)), "a snapshot must load both books")
	share(false)
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[3,99,1],1792128962862`)), "an update must apply to spot's book while margin's is unshared")
	_, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Margin)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a margin book its spot channel no longer updates should be invalidated")

	share(true)
	err = ex.wsHandleData(t.Context(), conn, frame(`[4,98,1],1792128962863`))
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "an update for a margin book left stale while unshared should call for a snapshot")
	// The test connection is not connected, so the renewal gives up
	waitForRenewals(t, ex)
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[[1,100,1],[2,101,-1],[3,99,1],[4,98,1]],1792128962864`)), "a fresh snapshot must load both books")
	spotBook, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Spot)
	require.NoError(t, err, "GetOrderbook must not error for spot")
	marginBook, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Margin)
	require.NoError(t, err, "GetOrderbook must not error for margin")
	assert.Len(t, marginBook.Bids, 3, "a fresh snapshot should restore margin's book in full")
	assert.Equal(t, spotBook.Bids, marginBook.Bids, "a fresh snapshot should leave margin's book as spot's")
}

func TestWebsocketHandleSharedTickerRelayFull(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	s := subscribeFixtureChannel(t, ex, conn, "ticker:tBTCUSD")
	s.QualifiedChannel = "ticker:tBTCUSD"
	ex.wsSharedChannels.set(map[string]*subscription.Subscription{s.QualifiedChannel: s})
	ex.Websocket.DataHandler = stream.NewRelay(1)
	require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "full"), "the relay must fill")
	assert.Error(t, ex.wsHandleData(t.Context(), conn, mustPublicFixture(t).frames["ticker:tBTCUSD"][0]), "a ticker the relay cannot take should error")
	_, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Margin)
	assert.NoError(t, err, "margin's ticker should be stored though spot's relay failed")
}

func TestWebsocketHandleTrades(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	require.Empty(t, replayFixtureChannel(t, ex, conn, "trades:tBTCUSD"), "the trading pair trades' frames must be handled without error")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 5, "the snapshot and each trade executed must be relayed, and execution updates not")
	snapshot, ok := relayed[0].([]trade.Data)
	require.True(t, ok, "the snapshot must be relayed as trades")
	require.Len(t, snapshot, 5, "every trade of the snapshot must be relayed")
	assert.Equal(t, trade.Data{
		Exchange: ex.Name, TID: "1994606740", CurrencyPair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy, Price: 82200, Amount: 0.00014877,
		Timestamp: time.UnixMilli(1792128951519),
	}, snapshot[3], "a snapshot trade with a positive amount should be a buy")
	executed, ok := relayed[2].([]trade.Data)
	require.True(t, ok, "a trade executed must be relayed as trades")
	assert.Equal(t, []trade.Data{{
		Exchange: ex.Name, TID: "1994606752", CurrencyPair: currency.NewBTCUSD(), AssetType: asset.Spot, Side: order.Buy, Price: 82200, Amount: 0.00081636,
		Timestamp: time.UnixMilli(1792128966407),
	}}, executed, "a trade executed should be relayed as it executed")

	require.Empty(t, replayFixtureChannel(t, ex, conn, "trades:fUSD"), "the funding currency trades' frames must be handled without error")
	relayed = relayedPayloads(ex)
	require.Len(t, relayed, 2, "the funding snapshot and its trade executed must be relayed")
	snapshot, ok = relayed[0].([]trade.Data)
	require.True(t, ok, "the funding snapshot must be relayed as trades")
	require.Len(t, snapshot, 5, "every funding trade of the snapshot must be relayed")
	assert.Equal(t, trade.Data{
		Exchange: ex.Name, TID: "434121307", CurrencyPair: testFundingUSD, AssetType: asset.MarginFunding, Side: order.Sell, Price: 0.00013466, Amount: 500,
		Timestamp: time.UnixMilli(1792128672000),
	}, snapshot[1], "a funding trade should be priced at its rate, a negative amount selling")
	assert.Equal(t, []trade.Data{{
		Exchange: ex.Name, TID: "434121334", CurrencyPair: testFundingUSD, AssetType: asset.MarginFunding, Side: order.Buy, Price: 0.00012194, Amount: 169.74,
		Timestamp: time.UnixMilli(1792129281000),
	}}, relayed[1], "a funding trade executed should be relayed as it executed")

	quiet, quietConn := newChannelTestExchange(t)
	quiet.Features.Enabled.TradeFeed = false
	require.Empty(t, replayFixtureChannel(t, quiet, quietConn, "trades:tBTCUSD"), "trades must be handled without error with the trade feed off")
	assert.Empty(t, relayedPayloads(quiet), "trades should not be relayed with the trade feed off")
}

func TestWebsocketHandleCandles(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	require.Empty(t, replayFixtureChannel(t, ex, conn, "candles:trade:1m:tBTCUSD"), "the candles' frames must be handled without error")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 4, "the snapshot and each candle update must be relayed")
	assert.Equal(t, kline.Item{
		Exchange: ex.Name, Pair: currency.NewBTCUSD(), Asset: asset.Spot, Interval: kline.OneMin,
		Candles: []kline.Candle{
			{Time: time.UnixMilli(1792128780000), Open: 82267, Close: 82220, High: 82271, Low: 82203, Volume: 0.03861858},
			{Time: time.UnixMilli(1792128840000), Open: 82212, Close: 82217, High: 82217, Low: 82190, Volume: 0.02480366},
			{Time: time.UnixMilli(1792128900000), Open: 82206, Close: 82187, High: 82210, Low: 82186, Volume: 0.01000381},
		},
	}, relayed[0], "a snapshot should be relayed oldest candle first")
	assert.Equal(t, kline.Item{
		Exchange: ex.Name, Pair: currency.NewBTCUSD(), Asset: asset.Spot, Interval: kline.OneMin,
		Candles: []kline.Candle{{Time: time.UnixMilli(1792128960000), Open: 82200, Close: 82202, High: 82202, Low: 82200, Volume: 0.00170513}},
	}, relayed[2], "a candle update should be relayed as one candle")

	require.Empty(t, replayFixtureChannel(t, ex, conn, "candles:trade:1m:fUSD:p30"), "the funding candles' frames must be handled without error")
	relayed = relayedPayloads(ex)
	require.Len(t, relayed, 1, "the funding snapshot must be relayed")
	item, ok := relayed[0].(kline.Item)
	require.True(t, ok, "the funding snapshot must be relayed as a kline item")
	assert.Equal(t, asset.MarginFunding, item.Asset, "funding candles should be relayed for the funding asset")
}

func TestWebsocketHandleBook(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		key      string
		pair     currency.Pair
		a        asset.Item
		bestBid  orderbook.Level
		bestAsk  orderbook.Level
		lastSent int64
	}{
		{
			// The book keeps orders sharing a price in the order they arrived, which its checksum takes by ID instead
			key: "book:tBTCUSD:R0", pair: currency.NewBTCUSD(), a: asset.Spot, lastSent: 1792128969756,
			bestBid: orderbook.Level{ID: 246641142863, Price: 82191, Amount: 0.0002},
			bestAsk: orderbook.Level{ID: 246646408161, Price: 82201, Amount: 0.07017366},
		},
		{
			key: "book:tETHUSD:P0", pair: testETHUSD, a: asset.Spot, lastSent: 1792128959234,
			bestBid: orderbook.Level{Price: 2447.4, Amount: 0.0011, OrderCount: 1},
			bestAsk: orderbook.Level{Price: 2447.7, Amount: 0.001, OrderCount: 1},
		},
		{
			key: "book:fUSD:P0", pair: testFundingUSD, a: asset.MarginFunding, lastSent: 1792128962860,
			bestBid: orderbook.Level{ID: 1, Price: 0.0001712, Period: 120, Amount: 20000, OrderCount: 1},
			bestAsk: orderbook.Level{ID: 26, Price: 0.0001292255, Period: 2, Amount: 2065.35, OrderCount: 2},
		},
		{
			key: "book:fUSD:R0", pair: testFundingUSD, a: asset.MarginFunding, lastSent: 1792128962860,
			bestBid: orderbook.Level{ID: 5038297953, Price: 0.0001712, Period: 120, Amount: 20000},
			bestAsk: orderbook.Level{ID: 5148998794, Price: 0.0001292255, Period: 2, Amount: 227.38},
		},
		{
			key: "book:fUSD:P2", pair: testFundingUSD, a: asset.MarginFunding, lastSent: 1792129180984,
			bestBid: orderbook.Level{ID: 1, Price: 0.000171, Period: 120, Amount: 2020000, OrderCount: 2},
			bestAsk: orderbook.Level{ID: 26, Price: 0.000129, Period: 2, Amount: 1593.85647, OrderCount: 4},
		},
	} {
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()
			ex, conn := newChannelTestExchange(t)
			// Each recording ends with a checksum of the book its frames leave, which an error here would report missed
			require.Empty(t, replayFixtureChannel(t, ex, conn, tc.key), "the book's frames must be handled without error, and its checksums must match")
			book, err := ex.Websocket.Orderbook.GetOrderbook(tc.pair, tc.a)
			require.NoError(t, err, "GetOrderbook must not error")
			require.Len(t, book.Bids, 25, "the book must hold 25 bids")
			require.Len(t, book.Asks, 25, "the book must hold 25 asks")
			assert.Equal(t, tc.bestBid, book.Bids[0], "the best bid should be Bitfinex's")
			assert.Equal(t, tc.bestAsk, book.Asks[0], "the best ask should be Bitfinex's")
			assert.Equal(t, tc.a == asset.MarginFunding, book.IsFundingRate, "a funding book should be marked as one")
			assert.Equal(t, time.UnixMilli(tc.lastSent), book.LastUpdated, "the book should be updated as of the time Bitfinex sent its last update")
		})
	}
}

func TestWebsocketHandleChecksumMismatch(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	require.Empty(t, replayFixtureChannel(t, ex, conn, "book:fUSD:R0"), "the book's frames must be handled without error")
	channelID, err := jsonparser.GetInt(mustPublicFixture(t).subscribed["book:fUSD:R0"], "chanId")
	require.NoError(t, err, "the fixture's channel ID must be read")
	frame := func(data string) []byte { return []byte("[" + strconv.FormatInt(channelID, 10) + "," + data + "]") }

	bookKey := key.PairAsset{Base: currency.USD.Item, Asset: asset.MarginFunding}

	err = ex.wsHandleData(t.Context(), conn, frame(`"cs",12345,1792128962861`))
	require.ErrorIs(t, err, errChecksumMismatch, "a checksum the book cannot match must error")
	_, err = ex.Websocket.Orderbook.GetOrderbook(testFundingUSD, asset.MarginFunding)
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a book failing its checksum should be invalidated")
	// The test connection is not connected, so resubscribing fails, and lets the book's next failure try again
	waitForRenewals(t, ex)
	assert.False(t, ex.wsBookResync.pending(bookKey), "a failed resubscription should not leave the book awaiting a snapshot")
	err = ex.wsHandleData(t.Context(), conn, frame(`"cs",12345,1792128962862`))
	assert.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "a checksum of a book left invalid should resubscribe it again")
	waitForRenewals(t, ex)

	_, started := ex.wsBookResync.start(bookKey)
	require.True(t, started, "start must mark the book as resubscribing")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`"cs",12345,1792128962863`)), "a checksum awaiting the book's fresh snapshot should be skipped")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[5148998794,2,0,1],1792128962864`)), "an update awaiting the book's fresh snapshot should be dropped")
	_, started = ex.wsBookResync.start(bookKey)
	assert.False(t, started, "a book awaiting its snapshot should not be resubscribed twice")

	for _, f := range mustPublicFixture(t).frames["book:fUSD:R0"] {
		require.NoError(t, ex.wsHandleData(t.Context(), conn, f), "the book's frames must be handled without error once resubscribed")
	}
	_, err = ex.Websocket.Orderbook.GetOrderbook(testFundingUSD, asset.MarginFunding)
	assert.NoError(t, err, "a fresh snapshot should restore the book")
}

// waitForRenewals waits for the channel renewals an exchange started, which give up once their connection is closed,
// failing rather than hanging should one keep retrying
func waitForRenewals(t *testing.T, ex *Exchange) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		ex.Websocket.Wg.Wait()
		close(done)
	}()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		require.FailNow(t, "channel renewals must give up on a closed connection")
	}
}

// mustPublicFixture returns the public websocket fixture
func mustPublicFixture(t *testing.T) *wsFixture {
	t.Helper()
	fixture, err := wsPublicFixture()
	require.NoError(t, err, "wsPublicFixture must not error")
	return fixture
}

func TestWebsocketProcessBookErrors(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	s := subscribeFixtureChannel(t, ex, conn, "book:fUSD:P0")
	channelID, ok := ex.wsChannels.idOf(conn, s)
	require.True(t, ok, "the book's channel must be confirmed")
	frame := func(data string) []byte { return []byte("[" + strconv.FormatInt(channelID, 10) + "," + data + "]") }
	err := ex.wsHandleData(t.Context(), conn, frame(`[0.0001,2,1,10],1792128962861`))
	assert.ErrorIs(t, err, orderbook.ErrDepthNotFound, "an update before the book's snapshot should error")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[[0.0002,30,1,-10],[0.0001,2,1,10]],1792128962861`)), "a snapshot must load")
	err = ex.wsHandleData(t.Context(), conn, frame(`[0.0003,2,0,1],1792128962862`))
	assert.ErrorIs(t, err, errFundingLevelUnknown, "removing a rate and period level the book does not hold should error")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[0.0002,2,1,-5],1792128962863`)), "a level sharing a rate with another must be added")
	book, err := ex.Websocket.Orderbook.GetOrderbook(testFundingUSD, asset.MarginFunding)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{ID: 1, Price: 0.0002, Period: 30, Amount: 10, OrderCount: 1}, {ID: 3, Price: 0.0002, Period: 2, Amount: 5, OrderCount: 1}}, book.Bids, "a level sharing a rate should follow the levels at it, matched by period")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, frame(`[0.0002,30,0,-1],1792128962864`)), "a level must be removed by rate and period")
	book, err = ex.Websocket.Orderbook.GetOrderbook(testFundingUSD, asset.MarginFunding)
	require.NoError(t, err, "GetOrderbook must not error")
	assert.Equal(t, orderbook.Levels{{ID: 3, Price: 0.0002, Period: 2, Amount: 5, OrderCount: 1}}, book.Bids, "removing a level should leave the one sharing its rate")

	raw, rawConn := newChannelTestExchange(t)
	rs := subscribeFixtureChannel(t, raw, rawConn, "book:tBTCUSD:R0")
	rawID, ok := raw.wsChannels.idOf(rawConn, rs)
	require.True(t, ok, "the raw book's channel must be confirmed")
	rawFrame := func(data string) []byte { return []byte("[" + strconv.FormatInt(rawID, 10) + "," + data + "]") }
	require.NoError(t, raw.wsHandleData(t.Context(), rawConn, rawFrame(`[[1,100,1],[2,101,-1]],1792128962861`)), "a raw snapshot must load")
	rawKey := key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.Spot}
	err = raw.wsHandleData(t.Context(), rawConn, rawFrame(`[3,0,1],1792128962862`))
	require.ErrorIs(t, err, orderbook.ErrOrderbookInvalid, "removing an order the book does not hold must invalidate it")
	waitForRenewals(t, raw)
	_, started := raw.wsBookResync.start(rawKey)
	require.True(t, started, "start must mark the book as resubscribing, its resubscription having failed")
	assert.NoError(t, raw.wsHandleData(t.Context(), rawConn, rawFrame(`[1,0,1],1792128962863`)), "updates awaiting the resubscribed book's snapshot should be dropped")
}

func TestBookChecksum(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		book *orderbook.Book
		raw  bool
		exp  string
	}{
		{
			name: "raw trading book",
			book: &orderbook.Book{
				Bids: orderbook.Levels{{ID: 9, Price: 100, Amount: 1}, {ID: 2, Price: 100, Amount: 2}, {ID: 5, Price: 99, Amount: 0.5}},
				Asks: orderbook.Levels{{ID: 7, Price: 101, Amount: 0.0000001}},
			},
			raw: true,
			exp: "2:2:7:-1e-7:9:1:5:0.5",
		},
		{
			name: "aggregated trading book",
			book: &orderbook.Book{
				Bids: orderbook.Levels{{Price: 2447.4, Amount: 0.0011}},
				Asks: orderbook.Levels{{Price: 2447.7, Amount: 1e21}, {Price: 2447.8, Amount: 0.000001}},
			},
			exp: "2447.4:0.0011:2447.7:-1e+21:2447.8:-0.000001",
		},
		{
			name: "raw funding book",
			book: &orderbook.Book{
				IsFundingRate: true,
				Bids:          orderbook.Levels{{ID: 8, Price: 0.0002, Period: 30, Amount: 10}, {ID: 4, Price: 0.0002, Period: 2, Amount: 5}},
				Asks:          orderbook.Levels{{ID: 6, Price: 0.0001, Period: 2, Amount: 1.5e-7}},
			},
			raw: true,
			exp: "4:-5:6:1.5e-7:8:-10",
		},
		{
			name: "aggregated funding book",
			book: &orderbook.Book{
				IsFundingRate: true,
				Bids:          orderbook.Levels{{ID: 2, Price: 0.0002, Period: 30, Amount: 10}, {ID: 1, Price: 0.0002, Period: 2, Amount: 5}},
				Asks:          orderbook.Levels{{ID: 3, Price: 0.00000012, Period: 2, Amount: 1}},
			},
			exp: "0.0002:-10:1.2e-7:1:0.0002:-5",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, crc32.ChecksumIEEE([]byte(tc.exp)), bookChecksum(tc.book, tc.raw), "bookChecksum should checksum the top levels as Bitfinex does")
		})
	}

	deep := &orderbook.Book{}
	for i := range 30 {
		deep.Bids = append(deep.Bids, orderbook.Level{ID: int64(100 - i), Price: 100, Amount: 1})
		deep.Asks = append(deep.Asks, orderbook.Level{ID: int64(200 + i), Price: float64(101 + i), Amount: 1})
	}
	exp := make([]byte, 0, wsChecksumDepth*24)
	for i := range wsChecksumDepth {
		exp = append(exp, []byte(strconv.Itoa(71+i)+":1:"+strconv.Itoa(200+i)+":-1:")...)
	}
	assert.Equal(t, crc32.ChecksumIEEE(exp[:len(exp)-1]), bookChecksum(deep, true), "bookChecksum should take the 25 lowest IDs of the orders sharing a price, from beyond the top 25 too")
}

func TestAppendJSNumber(t *testing.T) {
	t.Parallel()
	for f, exp := range map[float64]string{
		0: "0", 82191: "82191", -0.0002: "-0.0002", 0.000001: "0.000001", 1e-7: "1e-7", 1.5e-7: "1.5e-7", -9.99e-7: "-9.99e-7",
		1e20: "100000000000000000000", 1e21: "1e+21", 1.5e22: "1.5e+22", 0.00013122534246575346: "0.00013122534246575346", 1e-100: "1e-100",
	} {
		assert.Equalf(t, exp, string(appendJSNumber(nil, f)), "appendJSNumber should write %v as JavaScript does", f)
	}
	assert.Equal(t, "x:1e-7", string(appendJSNumber([]byte("x:"), 1e-7)), "appendJSNumber should append to what the buffer holds")
}

func TestWebsocketHandleDataErrors(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	subscribeFixtureChannel(t, ex, conn, "ticker:tBTCUSD")
	for _, tc := range []struct {
		frame string
		err   error
	}{
		{frame: ``, err: errWebsocketMalformedFrame},
		{frame: `  `, err: errWebsocketMalformedFrame},
		{frame: `"text"`, err: errWebsocketMalformedFrame},
		{frame: `["x"]`, err: errWebsocketMalformedFrame},
		{frame: `[404,[1]]`, err: errWebsocketChannelUnknown},
		{frame: `[27793]`, err: errWebsocketMalformedFrame},
		{frame: `[27793,"zz",1]`, err: errWebsocketMalformedFrame},
		{frame: `[27793,"cs",1,1]`, err: errWebsocketMalformedFrame},
		{frame: `[27793,"te",[1,1,1,1],1]`, err: errWebsocketMalformedFrame},
		{frame: `[27793,[1,"x"],1]`, err: errTupleElementType},
		{frame: `[0]`, err: errWebsocketMalformedFrame},
		{frame: `[0,[1]]`, err: errWebsocketMalformedFrame},
		{frame: `{"event":`, err: nil},
	} {
		err := ex.wsHandleData(t.Context(), conn, []byte(tc.frame))
		if tc.err == nil {
			assert.Errorf(t, err, "wsHandleData should error for %s", tc.frame)
			continue
		}
		assert.ErrorIsf(t, err, tc.err, "wsHandleData should return the expected error for %s", tc.frame)
	}
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`[27793,"hb",1792128965585]`)), "a heartbeat should be handled")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`[404,"hb",1792128965585]`)), "a heartbeat Bitfinex sends before confirming its channel should be handled")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`[27793,"tu",[1,1,1,1],1]`)), "a trade execution update should be ignored")
}

func TestWebsocketHandleEvent(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	err := ex.wsHandleData(t.Context(), conn, []byte(`{"event":"subscribed","channel":"ticker","chanId":1,"symbol":"tBTCUSD","subId":"unknown"}`))
	assert.ErrorIs(t, err, errWebsocketChannelUnknown, "a subscription Bitfinex confirms without a request should error")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"conf","status":"OK","flags":163840}`)), "an accepted configuration should be handled")
	assert.ErrorIs(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"conf","status":"FAILED"}`)), errWebsocketEventFailed, "a refused configuration should error")
	for _, event := range []string{`{"event":"auth","status":"OK"}`, `{"event":"unsubscribed","chanId":5}`} {
		assert.ErrorIsf(t, ex.wsHandleData(t.Context(), conn, []byte(event)), websocket.ErrSignatureNotMatched, "a reply no request awaits should error for %s", event)
	}
	err = ex.wsHandleData(t.Context(), conn, []byte(`{"event":"error","msg":"something","code":10000}`))
	assert.ErrorIs(t, err, errWebsocketEventFailed, "an error answering no request should error")
	assert.ErrorContains(t, err, "something (code 10000)", "an error answering no request should report Bitfinex's message and code")

	responses, err := conn.MatchReturnResponses(t.Context(), wsSubscribeSignature("14"), 1)
	require.NoError(t, err, "MatchReturnResponses must not error")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"channel":"book","symbol":"tNOPEUSD","subId":"14","event":"error","msg":"symbol: invalid","code":10300,"pair":"NOPEUSD"}`)), "a subscription error must reach its request")
	assert.Contains(t, string(matchedResponse(t, responses)), "symbol: invalid", "the subscription request should receive Bitfinex's error")
	responses, err = conn.MatchReturnResponses(t.Context(), wsUnsubscribeSignature(999999), 1)
	require.NoError(t, err, "MatchReturnResponses must not error")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"error","msg":"unsubscribe: invalid","chanId":999999,"code":10400}`)), "an unsubscription error must reach its request")
	assert.Contains(t, string(matchedResponse(t, responses)), "unsubscribe: invalid", "the unsubscription request should receive Bitfinex's error")

	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"pong","cid":1234,"ts":1792128975841}`)), "a pong should be handled")
	assert.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"info","version":2,"serverId":"x","platform":{"status":1}}`)), "the version info Bitfinex sends on connecting should be handled")
	assert.Empty(t, relayedPayloads(ex), "the version info should not be relayed")
	for _, code := range []int64{wsInfoServerRestart, wsInfoMaintenanceStart, wsInfoMaintenanceEnd} {
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"info","code":`+strconv.FormatInt(code, 10)+`,"msg":"platform info"}`)), "info code %d must be handled", code)
		assert.Equalf(t, []any{&PlatformInfoEvent{Code: code, Message: "platform info"}}, relayedPayloads(ex), "info code %d should be relayed", code)
	}
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"novel"}`)), "an unknown event must be handled")
	relayed := relayedPayloads(ex)
	require.Len(t, relayed, 1, "an unknown event must be relayed")
	assert.IsType(t, websocket.UnhandledMessageWarning{}, relayed[0], "an unknown event should be relayed as unhandled")
}

func TestWebsocketChannelRegistry(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	other, err := ex.Websocket.CreateTestConnection(wsPublicConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	var r wsChannelRegistry
	confirm := func(subID string, c websocket.Connection, channelID int64) error {
		_, err := r.confirm(subID, c, channelID)
		return err
	}
	s := fixtureSubscription(t, "ticker:tBTCUSD")
	assert.ErrorIs(t, confirm("a", conn, 1), errWebsocketChannelUnknown, "confirming a request never sent should error")
	r.addPending("a", conn, s)
	assert.ErrorIs(t, confirm("a", other, 1), errWebsocketChannelUnknown, "confirming a request on another connection should error")
	require.NoError(t, confirm("a", conn, 7), "confirm must not error")
	assert.Same(t, s, r.get(conn, 7), "get should return the confirmed subscription")
	assert.Nil(t, r.get(other, 7), "get should keep each connection's channels apart")
	id, ok := r.idOf(conn, fixtureSubscription(t, "ticker:tBTCUSD"))
	assert.True(t, ok, "idOf should find a subscription equal to the confirmed one")
	assert.Equal(t, int64(7), id, "idOf should return the channel ID")
	_, ok = r.idOf(other, s)
	assert.False(t, ok, "idOf should keep each connection's channels apart")
	r.remove(conn, 7)
	assert.Nil(t, r.get(conn, 7), "remove should forget the channel")

	open := &stateConnection{Connection: conn, connected: true}
	r.addPending("c", open, s)
	require.NoError(t, confirm("c", open, 8), "confirm must not error")
	r.pruneClosed()
	assert.Same(t, s, r.get(open, 8), "pruneClosed should keep a connection reporting itself connected")
	closed := &stateConnection{Connection: conn}
	r.addPending("b", closed, s)
	require.NoError(t, confirm("b", closed, 9), "confirm must not error")
	r.addPending("d", closed, s)
	r.pruneClosed()
	assert.Nil(t, r.get(closed, 9), "pruneClosed should forget a closed connection's channels")
	assert.ErrorIs(t, confirm("d", closed, 10), errWebsocketChannelUnknown, "pruneClosed should forget a closed connection's requests")
	assert.ErrorIs(t, confirm("b", closed, 9), errWebsocketChannelUnknown, "confirm should forget a confirmed request")
}

// stateConnection is a connection reporting itself connected or not
type stateConnection struct {
	websocket.Connection
	connected bool
}

func (c *stateConnection) IsConnected() bool { return c.connected }

func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	// A pair enabled for spot and margin has one channel, spot's
	require.NoError(t, ex.CurrencyPairs.EnablePair(asset.Margin, currency.NewBTCUSD()), "EnablePair must not error")
	pairs := map[asset.Item]currency.Pairs{}
	assets := []asset.Item{asset.Spot, asset.Margin, asset.MarginFunding, asset.USDTMarginedFutures, asset.CoinMarginedFutures}
	for _, a := range assets {
		enabled, err := ex.GetEnabledPairs(a)
		require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
		require.NotEmptyf(t, enabled, "the test config must enable %s pairs", a)
		pairs[a] = enabled
	}
	spot := pairs[asset.Spot]
	marginOnly := slices.DeleteFunc(slices.Clone(pairs[asset.Margin]), func(p currency.Pair) bool { return spot.Contains(p, true) })
	var exp subscription.List
	add := func(channel string, a asset.Item, p currency.Pair, interval kline.Interval, levels int, params map[string]any, qualified string) {
		exp = append(exp, &subscription.Subscription{Enabled: true, Channel: channel, Asset: a, Pairs: currency.Pairs{p}, Interval: interval, Levels: levels, Params: params, QualifiedChannel: qualified})
	}
	for _, a := range assets {
		assetPairs := pairs[a]
		if a == asset.Margin {
			assetPairs = marginOnly
		}
		for _, p := range assetPairs {
			symbol, err := symbolFor(a, p)
			require.NoError(t, err, "symbolFor must not error")
			add(subscription.TickerChannel, a, p, 0, 0, nil, `{"channel":"ticker","symbol":"`+symbol+`"}`)
			add(subscription.AllTradesChannel, a, p, 0, 0, nil, `{"channel":"trades","symbol":"`+symbol+`"}`)
			if a == asset.MarginFunding {
				add(subscription.CandlesChannel, a, p, kline.OneMin, 0, map[string]any{CandlesPeriodKey: "p30"}, `{"channel":"candles","key":"trade:1m:`+symbol+`:p30"}`)
			} else {
				add(subscription.CandlesChannel, a, p, kline.OneMin, 0, nil, `{"channel":"candles","key":"trade:1m:`+symbol+`"}`)
			}
			add(subscription.OrderbookChannel, a, p, 0, 100, map[string]any{"prec": "R0"}, `{"channel":"book","len":"100","prec":"R0","symbol":"`+symbol+`"}`)
			if isDerivativeAsset(a) {
				add(wsStatusChannel, a, p, 0, 0, nil, `{"channel":"status","key":"deriv:`+symbol+`"}`)
			}
		}
	}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs.Clone())
	for _, s := range subs {
		shared := s.Asset == asset.Spot && s.Pairs[0].Equal(currency.NewBTCUSD())
		assert.Equalf(t, shared, ex.wsSharedChannels.carriesMargin(s), "%s should carry margin's data only when margin enables its pair", s)
	}

	ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
	accountSubs, err := ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error")
	assert.Empty(t, accountSubs, "generateAccountSubscriptions should generate nothing without authenticated websocket use")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	accountSubs, err = ex.generateAccountSubscriptions()
	require.NoError(t, err, "generateAccountSubscriptions must not error")
	assert.Equal(t, subscription.List{{Channel: subscription.MyAccountChannel, Authenticated: true, Asset: asset.All}}, accountSubs, "generateAccountSubscriptions should generate the account channel")
}

func TestGenerateSubscriptionsFromMigratedConfig(t *testing.T) {
	t.Parallel()
	raw, err := testExchangeConfig()
	require.NoError(t, err, "testExchangeConfig must not error")
	// A configuration saved before perpetual contracts holds the defaults of the time, which the v18 upgrade replaces
	saved, err := jsonparser.Set(raw, []byte(`[{"enabled":true,"channel":"ticker","asset":"all"},{"enabled":true,"channel":"allTrades","asset":"all"},`+
		`{"enabled":true,"channel":"candles","asset":"spot","interval":"1m"},{"enabled":true,"channel":"candles","asset":"margin","interval":"1m"},`+
		`{"enabled":true,"channel":"candles","asset":"marginfunding","params":{"_period":"p30"},"interval":"1m"},`+
		`{"enabled":true,"channel":"orderbook","asset":"all","params":{"prec":"R0"},"levels":100}]`), "features", "subscriptions")
	require.NoError(t, err, "jsonparser.Set must not error")
	upgraded, err := new(v18.Version).UpgradeExchange(t.Context(), saved)
	require.NoError(t, err, "UpgradeExchange must not error")
	var exchCfg config.Exchange
	require.NoError(t, json.Unmarshal(upgraded, &exchCfg), "the upgraded configuration must load")
	testsubs.EqualLists(t, defaultSubscriptions.Clone(), exchCfg.Features.Subscriptions.Clone())

	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	require.NoError(t, ex.Setup(&exchCfg), "Setup must not error for the upgraded configuration")
	ex.Name = t.Name()
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error for the upgraded configuration")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		enabled, err := ex.GetEnabledPairs(a)
		require.NoErrorf(t, err, "GetEnabledPairs must not error for %s", a)
		require.NotEmptyf(t, enabled, "the test config must enable %s pairs", a)
		for _, channel := range []string{subscription.TickerChannel, subscription.AllTradesChannel, subscription.CandlesChannel, subscription.OrderbookChannel, wsStatusChannel} {
			n := slices.IndexFunc(subs, func(s *subscription.Subscription) bool { return s.Channel == channel && s.Asset == a })
			assert.NotEqualf(t, -1, n, "the upgraded configuration should subscribe %s's %s channel", a, channel)
		}
	}
}

func TestSubToMap(t *testing.T) {
	t.Parallel()
	_, err := subToMap(&subscription.Subscription{Channel: subscription.CandlesChannel, Params: map[string]any{CandlesPeriodKey: 30}}, asset.MarginFunding, testFundingUSD)
	assert.ErrorIs(t, err, common.ErrTypeAssertFailure, "subToMap should error for a candle period that is not a string")
	for _, k := range []string{"key", "symbol", "len", "event", "subId"} {
		_, err = subToMap(&subscription.Subscription{Channel: subscription.TickerChannel, Params: map[string]any{k: "x"}}, asset.Spot, currency.NewBTCUSD())
		assert.ErrorIsf(t, err, errSubscriptionParamReserved, "subToMap should refuse the reserved parameter %s", k)
	}
	_, err = subToMap(&subscription.Subscription{Channel: subscription.TickerChannel}, asset.Options, currency.NewBTCUSD())
	assert.ErrorIs(t, err, asset.ErrNotSupported, "subToMap should refuse an asset Bitfinex has no symbol for")
	got, err := subToMap(&subscription.Subscription{Channel: "novel", Params: map[string]any{"freq": "F1"}}, asset.Spot, currency.NewPair(currency.NewCode("AVAX"), currency.USD))
	require.NoError(t, err, "subToMap must not error")
	assert.Equal(t, map[string]any{"channel": "novel", "freq": "F1", "symbol": "tAVAX:USD"}, got, "subToMap should pass other channels and parameters through, with the pair's symbol")
	for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: testDerivativePair, asset.CoinMarginedFutures: testCoinMarginedPair} {
		got, err = subToMap(&subscription.Subscription{Channel: wsStatusChannel}, a, p)
		require.NoErrorf(t, err, "subToMap must not error for %s's status", a)
		symbol, err := symbolFor(a, p)
		require.NoError(t, err, "symbolFor must not error")
		assert.Equalf(t, map[string]any{"channel": "status", "key": "deriv:" + symbol}, got, "subToMap should key %s's status by its derivative pair", a)
	}
	_, err = subToMap(&subscription.Subscription{Channel: wsStatusChannel}, asset.Spot, currency.NewBTCUSD())
	assert.ErrorIs(t, err, errStatusChannelAsset, "subToMap should refuse the status of a pair that is not a derivative's")
}

func TestWebsocketSubscribeErrors(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	err := ex.wsSubscribe(t.Context(), conn, subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD(), testETHUSD}, QualifiedChannel: `{"channel":"ticker"}`}})
	assert.ErrorIs(t, err, errSubscriptionNotSinglePair, "wsSubscribe should refuse a subscription to several pairs")
	err = ex.wsSubscribe(t.Context(), conn, subscription.List{{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, QualifiedChannel: `{`}})
	assert.Error(t, err, "wsSubscribe should error for a subscription whose request cannot be read")
	unconfirmed := &subscription.Subscription{Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, QualifiedChannel: `{"channel":"ticker"}`}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, unconfirmed), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subscription.List{unconfirmed}), "wsUnsubscribe must not error for a subscription without a channel")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "wsUnsubscribe should forget a subscription without a channel, without a request")
}

func TestWebsocketSubscribe(t *testing.T) {
	t.Parallel()
	subs := subscription.List{
		{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}},
		{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, Levels: 25, Params: map[string]any{"prec": "R0"}},
	}
	ex, venue := newWebsocketTestExchange(t, subs)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		_, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
		assert.NoError(c, err, "GetTicker should find the subscribed ticker")
		book, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Spot)
		if assert.NoError(c, err, "GetOrderbook should find the subscribed book") {
			assert.NotEmpty(c, book.Bids, "the subscribed book should hold bids")
			assert.NotEmpty(c, book.Asks, "the subscribed book should hold asks")
		}
	}, 30*time.Second, 50*time.Millisecond, "the subscribed channels must deliver data")
	if !mockTests {
		// Bitfinex sends a busy book's checksum every few seconds
		assert.Never(t, func() bool { return len(recordedErrors(t, ex)) > 0 }, 10*time.Second, 100*time.Millisecond, "the book's checksums should match while it updates")
	}
	assert.Empty(t, recordedErrors(t, ex), "the subscribed channels should be handled without error, their checksums matching")
	if mockTests {
		confs := venue.events("conf")
		require.NotEmpty(t, confs, "each connection must configure its flags")
		assert.Equal(t, map[string]any{"event": "conf", "flags": float64(wsConfFlags)}, confs[0], "the connection should ask for event times and checksums")
		requests := venue.events("subscribe")
		require.Len(t, requests, 2, "each subscription must be sent")
		for _, req := range requests {
			assert.NotEmpty(t, req["subId"], "a subscription should carry an ID to match its confirmation with")
		}
	}

	conn, err := ex.Websocket.GetConnection(wsPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	stored := ex.Websocket.GetSubscriptions().Public()
	require.Len(t, stored, 2, "both subscriptions must be stored")
	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, stored), "UnsubscribeChannels must not error")
	assert.Empty(t, ex.Websocket.GetSubscriptions().Public(), "unsubscribing should forget the subscriptions")
	for _, s := range stored {
		_, ok := ex.wsChannels.idOf(conn, s)
		assert.Falsef(t, ok, "unsubscribing should forget %s's channel", s)
	}
	if mockTests {
		assert.Len(t, venue.events("unsubscribe"), 2, "each channel should be unsubscribed")
	}
}

func TestWebsocketChecksumResubscribes(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("Bitfinex's checksums cannot be made to fail live")
	}
	subs := subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, Levels: 25, Params: map[string]any{"prec": "R0"}}}
	ex, venue := newWebsocketTestExchange(t, subs, withBadChecksum("book:tBTCUSD:R0"))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Len(c, venue.events("subscribe"), 2, "the book should be resubscribed after failing its checksum")
		_, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Spot)
		assert.NoError(c, err, "the resubscribed book should be valid")
	}, 30*time.Second, 50*time.Millisecond, "the book must recover")
	assert.Len(t, venue.events("unsubscribe"), 1, "the failed channel should be unsubscribed before resubscribing")
}

func TestWebsocketChecksumResubscribeRetries(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("Bitfinex's checksums cannot be made to fail live")
	}
	subs := subscription.List{{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}, Levels: 25, Params: map[string]any{"prec": "R0"}}}
	ex, venue := newWebsocketTestExchange(t, subs, withBadChecksum("book:tBTCUSD:R0"), withRefusedResubscriptions("book:tBTCUSD:R0", 1))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.Len(c, venue.events("subscribe"), 3, "the book should be subscribed again after its refused resubscription")
		_, err := ex.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSD(), asset.Spot)
		assert.NoError(c, err, "the resubscribed book should be valid")
	}, 30*time.Second, 50*time.Millisecond, "the book must recover from a refused resubscription")
	assert.Len(t, venue.events("unsubscribe"), 1, "a retry should subscribe the channel its refused resubscription left unsubscribed")
	assert.Len(t, ex.Websocket.GetSubscriptions().Public(), 1, "the manager should keep the book's subscription throughout")
}

func TestWebsocketRenewChannel(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("renewing a channel needs the mock's replies")
	}
	subs := subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSD()}}}
	ex, venue := newWebsocketTestExchange(t, subs, withRefusedResubscriptions("ticker:tBTCUSD", 1))
	conn, err := ex.Websocket.GetConnection(wsPublicConnection)
	require.NoError(t, err, "GetConnection must not error")
	stored := ex.Websocket.GetSubscriptions().Public()
	require.Len(t, stored, 1, "the ticker's subscription must be stored")
	s := stored[0]
	first, ok := ex.wsChannels.idOf(conn, s)
	require.True(t, ok, "the ticker's channel must be confirmed")

	assert.ErrorIs(t, ex.wsRenewChannel(t.Context(), conn, s), websocket.ErrSubscriptionFailure, "a renewal whose subscription is refused should error")
	_, ok = ex.wsChannels.idOf(conn, s)
	assert.False(t, ok, "a refused renewal should leave the channel unsubscribed")
	assert.True(t, ex.wsChannels.isParked(conn, s), "a refused renewal should park the subscription for its retry")
	require.NoError(t, ex.wsRenewChannel(t.Context(), conn, s), "retrying the renewal must not error")
	renewed, ok := ex.wsChannels.idOf(conn, s)
	require.True(t, ok, "a renewal must map the channel Bitfinex gives it")
	assert.NotEqual(t, first, renewed, "a renewal should map the new channel")
	assert.False(t, ex.wsChannels.isParked(conn, s), "a renewal should unpark the subscription once subscribed")
	assert.Len(t, venue.events("unsubscribe"), 1, "a retry should not unsubscribe a channel already unsubscribed")
	assert.Len(t, venue.events("subscribe"), 3, "each renewal should subscribe the channel")

	require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, stored), "UnsubscribeChannels must not error")
	assert.ErrorIs(t, ex.wsRenewChannel(t.Context(), conn, s), errWebsocketChannelDropped, "a channel the manager unsubscribed should not be renewed")
	assert.Len(t, venue.events("subscribe"), 3, "a dropped channel should not be subscribed")

	require.NoError(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{s.Clone()}), "SubscribeToChannels must not error")
	assert.ErrorIs(t, ex.wsRenewChannel(t.Context(), conn, s), errWebsocketChannelDropped, "a renewal should follow its own subscription, not an equal one subscribed since")
	assert.Len(t, venue.events("subscribe"), 4, "a dropped renewal should not subscribe")
	assert.Len(t, venue.events("unsubscribe"), 2, "a dropped renewal should not unsubscribe the channel of an equal subscription")
}

// scriptedConnection is a connection answering the requests awaiting replies with its script, in turn, and recording
// the messages sent without awaiting one, as encoded
type scriptedConnection struct {
	websocket.Connection
	script []func(payload any) ([]byte, error)
	sent   []string
}

// SendJSONMessage records the message
func (c *scriptedConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, data any) error {
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	c.sent = append(c.sent, string(msg))
	return nil
}

// SendMessageReturnResponse answers a request with the next reply of the script
func (c *scriptedConnection) SendMessageReturnResponse(_ context.Context, _ request.EndpointLimit, _, payload any) ([]byte, error) {
	if len(c.script) == 0 {
		return nil, fmt.Errorf("unscripted request %v", payload)
	}
	reply := c.script[0]
	c.script = c.script[1:]
	return reply(payload)
}

func TestWebsocketLateSubscribeConfirmation(t *testing.T) {
	t.Parallel()
	ex, test := newChannelTestExchange(t)
	var lateID string
	conn := &scriptedConnection{Connection: test, script: []func(any) ([]byte, error){
		func(any) ([]byte, error) { return []byte(`{"event":"unsubscribed","status":"OK","chanId":1}`), nil },
		func(payload any) ([]byte, error) {
			lateID, _ = payload.(map[string]any)["subId"].(string)
			return nil, fmt.Errorf("%w: %s", websocket.ErrSignatureTimeout, lateID)
		},
		func(any) ([]byte, error) {
			return []byte(`{"event":"error","msg":"subscribe: dup","channel":"ticker","code":10301,"chanId":2,"symbol":"tBTCUSD"}`), nil
		},
	}}
	s := fixtureSubscription(t, "ticker:tBTCUSD")
	s.QualifiedChannel = `{"channel":"ticker","symbol":"tBTCUSD"}`
	ex.wsChannels.addPending("initial", conn, s)
	_, err := ex.wsChannels.confirm("initial", conn, 1)
	require.NoError(t, err, "confirm must not error")

	require.ErrorIs(t, ex.wsRenewChannel(t.Context(), conn, s), websocket.ErrSignatureTimeout, "a renewal whose confirmation is late must error")
	require.NotEmpty(t, lateID, "the renewal must send its subscription")
	assert.True(t, ex.wsChannels.isParked(conn, s), "a renewal awaiting its confirmation should keep the subscription parked")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"subscribed","channel":"ticker","chanId":2,"symbol":"tBTCUSD","subId":"`+lateID+`"}`)), "a late confirmation must be handled")
	channelID, ok := ex.wsChannels.channelOf(conn, s)
	require.True(t, ok, "a late confirmation must map its channel")
	assert.Equal(t, int64(2), channelID, "a late confirmation should map the channel Bitfinex gave")
	assert.False(t, ex.wsChannels.isParked(conn, s), "a late confirmation should unpark its subscription")

	equal := s.Clone()
	require.NoError(t, ex.wsSendSubscribe(t.Context(), conn, equal), "a subscription Bitfinex refuses for its channel already carrying it must succeed")
	assert.Same(t, equal, ex.wsChannels.get(conn, 2), "a subscription refused as a duplicate should take its channel over")
	other := fixtureSubscription(t, "ticker:fUSD")
	other.QualifiedChannel = `{"channel":"ticker","symbol":"fUSD"}`
	conn.script = append(conn.script, func(any) ([]byte, error) {
		return []byte(`{"event":"error","msg":"subscribe: dup","code":10301,"chanId":2}`), nil
	})
	assert.ErrorIs(t, ex.wsSendSubscribe(t.Context(), conn, other), websocket.ErrSubscriptionFailure, "a duplicate of another subscription's channel should fail")

	cancelled := fixtureSubscription(t, "trades:tBTCUSD")
	cancelled.QualifiedChannel = `{"channel":"trades","symbol":"tBTCUSD"}`
	var cancelledID string
	conn.script = append(conn.script, func(payload any) ([]byte, error) {
		cancelledID, _ = payload.(map[string]any)["subId"].(string)
		return nil, context.Canceled
	})
	require.ErrorIs(t, ex.wsSendSubscribe(t.Context(), conn, cancelled), context.Canceled, "a subscription whose wait is cancelled must error")
	require.NoError(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"subscribed","channel":"trades","chanId":4,"symbol":"tBTCUSD","subId":"`+cancelledID+`"}`)), "a confirmation arriving after its wait was cancelled must be handled")
	assert.Same(t, cancelled, ex.wsChannels.get(conn, 4), "a confirmation arriving after its wait was cancelled should map its channel")

	ex.wsChannels.addPending("refused", conn, other)
	err = ex.wsHandleData(t.Context(), conn, []byte(`{"event":"error","msg":"subscribe: failed","code":10300,"subId":"refused"}`))
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "a refusal arriving after its request stopped waiting should be reported")
	_, err = ex.wsChannels.confirm("refused", conn, 3)
	assert.ErrorIs(t, err, errWebsocketChannelUnknown, "a refused request should no longer be pending")
}

func TestWebsocketLateConfirmationAfterUnsubscribe(t *testing.T) {
	t.Parallel()
	ex, test := newChannelTestExchange(t)
	var subIDs []string
	late := func(payload any) ([]byte, error) {
		subID, _ := payload.(map[string]any)["subId"].(string)
		subIDs = append(subIDs, subID)
		return nil, fmt.Errorf("%w: %s", websocket.ErrSignatureTimeout, subID)
	}
	conn := &scriptedConnection{Connection: test, script: []func(any) ([]byte, error){late, late}}
	tickerFrame := func(channelID string) []byte {
		return []byte(`[` + channelID + `,[82191,2.9326838,82200,1.3811192,1108,0.01366484,82192,530.82146983,82641,79821,1358182043000],1792128955839]`)
	}
	for i, reply := range []string{`{"event":"unsubscribed","status":"OK","chanId":7}`, `{"event":"error","msg":"unsubscribe: invalid","code":10400,"chanId":8}`} {
		s := fixtureSubscription(t, "ticker:tBTCUSD")
		s.QualifiedChannel = `{"channel":"ticker","symbol":"tBTCUSD"}`
		require.ErrorIs(t, ex.wsSendSubscribe(t.Context(), conn, s), websocket.ErrSignatureTimeout, "a subscription whose confirmation is late must error")
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, s), "AddSuccessfulSubscriptions must not error")
		require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subscription.List{s}), "wsUnsubscribe must not error for a subscription Bitfinex has not confirmed")
		require.Empty(t, ex.Websocket.GetSubscriptions(), "unsubscribing must forget the subscription")

		channelID := strconv.Itoa(7 + i)
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(`{"event":"subscribed","channel":"ticker","chanId":`+channelID+`,"symbol":"tBTCUSD","subId":"`+subIDs[i]+`"}`)), "a confirmation of an unsubscribed subscription must be handled for channel %s", channelID)
		assert.Nilf(t, ex.wsChannels.get(conn, int64(7+i)), "a confirmation of an unsubscribed subscription should not map channel %s", channelID)
		assert.Equalf(t, `{"event":"unsubscribe","chanId":`+channelID+`}`, conn.sent[i], "channel %s, which no subscription wants, should be unsubscribed", channelID)
		assert.NoErrorf(t, ex.wsHandleData(t.Context(), conn, tickerFrame(channelID)), "an event on draining channel %s should be dropped", channelID)
		require.NoErrorf(t, ex.wsHandleData(t.Context(), conn, []byte(reply)), "the reply unsubscribing draining channel %s must be handled", channelID)
		assert.ErrorIsf(t, ex.wsHandleData(t.Context(), conn, tickerFrame(channelID)), errWebsocketChannelUnknown, "an event on drained channel %s should be unknown", channelID)
	}
	_, err := ticker.GetTicker(ex.Name, currency.NewBTCUSD(), asset.Spot)
	assert.Error(t, err, "a draining channel's events should not be stored")
}

func TestWebsocketUnsubscribeParkedChannel(t *testing.T) {
	t.Parallel()
	ex, conn := newChannelTestExchange(t)
	s := subscribeFixtureChannel(t, ex, conn, "ticker:tBTCUSD")
	channelID, ok := ex.wsChannels.idOf(conn, s)
	require.True(t, ok, "the ticker's channel must be confirmed")
	ex.wsChannels.park(conn, channelID)
	require.True(t, ex.wsChannels.isParked(conn, s), "park must keep the subscription for its renewal")
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, s), "AddSuccessfulSubscriptions must not error")
	require.NoError(t, ex.wsUnsubscribe(t.Context(), conn, subscription.List{s}), "wsUnsubscribe must not error for a parked subscription")
	assert.False(t, ex.wsChannels.isParked(conn, s), "unsubscribing a parked subscription should cancel its renewal")
	assert.Empty(t, ex.Websocket.GetSubscriptions(), "unsubscribing a parked subscription should forget it, without a request")
	assert.ErrorIs(t, ex.wsRenewChannel(t.Context(), conn, s), errWebsocketChannelDropped, "a renewal cancelled by unsubscribing should not subscribe")
}

func TestWebsocketAuthenticate(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex := newTestExchange(t)
	// Bitfinex keeps one nonce per key across REST and the websocket
	before := ex.Requester.GetNonce(nonce.UnixNano)
	venue := connectWebsocketTestExchange(t, ex, nil)
	assert.True(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "authenticating should allow authenticated websocket use")
	if !mockTests {
		return
	}
	auth := venue.events("auth")
	require.Len(t, auth, 1, "the account connection must authenticate")
	assert.Equal(t, testAPIKey, auth[0]["apiKey"], "authentication should send the API key")
	authNonce, ok := auth[0]["authNonce"].(string)
	require.True(t, ok, "authentication must send its nonce as a string")
	assert.Equal(t, "AUTH"+authNonce, auth[0]["authPayload"], "authentication should sign AUTH and the nonce")
	sent, err := strconv.ParseInt(authNonce, 10, 64)
	require.NoError(t, err, "authentication's nonce must be a number")
	assert.Greater(t, sent, int64(before), "authentication's nonce should follow the REST requests signed before it")
	assert.Greater(t, int64(ex.Requester.GetNonce(nonce.UnixNano)), sent, "REST requests signed after authenticating should follow its nonce")

	// A nonce a REST request overtook is refused, and authentication tries again with a fresh one
	retried := newTestExchange(t)
	retriedVenue := connectWebsocketTestExchange(t, retried, nil, withRefusedAuthNonces(wsAuthNonceAttempts-1))
	assert.True(t, retried.Websocket.CanUseAuthenticatedEndpoints(), "authentication should succeed once Bitfinex takes its nonce")
	retriedAuth := retriedVenue.events("auth")
	require.Len(t, retriedAuth, wsAuthNonceAttempts, "authentication must try again while its nonce is refused")
	var last int64
	for _, a := range retriedAuth {
		n, ok := a["authNonce"].(string)
		require.True(t, ok, "authentication must send its nonce as a string")
		sent, err := strconv.ParseInt(n, 10, 64)
		require.NoError(t, err, "authentication's nonce must be a number")
		assert.Greater(t, sent, last, "each authentication should send a larger nonce than the last")
		last = sent
	}
	refused := newTestExchange(t)
	connectWebsocketTestExchange(t, refused, nil, withRefusedAuthNonces(wsAuthNonceAttempts))
	assert.False(t, refused.Websocket.CanUseAuthenticatedEndpoints(), "authentication should give up once Bitfinex refuses every nonce")

	// The mock authenticates only the placeholder key's signature
	bad := newTestExchange(t)
	bad.SetCredentials(&accounts.Credentials{Key: "other", Secret: testAPISecret})
	connectWebsocketTestExchange(t, bad, nil)
	assert.False(t, bad.Websocket.CanUseAuthenticatedEndpoints(), "a rejected authentication should turn authenticated websocket use off")
	assert.True(t, bad.Websocket.IsConnected(), "a rejected authentication should leave the public connection up")
}
