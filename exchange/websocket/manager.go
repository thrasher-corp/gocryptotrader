package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Public websocket errors
var (
	ErrWebsocketNotEnabled  = errors.New("websocket not enabled")
	ErrAlreadyDisabled      = errors.New("websocket already disabled")
	ErrNotConnected         = errors.New("websocket is not connected")
	ErrSignatureTimeout     = errors.New("websocket timeout waiting for response with signature")
	ErrRequestRouteNotFound = errors.New("request route not found")
	ErrSignatureNotSet      = errors.New("signature not set")
)

// Private websocket errors
var (
	errWebsocketAlreadyInitialised          = errors.New("websocket already initialised")
	errWebsocketAlreadyEnabled              = errors.New("websocket already enabled")
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
	errInvalidMaxSubscriptions              = errors.New("max subscriptions cannot be less than 0")
	errSameProxyAddress                     = errors.New("cannot set proxy address to the same address")
	errNoConnectFunc                        = errors.New("websocket connect func not set")
	errAlreadyConnected                     = errors.New("websocket already connected")
	errCannotShutdown                       = errors.New("websocket cannot shutdown")
	errAlreadyReconnecting                  = errors.New("websocket in the process of reconnection")
	errConnSetup                            = errors.New("error in connection setup")
	errNoPendingConnections                 = errors.New("no pending connections, call SetupNewConnection first")
	errDuplicateConnectionSetup             = errors.New("duplicate connection setup")
	errCannotChangeConnectionURL            = errors.New("cannot change connection URL when using multi connection management")
	errExchangeConfigEmpty                  = errors.New("exchange config is empty")
	errCannotObtainOutboundConnection       = errors.New("cannot obtain outbound connection")
	errMessageFilterNotComparable           = errors.New("message filter is not comparable")
)

// Websocket functionality list and state consts
const (
	UnhandledMessage = " - Unhandled websocket message: "
	jobBuffer        = 5000
)

const (
	uninitialisedState uint32 = iota
	disconnectedState
	connectingState
	connectedState
)

// Manager provides connection and subscription management and routing
type Manager struct {
	enabled                       atomic.Bool
	state                         atomic.Uint32
	verbose                       bool
	canUseAuthenticatedEndpoints  atomic.Bool
	connectionMonitorRunning      atomic.Bool
	trafficTimeout                time.Duration
	connectionMonitorDelay        time.Duration
	proxyAddr                     string
	defaultURL                    string
	defaultURLAuth                string
	runningURL                    string
	runningURLAuth                string
	exchangeName                  string
	features                      *protocol.Features
	m                             sync.Mutex
	connections                   map[Connection]*connectionWrapper
	subscriptions                 *subscription.Store
	connector                     func() error
	rateLimitDefinitions          request.RateLimitDefinitions // rate limiters shared between Websocket and REST connections
	Subscriber                    func(subscription.List) error
	Unsubscriber                  func(subscription.List) error
	GenerateSubs                  func() (subscription.List, error)
	useMultiConnectionManagement  bool
	DataHandler                   chan any
	ToRoutine                     chan any
	Match                         *Match
	ShutdownC                     chan struct{}
	Wg                            sync.WaitGroup
	Orderbook                     buffer.Orderbook
	Trade                         trade.Trade // Trade is a notifier for trades
	Fills                         fill.Fills  // Fills is a notifier for fills
	TrafficAlert                  chan struct{}
	ReadMessageErrors             chan error
	Conn                          Connection // Public connection
	AuthConn                      Connection // Authenticated Private connection
	ExchangeLevelReporter         Reporter   // Latency reporter
	MaxSubscriptionsPerConnection int

	// connectionManager stores all *potential* connections for the exchange, organised within connectionWrapper structs.
	// Each connectionWrapper one connection (will be expanded soon) tailored for specific exchange functionalities or asset types. // TODO: Expand this to support multiple connections per connectionWrapper
	// For example, separate connections can be used for Spot, Margin, and Futures trading. This structure is especially useful
	// for exchanges that differentiate between trading pairs by using different connection endpoints or protocols for various asset classes.
	// If an exchange does not require such differentiation, all connections may be managed under a single connectionWrapper.

	connectionManager []*connectionWrapper
}

