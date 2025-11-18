package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	mockws "github.com/thrasher-corp/gocryptotrader/internal/testing/websocket"
)

// TestSubscribe logic test
func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	ws := NewManager()
	assert.NoError(t, ws.Setup(newDefaultSetup()), "WS Setup should not error")

	ws.Subscriber = currySimpleSub(ws)
	ws.Unsubscriber = currySimpleUnsub(ws)

	subs, err := ws.GenerateSubs()
	require.NoError(t, err, "Generating test subscriptions must not error")
	assert.ErrorIs(t, new(Manager).UnsubscribeChannels(nil, subs), common.ErrNilPointer, "Should error when unsubscribing with nil unsubscribe function")
	assert.NoError(t, ws.UnsubscribeChannels(nil, nil), "Unsubscribing from nil should not error")
	assert.ErrorIs(t, ws.UnsubscribeChannels(nil, subs), subscription.ErrNotFound, "Unsubscribing should error when not subscribed")
	assert.Nil(t, ws.GetSubscription(42), "GetSubscription on empty internal map should return")
	assert.NoError(t, ws.SubscribeToChannels(nil, subs), "Basic Subscribing should not error")
	assert.Len(t, ws.GetSubscriptions(), 4, "Should have 4 subscriptions")
	bySub := ws.GetSubscription(subscription.Subscription{Channel: "TestSub"})
	if assert.NotNil(t, bySub, "GetSubscription by subscription should find a channel") {
		assert.Equal(t, "TestSub", bySub.Channel, "GetSubscription by default key should return a pointer a copy of the right channel")
		assert.Same(t, bySub, subs[0], "GetSubscription returns the same pointer")
	}
	if assert.NotNil(t, ws.GetSubscription("purple"), "GetSubscription by string key should find a channel") {
		assert.Equal(t, "TestSub2", ws.GetSubscription("purple").Channel, "GetSubscription by string key should return a pointer a copy of the right channel")
	}
	if assert.NotNil(t, ws.GetSubscription(testSubKey{"mauve"}), "GetSubscription by type key should find a channel") {
		assert.Equal(t, "TestSub3", ws.GetSubscription(testSubKey{"mauve"}).Channel, "GetSubscription by type key should return a pointer a copy of the right channel")
	}
	if assert.NotNil(t, ws.GetSubscription(42), "GetSubscription by int key should find a channel") {
		assert.Equal(t, "TestSub4", ws.GetSubscription(42).Channel, "GetSubscription by int key should return a pointer a copy of the right channel")
	}
	assert.Nil(t, ws.GetSubscription(nil), "GetSubscription by nil should return nil")
	assert.Nil(t, ws.GetSubscription(45), "GetSubscription by invalid key should return nil")
	assert.ErrorIs(t, ws.SubscribeToChannels(nil, subs), subscription.ErrDuplicate, "Subscribe should error when already subscribed")
	assert.NoError(t, ws.SubscribeToChannels(nil, nil), "Subscribe to an nil List should not error")
	assert.NoError(t, ws.UnsubscribeChannels(nil, subs), "Unsubscribing should not error")

	ws.Subscriber = func(subscription.List) error { return errDastardlyReason }
	assert.ErrorIs(t, ws.SubscribeToChannels(nil, subs), errDastardlyReason, "Should error correctly when error returned from Subscriber")

	err = ws.SubscribeToChannels(nil, subscription.List{nil})
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly when list contains a nil subscription")

	multi := NewManager()
	set := newDefaultSetup()
	set.UseMultiConnectionManagement = true
	assert.NoError(t, multi.Setup(set))

	amazingCandidate := &ConnectionSetup{
		URL:                   "AMAZING",
		Connector:             func(context.Context, Connection) error { return nil },
		GenerateSubscriptions: ws.GenerateSubs,
		Subscriber: func(ctx context.Context, c Connection, s subscription.List) error {
			return currySimpleSubConn(multi)(ctx, c, s)
		},
		Unsubscriber: func(ctx context.Context, c Connection, s subscription.List) error {
			return currySimpleUnsubConn(multi)(ctx, c, s)
		},
		Handler: func(context.Context, Connection, []byte) error { return nil },
	}
	require.NoError(t, multi.SetupNewConnection(amazingCandidate))

	amazingConn := multi.getConnectionFromSetup(amazingCandidate)
	multi.connections = map[Connection]*connectionWrapper{
		amazingConn: multi.connectionManager[0],
	}

	subs, err = amazingCandidate.GenerateSubscriptions()
	require.NoError(t, err, "Generating test subscriptions must not error")
	assert.ErrorIs(t, new(Manager).UnsubscribeChannels(nil, subs), common.ErrNilPointer, "Should error when unsubscribing with nil unsubscribe function")
	assert.ErrorIs(t, new(Manager).UnsubscribeChannels(amazingConn, subs), common.ErrNilPointer, "Should error when unsubscribing with nil unsubscribe function")
	assert.NoError(t, multi.UnsubscribeChannels(amazingConn, nil), "Unsubscribing from nil should not error")
	assert.ErrorIs(t, multi.UnsubscribeChannels(amazingConn, subs), subscription.ErrNotFound, "Unsubscribing should error when not subscribed")
	assert.Nil(t, multi.GetSubscription(42), "GetSubscription on empty internal map should return")

	assert.ErrorIs(t, multi.SubscribeToChannels(nil, subs), common.ErrNilPointer, "If no connection is set, Subscribe should error")

	assert.NoError(t, multi.SubscribeToChannels(amazingConn, subs), "Basic Subscribing should not error")
	assert.Len(t, multi.GetSubscriptions(), 4, "Should have 4 subscriptions")
	bySub = multi.GetSubscription(subscription.Subscription{Channel: "TestSub"})
	if assert.NotNil(t, bySub, "GetSubscription by subscription should find a channel") {
		assert.Equal(t, "TestSub", bySub.Channel, "GetSubscription by default key should return a pointer a copy of the right channel")
		assert.Same(t, bySub, subs[0], "GetSubscription returns the same pointer")
	}
	if assert.NotNil(t, multi.GetSubscription("purple"), "GetSubscription by string key should find a channel") {
		assert.Equal(t, "TestSub2", multi.GetSubscription("purple").Channel, "GetSubscription by string key should return a pointer a copy of the right channel")
	}
	if assert.NotNil(t, multi.GetSubscription(testSubKey{"mauve"}), "GetSubscription by type key should find a channel") {
		assert.Equal(t, "TestSub3", multi.GetSubscription(testSubKey{"mauve"}).Channel, "GetSubscription by type key should return a pointer a copy of the right channel")
	}
	if assert.NotNil(t, multi.GetSubscription(42), "GetSubscription by int key should find a channel") {
		assert.Equal(t, "TestSub4", multi.GetSubscription(42).Channel, "GetSubscription by int key should return a pointer a copy of the right channel")
	}
	assert.Nil(t, multi.GetSubscription(nil), "GetSubscription by nil should return nil")
	assert.Nil(t, multi.GetSubscription(45), "GetSubscription by invalid key should return nil")
	assert.ErrorIs(t, multi.SubscribeToChannels(amazingConn, subs), subscription.ErrDuplicate, "Subscribe should error when already subscribed")
	assert.NoError(t, multi.SubscribeToChannels(amazingConn, nil), "Subscribe to an nil List should not error")
	assert.NoError(t, multi.UnsubscribeChannels(amazingConn, subs), "Unsubscribing should not error")

	amazingCandidate.Subscriber = func(context.Context, Connection, subscription.List) error { return errDastardlyReason }
	assert.ErrorIs(t, multi.SubscribeToChannels(amazingConn, subs), errDastardlyReason, "Should error correctly when error returned from Subscriber")

	err = multi.SubscribeToChannels(amazingConn, subscription.List{nil})
	assert.ErrorIs(t, err, common.ErrNilPointer, "Should error correctly when list contains a nil subscription")
}

