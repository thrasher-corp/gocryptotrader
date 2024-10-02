package stream

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	jobBuffer = 5000
)

// Public websocket errors
var (
	ErrWebsocketNotEnabled      = errors.New("websocket not enabled")
	ErrSubscriptionFailure      = errors.New("subscription failure")
	ErrSubscriptionNotSupported = errors.New("subscription channel not supported ")
	ErrUnsubscribeFailure       = errors.New("unsubscribe failure")
	ErrAlreadyDisabled          = errors.New("websocket already disabled")
	ErrNotConnected             = errors.New("websocket is not connected")
	ErrNoMessageListener        = errors.New("websocket listener not found for message")
	ErrSignatureTimeout         = errors.New("websocket timeout waiting for response with signature")
)

// Private websocket errors
var (
	errAlreadyRunning                       = errors.New("connection monitor is already running")
	errExchangeConfigIsNil                  = errors.New("exchange config is nil")
	errWebsocketIsNil                       = errors.New("websocket is nil")
	errWebsocketSetupIsNil                  = errors.New("websocket setup is nil")
	errWebsocketAlreadyInitialised          = errors.New("websocket already initialised")
	errWebsocketAlreadyEnabled              = errors.New("websocket already enabled")
	errWebsocketFeaturesIsUnset             = errors.New("websocket features is unset")
	errConfigFeaturesIsNil                  = errors.New("exchange config features is nil")
	errDefaultURLIsEmpty                    = errors.New("default url is empty")
	errRunningURLIsEmpty                    = errors.New("running url cannot be empty")
	errInvalidWebsocketURL                  = errors.New("invalid websocket url")
	errExchangeConfigNameEmpty              = errors.New("exchange config name empty")
	errInvalidTrafficTimeout                = errors.New("invalid traffic timeout")
	errTrafficAlertNil                      = errors.New("traffic alert is nil")
	errWebsocketSubscriberUnset             = errors.New("websocket subscriber function needs to be set")
	errWebsocketUnsubscriberUnset           = errors.New("websocket unsubscriber functionality allowed but unsubscriber function not set")
	errWebsocketConnectorUnset              = errors.New("websocket connector function not set")
	errWebsocketDataHandlerUnset            = errors.New("websocket data handler not set")
	errReadMessageErrorsNil                 = errors.New("read message errors is nil")
	errWebsocketSubscriptionsGeneratorUnset = errors.New("websocket subscriptions generator function needs to be set")
	errClosedConnection                     = errors.New("use of closed network connection")
	errSubscriptionsExceedsLimit            = errors.New("subscriptions exceeds limit")
	errInvalidMaxSubscriptions              = errors.New("max subscriptions cannot be less than 0")
	errSameProxyAddress                     = errors.New("cannot set proxy address to the same address")
	errNoConnectFunc                        = errors.New("websocket connect func not set")
	errAlreadyConnected                     = errors.New("websocket already connected")
	errCannotShutdown                       = errors.New("websocket cannot shutdown")
	errAlreadyReconnecting                  = errors.New("websocket in the process of reconnection")
	errConnSetup                            = errors.New("error in connection setup")
	errNoPendingConnections                 = errors.New("no pending connections, call SetupNewConnection first")
	errConnectionWrapperDuplication         = errors.New("connection wrapper duplication")
	errCannotChangeConnectionURL            = errors.New("cannot change connection URL when using multi connection management")
	errExchangeConfigEmpty                  = errors.New("exchange config is empty")
)

var globalReporter Reporter

// SetupGlobalReporter sets a reporter interface to be used
// for all exchange requests
func SetupGlobalReporter(r Reporter) {
	globalReporter = r
}

// NewWebsocket initialises the websocket struct
func NewWebsocket() *Websocket {
	return &Websocket{
		DataHandler:  make(chan interface{}, jobBuffer),
		ToRoutine:    make(chan interface{}, jobBuffer),
		ShutdownC:    make(chan struct{}),
		TrafficAlert: make(chan struct{}, 1),
		// ReadMessageErrors is buffered for an edge case when `Connect` fails
		// after subscriptions are made but before the connectionMonitor has
		// started. This allows the error to be read and handled in the
		// connectionMonitor and start a connection cycle again.
		ReadMessageErrors: make(chan error, 1),
		Match:             NewMatch(),
		subscriptions:     subscription.NewStore(),
		features:          &protocol.Features{},
		Orderbook:         buffer.Orderbook{},
		connections:       make(map[Connection]*ConnectionWrapper),
	}
}

