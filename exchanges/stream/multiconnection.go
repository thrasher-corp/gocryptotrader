package stream

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
)

var (
	errMultiConnectionFunctionalityRequired = errors.New("multi-connection functionality required")
	errMultiConnectionFailed                = errors.New("multi-connection failed")
	errUnsubscribeFailure                   = errors.New("failed to unsubscribe")
	errSubscribeFailure                     = errors.New("failed to subscribe")
	errSubscriptionNotFound                 = errors.New("subscription not found")
)

// Connections is a map of connection configurations to a slice of connections.
// This will allow for multiple connections to be set up and managed and allow
// it to expand and contract multiple connections for subscriptions.
type Connections map[*ConnectionSetup]*[]Connection

// Routes is a map that contains the connection setup and a map of subscriptions
// to connections. This is used for multi-connection management.
type Routes map[*ConnectionSetup]map[Connection]*[]subscription.Subscription

// multiSubscribe is a helper function that will subscribe to multiple connections
// based on the incoming subscriptions. It will check if there are any existing
// connections and append to them if the maximum subscriptions per connection
// has not been reached. If the maximum subscriptions per connection has been
// reached, it will spawn a new connection and subscribe to that.
func (w *Websocket) multiSubscribe(configuration *ConnectionSetup, conns *[]Connection, incoming []subscription.Subscription) error {
	if len(incoming) == 0 {
		return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errSubscribeFailure, errNoSubscriptionsSupplied)
	}

	err := w.checkSubscriptions(incoming)
	if err != nil {
		return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errSubscribeFailure, err)
	}

	left := 0
	actualsubs := w.GetSubscriptions()

	stateRoutes, err := getRoutesFromSubscriptions(actualsubs)
	if err != nil && !errors.Is(err, errNoSubscriptionsSupplied) {
		return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errSubscribeFailure, err)
	}

	// This determines if there are any connections already established and
	// if so, we will append to the existing connections.
	for existingConn, existingSubs := range stateRoutes[configuration] {
		if left >= len(incoming) {
			break
		}
		if w.MaxSubscriptionsPerConnection == 0 {
			// If there is no limit to the number of subscriptions per connection
			// then we will append to the existing connection.
			err = applyConnectionStateToSubscriptions(configuration, existingConn, incoming)
			if err != nil {
				return err
			}

			err = configuration.Subscriber(existingConn, incoming)
			if err != nil {
				return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errSubscribeFailure, err)
			}
			break
		}
		if len(*existingSubs) < w.MaxSubscriptionsPerConnection {
			// If the number of subscriptions on the existing connection is less
			// than the maximum allowed per connection, then we will append to
			// the existing connection.
			right := left + (w.MaxSubscriptionsPerConnection - len(*existingSubs))
			if right > len(incoming) {
				right = len(incoming)
			}

			connectionSubs := incoming[left:right]
			err = applyConnectionStateToSubscriptions(configuration, existingConn, connectionSubs)
			if err != nil {
				return err
			}

			err = configuration.Subscriber(existingConn, connectionSubs)
			if err != nil {
				return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errSubscribeFailure, err)
			}

			left = right
		}
		// If the number of subscriptions on the existing connection is equal
		// to or greater than the maximum allowed per connection, then we will
		// check another connection.
	}

	window := w.MaxSubscriptionsPerConnection
	if window == 0 || window > len(incoming) {
		window = len(incoming)
	}

	// Split subscriptions into windows
	for left := 0; left < len(incoming); left += window {
		right := left + window
		if right > len(incoming) {
			right = len(incoming)
		}

		connectionSubs := incoming[left:right]

		// Only spawn connection if there are subscriptions
		newConn := w.newWebsocketConnection(configuration)

		// Apply connection details to subscriptions
		err = applyConnectionStateToSubscriptions(configuration, newConn, connectionSubs)
		if err != nil {
			return err
		}

		// Append connection to connections pool which are mapped by
		// the connection setup configuration
		*conns = append(*conns, newConn)

		// Initiate the connection
		err = w.initConnection(newConn, configuration.Bootstrap, configuration.Handler)
		if err != nil {
			w.state.Store(disconnected)
			return fmt.Errorf("%v Error connecting %w", w.exchangeName, err)
		}

		// Subscribe
		go func() {
			err = configuration.Subscriber(newConn, incoming[left:right])
			if err != nil {
				fmt.Println("error subscribing")
			}
		}()
	}

	return nil
}