// ManagerSetup defines variables for setting up a websocket manager
type ManagerSetup struct {
	ExchangeConfig        *config.Exchange
	DefaultURL            string
	RunningURL            string
	RunningURLAuth        string
	Connector             func() error
	Subscriber            func(subscription.List) error
	Unsubscriber          func(subscription.List) error
	GenerateSubscriptions func() (subscription.List, error)
	Features              *protocol.Features
	OrderbookBufferConfig buffer.Config

	// UseMultiConnectionManagement allows the connections to be managed by the
	// connection manager. If false, this will default to the global fields
	// provided in this struct.
	UseMultiConnectionManagement bool

	TradeFeed bool
	FillsFeed bool

	MaxWebsocketSubscriptionsPerConnection int

	// RateLimitDefinitions contains the rate limiters shared between WebSocket and REST connections for all endpoints.
	// These rate limits take precedence over any rate limits specified in individual connection configurations.
	// If no connection-specific rate limit is provided and the endpoint does not match any of these definitions,
	// an error will be returned. However, if a connection configuration includes its own rate limit,
	// it will fall back to that configuration’s rate limit without raising an error.
	RateLimitDefinitions request.RateLimitDefinitions
}

// connectionWrapper contains the connection setup details to be used when
// attempting a new connection. It also contains the subscriptions that are
// associated with the specific connection.
type connectionWrapper struct {
	setup         *ConnectionSetup
	subscriptions *subscription.Store
	connection    Connection
}

var globalReporter Reporter

// SetupGlobalReporter sets a reporter interface to be used
// for all exchange requests
func SetupGlobalReporter(r Reporter) {
	globalReporter = r
}

// NewManager initialises the websocket struct
func NewManager() *Manager {
	return &Manager{
		DataHandler:  make(chan any, jobBuffer),
		ToRoutine:    make(chan any, jobBuffer),
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
		connections:       make(map[Connection]*connectionWrapper),
	}
}

// Setup sets main variables for websocket connection
func (m *Manager) Setup(s *ManagerSetup) error {
	if err := common.NilGuard(m, s); err != nil {
		return err
	}
	if s.ExchangeConfig == nil {
		return fmt.Errorf("%w: ManagerSetup.ExchangeConfig", common.ErrNilPointer)
	}
	if s.ExchangeConfig.Features == nil {
		return fmt.Errorf("%w: ManagerSetup.ExchangeConfig.Features", common.ErrNilPointer)
	}
	if s.Features == nil {
		return fmt.Errorf("%w: ManagerSetup.Features", common.ErrNilPointer)
	}

	m.m.Lock()
	defer m.m.Unlock()

	if m.IsInitialised() {
		return fmt.Errorf("%s %w", m.exchangeName, errWebsocketAlreadyInitialised)
	}

	if s.ExchangeConfig.Name == "" {
		return errExchangeConfigNameEmpty
	}
	m.exchangeName = s.ExchangeConfig.Name
	m.verbose = s.ExchangeConfig.Verbose

	m.features = s.Features

	m.setEnabled(s.ExchangeConfig.Features.Enabled.Websocket)

	m.useMultiConnectionManagement = s.UseMultiConnectionManagement

	if !m.useMultiConnectionManagement {
		// TODO: Remove this block when all exchanges are updated and backwards
		// compatibility is no longer required.
		if s.Connector == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketConnectorUnset)
		}
		if s.Subscriber == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriberUnset)
		}
		if s.Unsubscriber == nil && m.features.Unsubscribe {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketUnsubscriberUnset)
		}
		if s.GenerateSubscriptions == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriptionsGeneratorUnset)
		}
		if s.DefaultURL == "" {
			return fmt.Errorf("%s websocket %w", m.exchangeName, errDefaultURLIsEmpty)
		}
		m.defaultURL = s.DefaultURL
		if s.RunningURL == "" {
			return fmt.Errorf("%s websocket %w", m.exchangeName, errRunningURLIsEmpty)
		}

		m.connector = s.Connector
		m.Subscriber = s.Subscriber
		m.Unsubscriber = s.Unsubscriber
		m.GenerateSubs = s.GenerateSubscriptions

		err := m.SetWebsocketURL(s.RunningURL, false, false)
		if err != nil {
			return fmt.Errorf("%s %w", m.exchangeName, err)
		}

		if s.RunningURLAuth != "" {
			err = m.SetWebsocketURL(s.RunningURLAuth, true, false)
			if err != nil {
				return fmt.Errorf("%s %w", m.exchangeName, err)
			}
		}
	}

	m.connectionMonitorDelay = s.ExchangeConfig.ConnectionMonitorDelay
	if m.connectionMonitorDelay <= 0 {
		m.connectionMonitorDelay = config.DefaultConnectionMonitorDelay
	}

	if s.ExchangeConfig.WebsocketTrafficTimeout < time.Second {
		return fmt.Errorf("%s %w cannot be less than %s",
			m.exchangeName,
			errInvalidTrafficTimeout,
			time.Second)
	}
	m.trafficTimeout = s.ExchangeConfig.WebsocketTrafficTimeout

	m.SetCanUseAuthenticatedEndpoints(s.ExchangeConfig.API.AuthenticatedWebsocketSupport)

	if err := m.Orderbook.Setup(s.ExchangeConfig, &s.OrderbookBufferConfig, m.DataHandler); err != nil {
		return err
	}

	m.Trade.Setup(s.TradeFeed, m.DataHandler)
	m.Fills.Setup(s.FillsFeed, m.DataHandler)

	if s.MaxWebsocketSubscriptionsPerConnection < 0 {
		return fmt.Errorf("%s %w", m.exchangeName, errInvalidMaxSubscriptions)
	}
	m.MaxSubscriptionsPerConnection = s.MaxWebsocketSubscriptionsPerConnection
	m.setState(disconnectedState)

	m.rateLimitDefinitions = s.RateLimitDefinitions
	return nil
}

