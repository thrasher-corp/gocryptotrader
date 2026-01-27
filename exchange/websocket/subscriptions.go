package websocket

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
		ws, ok := m.connections[conn]
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

// ResubscribeToChannel resubscribes to channel
// Sets state to Resubscribing, and exchanges which want to maintain a lock on it can respect this state and not RemoveSubscription
// Errors if subscription is already subscribing
func (m *Manager) ResubscribeToChannel(ctx context.Context, conn Connection, s *subscription.Subscription) error {
	l := subscription.List{s}
	if err := s.SetState(subscription.ResubscribingState); err != nil {
		return fmt.Errorf("%w: %s", err, s)
	}
	if err := m.UnsubscribeChannels(ctx, conn, l); err != nil {
		return err
	}
	return m.SubscribeToChannels(ctx, conn, l)
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

	if ws, ok := m.connections[conn]; ok && conn != nil {
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
	var subscriptionStore **subscription.Store
	if ws, ok := m.connections[conn]; ok && conn != nil {
		subscriptionStore = &ws.subscriptions
	} else {
		subscriptionStore = &m.subscriptions
	}

	if *subscriptionStore == nil {
		*subscriptionStore = subscription.NewStore()
	}
	var errs error
	for _, s := range subs {
		if s.State() == subscription.InactiveState {
			if err := s.SetState(subscription.SubscribingState); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
			}
		}
		if err := (*subscriptionStore).Add(s); err != nil {
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

	var subscriptionStore **subscription.Store
	if ws, ok := m.connections[conn]; ok && conn != nil {
		subscriptionStore = &ws.subscriptions
	} else {
		subscriptionStore = &m.subscriptions
	}

	if *subscriptionStore == nil {
		*subscriptionStore = subscription.NewStore()
	}

	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.SubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if err := (*subscriptionStore).Add(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// RemoveSubscriptions removes subscriptions from the subscription list and sets the status to Unsubscribed
func (m *Manager) RemoveSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if m == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on nil Websocket", common.ErrNilPointer)
	}

	var subscriptionStore *subscription.Store
	if ws, ok := m.connections[conn]; ok && conn != nil {
		subscriptionStore = ws.subscriptions
	} else {
		subscriptionStore = m.subscriptions
	}

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
	for _, c := range m.connectionManager {
		if c.subscriptions == nil {
			continue
		}
		sub := c.subscriptions.Get(key)
		if sub != nil {
			return sub
		}
	}
	if m.subscriptions == nil {
		return nil
	}
	return m.subscriptions.Get(key)
}

// GetSubscriptions returns a new slice of the subscriptions
func (m *Manager) GetSubscriptions() subscription.List {
	if m == nil {
		return nil
	}
	var subs subscription.List
	for _, c := range m.connectionManager {
		if c.subscriptions != nil {
			subs = append(subs, c.subscriptions.List()...)
		}
	}
	if m.subscriptions != nil {
		subs = append(subs, m.subscriptions.List()...)
	}
	return subs
}

// checkSubscriptions checks subscriptions against the max subscription limit and if the subscription already exists
// The subscription state is not considered when counting existing subscriptions
func (m *Manager) checkSubscriptions(conn Connection, subs subscription.List) error {
	var subscriptionStore *subscription.Store
	var usedCapacity int
	if ws, ok := m.connections[conn]; ok && conn != nil {
		if ws.subscriptions == nil {
			return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
		}
		var connSubStore *subscription.Store
		for _, c := range ws.connections { // ensure connection is actually managed
			if c == conn {
				connSubStore = c.Subscriptions()
				break
			}
		}
		if connSubStore == nil {
			return fmt.Errorf("%w: connection subscription store not found", common.ErrNilPointer)
		}
		subscriptionStore = ws.subscriptions
		usedCapacity = connSubStore.Len()
	} else {
		if m.subscriptions == nil {
			return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
		}
		subscriptionStore = m.subscriptions
		usedCapacity = subscriptionStore.Len()
	}

	if m.MaxSubscriptionsPerConnection > 0 && usedCapacity+len(subs) > m.MaxSubscriptionsPerConnection {
		return fmt.Errorf("%w: current subscriptions: %v, incoming subscriptions: %v, max subscriptions per connection: %v",
			errSubscriptionsExceedsLimit,
			usedCapacity,
			len(subs),
			m.MaxSubscriptionsPerConnection)
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
	if !m.IsEnabled() {
		return fmt.Errorf("%s %w", m.exchangeName, ErrWebsocketNotEnabled)
	}

	if !m.IsConnected() {
		return fmt.Errorf("%s %w", m.exchangeName, ErrNotConnected)
	}

	// If the exchange does not support subscribing and or unsubscribing the full connection needs to be flushed to
	// maintain consistency.
	if !m.features.Subscribe || !m.features.Unsubscribe {
		m.m.Lock()
		defer m.m.Unlock()
		if err := m.shutdown(); err != nil {
			return err
		}
		return m.connect(ctx)
	}

	if !m.useMultiConnectionManagement {
		newSubs, err := m.GenerateSubs()
		if err != nil {
			return err
		}
		return m.updateChannelSubscriptions(ctx, m.subscriptions, newSubs)
	}

	for _, ws := range m.connectionManager {
		if ws.setup.SubscriptionsNotRequired {
			continue
		}

		newSubs, err := ws.setup.GenerateSubscriptions()
		if err != nil {
			return err
		}

		// Case if there is nothing to unsubscribe from and the connection is nil
		if len(newSubs) == 0 && len(ws.connections) == 0 {
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

// scaleConnectionsToSubscriptions scales connections to subscriptions based off current subscription list and subscription limit
func (m *Manager) scaleConnectionsToSubscriptions(ctx context.Context, ws *websocket, incoming subscription.List) error {
	if err := common.NilGuard(ws); err != nil {
		return err
	}
	subs, unsubs := ws.subscriptions.Diff(incoming)
	if len(unsubs) != 0 {
		currentUnsubs := slices.Clone(unsubs)
		// Unsubscribe first to free up capacity on existing connections
		for _, conn := range ws.connections {
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
		// Subscribe to existing connections to use up existing capacity
		currentSubs := slices.Clone(subs)
		for _, conn := range ws.connections {
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
			if err := m.createConnectAndSubscribe(ctx, ws, batch); err != nil {
				return err
			}
		}

		if missing := ws.subscriptions.Missing(subs); len(missing) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
		}
	}

	// Clean up any connections that have no subscriptions left to reduce resource usage
	clean := make([]Connection, 0, len(ws.connections))
	for _, conn := range ws.connections {
		if conn.Subscriptions().Len() != 0 {
			clean = append(clean, conn)
			continue
		}
		delete(m.connections, conn)
		if err := conn.Shutdown(); err != nil {
			log.Warnf(log.WebsocketMgr, "%v websocket: failed to shutdown connection: %v", m.exchangeName, err)
		}
	}
	ws.connections = clean
	return nil
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

	usedCap := store.Len()
	if m.MaxSubscriptionsPerConnection > 0 && usedCap == m.MaxSubscriptionsPerConnection {
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
	if err := m.SubscribeToChannels(ctx, conn, toSubscribe); err != nil {
		return nil, err
	}

	for _, s := range toSubscribe {
		if err := store.Add(s); err != nil {
			return nil, err
		}
	}

	return subs[availableCap:], nil
}
