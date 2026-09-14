package websocket

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Public subscription errors
var (
	ErrSubscriptionFailure     = errors.New("subscription failure")
	ErrSubscriptionsNotAdded   = errors.New("subscriptions not added")
	ErrSubscriptionsNotRemoved = errors.New("subscriptions not removed")
)

// Public subscription errors
var (
	errSubscriptionsExceedsLimit = errors.New("subscriptions exceeds limit")
	errConnectionNotFound        = errors.New("connection not found")
)

// UnsubscribeChannels unsubscribes from a list of websocket channel
func (m *Manager) UnsubscribeChannels(ctx context.Context, conn Connection, channels subscription.List) error {
	if len(channels) == 0 {
		return nil // No channels to unsubscribe from is not an error
	}

	if m.useMultiConnectionManagement {
		if err := common.NilGuard(conn); err != nil {
			return err
		}
		ws, ok := m.managedWebsocket(conn)
		if !ok {
			return fmt.Errorf("%w: %q", errConnectionNotFound, conn.GetURL())
		}
		return m.unsubscribe(ws.subscriptions, channels, func(channels subscription.List) error {
			return ws.setup.Unsubscriber(ctx, conn, channels)
		})
	}

	if m.Unsubscriber == nil {
		return fmt.Errorf("%w: Global Unsubscriber not set", common.ErrNilPointer)
	}

	return m.unsubscribe(m.subscriptions, channels, func(channels subscription.List) error {
		return m.Unsubscriber(channels)
	})
}

func (m *Manager) unsubscribe(store *subscription.Store, channels subscription.List, unsub func(channels subscription.List) error) error {
	if store == nil {
		return nil // No channels to unsubscribe from is not an error
	}
	for _, s := range channels {
		if store.Get(s) == nil {
			return fmt.Errorf("%w: %s", subscription.ErrNotFound, s)
		}
	}
	return unsub(channels)
}

func (m *Manager) managedWebsocket(conn Connection) (*websocket, bool) {
	if conn == nil {
		return nil, false
	}
	m.connectionManagerMu.RLock()
	defer m.connectionManagerMu.RUnlock()
	ws := m.connections[conn]
	return ws, ws != nil
}

func (m *Manager) subscriptionStore(conn Connection) *subscription.Store {
	m.connectionManagerMu.RLock()
	defer m.connectionManagerMu.RUnlock()
	if ws, ok := m.connections[conn]; ok && conn != nil {
		return ws.subscriptions
	}
	return m.subscriptions
}

func (m *Manager) initSubscriptionStore(conn Connection) *subscription.Store {
	m.connectionManagerMu.Lock()
	defer m.connectionManagerMu.Unlock()
	if ws, ok := m.connections[conn]; ok && conn != nil {
		if ws.subscriptions == nil {
			ws.subscriptions = subscription.NewStore()
		}
		return ws.subscriptions
	}
	if m.subscriptions == nil {
		m.subscriptions = subscription.NewStore()
	}
	return m.subscriptions
}

type resubscribeTracker struct {
	done      chan struct{}
	err       error
	closeOnce sync.Once
}

// ResubscribeToChannel resubscribes to channel
// Sets state to Resubscribing, and exchanges which want to maintain a lock on it can respect this state and not RemoveSubscription.
// A subscription already in ResubscribingState is retried.
func (m *Manager) ResubscribeToChannel(ctx context.Context, conn Connection, s *subscription.Subscription) error {
	return m.resubscribeToChannel(ctx, conn, s, true)
}

