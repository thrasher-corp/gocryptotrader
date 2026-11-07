package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

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

	// A managed connection holding one subscription is held to its setup's own cap, or to the manager's without one
	for _, tc := range []struct {
		managerCap, setupCap, incoming int
		err                            error
	}{
		{managerCap: 1, setupCap: 2, incoming: 1},
		{managerCap: 1, setupCap: 2, incoming: 2, err: errSubscriptionsExceedsLimit},
		{managerCap: 2, setupCap: 1, incoming: 1, err: errSubscriptionsExceedsLimit},
		{setupCap: 2, incoming: 1},
		{managerCap: 2, incoming: 1},
		{managerCap: 1, incoming: 1, err: errSubscriptionsExceedsLimit},
	} {
		m := &Manager{MaxSubscriptionsPerConnection: tc.managerCap}
		managed := &websocket{setup: &ConnectionSetup{MaxSubscriptionsPerConnection: tc.setupCap}, subscriptions: subscription.NewStore()}
		managedConn := &connection{subscriptions: subscription.NewStore()}
		require.NoError(t, managedConn.subscriptions.Add(&subscription.Subscription{Channel: "held"}), "Add must not error")
		m.trackConnection(managedConn, managed)
		incoming := make(subscription.List, tc.incoming)
		for i := range incoming {
			incoming[i] = &subscription.Subscription{Channel: fmt.Sprint("incoming-", i)}
		}
		assert.ErrorIsf(t, m.checkSubscriptions(managedConn, incoming), tc.err, "checkSubscriptions should apply the cap for a manager cap of %d, a setup cap of %d and %d incoming", tc.managerCap, tc.setupCap, tc.incoming)
	}
}

func TestMaxSubscriptionsPerConnection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		managerCap int
		ws         *websocket
		exp        int
	}{
		{name: "an unmanaged connection", managerCap: 3, exp: 3},
		{name: "a websocket without a setup", managerCap: 3, ws: &websocket{}, exp: 3},
		{name: "a setup without a cap", managerCap: 3, ws: &websocket{setup: &ConnectionSetup{}}, exp: 3},
		{name: "a negative setup cap", managerCap: 3, ws: &websocket{setup: &ConnectionSetup{MaxSubscriptionsPerConnection: -1}}, exp: 3},
		{name: "a setup cap below the manager's", managerCap: 3, ws: &websocket{setup: &ConnectionSetup{MaxSubscriptionsPerConnection: 2}}, exp: 2},
		{name: "a setup cap above the manager's", managerCap: 3, ws: &websocket{setup: &ConnectionSetup{MaxSubscriptionsPerConnection: 5}}, exp: 5},
		{name: "a setup cap without a manager cap", ws: &websocket{setup: &ConnectionSetup{MaxSubscriptionsPerConnection: 5}}, exp: 5},
		{name: "no cap", ws: &websocket{setup: &ConnectionSetup{}}},
	} {
		m := &Manager{MaxSubscriptionsPerConnection: tc.managerCap}
		assert.Equalf(t, tc.exp, m.maxSubscriptionsPerConnection(tc.ws), "maxSubscriptionsPerConnection should return the cap applying to %s", tc.name)
	}
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

func TestInitSubscriptionStore(t *testing.T) {
	t.Parallel()

	t.Run("GlobalStore", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{}
		store := manager.initSubscriptionStore(nil)

		require.NotNil(t, store, "global subscription store must be initialised")
		assert.Same(t, store, manager.subscriptions, "global subscription store should be retained on the manager")
	})

	t.Run("ManagedConnectionStore", func(t *testing.T) {
		t.Parallel()

		manager, conn := newManagedSubscriptionTestManagerWithStore(t, nil)
		require.Nil(t, manager.connectionManager[0].subscriptions, "managed websocket store must start nil for this test")

		store := manager.initSubscriptionStore(conn)

		require.NotNil(t, store, "managed connection store must be initialised")
		assert.Same(t, store, manager.connectionManager[0].subscriptions, "managed websocket should retain the initialised store")
		assert.NotSame(t, store, manager.subscriptions, "managed connection should keep an isolated subscription store")
	})
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

func newManagedSubscriptionTestManagerWithStore(t *testing.T, store *subscription.Store) (*Manager, Connection) {
	t.Helper()

	manager := NewManager()
	setup := newDefaultSetup()
	setup.UseMultiConnectionManagement = true
	require.NoError(t, manager.Setup(setup))

	ws := &websocket{
		setup: &ConnectionSetup{
			URL: "wss://managed-subscriptions.test/ws",
			Subscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
				return manager.AddSuccessfulSubscriptions(conn, subs...)
			},
			Unsubscriber: func(_ context.Context, conn Connection, subs subscription.List) error {
				return manager.RemoveSubscriptions(conn, subs...)
			},
		},
		subscriptions: store,
	}
	conn := &connection{
		URL:           ws.setup.URL,
		subscriptions: subscription.NewStore(),
	}

	manager.connectionManagerMu.Lock()
	manager.connectionManager = append(manager.connectionManager, ws)
	manager.connections[conn] = ws
	ws.connections = []Connection{conn}
	manager.connectionManagerMu.Unlock()

	return manager, conn
}

func newManagedSubscriptionTestManager(t *testing.T) (*Manager, Connection) {
	t.Helper()
	return newManagedSubscriptionTestManagerWithStore(t, subscription.NewStore())
}

func startSubscriptionReaders(manager *Manager) func() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	var once sync.Once
	for range 4 {
		wg.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = manager.GetSubscription("missing")
					_ = manager.GetSubscriptions()
				}
			}
		})
	}

	return func() {
		once.Do(func() {
			close(done)
			wg.Wait()
		})
	}
}

func runConcurrentSubscriptionOps(t *testing.T, workers int, op func(int) error) {
	t.Helper()

	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup

	for i := range workers {
		wg.Go(func() {
			<-start
			errs <- op(i)
		})
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		require.NoError(t, err)
	}
}

func newConcurrentSubscription(channel string, index int) *subscription.Subscription {
	name := fmt.Sprintf("%s-%d", channel, index)
	return &subscription.Subscription{
		Key:     name,
		Channel: name,
	}
}

func TestExportedManagedSubscriptionFunctionsConcurrent(t *testing.T) {
	t.Parallel()

	const workers = 32

	t.Run("AddSubscriptions", func(t *testing.T) {
		t.Parallel()

		manager, conn := newManagedSubscriptionTestManager(t)
		stopReaders := startSubscriptionReaders(manager)
		defer stopReaders()

		subs := make(subscription.List, workers)
		for i := range workers {
			subs[i] = newConcurrentSubscription("add", i)
		}

		runConcurrentSubscriptionOps(t, workers, func(i int) error {
			return manager.AddSubscriptions(conn, subs[i])
		})

		stopReaders()

		require.Len(t, manager.GetSubscriptions(), workers)
		for _, sub := range subs {
			require.Same(t, sub, manager.GetSubscription(sub))
			assert.Equal(t, subscription.SubscribingState, sub.State())
		}
	})

	t.Run("AddSuccessfulSubscriptions", func(t *testing.T) {
		t.Parallel()

		manager, conn := newManagedSubscriptionTestManager(t)
		stopReaders := startSubscriptionReaders(manager)
		defer stopReaders()

		subs := make(subscription.List, workers)
		for i := range workers {
			subs[i] = newConcurrentSubscription("success", i)
		}

		runConcurrentSubscriptionOps(t, workers, func(i int) error {
			return manager.AddSuccessfulSubscriptions(conn, subs[i])
		})

		stopReaders()

		require.Len(t, manager.GetSubscriptions(), workers)
		for _, sub := range subs {
			require.Same(t, sub, manager.GetSubscription(sub))
			assert.Equal(t, subscription.SubscribedState, sub.State())
		}
	})

	t.Run("RemoveSubscriptions", func(t *testing.T) {
		t.Parallel()

		manager, conn := newManagedSubscriptionTestManager(t)
		subs := make(subscription.List, workers)
		for i := range workers {
			subs[i] = newConcurrentSubscription("remove", i)
		}
		require.NoError(t, manager.AddSuccessfulSubscriptions(conn, subs...))

		stopReaders := startSubscriptionReaders(manager)
		defer stopReaders()

		runConcurrentSubscriptionOps(t, workers, func(i int) error {
			return manager.RemoveSubscriptions(conn, subs[i])
		})

		stopReaders()

		assert.Empty(t, manager.GetSubscriptions())
		for _, sub := range subs {
			assert.Nil(t, manager.GetSubscription(sub))
			assert.Equal(t, subscription.UnsubscribedState, sub.State())
		}
	})

	t.Run("SubscribeToChannels", func(t *testing.T) {
		t.Parallel()

		manager, conn := newManagedSubscriptionTestManager(t)
		stopReaders := startSubscriptionReaders(manager)
		defer stopReaders()

		subs := make(subscription.List, workers)
		for i := range workers {
			subs[i] = newConcurrentSubscription("subscribe", i)
		}

		runConcurrentSubscriptionOps(t, workers, func(i int) error {
			return manager.SubscribeToChannels(t.Context(), conn, subscription.List{subs[i]})
		})

		stopReaders()

		require.Len(t, manager.GetSubscriptions(), workers)
		for _, sub := range subs {
			require.Same(t, sub, manager.GetSubscription(sub))
			assert.Equal(t, subscription.SubscribedState, sub.State())
		}
	})

	t.Run("UnsubscribeChannels", func(t *testing.T) {
		t.Parallel()

		manager, conn := newManagedSubscriptionTestManager(t)
		subs := make(subscription.List, workers)
		for i := range workers {
			subs[i] = newConcurrentSubscription("unsubscribe", i)
		}
		require.NoError(t, manager.AddSuccessfulSubscriptions(conn, subs...))

		stopReaders := startSubscriptionReaders(manager)
		defer stopReaders()

		runConcurrentSubscriptionOps(t, workers, func(i int) error {
			return manager.UnsubscribeChannels(t.Context(), conn, subscription.List{subs[i]})
		})

		stopReaders()

		assert.Empty(t, manager.GetSubscriptions())
		for _, sub := range subs {
			assert.Nil(t, manager.GetSubscription(sub))
			assert.Equal(t, subscription.UnsubscribedState, sub.State())
		}
	})
}

func TestManagedSubscriptionGettersConcurrentStoreInit(t *testing.T) {
	t.Parallel()

	const workers = 32

	manager, conn := newManagedSubscriptionTestManagerWithStore(t, nil)
	stopReaders := startSubscriptionReaders(manager)
	defer stopReaders()

	subs := make(subscription.List, workers)
	for i := range workers {
		subs[i] = newConcurrentSubscription("initialise", i)
	}

	runConcurrentSubscriptionOps(t, workers, func(i int) error {
		return manager.AddSuccessfulSubscriptions(conn, subs[i])
	})

	stopReaders()

	require.Len(t, manager.GetSubscriptions(), workers)
	for _, sub := range subs {
		require.Same(t, sub, manager.GetSubscription(sub))
	}
}