// SetupNewConnection sets up an auth or unauth streaming connection
func (m *Manager) SetupNewConnection(c *ConnectionSetup) error {
	if err := common.NilGuard(m, c); err != nil {
		return err
	}

	if c == nil || c.ResponseCheckTimeout == 0 &&
		c.ResponseMaxLimit == 0 &&
		c.RateLimit == nil &&
		c.URL == "" &&
		c.ConnectionLevelReporter == nil &&
		c.BespokeGenerateMessageID == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errExchangeConfigEmpty)
	}

	if m.exchangeName == "" {
		return fmt.Errorf("%w: %w", errConnSetup, errExchangeConfigNameEmpty)
	}
	if m.TrafficAlert == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errTrafficAlertNil)
	}
	if m.ReadMessageErrors == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errReadMessageErrorsNil)
	}
	if c.ConnectionLevelReporter == nil {
		c.ConnectionLevelReporter = m.ExchangeLevelReporter
	}
	if c.ConnectionLevelReporter == nil {
		c.ConnectionLevelReporter = globalReporter
	}

	if m.useMultiConnectionManagement {
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
		if c.Unsubscriber == nil && m.features.Unsubscribe {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketUnsubscriberUnset)
		}
		if c.Handler == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketDataHandlerUnset)
		}

		if c.MessageFilter != nil && !reflect.TypeOf(c.MessageFilter).Comparable() {
			return errMessageFilterNotComparable
		}

		for x := range m.connectionManager {
			// Below allows for multiple connections to the same URL with different outbound request signatures. This
			// allows for easier determination of inbound and outbound messages. e.g. Gateio cross_margin, margin on
			// a spot connection.
			if m.connectionManager[x].setup.URL == c.URL && c.MessageFilter == m.connectionManager[x].setup.MessageFilter {
				return fmt.Errorf("%w: %w", errConnSetup, errDuplicateConnectionSetup)
			}
		}
		m.connectionManager = append(m.connectionManager, &connectionWrapper{
			setup:         c,
			subscriptions: subscription.NewStore(),
		})
		return nil
	}

	if c.Authenticated {
		m.AuthConn = m.getConnectionFromSetup(c)
	} else {
		m.Conn = m.getConnectionFromSetup(c)
	}

	return nil
}

