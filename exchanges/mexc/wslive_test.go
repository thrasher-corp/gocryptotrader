//go:build mock_test_off

package mexc

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// TestLiveSpotTickerFlow subscribes to the public spot channels against the live exchange and counts
// what actually arrives. The spot ticker defect was silent — no errors, simply no ticker — so only a
// live count can show the ticker flowing and the orderbook path still intact.
// Public channels only, no credentials, no orders.
func TestLiveSpotTickerFlow(t *testing.T) {
	window := 60 * time.Second
	if v := os.Getenv("MEXC_LIVE_WINDOW_SEC"); v != "" {
		s, err := strconv.Atoi(v)
		require.NoError(t, err, "MEXC_LIVE_WINDOW_SEC must be an integer")
		window = time.Duration(s) * time.Second
	}

	// Pin the pair rather than taking whatever TestMain settled on: the candle assertions want a
	// liquid market, and BTC-USDT is the same pair the rest of the suite is written against.
	pairFormat, err := e.GetPairFormat(asset.Spot, false)
	require.NoError(t, err, "GetPairFormat must not error")
	pair := currency.NewBTCUSDT().Format(pairFormat)
	// Restore the pair sets afterwards: narrowing them to this one pair and leaving it that way
	// broke the tests running in parallel, which then failed only in a full run.
	origAvailable, err := e.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "GetAvailablePairs must not error")
	origEnabled, err := e.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	t.Cleanup(func() {
		require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, origAvailable, false), "restoring the available pairs must not error")
		require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, origEnabled, true), "restoring the enabled pairs must not error")
	})
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, false), "StorePairs must not error")
	require.NoError(t, e.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{pair}, true), "StorePairs must not error")

	testexch.SetupWs(t, e)
	conn, err := e.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")

	all, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	var subs subscription.List
	for _, s := range all {
		if s.Asset == asset.Spot && !s.Authenticated {
			subs = append(subs, s)
		}
	}
	require.NotEmpty(t, subs, "there must be public spot subscriptions")
	for _, s := range subs {
		t.Logf("subscribing: %s", s.QualifiedChannel)
	}
	require.NoError(t, e.Subscribe(t.Context(), conn, subs), "Subscribe must not error")

	// The orderbook is counted from the depth store rather than the data handler: the test harness
	// swaps the relay after connecting, while the orderbook buffer keeps the one captured at setup.
	var bookUpdates int
	var lastBook time.Time
	bookPoll := time.NewTicker(100 * time.Millisecond)
	defer bookPoll.Stop()

	var tickers, tickersWithBBO, tickersWithLast int
	deadline := time.After(window)
collect:
	for {
		select {
		case <-deadline:
			break collect
		case <-bookPoll.C:
			if b, err := orderbook.Get(e.Name, pair, asset.Spot); err == nil && b.LastUpdated.After(lastBook) {
				lastBook = b.LastUpdated
				bookUpdates++
			}
		case p := <-e.Websocket.DataHandler.C:
			if v, ok := p.Data.(*ticker.Price); ok {
				tickers++
				if v.Bid > 0 && v.Ask > 0 {
					tickersWithBBO++
				}
				if v.Last > 0 {
					tickersWithLast++
				}
			}
		}
	}

	t.Logf("over %s: %d tickers (%d with a non-zero bid/ask, %d with a non-zero last), %d orderbook updates",
		window, tickers, tickersWithBBO, tickersWithLast, bookUpdates)
	assert.Positive(t, tickersWithBBO, "websocket ticker with a non-zero best bid/offer should arrive")
	assert.Positive(t, tickersWithLast, "websocket ticker should carry a last price")
	assert.Positive(t, bookUpdates, "the existing orderbook path should keep publishing")
}

// TestLiveSpotPrivateSubscriptions exercises the authenticated websocket path against the live
// exchange: the listen key is issued, the connection carries it, and the exchange accepts the three
// private channels the default subscriptions declare.
//
// It deliberately proves reachability, not delivery. Private events only appear when the account
// trades, and this test places no orders - the whole authenticated path had never been exercised
// with a real key, and every defect found in this exchange so far compiled and stayed silent until
// a live request was made.
func TestLiveSpotPrivateSubscriptions(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	// Fail here rather than inside the connection: a rejected key must be distinguishable from a
	// websocket that refuses the subscription.
	listenKey, err := e.GenerateListenKey(t.Context())
	require.NoError(t, err, "GenerateListenKey must not error")
	require.NotEmpty(t, listenKey, "the exchange must issue a listen key")

	testexch.SetupWs(t, e)
	require.True(t, e.Websocket.CanUseAuthenticatedEndpoints(), "the authenticated path must be enabled")
	conn, err := e.Websocket.GetConnection(asset.Spot)
	require.NoError(t, err, "GetConnection must not error")

	all, err := e.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	var subs subscription.List
	for _, s := range all {
		if s.Asset == asset.Spot && s.Authenticated {
			subs = append(subs, s)
		}
	}
	require.Len(t, subs, 3, "the spot defaults must declare my-trades, my-orders and my-account")
	for _, s := range subs {
		t.Logf("subscribing: %s", s.QualifiedChannel)
	}
	require.NoError(t, e.Subscribe(t.Context(), conn, subs), "Subscribe must not error")

	// Subscribe returning nil is not proof the exchange took them: check they are registered as
	// active afterwards.
	active := e.Websocket.GetSubscriptions()
	for _, want := range subs {
		found := false
		for _, got := range active {
			if got.QualifiedChannel == want.QualifiedChannel {
				found = true
				break
			}
		}
		assert.Truef(t, found, "%s should be an active subscription after Subscribe", want.QualifiedChannel)
	}

	// Listen for a short while and report whatever the account produced. This is not asserted:
	// private events require account activity, and a test that demanded them would pass or fail on
	// whether someone happened to be trading. Set MEXC_LIVE_WINDOW_SEC and trade to see them.
	window := 15 * time.Second
	if v := os.Getenv("MEXC_LIVE_WINDOW_SEC"); v != "" {
		s, err := strconv.Atoi(v)
		require.NoError(t, err, "MEXC_LIVE_WINDOW_SEC must be an integer")
		window = time.Duration(s) * time.Second
	}
	counts := map[string]int{}
	deadline := time.After(window)
collect:
	for {
		select {
		case <-deadline:
			break collect
		case p := <-e.Websocket.DataHandler.C:
			counts[fmt.Sprintf("%T", p.Data)]++
		}
	}
	if len(counts) == 0 {
		t.Logf("over %s: no private events - expected on an idle account", window)
	}
	for typ, n := range counts {
		t.Logf("over %s: %d x %s", window, n, typ)
	}
}