func TestFlushChannelsConcurrentReaders(t *testing.T) {
	t.Parallel()

	manager, conn := newManagedSubscriptionTestManager(t)
	manager.setEnabled(true)
	manager.setState(connectedState)
	manager.MaxSubscriptionsPerConnection = 2

	expected := subscription.List{
		newConcurrentSubscription("flush", 0),
		newConcurrentSubscription("flush", 1),
	}
	manager.connectionManager[0].setup.GenerateSubscriptions = func() (subscription.List, error) {
		return expected, nil
	}

	stopReaders := startSubscriptionReaders(manager)
	defer stopReaders()

	runConcurrentSubscriptionOps(t, 8, func(int) error {
		return manager.FlushChannels(t.Context())
	})

	stopReaders()

	require.Len(t, manager.GetSubscriptions(), len(expected))
	require.Len(t, conn.Subscriptions().List(), len(expected))
	for _, sub := range expected {
		require.Same(t, sub, manager.GetSubscription(sub))
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
	var cleanupMonitors sync.Once
	cleanupW := func() {
		cleanupMonitors.Do(func() { cleanupManagerMonitors(t, w) })
	}
	t.Cleanup(cleanupW)
	w.exchangeName = "test"
	w.connector = noopConnect
	w.Subscriber = newgen.SUBME
	w.Unsubscriber = newgen.UNSUBME
	// Keep enough headroom for FlushChannels connection cycles without leaving long-lived monitor goroutines behind.
	w.trafficTimeout = time.Second

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
	mock, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	t.Cleanup(cleanupW)

	w.subscriptions = subscription.NewStore()

	amazingCandidate := &ConnectionSetup{
		URL: "ws" + mock.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
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
	subscriptions     *subscription.Store
	subscriptionsHook func()
	shutdownCalled    bool
	shutdownErr       error
}

func (f *fakeConnection) Shutdown() error {
	f.shutdownCalled = true
	return f.shutdownErr
}

func (f *fakeConnection) Subscriptions() *subscription.Store {
	if f.subscriptionsHook != nil {
		f.subscriptionsHook()
	}
	return f.subscriptions
}

func cleanupManagedConnectionReaders(t *testing.T, m *Manager, ws *websocket) {
	t.Helper()
	if m == nil || ws == nil {
		return
	}
	for _, conn := range m.snapshotManagedConnections(ws) {
		_ = conn.Shutdown()
	}
	resetManagerForNextConnectAttempt(t, m)
}

func TestApplyTrackedSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("NoTrackedSubscriptionsNoOp", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		conn := &fakeConnection{}
		require.NoError(t, m.applyTrackedSubscriptions(conn, nil))
	})

	t.Run("RecordsInManagerAndConnectionStores", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		sub := &subscription.Subscription{Channel: "tracked"}
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{setup: &ConnectionSetup{}, subscriptions: subscription.NewStore()}
		m.trackConnection(conn, ws)

		require.NoError(t, m.applyTrackedSubscriptions(conn, subscription.List{sub}))
		require.NotNil(t, ws.subscriptions.Get(sub), "tracked subscription must be recorded in websocket store")
		require.NotNil(t, conn.subscriptions.Get(sub), "tracked subscription must be recorded in connection store")
		assert.Equal(t, subscription.SubscribedState, sub.State())
	})

	t.Run("ErrorsWhenConnectionStoreIsNil", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		sub := &subscription.Subscription{Channel: "tracked"}
		conn := &fakeConnection{}
		ws := &websocket{setup: &ConnectionSetup{}, subscriptions: subscription.NewStore()}
		m.trackConnection(conn, ws)

		err := m.applyTrackedSubscriptions(conn, subscription.List{sub})
		require.ErrorIs(t, err, common.ErrNilPointer)
	})
}

func TestAbsorbSubscriptionsAndValidate(t *testing.T) {
	t.Parallel()

	t.Run("PassThroughWithoutTrackHook", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		subs := subscription.List{{Channel: "A"}}
		ws := &websocket{
			setup:         &ConnectionSetup{},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{&fakeConnection{subscriptions: subscription.NewStore()}},
		}

		remaining, err := m.absorbTrackableSubscriptionsAndValidate(t.Context(), ws, subs)
		require.NoError(t, err)
		assert.Equal(t, subs, remaining)
	})

	t.Run("PropagatesTrackError", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		expectedErr := errors.New("track failed")
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(context.Context, Connection, subscription.List) (subscription.List, subscription.List, error) {
					return nil, nil, expectedErr
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{&fakeConnection{subscriptions: subscription.NewStore()}},
		}

		remaining, err := m.absorbTrackableSubscriptionsAndValidate(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, remaining)
	})

	t.Run("ErrorsWhenTrackedSubsNotRecordedOnWebsocketStore", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		trackedSub := &subscription.Subscription{Channel: "tracked"}
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(context.Context, Connection, subscription.List) (subscription.List, subscription.List, error) {
					return nil, subscription.List{trackedSub}, nil
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn},
		}

		// Intentionally do not map conn -> ws; manager-level tracking goes to
		// the global store, allowing validation to detect missing websocket state.
		remaining, err := m.absorbTrackableSubscriptionsAndValidate(t.Context(), ws, subscription.List{trackedSub})
		require.ErrorIs(t, err, ErrSubscriptionFailure)
		require.ErrorIs(t, err, ErrSubscriptionsNotAdded)
		assert.Nil(t, remaining)
	})

	t.Run("ReturnsRemainingAndRecordsTracked", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		trackedSub := &subscription.Subscription{Channel: "tracked"}
		remainingSub := &subscription.Subscription{Channel: "remaining"}
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(_ context.Context, _ Connection, _ subscription.List) (subscription.List, subscription.List, error) {
					return subscription.List{remainingSub}, subscription.List{trackedSub}, nil
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn},
		}
		m.trackConnection(conn, ws)

		remaining, err := m.absorbTrackableSubscriptionsAndValidate(t.Context(), ws, subscription.List{trackedSub, remainingSub})
		require.NoError(t, err)
		require.Len(t, remaining, 1)
		assert.Same(t, remainingSub, remaining[0])
		require.NotNil(t, ws.subscriptions.Get(trackedSub))
		require.NotNil(t, conn.subscriptions.Get(trackedSub))
	})
}

func TestAbsorbTrackedSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("PassthroughWithoutTrackHook", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		subs := subscription.List{{Channel: "A"}}
		ws := &websocket{
			setup:       &ConnectionSetup{},
			connections: []Connection{&fakeConnection{subscriptions: subscription.NewStore()}},
		}

		remaining, tracked, err := m.absorbTrackedSubscriptions(t.Context(), ws, subs)
		require.NoError(t, err)
		assert.Equal(t, subs, remaining)
		assert.Empty(t, tracked)
	})

	t.Run("TracksAcrossConnectionsUntilEmpty", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		tracked := &subscription.Subscription{Channel: "tracked"}
		conn0 := &fakeConnection{subscriptions: subscription.NewStore()}
		conn1 := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, subscription.List, error) {
					if conn != conn1 {
						return subs, nil, nil
					}
					return nil, subscription.List{tracked}, nil
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn0, conn1},
		}
		m.connections[conn0] = ws
		m.connections[conn1] = ws

		remaining, trackedSubs, err := m.absorbTrackedSubscriptions(t.Context(), ws, subscription.List{tracked})
		require.NoError(t, err)
		require.Nil(t, remaining)
		require.Len(t, trackedSubs, 1)
		require.Same(t, tracked, trackedSubs[0])
		require.NotNil(t, ws.subscriptions.Get(tracked))
		require.NotNil(t, conn1.subscriptions.Get(tracked))
		assert.Nil(t, conn0.subscriptions.Get(tracked))
	})

	t.Run("PropagatesErrors", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		expectedErr := errors.New("track failed")
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(context.Context, Connection, subscription.List) (subscription.List, subscription.List, error) {
					return nil, nil, expectedErr
				},
			},
			connections: []Connection{&fakeConnection{subscriptions: subscription.NewStore()}},
		}

		remaining, tracked, err := m.absorbTrackedSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, remaining)
		assert.Nil(t, tracked)
	})
}