func (m *Manager) resubscribeToChannel(ctx context.Context, conn Connection, s *subscription.Subscription, allowRetry bool) error {
	if s == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	m.resubscriptionsMu.Lock()
	if m.resubscriptions == nil {
		m.resubscriptions = make(map[*subscription.Subscription]*resubscribeTracker)
	}
	tracker, inFlight := m.resubscriptions[s]
	if !inFlight {
		tracker = &resubscribeTracker{
			done: make(chan struct{}),
		}
		m.resubscriptions[s] = tracker
	}
	m.resubscriptionsMu.Unlock()
	if inFlight {
		if m.resubscribeWaiterHook != nil {
			m.resubscribeWaiterHook(s)
		}
		<-tracker.done
		if tracker.err == nil {
			return nil
		}
		if allowRetry {
			return m.resubscribeToChannel(ctx, conn, s, false)
		}
		return tracker.err
	}
	var resErr error
	defer func() {
		finishResubscriptionTracker(m, s, tracker, resErr)
	}()
	if m.resubscribePreLockHook != nil {
		m.resubscribePreLockHook(s)
	}
	m.m.Lock()
	l := subscription.List{s}
	setResubscribingState(l)
	wsStore := m.subscriptionStore(conn)
	connStore := connectionSubscriptionStore(conn)
	var origKey any
	if wsStore != nil {
		if origSub := wsStore.Get(s); origSub != nil {
			origKey = origSub.Key
		}
	}
	m.m.Unlock()
	if err := m.UnsubscribeChannels(ctx, conn, l); err != nil {
		resErr = err
		return err
	}
	if err := m.SubscribeToChannels(ctx, conn, l); err != nil {
		m.m.Lock()
		// Once Shutdown or a scale-down has untracked the connection its subscriptions are gone; restoring would resurrect one
		if m.subscriptionStore(conn) == wsStore {
			restoreFailedRecovery(wsStore, connStore, s, origKey)
		}
		m.m.Unlock()
		resErr = err
		return err
	}

	return nil
}

type recoverySnapshot struct {
	sub     *subscription.Subscription
	origKey any
}

func connectionSubscriptionStore(conn Connection) *subscription.Store {
	if conn == nil {
		return nil
	}
	return conn.Subscriptions()
}

func finishResubscriptionTracker(m *Manager, s *subscription.Subscription, tracker *resubscribeTracker, err error) {
	if tracker == nil {
		return
	}
	m.resubscriptionsMu.Lock()
	defer m.resubscriptionsMu.Unlock()
	tracker.err = err
	if currentTracker, ok := m.resubscriptions[s]; ok && currentTracker == tracker {
		delete(m.resubscriptions, s)
	}
	tracker.closeOnce.Do(func() {
		close(tracker.done)
	})
}

func restoreFailedRecovery(wsStore, connStore *subscription.Store, sub *subscription.Subscription, origKey any) {
	if sub == nil {
		return
	}
	if connStore != nil {
		restoreSubscriptionAfterFailedRecovery(connStore, sub, origKey)
	}
	if wsStore != nil && wsStore != connStore {
		restoreSubscriptionAfterFailedRecovery(wsStore, sub, origKey)
	}
}

func connectionUsedCapacity(store *subscription.Store, incoming subscription.List) int {
	if store == nil {
		return 0
	}

	usedCap := store.Len()
	discounted := make(map[*subscription.Subscription]struct{}, len(incoming))
	for _, s := range incoming {
		if s == nil || s.State() != subscription.ResubscribingState || store.Get(s) == nil {
			continue
		}
		if _, seen := discounted[s]; seen {
			continue
		}
		discounted[s] = struct{}{}
		usedCap--
	}
	return usedCap
}

func hasLiveRecoveryReplacement(store *subscription.Store, sub *subscription.Subscription, origKey any) bool {
	if store == nil || sub == nil {
		return false
	}
	want := subscription.ExactKey{Subscription: sub}
	for _, existing := range store.List() {
		if existing == sub {
			continue
		}
		if !want.Match(subscription.ExactKey{Subscription: existing}) {
			continue
		}
		if existing.State() != subscription.SubscribedState {
			continue
		}
		if origKey != nil && existing.EnsureKeyed() == origKey {
			continue
		}
		return true
	}
	return false
}

func restoreSubscriptionAfterFailedRecovery(store *subscription.Store, sub *subscription.Subscription, origKey any) {
	if store == nil || sub == nil {
		return
	}
	if hasLiveRecoveryReplacement(store, sub, origKey) {
		// Get matches a default key by ExactKey and can return the replacement, so only remove the stale pointer itself
		if store.Get(sub) == sub {
			_ = store.Remove(sub)
		}
		if origKey != nil && store.Get(origKey) == sub {
			_ = store.Remove(origKey)
		}
		return
	}
	if origKey != nil {
		sub.SetKey(origKey)
	}
	_ = sub.SetState(subscription.ResubscribingState)
	if store.Get(sub) == nil {
		_ = store.Add(sub)
	}
}

