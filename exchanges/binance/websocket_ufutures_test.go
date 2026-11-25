package binance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

var (
	errTestNoSnapshot   = errors.New("no snapshot in this test")
	errTestDial         = errors.New("test dial failure")
	errTestRetryAttempt = errors.New("test retry attempt failure")
)

// fixtureTime converts a millisecond fixture timestamp
func fixtureTime(ms int64) types.Time {
	return types.Time(time.UnixMilli(ms))
}

// streamRequestRecorder records the requests a mock stream server receives and answers them. reply returns the frame
// sent back for a request, or nil for the success reply
type streamRequestRecorder struct {
	mu       sync.Mutex
	requests []DerivativesStreamRequest
	reply    func(*DerivativesStreamRequest) []byte
}

func (r *streamRequestRecorder) handle(tb testing.TB, msg []byte, w *gws.Conn) error {
	tb.Helper()
	var req DerivativesStreamRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}
	r.mu.Lock()
	r.requests = append(r.requests, req)
	r.mu.Unlock()
	resp := fmt.Appendf(nil, `{"result":null,"id":%d}`, req.ID)
	if r.reply != nil {
		if custom := r.reply(&req); custom != nil {
			resp = custom
		}
	}
	return w.WriteMessage(gws.TextMessage, resp)
}

func (r *streamRequestRecorder) recorded() []DerivativesStreamRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]DerivativesStreamRequest(nil), r.requests...)
}

// newDerivativesWsTestExchange returns a test exchange whose REST requests are served from testdata/http.json and
// whose connections are dialled, without subscriptions, to a mock stream server answering with recorder
func newDerivativesWsTestExchange(t *testing.T, recorder *streamRequestRecorder) *Exchange {
	t.Helper()
	ex := newDerivativesTestExchange(t)
	require.NoError(t, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
	s := httptest.NewTestServer(t, mockws.CurryWsMockUpgrader(t, recorder.handle))
	s.Start()
	require.NoError(t, ex.Websocket.SetAllConnectionURLs("ws"+strings.TrimPrefix(s.URL, "http")), "SetAllConnectionURLs must not error")
	ex.Features.Subscriptions = subscription.List{}
	ex.Websocket.SetSubscriptionsNotRequired()
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() {
		// Disabled first, as the connection monitor would otherwise redial the closed mock server; the monitor may then
		// shut the websocket down itself
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail once the monitor has shut the websocket down")
		}
	})
	return ex
}

// liveStreamsSkip is why the live derivatives stream cases skip in the mock build
const liveStreamsSkip = "live websocket streams are only tested with -tags mock_test_off"

// newLiveDerivativesExchange returns an exchange set up from the test configuration with only asset a enabled, so the
// WebSocket API connection, which the derivatives streams do not use, is not registered, and with the test credentials
// when they are set
func newLiveDerivativesExchange(t *testing.T, a asset.Item) *Exchange {
	t.Helper()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	exchCfg, err := cfg.GetExchangeConfig("Binance")
	require.NoError(t, err, "GetExchangeConfig must not error")
	for _, other := range exchCfg.CurrencyPairs.GetAssetTypes(false) {
		if other != a {
			require.NoErrorf(t, exchCfg.CurrencyPairs.SetAssetEnabled(other, false), "SetAssetEnabled must not error for %s", other)
		}
	}
	ex := new(Exchange)
	ex.SetDefaults()
	ex.Websocket = sharedtestvalues.NewTestWebsocket()
	ex.Accounts = accounts.MustNewAccounts(ex)
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error")
	ex.Name = t.Name()
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(apiCredentials)
	}
	return ex
}

// connectLiveDerivatives connects ex's websocket to Binance with subs as its configured subscriptions, so the stream
// connections made are those of the subscriptions and, with authenticated websocket use, the enabled asset's private
// connection. A pair other than the empty pair becomes a's one pair
func connectLiveDerivatives(t *testing.T, ex *Exchange, a asset.Item, pair currency.Pair, subs subscription.List) {
	t.Helper()
	if !pair.IsEmpty() {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, false), "StorePairs must not error for %s", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{pair}, true), "StorePairs must not error for %s", a)
	}
	ex.Features.Subscriptions = subs
	// Registered before connecting, as a failed connection leaves the connection monitor trying again
	t.Cleanup(func() {
		// Disabled first, as the connection monitor would otherwise keep reconnecting after the test; the monitor may
		// then shut the websocket down itself
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail once the monitor has shut the websocket down")
		}
	})
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
}

// liveTickerSubscriptions returns a's stored ticker subscriptions once connectLiveDerivatives has subscribed them
func liveTickerSubscriptions(t *testing.T, ex *Exchange, a asset.Item) subscription.List {
	t.Helper()
	subs := slices.DeleteFunc(ex.Websocket.GetSubscriptions(), func(s *subscription.Subscription) bool {
		return s.Asset != a || s.Channel != subscription.TickerChannel
	})
	require.Lenf(t, subs, 1, "the %s ticker must be subscribed", a)
	return subs
}

// liveListenKeys is held for reading by the live cases that need the account's listen keys to stay valid until they
// end, and for writing by the live cases closing the keys, which run in parallel with them
var liveListenKeys sync.RWMutex

// liveUserDataConnections are the private connections whose user data streams the live cases subscribe to
var liveUserDataConnections = []struct {
	filter string
	a      asset.Item
}{
	{usdtmPrivateFilter, asset.USDTMarginedFutures},
	{coinmPrivateFilter, asset.CoinMarginedFutures},
	{optionsPrivateFilter, asset.Options},
}

// liveUserDataSubscription returns a's stored user data stream subscription once authenticated websocket use has
// subscribed it
func liveUserDataSubscription(t *testing.T, ex *Exchange, a asset.Item) *subscription.Subscription {
	t.Helper()
	subs := slices.DeleteFunc(ex.Websocket.GetSubscriptions(), func(s *subscription.Subscription) bool {
		return s.Asset != a || s.Channel != userDataStreamChannel
	})
	require.Lenf(t, subs, 1, "the %s user data stream must be subscribed", a)
	return subs[0]
}

// failingDialConnection is a connection whose dial fails
type failingDialConnection struct{ websocket.Connection }

func (failingDialConnection) Dial(context.Context, *gws.Dialer, http.Header, url.Values) error {
	return errTestDial
}

// matchRecordingConnection records the signatures of the request replies routed to it
type matchRecordingConnection struct {
	websocket.Connection
	signatures []any
}

func (c *matchRecordingConnection) RequireMatchWithData(signature any, _ []byte) error {
	c.signatures = append(c.signatures, signature)
	return nil
}

func TestWebsocketDerivativesConnect(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	err := e.WebsocketDerivativesConnect(t.Context(), failingDialConnection{})
	assert.ErrorIs(t, err, errTestDial, "WebsocketDerivativesConnect should return the dial error")

	ex := testexch.MockWebsocketInstance[Exchange](t, mockws.CurryWsMockUpgrader(t, mockws.EchoHandler))
	// The cases share ex and its connections, so they run in order
	for _, filter := range []string{usdtmPublicFilter, usdtmMarketFilter, usdtmPrivateFilter, coinmFilter, coinmPrivateFilter, optionsPublicFilter, optionsMarketFilter, optionsPrivateFilter} {
		t.Run(subtestName(filter), func(t *testing.T) {
			_, err := ex.Websocket.GetConnection(filter)
			assert.NoErrorf(t, err, "WebsocketDerivativesConnect should dial the %s connection at its configured URL", filter)
		})
	}
}

func TestGenerateUSDTMarginedFuturesPublicSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	subs, err := ex.generateUSDTMarginedFuturesPublicSubscriptions()
	require.NoError(t, err, "generateUSDTMarginedFuturesPublicSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.OrderbookChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, Interval: kline.HundredMilliseconds, QualifiedChannel: "btcusdt@depth@100ms"},
	}, subs)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.generateUSDTMarginedFuturesPublicSubscriptions()
	require.NoError(t, err, "generating subscriptions for a disabled asset must not error")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions, so its connection is skipped")
}

func TestGenerateUSDTMarginedFuturesMarketSubscriptions(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	subs, err := ex.generateUSDTMarginedFuturesMarketSubscriptions()
	require.NoError(t, err, "generateUSDTMarginedFuturesMarketSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@ticker"},
		{Channel: subscription.AllTradesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@aggTrade"},
		{Channel: subscription.CandlesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, Interval: kline.OneMin, QualifiedChannel: "btcusdt@kline_1m"},
	}, subs)

	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
	subs, err = ex.generateUSDTMarginedFuturesMarketSubscriptions()
	require.NoError(t, err, "generating subscriptions for a disabled asset must not error")
	assert.Empty(t, subs, "a disabled asset should generate no subscriptions, so its connection is skipped")
}

func TestGenerateDerivativesSubscriptions(t *testing.T) {
	t.Parallel()
	t.Run("partial failure", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		// An enabled asset whose pairs cannot be formatted must not fail every other connection
		require.NoError(t, ex.CurrencyPairs.Store(asset.USDTMarginedFutures, &currency.PairStore{
			AssetEnabled: true,
			Available:    currency.Pairs{testUSDTMarginedFuturesPair},
			Enabled:      currency.Pairs{testUSDTMarginedFuturesPair},
		}), "Store must not error")
		_, err := ex.generateDerivativesSubscriptions(asset.USDTMarginedFutures, func(*subscription.Subscription) bool { return true })
		assert.ErrorIs(t, err, websocket.ErrSubscriptionPartial, "generateDerivativesSubscriptions should return a partial subscription error")
	})
}

func TestGenerateConfiguredDerivativesSubscriptions(t *testing.T) {
	t.Parallel()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	// The configuration asks for a USDⓈ-M partial book, USDⓈ-M trades of a pair of its own and no USDⓈ-M candles, for
	// COIN-M tickers alone, for options candles of another interval without tickers, and for nothing on spot; the order
	// book subscriptions of an asset keep to one a pair, and a subscription Binance has no stream for is reported
	// without holding up the others
	setup := func(t *testing.T, subs subscription.List) *Exchange {
		t.Helper()
		cfg := &config.Config{}
		require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
		exchCfg, err := cfg.GetExchangeConfig("Binance")
		require.NoError(t, err, "GetExchangeConfig must not error")
		exchCfg.Features.Subscriptions = subs
		ex := new(Exchange)
		ex.SetDefaults()
		ex.Websocket = sharedtestvalues.NewTestWebsocket()
		ex.Accounts = accounts.MustNewAccounts(ex)
		require.NoError(t, ex.Setup(exchCfg), "Setup must not error")
		for a, p := range map[asset.Item]currency.Pair{asset.USDTMarginedFutures: testUSDTMarginedFuturesPair, asset.CoinMarginedFutures: testCoinMarginedFuturesPair, asset.Options: testOptionsPair} {
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for %s available pairs", a)
			require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for %s enabled pairs", a)
		}
		return ex
	}
	ex := setup(t, subscription.List{
		{Enabled: false, Asset: asset.Spot, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{testUSDTMarginedFuturesDisabledPair}},
		{Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel, Levels: 20, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel, Interval: kline.OneMin},
		{Enabled: false, Asset: asset.Options, Channel: subscription.TickerChannel},
		{Enabled: true, Asset: asset.Options, Channel: subscription.CandlesChannel, Interval: kline.FiveMin},
		{Enabled: true, Asset: asset.Options, Channel: subscription.OrderbookChannel, Levels: 10, Interval: kline.HundredMilliseconds},
		{Enabled: true, Asset: asset.Options, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
	})
	type generationCase struct {
		name     string
		generate func(*Exchange) (subscription.List, error)
		exp      subscription.List
		partial  bool
		setup    func(*testing.T) *Exchange
	}
	type listedStream struct {
		pair  currency.Pair
		topic string
	}
	const explicitPairs = "explicit pairs determine the streams/"
	listing := func(name string, a asset.Item, template *subscription.Subscription, available, enabled currency.Pairs, generate func(*Exchange) (subscription.List, error), streams ...listedStream) generationCase {
		listed := make(currency.Pairs, 0, len(streams))
		want := make(subscription.List, 0, len(streams))
		for _, stream := range streams {
			listed = append(listed, stream.pair)
			s := template.Clone()
			s.Asset = a
			s.Pairs = currency.Pairs{stream.pair}
			s.QualifiedChannel = stream.topic
			want = append(want, s)
		}
		return generationCase{
			name: explicitPairs + name, generate: generate, exp: want,
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				configured := template.Clone()
				configured.Pairs = listed
				rowExchange := setup(t, subscription.List{configured})
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(a, available, false), "StorePairs must set the available pairs")
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(a, enabled, true), "StorePairs must set the enabled pairs")
				return rowExchange
			},
		}
	}
	trades := func(name string, a, configuredAsset asset.Item, available, enabled currency.Pairs, generate func(*Exchange) (subscription.List, error), streams ...listedStream) generationCase {
		return listing(name, a, &subscription.Subscription{Enabled: true, Asset: configuredAsset, Channel: subscription.AllTradesChannel}, available, enabled, generate, streams...)
	}
	btcUSDC := currency.NewPairWithDelimiter("BTC", "USDC", currency.UnderscoreDelimiter)
	bnbUSDT := currency.NewPairWithDelimiter("BNB", "USDT", currency.UnderscoreDelimiter)
	dotPerpetual := currency.NewPairWithDelimiter("DOTUSD", "PERP", currency.UnderscoreDelimiter)
	ethPut := currency.NewPairWithDelimiter("ETH", "261009-2700-P", currency.DashDelimiter)
	firstStrike := currency.NewPairWithDelimiter("ETH", "261225-2700-C", currency.DashDelimiter)
	strikes := make([]listedStream, 0, 51)
	for i := range 50 {
		strike := strconv.Itoa(2700 + 10*i)
		pair := currency.NewPairWithDelimiter("ETH", "261225-"+strike+"-C", currency.UnderscoreDelimiter)
		strikes = append(strikes, listedStream{pair, "eth-261225-" + strike + "-c@optionTrade"})
	}
	strikes = append(strikes, listedStream{currency.NewPairWithDelimiter("BTC", "261225-2700-C", currency.DashDelimiter), "btc-261225-2700-c@optionTrade"})
	delimiters := []string{currency.UnderscoreDelimiter, currency.DashDelimiter, currency.ForwardSlashDelimiter}
	moreStrikes := make([]listedStream, 0, 202)
	for i := range 201 {
		strike := strconv.Itoa(2700 + 10*i)
		pair := currency.NewPairWithDelimiter("ETH", "261225-"+strike+"-C", delimiters[i%len(delimiters)])
		moreStrikes = append(moreStrikes, listedStream{pair, "eth-261225-" + strike + "-c@optionTrade"})
	}
	moreStrikes = append(moreStrikes, listedStream{currency.NewPairWithDelimiter("BTC", "261225-2700-C", currency.ForwardSlashDelimiter), "btc-261225-2700-c@optionTrade"})
	usdtMarginedPerpetuals := make([]listedStream, 0, 51)
	coinMarginedPerpetuals := make([]listedStream, 0, 51)
	for i := range 51 {
		base := "ALT" + strconv.Itoa(i)
		usdtMarginedPerpetuals = append(usdtMarginedPerpetuals, listedStream{currency.NewPairWithDelimiter(base, "USDT", currency.UnderscoreDelimiter), strings.ToLower(base) + "usdt@aggTrade"})
		coinMarginedPerpetuals = append(coinMarginedPerpetuals, listedStream{currency.NewPairWithDelimiter(base+"USD", "PERP", currency.UnderscoreDelimiter), strings.ToLower(base) + "usd_perp@aggTrade"})
	}
	ethfiUSDT := currency.NewPairWithDelimiter("ETHFI", "USDT", currency.UnderscoreDelimiter)
	xrpUSDT := currency.NewPairWithDelimiter("XRP", "USDT", currency.UnderscoreDelimiter)
	expand := func(template *subscription.Subscription, streams ...listedStream) subscription.List {
		subs := make(subscription.List, 0, len(streams))
		for _, stream := range streams {
			s := template.Clone()
			s.Pairs = currency.Pairs{stream.pair}
			s.QualifiedChannel = stream.topic
			subs = append(subs, s)
		}
		return subs
	}
	withPairs := func(template *subscription.Subscription, pairs ...currency.Pair) *subscription.Subscription {
		s := template.Clone()
		s.Pairs = slices.Clone(pairs)
		return s
	}
	bnbUSDTSlash := currency.NewPairWithDelimiter("BNB", "USDT", currency.ForwardSlashDelimiter)
	usdtMarginedTrades := &subscription.Subscription{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel}
	usdtMarginedTicker := &subscription.Subscription{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel}
	usdtMarginedCandles := &subscription.Subscription{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin}
	usdtMarginedMonthlyCandles := &subscription.Subscription{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMonth}
	usdtMarginedBook := &subscription.Subscription{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.OrderbookChannel, Levels: 5, Interval: kline.HundredMilliseconds}
	coinMarginedTicker := &subscription.Subscription{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel}
	coinMarginedCandles := &subscription.Subscription{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.CandlesChannel, Interval: kline.OneMin}
	coinMarginedBook := &subscription.Subscription{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.OrderbookChannel}
	optionsTicker := &subscription.Subscription{Enabled: true, Asset: asset.Options, Channel: subscription.TickerChannel}
	optionsBook := &subscription.Subscription{Enabled: true, Asset: asset.Options, Channel: subscription.OrderbookChannel, Levels: 10, Interval: kline.HundredMilliseconds}
	optionsCandles := &subscription.Subscription{Enabled: true, Asset: asset.Options, Channel: subscription.CandlesChannel, Interval: kline.OneMin}
	usdtMarginedPairs := currency.Pairs{testUSDTMarginedFuturesPair, testUSDTMarginedFuturesDisabledPair}
	coinMarginedPairs := currency.Pairs{testCoinMarginedFuturesDisabledPair, dotPerpetual}
	optionsPairs := currency.Pairs{testOptionsPair, testOptionsDisabledPair}
	everyChannel := func(t *testing.T) *Exchange {
		t.Helper()
		return setup(t, subscription.List{
			withPairs(usdtMarginedTicker, usdtMarginedPairs...), withPairs(usdtMarginedCandles, usdtMarginedPairs...), withPairs(usdtMarginedMonthlyCandles, usdtMarginedPairs...),
			withPairs(usdtMarginedBook, usdtMarginedPairs...),
			withPairs(coinMarginedTicker, coinMarginedPairs...), withPairs(coinMarginedCandles, coinMarginedPairs...), withPairs(coinMarginedBook, coinMarginedPairs...),
			withPairs(optionsTicker, optionsPairs...), withPairs(optionsBook, optionsPairs...), withPairs(optionsCandles, optionsPairs...),
		})
	}
	const everyChannelOn = explicitPairs + "tickers, candles and order books of two pairs on "
	allAssets := func(template *subscription.Subscription) *subscription.Subscription {
		s := template.Clone()
		s.Asset = asset.All
		return s
	}
	asAsset := func(a asset.Item, subs subscription.List) subscription.List {
		for _, s := range subs {
			s.Asset = a
		}
		return subs
	}
	everyAllAssetsChannel := func(templates ...*subscription.Subscription) func(*testing.T) *Exchange {
		return func(t *testing.T) *Exchange {
			t.Helper()
			configured := make(subscription.List, 0, len(templates))
			for _, template := range templates {
				configured = append(configured, allAssets(template))
			}
			return setup(t, configured)
		}
	}
	const everyAllAssetsChannelOn = explicitPairs + "all-assets tickers, candles and order books of two pairs on "
	usdtMarginedPerpetualsEnabled := make(currency.Pairs, 0, len(usdtMarginedPerpetuals))
	for _, stream := range usdtMarginedPerpetuals {
		usdtMarginedPerpetualsEnabled = append(usdtMarginedPerpetualsEnabled, stream.pair)
	}
	optionsTrades := &subscription.Subscription{Enabled: true, Asset: asset.Options, Channel: subscription.AllTradesChannel}
	// An all-assets subscription's listed pairs reach every derivatives asset unfiltered, as with subscription templates
	mixedAssets := func(connection string, a asset.Item, generate func(*Exchange) (subscription.List, error), enabled currency.Pair, streams ...listedStream) generationCase {
		return trades("all-assets list of every derivatives asset's pairs on "+connection, a, asset.All, currency.Pairs{enabled}, currency.Pairs{enabled}, generate, streams...)
	}
	for _, tc := range []generationCase{
		{"spot", (*Exchange).generateSubscriptions, subscription.List{}, false, nil},
		{"USDⓈ-M /public", (*Exchange).generateUSDTMarginedFuturesPublicSubscriptions, subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, Levels: 20, Interval: kline.HundredMilliseconds, QualifiedChannel: "btcusdt@depth20@100ms"},
		}, false, nil},
		{"USDⓈ-M /market", (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions, subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@ticker"},
			{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesDisabledPair}, QualifiedChannel: "ethusdt@aggTrade"},
		}, false, nil},
		{"COIN-M", (*Exchange).generateCoinMarginedFuturesSubscriptions, subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCoinMarginedFuturesPair}, QualifiedChannel: "btcusd_perp@ticker"},
		}, true, nil},
		{"options /public", (*Exchange).generateOptionsPublicSubscriptions, subscription.List{
			{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, Interval: kline.HundredMilliseconds, QualifiedChannel: "btc-261030-85000-c@depth@100ms"},
		}, true, nil},
		{"options /market", (*Exchange).generateOptionsMarketSubscriptions, subscription.List{
			{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, Interval: kline.FiveMin, QualifiedChannel: "btc-261030-85000-c@kline_5m"},
		}, false, nil},
		trades("USDⓈ-M with one enabled pair listed", asset.USDTMarginedFutures, asset.USDTMarginedFutures,
			currency.Pairs{testUSDTMarginedFuturesPair, testUSDTMarginedFuturesDisabledPair, bnbUSDT},
			currency.Pairs{testUSDTMarginedFuturesPair, bnbUSDT}, (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@aggTrade"},
			listedStream{testUSDTMarginedFuturesPair, "btcusdt@aggTrade"}, listedStream{btcUSDC, "btcusdc@aggTrade"}),
		trades("COIN-M with a subset of enabled pairs listed", asset.CoinMarginedFutures, asset.CoinMarginedFutures,
			currency.Pairs{testCoinMarginedFuturesPair, testCoinMarginedFuturesDisabledPair, dotPerpetual},
			currency.Pairs{testCoinMarginedFuturesPair, testCoinMarginedFuturesDisabledPair, dotPerpetual}, (*Exchange).generateCoinMarginedFuturesSubscriptions,
			listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@aggTrade"}, listedStream{testCoinMarginedFuturesPair, "btcusd_perp@aggTrade"}),
		trades("options without an enabled pair listed", asset.Options, asset.Options,
			currency.Pairs{testOptionsPair, testOptionsDisabledPair}, currency.Pairs{testOptionsPair}, (*Exchange).generateOptionsPublicSubscriptions,
			listedStream{testOptionsDisabledPair, "eth-261009-2700-c@optionTrade"}, listedStream{ethPut, "eth-261009-2700-p@optionTrade"}),
		trades("all-assets list with a dated USDⓈ-M contract", asset.USDTMarginedFutures, asset.All,
			currency.Pairs{testUSDTMarginedFuturesPair, bnbUSDT}, currency.Pairs{testUSDTMarginedFuturesPair, bnbUSDT}, (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			listedStream{testUSDTMarginedFuturesPair, "btcusdt@aggTrade"},
			listedStream{currency.NewPairWithDelimiter("BTCUSDT", "261225", currency.UnderscoreDelimiter), "btcusdt_261225@aggTrade"}),
		trades("COIN-M perpetual and quarterly contracts", asset.CoinMarginedFutures, asset.All,
			currency.Pairs{testCoinMarginedFuturesPair, testCoinMarginedFuturesDisabledPair, dotPerpetual}, currency.Pairs{testCoinMarginedFuturesPair, dotPerpetual}, (*Exchange).generateCoinMarginedFuturesSubscriptions,
			listedStream{currency.NewPairWithDelimiter("ETHUSD", "PERP", currency.DashDelimiter), "ethusd_perp@aggTrade"},
			listedStream{testCoinMarginedFuturesPair, "btcusd_perp@aggTrade"}, listedStream{testCoinMarginedFuturesUnlistedPair, "btcusd_261225@aggTrade"}),
		trades("all-assets list of 51 options contracts", asset.Options, asset.All,
			currency.Pairs{firstStrike}, currency.Pairs{firstStrike}, (*Exchange).generateOptionsPublicSubscriptions, strikes...),
		{
			name:     explicitPairs + "pairs listed on two trades subscriptions and a ticker",
			generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			exp: subscription.List{
				{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@aggTrade"},
				{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesDisabledPair}, QualifiedChannel: "ethusdt@aggTrade"},
				{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@ticker"},
			},
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				return setup(t, subscription.List{
					{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}},
					{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel, Pairs: currency.Pairs{testUSDTMarginedFuturesDisabledPair}},
					{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}},
				})
			},
		},
		{
			name:     explicitPairs + "trades subscriptions listing overlapping pairs, one of them twice",
			generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			exp: expand(usdtMarginedTrades, listedStream{testUSDTMarginedFuturesPair, "btcusdt@aggTrade"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@aggTrade"},
				listedStream{xrpUSDT, "xrpusdt@aggTrade"}, listedStream{ethfiUSDT, "ethfiusdt@aggTrade"}, listedStream{bnbUSDTSlash, "bnbusdt@aggTrade"}),
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				return setup(t, subscription.List{
					withPairs(usdtMarginedTrades, testUSDTMarginedFuturesPair, testUSDTMarginedFuturesDisabledPair, testUSDTMarginedFuturesPair, xrpUSDT),
					withPairs(usdtMarginedTrades, testUSDTMarginedFuturesDisabledPair, ethfiUSDT, bnbUSDTSlash),
				})
			},
		},
		{
			name: everyChannelOn + "USDⓈ-M /public", generate: (*Exchange).generateUSDTMarginedFuturesPublicSubscriptions, setup: everyChannel,
			exp: expand(usdtMarginedBook, listedStream{testUSDTMarginedFuturesPair, "btcusdt@depth5@100ms"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@depth5@100ms"}),
		},
		{
			name: everyChannelOn + "USDⓈ-M /market", generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions, setup: everyChannel,
			exp: slices.Concat(
				expand(usdtMarginedTicker, listedStream{testUSDTMarginedFuturesPair, "btcusdt@ticker"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@ticker"}),
				expand(usdtMarginedCandles, listedStream{testUSDTMarginedFuturesPair, "btcusdt@kline_1m"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@kline_1m"}),
				expand(usdtMarginedMonthlyCandles, listedStream{testUSDTMarginedFuturesPair, "btcusdt@kline_1M"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@kline_1M"}),
			),
		},
		{
			name: everyChannelOn + "COIN-M", generate: (*Exchange).generateCoinMarginedFuturesSubscriptions, setup: everyChannel,
			exp: slices.Concat(
				expand(coinMarginedTicker, listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@ticker"}, listedStream{dotPerpetual, "dotusd_perp@ticker"}),
				expand(coinMarginedCandles, listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@kline_1m"}, listedStream{dotPerpetual, "dotusd_perp@kline_1m"}),
				expand(coinMarginedBook, listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@depth"}, listedStream{dotPerpetual, "dotusd_perp@depth"}),
			),
		},
		{
			name: everyChannelOn + "options /public", generate: (*Exchange).generateOptionsPublicSubscriptions, setup: everyChannel,
			exp: slices.Concat(
				expand(optionsTicker, listedStream{testOptionsPair, "btc-261030-85000-c@optionTicker"}, listedStream{testOptionsDisabledPair, "eth-261009-2700-c@optionTicker"}),
				expand(optionsBook, listedStream{testOptionsPair, "btc-261030-85000-c@depth10@100ms"}, listedStream{testOptionsDisabledPair, "eth-261009-2700-c@depth10@100ms"}),
			),
		},
		{
			name: everyChannelOn + "options /market", generate: (*Exchange).generateOptionsMarketSubscriptions, setup: everyChannel,
			exp: expand(optionsCandles, listedStream{testOptionsPair, "btc-261030-85000-c@kline_1m"}, listedStream{testOptionsDisabledPair, "eth-261009-2700-c@kline_1m"}),
		},
		trades("all-assets list of USDⓈ-M pairs none of them available, dated contracts written with a dash and a slash", asset.USDTMarginedFutures, asset.All,
			currency.Pairs{testUSDTMarginedFuturesPair}, currency.Pairs{testUSDTMarginedFuturesPair}, (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@aggTrade"},
			listedStream{currency.NewPairWithDelimiter("ETHUSDT", "260327", currency.DashDelimiter), "ethusdt_260327@aggTrade"},
			listedStream{currency.NewPairWithDelimiter("BTCUSDT", "260626", currency.ForwardSlashDelimiter), "btcusdt_260626@aggTrade"}),
		trades("all-assets list of COIN-M quarterly contracts", asset.CoinMarginedFutures, asset.All,
			currency.Pairs{testCoinMarginedFuturesPair}, currency.Pairs{testCoinMarginedFuturesPair}, (*Exchange).generateCoinMarginedFuturesSubscriptions,
			listedStream{testCoinMarginedFuturesUnlistedPair, "btcusd_261225@aggTrade"},
			listedStream{currency.NewPairWithDelimiter("ETHUSD", "260327", currency.DashDelimiter), "ethusd_260327@aggTrade"},
			listedStream{currency.NewPairWithDelimiter("XRPUSD", "260626", currency.ForwardSlashDelimiter), "xrpusd_260626@aggTrade"}),
		trades("all-assets list of options puts differing only in expiry", asset.Options, asset.All,
			currency.Pairs{testOptionsPair}, currency.Pairs{testOptionsPair}, (*Exchange).generateOptionsPublicSubscriptions,
			listedStream{currency.NewPairWithDelimiter("ETH", "261225-2700-P", currency.DashDelimiter), "eth-261225-2700-p@optionTrade"},
			listedStream{currency.NewPairWithDelimiter("ETH", "260327-2700-P", currency.DashDelimiter), "eth-260327-2700-p@optionTrade"}),
		{
			name:     explicitPairs + "USDⓈ-M trades listing exactly its enabled pairs beside a ticker of every enabled pair",
			generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			exp: slices.Concat(
				expand(usdtMarginedTrades, listedStream{testUSDTMarginedFuturesPair, "btcusdt@aggTrade"}, listedStream{bnbUSDT, "bnbusdt@aggTrade"}),
				expand(usdtMarginedTicker, listedStream{testUSDTMarginedFuturesPair, "btcusdt@ticker"}, listedStream{bnbUSDT, "bnbusdt@ticker"}),
			),
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				rowExchange := setup(t, subscription.List{withPairs(usdtMarginedTrades, testUSDTMarginedFuturesPair, bnbUSDT), usdtMarginedTicker.Clone()})
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{testUSDTMarginedFuturesPair, testUSDTMarginedFuturesDisabledPair, bnbUSDT}, false), "StorePairs must set the available pairs")
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{testUSDTMarginedFuturesPair, bnbUSDT}, true), "StorePairs must set the enabled pairs")
				return rowExchange
			},
		},
		mixedAssets("USDⓈ-M /market", asset.USDTMarginedFutures, (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions, testUSDTMarginedFuturesPair,
			listedStream{dotPerpetual, "dotusdperp@aggTrade"}, listedStream{testOptionsPair, "btc261030-85000-c@aggTrade"}, listedStream{testUSDTMarginedFuturesPair, "btcusdt@aggTrade"}),
		mixedAssets("COIN-M", asset.CoinMarginedFutures, (*Exchange).generateCoinMarginedFuturesSubscriptions, dotPerpetual,
			listedStream{testUSDTMarginedFuturesPair, "btc_usdt@aggTrade"}, listedStream{testOptionsPair, "btc_261030-85000-c@aggTrade"}, listedStream{dotPerpetual, "dotusd_perp@aggTrade"}),
		mixedAssets("options /public", asset.Options, (*Exchange).generateOptionsPublicSubscriptions, testOptionsPair,
			listedStream{testUSDTMarginedFuturesPair, "btc-usdt@optionTrade"}, listedStream{dotPerpetual, "dotusd-perp@optionTrade"}, listedStream{testOptionsPair, "btc-261030-85000-c@optionTrade"}),
		trades("USDⓈ-M list of 51 pairs", asset.USDTMarginedFutures, asset.USDTMarginedFutures,
			currency.Pairs{testUSDTMarginedFuturesPair}, currency.Pairs{testUSDTMarginedFuturesPair}, (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions, usdtMarginedPerpetuals...),
		trades("COIN-M list of 51 contracts", asset.CoinMarginedFutures, asset.CoinMarginedFutures,
			currency.Pairs{testCoinMarginedFuturesPair}, currency.Pairs{testCoinMarginedFuturesPair}, (*Exchange).generateCoinMarginedFuturesSubscriptions, coinMarginedPerpetuals...),
		trades("all-assets list of 202 options contracts, 201 of them on one underlying", asset.Options, asset.All,
			currency.Pairs{firstStrike}, currency.Pairs{firstStrike}, (*Exchange).generateOptionsPublicSubscriptions, moreStrikes...),
		listing("all-assets candles listing options contracts", asset.Options, &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.CandlesChannel, Interval: kline.OneMin},
			currency.Pairs{testOptionsPair}, currency.Pairs{testOptionsPair}, (*Exchange).generateOptionsMarketSubscriptions,
			listedStream{testOptionsPair, "btc-261030-85000-c@kline_1m"}, listedStream{testOptionsDisabledPair, "eth-261009-2700-c@kline_1m"}),
		listing("all-assets order books listing USDⓈ-M pairs", asset.USDTMarginedFutures, &subscription.Subscription{Enabled: true, Asset: asset.All, Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds},
			currency.Pairs{testUSDTMarginedFuturesPair}, currency.Pairs{testUSDTMarginedFuturesPair}, (*Exchange).generateUSDTMarginedFuturesPublicSubscriptions,
			listedStream{testUSDTMarginedFuturesPair, "btcusdt@depth@100ms"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@depth@100ms"}),
		{
			name: everyAllAssetsChannelOn + "USDⓈ-M /market", generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			setup: everyAllAssetsChannel(withPairs(usdtMarginedTicker, usdtMarginedPairs...), withPairs(usdtMarginedCandles, usdtMarginedPairs...)),
			exp: asAsset(asset.USDTMarginedFutures, slices.Concat(
				expand(usdtMarginedTicker, listedStream{testUSDTMarginedFuturesPair, "btcusdt@ticker"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@ticker"}),
				expand(usdtMarginedCandles, listedStream{testUSDTMarginedFuturesPair, "btcusdt@kline_1m"}, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@kline_1m"}),
			)),
		},
		{
			name: everyAllAssetsChannelOn + "COIN-M", generate: (*Exchange).generateCoinMarginedFuturesSubscriptions,
			setup: everyAllAssetsChannel(withPairs(coinMarginedTicker, coinMarginedPairs...), withPairs(coinMarginedCandles, coinMarginedPairs...), withPairs(coinMarginedBook, coinMarginedPairs...)),
			exp: asAsset(asset.CoinMarginedFutures, slices.Concat(
				expand(coinMarginedTicker, listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@ticker"}, listedStream{dotPerpetual, "dotusd_perp@ticker"}),
				expand(coinMarginedCandles, listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@kline_1m"}, listedStream{dotPerpetual, "dotusd_perp@kline_1m"}),
				expand(coinMarginedBook, listedStream{testCoinMarginedFuturesDisabledPair, "ethusd_perp@depth"}, listedStream{dotPerpetual, "dotusd_perp@depth"}),
			)),
		},
		{
			name: everyAllAssetsChannelOn + "options /public", generate: (*Exchange).generateOptionsPublicSubscriptions,
			setup: everyAllAssetsChannel(withPairs(optionsTicker, optionsPairs...), withPairs(optionsBook, optionsPairs...)),
			exp: asAsset(asset.Options, slices.Concat(
				expand(optionsTicker, listedStream{testOptionsPair, "btc-261030-85000-c@optionTicker"}, listedStream{testOptionsDisabledPair, "eth-261009-2700-c@optionTicker"}),
				expand(optionsBook, listedStream{testOptionsPair, "btc-261030-85000-c@depth10@100ms"}, listedStream{testOptionsDisabledPair, "eth-261009-2700-c@depth10@100ms"}),
			)),
		},
		{
			name:     explicitPairs + "USDⓈ-M trades listing pairs beside a ticker of other enabled pairs",
			generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			exp: slices.Concat(
				expand(usdtMarginedTrades, listedStream{testUSDTMarginedFuturesDisabledPair, "ethusdt@aggTrade"}, listedStream{bnbUSDT, "bnbusdt@aggTrade"}),
				expand(usdtMarginedTicker, listedStream{testUSDTMarginedFuturesPair, "btcusdt@ticker"}, listedStream{xrpUSDT, "xrpusdt@ticker"}),
			),
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				rowExchange := setup(t, subscription.List{withPairs(usdtMarginedTrades, testUSDTMarginedFuturesDisabledPair, bnbUSDT), usdtMarginedTicker.Clone()})
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{testUSDTMarginedFuturesPair, testUSDTMarginedFuturesDisabledPair, bnbUSDT, xrpUSDT}, false), "StorePairs must set the available pairs")
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, currency.Pairs{testUSDTMarginedFuturesPair, xrpUSDT}, true), "StorePairs must set the enabled pairs")
				return rowExchange
			},
		},
		{
			name:     explicitPairs + "USDⓈ-M ticker of 51 enabled pairs",
			generate: (*Exchange).generateUSDTMarginedFuturesMarketSubscriptions,
			exp: expand(usdtMarginedTicker, func() []listedStream {
				tickers := make([]listedStream, 0, len(usdtMarginedPerpetuals))
				for _, stream := range usdtMarginedPerpetuals {
					tickers = append(tickers, listedStream{stream.pair, strings.TrimSuffix(stream.topic, "@aggTrade") + "@ticker"})
				}
				return tickers
			}()...),
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				rowExchange := setup(t, subscription.List{usdtMarginedTicker.Clone()})
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, usdtMarginedPerpetualsEnabled, false), "StorePairs must set the available pairs")
				require.NoError(t, rowExchange.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, usdtMarginedPerpetualsEnabled, true), "StorePairs must set the enabled pairs")
				return rowExchange
			},
		},
		{
			name:     explicitPairs + "options trades of 200 contracts beside a ticker",
			generate: (*Exchange).generateOptionsPublicSubscriptions,
			exp: slices.Concat(
				expand(optionsTrades, moreStrikes[:200]...),
				expand(optionsTicker, listedStream{testOptionsPair, "btc-261030-85000-c@optionTicker"}),
			),
			setup: func(t *testing.T) *Exchange {
				t.Helper()
				contracts := make(currency.Pairs, 0, 200)
				for _, stream := range moreStrikes[:200] {
					contracts = append(contracts, stream.pair)
				}
				return setup(t, subscription.List{withPairs(optionsTrades, contracts...), withPairs(optionsTicker, testOptionsPair)})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rowExchange := ex
			if tc.setup != nil {
				rowExchange = tc.setup(t)
			}
			configured := rowExchange.Features.Subscriptions.Clone()
			saved, err := json.Marshal(rowExchange.Config.Features.Subscriptions)
			require.NoError(t, err, "Marshal must not error")
			subs, err := tc.generate(rowExchange)
			if tc.partial {
				assert.ErrorIsf(t, err, websocket.ErrSubscriptionPartial, "%s generation should report the subscriptions it left out", tc.name)
			} else {
				assert.NoErrorf(t, err, "%s generation should not error", tc.name)
			}
			assert.Len(t, subs, len(tc.exp), "each expected stream should be generated once")
			testsubs.EqualLists(t, tc.exp, subs)
			assert.Equal(t, configured, rowExchange.Features.Subscriptions.Clone(), "generation should leave the configured subscriptions as they are")
			savedAfter, err := json.Marshal(rowExchange.Config.Features.Subscriptions)
			require.NoError(t, err, "Marshal must not error")
			assert.JSONEq(t, string(saved), string(savedAfter), "generation should leave the saved subscriptions as they are")
		})
	}
	t.Run("generation leaves the nested parameters of the configured subscriptions as they are", func(t *testing.T) {
		t.Parallel()
		book := withPairs(usdtMarginedBook, usdtMarginedPairs...)
		require.NoError(t, json.Unmarshal([]byte(`{"depth":{"levels":10}}`), &book.Params), "Unmarshal must decode the parameters as the loader does")
		rowExchange := setup(t, subscription.List{book})
		configured, err := json.Marshal(rowExchange.Features.Subscriptions)
		require.NoError(t, err, "Marshal must not error")
		saved, err := json.Marshal(rowExchange.Config.Features.Subscriptions)
		require.NoError(t, err, "Marshal must not error")
		for _, generate := range []func() (subscription.List, error){
			rowExchange.generateUSDTMarginedFuturesPublicSubscriptions, rowExchange.generateUSDTMarginedFuturesMarketSubscriptions, rowExchange.generateCoinMarginedFuturesSubscriptions,
			rowExchange.generateOptionsPublicSubscriptions, rowExchange.generateOptionsMarketSubscriptions,
		} {
			_, err := generate()
			require.NoError(t, err, "generating subscriptions must not error")
		}
		generated, err := json.Marshal(rowExchange.Features.Subscriptions)
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, string(configured), string(generated), "generation should leave the configured subscriptions' nested parameters as they are")
		savedAfter, err := json.Marshal(rowExchange.Config.Features.Subscriptions)
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEq(t, string(saved), string(savedAfter), "generation should leave the saved subscriptions' nested parameters as they are")
	})

	// An asset the configuration has no subscription of, enabled or not, streams its defaults
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  int
	}{
		{"an asset without configured subscriptions streams its four defaults", subscription.List{{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}}, 4},
		{"an asset whose configured subscriptions are all disabled streams nothing", subscription.List{{Enabled: false, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			subs, err := setup(t, tc.subs).generateCoinMarginedFuturesSubscriptions()
			require.NoError(t, err, "generateCoinMarginedFuturesSubscriptions must not error")
			assert.Len(t, subs, tc.exp, "COIN-M should stream its defaults only where the configuration has no subscription of it")
		})
	}

	// An all-assets subscription, enabled or not, is a configured subscription of spot and of each derivatives asset, so
	// none of them streams its defaults beside it, while an asset's own subscriptions still stream
	generateAll := func(t *testing.T, ex *Exchange) subscription.List {
		t.Helper()
		var all subscription.List
		for _, generate := range []func() (subscription.List, error){
			ex.generateSubscriptions, ex.generateUSDTMarginedFuturesPublicSubscriptions, ex.generateUSDTMarginedFuturesMarketSubscriptions,
			ex.generateCoinMarginedFuturesSubscriptions, ex.generateOptionsPublicSubscriptions, ex.generateOptionsMarketSubscriptions,
		} {
			subs, err := generate()
			require.NoError(t, err, "generating subscriptions must not error")
			all = append(all, subs...)
		}
		return all
	}
	tickers := func() subscription.List {
		return subscription.List{
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, QualifiedChannel: "btcusdt@ticker"},
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@ticker"},
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.CoinMarginedFutures, Pairs: currency.Pairs{testCoinMarginedFuturesPair}, QualifiedChannel: "btcusd_perp@ticker"},
			{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.Options, Pairs: currency.Pairs{testOptionsPair}, QualifiedChannel: "btc-261030-85000-c@optionTicker"},
		}
	}
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  subscription.List
	}{
		{"enabled all-assets ticker", subscription.List{{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel}}, tickers()},
		{"disabled all-assets ticker", subscription.List{{Enabled: false, Asset: asset.All, Channel: subscription.TickerChannel}}, subscription.List{}},
		{
			"all-assets ticker beside USDⓈ-M trades",
			subscription.List{
				{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.AllTradesChannel},
			},
			append(tickers(), &subscription.Subscription{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.USDTMarginedFutures, Pairs: currency.Pairs{testUSDTMarginedFuturesPair}, QualifiedChannel: "btcusdt@aggTrade"}),
		},
		{
			// Each asset's own ticker is the same stream as the all-assets one's, whatever delimiter its pairs are given
			// with, so each stream is subscribed once
			"all-assets ticker beside each asset's own",
			subscription.List{
				{Enabled: true, Asset: asset.All, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.TickerChannel},
				{Enabled: true, Asset: asset.Options, Channel: subscription.TickerChannel},
			},
			tickers(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := setup(t, tc.subs)
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, false), "StorePairs must not error for spot available pairs")
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{currency.NewBTCUSDT()}, true), "StorePairs must not error for spot enabled pairs")
			generated := generateAll(t, ex)
			require.Len(t, generated, len(tc.exp), "each stream must be generated once")
			testsubs.EqualLists(t, tc.exp, generated)
		})
	}

	// Each asset's user data stream already carries its orders, so MyOrdersChannel has no stream of its own
	myOrders := func(a asset.Item) subscription.List {
		return subscription.List{{Enabled: true, Asset: a, Channel: subscription.MyOrdersChannel, Authenticated: true}}
	}
	allMyOrders := func() subscription.List {
		subs := make(subscription.List, 0, 5)
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options, asset.All} {
			subs = append(subs, myOrders(a)...)
		}
		return subs
	}
	for _, auth := range []struct {
		name                 string
		canUseAuthenticated  bool
		flaggedAuthenticated bool
	}{
		{"my orders only generates no streams", true, true},
		{"my orders only without usable credentials generates no streams", false, true},
		{"my orders only not flagged as authenticated generates no streams", true, false},
		{"my orders only neither flagged nor with usable credentials generates no streams", false, false},
	} {
		t.Run(auth.name, func(t *testing.T) {
			t.Parallel()
			subs := allMyOrders()
			for _, s := range subs {
				s.Authenticated = auth.flaggedAuthenticated
			}
			// MyOrdersChannel leaves each asset's defaults in place, so they are taken out again to leave each generator
			// only MyOrdersChannel to consider
			myOrdersEx := setup(t, subs)
			myOrdersEx.Features.Subscriptions = subs
			myOrdersEx.Websocket.SetCanUseAuthenticatedEndpoints(auth.canUseAuthenticated)
			for _, tc := range []struct {
				name     string
				generate func() (subscription.List, error)
			}{
				{"spot", myOrdersEx.generateSubscriptions},
				{"USDⓈ-M /public", myOrdersEx.generateUSDTMarginedFuturesPublicSubscriptions},
				{"USDⓈ-M /market", myOrdersEx.generateUSDTMarginedFuturesMarketSubscriptions},
				{"COIN-M", myOrdersEx.generateCoinMarginedFuturesSubscriptions},
				{"options /public", myOrdersEx.generateOptionsPublicSubscriptions},
				{"options /market", myOrdersEx.generateOptionsMarketSubscriptions},
			} {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					subs, err := tc.generate()
					require.NoErrorf(t, err, "%s generation must not error with only MyOrdersChannel configured", tc.name)
					assert.Emptyf(t, subs, "%s generation should generate no streams with only MyOrdersChannel configured", tc.name)
				})
			}
		})
	}

	// MyOrdersChannel has no stream of its own, so beside it a derivatives asset streams its defaults where nothing else
	// configures the asset, and exactly what else does where something does
	spotTicker := &subscription.Subscription{Enabled: true, Asset: asset.Spot, Channel: subscription.TickerChannel}
	for _, tc := range []struct {
		name string
		subs subscription.List
		exp  map[string]int // streams of the generators that do not stream their defaults
	}{
		{"my orders of USDⓈ-M", subscription.List{spotTicker, {Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}}, nil},
		{"my orders of USDⓈ-M not flagged as authenticated", subscription.List{spotTicker, {Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel}}, nil},
		{"my orders of COIN-M", subscription.List{spotTicker, {Enabled: true, Asset: asset.CoinMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}}, nil},
		{"my orders of options", subscription.List{spotTicker, {Enabled: true, Asset: asset.Options, Channel: subscription.MyOrdersChannel, Authenticated: true}}, nil},
		{"my orders of all assets", subscription.List{spotTicker, {Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true}}, nil},
		{"my orders of all assets not flagged as authenticated", subscription.List{spotTicker, {Enabled: true, Asset: asset.All, Channel: subscription.MyOrdersChannel}}, nil},
		{"disabled my orders of all assets", subscription.List{spotTicker, {Enabled: false, Asset: asset.All, Channel: subscription.MyOrdersChannel, Authenticated: true}}, nil},
		{
			"my orders beside a USDⓈ-M ticker",
			subscription.List{spotTicker, {Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel}, {Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}},
			map[string]int{"USDⓈ-M /public": 0, "USDⓈ-M /market": 1},
		},
		{
			"my orders beside a disabled USDⓈ-M ticker",
			subscription.List{spotTicker, {Enabled: false, Asset: asset.USDTMarginedFutures, Channel: subscription.TickerChannel}, {Enabled: true, Asset: asset.USDTMarginedFutures, Channel: subscription.MyOrdersChannel, Authenticated: true}},
			map[string]int{"USDⓈ-M /public": 0, "USDⓈ-M /market": 0},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := setup(t, tc.subs)
			for _, g := range []struct {
				name     string
				generate func() (subscription.List, error)
				defaults int
			}{
				{"USDⓈ-M /public", ex.generateUSDTMarginedFuturesPublicSubscriptions, 1},
				{"USDⓈ-M /market", ex.generateUSDTMarginedFuturesMarketSubscriptions, 3},
				{"COIN-M", ex.generateCoinMarginedFuturesSubscriptions, 4},
				{"options /public", ex.generateOptionsPublicSubscriptions, 3},
				{"options /market", ex.generateOptionsMarketSubscriptions, 1},
			} {
				t.Run(g.name, func(t *testing.T) {
					t.Parallel()
					exp, ok := tc.exp[g.name]
					if !ok {
						exp = g.defaults
					}
					subs, err := g.generate()
					require.NoErrorf(t, err, "%s generation must not error", g.name)
					assert.Lenf(t, subs, exp, "%s should stream its defaults beside MyOrdersChannel, or only what else configures it", g.name)
				})
			}
		})
	}

	t.Run("my orders only still subscribes each private user data stream", func(t *testing.T) {
		t.Parallel()
		ex := setup(t, allMyOrders())
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		for _, tc := range []struct {
			a        asset.Item
			generate func() (subscription.List, error)
		}{
			{asset.USDTMarginedFutures, ex.generateUSDTMarginedFuturesUserDataSubscriptions},
			{asset.CoinMarginedFutures, ex.generateCoinMarginedFuturesUserDataSubscriptions},
			{asset.Options, ex.generateOptionsUserDataSubscriptions},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				subs, err := tc.generate()
				require.NoErrorf(t, err, "%s user data generation must not error", tc.a)
				testsubs.EqualLists(t, subscription.List{{Channel: userDataStreamChannel, Asset: tc.a, Authenticated: true}}, subs)
			})
		}
	})
}

func TestDerivativesStreamName(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		a       asset.Item
		sub     *subscription.Subscription
		exp     string
		wantErr error
	}{
		{name: "USDⓈ-M ticker", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "ticker"},
		{name: "options ticker", a: asset.Options, sub: &subscription.Subscription{Channel: subscription.TickerChannel}, exp: "optionTicker"},
		{name: "COIN-M trades", a: asset.CoinMarginedFutures, sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "aggTrade"},
		{name: "options trades", a: asset.Options, sub: &subscription.Subscription{Channel: subscription.AllTradesChannel}, exp: "optionTrade"},
		{name: "USDⓈ-M daily candles", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneDay}, exp: "kline_1d"},
		{name: "COIN-M monthly candles", a: asset.CoinMarginedFutures, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneMonth}, exp: "kline_1M"},
		{name: "options weekly candles", a: asset.Options, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.OneWeek}, exp: "kline_1w"},
		{name: "options eight hour candles", a: asset.Options, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.EightHour}, wantErr: kline.ErrUnsupportedInterval},
		{name: "USDⓈ-M one second candles", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.ThousandMilliseconds}, wantErr: kline.ErrUnsupportedInterval},
		{name: "USDⓈ-M order book", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, exp: "depth"},
		{name: "USDⓈ-M order book at 100ms", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.HundredMilliseconds}, exp: "depth@100ms"},
		{name: "COIN-M 20 levels at 500ms", a: asset.CoinMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.FiveHundredMilliseconds, Levels: 20}, exp: "depth20@500ms"},
		{name: "USDⓈ-M 5 levels", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 5}, exp: "depth5"},
		{name: "options 10 levels at 500ms", a: asset.Options, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.FiveHundredMilliseconds, Levels: 10}, exp: "depth10@500ms"},
		{name: "options order book without a speed", a: asset.Options, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel}, wantErr: errUnsupportedStreamOption},
		{name: "USDⓈ-M order book at 1m", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Interval: kline.OneMin}, wantErr: errUnsupportedStreamOption},
		{name: "USDⓈ-M 50 levels", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.OrderbookChannel, Levels: 50}, wantErr: errUnsupportedStreamOption},
		{name: "USDⓈ-M my orders", a: asset.USDTMarginedFutures, sub: &subscription.Subscription{Channel: subscription.MyOrdersChannel}, wantErr: subscription.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.derivativesStreamName(tc.a, tc.sub)
			if tc.wantErr != nil {
				assert.ErrorIsf(t, err, tc.wantErr, "derivativesStreamName should reject %s %s", tc.a, tc.sub.Channel)
				return
			}
			require.NoErrorf(t, err, "derivativesStreamName must not error for %s %s", tc.a, tc.sub.Channel)
			assert.Equalf(t, tc.exp, got, "derivativesStreamName should name %s %s", tc.a, tc.sub.Channel)
		})
	}
	futuresCandles := map[kline.Interval]string{
		kline.OneMin: "kline_1m", kline.ThreeMin: "kline_3m", kline.FiveMin: "kline_5m", kline.FifteenMin: "kline_15m", kline.ThirtyMin: "kline_30m",
		kline.OneHour: "kline_1h", kline.TwoHour: "kline_2h", kline.FourHour: "kline_4h", kline.SixHour: "kline_6h", kline.EightHour: "kline_8h",
		kline.TwelveHour: "kline_12h", kline.OneDay: "kline_1d", kline.ThreeDay: "kline_3d", kline.OneWeek: "kline_1w", kline.OneMonth: "kline_1M",
	}
	for _, tc := range []struct {
		name  string
		a     asset.Item
		names map[kline.Interval]string
	}{
		{"USDⓈ-M", asset.USDTMarginedFutures, futuresCandles},
		{"COIN-M", asset.CoinMarginedFutures, futuresCandles},
		{"options", asset.Options, map[kline.Interval]string{
			kline.OneMin: "kline_1m", kline.ThreeMin: "kline_3m", kline.FiveMin: "kline_5m", kline.FifteenMin: "kline_15m", kline.ThirtyMin: "kline_30m",
			kline.OneHour: "kline_1h", kline.TwoHour: "kline_2h", kline.FourHour: "kline_4h", kline.SixHour: "kline_6h", kline.TwelveHour: "kline_12h",
			kline.OneDay: "kline_1d", kline.ThreeDay: "kline_3d", kline.OneWeek: "kline_1w",
		}},
	} {
		for interval, exp := range tc.names {
			t.Run(tc.name+" candles at "+interval.Short(), func(t *testing.T) {
				t.Parallel()
				got, err := e.derivativesStreamName(tc.a, &subscription.Subscription{Channel: subscription.CandlesChannel, Interval: interval})
				require.NoErrorf(t, err, "derivativesStreamName must not error for %s candles at %s", tc.a, interval)
				assert.Equalf(t, exp, got, "derivativesStreamName should name %s candles at %s", tc.a, interval)
			})
		}
	}
}

func TestSubscribeDerivatives(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		switch req.Params[0] {
		case "btcusdt@rejected":
			return fmt.Appendf(nil, `{"error":{"code":2,"msg":"Invalid request: unknown variant"},"id":%d}`, req.ID)
		case "btcusdt@unexpected":
			return fmt.Appendf(nil, `{"result":true,"id":%d}`, req.ID)
		}
		return nil
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
	require.NoError(t, err, "GetConnection must not error")
	// The connection was set up without subscriptions, so the subscriptions come from the defaults
	ex.Features.Subscriptions = defaultSubscriptions.Clone()
	subs, err := ex.generateUSDTMarginedFuturesMarketSubscriptions()
	require.NoError(t, err, "generateUSDTMarginedFuturesMarketSubscriptions must not error")

	require.NoError(t, ex.SubscribeDerivatives(t.Context(), conn, subs), "SubscribeDerivatives must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.SubscribedState, s.State(), "%s should be subscribed", s.QualifiedChannel)
	}
	require.NoError(t, ex.UnsubscribeDerivatives(t.Context(), conn, subs), "UnsubscribeDerivatives must not error")
	for _, s := range subs {
		assert.Equalf(t, subscription.UnsubscribedState, s.State(), "%s should be unsubscribed", s.QualifiedChannel)
	}
	requests := recorder.recorded()
	require.Len(t, requests, 2, "SubscribeDerivatives and UnsubscribeDerivatives must send one request each")
	for i := range requests {
		slices.SortFunc(requests[i].Params, func(a, b any) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
	}
	exp := []DerivativesStreamRequest{
		{Method: "SUBSCRIBE", Params: []any{"btcusdt@aggTrade", "btcusdt@kline_1m", "btcusdt@ticker"}, ID: requests[0].ID},
		{Method: "UNSUBSCRIBE", Params: []any{"btcusdt@aggTrade", "btcusdt@kline_1m", "btcusdt@ticker"}, ID: requests[1].ID},
	}
	assert.Equal(t, exp, requests, "the requests should carry the stream names")
	assert.NotEqual(t, requests[0].ID, requests[1].ID, "each request should have its own ID")

	rejected := subscription.List{{Channel: "rejected", Asset: asset.USDTMarginedFutures, QualifiedChannel: "btcusdt@rejected"}}
	err = ex.SubscribeDerivatives(t.Context(), conn, rejected)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "SubscribeDerivatives should report a subscription failure")
	assert.ErrorIs(t, err, errStreamRequestRejected, "SubscribeDerivatives should report the rejection")
	assert.Nil(t, ex.Websocket.GetSubscription(rejected[0]), "a rejected subscription should not be stored")

	unexpected := subscription.List{{Channel: "unexpected", Asset: asset.USDTMarginedFutures, QualifiedChannel: "btcusdt@unexpected"}}
	assert.ErrorIs(t, ex.SubscribeDerivatives(t.Context(), conn, unexpected), errUnexpectedStreamResult, "SubscribeDerivatives should reject a result other than null")
}

func TestUnsubscribeDerivatives(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		if req.Method == wsUnsubscribeMethod {
			return fmt.Appendf(nil, `{"error":{"code":2,"msg":"Invalid request: unknown variant"},"id":%d}`, req.ID)
		}
		return nil
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
	require.NoError(t, err, "GetConnection must not error")
	subs := subscription.List{{Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures, QualifiedChannel: "btcusdt@ticker"}}
	require.NoError(t, ex.SubscribeDerivatives(t.Context(), conn, subs), "SubscribeDerivatives must not error")
	err = ex.UnsubscribeDerivatives(t.Context(), conn, subs)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "UnsubscribeDerivatives should report a subscription failure")
	assert.ErrorIs(t, err, errStreamRequestRejected, "UnsubscribeDerivatives should report the rejection")
	requests := recorder.recorded()
	require.Len(t, requests, 2, "SubscribeDerivatives and UnsubscribeDerivatives must send one request each")
	assert.Equal(t, DerivativesStreamRequest{Method: "UNSUBSCRIBE", Params: []any{"btcusdt@ticker"}, ID: requests[1].ID}, requests[1], "UnsubscribeDerivatives should name the stream")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip(liveStreamsSkip)
		}
		ex := newLiveDerivativesExchange(t, asset.USDTMarginedFutures)
		connectLiveDerivatives(t, ex, asset.USDTMarginedFutures, currency.NewBTCUSDT(), subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures}})
		conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
		require.NoError(t, err, "GetConnection must not error")
		subs := liveTickerSubscriptions(t, ex, asset.USDTMarginedFutures)
		require.NoError(t, ex.UnsubscribeDerivatives(t.Context(), conn, subs), "UnsubscribeDerivatives must not error")
		assert.Equal(t, subscription.UnsubscribedState, subs[0].State(), "the ticker should be unsubscribed")
		assert.Nil(t, ex.Websocket.GetSubscription(subs[0]), "the ticker should no longer be stored")
		streams, err := ex.ListSubscriptions(t.Context(), conn)
		require.NoError(t, err, "ListSubscriptions must not error")
		assert.Empty(t, streams, "Binance should list no stream after the unsubscription")
	})
}

func TestListSubscriptions(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		return fmt.Appendf(nil, `{"result":["btcusdt@aggTrade","btcusdt@kline_1m"],"id":%d}`, req.ID)
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
	require.NoError(t, err, "GetConnection must not error")
	result, err := ex.ListSubscriptions(t.Context(), conn)
	require.NoError(t, err, "ListSubscriptions must not error")
	assert.Equal(t, []string{"btcusdt@aggTrade", "btcusdt@kline_1m"}, result, "ListSubscriptions should decode the stream names")
	requests := recorder.recorded()
	require.Len(t, requests, 1, "ListSubscriptions must send one request")
	assert.Equal(t, DerivativesStreamRequest{Method: "LIST_SUBSCRIPTIONS", ID: requests[0].ID}, requests[0], "ListSubscriptions should send a request without params")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip(liveStreamsSkip)
		}
		ex := newLiveDerivativesExchange(t, asset.USDTMarginedFutures)
		connectLiveDerivatives(t, ex, asset.USDTMarginedFutures, currency.NewBTCUSDT(), subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.USDTMarginedFutures}})
		conn, err := ex.Websocket.GetConnection(usdtmMarketFilter)
		require.NoError(t, err, "GetConnection must not error")
		streams, err := ex.ListSubscriptions(t.Context(), conn)
		require.NoError(t, err, "ListSubscriptions must not error")
		assert.Equal(t, liveTickerSubscriptions(t, ex, asset.USDTMarginedFutures).QualifiedChannels(), streams, "ListSubscriptions should list the subscribed stream")
	})
}

func TestSetProperty(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
		if req.Params[0] == "unknown" {
			return fmt.Appendf(nil, `{"error":{"code":0,"msg":"Unknown property"},"id":%d}`, req.ID)
		}
		return nil
	}}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(coinmFilter)
	require.NoError(t, err, "GetConnection must not error")
	require.NoError(t, ex.SetProperty(t.Context(), conn, "combined", true), "SetProperty must not error")
	requests := recorder.recorded()
	require.Len(t, requests, 1, "SetProperty must send one request")
	assert.Equal(t, DerivativesStreamRequest{Method: "SET_PROPERTY", Params: []any{"combined", true}, ID: requests[0].ID}, requests[0], "SetProperty should send the property and value")
	assert.ErrorIs(t, ex.SetProperty(t.Context(), conn, "unknown", true), errStreamRequestRejected, "SetProperty should report a rejected property")

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip(liveStreamsSkip)
		}
		ex := newLiveDerivativesExchange(t, asset.CoinMarginedFutures)
		connectLiveDerivatives(t, ex, asset.CoinMarginedFutures, currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter), subscription.List{{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.CoinMarginedFutures}})
		conn, err := ex.Websocket.GetConnection(coinmFilter)
		require.NoError(t, err, "GetConnection must not error")
		assert.NoError(t, ex.SetProperty(t.Context(), conn, "combined", true), "SetProperty should keep the connection's streams combined")
	})
}

func TestGenerateUserDataSubscriptions(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	generators := map[asset.Item]func() (subscription.List, error){
		asset.USDTMarginedFutures: ex.generateUSDTMarginedFuturesUserDataSubscriptions,
		asset.CoinMarginedFutures: ex.generateCoinMarginedFuturesUserDataSubscriptions,
		asset.Options:             ex.generateOptionsUserDataSubscriptions,
	}
	// The cases share ex, whose authenticated use is turned on after them, so they run in order
	for a, generate := range generators {
		t.Run("unauthenticated "+a.String(), func(t *testing.T) {
			subs, err := generate()
			require.NoErrorf(t, err, "%s user data generation must not error", a)
			assert.Emptyf(t, subs, "%s user data should not be subscribed without authenticated websocket use", a)
		})
	}
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	// The cases share ex, which the statements after them change, so they run in order
	for a, generate := range generators {
		t.Run("authenticated "+a.String(), func(t *testing.T) {
			subs, err := generate()
			require.NoErrorf(t, err, "%s user data generation must not error", a)
			require.Lenf(t, subs, 1, "%s user data generation must return one subscription", a)
			testsubs.EqualLists(t, subscription.List{{Channel: userDataStreamChannel, Asset: a}}, subs)
			assert.Truef(t, subs[0].Authenticated, "%s user data subscription should be authenticated", a)
		})
	}
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	subs, err := ex.generateOptionsUserDataSubscriptions()
	require.NoError(t, err, "options user data generation must not error")
	assert.Empty(t, subs, "a disabled asset should not subscribe its user data")
	ex.SetCredentials(&accounts.Credentials{})
	subs, err = ex.generateUSDTMarginedFuturesUserDataSubscriptions()
	require.NoError(t, err, "USDⓈ-M user data generation must not error")
	assert.Empty(t, subs, "user data should not be subscribed without usable credentials")
}

func TestSubscribeUserData(t *testing.T) {
	t.Parallel()
	recorder := &streamRequestRecorder{}
	ex := newDerivativesWsTestExchange(t, recorder)
	// The cases share ex and the recorder, whose requests are checked in order after them, so they run in order
	for _, tc := range []struct {
		filter string
		a      asset.Item
	}{
		{usdtmPrivateFilter, asset.USDTMarginedFutures},
		{coinmPrivateFilter, asset.CoinMarginedFutures},
		{optionsPrivateFilter, asset.Options},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			conn, err := ex.Websocket.GetConnection(tc.filter)
			require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
			subs := subscription.List{{Channel: userDataStreamChannel, Asset: tc.a, Authenticated: true}}
			require.NoErrorf(t, ex.SubscribeUserData(t.Context(), conn, subs), "SubscribeUserData must not error for %s", tc.a)
			assert.Equalf(t, subscription.SubscribedState, subs[0].State(), "%s user data should be subscribed", tc.a)
			require.NoErrorf(t, ex.UnsubscribeUserData(t.Context(), conn, subs), "UnsubscribeUserData must not error for %s", tc.a)
			assert.Equalf(t, subscription.UnsubscribedState, subs[0].State(), "%s user data should be unsubscribed", tc.a)
		})
	}
	requests := recorder.recorded()
	require.Len(t, requests, 6, "each subscribe and unsubscribe must send one request")
	for i := range requests {
		exp := DerivativesStreamRequest{Method: "SUBSCRIBE", Params: []any{testListenKey}, ID: requests[i].ID}
		if i%2 == 1 {
			exp.Method = "UNSUBSCRIBE"
		}
		assert.Equal(t, exp, requests[i], "user data requests should name the listen key as the stream")
	}

	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	spot := subscription.List{{Channel: userDataStreamChannel, Asset: asset.Spot, Authenticated: true}}
	err = ex.SubscribeUserData(t.Context(), conn, spot)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "SubscribeUserData should report a subscription failure")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "SubscribeUserData should reject an asset without user data streams")
	assert.Nil(t, ex.Websocket.GetSubscription(spot[0]), "a failed user data subscription should not be stored")

	t.Run("resubscribed by the websocket manager", func(t *testing.T) {
		t.Parallel()
		ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
				return
			}
			_, err := fmt.Fprintf(w, `{"listenKey":%q}`, testListenKey)
			assert.NoError(t, err, "Write should not error")
		})
		recorder := &streamRequestRecorder{}
		useMockWebsocket(t, ex, &websocket.ConnectionSetup{
			MessageFilter:            usdtmPrivateFilter,
			Connector:                ex.WebsocketDerivativesConnect,
			Subscriber:               ex.SubscribeUserData,
			Unsubscriber:             ex.UnsubscribeUserData,
			Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
			ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
			SubscriptionsNotRequired: true,
		}, recorder.handle)
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
		require.NoError(t, err, "GetConnection must not error")
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{s}), "SubscribeUserData must not error")
		// The manager unsubscribes the subscription and subscribes the same one again
		require.NoError(t, ex.Websocket.ResubscribeToChannel(t.Context(), conn, s), "ResubscribeToChannel must not error")
		recorded := recorder.recorded()
		methods := make([]string, 0, len(recorded))
		for _, req := range recorded {
			methods = append(methods, req.Method)
		}
		assert.Equal(t, []string{"SUBSCRIBE", "UNSUBSCRIBE", "SUBSCRIBE"}, methods, "the resubscription should subscribe the stream again")
		assert.Equal(t, subscription.SubscribedState, s.State(), "the resubscribed stream should be subscribed")
	})

	t.Run("refused as a duplicate while the stream's failed subscription is tried again", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the first subscription's and 2 the retry's, made once the duplicate is refused
		v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }}
		p := userDataRetryProducts[0]
		ex := newUserDataRetryExchange(t, p, v, 200*time.Millisecond)
		require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
		conn, err := ex.Websocket.GetConnection(p.filter)
		require.NoError(t, err, "GetConnection must not error")
		failed := ex.Websocket.GetSubscription(&subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a})
		require.NotNil(t, failed, "the failed subscription must stay registered")
		duplicate := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a, Authenticated: true}
		assert.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{duplicate}), subscription.ErrDuplicate, "SubscribeUserData should refuse a second subscription to the stream")
		ex.userDataRetry.mu.Lock()
		_, retried := ex.userDataRetry.running[duplicate]
		ex.userDataRetry.mu.Unlock()
		assert.False(t, retried, "the refused subscription should not be tried again")
		require.Eventually(t, func() bool { return failed.State() == subscription.SubscribedState }, 30*time.Second, time.Millisecond, "the retry must subscribe the failed subscription")
		assert.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry should end once the stream is subscribed")
		assert.Same(t, failed, ex.Websocket.GetSubscription(duplicate), "the failed subscription should stay registered")
		assert.Equal(t, []string{"SUBSCRIBE"}, v.methods(), "the stream should be subscribed once")
	})

	t.Run("tried again after its caller's context ends, and recorded against its connection once subscribed", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the subscription's and 2 the retry's
		v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }}
		ex, conn := newUserDataVenueConnection(t, v)
		ex.userDataRetry.delay = 50 * time.Millisecond
		ex.userDataRetry.keepalive = 20 * time.Millisecond
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		ctx, cancel := context.WithCancel(t.Context())
		require.ErrorIs(t, ex.SubscribeUserData(ctx, conn, subscription.List{s}), websocket.ErrSubscriptionFailure, "SubscribeUserData must report the failed subscription")
		cancel()
		require.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 30*time.Second, time.Millisecond, "the retry must subscribe the stream after its caller's context ends")
		// The retry records the subscription once it is marked subscribed
		assert.Eventually(t, func() bool { return conn.Subscriptions().Get(s) == s }, 30*time.Second, time.Millisecond, "the retry should record the subscription it made against its connection")
		assert.Eventually(t, func() bool { return v.keepalives.Load() > 0 }, 30*time.Second, time.Millisecond, "the listen key should be kept alive after the caller's context ends")
		assert.Equal(t, []string{"SUBSCRIBE"}, v.methods(), "the stream should be subscribed once")
	})

	t.Run("tried again after a resubscription fails", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the resubscription's and 2 the retry's
		v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }}
		ex, conn := newUserDataVenueConnection(t, v)
		ex.userDataRetry.delay = 20 * time.Millisecond
		// The websocket manager's resubscription marks its subscription resubscribing and passes it back unregistered once
		// it has unsubscribed the stream through the connection's record of it
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, s.SetState(subscription.ResubscribingState), "SetState must not error")
		require.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{s}), websocket.ErrSubscriptionFailure, "SubscribeUserData must report the failed resubscription")
		require.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 30*time.Second, time.Millisecond, "the retry must subscribe the stream whose resubscription failed")
		assert.Same(t, s, ex.Websocket.GetSubscription(s), "the subscription should stay registered")
		assert.Equal(t, []string{"SUBSCRIBE"}, v.methods(), "the stream should be subscribed once")
	})

	t.Run("not tried again once unsubscribed during its failing request", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the subscription's, held until the unsubscription, whose request is 2, has returned
		held := newHeldRequest()
		v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }, heldKeys: map[int32]*heldRequest{1: held}}
		ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
				return
			}
			n := v.listenKeys.Add(1)
			if h := v.heldKeys[n]; h != nil {
				h.hold()
			}
			writeListenKeyReply(t, w, v.refuseKey(n))
		})
		ex.userDataRetry.delay = time.Millisecond
		useMockWebsocket(t, ex, &websocket.ConnectionSetup{
			MessageFilter:            usdtmPrivateFilter,
			Connector:                ex.WebsocketDerivativesConnect,
			Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
			ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
			SubscriptionsNotRequired: true,
		}, v.recorder.handle)
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
		require.NoError(t, err, "GetConnection must not error")
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		subscribed := make(chan error, 1)
		go func() { subscribed <- ex.SubscribeUserData(t.Context(), conn, subscription.List{s}) }()
		select {
		case <-held.entered:
		case <-time.After(30 * time.Second):
			require.FailNow(t, "the subscription must request its listen key")
		}
		require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must not error")
		close(held.release)
		select {
		case err := <-subscribed:
			require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "SubscribeUserData must report its refused listen key request")
		case <-time.After(30 * time.Second):
			require.FailNow(t, "SubscribeUserData must return")
		}
		assert.False(t, userDataRetrying(ex, asset.USDTMarginedFutures), "a subscription unsubscribed during its failing request should not be tried again")
		assert.Never(t, func() bool { return ex.Websocket.GetSubscription(s) != nil || v.listenKeys.Load() != 2 }, 100*time.Millisecond, time.Millisecond, "the stream unsubscribed during the failing request should stay unsubscribed")
		assert.Equal(t, []string{"UNSUBSCRIBE"}, v.methods(), "only the unsubscription should reach the stream")
	})

	t.Run("a retry's request under way is cancelled when the websocket shuts down", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the subscription's and 2 the retry's, held until the case ends
		held := newHeldRequest()
		v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }, heldKeys: map[int32]*heldRequest{2: held}}
		ex, conn := newUserDataVenueConnection(t, v)
		ex.userDataRetry.delay = time.Millisecond
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{s}), websocket.ErrSubscriptionFailure, "SubscribeUserData must report the failed subscription")
		waitFor(t, held.entered, "the retry must request its listen key")
		shutdown := make(chan error, 1)
		go func() { shutdown <- ex.Websocket.Shutdown() }()
		select {
		case err := <-shutdown:
			require.NoError(t, err, "Shutdown must not error")
		case <-time.After(10 * time.Second):
			require.FailNow(t, "Shutdown must not wait for the retry's request under way")
		}
		close(held.release)
		assert.Empty(t, v.methods(), "the cancelled retry should subscribe nothing")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip(liveStreamsSkip)
		}
		for _, tc := range liveUserDataConnections {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				ex := newLiveDerivativesExchange(t, tc.a)
				sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
				liveListenKeys.RLock()
				t.Cleanup(liveListenKeys.RUnlock)
				// Connecting with authenticated websocket use subscribes the private connection through SubscribeUserData
				ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
				connectLiveDerivatives(t, ex, tc.a, currency.EMPTYPAIR, subscription.List{})
				conn, err := ex.Websocket.GetConnection(tc.filter)
				require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
				assert.Equalf(t, subscription.SubscribedState, liveUserDataSubscription(t, ex, tc.a).State(), "%s user data should be subscribed", tc.a)
				listenKey, err := ex.startUserDataStream(t.Context(), tc.a)
				require.NoErrorf(t, err, "startUserDataStream must return the %s account's listen key", tc.a)
				streams, err := ex.ListSubscriptions(t.Context(), conn)
				require.NoError(t, err, "ListSubscriptions must not error")
				assert.Equalf(t, []string{listenKey}, streams, "Binance should list the %s account's listen key as the subscribed stream", tc.a)
			})
		}
	})
}

func TestUnsubscribeUserData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesWsTestExchange(t, &streamRequestRecorder{})
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	spot := subscription.List{{Channel: userDataStreamChannel, Asset: asset.Spot, Authenticated: true}}
	// The unsubscription follows a refused subscription, which leaves the subscription unsubscribed
	require.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, spot), asset.ErrNotSupported, "SubscribeUserData must refuse an asset without user data streams")
	err = ex.UnsubscribeUserData(t.Context(), conn, spot)
	assert.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "UnsubscribeUserData should report a subscription failure")
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UnsubscribeUserData should reject an asset without user data streams")

	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		for _, tc := range []struct {
			name string
			// refused has Binance refuse the UNSUBSCRIBE rather than the listen key request failing
			refused bool
			methods []string
		}{
			{name: "fails at the listen key request", methods: []string{"SUBSCRIBE"}},
			{name: "is refused by Binance", refused: true, methods: []string{"SUBSCRIBE", "UNSUBSCRIBE"}},
		} {
			t.Run("ends a "+a.String()+" subscription whose unsubscription "+tc.name, func(t *testing.T) {
				t.Parallel()
				var listenKeys atomic.Int32
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if !assert.Contains(t, []string{"/fapi/v1/listenKey", "/dapi/v1/listenKey", "/eapi/v1/listenKey"}, r.URL.Path, "only listen keys should be requested") {
						return
					}
					// The second listen key request is the unsubscription's
					writeListenKeyReply(t, w, !tc.refused && listenKeys.Add(1) == 2)
				})
				recorder := &streamRequestRecorder{reply: func(req *DerivativesStreamRequest) []byte {
					if tc.refused && req.Method == wsUnsubscribeMethod {
						return fmt.Appendf(nil, `{"error":{"code":2,"msg":"Invalid request: unknown variant"},"id":%d}`, req.ID)
					}
					return nil
				}}
				useMockWebsocket(t, ex, &websocket.ConnectionSetup{
					MessageFilter:            usdtmPrivateFilter,
					Connector:                ex.WebsocketDerivativesConnect,
					Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
					ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
					ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
					SubscriptionsNotRequired: true,
				}, recorder.handle)
				require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
				conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
				require.NoError(t, err, "GetConnection must not error")
				s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: a, Authenticated: true}
				require.NoError(t, ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s), "subscribeUserDataKeptAlive must not error")
				err = ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s})
				require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "UnsubscribeUserData must report the failed unsubscription")
				assert.Equal(t, subscription.UnsubscribedState, s.State(), "the failed unsubscription should end the subscription")
				assert.Nil(t, ex.Websocket.GetSubscription(s), "the failed unsubscription should remove the subscription")
				methods := make([]string, 0, len(tc.methods))
				for _, req := range recorder.recorded() {
					methods = append(methods, req.Method)
				}
				assert.Equal(t, tc.methods, methods, "the unsubscription should send an UNSUBSCRIBE only once it has the listen key")
			})
		}
	}

	for _, tc := range []struct {
		name string
		// resubscribe fails the websocket manager's resubscription rather than a flush of the disabled asset
		resubscribe bool
	}{
		{name: "failed by a flush of the disabled asset"},
		{name: "failed by the websocket manager's resubscription", resubscribe: true},
	} {
		t.Run("ends a subscription whose unsubscription is "+tc.name, func(t *testing.T) {
			t.Parallel()
			var listenKeys, keepalives atomic.Int32
			var wanted atomic.Bool
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
					return
				}
				if r.Method == http.MethodPut {
					// Keepalives before the flush wanting the stream again, and those after the fourth since, wait for the
					// case to end, so the test server's request allowance holds however long the case is held up
					if !wanted.Load() || keepalives.Add(1) > 4 {
						<-r.Context().Done()
						return
					}
					writeListenKeyReply(t, w, false)
					return
				}
				// The second listen key request is the unsubscription's
				writeListenKeyReply(t, w, listenKeys.Add(1) == 2)
			})
			ex.userDataRetry.keepalive = 20 * time.Millisecond
			recorder := &streamRequestRecorder{}
			useMockWebsocket(t, ex, &websocket.ConnectionSetup{
				MessageFilter:         usdtmPrivateFilter,
				Connector:             ex.WebsocketDerivativesConnect,
				GenerateSubscriptions: ex.generateUSDTMarginedFuturesUserDataSubscriptions,
				Subscriber:            ex.SubscribeUserData,
				Unsubscriber:          ex.UnsubscribeUserData,
				Handler:               ex.wsHandleUSDTMarginedFuturesUserData,
				ResponseCheckTimeout:  ex.Config.WebsocketResponseCheckTimeout,
				ResponseMaxLimit:      ex.Config.WebsocketResponseMaxLimit,
			}, recorder.handle)
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
			require.NoError(t, err, "GetConnection must not error")
			key := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures}
			s := ex.Websocket.GetSubscription(key)
			require.NotNil(t, s, "connecting must register the user data subscription")
			if tc.resubscribe {
				err = ex.Websocket.ResubscribeToChannel(t.Context(), conn, s)
			} else {
				require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, false), "SetAssetEnabled must not error")
				err = ex.Websocket.FlushChannels(t.Context())
			}
			require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "the failed unsubscription must be reported")
			assert.Equal(t, subscription.UnsubscribedState, s.State(), "the failed unsubscription should end the subscription")
			require.Nil(t, ex.Websocket.GetSubscription(key), "the failed unsubscription must remove the subscription")
			if !tc.resubscribe {
				assert.Nil(t, conn.Subscriptions().Get(key), "the flush should release the subscription from its connection")
				require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.USDTMarginedFutures, true), "SetAssetEnabled must not error")
			}
			wanted.Store(true)
			err = ex.Websocket.FlushChannels(t.Context())
			if tc.resubscribe {
				// The manager's resubscription returns without releasing the subscription from its connection
				assert.ErrorIs(t, err, subscription.ErrDuplicate, "the flush should report the subscription the resubscription left on its connection")
				require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "the flush after it must not error")
			} else {
				require.NoError(t, err, "the flush wanting the stream again must not error")
			}
			fresh := ex.Websocket.GetSubscription(key)
			require.NotNil(t, fresh, "the flush must subscribe the stream afresh")
			assert.NotSame(t, s, fresh, "the flush should subscribe a new subscription")
			assert.Equal(t, subscription.SubscribedState, fresh.State(), "the stream should be subscribed again")
			assert.Eventually(t, func() bool { return keepalives.Load() >= 2 }, 30*time.Second, time.Millisecond, "the listen key of the stream subscribed again should be kept alive")
			require.NoError(t, ex.processListenKeyExpired(t.Context(), conn, []byte(`{"e":"listenKeyExpired","E":1791284812500,"listenKey":"expired"}`), asset.USDTMarginedFutures), "the expiry must be handled")
			assert.Eventually(t, func() bool { return len(recorder.recorded()) == 3 }, 30*time.Second, time.Millisecond, "the renewal after the expiry should subscribe the stream's new listen key")
			methods := make([]string, 0, 3)
			for _, req := range recorder.recorded() {
				methods = append(methods, req.Method)
			}
			assert.Equal(t, []string{"SUBSCRIBE", "SUBSCRIBE", "SUBSCRIBE"}, methods, "the stream should be subscribed when connecting, by the flush wanting it and after its listen key expires")
		})
	}

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip(liveStreamsSkip)
		}
		for _, tc := range liveUserDataConnections {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				ex := newLiveDerivativesExchange(t, tc.a)
				sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
				liveListenKeys.RLock()
				t.Cleanup(liveListenKeys.RUnlock)
				ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
				connectLiveDerivatives(t, ex, tc.a, currency.EMPTYPAIR, subscription.List{})
				conn, err := ex.Websocket.GetConnection(tc.filter)
				require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
				s := liveUserDataSubscription(t, ex, tc.a)
				require.NoErrorf(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must not error for %s", tc.a)
				assert.Equalf(t, subscription.UnsubscribedState, s.State(), "%s user data should be unsubscribed", tc.a)
				assert.Nilf(t, ex.Websocket.GetSubscription(s), "%s user data should no longer be stored", tc.a)
				streams, err := ex.ListSubscriptions(t.Context(), conn)
				require.NoError(t, err, "ListSubscriptions must not error")
				assert.Emptyf(t, streams, "Binance should list no stream after the %s unsubscription", tc.a)
			})
		}
	})
}

func TestSubscribeUserDataRetried(t *testing.T) { //nolint:tparallel // the cases share ex and its recorder, so they run in order while the test runs in parallel
	t.Parallel()
	// A user data stream subscription failing as its private connection connects stays registered, and recorded against
	// the connection by the websocket manager, so that a flush sees the connection in use, and is tried again in the
	// background, kept alive as a first subscription is
	var listenKeys, keepalives atomic.Int32
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
			return
		}
		if r.Method == http.MethodPut {
			// Keepalives after the first wait for the test to end, so the test server's request allowance holds however
			// long the test is held up
			if keepalives.Add(1) > 1 {
				<-r.Context().Done()
				return
			}
		} else if listenKeys.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"code":-1001,"msg":"Internal error; unable to process your request. Please try again."}`))
			assert.NoError(t, err, "Write should not error")
			return
		}
		_, err := fmt.Fprintf(w, `{"listenKey":%q}`, testListenKey)
		assert.NoError(t, err, "Write should not error")
	})
	ex.userDataRetry.delay = time.Millisecond
	ex.userDataRetry.keepalive = 20 * time.Millisecond
	recorder := &streamRequestRecorder{}
	useMockWebsocket(t, ex, &websocket.ConnectionSetup{
		MessageFilter:         usdtmPrivateFilter,
		Connector:             ex.WebsocketDerivativesConnect,
		GenerateSubscriptions: ex.generateUSDTMarginedFuturesUserDataSubscriptions,
		Subscriber:            ex.SubscribeUserData,
		Unsubscriber:          ex.UnsubscribeUserData,
		Handler:               ex.wsHandleUSDTMarginedFuturesUserData,
		ResponseCheckTimeout:  ex.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:      ex.Config.WebsocketResponseMaxLimit,
	}, recorder.handle)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	sub := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures}
	failed := ex.Websocket.GetSubscription(sub)
	require.NotNil(t, failed, "the failed subscription must stay registered while it is tried again")
	assert.Same(t, failed, conn.Subscriptions().Get(sub), "the failed subscription should be recorded against its connection")
	require.Eventually(t, func() bool { return failed.State() == subscription.SubscribedState }, 30*time.Second, time.Millisecond, "the failed subscription must be made again")
	assert.Same(t, failed, ex.Websocket.GetSubscription(sub), "the subscription made again should stay registered")
	assert.Same(t, failed, conn.Subscriptions().Get(sub), "the subscription made again should stay recorded against its connection")
	assert.Eventually(t, func() bool { return keepalives.Load() > 0 }, 30*time.Second, time.Millisecond, "the listen key of the subscription made again should be kept alive")
	assert.Equal(t, int32(2), listenKeys.Load(), "the listen key should be requested again once")
	requests := recorder.recorded()
	require.Len(t, requests, 1, "the stream must be subscribed once")
	assert.Equal(t, DerivativesStreamRequest{Method: "SUBSCRIBE", Params: []any{testListenKey}, ID: requests[0].ID}, requests[0], "the retry should subscribe the listen key")
	// Unsubscribing ends the keepalives, which would otherwise outlast the test server's request allowance
	require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{failed}), "UnsubscribeUserData must not error")

	// A retry stops without asking again for a stream whose failed subscription another has replaced or is no longer
	// registered
	retryTo := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.CoinMarginedFutures, Authenticated: true}
	require.NoError(t, ex.Websocket.AddSubscriptions(conn, retryTo), "AddSubscriptions must not error")
	ex.userDataRetry.mu.Lock()
	ex.userDataRetry.delay = 50 * time.Millisecond
	ex.userDataRetry.mu.Unlock()
	retrying := func(a asset.Item) bool { return userDataRetrying(ex, a) }
	// The cases share ex and the recorder, so they run in order
	for _, tc := range []struct {
		name string
		a    asset.Item
	}{
		{"subscribed since", asset.CoinMarginedFutures},
		{"no longer registered", asset.Options},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := len(recorder.recorded())
			failed := &subscription.Subscription{Channel: userDataStreamChannel, Asset: tc.a, Authenticated: true}
			ex.retryUserDataSubscription(t.Context(), conn, failed, failed.Asset, false, func(ctx context.Context) (bool, error) {
				return ex.resubscribeUserData(ctx, t.Context(), conn, failed)
			})
			assert.Eventuallyf(t, func() bool { return !retrying(tc.a) }, 30*time.Second, time.Millisecond, "the retry should stop for a stream %s", tc.name)
			assert.Lenf(t, recorder.recorded(), before, "a retry should not subscribe a stream %s", tc.name)
			assert.Emptyf(t, relayedPayloads(ex), "a retry should report nothing for a stream %s", tc.name)
		})
	}

	t.Run("waits while the stream is not wanted", func(t *testing.T) {
		failed := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.Websocket.AddSubscriptions(conn, failed), "AddSubscriptions must not error")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(false)
		before := len(recorder.recorded())
		ex.retryUserDataSubscription(t.Context(), conn, failed, failed.Asset, false, func(ctx context.Context) (bool, error) {
			return ex.resubscribeUserData(ctx, t.Context(), conn, failed)
		})
		assert.Never(t, func() bool { return !retrying(asset.USDTMarginedFutures) || len(recorder.recorded()) != before }, 200*time.Millisecond, time.Millisecond, "the retry should wait without subscribing while the stream is not wanted")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
		require.Eventually(t, func() bool { return failed.State() == subscription.SubscribedState }, 30*time.Second, time.Millisecond, "the retry must subscribe the stream once it is wanted again")
		assert.Len(t, recorder.recorded(), before+1, "the retry should subscribe the stream once")
		require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{failed}), "UnsubscribeUserData must not error")
	})

	// An asset without user data streams cannot be subscribed however often it is tried
	spot := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.Spot, Authenticated: true}
	require.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{spot}), asset.ErrNotSupported, "SubscribeUserData must reject an asset without user data streams")
	assert.Nil(t, ex.Websocket.GetSubscription(spot), "a subscription nothing tries again should not stay registered")
	assert.False(t, retrying(asset.Spot), "an asset without user data streams should not be tried again")
}

// userDataRetryProduct is a derivatives product whose private connection the user data retry cases set up as Setup
// does
type userDataRetryProduct struct {
	a        asset.Item
	filter   string
	keyPath  string
	generate func(*Exchange) (subscription.List, error)
	handle   func(*Exchange, context.Context, websocket.Connection, []byte) error
}

var userDataRetryProducts = []userDataRetryProduct{
	{asset.USDTMarginedFutures, usdtmPrivateFilter, "/fapi/v1/listenKey", (*Exchange).generateUSDTMarginedFuturesUserDataSubscriptions, (*Exchange).wsHandleUSDTMarginedFuturesUserData},
	{asset.CoinMarginedFutures, coinmPrivateFilter, "/dapi/v1/listenKey", (*Exchange).generateCoinMarginedFuturesUserDataSubscriptions, (*Exchange).wsHandleCoinMarginedFuturesUserData},
	{asset.Options, optionsPrivateFilter, "/eapi/v1/listenKey", (*Exchange).generateOptionsUserDataSubscriptions, (*Exchange).wsHandleOptionsUserData},
}

// heldRequest is a mock venue request held until release is closed; entered is closed once the request arrives
type heldRequest struct {
	entered, release chan struct{}
}

func newHeldRequest() *heldRequest {
	return &heldRequest{entered: make(chan struct{}), release: make(chan struct{})}
}

// waitFor requires c to be closed within 30 seconds, failing with msg otherwise
func waitFor(t *testing.T, c <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(30 * time.Second):
		require.FailNow(t, msg)
	}
}

// userDataRetryVenue answers a product's listen key requests and its private connection's stream requests, each numbered
// from 1, refusing those refuseKey and refuseStream name and holding those in heldKeys and heldStreams, and counts the
// keepalives it is sent
type userDataRetryVenue struct {
	refuseKey, refuseStream func(n int32) bool
	heldKeys, heldStreams   map[int32]*heldRequest
	// heldKeepalive holds the first keepalive, and refuseKeepalive refuses every keepalive
	heldKeepalive          *heldRequest
	refuseKeepalive        bool
	listenKeys, keepalives atomic.Int32
	streams                atomic.Int32
	recorder               streamRequestRecorder
}

// hold waits for a held request's release, giving up after 30 seconds so a failed case cannot hang its server
func (h *heldRequest) hold() {
	close(h.entered)
	select {
	case <-h.release:
	case <-time.After(30 * time.Second):
	}
}

// methods returns the methods of the stream requests the venue received, in order
func (v *userDataRetryVenue) methods() []string {
	recorded := v.recorder.recorded()
	methods := make([]string, len(recorded))
	for i := range recorded {
		methods[i] = recorded[i].Method
	}
	return methods
}

// newUserDataRetryExchange returns an exchange whose private connection for product p is set up as Setup sets it up,
// subscribed and flushed through the websocket manager with authenticated websocket use on, and served by v, with failed
// subscriptions tried again after delay
func newUserDataRetryExchange(t *testing.T, p userDataRetryProduct, v *userDataRetryVenue, delay time.Duration) *Exchange {
	t.Helper()
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, p.keyPath, r.URL.Path, "only listen keys should be requested") {
			return
		}
		if r.Method == http.MethodPut {
			if v.keepalives.Add(1) == 1 && v.heldKeepalive != nil {
				v.heldKeepalive.hold()
			}
			writeListenKeyReply(t, w, v.refuseKeepalive)
			return
		}
		n := v.listenKeys.Add(1)
		if h := v.heldKeys[n]; h != nil {
			h.hold()
		}
		writeListenKeyReply(t, w, v.refuseKey != nil && v.refuseKey(n))
	})
	ex.userDataRetry.delay = delay
	v.recorder.reply = func(req *DerivativesStreamRequest) []byte {
		n := v.streams.Add(1)
		if h := v.heldStreams[n]; h != nil {
			h.hold()
		}
		if v.refuseStream != nil && v.refuseStream(n) {
			return fmt.Appendf(nil, `{"error":{"code":2,"msg":"Invalid request: unknown variant"},"id":%d}`, req.ID)
		}
		return nil
	}
	useMockWebsocket(t, ex, &websocket.ConnectionSetup{
		MessageFilter:         p.filter,
		Connector:             ex.WebsocketDerivativesConnect,
		GenerateSubscriptions: func() (subscription.List, error) { return p.generate(ex) },
		Subscriber:            ex.SubscribeUserData,
		Unsubscriber:          ex.UnsubscribeUserData,
		Handler: func(ctx context.Context, conn websocket.Connection, msg []byte) error {
			return p.handle(ex, ctx, conn, msg)
		},
		ResponseCheckTimeout: ex.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:     ex.Config.WebsocketResponseMaxLimit,
	}, v.recorder.handle)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	return ex
}

// userDataRetrying reports whether a user data subscription of asset a, or the asset itself, is being tried again
func userDataRetrying(ex *Exchange, a asset.Item) bool {
	ex.userDataRetry.mu.Lock()
	defer ex.userDataRetry.mu.Unlock()
	for key := range ex.userDataRetry.running {
		if s, ok := key.(*subscription.Subscription); key == a || ok && s.Asset == a {
			return true
		}
	}
	return false
}

func TestResubscribeUserData(t *testing.T) {
	t.Parallel()
	// A retry of a refused subscription whose stream is unsubscribed, or is being unsubscribed, by the time the retry's
	// listen key arrives leaves the stream unsubscribed
	for _, tc := range []struct {
		name string
		// held keeps the unsubscription's own listen key request waiting until the retry's request is answered, and
		// equivalent unsubscribes through another subscription to the stream, as a caller holding its own may
		held, equivalent bool
	}{
		{name: "unsubscribed"},
		{name: "being unsubscribed", held: true},
		{name: "unsubscribed through an equivalent subscription", equivalent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			retryRequested, releaseRetry := make(chan struct{}), make(chan struct{})
			unsubscriptionRequested, releaseUnsubscription := make(chan struct{}), make(chan struct{})
			var listenKeys atomic.Int32
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
					return
				}
				var release chan struct{}
				switch listenKeys.Add(1) {
				case 1:
					w.WriteHeader(http.StatusBadRequest)
					_, err := w.Write([]byte(`{"code":-1001,"msg":"Internal error; unable to process your request. Please try again."}`))
					assert.NoError(t, err, "Write should not error")
					return
				case 2:
					close(retryRequested)
					release = releaseRetry
				case 3:
					if tc.held {
						close(unsubscriptionRequested)
						release = releaseUnsubscription
					}
				}
				if release != nil {
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				_, err := fmt.Fprintf(w, `{"listenKey":%q}`, testListenKey)
				assert.NoError(t, err, "Write should not error")
			})
			ex.userDataRetry.delay = time.Millisecond
			recorder := &streamRequestRecorder{}
			useMockWebsocket(t, ex, &websocket.ConnectionSetup{
				MessageFilter:            usdtmPrivateFilter,
				Connector:                ex.WebsocketDerivativesConnect,
				Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
				ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
				ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
				SubscriptionsNotRequired: true,
			}, recorder.handle)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
			require.NoError(t, err, "GetConnection must not error")
			failed := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
			require.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{failed}), websocket.ErrSubscriptionFailure, "SubscribeUserData must report the refused subscription")
			select {
			case <-retryRequested:
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the retry must request a listen key")
			}
			require.Same(t, failed, ex.Websocket.GetSubscription(failed), "the retry must subscribe the refused subscription, registered during its request")
			unsubscribing := failed
			if tc.equivalent {
				unsubscribing = &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
			}
			unsubscribed := make(chan error, 1)
			go func() { unsubscribed <- ex.UnsubscribeUserData(t.Context(), conn, subscription.List{unsubscribing}) }()
			waitUnsubscribed := func() {
				t.Helper()
				select {
				case err := <-unsubscribed:
					require.NoError(t, err, "UnsubscribeUserData must not error")
				case <-time.After(30 * time.Second):
					require.FailNow(t, "the unsubscription must return")
				}
			}
			if tc.held {
				select {
				case <-unsubscriptionRequested:
				case <-time.After(30 * time.Second):
					require.FailNow(t, "the unsubscription must request a listen key")
				}
			} else {
				waitUnsubscribed()
			}
			close(releaseRetry)
			require.Eventually(t, func() bool { return !userDataRetrying(ex, asset.USDTMarginedFutures) }, 30*time.Second, time.Millisecond, "the retry must stop")
			if tc.held {
				close(releaseUnsubscription)
				waitUnsubscribed()
			}
			recorded := recorder.recorded()
			methods := make([]string, 0, len(recorded))
			for _, req := range recorded {
				methods = append(methods, req.Method)
			}
			assert.Equal(t, []string{"UNSUBSCRIBE"}, methods, "the retry should not subscribe the stream unsubscribed during its request")
			assert.Nil(t, ex.Websocket.GetSubscription(failed), "the stream should stay unsubscribed")
			assert.Nil(t, conn.Subscriptions().Get(failed), "the retry should not record the stream against its connection")
		})
	}

	// A failed subscription stays registered, and recorded against its connection by the websocket manager, while it is
	// tried again, so a flush keeps the connection up, leaves the stream to the retry and sends nothing
	// requireLive requires conn to be the connected private connection of p's websocket, which a flush shutting it down
	// would have released
	requireLive := func(t *testing.T, ex *Exchange, p userDataRetryProduct, conn websocket.Connection) {
		t.Helper()
		got, err := ex.Websocket.GetConnection(p.filter)
		require.NoError(t, err, "GetConnection must find the private connection")
		require.Same(t, conn, got, "the private connection must be the one subscribed")
		require.True(t, isConnected(conn), "the private connection must stay up")
	}
	// requireSubscribed requires s to be the stream's only registered subscription, subscribed and recorded against conn
	requireSubscribed := func(t *testing.T, ex *Exchange, conn websocket.Connection, s *subscription.Subscription) {
		t.Helper()
		require.Eventually(t, func() bool { return s.State() == subscription.SubscribedState }, 30*time.Second, time.Millisecond, "the stream must be subscribed")
		// A retry records a subscription it made once it is marked subscribed
		require.Eventually(t, func() bool { return conn.Subscriptions().Get(s) == s }, 30*time.Second, time.Millisecond, "the subscription must be recorded against its connection")
		require.Equal(t, subscription.List{s}, ex.Websocket.GetSubscriptions(), "the subscription must be the only one registered")
	}
	// requireQuietFlush requires a flush wanting the stream to send nothing and keep its subscription
	requireQuietFlush := func(t *testing.T, ex *Exchange, v *userDataRetryVenue, s *subscription.Subscription) {
		t.Helper()
		sent := len(v.recorder.recorded())
		require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "a flush wanting the stream must not error")
		assert.Len(t, v.recorder.recorded(), sent, "a flush wanting the stream should send nothing")
		assert.Same(t, s, ex.Websocket.GetSubscription(s), "a flush wanting the stream should keep its subscription")
	}
	const atListenKey, atSubscribe = "listen key request", "SUBSCRIBE"
	for _, p := range userDataRetryProducts {
		for _, initial := range []string{atListenKey, atSubscribe} {
			for _, heldAt := range []string{atListenKey, atSubscribe} {
				for _, refused := range []bool{false, true} {
					outcome := "answered"
					if refused {
						outcome = "refused"
					}
					t.Run(fmt.Sprintf("%s subscription failing at its %s flushed while the retry's %s is held and %s", p.a, initial, heldAt, outcome), func(t *testing.T) {
						t.Parallel()
						// Listen key request 1 and stream request 1, when its key arrives, are the first subscription's; the
						// retry's first attempt makes the next of each, and its second, after a refusal, the one after
						held := newHeldRequest()
						retrySubscribe := int32(1)
						if initial == atSubscribe {
							retrySubscribe = 2
						}
						v := &userDataRetryVenue{
							refuseKey: func(n int32) bool {
								return (n == 1 && initial == atListenKey) || (n == 2 && heldAt == atListenKey && refused)
							},
							refuseStream: func(n int32) bool {
								return (n == 1 && initial == atSubscribe) || (n == retrySubscribe && heldAt == atSubscribe && refused)
							},
						}
						if heldAt == atListenKey {
							v.heldKeys = map[int32]*heldRequest{2: held}
						} else {
							v.heldStreams = map[int32]*heldRequest{retrySubscribe: held}
						}
						ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
						require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
						conn, err := ex.Websocket.GetConnection(p.filter)
						require.NoError(t, err, "GetConnection must not error")
						failed := ex.Websocket.GetSubscription(&subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a})
						require.NotNil(t, failed, "the failed subscription must stay registered")
						require.Same(t, failed, conn.Subscriptions().Get(failed), "the failed subscription must be recorded against its connection")
						waitFor(t, held.entered, "the retry must reach its held request")
						requireQuietFlush(t, ex, v, failed)
						requireLive(t, ex, p, conn)
						close(held.release)
						requireSubscribed(t, ex, conn, failed)
						assert.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry should end once the stream is subscribed")
						subscribes := 1
						if initial == atSubscribe {
							subscribes++
						}
						if heldAt == atSubscribe && refused {
							subscribes++
						}
						assert.Equal(t, slices.Repeat([]string{"SUBSCRIBE"}, subscribes), v.methods(), "the stream should be subscribed once after the refusals")
						requireQuietFlush(t, ex, v, failed)
						requireLive(t, ex, p, conn)
					})
				}
			}
			t.Run(fmt.Sprintf("%s subscription failing at its %s flushed before the retry's attempt", p.a, initial), func(t *testing.T) {
				t.Parallel()
				v := &userDataRetryVenue{
					refuseKey:    func(n int32) bool { return n == 1 && initial == atListenKey },
					refuseStream: func(n int32) bool { return n == 1 && initial == atSubscribe },
				}
				// The retry waits past the case's end, so its attempt is made here once the flush has run
				ex := newUserDataRetryExchange(t, p, v, time.Hour)
				require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
				conn, err := ex.Websocket.GetConnection(p.filter)
				require.NoError(t, err, "GetConnection must not error")
				failed := ex.Websocket.GetSubscription(&subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a})
				require.NotNil(t, failed, "the failed subscription must stay registered")
				requireQuietFlush(t, ex, v, failed)
				requireLive(t, ex, p, conn)
				require.True(t, userDataRetrying(ex, p.a), "the retry must still be waiting")
				done, err := ex.resubscribeUserData(t.Context(), t.Context(), conn, failed)
				require.NoError(t, err, "the retry's attempt must not error")
				assert.True(t, done, "the retry's attempt should report the stream subscribed")
				requireSubscribed(t, ex, conn, failed)
				requireQuietFlush(t, ex, v, failed)
				requireLive(t, ex, p, conn)
			})
		}

		for _, heldAt := range []string{atListenKey, atSubscribe} {
			t.Run(fmt.Sprintf("%s stream disabled while the retry's %s is held, then enabled", p.a, heldAt), func(t *testing.T) {
				t.Parallel()
				// Listen key request 1 is the first subscription's, 2 the retry's and 3 the unsubscription's
				held := newHeldRequest()
				v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }}
				if heldAt == atListenKey {
					v.heldKeys = map[int32]*heldRequest{2: held}
				} else {
					v.heldStreams = map[int32]*heldRequest{1: held}
				}
				ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
				require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
				key := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a}
				waitFor(t, held.entered, "the retry must reach its held request")
				require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, false), "SetAssetEnabled must not error")
				flushed := make(chan error, 1)
				go func() { flushed <- ex.Websocket.FlushChannels(t.Context()) }()
				if heldAt == atListenKey {
					select {
					case err := <-flushed:
						require.NoError(t, err, "the flush of the disabled asset must not error")
					case <-time.After(30 * time.Second):
						require.FailNow(t, "the flush of the disabled asset must return")
					}
					close(held.release)
				} else {
					// The unsubscription's UNSUBSCRIBE waits behind the held SUBSCRIBE, as Binance answers in order
					require.Eventually(t, func() bool { return v.listenKeys.Load() == 3 }, 30*time.Second, time.Millisecond, "the unsubscription must request its listen key")
					close(held.release)
					select {
					case err := <-flushed:
						require.NoError(t, err, "the flush of the disabled asset must not error")
					case <-time.After(30 * time.Second):
						require.FailNow(t, "the flush of the disabled asset must return")
					}
				}
				require.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry must end")
				assert.Nil(t, ex.Websocket.GetSubscription(key), "the flush of the disabled asset should leave no subscription registered")
				exp := make([]string, 0, 3)
				if heldAt == atSubscribe {
					exp = append(exp, "SUBSCRIBE")
				}
				exp = append(exp, "UNSUBSCRIBE")
				assert.Equal(t, exp, v.methods(), "the retry should not subscribe the stream after its unsubscription")
				require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, true), "SetAssetEnabled must not error")
				require.NoError(t, ex.Websocket.FlushChannels(t.Context()), "the flush of the enabled asset must not error")
				conn, err := ex.Websocket.GetConnection(p.filter)
				require.NoError(t, err, "GetConnection must find the private connection")
				fresh := ex.Websocket.GetSubscription(key)
				require.NotNil(t, fresh, "the flush of the enabled asset must subscribe the stream afresh")
				requireSubscribed(t, ex, conn, fresh)
				assert.Equal(t, append(exp, "SUBSCRIBE"), v.methods(), "the flush of the enabled asset should subscribe the stream")
			})
		}

		t.Run(p.a.String()+" retry's SUBSCRIBE acknowledged during a flush of the disabled asset", func(t *testing.T) {
			t.Parallel()
			// Listen key request 1 is the first subscription's, 2 the retry's and 3 the unsubscription's; stream request 1
			// is the retry's SUBSCRIBE and 2 the unsubscription's UNSUBSCRIBE
			heldSubscribe, heldUnsubscribe := newHeldRequest(), newHeldRequest()
			v := &userDataRetryVenue{
				refuseKey:   func(n int32) bool { return n == 1 },
				heldStreams: map[int32]*heldRequest{1: heldSubscribe, 2: heldUnsubscribe},
			}
			ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
			ex.userDataRetry.keepalive = 20 * time.Millisecond
			require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
			key := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a}
			failed := ex.Websocket.GetSubscription(key)
			require.NotNil(t, failed, "the failed subscription must stay registered")
			waitFor(t, heldSubscribe.entered, "the retry must send its SUBSCRIBE")
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, false), "SetAssetEnabled must not error")
			flushed := make(chan error, 1)
			go func() { flushed <- ex.Websocket.FlushChannels(t.Context()) }()
			require.Eventually(t, func() bool { return v.listenKeys.Load() == 3 }, 30*time.Second, time.Millisecond, "the unsubscription must request its listen key")
			close(heldSubscribe.release)
			require.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry must end on the acknowledgement")
			waitFor(t, heldUnsubscribe.entered, "the unsubscription must send its UNSUBSCRIBE")
			assert.Never(t, func() bool { return v.keepalives.Load() != 0 }, 200*time.Millisecond, time.Millisecond, "the subscription being unsubscribed should not be kept alive")
			assert.NotEqual(t, subscription.SubscribedState, failed.State(), "the subscription being unsubscribed should not be marked subscribed")
			close(heldUnsubscribe.release)
			select {
			case err := <-flushed:
				require.NoError(t, err, "the flush of the disabled asset must not error")
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the flush of the disabled asset must return")
			}
			assert.Nil(t, ex.Websocket.GetSubscription(key), "the flush should leave no subscription registered")
			assert.Never(t, func() bool { return v.keepalives.Load() != 0 }, 200*time.Millisecond, time.Millisecond, "the retired subscription should send no keepalive")
		})

		t.Run(p.a.String()+" retry failing late after the manager replaced its subscription", func(t *testing.T) {
			t.Parallel()
			// Listen key request 1 is the first subscription's, 2 the retry's, 3 the unsubscription's and 4 the replacement's
			held := newHeldRequest()
			v := &userDataRetryVenue{
				refuseKey: func(n int32) bool { return n == 1 || n == 2 },
				heldKeys:  map[int32]*heldRequest{2: held},
			}
			ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
			require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
			conn, err := ex.Websocket.GetConnection(p.filter)
			require.NoError(t, err, "GetConnection must not error")
			key := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a}
			failed := ex.Websocket.GetSubscription(key)
			require.NotNil(t, failed, "the failed subscription must stay registered")
			waitFor(t, held.entered, "the retry must request its listen key")
			require.NoError(t, ex.Websocket.UnsubscribeChannels(t.Context(), conn, subscription.List{failed}), "UnsubscribeChannels must not error")
			replacement := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a, Authenticated: true}
			require.NoError(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{replacement}), "SubscribeToChannels must not error")
			close(held.release)
			require.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry must end once its subscription is replaced")
			assert.Equal(t, []string{"UNSUBSCRIBE", "SUBSCRIBE"}, v.methods(), "only the replacement should subscribe the stream")
			assert.Equal(t, subscription.List{replacement}, ex.Websocket.GetSubscriptions(), "the replacement should stay the only subscription registered")
			assert.Equal(t, subscription.SubscribedState, replacement.State(), "the replacement should stay subscribed")
			// The manager's own unsubscription and subscription calls leave its connection's record alone
			assert.Same(t, failed, conn.Subscriptions().Get(key), "the connection should keep the record of the stream it held before")
			requireQuietFlush(t, ex, v, replacement)
			requireLive(t, ex, p, conn)
		})

		t.Run(p.a.String()+" stream subscribed through the manager while the retry waits", func(t *testing.T) {
			t.Parallel()
			v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 1 }}
			// The retry waits past the case's end
			ex := newUserDataRetryExchange(t, p, v, time.Hour)
			require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
			conn, err := ex.Websocket.GetConnection(p.filter)
			require.NoError(t, err, "GetConnection must not error")
			failed := ex.Websocket.GetSubscription(&subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a})
			require.NotNil(t, failed, "the failed subscription must stay registered")
			other := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a, Authenticated: true}
			assert.ErrorIs(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{other}), subscription.ErrDuplicate, "SubscribeToChannels should refuse a second subscription to the stream being tried again")
			assert.Empty(t, v.methods(), "the refused subscription should send nothing")
			requireQuietFlush(t, ex, v, failed)
			requireLive(t, ex, p, conn)
			require.NoError(t, ex.Websocket.ResubscribeToChannel(t.Context(), conn, failed), "ResubscribeToChannel must subscribe the stream being tried again")
			requireSubscribed(t, ex, conn, failed)
			assert.Equal(t, []string{"UNSUBSCRIBE", "SUBSCRIBE"}, v.methods(), "the resubscription should subscribe the stream")
			requireQuietFlush(t, ex, v, failed)
			requireLive(t, ex, p, conn)
			done, err := ex.resubscribeUserData(t.Context(), t.Context(), conn, failed)
			require.NoError(t, err, "the retry's next attempt must not error")
			assert.True(t, done, "the retry's next attempt should find the stream subscribed")
			assert.Len(t, v.methods(), 2, "the retry's next attempt should send nothing")
		})

		t.Run(p.a.String()+" resubscription failing at its listen key request", func(t *testing.T) {
			t.Parallel()
			// Listen key request 1 is the first subscription's, 2 the resubscription's unsubscription's, 3 its subscription's
			// and 4 the retry's
			v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 3 }}
			ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			conn, err := ex.Websocket.GetConnection(p.filter)
			require.NoError(t, err, "GetConnection must not error")
			s := ex.Websocket.GetSubscription(&subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a})
			require.NotNil(t, s, "connecting must subscribe the stream")
			require.ErrorIs(t, ex.Websocket.ResubscribeToChannel(t.Context(), conn, s), websocket.ErrSubscriptionFailure, "ResubscribeToChannel must report the failed subscription")
			requireSubscribed(t, ex, conn, s)
			assert.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry should end once the stream is subscribed")
			assert.Equal(t, []string{"SUBSCRIBE", "UNSUBSCRIBE", "SUBSCRIBE"}, v.methods(), "the retry should subscribe the stream again without a flush")
			assert.Empty(t, relayedPayloads(ex), "the retry that subscribed the stream should report nothing")
			requireQuietFlush(t, ex, v, s)
			requireLive(t, ex, p, conn)
		})

		t.Run(p.a.String()+" stream failing through the manager's own subscription call, then subscribed by the retry", func(t *testing.T) {
			t.Parallel()
			// Listen key request 1 is the first subscription's, 2 the failed unsubscription's, 3 the manager's subscription's
			// and 4 the retry's
			v := &userDataRetryVenue{refuseKey: func(n int32) bool { return n == 2 || n == 3 }}
			ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			conn, err := ex.Websocket.GetConnection(p.filter)
			require.NoError(t, err, "GetConnection must not error")
			// The failed unsubscription ends the subscription, leaving the connection up and recording nothing
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, false), "SetAssetEnabled must not error")
			require.ErrorIs(t, ex.Websocket.FlushChannels(t.Context()), websocket.ErrSubscriptionFailure, "the flush must report the failed unsubscription")
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, true), "SetAssetEnabled must not error")
			s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a, Authenticated: true}
			require.ErrorIs(t, ex.Websocket.SubscribeToChannels(t.Context(), conn, subscription.List{s}), websocket.ErrSubscriptionFailure, "SubscribeToChannels must report the failed subscription")
			requireSubscribed(t, ex, conn, s)
			requireQuietFlush(t, ex, v, s)
			requireLive(t, ex, p, conn)
			assert.Equal(t, []string{"SUBSCRIBE", "SUBSCRIBE"}, v.methods(), "the stream should be subscribed when connecting and by the retry")
		})

		t.Run(p.a.String()+" subscription failing while its stream's retry runs on the connection", func(t *testing.T) {
			t.Parallel()
			// Listen key request 1 is the first subscription's, 2 the retry's, 3 the failed unsubscription's, 4 the next
			// subscription's and 5 the retry's attempt at that one
			held := newHeldRequest()
			v := &userDataRetryVenue{
				refuseKey: func(n int32) bool { return n == 1 || n == 3 || n == 4 },
				heldKeys:  map[int32]*heldRequest{2: held},
			}
			ex := newUserDataRetryExchange(t, p, v, time.Millisecond)
			require.ErrorIs(t, ex.Websocket.Connect(t.Context()), websocket.ErrSubscriptionFailure, "Connect must report the failed subscription")
			conn, err := ex.Websocket.GetConnection(p.filter)
			require.NoError(t, err, "GetConnection must not error")
			key := &subscription.Subscription{Channel: userDataStreamChannel, Asset: p.a}
			waitFor(t, held.entered, "the retry must request its listen key")
			// The failed unsubscription ends the subscription but leaves the connection, which the flush then keeps
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, false), "SetAssetEnabled must not error")
			require.ErrorIs(t, ex.Websocket.FlushChannels(t.Context()), websocket.ErrSubscriptionFailure, "the flush must report the failed unsubscription")
			require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(p.a, true), "SetAssetEnabled must not error")
			require.ErrorIs(t, ex.Websocket.FlushChannels(t.Context()), websocket.ErrSubscriptionFailure, "the flush must report the failed subscription")
			next := ex.Websocket.GetSubscription(key)
			require.NotNil(t, next, "the subscription that failed must stay registered")
			close(held.release)
			requireSubscribed(t, ex, conn, next)
			assert.Eventually(t, func() bool { return !userDataRetrying(ex, p.a) }, 30*time.Second, time.Millisecond, "the retry should end once the stream is subscribed")
			assert.Equal(t, []string{"SUBSCRIBE"}, v.methods(), "the retry should subscribe the stream once")
			requireLive(t, ex, p, conn)
		})
	}
}

func TestSubscribeUserDataKeptAlive(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		// unsubscribe unsubscribes the stream while the subscription's listen key request waits, held keeps the
		// unsubscription's own listen key request waiting until the subscription has returned, and equivalent unsubscribes
		// through another subscription to the same stream, as a caller holding its own may
		unsubscribe, held, equivalent bool
	}{
		{name: "subscribes and keeps the listen key alive after its context ends"},
		{name: "an unsubscription during the listen key request is not undone", unsubscribe: true},
		{name: "an unsubscription under way when the listen key arrives is not undone", unsubscribe: true, held: true},
		{name: "an unsubscription through an equivalent subscription during the listen key request is not undone", unsubscribe: true, equivalent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			subscriptionRequested, releaseSubscription := make(chan struct{}), make(chan struct{})
			unsubscriptionRequested, releaseUnsubscription := make(chan struct{}), make(chan struct{})
			// Keepalives wait for the subscription's context to end, and those after the fourth for the case to end, so the
			// test server's request allowance holds however long the case is held up
			subscriptionEnded := make(chan struct{})
			var listenKeys, keepalives atomic.Int32
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
					return
				}
				var release chan struct{}
				if r.Method == http.MethodPut {
					select {
					case <-subscriptionEnded:
					case <-r.Context().Done():
						return
					}
					if keepalives.Add(1) > 4 {
						<-r.Context().Done()
						return
					}
				} else {
					switch listenKeys.Add(1) {
					case 1:
						close(subscriptionRequested)
						release = releaseSubscription
					case 2:
						if tc.held {
							close(unsubscriptionRequested)
							release = releaseUnsubscription
						}
					}
				}
				if release != nil {
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				_, err := fmt.Fprintf(w, `{"listenKey":%q}`, testListenKey)
				assert.NoError(t, err, "Write should not error")
			})
			ex.userDataRetry.keepalive = 20 * time.Millisecond
			recorder := &streamRequestRecorder{}
			useMockWebsocket(t, ex, &websocket.ConnectionSetup{
				MessageFilter:            usdtmPrivateFilter,
				Connector:                ex.WebsocketDerivativesConnect,
				Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
				ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
				ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
				SubscriptionsNotRequired: true,
			}, recorder.handle)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
			require.NoError(t, err, "GetConnection must not error")
			s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
			// The subscription is made under a context that ends once it returns, as a retry's attempt is, and its listen key
			// is kept alive under one that ends with the case
			subscribeCtx, endSubscription := context.WithCancel(t.Context())
			defer endSubscription()
			keepAliveCtx, endKeepalives := context.WithCancel(t.Context())
			defer endKeepalives()
			subscribed := make(chan error, 1)
			go func() { subscribed <- ex.subscribeUserDataKeptAlive(subscribeCtx, keepAliveCtx, conn, s) }()
			select {
			case <-subscriptionRequested:
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the subscription must request a listen key")
			}
			unsubscribed := make(chan error, 1)
			waitUnsubscribed := func() {
				t.Helper()
				select {
				case err := <-unsubscribed:
					require.NoError(t, err, "UnsubscribeUserData must not error")
				case <-time.After(30 * time.Second):
					require.FailNow(t, "the unsubscription must return")
				}
			}
			if tc.unsubscribe {
				unsubscribing := s
				if tc.equivalent {
					unsubscribing = &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
				}
				go func() { unsubscribed <- ex.UnsubscribeUserData(t.Context(), conn, subscription.List{unsubscribing}) }()
				if tc.held {
					select {
					case <-unsubscriptionRequested:
					case <-time.After(30 * time.Second):
						require.FailNow(t, "the unsubscription must request a listen key")
					}
				} else {
					waitUnsubscribed()
				}
			}
			close(releaseSubscription)
			select {
			case err := <-subscribed:
				require.NoError(t, err, "subscribeUserDataKeptAlive must not error")
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the subscription must return")
			}
			endSubscription()
			close(subscriptionEnded)
			if tc.held {
				close(releaseUnsubscription)
				waitUnsubscribed()
			}
			recorded := recorder.recorded()
			methods := make([]string, 0, len(recorded))
			for _, req := range recorded {
				methods = append(methods, req.Method)
			}
			if !tc.unsubscribe {
				assert.Equal(t, []string{"SUBSCRIBE"}, methods, "the subscription should subscribe the listen key")
				assert.Equal(t, subscription.SubscribedState, s.State(), "the stream should be subscribed")
				assert.Eventually(t, func() bool { return keepalives.Load() >= 4 }, 30*time.Second, time.Millisecond, "the listen key should be kept alive after the subscription's context ends")
				return
			}
			assert.Equal(t, []string{"UNSUBSCRIBE"}, methods, "the subscription should not subscribe the stream unsubscribed during its listen key request")
			assert.False(t, isSubscribedUserDataStream(s), "the stream should not be subscribed")
			if !tc.equivalent {
				assert.Equal(t, subscription.UnsubscribedState, s.State(), "the stream should stay unsubscribed")
			}
			assert.Nil(t, ex.Websocket.GetSubscription(s), "the stream should not be stored")
		})
	}

	for _, tc := range []struct {
		name string
		// failed has the unsubscription fail its listen key request, which ends the subscription as a completed one does,
		// underWay leaves the subscription unsubscribing and registered, as an unsubscription still waiting on its request
		// does, unregistered then removes it through an equivalent subscription, and replaced registers another
		// subscription to the stream before the subscription is subscribed again
		failed, underWay, unregistered, replaced bool
		methods                                  []string
	}{
		{name: "subscribes afresh a subscription unsubscribed before", methods: []string{"SUBSCRIBE", "UNSUBSCRIBE", "SUBSCRIBE"}},
		{name: "subscribes afresh a subscription a failed unsubscription ended", failed: true, methods: []string{"SUBSCRIBE", "SUBSCRIBE"}},
		{name: "subscribes afresh a subscription left unsubscribing and no longer registered", underWay: true, unregistered: true, methods: []string{"SUBSCRIBE", "SUBSCRIBE"}},
		{name: "refuses a subscription still registered while its unsubscription is under way", underWay: true, methods: []string{"SUBSCRIBE"}},
		{name: "refuses a subscription left unsubscribing that another has replaced and lets it unsubscribe the replacement", underWay: true, unregistered: true, replaced: true, methods: []string{"SUBSCRIBE", "SUBSCRIBE", "UNSUBSCRIBE"}},
		{name: "refuses a subscription another has replaced and lets it unsubscribe the replacement", failed: true, replaced: true, methods: []string{"SUBSCRIBE", "SUBSCRIBE", "UNSUBSCRIBE"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var listenKeys atomic.Int32
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
					return
				}
				// The second listen key request is the unsubscription's
				writeListenKeyReply(t, w, tc.failed && listenKeys.Add(1) == 2)
			})
			recorder := &streamRequestRecorder{}
			useMockWebsocket(t, ex, &websocket.ConnectionSetup{
				MessageFilter:            usdtmPrivateFilter,
				Connector:                ex.WebsocketDerivativesConnect,
				Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
				ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
				ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
				SubscriptionsNotRequired: true,
			}, recorder.handle)
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
			require.NoError(t, err, "GetConnection must not error")
			s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
			require.NoError(t, ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s), "subscribeUserDataKeptAlive must not error")
			switch {
			case tc.underWay:
				require.NoError(t, s.SetState(subscription.UnsubscribingState), "SetState must not error")
				if tc.unregistered {
					equivalent := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
					require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, equivalent), "RemoveSubscriptions must not error for the equivalent subscription")
					require.Nil(t, ex.Websocket.GetSubscription(s), "the equivalent subscription's removal must remove the stream")
				}
			case tc.failed:
				require.Error(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must report its failed request")
				require.Equal(t, subscription.UnsubscribedState, s.State(), "the failed unsubscription must end the stream")
				require.Nil(t, ex.Websocket.GetSubscription(s), "the failed unsubscription must remove the stream")
			default:
				require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must not error")
				require.Equal(t, subscription.UnsubscribedState, s.State(), "the stream must be unsubscribed")
			}
			if tc.replaced {
				replacement := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
				require.NoError(t, ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, replacement), "subscribeUserDataKeptAlive must not error for the replacement")
			}
			err = ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s)
			if tc.replaced {
				assert.ErrorIs(t, err, subscription.ErrDuplicate, "subscribeUserDataKeptAlive should refuse a subscription another has replaced")
				require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must not error for the replaced subscription")
				assert.Nil(t, ex.Websocket.GetSubscription(s), "unsubscribing through the replaced subscription should remove its replacement")
			}
			recorded := recorder.recorded()
			methods := make([]string, 0, len(recorded))
			for _, req := range recorded {
				methods = append(methods, req.Method)
			}
			switch {
			case tc.replaced:
				assert.Equal(t, tc.methods, methods, "only the replacement should be subscribed and then unsubscribed")
			case tc.underWay && !tc.unregistered:
				assert.ErrorIs(t, err, subscription.ErrDuplicate, "subscribeUserDataKeptAlive should refuse a subscription still registered")
				assert.Equal(t, tc.methods, methods, "the refused subscription should send nothing")
				assert.Equal(t, subscription.UnsubscribingState, s.State(), "the refused subscription should keep its state")
			default:
				require.NoError(t, err, "subscribeUserDataKeptAlive must not error for the subscription no longer registered")
				assert.Equal(t, tc.methods, methods, "the subscription no longer registered should be subscribed again")
				assert.Equal(t, subscription.SubscribedState, s.State(), "the stream should be subscribed again")
				assert.Same(t, s, ex.Websocket.GetSubscription(s), "the stream should be stored again")
			}
		})
	}

	// heldListenKeys returns an exchange whose listen key request numbered heldRequest is answered only once release is
	// closed, and the others at once
	heldListenKeys := func(t *testing.T, heldRequest int32) (ex *Exchange, conn websocket.Connection, recorder *streamRequestRecorder, entered, release chan struct{}) {
		t.Helper()
		entered, release = make(chan struct{}), make(chan struct{})
		var listenKeys atomic.Int32
		ex = newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
				return
			}
			if listenKeys.Add(1) == heldRequest {
				close(entered)
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
			}
			writeListenKeyReply(t, w, false)
		})
		recorder = &streamRequestRecorder{}
		useMockWebsocket(t, ex, &websocket.ConnectionSetup{
			MessageFilter:            usdtmPrivateFilter,
			Connector:                ex.WebsocketDerivativesConnect,
			Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
			ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
			ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
			SubscriptionsNotRequired: true,
		}, recorder.handle)
		require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
		var err error
		conn, err = ex.Websocket.GetConnection(usdtmPrivateFilter)
		require.NoError(t, err, "GetConnection must not error")
		return ex, conn, recorder, entered, release
	}
	requestMethods := func(recorder *streamRequestRecorder) []string {
		recorded := recorder.recorded()
		methods := make([]string, 0, len(recorded))
		for _, req := range recorded {
			methods = append(methods, req.Method)
		}
		return methods
	}
	waitResult := func(t *testing.T, c <-chan error, msg string) error {
		t.Helper()
		select {
		case err := <-c:
			return err
		case <-time.After(30 * time.Second):
			require.FailNow(t, msg)
			return nil
		}
	}

	t.Run("a subscription replaced during its listen key request leaves the stream to its replacement", func(t *testing.T) {
		t.Parallel()
		ex, conn, recorder, entered, release := heldListenKeys(t, 1)
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		subscribed := make(chan error, 1)
		go func() { subscribed <- ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s) }()
		waitFor(t, entered, "the subscription must request a listen key")
		equivalent := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{equivalent}), "UnsubscribeUserData must not error")
		replacement := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, replacement), "subscribeUserDataKeptAlive must not error for the replacement")
		close(release)
		require.NoError(t, waitResult(t, subscribed, "the subscription must return"), "subscribeUserDataKeptAlive must not error")
		assert.Equal(t, []string{"UNSUBSCRIBE", "SUBSCRIBE"}, requestMethods(recorder), "only the replacement should subscribe the stream")
		assert.Same(t, replacement, ex.Websocket.GetSubscription(s), "the replacement should stay registered")
	})

	t.Run("a renewal during a fresh subscription's listen key request leaves the subscription to it", func(t *testing.T) {
		t.Parallel()
		// The second subscription's listen key request is the third
		ex, conn, recorder, entered, release := heldListenKeys(t, 3)
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s), "subscribeUserDataKeptAlive must not error")
		require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must not error")
		subscribed := make(chan error, 1)
		go func() { subscribed <- ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s) }()
		waitFor(t, entered, "the fresh subscription must request a listen key")
		ex.renewUserDataStream(t.Context(), conn, asset.USDTMarginedFutures, errListenKeyExpired)
		assert.Equal(t, []string{"SUBSCRIBE", "UNSUBSCRIBE"}, requestMethods(recorder), "the renewal should not subscribe a stream being subscribed afresh")
		close(release)
		require.NoError(t, waitResult(t, subscribed, "the fresh subscription must return"), "subscribeUserDataKeptAlive must not error for the fresh subscription")
		assert.Equal(t, []string{"SUBSCRIBE", "UNSUBSCRIBE", "SUBSCRIBE"}, requestMethods(recorder), "the fresh subscription should subscribe the stream")
	})

	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		for _, tc := range []struct {
			name string
			// unsubscriptionFails fails the unsubscription's listen key request, which ends the subscription as a
			// completed unsubscription does
			unsubscriptionFails bool
			methods             []string
		}{
			{name: "a failed listen key request after a failed unsubscription leaves the stream to a fresh subscription", unsubscriptionFails: true, methods: []string{"SUBSCRIBE"}},
			{name: "a failed listen key request after an unsubscription leaves the stream to a fresh subscription", methods: []string{"UNSUBSCRIBE", "SUBSCRIBE"}},
		} {
			t.Run(a.String()+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				entered, release := make(chan struct{}), make(chan struct{})
				var listenKeys atomic.Int32
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if !assert.Contains(t, []string{"/fapi/v1/listenKey", "/dapi/v1/listenKey", "/eapi/v1/listenKey"}, r.URL.Path, "only listen keys should be requested") {
						return
					}
					// The subscription's listen key request is the first, and fails once the fresh subscription is made; the
					// unsubscription's is the second
					switch listenKeys.Add(1) {
					case 1:
						close(entered)
						select {
						case <-release:
						case <-r.Context().Done():
							return
						}
						writeListenKeyReply(t, w, true)
					case 2:
						writeListenKeyReply(t, w, tc.unsubscriptionFails)
					default:
						writeListenKeyReply(t, w, false)
					}
				})
				recorder := &streamRequestRecorder{}
				useMockWebsocket(t, ex, &websocket.ConnectionSetup{
					MessageFilter:            usdtmPrivateFilter,
					Connector:                ex.WebsocketDerivativesConnect,
					Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
					ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
					ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
					SubscriptionsNotRequired: true,
				}, recorder.handle)
				require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
				conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
				require.NoError(t, err, "GetConnection must not error")
				s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: a, Authenticated: true}
				subscribed := make(chan error, 1)
				go func() { subscribed <- ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, s) }()
				waitFor(t, entered, "the subscription must request a listen key")
				err = ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s})
				if tc.unsubscriptionFails {
					require.ErrorIs(t, err, websocket.ErrSubscriptionFailure, "UnsubscribeUserData must report its failed listen key request")
				} else {
					require.NoError(t, err, "UnsubscribeUserData must not error")
				}
				fresh := &subscription.Subscription{Channel: userDataStreamChannel, Asset: a, Authenticated: true}
				require.NoError(t, ex.subscribeUserDataKeptAlive(t.Context(), t.Context(), conn, fresh), "subscribeUserDataKeptAlive must not error for the fresh subscription")
				close(release)
				require.ErrorIs(t, waitResult(t, subscribed, "the subscription must return"), websocket.ErrSubscriptionFailure, "the subscription must report its failed listen key request")
				assert.Same(t, fresh, ex.Websocket.GetSubscription(s), "the fresh subscription should stay registered")
				assert.Equal(t, subscription.SubscribedState, fresh.State(), "the fresh subscription should stay subscribed")
				assert.Equal(t, tc.methods, requestMethods(recorder), "only the fresh subscription should subscribe the stream")
			})
		}
	}
}

// newUserDataVenueConnection returns an exchange connected, without subscriptions, to a USDⓈ-M private connection served
// by v, and that connection
func newUserDataVenueConnection(t *testing.T, v *userDataRetryVenue) (*Exchange, websocket.Connection) {
	t.Helper()
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
			return
		}
		if r.Method == http.MethodPut {
			if v.keepalives.Add(1) == 1 && v.heldKeepalive != nil {
				v.heldKeepalive.hold()
			}
			writeListenKeyReply(t, w, v.refuseKeepalive)
			return
		}
		n := v.listenKeys.Add(1)
		if h := v.heldKeys[n]; h != nil {
			h.hold()
		}
		writeListenKeyReply(t, w, v.refuseKey != nil && v.refuseKey(n))
	})
	v.recorder.reply = func(req *DerivativesStreamRequest) []byte {
		n := v.streams.Add(1)
		if h := v.heldStreams[n]; h != nil {
			h.hold()
		}
		if v.refuseStream != nil && v.refuseStream(n) {
			return fmt.Appendf(nil, `{"error":{"code":2,"msg":"Invalid request: unknown variant"},"id":%d}`, req.ID)
		}
		return nil
	}
	useMockWebsocket(t, ex, &websocket.ConnectionSetup{
		MessageFilter:            usdtmPrivateFilter,
		Connector:                ex.WebsocketDerivativesConnect,
		Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
		ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
		SubscriptionsNotRequired: true,
	}, v.recorder.handle)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	return ex, conn
}

func TestSubscribeListenKey(t *testing.T) {
	t.Parallel()
	register := func(t *testing.T, ex *Exchange, conn websocket.Connection) *subscription.Subscription {
		t.Helper()
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.Websocket.AddSubscriptions(conn, s), "AddSubscriptions must not error")
		return s
	}

	t.Run("subscribes a registered subscription and keeps its listen key alive", func(t *testing.T) {
		t.Parallel()
		v := &userDataRetryVenue{}
		ex, conn := newUserDataVenueConnection(t, v)
		ex.userDataRetry.keepalive = 20 * time.Millisecond
		s := register(t, ex, conn)
		keepAliveCtx, endKeepalives := context.WithCancel(t.Context())
		defer endKeepalives()
		require.NoError(t, ex.subscribeListenKey(t.Context(), keepAliveCtx, conn, s), "subscribeListenKey must not error")
		assert.Equal(t, subscription.SubscribedState, s.State(), "the subscription should be marked subscribed")
		assert.Equal(t, []string{"SUBSCRIBE"}, v.methods(), "the listen key should be subscribed")
		assert.Eventually(t, func() bool { return v.keepalives.Load() > 0 }, 30*time.Second, time.Millisecond, "the listen key should be kept alive")
	})

	for _, tc := range []struct {
		name      string
		refuseKey bool
	}{
		{name: "leaves a subscription whose listen key request is refused registered", refuseKey: true},
		{name: "leaves a subscription whose SUBSCRIBE is refused registered"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := &userDataRetryVenue{
				refuseKey:    func(int32) bool { return tc.refuseKey },
				refuseStream: func(int32) bool { return !tc.refuseKey },
			}
			ex, conn := newUserDataVenueConnection(t, v)
			s := register(t, ex, conn)
			assert.ErrorIs(t, ex.subscribeListenKey(t.Context(), t.Context(), conn, s), websocket.ErrSubscriptionFailure, "subscribeListenKey should report the refusal")
			assert.Same(t, s, ex.Websocket.GetSubscription(s), "the subscription should stay registered")
			assert.Equal(t, subscription.SubscribingState, s.State(), "the subscription should stay being subscribed")
		})
	}

	t.Run("leaves a subscription unsubscribed during its listen key request", func(t *testing.T) {
		t.Parallel()
		held := newHeldRequest()
		v := &userDataRetryVenue{heldKeys: map[int32]*heldRequest{1: held}}
		ex, conn := newUserDataVenueConnection(t, v)
		s := register(t, ex, conn)
		subscribed := make(chan error, 1)
		go func() { subscribed <- ex.subscribeListenKey(t.Context(), t.Context(), conn, s) }()
		waitFor(t, held.entered, "the subscription must request its listen key")
		require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must not error")
		close(held.release)
		select {
		case err := <-subscribed:
			require.NoError(t, err, "subscribeListenKey must not error")
		case <-time.After(30 * time.Second):
			require.FailNow(t, "subscribeListenKey must return")
		}
		assert.Equal(t, []string{"UNSUBSCRIBE"}, v.methods(), "the stream unsubscribed during the request should not be subscribed")
		assert.Equal(t, subscription.UnsubscribedState, s.State(), "the subscription should stay unsubscribed")
	})

	t.Run("subscribes a registered subscription being resubscribed", func(t *testing.T) {
		t.Parallel()
		v := &userDataRetryVenue{}
		ex, conn := newUserDataVenueConnection(t, v)
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, s.SetState(subscription.ResubscribingState), "SetState must not error")
		require.NoError(t, ex.Websocket.AddSubscriptions(conn, s), "AddSubscriptions must not error")
		require.NoError(t, ex.subscribeListenKey(t.Context(), t.Context(), conn, s), "subscribeListenKey must not error")
		assert.Equal(t, []string{"SUBSCRIBE"}, v.methods(), "the subscription being resubscribed should be subscribed")
		assert.Equal(t, subscription.SubscribedState, s.State(), "the subscription should be marked subscribed")
	})

	for _, tc := range []struct {
		name string
		// replace removes the subscription through an equivalent one and registers a replacement, rather than leaving it
		// unsubscribing as an unsubscription under way does
		replace bool
	}{
		{name: "neither marks subscribed nor keeps alive a subscription being unsubscribed when Binance acknowledges it"},
		{name: "neither marks subscribed nor keeps alive a replaced subscription when Binance acknowledges it", replace: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			held := newHeldRequest()
			v := &userDataRetryVenue{heldStreams: map[int32]*heldRequest{1: held}}
			ex, conn := newUserDataVenueConnection(t, v)
			ex.userDataRetry.keepalive = 20 * time.Millisecond
			s := register(t, ex, conn)
			subscribed := make(chan error, 1)
			go func() { subscribed <- ex.subscribeListenKey(t.Context(), t.Context(), conn, s) }()
			waitFor(t, held.entered, "the subscription must send its SUBSCRIBE")
			state := subscription.UnsubscribingState
			if tc.replace {
				state = subscription.SubscribingState
				equivalent := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
				require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, equivalent), "RemoveSubscriptions must not error for the equivalent subscription")
				replacement := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
				require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, replacement), "AddSuccessfulSubscriptions must not error for the replacement")
			} else {
				require.NoError(t, s.SetState(state), "SetState must not error")
			}
			close(held.release)
			select {
			case err := <-subscribed:
				require.NoError(t, err, "subscribeListenKey must not error")
			case <-time.After(30 * time.Second):
				require.FailNow(t, "subscribeListenKey must return")
			}
			assert.Equal(t, state, s.State(), "the subscription should keep its state")
			assert.Never(t, func() bool { return v.keepalives.Load() != 0 }, 100*time.Millisecond, time.Millisecond, "the subscription should not be kept alive")
		})
	}
}

func TestKeepUserDataStreamAlive(t *testing.T) {
	t.Parallel()
	// subscribed returns a subscribed user data stream subscription registered on a venue's private connection
	subscribed := func(t *testing.T, v *userDataRetryVenue) (*Exchange, websocket.Connection, *subscription.Subscription) {
		t.Helper()
		ex, conn := newUserDataVenueConnection(t, v)
		ex.userDataRetry.keepalive = 20 * time.Millisecond
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
		require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, s), "AddSuccessfulSubscriptions must not error")
		return ex, conn, s
	}

	t.Run("cancels a keepalive under way when the websocket shuts down", func(t *testing.T) {
		t.Parallel()
		held := newHeldRequest()
		v := &userDataRetryVenue{heldKeepalive: held}
		ex, conn, s := subscribed(t, v)
		ex.keepUserDataStreamAlive(context.WithoutCancel(t.Context()), conn, s)
		waitFor(t, held.entered, "the keepalive must reach the venue")
		shutdown := make(chan error, 1)
		go func() { shutdown <- ex.Websocket.Shutdown() }()
		select {
		case err := <-shutdown:
			require.NoError(t, err, "Shutdown must not error")
		case <-time.After(10 * time.Second):
			require.FailNow(t, "Shutdown must not wait for the keepalive under way")
		}
		assert.Empty(t, relayedPayloads(ex), "the cancelled keepalive should neither renew the listen key nor report a failure")
		close(held.release)
	})

	t.Run("neither renews nor reports a keepalive cancelled with its context", func(t *testing.T) {
		t.Parallel()
		held := newHeldRequest()
		v := &userDataRetryVenue{heldKeepalive: held}
		ex, conn, s := subscribed(t, v)
		ctx, cancel := context.WithCancel(t.Context())
		ex.keepUserDataStreamAlive(ctx, conn, s)
		waitFor(t, held.entered, "the keepalive must reach the venue")
		cancel()
		assert.Never(t, func() bool { return v.listenKeys.Load() != 0 || len(relayedPayloads(ex)) != 0 }, 200*time.Millisecond, time.Millisecond, "a keepalive cancelled with its context should neither renew the listen key nor report a failure")
		close(held.release)
	})

	t.Run("neither renews nor reports a keepalive refused once the connection is down", func(t *testing.T) {
		t.Parallel()
		held := newHeldRequest()
		v := &userDataRetryVenue{heldKeepalive: held, refuseKeepalive: true}
		ex, conn, s := subscribed(t, v)
		ex.keepUserDataStreamAlive(context.WithoutCancel(t.Context()), conn, s)
		waitFor(t, held.entered, "the keepalive must reach the venue")
		require.NoError(t, conn.Shutdown(), "Shutdown must not error for the connection")
		close(held.release)
		assert.Never(t, func() bool { return v.listenKeys.Load() != 0 || len(relayedPayloads(ex)) != 0 }, 200*time.Millisecond, time.Millisecond, "a keepalive refused once the connection is down should neither renew the listen key nor report a failure")
	})

	t.Run("stops keeping the listen key alive once the connection is down", func(t *testing.T) {
		t.Parallel()
		v := &userDataRetryVenue{}
		ex, conn, s := subscribed(t, v)
		ex.keepUserDataStreamAlive(context.WithoutCancel(t.Context()), conn, s)
		require.Eventually(t, func() bool { return v.keepalives.Load() > 0 }, 30*time.Second, time.Millisecond, "the listen key must be kept alive")
		require.NoError(t, conn.Shutdown(), "Shutdown must not error for the connection")
		sent := v.keepalives.Load()
		assert.Never(t, func() bool { return v.keepalives.Load() > sent+1 }, 200*time.Millisecond, time.Millisecond, "the listen key should not be kept alive once the connection is down")
	})
}

func TestUserDataRetriesTracker(t *testing.T) {
	t.Parallel()
	var r userDataRetries
	r.mu.Lock()
	defer r.mu.Unlock()
	running := r.tracker(false)
	require.NotNil(t, running, "tracker must make the map of subscription retries on first use")
	running[asset.Spot] = &userDataRetry{}
	assert.Equal(t, running, r.tracker(false), "tracker should return the map of subscription retries it made")
	renewing := r.tracker(true)
	require.NotNil(t, renewing, "tracker must make the map of renewals on first use")
	assert.Empty(t, renewing, "tracker should track renewals apart from subscription retries")
	renewing[asset.Spot] = &userDataRetry{}
	assert.Equal(t, renewing, r.tracker(true), "tracker should return the map of renewals it made")
	assert.Len(t, r.tracker(false), 1, "tracking a renewal should leave the subscription retries alone")
}

func TestIsUnsubscribedState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		state subscription.State
		exp   bool
	}{
		{"inactive", subscription.InactiveState, false},
		{"subscribing", subscription.SubscribingState, false},
		{"subscribed", subscription.SubscribedState, false},
		{"resubscribing", subscription.ResubscribingState, false},
		{"unsubscribing", subscription.UnsubscribingState, true},
		{"unsubscribed", subscription.UnsubscribedState, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, isUnsubscribedState(tc.state), "isUnsubscribedState should report whether the %s state is left by an unsubscription", tc.name)
		})
	}
}

// fakeRetryConnection is a connection whose state the retry tests set, standing in for a private connection. The
// check numbered heldCheck, from 1, waits for release once it has closed checked, as a retry checks the connection
// before each attempt
type fakeRetryConnection struct {
	websocket.Connection
	connected         atomic.Bool
	checks            atomic.Int32
	heldCheck         int32
	checked, released chan struct{}
}

func (c *fakeRetryConnection) IsConnected() bool {
	if c.checks.Add(1) == c.heldCheck {
		close(c.checked)
		select {
		case <-c.released:
		case <-time.After(30 * time.Second):
		}
	}
	return c.connected.Load()
}

func newFakeRetryConnection() *fakeRetryConnection {
	c := new(fakeRetryConnection)
	c.connected.Store(true)
	return c
}

func TestRetryUserDataSubscription(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	ex.userDataRetry.delay = time.Millisecond
	conn := newFakeRetryConnection()
	var attempts atomic.Int32
	ex.retryUserDataSubscription(t.Context(), conn, asset.USDTMarginedFutures, asset.USDTMarginedFutures, false, func(context.Context) (bool, error) {
		if attempts.Add(1) < 3 {
			return false, errTestRetryAttempt
		}
		return true, nil
	})
	require.Eventually(t, func() bool { return !userDataRetrying(ex, asset.USDTMarginedFutures) }, 30*time.Second, time.Millisecond, "the retry must end once an attempt is done")
	assert.Equal(t, int32(3), attempts.Load(), "the retry should try until an attempt is done")
	reports := relayedPayloads(ex)
	require.Len(t, reports, 2, "each failed attempt must be reported")
	for _, report := range reports {
		reported, ok := report.(error)
		require.True(t, ok, "a failed attempt must be reported as an error")
		assert.ErrorIs(t, reported, errTestRetryAttempt, "the report should carry the attempt's error")
	}

	down := newFakeRetryConnection()
	down.connected.Store(false)
	var attemptsWhileDown atomic.Int32
	ex.retryUserDataSubscription(t.Context(), down, asset.CoinMarginedFutures, asset.CoinMarginedFutures, false, func(context.Context) (bool, error) {
		attemptsWhileDown.Add(1)
		return true, nil
	})
	require.Eventually(t, func() bool { return !userDataRetrying(ex, asset.CoinMarginedFutures) }, 30*time.Second, time.Millisecond, "the retry must end once its connection is down")
	assert.Zero(t, attemptsWhileDown.Load(), "the retry should not try on a connection that is down")
}

func TestRetryUserData(t *testing.T) {
	t.Parallel()
	// attempt returns a retry attempt counting its runs, which waits for release when given one, and reports done
	attempt := func(done bool, entered chan<- struct{}, release <-chan struct{}) (*atomic.Int32, func(context.Context) (bool, error)) {
		runs := new(atomic.Int32)
		return runs, func(context.Context) (bool, error) {
			runs.Add(1)
			if entered != nil {
				entered <- struct{}{}
			}
			if release != nil {
				select {
				case <-release:
				case <-time.After(30 * time.Second):
				}
			}
			return done, nil
		}
	}
	renewing := func(ex *Exchange, key any) bool {
		ex.userDataRetry.mu.Lock()
		defer ex.userDataRetry.mu.Unlock()
		_, running := ex.userDataRetry.renewing[key]
		return running
	}
	running := func(ex *Exchange, key any) bool {
		ex.userDataRetry.mu.Lock()
		defer ex.userDataRetry.mu.Unlock()
		_, running := ex.userDataRetry.running[key]
		return running
	}
	const a = asset.USDTMarginedFutures

	t.Run("makes another attempt when asked again during its own", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		ex.userDataRetry.delay = time.Millisecond
		conn := newFakeRetryConnection()
		key := new(subscription.Subscription)
		entered, release := make(chan struct{}, 2), make(chan struct{})
		runs, first := attempt(true, entered, release)
		otherRuns, other := attempt(true, nil, nil)
		ex.retryUserData(t.Context(), conn, key, a, true, false, first)
		waitFor(t, entered, "the retry must make its first attempt")
		ex.retryUserData(t.Context(), conn, key, a, true, false, other)
		close(release)
		require.Eventually(t, func() bool { return !running(ex, key) }, 30*time.Second, time.Millisecond, "the retry must end")
		assert.Equal(t, int32(2), runs.Load(), "the retry should make another attempt when asked again during its own")
		assert.Zero(t, otherRuns.Load(), "asking again should start no second retry")
	})

	t.Run("an attempt covers asking again before it", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		ex.userDataRetry.delay = time.Millisecond
		// The retry checks its connection as it starts and before each attempt, and the check before its second attempt
		// waits until it has been asked again
		conn := newFakeRetryConnection()
		conn.heldCheck, conn.checked, conn.released = 3, make(chan struct{}), make(chan struct{})
		key := new(subscription.Subscription)
		var calls atomic.Int32
		ex.retryUserData(t.Context(), conn, key, a, true, false, func(context.Context) (bool, error) {
			return calls.Add(1) > 1, nil
		})
		waitFor(t, conn.checked, "the retry must wait to make its second attempt")
		ex.retryUserData(t.Context(), conn, key, a, true, false, func(context.Context) (bool, error) { return true, nil })
		close(conn.released)
		require.Eventually(t, func() bool { return !running(ex, key) }, 30*time.Second, time.Millisecond, "the retry must end")
		assert.Equal(t, int32(2), calls.Load(), "the attempt after asking again should be the last")
	})

	t.Run("waits its first retry delay before another attempt asked for during its own", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		ex.userDataRetry.delay = time.Hour
		conn := newFakeRetryConnection()
		key := new(subscription.Subscription)
		entered, release := make(chan struct{}, 2), make(chan struct{})
		runs, first := attempt(true, entered, release)
		ex.retryUserData(t.Context(), conn, key, a, true, false, first)
		waitFor(t, entered, "the retry must make its first attempt")
		ex.retryUserData(t.Context(), conn, key, a, true, false, first)
		close(release)
		assert.Never(t, func() bool { return runs.Load() > 1 }, 50*time.Millisecond, time.Millisecond, "the retry should wait before another attempt")
		assert.True(t, running(ex, key), "the retry should keep waiting to make another attempt")
	})

	t.Run("a renewal is not asked again", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		ex.userDataRetry.delay = time.Millisecond
		conn := newFakeRetryConnection()
		entered, release := make(chan struct{}, 2), make(chan struct{})
		runs, first := attempt(true, entered, release)
		ex.retryUserData(t.Context(), conn, a, a, true, true, first)
		waitFor(t, entered, "the renewal must make its first attempt")
		ex.retryUserData(t.Context(), conn, a, a, true, true, first)
		close(release)
		require.Eventually(t, func() bool { return !renewing(ex, a) }, 30*time.Second, time.Millisecond, "the renewal must end")
		assert.Never(t, func() bool { return runs.Load() > 1 }, 50*time.Millisecond, time.Millisecond, "the renewal should make no other attempt")
	})

	t.Run("subscriptions are tried again apart", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		ex.userDataRetry.delay = time.Millisecond
		conn := newFakeRetryConnection()
		first, second := new(subscription.Subscription), new(subscription.Subscription)
		entered, release := make(chan struct{}, 2), make(chan struct{})
		firstRuns, firstAttempt := attempt(true, entered, release)
		secondRuns, secondAttempt := attempt(true, nil, nil)
		ex.retryUserData(t.Context(), conn, first, a, true, false, firstAttempt)
		waitFor(t, entered, "the first subscription's retry must make its attempt")
		ex.retryUserData(t.Context(), conn, second, a, true, false, secondAttempt)
		require.Eventually(t, func() bool { return secondRuns.Load() == 1 && !running(ex, second) }, 30*time.Second, time.Millisecond, "the second subscription's retry must run while the first's attempt waits")
		close(release)
		require.Eventually(t, func() bool { return !running(ex, first) }, 30*time.Second, time.Millisecond, "the first subscription's retry must end")
		assert.Equal(t, int32(1), firstRuns.Load(), "the first subscription's retry should make its attempt once")
	})

	t.Run("a retry on a replaced connection leaves its successor", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		ex.userDataRetry.delay = time.Millisecond
		replaced, successor := newFakeRetryConnection(), newFakeRetryConnection()
		entered, release := make(chan struct{}, 2), make(chan struct{})
		replacedRuns, replacedAttempt := attempt(true, entered, release)
		ex.retryUserData(t.Context(), replaced, a, a, true, false, replacedAttempt)
		waitFor(t, entered, "the replaced connection's retry must make its attempt")
		replaced.connected.Store(false)
		successorEntered, successorRelease := make(chan struct{}, 2), make(chan struct{})
		successorRuns, successorAttempt := attempt(true, successorEntered, successorRelease)
		ex.retryUserData(t.Context(), successor, a, a, true, false, successorAttempt)
		waitFor(t, successorEntered, "the successor's retry must make its attempt")
		// Asked again on the replaced connection, the successor's retry is not
		ex.retryUserData(t.Context(), replaced, a, a, true, false, replacedAttempt)
		close(release)
		assert.Never(t, func() bool { return !running(ex, a) }, 50*time.Millisecond, time.Millisecond, "the replaced connection's retry should leave its successor tracked")
		close(successorRelease)
		require.Eventually(t, func() bool { return !running(ex, a) }, 30*time.Second, time.Millisecond, "the successor's retry must end")
		assert.Equal(t, int32(1), replacedRuns.Load(), "the replaced connection's retry should make its attempt once")
		assert.Equal(t, int32(1), successorRuns.Load(), "the successor's retry should make its attempt once")
	})
}

func TestUserDataRetryReportsOnlyBinanceRefusal(t *testing.T) {
	t.Parallel()
	// A retry that Binance refuses again is reported with Binance's error alone
	var listenKeys atomic.Int32
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "only listen keys should be requested") {
			return
		}
		if r.Method != http.MethodPut && listenKeys.Add(1) <= 2 {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(`{"code":-4109,"msg":"This account is inactive, please activate the account first."}`))
			assert.NoError(t, err, "Write should not error")
			return
		}
		_, err := fmt.Fprintf(w, `{"listenKey":%q}`, testListenKey)
		assert.NoError(t, err, "Write should not error")
	})
	ex.userDataRetry.delay = time.Millisecond
	ex.userDataRetry.keepalive = time.Hour
	recorder := &streamRequestRecorder{}
	useMockWebsocket(t, ex, &websocket.ConnectionSetup{
		MessageFilter:            usdtmPrivateFilter,
		Connector:                ex.WebsocketDerivativesConnect,
		Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
		ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
		SubscriptionsNotRequired: true,
	}, recorder.handle)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	sub := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
	require.ErrorIs(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{sub}), websocket.ErrSubscriptionFailure, "SubscribeUserData must report the failed subscription")
	require.Eventually(t, func() bool {
		s := ex.Websocket.GetSubscription(sub)
		return s != nil && s.State() == subscription.SubscribedState
	}, 30*time.Second, time.Millisecond, "the subscription must be made by the retry after the refused one")
	reports := relayedPayloads(ex)
	require.Len(t, reports, 1, "the refused retry must be reported once")
	reported, ok := reports[0].(error)
	require.True(t, ok, "the refused retry must be reported as an error")
	assert.ErrorIs(t, reported, websocket.ErrSubscriptionFailure, "the refused retry should be reported as a subscription failure")
	assert.ErrorContains(t, reported, "-4109", "the refused retry should report Binance's error")
	assert.NotErrorIs(t, reported, subscription.ErrInStateAlready, "the refused retry should not report its subscription's state")
	require.NoError(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{ex.Websocket.GetSubscription(sub)}), "UnsubscribeUserData must not error")
}

func TestNextUserDataRetryWait(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		wait, exp time.Duration
	}{
		{0, userDataRetryDelay},
		{userDataRetryDelay, 2 * userDataRetryDelay},
		{2 * userDataRetryDelay, 4 * userDataRetryDelay},
		{userDataRetryMaxDelay - time.Second, userDataRetryMaxDelay},
		{userDataRetryMaxDelay, userDataRetryMaxDelay},
	} {
		t.Run(tc.wait.String(), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, nextUserDataRetryWait(tc.wait, userDataRetryDelay), "nextUserDataRetryWait should follow a wait of %s with the expected one", tc.wait)
		})
	}
}

func TestStartUserDataStream(t *testing.T) { //nolint:tparallel // the cases share active and empty, so they run in order while the test runs in parallel
	t.Parallel()
	serve := func(listenKey string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/fapi/v1/listenKey", "/dapi/v1/listenKey", "/eapi/v1/listenKey":
				if r.Method == http.MethodPost {
					_, _ = fmt.Fprintf(w, `{"listenKey":%q}`, listenKey)
					return
				}
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}
	active := newTradingTestExchange(t, serve(testListenKey))
	empty := newTradingTestExchange(t, serve(""))
	// The cases share active and empty, which the statement after them calls too, so they run in order
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			listenKey, err := active.startUserDataStream(t.Context(), a)
			require.NoErrorf(t, err, "startUserDataStream must not error for %s", a)
			assert.Equalf(t, testListenKey, listenKey, "startUserDataStream should return the %s listen key", a)
			_, err = empty.startUserDataStream(t.Context(), a)
			assert.ErrorIsf(t, err, errListenKeyEmpty, "startUserDataStream should reject an empty %s listen key", a)
		})
	}
	_, err := active.startUserDataStream(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "startUserDataStream should reject an asset without user data streams")
}

func TestKeepaliveUserDataSubscription(t *testing.T) { //nolint:tparallel // the cases share ex, conn and the recorder, so they run in order while the test runs in parallel
	t.Parallel()
	recorder := &streamRequestRecorder{}
	ex := newDerivativesWsTestExchange(t, recorder)
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	// The cases share ex, conn and the recorder, which are checked after them, so they run in order
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures, asset.Options} {
		t.Run(a.String(), func(t *testing.T) {
			s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: a, Authenticated: true}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, s), "AddSuccessfulSubscriptions must not error")
			assert.Truef(t, ex.keepaliveUserDataSubscription(t.Context(), conn, s), "keepaliveUserDataSubscription should keep the %s stream alive", a)
			require.NoError(t, s.SetState(subscription.UnsubscribedState), "SetState must not error")
			assert.Falsef(t, ex.keepaliveUserDataSubscription(t.Context(), conn, s), "keepaliveUserDataSubscription should stop for the unsubscribed %s stream", a)
			// A subscription marked subscribed after an unsubscription or a replacement removed it keeps nothing alive
			require.NoError(t, s.SetState(subscription.UnsubscribingState), "SetState must not error")
			require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, s), "RemoveSubscriptions must not error")
			require.NoError(t, s.SetState(subscription.SubscribedState), "SetState must not error")
			assert.Falsef(t, ex.keepaliveUserDataSubscription(t.Context(), conn, s), "keepaliveUserDataSubscription should stop for the %s stream no longer registered", a)
			replacement := &subscription.Subscription{Channel: userDataStreamChannel, Asset: a, Authenticated: true}
			require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, replacement), "AddSuccessfulSubscriptions must not error for the replacement")
			assert.Falsef(t, ex.keepaliveUserDataSubscription(t.Context(), conn, s), "keepaliveUserDataSubscription should stop for the replaced %s subscription", a)
			require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, replacement), "RemoveSubscriptions must not error for the replacement")
		})
	}
	assert.Empty(t, recorder.recorded(), "a successful keepalive should not resubscribe")
	assert.Empty(t, relayedPayloads(ex), "a successful keepalive should relay nothing")

	// A failed keepalive renews the stream; an asset without user data streams makes the renewal fail too
	unsupported := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.Spot, Authenticated: true}
	require.NoError(t, ex.Websocket.AddSuccessfulSubscriptions(conn, unsupported), "AddSuccessfulSubscriptions must not error")
	assert.True(t, ex.keepaliveUserDataSubscription(t.Context(), conn, unsupported), "keepaliveUserDataSubscription should keep trying while subscribed")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 1, "a failed renewal must relay one error")
	relayedErr, ok := payloads[0].(error)
	require.True(t, ok, "the relayed payload must be an error")
	assert.ErrorIs(t, relayedErr, asset.ErrNotSupported, "the renewal error should carry its cause")
}

func TestProcessListenKeyExpired(t *testing.T) { //nolint:tparallel // the cases share ex and the recorder, so they run in order while the test runs in parallel
	t.Parallel()
	recorder := &streamRequestRecorder{}
	ex := newDerivativesWsTestExchange(t, recorder)
	expiries := []struct {
		a       asset.Item
		filter  string
		fixture string
		handle  func(context.Context, websocket.Connection, []byte) error
		exp     *ListenKeyExpired
	}{
		{asset.USDTMarginedFutures, usdtmPrivateFilter, "testdata/wsUFuturesUserData.jsonl", ex.wsHandleUSDTMarginedFuturesUserData, &ListenKeyExpired{EventType: "listenKeyExpired", EventTime: fixtureTime(1736996475556), ListenKey: testListenKey}},
		{asset.CoinMarginedFutures, coinmPrivateFilter, "testdata/wsCFuturesUserData.jsonl", ex.wsHandleCoinMarginedFuturesUserData, &ListenKeyExpired{EventType: "listenKeyExpired", EventTime: fixtureTime(1576653824250), ListenKey: testListenKey}},
		{asset.Options, optionsPrivateFilter, "testdata/wsOptionsUserData.jsonl", ex.wsHandleOptionsUserData, &ListenKeyExpired{EventType: "listenKeyExpired", EventTime: fixtureTime(1736996475556), ListenKey: testListenKey}},
	}
	// A listen key expires on a connection subscribed to its user data stream. A failed renewal of a subscribed stream is
	// tried again, which the test's counts leave out by waiting longer than the test lasts
	ex.userDataRetry.mu.Lock()
	ex.userDataRetry.delay = time.Hour
	ex.userDataRetry.mu.Unlock()
	for _, tc := range expiries {
		conn, err := ex.Websocket.GetConnection(tc.filter)
		require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
		s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: tc.a, Authenticated: true}
		require.NoErrorf(t, ex.Websocket.AddSubscriptions(conn, s), "AddSubscriptions must not error for %s", tc.a)
		require.NoErrorf(t, s.SetState(subscription.SubscribedState), "SetState must not error for %s", tc.a)
	}
	// Each expiry starts a renewal that outlives its case, so the cases give the handlers the test's context
	ctx := t.Context()
	// The cases share ex and the recorder, whose renewals are counted after them, so they run in order
	for _, tc := range expiries {
		t.Run(subtestName(tc.filter), func(t *testing.T) {
			conn, err := ex.Websocket.GetConnection(tc.filter)
			require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
			require.NoErrorf(t, tc.handle(ctx, conn, fixtureFrame(t, tc.fixture, "listenKeyExpired")), "handling listenKeyExpired must not error for %s", tc.filter)
			assert.Equalf(t, []any{tc.exp}, relayedPayloads(ex), "listenKeyExpired should be relayed for %s", tc.filter)
		})
	}
	require.Eventually(t, func() bool { return len(recorder.recorded()) == 3 }, time.Second*30, time.Millisecond*10, "each expiry must renew its stream")
	for _, req := range recorder.recorded() {
		assert.Equal(t, DerivativesStreamRequest{Method: "SUBSCRIBE", Params: []any{testListenKey}, ID: req.ID}, req, "the renewal should subscribe the new listen key")
	}
	assert.Empty(t, relayedPayloads(ex), "a successful renewal should relay nothing")

	// A renewal whose listen key request fails relays the failure as the expiry's, beside the expiry itself, in either
	// order, as the renewal runs in the background
	failing := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"code":-1001,"msg":"Internal error; unable to process your request. Please try again."}`))
		assert.NoError(t, err, "Write should not error")
	}))
	failing.Start()
	for _, u := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined, exchange.RestOptions} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(u.String(), failing.URL), "SetRunningURL must not error")
	}
	for _, tc := range expiries {
		conn, err := ex.Websocket.GetConnection(tc.filter)
		require.NoErrorf(t, err, "GetConnection must not error for %s", tc.filter)
		require.NoErrorf(t, tc.handle(t.Context(), conn, fixtureFrame(t, tc.fixture, "listenKeyExpired")), "handling listenKeyExpired must not error for %s", tc.filter)
	}
	var relayed []any
	require.Eventually(t, func() bool {
		relayed = append(relayed, relayedPayloads(ex)...)
		return len(relayed) >= 2*len(expiries)
	}, time.Second*30, time.Millisecond*10, "each expiry and its failed renewal must be relayed")
	var expired, failed int
	for _, p := range relayed {
		if err, ok := p.(error); ok {
			failed++
			assert.ErrorIs(t, err, errListenKeyExpired, "a failed renewal should relay its error as the listen key expiry's")
			continue
		}
		expired++
		assert.IsType(t, &ListenKeyExpired{}, p, "the expiry should be relayed beside its failed renewal")
	}
	assert.Equal(t, len(expiries), failed, "each failed renewal should relay its error")
	assert.Equal(t, len(expiries), expired, "each expiry should be relayed")
	assert.Len(t, recorder.recorded(), 3, "a failed renewal should subscribe nothing")
}

// newRenewalTestExchange returns an exchange whose USDⓈ-M private connection has subscribed its user data stream, its
// retries waiting a millisecond, with each listen key request answered by reply and numbered from 1, the first
// subscription's, and the count of those requests
func newRenewalTestExchange(t *testing.T, reply func(n int32, w http.ResponseWriter, r *http.Request)) (*Exchange, websocket.Connection, *subscription.Subscription, *streamRequestRecorder, *atomic.Int32) {
	t.Helper()
	requests := new(atomic.Int32)
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/fapi/v1/listenKey", r.URL.Path, "the request should manage a listen key")
		assert.Equal(t, http.MethodPost, r.Method, "the request should create a listen key")
		reply(requests.Add(1), w, r)
	})
	ex.userDataRetry.delay = time.Millisecond
	recorder := &streamRequestRecorder{}
	useMockWebsocket(t, ex, &websocket.ConnectionSetup{
		MessageFilter:            usdtmPrivateFilter,
		Connector:                ex.WebsocketDerivativesConnect,
		Handler:                  ex.wsHandleUSDTMarginedFuturesUserData,
		ResponseCheckTimeout:     ex.Config.WebsocketResponseCheckTimeout,
		ResponseMaxLimit:         ex.Config.WebsocketResponseMaxLimit,
		SubscriptionsNotRequired: true,
	}, recorder.handle)
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	conn, err := ex.Websocket.GetConnection(usdtmPrivateFilter)
	require.NoError(t, err, "GetConnection must not error")
	s := &subscription.Subscription{Channel: userDataStreamChannel, Asset: asset.USDTMarginedFutures, Authenticated: true}
	require.NoError(t, ex.SubscribeUserData(t.Context(), conn, subscription.List{s}), "the first subscription must succeed")
	require.Len(t, recorder.recorded(), 1, "the first subscription must reach the server")
	return ex, conn, s, recorder, requests
}

// writeListenKeyReply answers a listen key request with the test listen key, or with Binance's transient internal error
func writeListenKeyReply(t *testing.T, w http.ResponseWriter, fail bool) {
	t.Helper()
	if fail {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"code":-1001,"msg":"Internal error; unable to process your request. Please try again."}`))
		assert.NoError(t, err, "Write should not error")
		return
	}
	_, err := fmt.Fprintf(w, `{"listenKey":%q}`, testListenKey)
	assert.NoError(t, err, "Write should not error")
}

// TestRenewUserDataStream ensures that a failed renewal of an expired listen key is tried again while the stream is
// subscribed, as no user data arrives until one succeeds, beside any subscription retry of the asset, and that the
// retries end once one has succeeded or the stream is no longer subscribed, without undoing an unsubscription made
// during the renewal's or a retry's request
func TestRenewUserDataStream(t *testing.T) {
	t.Parallel()
	renewing := func(ex *Exchange) bool {
		ex.userDataRetry.mu.Lock()
		defer ex.userDataRetry.mu.Unlock()
		_, ok := ex.userDataRetry.renewing[asset.USDTMarginedFutures]
		return ok
	}
	expire := func(t *testing.T, ex *Exchange, conn websocket.Connection) {
		t.Helper()
		require.NoError(t, ex.processListenKeyExpired(t.Context(), conn, []byte(`{"e":"listenKeyExpired","E":1791284812500,"listenKey":"expired"}`), asset.USDTMarginedFutures), "the expiry must be handled")
	}
	for _, tc := range []struct {
		name  string
		fails func(n int32) bool
		// end, when set, leaves the stream no longer subscribed once a retry has failed
		end func(*testing.T, *Exchange, websocket.Connection, *subscription.Subscription)
	}{
		{"renewed after a failure", func(n int32) bool { return n == 2 }, nil},
		{"given up once unsubscribed", func(n int32) bool { return n > 1 }, func(t *testing.T, ex *Exchange, conn websocket.Connection, s *subscription.Subscription) {
			t.Helper()
			require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, s), "RemoveSubscriptions must not error")
		}},
		{"given up while unsubscribing", func(n int32) bool { return n > 1 }, func(t *testing.T, _ *Exchange, _ websocket.Connection, s *subscription.Subscription) {
			t.Helper()
			// An unsubscription of the stream under way leaves it registered as unsubscribing
			require.NoError(t, s.SetState(subscription.UnsubscribingState), "SetState must not error")
		}},
		{"given up once a failed unsubscription ended the stream", func(n int32) bool { return n > 1 }, func(t *testing.T, ex *Exchange, conn websocket.Connection, s *subscription.Subscription) {
			t.Helper()
			require.Error(t, ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}), "UnsubscribeUserData must report its failed request")
			require.Equal(t, subscription.UnsubscribedState, s.State(), "the failed unsubscription must end the stream")
			require.Nil(t, ex.Websocket.GetSubscription(s), "the failed unsubscription must remove the stream")
		}},
		{"given up while subscribed afresh", func(n int32) bool { return n > 1 }, func(t *testing.T, _ *Exchange, _ websocket.Connection, s *subscription.Subscription) {
			t.Helper()
			// A subscription of the stream under way subscribes its own listen key
			require.NoError(t, s.SetState(subscription.SubscribingState), "SetState must not error")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, conn, s, recorder, requests := newRenewalTestExchange(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
				writeListenKeyReply(t, w, tc.fails(n))
			})
			expire(t, ex, conn)
			if tc.end == nil {
				require.Eventually(t, func() bool { return len(recorder.recorded()) == 2 }, 30*time.Second, time.Millisecond, "the renewed listen key must be subscribed")
			} else {
				require.Eventually(t, func() bool { return requests.Load() >= 4 }, 30*time.Second, time.Millisecond, "the failed renewal must be tried again")
				tc.end(t, ex, conn, s)
			}
			require.Eventually(t, func() bool { return !renewing(ex) }, 30*time.Second, time.Millisecond, "the renewal must stop being tried")
			if tc.end != nil {
				assert.Len(t, recorder.recorded(), 1, "a stream no longer subscribed should not be subscribed again")
				return
			}
			assert.Equal(t, int32(3), requests.Load(), "the renewal should be tried until it succeeds, and no more")
			assert.Len(t, recorder.recorded(), 2, "the renewed listen key should be subscribed once")
		})
	}

	t.Run("tried again after the context that asked for it ends", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the first subscription's, 2 the renewal's and 3 its retry's
		ex, conn, _, recorder, requests := newRenewalTestExchange(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
			writeListenKeyReply(t, w, n == 2)
		})
		ex.userDataRetry.mu.Lock()
		ex.userDataRetry.delay = 50 * time.Millisecond
		ex.userDataRetry.mu.Unlock()
		// A keepalive loop's context ends with the loop, which a replacement of its subscription ends while the stream stays
		// wanted
		ctx, cancel := context.WithCancel(t.Context())
		ex.renewUserDataStream(ctx, conn, asset.USDTMarginedFutures, errTestRetryAttempt)
		require.True(t, renewing(ex), "the failed renewal must be tried again")
		cancel()
		require.Eventually(t, func() bool { return len(recorder.recorded()) == 2 }, 30*time.Second, time.Millisecond, "the renewal must be tried again after the context that asked for it ends")
		require.Eventually(t, func() bool { return !renewing(ex) }, 30*time.Second, time.Millisecond, "the renewal must stop being tried")
		assert.Equal(t, int32(3), requests.Load(), "the renewal should be tried until it succeeds, and no more")
	})

	t.Run("requests no listen key once the stream is no longer subscribed", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the first subscription's and 2 the renewal's, refused once the stream is no longer
		// subscribed, so the retry finds it unsubscribed before its first attempt
		held := newHeldRequest()
		ex, conn, s, recorder, requests := newRenewalTestExchange(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
			if n == 2 {
				held.hold()
			}
			writeListenKeyReply(t, w, n > 1)
		})
		expire(t, ex, conn)
		waitFor(t, held.entered, "the renewal must request its listen key")
		require.NoError(t, ex.Websocket.RemoveSubscriptions(conn, s), "RemoveSubscriptions must not error")
		close(held.release)
		var reported bool
		require.Eventually(t, func() bool {
			for _, p := range relayedPayloads(ex) {
				if _, ok := p.(error); ok {
					reported = true
				}
			}
			return reported && !renewing(ex)
		}, 30*time.Second, time.Millisecond, "the failed renewal must be reported and stop being tried")
		assert.Never(t, func() bool { return requests.Load() != 2 }, 100*time.Millisecond, time.Millisecond, "the renewal should request no listen key once the stream is no longer subscribed")
		assert.Len(t, recorder.recorded(), 1, "a stream no longer subscribed should not be subscribed again")
	})

	t.Run("a retry's request under way is cancelled when the websocket shuts down", func(t *testing.T) {
		t.Parallel()
		// Listen key request 1 is the first subscription's, 2 the renewal's and 3 its retry's, held until the case ends
		held := newHeldRequest()
		ex, conn, _, recorder, _ := newRenewalTestExchange(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
			if n == 3 {
				held.hold()
			}
			writeListenKeyReply(t, w, n > 1)
		})
		ex.renewUserDataStream(t.Context(), conn, asset.USDTMarginedFutures, errTestRetryAttempt)
		waitFor(t, held.entered, "the failed renewal must be tried again")
		shutdown := make(chan error, 1)
		go func() { shutdown <- ex.Websocket.Shutdown() }()
		select {
		case err := <-shutdown:
			require.NoError(t, err, "Shutdown must not error")
		case <-time.After(10 * time.Second):
			require.FailNow(t, "Shutdown must not wait for the renewal's request under way")
		}
		close(held.release)
		assert.False(t, renewing(ex), "the renewal should stop being tried once the websocket shuts down")
		assert.Len(t, recorder.recorded(), 1, "the cancelled renewal should subscribe nothing")
	})

	t.Run("renewed beside a subscription retry", func(t *testing.T) {
		t.Parallel()
		ex, conn, _, recorder, requests := newRenewalTestExchange(t, func(n int32, w http.ResponseWriter, _ *http.Request) {
			writeListenKeyReply(t, w, n == 2)
		})
		// A subscription retry of the asset on the connection, as a failed subscription leaves, ends without renewing the
		// stream once it finds the stream subscribed
		held, release := make(chan struct{}), make(chan struct{})
		ex.retryUserDataSubscription(t.Context(), conn, asset.USDTMarginedFutures, asset.USDTMarginedFutures, true, func(ctx context.Context) (bool, error) {
			close(held)
			select {
			case <-release:
			case <-ctx.Done():
			}
			return true, nil
		})
		select {
		case <-held:
		case <-time.After(30 * time.Second):
			require.Fail(t, "the subscription retry must start")
		}
		expire(t, ex, conn)
		require.Eventually(t, func() bool { return len(recorder.recorded()) == 2 }, 30*time.Second, time.Millisecond, "the renewal must be tried again beside the subscription retry")
		close(release)
		require.Eventually(t, func() bool { return !renewing(ex) }, 30*time.Second, time.Millisecond, "the renewal must stop being tried")
		assert.Equal(t, int32(3), requests.Load(), "the renewal should be tried until it succeeds, and no more")
	})

	for _, tc := range []struct {
		name string
		// held is the listen key request answered only once the stream has been, or is being, unsubscribed: the
		// renewal's own, or the first retry's after the renewal's request fails
		held int32
		// unsubscribing keeps the unsubscription's own listen key request waiting until the renewal has returned
		unsubscribing bool
	}{
		{name: "an unsubscription during the renewal's request is not undone", held: 2},
		{name: "an unsubscription under way when the renewal's key arrives is not undone", held: 2, unsubscribing: true},
		{name: "an unsubscription during a retry's request is not undone", held: 3},
		{name: "an unsubscription under way when a retry's key arrives is not undone", held: 3, unsubscribing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			entered, releaseRenewal := make(chan struct{}), make(chan struct{})
			unsubscriptionRequested, releaseUnsubscription := make(chan struct{}), make(chan struct{})
			ex, conn, s, recorder, _ := newRenewalTestExchange(t, func(n int32, w http.ResponseWriter, r *http.Request) {
				var release chan struct{}
				switch {
				case n == tc.held:
					close(entered)
					release = releaseRenewal
				case n == tc.held+1 && tc.unsubscribing:
					close(unsubscriptionRequested)
					release = releaseUnsubscription
				}
				if release != nil {
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
				}
				writeListenKeyReply(t, w, n > 1 && n < tc.held)
			})
			// The renewal is run here rather than from an expiry, so that the test can wait for it to return
			renewed := make(chan struct{})
			go func() {
				defer close(renewed)
				ex.renewUserDataStream(t.Context(), conn, asset.USDTMarginedFutures, errListenKeyExpired)
			}()
			select {
			case <-entered:
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the held listen key request must be made")
			}
			unsubscribed := make(chan error, 1)
			go func() { unsubscribed <- ex.UnsubscribeUserData(t.Context(), conn, subscription.List{s}) }()
			waitUnsubscribed := func() {
				t.Helper()
				select {
				case err := <-unsubscribed:
					require.NoError(t, err, "UnsubscribeUserData must not error")
				case <-time.After(30 * time.Second):
					require.FailNow(t, "the unsubscription must return")
				}
			}
			if tc.unsubscribing {
				select {
				case <-unsubscriptionRequested:
				case <-time.After(30 * time.Second):
					require.FailNow(t, "the unsubscription must request a listen key")
				}
			} else {
				waitUnsubscribed()
			}
			close(releaseRenewal)
			select {
			case <-renewed:
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the renewal must return")
			}
			require.Eventually(t, func() bool { return !renewing(ex) }, 30*time.Second, time.Millisecond, "the renewal must stop being tried")
			if tc.unsubscribing {
				close(releaseUnsubscription)
				waitUnsubscribed()
			}
			recorded := recorder.recorded()
			methods := make([]string, 0, len(recorded))
			for _, req := range recorded {
				methods = append(methods, req.Method)
			}
			assert.Equal(t, []string{"SUBSCRIBE", "UNSUBSCRIBE"}, methods, "the renewal should not subscribe the stream unsubscribed during its request")
		})
	}
}

func TestFetchDerivativesOrderbook(t *testing.T) { //nolint:tparallel // the cases share ex and its server, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	srv := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body string
		switch r.URL.Path {
		case "/fapi/v1/depth", "/dapi/v1/depth":
			assert.Equal(t, "1000", r.URL.Query().Get("limit"), "the futures order book should request 1000 levels")
			body = `{"lastUpdateId":1027024,"E":1589436922972,"T":1589436922959,"bids":[["4.00000000","431.00000000"]],"asks":[["4.00000200","12.00000000"]]}`
		case "/eapi/v1/depth":
			assert.Equal(t, "50", r.URL.Query().Get("limit"), "the options order book should request 50 levels")
			body = `{"T":1762866729358,"lastUpdateId":42788078000,"bids":[["3645.000","1.00"]],"asks":[["3675.000","5.00"]]}`
		default:
			http.NotFound(w, r)
			return
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	}))
	srv.Start()
	for _, u := range []exchange.URL{exchange.RestUSDTMargined, exchange.RestCoinMargined, exchange.RestOptions} {
		require.NoError(t, ex.API.Endpoints.SetRunningURL(u.String(), srv.URL), "SetRunningURL must not error")
	}
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")

	// The cases share ex and its server, which the statements after them repoint, so they run in order
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  *orderbook.Book
	}{
		{asset.USDTMarginedFutures, currency.NewBTCUSDT(), &orderbook.Book{Bids: orderbook.Levels{{Price: 4, Amount: 431}}, Asks: orderbook.Levels{{Price: 4.000002, Amount: 12}}, LastUpdateID: 1027024, LastUpdated: time.UnixMilli(1589436922959)}},
		{asset.CoinMarginedFutures, testCoinMarginedFuturesPair, &orderbook.Book{Bids: orderbook.Levels{{Price: 4, Amount: 431}}, Asks: orderbook.Levels{{Price: 4.000002, Amount: 12}}, LastUpdateID: 1027024, LastUpdated: time.UnixMilli(1589436922959)}},
		{asset.Options, testOptionsPair, &orderbook.Book{Bids: orderbook.Levels{{Price: 3645, Amount: 1}}, Asks: orderbook.Levels{{Price: 3675, Amount: 5}}, LastUpdateID: 42788078000, LastUpdated: time.UnixMilli(1762866729358)}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			book, err := ex.fetchDerivativesOrderbook(t.Context(), tc.pair, tc.a)
			require.NoErrorf(t, err, "fetchDerivativesOrderbook must not error for %s", tc.a)
			tc.exp.Exchange, tc.exp.Pair, tc.exp.Asset, tc.exp.ValidateOrderbook = ex.Name, tc.pair, tc.a, ex.ValidateOrderbook
			assert.Equalf(t, tc.exp, book, "fetchDerivativesOrderbook should convert the %s snapshot", tc.a)
		})
	}
	_, err := ex.fetchDerivativesOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "fetchDerivativesOrderbook should reject an asset without derivatives depth")

	overflow := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`{"lastUpdateId":18446744073709551615,"T":1589436922959,"bids":[],"asks":[]}`))
		assert.NoError(t, err, "Write should not error")
	}))
	overflow.Start()
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestUSDTMargined.String(), overflow.URL), "SetRunningURL must not error")
	_, err = ex.fetchDerivativesOrderbook(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures)
	assert.ErrorIs(t, err, errUpdateIDOutOfRange, "fetchDerivativesOrderbook should reject an update ID beyond int64")
}

func TestCheckDerivativesPendingUpdate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                  string
		lastUpdateID          int64
		previousFinalUpdateID int64
		finalUpdateID         int64
		skip                  bool
		wantErr               error
	}{
		{name: "ends before the snapshot", lastUpdateID: 100, previousFinalUpdateID: 80, finalUpdateID: 99, skip: true},
		{name: "spans the snapshot", lastUpdateID: 100, previousFinalUpdateID: 90, finalUpdateID: 110},
		{name: "ends at the snapshot", lastUpdateID: 100, previousFinalUpdateID: 90, finalUpdateID: 100},
		{name: "follows the snapshot", lastUpdateID: 100, previousFinalUpdateID: 100, finalUpdateID: 120},
		{name: "starts after the snapshot", lastUpdateID: 100, previousFinalUpdateID: 101, finalUpdateID: 120, wantErr: orderbookmanager.ErrOrderbookSnapshotOutdated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			skip, err := checkDerivativesPendingUpdate(tc.lastUpdateID, tc.previousFinalUpdateID+1, &orderbook.Update{UpdateID: tc.finalUpdateID})
			assert.ErrorIsf(t, err, tc.wantErr, "checkDerivativesPendingUpdate should return the expected error for an event that %s", tc.name)
			assert.Equalf(t, tc.skip, skip, "checkDerivativesPendingUpdate should decide whether to skip an event that %s", tc.name)
		})
	}
}

func TestProcessDerivativesDepthUpdate(t *testing.T) {
	t.Parallel()
	err := new(Exchange).processDerivativesDepthUpdate(t.Context(), 1, &orderbook.Update{})
	assert.ErrorIs(t, err, errOrderbookSyncNotSetUp, "processDerivativesDepthUpdate should require the order book synchronisation")
}

// orderbookState is the part of a stored order book the synchronisation tests compare
type orderbookState struct {
	Bids, Asks   orderbook.Levels
	LastUpdateID int64
	LastUpdated  time.Time
	LastPushed   time.Time
}

func storedOrderbookState(ex *Exchange, pair currency.Pair, a asset.Item) (*orderbookState, error) {
	book, err := ex.Websocket.Orderbook.GetOrderbook(pair, a)
	if err != nil {
		return nil, err
	}
	return &orderbookState{Bids: book.Bids, Asks: book.Asks, LastUpdateID: book.LastUpdateID, LastUpdated: book.LastUpdated, LastPushed: book.LastPushed}, nil
}

func TestDerivativesOrderbookSync(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		fixture  string
		stream   string
		handle   func(*Exchange) func(context.Context, websocket.Connection, []byte) error
		a        asset.Item
		pair     currency.Pair
		snapshot *orderbook.Book
		exp      *orderbookState
	}{
		{
			name: "USDⓈ-M", fixture: "testdata/wsUFuturesPublic.jsonl", stream: "btcusdt@depth@100ms", a: asset.USDTMarginedFutures, pair: testUSDTMarginedFuturesPair,
			handle: func(ex *Exchange) func(context.Context, websocket.Connection, []byte) error {
				return ex.wsHandleUSDTMarginedFuturesData
			},
			snapshot: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 85999.9, Amount: 15}, {Price: 85999.8, Amount: 0.2}, {Price: 85999.7, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 86000, Amount: 4}, {Price: 86000.1, Amount: 0.5}},
				LastUpdateID: 11747004259000,
			},
			exp: &orderbookState{
				Bids:         orderbook.Levels{{Price: 85999.9, Amount: 16.201}, {Price: 85999.8, Amount: 0.381}},
				Asks:         orderbook.Levels{{Price: 86000.1, Amount: 0.702}, {Price: 86000.2, Amount: 1.157}},
				LastUpdateID: 11747004265102,
				LastUpdated:  time.UnixMilli(1791283218271),
				LastPushed:   time.UnixMilli(1791283218273),
			},
		},
		{
			name: "COIN-M", fixture: "testdata/wsCFutures.jsonl", stream: "btcusd_perp@depth@100ms", a: asset.CoinMarginedFutures, pair: testCoinMarginedFuturesPair,
			handle: func(ex *Exchange) func(context.Context, websocket.Connection, []byte) error {
				return ex.wsHandleCoinMarginedFuturesData
			},
			snapshot: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 86135.4, Amount: 2649}, {Price: 86135.3, Amount: 21}, {Price: 86133.6, Amount: 1}},
				Asks:         orderbook.Levels{{Price: 86135.5, Amount: 3502}, {Price: 86135.6, Amount: 32}},
				LastUpdateID: 11747023540000,
			},
			exp: &orderbookState{
				Bids:         orderbook.Levels{{Price: 86135.4, Amount: 2700}, {Price: 86135.3, Amount: 815}},
				Asks:         orderbook.Levels{{Price: 86135.6, Amount: 858}, {Price: 86135.7, Amount: 12}},
				LastUpdateID: 11747023544100,
				LastUpdated:  time.UnixMilli(1791283450364),
				LastPushed:   time.UnixMilli(1791283450370),
			},
		},
		{
			name: "options", fixture: "testdata/wsOptions.jsonl", stream: "btc-261030-85000-c@depth@100ms", a: asset.Options, pair: testOptionsPair,
			handle: func(ex *Exchange) func(context.Context, websocket.Connection, []byte) error {
				return ex.wsHandleOptionsData
			},
			snapshot: &orderbook.Book{
				Bids:         orderbook.Levels{{Price: 3645, Amount: 1}, {Price: 3640, Amount: 5.12}},
				Asks:         orderbook.Levels{{Price: 3675, Amount: 5}, {Price: 3680, Amount: 4.01}, {Price: 3690, Amount: 4.01}},
				LastUpdateID: 42788078000,
			},
			exp: &orderbookState{
				Bids:         orderbook.Levels{{Price: 3645, Amount: 2.17}, {Price: 3640, Amount: 5.12}},
				Asks:         orderbook.Levels{{Price: 3670, Amount: 0.56}, {Price: 3680, Amount: 4.01}, {Price: 3690, Amount: 4.01}},
				LastUpdateID: 42788078248,
				LastUpdated:  time.UnixMilli(1791283337855),
				LastPushed:   time.UnixMilli(1791283337936),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newDerivativesTestExchange(t)
			var fetches atomic.Int32
			useTestOrderbookSync(ex, func(_ context.Context, p currency.Pair, a asset.Item) (*orderbook.Book, error) {
				fetches.Add(1)
				book := *tc.snapshot
				book.Exchange, book.Pair, book.Asset, book.LastUpdated = ex.Name, p, a, time.UnixMilli(1)
				book.Bids, book.Asks = append(orderbook.Levels(nil), tc.snapshot.Bids...), append(orderbook.Levels(nil), tc.snapshot.Asks...)
				return &book, nil
			})
			handle := tc.handle(ex)
			var last []byte
			var depthFrames [][]byte
			for _, frame := range fixtureFrames(t, tc.fixture) {
				if stream, _, _, _ := decodeStreamFrame(frame); stream == tc.stream {
					depthFrames = append(depthFrames, frame)
					last = frame
				}
			}
			require.NoError(t, handleFramesWithin(t.Context(), handle, depthFrames), "handling all diff depth frames must return")
			var got *orderbookState
			require.Eventually(t, func() bool {
				var err error
				got, err = storedOrderbookState(ex, tc.pair, tc.a)
				return err == nil && got.LastUpdateID == tc.exp.LastUpdateID
			}, time.Second*30, time.Millisecond*10, "the order book must synchronise to the last event")
			assert.Equal(t, tc.exp, got, "the synchronised order book should apply every event to the snapshot")

			// An event delivered again chains by pu to the event before it, not to the book it moved on, so it must be
			// dropped rather than invalidate a book that is in sync
			require.NoError(t, handle(t.Context(), nil, last), "handling a diff depth frame delivered again must not error")
			got, err := storedOrderbookState(ex, tc.pair, tc.a)
			require.NoError(t, err, "a re-delivered event must leave the book valid")
			assert.Equal(t, tc.exp, got, "a re-delivered event should leave the book as it was")
			assert.Equal(t, int32(1), fetches.Load(), "a re-delivered event should not take a new snapshot")
		})
	}
}

func TestProcessFuturesDepth(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	require.NoError(t, ex.wsHandleUSDTMarginedFuturesData(t.Context(), nil, fixtureFrame(t, "testdata/wsUFuturesPublic.jsonl", "btcusdt@depth5@100ms")), "wsHandleUSDTMarginedFuturesData must not error")
	got, err := storedOrderbookState(ex, testUSDTMarginedFuturesPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "the partial depth snapshot must be stored")
	exp := &orderbookState{
		Bids:         orderbook.Levels{{Price: 85999.9, Amount: 15.739}, {Price: 85999.8, Amount: 0.195}},
		Asks:         orderbook.Levels{{Price: 86000, Amount: 4.613}, {Price: 86000.1, Amount: 0.005}},
		LastUpdateID: 11747004261842,
		LastUpdated:  time.UnixMilli(1791283218111),
		LastPushed:   time.UnixMilli(1791283218113),
	}
	assert.Equal(t, exp, got, "partial depth should load the top levels as a snapshot")
}

func TestIsPartialDepthStream(t *testing.T) {
	t.Parallel()
	for stream, exp := range map[string]bool{
		"btcusdt@depth":                    false,
		"btcusdt@depth@100ms":              false,
		"btcusd_perp@depth@500ms":          false,
		"btcusdt@depth5":                   true,
		"btcusdt@depth10@100ms":            true,
		"btc-261030-85000-c@depth20@500ms": true,
		"btcusdt@rpiDepth@500ms":           false,
		"btcusdt@bookTicker":               false,
	} {
		t.Run(subtestName(stream), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, isPartialDepthStream(stream), "isPartialDepthStream should classify %q", stream)
		})
	}
}

func TestFuturesSymbolTypeAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		symbolType uint64
		connAsset  asset.Item
		exp        asset.Item
		wantErr    error
	}{
		{"unset on a USDⓈ-M connection", 0, asset.USDTMarginedFutures, asset.USDTMarginedFutures, nil},
		{"unset on a COIN-M connection", 0, asset.CoinMarginedFutures, asset.CoinMarginedFutures, nil},
		{"USDⓈ-M on a COIN-M connection", 1, asset.CoinMarginedFutures, asset.USDTMarginedFutures, nil},
		{"COIN-M on a USDⓈ-M connection", 2, asset.USDTMarginedFutures, asset.CoinMarginedFutures, nil},
		{"unknown symbol type", 3, asset.USDTMarginedFutures, asset.Empty, errUnknownSymbolType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := futuresSymbolTypeAsset(tc.symbolType, tc.connAsset)
			assert.ErrorIsf(t, err, tc.wantErr, "futuresSymbolTypeAsset should return the expected error for symbol type %d", tc.symbolType)
			assert.Equalf(t, tc.exp, got, "futuresSymbolTypeAsset should map symbol type %d", tc.symbolType)
		})
	}
}

func TestOwnProductEntries(t *testing.T) {
	t.Parallel()
	entries := []FuturesMarkPrice{{Symbol: "BTCUSDT", SymbolType: 1}, {Symbol: "BTCUSD_PERP", SymbolType: 2}, {Symbol: "ETHUSDT"}, {Symbol: "UNKNOWN", SymbolType: 9}}
	got := ownProductEntries(entries, func(m *FuturesMarkPrice) uint64 { return m.SymbolType }, asset.USDTMarginedFutures)
	assert.Equal(t, []FuturesMarkPrice{{Symbol: "BTCUSDT", SymbolType: 1}, {Symbol: "ETHUSDT"}}, got, "ownProductEntries should keep the connection's product")
}

func TestMatchDerivativesPair(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// The cases share ex, so they run in order
	for _, tc := range []struct {
		name    string
		symbol  string
		a       asset.Item
		exp     currency.Pair
		enabled bool
		wantErr error
	}{
		{"enabled USDⓈ-M pair", "BTCUSDT", asset.USDTMarginedFutures, testUSDTMarginedFuturesPair, true, nil},
		{"disabled USDⓈ-M pair", "ETHUSDT", asset.USDTMarginedFutures, testUSDTMarginedFuturesDisabledPair, false, nil},
		{"enabled COIN-M pair", "BTCUSD_PERP", asset.CoinMarginedFutures, testCoinMarginedFuturesPair, true, nil},
		{"disabled COIN-M pair", "ETHUSD_PERP", asset.CoinMarginedFutures, testCoinMarginedFuturesDisabledPair, false, nil},
		{"enabled options pair", "BTC-261030-85000-C", asset.Options, testOptionsPair, true, nil},
		{"disabled options pair", "ETH-261009-2700-C", asset.Options, testOptionsDisabledPair, false, nil},
		{"COIN-M symbol as USDⓈ-M", "BTCUSD_PERP", asset.USDTMarginedFutures, currency.EMPTYPAIR, false, currency.ErrPairNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pair, enabled, err := ex.matchDerivativesPair(tc.symbol, tc.a)
			assert.ErrorIsf(t, err, tc.wantErr, "matchDerivativesPair should return the expected error for %s %s", tc.a, tc.symbol)
			assert.Equalf(t, tc.exp, pair, "matchDerivativesPair should match %s %s", tc.a, tc.symbol)
			assert.Equalf(t, tc.enabled, enabled, "matchDerivativesPair should report whether %s %s is enabled", tc.a, tc.symbol)
		})
	}
}

func TestDecodeStreamFrame(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		frame   string
		stream  string
		data    string
		event   string
		wantErr error
		errText string
	}{
		{name: "event", frame: `{"stream":"btcusdt@ticker","data":{"e":"24hrTicker"}}`, stream: "btcusdt@ticker", data: `{"e":"24hrTicker"}`, event: "24hrTicker"},
		{name: "event array", frame: `{"stream":"!ticker@arr","data":[{"e":"24hrTicker"},{"e":"24hrTicker"}]}`, stream: "!ticker@arr", data: `[{"e":"24hrTicker"},{"e":"24hrTicker"}]`, event: "24hrTicker"},
		{name: "empty array", frame: `{"stream":"!ticker@arr","data":[]}`, stream: "!ticker@arr", data: `[]`},
		{name: "empty array with a space", frame: `{"stream":"!ticker@arr","data":[ ]}`, stream: "!ticker@arr", data: `[ ]`},
		{name: "without a stream", frame: `{"data":{"e":"24hrTicker"}}`, wantErr: errUnhandledStreamEvent, errText: "frame without a stream"},
		{name: "without data", frame: `{"stream":"btcusdt@ticker"}`, wantErr: errUnhandledStreamEvent, errText: "frame without data"},
		{name: "without an event type", frame: `{"stream":"btcusdt@ticker","data":{"s":"BTCUSDT"}}`, wantErr: errUnhandledStreamEvent, errText: "frame without an event type"},
		{name: "array without an event type", frame: `{"stream":"!ticker@arr","data":["24hrTicker"]}`, wantErr: errUnhandledStreamEvent, errText: "frame without an event type"},
		// jsonparser reports an unreadable first element as missing, as it does for an empty array
		{name: "invalid array", frame: `{"stream":"!ticker@arr","data":[x]}`, wantErr: errUnhandledStreamEvent, errText: "frame with an invalid array"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stream, data, event, err := decodeStreamFrame([]byte(tc.frame))
			require.ErrorIsf(t, err, tc.wantErr, "decodeStreamFrame must return the expected error for %s", tc.frame)
			if tc.wantErr != nil {
				assert.ErrorContainsf(t, err, tc.errText, "decodeStreamFrame should give the reason it rejects %s", tc.frame)
				return
			}
			assert.Equalf(t, tc.stream, stream, "decodeStreamFrame should return the stream of %s", tc.frame)
			assert.Equalf(t, tc.data, string(data), "decodeStreamFrame should return the data of %s", tc.frame)
			assert.Equalf(t, tc.event, event, "decodeStreamFrame should return the event of %s", tc.frame)
		})
	}
}

func TestDerivativesOrderType(t *testing.T) {
	t.Parallel()
	for in, exp := range map[OrderType]order.Type{
		"LIMIT": order.Limit, "MARKET": order.Market, "STOP": order.StopLimit, "STOP_MARKET": order.StopMarket, "TAKE_PROFIT": order.TakeProfitLimit,
		"TAKE_PROFIT_MARKET": order.TakeProfitMarket, "TRAILING_STOP_MARKET": order.TrailingStop, "LIQUIDATION": order.Liquidation,
	} {
		t.Run("order type "+string(in), func(t *testing.T) {
			t.Parallel()
			got, err := derivativesOrderType(in)
			require.NoErrorf(t, err, "derivativesOrderType must not error for %s", in)
			assert.Equalf(t, exp, got, "derivativesOrderType should convert %s", in)
		})
	}
	_, err := derivativesOrderType("ICEBERG")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "derivativesOrderType should reject an unknown type")
}

func TestDerivativesOrderStatus(t *testing.T) {
	t.Parallel()
	for in, exp := range map[string]order.Status{
		"NEW": order.New, "PARTIALLY_FILLED": order.PartiallyFilled, "FILLED": order.Filled, "CANCELED": order.Cancelled, "EXPIRED": order.Expired,
		"EXPIRED_IN_MATCH": order.Expired, "NEW_INSURANCE": order.Liquidated, "NEW_ADL": order.AutoDeleverage,
	} {
		t.Run("status "+in, func(t *testing.T) {
			t.Parallel()
			got, err := derivativesOrderStatus(in)
			require.NoErrorf(t, err, "derivativesOrderStatus must not error for %s", in)
			assert.Equalf(t, exp, got, "derivativesOrderStatus should convert %s", in)
		})
	}
	_, err := derivativesOrderStatus("SUSPENDED")
	assert.ErrorIs(t, err, errUnknownOrderStatus, "derivativesOrderStatus should reject an unknown status")
}

func TestDerivativesTimeInForce(t *testing.T) {
	t.Parallel()
	for in, exp := range map[TimeInForce]order.TimeInForce{
		"GTC": order.GoodTillCancel, "IOC": order.ImmediateOrCancel, "FOK": order.FillOrKill, "GTX": order.PostOnly, "GTD": order.GoodTillDay, "RPI": order.PostOnly,
		"GTE_GTC": order.GoodTillCancel,
	} {
		t.Run("time in force "+string(in), func(t *testing.T) {
			t.Parallel()
			got, err := derivativesTimeInForce(in)
			require.NoErrorf(t, err, "derivativesTimeInForce must not error for %s", in)
			assert.Equalf(t, exp, got, "derivativesTimeInForce should convert %s", in)
		})
	}
	_, err := derivativesTimeInForce("DAY")
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "derivativesTimeInForce should reject an unknown time in force")
}

func TestFuturesPosition(t *testing.T) {
	t.Parallel()
	updated := time.UnixMilli(1564745798938)
	for _, tc := range []struct {
		name         string
		positionSide string
		amount       float64
		direction    order.Side
		closed       bool
	}{
		{"one-way long", "BOTH", 1, order.Long, false},
		{"one-way short", "BOTH", -1, order.Short, false},
		{"one-way closed", "BOTH", 0, order.UnknownSide, true},
		{"hedge long closed", "LONG", 0, order.Long, true},
		{"hedge short", "SHORT", 2, order.Short, false},
		{"no position side", "", -0.1, order.Short, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			amount := decimal.MustFromFloat(tc.amount)
			exp := futures.Position{
				Exchange: "test", Asset: asset.USDTMarginedFutures, Pair: testUSDTMarginedFuturesPair, LatestSize: amount.Abs(), Status: order.Open,
				OpeningDirection: tc.direction, LatestDirection: tc.direction, LastUpdated: updated,
			}
			if tc.closed {
				exp.Status, exp.CloseDate = order.Closed, updated
			}
			got := futuresPosition("test", asset.USDTMarginedFutures, testUSDTMarginedFuturesPair, tc.positionSide, amount, updated)
			assert.Equalf(t, exp, got, "futuresPosition should build the %s %v position", tc.positionSide, tc.amount)
		})
	}
}

func TestProcessAndSendTickers(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	valid := ticker.Price{Last: 1, Bid: 1, Ask: 2, Pair: testUSDTMarginedFuturesPair, ExchangeName: ex.Name, AssetType: asset.USDTMarginedFutures}
	crossed := ticker.Price{Last: 1, Bid: 3, Ask: 3, Pair: testCoinMarginedFuturesPair, ExchangeName: ex.Name, AssetType: asset.CoinMarginedFutures}

	err := ex.processAndSendTicker(t.Context(), &crossed)
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processAndSendTicker should return the processing error")
	assert.Empty(t, relayedPayloads(ex), "processAndSendTicker should not relay a ticker that fails processing")

	require.NoError(t, ex.processAndSendTicker(t.Context(), &valid), "processAndSendTicker must not error")
	assert.Equal(t, []any{&valid}, relayedPayloads(ex), "processAndSendTicker should relay the processed ticker")
	stored, err := ticker.GetTicker(ex.Name, testUSDTMarginedFuturesPair, asset.USDTMarginedFutures)
	require.NoError(t, err, "GetTicker must not error")
	assert.Equal(t, valid.Last, stored.Last, "processAndSendTicker should store the ticker")

	err = ex.processAndSendTickers(t.Context(), []ticker.Price{valid, crossed})
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processAndSendTickers should report the entry that fails processing")
	assert.Equal(t, []any{[]ticker.Price{valid}}, relayedPayloads(ex), "processAndSendTickers should relay only the processed entries")

	err = ex.processAndSendTickers(t.Context(), []ticker.Price{crossed})
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "processAndSendTickers should report a batch that fails processing")
	assert.Empty(t, relayedPayloads(ex), "processAndSendTickers should not relay an empty batch")

	assert.NoError(t, ex.processAndSendTickers(t.Context(), nil), "processAndSendTickers should not error without tickers")
	assert.Empty(t, relayedPayloads(ex), "processAndSendTickers should not relay without tickers")
}

func TestHandleFuturesStreamErrors(t *testing.T) { //nolint:tparallel // the cases share ex and conn, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	conn := &matchRecordingConnection{}
	require.NoError(t, ex.wsHandleUSDTMarginedFuturesData(t.Context(), conn, []byte(`{"result":null,"id":7}`)), "a request reply must be routed")
	assert.Equal(t, []any{int64(7)}, conn.signatures, "a request reply should be routed by its id")

	// The cases share ex and conn, whose relayed payloads are checked after them, so they run in order
	for frame, wantErr := range map[string]error{
		`{"stream":"btcusdt@unknown","data":{"e":"unknownEvent"}}`:                                 errUnhandledStreamEvent,
		`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","s":"BTCUSDT","st":3}}`:               errUnknownSymbolType,
		`{"stream":"btcusdt@depth@100ms","data":{"e":"depthUpdate","s":"BTCUSDT","st":9}}`:         errUnknownSymbolType,
		`{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","s":"BTCUSDT","st":7}}`:               errUnknownSymbolType,
		`{"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","s":"BTCUSDT","st":5}}`:       errUnknownSymbolType,
		`{"stream":"notlisted@kline_1m","data":{"e":"kline","s":"NOTLISTED","k":{"i":"1m"}}}`:      currency.ErrPairNotFound,
		`{"stream":"btcusdt@kline_1m","data":{"e":"kline","s":"BTCUSDT","k":{"i":"1s0"}}}`:         kline.ErrInvalidInterval,
		`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","s":"NOTLISTEDUSDT","st":1}}`:         currency.ErrPairNotFound,
		`{"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","s":"NOTLISTEDUSDT","st":1}}`: currency.ErrPairNotFound,
	} {
		t.Run(subtestName(frame), func(t *testing.T) {
			assert.ErrorIsf(t, ex.wsHandleUSDTMarginedFuturesData(t.Context(), conn, []byte(frame)), wantErr, "wsHandleUSDTMarginedFuturesData should reject %s", frame)
		})
	}
	// The cases share ex and conn, as the cases above do, so they run in order
	for _, frame := range []string{
		`{"stream":"btcusdt@ticker","data":{"e":"24hrTicker","E":"invalid"}}`,
		`{"stream":"!ticker@arr","data":[{"e":"24hrTicker","E":"invalid"}]}`,
		`{"stream":"btcusdt@miniTicker","data":{"e":"24hrMiniTicker","E":"invalid"}}`,
		`{"stream":"!miniTicker@arr","data":[{"e":"24hrMiniTicker","E":"invalid"}]}`,
		`{"stream":"btcusdt@bookTicker","data":{"e":"bookTicker","u":"invalid"}}`,
		`{"stream":"btcusdt@depth@100ms","data":{"e":"depthUpdate","U":"invalid"}}`,
		`{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","a":"invalid"}}`,
		`{"stream":"btcusdt@forceOrder","data":{"e":"forceOrder","E":"invalid"}}`,
		`{"stream":"!contractInfo","data":{"e":"contractInfo","E":"invalid"}}`,
		`{"stream":"btcusdt@kline_1m","data":{"e":"kline","E":"invalid"}}`,
		`{"stream":"btcusdt@markPrice","data":{"e":"markPriceUpdate","E":"invalid"}}`,
		`{"stream":"!markPrice@arr","data":[{"e":"markPriceUpdate","E":"invalid"}]}`,
		`{"stream":"!assetIndex@arr","data":[{"e":"assetIndexUpdate","E":"invalid"}]}`,
	} {
		t.Run(subtestName(frame), func(t *testing.T) {
			assert.Errorf(t, ex.wsHandleUSDTMarginedFuturesData(t.Context(), conn, []byte(frame)), "wsHandleUSDTMarginedFuturesData should report the decoding error of %s", frame)
		})
	}
	require.NoError(t, ex.wsHandleUSDTMarginedFuturesData(t.Context(), conn, []byte(`{"stream":"ethusdt@kline_1m","data":{"e":"kline","s":"ETHUSDT","k":{"i":"1m"}}}`)), "a disabled pair must be skipped")
	assert.Empty(t, relayedPayloads(ex), "nothing should be relayed for rejected or disabled frames")
}

func TestHandleFuturesUserData(t *testing.T) { //nolint:tparallel // the cases share ex and conn, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	conn := &matchRecordingConnection{}
	require.NoError(t, ex.wsHandleUSDTMarginedFuturesUserData(t.Context(), conn, []byte(`{"result":null,"id":3}`)), "a request reply must be routed")
	assert.Equal(t, []any{int64(3)}, conn.signatures, "a request reply should be routed by its id")
	// The cases share ex and conn, whose relayed payloads are checked after them, so they run in order
	for data, wantErr := range map[string]error{
		``:                      errUnhandledStreamEvent,
		`{"E":1}`:               errUnhandledStreamEvent,
		`{"e":"UNKNOWN_EVENT"}`: errUnhandledStreamEvent,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"UP"}}`:                                  order.ErrSideIsInvalid,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"BUY","o":"ICEBERG"}}`:                   order.ErrUnrecognisedOrderType,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"SUSPENDED"}}`:     errUnknownOrderStatus,
		`{"e":"ORDER_TRADE_UPDATE","o":{"s":"BTCUSDT","S":"BUY","o":"LIMIT","X":"NEW","f":"GTE"}}`: order.ErrInvalidTimeInForce,
	} {
		t.Run(subtestName(data), func(t *testing.T) {
			frame := `{"stream":"` + testListenKey + `"}`
			if data != "" {
				frame = `{"stream":"` + testListenKey + `","data":` + data + `}`
			}
			err := ex.wsHandleUSDTMarginedFuturesUserData(t.Context(), conn, []byte(frame))
			require.ErrorIsf(t, err, wantErr, "wsHandleUSDTMarginedFuturesUserData must reject %s", data)
			assert.NotContains(t, err.Error(), testListenKey, "user data errors should never quote the listen key")
		})
	}
	// The cases share ex and conn, as the cases above do, so they run in order
	for _, data := range []string{
		`{"e":"ORDER_TRADE_UPDATE","E":"invalid"}`,
		`{"e":"ALGO_UPDATE","E":"invalid"}`,
		`{"e":"ACCOUNT_UPDATE","E":"invalid"}`,
		`{"e":"MARGIN_CALL","E":"invalid"}`,
		`{"e":"listenKeyExpired","E":"invalid"}`,
	} {
		t.Run(subtestName(data), func(t *testing.T) {
			assert.Errorf(t, ex.wsHandleUSDTMarginedFuturesUserData(t.Context(), conn, []byte(`{"stream":"`+testListenKey+`","data":`+data+`}`)), "wsHandleUSDTMarginedFuturesUserData should reject %s", data)
		})
	}
	assert.Empty(t, relayedPayloads(ex), "nothing should be relayed for rejected frames")
}

func TestWebsocketHandleUSDTMarginedFuturesData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	useTestOrderbookSync(ex, func(context.Context, currency.Pair, asset.Item) (*orderbook.Book, error) {
		return nil, errTestNoSnapshot
	})
	for _, path := range []string{"testdata/wsUFuturesPublic.jsonl", "testdata/wsUFuturesMarket.jsonl"} {
		require.NoError(t, handleFramesWithin(t.Context(), ex.wsHandleUSDTMarginedFuturesData, fixtureFrames(t, path)), "handling the fixture must return")
	}
	miniTicker := ticker.Price{
		Last: 86026.4, High: 86683.9, Low: 84910, Open: 86016.5, BaseVolume: 119489.809, QuoteVolume: 10238568766.13,
		Pair: testUSDTMarginedFuturesPair, ExchangeName: ex.Name, AssetType: asset.USDTMarginedFutures, LastUpdated: time.UnixMilli(1791283239234),
	}
	tick := ticker.Price{
		Last: 86026.4, LastSize: 0.05, VolumeWeightedAveragePrice: 85685.71, High: 86683.9, Low: 84910, Open: 86016.5, PercentChange24Hour: 0.012,
		BaseVolume: 119489.809, QuoteVolume: 10238568766.13, Pair: testUSDTMarginedFuturesPair, ExchangeName: ex.Name, AssetType: asset.USDTMarginedFutures,
		LastUpdated: time.UnixMilli(1791283239234),
	}
	markPrice := FuturesMarkPrice{
		EventType: "markPriceUpdate", EventTime: fixtureTime(1791283239000), Symbol: "BTCUSDT", MarkPrice: 86026.4, IndexPrice: 86059.26913043,
		EstimatedSettlePrice: 86090.55903865, FundingRate: -0.00003629, MarkPriceMovingAverage: 86026.3, NextFundingTime: fixtureTime(1791302400000), SymbolType: 1,
	}
	assetIndex := FuturesAssetIndex{
		EventType: "assetIndexUpdate", EventTime: fixtureTime(1791285204000), Symbol: "BTCUSD", IndexPrice: 86156.1448564, BidBuffer: 0.05, AskBuffer: 0.06,
		BidRate: 81848.33761358, AskRate: 90463.95209922, AutoExchangeBidBuffer: 0.025, AutoExchangeAskBuffer: 0.035, AutoExchangeBidRate: 84002.24123499,
		AutoExchangeAskRate: 88310.04847781,
	}
	exp := []any{
		&FuturesBookTicker{
			EventType: "bookTicker", UpdateID: 11747004261523, EventTime: fixtureTime(1791283218110), TransactionTime: fixtureTime(1791283218109),
			Symbol: "BTCUSDT", Pair: "BTCUSDT", BestBidPrice: 85999.9, BestBidQuantity: 15.739, BestAskPrice: 86000, BestAskQuantity: 4.613, SymbolType: 1,
		},
		&FuturesDepthUpdate{
			EventType: "depthUpdate", EventTime: fixtureTime(1791285277442), TransactionTime: fixtureTime(1791285277441), Symbol: "BTCUSDT", Pair: "BTCUSDT",
			FirstUpdateID: 11747185666708, FinalUpdateID: 11747185738417, PreviousFinalUpdateID: 11747185666537,
			Bids:       orderbook.LevelsArrayPriceAmount{{Price: 84441.6, Amount: 0.019}, {Price: 84444.9, Amount: 0.01}},
			Asks:       orderbook.LevelsArrayPriceAmount{{Price: 86160.1, Amount: 0.32}, {Price: 86160.2, Amount: 0.004}},
			SymbolType: 1,
		},
		[]trade.Data{{
			TID: "3476465318", Exchange: ex.Name, CurrencyPair: testUSDTMarginedFuturesPair, AssetType: asset.USDTMarginedFutures, Side: order.Sell,
			Price: 86000, Amount: 4.577, Timestamp: time.UnixMilli(1791283227688),
		}},
		&FuturesLiquidationOrder{
			EventType: "forceOrder", EventTime: fixtureTime(1791285206184),
			Order: FuturesLiquidationOrderDetail{
				Symbol: "BTCUSDT", Pair: "BTCUSDT", Side: "SELL", OrderType: "LIMIT", TimeInForce: "IOC", OriginalQuantity: 0.042, Price: 85910.1,
				AveragePrice: 86012.3, OrderStatus: "FILLED", OrderLastFilledQuantity: 0.012, OrderFilledAccumulatedQuantity: 0.042,
				OrderTradeTime: fixtureTime(1791285205175), SymbolType: 1,
			},
		},
		[]ticker.Price{miniTicker},
		&miniTicker,
		[]ticker.Price{tick},
		&tick,
		&FuturesCompositeIndex{
			EventType: "compositeIndex", EventTime: fixtureTime(1791285204001), Symbol: "BTCDOMUSDT", Price: 5202.64806682, BaseAssetCategory: "quoteAsset",
			Composition: []FuturesCompositeIndexComposition{
				{BaseAsset: currency.BTC, QuoteAsset: currency.ETH, WeightInQuantity: 73.39600873, WeightInPercentage: 0.447459, IndexPrice: 31.76492999},
				{BaseAsset: currency.BTC, QuoteAsset: currency.SOL, WeightInQuantity: 0.68201057, WeightInPercentage: 0.094744, IndexPrice: 716.7927661},
			},
		},
		&FuturesContinuousKline{
			EventType: "continuous_kline", EventTime: fixtureTime(1791283239106), Pair: "BTCUSDT", ContractType: "PERPETUAL",
			Kline: FuturesContinuousCandle{
				StartTime: fixtureTime(1791283200000), CloseTime: fixtureTime(1791283259999), Interval: "1m", FirstUpdateID: 11747002966903,
				LastUpdateID: 11747005833109, OpenPrice: 85999.9, ClosePrice: 86026.5, HighPrice: 86026.5, LowPrice: 85999.8, Volume: 10.516,
				NumberOfTrades: 447, IsClosed: true, QuoteAssetVolume: 904406.8029, TakerBuyVolume: 7.507, TakerBuyQuoteAssetVolume: 645620.8782, Ignore: 1,
			},
		},
		&FuturesContractInfo{
			EventType: "contractInfo", EventTime: fixtureTime(1669356423908), Symbol: "BTCUSDT", ContractType: "PERPETUAL",
			DeliveryDateTime: fixtureTime(4133404800000), OnboardDateTime: fixtureTime(1569398400000), ContractStatus: "TRADING",
			NotionalBrackets: []FuturesContractInfoBracket{
				{NotionalBracket: 1, BracketFloorNotional: 0.5, BracketNotionalCap: 5000, MaintenanceRatio: 0.01, AuxiliaryNumber: 0.25, MinLeverage: 21, MaxLeverage: 50},
				{NotionalBracket: 2, BracketFloorNotional: 5000, BracketNotionalCap: 25000, MaintenanceRatio: 0.025, AuxiliaryNumber: 75, MinLeverage: 11, MaxLeverage: 20},
			},
			SymbolType: 1,
		},
		kline.Item{
			Exchange: ex.Name, Pair: testUSDTMarginedFuturesPair, Asset: asset.USDTMarginedFutures, Interval: kline.OneMin,
			Candles: []kline.Candle{{Time: time.UnixMilli(1791283200000), Open: 85999.9, High: 86000.1, Low: 85999.8, Close: 86000, Volume: 5.427}},
		},
		&markPrice,
		[]FuturesMarkPrice{markPrice},
		[]FuturesAssetIndex{assetIndex, {
			EventType: "assetIndexUpdate", EventTime: fixtureTime(1791285204000), Symbol: "USDTUSD", IndexPrice: 0.99968617, BidBuffer: 0.0001, AskBuffer: 0.0002,
			BidRate: 0.9995862, AskRate: 0.99978614, AutoExchangeBidBuffer: 0.0001, AutoExchangeAskBuffer: 0.0003, AutoExchangeBidRate: 0.9995862,
			AutoExchangeAskRate: 0.99978614,
		}},
		&assetIndex,
		&FuturesTradingSession{EventType: "EquityUpdate", EventTime: fixtureTime(1791285203779), SessionStartTime: fixtureTime(1791273600000), SessionEndTime: fixtureTime(1791293400000), SessionType: "PRE_MARKET"},
	}
	assert.Equal(t, exp, relayedPayloads(ex), "wsHandleUSDTMarginedFuturesData should relay every stream of the connection's product and enabled pairs")
}

// normaliseBalanceTimes clears the local arrival times UpdateBalance stamps, after checking they are set
func normaliseBalanceTimes(tb testing.TB, payloads []any) {
	tb.Helper()
	for _, p := range payloads {
		subAccounts, ok := p.(accounts.SubAccounts)
		if !ok {
			continue
		}
		for _, s := range subAccounts {
			for c, b := range s.Balances {
				assert.WithinDurationf(tb, time.Now(), b.UpdatedAt, time.Minute, "%s balance should be stamped with its arrival time", c)
				b.UpdatedAt = time.Time{}
				s.Balances[c] = b
			}
		}
	}
}

// isEventFrame reports whether a user data frame carries an event, which some tests exercise separately
func isEventFrame(frame []byte, event string) bool {
	_, _, e, err := decodeStreamFrame(frame)
	return err == nil && e == event
}

func TestWebsocketHandleUSDTMarginedFuturesUserData(t *testing.T) {
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	for _, frame := range fixtureFrames(t, "testdata/wsUFuturesUserData.jsonl") {
		if isEventFrame(frame, "listenKeyExpired") {
			continue // TestProcessListenKeyExpired renews the stream over a connection
		}
		require.NoErrorf(t, ex.wsHandleUSDTMarginedFuturesUserData(t.Context(), nil, frame), "wsHandleUSDTMarginedFuturesUserData must not error for %s", frame)
	}
	payloads := relayedPayloads(ex)
	normaliseBalanceTimes(t, payloads)
	updated := time.UnixMilli(1564745798938)
	exp := []any{
		&FuturesMarginCall{
			EventType: "MARGIN_CALL", EventTime: fixtureTime(1587727187525), CrossWalletBalance: 3.16812045,
			Positions: []FuturesMarginCallPosition{{
				Symbol: "BTCUSDT", PositionSide: "LONG", PositionAmount: 1.327, MarginType: "CROSSED", IsolatedWallet: 0.5, MarkPrice: 86187.17127,
				UnrealisedPNL: -1.166074, MaintenanceMarginRequired: 1.614445,
			}},
		},
		accounts.SubAccounts{{AssetType: asset.USDTMarginedFutures, Balances: accounts.CurrencyBalances{
			currency.USDT: {Currency: currency.USDT, Total: 122624.12345678, Free: 122624.12345678},
			currency.BNB:  {Currency: currency.BNB, Total: 1, Free: 1},
		}}},
		[]futures.Position{
			{
				Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: testUSDTMarginedFuturesPair, PositionMargin: decimal.MustFromFloat(86.1),
				RealisedPNL: decimal.MustFromFloat(200), UnrealisedPNL: decimal.MustFromFloat(1.8245), Status: order.Open,
				OpeningPrice: decimal.MustFromFloat(86100), OpeningDirection: order.Short, LatestSize: decimal.MustFromFloat(-0.02).Abs(),
				LatestDirection: order.Short, LastUpdated: updated,
			},
			{
				Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: testUSDTMarginedFuturesPair, PositionMargin: decimal.MustFromFloat(0),
				RealisedPNL: decimal.MustFromFloat(12.5), UnrealisedPNL: decimal.MustFromFloat(0), Status: order.Closed,
				OpeningPrice: decimal.MustFromFloat(0), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(0).Abs(),
				LatestDirection: order.Long, LastUpdated: updated, CloseDate: updated,
			},
		},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 86100.1, Amount: 0.5, TriggerPrice: 86476.89, AverageExecutedPrice: 86104.2,
			ExecutedAmount: 0.25, Cost: 21526.05, RemainingAmount: 0.25, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "8886774", ClientOrderID: "TEST",
			Type: order.TrailingStop, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.USDTMarginedFutures,
			Date: time.UnixMilli(1568879465650), LastUpdated: time.UnixMilli(1568879465650), Pair: testUSDTMarginedFuturesPair,
			Trades: []order.TradeHistory{{
				Price: 86104.2, Amount: 0.125, Fee: 0.03444168, Exchange: ex.Name, TID: "10125", Type: order.TrailingStop, Side: order.Sell,
				Timestamp: time.UnixMilli(1568879465650), IsMaker: true, FeeAsset: "USDT",
			}},
		},
		&FuturesAccountConfigUpdate{
			EventType: "ACCOUNT_CONFIG_UPDATE", EventTime: fixtureTime(1611646737479), TransactionTime: fixtureTime(1611646737476),
			AccountConfiguration: &FuturesAccountConfiguration{Symbol: "BTCUSDT", Leverage: 25},
		},
		&FuturesAccountConfigUpdate{
			EventType: "ACCOUNT_CONFIG_UPDATE", EventTime: fixtureTime(1611646737480), TransactionTime: fixtureTime(1611646737477),
			UserAccountConfiguration: &FuturesUserAccountConfiguration{MultiAssetsMode: true},
		},
		&FuturesTradeLite{
			EventType: "TRADE_LITE", EventTime: fixtureTime(1721895408092), TransactionTime: fixtureTime(1721895408214), Symbol: "BTCUSDT",
			OriginalQuantity: 0.001, OriginalPrice: 86000.1, IsMaker: true, ClientOrderID: "z8hcUoOsqEdKMeKPSABslD", Side: "BUY",
			LastFilledPrice: 86000.1, OrderLastFilledQuantity: 0.001, TradeID: 109100866, OrderID: 8886775,
		},
		&FuturesConditionalOrderTriggerReject{
			EventType: "CONDITIONAL_ORDER_TRIGGER_REJECT", EventTime: fixtureTime(1685517224945), MessageSendTime: fixtureTime(1685517224955),
			OrderReject: FuturesConditionalOrderReject{Symbol: "BTCUSDT", OrderID: 155618472834, RejectReason: "Due to the order could not be filled immediately, the FOK order has been rejected."},
		},
		&FuturesStrategyUpdate{
			EventType: "STRATEGY_UPDATE", TransactionTime: fixtureTime(1669261797627), EventTime: fixtureTime(1669261797628),
			StrategyUpdate: FuturesStrategy{StrategyID: 176054594, StrategyType: "GRID", StrategyStatus: "NEW", Symbol: "BTCUSDT", UpdateTime: fixtureTime(1669261797627), OpCode: 8007},
		},
		&FuturesGridUpdate{
			EventType: "GRID_UPDATE", TransactionTime: fixtureTime(1669262908216), EventTime: fixtureTime(1669262908218),
			GridUpdate: FuturesGridStrategy{
				StrategyID: 176057039, StrategyType: "GRID", StrategyStatus: "WORKING", Symbol: "BTCUSDT", RealisedPNL: -0.00300716,
				UnmatchedAveragePrice: 86720, UnmatchedQuantity: -0.001, UnmatchedFee: -0.00300716, MatchedPNL: 0.5, UpdateTime: fixtureTime(1669262908197),
			},
		},
		&FuturesAlgoUpdate{
			EventType: "ALGO_UPDATE", TransactionTime: fixtureTime(1750515742297), EventTime: fixtureTime(1750515742303),
			Order: FuturesAlgoUpdateOrder{
				ClientAlgoID: "Q5xaq5EGKgXXa0fD7fs0Ip", AlgoID: 2148719, AlgoType: "CONDITIONAL", OrderType: "TAKE_PROFIT", Symbol: "BTCUSDT",
				Side: "SELL", PositionSide: "BOTH", TimeInForce: "GTC", Quantity: 0.01, AlgoStatus: "TRIGGERED", OrderID: "8886776",
				AverageFillPrice: 86750, ExecutedQuantity: 0.01, ActualOrderType: "LIMIT", TriggerPrice: 86750, OrderPrice: 86750,
				SelfTradePreventionMode: "EXPIRE_MAKER", WorkingType: "CONTRACT_PRICE", PriceMatchMode: "NONE", IsCloseAll: true,
				IsPriceProtected: true, IsReduceOnly: true, TriggerTime: fixtureTime(1750515742290), GoodTillDate: fixtureTime(1750602142297),
				FailedReason: "Reduce Only reject", IsActivated: true,
			},
		},
		&order.Detail{
			TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 86750, Amount: 0.01, TriggerPrice: 86750, AverageExecutedPrice: 86750,
			ExecutedAmount: 0.01, Exchange: ex.Name, OrderID: "algo-2148719", ClientOrderID: "Q5xaq5EGKgXXa0fD7fs0Ip", Type: order.TakeProfitLimit,
			Side: order.Sell, Status: order.Active, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1750515742297),
			LastUpdated: time.UnixMilli(1750515742297), Pair: testUSDTMarginedFuturesPair,
		},
	}
	assert.Equal(t, exp, payloads, "wsHandleUSDTMarginedFuturesUserData should relay every user data event")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
	require.NoError(t, err, "the account update must store the balance")
	assert.Equal(t, 122624.12345678, stored.Total, "the account update should store the wallet balance")
}

// Pairs of symbols listed since the available pairs were stored, as user data derives them. Formatting a pair upper
// case, as the stored format is, marks its codes upper case, so a delivery date has to be marked too
var (
	testCoinMarginedFuturesUnlistedPair = currency.NewPairWithDelimiter("BTCUSD", "261225", currency.UnderscoreDelimiter).Upper()
	testOptionsUnlistedPair             = currency.NewPairWithDelimiter("BTC", "261031-90000-C", currency.DashDelimiter)
)

func TestUserDataPair(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// The cases share ex, whose options pairs are deleted after them, so they run in order
	for _, tc := range []struct {
		symbol  string
		a       asset.Item
		exp     currency.Pair
		wantErr error
	}{
		{"BTCUSDT", asset.USDTMarginedFutures, testUSDTMarginedFuturesPair, nil},
		{"ETHUSD_PERP", asset.CoinMarginedFutures, testCoinMarginedFuturesDisabledPair, nil},
		{"BTC-261030-85000-C", asset.Options, testOptionsPair, nil},
		{"BTCUSD_261225", asset.CoinMarginedFutures, testCoinMarginedFuturesUnlistedPair, nil},
		{"BTC-261031-90000-C", asset.Options, testOptionsUnlistedPair, nil},
		{"NEWCOINUSDT", asset.USDTMarginedFutures, currency.EMPTYPAIR, currency.ErrPairNotFound},
		{"BTCUSDT_261225", asset.USDTMarginedFutures, currency.NewPairWithDelimiter("BTCUSDT", "261225", currency.UnderscoreDelimiter).Upper(), nil},
		{"BTCUSD261225", asset.CoinMarginedFutures, currency.EMPTYPAIR, currency.ErrPairNotFound},
		{"BTC261031C", asset.Options, currency.EMPTYPAIR, currency.ErrPairNotFound},
		{"", asset.Options, currency.EMPTYPAIR, currency.ErrSymbolStringEmpty},
	} {
		t.Run(subtestName(tc.symbol), func(t *testing.T) {
			pair, err := ex.userDataPair(tc.symbol, tc.a)
			require.ErrorIsf(t, err, tc.wantErr, "userDataPair must return the expected error for %s %q", tc.a, tc.symbol)
			assert.Equalf(t, tc.exp, pair, "userDataPair should return the pair of %s %q", tc.a, tc.symbol)
		})
	}
	ex.CurrencyPairs.Delete(asset.Options)
	_, err := ex.userDataPair("BTC-261031-90000-C", asset.Options)
	assert.ErrorIs(t, err, currency.ErrAssetNotFound, "userDataPair should return the error of a missing pair format")
}

func TestUserDataUnknownSymbols(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	updated := time.UnixMilli(1791300000000)
	// The cases share ex, whose relayed payloads each drains, so they run in order
	for _, tc := range []struct {
		name   string
		handle func(context.Context, websocket.Connection, []byte) error
		data   string
		exp    []any
	}{
		{
			name:   "USDⓈ-M order of a symbol without a pair",
			handle: ex.wsHandleUSDTMarginedFuturesUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"NEWCOINUSDT","c":"new-coin","S":"BUY","o":"LIMIT","f":"GTC","q":"5","p":"1.5","ap":"0","sp":"0","x":"NEW","X":"NEW","i":31,"l":"0","z":"0","L":"0","N":"USDT","n":"0","T":1791300000000,"t":0,"b":"7.5","a":"0","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"V":"NONE","pm":"NONE"}}`,
		},
		{
			name:   "USDⓈ-M account update with a position of a symbol without a pair",
			handle: ex.wsHandleUSDTMarginedFuturesUserData,
			data:   `{"e":"ACCOUNT_UPDATE","E":1791300000001,"T":1791300000000,"a":{"m":"ORDER","B":[{"a":"USDT","wb":"1000","cw":"1000","bc":"0"}],"P":[{"s":"NEWCOINUSDT","pa":"5","ep":"1.5","bep":"1.5","cr":"0","up":"0","mt":"cross","iw":"0","ps":"BOTH"},{"s":"BTCUSDT","pa":"0.1","ep":"86000","bep":"86034.4","cr":"0","up":"1.5","mt":"cross","iw":"0","ps":"BOTH"}]}}`,
			exp: []any{
				accounts.SubAccounts{{AssetType: asset.USDTMarginedFutures, Balances: accounts.CurrencyBalances{
					currency.USDT: {Currency: currency.USDT, Total: 1000, Free: 1000},
				}}},
				[]futures.Position{{
					Exchange: ex.Name, Asset: asset.USDTMarginedFutures, Pair: testUSDTMarginedFuturesPair, PositionMargin: decimal.MustFromFloat(0),
					RealisedPNL: decimal.MustFromFloat(0), UnrealisedPNL: decimal.MustFromFloat(1.5), Status: order.Open,
					OpeningPrice: decimal.MustFromFloat(86000), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(0.1).Abs(),
					LatestDirection: order.Long, LastUpdated: updated,
				}},
			},
		},
		{
			name:   "USDⓈ-M account update whose only position has no pair",
			handle: ex.wsHandleUSDTMarginedFuturesUserData,
			data:   `{"e":"ACCOUNT_UPDATE","E":1791300000003,"T":1791300000002,"a":{"m":"ORDER","B":[{"a":"USDT","wb":"990","cw":"990","bc":"0"}],"P":[{"s":"NEWCOINUSDT","pa":"0","ep":"0","bep":"0","cr":"-10","up":"0","mt":"cross","iw":"0","ps":"BOTH"}]}}`,
			exp: []any{accounts.SubAccounts{{AssetType: asset.USDTMarginedFutures, Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 990, Free: 990},
			}}}},
		},
		{
			name:   "COIN-M order of a contract listed since the pairs were stored",
			handle: ex.wsHandleCoinMarginedFuturesUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","o":{"s":"BTCUSD_261225","c":"quarterly","S":"SELL","o":"LIMIT","f":"GTC","q":"10","p":"88000","ap":"0","sp":"0","x":"NEW","X":"NEW","i":32,"l":"0","z":"0","L":"0","ma":"BTC","N":"BTC","n":"0","T":1791300000000,"t":0,"b":"0","a":"0.01136364","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"V":"NONE","pm":"NONE"}}`,
			exp: []any{&order.Detail{
				TimeInForce: order.GoodTillCancel, Price: 88000, Amount: 10, RemainingAmount: 10, FeeAsset: currency.BTC, Exchange: ex.Name,
				OrderID: "32", ClientOrderID: "quarterly", Type: order.Limit, Side: order.Sell, Status: order.New, AssetType: asset.CoinMarginedFutures,
				Date: updated, LastUpdated: updated, Pair: testCoinMarginedFuturesUnlistedPair,
			}},
		},
		{
			name:   "COIN-M account update with positions of unlisted symbols",
			handle: ex.wsHandleCoinMarginedFuturesUserData,
			data:   `{"e":"ACCOUNT_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","a":{"m":"ORDER","B":[{"a":"BTC","wb":"1.5","cw":"1.5","bc":"0"}],"P":[{"s":"BTCUSD_261225","pa":"-10","ep":"88000","bep":"88010","cr":"0","up":"0","mt":"cross","iw":"0","ps":"BOTH"},{"s":"BTCUSD261225","pa":"1","ep":"88000","bep":"88000","cr":"0","up":"0","mt":"cross","iw":"0","ps":"BOTH"}]}}`,
			exp: []any{
				accounts.SubAccounts{{AssetType: asset.CoinMarginedFutures, Balances: accounts.CurrencyBalances{
					currency.BTC: {Currency: currency.BTC, Total: 1.5, Free: 1.5},
				}}},
				[]futures.Position{{
					Exchange: ex.Name, Asset: asset.CoinMarginedFutures, Pair: testCoinMarginedFuturesUnlistedPair, PositionMargin: decimal.MustFromFloat(0),
					RealisedPNL: decimal.MustFromFloat(0), UnrealisedPNL: decimal.MustFromFloat(0), Status: order.Open,
					OpeningPrice: decimal.MustFromFloat(88000), OpeningDirection: order.Short, LatestSize: decimal.MustFromFloat(-10).Abs(),
					LatestDirection: order.Short, LastUpdated: updated,
				}},
			},
		},
		{
			name:   "options order of an option listed since the pairs were stored",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"BTC-261031-90000-C","c":"new-option","S":"BUY","o":"LIMIT","f":"GTC","q":"1","p":"500","ap":"0","x":"NEW","X":"NEW","i":33,"l":"0","z":"0","L":"0","N":"USDT","n":"0","T":1791300000000,"t":0,"b":"1","a":"0","m":false,"R":false,"ot":"LIMIT","rp":"0","V":"NONE"}}`,
			exp: []any{&order.Detail{
				TimeInForce: order.GoodTillCancel, Price: 500, Amount: 1, RemainingAmount: 1, FeeAsset: currency.USDT, Exchange: ex.Name,
				OrderID: "33", ClientOrderID: "new-option", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: asset.Options,
				Date: updated, LastUpdated: updated, Pair: testOptionsUnlistedPair,
			}},
		},
		{
			name:   "options balance and position update with positions of unlisted symbols",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"BALANCE_POSITION_UPDATE","E":1791300000001,"T":1791300000000,"m":"ORDER","B":[{"a":"USDT","b":"9500","bc":"0"}],"P":[{"s":"BTC261031C","c":"2","p":"1000","a":"500"},{"s":"BTC-261031-90000-C","c":"1","p":"500","a":"500"}]}`,
			exp: []any{
				accounts.SubAccounts{{AssetType: asset.Options, Balances: accounts.CurrencyBalances{
					currency.USDT: {Currency: currency.USDT, Total: 9500, Free: 9500},
				}}},
				[]futures.Position{{
					Exchange: ex.Name, Asset: asset.Options, Pair: testOptionsUnlistedPair, NotionalSize: decimal.MustFromFloat(500).Abs(), Status: order.Open,
					OpeningPrice: decimal.MustFromFloat(500), OpeningDirection: order.Long, LatestSize: decimal.MustFromFloat(1).Abs(),
					LatestDirection: order.Long, LastUpdated: updated,
				}},
			},
		},
		{
			name:   "options balance and position update whose only position has no pair",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"BALANCE_POSITION_UPDATE","E":1791300000003,"T":1791300000002,"m":"ORDER","B":[{"a":"USDT","b":"9400","bc":"0"}],"P":[{"s":"BTC261031C","c":"0","p":"0","a":"0"}]}`,
			exp: []any{accounts.SubAccounts{{AssetType: asset.Options, Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 9400, Free: 9400},
			}}}},
		},
		{
			name:   "options order of a symbol without a pair",
			handle: ex.wsHandleOptionsUserData,
			data:   `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"BTC261031C","c":"odd-option","S":"BUY","o":"LIMIT","f":"GTC","q":"1","p":"500","ap":"0","x":"NEW","X":"NEW","i":34,"l":"0","z":"0","L":"0","N":"USDT","n":"0","T":1791300000000,"t":0,"b":"1","a":"0","m":false,"R":false,"ot":"LIMIT","rp":"0","V":"NONE"}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoErrorf(t, tc.handle(t.Context(), nil, []byte(`{"stream":"`+testListenKey+`","data":`+tc.data+`}`)), "handling the %s must not error", tc.name)
			payloads := relayedPayloads(ex)
			normaliseBalanceTimes(t, payloads)
			assert.Equalf(t, tc.exp, payloads, "the %s should be relayed without the entries that have no pair", tc.name)
		})
	}
}

// balanceRefreshReplies are REST balance snapshots, by path, of each derivatives product's USDT or BTC margin asset
var balanceRefreshReplies = map[string]string{
	"/fapi/v3/balance":       `[{"accountAlias":"SgsR","asset":"USDT","balance":"999.96","crossWalletBalance":"999.96","crossUnPnl":"0","availableBalance":"159.96","maxWithdrawAmount":"159.96","marginAvailable":true,"updateTime":1791300000000},{"accountAlias":"SgsR","asset":"BNB","balance":"0.5","crossWalletBalance":"0.5","crossUnPnl":"0","availableBalance":"0.5","maxWithdrawAmount":"0.5","marginAvailable":true,"updateTime":1791200000000}]`,
	"/dapi/v1/account":       `{"assets":[{"asset":"BTC","walletBalance":"1.5","marginBalance":"1.5","availableBalance":"0.4","updateTime":1791300000000},{"asset":"ETH","walletBalance":"2","marginBalance":"2","availableBalance":"2","updateTime":1791200000000}],"positions":[],"canTrade":true,"updateTime":1791300000000}`,
	"/eapi/v1/marginAccount": `{"asset":[{"asset":"USDT","marginBalance":"10099.448","equity":"10094.44662","available":"8725.92524","initialMargin":"1084.52138","maintMargin":"151.00138","unrealizedPNL":"-5.00138"},{"asset":"BTC","marginBalance":"0.1","equity":"0.1","available":"0.1","initialMargin":"0","maintMargin":"0","unrealizedPNL":"0"}],"time":1791300000000}`,
}

// balanceEvent is a user data stream balance event of a derivatives product, with how its handler processes it
type balanceEvent struct {
	a           asset.Item
	code        currency.Code
	path        string
	processedBy func(*Exchange) func(context.Context, []byte) error
	// walletEvent returns the event reporting a wallet balance for code
	walletEvent func(walletBalance float64) []byte
	// eventTotal is a wallet balance the REST snapshot predates
	eventTotal float64
	restTotal  float64
	restFree   float64
	// restOnly is a currency the REST reply has and no event reports, with its wallet balance
	restOnly      currency.Code
	restOnlyTotal float64
	// flatReply is a REST snapshot of the same wallet balance with no margin held, last updated before the events
	flatReply string
	// restOnlyReply is a REST snapshot of restOnly alone, last updated at the time its %d verb takes
	restOnlyReply string
}

// walletBalanceEvent returns a function filling a balance event's wallet balance, its %[1]q verbs, in Binance's form
func walletBalanceEvent(format string) func(float64) []byte {
	return func(walletBalance float64) []byte {
		return fmt.Appendf(nil, format, strconv.FormatFloat(walletBalance, 'f', -1, 64))
	}
}

var balanceEvents = []balanceEvent{
	{
		a: asset.USDTMarginedFutures, code: currency.USDT, path: "/fapi/v3/balance",
		walletEvent: walletBalanceEvent(`{"e":"ACCOUNT_UPDATE","E":1791300000001,"T":1791300000000,"a":{"m":"ORDER","B":[{"a":"USDT","wb":%[1]q,"cw":%[1]q,"bc":"0"}],"P":[]}}`),
		processedBy: func(ex *Exchange) func(context.Context, []byte) error {
			return func(ctx context.Context, data []byte) error {
				return ex.processFuturesAccountUpdate(ctx, data, asset.USDTMarginedFutures)
			}
		},
		eventTotal: 1012, restTotal: 999.96, restFree: 159.96, restOnly: currency.BNB, restOnlyTotal: 0.5,
		flatReply:     `[{"accountAlias":"SgsR","asset":"USDT","balance":"999.96","crossWalletBalance":"999.96","crossUnPnl":"0","availableBalance":"999.96","maxWithdrawAmount":"999.96","marginAvailable":true,"updateTime":1791200000000}]`,
		restOnlyReply: `[{"accountAlias":"SgsR","asset":"BNB","balance":"0.5","crossWalletBalance":"0.5","crossUnPnl":"0","availableBalance":"0.5","maxWithdrawAmount":"0.5","marginAvailable":true,"updateTime":%d}]`,
	},
	{
		a: asset.CoinMarginedFutures, code: currency.BTC, path: "/dapi/v1/account",
		walletEvent: walletBalanceEvent(`{"e":"ACCOUNT_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","a":{"m":"ORDER","B":[{"a":"BTC","wb":%[1]q,"cw":%[1]q,"bc":"0"}],"P":[]}}`),
		processedBy: func(ex *Exchange) func(context.Context, []byte) error {
			return func(ctx context.Context, data []byte) error {
				return ex.processFuturesAccountUpdate(ctx, data, asset.CoinMarginedFutures)
			}
		},
		eventTotal: 1.6, restTotal: 1.5, restFree: 0.4, restOnly: currency.ETH, restOnlyTotal: 2,
		flatReply:     `{"assets":[{"asset":"BTC","walletBalance":"1.5","marginBalance":"1.5","availableBalance":"1.5","updateTime":1791200000000}],"positions":[],"canTrade":true,"updateTime":1791200000000}`,
		restOnlyReply: `{"assets":[{"asset":"ETH","walletBalance":"2","marginBalance":"2","availableBalance":"2","updateTime":%d}],"positions":[],"canTrade":true}`,
	},
	{
		a: asset.Options, code: currency.USDT, path: "/eapi/v1/marginAccount",
		walletEvent: walletBalanceEvent(`{"e":"BALANCE_POSITION_UPDATE","E":1791300000001,"T":1791300000000,"m":"ORDER","B":[{"a":"USDT","b":%[1]q,"bc":"0"}],"P":[]}`),
		processedBy: func(ex *Exchange) func(context.Context, []byte) error {
			return ex.processOptionsBalancePositionUpdate
		},
		eventTotal: 9500, restTotal: 10099.448, restFree: 8725.92524, restOnly: currency.BTC, restOnlyTotal: 0.1,
		flatReply:     `{"asset":[{"asset":"USDT","marginBalance":"10099.448","equity":"10099.448","available":"10099.448","initialMargin":"0","maintMargin":"0","unrealizedPNL":"0"}],"time":1791200000000}`,
		restOnlyReply: `{"asset":[{"asset":"BTC","marginBalance":"0.1","equity":"0.1","available":"0.1","initialMargin":"0","maintMargin":"0","unrealizedPNL":"0"}],"time":%d}`,
	},
}

// serveBalanceRefreshReplies answers the REST balance requests with balanceRefreshReplies
func serveBalanceRefreshReplies(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		reply, ok := balanceRefreshReplies[r.URL.Path]
		if !assert.Truef(t, ok, "%s should not be requested", r.URL.Path) {
			return
		}
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	}
}

// storedBalance returns the exchange's stored balance of a currency
func storedBalance(t *testing.T, ex *Exchange, a asset.Item, code currency.Code) accounts.Balance {
	t.Helper()
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	balance, err := ex.Accounts.GetBalance("", creds, a, code)
	require.NoError(t, err, "GetBalance must not error")
	return balance
}

func TestStreamBalanceKeepsRESTAvailableBalance(t *testing.T) {
	t.Parallel()
	// A REST snapshot, then a stream event carrying the same balance, leaves the available balance as REST reported it,
	// and a later change of wallet balance keeps the margin REST reported held
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			process := tc.processedBy(ex)
			require.NoError(t, process(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
			stored := storedBalance(t, ex, tc.a, tc.code)
			assert.Equal(t, tc.restTotal, stored.Total, "the stream event should store its balance")
			assert.InDelta(t, tc.restFree, stored.Free, 1e-9, "an unchanged stream balance should leave the free balance REST reported")
			require.NoError(t, process(t.Context(), tc.walletEvent(tc.restTotal+1)), "processing the balance event must not error")
			stored = storedBalance(t, ex, tc.a, tc.code)
			assert.InDelta(t, tc.restTotal+1, stored.Total, 1e-9, "the stream event should store its balance")
			assert.InDelta(t, tc.restFree+1, stored.Free, 1e-9, "a changed stream balance should keep the margin REST reported held")
		})
	}
}

func TestStreamBalanceEventFree(t *testing.T) {
	t.Parallel()
	// The events carry no available balance. An unchanged wallet balance keeps the one stored, which unrealised profit
	// can lift above the wallet balance, and a changed one keeps the margin stored as held
	for _, tc := range balanceEvents {
		for _, c := range []struct {
			name   string
			stored accounts.Balance
			events []float64
			free   []float64 // the free balance after each event
		}{
			{name: "profit above the wallet balance", stored: accounts.Balance{Total: 100, Free: 120}, events: []float64{100}, free: []float64{120}},
			{name: "margin held above the wallet balance", stored: accounts.Balance{Total: 100, Hold: 90, Free: 10}, events: []float64{50, 100}, free: []float64{0, 10}},
		} {
			t.Run(tc.a.String()+" "+c.name, func(t *testing.T) {
				t.Parallel()
				ex := newTradingTestExchange(t, func(_ http.ResponseWriter, r *http.Request) {
					assert.Failf(t, "no request should be sent", "%s was requested", r.URL.Path)
				})
				stored := accounts.NewSubAccount(tc.a, "")
				stored.Balances.Set(tc.code, c.stored)
				require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
				for i, walletBalance := range c.events {
					require.NoErrorf(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(walletBalance)), "processing the balance event of %v must not error", walletBalance)
					got := storedBalance(t, ex, tc.a, tc.code)
					assert.Equalf(t, walletBalance, got.Total, "the stream event of %v should store its balance", walletBalance)
					assert.Equalf(t, c.free[i], got.Free, "the stream event of %v should leave the expected free balance", walletBalance)
				}
			})
		}
	}
}

func TestRefreshAvailableBalances(t *testing.T) {
	t.Parallel()
	t.Run("REST snapshot ordering", func(t *testing.T) {
		t.Parallel()
		testRESTBalanceSnapshotOrdering(t, true)
	})
	// A REST snapshot that predates a stream event disagrees with the wallet balance the event reported, so the refresh
	// stores none of it: the event's balance stands, and the margin held stays as the last snapshot that agreed
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
			kept := storedBalance(t, ex, tc.a, tc.code)
			lagging, err := ex.refreshAvailableBalances(t.Context(), tc.a)
			require.NoError(t, err, "refreshAvailableBalances must not error")
			assert.True(t, lagging, "the refresh should report a snapshot disagreeing with the stream so that REST is asked again")
			stored := storedBalance(t, ex, tc.a, tc.code)
			assert.Equal(t, kept.UpdatedAt, stored.UpdatedAt, "the refresh should not stamp a balance it keeps as current")
			assert.Equal(t, tc.eventTotal, stored.Total, "the refresh should keep the wallet balance the stream reported")
			assert.InDelta(t, max(tc.eventTotal-(tc.restTotal-tc.restFree), 0), stored.Free, 1e-9, "the refresh should keep the free balance the event left")
			assert.InDelta(t, tc.restTotal-tc.restFree, stored.Hold, 1e-9, "the refresh should keep the margin held the last snapshot reported")
		})
	}
	t.Run("other account", func(t *testing.T) {
		t.Parallel()
		// The event and snapshot times are the configured account's, so another account's refresh stores what REST reports
		// for that account however recent the configured account's snapshot or event
		for i := range balanceEvents {
			tc := &balanceEvents[i]
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				var changed atomic.Bool
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					reply := balanceReply(tc, 100, 80, 1791300001000)
					if r.Header.Get("X-MBX-APIKEY") == "other-key" {
						free := 150.0
						if changed.Load() {
							free = 125
						}
						reply = balanceReply(tc, 200, free, 1791300000000)
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "the configured account's poll must not error")
				newerEvent := strings.ReplaceAll(strings.ReplaceAll(string(tc.walletEvent(100)), "1791300000000", "1791300002000"), "1791300000001", "1791300002001")
				require.NoError(t, tc.processedBy(ex)(t.Context(), []byte(newerEvent)), "the configured account's event must be processed")
				creds := &accounts.Credentials{Key: "other-key", Secret: "other-secret"}
				ctx := accounts.DeployCredentialsToContext(t.Context(), creds)
				_, err = ex.UpdateAccountBalances(ctx, tc.a)
				require.NoError(t, err, "the other account's poll must not error")
				changed.Store(true)
				lagging, err := ex.refreshAvailableBalances(ctx, tc.a)
				require.NoError(t, err, "the other account's refresh must not error")
				assert.False(t, lagging, "the other account's refresh should not be judged against the configured account's snapshot")
				got, err := ex.Accounts.GetBalance("", creds, tc.a, tc.code)
				require.NoError(t, err, "GetBalance must not error for the other account")
				assert.Equal(t, 125.0, got.Free, "the other account's refresh should store its own available balance")
				assert.Equal(t, 100.0, storedBalance(t, ex, tc.a, tc.code).Total, "the configured account's balance should be left as its event stored it")
			})
		}
	})
}

func TestRefreshKeepsNewerBalance(t *testing.T) {
	t.Parallel()
	// A REST poll stored while a refresh's request is answered is newer than the refresh's snapshot, though both report
	// the same wallet balance
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			entered, release := make(chan struct{}), make(chan struct{})
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
					return
				}
				reply := balanceRefreshReplies[tc.path]
				if requests.Add(1) == 1 {
					close(entered)
					select {
					case <-release:
					case <-time.After(30 * time.Second):
						assert.Fail(t, "the refresh's request should be released")
					}
					reply = tc.flatReply
				}
				_, err := w.Write([]byte(reply))
				assert.NoError(t, err, "Write should not error")
			})
			type refresh struct {
				lagging bool
				err     error
			}
			refreshed := make(chan refresh, 1)
			go func() {
				lagging, err := ex.refreshAvailableBalances(t.Context(), tc.a)
				refreshed <- refresh{lagging, err}
			}()
			select {
			case <-entered:
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the refresh must request the balances")
			}
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			polled := storedBalance(t, ex, tc.a, tc.code)
			close(release)
			select {
			case r := <-refreshed:
				require.NoError(t, r.err, "refreshAvailableBalances must not error")
				assert.False(t, r.lagging, "the refresh should not ask again for a balance stored after its request")
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the refresh must finish")
			}
			stored := storedBalance(t, ex, tc.a, tc.code)
			assert.Equal(t, polled.UpdatedAt, stored.UpdatedAt, "the refresh should leave the newer poll's update time")
			assert.InDelta(t, tc.restFree, stored.Free, 1e-9, "the refresh should keep the newer poll's available balance")
			assert.InDelta(t, tc.restTotal-tc.restFree, stored.Hold, 1e-9, "the refresh should keep the newer poll's margin held")
		})
	}
}

func TestRefreshSkipsSnapshotBeforeEvent(t *testing.T) {
	t.Parallel()
	// A REST snapshot Binance reports as last updated before a stream event predates it, though its wallet balance
	// matches the event's
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
					return
				}
				_, err := w.Write([]byte(tc.flatReply))
				assert.NoError(t, err, "Write should not error")
			})
			stored := accounts.NewSubAccount(tc.a, "")
			stored.Balances.Set(tc.code, accounts.Balance{Total: tc.restTotal, Hold: tc.restTotal - tc.restFree, Free: tc.restFree})
			require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
			require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
			lagging, err := ex.refreshAvailableBalances(t.Context(), tc.a)
			require.NoError(t, err, "refreshAvailableBalances must not error")
			assert.True(t, lagging, "the refresh should report a snapshot older than the stream event so that REST is asked again")
			got := storedBalance(t, ex, tc.a, tc.code)
			assert.InDelta(t, tc.restFree, got.Free, 1e-9, "a snapshot older than the stream event should not replace the available balance")
			assert.InDelta(t, tc.restTotal-tc.restFree, got.Hold, 1e-9, "a snapshot older than the stream event should not replace the margin held")
		})
	}
}

func TestStreamBalanceRefreshOutlivesEventContext(t *testing.T) {
	t.Parallel()
	// The refresh an event starts carries on for later events after that event's context ends
	tc := balanceEvents[0]
	var requests atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
			return
		}
		if requests.Add(1) == 1 {
			close(entered)
			select {
			case <-release:
			case <-time.After(30 * time.Second):
				assert.Fail(t, "the first refresh should be released")
			}
		}
		_, err := w.Write([]byte(balanceRefreshReplies[tc.path]))
		assert.NoError(t, err, "Write should not error")
	})
	setup := ex.spotStreamConnectionSetup(ex.Config)
	setup.SubscriptionsNotRequired = true
	useMockWebsocket(t, ex, setup, func(testing.TB, []byte, *gws.Conn) error { return nil })
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ex.balanceRefresh.interval = 10 * time.Millisecond
	process := tc.processedBy(ex)
	eventCtx, cancel := context.WithCancel(t.Context())
	require.NoError(t, process(eventCtx, tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
	select {
	case <-entered:
	case <-time.After(30 * time.Second):
		require.FailNow(t, "the refresh must request the balances")
	}
	require.NoError(t, process(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
	cancel()
	close(release)
	// REST can still lag the event and cause further retries, so observing exactly two requests depends on scheduling.
	require.Eventually(t, func() bool { return requests.Load() >= 2 }, 30*time.Second, 5*time.Millisecond, "the later event must be refreshed")
	require.Eventually(t, func() bool {
		ex.balanceRefresh.mu.Lock()
		defer ex.balanceRefresh.mu.Unlock()
		return !ex.balanceRefresh.running[tc.a]
	}, 30*time.Second, 5*time.Millisecond, "the owned refresh must finish before the test returns")
}

func TestStreamBalanceRefresh(t *testing.T) {
	t.Parallel()
	// A stream balance event carries no available balance, which a fill's margin changes, so it starts a REST refresh.
	// Events arriving while a refresh runs are covered by one more, and nothing is refreshed while disconnected
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var requests int
			release := make(chan struct{})
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
					return
				}
				mu.Lock()
				requests++
				first := requests == 1
				mu.Unlock()
				if first {
					select {
					case <-release:
					case <-time.After(30 * time.Second):
						assert.Fail(t, "the first refresh should be released")
					}
				}
				_, err := w.Write([]byte(balanceRefreshReplies[tc.path]))
				assert.NoError(t, err, "Write should not error")
			})
			process := tc.processedBy(ex)
			running := func() bool {
				ex.balanceRefresh.mu.Lock()
				defer ex.balanceRefresh.mu.Unlock()
				return ex.balanceRefresh.running[tc.a]
			}
			event := tc.walletEvent(tc.restTotal)
			require.NoError(t, process(t.Context(), event), "processing the balance event must not error")
			assert.False(t, running(), "a balance event should not start a refresh while the websocket is disconnected")

			setup := ex.spotStreamConnectionSetup(ex.Config)
			setup.SubscriptionsNotRequired = true
			useMockWebsocket(t, ex, setup, func(testing.TB, []byte, *gws.Conn) error { return nil })
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.balanceRefresh.interval = 10 * time.Millisecond
			for range 3 {
				require.NoError(t, process(t.Context(), event), "processing the balance event must not error")
			}
			close(release)
			require.Eventually(t, func() bool { return !running() }, 30*time.Second, 5*time.Millisecond, "the refreshes must finish")
			mu.Lock()
			assert.Equal(t, 2, requests, "three events during a refresh should be covered by one more refresh")
			mu.Unlock()
			stored := storedBalance(t, ex, tc.a, tc.code)
			assert.Equal(t, tc.restTotal, stored.Total, "the refresh should keep the wallet balance the stream reported")
			assert.Equal(t, tc.restFree, stored.Free, "the refresh should store the available balance REST reports")
			assert.InDelta(t, tc.restTotal-tc.restFree, stored.Hold, 1e-9, "the refresh should store the margin REST reports held")
			assert.Equal(t, tc.restOnlyTotal, storedBalance(t, ex, tc.a, tc.restOnly).Total, "the refresh should store the wallet balance of a currency without one")
		})
	}
}

func TestStreamBalanceRefreshStopsOnDisconnect(t *testing.T) {
	t.Parallel()
	// A refresh running when the websocket disconnects finishes its request, and the events that arrived meanwhile start
	// no other
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			release := make(chan struct{})
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
					return
				}
				if requests.Add(1) == 1 {
					select {
					case <-release:
					case <-time.After(30 * time.Second):
						assert.Fail(t, "the first refresh should be released")
					}
				}
				_, err := w.Write([]byte(balanceRefreshReplies[tc.path]))
				assert.NoError(t, err, "Write should not error")
			})
			process := tc.processedBy(ex)
			setup := ex.spotStreamConnectionSetup(ex.Config)
			setup.SubscriptionsNotRequired = true
			useMockWebsocket(t, ex, setup, func(testing.TB, []byte, *gws.Conn) error { return nil })
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.balanceRefresh.interval = 10 * time.Millisecond
			require.NoError(t, process(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
			require.Eventually(t, func() bool { return requests.Load() == 1 }, 30*time.Second, 5*time.Millisecond, "the refresh must request the balances")
			require.NoError(t, process(t.Context(), tc.walletEvent(tc.eventTotal)), "processing the balance event must not error")
			require.NoError(t, ex.Websocket.Shutdown(), "Shutdown must not error")
			close(release)
			require.Eventually(t, func() bool {
				ex.balanceRefresh.mu.Lock()
				defer ex.balanceRefresh.mu.Unlock()
				return !ex.balanceRefresh.running[tc.a]
			}, 30*time.Second, 5*time.Millisecond, "the refresh must finish")
			assert.Equal(t, int32(1), requests.Load(), "no refresh should be requested once the websocket disconnects")
		})
	}
}

// connectBalanceRefreshes connects the exchange's websocket, which balance refreshes run while, and shortens the
// interval between them
func connectBalanceRefreshes(t *testing.T, ex *Exchange) (running func(asset.Item) bool) {
	t.Helper()
	setup := ex.spotStreamConnectionSetup(ex.Config)
	setup.SubscriptionsNotRequired = true
	useMockWebsocket(t, ex, setup, func(testing.TB, []byte, *gws.Conn) error { return nil })
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	ex.balanceRefresh.interval = time.Millisecond
	return func(a asset.Item) bool {
		ex.balanceRefresh.mu.Lock()
		defer ex.balanceRefresh.mu.Unlock()
		return ex.balanceRefresh.running[a]
	}
}

func TestStreamBalanceRefreshAsksAgainWhileRESTLags(t *testing.T) {
	t.Parallel()
	// REST can lag the stream, reporting a balance as last updated before the event or with another wallet balance, so the
	// refresh asks again after each interval until it reports the event, at most balanceRefreshAttempts times. A currency
	// the stream does not report is not asked about again, as REST cannot catch up with a stream that never reports it
	for _, tc := range balanceEvents {
		for _, c := range []struct {
			name          string
			walletBalance float64 // the event's
			lagging       int32   // the requests answered with the snapshot from before the event
			requests      int32
			exp           accounts.Balance
			// restOnlyStored is the wallet balance stored for the currency no event reports, when not zero
			restOnlyStored float64
		}{
			{
				name:           "snapshot disagreeing over a currency the stream does not report",
				walletBalance:  tc.restTotal,
				requests:       1,
				exp:            accounts.Balance{Total: tc.restTotal, Hold: tc.restTotal - tc.restFree, Free: tc.restFree},
				restOnlyStored: tc.restOnlyTotal + 1,
			},
			{
				name:          "first snapshot before the event",
				walletBalance: tc.restTotal,
				lagging:       1,
				requests:      2,
				exp:           accounts.Balance{Total: tc.restTotal, Hold: tc.restTotal - tc.restFree, Free: tc.restFree},
			},
			{
				name:          "every snapshot before the event",
				walletBalance: tc.restTotal,
				lagging:       math.MaxInt32,
				requests:      balanceRefreshAttempts,
				exp:           accounts.Balance{Total: tc.restTotal, Free: tc.restTotal},
			},
			{
				name:          "every snapshot of another wallet balance",
				walletBalance: tc.eventTotal,
				requests:      balanceRefreshAttempts,
				exp:           accounts.Balance{Total: tc.eventTotal, Free: tc.eventTotal},
			},
		} {
			t.Run(tc.a.String()+" "+c.name, func(t *testing.T) {
				t.Parallel()
				var requests atomic.Int32
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
						return
					}
					reply := balanceRefreshReplies[tc.path]
					if requests.Add(1) <= c.lagging {
						reply = tc.flatReply
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				running := connectBalanceRefreshes(t, ex)
				stored := accounts.NewSubAccount(tc.a, "")
				stored.Balances.Set(tc.code, accounts.Balance{Total: tc.restTotal + 1, Free: tc.restTotal + 1})
				if c.restOnlyStored != 0 {
					stored.Balances.Set(tc.restOnly, accounts.Balance{Total: c.restOnlyStored, Free: c.restOnlyStored})
				}
				require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
				require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(c.walletBalance)), "processing the balance event must not error")
				require.Eventually(t, func() bool { return !running(tc.a) }, 30*time.Second, time.Millisecond, "the refreshes must finish")
				assert.Equal(t, c.requests, requests.Load(), "the refresh should ask REST again while it lags the stream, a bounded number of times")
				got := storedBalance(t, ex, tc.a, tc.code)
				assert.InDelta(t, c.exp.Total, got.Total, 1e-9, "the refreshes should leave the expected wallet balance")
				assert.InDelta(t, c.exp.Hold, got.Hold, 1e-9, "the refreshes should leave the expected margin held")
				assert.InDelta(t, c.exp.Free, got.Free, 1e-9, "the refreshes should leave the expected free balance")
			})
		}
	}
}

func TestStreamBalanceRefreshAttemptsPerEvent(t *testing.T) {
	t.Parallel()
	// An event arriving while REST lags has refreshes of its own, rather than the attempts the earlier event left
	tc := balanceEvents[0]
	var requests atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
			return
		}
		if requests.Add(1) == 3 {
			close(entered)
			select {
			case <-release:
			case <-time.After(30 * time.Second):
				assert.Fail(t, "the third refresh should be released")
			}
		}
		_, err := w.Write([]byte(tc.flatReply))
		assert.NoError(t, err, "Write should not error")
	})
	running := connectBalanceRefreshes(t, ex)
	process := tc.processedBy(ex)
	require.NoError(t, process(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
	select {
	case <-entered:
	case <-time.After(30 * time.Second):
		require.FailNow(t, "the refresh must ask REST again")
	}
	require.NoError(t, process(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
	close(release)
	require.Eventually(t, func() bool { return !running(tc.a) }, 30*time.Second, time.Millisecond, "the refreshes must finish")
	assert.Equal(t, int32(3+balanceRefreshAttempts), requests.Load(), "the later event should be refreshed balanceRefreshAttempts times")
}

func TestStreamBalanceRefreshWhileConnecting(t *testing.T) {
	t.Parallel()
	// A private connection streams while the manager dials the connections set up after it, so an event arriving then is
	// refreshed as one arriving once every connection is up
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
					return
				}
				requests.Add(1)
				_, err := w.Write([]byte(balanceRefreshReplies[tc.path]))
				assert.NoError(t, err, "Write should not error")
			})
			// A connection whose handshake is held keeps the websocket connecting
			released := make(chan struct{})
			dialling := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				select {
				case <-released:
				case <-time.After(30 * time.Second):
				}
			}))
			t.Cleanup(dialling.Close)
			release := sync.OnceFunc(func() { close(released) })
			t.Cleanup(release)
			setup := ex.spotStreamConnectionSetup(ex.Config)
			setup.SubscriptionsNotRequired = true
			setup.URL = "ws" + strings.TrimPrefix(dialling.URL, "http")
			useWebsocket(t, ex, setup)
			ex.balanceRefresh.interval = time.Millisecond
			connected := make(chan error, 1)
			go func() { connected <- ex.Websocket.Connect(t.Context()) }()
			require.Eventually(t, ex.Websocket.IsConnecting, 30*time.Second, time.Millisecond, "the websocket must be connecting")
			require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
			assert.Eventually(t, func() bool { return requests.Load() > 0 }, 30*time.Second, time.Millisecond, "an event arriving while the websocket connects should be refreshed")
			release()
			select {
			case err := <-connected:
				assert.Error(t, err, "Connect should fail for the handshake the server never completes")
			case <-time.After(30 * time.Second):
				assert.Fail(t, "Connect should return once the handshake is released")
			}
		})
	}
}

func TestStreamBalanceRefreshRetriesFailedRequest(t *testing.T) {
	t.Parallel()
	// A refresh whose REST request fails asks again within its attempts, as one REST still lagging does
	for _, tc := range balanceEvents {
		for _, c := range []struct {
			name     string
			failures int32
			requests int32
			expFree  float64
		}{
			{"one failure", 1, 2, tc.restFree},
			{"every request failing", math.MaxInt32, balanceRefreshAttempts, tc.restTotal},
		} {
			t.Run(tc.a.String()+" "+c.name, func(t *testing.T) {
				t.Parallel()
				var requests atomic.Int32
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					if !assert.Equal(t, tc.path, r.URL.Path, "only the balance endpoint should be requested") {
						return
					}
					if requests.Add(1) <= c.failures {
						w.WriteHeader(http.StatusBadRequest)
						_, err := w.Write([]byte(`{"code":-1001,"msg":"Internal error; unable to process your request. Please try again."}`))
						assert.NoError(t, err, "Write should not error")
						return
					}
					_, err := w.Write([]byte(balanceRefreshReplies[tc.path]))
					assert.NoError(t, err, "Write should not error")
				})
				running := connectBalanceRefreshes(t, ex)
				require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
				require.Eventually(t, func() bool { return !running(tc.a) }, 30*time.Second, time.Millisecond, "the refreshes must finish")
				assert.Equal(t, c.requests, requests.Load(), "a failed refresh should ask again, a bounded number of times")
				assert.InDelta(t, c.expFree, storedBalance(t, ex, tc.a, tc.code).Free, 1e-9, "the refreshes should leave the expected free balance")
			})
		}
	}
}

// storeWatchingExchange is the exchange an accounts store asks for credentials before each balance it stores, which
// lets a test see what has happened by the time of a store
type storeWatchingExchange struct {
	*Exchange
	beforeStore func()
}

func (s *storeWatchingExchange) GetCredentials(ctx context.Context) (*accounts.Credentials, error) {
	s.beforeStore()
	return s.Exchange.GetCredentials(ctx)
}

func TestStreamBalanceRefreshFollowsEventStores(t *testing.T) {
	t.Parallel()
	// The refresh a fill's event starts requests REST once the event's balances are stored: a refresh sent first would
	// have its snapshot of the fill taken as older than them and skipped
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
			running := connectBalanceRefreshes(t, ex)
			stored := accounts.NewSubAccount(tc.a, "")
			stored.Balances.Set(tc.code, accounts.Balance{Total: tc.restTotal + 1, Free: tc.restTotal + 1})
			require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
			var stores atomic.Int32
			ex.Accounts.Exchange = &storeWatchingExchange{Exchange: ex, beforeStore: func() {
				if stores.Add(1) == 1 {
					assert.False(t, running(tc.a), "the refresh should not start before the event's balances are stored")
				}
			}}
			require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
			require.Positive(t, stores.Load(), "the balance event must store its balances")
			require.Eventually(t, func() bool { return !running(tc.a) }, 30*time.Second, time.Millisecond, "the refresh must finish")
			assert.InDelta(t, tc.restFree, storedBalance(t, ex, tc.a, tc.code).Free, 1e-9, "the refresh should store the available balance REST reports")
		})
	}
}

func TestUpdateAccountBalancesKeepsLaterStreamedBalance(t *testing.T) {
	t.Parallel()
	// A balance event arriving while a poll's snapshot is being stored waits for it, so the event's balance stands
	for _, tc := range balanceEvents {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
			process := tc.processedBy(ex)
			require.NoError(t, process(t.Context(), tc.walletEvent(tc.restTotal)), "processing the balance event must not error")
			later := make(chan error, 1)
			var stores atomic.Int32
			ex.Accounts.Exchange = &storeWatchingExchange{Exchange: ex, beforeStore: func() {
				// The poll's store is the first
				if stores.Add(1) == 1 {
					go func() { later <- process(t.Context(), tc.walletEvent(tc.restTotal+1)) }()
				}
			}}
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			select {
			case err := <-later:
				require.NoError(t, err, "processing the later balance event must not error")
			case <-time.After(30 * time.Second):
				require.FailNow(t, "the later balance event must be stored")
			}
			assert.Equal(t, tc.restTotal+1, storedBalance(t, ex, tc.a, tc.code).Total, "the later stream event's balance should stand")
		})
	}
}

func TestBalanceStoresHoldTheStoreLock(t *testing.T) {
	t.Parallel()
	// Events store their balances, and refreshes and polls accept REST snapshots, under the balance store lock, so no
	// event is stored between a snapshot's checks and its store
	for _, tc := range balanceEvents {
		for _, c := range []struct {
			name  string
			store func(*testing.T, *Exchange) error
		}{
			{"balance event", func(t *testing.T, ex *Exchange) error {
				t.Helper()
				return tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal))
			}},
			{"refresh", func(t *testing.T, ex *Exchange) error {
				t.Helper()
				_, err := ex.refreshAvailableBalances(t.Context(), tc.a)
				return err
			}},
			{"poll", func(t *testing.T, ex *Exchange) error {
				t.Helper()
				_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				return err
			}},
		} {
			t.Run(tc.a.String()+" "+c.name, func(t *testing.T) {
				t.Parallel()
				ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
				var stores atomic.Int32
				ex.Accounts.Exchange = &storeWatchingExchange{Exchange: ex, beforeStore: func() {
					stores.Add(1)
					if ex.balanceRefresh.storeMu.TryLock() {
						ex.balanceRefresh.storeMu.Unlock()
						assert.Fail(t, "a balance should be stored under the balance store lock")
					}
				}}
				require.NoError(t, c.store(t, ex), "storing the balances must not error")
				assert.Positive(t, stores.Load(), "the balances should be stored")
			})
		}
	}
}

// credentialsWatchingContext calls onCredentials whenever its context credentials are looked up, as the exchange does
// for each request and store
type credentialsWatchingContext struct {
	context.Context //nolint:containedctx // The context is wrapped to watch its credential lookups
	onCredentials   func()
}

func (c credentialsWatchingContext) Value(key any) any {
	if key == accounts.ContextCredentialsFlag {
		c.onCredentials()
	}
	return c.Context.Value(key)
}

func TestBalanceRequestsOutsideTheStoreLock(t *testing.T) {
	t.Parallel()
	// REST balance requests are made outside the balance store lock, so a slow reply holds up no stream event
	for _, tc := range balanceEvents {
		for _, c := range []struct {
			name    string
			request func(*testing.T, *Exchange) error
		}{
			{"refresh", func(t *testing.T, ex *Exchange) error {
				t.Helper()
				_, err := ex.refreshAvailableBalances(t.Context(), tc.a)
				return err
			}},
			{"poll", func(t *testing.T, ex *Exchange) error {
				t.Helper()
				_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				return err
			}},
		} {
			t.Run(tc.a.String()+" "+c.name, func(t *testing.T) {
				t.Parallel()
				var ex *Exchange
				var requests atomic.Int32
				ex = newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if !ex.balanceRefresh.storeMu.TryLock() {
						assert.Fail(t, "a REST balance request should be made outside the balance store lock")
					} else {
						ex.balanceRefresh.storeMu.Unlock()
					}
					serveBalanceRefreshReplies(t)(w, r)
				})
				require.NoError(t, c.request(t, ex), "the balance request must not error")
				assert.Equal(t, int32(1), requests.Load(), "the balances should be requested once")
			})
		}
	}
}

func TestProcessFuturesOrderTradeUpdate(t *testing.T) {
	t.Parallel()
	t.Run("fills", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			a        asset.Item
			symbol   string
			pair     currency.Pair
			feeAsset currency.Code
			fees     [2]string
			expFees  [2]float64
			expCosts [2]float64
		}{
			{asset.USDTMarginedFutures, "BTCUSDT", testUSDTMarginedFuturesPair, currency.USDT, [2]string{"34.4", "68.8"}, [2]float64{34.4, 68.8}, [2]float64{86000, 258000}},
			{asset.CoinMarginedFutures, "BTCUSD_PERP", testCoinMarginedFuturesPair, currency.BTC, [2]string{"0.00000047", "0.00000093"}, [2]float64{0.00000047, 0.00000093}, [2]float64{}},
		} {
			t.Run(tc.a.String(), func(t *testing.T) {
				t.Parallel()
				ex := newDerivativesTestExchange(t)
				// A limit buy of 3 is placed, then filled by two trades
				for _, u := range []struct {
					execution, status, last, cumulative, averagePrice, commission string
					tradeID                                                       uint64
					tradeTime                                                     int64
				}{
					{"NEW", "NEW", "0", "0", "0", "0", 0, 1791300000000},
					{"TRADE", "PARTIALLY_FILLED", "1", "1", "86000", tc.fees[0], 5001, 1791300000100},
					{"TRADE", "FILLED", "2", "3", "86000", tc.fees[1], 5002, 1791300000200},
				} {
					frame := fmt.Appendf(nil, `{"stream":%q,"data":{"e":"ORDER_TRADE_UPDATE","E":%d,"T":%d,"o":{"s":%q,"c":"two-fills","S":"BUY","o":"LIMIT","f":"GTC","q":"3","p":"86000","ap":%q,"sp":"0","x":%q,"X":%q,"i":4242,"l":%q,"z":%q,"L":"86000","N":%q,"n":%q,"T":%d,"t":%d,"b":"0","a":"0","m":true,"R":false,"wt":"CONTRACT_PRICE","ot":"LIMIT","ps":"BOTH","cp":false,"rp":"0","pP":false,"V":"NONE","pm":"NONE"}}}`,
						testListenKey, u.tradeTime+1, u.tradeTime, tc.symbol, u.averagePrice, u.execution, u.status, u.last, u.cumulative, tc.feeAsset, u.commission, u.tradeTime, u.tradeID)
					require.NoErrorf(t, ex.handleFuturesUserData(t.Context(), nil, frame, tc.a), "handleFuturesUserData must not error for the %s update", u.status)
				}
				trades := []order.TradeHistory{
					{
						Price: 86000, Amount: 1, Fee: tc.expFees[0], Exchange: ex.Name, TID: "5001", Type: order.Limit, Side: order.Buy,
						Timestamp: time.UnixMilli(1791300000100), IsMaker: true, FeeAsset: tc.feeAsset.String(),
					},
					{
						Price: 86000, Amount: 2, Fee: tc.expFees[1], Exchange: ex.Name, TID: "5002", Type: order.Limit, Side: order.Buy,
						Timestamp: time.UnixMilli(1791300000200), IsMaker: true, FeeAsset: tc.feeAsset.String(),
					},
				}
				exp := []any{
					&order.Detail{
						TimeInForce: order.GoodTillCancel, Price: 86000, Amount: 3, RemainingAmount: 3, FeeAsset: tc.feeAsset, Exchange: ex.Name,
						OrderID: "4242", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Buy, Status: order.New, AssetType: tc.a,
						Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000), Pair: tc.pair,
					},
					&order.Detail{
						TimeInForce: order.GoodTillCancel, Price: 86000, Amount: 3, AverageExecutedPrice: 86000, ExecutedAmount: 1, Cost: tc.expCosts[0],
						RemainingAmount: 2, FeeAsset: tc.feeAsset, Exchange: ex.Name, OrderID: "4242", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Buy,
						Status: order.PartiallyFilled, AssetType: tc.a, Date: time.UnixMilli(1791300000100), LastUpdated: time.UnixMilli(1791300000100),
						Pair: tc.pair, Trades: trades[:1],
					},
					&order.Detail{
						TimeInForce: order.GoodTillCancel, Price: 86000, Amount: 3, AverageExecutedPrice: 86000, ExecutedAmount: 3, Cost: tc.expCosts[1],
						FeeAsset: tc.feeAsset, Exchange: ex.Name, OrderID: "4242", ClientOrderID: "two-fills", Type: order.Limit, Side: order.Buy, Status: order.Filled,
						AssetType: tc.a, Date: time.UnixMilli(1791300000200), LastUpdated: time.UnixMilli(1791300000200), CloseTime: time.UnixMilli(1791300000200),
						Pair: tc.pair, Trades: trades[1:],
					},
				}
				payloads := relayedPayloads(ex)
				require.Equal(t, exp, payloads, "handleFuturesUserData must relay each update with its own fill")

				// The engine stores the first detail of an order, merges later ones into it and tracks the merged order's
				// position, which needs the order's date
				details := make([]*order.Detail, len(payloads))
				for i := range payloads {
					var ok bool
					details[i], ok = payloads[i].(*order.Detail)
					require.True(t, ok, "every payload must be an order detail")
				}
				stored := details[0].Copy()
				controller := futures.SetupPositionController()
				require.NoError(t, controller.TrackNewOrder(stored.CopyToPointer()), "TrackNewOrder must accept an order first seen on the stream")
				for _, d := range details[1:] {
					require.NoError(t, stored.UpdateOrderFromDetail(d), "UpdateOrderFromDetail must not error")
					require.NoError(t, controller.TrackNewOrder(stored.CopyToPointer()), "TrackNewOrder must accept the order's updates")
				}
				assert.Equal(t, trades, stored.Trades, "the stored order should keep every fill with its own commission")
				assert.Zero(t, stored.Fee, "the stored order should leave the commissions to its trades")
				assert.Equal(t, tc.expCosts[1], stored.Cost, "the stored order should keep the cost of its latest update")
				positions, err := controller.GetPositionsForExchange(ex.Name, tc.a, tc.pair)
				require.NoError(t, err, "GetPositionsForExchange must not error")
				require.Len(t, positions, 1, "the order must open one position")
				assert.Equal(t, time.UnixMilli(1791300000000), positions[0].OpeningDate, "the position should open when the order was placed")
			})
		}
	})

	t.Run("liquidation", func(t *testing.T) {
		t.Parallel()
		ex := newDerivativesTestExchange(t)
		// A liquidation closes the position with an order of its own, whose execution is CALCULATED rather than TRADE
		data := `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"o":{"s":"BTCUSDT","c":"autoclose-1791300000000","S":"SELL","o":"LIQUIDATION","f":"IOC","q":"0.1","p":"84000","ap":"84100","sp":"0","x":"CALCULATED","X":"FILLED","i":8886780,"l":"0.1","z":"0.1","L":"84100","N":"USDT","n":"42.05","T":1791300000000,"t":10200,"b":"0","a":"0","m":false,"R":false,"wt":"CONTRACT_PRICE","ot":"LIQUIDATION","ps":"BOTH","cp":false,"rp":"-190","pP":false,"si":0,"ss":0,"V":"NONE","pm":"NONE","gtd":0,"er":"0"}}`
		require.NoError(t, ex.wsHandleUSDTMarginedFuturesUserData(t.Context(), nil, []byte(`{"stream":"`+testListenKey+`","data":`+data+`}`)), "wsHandleUSDTMarginedFuturesUserData must not error")
		exp := []any{&order.Detail{
			TimeInForce: order.ImmediateOrCancel, Price: 84000, Amount: 0.1, AverageExecutedPrice: 84100, ExecutedAmount: 0.1, Cost: 8410,
			FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "8886780", ClientOrderID: "autoclose-1791300000000", Type: order.Liquidation, Side: order.Sell,
			Status: order.Filled, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000),
			CloseTime: time.UnixMilli(1791300000000), Pair: testUSDTMarginedFuturesPair,
			Trades: []order.TradeHistory{{
				Price: 84100, Amount: 0.1, Fee: 42.05, Exchange: ex.Name, TID: "10200", Type: order.Liquidation, Side: order.Sell,
				Timestamp: time.UnixMilli(1791300000000), FeeAsset: "USDT",
			}},
		}}
		assert.Equal(t, exp, relayedPayloads(ex), "wsHandleUSDTMarginedFuturesUserData should relay a liquidation's execution as the order's trade")
	})
}

func TestFuturesOrderTradeUpdateTrailingStop(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	// The REST conversions of the same trailing stops, which the stream must agree with
	var usdtMarginedOrder USDTMarginedFuturesOrderResponse
	require.NoError(t, json.Unmarshal([]byte(`{"orderId":777,"symbol":"BTCUSDT","status":"NEW","clientOrderId":"trail","price":"0","avgPrice":"0","origQty":"0.3","executedQty":"0","cumQuote":"0","timeInForce":"GTC","type":"TRAILING_STOP_MARKET","origType":"TRAILING_STOP_MARKET","reduceOnly":true,"side":"SELL","positionSide":"BOTH","stopPrice":"86100.1","activatePrice":"86476.8","priceRate":"5.0","time":1791300000000,"updateTime":1791300000000}`), &usdtMarginedOrder), "Unmarshal must not error")
	usdtMarginedDetail, err := ex.usdtMarginedFuturesOrderDetail(&usdtMarginedOrder, testUSDTMarginedFuturesPair)
	require.NoError(t, err, "usdtMarginedFuturesOrderDetail must not error")
	require.Equal(t, 86476.8, usdtMarginedDetail.TriggerPrice, "the REST USDⓈ-M conversion must trigger a trailing stop at its activation price")
	var coinMarginedOrder CoinMarginedFuturesOrderDetailResponse
	require.NoError(t, json.Unmarshal([]byte(`{"orderId":778,"symbol":"BTCUSD_PERP","pair":"BTCUSD","status":"NEW","clientOrderId":"trail","price":"0","avgPrice":"0","origQty":"2","executedQty":"0","cumBase":"0","timeInForce":"GTC","type":"TRAILING_STOP_MARKET","origType":"TRAILING_STOP_MARKET","reduceOnly":true,"side":"SELL","positionSide":"BOTH","stopPrice":"86100.1","activatePrice":"86476.8","priceRate":"5.0","time":1791300000000,"updateTime":1791300000000}`), &coinMarginedOrder), "Unmarshal must not error")
	coinMarginedDetail, err := ex.coinMarginedFuturesOrderDetail(&coinMarginedOrder, testCoinMarginedFuturesPair)
	require.NoError(t, err, "coinMarginedFuturesOrderDetail must not error")
	require.Equal(t, 86476.8, coinMarginedDetail.TriggerPrice, "the REST COIN-M conversion must trigger a trailing stop at its activation price")
	// A triggered trailing stop reports MARKET as its type and keeps its original type
	usdtMarginedOrder.Type, coinMarginedOrder.OrderType = "MARKET", "MARKET"
	usdtMarginedDetail, err = ex.usdtMarginedFuturesOrderDetail(&usdtMarginedOrder, testUSDTMarginedFuturesPair)
	require.NoError(t, err, "usdtMarginedFuturesOrderDetail must not error for a triggered trailing stop")
	require.Equal(t, 86476.8, usdtMarginedDetail.TriggerPrice, "the REST USDⓈ-M conversion must keep a triggered trailing stop's activation price")
	coinMarginedDetail, err = ex.coinMarginedFuturesOrderDetail(&coinMarginedOrder, testCoinMarginedFuturesPair)
	require.NoError(t, err, "coinMarginedFuturesOrderDetail must not error for a triggered trailing stop")
	require.Equal(t, 86476.8, coinMarginedDetail.TriggerPrice, "the REST COIN-M conversion must keep a triggered trailing stop's activation price")

	// The cases share ex, whose relayed payloads each drains, so they run in order
	for _, tc := range []struct {
		name string
		a    asset.Item
		data string
		exp  *order.Detail
	}{
		{
			name: "placed COIN-M trailing stop",
			a:    asset.CoinMarginedFutures,
			data: `{"e":"ORDER_TRADE_UPDATE","E":1791300000001,"T":1791300000000,"i":"SfsR","o":{"s":"BTCUSD_PERP","c":"trail","S":"SELL","o":"TRAILING_STOP_MARKET","f":"GTC","q":"2","p":"0","ap":"0","sp":"86100.1","x":"NEW","X":"NEW","i":778,"l":"0","z":"0","L":"0","ma":"BTC","N":"BTC","n":"0","T":1791300000000,"t":0,"b":"0","a":"0","m":false,"R":true,"wt":"CONTRACT_PRICE","ot":"TRAILING_STOP_MARKET","ps":"BOTH","cp":false,"AP":"86476.8","cr":"5.0","pP":false,"rp":"0","V":"NONE","pm":"NONE"}}`,
			exp: &order.Detail{
				TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 2, TriggerPrice: 86476.8, RemainingAmount: 2, FeeAsset: currency.BTC,
				Exchange: ex.Name, OrderID: "778", ClientOrderID: "trail", Type: order.TrailingStop, Side: order.Sell, Status: order.New,
				AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000), Pair: testCoinMarginedFuturesPair,
			},
		},
		{
			name: "triggered COIN-M trailing stop",
			a:    asset.CoinMarginedFutures,
			data: `{"e":"ORDER_TRADE_UPDATE","E":1791300000501,"T":1791300000500,"i":"SfsR","o":{"s":"BTCUSD_PERP","c":"trail","S":"SELL","o":"MARKET","f":"GTC","q":"2","p":"0","ap":"86400.5","sp":"86100.1","x":"TRADE","X":"PARTIALLY_FILLED","i":778,"l":"1","z":"1","L":"86400.5","ma":"BTC","N":"BTC","n":"0.00000046","T":1791300000500,"t":9001,"b":"0","a":"0","m":false,"R":true,"wt":"CONTRACT_PRICE","ot":"TRAILING_STOP_MARKET","ps":"BOTH","cp":false,"AP":"86476.8","cr":"5.0","pP":false,"rp":"0","V":"NONE","pm":"NONE"}}`,
			exp: &order.Detail{
				TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 2, TriggerPrice: 86476.8, AverageExecutedPrice: 86400.5, ExecutedAmount: 1,
				RemainingAmount: 1, FeeAsset: currency.BTC, Exchange: ex.Name, OrderID: "778", ClientOrderID: "trail", Type: order.Market,
				Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.CoinMarginedFutures, Date: time.UnixMilli(1791300000500),
				LastUpdated: time.UnixMilli(1791300000500), Pair: testCoinMarginedFuturesPair,
				Trades: []order.TradeHistory{{
					Price: 86400.5, Amount: 1, Fee: 0.00000046, Exchange: ex.Name, TID: "9001", Type: order.Market, Side: order.Sell,
					Timestamp: time.UnixMilli(1791300000500), FeeAsset: "BTC",
				}},
			},
		},
		{
			name: "triggered USDⓈ-M trailing stop",
			a:    asset.USDTMarginedFutures,
			data: `{"e":"ORDER_TRADE_UPDATE","E":1791300000501,"T":1791300000500,"o":{"s":"BTCUSDT","c":"trail","S":"SELL","o":"MARKET","f":"GTC","q":"0.3","p":"0","ap":"86400.5","sp":"86100.1","x":"TRADE","X":"PARTIALLY_FILLED","i":777,"l":"0.1","z":"0.1","L":"86400.5","N":"USDT","n":"3.45602","T":1791300000500,"t":9002,"b":"0","a":"0","m":false,"R":true,"wt":"CONTRACT_PRICE","ot":"TRAILING_STOP_MARKET","ps":"BOTH","cp":false,"AP":"86476.8","cr":"5.0","pP":false,"si":0,"ss":0,"rp":"0","V":"NONE","pm":"NONE","gtd":0,"er":"0"}}`,
			exp: &order.Detail{
				TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 0.3, TriggerPrice: 86476.8, AverageExecutedPrice: 86400.5,
				ExecutedAmount: 0.1, Cost: 8640.05, RemainingAmount: 0.2, FeeAsset: currency.USDT, Exchange: ex.Name, OrderID: "777", ClientOrderID: "trail",
				Type: order.Market, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.USDTMarginedFutures,
				Date: time.UnixMilli(1791300000500), LastUpdated: time.UnixMilli(1791300000500), Pair: testUSDTMarginedFuturesPair,
				Trades: []order.TradeHistory{{
					Price: 86400.5, Amount: 0.1, Fee: 3.45602, Exchange: ex.Name, TID: "9002", Type: order.Market, Side: order.Sell,
					Timestamp: time.UnixMilli(1791300000500), FeeAsset: "USDT",
				}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoErrorf(t, ex.handleFuturesUserData(t.Context(), nil, []byte(`{"stream":"`+testListenKey+`","data":`+tc.data+`}`), tc.a), "handleFuturesUserData must not error for the %s", tc.name)
			assert.Equalf(t, []any{tc.exp}, relayedPayloads(ex), "the %s should trigger at its activation price", tc.name)
		})
	}
}

func TestProcessFuturesAlgoUpdate(t *testing.T) { //nolint:tparallel // the cases share ex, so they run in order while the test runs in parallel
	t.Parallel()
	ex := newDerivativesTestExchange(t)
	algoUpdate := func(symbol, side, orderType, timeInForce, status string) []byte {
		return fmt.Appendf(nil, `{"e":"ALGO_UPDATE","T":1791300000000,"E":1791300000006,"o":{"caid":"stop-loss","aid":2148720,"at":"CONDITIONAL","o":%q,"s":%q,"S":%q,"ps":"BOTH","f":%q,"q":"0.01","X":%q,"ai":"","tp":"85000","p":"0","V":"EXPIRE_MAKER","wt":"MARK_PRICE","pm":"NONE","cp":false,"pP":true,"R":true,"tt":0,"gtd":0,"rm":"","ia":false}}`,
			orderType, symbol, side, timeInForce, status)
	}
	require.NoError(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTE_GTC", "CANCELED"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate must not error")
	payloads := relayedPayloads(ex)
	require.Len(t, payloads, 2, "processFuturesAlgoUpdate must relay the event and its algo order")
	assert.IsType(t, &FuturesAlgoUpdate{}, payloads[0], "processFuturesAlgoUpdate should relay the event first")
	exp := &order.Detail{
		TimeInForce: order.GoodTillCancel, ReduceOnly: true, Amount: 0.01, RemainingAmount: 0.01, TriggerPrice: 85000, Exchange: ex.Name,
		OrderID: "algo-2148720", ClientOrderID: "stop-loss", Type: order.StopMarket, Side: order.Sell, Status: order.Cancelled,
		AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791300000000), LastUpdated: time.UnixMilli(1791300000000),
		CloseTime: time.UnixMilli(1791300000000), Pair: testUSDTMarginedFuturesPair,
	}
	assert.Equal(t, exp, payloads[1], "processFuturesAlgoUpdate should relay the algo order identified by its algo order ID")
	rest, err := ex.algoOrderDetail(asset.USDTMarginedFutures, &USDTMarginedFuturesAlgoOrderBase{
		AlgoID: 2148720, ClientAlgoID: "stop-loss", AlgoType: "CONDITIONAL", OrderType: "STOP_MARKET", Symbol: "BTCUSDT", Side: "SELL",
		PositionSide: "BOTH", TimeInForce: "GTE_GTC", Quantity: 0.01, AlgoStatus: "CANCELED", TriggerPrice: 85000, ReduceOnly: true,
		CreateTime: types.Time(time.UnixMilli(1791300000000)), UpdateTime: types.Time(time.UnixMilli(1791300000000)),
	}, 0, testUSDTMarginedFuturesPair)
	require.NoError(t, err, "algoOrderDetail must not error")
	assert.Equal(t, &rest, payloads[1], "processFuturesAlgoUpdate should convert the algo order as the REST algo order queries do")

	t.Run("open order", func(t *testing.T) {
		require.NoError(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTC", "NEW"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate must not error for a new algo order")
		openPayloads := relayedPayloads(ex)
		require.Len(t, openPayloads, 2, "processFuturesAlgoUpdate must relay the event and its algo order")
		d, ok := openPayloads[1].(*order.Detail)
		require.True(t, ok, "the second payload must be an order detail")
		assert.Zero(t, d.CloseTime, "an open algo order should carry no close time")
	})

	// The order manager holds a conditional order SubmitOrder placed under its algo order ID, so the update reaches it
	update, ok := payloads[1].(*order.Detail)
	require.True(t, ok, "the second payload must be an order detail")
	submitted := order.Detail{
		Amount: 0.01, TriggerPrice: 85000, Exchange: ex.Name, OrderID: "algo-2148720", ClientOrderID: "stop-loss", Type: order.StopMarket,
		Side: order.Sell, Status: order.New, AssetType: asset.USDTMarginedFutures, Date: time.UnixMilli(1791300000000), Pair: testUSDTMarginedFuturesPair,
	}
	require.NoError(t, submitted.UpdateOrderFromDetail(update), "UpdateOrderFromDetail must not error")
	assert.Equal(t, order.Cancelled, submitted.Status, "the submitted conditional order should take the update's status")

	// A triggered order's update reports what it executed, which leaves the rest of its quantity; a close position order
	// is placed without a quantity, so its fills leave nothing. The cases share ex, whose relayed payloads each drains,
	// so they run in order
	for _, tc := range []struct {
		name               string
		quantity, executed string
		exp                float64
	}{
		{"partly executed", "0.01", "0.004", 0.006},
		{"fully executed", "0.01", "0.01", 0},
		{"close position", "0", "0.002", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := fmt.Appendf(nil, `{"e":"ALGO_UPDATE","T":1791300000000,"E":1791300000006,"o":{"caid":"stop-loss","aid":2148720,"at":"CONDITIONAL","o":"STOP_MARKET","s":"BTCUSDT","S":"SELL","ps":"BOTH","f":"GTC","q":%q,"X":"FINISHED","ai":"7","ap":"84990","aq":%q,"act":"MARKET","tp":"85000","p":"0","cp":%t}}`,
				tc.quantity, tc.executed, tc.quantity == "0")
			require.NoErrorf(t, ex.processFuturesAlgoUpdate(t.Context(), data, asset.USDTMarginedFutures), "processFuturesAlgoUpdate must not error for %s of %s executed", tc.executed, tc.quantity)
			payloads = relayedPayloads(ex)
			require.Lenf(t, payloads, 2, "processFuturesAlgoUpdate must relay the event and its algo order for %s of %s executed", tc.executed, tc.quantity)
			d, ok := payloads[1].(*order.Detail)
			require.Truef(t, ok, "the second payload must be an order detail for %s of %s executed", tc.executed, tc.quantity)
			assert.Equalf(t, tc.executed, strconv.FormatFloat(d.ExecutedAmount, 'f', -1, 64), "processFuturesAlgoUpdate should relay the executed amount %s", tc.executed)
			assert.Equalf(t, tc.exp, d.RemainingAmount, "processFuturesAlgoUpdate should relay what remains of %s after %s executed", tc.quantity, tc.executed)
		})
	}

	// The cases share ex, whose relayed payloads each drains, so they run in order
	for _, tc := range []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{"invalid side", algoUpdate("BTCUSDT", "UP", "STOP_MARKET", "GTC", "NEW"), order.ErrSideIsInvalid},
		{"unknown order type", algoUpdate("BTCUSDT", "SELL", "ICEBERG", "GTC", "NEW"), order.ErrUnrecognisedOrderType},
		{"unknown time in force", algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "DAY", "NEW"), order.ErrInvalidTimeInForce},
		{"unknown status", algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTC", "SUSPENDED"), errUnknownOrderStatus},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.ErrorIsf(t, ex.processFuturesAlgoUpdate(t.Context(), tc.data, asset.USDTMarginedFutures), tc.wantErr, "processFuturesAlgoUpdate should reject %s", tc.data)
			payloads = relayedPayloads(ex)
			require.Lenf(t, payloads, 1, "processFuturesAlgoUpdate must relay only the event of %s", tc.data)
			assert.IsTypef(t, &FuturesAlgoUpdate{}, payloads[0], "processFuturesAlgoUpdate should relay the event of %s", tc.data)
		})
	}

	require.NoError(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("NEWCOINUSDT", "SELL", "STOP_MARKET", "GTC", "NEW"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate must not error for a symbol without a pair")
	payloads = relayedPayloads(ex)
	require.Len(t, payloads, 1, "processFuturesAlgoUpdate must relay only the event of a symbol without a pair")
	assert.IsType(t, &FuturesAlgoUpdate{}, payloads[0], "processFuturesAlgoUpdate should relay the event of a symbol without a pair")

	assert.Error(t, ex.processFuturesAlgoUpdate(t.Context(), []byte(`{"e":"ALGO_UPDATE","E":"invalid"}`), asset.USDTMarginedFutures), "processFuturesAlgoUpdate should report a decoding error")
	assert.Empty(t, relayedPayloads(ex), "processFuturesAlgoUpdate should relay nothing it cannot decode")

	for len(ex.Websocket.DataHandler.C) < cap(ex.Websocket.DataHandler.C) {
		require.NoError(t, ex.Websocket.DataHandler.Send(t.Context(), "backlog"), "DataHandler must accept the backlog")
	}
	assert.Error(t, ex.processFuturesAlgoUpdate(t.Context(), algoUpdate("BTCUSDT", "SELL", "STOP_MARKET", "GTC", "NEW"), asset.USDTMarginedFutures), "processFuturesAlgoUpdate should return the error of an event it cannot relay")
}

func TestFuturesPayloadDecoding(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		fixture string
		key     string
		target  any
		exp     any
	}{
		{
			name: "ticker", fixture: "testdata/wsUFuturesMarket.jsonl", key: "btcusdt@ticker", target: new(FuturesTicker),
			exp: &FuturesTicker{
				EventType: "24hrTicker", EventTime: fixtureTime(1791283239234), Symbol: "BTCUSDT", Pair: "BTCUSDT", PriceChange: 9.9, PriceChangePercent: 0.012,
				WeightedAveragePrice: 85685.71, LastPrice: 86026.4, LastQuantity: 0.05, OpenPrice: 86016.5, HighPrice: 86683.9, LowPrice: 84910,
				TotalTradedBaseAssetVolume: 119489.809, TotalTradedQuoteAssetVolume: 10238568766.13, StatisticsOpenTime: fixtureTime(1791196800000),
				StatisticsCloseTime: fixtureTime(1791283239234), FirstTradeID: 8146765064, LastTradeID: 8149534072, TotalNumberOfTrades: 2756829, SymbolType: 1,
			},
		},
		{
			name: "mini ticker", fixture: "testdata/wsCFutures.jsonl", key: "btcusd_perp@miniTicker", target: new(FuturesMiniTicker),
			exp: &FuturesMiniTicker{
				EventType: "24hrMiniTicker", EventTime: fixtureTime(1791283463856), Symbol: "BTCUSD_PERP", Pair: "BTCUSD", ClosePrice: 86137.8,
				OpenPrice: 85973.7, HighPrice: 86664.7, LowPrice: 84911.4, TotalTradedBaseAssetVolume: 7354554, TotalTradedQuoteAssetVolume: 8584, SymbolType: 2,
			},
		},
		{
			name: "aggregate trade", fixture: "testdata/wsUFuturesMarket.jsonl", key: "btcusdt@aggTrade", target: new(FuturesAggregateTrade),
			exp: &FuturesAggregateTrade{
				EventType: "aggTrade", EventTime: fixtureTime(1791283227690), Symbol: "BTCUSDT", AggregateTradeID: 3476465318, Price: 86000, Quantity: 4.577,
				NormalQuantity: 4.571, FirstTradeID: 8149533714, LastTradeID: 8149533773, TradeTime: fixtureTime(1791283227688), IsBuyerMaker: true, SymbolType: 1,
			},
		},
		{
			name: "diff depth", fixture: "testdata/wsCFutures.jsonl", key: "btcusd_perp@depth@100ms", target: new(FuturesDepthUpdate),
			exp: &FuturesDepthUpdate{
				EventType: "depthUpdate", EventTime: fixtureTime(1791283450270), TransactionTime: fixtureTime(1791283450264), Symbol: "BTCUSD_PERP", Pair: "BTCUSD",
				FirstUpdateID: 11747023537518, FinalUpdateID: 11747023543725, PreviousFinalUpdateID: 11747023536504,
				Bids:       orderbook.LevelsArrayPriceAmount{{Price: 86135.3, Amount: 815}, {Price: 86133.6, Amount: 0}},
				Asks:       orderbook.LevelsArrayPriceAmount{{Price: 86135.6, Amount: 858}},
				SymbolType: 2,
			},
		},
		{
			name: "kline", fixture: "testdata/wsUFuturesMarket.jsonl", key: "btcusdt@kline_1m", target: new(FuturesKline),
			exp: &FuturesKline{
				EventType: "kline", EventTime: fixtureTime(1791283227688), Symbol: "BTCUSDT",
				Kline: FuturesCandle{
					StartTime: fixtureTime(1791283200000), CloseTime: fixtureTime(1791283259999), Symbol: "BTCUSDT", Interval: "1m", FirstTradeID: 8149533620,
					LastTradeID: 8149533714, OpenPrice: 85999.9, ClosePrice: 86000, HighPrice: 86000.1, LowPrice: 85999.8, Volume: 5.427, NumberOfTrades: 92,
					IsClosed: true, QuoteAssetVolume: 466721.747, TakerBuyBaseAssetVolume: 2.897, TakerBuyQuoteAssetVolume: 249142, Ignore: 1,
				},
			},
		},
		{
			name: "account update", fixture: "testdata/wsUFuturesUserData.jsonl", key: "ACCOUNT_UPDATE", target: new(FuturesAccountUpdate),
			exp: &FuturesAccountUpdate{
				EventType: "ACCOUNT_UPDATE", EventTime: fixtureTime(1564745798939), TransactionTime: fixtureTime(1564745798938),
				UpdateData: FuturesAccountChange{
					EventReasonType: "FUNDING_FEE",
					Balances: []FuturesAccountUpdateBalance{
						{Asset: currency.USDT, WalletBalance: 122624.12345678, CrossWalletBalance: 100.12345678, BalanceChange: 50.12345678},
						{Asset: currency.BNB, WalletBalance: 1, CrossWalletBalance: 1, BalanceChange: 0.5},
					},
					Positions: []FuturesAccountUpdatePosition{
						{Symbol: "BTCUSDT", PositionAmount: -0.02, EntryPrice: 86100, BreakevenPrice: 86134.44, AccumulatedRealised: 200, UnrealisedPNL: 1.8245, MarginType: "isolated", IsolatedWallet: 86.1, PositionSide: "BOTH"},
						{Symbol: "BTCUSDT", AccumulatedRealised: 12.5, MarginType: "cross", PositionSide: "LONG"},
					},
					Symbol: "BTCUSDT",
				},
			},
		},
		{
			name: "COIN-M account update", fixture: "testdata/wsCFuturesUserData.jsonl", key: "ACCOUNT_UPDATE", target: new(FuturesAccountUpdate),
			exp: &FuturesAccountUpdate{
				EventType: "ACCOUNT_UPDATE", EventTime: fixtureTime(1564745798939), TransactionTime: fixtureTime(1564745798938), AccountAlias: "SfsR",
				UpdateData: FuturesAccountChange{
					EventReasonType: "ORDER",
					Balances:        []FuturesAccountUpdateBalance{{Asset: currency.BTC, WalletBalance: 1.12345678, CrossWalletBalance: 1.00345678, BalanceChange: 0.02345678}},
					Positions: []FuturesAccountUpdatePosition{{
						Symbol: "BTCUSD_PERP", PositionAmount: 20, EntryPrice: 86000, BreakevenPrice: 86012.5, AccumulatedRealised: 0.0002, UnrealisedPNL: 0.00001,
						MarginType: "isolated", IsolatedWallet: 0.00002325, PositionSide: "LONG",
					}},
					Symbol: "BTCUSD_PERP",
				},
			},
		},
		{
			name: "order trade update", fixture: "testdata/wsUFuturesUserData.jsonl", key: "ORDER_TRADE_UPDATE", target: new(FuturesOrderTradeUpdate),
			exp: &FuturesOrderTradeUpdate{
				EventType: "ORDER_TRADE_UPDATE", EventTime: fixtureTime(1568879465651), TransactionTime: fixtureTime(1568879465650),
				Order: FuturesOrderTradeUpdateOrder{
					Symbol: "BTCUSDT", ClientOrderID: "TEST", Side: "SELL", OrderType: "TRAILING_STOP_MARKET", TimeInForce: "GTC", OriginalQuantity: 0.5,
					OriginalPrice: 86100.1, AveragePrice: 86104.2, StopPrice: 86103.04, ExecutionType: "TRADE", OrderStatus: "PARTIALLY_FILLED", OrderID: 8886774,
					ModifyID: "44444", OrderLastFilledQuantity: 0.125, OrderFilledAccumulatedQuantity: 0.25, LastFilledPrice: 86104.2, CommissionAsset: currency.USDT,
					Commission: 0.03444168, OrderTradeTime: fixtureTime(1568879465650), TradeID: 10125, BidsNotional: 172.2, AskNotional: 9.91, IsMaker: true,
					IsReduceOnly: true, StopPriceWorkingType: "CONTRACT_PRICE", OriginalOrderType: "TRAILING_STOP_MARKET", PositionSide: "LONG", IsCloseAll: true,
					ActivationPrice: 86476.89, CallbackRate: 5, IsPriceProtected: true, IgnoreSI: 3, IgnoreSS: 4, RealisedProfit: 0.24,
					SelfTradePreventionMode: "EXPIRE_TAKER", PriceMatchMode: "OPPONENT", GoodTillDate: fixtureTime(1568879965650), ExpiryReason: "1",
				},
			},
		},
		{
			name: "COIN-M order trade update", fixture: "testdata/wsCFuturesUserData.jsonl", key: "ORDER_TRADE_UPDATE", target: new(FuturesOrderTradeUpdate),
			exp: &FuturesOrderTradeUpdate{
				EventType: "ORDER_TRADE_UPDATE", EventTime: fixtureTime(1591274595442), TransactionTime: fixtureTime(1591274595441), AccountAlias: "SfsR",
				Order: FuturesOrderTradeUpdateOrder{
					Symbol: "BTCUSD_PERP", ClientOrderID: "TEST", Side: "BUY", OrderType: "LIMIT", TimeInForce: "GTX", OriginalQuantity: 2, OriginalPrice: 86000.1,
					AveragePrice: 86000.1, StopPrice: 86100.1, ExecutionType: "TRADE", OrderStatus: "FILLED", OrderID: 8888888, ModifyID: "44445",
					OrderLastFilledQuantity: 1, OrderFilledAccumulatedQuantity: 2, LastFilledPrice: 86000.1, MarginAsset: currency.BTC, CommissionAsset: currency.BTC,
					Commission: 0.00000116, OrderTradeTime: fixtureTime(1591274595442), TradeID: 1157454910, BidsNotional: 0.00002325, AskNotional: 0.00001163,
					IsMaker: true, IsReduceOnly: true, StopPriceWorkingType: "MARK_PRICE", OriginalOrderType: "LIMIT", PositionSide: "LONG", IsCloseAll: true,
					ActivationPrice: 86476.8, CallbackRate: 5, IsPriceProtected: true, RealisedProfit: 0.00000005, SelfTradePreventionMode: "EXPIRE_TAKER",
					PriceMatchMode: "OPPONENT", ExpiryReason: "2",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.NoErrorf(t, json.Unmarshal(fixtureFrameData(t, tc.fixture, tc.key), tc.target), "Unmarshal must not error for the %s", tc.name)
			assert.Equalf(t, tc.exp, tc.target, "the %s should decode every field", tc.name)
		})
	}
}

func TestDerivativesStreamsLive(t *testing.T) {
	t.Parallel()
	if mockTests {
		t.Skip("live websocket streams are only tested with -tags mock_test_off")
	}
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.Features.Enabled.TradeFeed = true
	ex.Websocket.Trade.Setup(true, ex.Websocket.DataHandler)
	// Only the derivatives streams are tested; asset-less subscriptions are spot streams too
	ex.Features.Subscriptions = slices.DeleteFunc(ex.Features.Subscriptions, func(s *subscription.Subscription) bool {
		return s.Asset == asset.Spot || s.Asset == asset.Empty
	})
	for a, p := range map[asset.Item]currency.Pair{
		asset.USDTMarginedFutures: currency.NewBTCUSDT(),
		asset.CoinMarginedFutures: currency.NewPairWithDelimiter("BTCUSD", "PERP", currency.UnderscoreDelimiter),
	} {
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, false), "StorePairs must not error for %s", a)
		require.NoErrorf(t, ex.CurrencyPairs.StorePairs(a, currency.Pairs{p}, true), "StorePairs must not error for %s", a)
	}
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Options, false), "SetAssetEnabled must not error")
	require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
	t.Cleanup(func() {
		// Disabled first, as the connection monitor would otherwise keep reconnecting after the test; the monitor may
		// then shut the websocket down itself
		assert.NoError(t, ex.Websocket.Disable(), "Disable should not error")
		if err := ex.Websocket.Shutdown(); err != nil {
			assert.ErrorIs(t, err, websocket.ErrNotConnected, "Shutdown should only fail once the monitor has shut the websocket down")
		}
	})

	type key struct {
		a    asset.Item
		kind string
	}
	seen := map[key]bool{}
	want := []key{
		{asset.USDTMarginedFutures, "ticker"},
		{asset.USDTMarginedFutures, "kline"},
		{asset.USDTMarginedFutures, "trade"},
		{asset.USDTMarginedFutures, "orderbook"},
		{asset.CoinMarginedFutures, "ticker"},
		{asset.CoinMarginedFutures, "kline"},
		{asset.CoinMarginedFutures, "orderbook"},
	}
	deadline := time.After(time.Minute)
	for {
		complete := true
		for _, k := range want {
			complete = complete && seen[k]
		}
		if complete {
			return
		}
		select {
		case p := <-ex.Websocket.DataHandler.C:
			switch d := p.Data.(type) {
			case *ticker.Price:
				seen[key{d.AssetType, "ticker"}] = true
			case kline.Item:
				seen[key{d.Asset, "kline"}] = true
			case []trade.Data:
				seen[key{d[0].AssetType, "trade"}] = true
			case *orderbook.Depth:
				// A REST synchronised book holds far more levels than one event carries
				if book, err := d.Retrieve(); err == nil && len(book.Bids) > 100 {
					seen[key{book.Asset, "orderbook"}] = true
				}
			case error:
				assert.NoError(t, d, "the live streams should not relay errors")
			}
		case <-deadline:
			require.Failf(t, "live streams must deliver every default stream", "seen %v of %v", seen, want)
		}
	}
}

func TestProcessFuturesAccountUpdate(t *testing.T) {
	t.Parallel()
	for i := range balanceEvents {
		tc := &balanceEvents[i]
		if tc.a == asset.Options {
			continue
		}
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			testBalanceEventOrdering(t, tc)
			t.Run("mixed currency freshness", func(t *testing.T) {
				t.Parallel()
				// An event older than an earlier event of one of its currencies stores only its other currencies
				ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
				require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.eventTotal)), "the first event must be processed")
				relayedPayloads(ex)
				data := fmt.Appendf(nil, `{"e":"ACCOUNT_UPDATE","T":1791299999000,"a":{"B":[{"a":%q,"wb":"1"},{"a":%q,"wb":"3"}]}}`, tc.code.String(), tc.restOnly.String())
				require.NoError(t, tc.processedBy(ex)(t.Context(), data), "the mixed event must be processed")
				assert.Equal(t, tc.eventTotal, storedBalance(t, ex, tc.a, tc.code).Total, "a currency with a later event should keep that event's balance")
				assert.Equal(t, 3.0, storedBalance(t, ex, tc.a, tc.restOnly).Total, "the current currency should update")
				require.Len(t, ex.Websocket.DataHandler.C, 1, "only one balance payload must be relayed")
				relayed, ok := (<-ex.Websocket.DataHandler.C).Data.(accounts.SubAccounts)
				require.True(t, ok, "the payload must hold balances")
				require.Len(t, relayed, 1, "one sub-account must be relayed")
				assert.Len(t, relayed[0].Balances, 1, "only the current currency should be relayed")
				assert.Contains(t, relayed[0].Balances, tc.restOnly, "the relay should include the current currency")
			})
		})
	}
}

// testRESTBalanceSnapshotOrdering checks that an older REST reply cannot replace a newer accepted snapshot.
func testRESTBalanceSnapshotOrdering(t *testing.T, refresh bool) {
	t.Helper()
	for i := range balanceEvents {
		tc := &balanceEvents[i]
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var older atomic.Bool
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				reply := balanceRefreshReplies[r.URL.Path]
				if older.Load() {
					reply = strings.ReplaceAll(reply, "1791300000000", "1791299999000")
					field := "availableBalance"
					if tc.a == asset.Options {
						field = "available"
					}
					reply = strings.Replace(reply, `"`+field+`":"`+strconv.FormatFloat(tc.restFree, 'f', -1, 64)+`"`, `"`+field+`":"0"`, 1)
				}
				_, err := w.Write([]byte(reply))
				assert.NoError(t, err, "the snapshot should be written")
			})
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "the newer snapshot must load")
			before := storedBalance(t, ex, tc.a, tc.code)
			older.Store(true)
			if refresh {
				lagging, err := ex.refreshAvailableBalances(t.Context(), tc.a)
				require.NoError(t, err, "the older refresh must return")
				assert.True(t, lagging, "an older snapshot should trigger another refresh")
			} else {
				_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "the older poll must return")
			}
			assert.Equal(t, before, storedBalance(t, ex, tc.a, tc.code), "the older reply should preserve every stored balance field")
		})
	}
}

func TestStoreBalanceEvent(t *testing.T) {
	t.Parallel()
	base := time.UnixMilli(1791300000000)
	both := map[*currency.Item]bool{currency.BTC.Item: true, currency.ETH.Item: true}
	// An event is held back only by an earlier event of its currency, whose transaction time is on its clock; REST
	// update times are not, so no REST snapshot holds an event back
	for _, tc := range []struct {
		name        string
		snapshot    time.Time
		btc         time.Time
		eth         time.Time
		event       time.Time
		btcPrevious time.Time
		ethPrevious time.Time
		storeErr    error
		expected    map[*currency.Item]bool
	}{
		{name: "no recorded time", event: base, expected: both},
		{name: "account snapshot newer", snapshot: base.Add(time.Millisecond), event: base, expected: both},
		{name: "currency snapshots newer", btc: base.Add(time.Millisecond), eth: base.Add(time.Millisecond), event: base, expected: both},
		{name: "previous events newer", btcPrevious: base.Add(time.Millisecond), ethPrevious: base.Add(time.Millisecond), event: base},
		{name: "previous event times differ", btcPrevious: base.Add(time.Millisecond), ethPrevious: base.Add(-time.Millisecond), event: base, expected: map[*currency.Item]bool{currency.ETH.Item: true}},
		{name: "same previous millisecond", btcPrevious: base, ethPrevious: base, event: base, expected: both},
		{name: "store error", event: base, storeErr: accounts.ErrNoBalances, expected: both},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, func(_ http.ResponseWriter, _ *http.Request) {
				assert.Fail(t, "no REST request should be needed while disconnected")
			})
			ex.balanceRefresh.snapshotTimes = map[asset.Item]time.Time{asset.USDTMarginedFutures: tc.snapshot}
			ex.balanceRefresh.currencySnapshotTimes = map[asset.Item]map[*currency.Item]time.Time{
				asset.USDTMarginedFutures: {currency.BTC.Item: tc.btc, currency.ETH.Item: tc.eth},
			}
			ex.balanceRefresh.eventTimes = map[asset.Item]map[*currency.Item]time.Time{
				asset.USDTMarginedFutures: {currency.BTC.Item: tc.btcPrevious, currency.ETH.Item: tc.ethPrevious},
			}
			var got map[*currency.Item]bool
			err := ex.storeBalanceEvent(t.Context(), asset.USDTMarginedFutures, tc.event, []currency.Code{currency.BTC, currency.ETH}, func(accepted map[*currency.Item]bool) error {
				got = accepted
				if ex.balanceRefresh.storeMu.TryLock() {
					ex.balanceRefresh.storeMu.Unlock()
					assert.Fail(t, "the store should hold the balance lock")
				}
				return tc.storeErr
			})
			if tc.storeErr != nil {
				require.ErrorIs(t, err, tc.storeErr, "the store error must be returned")
			} else {
				require.NoError(t, err, "storing accepted balances must succeed")
			}
			assert.Equal(t, tc.expected, got, "only current currencies should reach the store")
			times := ex.balanceEventTimes(asset.USDTMarginedFutures)
			previous := map[*currency.Item]time.Time{currency.BTC.Item: tc.btcPrevious, currency.ETH.Item: tc.ethPrevious}
			for _, c := range []currency.Code{currency.BTC, currency.ETH} {
				t.Run(c.String(), func(t *testing.T) {
					t.Parallel()
					expected := previous[c.Item]
					if tc.expected[c.Item] {
						expected = tc.event
					}
					assert.Equal(t, expected, times[c.Item], "only accepted currencies should advance their event time")
				})
			}
		})
	}
}

func TestRecordBalanceEvent(t *testing.T) { //nolint:tparallel // the cases record events on one exchange, so they run in order while the test runs in parallel
	t.Parallel()
	base := time.UnixMilli(1791300000000)
	ex := new(Exchange)
	usdtMargined := map[*currency.Item]time.Time{currency.BTC.Item: base, currency.ETH.Item: base.Add(time.Millisecond)}
	usdtMarginedNumbers := map[*currency.Item]uint64{currency.BTC.Item: 2, currency.ETH.Item: 3}
	for _, tc := range []struct {
		name    string
		a       asset.Item
		time    time.Time
		codes   []currency.Code
		times   map[*currency.Item]time.Time
		numbers map[*currency.Item]uint64
	}{
		{
			name: "first event", a: asset.USDTMarginedFutures, time: base, codes: []currency.Code{currency.BTC, currency.ETH},
			times:   map[*currency.Item]time.Time{currency.BTC.Item: base, currency.ETH.Item: base},
			numbers: map[*currency.Item]uint64{currency.BTC.Item: 1, currency.ETH.Item: 1},
		},
		// A second event in a currency's latest millisecond leaves its time as it is but numbers it again, so that a poll
		// can tell the two events apart
		{
			name: "second event in the same millisecond", a: asset.USDTMarginedFutures, time: base, codes: []currency.Code{currency.BTC},
			times:   map[*currency.Item]time.Time{currency.BTC.Item: base, currency.ETH.Item: base},
			numbers: map[*currency.Item]uint64{currency.BTC.Item: 2, currency.ETH.Item: 1},
		},
		{
			name: "later event of one currency", a: asset.USDTMarginedFutures, time: base.Add(time.Millisecond), codes: []currency.Code{currency.ETH},
			times: usdtMargined, numbers: usdtMarginedNumbers,
		},
		{
			name: "another asset's event", a: asset.Options, time: base, codes: []currency.Code{currency.USDT},
			times:   map[*currency.Item]time.Time{currency.USDT.Item: base},
			numbers: map[*currency.Item]uint64{currency.USDT.Item: 4},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex.recordBalanceEvent(tc.a, tc.time, tc.codes...)
			assert.Equal(t, tc.times, ex.balanceEventTimes(tc.a), "each currency should keep its latest event time")
			assert.Equal(t, tc.numbers, ex.balanceRefresh.eventNumbers[tc.a], "each currency the event reports should take its number")
		})
	}
	assert.Equal(t, usdtMargined, ex.balanceEventTimes(asset.USDTMarginedFutures), "another asset's event should leave the first asset's times alone")
	assert.Equal(t, usdtMarginedNumbers, ex.balanceRefresh.eventNumbers[asset.USDTMarginedFutures], "another asset's event should leave the first asset's numbers alone")
}

func TestBalanceEventsRecorded(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.Zero(t, ex.balanceEventsRecorded(), "no event should be counted before one is recorded")
	ex.recordBalanceEvent(asset.USDTMarginedFutures, time.UnixMilli(1791300000000), currency.USDT, currency.BNB)
	assert.Equal(t, uint64(1), ex.balanceEventsRecorded(), "an event reporting two currencies should count once")
	ex.recordBalanceEvent(asset.CoinMarginedFutures, time.UnixMilli(1791300000000), currency.BTC)
	assert.Equal(t, uint64(2), ex.balanceEventsRecorded(), "each asset's events should count towards the one total")
	// A poll reads the count while the user data streams record events
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			var last uint64
			for range 50 {
				ex.recordBalanceEvent(asset.Options, time.UnixMilli(1791300000000), currency.USDT)
				count := ex.balanceEventsRecorded()
				assert.Greater(t, count, last, "the count should rise with each event recorded")
				last = count
			}
		})
	}
	wg.Wait()
	assert.Equal(t, uint64(202), ex.balanceEventsRecorded(), "every event recorded beside the reads should be counted")
}

func TestRecordBalanceSnapshot(t *testing.T) {
	t.Parallel()
	base := time.UnixMilli(1791300000000)
	for _, tc := range []struct {
		name           string
		otherAccount   bool
		noCredentials  bool
		contextOnly    bool
		invalidContext bool
	}{
		{name: "configured account"},
		{name: "different account", otherAccount: true},
		{name: "credentials unavailable", noCredentials: true},
		{name: "context credentials without a configured account", noCredentials: true, contextOnly: true},
		{name: "invalid contextual credentials", invalidContext: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, func(_ http.ResponseWriter, _ *http.Request) {
				assert.Fail(t, "recording a snapshot should send no request")
			})
			ctx := t.Context()
			if tc.otherAccount {
				ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "other-key", Secret: "other-secret"})
			}
			if tc.noCredentials {
				ex.SetCredentials(nil)
				if tc.contextOnly {
					ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "context-key", Secret: "context-secret"})
				}
			}
			if tc.invalidContext {
				ctx = context.WithValue(ctx, accounts.ContextCredentialsFlag, "invalid")
			}
			ex.recordBalanceSnapshot(ctx, asset.Options, base, map[*currency.Item]time.Time{currency.BTC.Item: base, currency.ETH.Item: base.Add(time.Millisecond)})
			ex.recordBalanceSnapshot(ctx, asset.Options, base.Add(-time.Millisecond), map[*currency.Item]time.Time{currency.BTC.Item: base.Add(-time.Millisecond)})
			ex.recordBalanceSnapshot(ctx, asset.Options, time.Time{}, nil)
			if tc.otherAccount || tc.noCredentials || tc.invalidContext {
				assert.Empty(t, ex.balanceRefresh.snapshotTimes, "another account should not set a stream watermark")
				assert.Empty(t, ex.balanceRefresh.currencySnapshotTimes, "another account should not set currency watermarks")
				return
			}
			assert.Equal(t, base, ex.balanceSnapshotTime(asset.Options), "an older snapshot should not lower the account watermark")
			assert.Equal(t, map[*currency.Item]time.Time{currency.BTC.Item: base, currency.ETH.Item: base.Add(time.Millisecond)}, ex.balanceRefresh.currencySnapshotTimes[asset.Options], "currency watermarks should retain the latest times independently")
		})
	}
}

// testBalanceEventOrdering checks that a stream event's balance is stored and relayed whatever the update time of the
// REST snapshot before it, without relying on goroutine scheduling. The transaction time is the matching engine's, and
// REST update times are not documented to keep its clock, while an event older than a snapshot is followed on its
// connection by the event of the later change the snapshot saw (see storeBalanceEvent)
func testBalanceEventOrdering(t *testing.T, event *balanceEvent) {
	t.Helper()
	for _, tc := range []struct {
		name         string
		offset       int64
		refresh      bool
		otherAccount bool
	}{
		{name: "older event after poll", offset: -1000},
		{name: "older event after refresh", offset: -1000, refresh: true},
		{name: "same millisecond event", offset: 0},
		{name: "newer event", offset: 1000},
		{name: "another account's snapshot", offset: -1000, otherAccount: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, serveBalanceRefreshReplies(t))
			ctx := t.Context()
			if tc.otherAccount {
				ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "other-key", Secret: "other-secret"})
			}
			if tc.refresh {
				_, err := ex.refreshAvailableBalances(ctx, event.a)
				require.NoError(t, err, "the snapshot refresh must succeed")
			} else {
				_, err := ex.UpdateAccountBalances(ctx, event.a)
				require.NoError(t, err, "the snapshot poll must succeed")
			}
			payload := strings.ReplaceAll(string(event.walletEvent(event.eventTotal)), "1791300000000", strconv.FormatInt(1791300000000+tc.offset, 10))
			payload = strings.ReplaceAll(payload, "1791300000001", strconv.FormatInt(1791300000001+tc.offset, 10))
			require.NoError(t, event.processedBy(ex)(t.Context(), []byte(payload)), "the event must be handled")
			assert.Equal(t, event.eventTotal, storedBalance(t, ex, event.a, event.code).Total, "the event should store its balance")
			assert.Len(t, ex.Websocket.DataHandler.C, 1, "the event's balances should be relayed")
		})
	}
}

// balanceReply returns a REST balance reply for a balance event's asset holding its currency alone
func balanceReply(tc *balanceEvent, total, free float64, updateTime int64) string {
	totalValue, freeValue := strconv.FormatFloat(total, 'f', -1, 64), strconv.FormatFloat(free, 'f', -1, 64)
	switch tc.a {
	case asset.USDTMarginedFutures:
		return fmt.Sprintf(`[{"asset":%q,"balance":%q,"availableBalance":%q,"updateTime":%d}]`, tc.code, totalValue, freeValue, updateTime)
	case asset.CoinMarginedFutures:
		return fmt.Sprintf(`{"assets":[{"asset":%q,"walletBalance":%q,"availableBalance":%q,"updateTime":%d}]}`, tc.code, totalValue, freeValue, updateTime)
	default:
		return fmt.Sprintf(`{"asset":[{"asset":%q,"marginBalance":%q,"available":%q}],"time":%d}`, tc.code, totalValue, freeValue, updateTime)
	}
}

func TestKeepStreamedBalances(t *testing.T) {
	t.Parallel()
	// Transactions can share a millisecond, so a snapshot showing a currency at its latest event's time may predate the
	// event: the stored balance stands when that event was recorded since the poll's request, and otherwise the snapshot
	// is taken to include the event
	event := balanceEvents[0]
	eventTime := time.UnixMilli(1791300000000)
	rest := accounts.Balance{Total: event.restTotal, Hold: event.restTotal - event.restFree, Free: event.restFree}
	for _, tc := range []struct {
		name string
		// shown is the snapshot's time for the event's currency: its update time where the snapshot reports it, and
		// otherwise the newest time the snapshot shows, another currency's update time or, with asOf, its own
		shown    time.Time
		reported bool
		asOf     bool
		// sinceRequest records the event after the poll has counted the events recorded before its request, and repeated
		// records an earlier event of the same millisecond before the count
		sinceRequest, repeated bool
		kept                   bool
	}{
		{name: "reported at the event's time, recorded since the request", shown: eventTime, reported: true, sinceRequest: true, kept: true},
		{name: "reported at the event's time, recorded before the request", shown: eventTime, reported: true},
		{name: "reported at the event's time, its millisecond's second event recorded since the request", shown: eventTime, reported: true, sinceRequest: true, repeated: true, kept: true},
		{name: "reported after the event, recorded since the request", shown: eventTime.Add(time.Millisecond), reported: true, sinceRequest: true},
		{name: "left out at the event's time, recorded since the request", shown: eventTime, sinceRequest: true, kept: true},
		{name: "left out at the event's time, recorded before the request", shown: eventTime},
		{name: "left out after the event, recorded since the request", shown: eventTime.Add(time.Millisecond), sinceRequest: true},
		{name: "left out of a snapshot timed at the event, recorded since the request", shown: eventTime, asOf: true, sinceRequest: true, kept: true},
		{name: "left out of a snapshot timed at the event, recorded before the request", shown: eventTime, asOf: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTradingTestExchange(t, func(_ http.ResponseWriter, _ *http.Request) {
				assert.Fail(t, "no REST request should be made")
			})
			if tc.repeated {
				require.NoError(t, event.processedBy(ex)(t.Context(), event.walletEvent(event.eventTotal-1)), "processing the first balance event must not error")
			}
			eventsBefore := ex.balanceEventsRecorded()
			require.NoError(t, event.processedBy(ex)(t.Context(), event.walletEvent(event.eventTotal)), "processing the balance event must not error")
			if !tc.sinceRequest {
				eventsBefore = ex.balanceEventsRecorded()
			}
			streamed := storedBalance(t, ex, event.a, event.code)
			require.Equal(t, event.eventTotal, streamed.Total, "the event must store its wallet balance")
			subAccount := accounts.NewSubAccount(event.a, "")
			updated := make(map[*currency.Item]time.Time)
			var asOf time.Time
			var snapshotBalance accounts.Balance
			switch {
			case tc.reported:
				subAccount.Balances.Set(event.code, rest)
				snapshotBalance = subAccount.Balances[event.code]
				updated[event.code.Item] = tc.shown
				// Another currency updated later is the snapshot's newest, apart from the reported currency's own time
				subAccount.Balances.Set(event.restOnly, accounts.Balance{Total: event.restOnlyTotal, Free: event.restOnlyTotal})
				updated[event.restOnly.Item] = tc.shown.Add(time.Millisecond)
			case tc.asOf:
				asOf = tc.shown
			default:
				subAccount.Balances.Set(event.restOnly, accounts.Balance{Total: event.restOnlyTotal, Free: event.restOnlyTotal})
				updated[event.restOnly.Item] = tc.shown
			}
			require.NoError(t, ex.withBalanceStore(func() error {
				ex.keepStreamedBalances(t.Context(), event.a, subAccount, updated, asOf, eventsBefore)
				return nil
			}), "withBalanceStore must not error")
			got, ok := subAccount.Balances[event.code]
			switch {
			case tc.kept:
				require.True(t, ok, "the snapshot must hold the event's currency")
				assert.Equal(t, streamed, got, "the balance of an event recorded since the request should stand")
			case tc.reported:
				assert.Equal(t, snapshotBalance, got, "the balance the snapshot reports should stand")
			default:
				assert.False(t, ok, "the currency the snapshot leaves out should stay out")
				assert.Equal(t, tc.shown, updated[event.code.Item], "the currency the snapshot leaves out should take its newest time")
			}
		})
	}
}

func TestOlderPollKeepsRefreshedBalance(t *testing.T) {
	t.Parallel()
	// A balance a refresh stores records its REST update time, so a poll answered with an older snapshot keeps it
	for i := range balanceEvents {
		tc := &balanceEvents[i]
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var older atomic.Bool
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				reply := balanceReply(tc, 100, 80, 1791300000000)
				if older.Load() {
					reply = balanceReply(tc, 90, 70, 1791299999000)
				}
				_, err := w.Write([]byte(reply))
				assert.NoError(t, err, "Write should not error")
			})
			_, err := ex.refreshAvailableBalances(t.Context(), tc.a)
			require.NoError(t, err, "the refresh must not error")
			require.Equal(t, 100.0, storedBalance(t, ex, tc.a, tc.code).Total, "the refresh must store a balance nothing had stored")
			older.Store(true)
			_, err = ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "the older poll must not error")
			got := storedBalance(t, ex, tc.a, tc.code)
			assert.Equal(t, 100.0, got.Total, "an older poll should keep the refreshed total")
			assert.Equal(t, 80.0, got.Free, "an older poll should keep the refreshed free balance")
		})
	}
}

func TestOlderReplyKeepsRemovedCurrencyOut(t *testing.T) {
	t.Parallel()
	// A newer full snapshot that leaves a currency out removes it and records the time it was removed by, so a poll or a
	// refresh answered with an older snapshot that still lists the currency does not bring it back
	for i := range balanceEvents {
		tc := &balanceEvents[i]
		for _, refresh := range []bool{false, true} {
			t.Run(tc.a.String()+map[bool]string{false: " poll", true: " refresh"}[refresh], func(t *testing.T) {
				t.Parallel()
				var stage atomic.Int32
				ex := newTradingTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
					reply := balanceReply(tc, 100, 80, 1791300000000)
					switch stage.Load() {
					case 1:
						reply = fmt.Sprintf(tc.restOnlyReply, int64(1791300001000))
					case 2:
						reply = balanceReply(tc, 90, 70, 1791300000500)
					}
					_, err := w.Write([]byte(reply))
					assert.NoError(t, err, "Write should not error")
				})
				_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "the first poll must not error")
				stage.Store(1)
				_, err = ex.UpdateAccountBalances(t.Context(), tc.a)
				require.NoError(t, err, "the newer poll must not error")
				_, err = ex.Accounts.GetBalance("", ex.GetDefaultCredentials(), tc.a, tc.code)
				require.ErrorIs(t, err, accounts.ErrNoBalances, "the newer full snapshot must remove the currency it leaves out")
				stage.Store(2)
				if refresh {
					lagging, err := ex.refreshAvailableBalances(t.Context(), tc.a)
					require.NoError(t, err, "the older refresh must not error")
					assert.True(t, lagging, "the older refresh should be asked again")
				} else {
					_, err = ex.UpdateAccountBalances(t.Context(), tc.a)
					require.NoError(t, err, "the older poll must not error")
				}
				_, err = ex.Accounts.GetBalance("", ex.GetDefaultCredentials(), tc.a, tc.code)
				assert.ErrorIs(t, err, accounts.ErrNoBalances, "an older reply should not bring back a currency a newer snapshot removed")
			})
		}
	}
}

func TestBalanceEventHeldBack(t *testing.T) {
	t.Parallel()
	// An event older than the last event of each of its currencies stores and relays no balance and starts no refresh,
	// while its positions are still processed
	for i := range balanceEvents {
		tc := &balanceEvents[i]
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			ex := newTradingTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				_, err := w.Write([]byte(balanceRefreshReplies[r.URL.Path]))
				assert.NoError(t, err, "Write should not error")
			})
			setup := ex.spotStreamConnectionSetup(ex.Config)
			setup.SubscriptionsNotRequired = true
			useMockWebsocket(t, ex, setup, func(testing.TB, []byte, *gws.Conn) error { return nil })
			require.NoError(t, ex.Websocket.Connect(t.Context()), "Connect must not error")
			ex.balanceRefresh.interval = time.Millisecond
			running := func() bool {
				ex.balanceRefresh.mu.Lock()
				defer ex.balanceRefresh.mu.Unlock()
				return ex.balanceRefresh.running[tc.a]
			}
			require.NoError(t, tc.processedBy(ex)(t.Context(), tc.walletEvent(tc.restTotal)), "the first event must be processed")
			require.Eventually(t, func() bool { return requests.Load() == 1 && !running() }, 30*time.Second, time.Millisecond, "the first event's refresh must finish")
			relayedPayloads(ex)

			position := `[{"s":"BTCUSDT","pa":"1","ep":"100","cr":"0","up":"0","iw":"0","ps":"BOTH"}]`
			switch tc.a {
			case asset.CoinMarginedFutures:
				position = `[{"s":"BTCUSD_PERP","pa":"1","ep":"100","cr":"0","up":"0","iw":"0","ps":"BOTH"}]`
			case asset.Options:
				position = `[{"s":"BTC-261031-90000-C","c":"1","p":"500","a":"500"}]`
			}
			older := strings.ReplaceAll(strings.ReplaceAll(string(tc.walletEvent(tc.eventTotal)), "1791300000000", "1791299999000"), "1791300000001", "1791299999001")
			older = strings.Replace(older, `"P":[]`, `"P":`+position, 1)
			require.NoError(t, tc.processedBy(ex)(t.Context(), []byte(older)), "the held back event must be processed")
			assert.False(t, running(), "a held back event should not start a refresh")
			assert.Equal(t, int32(1), requests.Load(), "a held back event should send no request")
			assert.Equal(t, tc.restTotal, storedBalance(t, ex, tc.a, tc.code).Total, "a held back event should leave the balance as the later event stored it")
			payloads := relayedPayloads(ex)
			require.Len(t, payloads, 1, "only the held back event's positions must be relayed")
			positions, ok := payloads[0].([]futures.Position)
			require.True(t, ok, "the relayed payload must hold positions")
			assert.Len(t, positions, 1, "the held back event's position should be relayed")
		})
	}
}
