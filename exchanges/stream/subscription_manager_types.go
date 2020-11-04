package stream

import "sync"

// Subscription defines a subscription type
type Subscription int

// Consts here define difference subscription types
const (
	Orderbook Subscription = iota + 1
	Kline
	Trade
	Ticker
)

// SubscriptionManager defines a subscription system attached to an individual
// connection
type SubscriptionManager struct {
	sync.Mutex
	m map[Subscription]*[]ChannelSubscription
}