// Setup sets main variables for websocket connection
func (w *Websocket) Setup(s *WebsocketSetup) error {
	if w == nil {
		return errWebsocketIsNil
	}

	if s == nil {
		return errWebsocketSetupIsNil
	}

	w.m.Lock()
	defer w.m.Unlock()

	if w.IsInitialised() {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketAlreadyInitialised)
	}

	if s.ExchangeConfig == nil {
		return errExchangeConfigIsNil
	}

	if s.ExchangeConfig.Name == "" {
		return errExchangeConfigNameEmpty
	}
	w.exchangeName = s.ExchangeConfig.Name
	w.verbose = s.ExchangeConfig.Verbose

	if s.Features == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketFeaturesIsUnset)
	}
	w.features = s.Features

	if s.ExchangeConfig.Features == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errConfigFeaturesIsNil)
	}
	w.setEnabled(s.ExchangeConfig.Features.Enabled.Websocket)

	w.useMultiConnectionManagement = s.UseMultiConnectionManagement

	if !w.useMultiConnectionManagement {
		// TODO: Remove this block when all exchanges are updated and backwards
		// compatibility is no longer required.
		if s.Connector == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketConnectorUnset)
		}
		if s.Subscriber == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriberUnset)
		}
		if s.Unsubscriber == nil && w.features.Unsubscribe {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketUnsubscriberUnset)
		}
		if s.GenerateSubscriptions == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriptionsGeneratorUnset)
		}
		if s.DefaultURL == "" {
			return fmt.Errorf("%s websocket %w", w.exchangeName, errDefaultURLIsEmpty)
		}
		w.defaultURL = s.DefaultURL
		if s.RunningURL == "" {
			return fmt.Errorf("%s websocket %w", w.exchangeName, errRunningURLIsEmpty)
		}

		w.connector = s.Connector
		w.Subscriber = s.Subscriber
		w.Unsubscriber = s.Unsubscriber
		w.GenerateSubs = s.GenerateSubscriptions

		err := w.SetWebsocketURL(s.RunningURL, false, false)
		if err != nil {
			return fmt.Errorf("%s %w", w.exchangeName, err)
		}

		if s.RunningURLAuth != "" {
			err = w.SetWebsocketURL(s.RunningURLAuth, true, false)
			if err != nil {
				return fmt.Errorf("%s %w", w.exchangeName, err)
			}
		}
	}

	w.connectionMonitorDelay = s.ExchangeConfig.ConnectionMonitorDelay
	if w.connectionMonitorDelay <= 0 {
		w.connectionMonitorDelay = config.DefaultConnectionMonitorDelay
	}

	if s.ExchangeConfig.WebsocketTrafficTimeout < time.Second {
		return fmt.Errorf("%s %w cannot be less than %s",
			w.exchangeName,
			errInvalidTrafficTimeout,
			time.Second)
	}
	w.trafficTimeout = s.ExchangeConfig.WebsocketTrafficTimeout

	w.SetCanUseAuthenticatedEndpoints(s.ExchangeConfig.API.AuthenticatedWebsocketSupport)

	if err := w.Orderbook.Setup(s.ExchangeConfig, &s.OrderbookBufferConfig, w.DataHandler); err != nil {
		return err
	}

	w.Trade.Setup(w.exchangeName, s.TradeFeed, w.DataHandler)
	w.Fills.Setup(s.FillsFeed, w.DataHandler)

	if s.MaxWebsocketSubscriptionsPerConnection < 0 {
		return fmt.Errorf("%s %w", w.exchangeName, errInvalidMaxSubscriptions)
	}
	w.MaxSubscriptionsPerConnection = s.MaxWebsocketSubscriptionsPerConnection
	w.setState(disconnectedState)

	w.rateLimitDefinitions = s.RateLimitDefinitions
	return nil
}

// SetupNewConnection sets up an auth or unauth streaming connection
func (w *Websocket) SetupNewConnection(c *ConnectionSetup) error {
	if w == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errWebsocketIsNil)
	}

	if c == nil || c.ResponseCheckTimeout == 0 &&
		c.ResponseMaxLimit == 0 &&
		c.RateLimit == nil &&
		c.URL == "" &&
		c.ConnectionLevelReporter == nil &&
		c.BespokeGenerateMessageID == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errExchangeConfigEmpty)
	}

	if w.exchangeName == "" {
		return fmt.Errorf("%w: %w", errConnSetup, errExchangeConfigNameEmpty)
	}
	if w.TrafficAlert == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errTrafficAlertNil)
	}
	if w.ReadMessageErrors == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errReadMessageErrorsNil)
	}
	if c.ConnectionLevelReporter == nil {
		c.ConnectionLevelReporter = w.ExchangeLevelReporter
	}
	if c.ConnectionLevelReporter == nil {
		c.ConnectionLevelReporter = globalReporter
	}

	if w.useMultiConnectionManagement {
		// The connection and supporting functions are defined per connection
		// and the connection wrapper is stored in the connection manager.
		if c.URL == "" {
			return fmt.Errorf("%w: %w", errConnSetup, errDefaultURLIsEmpty)
		}
		if c.Connector == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketConnectorUnset)
		}
		if c.GenerateSubscriptions == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriptionsGeneratorUnset)
		}
		if c.Subscriber == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriberUnset)
		}
		if c.Unsubscriber == nil && w.features.Unsubscribe {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketUnsubscriberUnset)
		}
		if c.Handler == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketDataHandlerUnset)
		}

		for x := range w.connectionManager {
			if w.connectionManager[x].Setup.URL == c.URL {
				return fmt.Errorf("%w: %w", errConnSetup, errConnectionWrapperDuplication)
			}
		}

		w.connectionManager = append(w.connectionManager, ConnectionWrapper{
			Setup:         c,
			Subscriptions: subscription.NewStore(),
		})
		return nil
	}

	if c.Authenticated {
		w.AuthConn = w.getConnectionFromSetup(c)
	} else {
		w.Conn = w.getConnectionFromSetup(c)
	}

	return nil
}

