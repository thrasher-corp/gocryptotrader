package websocket

import (
	"context"
	"errors"
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

func TestSubscribeUnsubscribe(t *testing.T) {
	t.Parallel()
	ws := NewManager()
	assert.NoError(t, ws.Setup(newDefaultSetup()), "WS Setup should not error")

	ws.Subscriber = currySimpleSub(ws)
	ws.Unsubscriber = currySimpleUnsub(ws)

	subs, err := ws.GenerateSubs()
	require.NoError(t, err, "Generating test subscriptions must not error")
	assert.ErrorIs(t, new(Manager).UnsubscribeChannels(t.Context(), nil, subs), common.ErrNilPointer, "Should error when unsubscribing with nil unsubscribe function")
	assert.NoError(t, ws.UnsubscribeChannels(t.Context(), nil, nil), "Unsubscribing from nil should not error")
	assert.ErrorIs(t, ws.UnsubscribeChannels(t.Context(), nil, subs), subscription.ErrNotFound, "Unsubscribing should error when not subscribed")
	assert.Nil(t, ws.GetSubscription(42), "GetSubscription on empty internal map should return")
	assert.NoError(t, ws.SubscribeToChannels(t.Context(), nil, subs), "Basic Subscribing should not error")
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
	assert.ErrorIs(t, ws.SubscribeToChannels(t.Context(), nil, subs), subscription.ErrDuplicate, "Subscribe should error when already subscribed")
	assert.NoError(t, ws.SubscribeToChannels(t.Context(), nil, nil), "Subscribe to an nil List should not error")
	assert.NoError(t, ws.UnsubscribeChannels(t.Context(), nil, subs), "Unsubscribing should not error")

	ws.Subscriber = func(subscription.List) error { return errDastardlyReason }
	assert.ErrorIs(t, ws.SubscribeToChannels(t.Context(), nil, subs), errDastardlyReason, "Should error correctly when error returned from Subscriber")

	err = ws.SubscribeToChannels(t.Context(), nil, subscription.List{nil})
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

	amazingConn := multi.createConnectionFromSetup(amazingCandidate)
	multi.connections = map[Connection]*websocket{
		amazingConn: multi.connectionManager[0],
	}

	multiEmpty := NewManager()
	multiEmpty.useMultiConnectionManagement = true

	subs, err = amazingCandidate.GenerateSubscriptions()
	require.NoError(t, err, "Generating test subscriptions must not error")
	assert.ErrorIs(t, new(Manager).UnsubscribeChannels(t.Context(), nil, subs), common.ErrNilPointer, "Should error when unsubscribing with nil unsubscribe function")
	assert.ErrorIs(t, new(Manager).UnsubscribeChannels(t.Context(), amazingConn, subs), common.ErrNilPointer, "Should error when unsubscribing with nil unsubscribe function")
	assert.NoError(t, multi.UnsubscribeChannels(t.Context(), amazingConn, nil), "Unsubscribing from nil should not error")
	assert.ErrorIs(t, multi.UnsubscribeChannels(t.Context(), amazingConn, subs), subscription.ErrNotFound, "Unsubscribing should error when not subscribed")
	assert.Nil(t, multi.GetSubscription(42), "GetSubscription on empty internal map should return")

	assert.ErrorIs(t, multi.SubscribeToChannels(t.Context(), nil, subs), common.ErrNilPointer, "If no connection is set, Subscribe should error")
	assert.ErrorIs(t, multi.SubscribeToChannels(t.Context(), amazingConn, subs), common.ErrNilPointer, "Basic Subscribing should error when connection is not present in ws map")

	multi.connectionManager[0].connections = []Connection{amazingConn}
	assert.NoError(t, multi.SubscribeToChannels(t.Context(), amazingConn, subs), "Basic Subscribing should not error")
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
	assert.ErrorIs(t, multi.SubscribeToChannels(t.Context(), amazingConn, subs), subscription.ErrDuplicate, "Subscribe should error when already subscribed")
	assert.NoError(t, multi.SubscribeToChannels(t.Context(), amazingConn, nil), "Subscribe to an nil List should not error")
	assert.NoError(t, multi.UnsubscribeChannels(t.Context(), amazingConn, subs), "Unsubscribing should not error")

	amazingCandidate.Subscriber = func(context.Context, Connection, subscription.List) error { return errDastardlyReason }
	assert.ErrorIs(t, multi.SubscribeToChannels(t.Context(), amazingConn, subs), errDastardlyReason, "Should error correctly when error returned from Subscriber")

	err = multi.SubscribeToChannels(t.Context(), amazingConn, subscription.List{nil})
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

	assert.ErrorIs(t, ws.ResubscribeToChannel(t.Context(), nil, channel[0]), subscription.ErrNotFound, "Resubscribe should error when channel isn't subscribed yet")
	assert.NoError(t, ws.SubscribeToChannels(t.Context(), nil, channel), "Subscribe should not error")
	assert.NoError(t, ws.ResubscribeToChannel(t.Context(), nil, channel[0]), "Resubscribe should not error now the channel is subscribed")
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

	ws.subscriptions = subscription.NewStore()
	conn := &connection{}
	ws.connections = map[Connection]*websocket{
		conn: {},
	}
	err = ws.checkSubscriptions(conn, subscription.List{{}})
	assert.ErrorContains(t, err, "nil pointer: Websocket.subscriptions", "nil store for a specific connection should error correctly")
}

func TestUpdateChannelSubscriptions(t *testing.T) {
	t.Parallel()

	ws := NewManager()
	store := subscription.NewStore()
	err := ws.updateChannelSubscriptions(t.Context(), store, subscription.List{{Channel: "test"}})
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
	err = ws.updateChannelSubscriptions(t.Context(), store, subscription.List{{Channel: "test"}})
	require.NoError(t, err)
	require.Equal(t, 1, store.Len())

	err = ws.updateChannelSubscriptions(t.Context(), store, subscription.List{})
	require.ErrorIs(t, err, common.ErrNilPointer)

	ws.Unsubscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := store.Remove(sub); err != nil {
				return err
			}
		}
		return nil
	}

	err = ws.updateChannelSubscriptions(t.Context(), store, subscription.List{})
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
	err := dodgyWs.FlushChannels(t.Context())
	assert.ErrorIs(t, err, ErrWebsocketNotEnabled, "FlushChannels should error correctly")

	dodgyWs.setEnabled(true)
	err = dodgyWs.FlushChannels(t.Context())
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

	require.ErrorIs(t, w.FlushChannels(t.Context()), ErrSubscriptionsNotAdded, "FlushChannels must error correctly on no subscriptions added")

	w.Subscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := w.subscriptions.Add(sub); err != nil {
				return err
			}
		}
		return nil
	}

	require.NoError(t, w.FlushChannels(t.Context()), "FlushChannels must not error")

	w.GenerateSubs = func() (subscription.List, error) { return nil, errDastardlyReason } // error on generateSubs
	err = w.FlushChannels(t.Context())                                                    // error on full subscribeToChannels
	assert.ErrorIs(t, err, errDastardlyReason, "FlushChannels should error correctly on GenerateSubs")

	w.GenerateSubs = func() (subscription.List, error) { return nil, nil } // No subs to sub

	require.ErrorIs(t, w.FlushChannels(t.Context()), ErrSubscriptionsNotRemoved)

	w.Unsubscriber = func(subs subscription.List) error {
		for _, sub := range subs {
			if err := w.subscriptions.Remove(sub); err != nil {
				return err
			}
		}
		return nil
	}
	assert.NoError(t, w.FlushChannels(t.Context()), "FlushChannels should not error")

	w.GenerateSubs = newgen.generateSubs
	subs, err := w.GenerateSubs()
	require.NoError(t, err, "GenerateSubs must not error")
	require.NoError(t, w.AddSubscriptions(nil, subs...), "AddSubscriptions must not error")
	err = w.FlushChannels(t.Context())
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

	err = w.FlushChannels(t.Context())
	assert.NoError(t, err, "FlushChannels should not error")

	w.setState(connectedState)
	err = w.FlushChannels(t.Context())
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
	require.ErrorIs(t, w.FlushChannels(t.Context()), ErrSubscriptionsNotAdded, "Must error when no subscriptions are added to the subscription store")

	w.connectionManager[0].setup.Subscriber = func(ctx context.Context, c Connection, s subscription.List) error {
		return currySimpleSubConn(w)(ctx, c, s)
	}
	require.NoError(t, w.FlushChannels(t.Context()), "FlushChannels must not error")

	// Forces full connection cycle (shutdown, connect, subscribe). This will also start monitoring routines.
	w.features.Subscribe = false
	require.NoError(t, w.FlushChannels(t.Context()), "FlushChannels must not error")
	// Unsubscribe what's already subscribed. No subscriptions left over, which then forces the shutdown and removal
	// of the connection from management.
	w.features.Subscribe = true
	w.connectionManager[0].setup.GenerateSubscriptions = func() (subscription.List, error) { return nil, nil }
	require.ErrorIs(t, w.FlushChannels(t.Context()), ErrSubscriptionsNotRemoved, "Must error when no subscriptions are removed from subscription store")

	w.connectionManager[0].setup.Unsubscriber = func(ctx context.Context, c Connection, s subscription.List) error {
		return currySimpleUnsubConn(w)(ctx, c, s)
	}
	require.NoError(t, w.FlushChannels(t.Context()), "FlushChannels must not error")
}

