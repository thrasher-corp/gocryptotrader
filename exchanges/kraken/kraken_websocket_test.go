package kraken

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

type mockAuthSubConnection struct {
	websocket.Connection
	responses [][]byte
	expected  int
}

func (m *mockAuthSubConnection) SendMessageReturnResponses(_ context.Context, _ request.EndpointLimit, _, _ any, expected int) ([][]byte, error) {
	m.expected = expected
	return m.responses, nil
}

// TestGenerateSubscriptions tests the subscriptions generated from configuration
func TestGenerateSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup instance must not error")

	pairs, err := ex.GetEnabledPairs(asset.Spot)
	require.NoError(t, err, "GetEnabledPairs must not error")
	require.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "Websocket must not be authenticated by default")
	exp := subscription.List{
		{Channel: subscription.TickerChannel},
		{Channel: subscription.AllTradesChannel},
		{Channel: subscription.CandlesChannel, Interval: kline.OneMin},
		{Channel: subscription.OrderbookChannel, Levels: 1000},
	}
	for _, s := range exp {
		s.QualifiedChannel = channelName(s)
		s.Asset = asset.Spot
		s.Pairs = pairs
	}
	subs, err := ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)

	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	exp = append(exp, subscription.List{
		{Channel: subscription.MyOrdersChannel, QualifiedChannel: krakenWsOpenOrders},
		{Channel: subscription.MyTradesChannel, QualifiedChannel: krakenWsOwnTrades},
	}...)
	subs, err = ex.generateSubscriptions()
	require.NoError(t, err, "generateSubscriptions must not error")
	testsubs.EqualLists(t, exp, subs)
}

func TestGeneratePrivateSubscriptions(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup instance must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	subs, err := ex.generatePrivateSubscriptions()
	require.NoError(t, err, "generatePrivateSubscriptions must not error")
	testsubs.EqualLists(t, subscription.List{
		{Channel: subscription.MyOrdersChannel, QualifiedChannel: krakenWsOpenOrders},
		{Channel: subscription.MyTradesChannel, QualifiedChannel: krakenWsOwnTrades},
	}, subs)
}

func TestSplitPairSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("splits multi-pair subscriptions", func(t *testing.T) {
		t.Parallel()

		ethUSDPair := currency.NewPair(currency.ETH, currency.USD)
		onePairSub := &subscription.Subscription{
			Asset:   asset.Spot,
			Channel: subscription.TickerChannel,
			Pairs:   currency.Pairs{spotTestPair},
		}
		multiPairSub := &subscription.Subscription{
			Asset:   asset.Spot,
			Channel: subscription.OrderbookChannel,
			Pairs:   currency.Pairs{spotTestPair, ethUSDPair},
			Levels:  1000,
		}
		noPairSub := &subscription.Subscription{
			Channel:       subscription.MyOrdersChannel,
			Authenticated: true,
		}

		subs := splitPairSubscriptions(subscription.List{noPairSub, onePairSub, multiPairSub})
		require.Len(t, subs, 4, "splitPairSubscriptions must split multi-pair subscriptions")
		assert.Empty(t, subs[0].Pairs, "no-pair subscription should be retained")
		assert.Equal(t, currency.Pairs{spotTestPair}, subs[1].Pairs, "single-pair subscription should be retained")
		assert.Equal(t, currency.Pairs{spotTestPair}, subs[2].Pairs, "first split subscription should contain the first pair")
		assert.Equal(t, currency.Pairs{ethUSDPair}, subs[3].Pairs, "second split subscription should contain the second pair")
		assert.Len(t, multiPairSub.Pairs, 2, "original subscription should not be modified")
	})

	t.Run("keeps connection batches under configured pair budget", func(t *testing.T) {
		t.Parallel()

		const pairLimit = 200
		makePairs := func(prefix string) currency.Pairs {
			pairs := make(currency.Pairs, 100)
			for i := range pairs {
				pairs[i] = currency.NewPair(currency.XBT, currency.NewCode(prefix+strconv.Itoa(i)))
			}
			return pairs
		}

		reviewerExample := subscription.List{
			{Asset: asset.Spot, Channel: subscription.TickerChannel, Pairs: makePairs("AAA")},
			{Asset: asset.Spot, Channel: subscription.AllTradesChannel, Pairs: makePairs("BBB")},
			{Asset: asset.Spot, Channel: subscription.OrderbookChannel, Pairs: makePairs("CCC"), Levels: 1000},
		}
		var totalPairs int
		for _, sub := range reviewerExample {
			totalPairs += len(sub.Pairs)
		}
		require.Equal(t, 300, totalPairs, "reviewer example must contain 300 symbols across three requests")

		unsplitRequests := groupSubscriptionsByRequestLimit(reviewerExample, pairLimit)
		require.Len(t, unsplitRequests, 3, "reviewer example must start as three grouped requests")
		for _, req := range unsplitRequests {
			assert.Len(t, req.Pairs, 100, "each grouped request should contain 100 pairs")
		}

		subs := splitPairSubscriptions(reviewerExample)
		require.Len(t, subs, 300, "splitPairSubscriptions must create one logical subscription per channel symbol")

		countPairs := func(requests subscription.List) int {
			var count int
			for _, req := range requests {
				count += len(req.Pairs)
			}
			return count
		}

		firstConnectionRequests := groupSubscriptionsByRequestLimit(subs[:pairLimit], pairLimit)
		require.Len(t, firstConnectionRequests, 2, "first connection subscriptions must group into two requests")
		assert.Equal(t, pairLimit, countPairs(firstConnectionRequests), "first connection requests should contain the configured pair budget")

		secondConnectionRequests := groupSubscriptionsByRequestLimit(subs[pairLimit:], pairLimit)
		require.Len(t, secondConnectionRequests, 1, "remaining subscription must group into one request")
		assert.Equal(t, 100, countPairs(secondConnectionRequests), "second connection requests should contain the remaining symbols")
	})
}

func TestGeneratePublicSubscriptions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		canAuth bool
	}{
		{name: "without authenticated endpoints"},
		{name: "with authenticated endpoints", canAuth: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup instance must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(tc.canAuth)

			pairs, err := ex.GetEnabledPairs(asset.Spot)
			require.NoError(t, err, "GetEnabledPairs must not error")
			templates := subscription.List{
				{Channel: subscription.TickerChannel},
				{Channel: subscription.AllTradesChannel},
				{Channel: subscription.CandlesChannel, Interval: kline.OneMin},
				{Channel: subscription.OrderbookChannel, Levels: 1000},
			}
			exp := make(subscription.List, 0, len(templates)*len(pairs))
			for _, s := range templates {
				for _, pair := range pairs {
					exp = append(exp, &subscription.Subscription{
						Channel:          s.Channel,
						QualifiedChannel: channelName(s),
						Asset:            asset.Spot,
						Interval:         s.Interval,
						Levels:           s.Levels,
						Pairs:            currency.Pairs{pair},
					})
				}
			}

			subs, err := ex.generatePublicSubscriptions()
			require.NoError(t, err, "generatePublicSubscriptions must not error")
			require.Len(t, subs, len(exp), "generatePublicSubscriptions must return public channel-pair subscriptions only")
			testsubs.EqualLists(t, exp, subs)
			for _, sub := range subs {
				assert.False(t, sub.Authenticated, "public subscription should not be authenticated")
				assert.Len(t, sub.Pairs, 1, "public subscription should contain one pair for connection scaling")
			}
		})
	}
}

func TestWsAuthenticate(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup instance must not error")
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)

	err := ex.wsAuthenticate(t.Context(), nil)
	require.Error(t, err, "wsAuthenticate must error without credentials")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "websocket auth endpoints should be disabled after auth failure")
}