func TestScaleConnectionsToSubscriptions(t *testing.T) {
	t.Parallel()

	// Common setup helper
	setup := func(t *testing.T, isMultiConn bool) (*Manager, *websocket) {
		t.Helper()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 2
		m.useMultiConnectionManagement = isMultiConn

		// Mock server for dialing
		srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))

		ws := &websocket{
			setup: &ConnectionSetup{
				URL: "ws" + srv.URL[len("http"):] + "/ws",
				Connector: func(ctx context.Context, c Connection) error {
					return c.Dial(ctx, dialer, nil, nil)
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
		t.Cleanup(func() { cleanupManagedConnectionReaders(t, m, ws) })
		return m, ws
	}

	t.Run("Nil ws", func(t *testing.T) {
		t.Parallel()
		m, _ := setup(t, false)
		err := m.scaleConnectionsToSubscriptions(t.Context(), nil, nil)
		require.ErrorIs(t, err, common.ErrNilPointer)
	})

	t.Run("No Changes", func(t *testing.T) {
		t.Parallel()
		m, ws := setup(t, false)
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, nil)
		require.NoError(t, err)
	})

	t.Run("Scale Up (Add Subs)", func(t *testing.T) {
		t.Parallel()
		m, ws := setup(t, false)

		subs := subscription.List{{Channel: "A"}, {Channel: "B"}, {Channel: "C"}}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subs)
		require.NoError(t, err)

		assert.Equal(t, 3, ws.subscriptions.Len())
		assert.Len(t, ws.connections, 2) // 2 per conn -> 2 conns
	})

	t.Run("Scale Down (Remove Subs)", func(t *testing.T) {
		t.Parallel()
		m, ws := setup(t, true)

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
		m, ws := setup(t, true)

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
		m, ws := setup(t, false)

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
		m, ws := setup(t, false)

		// Set connector error
		ws.setup.Connector = func(context.Context, Connection) error {
			return errors.New("connect fail")
		}

		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorContains(t, err, "connect fail")
	})

	t.Run("Global Unsubscribe Fallback Success", func(t *testing.T) {
		t.Parallel()
		m, ws := setup(t, false)

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
		m, ws := setup(t, false)

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
		m, ws := setup(t, false)

		m.MaxSubscriptionsPerConnection = 2

		in := subscription.List{{Channel: "A"}, {Channel: "B"}, {Channel: "C"}, {Channel: "D"}, {Channel: "E"}}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, in))

		// With max 2 subs per connection and 5 total, we expect 3 connections
		assert.Len(t, ws.connections, 3)
	})

	t.Run("Setup Cap Overrides Manager Cap", func(t *testing.T) {
		t.Parallel()
		// The manager caps each connection at 2; each step scales to the first n subscriptions of the pool
		for _, tc := range []struct {
			setupCap int
			steps    []int
			loads    [][]int
		}{
			{setupCap: 3, steps: []int{5, 6, 7}, loads: [][]int{{3, 2}, {3, 3}, {3, 3, 1}}},
			{setupCap: 1, steps: []int{3}, loads: [][]int{{1, 1, 1}}},
		} {
			m, ws := setup(t, true)
			ws.setup.MaxSubscriptionsPerConnection = tc.setupCap
			pool := make(subscription.List, tc.steps[len(tc.steps)-1])
			for i := range pool {
				pool[i] = &subscription.Subscription{Channel: fmt.Sprint("capped-", i)}
			}
			for i, n := range tc.steps {
				require.NoErrorf(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, pool[:n]), "scaleConnectionsToSubscriptions must not error scaling to %d subscriptions with a setup cap of %d", n, tc.setupCap)
				conns := m.snapshotManagedConnections(ws)
				loads := make([]int, 0, len(conns))
				for _, conn := range conns {
					loads = append(loads, conn.Subscriptions().Len())
				}
				assert.Equalf(t, tc.loads[i], loads, "scaleConnectionsToSubscriptions should fill each connection to the setup cap of %d before adding one for %d subscriptions", tc.setupCap, n)
			}
		}
	})

	t.Run("Global Unsubscribe Fallback Remove Error", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.useMultiConnectionManagement = true
		m.connections = make(map[Connection]*websocket)
		held := &subscription.Subscription{Channel: "A"}
		orphan := &subscription.Subscription{Channel: "B"} // Stored for the setup but on no connection
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn.subscriptions.Add(held))
		ws := &websocket{subscriptions: subscription.NewStore(), connections: []Connection{conn}}
		ws.setup = &ConnectionSetup{
			// An unsubscriber that also drops a subscription it was not given leaves the fallback nothing to remove
			Unsubscriber: func(_ context.Context, c Connection, subs subscription.List) error {
				return common.AppendError(m.RemoveSubscriptions(c, subs...), ws.subscriptions.Remove(orphan))
			},
		}
		require.NoError(t, ws.subscriptions.Add(held))
		require.NoError(t, ws.subscriptions.Add(orphan))
		m.connections[conn] = ws
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, nil)
		require.ErrorIs(t, err, subscription.ErrNotFound, "scaleConnectionsToSubscriptions must return the fallback's failure to remove a subscription")
	})

	t.Run("Unsubscriber Leaving The Subscription Stored", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.useMultiConnectionManagement = true
		m.connections = make(map[Connection]*websocket)
		held := &subscription.Subscription{Channel: "A"}
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn.subscriptions.Add(held))
		ws := &websocket{
			setup:         &ConnectionSetup{Unsubscriber: func(context.Context, Connection, subscription.List) error { return nil }},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn},
		}
		require.NoError(t, ws.subscriptions.Add(held))
		m.connections[conn] = ws
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, nil)
		require.ErrorIs(t, err, ErrSubscriptionsNotRemoved, "scaleConnectionsToSubscriptions must report a subscription its unsubscriber left stored")
	})

	// trackingSetup returns a websocket with one full connection, whose TrackOnExistingConnection answers its second
	// call with second, and whose Connector refuses to dial
	trackingSetup := func(t *testing.T, second func(Connection, subscription.List) (subscription.List, subscription.List, error)) (*Manager, *websocket, *subscription.Subscription) {
		t.Helper()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 1
		m.connections = make(map[Connection]*websocket)
		existing := &subscription.Subscription{Channel: "A"}
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn.subscriptions.Add(existing))
		var calls int
		ws := &websocket{
			setup: &ConnectionSetup{
				Connector: func(context.Context, Connection) error { return errDastardlyReason },
				TrackOnExistingConnection: func(_ context.Context, c Connection, subs subscription.List) (subscription.List, subscription.List, error) {
					calls++
					if calls == 1 {
						return subs, nil, nil
					}
					return second(c, subs)
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn},
		}
		require.NoError(t, ws.subscriptions.Add(existing))
		m.connections[conn] = ws
		return m, ws, existing
	}

	t.Run("Track On Existing Connection Error", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.connections = make(map[Connection]*websocket)
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(context.Context, Connection, subscription.List) (subscription.List, subscription.List, error) {
					return nil, nil, errDastardlyReason
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn},
		}
		m.connections[conn] = ws
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorIs(t, err, errDastardlyReason, "scaleConnectionsToSubscriptions must return the tracking error")
	})

	t.Run("Track On Existing Connection Error For A New Batch", func(t *testing.T) {
		t.Parallel()
		m, ws, existing := trackingSetup(t, func(Connection, subscription.List) (subscription.List, subscription.List, error) {
			return nil, nil, errSubscriptionRejected
		})
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{existing, {Channel: "B"}})
		require.ErrorIs(t, err, errSubscriptionRejected, "scaleConnectionsToSubscriptions must return the tracking error for a batch needing a new connection")
		assert.Len(t, ws.connections, 1, "scaleConnectionsToSubscriptions should not connect after the tracking error")
	})

	t.Run("New Batch Tracked On Existing Connection", func(t *testing.T) {
		t.Parallel()
		m, ws, existing := trackingSetup(t, func(_ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
			return nil, subs, nil
		})
		logical := &subscription.Subscription{Channel: "B"}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{existing, logical}), "scaleConnectionsToSubscriptions must not connect for a batch tracked on an existing connection")
		assert.Len(t, ws.connections, 1, "scaleConnectionsToSubscriptions should not add a connection for a tracked batch")
		assert.Same(t, logical, ws.subscriptions.Get(logical), "scaleConnectionsToSubscriptions should store the tracked subscription")
	})

	t.Run("Subscriber Storing Nothing On An Existing Connection", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.connections = make(map[Connection]*websocket)
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{
			setup: &ConnectionSetup{
				Subscriber: func(context.Context, Connection, subscription.List) error { return nil },
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn},
		}
		m.connections[conn] = ws
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorIs(t, err, ErrSubscriptionsNotAdded, "scaleConnectionsToSubscriptions must report a subscription its subscriber did not store")
	})

	// A connection refusing a subscription, wherever it falls, does not stop those after it, which other connections take.
	// A connection left with nothing takes the next batch, so that refusals leave at most one connection empty
	for _, tc := range []struct {
		name    string
		refused []int
		loads   []int
	}{
		{name: "On An Existing Connection", refused: []int{0}, loads: []int{1, 2, 2, 2}},
		{name: "On A New Connection", refused: []int{1}, loads: []int{2, 1, 2, 2}},
		{name: "On Every Connection", refused: []int{0, 1, 2, 3, 4, 5, 6}, loads: []int{1, 0}},
	} {
		t.Run("Refused Subscription "+tc.name, func(t *testing.T) {
			t.Parallel()
			m, ws := setup(t, true)
			held := &subscription.Subscription{Channel: "held"}
			require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{held}), "scaleConnectionsToSubscriptions must not error")
			incoming := make(subscription.List, 7)
			refused := make(subscription.List, 0, len(tc.refused))
			for i := range incoming {
				incoming[i] = &subscription.Subscription{Channel: fmt.Sprint("incoming-", i)}
				if slices.Contains(tc.refused, i) {
					refused = append(refused, incoming[i])
				}
			}
			ws.setup.Subscriber = func(_ context.Context, c Connection, subs subscription.List) error {
				var errs error
				for _, s := range subs {
					if slices.Contains(refused, s) {
						errs = common.AppendError(errs, fmt.Errorf("%w: %s", errSubscriptionRejected, s.Channel))
						continue
					}
					errs = common.AppendError(errs, m.AddSuccessfulSubscriptions(c, s))
				}
				return errs
			}
			for range 2 { // A flush while the subscription is still refused changes nothing
				err := m.scaleConnectionsToSubscriptions(t.Context(), ws, append(subscription.List{held}, incoming...))
				require.ErrorIs(t, err, errSubscriptionRejected, "scaleConnectionsToSubscriptions must report the refused subscription")
				for _, s := range incoming {
					if slices.Contains(refused, s) {
						assert.Nilf(t, ws.subscriptions.Get(s), "refused %s should not be stored", s.Channel)
						assert.ErrorContainsf(t, err, s.Channel, "scaleConnectionsToSubscriptions should report refused %s", s.Channel)
					} else {
						assert.Samef(t, s, ws.subscriptions.Get(s), "%s should be subscribed despite the refused subscription", s.Channel)
					}
				}
				conns := m.snapshotManagedConnections(ws)
				loads := make([]int, 0, len(conns))
				for _, conn := range conns {
					loads = append(loads, conn.Subscriptions().Len())
				}
				assert.Equal(t, tc.loads, loads, "each connection should hold the subscriptions it took")
			}
		})
	}

	// A flush whose context ends sends nothing more, to an existing connection or a new one
	for _, tc := range []struct {
		name string
		held int
		kept []int
	}{
		{name: "With Room On Two Existing Connections", held: 4, kept: []int{0, 2}},
		{name: "With Only New Connections", held: 2, kept: []int{0, 1}},
	} {
		t.Run("Context Ending During A Subscribe "+tc.name, func(t *testing.T) {
			t.Parallel()
			m, ws := setup(t, true)
			held := make(subscription.List, tc.held)
			for i := range held {
				held[i] = &subscription.Subscription{Channel: fmt.Sprint("held-", i)}
			}
			require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, held), "scaleConnectionsToSubscriptions must not error")
			kept := make(subscription.List, 0, len(tc.kept))
			for _, i := range tc.kept {
				kept = append(kept, held[i])
			}
			require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, kept), "scaleConnectionsToSubscriptions must not error")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var subscribes, tracks int
			ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
				subscribes++
				cancel()
				return errSubscriptionRejected
			}
			// Tracking can send requests of its own, so it stops with the context too
			ws.setup.TrackOnExistingConnection = func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
				tracks++
				return subs, nil, nil
			}
			incoming := append(slices.Clone(kept), &subscription.Subscription{Channel: "A"}, &subscription.Subscription{Channel: "B"}, &subscription.Subscription{Channel: "C"}, &subscription.Subscription{Channel: "D"})
			err := m.scaleConnectionsToSubscriptions(ctx, ws, incoming)
			require.ErrorIs(t, err, errSubscriptionRejected, "scaleConnectionsToSubscriptions must report the subscribe that failed")
			require.ErrorIs(t, err, context.Canceled, "scaleConnectionsToSubscriptions must report the context ending")
			assert.Equal(t, 1, subscribes, "scaleConnectionsToSubscriptions should send nothing after the context ends")
			assert.Equal(t, 2, tracks, "scaleConnectionsToSubscriptions should track nothing after the context ends")
		})
	}

	for _, tc := range []struct {
		name    string
		tracked bool
	}{
		{name: "With A Batch Left To Send"},
		{name: "Which Took The Last Batch", tracked: true},
	} {
		t.Run("Context Ending During Tracking "+tc.name, func(t *testing.T) {
			t.Parallel()
			m, ws := setup(t, true)
			var subscribes int
			ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
				subscribes++
				return errSubscriptionRejected
			}
			a := &subscription.Subscription{Channel: "A"}
			require.ErrorIs(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{a}), errSubscriptionRejected, "scaleConnectionsToSubscriptions must leave a connection empty")
			require.Len(t, m.snapshotManagedConnections(ws), 1, "scaleConnectionsToSubscriptions must leave one connection")
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var tracks int
			ws.setup.TrackOnExistingConnection = func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
				tracks++
				if tracks != 2 {
					return subs, nil, nil
				}
				cancel()
				if tc.tracked {
					return nil, subs, nil
				}
				return subs, nil, nil
			}
			subscribes = 0
			err := m.scaleConnectionsToSubscriptions(ctx, ws, subscription.List{a, {Channel: "B"}, {Channel: "C"}, {Channel: "D"}})
			require.ErrorIs(t, err, errSubscriptionRejected, "scaleConnectionsToSubscriptions must report the subscribe that failed")
			require.ErrorIs(t, err, context.Canceled, "scaleConnectionsToSubscriptions must report the context ending")
			assert.Equal(t, 1, subscribes, "scaleConnectionsToSubscriptions should send nothing after the context ends")
		})
	}

	t.Run("Connection Failure After A Refused Subscription", func(t *testing.T) {
		t.Parallel()
		m, ws := setup(t, true)
		connect := ws.setup.Connector
		var dials int
		ws.setup.Connector = func(ctx context.Context, c Connection) error {
			dials++
			if dials > 1 {
				return errDastardlyReason
			}
			return connect(ctx, c)
		}
		refused := &subscription.Subscription{Channel: "refused"}
		ws.setup.Subscriber = func(_ context.Context, c Connection, subs subscription.List) error {
			var errs error
			for _, s := range subs {
				if s == refused {
					errs = common.AppendError(errs, errSubscriptionRejected)
					continue
				}
				errs = common.AppendError(errs, m.AddSuccessfulSubscriptions(c, s))
			}
			return errs
		}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{refused, {Channel: "A"}, {Channel: "B"}, {Channel: "C"}})
		require.ErrorIs(t, err, errSubscriptionRejected, "scaleConnectionsToSubscriptions must keep the earlier refusal")
		require.ErrorIs(t, err, errDastardlyReason, "scaleConnectionsToSubscriptions must report the connection failure")
		assert.Equal(t, 2, dials, "scaleConnectionsToSubscriptions should not dial again after a connection failure")
	})

	t.Run("Tracking Failure After A Refused Subscription", func(t *testing.T) {
		t.Parallel()
		m, ws := setup(t, true)
		held := &subscription.Subscription{Channel: "held"}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{held}), "scaleConnectionsToSubscriptions must not error")
		var tracks int
		ws.setup.TrackOnExistingConnection = func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
			tracks++
			if tracks > 1 {
				return nil, nil, errDastardlyReason
			}
			return subs, nil, nil
		}
		ws.setup.Subscriber = func(context.Context, Connection, subscription.List) error {
			return errSubscriptionRejected
		}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{held, {Channel: "A"}, {Channel: "B"}, {Channel: "C"}})
		require.ErrorIs(t, err, errSubscriptionRejected, "scaleConnectionsToSubscriptions must keep the earlier refusal")
		require.ErrorIs(t, err, errDastardlyReason, "scaleConnectionsToSubscriptions must report the tracking failure")
		assert.Len(t, m.snapshotManagedConnections(ws), 1, "scaleConnectionsToSubscriptions should open no connection after a tracking failure")
	})

	t.Run("Connection Failure Stops Later Connections", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 1
		var dials int
		ws := &websocket{
			setup: &ConnectionSetup{
				Connector: func(context.Context, Connection) error {
					dials++
					return errDastardlyReason
				},
			},
			subscriptions: subscription.NewStore(),
		}
		err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{{Channel: "A"}, {Channel: "B"}, {Channel: "C"}})
		require.ErrorIs(t, err, common.ErrFatal, "scaleConnectionsToSubscriptions must return a connection failure as fatal")
		require.ErrorIs(t, err, errDastardlyReason, "scaleConnectionsToSubscriptions must keep the connection failure's cause")
		assert.Equal(t, 1, dials, "scaleConnectionsToSubscriptions should not dial again after a connection failure")
	})

	t.Run("Stale Connection Shutdown Error", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.connections = make(map[Connection]*websocket)
		conn := &fakeConnection{subscriptions: subscription.NewStore(), shutdownErr: errDastardlyReason}
		ws := &websocket{setup: &ConnectionSetup{}, subscriptions: subscription.NewStore(), connections: []Connection{conn}}
		m.connections[conn] = ws
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, nil), "scaleConnectionsToSubscriptions must not fail because a stale connection fails to shut down")
		assert.True(t, conn.shutdownCalled, "scaleConnectionsToSubscriptions should shut the stale connection down")
		assert.Empty(t, ws.connections, "scaleConnectionsToSubscriptions should drop the stale connection")
	})

	t.Run("Track On Existing Connection Prevents New Connection", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 1
		m.connections = make(map[Connection]*websocket)

		existing := &subscription.Subscription{Channel: "A"}
		logical := &subscription.Subscription{Channel: "B"}
		activeConn := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, activeConn.subscriptions.Add(existing))

		ws := &websocket{
			setup: &ConnectionSetup{
				Connector: func(context.Context, Connection) error {
					return errors.New("should not create a new connection")
				},
				TrackOnExistingConnection: func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
					if len(subs) != 1 || subs[0].Channel != logical.Channel {
						return subs, nil, nil
					}
					return nil, subscription.List{logical}, nil
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{activeConn},
		}
		require.NoError(t, ws.subscriptions.Add(existing))
		m.connections[activeConn] = ws

		incoming := subscription.List{existing, logical}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, incoming))
		assert.Len(t, ws.connections, 1)
		assert.Equal(t, 2, ws.subscriptions.Len())
		assert.Equal(t, 2, activeConn.subscriptions.Len())
	})

	t.Run("Track On Existing Connection Targets Owning Connection", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 1
		m.connections = make(map[Connection]*websocket)

		// conn0 owns subA, conn1 owns subB. A logical sub "C" whose inverse
		// lives on conn1 must be tracked on conn1, not conn0.
		subA := &subscription.Subscription{Channel: "A"}
		subB := &subscription.Subscription{Channel: "B"}
		logical := &subscription.Subscription{Channel: "C"}

		conn0 := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn0.subscriptions.Add(subA))

		conn1 := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn1.subscriptions.Add(subB))

		var trackedOnConn Connection
		ws := &websocket{
			setup: &ConnectionSetup{
				Connector: func(context.Context, Connection) error {
					return errors.New("should not create a new connection")
				},
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, subscription.List, error) {
					// Only track when the connection owns subB (the inverse).
					if conn.Subscriptions().Get(subB) == nil {
						return subs, nil, nil
					}
					trackedOnConn = conn
					return nil, subscription.List{logical}, nil
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn0, conn1},
		}
		require.NoError(t, ws.subscriptions.Add(subA))
		require.NoError(t, ws.subscriptions.Add(subB))
		m.connections[conn0] = ws
		m.connections[conn1] = ws

		incoming := subscription.List{subA, subB, logical}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, incoming))

		assert.Len(t, ws.connections, 2, "no new connections should be created")
		assert.Same(t, conn1, trackedOnConn, "logical sub should be tracked on the connection owning the inverse, not connections[0]")
		assert.Equal(t, 1, conn0.subscriptions.Len(), "conn0 should not gain the logical sub")
		assert.Equal(t, 2, conn1.subscriptions.Len(), "conn1 should own both subB and logical")
	})

	t.Run("Track Before Generic Subscribe Prevents Misrouting", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 2
		m.connections = make(map[Connection]*websocket)

		// conn0 has subA and spare capacity (max=2, used=1).
		// conn1 has subB (the inverse of logical sub C).
		// Without the pre-subscribe tracking pass, the generic
		// subscribeToConnection loop would route C to conn0 because
		// it has capacity. The fix ensures absorbTrackedSubscriptions
		// runs first, absorbing C onto conn1.
		subA := &subscription.Subscription{Channel: "A"}
		subB := &subscription.Subscription{Channel: "B"}
		logical := &subscription.Subscription{Channel: "C"}

		conn0 := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn0.subscriptions.Add(subA))

		conn1 := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, conn1.subscriptions.Add(subB))

		var trackedOnConn Connection
		ws := &websocket{
			setup: &ConnectionSetup{
				Subscriber: func(_ context.Context, c Connection, s subscription.List) error {
					return m.AddSuccessfulSubscriptions(c, s...)
				},
				Connector: func(context.Context, Connection) error {
					return errors.New("should not create a new connection")
				},
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, subscription.List, error) {
					// Only track when the connection owns subB (the inverse).
					if conn.Subscriptions().Get(subB) == nil {
						return subs, nil, nil
					}
					var remaining subscription.List
					var tracked subscription.List
					for _, s := range subs {
						if s.Channel != logical.Channel {
							remaining = append(remaining, s)
							continue
						}
						trackedOnConn = conn
						tracked = append(tracked, s)
					}
					return remaining, tracked, nil
				},
				Handler: func(context.Context, Connection, []byte) error { return nil },
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn0, conn1},
		}
		require.NoError(t, ws.subscriptions.Add(subA))
		require.NoError(t, ws.subscriptions.Add(subB))
		m.connections[conn0] = ws
		m.connections[conn1] = ws

		incoming := subscription.List{subA, subB, logical}
		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, incoming))

		assert.Len(t, ws.connections, 2, "no new connections should be created")
		assert.Same(t, conn1, trackedOnConn, "logical sub should be tracked on the connection owning the inverse, not conn0 which has spare capacity")
		assert.Equal(t, 1, conn0.subscriptions.Len(), "conn0 should not gain the logical sub")
		assert.Equal(t, 2, conn1.subscriptions.Len(), "conn1 should own both subB and logical")
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

	t.Run("Cleanup Keeps Freshly Added Connections", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 2
		m.connections = make(map[Connection]*websocket)

		ws := &websocket{
			setup:         &ConnectionSetup{},
			subscriptions: subscription.NewStore(),
		}

		emptyConn := &fakeConnection{subscriptions: subscription.NewStore()}
		activeConn := &fakeConnection{subscriptions: subscription.NewStore()}
		freshConn := &fakeConnection{subscriptions: subscription.NewStore()}
		require.NoError(t, activeConn.subscriptions.Add(&subscription.Subscription{Channel: "A"}))
		require.NoError(t, freshConn.subscriptions.Add(&subscription.Subscription{Channel: "B"}))

		var addFreshOnce sync.Once
		emptyConn.subscriptionsHook = func() {
			addFreshOnce.Do(func() {
				m.connectionManagerMu.Lock()
				m.connections[freshConn] = ws
				ws.connections = append(ws.connections, freshConn)
				m.connectionManagerMu.Unlock()
			})
		}

		ws.connections = []Connection{emptyConn, activeConn}
		m.connections[emptyConn] = ws
		m.connections[activeConn] = ws

		require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, nil))

		assert.True(t, emptyConn.shutdownCalled)
		assert.False(t, activeConn.shutdownCalled)
		assert.False(t, freshConn.shutdownCalled)
		assert.Len(t, ws.connections, 2)

		var haveActive, haveFresh bool
		for _, conn := range ws.connections {
			if conn == activeConn {
				haveActive = true
			}
			if conn == freshConn {
				haveFresh = true
			}
		}
		assert.True(t, haveActive)
		assert.True(t, haveFresh)
		assert.NotContains(t, m.connections, emptyConn)
		assert.Same(t, ws, m.connections[freshConn])
	})
}

