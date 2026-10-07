package binance

import (
	"bufio"
	"context"
	"os"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// Derivatives pairs the websocket fixtures use, stored as the config formats store them
var (
	testUFuturesPair         = currency.NewPairWithDelimiter("BTC", "USDT", currency.UnderscoreDelimiter)
	testUFuturesDisabledPair = currency.NewPairWithDelimiter("ETH", "USDT", currency.UnderscoreDelimiter)
	testCFuturesPair         = currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter)
	testCFuturesDisabledPair = currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.UnderscoreDelimiter)
	testOptionsPair          = currency.NewPairWithDelimiter("BTC", "261030-85000-C", currency.DashDelimiter)
	testOptionsDisabledPair  = currency.NewPairWithDelimiter("ETH", "261009-2700-C", currency.DashDelimiter)
)

const testListenKey = "pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1"

// newDerivativesTestExchange returns an exchange named after the test, so tests writing the shared ticker, order book
// and account stores cannot collide. It has one enabled and one disabled pair per derivatives asset, placeholder
// credentials and a trade feed
func newDerivativesTestExchange(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	ex.Name = tb.Name()
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	for a, pairs := range map[asset.Item]currency.Pairs{
		asset.USDTMarginedFutures: {testUFuturesPair, testUFuturesDisabledPair},
		asset.CoinMarginedFutures: {testCFuturesPair, testCFuturesDisabledPair},
		asset.Options:             {testOptionsPair, testOptionsDisabledPair},
	} {
		require.NoErrorf(tb, ex.CurrencyPairs.StorePairs(a, pairs, false), "StorePairs must not error for %s available pairs", a)
		require.NoErrorf(tb, ex.CurrencyPairs.StorePairs(a, pairs[:1], true), "StorePairs must not error for %s enabled pairs", a)
	}
	return ex
}

// useTestOrderbookSync replaces the order book synchronisation with one taking its snapshots from fetch, without the
// fetch delay
func useTestOrderbookSync(ex *Exchange, fetch func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error)) {
	ex.orderbookSync = orderbookmanager.NewUpdateManager(&orderbookmanager.UpdateManagerParams{
		FetchDeadline:      time.Second * 5,
		FetchOrderbook:     fetch,
		CheckPendingUpdate: checkPendingUpdate,
		Orderbook:          &ex.Websocket.Orderbook,
	})
}

// fixtureFrames returns the frames of a websocket fixture file, one per line
func fixtureFrames(tb testing.TB, path string) [][]byte {
	tb.Helper()
	f, err := os.Open(path)
	require.NoErrorf(tb, err, "Opening fixture %q must not error", path)
	defer func() { assert.NoError(tb, f.Close(), "Closing the fixture should not error") }()
	var frames [][]byte
	s := bufio.NewScanner(f)
	for s.Scan() {
		frames = append(frames, append([]byte(nil), s.Bytes()...))
	}
	require.NoError(tb, s.Err(), "Fixture scanner must not error")
	require.NotEmpty(tb, frames, "Fixture must contain frames")
	return frames
}

// fixtureFrame returns the first frame of a fixture whose stream, or for user data whose event type, matches
func fixtureFrame(tb testing.TB, path, streamOrEvent string) []byte {
	tb.Helper()
	for _, frame := range fixtureFrames(tb, path) {
		stream, _ := jsonparser.GetString(frame, "stream")
		event, _ := jsonparser.GetString(frame, "data", "e")
		if stream == streamOrEvent || event == streamOrEvent {
			return frame
		}
	}
	require.Failf(tb, "Fixture frame must exist", "%s has no %q frame", path, streamOrEvent)
	return nil
}

// fixtureFrameData returns the data of fixtureFrame
func fixtureFrameData(tb testing.TB, path, streamOrEvent string) []byte {
	tb.Helper()
	data, _, _, err := jsonparser.Get(fixtureFrame(tb, path, streamOrEvent), "data")
	require.NoError(tb, err, "Fixture frame data must exist")
	return data
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

func TestFormatToInterval(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		expected  kline.Interval
		expectErr bool
	}{
		{name: "one minute", input: "1m", expected: kline.OneMin},
		{name: "three minute", input: "3m", expected: kline.ThreeMin},
		{name: "five minute", input: "5m", expected: kline.FiveMin},
		{name: "fifteen minute", input: "15m", expected: kline.FifteenMin},
		{name: "thirty minute", input: "30m", expected: kline.ThirtyMin},
		{name: "one hour", input: "1h", expected: kline.OneHour},
		{name: "two hour", input: "2h", expected: kline.TwoHour},
		{name: "four hour", input: "4h", expected: kline.FourHour},
		{name: "six hour", input: "6h", expected: kline.SixHour},
		{name: "eight hour", input: "8h", expected: kline.EightHour},
		{name: "twelve hour", input: "12h", expected: kline.TwelveHour},
		{name: "one day", input: "1d", expected: kline.OneDay},
		{name: "three day", input: "3d", expected: kline.ThreeDay},
		{name: "one week", input: "1w", expected: kline.OneWeek},
		{name: "one month", input: "1M", expected: kline.OneMonth},
		{name: "invalid empty", input: "", expected: 0, expectErr: true},
		{name: "invalid casing", input: "1H", expected: 0, expectErr: true},
		{name: "invalid value", input: "10m", expected: 0, expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := formatToInterval(tc.input)
			if tc.expectErr {
				assert.ErrorIs(t, err, kline.ErrInvalidInterval, "formatToInterval should reject the interval")
				assert.Zero(t, got, "formatToInterval should return no interval")
				return
			}
			assert.NoError(t, err, "formatToInterval should not error")
			assert.Equal(t, tc.expected, got, "formatToInterval should return the interval")
		})
	}
}