// TestResubscribe tests Resubscribing to existing subscriptions
func TestResubscribe(t *testing.T) {
	t.Parallel()
	ws := NewManager()

	wackedOutSetup := newDefaultSetup()
	wackedOutSetup.MaxWebsocketSubscriptionsPerConnection = -1
	err := ws.Setup(wackedOutSetup)
	assert.ErrorIs(t, err, errInvalidMaxSubscriptions, "Invalid MaxWebsocketSubscriptionsPerConnection should error")

	err = ws.Setup(newDefaultSetup())
	assert.NoError(t, err, "WS Setup should not error")

	ws.Subscriber = currySimpleSub(ws)
	ws.Unsubscriber = currySimpleUnsub(ws)

	channel := subscription.List{{Channel: "resubTest"}}

	assert.ErrorIs(t, ws.ResubscribeToChannel(nil, channel[0]), subscription.ErrNotFound, "Resubscribe should error when channel isn't subscribed yet")
	assert.NoError(t, ws.SubscribeToChannels(nil, channel), "Subscribe should not error")
	assert.NoError(t, ws.ResubscribeToChannel(nil, channel[0]), "Resubscribe should not error now the channel is subscribed")
}

// TestSubscriptions tests adding, getting and removing subscriptions
func TestSubscriptions(t *testing.T) {
	t.Parallel()
	w := new(Manager) // Do not use NewManager; We want to exercise w.subs == nil
	assert.ErrorIs(t, (*Manager)(nil).AddSubscriptions(nil), common.ErrNilPointer, "Should error correctly when nil websocket")
	s := &subscription.Subscription{Key: 42, Channel: subscription.TickerChannel}
	require.NoError(t, w.AddSubscriptions(nil, s), "Adding first subscription must not error")
	assert.Same(t, s, w.GetSubscription(42), "Get Subscription should retrieve the same subscription")
	assert.ErrorIs(t, w.AddSubscriptions(nil, s), subscription.ErrDuplicate, "Adding same subscription should return error")
	assert.Equal(t, subscription.SubscribingState, s.State(), "Should set state to Subscribing")

	err := w.RemoveSubscriptions(nil, s)
	require.NoError(t, err, "RemoveSubscriptions must not error")
	assert.Nil(t, w.GetSubscription(42), "Remove should have removed the sub")
	assert.Equal(t, subscription.UnsubscribedState, s.State(), "Should set state to Unsubscribed")

	require.NoError(t, s.SetState(subscription.ResubscribingState), "SetState must not error")
	require.NoError(t, w.AddSubscriptions(nil, s), "Adding first subscription must not error")
	assert.Equal(t, subscription.ResubscribingState, s.State(), "Should not change resubscribing state")
}

