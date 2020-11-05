package stream

import (
	"errors"
	"fmt"
)

var (
	errNoMainConfiguration          = errors.New("main configuration cannot be nil")
	errNoExchangeConnectionFunction = errors.New("exchange connector function cannot be nil")
	errNoConfigurations             = errors.New("at least one general configuration must be supplied")
	errNoSubscribeFunction          = errors.New("exchange subscriber function must be supplied")
	errNoUnsubscribeFunction        = errors.New("exchange unsubscriber function must be supplied")
	errNoGenerateConnFunc           = errors.New("exchange connection generator function cannot be nil")
	errNoFeatures                   = errors.New("exchange features cannot be nil")
	errNoGenerateSubsFunc           = errors.New("exchange generate subscription function cannot be nil")
	errMissingURLInConfig           = errors.New("connection URL must be supplied")
	errNoAssociation                = errors.New("could not associate a subscription with a configuration")
)

// NewConnectionManager returns a new connection manager
func NewConnectionManager(cfg *ConnectionManagerConfig) (*ConnectionManager, error) {
	if cfg == nil {
		return nil, errNoMainConfiguration
	}
	if cfg.ExchangeConnector == nil {
		return nil, errNoExchangeConnectionFunction
	}
	if cfg.ExchangeGenerateSubscriptions == nil {
		return nil, errNoGenerateSubsFunc
	}
	if cfg.ExchangeSubscriber == nil {
		return nil, errNoSubscribeFunction
	}
	if cfg.ExchangeUnsubscriber == nil {
		return nil, errNoUnsubscribeFunction
	}
	if cfg.ExchangeGenerateConnection == nil {
		return nil, errNoGenerateConnFunc
	}
	if cfg.Features == nil {
		return nil, errNoFeatures
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
		connector:          cfg.ExchangeConnector,
		generator:          cfg.ExchangeGenerateSubscriptions,
		subscriber:         cfg.ExchangeSubscriber,
		unsubscriber:       cfg.ExchangeUnsubscriber,
		generateConnection: cfg.ExchangeGenerateConnection,
		features:           cfg.Features,
		configurations:     cfg.Configurations,
	}, nil
}

// GenerateConnections returns generated connections from the service
func (c *ConnectionManager) GenerateConnections(subs []ChannelSubscription) (map[Connection][]ChannelSubscription, error) {
	subscriptionsToConfig := make(map[*ConnectionSetup]*[]ChannelSubscription)
	for y := range c.configurations {
		// Populate configurations in map
		if _, ok := subscriptionsToConfig[&c.configurations[y]]; !ok {
			subscriptionsToConfig[&c.configurations[y]] = &[]ChannelSubscription{}
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
			err = conn.LoadSubscriptionManager(NewSubscriptionManager())
			if err != nil {
				return nil, err
			}
			reference[conn] = *v
			continue
		}
		// If includes max subscriptions sub-divide into windows
		for remaining := len(*v); remaining > 0; remaining -= int(k.MaxSubscriptions) {
			left := remaining - int(k.MaxSubscriptions)
			if left < 0 {
				left = 0
			}
			conn, err := c.generateConnection(k.URL, k.DedicatedAuthenticatedConn)
			if err != nil {
				return nil, err
			}
			err = conn.LoadSubscriptionManager(NewSubscriptionManager())
			if err != nil {
				return nil, err
			}
			reference[conn] = append(reference[conn], (*v)[left:remaining]...)
		}
	}
	return reference, nil
}

