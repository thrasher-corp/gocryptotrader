package stream

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
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
	errNoResponseDataHandler        = errors.New("exchange response data handler not set")
	errNoDataHandler                = errors.New("websocket data handler not set")
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
	if cfg.ExchangeReadConnection == nil {
		return nil, errNoResponseDataHandler
	}
	if cfg.Features == nil {
		return nil, errNoFeatures
	}
	if cfg.dataHandler == nil {
		return nil, errNoDataHandler
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
		wg:                 cfg.Wg,
		connector:          cfg.ExchangeConnector,
		generator:          cfg.ExchangeGenerateSubscriptions,
		subscriber:         cfg.ExchangeSubscriber,
		unsubscriber:       cfg.ExchangeUnsubscriber,
		generateConnection: cfg.ExchangeGenerateConnection,
		responseHandler:    cfg.ExchangeReadConnection,
		features:           cfg.Features,
		configurations:     cfg.Configurations,
		dataHandler:        cfg.dataHandler,
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
	subs, err := c.GenerateSubscriptions()
	if err != nil {
		return err
	}

	connections, err := c.GenerateConnections(subs)
	if err != nil {
		return err
	}

	for conn, subs := range connections {
		err = c.LoadNewConnection(conn)
		if err != nil {
			return err
		}

		err = c.Connect(conn)
		if err != nil {
			return err
		}

		err = c.Subscribe([]SubscriptionParameters{{subs, conn}})
		if err != nil {
			return err
		}
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
func (c *ConnectionManager) Connect(conn Connection) error {
	c.Lock()
	defer c.Unlock()
	err := c.connector(conn)
	if err != nil {
		return err
	}
	return c.ReadStream(conn, c.responseHandler)
}

// AssociateAndSubscribe associates the subscription with a connection and
// subscribes
func (c *ConnectionManager) AssociateAndSubscribe(subs []ChannelSubscription) error {
	if subs == nil {
		return errors.New("no subscription data cannot subscribe")
	}

	c.Lock()
	// for i := range c.connections {
	// 	subs := c.connections[i].GetAllSubscriptions()
	// 	for j := range subs {
	// 		if subs[j].Equal(sub) {
	// 			c.Unlock()
	// 			return c.sub(c.connections[i], []ChannelSubscription{*sub})
	// 		}
	// 	}
	// }
	c.Unlock()
	return errors.New("could not find subscription to unsubscribe from across all connections")
}

// Subscribe subscribes and sets subscription by stream connection
func (c *ConnectionManager) Subscribe(cs []SubscriptionParameters) error {
	c.Lock()
	defer c.Unlock()

	if c.subscriber == nil {
		return errors.New("exchange subscriber functionality not set, cannot subscribe")
	}

	if cs == nil {
		return errors.New("no subscription data cannot subscribe")
	}

	var errs common.Errors
	for i := range cs {
		err := c.subscriber(cs[i])
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}

	return nil
}

// AssociateAndUnsubscribe associates subscriptions that need to be unsubscribed
// with their respective connections
func (c *ConnectionManager) AssociateAndUnsubscribe(sub []ChannelSubscription) error {
	c.Lock()
	// for i := range c.connections {
	// 	subs := c.connections[i].GetAllSubscriptions()
	// 	for j := range subs {
	// 		if subs[j].Equal(sub) {
	// 			c.Unlock()
	// 			return c.unsub([]SubscriptionParameters{
	// 				{
	// 					Items: []ChannelSubscription{*sub},
	// 					Conn:  c.connections[i],
	// 				},
	// 			})
	// 		}
	// 	}
	// }
	c.Unlock()
	return errors.New("could not find subscription to unsubscribe from across all connections")
}

// Unsubscribe unsubscribes and removes subscription by stream connection
func (c *ConnectionManager) Unsubscribe(cs []SubscriptionParameters) error {
	c.Lock()
	defer c.Unlock()

	if c.unsubscriber == nil {
		return errors.New("exchange unsubscriber functionality not set, cannot unsubscribe")
	}

	if cs == nil {
		return errors.New("no subscription data cannot unsubscribe")
	}

	var errs common.Errors
	for i := range cs {
		err := c.unsubscriber(cs[i])
		if err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return errs
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

// ReadStream handles reading from the streaming end point
func (c *ConnectionManager) ReadStream(conn Connection, respHandler func([]byte, Connection) error) error {
	if conn == nil {
		return errors.New("connection cannot be nil")
	}

	if respHandler == nil {
		return errors.New("response handler cannot be nil")
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			resp := conn.ReadMessage()
			if resp.Raw == nil {
				return
			}
			go func() {
				err := respHandler(resp.Raw, conn)
				if err != nil {
					c.dataHandler <- err
				}
			}()
		}
	}()
	return nil
}

// Shutdown shuts down all associated connections
func (c *ConnectionManager) Shutdown() error {
	c.Lock()
	defer c.Unlock()

	var errs common.Errors
	for i := range c.connections {
		err := c.connections[i].Shutdown()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if errs != nil {
		return errs
	}
	return nil
}
