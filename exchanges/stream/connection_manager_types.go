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
}

// // ConnectionConfig defines a singular connection configuration
// type ConnectionConfig struct {
// 	DedicatedAuth bool
// }

// ConnectionManager manages connections
type ConnectionManager struct {
	sync.Mutex
	connections        []Connection
	features           *protocol.Features
	connector          func(conn Connection) error
	generator          func(options SubscriptionOptions) ([]ChannelSubscription, error)
	subscriber         func(sub SubscriptionParamaters) error
	unsubscriber       func(unsub SubscriptionParamaters) error
	generateConnection func(ConnectionSetup, []ChannelSubscription) ([]Connection, error)

	generalConfigurations      []ConnectionSetup
	dedicatedAuthConfiguration ConnectionSetup
}

// ConnectionManagerConfig defines the needed variables for stream connections
type ConnectionManagerConfig struct {
	ExchangeConnector             func(conn Connection) error
	ExchangeGenerateSubscriptions func(options SubscriptionOptions) ([]ChannelSubscription, error)
	ExchangeSubscriber            func(sub SubscriptionParamaters) error
	ExchangeUnsubscriber          func(unsub SubscriptionParamaters) error
	ExchangeGenerateConnection    func(ConnectionSetup, []ChannelSubscription) ([]Connection, error)
	Features                      *protocol.Features
}

// SubscriptionParamaters defines payload for subscribing and unsibscribing
type SubscriptionParamaters struct {
	Items   []ChannelSubscription
	Conn    Connection
	Manager *ConnectionManager
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