func TestConnectTracksOnExistingConnectionBeforeNewConnection(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.exchangeName = "test"
	m.useMultiConnectionManagement = true
	m.MaxSubscriptionsPerConnection = 1
	m.trafficTimeout = time.Minute
	m.setEnabled(true)

	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })

	subA := &subscription.Subscription{Channel: "A"}
	subB := &subscription.Subscription{Channel: "B"}
	var connectorCalls int
	var trackCalls int
	require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
		URL: "ws" + srv.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			connectorCalls++
			return conn.Dial(ctx, dialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{subA, subB}, nil
		},
		Subscriber: func(_ context.Context, c Connection, s subscription.List) error {
			return m.AddSuccessfulSubscriptions(c, s...)
		},
		TrackOnExistingConnection: func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
			if len(subs) != 1 || subs[0] != subB {
				return subs, nil, nil
			}
			trackCalls++
			return nil, subscription.List{subB}, nil
		},
		Handler: func(context.Context, Connection, []byte) error { return nil },
	}))

	require.NoError(t, m.Connect(t.Context()))

	require.Len(t, m.connectionManager, 1)
	assert.Equal(t, 1, connectorCalls, "connect path should only dial once when later subscriptions are tracked on the existing connection")
	assert.Equal(t, 1, trackCalls, "later subscription batch should be handled by TrackOnExistingConnection")
	assert.NotNil(t, m.connectionManager[0].subscriptions.Get(subA), "first subscription should be tracked logically")
	assert.NotNil(t, m.connectionManager[0].subscriptions.Get(subB), "later tracked subscription should be tracked logically")
}