// getConnectionFromSetup returns a websocket connection from a setup
// configuration. This is used for setting up new connections on the fly.
func (m *Manager) getConnectionFromSetup(c *ConnectionSetup) *connection {
	connectionURL := m.GetWebsocketURL()
	if c.URL != "" {
		connectionURL = c.URL
	}
	return &connection{
		ExchangeName:             m.exchangeName,
		URL:                      connectionURL,
		ProxyURL:                 m.GetProxyAddress(),
		Verbose:                  m.verbose,
		ResponseMaxLimit:         c.ResponseMaxLimit,
		Traffic:                  m.TrafficAlert,
		readMessageErrors:        m.ReadMessageErrors,
		shutdown:                 m.ShutdownC,
		Wg:                       &m.Wg,
		Match:                    m.Match,
		RateLimit:                c.RateLimit,
		Reporter:                 c.ConnectionLevelReporter,
		bespokeGenerateMessageID: c.BespokeGenerateMessageID,
		RateLimitDefinitions:     m.rateLimitDefinitions,
	}
}

// Connect initiates a websocket connection by using a package defined connection
// function
func (m *Manager) Connect() error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.connect()
}

func (m *Manager) connect() error {
	if !m.IsEnabled() {
		return ErrWebsocketNotEnabled
	}
	if m.IsConnecting() {
		return fmt.Errorf("%v %w", m.exchangeName, errAlreadyReconnecting)
	}
	if m.IsConnected() {
		return fmt.Errorf("%v %w", m.exchangeName, errAlreadyConnected)
	}

	if m.subscriptions == nil {
		return fmt.Errorf("%w: subscriptions", common.ErrNilPointer)
	}
	m.subscriptions.Clear()

	m.setState(connectingState)

	m.Wg.Add(2)
	go m.monitorFrame(&m.Wg, m.monitorData)
	go m.monitorFrame(&m.Wg, m.monitorTraffic)

	if !m.useMultiConnectionManagement {
		if m.connector == nil {
			return fmt.Errorf("%v %w", m.exchangeName, errNoConnectFunc)
		}
		err := m.connector()
		if err != nil {
			m.setState(disconnectedState)
			return fmt.Errorf("%v Error connecting %w", m.exchangeName, err)
		}
		m.setState(connectedState)

		if m.connectionMonitorRunning.CompareAndSwap(false, true) {
			// This oversees all connections and does not need to be part of wait group management.
			go m.monitorFrame(nil, m.monitorConnection)
		}

		subs, err := m.GenerateSubs() // regenerate state on new connection
		if err != nil {
			return fmt.Errorf("%s websocket: %w", m.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
		}
		if len(subs) != 0 {
			if err := m.SubscribeToChannels(nil, subs); err != nil {
				return err
			}

			if missing := m.subscriptions.Missing(subs); len(missing) > 0 {
				return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
			}
		}
		return nil
	}

	if len(m.connectionManager) == 0 {
		m.setState(disconnectedState)
		return fmt.Errorf("cannot connect: %w", errNoPendingConnections)
	}

	// multiConnectFatalError is a fatal error that will cause all connections to
	// be shutdown and the websocket to be disconnected.
	var multiConnectFatalError error

	// subscriptionError is a non-fatal error that does not shutdown connections
	var subscriptionError error

	// TODO: Implement concurrency below.
	for i := range m.connectionManager {
		if m.connectionManager[i].setup.GenerateSubscriptions == nil {
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, m.connectionManager[i].setup.URL, errWebsocketSubscriptionsGeneratorUnset)
			break
		}

		subs, err := m.connectionManager[i].setup.GenerateSubscriptions() // regenerate state on new connection
		if err != nil {
			multiConnectFatalError = fmt.Errorf("%s websocket: %w", m.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
			break
		}

		if len(subs) == 0 {
			// If no subscriptions are generated, we skip the connection
			if m.verbose {
				log.Warnf(log.WebsocketMgr, "%s websocket: no subscriptions generated", m.exchangeName)
			}
			continue
		}

		if m.connectionManager[i].setup.Connector == nil {
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, m.connectionManager[i].setup.URL, errNoConnectFunc)
			break
		}
		if m.connectionManager[i].setup.Handler == nil {
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, m.connectionManager[i].setup.URL, errWebsocketDataHandlerUnset)
			break
		}
		if m.connectionManager[i].setup.Subscriber == nil {
			multiConnectFatalError = fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, m.connectionManager[i].setup.URL, errWebsocketSubscriberUnset)
			break
		}

		// TODO: Add window for max subscriptions per connection, to spawn new connections if needed.

		conn := m.getConnectionFromSetup(m.connectionManager[i].setup)

		err = m.connectionManager[i].setup.Connector(context.TODO(), conn)
		if err != nil {
			multiConnectFatalError = fmt.Errorf("%v Error connecting %w", m.exchangeName, err)
			break
		}

		if !conn.IsConnected() {
			multiConnectFatalError = fmt.Errorf("%s websocket: [conn:%d] [URL:%s] failed to connect", m.exchangeName, i+1, conn.URL)
			break
		}

		m.connections[conn] = m.connectionManager[i]
		m.connectionManager[i].connection = conn

		m.Wg.Add(1)
		go m.Reader(context.TODO(), conn, m.connectionManager[i].setup.Handler)

		if m.connectionManager[i].setup.Authenticate != nil && m.CanUseAuthenticatedEndpoints() {
			err = m.connectionManager[i].setup.Authenticate(context.TODO(), conn)
			if err != nil {
				multiConnectFatalError = fmt.Errorf("%s websocket: [conn:%d] [URL:%s] failed to authenticate %w", m.exchangeName, i+1, conn.URL, err)
				break
			}
		}

		err = m.connectionManager[i].setup.Subscriber(context.TODO(), conn, subs)
		if err != nil {
			subscriptionError = common.AppendError(subscriptionError, fmt.Errorf("%v Error subscribing %w", m.exchangeName, err))
			continue
		}

		if missing := m.connectionManager[i].subscriptions.Missing(subs); len(missing) > 0 {
			subscriptionError = common.AppendError(subscriptionError, fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing))
			continue
		}

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: [conn:%d] [URL:%s] connected. [Subscribed: %d]",
				m.exchangeName,
				i+1,
				conn.URL,
				len(subs))
		}
	}

	if multiConnectFatalError != nil {
		// Roll back any successful connections and flush subscriptions
		for x := range m.connectionManager {
			if m.connectionManager[x].connection != nil {
				if err := m.connectionManager[x].connection.Shutdown(); err != nil {
					log.Errorln(log.WebsocketMgr, err)
				}
				m.connectionManager[x].connection = nil
			}
			m.connectionManager[x].subscriptions.Clear()
		}
		clear(m.connections)
		m.setState(disconnectedState) // Flip from connecting to disconnected.

		// Drain residual error in the single buffered channel, this mitigates
		// the cycle when `Connect` is called again and the connectionMonitor
		// starts but there is an old error in the channel.
		drain(m.ReadMessageErrors)

		return multiConnectFatalError
	}

	// Assume connected state here. All connections have been established.
	// All subscriptions have been sent and stored. All data received is being
	// handled by the appropriate data handler.
	m.setState(connectedState)

	if m.connectionMonitorRunning.CompareAndSwap(false, true) {
		// This oversees all connections and does not need to be part of wait group management.
		go m.monitorFrame(nil, m.monitorConnection)
	}

	return subscriptionError
}