// TestSuccessfulSubscriptions tests adding, getting and removing subscriptions
func TestSuccessfulSubscriptions(t *testing.T) {
	t.Parallel()
	w := new(Manager) // Do not use NewManager; We want to exercise w.subs == nil
	assert.ErrorIs(t, (*Manager)(nil).AddSuccessfulSubscriptions(nil, nil), common.ErrNilPointer, "Should error correctly when nil websocket")
	c := &subscription.Subscription{Key: 42, Channel: subscription.TickerChannel}
	require.NoError(t, w.AddSuccessfulSubscriptions(nil, c), "Adding first subscription must not error")
	assert.Same(t, c, w.GetSubscription(42), "Get Subscription should retrieve the same subscription")
	assert.ErrorIs(t, w.AddSuccessfulSubscriptions(nil, c), subscription.ErrInStateAlready, "Adding subscription in same state should return error")
	require.NoError(t, c.SetState(subscription.SubscribingState), "SetState must not error")
	assert.ErrorIs(t, w.AddSuccessfulSubscriptions(nil, c), subscription.ErrDuplicate, "Adding same subscription should return error")

	err := w.RemoveSubscriptions(nil, c)
	require.NoError(t, err, "RemoveSubscriptions must not error")
	assert.Nil(t, w.GetSubscription(42), "Remove should have removed the sub")
	assert.ErrorIs(t, w.RemoveSubscriptions(nil, c), subscription.ErrNotFound, "Should error correctly when not found")
	assert.ErrorIs(t, (*Manager)(nil).RemoveSubscriptions(nil, nil), common.ErrNilPointer, "Should error correctly when nil websocket")
	w.subscriptions = nil
	assert.ErrorIs(t, w.RemoveSubscriptions(nil, c), common.ErrNilPointer, "Should error correctly when nil websocket")
}

// TestGetSubscription logic test
func TestGetSubscription(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (*Manager).GetSubscription(nil, "imaginary"), "GetSubscription on a nil Websocket should return nil")
	assert.Nil(t, (&Manager{}).GetSubscription("empty"), "GetSubscription on a Websocket with no sub store should return nil")
	w := NewManager()
	assert.Nil(t, w.GetSubscription(nil), "GetSubscription with a nil key should return nil")
	s := &subscription.Subscription{Key: 42, Channel: "hello3"}
	require.NoError(t, w.AddSubscriptions(nil, s), "AddSubscriptions must not error")
	assert.Same(t, s, w.GetSubscription(42), "GetSubscription should delegate to the store")
}

