package gateio

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestNewWSSubscriptionManager(t *testing.T) {
	t.Parallel()

	m := newWSSubscriptionManager()
	require.NotNil(t, m)
	require.NotNil(t, m.lookup)
}

func TestIsResubscribing(t *testing.T) {
	t.Parallel()

	m := newWSSubscriptionManager()
	m.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true
	require.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Futures))
}

func TestResubscribe(t *testing.T) {
	t.Parallel()

	m := newWSSubscriptionManager()

	conn := &FixtureConnection{}

	err := m.Resubscribe(e, conn, "notfound", currency.NewBTCUSDT(), asset.Spot)
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
	err = m.Resubscribe(e, conn, "notfound", currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, subscription.ErrNotFound)

	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))

	e := new(Exchange) //nolint:govet // Intentional shadow
	require.NoError(t, testexch.Setup(e))
	e.Name = "Resubscribe"
	e.Features.Subscriptions = subscription.List{
		{Enabled: true, Channel: spotOrderbookUpdateWithSnapshotChannel, Asset: asset.Spot, Levels: 50},
	}
	expanded, err := e.Features.Subscriptions.ExpandTemplates(e)
	require.NoError(t, err)

	err = e.Websocket.AddSubscriptions(conn, expanded...)
	require.NoError(t, err)

	err = e.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
		Asks:        []orderbook.Level{{Price: 50000, Amount: 0.1}},
		Bids:        []orderbook.Level{{Price: 49000, Amount: 0.2}},
		Exchange:    e.Name,
		Pair:        currency.NewBTCUSDT(),
		Asset:       asset.Spot,
		LastUpdated: time.Now(),
	})
	require.NoError(t, err)
	err = m.Resubscribe(e, conn, "ob.BTC_USDT.50", currency.NewBTCUSDT(), asset.Spot)
	require.NoError(t, err)
	require.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
}

func TestCompletedResubscribe(t *testing.T) {
	t.Parallel()

	m := newWSSubscriptionManager()
	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot) // no-op
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	m.lookup[key.PairAsset{Base: currency.BTC.Item, Quote: currency.USDT.Item, Asset: asset.Spot}] = true
	require.True(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
	m.CompletedResubscribe(currency.NewBTCUSDT(), asset.Spot)
	require.False(t, m.IsResubscribing(currency.NewBTCUSDT(), asset.Spot))
}

func TestQualifiedChannelKey_Match(t *testing.T) {
	t.Parallel()

	require.Implements(t, (*subscription.MatchableKey)(nil), new(qualifiedChannelKey))

	k := qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "test.channel"}}
	require.True(t, k.Match(k))
	require.False(t, k.Match(qualifiedChannelKey{&subscription.Subscription{QualifiedChannel: "TEST.channel"}}))
	require.NotNil(t, k.GetSubscription())
}