// Disable disables the exchange websocket protocol
// Note that connectionMonitor will be responsible for shutting down the websocket after disabling
func (m *Manager) Disable() error {
	if !m.IsEnabled() {
		return fmt.Errorf("%s %w", m.exchangeName, ErrAlreadyDisabled)
	}

	m.setEnabled(false)
	return nil
}

// Enable enables the exchange websocket protocol
func (m *Manager) Enable() error {
	if m.IsConnected() || m.IsEnabled() {
		return fmt.Errorf("%s %w", m.exchangeName, errWebsocketAlreadyEnabled)
	}

	m.setEnabled(true)
	return m.Connect()
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (m *Manager) Shutdown() error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.shutdown()
}

func (m *Manager) shutdown() error {
	if !m.IsConnected() {
		return fmt.Errorf("%v %w: %w", m.exchangeName, errCannotShutdown, ErrNotConnected)
	}

	// TODO: Interrupt connection and or close connection when it is re-established.
	if m.IsConnecting() {
		return fmt.Errorf("%v %w: %w ", m.exchangeName, errCannotShutdown, errAlreadyReconnecting)
	}

	if m.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: shutting down websocket", m.exchangeName)
	}

	defer m.Orderbook.FlushBuffer()

	// During the shutdown process, all errors are treated as non-fatal to avoid issues when the connection has already
	// been closed. In such cases, attempting to close the connection may result in a
	// "failed to send closeNotify alert (but connection was closed anyway)" error. Treating these errors as non-fatal
	// prevents the shutdown process from being interrupted, which could otherwise trigger a continuous traffic monitor
	// cycle and potentially block the initiation of a new connection.
	var nonFatalCloseConnectionErrors error

	// Shutdown managed connections
	for x := range m.connectionManager {
		if m.connectionManager[x].connection != nil {
			if err := m.connectionManager[x].connection.Shutdown(); err != nil {
				nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
			}
			m.connectionManager[x].connection = nil
			// Flush any subscriptions from last connection across any managed connections
			m.connectionManager[x].subscriptions.Clear()
		}
	}
	// Clean map of old connections
	clear(m.connections)

	if m.Conn != nil {
		if err := m.Conn.Shutdown(); err != nil {
			nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
		}
	}
	if m.AuthConn != nil {
		if err := m.AuthConn.Shutdown(); err != nil {
			nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
		}
	}
	// flush any subscriptions from last connection if needed
	m.subscriptions.Clear()

	m.setState(disconnectedState)

	close(m.ShutdownC)
	m.Wg.Wait()
	m.ShutdownC = make(chan struct{})
	if m.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: completed websocket shutdown", m.exchangeName)
	}

	// Drain residual error in the single buffered channel, this mitigates
	// the cycle when `Connect` is called again and the connectionMonitor
	// starts but there is an old error in the channel.
	drain(m.ReadMessageErrors)

	if nonFatalCloseConnectionErrors != nil {
		log.Warnf(log.WebsocketMgr, "%v websocket: shutdown error: %v", m.exchangeName, nonFatalCloseConnectionErrors)
	}

	return nil
}