// fakeConnection is a minimal Connection implementation used in cleanup tests.
type fakeConnection struct {
	Connection
	subscriptions  *subscription.Store
	shutdownCalled bool
}

func (f *fakeConnection) Shutdown() error {
	f.shutdownCalled = true
	return nil
}

func (f *fakeConnection) Subscriptions() *subscription.Store { return f.subscriptions }

func TestScaleConnectionsToSubscriptions(t *testing.T) {
	t.Parallel()

	// Common setup helper
	setup := func(isMultiConn bool) (*Manager, *websocket, *httptest.Server) {
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 2
		m.useMultiConnectionManagement = isMultiConn

		// Mock server for dialing
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))

		ws := &websocket{
			setup: &ConnectionSetup{
				URL: "ws" + srv.URL[len("http"):] + "/ws",
				Connector: func(ctx context.Context, c Connection) error {
					return c.Dial(ctx, gws.DefaultDialer, nil)
				},
				Subscriber: func(_ context.Context, c Connection, s subscription.List) error {
					return m.AddSuccessfulSubscriptions(c, s...)
				},
				Unsubscriber: func(_ context.Context, c Connection, s subscription.List) error {
					return m.RemoveSubscriptions(c, s...)
				},
				Handler: func(context.Context, Connection, []byte) error { return nil },
			},
			subscriptions: subscription.NewStore(),
		}
		return m, ws, srv
	}

	t.Run("Nil ws", func(t *testing.T) {
		t.Parallel()
		m, _, srv := setup(false)
		defer srv.Close()
		err := m.scaleConnectionsToSubscriptions(t.Context(), nil, nil)
		require.ErrorIs(t, err, common.ErrNilPointer)
	})

	t.Run("No Changes", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(false)
		defer srv.Close()
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, nil)
		require.NoError(t, err)
	})

	t.Run("Scale Up (Add Subs)", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(false)
		defer srv.Close()

		subs := subscription.List{{Channel: "A"}, {Channel: "B"}, {Channel: "C"}}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subs)
		require.NoError(t, err)

		assert.Equal(t, 3, ws.subscriptions.Len())
		assert.Len(t, ws.connections, 2) // 2 per conn -> 2 conns
	})

	t.Run("Scale Down (Remove Subs)", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(true)
		defer srv.Close()

		// Add subs first
		subs := subscription.List{{Channel: "A"}, {Channel: "B"}}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subs)
		require.NoError(t, err)
		require.Equal(t, 2, ws.subscriptions.Len())

		// Remove all
		err = m.scaleConnectionsToSubscriptions(t.Context(), ws, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, ws.subscriptions.Len())
		assert.Empty(t, ws.connections)
	})

	t.Run("Unsubscribe Error", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(true)
		defer srv.Close()

		// Add sub first
		sub := subscription.List{{Channel: "A"}}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, sub))

		// Now set error and remove
		ws.setup.Unsubscriber = func(context.Context, Connection, subscription.List) error {
			return errors.New("unsub fail")
		}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, nil)
		require.ErrorContains(t, err, "unsub fail")
	})

	t.Run("Subscribe Error (Existing Connection)", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(false)
		defer srv.Close()

		// Add one sub (capacity 2)
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}}))

		// Set error
		ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
			return errors.New("sub fail")
		}

		// Add another sub (should use existing connection)
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}, {Channel: "B"}})
		require.ErrorContains(t, err, "sub fail")
	})

	t.Run("Subscribe Error (New Connection)", func(t *testing.T) {
		m, ws, srv := setup(false)
		defer srv.Close()

		// Set connector error
		ws.setup.Connector = func(context.Context, Connection) error {
			return errors.New("connect fail")
		}

		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorContains(t, err, "connect fail")
	})

	t.Run("Global Unsubscribe Fallback Success", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(false)
		defer srv.Close()

		s1 := &subscription.Subscription{Channel: "A"}
		s2 := &subscription.Subscription{Channel: "B"}
		require.NoError(t, ws.subscriptions.Add(s1))
		require.NoError(t, ws.subscriptions.Add(s2))
		// empty incoming subscriptions will remove existing subs
		in := subscription.List{}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, in))
		assert.Equal(t, 0, ws.subscriptions.Len())
	})

	t.Run("Missing Subscriptions After Subscribe", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(false)
		defer srv.Close()

		s1 := &subscription.Subscription{Channel: "A"}
		s2 := &subscription.Subscription{Channel: "B"}
		require.NoError(t, ws.subscriptions.Add(s1))

		in := subscription.List{s1, s2}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, in)
		require.NoError(t, err)

		// After scaling, both subs should be present in the store
		assert.NotNil(t, ws.subscriptions.Get(s1))
		assert.NotNil(t, ws.subscriptions.Get(s2))
	})

	t.Run("Multi-batch ConnectAndSubscribe Success", func(t *testing.T) {
		t.Parallel()
		m, ws, srv := setup(false)
		defer srv.Close()

		m.MaxSubscriptionsPerConnection = 2

		in := subscription.List{{Channel: "A"}, {Channel: "B"}, {Channel: "C"}, {Channel: "D"}, {Channel: "E"}}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, in))

		// With max 2 subs per connection and 5 total, we expect 3 connections
		assert.Len(t, ws.connections, 3)
	})

	t.Run("Cleanup Removes Empty Connections", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 2
		m.connections = make(map[Connection]*websocket)

		ws := &websocket{
			setup:         &ConnectionSetup{},
			subscriptions: subscription.NewStore(),
		}

		// One connection with zero subs, one with non-zero
		emptyConn := &fakeConnection{subscriptions: subscription.NewStore()}
		activeConn := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, activeConn.subscriptions.Add(&subscription.Subscription{Channel: "A"}))

		ws.connections = []Connection{emptyConn, activeConn}

		m.connections[emptyConn] = ws
		m.connections[activeConn] = ws

		// No incoming/sub changes; trigger only cleanup logic
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, nil))

		assert.True(t, emptyConn.shutdownCalled)
		assert.False(t, activeConn.shutdownCalled)
		assert.Len(t, ws.connections, 1)
		assert.Same(t, activeConn, ws.connections[0])
	})
}

