package stream

import (
	"errors"
	"fmt"
)

var (
	errNoConfigurations   = errors.New("at least one general configuration must be supplied")
	errNoGenerateConnFunc = errors.New("exchange connection generator function cannot be nil")
	errNoGenerateSubsFunc = errors.New("exchange generate subscription function cannot be nil")
	errMissingURLInConfig = errors.New("connection URL must be supplied")
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
		return nil, errNoGenerateSubsFunc
	}
	if cfg.ExchangeGenerateConnection == nil {
		return nil, errNoGenerateConnFunc
	}
	if cfg.Features == nil {
		return nil, errors.New("exchange features cannot be nil")
	}
	if len(cfg.Configurations) < 1 {
		return nil, errNoConfigurations
	}
	for i := range cfg.Configurations {
		if cfg.Configurations[i].URL == "" {
			return nil, errMissingURLInConfig
		}
	}

	return &ConnectionManager{
		connector:             cfg.ExchangeConnector,
		generator:             cfg.ExchangeGenerateSubscriptions,
		subscriber:            cfg.ExchangeSubscriber,
		unsubscriber:          cfg.ExchangeUnsubscriber,
		generateConnection:    cfg.ExchangeGenerateConnection,
		features:              cfg.Features,
		generalConfigurations: cfg.Configurations,
	}, nil
}

// GenerateConnections returns generated connections from the service
func (c *ConnectionManager) GenerateConnections(subs []ChannelSubscription) (map[Connection][]ChannelSubscription, error) {
	subscriptionsToConfig := make(map[*ConnectionSetup]*[]ChannelSubscription)
	for y := range c.generalConfigurations {
		// Populate configurations in map
		if _, ok := subscriptionsToConfig[&c.generalConfigurations[y]]; !ok {
			subscriptionsToConfig[&c.generalConfigurations[y]] = &[]ChannelSubscription{}
		}
	}

	// Associate individual subscription to configuration
subscriptions:
	for z := range subs {
		for k, v := range subscriptionsToConfig {
			if k.DedicatedAuthenticatedConn {
				// ADD directive to ChannelSubs
			}

			if len(k.AllowableAssets) != 0 {
				if !k.AllowableAssets.Contains(subs[z].Asset) {
					continue
				}
			}

			*v = append(*v, subs[z])
			continue subscriptions
		}
		return nil, fmt.Errorf("subscription [%v] could not be associated with a connection", subs[z])
	}

	// reference our subscriptions to a new connection
	reference := make(map[Connection][]ChannelSubscription)
	for k, v := range subscriptionsToConfig {
		if int(k.MaxSubscriptions) == 0 || len(*v) < int(k.MaxSubscriptions) {
			conn, err := c.generateConnection(k.URL, k.DedicatedAuthenticatedConn)
			if err != nil {
				return nil, err
			}
			reference[conn] = *v
			continue
		}
		// If includes max subscriptions sub-divide into windows
		for i := len(*v); i > 0; i -= int(k.MaxSubscriptions) {
			conn, err := c.generateConnection(k.URL, k.DedicatedAuthenticatedConn)
			if err != nil {
				return nil, err
			}
			reference[conn] = append(reference[conn], (*v)[i-int(k.MaxSubscriptions):i]...)
		}
	}
	return reference, nil
}

// // LoadConfiguration loads a connection configuration defining limitting
// // paramaters for scalable streaming connections
// func (c *ConnectionManager) LoadConfiguration(cfg ConnectionSetup) error {
// 	// if cfg.DedicatedAuthenticatedConn {
// 	// 	c.dedicatedAuthConfiguration = cfg
// 	// 	return nil
// 	// }
// 	// c.generalConfigurations = append(c.generalConfigurations, cfg)
// 	// return nil
// 	return errors.New("not yet implemented")
// }

// FullConnect generates subscriptions, deploys new connections and subscribes
// to channels
func (c *ConnectionManager) FullConnect() error {
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

	// err = c.Connect()
	// if err != nil {
	// 	return err
	// }

	// err = c.Subscribe(subscriptions)
	// if err != nil {
	// 	return err
	// }
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
func (c *ConnectionManager) Connect(conn Connection) error {
	c.Lock()
	defer c.Unlock()
	return c.connector(conn)
}

// Subscribe subscribes and sets subscription by stream connection
func (c *ConnectionManager) Subscribe(conn Connection, subs []ChannelSubscription) error {
	c.Lock()
	defer c.Unlock()

	if c.subscriber == nil {
		return errors.New("exchange subscriber functionality not set, cannot subscribe")
	}

	if subs == nil {
		return errors.New("no subscription data cannot subscribe")
	}

	return c.subscriber(SubscriptionParamaters{
		Items:   subs,
		Conn:    conn,
		Manager: c,
	})
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