func TestConnectReducesTrackedSubscriptionsBeforeBatching(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.exchangeName = "test"
	m.useMultiConnectionManagement = true
	m.MaxSubscriptionsPerConnection = 2
	m.trafficTimeout = time.Minute
	m.setEnabled(true)

	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })

	realA := &subscription.Subscription{Channel: "real-A"}
	trackedA := &subscription.Subscription{Channel: "tracked-A"}
	realB := &subscription.Subscription{Channel: "real-B"}
	trackedB := &subscription.Subscription{Channel: "tracked-B"}
	trackable := map[*subscription.Subscription]bool{
		trackedA: true,
		trackedB: true,
	}

	var connectorCalls int
	var trackBatchLens []int
	require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
		URL: "ws" + srv.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			connectorCalls++
			return conn.Dial(ctx, dialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{realA, trackedA, realB, trackedB}, nil
		},
		Subscriber: func(_ context.Context, c Connection, s subscription.List) error {
			return m.AddSuccessfulSubscriptions(c, s...)
		},
		TrackOnExistingConnection: func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
			trackBatchLens = append(trackBatchLens, len(subs))
			remaining := make(subscription.List, 0, len(subs))
			tracked := make(subscription.List, 0, len(subs))
			for _, sub := range subs {
				if !trackable[sub] {
					remaining = append(remaining, sub)
					continue
				}
				tracked = append(tracked, sub)
			}
			if len(tracked) == 0 {
				return subs, nil, nil
			}
			return remaining, tracked, nil
		},
		Handler: func(context.Context, Connection, []byte) error { return nil },
	}))

	ws := m.connectionManager[0]
	existingConn := &fakeConnection{subscriptions: subscription.NewStore()}
	m.trackConnection(existingConn, ws)

	require.NoError(t, m.Connect(t.Context()))

	assert.Equal(t, 1, connectorCalls, "pre-batch tracking reduces the real subscriptions to a single connection")
	assert.Equal(t, []int{4, 2}, trackBatchLens, "Connect reduces against existing connections before batching and still re-checks reduced batches")
	assert.NotNil(t, ws.subscriptions.Get(trackedA), "tracked-A is tracked logically before batching")
	assert.NotNil(t, ws.subscriptions.Get(trackedB), "tracked-B is tracked logically before batching")
	assert.NotNil(t, ws.subscriptions.Get(realA), "real-A is tracked logically")
	assert.NotNil(t, ws.subscriptions.Get(realB), "real-B is tracked logically")
}

func TestConnectPreBatchTrackedSubscriptionsAutoRecordState(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.exchangeName = "test"
	m.useMultiConnectionManagement = true
	m.MaxSubscriptionsPerConnection = 2
	m.trafficTimeout = time.Minute
	m.setEnabled(true)

	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })

	realSub := &subscription.Subscription{Channel: "real"}
	trackedSub := &subscription.Subscription{Channel: "tracked"}
	require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
		URL: "ws" + srv.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{realSub, trackedSub}, nil
		},
		Subscriber: func(_ context.Context, c Connection, s subscription.List) error {
			return m.AddSuccessfulSubscriptions(c, s...)
		},
		TrackOnExistingConnection: func(_ context.Context, _ Connection, subs subscription.List) (subscription.List, subscription.List, error) {
			remaining := make(subscription.List, 0, len(subs))
			tracked := make(subscription.List, 0, len(subs))
			for _, sub := range subs {
				if sub == trackedSub {
					tracked = append(tracked, sub)
					continue
				}
				remaining = append(remaining, sub)
			}
			return remaining, tracked, nil
		},
		Handler: func(context.Context, Connection, []byte) error { return nil },
	}))

	ws := m.connectionManager[0]
	m.trackConnection(&fakeConnection{subscriptions: subscription.NewStore()}, ws)

	require.NoError(t, m.Connect(t.Context()))
	require.NotNil(t, ws.subscriptions.Get(trackedSub), "tracked subscriptions must be recorded by the manager")
}

// TestConnectionSetupSubscriptionCap checks that Connect and FlushChannels hold the connections of each setup in one
// manager to the setup's own subscription cap, or to the manager's when the setup sets none
func TestConnectionSetupSubscriptionCap(t *testing.T) {
	t.Parallel()

	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	m := NewManager()
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })
	s := newDefaultSetup()
	s.UseMultiConnectionManagement = true
	s.MaxWebsocketSubscriptionsPerConnection = 3
	s.ExchangeConfig.WebsocketTrafficTimeout = time.Minute
	require.NoError(t, m.Setup(s), "Setup must not error")

	var mu sync.Mutex
	generated := 5
	setups := []struct {
		name  string
		limit int
	}{{name: "below", limit: 2}, {name: "above", limit: 4}, {name: "unset"}}
	for _, tc := range setups {
		require.NoErrorf(t, m.SetupNewConnection(&ConnectionSetup{
			URL: "ws" + srv.URL[len("http"):] + "/" + tc.name,
			Connector: func(ctx context.Context, conn Connection) error {
				return conn.Dial(ctx, dialer, nil, nil)
			},
			GenerateSubscriptions: func() (subscription.List, error) {
				mu.Lock()
				defer mu.Unlock()
				subs := make(subscription.List, generated)
				for i := range subs {
					subs[i] = &subscription.Subscription{Channel: fmt.Sprint(tc.name, "-", i)}
				}
				return subs, nil
			},
			Subscriber:                    currySimpleSubConn(m),
			Unsubscriber:                  currySimpleUnsubConn(m),
			Handler:                       func(context.Context, Connection, []byte) error { return nil },
			MaxSubscriptionsPerConnection: tc.limit,
		}), "SetupNewConnection must not error for %s", tc.name)
	}
	loads := func() map[string][]int {
		l := make(map[string][]int, len(setups))
		for i, ws := range m.snapshotConnectionManager() {
			for _, conn := range m.snapshotManagedConnections(ws) {
				l[setups[i].name] = append(l[setups[i].name], conn.Subscriptions().Len())
			}
		}
		return l
	}

	require.NoError(t, m.Connect(t.Context()), "Connect must not error")
	assert.Equal(t, map[string][]int{"below": {2, 2, 1}, "above": {4, 1}, "unset": {3, 2}}, loads(), "Connect should batch each setup's subscriptions by its own cap, or by the manager's when it sets none")

	mu.Lock()
	generated = 12
	mu.Unlock()
	require.NoError(t, m.FlushChannels(t.Context()), "FlushChannels must not error")
	assert.Equal(t, map[string][]int{"below": {2, 2, 2, 2, 2, 2}, "above": {4, 4, 4}, "unset": {3, 3, 3, 3}}, loads(), "FlushChannels should fill each connection to its setup's cap, or to the manager's, before batching the rest onto new ones")
}