func TestUnsubscribeFromConnection(t *testing.T) {
	t.Parallel()
	m := NewManager()

	_, err := m.unsubscribeFromConnection(t.Context(), &connection{}, nil)
	require.ErrorContains(t, err, "websocket connection nil pointer: *subscription.Store")

	m.subscriptions = subscription.NewStore()

	store := subscription.NewStore()
	sub1 := &subscription.Subscription{Channel: "sub1"}
	subs := subscription.List{sub1}

	remaining, err := m.unsubscribeFromConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err, "unsubscribeFromConnection must not error when no subs in store")
	assert.Equal(t, subs, remaining, "remaining should equal input subs when none removed")

	require.NoError(t, store.Add(sub1))
	m.Unsubscriber = func(subscription.List) error { return nil }
	_, err = m.unsubscribeFromConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.ErrorIs(t, err, subscription.ErrNotFound, "must error if sub not in manager store")

	require.NoError(t, m.subscriptions.Add(sub1))
	m.Unsubscriber = func(subscription.List) error {
		return errors.New("unsub failed")
	}
	_, err = m.unsubscribeFromConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.ErrorContains(t, err, "unsub failed")

	m.Unsubscriber = func(subscription.List) error { return nil }
	sub2 := &subscription.Subscription{Channel: "sub2"}
	subs = subscription.List{sub1, sub2}

	remaining, err = m.unsubscribeFromConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err, "unsubscribeFromConnection must not error when unsubscribing existing subs")

	assert.Nil(t, store.Get(sub1), "sub1 should be removed from store")
	assert.NotNil(t, m.subscriptions.Get(sub1), "sub1 should still be in global store")

	assert.Len(t, remaining, 1)
	assert.Equal(t, "sub2", remaining[0].Channel)
}

