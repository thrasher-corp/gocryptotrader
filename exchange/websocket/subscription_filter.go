package websocket

import "github.com/thrasher-corp/gocryptotrader/exchanges/subscription"

// SetSubscriptionFilter filters subscriptions before they are sent to the exchange
func (m *Manager) SetSubscriptionFilter(filter subscription.FilterHook) {
	m.m.Lock()
	defer m.m.Unlock()
	m.subscriptionFilter = filter
}