// getConnectionFromSetup returns a websocket connection from a setup
// configuration. This is used for setting up new connections on the fly.
func (w *Websocket) getConnectionFromSetup(c *ConnectionSetup) *WebsocketConnection {
	connectionURL := w.GetWebsocketURL()
	if c.URL != "" {
		connectionURL = c.URL
	}
	return &WebsocketConnection{
		ExchangeName:             w.exchangeName,
		URL:                      connectionURL,
		ProxyURL:                 w.GetProxyAddress(),
		Verbose:                  w.verbose,
		ResponseMaxLimit:         c.ResponseMaxLimit,
		Traffic:                  w.TrafficAlert,
		readMessageErrors:        w.ReadMessageErrors,
		shutdown:                 w.ShutdownC,
		Wg:                       &w.Wg,
		Match:                    w.Match,
		RateLimit:                c.RateLimit,
		Reporter:                 c.ConnectionLevelReporter,
		bespokeGenerateMessageID: c.BespokeGenerateMessageID,
		RateLimitDefinitions:     w.rateLimitDefinitions,
	}
}

// Connect initiates a websocket connection by using a package defined connection
// function
func (w *Websocket) Connect() error {
	w.m.Lock()
	defer w.m.Unlock()
	return w.connect()
}

func (w *Websocket) connect() error {
	if !w.IsEnabled() {
		return ErrWebsocketNotEnabled
	}
	if w.IsConnecting() {
		return fmt.Errorf("%v %w", w.exchangeName, errAlreadyReconnecting)
	}
	if w.IsConnected() {
		return fmt.Errorf("%v %w", w.exchangeName, errAlreadyConnected)
	}

	if w.subscriptions == nil {
		return fmt.Errorf("%w: subscriptions", common.ErrNilPointer)
	}
	w.subscriptions.Clear()

	w.setState(connectingState)

	w.Wg.Add(2)
	go w.monitorFrame(&w.Wg, w.monitorData)
	go w.monitorFrame(&w.Wg, w.monitorTraffic)

	if !w.useMultiConnectionManagement {
		if w.connector == nil {
			return fmt.Errorf("%v %w", w.exchangeName, errNoConnectFunc)
		}
		err := w.connector()
		if err != nil {
			w.setState(disconnectedState)
			return fmt.Errorf("%v Error connecting %w", w.exchangeName, err)
		}
		w.setState(connectedState)

		if w.connectionMonitorRunning.CompareAndSwap(false, true) {
			// This oversees all connections and does not need to be part of wait group management.
			go w.monitorFrame(nil, w.monitorConnection)
		}

		subs, err := w.GenerateSubs() // regenerate state on new connection
		if err != nil {
			return fmt.Errorf("%s websocket: %w", w.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
		}
		if len(subs) != 0 {
			if err := w.SubscribeToChannels(nil, subs); err != nil {
				return err
			}
		}
		return nil
	}

	if len(w.connectionManager) == 0 {
		w.setState(disconnectedState)
		return fmt.Errorf("cannot connect: %w", errNoPendingConnections)
	}

	// multiConnectFatalError is a fatal error that will cause all connections to
	// be shutdown and the websocket to be disconnected.
	var multiConnectFatalError error
	var m sync.Mutex
	wg := sync.WaitGroup{}
	subResult := make(map[Connection]subscription.List)
	for i := range w.connectionManager {
		if w.connectionManager[i].Setup.GenerateSubscriptions == nil {
			m.Lock()
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, w.connectionManager[i].Setup.URL, errWebsocketSubscriptionsGeneratorUnset)
			m.Unlock()
			break
		}

		subs, err := w.connectionManager[i].Setup.GenerateSubscriptions() // regenerate state on new connection
		if err != nil {
			m.Lock()
			multiConnectFatalError = fmt.Errorf("%s websocket: %w", w.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
			m.Unlock()
			break
		}

		if len(subs) == 0 {
			// If no subscriptions are generated, we skip the connection
			if w.verbose {
				log.Warnf(log.WebsocketMgr, "%s websocket: no subscriptions generated", w.exchangeName)
			}
			continue
		}

		if w.connectionManager[i].Setup.Connector == nil {
			m.Lock()
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, w.connectionManager[i].Setup.URL, errNoConnectFunc)
			m.Unlock()
			break
		}
		if w.connectionManager[i].Setup.Handler == nil {
			m.Lock()
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, w.connectionManager[i].Setup.URL, errWebsocketDataHandlerUnset)
			m.Unlock()
			break
		}
		if w.connectionManager[i].Setup.Subscriber == nil {
			m.Lock()
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, w.connectionManager[i].Setup.URL, errWebsocketSubscriberUnset)
			m.Unlock()
			break
		}

		// TODO: Add window for max subscriptions per connection, to spawn new connections if needed.

		conn := w.getConnectionFromSetup(w.connectionManager[i].Setup)

		// Set these values right away, as any failure further down the line will clean
		// the values anyway, but they can be done in parallel.
		w.connectionManager[i].Connection = conn
		w.connections[conn] = &w.connectionManager[i]

		// This connection function cannot be in a go routine as there could be
		// supplementary routines generated within that need to be inline with
		// wg.Add calls which could leak. e.g. ping handler, bespoke connection
		// logic.
		err = w.connectionManager[i].Setup.Connector(context.TODO(), conn)
		if err != nil {
			m.Lock()
			multiConnectFatalError = fmt.Errorf("%v Error connecting %w", w.exchangeName, err)
			m.Unlock()
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if !conn.IsConnected() {
				m.Lock()
				multiConnectFatalError = fmt.Errorf("%s websocket: [conn:%d] [URL:%s] failed to connect", w.exchangeName, i+1, conn.URL)
				m.Unlock()
				return
			}

			w.Wg.Add(1)
			go w.Reader(context.TODO(), conn, w.connectionManager[i].Setup.Handler)

			var result *subscription.Result
			result, err = w.connectionManager[i].Setup.Subscriber(context.TODO(), conn, subs)
			if err != nil {
				m.Lock()
				multiConnectFatalError = fmt.Errorf("%v Error subscribing %w", w.exchangeName, err)
				m.Unlock()
				return
			}

			if issues := result.GetUnsuccessful(); len(issues) > 0 {
				m.Lock()
				multiConnectFatalError = fmt.Errorf("%v %w: %v", w.exchangeName, ErrSubscriptionFailure, issues)
				m.Unlock()
			}

			m.Lock()
			subResult[conn] = result.GetSuccessful()
			m.Unlock()

			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%s websocket: [conn:%d] [URL:%s] connected. [Subscribed: %d]",
					w.exchangeName,
					i+1,
					conn.URL,
					len(subs))
			}
		}()
	}

	wg.Wait()

	if multiConnectFatalError != nil {
		// Roll back any successful connections and flush subscriptions
		for x := range w.connectionManager {
			if w.connectionManager[x].Connection != nil {
				if err := w.connectionManager[x].Connection.Shutdown(); err != nil {
					log.Errorln(log.WebsocketMgr, err)
				}
				w.connectionManager[x].Connection = nil
			}
			w.connectionManager[x].Subscriptions.Clear()
		}
		close(w.ShutdownC)
		w.Wg.Wait()
		clear(w.connections)
		w.setState(disconnectedState) // Flip from connecting to disconnected.
		w.ShutdownC = make(chan struct{})

		// Drain residual error in the single buffered channel, this mitigates
		// the cycle when `Connect` is called again and the connectionMonitor
		// starts but there is an old error in the channel.
		drain(w.ReadMessageErrors)

		return multiConnectFatalError
	}

	for conn, result := range subResult {
		if err := w.AddSuccessfulSubscriptions(conn, result...); err != nil {
			return err
		}
	}

	// Assume connected state here. All connections have been established.
	// All subscriptions have been sent and stored. All data received is being
	// handled by the appropriate data handler.
	w.setState(connectedState)

	if w.connectionMonitorRunning.CompareAndSwap(false, true) {
		// This oversees all connections and does not need to be part of wait group management.
		go w.monitorFrame(nil, w.monitorConnection)
	}

	return nil
}