func TestSubscribeToConnection(t *testing.T) {
	t.Parallel()
	m := NewManager()

	_, err := m.subscribeToConnection(t.Context(), &connection{}, nil)
	require.ErrorContains(t, err, "websocket connection nil pointer: *subscription.Store")

	m.subscriptions = subscription.NewStore()
	m.Subscriber = func(subscription.List) error { return nil }

	store := subscription.NewStore()
	sub1 := &subscription.Subscription{Channel: "sub1"}
	sub2 := &subscription.Subscription{Channel: "sub2"}
	sub3 := &subscription.Subscription{Channel: "sub3"}
	subs := subscription.List{sub1, sub2, sub3}

	m.MaxSubscriptionsPerConnection = 1
	require.NoError(t, store.Add(&subscription.Subscription{Channel: "existing"}))

	remaining, err := m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err, "subscribeToConnection must not error when full capacity")
	assert.Equal(t, subs, remaining, "remaining should equal input subs when capacity full")

	m = NewManager()
	m.subscriptions = subscription.NewStore()
	m.Subscriber = func(subscription.List) error { return nil }
	store = subscription.NewStore()
	require.NoError(t, store.Add(&subscription.Subscription{Channel: "existing"}))
	m.MaxSubscriptionsPerConnection = 3

	// subs has 3 items. Capacity is 3. Used is 1. Available is 2.
	// Should subscribe to sub1, sub2. Return sub3.
	remaining, err = m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err)
	assert.Len(t, remaining, 1, "should return 1 remaining subscription")
	assert.Equal(t, sub3, remaining[0], "should return sub3")

	assert.NotNil(t, store.Get(sub1), "sub1 should be added to store")
	assert.NotNil(t, store.Get(sub2), "sub2 should be added to store")
	assert.Nil(t, store.Get(sub3), "sub3 should not be added to store")

	m = NewManager()
	m.subscriptions = subscription.NewStore()
	m.Subscriber = func(subscription.List) error { return nil }
	store = subscription.NewStore()
	m.MaxSubscriptionsPerConnection = 0

	remaining, err = m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err)
	assert.Empty(t, remaining, "should return no remaining subscriptions with no capacity limit")
	assert.Equal(t, 3, store.Len(), "store should have 3 subscriptions when no capacity limit")

	m = NewManager()
	m.subscriptions = subscription.NewStore()
	m.Subscriber = func(subscription.List) error { return errors.New("sub failed") }
	store = subscription.NewStore()

	_, err = m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.ErrorContains(t, err, "sub failed")

	m = NewManager()
	m.subscriptions = subscription.NewStore()
	m.Subscriber = func(subscription.List) error { return nil }
	store = subscription.NewStore()
	require.NoError(t, store.Add(sub1)) // sub1 already in store

	_, err = m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.ErrorIs(t, err, subscription.ErrDuplicate, "must error when subscription already in store")

	m = NewManager()
	m.MaxSubscriptionsPerConnection = 50
	m.subscriptions = subscription.NewStore()
	m.Subscriber = func(subscription.List) error { return nil }
	store = subscription.NewStore()

	_, err = m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err, "must not error when all subscriptions can be added, this exercises the path where available > len(subs)")
}
