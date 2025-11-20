package subscription

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// DummyKey is a test key type that ensures that cross compatible keys can be used
// It will panic if Match() is called
type DummyKey struct {
	*Subscription
	detonator testing.TB
}

var _ MatchableKey = DummyKey{} // Enforce DummyKey must implement MatchableKey

// GetSubscription returns the underlying subscription
func (k DummyKey) GetSubscription() *Subscription {
	return k.Subscription
}

// Match implements MatchableKey
func (k DummyKey) Match(_ MatchableKey) bool {
	k.detonator.Fatal("DummyKey Match should never be called")
	return false
}

// TestExactKeyMatch exercises ExactKey.Match
func TestExactKeyMatch(t *testing.T) {
	t.Parallel()

	key := &ExactKey{&Subscription{Channel: TickerChannel}}
	try := &DummyKey{&Subscription{Channel: OrderbookChannel}, t}

	require.False(t, key.Match(nil), "Match on a nil must return false")
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
	require.False(t, key.Match(try), "Gate 3: Match must reject B has Pairs when key has empty Pairs")
	key.Pairs = currency.Pairs{btcusdtPair}
	require.True(t, key.Match(try), "Gate 3: Match must accept matching pairs")
	key.Pairs = currency.Pairs{ethusdcPair}
	require.False(t, key.Match(try), "Gate 3: Match must reject when key.Pairs not matching")
	try.Pairs = currency.Pairs{btcusdtPair, ethusdcPair}
	require.False(t, key.Match(try), "Gate 3: Match must reject when key.Pairs is only a subset")
	key.Pairs = currency.Pairs{ethusdcPair, btcusdtPair}
	require.True(t, key.Match(try), "Gate 3: Match accept when Pairs match in different order")
	key.Levels = 4
	require.False(t, key.Match(try), "Gate 4: Match must reject a bad Level")
	try.Levels = 4
	require.True(t, key.Match(try), "Gate 4: Match must accept a good Level")
	key.Interval = kline.FiveMin
	require.False(t, key.Match(try), "Gate 5: Match must reject a bad Interval")
	try.Interval = kline.FiveMin
	require.True(t, key.Match(try), "Gate 5: Match must accept a good Interval")
}

// TestExactKeyString exercises ExactKey.String
func TestExactKeyString(t *testing.T) {
	t.Parallel()
	key := &ExactKey{}
	assert.Equal(t, "Uninitialised ExactKey", key.String())
	key = &ExactKey{&Subscription{Asset: asset.Spot, Channel: TickerChannel, Pairs: currency.Pairs{ethusdcPair, btcusdtPair}}}
	assert.Equal(t, "ticker spot ETH/USDC,BTC/USDT", key.String())
}

// TestIgnoringPairsKeyMatch exercises IgnoringPairsKey.Match
func TestIgnoringPairsKeyMatch(t *testing.T) {
	t.Parallel()

	key := &IgnoringPairsKey{&Subscription{Channel: TickerChannel, Pairs: currency.Pairs{btcusdtPair}}}
	try := &DummyKey{&Subscription{Channel: OrderbookChannel, Pairs: currency.Pairs{ethusdcPair}}, t}

	require.False(t, key.Match(nil), "Match on a nil must return false")
	require.False(t, key.Match(try), "Gate 1: Match must reject a bad Channel")
	try.Channel = TickerChannel
	require.True(t, key.Match(try), "Gate 1: Match must accept a good Channel")
	key.Asset = asset.Spot
	require.False(t, key.Match(try), "Gate 2: Match must reject a bad Asset")
	try.Asset = asset.Spot
	require.True(t, key.Match(try), "Gate 2: Match must accept a good Asset")
	key.Levels = 4
	require.False(t, key.Match(try), "Gate 3: Match must reject a bad Level")
	try.Levels = 4
	require.True(t, key.Match(try), "Gate 3: Match must accept a good Level")
	key.Interval = kline.FiveMin
	require.False(t, key.Match(try), "Gate 4: Match must reject a bad Interval")
	try.Interval = kline.FiveMin
	require.True(t, key.Match(try), "Gate 4: Match must accept a good Interval")
}