// TestGetSubscriptions logic test
func TestGetSubscriptions(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (*Manager).GetSubscriptions(nil), "GetSubscription on a nil Websocket should return nil")
	assert.Nil(t, (&Manager{}).GetSubscriptions(), "GetSubscription on a Websocket with no sub store should return nil")
	w := NewManager()
	s := subscription.List{
		{Key: 42, Channel: "hello3"},
		{Key: 45, Channel: "hello4"},
	}
	err := w.AddSubscriptions(nil, s...)
	require.NoError(t, err, "AddSubscriptions must not error")
	assert.ElementsMatch(t, s, w.GetSubscriptions(), "GetSubscriptions should return the correct channel details")
}

func TestCheckSubscriptions(t *testing.T) {
	t.Parallel()
	ws := Manager{}
	err := ws.checkSubscriptions(nil, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "checkSubscriptions should error correctly on nil w.subscriptions")
	assert.ErrorContains(t, err, "Websocket.subscriptions", "checkSubscriptions should error giving context correctly on nil w.subscriptions")

	ws.subscriptions = subscription.NewStore()
	err = ws.checkSubscriptions(nil, nil)
	assert.NoError(t, err, "checkSubscriptions should not error on a nil list")

	ws.MaxSubscriptionsPerConnection = 1

	err = ws.checkSubscriptions(nil, subscription.List{{}})
	assert.NoError(t, err, "checkSubscriptions should not error when subscriptions is empty")

	ws.subscriptions = subscription.NewStore()
	err = ws.checkSubscriptions(nil, subscription.List{{}, {}})
	assert.ErrorIs(t, err, errSubscriptionsExceedsLimit, "checkSubscriptions should error correctly")

	ws.MaxSubscriptionsPerConnection = 2

	ws.subscriptions = subscription.NewStore()
	err = ws.subscriptions.Add(&subscription.Subscription{Key: 42, Channel: "test"})
	require.NoError(t, err, "Add subscription must not error")
	err = ws.checkSubscriptions(nil, subscription.List{{Key: 42, Channel: "test"}})
	assert.ErrorIs(t, err, subscription.ErrDuplicate, "checkSubscriptions should error correctly")

	err = ws.checkSubscriptions(nil, subscription.List{{}})
	assert.NoError(t, err, "checkSubscriptions should not error")
}

func TestUpdateChannelSubscriptions(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	store := subscription.NewStore()
	err := ws.updateChannelSubscriptions(nil, store, subscription.List{{Channel: "test"}})
	require.ErrorIs(t, err, common.ErrNilPointer)
	require.Zero(t, store.Len())

	ws.Subscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := store.Add(sub); err != nil {
				return err
			}
		}
		return nil
	}

	ws.subscriptions = store
	err = ws.updateChannelSubscriptions(nil, store, subscription.List{{Channel: "test"}})
	require.NoError(t, err)
	require.Equal(t, 1, store.Len())

	err = ws.updateChannelSubscriptions(nil, store, subscription.List{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	ws.Unsubscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := store.Remove(sub); err != nil {
				return err
			}
		}
		return nil
	}

	err = ws.updateChannelSubscriptions(nil, store, subscription.List{})
	require.NoError(t, err)
	require.Zero(t, store.Len())
}

func currySimpleSub(w *Manager) func(subscription.List) error {
	return func(subs subscription.List) error {
		return w.AddSuccessfulSubscriptions(nil, subs...)
	}
}

func currySimpleSubConn(w *Manager) func(context.Context, Connection, subscription.List) error {
	return func(_ context.Context, conn Connection, subs subscription.List) error {
		return w.AddSuccessfulSubscriptions(conn, subs...)
	}
}

func currySimpleUnsub(w *Manager) func(subscription.List) error {
	return func(unsubs subscription.List) error {
		return w.RemoveSubscriptions(nil, unsubs...)
	}
}

func currySimpleUnsubConn(w *Manager) func(context.Context, Connection, subscription.List) error {
	return func(_ context.Context, conn Connection, unsubs subscription.List) error {
		return w.RemoveSubscriptions(conn, unsubs...)
	}
}

