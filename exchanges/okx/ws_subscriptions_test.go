package okx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestGetSpotMarginEvaluator(t *testing.T) {
	t.Parallel()
	eval := e.getSpotMarginEvaluator(nil)
	require.Empty(t, eval)

	spotBTCUSDTSub := &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: "trades"}
	futuresBTCUSDTSub := &subscription.Subscription{Asset: asset.USDTMarginedFutures, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: "trades"}
	marginBTCUSDTSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{currency.NewBTCUSDT()}, Channel: "trades"}

	subs := []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub, marginBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	require.True(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.Spot))
	require.False(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures))
	require.True(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.Margin))

	needed, err := eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as no spot or margin subscription exists")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Margin)
	require.NoError(t, err)
	require.False(t, needed, "must not be needed as spot subscription will be used")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as no margin subscription exists")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	err = e.Websocket.AddSuccessfulSubscriptions(nil, marginBTCUSDTSub)
	require.NoError(t, err)
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.False(t, needed, "must not be needed as margin subscription exists")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.False(t, needed, "must not be needed as margin subscription exists and only the spot sub is being removed")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{spotBTCUSDTSub, futuresBTCUSDTSub}
	err = e.Websocket.RemoveSubscriptions(nil, marginBTCUSDTSub)
	require.NoError(t, err)
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as margin subscription does not exist and the subscription is no longer required")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")

	subs = []*subscription.Subscription{marginBTCUSDTSub, futuresBTCUSDTSub}
	eval = e.getSpotMarginEvaluator(subs)
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Margin)
	require.NoError(t, err)
	require.True(t, needed, "must be needed as spot subscription does not exist and the subscription is no longer required")
	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "must be needed due to being a futures subscription")
}

func TestNeedsOutboundSubscription(t *testing.T) {
	t.Parallel()
	eval := make(spotMarginEvaluator)
	eval.add(currency.NewBTCUSDT(), "trades", asset.Spot, true)
	require.True(t, eval.exists(currency.NewBTCUSDT(), "trades", asset.Spot), "subscription must exist")
	needed, err := eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Spot)
	require.NoError(t, err)
	require.True(t, needed, "subscription must be needed")

	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.USDTMarginedFutures)
	require.NoError(t, err)
	require.True(t, needed, "subscription must be needed")

	needed, err = eval.NeedsOutboundSubscription(currency.NewBTCUSDT(), "trades", asset.Margin)
	require.ErrorIs(t, err, subscription.ErrNotFound)
	require.False(t, needed, "subscription must not be needed")
}

type subscriptionRecorderConnection struct {
	websocket.Connection
	subscriptions *subscription.Store
	requests      []WSSubscriptionInformationList
}

func (c *subscriptionRecorderConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	req, ok := payload.(WSSubscriptionInformationList)
	if !ok {
		return nil
	}
	c.requests = append(c.requests, req)
	return nil
}

func (c *subscriptionRecorderConnection) Subscriptions() *subscription.Store { return c.subscriptions }

func TestInverseSpotMarginSubscription(t *testing.T) {
	t.Parallel()

	pair := currency.NewBTCUSDT()
	for _, tc := range []struct {
		name     string
		sub      *subscription.Subscription
		expOK    bool
		expAsset asset.Item
	}{
		{name: "nil", sub: nil, expOK: false},
		{name: "spot to margin", sub: &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}, expOK: true, expAsset: asset.Margin},
		{name: "margin to spot", sub: &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}, expOK: true, expAsset: asset.Spot},
		{name: "non spot margin", sub: &subscription.Subscription{Asset: asset.USDTMarginedFutures, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}, expOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inverse, ok := inverseSpotMarginSubscription(tc.sub)
			require.Equal(t, tc.expOK, ok)
			if !tc.expOK {
				require.Nil(t, inverse)
				return
			}
			require.NotNil(t, inverse)
			require.NotSame(t, tc.sub, inverse)
			require.Equal(t, tc.expAsset, inverse.Asset)
			require.Equal(t, tc.sub.Channel, inverse.Channel)
			require.Equal(t, tc.sub.Pairs, inverse.Pairs)
			require.NotEqual(t, tc.sub.Asset, inverse.Asset)
		})
	}
}

