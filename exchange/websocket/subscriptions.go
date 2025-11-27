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
)

// UnsubscribeChannels unsubscribes from a list of websocket channel
func (m *Manager) UnsubscribeChannels(conn Connection, channels subscription.List) error {
	if len(channels) == 0 {
		return nil // No channels to unsubscribe from is not an error
	}
	if wrapper, ok := m.connections[conn]; ok && conn != nil {
		return m.unsubscribe(wrapper.subscriptions, channels, func(channels subscription.List) error {
			return wrapper.setup.Unsubscriber(context.TODO(), conn, channels)
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
func (m *Manager) ResubscribeToChannel(conn Connection, s *subscription.Subscription) error {
	l := subscription.List{s}
	if err := s.SetState(subscription.ResubscribingState); err != nil {
		return fmt.Errorf("%w: %s", err, s)
	}
	if err := m.UnsubscribeChannels(conn, l); err != nil {
		return err
	}
	return m.SubscribeToChannels(conn, l)
}

// SubscribeToChannels subscribes to websocket channels using the exchange specific Subscriber method
// Errors are returned for duplicates or exceeding max Subscriptions
func (m *Manager) SubscribeToChannels(conn Connection, subs subscription.List) error {
	if slices.Contains(subs, nil) {
		return fmt.Errorf("%w: List parameter contains an nil element", common.ErrNilPointer)
	}
	if err := m.checkSubscriptions(conn, subs); err != nil {
		return err
	}

	if wrapper, ok := m.connections[conn]; ok && conn != nil {
		return wrapper.setup.Subscriber(context.TODO(), conn, subs)
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
	if wrapper, ok := m.connections[conn]; ok && conn != nil {
		subscriptionStore = &wrapper.subscriptions
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
	if wrapper, ok := m.connections[conn]; ok && conn != nil {
		subscriptionStore = &wrapper.subscriptions
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
	if wrapper, ok := m.connections[conn]; ok && conn != nil {
		subscriptionStore = wrapper.subscriptions
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
	if wrapper, ok := m.connections[conn]; ok && conn != nil {
		if subscriptionStore = wrapper.subscriptions; subscriptionStore == nil {
			return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
		}
		var connSubStore *subscription.Store
		for _, connSubs := range wrapper.connectionsSubs {
			if connSubs.Connection == conn {
				connSubStore = connSubs.Subscriptions
				break
			}
		}
		if connSubStore == nil {
			return fmt.Errorf("%w: connection subscription store not found", common.ErrNilPointer)
		}
		usedCapacity = connSubStore.Len()
	} else {
		subscriptionStore = m.subscriptions
		if subscriptionStore == nil {
			return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
		}
		usedCapacity = subscriptionStore.Len()
	}

	if m.MaxSubscriptionsPerConnection > 0 && usedCapacity+len(subs) > m.MaxSubscriptionsPerConnection {
		return fmt.Errorf("%w: current subscriptions: %v, incoming subscriptions: %v, max subscriptions per connection: %v - please reduce enabled pairs",
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
func (m *Manager) FlushChannels() error {
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
		return m.connect()
	}

	if !m.useMultiConnectionManagement {
		newSubs, err := m.GenerateSubs()
		if err != nil {
			return err
		}
		return m.updateChannelSubscriptions(m.subscriptions, newSubs)
	}

	for _, wrapper := range m.connectionManager {
		if wrapper.setup.SubscriptionsNotRequired {
			continue
		}

		newSubs, err := wrapper.setup.GenerateSubscriptions()
		if err != nil {
			return err
		}

		// Case if there is nothing to unsubscribe from and the connection is nil
		if len(newSubs) == 0 && len(wrapper.connectionsSubs) == 0 {
			continue
		}

		if err := m.scaleConnectionsToSubscriptions(context.TODO(), wrapper, newSubs); err != nil {
			return err
		}
	}
	return nil
}

// updateChannelSubscriptions subscribes or unsubscribes from channels and checks that the correct number of channels
// have been subscribed to or unsubscribed from.
func (m *Manager) updateChannelSubscriptions(store *subscription.Store, incoming subscription.List) error {
	subs, unsubs := store.Diff(incoming)
	if len(unsubs) != 0 {
		if err := m.UnsubscribeChannels(nil, unsubs); err != nil {
			return err
		}

		if contained := store.Contained(unsubs); len(contained) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotRemoved, contained)
		}
	}
	if len(subs) != 0 {
		if err := m.SubscribeToChannels(nil, subs); err != nil {
			return err
		}

		if missing := store.Missing(subs); len(missing) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
		}
	}
	return nil
}

// scaleConnectionsToSubscriptions scales connections to subscriptions based off current subscription list
func (m *Manager) scaleConnectionsToSubscriptions(ctx context.Context, wrapper *connectionWrapper, incoming subscription.List) error {
	if err := common.NilGuard(wrapper); err != nil {
		return err
	}
	subs, unsubs := wrapper.subscriptions.Diff(incoming)
	if len(unsubs) != 0 {
		currentUnsubs := slices.Clone(unsubs)
		// Unsubscribe first to free up capacity on existing connections
		for _, connSubs := range wrapper.connectionsSubs {
			leftOver, err := m.unsubscribeFromConnection(connSubs.Connection, connSubs.Subscriptions, currentUnsubs)
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
				if err := wrapper.subscriptions.Remove(s); err != nil {
					return err
				}
			}
		}
		if contained := wrapper.subscriptions.Contained(unsubs); len(contained) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotRemoved, contained)
		}
	}
	if len(subs) != 0 {
		// Subscribe to existing connections to use up existing capacity
		currentSubs := slices.Clone(subs)
		for _, connSubs := range wrapper.connectionsSubs {
			leftOver, err := m.subscribeToConnection(connSubs.Connection, connSubs.Subscriptions, currentSubs)
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
			if err := m.connectAndSubscribe(ctx, wrapper, batch); err != nil {
				return err
			}
		}

		if missing := wrapper.subscriptions.Missing(subs); len(missing) > 0 {
			return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
		}
	}

	// Clean up any connections that have no subscriptions left to reduce resource usage
	clean := make([]connectionSubscriptions, 0, len(wrapper.connectionsSubs))
	for _, connSubs := range wrapper.connectionsSubs {
		if connSubs.Subscriptions.Len() != 0 {
			clean = append(clean, connSubs)
			continue
		}
		delete(m.connections, connSubs.Connection)
		if err := connSubs.Connection.Shutdown(); err != nil {
			log.Warnf(log.WebsocketMgr, "%v websocket: failed to shutdown connection: %v", m.exchangeName, err)
		}
	}
	wrapper.connectionsSubs = clean

	return nil
}

// unsubscribeFromConnection unsubscribes from a connection and removes subscriptions from the store
func (m *Manager) unsubscribeFromConnection(conn Connection, store *subscription.Store, subs subscription.List) (subscription.List, error) {
	remove := store.Contained(subs)
	if len(remove) == 0 {
		return subs, nil
	}

	if err := m.UnsubscribeChannels(conn, remove); err != nil {
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

// subscribeToConnection subscribes to a connection and adds subscriptions to the store
func (m *Manager) subscribeToConnection(conn Connection, store *subscription.Store, subs subscription.List) (subscription.List, error) {
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
	if err := m.SubscribeToChannels(conn, toSubscribe); err != nil {
		return nil, err
	}

	for _, s := range toSubscribe {
		if err := store.Add(s); err != nil {
			return nil, err
		}
	}

	return subs[availableCap:], nil
}
