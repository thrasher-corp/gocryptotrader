//go:build mock_test_off

package mexc

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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
