package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestNewWSOBResubManager(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()
	require.NotNil(t, m)
	assert.NotNil(t, m.lookup)
}

func TestIsResubscribing(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()
	m.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true
	assert.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	assert.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Futures))
}

func TestResubscribe(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e))
	e.Name = "Resubscribe"

	baseConn, err := e.Websocket.CreateTestConnection(asset.Spot)
	require.NoError(t, err)
	conn := &FixtureConnection{Connection: baseConn}
	require.NoError(t, e.Websocket.TrackTestConnection(asset.Spot, conn))

	err = m.Resubscribe(t.Context(), e, conn, "notfound", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, orderbook.ErrDepthNotFound)
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))

	err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Asks:        []orderbook.Level{{Price: 50000, Amount: 0.1}},
		Bids:        []orderbook.Level{{Price: 49000, Amount: 0.2}},
		Exchange:    e.Name,
		Pair:        currency.NewBTCUSDT(),
		Asset:       asset.Spot,
		LastUpdated: time.Now(),
	})
	require.NoError(t, err)
	err = m.Resubscribe(t.Context(), e, conn, "notfound", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))

	e.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: spotOrderbookV2, Asset: asset.Spot, Levels: 50},
	}
	expanded, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err)

	err = e.Websocket.AddSubscriptions(conn, expanded...)
	require.NoError(t, err)

	qualifiedChannel := "ob.BTC_USDT.50"
	err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Asks:        []orderbook.Level{{Price: 50000, Amount: 0.1}},
		Bids:        []orderbook.Level{{Price: 49000, Amount: 0.2}},
		Exchange:    e.Name,
		Pair:        currency.NewBTCUSDT(),
		Asset:       asset.Spot,
		LastUpdated: time.Now(),
	})
	require.NoError(t, err)
	err = m.Resubscribe(t.Context(), e, conn, qualifiedChannel, currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err)
	assert.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot), "manager should mark the pair as resubscribing immediately")
	assert.Eventually(t,
		func() bool {
			sub := e.Websocket.GetSubscription(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: qualifiedChannel}})
			return sub != nil && sub.State() == subscription.SubscribedState
		},
		time.Second,
		10*time.Millisecond,
		"subscription should be resubscribed by the background routine",
	)

	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot)
	assert.Eventually(t,
		func() bool {
			return !m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot)
		},
		time.Second,
		10*time.Millisecond,
		"resubscription state should clear after completion is signalled",
	)
}

func TestCompletedResubscribe(t *testing.T) {
	t.Parallel()

	m := newWSOBResubManager()
	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot) // no-op
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	m.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true
	require.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot)
	assert.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
}

func TestQualifiedChannelKey_Match(t *testing.T) {
	t.Parallel()

	require.Implements(t, (*subscription.MatchableKey)(nil), new(qualifiedChannelKey))

	k := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "test.channel"}}
	require.True(t, k.Match(k))
	require.False(t, k.Match(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "TEST.channel"}}))
	assert.NotNil(t, k.GetSubscription())
}