// dropFailedRecoveries removes subscriptions a failed recovery left in ResubscribingState, so a flush treats them as
// absent and resubscribes them if they are still wanted. Exchanges read ResubscribingState as a recovery in flight,
// so a restored entry must not outlive the next flush.
func (m *Manager) dropFailedRecoveries() {
	stores := []*subscription.Store{m.subscriptions}
	for _, conn := range []Connection{m.Conn, m.AuthConn} {
		if conn != nil {
			stores = append(stores, conn.Subscriptions())
		}
	}
	for _, ws := range m.snapshotConnectionManager() {
		stores = append(stores, ws.subscriptions)
		for _, conn := range m.snapshotManagedConnections(ws) {
			stores = append(stores, conn.Subscriptions())
		}
	}
	m.resubscriptionsMu.Lock()
	defer m.resubscriptionsMu.Unlock()
	failed := make(map[*subscription.Subscription]struct{})
	for _, store := range stores {
		if store == nil {
			continue
		}
		for _, s := range store.List() {
			if _, inFlight := m.resubscriptions[s]; !inFlight && s.State() == subscription.ResubscribingState {
				failed[s] = struct{}{}
			}
		}
	}
	for s := range failed {
		for _, store := range stores {
			if store.Get(s) == s {
				_ = store.Remove(s)
			}
		}
		_ = s.SetState(subscription.UnsubscribedState)
	}
}

func setResubscribingState(subs subscription.List) {
	for _, sub := range subs {
		if sub.State() != subscription.ResubscribingState {
			_ = sub.SetState(subscription.ResubscribingState)
		}
	}
}

// SubscribeToChannels subscribes to websocket channels using the exchange specific Subscriber method
// Errors are returned for duplicates or exceeding max Subscriptions
func (m *Manager) SubscribeToChannels(ctx context.Context, conn Connection, subs subscription.List) error {
	if slices.Contains(subs, nil) {
		return fmt.Errorf("%w: List parameter contains an nil element", common.ErrNilPointer)
	}
	if err := m.checkSubscriptions(conn, subs); err != nil {
		return err
	}

	if ws, ok := m.managedWebsocket(conn); ok {
		return ws.setup.Subscriber(ctx, conn, subs)
	}

	if m.Subscriber == nil {
		return fmt.Errorf("%w: Global Subscriber not set", common.ErrNilPointer)
	}

	if err := m.Subscriber(subs); err != nil {
		return fmt.Errorf("%w: %w", ErrSubscriptionFailure, err)
	}

	return nil
}

