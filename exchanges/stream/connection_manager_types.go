package stream

import (
	"fmt"
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
	if c == nil {
		panic("connectionsetup nil")
	}
	if c == nil {
		panic("sub nil")
	}

	fmt.Println("SUB:", sub)
	fmt.Println("sub length:", currentSubLength)
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
	wg                 *sync.WaitGroup
	connections        []Connection
	features           *protocol.Features
	connector          func(conn Connection) error
	authConnector      func(conn Connection) error
	generator          func(options SubscriptionOptions) ([]ChannelSubscription, error)
	subscriber         func(sub SubscriptionParameters) error
	unsubscriber       func(unsub SubscriptionParameters) error
	generateConnection func(c ConnectionSetup) (Connection, error)
	responseHandler    func([]byte, Connection) error

	dataHandler chan interface{}

	configurations []ConnectionSetup
}

// ConnectionManagerConfig defines the needed variables for stream connections
type ConnectionManagerConfig struct {
	Wg                            *sync.WaitGroup
	ExchangeConnector             func(conn Connection) error
	ExchangeAuthConnector         func(conn Connection) error
	ExchangeGenerateSubscriptions func(options SubscriptionOptions) ([]ChannelSubscription, error)
	ExchangeSubscriber            func(sub SubscriptionParameters) error
	ExchangeUnsubscriber          func(unsub SubscriptionParameters) error
	ExchangeGenerateConnection    func(ConnectionSetup) (Connection, error)
	ExchangeReadConnection        func([]byte, Connection) error
	Features                      *protocol.Features

	dataHandler chan interface{}

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