func (w *Websocket) multiUnsubscribe(incoming Routes) error {
	if len(incoming) == 0 {
		return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errUnsubscribeFailure, errNoSubscriptionsSupplied)
	}

	stateRoutes, err := getRoutesFromSubscriptions(w.GetSubscriptions())
	if err != nil {
		return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errUnsubscribeFailure, err)
	}

	for configuration, conns := range incoming {
		for conn, unsubs := range conns {
			err = configuration.Unsubscriber(conn, *unsubs)
			if err != nil {
				return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errUnsubscribeFailure, err)
			}

			existingsubs, ok := stateRoutes[configuration][conn]
			if !ok {
				return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errUnsubscribeFailure, errSubscriptionNotFound)
			}

			if len(*existingsubs)-len(*unsubs) == 0 {
				existingConnections, ok := w.Connections[configuration]
				if !ok {
					return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errUnsubscribeFailure, errSubscriptionNotFound)
				}

				err = conn.Shutdown()
				if err != nil {
					return fmt.Errorf("%w %w: %w ", errMultiConnectionFailed, errUnsubscribeFailure, err)
				}

				for i := range *existingConnections {
					if (*existingConnections)[i] == conn {
						(*existingConnections)[i] = (*existingConnections)[len(*existingConnections)-1]
						(*existingConnections)[len(*existingConnections)-1] = nil
						*existingConnections = (*existingConnections)[:len(*existingConnections)-1]
						break
					}
				}
			}
		}
	}

	return nil

}

// getRoutesFromSubscriptions is a handy little helper function that takes a
// slice of subscriptions and returns a map of connection setups to a map of
// connections to a slice of subscriptions. This is used for multi-connection
// management.
func getRoutesFromSubscriptions(subs []subscription.Subscription) (Routes, error) {
	if len(subs) == 0 {
		return nil, errNoSubscriptionsSupplied
	}

	routes := make(map[*ConnectionSetup]map[Connection]*[]subscription.Subscription)
	for i := range subs {
		key1, ok := subs[i].ConnectionSetup.(*ConnectionSetup)
		if !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidChannelState, subs[i].Channel)
		}

		key2, ok := subs[i].Connection.(Connection)
		if !ok {
			return nil, fmt.Errorf("%w: %v", errInvalidChannelState, subs[i].Channel)
		}

		configConns, ok := routes[key1]
		if !ok {
			configConns = make(map[Connection]*[]subscription.Subscription)
			routes[key1] = configConns
		}

		conns, ok := configConns[key2]
		if !ok {
			conns = &[]subscription.Subscription{}
			configConns[key2] = conns
		}

		*conns = append(*conns, subs[i])
	}

	return routes, nil
}

// newWebsocketConnection allocates a new websocket connection
func (w *Websocket) newWebsocketConnection(configuration *ConnectionSetup) Connection {
	pocType := "public"
	if configuration.Authenticated {
		pocType = "private"
	}
	return &WebsocketConnection{
		ExchangeName:      w.exchangeName,
		URL:               configuration.URL,
		Verbose:           w.verbose,
		ResponseMaxLimit:  configuration.ResponseMaxLimit,
		Traffic:           w.TrafficAlert,
		readMessageErrors: w.ReadMessageErrors,
		ShutdownC:         w.ShutdownC,
		Wg:                w.Wg,
		Match:             w.Match,
		RateLimit:         configuration.RateLimit,
		Reporter:          configuration.ConnectionLevelReporter,
		ReadBufferSize:    configuration.ReadBufferSize,
		WriteBufferSize:   configuration.WriteBufferSize,
		Type:              pocType,
	}
}

// applyConnectionToSubscriptions applies a connection to each individual
// subscriptions. This is used for multi-connection management.
func applyConnectionStateToSubscriptions(configuration *ConnectionSetup, conn Connection, subs []subscription.Subscription) error {
	if configuration == nil {
		return fmt.Errorf("%w: %w %T", errConnSetup, common.ErrNilPointer, configuration)
	}
	if conn == nil {
		return fmt.Errorf("%w: %w %T", errConnSetup, common.ErrNilPointer, conn)
	}
	if len(subs) == 0 {
		return fmt.Errorf("%w: %w", errConnSetup, errNoSubscriptionsSupplied)
	}
	for i := range subs {
		subs[i].ConnectionSetup = configuration
		subs[i].Connection = conn
	}
	return nil
}