// Disable disables the exchange websocket protocol
// Note that connectionMonitor will be responsible for shutting down the websocket after disabling
func (w *Websocket) Disable() error {
	if !w.IsEnabled() {
		return fmt.Errorf("%s %w", w.exchangeName, ErrAlreadyDisabled)
	}

	w.setEnabled(false)
	return nil
}

// Enable enables the exchange websocket protocol
func (w *Websocket) Enable() error {
	if w.IsConnected() || w.IsEnabled() {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketAlreadyEnabled)
	}

	w.setEnabled(true)
	return w.Connect()
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *Websocket) Shutdown() error {
	w.m.Lock()
	defer w.m.Unlock()
	return w.shutdown()
}

func (w *Websocket) shutdown() error {
	if !w.IsConnected() {
		return fmt.Errorf("%v %w: %w", w.exchangeName, errCannotShutdown, ErrNotConnected)
	}

	// TODO: Interrupt connection and or close connection when it is re-established.
	if w.IsConnecting() {
		return fmt.Errorf("%v %w: %w ", w.exchangeName, errCannotShutdown, errAlreadyReconnecting)
	}

	if w.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: shutting down websocket", w.exchangeName)
	}

	defer w.Orderbook.FlushBuffer()

	// During the shutdown process, all errors are treated as non-fatal to avoid issues when the connection has already
	// been closed. In such cases, attempting to close the connection may result in a
	// "failed to send closeNotify alert (but connection was closed anyway)" error. Treating these errors as non-fatal
	// prevents the shutdown process from being interrupted, which could otherwise trigger a continuous traffic monitor
	// cycle and potentially block the initiation of a new connection.
	var nonFatalCloseConnectionErrors error

	// Shutdown managed connections
	for x := range w.connectionManager {
		if w.connectionManager[x].Connection != nil {
			if err := w.connectionManager[x].Connection.Shutdown(); err != nil {
				nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
			}
			w.connectionManager[x].Connection = nil
			// Flush any subscriptions from last connection across any managed connections
			w.connectionManager[x].Subscriptions.Clear()
		}
	}
	// Clean map of old connections
	clear(w.connections)

	if w.Conn != nil {
		if err := w.Conn.Shutdown(); err != nil {
			nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
		}
	}
	if w.AuthConn != nil {
		if err := w.AuthConn.Shutdown(); err != nil {
			nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
		}
	}
	// flush any subscriptions from last connection if needed
	w.subscriptions.Clear()

	w.setState(disconnectedState)

	close(w.ShutdownC)
	w.Wg.Wait()
	w.ShutdownC = make(chan struct{})
	if w.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: completed websocket shutdown", w.exchangeName)
	}

	// Drain residual error in the single buffered channel, this mitigates
	// the cycle when `Connect` is called again and the connectionMonitor
	// starts but there is an old error in the channel.
	drain(w.ReadMessageErrors)

	if nonFatalCloseConnectionErrors != nil {
		log.Warnf(log.WebsocketMgr, "%v websocket: shutdown error: %v", w.exchangeName, nonFatalCloseConnectionErrors)
	}

	return nil
}