func TestGroupSubscriptionsByRequestLimit(t *testing.T) {
	t.Parallel()

	pairs := make(currency.Pairs, 5)
	for i := range pairs {
		pairs[i] = currency.NewPair(currency.NewCode("XBT"), currency.NewCode("USD"+string(rune('A'+i))))
	}
	subs := subscription.List{
		{
			Asset:            asset.Spot,
			Channel:          subscription.CandlesChannel,
			QualifiedChannel: krakenWsOHLC,
			Interval:         kline.OneMin,
			Pairs:            pairs[:3],
		},
		{
			Asset:            asset.Spot,
			Channel:          subscription.CandlesChannel,
			QualifiedChannel: krakenWsOHLC,
			Interval:         kline.OneMin,
			Pairs:            pairs[3:],
		},
	}

	got := groupSubscriptionsByRequestLimit(subs, 2)
	require.Len(t, got, 3, "groupSubscriptionsByRequestLimit must batch grouped pairs")
	assert.Len(t, got[0].Pairs, 2, "first batch should contain pair limit")
	assert.Len(t, got[1].Pairs, 2, "second batch should contain pair limit")
	assert.Len(t, got[2].Pairs, 1, "third batch should contain remainder")
	for _, sub := range got {
		assert.LessOrEqual(t, len(sub.Pairs), 2, "batched subscription pair count should not exceed limit")
		assert.Equal(t, subscription.CandlesChannel, sub.Channel, "batched subscription should retain channel")
		assert.Equal(t, kline.OneMin, sub.Interval, "batched subscription should retain interval")
	}
}

func TestManageSubs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		channel          string
		qualifiedChannel string
		response         []byte
		responseCount    int
		errIs            error
		errContains      string
	}{
		{name: "own trades", channel: subscription.MyTradesChannel, qualifiedChannel: krakenWsOwnTrades, response: []byte(`{"channelName":"ownTrades","event":"subscriptionStatus","reqid":3,"status":"subscribed","subscription":{"name":"ownTrades"}}`), responseCount: 1},
		{name: "open orders", channel: subscription.MyOrdersChannel, qualifiedChannel: krakenWsOpenOrders, response: []byte(`{"channelName":"openOrders","event":"subscriptionStatus","reqid":3,"status":"subscribed","subscription":{"name":"openOrders"}}`), responseCount: 1},
		{name: "requires single response", channel: subscription.MyTradesChannel, qualifiedChannel: krakenWsOwnTrades, responseCount: 0, errIs: errExpectedOneSubResponse, errContains: "got 0; Channel: myTrades"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ex := new(Exchange)
			require.NoError(t, testexch.Setup(ex), "Setup Instance must not error")

			conn := &mockAuthSubConnection{expected: -1}
			if tc.responseCount > 0 {
				conn.responses = [][]byte{tc.response}
			}
			ex.Websocket.AuthConn = conn

			err := ex.manageSubs(t.Context(), krakenWsSubscribe, subscription.List{{
				Channel:          tc.channel,
				QualifiedChannel: tc.qualifiedChannel,
				Authenticated:    true,
			}}, conn)
			if tc.errIs != nil {
				require.ErrorIs(t, err, tc.errIs)
				require.ErrorContains(t, err, tc.errContains)
			} else {
				require.NoError(t, err, "auth subscription without pairs must not error")
			}
			assert.Equal(t, 1, conn.expected, "auth subscription without pairs waits for one response")
		})
	}
}

func TestWsProcessSubStatusInvalidPair(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup Instance must not error")
	s := &subscription.Subscription{
		Channel: subscription.TickerChannel,
		Pairs:   currency.Pairs{currency.NewBTCUSD()},
	}
	require.NoError(t, ex.Websocket.AddSubscriptions(nil, s), "subscription must be added in subscribing state")

	ex.wsProcessSubStatus(nil, []byte(`{"channelName":"ticker","event":"subscriptionStatus","pair":"not-a-pair","status":"subscribed","subscription":{"name":"ticker"}}`))
	assert.Equal(t, subscription.SubscribingState, s.State(), "invalid websocket subscription pair should leave the subscription state unchanged")
}