func (m *Manager) setState(s uint32) {
	m.state.Store(s)
}

// IsInitialised returns whether the websocket has been Setup() already
func (m *Manager) IsInitialised() bool {
	return m.state.Load() != uninitialisedState
}

// IsConnected returns whether the websocket is connected
func (m *Manager) IsConnected() bool {
	return m.state.Load() == connectedState
}

// IsConnecting returns whether the websocket is connecting
func (m *Manager) IsConnecting() bool {
	return m.state.Load() == connectingState
}

func (m *Manager) setEnabled(b bool) {
	m.enabled.Store(b)
}

// IsEnabled returns whether the websocket is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled.Load()
}

// CanUseAuthenticatedWebsocketForWrapper Handles a common check to
// verify whether a wrapper can use an authenticated websocket endpoint
func (m *Manager) CanUseAuthenticatedWebsocketForWrapper() bool {
	if m.IsConnected() {
		if m.CanUseAuthenticatedEndpoints() {
			return true
		}
		log.Infof(log.WebsocketMgr, "%v - Websocket not authenticated, using REST\n", m.exchangeName)
	}
	return false
}

// SetWebsocketURL sets websocket URL and can refresh underlying connections
func (m *Manager) SetWebsocketURL(url string, auth, reconnect bool) error {
	if m.useMultiConnectionManagement {
		// TODO: Add functionality for multi-connection management to change URL
		return fmt.Errorf("%s: %w", m.exchangeName, errCannotChangeConnectionURL)
	}
	defaultVals := url == "" || url == config.WebsocketURLNonDefaultMessage
	if auth {
		if defaultVals {
			url = m.defaultURLAuth
		}

		err := checkWebsocketURL(url)
		if err != nil {
			return err
		}
		m.runningURLAuth = url

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: setting authenticated websocket URL: %s\n", m.exchangeName, url)
		}

		if m.AuthConn != nil {
			m.AuthConn.SetURL(url)
		}
	} else {
		if defaultVals {
			url = m.defaultURL
		}
		err := checkWebsocketURL(url)
		if err != nil {
			return err
		}
		m.runningURL = url

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: setting unauthenticated websocket URL: %s\n", m.exchangeName, url)
		}

		if m.Conn != nil {
			m.Conn.SetURL(url)
		}
	}

	if m.IsConnected() && reconnect {
		log.Debugf(log.WebsocketMgr, "%s websocket: flushing websocket connection to %s\n", m.exchangeName, url)
		return m.Shutdown()
	}
	return nil
}