// FlushChannels flushes channel subscriptions when there is a pair/asset change
func (w *Websocket) FlushChannels() error {
	if !w.IsEnabled() {
		return fmt.Errorf("%s %w", w.exchangeName, ErrWebsocketNotEnabled)
	}

	if !w.IsConnected() {
		return fmt.Errorf("%s %w", w.exchangeName, ErrNotConnected)
	}

	// If the exchange does not support subscribing and or unsubscribing the full connection needs to be flushed to
	// maintain consistency.
	if !w.features.Subscribe || !w.features.Unsubscribe {
		w.m.Lock()
		defer w.m.Unlock()
		if err := w.shutdown(); err != nil {
			return err
		}
		return w.connect()
	}

	if !w.useMultiConnectionManagement {
		newSubs, err := w.GenerateSubs()
		if err != nil {
			return err
		}
		subs, unsubs := w.GetChannelDifference(nil, newSubs)
		if err := w.UnsubscribeChannels(nil, unsubs); err != nil {
			return err
		}
		if len(subs) == 0 {
			return nil
		}
		return w.SubscribeToChannels(nil, subs)
	}

	for x := range w.connectionManager {
		newSubs, err := w.connectionManager[x].Setup.GenerateSubscriptions()
		if err != nil {
			return err
		}

		// Case if there is nothing to unsubscribe from and the connection is nil
		if len(newSubs) == 0 && w.connectionManager[x].Connection == nil {
			continue
		}

		// If there are subscriptions to subscribe to but no connection to subscribe to, establish a new connection.
		if w.connectionManager[x].Connection == nil {
			conn := w.getConnectionFromSetup(w.connectionManager[x].Setup)
			if err := w.connectionManager[x].Setup.Connector(context.TODO(), conn); err != nil {
				return err
			}
			w.Wg.Add(1)
			go w.Reader(context.TODO(), conn, w.connectionManager[x].Setup.Handler)
			w.connections[conn] = &w.connectionManager[x]
			w.connectionManager[x].Connection = conn
		}

		subs, unsubs := w.GetChannelDifference(w.connectionManager[x].Connection, newSubs)

		if len(unsubs) != 0 {
			if err := w.UnsubscribeChannels(w.connectionManager[x].Connection, unsubs); err != nil {
				return err
			}
		}
		if len(subs) != 0 {
			if err := w.SubscribeToChannels(w.connectionManager[x].Connection, subs); err != nil {
				return err
			}
		}

		// If there are no subscriptions to subscribe to, close the connection as it is no longer needed.
		if w.connectionManager[x].Subscriptions.Len() == 0 {
			delete(w.connections, w.connectionManager[x].Connection) // Remove from lookup map
			if err := w.connectionManager[x].Connection.Shutdown(); err != nil {
				log.Warnf(log.WebsocketMgr, "%v websocket: failed to shutdown connection: %v", w.exchangeName, err)
			}
			w.connectionManager[x].Connection = nil
		}
	}
	return nil
}

func (w *Websocket) setState(s uint32) {
	w.state.Store(s)
}

// IsInitialised returns whether the websocket has been Setup() already
func (w *Websocket) IsInitialised() bool {
	return w.state.Load() != uninitialisedState
}

// IsConnected returns whether the websocket is connected
func (w *Websocket) IsConnected() bool {
	return w.state.Load() == connectedState
}

// IsConnecting returns whether the websocket is connecting
func (w *Websocket) IsConnecting() bool {
	return w.state.Load() == connectingState
}

func (w *Websocket) setEnabled(b bool) {
	w.enabled.Store(b)
}

// IsEnabled returns whether the websocket is enabled
func (w *Websocket) IsEnabled() bool {
	return w.enabled.Load()
}

// CanUseAuthenticatedWebsocketForWrapper Handles a common check to
// verify whether a wrapper can use an authenticated websocket endpoint
func (w *Websocket) CanUseAuthenticatedWebsocketForWrapper() bool {
	if w.IsConnected() {
		if w.CanUseAuthenticatedEndpoints() {
			return true
		}
		log.Infof(log.WebsocketMgr, WebsocketNotAuthenticatedUsingRest, w.exchangeName)
	}
	return false
}