func TestWsProcessSubStatus(t *testing.T) {
	t.Parallel()

	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup instance must not error")
	sub := &subscription.Subscription{
		Channel: subscription.TickerChannel,
		Pairs:   currency.Pairs{currency.NewPairWithDelimiter("XBT", "USD", "/")},
	}
	require.NoError(t, ex.Websocket.AddSubscriptions(nil, sub), "subscription must be added in subscribing state")

	ex.wsProcessSubStatus(nil, []byte(`{"channelName":"ticker","event":"subscriptionStatus","pair":"XBT/USD","status":"subscribed","subscription":{"name":"ticker"}}`))
	assert.Equal(t, subscription.SubscribedState, sub.State(), "slash-delimited websocket subscription pair should match the stored subscription")
}

func TestGetWSToken(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	testexch.SetupWs(t, e)

	resp, err := e.GetWebsocketToken(t.Context())
	require.NoError(t, err, "GetWebsocketToken must not error")
	assert.NotEmpty(t, resp, "Token should not be empty")
}

func TestSubscribeForConnection(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))

	conn, err := k.Websocket.GetConnection("auth")
	require.NoError(t, err, "GetConnection must not error")

	subs := subscription.List{
		{
			Asset:            asset.Spot,
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.OrderbookChannel}),
			Pairs:            currency.Pairs{spotTestPair},
			Levels:           1000,
		},
		{
			Asset:            asset.Spot,
			Channel:          subscription.OrderbookChannel,
			QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.OrderbookChannel}),
			Pairs:            currency.Pairs{currency.NewPair(currency.ETH, currency.USD)},
			Levels:           1000,
		},
	}

	require.NoError(t, k.subscribeForConnection(t.Context(), conn, subs), "subscribeForConnection must not error")
	for _, s := range subs {
		got := k.Websocket.GetSubscription(s)
		require.NotNil(t, got, "subscription must be stored")
		assert.Equal(t, subscription.SubscribedState, got.State(), "subscription should transition to subscribed state")
	}
}

func TestSubscribeForConnectionResubscribe(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))

	conn, err := k.Websocket.GetConnection("auth")
	require.NoError(t, err, "GetConnection must not error")

	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.TickerChannel,
		QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
		Pairs:            currency.Pairs{spotTestPair},
	}

	require.NoError(t, k.Websocket.AddSubscriptions(conn, sub), "AddSubscriptions must not error")
	require.NoError(t, sub.SetState(subscription.ResubscribingState), "SetState must not error")

	require.NoError(t, k.subscribeForConnection(t.Context(), conn, subscription.List{sub}), "subscribeForConnection must not error")
	got := k.Websocket.GetSubscription(sub)
	require.NotNil(t, got, "resubscribing subscription must be stored")
	assert.Equal(t, subscription.SubscribedState, got.State(), "resubscribing subscription should transition to subscribed state")
}

func TestSubscribeForConnectionSkipsFailedAdds(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))

	conn, err := k.Websocket.GetConnection("auth")
	require.NoError(t, err, "GetConnection must not error")

	existing := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.TickerChannel,
		QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
		Pairs:            currency.Pairs{spotTestPair},
	}
	require.NoError(t, k.Websocket.AddSubscriptions(conn, existing), "AddSubscriptions must not error")

	duplicate := existing.Clone()
	fresh := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.TickerChannel,
		QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
		Pairs:            currency.Pairs{currency.NewPair(currency.ETH, currency.USD)},
	}

	err = k.subscribeForConnection(t.Context(), conn, subscription.List{duplicate, fresh})
	require.ErrorIs(t, err, subscription.ErrDuplicate, "subscribeForConnection must surface duplicate subscription errors")
	assert.Equal(t, subscription.InactiveState, duplicate.State(), "failed subscription should be set inactive")
	assert.Equal(t, subscription.SubscribedState, fresh.State(), "valid subscription should still subscribe")
}