// GetWebsocketURL returns the running websocket URL
func (m *Manager) GetWebsocketURL() string {
	return m.runningURL
}

// SetProxyAddress sets websocket proxy address
func (m *Manager) SetProxyAddress(proxyAddr string) error {
	m.m.Lock()
	defer m.m.Unlock()
	if proxyAddr != "" {
		if _, err := url.ParseRequestURI(proxyAddr); err != nil {
			return fmt.Errorf("%v websocket: cannot set proxy address: %w", m.exchangeName, err)
		}

		if m.proxyAddr == proxyAddr {
			return fmt.Errorf("%v websocket: %w '%v'", m.exchangeName, errSameProxyAddress, m.proxyAddr)
		}

		log.Debugf(log.ExchangeSys, "%s websocket: setting websocket proxy: %s", m.exchangeName, proxyAddr)
	} else {
		log.Debugf(log.ExchangeSys, "%s websocket: removing websocket proxy", m.exchangeName)
	}

	for _, wrapper := range m.connectionManager {
		if wrapper.connection != nil {
			wrapper.connection.SetProxy(proxyAddr)
		}
	}
	if m.Conn != nil {
		m.Conn.SetProxy(proxyAddr)
	}
	if m.AuthConn != nil {
		m.AuthConn.SetProxy(proxyAddr)
	}

	m.proxyAddr = proxyAddr

	if !m.IsConnected() {
		return nil
	}
	if err := m.shutdown(); err != nil {
		return err
	}
	return m.connect()
}

// GetProxyAddress returns the current websocket proxy
func (m *Manager) GetProxyAddress() string {
	return m.proxyAddr
}

// GetName returns exchange name
func (m *Manager) GetName() string {
	return m.exchangeName
}

// SetCanUseAuthenticatedEndpoints sets canUseAuthenticatedEndpoints val in a thread safe manner
func (m *Manager) SetCanUseAuthenticatedEndpoints(b bool) {
	m.canUseAuthenticatedEndpoints.Store(b)
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in a thread safe manner
func (m *Manager) CanUseAuthenticatedEndpoints() bool {
	return m.canUseAuthenticatedEndpoints.Load()
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

// Reader reads and handles data from a specific connection
func (m *Manager) Reader(ctx context.Context, conn Connection, handler func(ctx context.Context, message []byte) error) {
	defer m.Wg.Done()
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return // Connection has been closed
		}
		if err := handler(ctx, resp.Raw); err != nil {
			m.DataHandler <- fmt.Errorf("connection URL:[%v] error: %w", conn.GetURL(), err)
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
func (m *Manager) monitorFrame(wg *sync.WaitGroup, fn ClosureFrame) {
	if wg != nil {
		defer m.Wg.Done()
	}
	observe := fn()
	for {
		if observe() {
			return
		}
	}
}

// monitorData monitors data throughput and logs if there is a back log of data
func (m *Manager) monitorData() func() bool {
	dropped := 0
	return func() bool { return m.observeData(&dropped) }
}

// observeData observes data throughput and logs if there is a back log of data
func (m *Manager) observeData(dropped *int) (exit bool) {
	select {
	case <-m.ShutdownC:
		return true
	case d := <-m.DataHandler:
		select {
		case m.ToRoutine <- d:
			if *dropped != 0 {
				log.Infof(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer recovered; %d messages were dropped", m.exchangeName, dropped)
				*dropped = 0
			}
		default:
			if *dropped == 0 {
				// If this becomes prone to flapping we could drain the buffer, but that's extreme and we'd like to avoid it if possible
				log.Warnf(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer full; dropping messages", m.exchangeName)
			}
			*dropped++
		}
		return false
	}
}

// monitorConnection monitors the connection and attempts to reconnect if the connection is lost
func (m *Manager) monitorConnection() func() bool {
	timer := time.NewTimer(m.connectionMonitorDelay)
	return func() bool { return m.observeConnection(timer) }
}

// observeConnection observes the connection and attempts to reconnect if the connection is lost
func (m *Manager) observeConnection(t *time.Timer) (exit bool) {
	select {
	case err := <-m.ReadMessageErrors:
		if errors.Is(err, errConnectionFault) {
			log.Warnf(log.WebsocketMgr, "%v websocket has been disconnected. Reason: %v", m.exchangeName, err)
			if m.IsConnected() {
				if shutdownErr := m.Shutdown(); shutdownErr != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket: connectionMonitor shutdown err: %s", m.exchangeName, shutdownErr)
				}
			}
		}
		// Speedier reconnection, instead of waiting for the next cycle.
		if m.IsEnabled() && (!m.IsConnected() && !m.IsConnecting()) {
			if connectErr := m.Connect(); connectErr != nil {
				log.Errorln(log.WebsocketMgr, connectErr)
			}
		}
		m.DataHandler <- err // hand over the error to the data handler (shutdown and reconnection is priority)
	case <-t.C:
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%v websocket: running connection monitor cycle", m.exchangeName)
		}
		if !m.IsEnabled() {
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "%v websocket: connectionMonitor - websocket disabled, shutting down", m.exchangeName)
			}
			if m.IsConnected() {
				if err := m.Shutdown(); err != nil {
					log.Errorln(log.WebsocketMgr, err)
				}
			}
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "%v websocket: connection monitor exiting", m.exchangeName)
			}
			t.Stop()
			m.connectionMonitorRunning.Store(false)
			return true
		}
		if !m.IsConnecting() && !m.IsConnected() {
			err := m.Connect()
			if err != nil {
				log.Errorln(log.WebsocketMgr, err)
			}
		}
		t.Reset(m.connectionMonitorDelay)
	}
	return false
}

