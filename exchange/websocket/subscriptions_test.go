package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler) }))
	t.Cleanup(mock.Close)
	t.Cleanup(cleanupW)

	w.subscriptions = subscription.NewStore()

	amazingCandidate := &ConnectionSetup{
		URL: "ws" + mock.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			return conn.Dial(ctx, gws.DefaultDialer, nil, nil)
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
}

func (f *fakeConnection) Shutdown() error {
	f.shutdownCalled = true
	return nil
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

func TestTrackOnExistingConnection(t *testing.T) {
	t.Parallel()

	t.Run("PassthroughWithoutTrackHook", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		subs := subscription.List{{Channel: "A"}}
		ws := &websocket{
			setup:       &ConnectionSetup{},
			connections: []Connection{&fakeConnection{subscriptions: subscription.NewStore()}},
		}

		remaining, err := m.trackOnExistingConnection(t.Context(), ws, subs)
		require.NoError(t, err)
		assert.Equal(t, subs, remaining)
	})

	t.Run("TracksAcrossConnectionsUntilEmpty", func(t *testing.T) {
		t.Parallel()
		m := NewManager()
		tracked := &subscription.Subscription{Channel: "tracked"}
		conn0 := &fakeConnection{subscriptions: subscription.NewStore()}
		conn1 := &fakeConnection{subscriptions: subscription.NewStore()}
		ws := &websocket{
			setup: &ConnectionSetup{
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
					if conn != conn1 {
						return subs, nil
					}
					require.NoError(t, m.AddSuccessfulSubscriptions(conn, tracked))
					require.NoError(t, conn.Subscriptions().Add(tracked))
					return nil, nil
				},
			},
			subscriptions: subscription.NewStore(),
			connections:   []Connection{conn0, conn1},
		}
		m.connections[conn0] = ws
		m.connections[conn1] = ws

		remaining, err := m.trackOnExistingConnection(t.Context(), ws, subscription.List{tracked})
		require.NoError(t, err)
		require.Nil(t, remaining)
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
				TrackOnExistingConnection: func(context.Context, Connection, subscription.List) (subscription.List, error) {
					return nil, expectedErr
				},
			},
			connections: []Connection{&fakeConnection{subscriptions: subscription.NewStore()}},
		}

		remaining, err := m.trackOnExistingConnection(t.Context(), ws, subscription.List{{Channel: "A"}})
		require.ErrorIs(t, err, expectedErr)
		assert.Nil(t, remaining)
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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
		}))
		t.Cleanup(srv.Close)

		ws := &websocket{
			setup: &ConnectionSetup{
				URL: "ws" + srv.URL[len("http"):] + "/ws",
				Connector: func(ctx context.Context, c Connection) error {
					return c.Dial(ctx, gws.DefaultDialer, nil, nil)
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
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
					if len(subs) != 1 || subs[0].Channel != logical.Channel {
						return subs, nil
					}
					require.NoError(t, m.AddSuccessfulSubscriptions(conn, logical))
					require.NoError(t, conn.Subscriptions().Add(logical))
					return nil, nil
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
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
					// Only track when the connection owns subB (the inverse).
					if conn.Subscriptions().Get(subB) == nil {
						return subs, nil
					}
					trackedOnConn = conn
					require.NoError(t, m.AddSuccessfulSubscriptions(conn, logical))
					require.NoError(t, conn.Subscriptions().Add(logical))
					return nil, nil
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
		// it has capacity. The fix ensures trackOnExistingConnection
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
				TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
					// Only track when the connection owns subB (the inverse).
					if conn.Subscriptions().Get(subB) == nil {
						return subs, nil
					}
					var remaining subscription.List
					for _, s := range subs {
						if s.Channel != logical.Channel {
							remaining = append(remaining, s)
							continue
						}
						trackedOnConn = conn
						require.NoError(t, m.AddSuccessfulSubscriptions(conn, s))
						require.NoError(t, conn.Subscriptions().Add(s))
					}
					return remaining, nil
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
	m.setEnabled(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mockws.WsMockUpgrader(t, w, r, mockws.EchoHandler)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { cleanupManagerMonitors(t, m) })

	subA := &subscription.Subscription{Channel: "A"}
	subB := &subscription.Subscription{Channel: "B"}
	var connectorCalls int
	var trackCalls int
	require.NoError(t, m.SetupNewConnection(&ConnectionSetup{
		URL: "ws" + srv.URL[len("http"):] + "/ws",
		Connector: func(ctx context.Context, conn Connection) error {
			connectorCalls++
			return conn.Dial(ctx, gws.DefaultDialer, nil, nil)
		},
		GenerateSubscriptions: func() (subscription.List, error) {
			return subscription.List{subA, subB}, nil
		},
		Subscriber: func(_ context.Context, c Connection, s subscription.List) error {
			return m.AddSuccessfulSubscriptions(c, s...)
		},
		TrackOnExistingConnection: func(_ context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
			if len(subs) != 1 || subs[0] != subB {
				return subs, nil
			}
			trackCalls++
			if err := m.AddSuccessfulSubscriptions(conn, subB); err != nil {
				return nil, err
			}
			if err := conn.Subscriptions().Add(subB); err != nil {
				return nil, err
			}
			return nil, nil
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
}