// SetWebsocketURL sets websocket URL and can refresh underlying connections
func (w *Websocket) SetWebsocketURL(url string, auth, reconnect bool) error {
	if w.useMultiConnectionManagement {
		// TODO: Add functionality for multi-connection management to change URL
		return fmt.Errorf("%s: %w", w.exchangeName, errCannotChangeConnectionURL)
	}
	defaultVals := url == "" || url == config.WebsocketURLNonDefaultMessage
	if auth {
		if defaultVals {
			url = w.defaultURLAuth
		}

		err := checkWebsocketURL(url)
		if err != nil {
			return err
		}
		w.runningURLAuth = url

		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: setting authenticated websocket URL: %s\n", w.exchangeName, url)
		}

		if w.AuthConn != nil {
			w.AuthConn.SetURL(url)
		}
	} else {
		if defaultVals {
			url = w.defaultURL
		}
		err := checkWebsocketURL(url)
		if err != nil {
			return err
		}
		w.runningURL = url

		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: setting unauthenticated websocket URL: %s\n", w.exchangeName, url)
		}

		if w.Conn != nil {
			w.Conn.SetURL(url)
		}
	}

	if w.IsConnected() && reconnect {
		log.Debugf(log.WebsocketMgr, "%s websocket: flushing websocket connection to %s\n", w.exchangeName, url)
		return w.Shutdown()
	}
	return nil
}

// GetWebsocketURL returns the running websocket URL
func (w *Websocket) GetWebsocketURL() string {
	return w.runningURL
}

// SetProxyAddress sets websocket proxy address
func (w *Websocket) SetProxyAddress(proxyAddr string) error {
	w.m.Lock()

	if proxyAddr != "" {
		if _, err := url.ParseRequestURI(proxyAddr); err != nil {
			w.m.Unlock()
			return fmt.Errorf("%v websocket: cannot set proxy address: %w", w.exchangeName, err)
		}

		if w.proxyAddr == proxyAddr {
			w.m.Unlock()
			return fmt.Errorf("%v websocket: %w '%v'", w.exchangeName, errSameProxyAddress, w.proxyAddr)
		}

		log.Debugf(log.ExchangeSys, "%s websocket: setting websocket proxy: %s", w.exchangeName, proxyAddr)
	} else {
		log.Debugf(log.ExchangeSys, "%s websocket: removing websocket proxy", w.exchangeName)
	}

	for _, wrapper := range w.connectionManager {
		if wrapper.Connection != nil {
			wrapper.Connection.SetProxy(proxyAddr)
		}
	}
	if w.Conn != nil {
		w.Conn.SetProxy(proxyAddr)
	}
	if w.AuthConn != nil {
		w.AuthConn.SetProxy(proxyAddr)
	}

	w.proxyAddr = proxyAddr

	if w.IsConnected() {
		w.m.Unlock()
		if err := w.Shutdown(); err != nil {
			return err
		}
		return w.Connect()
	}

	w.m.Unlock()

	return nil
}

// GetProxyAddress returns the current websocket proxy
func (w *Websocket) GetProxyAddress() string {
	return w.proxyAddr
}

// GetName returns exchange name
func (w *Websocket) GetName() string {
	return w.exchangeName
}

// GetChannelDifference finds the difference between the subscribed channels
// and the new subscription list when pairs are disabled or enabled.
func (w *Websocket) GetChannelDifference(conn Connection, newSubs subscription.List) (sub, unsub subscription.List) {
	var subscriptionStore **subscription.Store
	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		subscriptionStore = &wrapper.Subscriptions
	} else {
		subscriptionStore = &w.subscriptions
	}
	if *subscriptionStore == nil {
		*subscriptionStore = subscription.NewStore()
	}
	return (*subscriptionStore).Diff(newSubs)
}

// UnsubscribeChannels unsubscribes from a list of websocket channel
func (w *Websocket) UnsubscribeChannels(conn Connection, channels subscription.List) error {
	if len(channels) == 0 {
		return nil // No channels to unsubscribe from is not an error
	}
	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		return w.unsubscribe(wrapper.Subscriptions, channels, func(channels subscription.List) error {
			result, err := wrapper.Setup.Unsubscriber(context.TODO(), conn, channels)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrUnsubscribeFailure, err)
			}
			err = w.RemoveSubscriptions(conn, result.GetSuccessful()...)
			if err != nil {
				return err
			}
			if issues := result.GetUnsuccessful(); len(issues) > 0 {
				return fmt.Errorf("%w: %v", ErrSubscriptionFailure, issues)
			}
			return nil
		})
	}
	return w.unsubscribe(w.subscriptions, channels, func(channels subscription.List) error {
		return w.Unsubscriber(channels)
	})
}

func (w *Websocket) unsubscribe(store *subscription.Store, channels subscription.List, unsub func(channels subscription.List) error) error {
	if store == nil {
		return nil // No channels to unsubscribe from is not an error
	}
	for _, s := range channels {
		if store.Get(s) == nil {
			return fmt.Errorf("%w: %s", subscription.ErrNotFound, s)
		}
	}
	return unsub(channels)
}

// ResubscribeToChannel resubscribes to channel
// Sets state to Resubscribing, and exchanges which want to maintain a lock on it can respect this state and not RemoveSubscription
// Errors if subscription is already subscribing
func (w *Websocket) ResubscribeToChannel(conn Connection, s *subscription.Subscription) error {
	l := subscription.List{s}
	if err := s.SetState(subscription.ResubscribingState); err != nil {
		return fmt.Errorf("%w: %s", err, s)
	}
	if err := w.UnsubscribeChannels(conn, l); err != nil {
		return err
	}
	return w.SubscribeToChannels(conn, l)
}

