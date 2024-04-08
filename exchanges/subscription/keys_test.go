package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// TestSubsetKeyMatch exercises the HasPairKey MatchableKey interface implementation
// Given A.Match(B):
// Where A is the incoming key, and B is each key in the store
// Ensures A.Pairs must be a subset of B.Pairs
func TestHasPairKeyMatch(t *testing.T) {
	t.Parallel()

	key := &HasPairKey{&Subscription{Channel: TickerChannel}}
	try := &ExactKey{&Subscription{Channel: OrderbookChannel}}

	assert.NotNil(t, key.EnsureKeyed(), "EnsureKeyed should work")

	require.False(t, key.Match(try), "Gate 1: Match must reject a bad Channel")
	try.Channel = TickerChannel
	require.True(t, key.Match(try), "Gate 1: Match must accept a good Channel")
	key.Asset = asset.Spot
	require.False(t, key.Match(try), "Gate 2: Match must reject a bad Asset")
	try.Asset = asset.Spot
	require.True(t, key.Match(try), "Gate 2: Match must accept a good Asset")

	key.Pairs = currency.Pairs{btcusdtPair}
	require.False(t, key.Match(try), "Gate 3: Match must reject B empty Pairs when key has Pairs")
	try.Pairs = currency.Pairs{btcusdtPair}
	key.Pairs = nil
	require.False(t, key.Match(try), "Gate 4: Match must reject B has Pairs when key has empty Pairs")
	key.Pairs = currency.Pairs{btcusdtPair}
	require.True(t, key.Match(try), "Gate 5: Match must accept matching pairs")
	key.Pairs = currency.Pairs{ethusdcPair}
	require.False(t, key.Match(try), "Gate 5: Match must reject when key.Pairs not in try.Pairs")
	try.Pairs = currency.Pairs{btcusdtPair, ethusdcPair}
	require.True(t, key.Match(try), "Gate 5: Match must accept one of the key.Pairs in try.Pairs")
	key.Pairs = currency.Pairs{btcusdtPair, ethusdcPair}
	try.Pairs = currency.Pairs{btcusdtPair, ltcusdcPair}
	require.False(t, key.Match(try), "Gate 5: Match must reject when key.Pairs not in try.Pairs")
	try.Pairs = currency.Pairs{btcusdtPair, ethusdcPair, ltcusdcPair}
	require.True(t, key.Match(try), "Gate 5: Match must accept when all key.Pairs are subset of try.Pairs")
	key.Levels = 4
	require.False(t, key.Match(try), "Gate 6: Match must reject a bad Level")
	try.Levels = 4
	require.True(t, key.Match(try), "Gate 6: Match must accept a good Level")
	key.Interval = kline.FiveMin
	require.False(t, key.Match(try), "Gate 7: Match must reject a bad Interval")
	try.Interval = kline.FiveMin
	require.True(t, key.Match(try), "Gate 7: Match must accept a good Interval")
}
