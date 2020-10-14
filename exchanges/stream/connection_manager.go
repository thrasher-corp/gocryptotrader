package stream

import (
	"errors"
	"fmt"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

// Subscription defines a subscription type
type Subscription int

// Consts here define difference subscription types
const (
	Orderbook Subscription = iota + 1
	Kline
	Trade
	Ticker
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
	conn               map[Connection]map[Subscription][]ChannelSubscription
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
		conn:               make(map[Connection]map[Subscription][]ChannelSubscription),
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
func (c *ConnectionManager) LoadNewConnection(newConn Connection) error {
	c.Lock()
	defer c.Unlock()
	_, ok := c.conn[newConn]
	if !ok {
		c.conn[newConn] = nil
		return nil
	}
	return errors.New("connection already loaded")
}

// Connect connects all loaded connections
func (c *ConnectionManager) Connect() error {
	c.Lock()
	defer c.Unlock()

	for conn := range c.conn {
		err := c.connector(conn)
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

	for conn := range c.conn {
		if conn.IsAuthenticated() {
			continue
		}

		err := c.subscriber(SubscriptionParamaters{
			Items:   subs,
			Conn:    conn,
			Manager: c})
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

// GetAllSubscriptions returns current subscriptions for our streaming
// connections
func (c *ConnectionManager) GetAllSubscriptions() []ChannelSubscription {
	c.Lock()
	defer c.Unlock()
	var subscriptions []ChannelSubscription
	for _, subTypes := range c.conn {
		for _, subs := range subTypes {
			subscriptions = append(subscriptions, subs...)
		}
	}
	return subscriptions
}

// GetAssetsBySubscriptionType returns assets associated with the same channel
// subscription type. This is used for when margin and spot which collectively
// are the same thing but have different functionality
func (c *ConnectionManager) GetAssetsBySubscriptionType(conn Connection, subType Subscription, pair currency.Pair) (asset.Items, error) {
	// fmt.Printf("Getting assets from conn: %p\n", conn)
	c.Lock()
	defer c.Unlock()
	val, ok := c.conn[conn]
	if !ok {
		return nil, errors.New("cannot find connection")
	}

	// fmt.Println("CONNECTION Container:", c.conn)
	// fmt.Println("CONNECTION Container VAL:", val)
	fmt.Printf("subscriptions: %v Connection: %p\n", val, conn)

	var assets asset.Items
	for s, subscriptions := range val {
		fmt.Println("CATS DOOM!")
		if s != subType {
			fmt.Println("bad subtype")
			continue
		}

		for i := range subscriptions {
			if !subscriptions[i].Currency.Equal(pair) ||
				assets.Contains(subscriptions[i].Asset) {
				fmt.Println("THIS SUCKS!")
				continue
			}
			assets = append(assets, subscriptions[i].Asset)
		}
	}

	if len(assets) == 0 {
		return nil, errors.New("no asset associations found")
	}

	return assets, nil
}

// AddSuccessfulSubscriptions adds subs mate
func (c *ConnectionManager) AddSuccessfulSubscriptions(conn Connection, subs []ChannelSubscription) error {
	// fmt.Printf("Adding sucessful subs to conn: %p\n", conn)
	// c.Lock()
	// defer c.Unlock()
	subscriptions, ok := c.conn[conn]
	if !ok {
		return errors.New("connection not set in manager")
	}

	if subscriptions == nil {
		subscriptions = make(map[Subscription][]ChannelSubscription)
		for i := range subs {
			t := subs[i].SubscriptionType
			subscriptions[t] = append(subscriptions[t], subs[i])
		}
		c.conn[conn] = subscriptions
		return nil
	}

	for i := range subs {
		t := subs[i].SubscriptionType
		subscribed, ok := subscriptions[t]
		if !ok {
			return fmt.Errorf("connection does not have subscribed type %v", t)
		}
		for j := range subscribed {
			if subscribed[j].Channel == subs[i].Channel {
				return fmt.Errorf("channel %s already subscribed", subs[i].Channel)
			}
		}
		subscriptions[t] = append(subscriptions[t], subs[i])
	}
	return nil
}

// RemoveSuccessfulUnsubscriptions removes subs mate
func (c *ConnectionManager) RemoveSuccessfulUnsubscriptions(conn Connection, unsub []ChannelSubscription) error {
	// c.Lock()
	// defer c.Unlock()
	subscriptions, ok := c.conn[conn]
	if !ok {
		return errors.New("connection not set in manager")
	}

	if subscriptions == nil {
		return errors.New("connection does not have associated subscriptions")
	}

	for x := range unsub {
		t := unsub[x].SubscriptionType
		subscribed, ok := subscriptions[t]
		if !ok {
			return fmt.Errorf("connection does not have subscribed type %v", t)
		}
		for y := range subscribed {
			if subscribed[y].Channel == unsub[x].Channel {
				subscribed[y] = subscribed[len(subscribed)-1]
				subscribed[len(subscribed)-1] = ChannelSubscription{}
				subscribed = subscribed[:len(subscribed)-1]
			}
		}
		return errors.New("subscription not found")
	}
	return nil
}