// SubscribeToChannels subscribes to websocket channels using the exchange specific Subscriber method
// Errors are returned for duplicates or exceeding max Subscriptions
func (w *Websocket) SubscribeToChannels(conn Connection, subs subscription.List) error {
	if slices.Contains(subs, nil) {
		return fmt.Errorf("%w: List parameter contains an nil element", common.ErrNilPointer)
	}
	if err := w.checkSubscriptions(conn, subs); err != nil {
		return err
	}

	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		result, err := wrapper.Setup.Subscriber(context.TODO(), conn, subs)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSubscriptionFailure, err)
		}
		err = w.AddSuccessfulSubscriptions(conn, result.GetSuccessful()...)
		if err != nil {
			return err
		}
		if issues := result.GetUnsuccessful(); len(issues) > 0 {
			return fmt.Errorf("%w: %v", ErrSubscriptionFailure, issues)
		}
		return nil
	}

	if w.Subscriber == nil {
		return fmt.Errorf("%w: Global Subscriber not set", common.ErrNilPointer)
	}

	if err := w.Subscriber(subs); err != nil {
		return fmt.Errorf("%w: %w", ErrSubscriptionFailure, err)
	}
	return nil
}

// AddSubscriptions adds subscriptions to the subscription store
// Sets state to Subscribing unless the state is already set
func (w *Websocket) AddSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if w == nil {
		return fmt.Errorf("%w: AddSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	var subscriptionStore **subscription.Store
	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		subscriptionStore = &wrapper.Subscriptions
	} else {
		subscriptionStore = &w.subscriptions
	}

	if *subscriptionStore == nil {
		*subscriptionStore = subscription.NewStore()
	}
	var errs error
	for _, s := range subs {
		if s.State() == subscription.InactiveState {
			if err := s.SetState(subscription.SubscribingState); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
			}
		}
		if err := (*subscriptionStore).Add(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// AddSuccessfulSubscriptions marks subscriptions as subscribed and adds them to the subscription store
func (w *Websocket) AddSuccessfulSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if w == nil {
		return fmt.Errorf("%w: AddSuccessfulSubscriptions called on nil Websocket", common.ErrNilPointer)
	}

	var subscriptionStore **subscription.Store
	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		subscriptionStore = &wrapper.Subscriptions
	} else {
		subscriptionStore = &w.subscriptions
	}

	if *subscriptionStore == nil {
		*subscriptionStore = subscription.NewStore()
	}

	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.SubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if err := (*subscriptionStore).Add(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// RemoveSubscriptions removes subscriptions from the subscription list and sets the status to Unsubscribed
func (w *Websocket) RemoveSubscriptions(conn Connection, subs ...*subscription.Subscription) error {
	if w == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on nil Websocket", common.ErrNilPointer)
	}

	var subscriptionStore *subscription.Store
	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		subscriptionStore = wrapper.Subscriptions
	} else {
		subscriptionStore = w.subscriptions
	}

	if subscriptionStore == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on uninitialised Websocket", common.ErrNilPointer)
	}

	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.UnsubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if err := subscriptionStore.Remove(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// GetSubscription returns a subscription at the key provided
// returns nil if no subscription is at that key or the key is nil
// Keys can implement subscription.MatchableKey in order to provide custom matching logic
func (w *Websocket) GetSubscription(key any) *subscription.Subscription {
	if w == nil || key == nil {
		return nil
	}
	for _, c := range w.connectionManager {
		if c.Subscriptions == nil {
			continue
		}
		sub := c.Subscriptions.Get(key)
		if sub != nil {
			return sub
		}
	}
	if w.subscriptions == nil {
		return nil
	}
	return w.subscriptions.Get(key)
}

// GetSubscriptions returns a new slice of the subscriptions
func (w *Websocket) GetSubscriptions() subscription.List {
	if w == nil {
		return nil
	}
	var subs subscription.List
	for _, c := range w.connectionManager {
		if c.Subscriptions != nil {
			subs = append(subs, c.Subscriptions.List()...)
		}
	}
	if w.subscriptions != nil {
		subs = append(subs, w.subscriptions.List()...)
	}
	return subs
}

// SetCanUseAuthenticatedEndpoints sets canUseAuthenticatedEndpoints val in a thread safe manner
func (w *Websocket) SetCanUseAuthenticatedEndpoints(b bool) {
	w.canUseAuthenticatedEndpoints.Store(b)
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in a thread safe manner
func (w *Websocket) CanUseAuthenticatedEndpoints() bool {
	return w.canUseAuthenticatedEndpoints.Load()
}

// checkWebsocketURL checks for a valid websocket url
func checkWebsocketURL(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("cannot set %w %s", errInvalidWebsocketURL, s)
	}
	return nil
}

// checkSubscriptions checks subscriptions against the max subscription limit and if the subscription already exists
// The subscription state is not considered when counting existing subscriptions
func (w *Websocket) checkSubscriptions(conn Connection, subs subscription.List) error {
	var subscriptionStore *subscription.Store
	if wrapper, ok := w.connections[conn]; ok && conn != nil {
		subscriptionStore = wrapper.Subscriptions
	} else {
		subscriptionStore = w.subscriptions
	}
	if subscriptionStore == nil {
		return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
	}

	existing := subscriptionStore.Len()
	if w.MaxSubscriptionsPerConnection > 0 && existing+len(subs) > w.MaxSubscriptionsPerConnection {
		return fmt.Errorf("%w: current subscriptions: %v, incoming subscriptions: %v, max subscriptions per connection: %v - please reduce enabled pairs",
			errSubscriptionsExceedsLimit,
			existing,
			len(subs),
			w.MaxSubscriptionsPerConnection)
	}

	for _, s := range subs {
		if found := subscriptionStore.Get(s); found != nil {
			return fmt.Errorf("%w: %s", subscription.ErrDuplicate, s)
		}
	}

	return nil
}

// Reader reads and handles data from a specific connection
func (w *Websocket) Reader(ctx context.Context, conn Connection, handler func(ctx context.Context, message []byte) error) {
	defer w.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return // Connection has been closed
		}
		if err := handler(ctx, resp.Raw); err != nil {
			w.DataHandler <- fmt.Errorf("connection URL:[%v] error: %w", conn.GetURL(), err)
		}
	}
}