// AddSubscriptions adds subscriptions to the subscription store
// Sets state to Subscribing unless the state is already set
func (m *Manager) AddSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if m == nil {
		return fmt.Errorf("%w: AddSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	subscriptionStore := m.initSubscriptionStore(conn)

	var errs error
	for _, s := range subs {
		if s.State() == subscription.InactiveState {
			if err := s.SetState(subscription.SubscribingState); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
			}
		}
		if err := subscriptionStore.Add(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// AddSuccessfulSubscriptions marks subscriptions as subscribed and adds them to the subscription store
func (m *Manager) AddSuccessfulSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if m == nil {
		return fmt.Errorf("%w: AddSuccessfulSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	subscriptionStore := m.initSubscriptionStore(conn)

	var errs error
	for _, s := range subs {
		alreadyTracked := s.State() == subscription.ResubscribingState && subscriptionStore.Get(s) == s
		if err := s.SetState(subscription.SubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if !alreadyTracked {
			if err := subscriptionStore.Add(s); err != nil {
				errs = common.AppendError(errs, err)
			}
		}
	}
	return errs
}

// RemoveSubscriptions removes subscriptions from the subscription list and sets the status to Unsubscribed
func (m *Manager) RemoveSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if m == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	subscriptionStore := m.subscriptionStore(conn)

	if subscriptionStore == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on uninitialised Websocket", common.ErrNilPointer)
	}

	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.UnsubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if err := subscriptionStore.Remove(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// GetSubscription returns a subscription at the key provided
// returns nil if no subscription is at that key or the key is nil
// Keys can implement subscription.MatchableKey in order to provide custom matching logic
func (m *Manager) GetSubscription(key any) *subscription.Subscription {
	if m == nil || key == nil {
		return nil
	}

	for _, ws := range m.snapshotConnectionManager() {
		m.connectionManagerMu.RLock()
		store := ws.subscriptions
		var sub *subscription.Subscription
		if store != nil {
			sub = store.Get(key)
		}
		m.connectionManagerMu.RUnlock()
		if sub != nil {
			return sub
		}
	}
	if store := m.subscriptionStore(nil); store != nil {
		return store.Get(key)
	}

	return nil
}

// GetSubscriptions returns a new slice of the subscriptions
func (m *Manager) GetSubscriptions() subscription.List {
	if m == nil {
		return nil
	}
	var subs subscription.List
	for _, ws := range m.snapshotConnectionManager() {
		m.connectionManagerMu.RLock()
		store := ws.subscriptions
		if store != nil {
			subs = append(subs, store.List()...)
		}
		m.connectionManagerMu.RUnlock()
	}
	if store := m.subscriptionStore(nil); store != nil {
		subs = append(subs, store.List()...)
	}
	return subs
}

// checkSubscriptions checks subscriptions against the max subscription limit and if the subscription already exists
// The subscription state is not considered when counting existing subscriptions
func (m *Manager) checkSubscriptions(conn Connection, subs subscription.List) error {
	var subscriptionStore *subscription.Store
	var connSubStore *subscription.Store
	if ws, ok := m.managedWebsocket(conn); ok {
		if ws.subscriptions == nil {
			return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
		}
		for _, c := range m.snapshotManagedConnections(ws) { // ensure connection is actually managed
			if c == conn {
				connSubStore = c.Subscriptions()
				break
			}
		}
		if connSubStore == nil {
			return fmt.Errorf("%w: connection subscription store not found", common.ErrNilPointer)
		}
		subscriptionStore = ws.subscriptions
	} else {
		subscriptionStore = m.subscriptionStore(nil)
		if subscriptionStore == nil {
			return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
		}
		connSubStore = subscriptionStore
	}
	if m.MaxSubscriptionsPerConnection > 0 {
		usedCap := connectionUsedCapacity(connSubStore, subs)
		if usedCap+len(subs) > m.MaxSubscriptionsPerConnection {
			return fmt.Errorf("%w: current subscriptions: %v, incoming subscriptions: %v, max subscriptions per connection: %v",
				errSubscriptionsExceedsLimit,
				usedCap,
				len(subs),
				m.MaxSubscriptionsPerConnection)
		}
	}

	for _, s := range subs {
		if s.State() == subscription.ResubscribingState {
			continue
		}
		if found := subscriptionStore.Get(s); found != nil {
			return fmt.Errorf("%w: %s", subscription.ErrDuplicate, s)
		}
	}

	return nil
}

// FlushChannels flushes channel subscriptions when there is a pair/asset change
func (m *Manager) FlushChannels(ctx context.Context) error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.flushChannels(ctx)
}

func (m *Manager) flushChannels(ctx context.Context) error {
	if !m.IsEnabled() {
		return fmt.Errorf("%s %w", m.exchangeName, ErrWebsocketNotEnabled)
	}

	if !m.IsConnected() {
		return fmt.Errorf("%s %w", m.exchangeName, ErrNotConnected)
	}

	// If the exchange does not support subscribing and or unsubscribing the full connection needs to be flushed to
	// maintain consistency.
	if !m.features.Subscribe || !m.features.Unsubscribe {
		if err := m.shutdown(); err != nil {
			return err
		}
		return m.connect(ctx)
	}

	m.dropFailedRecoveries()

	if !m.useMultiConnectionManagement {
		newSubs, err := m.GenerateSubs()
		if err != nil {
			return err
		}
		return m.updateChannelSubscriptions(ctx, m.subscriptions, newSubs)
	}

	for _, ws := range m.snapshotConnectionManager() {
		if ws.setup.SubscriptionsNotRequired {
			continue
		}

		newSubs, err := ws.setup.GenerateSubscriptions()
		if err != nil {
			return err
		}

		// Case if there is nothing to unsubscribe from and the connection is nil
		if len(newSubs) == 0 && len(m.snapshotManagedConnections(ws)) == 0 {
			continue
		}

		if err := m.scaleConnectionsToSubscriptions(ctx, ws, newSubs); err != nil {
			return err
		}
	}

	return nil
}

// updateChannelSubscriptions subscribes or unsubscribes from channels and checks that the correct number of channels
// have been subscribed to or unsubscribed from.
func (m *Manager) updateChannelSubscriptions(ctx context.Context, store *subscription.Store, incoming subscription.List) error {
	subs, unsubs := store.Diff(incoming)
	if len(unsubs) != 0 {
		if err := m.UnsubscribeChannels(ctx, nil, unsubs); err != nil {
			return err
		}

		if contained := store.Contained(unsubs); len(contained) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotRemoved, contained)
		}
	}
	if len(subs) != 0 {
		if err := m.SubscribeToChannels(ctx, nil, subs); err != nil {
			return err
		}

		if missing := store.Missing(subs); len(missing) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
		}
	}

	return nil
}

// applyTrackedSubscriptions records tracked subscriptions in both manager-level and connection-level stores for the provided connection.
func (m *Manager) applyTrackedSubscriptions(conn Connection, tracked subscription.List) error {
	if len(tracked) == 0 {
		return nil
	}
	if err := m.AddSuccessfulSubscriptions(conn, tracked...); err != nil {
		return err
	}
	store := conn.Subscriptions()
	if err := common.NilGuard(store); err != nil {
		return fmt.Errorf("websocket connection %w", err)
	}
	for _, sub := range tracked {
		if err := store.Add(sub); err != nil {
			return err
		}
	}

	return nil
}

// absorbTrackedSubscriptions asks each managed connection whether a subset of subs can be logically tracked on that existing connection.
// It applies tracked subscriptions to manager and connection stores and returns the remaining subscriptions that still need outbound subscribe traffic/new capacity plus the list that were tracked.
func (m *Manager) absorbTrackedSubscriptions(ctx context.Context, ws *websocket, subs subscription.List) (remaining, tracked subscription.List, err error) {
	if len(subs) == 0 || ws == nil || ws.setup == nil || ws.setup.TrackOnExistingConnection == nil {
		return subs, nil, nil
	}
	connections := m.snapshotManagedConnections(ws)
	if len(connections) == 0 {
		return subs, nil, nil
	}

	remaining = subs
	tracked = make(subscription.List, 0, len(subs))
	for _, conn := range connections {
		connRemaining, connTracked, err := ws.setup.TrackOnExistingConnection(ctx, conn, remaining)
		if err != nil {
			return nil, nil, err
		}
		if err := m.applyTrackedSubscriptions(conn, connTracked); err != nil {
			return nil, nil, err
		}
		if len(connTracked) != 0 {
			tracked = append(tracked, connTracked...)
		}
		remaining = connRemaining
		if len(remaining) == 0 {
			return nil, tracked, nil
		}
	}
	return remaining, tracked, nil
}

// absorbTrackableSubscriptionsAndValidate absorbs trackable subscriptions onto existing connections and verifies that tracked subscriptions were recorded in the websocket-level subscription store.
func (m *Manager) absorbTrackableSubscriptionsAndValidate(ctx context.Context, ws *websocket, subs subscription.List) (subscription.List, error) {
	remaining, tracked, err := m.absorbTrackedSubscriptions(ctx, ws, subs)
	if err != nil {
		return nil, err
	}
	if len(tracked) == 0 {
		return remaining, nil
	}
	if missing := ws.subscriptions.Missing(tracked); len(missing) > 0 {
		return nil, fmt.Errorf("%w: %w %q", ErrSubscriptionFailure, ErrSubscriptionsNotAdded, missing)
	}
	return remaining, nil
}

// scaleConnectionsToSubscriptions scales connections to subscriptions based off current subscription list and subscription limit
func (m *Manager) scaleConnectionsToSubscriptions(ctx context.Context, ws *websocket, incoming subscription.List) error {
	if err := common.NilGuard(ws); err != nil {
		return err
	}
	subs, unsubs := ws.subscriptions.Diff(incoming)
	if len(unsubs) != 0 {
		currentUnsubs := slices.Clone(unsubs)
		// Unsubscribe first to free up capacity on existing connections
		for _, conn := range m.snapshotManagedConnections(ws) {
			leftOver, err := m.unsubscribeFromConnection(ctx, conn, currentUnsubs)
			if err != nil {
				return err
			}
			currentUnsubs = leftOver
			if len(currentUnsubs) == 0 {
				break
			}
		}

		if len(currentUnsubs) != 0 {
			log.Warnf(log.WebsocketMgr, "%v websocket: unable to find all subscriptions to remove on existing connections, attempting global unsubscribe for %v", m.exchangeName, currentUnsubs)
			for _, s := range currentUnsubs {
				if err := ws.subscriptions.Remove(s); err != nil {
					return err
				}
			}
		}
		if contained := ws.subscriptions.Contained(unsubs); len(contained) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotRemoved, contained)
		}
	}
	if len(subs) != 0 {
		// First, absorb subscriptions that should be tracked on existing
		// connections (e.g. OKX spot/margin equivalents) before the
		// generic capacity-based routing can misplace them.
		currentSubs, err := m.absorbTrackableSubscriptionsAndValidate(ctx, ws, subs)
		if err != nil {
			return err
		}

		// Subscribe to existing connections to use up existing capacity
		for _, conn := range m.snapshotManagedConnections(ws) {
			leftOver, err := m.subscribeToConnection(ctx, conn, currentSubs)
			if err != nil {
				return err
			}
			currentSubs = leftOver
			if len(currentSubs) == 0 {
				break
			}
		}

		// Spawn new connections if there are still subscriptions left to process
		for _, batch := range common.Batch(currentSubs, m.MaxSubscriptionsPerConnection) {
			toConnect, err := m.absorbTrackableSubscriptionsAndValidate(ctx, ws, batch)
			if err != nil {
				return err
			}
			if len(toConnect) == 0 {
				continue
			}
			if err := m.createConnectAndSubscribe(ctx, ws, toConnect); err != nil {
				return err
			}
		}

		if missing := ws.subscriptions.Missing(subs); len(missing) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
		}
	}

	// Clean up any connections that have no subscriptions left to reduce resource usage
	connections := m.snapshotManagedConnections(ws)
	clean := make([]Connection, 0, len(connections))
	stale := make([]Connection, 0, len(connections))
	for _, conn := range connections {
		if conn.Subscriptions().Len() != 0 {
			clean = append(clean, conn)
			continue
		}
		stale = append(stale, conn)
	}
	if len(stale) == 0 {
		return nil
	}
	staleSet := make(map[Connection]struct{}, len(stale))
	for _, conn := range stale {
		staleSet[conn] = struct{}{}
	}
	m.connectionManagerMu.Lock()
	for _, conn := range stale {
		delete(m.connections, conn)
	}
	clean = clean[:0]
	for _, conn := range ws.connections {
		if _, ok := staleSet[conn]; ok {
			continue
		}
		clean = append(clean, conn)
	}
	ws.connections = clean
	m.connectionManagerMu.Unlock()
	for _, conn := range stale {
		if err := conn.Shutdown(); err != nil {
			log.Warnf(log.WebsocketMgr, "%v websocket: failed to shutdown connection: %v", m.exchangeName, err)
		}
	}

	return nil
}