// TestIgnoringPairsKeyString exercises IgnoringPairsKey.String
func TestIgnoringPairsKeyString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Uninitialised IgnoringPairsKey", IgnoringPairsKey{}.String())
	key := &IgnoringPairsKey{&Subscription{Asset: asset.Spot, Channel: TickerChannel, Pairs: currency.Pairs{ethusdcPair, btcusdtPair}}}
	assert.Equal(t, "ticker spot", key.String())
}

// TestIgnoringAssetKeyMatch exercises IgnoringAssetKey.Match
func TestIgnoringAssetKeyMatch(t *testing.T) {
	t.Parallel()

	key := &IgnoringAssetKey{&Subscription{Channel: TickerChannel, Asset: asset.Spot}}
	try := &DummyKey{&Subscription{Channel: OrderbookChannel}, t}

	require.False(t, key.Match(nil), "Match on a nil must return false")
	require.False(t, key.Match(try), "Gate 1: Match must reject a bad Channel")
	try.Channel = TickerChannel
	require.True(t, key.Match(try), "Gate 1: Match must accept a good Channel")
	key.Pairs = currency.Pairs{btcusdtPair}
	require.False(t, key.Match(try), "Gate 2: Match must reject bad Pairs")
	try.Pairs = currency.Pairs{btcusdtPair}
	require.True(t, key.Match(try), "Gate 2: Match must accept a good Pairs")
	key.Levels = 4
	require.False(t, key.Match(try), "Gate 3: Match must reject a bad Level")
	try.Levels = 4
	require.True(t, key.Match(try), "Gate 3: Match must accept a good Level")
	key.Interval = kline.FiveMin
	require.False(t, key.Match(try), "Gate 4: Match must reject a bad Interval")
	try.Interval = kline.FiveMin
	require.True(t, key.Match(try), "Gate 4: Match must accept a good Interval")
}

// TestIgnoringAssetKeyString exercises IgnoringAssetKey.String
func TestIgnoringAssetKeyString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Uninitialised IgnoringAssetKey", IgnoringAssetKey{}.String())
	key := &IgnoringAssetKey{&Subscription{Asset: asset.Spot, Channel: TickerChannel, Pairs: currency.Pairs{ethusdcPair, btcusdtPair}}}
	assert.Equal(t, "ticker [ETHUSDC BTCUSDT]", key.String())
}

// TestGetSubscription exercises GetSubscription
func TestGetSubscription(t *testing.T) {
	t.Parallel()
	s := &Subscription{Asset: asset.Spot}
	assert.Same(t, s, ExactKey{s}.GetSubscription(), "ExactKey.GetSubscription Must return a pointer to the subscription")
	assert.Same(t, s, IgnoringPairsKey{s}.GetSubscription(), "IgnorePairKeys.GetSubscription Must return a pointer to the subscription")
}

func TestMustChannelKey(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { MustChannelKey("") }, "no channel string must panic")
	key := MustChannelKey(TickerChannel)
	assert.Equal(t, TickerChannel, key.Subscription.Channel)
}

func TestChannelKeyMatch(t *testing.T) {
	t.Parallel()
	key := ChannelKey{&Subscription{Channel: TickerChannel}}
	try := &DummyKey{&Subscription{Channel: OrderbookChannel}, t}

	require.Panics(t, func() { key.Match(nil) }, "Match on a nil must panic")
	require.False(t, key.Match(try), "Match must reject a different channel")
	try.Channel = TickerChannel
	assert.True(t, key.Match(try), "Match should accept an identical channel")
}

func TestChannelKeyGetSubscription(t *testing.T) {
	t.Parallel()
	key := ChannelKey{&Subscription{Channel: TickerChannel}}
	assert.Same(t, key.Subscription, key.GetSubscription(), "GetSubscription should return the underlying subscription")
}