func TestFlushChannels(t *testing.T) {
	t.Parallel()
	// Enabled pairs/setup system

	dodgyWs := Manager{}
	err := dodgyWs.FlushChannels()
	assert.ErrorIs(t, err, ErrWebsocketNotEnabled, "FlushChannels should error correctly")

	dodgyWs.setEnabled(true)
	err = dodgyWs.FlushChannels()
	assert.ErrorIs(t, err, ErrNotConnected, "FlushChannels should error correctly")

	newgen := GenSubs{EnabledPairs: []currency.Pair{
		currency.NewPair(currency.BTC, currency.AUD),
		currency.NewBTCUSDT(),
	}}

	w := NewManager()
	w.exchangeName = "test"
	w.connector = noopConnect
	w.Subscriber = newgen.SUBME
	w.Unsubscriber = newgen.UNSUBME
	// Added for when we utilise connect() in FlushChannels() so the traffic monitor doesn't time out and turn this to an unconnected state
	w.trafficTimeout = time.Second * 30

	w.setEnabled(true)
	w.setState(connectedState)

	// Allow subscribe and unsubscribe feature set, without these the tests will call shutdown and connect.
	w.features.Subscribe = true
	w.features.Unsubscribe = true

	// Disable pair and flush system
	newgen.EnabledPairs = []currency.Pair{currency.NewPair(currency.BTC, currency.AUD)}
	w.GenerateSubs = func() (subscription.List, error) { return subscription.List{{Channel: "test"}}, nil }

	require.ErrorIs(t, w.FlushChannels(), ErrSubscriptionsNotAdded, "FlushChannels must error correctly on no subscriptions added")

	w.Subscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := w.subscriptions.Add(sub); err != nil {
				return err
			}
		}
		return nil
	}

	require.NoError(t, w.FlushChannels(), "FlushChannels must not error")

	w.GenerateSubs = func() (subscription.List, error) { return nil, errDastardlyReason } // error on generateSubs
	err = w.FlushChannels()                                                               // error on full subscribeToChannels
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should error correctly on GenerateSubs")

	w.GenerateSubs = func() (subscription.List, error) { return nil, nil } // No subs to sub

	require.ErrorIs(t, w.FlushChannels(), ErrSubscriptionsNotRemoved)

	w.Unsubscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := w.subscriptions.Remove(sub); err != nil {
				return err
			}
		}
		return nil
	}
	assert.NoError(t, w.FlushChannels(), "FlushChannels should not error")

	w.GenerateSubs = newgen.generateSubs
	subs, err := w.GenerateSubs()
	require.NoError(t, err, "GenerateSubs must not error")
	require.NoError(t, w.AddSubscriptions(nil, subs...), "AddSubscriptions must not error")
	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")

	w.GenerateSubs = newgen.generateSubs
	w.subscriptions = subscription.NewStore()
	err = w.subscriptions.Add(&subscription.Subscription{
		Key:     41,
		Channel: "match channel",
		Pairs:   currency.Pairs{currency.NewPair(currency.BTC, currency.AUD)},
	})
	require.NoError(t, err, "AddSubscription must not error")
	err = w.subscriptions.Add(&subscription.Subscription{
		Key:     42,
		Channel: "unsub channel",
		Pairs:   currency.Pairs{currency.NewPair(currency.THETA, currency.USDT)},
	})
	require.NoError(t, err, "AddSubscription must not error")

	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")

	w.setState(connectedState)
	err = w.FlushChannels()
	assert.NoError(t, err, "FlushChannels should not error")

	// Multi connection management
	w.useMultiConnectionManagement = true
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()

	w.subscriptions = subscription.NewStore()

	amazingCandidate := &ConnectionSetup{
		URL: "ws" + mock.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, gws.DefaultDialer, nil)
		},
		GenerateSubscriptions: newgen.generateSubs,
		Subscriber:            func(context.Context, Connection, subscription.List) error { return nil },
		Unsubscriber:          func(context.Context, Connection, subscription.List) error { return nil },
		Handler:               func(context.Context, Connection, []byte) error { return nil },
	}
	require.NoError(t, w.SetupNewConnection(amazingCandidate))
	require.ErrorIs(t, w.FlushChannels(), ErrSubscriptionsNotAdded, "Must error when no subscriptions are added to the subscription store")

	w.connectionManager[0].setup.Subscriber = func(ctx context.Context, c Connection, s subscription.List) error {
		return currySimpleSubConn(w)(ctx, c, s)
	}
	require.NoError(t, w.FlushChannels(), "FlushChannels must not error")

	// Forces full connection cycle (shutdown, connect, subscribe). This will also start monitoring routines.
	w.features.Subscribe = false
	require.NoError(t, w.FlushChannels(), "FlushChannels must not error")

	// Unsubscribe what's already subscribed. No subscriptions left over, which then forces the shutdown and removal
	// of the connection from management.
	w.features.Subscribe = true
	w.connectionManager[0].setup.GenerateSubscriptions = func() (subscription.List, error) { return nil, nil }
	require.ErrorIs(t, w.FlushChannels(), ErrSubscriptionsNotRemoved, "Must error when no subscriptions are removed from subscription store")

	w.connectionManager[0].setup.Unsubscriber = func(ctx context.Context, c Connection, s subscription.List) error {
		return currySimpleUnsubConn(w)(ctx, c, s)
	}
	require.NoError(t, w.FlushChannels(), "FlushChannels must not error")
}