func TestResubscribeFromConnection(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		m.Unsubscriber = func(subscription.List) error { return nil }
		m.Subscriber = func(subscription.List) error { return nil }
		sub1 := &subscription.Subscription{Channel: "sub1"}
		sub2 := &subscription.Subscription{Channel: "sub2"}
		store := subscription.NewStore()
		require.NoError(t, store.Add(sub1))
		require.NoError(t, store.Add(sub2))
		m.subscriptions = store
		conn := &connection{subscriptions: store}

		err := m.ResubscribeFromConnection(t.Context(), conn, subscription.List{sub1})
		require.NoError(t, err)
		require.Contains(t, m.subscriptions.List(), sub1, "sub1 must still be in global store")
		require.Contains(t, conn.subscriptions.List(), sub1, "sub1 must still be in global store")
	})
	t.Run("NilConnection", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		err := m.ResubscribeFromConnection(t.Context(), nil, nil)
		require.ErrorIs(t, err, common.ErrNilPointer)
	})
	t.Run("Bad state", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		m.Unsubscriber = func(subscription.List) error { return nil }
		m.Subscriber = func(subscription.List) error { return nil }
		sub1 := &subscription.Subscription{Channel: "sub1"}
		require.NoError(t, sub1.SetState(subscription.ResubscribingState), "sub1 must be in unsubscribed state for this test")
		store := subscription.NewStore()
		require.NoError(t, store.Add(sub1))
		m.subscriptions = store
		conn := &connection{subscriptions: store}

		err := m.ResubscribeFromConnection(t.Context(), conn, subscription.List{sub1})
		require.ErrorIs(t, err, subscription.ErrInStateAlready, "must error when subscription is not in unsubscribed state")
	})
	t.Run("Bad unsub", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		m.Unsubscriber = func(subscription.List) error { return errAlreadyConnected }
		m.Subscriber = func(subscription.List) error { return nil }
		sub1 := &subscription.Subscription{Channel: "sub1"}
		store := subscription.NewStore()
		require.NoError(t, store.Add(sub1))
		m.subscriptions = store
		conn := &connection{subscriptions: store}

		err := m.ResubscribeFromConnection(t.Context(), conn, subscription.List{sub1})
		require.ErrorIs(t, err, errAlreadyConnected, "must error")
	})
	t.Run("Bad sub", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		m.Unsubscriber = func(subscription.List) error { return nil }
		m.Subscriber = func(subscription.List) error { return errAlreadyConnected }
		sub1 := &subscription.Subscription{Channel: "sub1"}
		store := subscription.NewStore()
		require.NoError(t, store.Add(sub1))
		m.subscriptions = store
		conn := &connection{subscriptions: store}

		err := m.ResubscribeFromConnection(t.Context(), conn, subscription.List{sub1})
		require.ErrorIs(t, err, errAlreadyConnected, "must error")
	})
	t.Run("Missing connection subscription", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		sub := &subscription.Subscription{Channel: "sub"}
		require.NoError(t, m.subscriptions.Add(sub), "subscription must be added to the manager store")
		conn := &connection{subscriptions: subscription.NewStore()}
		subscriberCalled := false
		m.Subscriber = func(subscription.List) error {
			subscriberCalled = true
			return nil
		}

		err := m.ResubscribeFromConnection(t.Context(), conn, subscription.List{sub})
		require.ErrorIs(t, err, ErrSubscriptionsNotRemoved, "must error when the subscription is not owned by the connection")
		assert.False(t, subscriberCalled, "subscriber should not be called for a subscription owned by another connection")
	})
	t.Run("Capacity consumed during unsubscribe", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.MaxSubscriptionsPerConnection = 1
		m.subscriptions = subscription.NewStore()
		connStore := subscription.NewStore()
		sub := &subscription.Subscription{Channel: "sub"}
		other := &subscription.Subscription{Channel: "other"}
		require.NoError(t, m.subscriptions.Add(sub), "subscription must be added to the manager store")
		require.NoError(t, connStore.Add(sub), "subscription must be added to the connection store")
		m.Unsubscriber = func(subscription.List) error {
			return connStore.Add(other)
		}
		subscriberCalled := false
		m.Subscriber = func(subscription.List) error {
			subscriberCalled = true
			return nil
		}
		conn := &connection{subscriptions: connStore}

		err := m.ResubscribeFromConnection(t.Context(), conn, subscription.List{sub})
		require.ErrorIs(t, err, ErrSubscriptionsNotAdded, "must error when connection capacity is consumed during resubscription")
		assert.False(t, subscriberCalled, "subscriber should not be called when the connection has no capacity")
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

	managed, conn := newManagedSubscriptionTestManager(t)
	dropped := &subscription.Subscription{Channel: "dropped"}
	require.NoError(t, managed.SubscribeToChannels(t.Context(), conn, subscription.List{dropped}), "SubscribeToChannels must not error")
	managed.connections[conn].setup.Unsubscriber = func(_ context.Context, c Connection, subs subscription.List) error {
		return common.AppendError(managed.RemoveSubscriptions(c, subs...), c.Subscriptions().Remove(subs[0]))
	}
	_, err = managed.unsubscribeFromConnection(t.Context(), conn, subscription.List{dropped})
	require.ErrorIs(t, err, subscription.ErrNotFound, "unsubscribeFromConnection must report a connection record the unsubscriber already dropped")
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
	require.NoError(t, store.Add(&subscription.Subscription{Channel: "existing-1"}))
	require.NoError(t, store.Add(&subscription.Subscription{Channel: "existing-2"}))
	m.MaxSubscriptionsPerConnection = 1

	remaining, err = m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subs)
	require.NoError(t, err, "subscribeToConnection must not error when connection is already over logical capacity")
	assert.Equal(t, subs, remaining, "remaining should equal input subs when used capacity exceeds the limit")

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

	// A managed connection stops at its setup's own cap, or at the manager's without one
	for _, tc := range []struct {
		managerCap, setupCap, held, subscribed int
	}{
		{managerCap: 1, setupCap: 2, subscribed: 2},
		{managerCap: 3, setupCap: 1, subscribed: 1},
		{setupCap: 2, subscribed: 2},
		{managerCap: 1, setupCap: 3, held: 1, subscribed: 2},
		{managerCap: 3, setupCap: 2, held: 2},
		{managerCap: 2, subscribed: 2},
	} {
		managed, conn := newManagedSubscriptionTestManager(t)
		managed.MaxSubscriptionsPerConnection = tc.managerCap
		managed.connectionManager[0].setup.MaxSubscriptionsPerConnection = tc.setupCap
		for i := range tc.held {
			require.NoError(t, conn.Subscriptions().Add(&subscription.Subscription{Channel: fmt.Sprint("held-", i)}), "Add must not error")
		}
		incoming := subscription.List{{Channel: "one"}, {Channel: "two"}, {Channel: "three"}}
		remaining, err := managed.subscribeToConnection(t.Context(), conn, incoming)
		require.NoErrorf(t, err, "subscribeToConnection must not error for a manager cap of %d, a setup cap of %d and %d held", tc.managerCap, tc.setupCap, tc.held)
		assert.Equalf(t, incoming[tc.subscribed:], remaining, "subscribeToConnection should return what does not fit for a manager cap of %d, a setup cap of %d and %d held", tc.managerCap, tc.setupCap, tc.held)
		assert.Equalf(t, tc.held+tc.subscribed, conn.Subscriptions().Len(), "subscribeToConnection should fill the connection to its cap for a manager cap of %d, a setup cap of %d and %d held", tc.managerCap, tc.setupCap, tc.held)
	}
}

var errSubscriptionRejected = errors.New("subscription rejected")

// partialSubscriber accepts the first subscription of each batch and rejects the rest, as an exchange does when the
// venue refuses part of a batch
func partialSubscriber(m *Manager) func(context.Context, Connection, subscription.List) error {
	return func(_ context.Context, c Connection, subs subscription.List) error {
		if len(subs) == 0 {
			return nil
		}
		if err := m.AddSuccessfulSubscriptions(c, subs[0]); err != nil {
			return err
		}
		if len(subs) == 1 {
			return nil
		}
		return fmt.Errorf("%w: %s", errSubscriptionRejected, subs[1:])
	}
}

// partialUnsubscriber removes the first subscription of each batch and fails on the rest
func partialUnsubscriber(m *Manager) func(context.Context, Connection, subscription.List) error {
	return func(_ context.Context, c Connection, subs subscription.List) error {
		if len(subs) == 0 {
			return nil
		}
		if err := m.RemoveSubscriptions(c, subs[0]); err != nil {
			return err
		}
		if len(subs) == 1 {
			return nil
		}
		return fmt.Errorf("%w: %s", errSubscriptionRejected, subs[1:])
	}
}

func TestEmptyConnection(t *testing.T) {
	t.Parallel()
	m := NewManager()
	ws := &websocket{}
	assert.Nil(t, m.emptyConnection(ws), "emptyConnection should return nothing when there are no connections")
	holding := &fakeConnection{subscriptions: subscription.NewStore()}
	require.NoError(t, holding.subscriptions.Add(&subscription.Subscription{Channel: "held"}), "Add must not error")
	m.trackConnection(holding, ws)
	assert.Nil(t, m.emptyConnection(ws), "emptyConnection should return nothing when each connection holds subscriptions")
	empty := &fakeConnection{subscriptions: subscription.NewStore()}
	m.trackConnection(empty, ws)
	assert.Same(t, empty, m.emptyConnection(ws), "emptyConnection should return the connection holding no subscriptions")
}

func TestSubscribeToConnectionRecordsPartialSubscriptions(t *testing.T) {
	t.Parallel()

	t.Run("Managed connection", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.useMultiConnectionManagement = true
		ws := &websocket{subscriptions: subscription.NewStore(), setup: &ConnectionSetup{Subscriber: partialSubscriber(m)}}
		conn := &fakeConnection{subscriptions: subscription.NewStore()}
		m.trackConnection(conn, ws)

		accepted := &subscription.Subscription{Channel: "accepted"}
		rejected := &subscription.Subscription{Channel: "rejected"}
		remaining, err := m.subscribeToConnection(t.Context(), conn, subscription.List{accepted, rejected})
		require.ErrorIs(t, err, errSubscriptionRejected, "subscribeToConnection must return the subscriber error")
		assert.Empty(t, remaining, "remaining should hold only what was not attempted")
		require.NotNil(t, ws.subscriptions.Get(accepted), "accepted subscription must be in the websocket store")
		assert.NotNil(t, conn.subscriptions.Get(accepted), "accepted subscription should be recorded against the connection")
		assert.Nil(t, conn.subscriptions.Get(rejected), "rejected subscription should not be recorded against the connection")
		assert.Equal(t, 1, conn.subscriptions.Len(), "connection store should only hold the accepted subscription")
	})

	t.Run("Global subscriber", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		m.Subscriber = func(subs subscription.List) error {
			if err := m.AddSuccessfulSubscriptions(nil, subs[0]); err != nil {
				return err
			}
			return errSubscriptionRejected
		}
		store := subscription.NewStore()

		accepted := &subscription.Subscription{Channel: "accepted"}
		rejected := &subscription.Subscription{Channel: "rejected"}
		_, err := m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subscription.List{accepted, rejected})
		require.ErrorIs(t, err, ErrSubscriptionFailure, "subscribeToConnection must return a subscription failure")
		require.ErrorIs(t, err, errSubscriptionRejected, "subscribeToConnection must wrap the subscriber error")
		assert.NotNil(t, store.Get(accepted), "accepted subscription should be recorded against the connection")
		assert.Nil(t, store.Get(rejected), "rejected subscription should not be recorded against the connection")
	})

	t.Run("Subscription held before the call", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		subscriberCalled := false
		m.Subscriber = func(subscription.List) error {
			subscriberCalled = true
			return nil
		}
		held := &subscription.Subscription{Channel: "held"}
		require.NoError(t, m.subscriptions.Add(held), "subscription must be added to the manager store")
		store := subscription.NewStore()

		_, err := m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subscription.List{held})
		require.ErrorIs(t, err, subscription.ErrDuplicate, "subscribeToConnection must error on a subscription the manager already holds")
		assert.False(t, subscriberCalled, "subscriber should not be called for a duplicate subscription")
		assert.Zero(t, store.Len(), "a subscription held before the call should not be recorded against the connection")
	})

	t.Run("Nil subscription", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		m.subscriptions = subscription.NewStore()
		m.Subscriber = func(subscription.List) error { return nil }
		store := subscription.NewStore()

		_, err := m.subscribeToConnection(t.Context(), &connection{subscriptions: store}, subscription.List{{Channel: "one"}, nil})
		require.ErrorIs(t, err, common.ErrNilPointer, "subscribeToConnection must error on a nil subscription")
		assert.Zero(t, store.Len(), "nothing should be recorded against the connection")
	})
}

func TestUnsubscribeFromConnectionReleasesPartialUnsubscriptions(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.useMultiConnectionManagement = true
	ws := &websocket{subscriptions: subscription.NewStore(), setup: &ConnectionSetup{Unsubscriber: partialUnsubscriber(m)}}
	conn := &fakeConnection{subscriptions: subscription.NewStore()}
	m.trackConnection(conn, ws)

	removed := &subscription.Subscription{Channel: "removed"}
	kept := &subscription.Subscription{Channel: "kept"}
	for _, s := range []*subscription.Subscription{removed, kept} {
		require.NoError(t, ws.subscriptions.Add(s), "subscription must be added to the websocket store")
		require.NoError(t, conn.subscriptions.Add(s), "subscription must be added to the connection store")
	}

	missing, err := m.unsubscribeFromConnection(t.Context(), conn, subscription.List{removed, kept})
	require.ErrorIs(t, err, errSubscriptionRejected, "unsubscribeFromConnection must return the unsubscriber error")
	assert.Nil(t, missing, "missing should be nil on error")
	require.Nil(t, ws.subscriptions.Get(removed), "removed subscription must be gone from the websocket store")
	assert.Nil(t, conn.subscriptions.Get(removed), "removed subscription should be released from the connection")
	assert.NotNil(t, conn.subscriptions.Get(kept), "subscription still held should remain on the connection")
	assert.NotNil(t, ws.subscriptions.Get(kept), "subscription still held should remain in the websocket store")
}

