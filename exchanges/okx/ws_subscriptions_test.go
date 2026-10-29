package okx

import (
	"context"
	"fmt"
	"slices"
	"sync"
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
	mu            sync.Mutex
	requests      []WSSubscriptionInformationList
}

func (c *subscriptionRecorderConnection) SendJSONMessage(_ context.Context, _ request.EndpointLimit, payload any) error {
	req, ok := payload.(WSSubscriptionInformationList)
	if !ok {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, req)
	return nil
}

func (c *subscriptionRecorderConnection) Subscriptions() *subscription.Store { return c.subscriptions }

// Requests returns a snapshot of recorded subscription requests.
func (c *subscriptionRecorderConnection) Requests() []WSSubscriptionInformationList {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.requests)
}

func TestSubscriptionRecorderConnectionRequests(t *testing.T) {
	t.Parallel()

	conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
	require.NoError(t, conn.SendJSONMessage(t.Context(), 0, WSSubscriptionInformationList{Operation: operationSubscribe}), "SendJSONMessage must not error")

	requests := conn.Requests()
	require.Len(t, requests, 1, "Requests must return the recorded request")
	requests[0].Operation = operationUnsubscribe
	require.Equal(t, operationSubscribe, conn.Requests()[0].Operation, "Requests must return an independent slice")
}

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
			QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`,
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
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange:          exp.Exchange,
			Pair:              spotSub.Pairs[0],
			Asset:             spotSub.Asset,
			LastUpdateID:      exp.LastUpdateID,
			LastUpdated:       exp.LastUpdated,
			Bids:              exp.Bids,
			Asks:              exp.Asks,
			ValidateOrderbook: exp.ValidateOrderbook,
		}))
		require.NoError(t, tracked.refreshEquivalentOrderbookSnapshot(t.Context(), marginSub))

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

		err := tracked.refreshEquivalentOrderbookSnapshot(t.Context(), &subscription.Subscription{
			Asset:            asset.Margin,
			Pairs:            []currency.Pair{currency.NewBTCUSDT()},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`,
		})
		require.NoError(t, err)
		_, err = tracked.Websocket.Orderbook.GetOrderbook(currency.NewBTCUSDT(), asset.Margin)
		require.ErrorIs(t, err, orderbook.ErrDepthNotFound)
	})
}