func TestCleanupUnsubscribedSubs(t *testing.T) {
	t.Parallel()

	t.Run("removes inactive subscriptions", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup instance must not error")
		inactiveSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Channel:          subscription.TickerChannel,
			QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
			Pairs:            currency.Pairs{spotTestPair},
		}
		subscribedSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Channel:          subscription.TickerChannel,
			QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
			Pairs:            currency.Pairs{currency.NewPair(currency.ETH, currency.USD)},
		}
		require.NoError(t, ex.Websocket.AddSubscriptions(nil, inactiveSub, subscribedSub), "AddSubscriptions must not error")
		require.NoError(t, subscribedSub.SetState(subscription.SubscribedState), "subscribed sub SetState must not error")

		err := ex.cleanupUnsubscribedSubs(nil, subscription.List{inactiveSub, subscribedSub})
		require.NoError(t, err, "cleanupUnsubscribedSubs must not error")
		assert.Equal(t, subscription.UnsubscribedState, inactiveSub.State(), "failed subscription should be removed from the store")
		assert.Equal(t, subscription.SubscribedState, subscribedSub.State(), "subscribed subscription should be left alone")
		assert.Nil(t, ex.Websocket.GetSubscription(inactiveSub), "failed subscription should be removed")
		assert.NotNil(t, ex.Websocket.GetSubscription(subscribedSub), "subscribed subscription should remain stored")
	})

	t.Run("returns remove errors", func(t *testing.T) {
		t.Parallel()

		ex := new(Exchange)
		require.NoError(t, testexch.Setup(ex), "Setup instance must not error")
		missingSub := &subscription.Subscription{
			Asset:            asset.Spot,
			Channel:          subscription.TickerChannel,
			QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
			Pairs:            currency.Pairs{spotTestPair},
		}

		err := ex.cleanupUnsubscribedSubs(nil, subscription.List{missingSub})
		require.ErrorIs(t, err, subscription.ErrNotFound, "cleanupUnsubscribedSubs must return remove subscription errors")
		assert.ErrorContains(t, err, "error removing failed subscription", "cleanupUnsubscribedSubs should include removal context")
	})
}

func TestSubscribeForConnectionCandlesInterval(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))

	conn, err := k.Websocket.GetConnection("auth")
	require.NoError(t, err, "GetConnection must not error")

	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.CandlesChannel,
		QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.CandlesChannel, Interval: kline.FiveMin}),
		Pairs:            currency.Pairs{spotTestPair},
		Interval:         kline.FiveMin,
	}

	require.NoError(t, k.subscribeForConnection(t.Context(), conn, subscription.List{sub}), "subscribeForConnection with candle interval must not error")
	got := k.Websocket.GetSubscription(sub)
	require.NotNil(t, got, "interval subscription must be stored")
	assert.Equal(t, subscription.SubscribedState, got.State(), "interval subscription should transition to subscribed state")
}

func TestUnsubscribeForConnection(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))

	conn, err := k.Websocket.GetConnection("auth")
	require.NoError(t, err, "GetConnection must not error")

	sub := &subscription.Subscription{
		Asset:            asset.Spot,
		Channel:          subscription.TickerChannel,
		QualifiedChannel: channelName(&subscription.Subscription{Channel: subscription.TickerChannel}),
		Pairs:            currency.Pairs{spotTestPair},
	}

	require.NoError(t, k.Websocket.AddSubscriptions(conn, sub), "AddSubscriptions must not error")

	require.NoError(t, k.unsubscribeForConnection(t.Context(), conn, subscription.List{sub}), "unsubscribeForConnection must not error")
	got := k.Websocket.GetSubscription(sub)
	assert.Nil(t, got, "subscription should be removed after unsubscribe")
}

// TestWsAddOrder exercises roundtrip of wsAddOrder; See also: mockWsAddOrder
func TestWsAddOrder(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))
	require.True(t, k.IsWebsocketAuthenticationSupported(), "WS must be authenticated")
	id, err := k.wsAddOrder(t.Context(), &WsAddOrderRequest{
		OrderType: order.Limit.Lower(),
		OrderSide: order.Buy.Lower(),
		Pair:      "XBT/USD",
		Price:     80000,
	})
	require.NoError(t, err, "wsAddOrder must not error")
	assert.Equal(t, "ONPNXH-KMKMU-F4MR5V", id, "wsAddOrder should return correct order ID")
}