// monitorTraffic monitors to see if there has been traffic within the trafficTimeout time window. If there is no traffic
// the connection is shutdown and will be reconnected by the connectionMonitor routine.
func (m *Manager) monitorTraffic() func() bool {
	timer := time.NewTimer(m.trafficTimeout)
	return func() bool { return m.observeTraffic(timer) }
}

func (m *Manager) observeTraffic(t *time.Timer) bool {
	select {
	case <-m.ShutdownC:
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown message received", m.exchangeName)
		}
	case <-t.C:
		if m.IsConnecting() || signalReceived(m.TrafficAlert) {
			t.Reset(m.trafficTimeout)
			return false
		}
		if m.verbose {
			log.Warnf(log.WebsocketMgr, "%v websocket: has not received a traffic alert in %v. Reconnecting", m.exchangeName, m.trafficTimeout)
		}
		if m.IsConnected() {
			go func() { // Without this the m.Shutdown() call below will deadlock
				if err := m.Shutdown(); err != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown err: %s", m.exchangeName, err)
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

// GetConnection returns a connection by message filter (defined in exchange package _wrapper.go websocket connection)
// for request and response handling in a multi connection context.
func (m *Manager) GetConnection(messageFilter any) (Connection, error) {
	if err := common.NilGuard(m); err != nil {
		return nil, err
	}
	if messageFilter == nil {
		return nil, fmt.Errorf("%w: messageFilter", common.ErrNilPointer)
	}

	m.m.Lock()
	defer m.m.Unlock()

	if !m.useMultiConnectionManagement {
		return nil, fmt.Errorf("%s: multi connection management not enabled %w please use exported Conn and AuthConn fields", m.exchangeName, errCannotObtainOutboundConnection)
	}

	if !m.IsConnected() {
		return nil, ErrNotConnected
	}

	for _, wrapper := range m.connectionManager {
		if wrapper.setup.MessageFilter == messageFilter {
			if wrapper.connection == nil {
				return nil, fmt.Errorf("%s: %s %w associated with message filter: '%v'", m.exchangeName, wrapper.setup.URL, ErrNotConnected, messageFilter)
			}
			return wrapper.connection, nil
		}
	}

	return nil, fmt.Errorf("%s: %w associated with message filter: '%v'", m.exchangeName, ErrRequestRouteNotFound, messageFilter)
}