// ResubscribeFromConnection unsubscribes and resubscribes to a subscription on a connection
func (m *Manager) ResubscribeFromConnection(ctx context.Context, conn Connection, subs subscription.List) error {
	return m.resubscribeFromConnection(ctx, conn, subs, true)
}

func (m *Manager) resubscribeFromConnection(ctx context.Context, conn Connection, subs subscription.List, allowRetry bool) error {
	if err := common.NilGuard(conn, subs); err != nil {
		return err
	}
	if len(subs) == 0 {
		return nil
	}
	m.resubscriptionsMu.Lock()
	if m.resubscriptions == nil {
		m.resubscriptions = make(map[*subscription.Subscription]*resubscribeTracker)
	}
	ownedSubs := make(subscription.List, 0, len(subs))
	ownedTrackers := make([]*resubscribeTracker, 0, len(subs))
	borrowed := make([]*resubscribeTracker, 0, len(subs))
	for _, s := range subs {
		tracker := m.resubscriptions[s]
		if tracker != nil {
			borrowed = append(borrowed, tracker)
			continue
		}
		tracker = &resubscribeTracker{
			done: make(chan struct{}),
		}
		m.resubscriptions[s] = tracker
		ownedSubs = append(ownedSubs, s)
		ownedTrackers = append(ownedTrackers, tracker)
	}
	m.resubscriptionsMu.Unlock()
	if len(ownedSubs) == 0 {
		if m.resubscribeWaiterHook != nil {
			m.resubscribeWaiterHook(subs[0])
		}
		var firstErr error
		for _, tracker := range borrowed {
			<-tracker.done
			if tracker.err != nil && firstErr == nil {
				firstErr = tracker.err
			}
		}
		if firstErr == nil {
			return nil
		}
		if allowRetry {
			return m.resubscribeFromConnection(ctx, conn, subs, false)
		}
		return firstErr
	}
	// The subscriber is handed subs and may reorder it; ownedSubs stays paired with ownedTrackers
	subs = slices.Clone(ownedSubs)
	var resErr error
	defer func() {
		for idx, s := range ownedSubs {
			finishResubscriptionTracker(m, s, ownedTrackers[idx], resErr)
		}
	}()
	m.m.Lock()
	wsStore := m.subscriptionStore(conn)
	connStore := connectionSubscriptionStore(conn)
	snapshots := make([]recoverySnapshot, len(subs))
	for i, s := range subs {
		snapshots[i] = recoverySnapshot{sub: s}
		if connStore != nil {
			if orig := connStore.Get(s); orig != nil {
				snapshots[i].origKey = orig.Key
			}
		} else if wsStore != nil {
			if orig := wsStore.Get(s); orig != nil {
				snapshots[i].origKey = orig.Key
			}
		}
	}
	setResubscribingState(subs)
	m.m.Unlock()
	missing, err := m.unsubscribeFromConnection(ctx, conn, subs)
	if err != nil {
		resErr = err
		return err
	}
	if len(missing) > 0 {
		var notRemoved subscription.List
		for _, s := range missing {
			if s.State() != subscription.UnsubscribedState {
				notRemoved = append(notRemoved, s)
			}
		}
		if len(notRemoved) > 0 {
			resErr = fmt.Errorf("%w: %q", ErrSubscriptionsNotRemoved, notRemoved)
			return resErr
		}
	}
	remaining, err := m.subscribeToConnection(ctx, conn, subs)
	if err != nil {
		m.m.Lock()
		// Once Shutdown or a scale-down has untracked the connection its subscriptions are gone; restoring would resurrect them
		if m.subscriptionStore(conn) == wsStore {
			for _, snap := range snapshots {
				if snap.sub.State() == subscription.SubscribedState {
					continue
				}
				restoreFailedRecovery(wsStore, connStore, snap.sub, snap.origKey)
			}
		}
		m.m.Unlock()
		resErr = err
		return err
	}
	if len(remaining) > 0 {
		resErr = fmt.Errorf("%w: %q", ErrSubscriptionsNotAdded, remaining)
		return resErr
	}

	return nil
}