// TestWsCancelOrders exercises roundtrip of wsCancelOrders; See also: mockWsCancelOrders
func TestWsCancelOrders(t *testing.T) {
	t.Parallel()

	k := testexch.MockWsInstance[Exchange](t, mockWsHandler(t, mockWsServer))
	require.True(t, k.IsWebsocketAuthenticationSupported(), "WS must be authenticated")

	err := k.wsCancelOrders(t.Context(), []string{"RABBIT", "BATFISH", "SQUIRREL", "CATFISH", "MOUSE"})
	assert.ErrorIs(t, err, errCancellingOrder, "Should error cancelling order")
	assert.ErrorContains(t, err, "BATFISH", "Should error containing txn id")
	assert.ErrorContains(t, err, "CATFISH", "Should error containing txn id")
	assert.ErrorContains(t, err, "[EOrder:Unknown order]", "Should error containing server error")

	err = k.wsCancelOrders(t.Context(), []string{"RABBIT", "SQUIRREL", "MOUSE"})
	assert.NoError(t, err, "Should not error with valid ids")
}

func TestWsCancelAllOrders(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	testexch.SetupWs(t, e)
	_, err := e.wsCancelAllOrders(t.Context())
	require.NoError(t, err, "wsCancelAllOrders must not error")
}

func TestWsHandleData(t *testing.T) {
	t.Parallel()
	// Use a dedicated exchange name so checksum-sensitive fixtures do not contend
	// with global orderbook cache entries updated by other websocket tests.
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	e.Name += "-WsHandleData"
	for _, l := range []int{10, 100} {
		err := e.Websocket.AddSuccessfulSubscriptions(nil, &subscription.Subscription{
			Channel: subscription.OrderbookChannel,
			Pairs:   currency.Pairs{spotTestPair},
			Asset:   asset.Spot,
			Levels:  l,
		})
		require.NoError(t, err, "AddSuccessfulSubscriptions must not error")
	}
	conn := testexch.GetMockConn(t, e, "")
	testexch.FixtureToDataHandler(t, "testdata/wsHandleData.json", func(ctx context.Context, b []byte) error { return e.wsHandleData(ctx, conn, b) })
}

func TestWSProcessTrades(t *testing.T) {
	t.Parallel()

	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	err := e.Websocket.AddSubscriptions(nil, &subscription.Subscription{Asset: asset.Spot, Pairs: currency.Pairs{spotTestPair}, Channel: subscription.AllTradesChannel, Key: 18788})
	require.NoError(t, err, "AddSubscriptions must not error")
	conn := testexch.GetMockConn(t, e, "")
	testexch.FixtureToDataHandler(t, "testdata/wsAllTrades.json", func(ctx context.Context, b []byte) error { return e.wsHandleData(ctx, conn, b) })
	e.Websocket.DataHandler.Close()

	invalid := []any{"trades", []any{[]any{"95873.80000", "0.00051182", "1708731380.3791859"}}}
	rawBytes, err := json.Marshal(invalid)
	require.NoError(t, err, "Marshal must not error marshalling invalid trade data")

	pair := currency.NewPair(currency.XBT, currency.USD)
	err = e.wsProcessTrades(t.Context(), json.RawMessage(rawBytes), pair)
	require.ErrorContains(t, err, "error unmarshalling trade data")

	expJSON := []string{
		`{"AssetType":"spot","CurrencyPair":"XBT/USD","Side":"BUY","Price":95873.80000,"Amount":0.00051182,"Timestamp":"2025-02-23T23:29:40.379186Z"}`,
		`{"AssetType":"spot","CurrencyPair":"XBT/USD","Side":"SELL","Price":95940.90000,"Amount":0.00011069,"Timestamp":"2025-02-24T02:01:12.853682Z"}`,
	}
	require.Len(t, e.Websocket.DataHandler.C, len(expJSON), "Must see correct number of trades")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case trade.Data:
			i := 1 - len(e.Websocket.DataHandler.C)
			exp := trade.Data{Exchange: e.Name, CurrencyPair: spotTestPair}
			require.NoErrorf(t, json.Unmarshal([]byte(expJSON[i]), &exp), "Must not error unmarshalling json %d: %s", i, expJSON[i])
			require.Equalf(t, exp, v, "Trade [%d] must be correct", i)
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}
}