func TestUnsubscribeFromConnectionKeepsSubscriptionsOnFailure(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.subscriptions = subscription.NewStore()
	m.Unsubscriber = func(subscription.List) error { return errSubscriptionRejected }
	store := subscription.NewStore()
	sub := &subscription.Subscription{Channel: "sub"}
	require.NoError(t, m.subscriptions.Add(sub), "subscription must be added to the manager store")
	require.NoError(t, store.Add(sub), "subscription must be added to the connection store")

	_, err := m.unsubscribeFromConnection(t.Context(), &connection{subscriptions: store}, subscription.List{sub})
	require.ErrorIs(t, err, errSubscriptionRejected, "unsubscribeFromConnection must return the unsubscriber error")
	assert.NotNil(t, store.Get(sub), "a subscription the manager still holds should remain on the connection")
}

func TestFlushChannelsKeepsConnectionWithPartialSubscriptions(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.exchangeName = "test"
	m.useMultiConnectionManagement = true
	m.features.Subscribe = true
	m.features.Unsubscribe = true
	m.trafficTimeout = time.Minute
	m.setEnabled(true)

	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })

	accepted := &subscription.Subscription{Channel: "accepted"}
	rejected := &subscription.Subscription{Channel: "rejected"}
	var enabledMu sync.Mutex
	enabled := subscription.List{accepted, rejected}
	require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
		URL: "ws" + srv.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			enabledMu.Lock()
			defer enabledMu.Unlock()
			return enabled, nil
		},
		Subscriber:   partialSubscriber(m),
		Unsubscriber: partialUnsubscriber(m),
		Handler:      func(context.Context, Connection, []byte) error { return nil },
	}))

	err := m.Connect(t.Context())
	require.ErrorIs(t, err, ErrSubscriptionFailure, "Connect must report the rejected subscription")
	require.ErrorIs(t, err, errSubscriptionRejected, "Connect must wrap the subscriber error")
	require.Len(t, m.connectionManager, 1, "must have one connection setup")
	ws := m.connectionManager[0]
	require.Len(t, ws.connections, 1, "connection must be established despite the rejected subscription")
	conn := ws.connections[0]

	enabledMu.Lock()
	enabled = subscription.List{accepted}
	enabledMu.Unlock()

	require.NoError(t, m.FlushChannels(t.Context()), "FlushChannels must not error")
	require.Len(t, ws.connections, 1, "connection holding the accepted subscription must not be flushed as stale")
	assert.Same(t, conn, ws.connections[0], "the original connection should be kept")
	wsConn, ok := conn.(*connection)
	require.True(t, ok, "connection must be a websocket connection")
	assert.True(t, wsConn.IsConnected(), "connection should remain connected")
	assert.NotNil(t, conn.Subscriptions().Get(accepted), "accepted subscription should remain recorded against the connection")
	assert.NotNil(t, ws.subscriptions.Get(accepted), "accepted subscription should remain in the websocket store")
}

func TestUnheldSubscriptions(t *testing.T) {
	t.Parallel()
	sub := &subscription.Subscription{Channel: "sub"}

	_, err := unheldSubscriptions(nil, subscription.List{sub})
	assert.ErrorIs(t, err, common.ErrNilPointer, "unheldSubscriptions should error without a manager store")

	_, err = unheldSubscriptions(subscription.NewStore(), subscription.List{sub, nil})
	assert.ErrorIs(t, err, common.ErrNilPointer, "unheldSubscriptions should error on a nil subscription")

	managerStore := subscription.NewStore()
	require.NoError(t, managerStore.Add(sub), "subscription must be added to the manager store")
	equivalent := &subscription.Subscription{Channel: "sub"}
	other := &subscription.Subscription{Channel: "other"}
	unheld, err := unheldSubscriptions(managerStore, subscription.List{sub, equivalent, other})
	require.NoError(t, err, "unheldSubscriptions must not error")
	assert.Equal(t, subscription.List{equivalent, other}, unheld, "unheldSubscriptions should return everything but the exact instance held")
}

func TestRecordConnectionSubscriptions(t *testing.T) {
	t.Parallel()
	sub := &subscription.Subscription{Channel: "sub"}
	managerStore := subscription.NewStore()
	require.NoError(t, managerStore.Add(sub), "subscription must be added to the manager store")
	connStore := subscription.NewStore()
	require.NoError(t, connStore.Add(sub), "subscription must be added to the connection store")

	err := recordConnectionSubscriptions(nil, managerStore, nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "recordConnectionSubscriptions should error without a connection store")

	err = recordConnectionSubscriptions(connStore, nil, subscription.List{sub})
	assert.ErrorIs(t, err, common.ErrNilPointer, "recordConnectionSubscriptions should error without a manager store")

	err = recordConnectionSubscriptions(connStore, managerStore, subscription.List{nil, sub})
	assert.ErrorIs(t, err, common.ErrNilPointer, "recordConnectionSubscriptions should error on a nil subscription")

	err = recordConnectionSubscriptions(connStore, managerStore, subscription.List{sub})
	assert.ErrorIs(t, err, ErrSubscriptionFailure, "recordConnectionSubscriptions should wrap a failed add as a subscription failure")
	assert.ErrorIs(t, err, subscription.ErrDuplicate, "recordConnectionSubscriptions should return the connection store error")
}

func TestReleaseConnectionSubscriptions(t *testing.T) {
	t.Parallel()
	sub := &subscription.Subscription{Channel: "sub"}

	err := releaseConnectionSubscriptions(nil, subscription.NewStore(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "releaseConnectionSubscriptions should error without a connection store")

	err = releaseConnectionSubscriptions(subscription.NewStore(), nil, subscription.List{sub})
	assert.ErrorIs(t, err, common.ErrNilPointer, "releaseConnectionSubscriptions should error without a manager store")

	err = releaseConnectionSubscriptions(subscription.NewStore(), subscription.NewStore(), subscription.List{nil, sub})
	assert.ErrorIs(t, err, common.ErrNilPointer, "releaseConnectionSubscriptions should error on a nil subscription")

	err = releaseConnectionSubscriptions(subscription.NewStore(), subscription.NewStore(), subscription.List{sub})
	assert.ErrorIs(t, err, subscription.ErrNotFound, "releaseConnectionSubscriptions should return the connection store error")
}

func TestScaleConnectionsToSubscriptionsDuplicateAcrossBatches(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.MaxSubscriptionsPerConnection = 1
	m.useMultiConnectionManagement = true
	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	ws := &websocket{
		setup: &ConnectionSetup{
			URL: "ws" + srv.URL[len("http"):] + "/ws",
			Connector: func(ctx context.Context, c Connection) error {
				return c.Dial(ctx, dialer, nil, nil)
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
	t.Cleanup(func() { cleanupManagedConnectionReaders(t, m, ws) })

	a := &subscription.Subscription{Channel: "a"}
	aDup := &subscription.Subscription{Channel: "a"}
	err := m.scaleConnectionsToSubscriptions(t.Context(), ws, subscription.List{a, aDup})
	require.ErrorIs(t, err, ErrSubscriptionFailure, "a duplicate in a later batch must fail its subscriber")
	require.ErrorIs(t, err, subscription.ErrDuplicate, "must return the duplicate subscription error")
	require.Len(t, ws.connections, 2, "each batch must get its own connection")
	assert.Same(t, a, ws.connections[0].Subscriptions().Get(a), "the first connection should hold the subscription it registered")
	assert.Zero(t, ws.connections[1].Subscriptions().Len(), "the duplicate's connection should not record the first connection's subscription")

	require.NoError(t, m.scaleConnectionsToSubscriptions(t.Context(), ws, nil), "removing all subscriptions must not error")
	assert.Zero(t, ws.subscriptions.Len(), "manager store should be empty")
	assert.Empty(t, ws.connections, "no connection should be left open once every subscription is removed")
}

func TestCreateConnectAndSubscribeRecordsEquivalentSubscriptions(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	mgr.useMultiConnectionManagement = true
	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	ws := &websocket{subscriptions: subscription.NewStore(), setup: &ConnectionSetup{
		URL: "ws" + srv.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, dialer, nil, nil)
		},
		Subscriber: func(_ context.Context, c Connection, subs subscription.List) error {
			for _, s := range subs {
				if err := mgr.AddSuccessfulSubscriptions(c, &subscription.Subscription{Channel: s.Channel}); err != nil {
					return err
				}
			}
			return nil
		},
		Handler: func(context.Context, Connection, []byte) error { return nil },
	}}
	t.Cleanup(func() { cleanupManagedConnectionReaders(t, mgr, ws) })

	subs := subscription.List{{Channel: "one"}, {Channel: "two"}}
	require.NoError(t, mgr.createConnectAndSubscribe(t.Context(), ws, subs), "createConnectAndSubscribe must succeed when the subscriber registers copies")
	require.Len(t, ws.connections, 1, "connection must be tracked by websocket")
	assert.Equal(t, len(subs), ws.connections[0].Subscriptions().Len(), "connection store should record the copies the subscriber registered")
}

func TestSubscribeToChannels(t *testing.T) {
	t.Parallel()

	t.Run("UnmanagedConnection", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		require.NoError(t, m.Setup(newDefaultSetup()), "Setup must not error")
		m.Subscriber = currySimpleSub(m)
		sub := &subscription.Subscription{Channel: "unmanaged"}
		require.NoError(t, m.SubscribeToChannels(t.Context(), nil, subscription.List{sub}), "SubscribeToChannels must not error")
		assert.Same(t, sub, m.GetSubscription(sub), "SubscribeToChannels should subscribe through the global subscriber")
	})

	t.Run("ManagedConnectionRecordsSubscriptions", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManager(t)
		subs := subscription.List{{Channel: "a"}, {Channel: "b"}}
		require.NoError(t, m.SubscribeToChannels(t.Context(), conn, subs), "SubscribeToChannels must not error")
		assert.ElementsMatch(t, subs, conn.Subscriptions().List(), "SubscribeToChannels should record the subscriptions against the connection")
		missing, err := m.unsubscribeFromConnection(t.Context(), conn, subs)
		require.NoError(t, err, "unsubscribeFromConnection must not error")
		assert.Empty(t, missing, "a flush should find every subscription made with SubscribeToChannels on the connection")
		assert.Empty(t, m.GetSubscriptions(), "a flush should unsubscribe every subscription made with SubscribeToChannels")
	})

	t.Run("ManagedConnectionRecordsWhatTheSubscriberAdded", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManager(t)
		m.connections[conn].setup.Subscriber = partialSubscriber(m)
		subs := subscription.List{{Channel: "accepted"}, {Channel: "rejected"}}
		err := m.SubscribeToChannels(t.Context(), conn, subs)
		require.ErrorIs(t, err, errSubscriptionRejected, "SubscribeToChannels must return the subscriber's error")
		assert.Equal(t, subs[:1], conn.Subscriptions().List(), "SubscribeToChannels should record the subscription the subscriber added before failing")
	})

	t.Run("ManagedConnectionWithoutStore", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManagerWithStore(t, nil)
		err := m.SubscribeToChannels(t.Context(), conn, subscription.List{{Channel: "a"}})
		require.ErrorIs(t, err, common.ErrNilPointer, "SubscribeToChannels must error without the manager's store")
		assert.Zero(t, conn.Subscriptions().Len(), "SubscribeToChannels should record nothing when it cannot subscribe")
	})

	t.Run("ManagedConnectionAlreadyRecorded", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManager(t)
		sub := &subscription.Subscription{Channel: "a"}
		require.NoError(t, conn.Subscriptions().Add(sub), "Add must not error")
		err := m.SubscribeToChannels(t.Context(), conn, subscription.List{sub})
		require.ErrorIs(t, err, subscription.ErrDuplicate, "SubscribeToChannels must report a subscription the connection already records")
		assert.Same(t, sub, m.GetSubscription(sub), "SubscribeToChannels should still subscribe")
	})
}