// FullConnect generates subscriptions, deploys new connections and subscribes
// to channels
func (c *ConnectionManager) FullConnect() error {
	// subscriptions, err := c.GenerateSubscriptions()
	// if err != nil {
	// 	return err
	// }

	// fmt.Println("generated subs:", subscriptions)

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

// Sub temp sub associates with a connection
func (c *ConnectionManager) Sub(sub *ChannelSubscription) error {
	c.Lock()
	for i := range c.connections {
		subs := c.connections[i].GetAllSubscriptions()
		for j := range subs {
			if subs[j].Equal(sub) {
				c.Unlock()
				return c.Subscribe(c.connections[i], []ChannelSubscription{*sub})
			}
		}
	}
	c.Unlock()
	return errors.New("could not find subscription to unsubscribe from across all connections")
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

	return c.subscriber(SubscriptionParameters{
		Items: subs,
		Conn:  conn,
	})
}

// Unsub temp associates subscription with connection
func (c *ConnectionManager) Unsub(sub *ChannelSubscription) error {
	c.Lock()
	for i := range c.connections {
		subs := c.connections[i].GetAllSubscriptions()
		for j := range subs {
			if subs[j].Equal(sub) {
				c.Unlock()
				return c.Unsubscribe([]SubscriptionParameters{
					{
						Items: []ChannelSubscription{*sub},
						Conn:  c.connections[i],
					},
				})
			}
		}
	}
	c.Unlock()
	return errors.New("could not find subscription to unsubscribe from across all connections")
}

// Unsubscribe unsubscribes and removes subscription by stream connection
func (c *ConnectionManager) Unsubscribe(unsubs []SubscriptionParameters) error {
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

// FlushSubscriptions removes all subscriptions associated with all connections
func (c *ConnectionManager) FlushSubscriptions() {
	c.Lock()
	for i := range c.connections {
		c.connections[i].FlushSubscriptions()
	}
	c.Unlock()
}

// GetChannelDifference finds the difference between the subscribed channels
// and the new subscription list when pairs are disabled or enabled.
func (c *ConnectionManager) GetChannelDifference(newSubs []ChannelSubscription) (subscribe, unsubscribe []SubscriptionParameters, err error) {
	c.Lock()
	defer c.Unlock()

	currentlySubscribed := make(map[Connection][]ChannelSubscription)
	for x := range c.connections {
		currentlySubscribed[c.connections[x]] = append(currentlySubscribed[c.connections[x]],
			c.connections[x].GetAllSubscriptions()...)
	}

	fmt.Println("currentlySubscribed:", currentlySubscribed)
	fmt.Println("newSubs:", newSubs)

	var newSubscribes []ChannelSubscription
	// Check connections to see if new subscription is not already subscribed
subscriptionCheck:
	for i := range newSubs {
		for _, subs := range currentlySubscribed {
			for j := range subs {
				if subs[j].Equal(&newSubs[i]) {
					continue subscriptionCheck
				}
			}
		}
		newSubscribes = append(newSubscribes, newSubs[i])
	}

	fmt.Println("newSubscribes", newSubscribes)

	var unsubscribeMe = make(map[Connection][]ChannelSubscription)
	// Check connections to see what needs to be removed in difference to the
	// newly generated subscriptions
	for conn, subs := range currentlySubscribed {
	unsubscriptionCheck:
		for x := range subs {
			for y := range newSubs {
				if subs[x].Equal(&newSubs[y]) {
					continue unsubscriptionCheck
				}
			}

			// Remove instance from currently subscribed so as to determine the
			// max connection allowance for new subscriptions
			subs[x] = subs[len(subs)-1]
			subs[len(subs)-1] = ChannelSubscription{}
			subs = subs[:len(subs)-1]

			unsubscribeMe[conn] = append(unsubscribeMe[conn], subs[x])
		}
	}

	fmt.Println("unsubscribeMe", unsubscribeMe)
	fmt.Println("currentlySubscribed again:", currentlySubscribed)

	var subscribeMe = make(map[Connection][]ChannelSubscription)
subbies:
	for i := range newSubscribes {
		for conn, subs := range currentlySubscribed {
			cfg := conn.GetConfiguration()
			if cfg.SubscriptionConforms(&newSubscribes[i], len(subs)) {
				continue
			}

			// Add to currently subscribed
			addSubs := append(subs, newSubscribes[i])
			currentlySubscribed[conn] = addSubs
			subscribeMe[conn] = append(subscribeMe[conn], newSubscribes[i])
			continue subbies
		}

		// Spawn new connection
		for j := range c.configurations {
			if !c.configurations[j].SubscriptionConforms(&newSubscribes[i], 0) {
				continue
			}
			var conn Connection
			conn, err = c.generateConnection("", false)
			if err != nil {
				return
			}

			subscribeMe[conn] = append(subscribeMe[conn], newSubscribes[i])
			continue subbies
		}

		err = errNoAssociation
		return
	}
	// package everything
	for k, v := range subscribeMe {
		subscribe = append(subscribe, SubscriptionParameters{v, k})
	}

	for k, v := range unsubscribeMe {
		unsubscribe = append(unsubscribe, SubscriptionParameters{v, k})
	}

	fmt.Println("subscribeMe:", subscribeMe)

	return
}