func TestWsOpenOrders(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Test instance Setup must not error")
	testexch.UpdatePairsOnce(t, e)
	conn := testexch.GetMockConn(t, e, "")
	testexch.FixtureToDataHandler(t, "testdata/wsOpenTrades.json", func(ctx context.Context, b []byte) error { return e.wsHandleData(ctx, conn, b) })
	e.Websocket.DataHandler.Close()
	assert.Len(t, e.Websocket.DataHandler.C, 7, "Should see 7 orders")
	for resp := range e.Websocket.DataHandler.C {
		switch v := resp.Data.(type) {
		case *order.Detail:
			switch len(e.Websocket.DataHandler.C) {
			case 6:
				assert.Equal(t, "OGTT3Y-C6I3P-XRI6HR", v.OrderID, "OrderID")
				assert.Equal(t, order.Limit, v.Type, "order type")
				assert.Equal(t, order.Sell, v.Side, "order side")
				assert.Equal(t, order.Open, v.Status, "order status")
				assert.Equal(t, 34.5, v.Price, "price")
				assert.Equal(t, 10.00345345, v.Amount, "amount")
			case 5:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.Market, v.Type, "order type")
				assert.Equal(t, order.Buy, v.Side, "order side")
				assert.Equal(t, order.Pending, v.Status, "order status")
				assert.Equal(t, 0.0, v.Price, "price")
				assert.Equal(t, 0.0001, v.Amount, "amount")
				assert.Equal(t, time.UnixMicro(1692851641361371).UTC(), v.Date.UTC(), "Date")
			case 4:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.Open, v.Status, "order status")
			case 3:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.UnknownStatus, v.Status, "order status")
				assert.Equal(t, 26425.2, v.AverageExecutedPrice, "AverageExecutedPrice")
				assert.Equal(t, 0.0001, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount") // Not in the message; Testing regression to bad derivation
				assert.Equal(t, 0.00687, v.Fee, "Fee")
			case 2:
				assert.Equal(t, "OKB55A-UEMMN-YUXM2A", v.OrderID, "OrderID")
				assert.Equal(t, order.Closed, v.Status, "order status")
				assert.Equal(t, 0.0001, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 26425.2, v.AverageExecutedPrice, "AverageExecutedPrice")
				assert.Equal(t, 0.00687, v.Fee, "Fee")
				assert.Equal(t, time.UnixMicro(1692851641361447).UTC(), v.LastUpdated.UTC(), "LastUpdated")
			case 1:
				assert.Equal(t, "OGTT3Y-C6I3P-XRI6HR", v.OrderID, "OrderID")
				assert.Equal(t, order.UnknownStatus, v.Status, "order status")
				assert.Equal(t, 10.00345345, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.001, v.Fee, "Fee")
				assert.Equal(t, 34.5, v.AverageExecutedPrice, "AverageExecutedPrice")
			case 0:
				assert.Equal(t, "OGTT3Y-C6I3P-XRI6HR", v.OrderID, "OrderID")
				assert.Equal(t, order.Closed, v.Status, "order status")
				assert.Equal(t, time.UnixMicro(1692675961789052).UTC(), v.LastUpdated.UTC(), "LastUpdated")
				assert.Equal(t, 10.00345345, v.ExecutedAmount, "ExecutedAmount")
				assert.Equal(t, 0.001, v.Fee, "Fee")
				assert.Equal(t, 34.5, v.AverageExecutedPrice, "AverageExecutedPrice")
			}
		case error:
			t.Error(v)
		default:
			t.Errorf("Unexpected type in DataHandler: %T (%s)", v, v)
		}
	}
}

