package stream

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
)

// ConnectionManager manages connections
type ConnectionManager struct {
	sync.RWMutex
	conn         map[Connection]*[]ChannelSubscription
	features     *protocol.Features
	connector    func(conn Connection) error
	generator    func(options SubscriptionOptions) ([]SubscriptionParamaters, error)
	subscriber   func(sub SubscriptionParamaters) error
	unsubscriber func(unsub SubscriptionParamaters) error
}

// ConnectionManagerConfig defines the needed variables for stream connections
type ConnectionManagerConfig struct {
	ExchangeConnector             func(conn Connection) error
	ExchangeGenerateSubscriptions func(options SubscriptionOptions) ([]SubscriptionParamaters, error)
	ExchangeSubscriber            func(sub SubscriptionParamaters) error
	ExchangeUnsubscriber          func(unsub SubscriptionParamaters) error
}

// SubscriptionParamaters defines payload for subscribing and unsibscribing
type SubscriptionParamaters struct {
	Items   []ChannelSubscription
	Conn    Connection
	Manager *ConnectionManager
}

// SubscriptionOptions defines subscriber options and updates
type SubscriptionOptions struct {
	Connections []Connection
	Features    *protocol.Features
	Manager     *ConnectionManager
}

// NewConnectionManager returns a new connection manager
func NewConnectionManager(cfg *ConnectionManagerConfig) (*ConnectionManager, error) {
	if cfg == nil {
		return nil, errors.New("configuration cannot be nil")
	}
	if cfg.ExchangeConnector == nil {
		return nil, errors.New("exchange connector function cannot be nil")
	}
	return &ConnectionManager{
		conn:         make(map[Connection]*[]ChannelSubscription),
		connector:    cfg.ExchangeConnector,
		generator:    cfg.ExchangeGenerateSubscriptions,
		subscriber:   cfg.ExchangeSubscriber,
		unsubscriber: cfg.ExchangeUnsubscriber,
	}, nil
}

// GenerateSubscriptions generates new connection profiles
func (c *ConnectionManager) GenerateSubscriptions() ([]SubscriptionParamaters, error) {
	c.Lock()
	defer c.Unlock()
	return c.generator(SubscriptionOptions{Features: c.features, Manager: c})
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
func (c *ConnectionManager) Connect(conn Connection) error {
	c.Lock()
	defer c.Unlock()

	err := c.connector(conn)
	if err != nil {
		return nil
	}
	return nil
}

// Subscribe subscribes and sets subscription by stream connection
func (c *ConnectionManager) Subscribe(subs []SubscriptionParamaters) error {
	c.Lock()
	defer c.Unlock()

	if c.subscriber == nil {
		return errors.New("exchange subscriber functionality not set, cannot subscribe")
	}

	if subs == nil {
		return errors.New("no subscription data cannot subscribe")
	}

	for i := range subs {
		err := c.subscriber(subs[i])
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
func (c *ConnectionManager) GetAllSubscriptions() ([]ChannelSubscription, error) {
	c.RLock()
	defer c.RUnlock()
	return nil, errors.New("life is complicated")
}

// GetAssetByConnectionSubscription returns connection channel asset
func (c *ConnectionManager) GetAssetByConnectionSubscription(conn Connection, channel string) (asset.Items, error) {
	c.RLock()
	defer c.RUnlock()
	return nil, errors.New("life is uber complicated")
}

// AddSuccessfulSubscriptions adds subs mate
func (c *ConnectionManager) AddSuccessfulSubscriptions(conn Connection, sub []ChannelSubscription) error {
	c.Lock()
	defer c.Unlock()
	val, ok := c.conn[conn]
	if !ok {
		return errors.New("connection not set in manager")
	}

	if val == nil {
		val = &sub
		return nil
	}

	for x := range sub {
		for y := range *val {
			if ([]ChannelSubscription)(*val)[y].Channel == sub[x].Channel {
				return errors.New("love it")
			}
		}
		*val = append(*val, sub[x])
	}
	return nil
}

// RemoveSuccessfulUnsubscriptions removes subs mate
func (c *ConnectionManager) RemoveSuccessfulUnsubscriptions(conn Connection, unsub []ChannelSubscription) error {
	c.Lock()
	defer c.Unlock()
	val, ok := c.conn[conn]
	if !ok {
		return errors.New("connection not set in manager")
	}

	if val == nil {
		return errors.New("channel subs have nothing")
	}

	for x := range unsub {
		for y := range *val {
			if ([]ChannelSubscription)(*val)[y].Channel == unsub[x].Channel {

				// delete and make sure it gets garbo collected
			}
		}
		return errors.New("subscription not found")
	}
	return nil
}