func allSubscriptionsState(subs subscription.List, state subscription.State) bool {
	for _, sub := range subs {
		if sub.State() != state {
			return false
		}
	}
	return true
}

// unsubscribeFromConnection unsubscribes for a connection and removes subscriptions from the connection's store
func (m *Manager) unsubscribeFromConnection(ctx context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
	store := conn.Subscriptions()
	if err := common.NilGuard(store); err != nil {
		return nil, fmt.Errorf("websocket connection %w", err)
	}

	remove := store.Contained(subs)
	if len(remove) == 0 {
		return subs, nil
	}

	if err := m.UnsubscribeChannels(ctx, conn, remove); err != nil {
		return nil, err
	}

	missing := store.Missing(subs)
	for _, r := range remove {
		if store.Get(r) == nil {
			continue
		}
		if err := store.Remove(r); err != nil {
			return nil, err
		}
	}
	return missing, nil
}

// subscribeToConnection subscribes for a connection and adds subscriptions to the connection's store
func (m *Manager) subscribeToConnection(ctx context.Context, conn Connection, subs subscription.List) (subscription.List, error) {
	store := conn.Subscriptions()
	if err := common.NilGuard(store); err != nil {
		return nil, fmt.Errorf("websocket connection %w", err)
	}
	usedCap := connectionUsedCapacity(store, subs)
	if m.MaxSubscriptionsPerConnection > 0 && usedCap >= m.MaxSubscriptionsPerConnection {
		return subs, nil // No capacity left for this connection
	}

	availableCap := len(subs)
	if m.MaxSubscriptionsPerConnection > 0 {
		availableCap = m.MaxSubscriptionsPerConnection - usedCap
	}

	if availableCap > len(subs) {
		availableCap = len(subs)
	}

	toSubscribe := subs[:availableCap]
	tracked := make(map[*subscription.Subscription]bool, len(toSubscribe))
	for _, s := range toSubscribe {
		tracked[s] = store.Get(s) != nil && s.State() == subscription.ResubscribingState
	}
	if err := m.SubscribeToChannels(ctx, conn, toSubscribe); err != nil {
		return nil, err
	}

	for _, s := range toSubscribe {
		if tracked[s] {
			continue
		}
		if err := store.Add(s); err != nil {
			return nil, err
		}
	}

	return subs[availableCap:], nil
}