func TestWsOrderbookMax10Depth(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	require.NoError(t, testexch.Setup(e), "Setup Instance must not error")
	pairs := currency.Pairs{
		currency.NewPairWithDelimiter("XDG", "USD", "/"),
		currency.NewPairWithDelimiter("LUNA", "EUR", "/"),
		currency.NewPairWithDelimiter("GST", "EUR", "/"),
	}
	for _, p := range pairs {
		err := e.Websocket.AddSuccessfulSubscriptions(nil, &subscription.Subscription{
			Channel: subscription.OrderbookChannel,
			Pairs:   currency.Pairs{p},
			Asset:   asset.Spot,
			Levels:  10,
		})
		require.NoError(t, err, "AddSuccessfulSubscriptions must not error")
	}
	conn := testexch.GetMockConn(t, e, "")

	for x := range websocketXDGUSDOrderbookUpdates {
		err := e.wsHandleData(t.Context(), conn, []byte(websocketXDGUSDOrderbookUpdates[x]))
		require.NoError(t, err, "wsHandleData must not error")
	}

	for x := range websocketLUNAEUROrderbookUpdates {
		err := e.wsHandleData(t.Context(), conn, []byte(websocketLUNAEUROrderbookUpdates[x]))
		// TODO: Known issue with LUNA pairs and big number float precision
		// storage and checksum calc. Might need to store raw strings as fields
		// in the orderbook.Level struct.
		// Required checksum: 7465000014735432016076747100005084881400000007476000097005027047670474990000293338023886300750000004333333333333375020000152914844934167507000014652990542161752500007370728572000475400000670061645671407546000098022663603417745900007102987806720745800001593557686404000745200003375861179634000743500003156650585902777434000030172726079999999743200006461149653837000743100001042285966000000074300000403660461058200074200000369021657320475740500001674242117790510
		if x != len(websocketLUNAEUROrderbookUpdates)-1 {
			require.NoError(t, err, "wsHandleData must not error")
		}
	}

	// This has less than 10 bids and still needs a checksum calc.
	for x := range websocketGSTEUROrderbookUpdates {
		err := e.wsHandleData(t.Context(), conn, []byte(websocketGSTEUROrderbookUpdates[x]))
		require.NoError(t, err, "wsHandleData must not error")
	}
}

func TestWebsocketAuthToken(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	e.setWebsocketAuthToken("meep")
	const n = 69
	var wg sync.WaitGroup
	wg.Add(2 * n)

	start := make(chan struct{})
	for range n {
		go func() {
			defer wg.Done()
			<-start
			e.setWebsocketAuthToken("69420")
		}()
	}
	for range n {
		go func() {
			defer wg.Done()
			<-start
			e.websocketAuthToken()
		}()
	}
	close(start)
	wg.Wait()
	assert.Equal(t, "69420", e.websocketAuthToken(), "websocketAuthToken should return correctly after concurrent reads and writes")
}

func TestSetWebsocketAuthToken(t *testing.T) {
	t.Parallel()
	e := new(Exchange)
	e.setWebsocketAuthToken("69420")
	assert.Equal(t, "69420", e.websocketAuthToken())
}

func mockWsHandler(tb testing.TB, h mockws.WsMockFunc) http.HandlerFunc {
	tb.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if mockWsTokenHandler(tb, w, r) {
			return
		}
		mockws.WsMockUpgrader(tb, w, r, h)
	}
}

func mockWsTokenHandler(tb testing.TB, w http.ResponseWriter, r *http.Request) bool {
	tb.Helper()
	if r.URL.Path != "/0/private/GetWebSocketsToken" {
		return false
	}
	_, err := w.Write([]byte(`{"result":{"token":"mockAuth"}}`))
	assert.NoError(tb, err, "Write should not error")
	return true
}
