package stream

import (
	"errors"
	"fmt"
)

// NewConnectionManager returns a new connection manager
func NewConnectionManager(cfg *ConnectionManagerConfig) (*ConnectionManager, error) {
	if cfg == nil {
		return nil, errors.New("configuration cannot be nil")
	}
	if cfg.ExchangeConnector == nil {
		return nil, errors.New("exchange connector function cannot be nil")
	}
	if cfg.ExchangeGenerateSubscriptions == nil {
		return nil, errors.New("exchange generate subscription function cannot be nil")
	}
	if cfg.ExchangeGenerateConnection == nil {
		return nil, errors.New("exchange generator function cannot be nil")
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

// GenerateConnections returns generated connections from the service
func (c *ConnectionManager) GenerateConnections(authEnabled bool, subs []ChannelSubscription) (map[Connection]ChannelSubscription, error) {
	// Get current connections
	for i := range c.connections {
		fmt.Println("CURRENT CONNECTION STATUS:", c.connections[i])
	}

	// var conns []Connection
	// var dividedSubs [][]ChannelSubscription

	// configurations:
	// 	for x := range c.generalConfigurations {
	// 		var relativeSubs []ChannelSubscription
	// 		for y := range subs {
	// 			if len(c.generalConfigurations[x].AllowableAssets) != 0 {
	// 				// Test asset allowance
	// 				if subs[y].Asset != "" &&
	// 					!c.generalConfigurations[x].AllowableAssets.Contains(subs[y].Asset) {
	// 					continue
	// 				}
	// 			}

	// 			if len(relativeSubs)+1 > 1024 {
	// 				continue configurations
	// 			}

	// 			relativeSubs = append(relativeSubs, subs[y])
	// 			subs = append(subs[:y], subs[y+1:]...)
	// 			y--
	// 		}

	// 		conn, err := c.generateConnection(c.generalConfigurations[x], relativeSubs)
	// 		if err != nil {
	// 			return nil, nil, err
	// 		}
	// 		conns = append(conns, conn...)
	// 		dividedSubs = append(dividedSubs, [][]ChannelSubscription{relativeSubs})
	// 	}

	// 	if subs != 0 {
	// 		return fmt.Errorf("dangly subscriptions not associated with a connection %v", subs)
	// 	}

	// 	if authEnabled {
	// 		conn, err := c.generateConnection(c.dedicatedAuthConfiguration, subs)
	// 		if err != nil {
	// 			return nil, nil, err
	// 		}
	// 		conns = append(conns, conn...)
	// 	}

	// 	return conns, dividedSubs, nil
	return make(map[Connection]ChannelSubscription), nil
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

// FullConnect generates subscriptions, deploys new connections and subscribes
// to channels
func (c *ConnectionManager) FullConnect(authEnabled bool) error {
	subscriptions, err := c.GenerateSubscriptions()
	if err != nil {
		return err
	}

	fmt.Println("generated subs:", subscriptions)

	// connections, subs, err := c.GenerateConnections(authEnabled, subscriptions)
	// if err != nil {
	// 	return err
	// }

	// fmt.Println("generated cons:", connections)

	// fmt.Println("SUBS BRA:", subs)

	// for i := range connections {
	// 	err = c.LoadNewConnection(connections[i])
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	err = c.Connect()
	if err != nil {
		return err
	}

	err = c.Subscribe(subscriptions)
	if err != nil {
		return err
	}
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
func (c *ConnectionManager) CreateConnectionsBySubscriptions(potentialSubs []ChannelSubscription) error {
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