func TestTrackEquivalentSubscriptionsOnExistingConnection(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{channelInstruments, liquidationOrders, channelOrders, subscription.MyOrdersChannel} {
		for _, a := range []asset.Item{asset.Spot, asset.Margin} {
			t.Run(channel+"/"+a.String(), func(t *testing.T) {
				t.Parallel()
				tracked := new(Exchange)
				require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
				other := asset.Margin
				if a == asset.Margin {
					other = asset.Spot
				}
				incoming := &subscription.Subscription{
					Asset: a, Channel: channel, Pairs: currency.Pairs{currency.NewBTCUSDT()},
					QualifiedChannel: fmt.Sprintf(`{"channel":%q,"instType":%q}`, channelName(&subscription.Subscription{Channel: channel}), GetInstrumentTypeFromAssetItem(a)),
				}
				existing := incoming.Clone()
				existing.Asset = other
				conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
				require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(conn, existing), "existing asset must be tracked by the manager")
				require.NoError(t, conn.subscriptions.Add(existing), "existing asset must belong to the connection")
				remaining, attached, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), conn, subscription.List{incoming})
				require.NoError(t, err, "asset-scoped subscription must be evaluated")
				require.Equal(t, subscription.List{incoming}, remaining, "different instrument type must remain for outbound subscription")
				require.Empty(t, attached, "asset-scoped subscriptions must not be attached as equivalents")
				require.Empty(t, conn.Requests(), "equivalence hook must leave outbound handling to the caller")
			})
		}
	}

	newEquivalentSubs := func() (*subscription.Subscription, *subscription.Subscription) {
		pair := currency.NewBTCUSDT()
		marginSub := &subscription.Subscription{Asset: asset.Margin, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}
		spotSub := &subscription.Subscription{Asset: asset.Spot, Pairs: []currency.Pair{pair}, Channel: subscription.TickerChannel}
		marginSub.QualifiedChannel = `{"channel":"tickers","instId":"BTC-USDT"}`
		spotSub.QualifiedChannel = marginSub.QualifiedChannel
		return marginSub, spotSub
	}

	t.Run("TracksEquivalentOnOwningConnection", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))
		marginSub, spotSub := newEquivalentSubs()

		existingConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(existingConn, marginSub))
		require.NoError(t, existingConn.subscriptions.Add(marginSub))

		remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), existingConn, subscription.List{spotSub})
		require.NoError(t, err)
		require.Empty(t, remaining, "equivalent spot subscription must be tracked on the existing connection")
		require.Len(t, trackedSubs, 1)
		require.Same(t, spotSub, trackedSubs[0])
		require.Empty(t, existingConn.Requests(), "tracking on an existing connection must not emit a new outbound subscribe request")
		require.Nil(t, tracked.Websocket.GetSubscription(spotSub), "hook must not mutate manager-level tracking state directly")
		require.Len(t, existingConn.subscriptions.List(), 1, "hook must not mutate connection subscription stores directly")
	})

	t.Run("SkipsConnectionWithoutInverse", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))
		marginSub, spotSub := newEquivalentSubs()

		// The inverse (margin sub) lives in the manager-level store via a
		// different connection, but NOT on wrongConn's own subscription store.
		otherConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, tracked.Websocket.AddSuccessfulSubscriptions(otherConn, marginSub))
		require.NoError(t, otherConn.subscriptions.Add(marginSub))

		wrongConn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}

		remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), wrongConn, subscription.List{spotSub})
		require.NoError(t, err)
		require.Len(t, remaining, 1, "spot sub must NOT be tracked on a connection that doesn't own the inverse")
		require.Same(t, spotSub, remaining[0])
		require.Empty(t, trackedSubs)
		require.Empty(t, wrongConn.subscriptions.List(), "wrongConn must not gain any subscriptions")
	})

	t.Run("RefreshesOrderbookSnapshotWhenTrackingEquivalentReenable", func(t *testing.T) {
		t.Parallel()
		tracked := new(Exchange)
		require.NoError(t, testexch.Setup(tracked))

		pair := currency.NewBTCUSDT()
		spotSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Pairs:            []currency.Pair{pair},
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: `{"channel":"books","instId":"BTC-USDT"}`,
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
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), spotSnapshot))
		require.NoError(t, tracked.Websocket.Orderbook.LoadSnapshot(t.Context(), &orderbook.Book{
			Exchange:          tracked.Name,
			Pair:              pair,
			Asset:             asset.Margin,
			LastUpdateID:      7,
			LastUpdated:       time.Unix(7, 0),
			Bids:              orderbook.Levels{{Price: 1, Amount: 1}},
			Asks:              orderbook.Levels{{Price: 2, Amount: 1}},
			ValidateOrderbook: tracked.ValidateOrderbook,
		}))

		remaining, trackedSubs, err := tracked.trackEquivalentSubscriptionsOnExistingConnection(t.Context(), existingConn, subscription.List{marginSub})
		require.NoError(t, err)
		require.Empty(t, remaining)
		require.Len(t, trackedSubs, 1)
		require.Same(t, marginSub, trackedSubs[0])
		require.Empty(t, existingConn.Requests(), "equivalent re-enable remains a logical track, not a new outbound subscribe")

		marginBook, err := tracked.Websocket.Orderbook.GetOrderbook(pair, asset.Margin)
		require.NoError(t, err)
		require.Equal(t, spotSnapshot.LastUpdateID, marginBook.LastUpdateID)
		require.Equal(t, spotSnapshot.LastUpdated, marginBook.LastUpdated)
		require.Equal(t, spotSnapshot.Bids, marginBook.Bids)
		require.Equal(t, spotSnapshot.Asks, marginBook.Asks)
	})
}

// Casing drift between the template and the tag drops the instrument silently.
func TestChunkRequestsCarriesInstrumentID(t *testing.T) {
	t.Parallel()
	subs, err := e.generateSubscriptions(true)
	require.NoError(t, err, "generateSubscriptions must not error")

	var checked int
	for _, sub := range subs {
		if len(sub.Pairs) == 0 {
			continue // channel is not pair specific, so it carries no instrument
		}
		reqs, err := e.chunkRequests(subscription.List{sub}, "subscribe")
		require.NoError(t, err, "chunkRequests must not error")
		for _, req := range reqs {
			for _, arg := range req.Arguments {
				require.Falsef(t, arg.InstrumentID.IsEmpty(),
					"%s must keep its instrument through the template round-trip; got %q", sub.Channel, sub.QualifiedChannel)
				checked++
			}
		}
	}
	require.Positive(t, checked, "must exercise at least one pair specific subscription")
}

func TestIsSymbolChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		want    bool
	}{
		{subscription.CandlesChannel, true},
		{subscription.TickerChannel, true},
		{subscription.OrderbookChannel, true},
		{subscription.AllTradesChannel, true},
		{channelTickers, true},
		{channelTrades, true},
		{channelOrderBooks, true},
		{channelOrderBooks5, true},
		{channelOrderBooks50TBT, true},
		{channelOrderBooksTBT, true},
		{channelBBOTBT, true},
		{channelFundingRate, true},
		{channelMarkPrice, true},
		{channelOpenInterest, true},
		{channelPriceLimit, true},
		{channelInstruments, false},
		{liquidationOrders, false},
		{channelOrders, false},
		{subscription.MyAccountChannel, false},
		{subscription.MyOrdersChannel, false},
		{"unknown", false},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isSymbolChannel(&subscription.Subscription{Channel: tc.channel}), "Symbol channel classification must match the API scope")
			if !tc.want {
				return
			}
			tracked := new(Exchange)
			require.NoError(t, testexch.Setup(tracked), "Test exchange setup must succeed")
			a, pair := asset.Spot, mainPair
			if tc.channel == channelFundingRate || tc.channel == channelMarkPrice || tc.channel == channelOpenInterest || tc.channel == channelPriceLimit {
				a, pair = asset.PerpetualSwap, perpetualSwapPair
			}
			tracked.Features.Subscriptions = subscription.List{{Enabled: true, Channel: tc.channel, Asset: a, Pairs: currency.Pairs{pair}}}
			got, err := tracked.generateSubscriptions(true)
			require.NoError(t, err, "Symbol subscription expansion must succeed")
			require.Len(t, got, 1, "Explicit single instrument must produce one subscription")
			require.Contains(t, got[0].QualifiedChannel, fmt.Sprintf(`"instId":%q`, pair.String()), "Symbol channels must include the subscribed instrument")
			require.Contains(t, got[0].QualifiedChannel, fmt.Sprintf(`"channel":%q`, channelName(tracked.Features.Subscriptions[0])), "Expansion must preserve the chosen exchange channel")
		})
	}
}

