package stream

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

// // ConnectionSubscriptions defines connection pool and associated current
// // subscriptions
// type ConnectionSubscriptions struct {
// 	m   map[Connection]map[Subscription]map[currency.Pair]map[asset.Item]ChannelSubscription
// 	mtx sync.Mtx
// }

// // loadConnection deploys a new connections
// func (c *ConnectionSubscriptions) loadConnection(conn Connnection) error {
// 	c.mtx.Lock()
// 	defer c.mtx.Unlock()
// 	if c.m == nil {
// 		c.m = make(map[Connection]map[Subscription]map[currency.Pair]map[asset.Item]ChannelSubscription)
// 		c.m[conn] = nil
// 		return nil
// 	}

// 	if _, ok := c.m[conn]; !ok {
// 		c.m[conn] = nil
// 		return nil
// 	}
// 	return errors.New("connection already loaded")
// }

// func ConnectionSubscriptions()

// ConnectionManager manages connections
type ConnectionManager struct {
	sync.Mutex
	connections        []Connection
	features           *protocol.Features
	connector          func(conn Connection) error
	generator          func(options SubscriptionOptions) ([]ChannelSubscription, error)
	subscriber         func(sub SubscriptionParamaters) error
	unsubscriber       func(unsub SubscriptionParamaters) error
	generateConnection func(ConnectionSetup) (Connection, error)

	generalConfigurations      []ConnectionSetup
	dedicatedAuthConfiguration ConnectionSetup
}

// ConnectionManagerConfig defines the needed variables for stream connections
type ConnectionManagerConfig struct {
	ExchangeConnector             func(conn Connection) error
	ExchangeGenerateSubscriptions func(options SubscriptionOptions) ([]ChannelSubscription, error)
	ExchangeSubscriber            func(sub SubscriptionParamaters) error
	ExchangeUnsubscriber          func(unsub SubscriptionParamaters) error
	ExchangeGenerateConnection    func(ConnectionSetup) (Connection, error)
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
	// Connections []Connection
	Features *protocol.Features
	// Manager     *ConnectionManager
}

// NewConnectionManager returns a new connection manager
func NewConnectionManager(cfg *ConnectionManagerConfig) (*ConnectionManager, error) {
	if cfg == nil {
		return nil, errors.New("configuration cannot be nil")
	}
	if cfg.ExchangeConnector == nil {
		return nil, errors.New("exchange connector function cannot be nil")
	}
	if cfg.ExchangeGenerateConnection == nil {
		return nil, errors.New("exchange generator function cannot be nil")
	}
	if cfg.ExchangeGenerateConnection == nil {
		return nil, errors.New("exchange generate connection function cannot be nil")
	}
	if cfg.Features == nil {
		return nil, errors.New("exchange features cannot be nil")
	}

	return &ConnectionManager{
		connector:          cfg.ExchangeConnector,
		generator:          cfg.ExchangeGenerateSubscriptions,
		subscriber:         cfg.ExchangeSubscriber,
		unsubscriber:       cfg.ExchangeUnsubscriber,
		generateConnection: cfg.ExchangeGenerateConnection,
		features:           cfg.Features,
	}, nil
}

// SubscriptionConnections defines a type that has a connection and relative
// subscriptions ready to go
type SubscriptionConnections struct {
	Subs []ChannelSubscription
	conn Connection
}

// GenerateConnections returns generated connections from the service
func (c *ConnectionManager) GenerateConnections() ([]Connection, error) {
	var conns []Connection
	for i := range c.generalConfigurations {
		conn, err := c.generateConnection(c.generalConfigurations[i])
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}

	if c.dedicatedAuthConfiguration.URL != "" {
		conn, err := c.generateConnection(c.dedicatedAuthConfiguration)
		if err != nil {
			return nil, err
		}
		conns = append(conns, conn)
	}

	return conns, nil
}

// LoadConfiguration loads a connection configuration defining limitting
// paramaters for scalable streaming connections
func (c *ConnectionManager) LoadConfiguration(cfg ConnectionSetup) error {
	if cfg.DedicatedAuthenticatedConn {
		c.dedicatedAuthConfiguration = cfg
		return nil
	}
	c.generalConfigurations = append(c.generalConfigurations, cfg)
	return nil
}

// GenerateSubscriptions generates new connection profiles
func (c *ConnectionManager) GenerateSubscriptions() ([]ChannelSubscription, error) {
	c.Lock()
	defer c.Unlock()
	return c.generator(SubscriptionOptions{Features: c.features})
}

// CreateConnectionsBySubscriptions create new connections by subscription
// params
func (c *ConnectionManager) CreateConnectionsBySubscriptions() error {
	return nil
}

// LoadNewConnection loads a newly established connection
func (c *ConnectionManager) LoadNewConnection(conn Connection) error {
	c.Lock()
	defer c.Unlock()
	for i := range c.connections {
		if c.connections[i] == conn {
			return errors.New("connection already loaded")
		}
	}
	c.connections = append(c.connections, conn)
	return nil
}

// Connect connects all loaded connections
func (c *ConnectionManager) Connect() error {
	c.Lock()
	defer c.Unlock()

	for i := range c.connections {
		err := c.connector(c.connections[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// Subscribe subscribes and sets subscription by stream connection
func (c *ConnectionManager) Subscribe(subs []ChannelSubscription) error {
	c.Lock()
	defer c.Unlock()

	if c.subscriber == nil {
		return errors.New("exchange subscriber functionality not set, cannot subscribe")
	}

	if subs == nil {
		return errors.New("no subscription data cannot subscribe")
	}

	for i := range c.connections {
		if c.connections[i].IsAuthenticated() {
			continue
		}

		err := c.subscriber(SubscriptionParamaters{
			Items:   subs,
			Conn:    c.connections[i],
			Manager: c,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Unsubscribe unsubscribes and removes subscription by stream connection
func (c *ConnectionManager) Unsubscribe(unsubs []SubscriptionParamaters) error {
	c.Lock()
	defer c.Unlock()

	if c.unsubscriber == nil {
		return errors.New("exchange unsubscriber functionality not set, cannot unsubscribe")
	}

	if unsubs == nil {
		return errors.New("no subscription data cannot unsubscribe")
	}

	for i := range unsubs {
		err := c.unsubscriber(unsubs[i])
		if err != nil {
			return err
		}
	}
	return nil
}