func TestSubscribeChannels(t *testing.T) {
	t.Parallel()

	m := NewManager()
	require.NoError(t, m.Setup(newDefaultSetup()), "Setup must not error")
	sub := &subscription.Subscription{Channel: "a"}
	require.ErrorIs(t, m.subscribeChannels(t.Context(), nil, subscription.List{nil}), common.ErrNilPointer, "subscribeChannels must reject a nil subscription")
	m.Subscriber = nil
	require.ErrorIs(t, m.subscribeChannels(t.Context(), nil, subscription.List{sub}), common.ErrNilPointer, "subscribeChannels must error without a subscriber")
	m.Subscriber = func(subscription.List) error { return errDastardlyReason }
	err := m.subscribeChannels(t.Context(), nil, subscription.List{sub})
	require.ErrorIs(t, err, ErrSubscriptionFailure, "subscribeChannels must report a subscription failure")
	require.ErrorIs(t, err, errDastardlyReason, "subscribeChannels must wrap the subscriber's error")
	m.Subscriber = currySimpleSub(m)
	require.NoError(t, m.subscribeChannels(t.Context(), nil, subscription.List{sub}), "subscribeChannels must not error")
	require.ErrorIs(t, m.subscribeChannels(t.Context(), nil, subscription.List{sub}), subscription.ErrDuplicate, "subscribeChannels must refuse a subscription the manager already stores")

	managed, conn := newManagedSubscriptionTestManager(t)
	managedSub := &subscription.Subscription{Channel: "managed"}
	require.NoError(t, managed.subscribeChannels(t.Context(), conn, subscription.List{managedSub}), "subscribeChannels must not error on a managed connection")
	assert.Same(t, managedSub, managed.GetSubscription(managedSub), "subscribeChannels should subscribe through the connection's subscriber")
	assert.Zero(t, conn.Subscriptions().Len(), "subscribeChannels should leave the connection's store to its caller")
}

func TestUnsubscribeChannels(t *testing.T) {
	t.Parallel()

	t.Run("ManagedConnectionDropsSubscriptions", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManager(t)
		subs := subscription.List{{Channel: "a"}, {Channel: "b"}}
		require.NoError(t, m.SubscribeToChannels(t.Context(), conn, subs), "SubscribeToChannels must not error")
		require.NoError(t, m.UnsubscribeChannels(t.Context(), conn, subs[:1]), "UnsubscribeChannels must not error")
		assert.Equal(t, subs[1:], conn.Subscriptions().List(), "UnsubscribeChannels should drop the subscription from the connection's store")
		assert.Equal(t, subs[1:], m.GetSubscriptions(), "UnsubscribeChannels should unsubscribe the subscription")
	})

	t.Run("ManagedConnectionReleasesWhatTheUnsubscriberRemoved", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManager(t)
		subs := subscription.List{{Channel: "removed"}, {Channel: "kept"}}
		require.NoError(t, m.SubscribeToChannels(t.Context(), conn, subs), "SubscribeToChannels must not error")
		m.connections[conn].setup.Unsubscriber = partialUnsubscriber(m)
		err := m.UnsubscribeChannels(t.Context(), conn, subs)
		require.ErrorIs(t, err, errSubscriptionRejected, "UnsubscribeChannels must return the unsubscriber's error")
		assert.Equal(t, subs[1:], conn.Subscriptions().List(), "UnsubscribeChannels should drop only what the unsubscriber removed from the connection's store")
	})

	t.Run("unsubscribeChannelsNeedsAManagedConnection", func(t *testing.T) {
		t.Parallel()
		m, _ := newManagedSubscriptionTestManager(t)
		subs := subscription.List{{Channel: "a"}}
		require.ErrorIs(t, m.unsubscribeChannels(t.Context(), nil, subs), common.ErrNilPointer, "unsubscribeChannels must refuse a nil connection")
		require.ErrorIs(t, m.unsubscribeChannels(t.Context(), &connection{URL: "wss://unknown.test/ws"}, subs), errConnectionNotFound, "unsubscribeChannels must refuse a connection the manager does not manage")
	})

	t.Run("unsubscribeChannelsLeavesConnectionStore", func(t *testing.T) {
		t.Parallel()
		m, conn := newManagedSubscriptionTestManager(t)
		sub := &subscription.Subscription{Channel: "a"}
		require.NoError(t, m.SubscribeToChannels(t.Context(), conn, subscription.List{sub}), "SubscribeToChannels must not error")
		require.NoError(t, m.unsubscribeChannels(t.Context(), conn, subscription.List{sub}), "unsubscribeChannels must not error")
		assert.Empty(t, m.GetSubscriptions(), "unsubscribeChannels should unsubscribe through the connection's unsubscriber")
		assert.Equal(t, subscription.List{sub}, conn.Subscriptions().List(), "unsubscribeChannels should leave the connection's store to its caller")
	})
}

func TestFlushChannelsFlushesEveryConnection(t *testing.T) {
	t.Parallel()

	m := NewManager()
	m.exchangeName = "test"
	m.useMultiConnectionManagement = true
	m.features.Subscribe = true
	m.features.Unsubscribe = true
	m.trafficTimeout = time.Minute
	m.setEnabled(true)

	srv, dialer := mockws.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })

	var mu sync.Mutex
	generated := map[string]subscription.List{
		"failing":   {{Channel: "failing-1"}},
		"working":   {{Channel: "working-1"}},
		"rejecting": {{Channel: "rejecting-1"}},
	}
	refusals := map[string]error{"failing": errDastardlyReason, "rejecting": errSubscriptionRejected}
	refuse := false
	for _, name := range []string{"failing", "working", "rejecting"} {
		require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
			URL: "ws" + srv.URL[len("http"):] + "/" + name,
			Connector: func(ctx context.Context, conn Connection) error {
				return conn.Dial(ctx, dialer, nil, nil)
			},
			GenerateSubscriptions: func() (subscription.List, error) {
				mu.Lock()
				defer mu.Unlock()
				return generated[name], nil
			},
			Subscriber: func(ctx context.Context, conn Connection, subs subscription.List) error {
				mu.Lock()
				refused := refuse
				mu.Unlock()
				if err := refusals[name]; refused && err != nil {
					return err
				}
				return currySimpleSubConn(m)(ctx, conn, subs)
			},
			Unsubscriber: currySimpleUnsubConn(m),
			Handler:      func(context.Context, Connection, []byte) error { return nil },
		}), "SetupNewConnection must not error for %s", name)
	}
	require.NoError(t, m.Connect(t.Context()), "Connect must not error")

	mu.Lock()
	refuse = true
	generated["failing"] = subscription.List{{Channel: "failing-2"}}
	generated["working"] = subscription.List{{Channel: "working-2"}}
	generated["rejecting"] = subscription.List{{Channel: "rejecting-2"}}
	mu.Unlock()
	err := m.flushChannels(t.Context())
	require.ErrorIs(t, err, errDastardlyReason, "flushChannels must report the first connection that failed")
	require.ErrorIs(t, err, errSubscriptionRejected, "flushChannels must report every connection that failed")

	require.Len(t, m.connectionManager, 3, "must have every connection setup")
	working := m.connectionManager[1]
	require.Equal(t, "working", working.setup.URL[len(working.setup.URL)-len("working"):], "the second setup must be the working one")
	channels := make([]string, 0, working.subscriptions.Len())
	for _, s := range working.subscriptions.List() {
		channels = append(channels, s.Channel)
	}
	assert.Equal(t, []string{"working-2"}, channels, "flushChannels should flush the connections after one that fails")
}

func TestFlushChannelsSkipsSetupsAndStopsOnGenerationError(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.useMultiConnectionManagement = true
	m.features.Subscribe = true
	m.features.Unsubscribe = true
	m.setEnabled(true)
	m.setState(connectedState)
	var generated []string
	generate := func(name string, err error) func() (subscription.List, error) {
		return func() (subscription.List, error) {
			generated = append(generated, name)
			return nil, err
		}
	}
	m.connectionManager = []*websocket{
		{setup: &ConnectionSetup{SubscriptionsNotRequired: true, GenerateSubscriptions: generate("not required", nil)}, subscriptions: subscription.NewStore()},
		{setup: &ConnectionSetup{GenerateSubscriptions: generate("nothing", nil)}, subscriptions: subscription.NewStore()},
		{setup: &ConnectionSetup{GenerateSubscriptions: generate("failing", errDastardlyReason)}, subscriptions: subscription.NewStore()},
		{setup: &ConnectionSetup{GenerateSubscriptions: generate("after", nil)}, subscriptions: subscription.NewStore()},
	}
	require.ErrorIs(t, m.flushChannels(t.Context()), errDastardlyReason, "flushChannels must stop on a generation error that is not partial")
	assert.Equal(t, []string{"nothing", "failing"}, generated, "flushChannels should skip a setup that needs no subscriptions and stop at the generation error")
}

func TestFlushChannelsKeepsScalingErrorsOnGenerationError(t *testing.T) {
	t.Parallel()
	m := NewManager()
	s := newDefaultSetup()
	s.UseMultiConnectionManagement = true
	require.NoError(t, m.Setup(s), "Setup must not error")
	m.setState(connectedState)
	// The first setup fails to scale and the second, flushed after it, fails to generate its subscriptions
	for _, setup := range []struct {
		name                    string
		connectErr, generateErr error
	}{
		{name: "scaling", connectErr: errDastardlyReason},
		{name: "generation", generateErr: errSubscriptionRejected},
	} {
		require.NoErrorf(t, m.SetupNewConnection(&ConnectionSetup{
			URL:       "wss://" + setup.name + ".invalid",
			Connector: func(context.Context, Connection) error { return setup.connectErr },
			GenerateSubscriptions: func() (subscription.List, error) {
				return subscription.List{{Channel: setup.name}}, setup.generateErr
			},
			Subscriber:   func(context.Context, Connection, subscription.List) error { return nil },
			Unsubscriber: func(context.Context, Connection, subscription.List) error { return nil },
			Handler:      func(context.Context, Connection, []byte) error { return nil },
		}), "SetupNewConnection must not error for %s", setup.name)
	}
	err := m.flushChannels(t.Context())
	require.ErrorIs(t, err, errSubscriptionRejected, "flushChannels must report the generation error")
	assert.ErrorIs(t, err, errDastardlyReason, "flushChannels should keep the scaling error of a connection flushed before the generation error")
}

func TestFlushChannelsReturnsShutdownError(t *testing.T) {
	t.Parallel()
	m := NewManager()
	m.setEnabled(true)
	m.setState(connectedState)
	conn := &fakeConnection{subscriptions: subscription.NewStore()}
	m.Conn = conn
	require.ErrorIs(t, m.flushChannels(t.Context()), common.ErrTypeAssertFailure, "flushChannels must return the error shutting down to reconnect")
	assert.True(t, conn.shutdownCalled, "flushChannels should shut the connection down when the exchange cannot subscribe and unsubscribe")
}