func TestIsAssetChannel(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		channel string
		want    bool
	}{
		{subscription.MyOrdersChannel, true},
		{channelOrders, true},
		{channelInstruments, true},
		{liquidationOrders, true},
		{subscription.MyAccountChannel, false},
		{subscription.TickerChannel, false},
		{channelTickers, false},
		{"unknown", false},
	} {
		t.Run(tc.channel, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isAssetChannel(&subscription.Subscription{Channel: tc.channel}), "asset channel classification must match the API scope")
			if !tc.want {
				return
			}
			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "test exchange setup must succeed")
			authenticated := tc.channel == channelOrders || tc.channel == subscription.MyOrdersChannel
			ex.Websocket.SetCanUseAuthenticatedEndpoints(authenticated)
			ex.Features.Subscriptions = subscription.List{{Enabled: true, Channel: tc.channel, Asset: asset.PerpetualSwap, Authenticated: authenticated}}
			subs, err := ex.generateSubscriptions(!authenticated)
			require.NoError(t, err, "asset subscription expansion must succeed")
			require.Len(t, subs, 1, "all enabled instruments must share one asset subscription")
			require.JSONEq(t, fmt.Sprintf(`{"channel":%q,"instType":"SWAP"}`, channelName(ex.Features.Subscriptions[0])), subs[0].QualifiedChannel, "asset channels must omit instrument-specific arguments")
		})
	}
}

func TestHandleSubscription(t *testing.T) {
	t.Parallel()
	for _, channel := range []string{channelInstruments, liquidationOrders, channelOrders, subscription.MyOrdersChannel} {
		for _, batch := range []bool{false, true} {
			for _, withPairs := range []bool{false, true} {
				for _, first := range []asset.Item{asset.Spot, asset.Margin} {
					t.Run(fmt.Sprintf("%s/batch=%t/pairs=%t/first=%s", channel, batch, withPairs, first), func(t *testing.T) {
						t.Parallel()
						tracked := new(Exchange)
						require.NoError(t, testexch.Setup(tracked), "exchange must initialise")
						second := asset.Margin
						if first == asset.Margin {
							second = asset.Spot
						}
						var subs subscription.List
						for _, a := range []asset.Item{first, second} {
							sub := &subscription.Subscription{
								Channel: channel, Asset: a,
								QualifiedChannel: fmt.Sprintf(`{"channel":%q,"instType":%q}`, channelName(&subscription.Subscription{Channel: channel}), GetInstrumentTypeFromAssetItem(a)),
							}
							if withPairs {
								sub.Pairs = currency.Pairs{currency.NewBTCUSDT()}
							}
							subs = append(subs, sub)
						}
						conn := &subscriptionRecorderConnection{subscriptions: subscription.NewStore()}
						for _, operation := range []string{operationSubscribe, operationUnsubscribe} {
							before := len(conn.Requests())
							if batch {
								require.NoError(t, tracked.handleSubscription(t.Context(), conn, operation, subs), "both assets must be sent together")
							} else {
								for _, sub := range subs {
									require.NoError(t, tracked.handleSubscription(t.Context(), conn, operation, subscription.List{sub}), "each asset must be sent independently")
								}
							}
							var arguments []SubscriptionInfo
							for _, req := range conn.Requests()[before:] {
								require.Equal(t, operation, req.Operation, "outbound operation must match")
								arguments = append(arguments, req.Arguments...)
							}
							require.Equal(t, []SubscriptionInfo{
								{Channel: channelName(subs[0]), InstrumentType: GetInstrumentTypeFromAssetItem(first)},
								{Channel: channelName(subs[1]), InstrumentType: GetInstrumentTypeFromAssetItem(second)},
							}, arguments, "outbound requests must retain both instrument types")
							for _, sub := range subs {
								if operation == operationSubscribe {
									require.NotNil(t, tracked.Websocket.GetSubscription(sub), "sent subscriptions must be tracked")
								} else {
									require.Nil(t, tracked.Websocket.GetSubscription(sub), "unsubscribed assets must be removed")
								}
							}
						}
					})
				}
			}
		}
	}
}