func TestScaleConnectionsToSubscriptions(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	err := ws.scaleConnectionsToSubscriptions(t.Context(), nil, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "must error with nil connectionWrapper")
	ws.MaxSubscriptionsPerConnection = 2

	wrapper := &connectionWrapper{
		setup: &ConnectionSetup{
			Connector: func(ctx context.Context, c Connection) error {
				return c.Dial(ctx, gws.DefaultDialer, nil)
			},
			Subscriber: func(ctx context.Context, c Connection, s subscription.List) error {
				return currySimpleSubConn(ws)(ctx, c, s)
			},
			Unsubscriber: func(ctx context.Context, c Connection, s subscription.List) error {
				return currySimpleUnsubConn(ws)(ctx, c, s)
			},
			Handler: func(context.Context, Connection, []byte) error { return nil },
		},
		connectionSubs: make(map[Connection]*subscription.Store),
		subscriptions:  subscription.NewStore(),
	}

	err = ws.scaleConnectionsToSubscriptions(t.Context(), wrapper, nil)
	require.NoError(t, err, "must not error with no subscriptions")

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	defer mock.Close()
	wrapper.setup.URL = "ws" + mock.URL[len("http"):] + "/ws"

	exp := subscription.List{
		{Channel: "test"},
		{Channel: "test2"},
		{Channel: "test3"},
	}

	err = ws.scaleConnectionsToSubscriptions(t.Context(), wrapper, exp)
	require.NoError(t, err, "must not error when adding subscriptions")
	require.Len(t, wrapper.connectionSubs, 2, "must have two connections when max subs per connection is 2")
	require.Len(t, wrapper.subscriptions.Contained(exp), 3, "subscriptions must match global store")
	var specificConnSubs subscription.List
	for _, store := range wrapper.connectionSubs {
		specificConnSubs = append(specificConnSubs, store.List()...)
	}
	require.Len(t, wrapper.subscriptions.Contained(specificConnSubs), 3, "connection subscriptions must match global store")

	exp = subscription.List{
		{Channel: "test4"},
		{Channel: "test5"},
		{Channel: "test6"},
		{Channel: "test7"},
		{Channel: "test8"},
	}

	err = ws.scaleConnectionsToSubscriptions(t.Context(), wrapper, exp)
	require.NoError(t, err, "must not error when scaling subscriptions to connections")
	require.Len(t, wrapper.connectionSubs, 3, "must have three connections when max subs per connection is 2")
	require.Len(t, wrapper.subscriptions.Contained(exp), 5, "subscriptions must match global store")
	specificConnSubs = nil
	for _, store := range wrapper.connectionSubs {
		specificConnSubs = append(specificConnSubs, store.List()...)
	}
	require.Len(t, wrapper.subscriptions.Contained(specificConnSubs), 5, "connection subscriptions must match global store")

	err = ws.scaleConnectionsToSubscriptions(t.Context(), wrapper, nil)
	require.NoError(t, err, "must not error when scaling subscriptions to connections with no new subscriptions")
	require.Len(t, wrapper.connectionSubs, 0, "must drop all connections when no subscriptions are present")
	require.Len(t, wrapper.subscriptions.List(), 0, "must drop all subscriptions when no subscriptions are present")
}