func drain(ch <-chan error) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// ClosureFrame is a closure function that wraps monitoring variables with observer, if the return is true the frame will exit
type ClosureFrame func() func() bool

// monitorFrame monitors a specific websocket component or critical system. It will exit if the observer returns true
// This is used for monitoring data throughput, connection status and other critical websocket components. The waitgroup
// is optional and is used to signal when the monitor has finished.
func (w *Websocket) monitorFrame(wg *sync.WaitGroup, fn ClosureFrame) {
	if wg != nil {
		defer w.Wg.Done()
	}
	observe := fn()
	for {
		if observe() {
			return
		}
	}
}

// monitorData monitors data throughput and logs if there is a back log of data
func (w *Websocket) monitorData() func() bool {
	dropped := 0
	return func() bool { return w.observeData(&dropped) }
}

// observeData observes data throughput and logs if there is a back log of data
func (w *Websocket) observeData(dropped *int) (exit bool) {
	select {
	case <-w.ShutdownC:
		return true
	case d := <-w.DataHandler:
		select {
		case w.ToRoutine <- d:
			if *dropped != 0 {
				log.Infof(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer recovered; %d messages were dropped", w.exchangeName, dropped)
				*dropped = 0
			}
		default:
			if *dropped == 0 {
				// If this becomes prone to flapping we could drain the buffer, but that's extreme and we'd like to avoid it if possible
				log.Warnf(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer full; dropping messages", w.exchangeName)
			}
			*dropped++
		}
		return false
	}
}

// monitorConnection monitors the connection and attempts to reconnect if the connection is lost
func (w *Websocket) monitorConnection() func() bool {
	timer := time.NewTimer(w.connectionMonitorDelay)
	return func() bool { return w.observeConnection(timer) }
}

// observeConnection observes the connection and attempts to reconnect if the connection is lost
func (w *Websocket) observeConnection(t *time.Timer) (exit bool) {
	select {
	case err := <-w.ReadMessageErrors:
		if errors.Is(err, errConnectionFault) {
			log.Warnf(log.WebsocketMgr, "%v websocket has been disconnected. Reason: %v", w.exchangeName, err)
			if w.IsConnected() {
				if shutdownErr := w.Shutdown(); shutdownErr != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket: connectionMonitor shutdown err: %s", w.exchangeName, shutdownErr)
				}
			}
		}
		// Speedier reconnection, instead of waiting for the next cycle.
		if w.IsEnabled() && (!w.IsConnected() && !w.IsConnecting()) {
			if connectErr := w.Connect(); connectErr != nil {
				log.Errorln(log.WebsocketMgr, connectErr)
			}
		}
		w.DataHandler <- err // hand over the error to the data handler (shutdown and reconnection is priority)
	case <-t.C:
		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%v websocket: running connection monitor cycle", w.exchangeName)
		}
		if !w.IsEnabled() {
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v websocket: connectionMonitor - websocket disabled, shutting down", w.exchangeName)
			}
			if w.IsConnected() {
				if err := w.Shutdown(); err != nil {
					log.Errorln(log.WebsocketMgr, err)
				}
			}
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v websocket: connection monitor exiting", w.exchangeName)
			}
			t.Stop()
			w.connectionMonitorRunning.Store(false)
			return true
		}
		if !w.IsConnecting() && !w.IsConnected() {
			err := w.Connect()
			if err != nil {
				log.Errorln(log.WebsocketMgr, err)
			}
		}
		t.Reset(w.connectionMonitorDelay)
	}
	return false
}

// monitorTraffic monitors to see if there has been traffic within the trafficTimeout time window. If there is no traffic
// the connection is shutdown and will be reconnected by the connectionMonitor routine.
func (w *Websocket) monitorTraffic() func() bool {
	timer := time.NewTimer(w.trafficTimeout)
	return func() bool { return w.observeTraffic(timer) }
}

func (w *Websocket) observeTraffic(t *time.Timer) bool {
	select {
	case <-w.ShutdownC:
		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown message received", w.exchangeName)
		}
	case <-t.C:
		if w.IsConnecting() || signalReceived(w.TrafficAlert) {
			t.Reset(w.trafficTimeout)
			return false
		}
		if w.verbose {
			log.Warnf(log.WebsocketMgr, "%v websocket: has not received a traffic alert in %v. Reconnecting", w.exchangeName, w.trafficTimeout)
		}
		if w.IsConnected() {
			go func() { // Without this the w.Shutdown() call below will deadlock
				if err := w.Shutdown(); err != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown err: %s", w.exchangeName, err)
				}
			}()
		}
	}
	t.Stop()
	return true
}

// signalReceived checks if a signal has been received, this also clears the signal.
func signalReceived(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
