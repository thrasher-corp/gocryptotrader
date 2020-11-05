package stream

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

// ConnectionSetup defines variables for an individual stream connection
type ConnectionSetup struct {
	URL                        string
	DedicatedAuthenticatedConn bool
	AllowableAssets            asset.Items
	MaxSubscriptions           uint16
}

// SubscriptionConforms checks to see if the subscription conforms to the
// configuration
func (c *ConnectionSetup) SubscriptionConforms(sub *ChannelSubscription, currentSubLength int) bool {
	if len(c.AllowableAssets) != 0 && !c.AllowableAssets.Contains(sub.Asset) {
		return false
	}

	if c.MaxSubscriptions != 0 && currentSubLength+1 > int(c.MaxSubscriptions) {
		return false
	}

	return true
}

// ConnectionManager manages connections
type ConnectionManager struct {
	sync.Mutex
	connections        []Connection
	features           *protocol.Features
	connector          func(conn Connection) error
	generator          func(options SubscriptionOptions) ([]ChannelSubscription, error)
	subscriber         func(sub SubscriptionParameters) error
	unsubscriber       func(unsub SubscriptionParameters) error
	generateConnection func(url string, auth bool) (Connection, error)

	configurations []ConnectionSetup
}

// ConnectionManagerConfig defines the needed variables for stream connections
type ConnectionManagerConfig struct {
	ExchangeConnector             func(conn Connection) error
	ExchangeGenerateSubscriptions func(options SubscriptionOptions) ([]ChannelSubscription, error)
	ExchangeSubscriber            func(sub SubscriptionParameters) error
	ExchangeUnsubscriber          func(unsub SubscriptionParameters) error
	ExchangeGenerateConnection    func(url string, auth bool) (Connection, error)
	Features                      *protocol.Features

	Configurations []ConnectionSetup
}

// SubscriptionParameters defines payload for subscribing and unsibscribing
type SubscriptionParameters struct {
	Items []ChannelSubscription
	Conn  Connection
}

// SubscriptionOptions defines subscriber options and updates
type SubscriptionOptions struct {
	Features *protocol.Features
}

// SubscriptionConnections defines a type that has a connection and relative
// subscriptions ready to go
type SubscriptionConnections struct {
	Subs []ChannelSubscription
	conn Connection
}