func TestRefreshEquivalentOrderbookSnapshot(t *testing.T) {
	t.Run("CopiesInverseSnapshotToEquivalentAsset", func(t *testing.T) {
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))

		pair := currency.NewBTCUSDT()
		spotSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Pairs:            []currency.Pair{pair},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instID":"BTC-USDT"}`,
		}
		marginSub := &subscription.Subscription{
			Asset:            asset.Margin,
			Pairs:            append([]currency.Pair(nil), spotSub.Pairs...),
			Channel:          spotSub.Channel,
			QualifiedChannel: spotSub.QualifiedChannel,
		}
		exp := &orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Margin,
			LastUpdateID:      123,
			LastUpdated:       time.Unix(123, 0),
			Bids:              orderbook.Levels{{Price: 99, Amount: 2}},
			Asks:              orderbook.Levels{{Price: 100, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Exchange:          exp.Exchange,
			Pair:              spotSub.Pairs[0],
			Asset:             spotSub.Asset,
			LastUpdateID:      exp.LastUpdateID,
			LastUpdated:       exp.LastUpdated,
			Bids:              exp.Bids,
			Asks:              exp.Asks,
			ValidateOrderbook: exp.ValidateOrderbook,
		}))
		require.NoError(t, tracked.refreshEquivalentOrderbookSnapshot(marginSub))

		book, err := tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
		require.NoError(t, err)
		require.Equal(t, exp.Exchange, book.Exchange)
		require.Equal(t, exp.Pair, book.Pair)
		require.Equal(t, exp.Asset, book.Asset)
		require.Equal(t, exp.LastUpdateID, book.LastUpdateID)
		require.Equal(t, exp.LastUpdated, book.LastUpdated)
		require.Equal(t, exp.Bids, book.Bids)
		require.Equal(t, exp.Asks, book.Asks)
	})

	t.Run("IgnoresMissingInverseSnapshot", func(t *testing.T) {
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))

		err := tracked.refreshEquivalentOrderbookSnapshot(&subscription.Subscription{
			Asset:            asset.Margin,
			Pairs:            []currency.Pair{currency.NewBTCUSDT()},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instID":"BTC-USDT"}`,
		})
		require.NoError(t, err)
		_, err = tracked.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Margin)
		require.ErrorIs(t, err, orderbook.ErrDepthNotFound)
	})
}

func TestTrackEquivalentSubscriptionsOnExistingConnection(t *testing.T) {
	newEquivalentSubs := func() (*subscription.Subscription, *subscription.Subscription) {
		pair := currency.NewBTCUSDT()
		marginSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}
		spotSub := &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}
		marginSub.QualifiedChannel = `{"channel":"tickers","instID":"BTC-USDT"}`
		spotSub.QualifiedChannel = marginSub.QualifiedChannel
		return marginSub, spotSub
	}

	t.Run("TracksEquivalentOnOwningConnection", func(t *testing.T) {
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))
		marginSub, spotSub := newEquivalentSubs()

		existingConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(existingConn, marginSub))
		require.NoError(t, existingConn.subscriptions.Add(marginSub))

		remaining, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), existingConn, subscription.List{spotSub})
		require.NoError(t, err)
		require.Empty(t, remaining, "equivalent spot subscription must be tracked on the existing connection")
		require.Empty(t, existingConn.requests, "tracking on an existing connection must not emit a new outbound subscribe request")
		require.NotNil(t, tracked.Websocket.GetSubscription(spotSub), "spot subscription must be tracked logically")
		require.Len(t, existingConn.subscriptions.List(), 2, "existing connection store must track both logical subscriptions")
	})

	t.Run("SkipsConnectionWithoutInverse", func(t *testing.T) {
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))
		marginSub, spotSub := newEquivalentSubs()

		// The inverse (margin sub) lives in the manager-level store via a
		// different connection, but NOT on wrongConn's own subscription store.
		otherConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(otherConn, marginSub))
		require.NoError(t, otherConn.subscriptions.Add(marginSub))

		wrongConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}

		remaining, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), wrongConn, subscription.List{spotSub})
		require.NoError(t, err)
		require.Len(t, remaining, 1, "spot sub must NOT be tracked on a connection that doesn't own the inverse")
		require.Same(t, spotSub, remaining[0])
		require.Empty(t, wrongConn.subscriptions.List(), "wrongConn must not gain any subscriptions")
	})

	t.Run("RefreshesOrderbookSnapshotWhenTrackingEquivalentReenable", func(t *testing.T) {
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))

		pair := currency.NewBTCUSDT()
		spotSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Pairs:            []currency.Pair{pair},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instID":"BTC-USDT"}`,
		}
		marginSub := &subscription.Subscription{
			Asset:            asset.Margin,
			Pairs:            []currency.Pair{pair},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: spotSub.QualifiedChannel,
		}

		existingConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(existingConn, spotSub))
		require.NoError(t, existingConn.subscriptions.Add(spotSub))

		spotSnapshot := &orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Spot,
			LastUpdateID:      123,
			LastUpdated:       time.Unix(123, 0),
			Bids:              orderbook.Levels{{Price: 99, Amount: 2}},
			Asks:              orderbook.Levels{{Price: 100, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(spotSnapshot))
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(&orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Margin,
			LastUpdateID:      7,
			LastUpdated:       time.Unix(7, 0),
			Bids:              orderbook.Levels{{Price: 1, Amount: 1}},
			Asks:              orderbook.Levels{{Price: 2, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}))

		remaining, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), existingConn, subscription.List{marginSub})
		require.NoError(t, err)
		require.Empty(t, remaining)
		require.Empty(t, existingConn.requests, "equivalent re-enable remains a logical track, not a new outbound subscribe")

		marginBook, err := tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
		require.NoError(t, err)
		require.Equal(t, spotSnapshot.LastUpdateID, marginBook.LastUpdateID)
		require.Equal(t, spotSnapshot.LastUpdated, marginBook.LastUpdated)
		require.Equal(t, spotSnapshot.Bids, marginBook.Bids)
		require.Equal(t, spotSnapshot.Asks, marginBook.Asks)
	})
}
