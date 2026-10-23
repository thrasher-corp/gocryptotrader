package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket/orderbookmanager"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// Public websocket errors
var (
	ErrWebsocketNotEnabled     = errors.New("websocket not enabled")
	ErrAlreadyDisabled         = errors.New("websocket already disabled")
	ErrWebsocketAlreadyEnabled = errors.New("websocket already enabled")
	ErrNotConnected            = errors.New("websocket is not connected")
	ErrSignatureTimeout        = errors.New("websocket timeout waiting for response with signature")
	ErrRequestRouteNotFound    = errors.New("request route not found")
	ErrSignatureNotSet         = errors.New("signature not set")
)

// Private websocket errors
var (
	errWebsocketAlreadyInitialised          = errors.New("websocket already initialised")
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
	errFailedToAuthenticate                 = errors.New("failed to authenticate")
	errAuthenticatedUseOff                  = errors.New("authenticated use is off")
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
	enabled                      atomic.Bool
	state                        atomic.Uint32
	verbose                      bool
	canUseAuthenticatedEndpoints atomic.Bool
	connectionMonitorRunning     atomic.Bool
	trafficTimeout               time.Duration
	connectionMonitorDelay       time.Duration
	proxyAddr                    string
	defaultURL                   string
	defaultURLAuth               string
	runningURL                   string
	runningURLAuth               string
	exchangeName                 string
	features                     *protocol.Features
	m                            sync.Mutex
	connectionManagerMu          sync.RWMutex
	connections                  map[Connection]*websocket
	// readersDone holds, for each managed connection, a channel closed once its reader and disconnect hook have finished
	readersDone                   map[Connection]chan struct{}
	subscriptions                 *subscription.Store
	connector                     func() error
	preConnect                    func(context.Context)
	rateLimitDefinitions          request.RateLimitDefinitions // rate limiters shared between Websocket and REST connections
	Subscriber                    func(subscription.List) error
	Unsubscriber                  func(subscription.List) error
	GenerateSubs                  func() (subscription.List, error)
	useMultiConnectionManagement  bool
	DataHandler                   *stream.Relay
	Match                         *Match
	ShutdownC                     chan struct{}
	Wg                            sync.WaitGroup
	Orderbook                     orderbookmanager.Orderbook
	Trade                         trade.Trade // Trade is a notifier for trades
	Fills                         fill.Fills  // Fills is a notifier for fills
	TrafficAlert                  chan struct{}
	ReadMessageErrors             chan error
	Conn                          Connection // Public connection
	AuthConn                      Connection // Authenticated Private connection
	ExchangeLevelReporter         Reporter   // Latency reporter
	MaxSubscriptionsPerConnection int

	// connectionManager stores all *potential* connections for the exchange, organised within websocket structs.
	// For example, separate connections can be used for Spot, Margin, and Futures trading. This structure is especially useful
	// for exchanges that differentiate between trading pairs by using different connection endpoints or protocols for various asset classes.
	// If an exchange does not require such differentiation, all connections may be managed under a single websocket.
	connectionManager []*websocket
}

// ManagerSetup defines variables for setting up a websocket manager
type ManagerSetup struct {
	ExchangeConfig *config.Exchange
	DefaultURL     string
	RunningURL     string
	RunningURLAuth string
	Connector      func() error
	// PreConnect performs context-aware preparation before the manager lock is acquired.
	PreConnect            func(context.Context)
	Subscriber            func(subscription.List) error
	Unsubscriber          func(subscription.List) error
	GenerateSubscriptions func() (subscription.List, error)
	Features              *protocol.Features
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

// websocket contains the connection setup details to be used when attempting a new connection. Its subscription store
// knows of all subscriptions for each connection. Each connection will have its own subscription
// store to track subscriptions made on that specific connection.
type websocket struct {
	setup         *ConnectionSetup
	subscriptions *subscription.Store
	connections   []Connection
	// retryFailures counts an optional setup's consecutive connection failures, and retryAt is when it may be dialled
	// again; both are guarded by the manager's m lock
	retryFailures int
	retryAt       time.Time
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
		DataHandler:  stream.NewRelay(jobBuffer),
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
		Orderbook:         orderbookmanager.Orderbook{},
		connections:       make(map[Connection]*websocket),
		readersDone:       make(map[Connection]chan struct{}),
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
	m.preConnect = s.PreConnect

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

	if err := m.Orderbook.Setup(s.ExchangeConfig.Name, m.DataHandler, s.ExchangeConfig.Verbose); err != nil {
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

	m.m.Lock()
	defer m.m.Unlock()

	if c.ResponseCheckTimeout == 0 && c.ResponseMaxLimit == 0 && c.RateLimit == nil && c.URL == "" && c.ConnectionLevelReporter == nil {
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
		// The connection and supporting functions are defined per connection and the connection websocket is stored in
		// the connection manager.
		if c.URL == "" {
			return fmt.Errorf("%w: %w", errConnSetup, errDefaultURLIsEmpty)
		}
		if c.Connector == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketConnectorUnset)
		}
		if c.GenerateSubscriptions == nil && !c.SubscriptionsNotRequired {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriptionsGeneratorUnset)
		}
		if c.Subscriber == nil && !c.SubscriptionsNotRequired {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketSubscriberUnset)
		}
		if c.Unsubscriber == nil && m.features.Unsubscribe && !c.SubscriptionsNotRequired {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketUnsubscriberUnset)
		}
		if c.Handler == nil {
			return fmt.Errorf("%w: %w", errConnSetup, errWebsocketDataHandlerUnset)
		}

		if c.MessageFilter != nil && !reflect.TypeOf(c.MessageFilter).Comparable() {
			return errMessageFilterNotComparable
		}

		m.connectionManagerMu.Lock()
		defer m.connectionManagerMu.Unlock()
		for x := range m.connectionManager {
			// Below allows for multiple connections to the same URL with different outbound request signatures. This
			// allows for easier determination of inbound and outbound messages. e.g. Gateio cross_margin, margin on
			// a spot connection.
			if m.connectionManager[x].setup.URL == c.URL && c.MessageFilter == m.connectionManager[x].setup.MessageFilter {
				return fmt.Errorf("%w: %w", errConnSetup, errDuplicateConnectionSetup)
			}
		}
		m.connectionManager = append(m.connectionManager, &websocket{setup: c, subscriptions: subscription.NewStore()})
		return nil
	}

	if c.Authenticated {
		m.AuthConn = m.createConnectionFromSetup(c)
	} else {
		m.Conn = m.createConnectionFromSetup(c)
	}

	return nil
}

// createConnectionFromSetup returns a websocket connection from a setup
// configuration. This is used for setting up new connections on the fly.
func (m *Manager) createConnectionFromSetup(c *ConnectionSetup) *connection {
	connectionURL := m.GetWebsocketURL()
	if c.URL != "" {
		connectionURL = c.URL
	}
	match := m.Match
	if m.useMultiConnectionManagement {
		// If we are using multi connection management, we can decouple
		// the match from the global match and have a match per connection.
		match = NewMatch()
	}
	rateLimit := c.RateLimit
	if c.ConnectionRateLimiter != nil {
		rateLimit = c.ConnectionRateLimiter()
	}
	var closed chan struct{}
	if m.useMultiConnectionManagement {
		closed = make(chan struct{})
	}
	return &connection{
		closed:               closed,
		ExchangeName:         m.exchangeName,
		URL:                  connectionURL,
		ProxyURL:             m.GetProxyAddress(),
		Verbose:              m.verbose,
		ResponseMaxLimit:     c.ResponseMaxLimit,
		Traffic:              m.TrafficAlert,
		readMessageErrors:    m.ReadMessageErrors,
		shutdown:             m.ShutdownC,
		Wg:                   &m.Wg,
		Match:                match,
		RateLimit:            rateLimit,
		Reporter:             c.ConnectionLevelReporter,
		RateLimitDefinitions: m.rateLimitDefinitions,
		subscriptions:        subscription.NewStore(),
	}
}

func (m *Manager) snapshotConnectionManager() []*websocket {
	m.connectionManagerMu.RLock()
	defer m.connectionManagerMu.RUnlock()
	return slices.Clone(m.connectionManager)
}

func (m *Manager) snapshotManagedConnections(ws *websocket) []Connection {
	if ws == nil {
		return nil
	}
	m.connectionManagerMu.RLock()
	defer m.connectionManagerMu.RUnlock()
	return slices.Clone(ws.connections)
}

func (m *Manager) trackConnection(conn Connection, ws *websocket) {
	m.connectionManagerMu.Lock()
	defer m.connectionManagerMu.Unlock()
	if m.connections == nil {
		m.connections = make(map[Connection]*websocket)
	}
	if existing, ok := m.connections[conn]; ok {
		if existing == ws {
			return
		}
		panic("trackConnection called with connection already associated with a different websocket")
	}
	m.connections[conn] = ws
	if !slices.Contains(ws.connections, conn) {
		ws.connections = append(ws.connections, conn)
	}
}

// Connect initiates a websocket connection by using a package defined connection
// function
func (m *Manager) Connect(ctx context.Context) error {
	if m.IsEnabled() && !m.IsConnecting() && !m.IsConnected() && m.preConnect != nil {
		m.preConnect(ctx)
	}
	m.m.Lock()
	defer m.m.Unlock()
	return m.connect(ctx)
}

func (m *Manager) connect(ctx context.Context) error {
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

	if !m.useMultiConnectionManagement {
		if m.connector == nil {
			return fmt.Errorf("%v %w", m.exchangeName, errNoConnectFunc)
		}
		err := m.connector()
		if err != nil {
			// The connector may have dialled and started readers before failing. Release them so the next attempt does
			// not stack another socket and reader on top. Close errors are expected here for sockets closed by an
			// earlier shutdown, so only teardown failures are reported.
			if _, teardownErr := m.teardown(); teardownErr != nil {
				err = common.AppendError(err, teardownErr)
			}
			return fmt.Errorf("%v Error connecting %w", m.exchangeName, err)
		}
		m.setState(connectedState)

		m.Wg.Add(1)
		go m.monitorFrame(ctx, &m.Wg, m.monitorTraffic)

		if m.connectionMonitorRunning.CompareAndSwap(false, true) {
			// This oversees all connections and does not need to be part of wait group management.
			go m.monitorFrame(ctx, nil, m.monitorConnection)
		}

		subs, err := m.GenerateSubs() // regenerate state on new connection
		if err != nil {
			return fmt.Errorf("%s websocket: %w", m.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
		}
		if len(subs) != 0 {
			if err := m.SubscribeToChannels(ctx, nil, subs); err != nil {
				return err
			}

			if missing := m.subscriptions.Missing(subs); len(missing) > 0 {
				return fmt.Errorf("%v %w %q", m.exchangeName, ErrSubscriptionsNotAdded, missing)
			}
		}
		return nil
	}

	connectionManager := m.snapshotConnectionManager()
	if len(connectionManager) == 0 {
		m.setState(disconnectedState)
		return fmt.Errorf("cannot connect: %w", errNoPendingConnections)
	}

	// multiConnectFatalError is a fatal error that will cause all connections to
	// be shutdown and the websocket to be disconnected.
	var multiConnectFatalError error

	// subscriptionError is a non-fatal error that does not shutdown connections
	var subscriptionError error

	// TODO: Implement concurrency below.
	for i, ws := range connectionManager {
		if ws.setup.Optional && time.Now().Before(ws.retryAt) {
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "%s websocket: [conn:%d] [URL:%s] is retried at %s, skipping", m.exchangeName, i+1, ws.setup.URL, ws.retryAt)
			}
			continue
		}
		setupSubscriptionError, fatalErr := m.connectSetup(ctx, i, ws)
		subscriptionError = common.AppendError(subscriptionError, setupSubscriptionError)
		switch {
		case fatalErr == nil:
			ws.retryFailures, ws.retryAt = 0, time.Time{}
		case ws.setup.Optional:
			// The other connections are up, so the optional one's failure is reported rather than returned
			log.Warnln(log.WebsocketMgr, m.optionalSetupFailed(i, ws, fatalErr))
		default:
			multiConnectFatalError = fatalErr
		}
		if multiConnectFatalError != nil {
			break
		}
	}

	if multiConnectFatalError != nil {
		// Roll back any successful connections and flush subscriptions
		for _, ws := range m.snapshotConnectionManager() {
			connections := m.snapshotManagedConnections(ws)
			if err := m.retireConnections(ws, connections); err != nil {
				log.Errorln(log.WebsocketMgr, err)
			}
			for _, conn := range connections {
				conn.Subscriptions().Clear()
			}
			ws.subscriptions.Clear()
		}
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

	m.Wg.Add(1)
	go m.monitorFrame(ctx, &m.Wg, m.monitorTraffic)

	if m.connectionMonitorRunning.CompareAndSwap(false, true) {
		// This oversees all connections and does not need to be part of wait group management.
		go m.monitorFrame(ctx, nil, m.monitorConnection)
	}

	return subscriptionError
}

// connectSetup dials a setup's connections and subscribes its generated subscriptions. A fatal error means the setup
// could not be connected; a subscription error leaves its connections up. The caller must hold m.m
func (m *Manager) connectSetup(ctx context.Context, i int, ws *websocket) (subscriptionError, fatalErr error) {
	return m.connectSetupWith(ctx, i, ws, nil)
}

// connectSetupWith is connectSetup using the dialled connection, when given, for the first connection it would otherwise
// dial; further connections, for subscriptions beyond one connection's limit, are dialled while m.m is held, moments
// after the first reached the host. The caller discards the dialled connection when it is left unused, and must hold m.m
func (m *Manager) connectSetupWith(ctx context.Context, i int, ws *websocket, dialled *dialledConnection) (subscriptionError, fatalErr error) {
	subs, dial, subscriptionError, fatalErr := m.prepareSetup(i, ws)
	if !dial {
		return subscriptionError, fatalErr
	}

	if len(subs) == 0 {
		if err := m.connectAndSubscribe(ctx, ws, nil, dialled); err != nil {
			return subscriptionError, fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, err)
		}
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: [URL:%s] connected", m.exchangeName, ws.setup.URL)
		}
		return subscriptionError, nil
	}

	// Pre-pass: absorb trackable logical subscriptions onto already-managed connections before batching/capacity routing to avoid misplacement.
	remainingSubs, err := m.absorbTrackableSubscriptionsAndValidate(ctx, ws, subs)
	if err != nil {
		return common.AppendError(subscriptionError, fmt.Errorf("subscription error on [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, err)), nil
	}
	if len(remainingSubs) == 0 {
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: [URL:%s] tracked logical subscriptions on existing connection. [Total Subs: %d] [Tracked: %d]", m.exchangeName, ws.setup.URL, len(subs), len(subs))
		}
		return subscriptionError, nil
	}

	for _, batchCandidates := range common.Batch(remainingSubs, m.MaxSubscriptionsPerConnection) {
		batchRemainingSubs, err := m.absorbTrackableSubscriptionsAndValidate(ctx, ws, batchCandidates)
		if err != nil {
			subscriptionError = common.AppendError(subscriptionError, fmt.Errorf("subscription error on [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, err))
			continue
		}
		if len(batchRemainingSubs) == 0 {
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "%s websocket: [URL:%s] tracked logical subscriptions on existing connection. [Total Subs: %d] [Tracked: %d]", m.exchangeName, ws.setup.URL, len(subs), len(batchCandidates))
			}
			continue
		}
		if err := m.connectAndSubscribe(ctx, ws, batchRemainingSubs, dialled); err != nil {
			if errors.Is(err, common.ErrFatal) {
				return subscriptionError, fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, err)
			}
			subscriptionError = common.AppendError(subscriptionError, fmt.Errorf("subscription error on [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, err))
			continue
		}
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: [URL:%s] connected. [Total Subs: %d] [Subscribed: %d]", m.exchangeName, ws.setup.URL, len(subs), len(batchRemainingSubs))
		}
	}
	return subscriptionError, nil
}

// prepareSetup generates the subscriptions a setup is connected for and checks it can be dialled, reporting whether
// connecting it dials a connection. A setup is skipped, which reads as connected with nothing, when it requires
// authenticated use while that is off, or generates no subscriptions and connects only for them. The caller must hold
// m.m
func (m *Manager) prepareSetup(i int, ws *websocket) (subs subscription.List, dial bool, subscriptionError, fatalErr error) {
	if ws.setup.RequiresAuthentication && !m.CanUseAuthenticatedEndpoints() {
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: [conn:%d] [URL:%s] requires authenticated use, which is off, skipping", m.exchangeName, i+1, ws.setup.URL)
		}
		return nil, false, nil, nil
	}

	if !ws.setup.SubscriptionsNotRequired {
		if ws.setup.GenerateSubscriptions == nil {
			return nil, false, nil, fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, errWebsocketSubscriptionsGeneratorUnset)
		}

		var err error
		subs, err = ws.setup.GenerateSubscriptions() // regenerate state on new connection
		var generationError error
		if err != nil {
			generationError = fmt.Errorf("subscription error on [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, err)
		}
		subscriptionError, fatalErr = collectSubscriptionGenerationError(nil, generationError)
		if fatalErr != nil {
			return nil, false, nil, fmt.Errorf("%s websocket: %w", m.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
		}

		if len(subs) == 0 && !ws.setup.ConnectWithoutSubscriptions {
			// If no subscriptions are generated, we skip this connection.
			if m.verbose {
				log.Debugf(log.WebsocketMgr, "%s websocket: no subscriptions generated for [conn:%d] [URL:%s], skipping", m.exchangeName, i+1, ws.setup.URL)
			}
			return nil, false, subscriptionError, nil
		}
	}

	if ws.setup.Connector == nil {
		return nil, false, subscriptionError, fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, errNoConnectFunc)
	}
	if ws.setup.Handler == nil {
		return nil, false, subscriptionError, fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, errWebsocketDataHandlerUnset)
	}
	if ws.setup.Subscriber == nil && !ws.setup.SubscriptionsNotRequired {
		return nil, false, subscriptionError, fmt.Errorf("cannot connect to [conn:%d] [URL:%s]: %w ", i+1, ws.setup.URL, errWebsocketSubscriberUnset)
	}
	return subs, true, subscriptionError, nil
}

// optionalSetupFailed closes an optional setup's connections after it failed to connect or lost a connection, so the
// others stay up without it, and schedules its next attempt. The caller must hold m.m
func (m *Manager) optionalSetupFailed(i int, ws *websocket, cause error) error {
	connections := m.snapshotManagedConnections(ws)
	cause = common.AppendError(cause, m.retireConnections(ws, connections))
	for _, conn := range connections {
		conn.Subscriptions().Clear()
	}
	ws.subscriptions.Clear()
	ws.retryFailures++
	delay := m.optionalRetryDelay(ws.retryFailures)
	ws.retryAt = time.Now().Add(delay)
	return fmt.Errorf("%s websocket: optional connection [conn:%d] [URL:%s] retried in %s: %w", m.exchangeName, i+1, ws.setup.URL, delay, cause)
}

// optionalRetryDelay returns how long an optional setup waits after its consecutive failures: the monitor delay,
// doubled after each further failure up to a minute, or the monitor delay when that is longer
func (m *Manager) optionalRetryDelay(failures int) time.Duration {
	ceiling := max(time.Minute, m.connectionMonitorDelay)
	delay := m.connectionMonitorDelay
	for range failures - 1 {
		if delay >= ceiling {
			break
		}
		delay *= 2
	}
	return min(delay, ceiling)
}

// isolateConnectionFault handles a connection fault without reconnecting the other connections when the connection is
// not registered, having been retired or not yet been connected by an optional setup's retry, or belongs to an optional
// setup, which is retired and retried alone. The fault channel holds one fault and drops the rest, so every managed
// connection is checked: a required setup's lost connection still needs the reconnection, and an optional setup's is
// retired too. It reports whether it handled the fault
func (m *Manager) isolateConnectionFault(err error) bool {
	var fault *connectionFaultError
	if !m.useMultiConnectionManagement || !errors.As(err, &fault) {
		return false
	}
	m.m.Lock()
	defer m.m.Unlock()
	faulted, managed := m.managedWebsocket(fault.connection)
	if managed && !faulted.setup.Optional {
		return false
	}
	// Connections are retired and shut down under m.m, so a managed connection that is not connected has been lost
	lost := func(c Connection) bool {
		status, ok := c.(interface{ IsConnected() bool })
		return ok && !status.IsConnected()
	}
	setups := m.snapshotConnectionManager()
	for _, ws := range setups {
		if !ws.setup.Optional && slices.ContainsFunc(m.snapshotManagedConnections(ws), lost) {
			return false
		}
	}
	for i, ws := range setups {
		switch {
		case !ws.setup.Optional:
		case ws == faulted:
			log.Warnln(log.WebsocketMgr, m.optionalSetupFailed(i, ws, err))
		case slices.ContainsFunc(m.snapshotManagedConnections(ws), lost):
			log.Warnln(log.WebsocketMgr, m.optionalSetupFailed(i, ws, ErrNotConnected))
		}
	}
	return true
}

// retryOptionalSetups reconnects each optional setup whose connection failed once its retry is due
func (m *Manager) retryOptionalSetups(ctx context.Context) {
	for i, ws := range m.snapshotConnectionManager() {
		m.retryOptionalSetup(ctx, i, ws)
	}
}

// retryOptionalSetup reconnects an optional setup whose connection failed once its retry is due. Its first connection
// is dialled with m.m released, as a connector reaching for an unreachable host takes as long as its timeouts allow, and
// every call taking the lock, such as GetConnection before each websocket request, would wait for it
func (m *Manager) retryOptionalSetup(ctx context.Context, i int, ws *websocket) {
	m.m.Lock()
	defer m.m.Unlock()
	if !m.optionalRetryDue(ws) {
		return
	}
	// A setup that would not be dialled, being skipped or failing before its dial, is settled without one
	_, dial, subscriptionError, fatalErr := m.prepareSetup(i, ws)
	if dial {
		dialled := m.dialUnlocked(ctx, i, ws)
		if dialled == nil {
			return
		}
		// The setup is prepared again, so it is subscribed to what it generates once dialled rather than before
		subscriptionError, fatalErr = m.connectSetupWith(ctx, i, ws, dialled)
		if !dialled.used {
			m.discardDialled(ctx, ws, dialled)
		}
	}
	if fatalErr != nil {
		log.Warnln(log.WebsocketMgr, m.optionalSetupFailed(i, ws, fatalErr))
		return
	}
	ws.retryFailures, ws.retryAt = 0, time.Time{}
	if subscriptionError != nil {
		log.Errorln(log.WebsocketMgr, subscriptionError)
	}
}

// optionalRetryDue reports whether an optional setup that failed to connect is due to be retried: the manager is
// enabled and connected, the setup's retry time has passed and nothing has connected it since. The caller must hold m.m
func (m *Manager) optionalRetryDue(ws *websocket) bool {
	return m.IsEnabled() && m.IsConnected() && ws.setup.Optional && ws.retryFailures != 0 && !time.Now().Before(ws.retryAt) &&
		len(m.snapshotManagedConnections(ws)) == 0
}

// dialledConnection is a connection whose setup's connector ran without the manager's lock, and the error it returned
type dialledConnection struct {
	conn *connection
	err  error
	// used is set once the connection is taken for one of the setup's connections
	used bool
}

// dialUnlocked runs a setup's connector on a new connection with m.m released, and returns the connection once m.m is
// held again, or nil once it has been discarded because the manager or the setup changed while it was dialled. The
// caller must hold m.m, which dialUnlocked releases and takes again
func (m *Manager) dialUnlocked(ctx context.Context, i int, ws *websocket) *dialledConnection {
	conn := m.createConnectionFromSetup(ws.setup)
	// A shutdown waits on m.Wg while holding m.m, so the routines the connector starts join a group of their own, as
	// adding to m.Wg during that wait would misuse it. Once m.m is held again, m.Wg waits on that group, and the
	// connection's later routines join m.Wg itself
	var routines sync.WaitGroup
	conn.Wg = &routines
	connector, shutdownC := ws.setup.Connector, m.ShutdownC
	m.m.Unlock()
	err := connector(ctx, conn)
	m.m.Lock()
	m.Wg.Go(routines.Wait)
	conn.Wg = &m.Wg
	dialled := &dialledConnection{conn: conn, err: err}
	// A shutdown meanwhile closed the shutdown channel the connection's routines stop on, even if the manager has
	// reconnected since, and another call may have connected the setup or counted a failure for it
	if m.ShutdownC != shutdownC || !m.optionalRetryDue(ws) {
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: [conn:%d] [URL:%s] changed while its retry was dialled, discarding the connection", m.exchangeName, i+1, ws.setup.URL)
		}
		m.discardDialled(ctx, ws, dialled)
		return nil
	}
	return dialled
}

// discardDialled shuts down a dialled connection a setup does not keep, without counting a failure for it, and calls the
// setup's disconnect hook when its connector succeeded, as no reader will. The caller must hold m.m
func (m *Manager) discardDialled(ctx context.Context, ws *websocket, dialled *dialledConnection) {
	if err := dialled.conn.Shutdown(); err != nil {
		log.Warnf(log.WebsocketMgr, "%v websocket: failed to shutdown connection: %v", m.exchangeName, err)
	}
	if dialled.err == nil && ws.setup.OnDisconnect != nil {
		ws.setup.OnDisconnect(ctx, dialled.conn)
	}
}

// createConnectAndSubscribe creates a new connection for the websocket and subscribes to subs on it
// The connection's store also records subscriptions accepted before the subscriber fails, because flushing uses it to decide whether the connection is still in use
func (m *Manager) createConnectAndSubscribe(ctx context.Context, ws *websocket, subs subscription.List) error {
	return m.connectAndSubscribe(ctx, ws, subs, nil)
}

// connectAndSubscribe is createConnectAndSubscribe using the dialled connection, when given and not yet used, rather
// than creating and dialling one
func (m *Manager) connectAndSubscribe(ctx context.Context, ws *websocket, subs subscription.List, dialled *dialledConnection) error {
	if m.MaxSubscriptionsPerConnection > 0 && len(subs) > m.MaxSubscriptionsPerConnection {
		return fmt.Errorf("%w %w: max subs allowed %d, requested %d", common.ErrFatal, errSubscriptionsExceedsLimit, m.MaxSubscriptionsPerConnection, len(subs))
	}
	if ws.setup.SubscriptionsNotRequired && len(subs) != 0 {
		return fmt.Errorf("%w %w: subscriptions were provided but not required", common.ErrFatal, ErrSubscriptionFailure)
	}

	var conn *connection
	var err error
	if dialled != nil && !dialled.used {
		dialled.used = true
		conn, err = dialled.conn, dialled.err
	} else {
		conn = m.createConnectionFromSetup(ws.setup)
		err = ws.setup.Connector(ctx, conn)
	}

	// The connection is only tracked once connected, so a rollback cannot reach it; close it here in case the
	// connector dialled before failing.
	if err != nil {
		return fmt.Errorf("%w: %w", common.ErrFatal, common.AppendError(err, conn.Shutdown()))
	}

	var connectErr error
	switch {
	case !conn.IsConnected():
		connectErr = ErrNotConnected
	case ws.setup.RequiresAuthentication && !m.CanUseAuthenticatedEndpoints():
		// Turned off while the setup was dialled, which leaves it nothing to connect for
		connectErr = errAuthenticatedUseOff
	}
	if connectErr != nil {
		err := fmt.Errorf("%w: %w", common.ErrFatal, common.AppendError(connectErr, conn.Shutdown()))
		// No reader will start, so the disconnect hook undoes what the connector set up here
		if ws.setup.OnDisconnect != nil {
			ws.setup.OnDisconnect(ctx, conn)
		}
		return err
	}

	m.trackConnection(conn, ws)

	done := make(chan struct{})
	m.connectionManagerMu.Lock()
	if m.readersDone == nil {
		m.readersDone = make(map[Connection]chan struct{})
	}
	m.readersDone[conn] = done
	m.connectionManagerMu.Unlock()
	m.Wg.Add(1)
	go m.managedReader(ctx, conn, ws.setup, done)

	// A setup requiring authenticated use found it on above, so it signs in even if it has been turned off since
	if ws.setup.Authenticate != nil && (ws.setup.RequiresAuthentication || m.CanUseAuthenticatedEndpoints()) {
		if err := ws.setup.Authenticate(ctx, conn); err != nil {
			// The connection is no use to its setup, and a flush keeps the setup's other connections rather than
			// rolling the setup back, so it is retired here
			return fmt.Errorf("%w %w: %w", common.ErrFatal, errFailedToAuthenticate, common.AppendError(err, m.retireConnections(ws, []Connection{conn})))
		}
	}

	if len(subs) == 0 {
		return nil // A connection kept without subscriptions
	}

	if err := ws.setup.Subscriber(ctx, conn, subs); err != nil {
		return common.AppendError(fmt.Errorf("%w: %w", ErrSubscriptionFailure, err), recordConnectionSubscriptions(conn.Subscriptions(), ws.subscriptions, subs))
	}
	if missing := ws.subscriptions.Missing(subs); len(missing) > 0 {
		return common.AppendError(fmt.Errorf("%w: %w %q", ErrSubscriptionFailure, ErrSubscriptionsNotAdded, missing), recordConnectionSubscriptions(conn.Subscriptions(), ws.subscriptions, subs))
	}

	connSubsStore := conn.Subscriptions()
	for _, sub := range ws.subscriptions.Contained(subs) {
		// Store subscription against this specific connection for tracking
		if err := connSubsStore.Add(sub); err != nil {
			return fmt.Errorf("%w: adding subscriptions to the specific connection subscription store: %w", ErrSubscriptionFailure, err)
		}
	}

	return nil
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
func (m *Manager) Enable(ctx context.Context) error {
	if m.IsConnected() || m.IsEnabled() {
		return fmt.Errorf("%s %w", m.exchangeName, ErrWebsocketAlreadyEnabled)
	}

	m.setEnabled(true)
	return m.Connect(ctx)
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (m *Manager) Shutdown() error {
	m.m.Lock()
	defer m.m.Unlock()
	return m.shutdown()
}

func (m *Manager) shutdown() error {
	if m.IsConnecting() {
		return fmt.Errorf("%v %w: %w ", m.exchangeName, errCannotShutdown, errAlreadyReconnecting)
	}

	if !m.IsConnected() {
		return fmt.Errorf("%v %w: %w", m.exchangeName, errCannotShutdown, ErrNotConnected)
	}

	if m.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: shutting down websocket", m.exchangeName)
	}

	nonFatalCloseConnectionErrors, err := m.teardown()
	if err != nil {
		return err
	}

	if m.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: completed websocket shutdown", m.exchangeName)
	}

	if nonFatalCloseConnectionErrors != nil {
		log.Warnf(log.WebsocketMgr, "%v websocket: shutdown error: %v", m.exchangeName, nonFatalCloseConnectionErrors)
	}

	return nil
}

// teardown closes every connection, waits for the routines tracked by Wg and renews ShutdownC, leaving the manager
// disconnected and ready for a fresh connection attempt; a connection an optional setup's retry is dialling is closed by
// the retry once its connector returns. It is shared by shutdown and a failed connect, which can leave sockets and
// readers behind. Close errors are returned separately as they are non-fatal. The caller must hold m.m.
func (m *Manager) teardown() (nonFatalCloseConnectionErrors, err error) {
	// During the shutdown process, all errors are treated as non-fatal to avoid issues when the connection has already
	// been closed. In such cases, attempting to close the connection may result in a
	// "failed to send closeNotify alert (but connection was closed anyway)" error. Treating these errors as non-fatal
	// prevents the shutdown process from being interrupted, which could otherwise trigger a continuous traffic monitor
	// cycle and potentially block the initiation of a new connection.
	//
	// Shutdown managed connections, which stay registered until their readers and disconnect hooks have finished
	m.connectionManagerMu.RLock()
	var managed []Connection
	for _, ws := range m.connectionManager {
		managed = append(managed, ws.connections...)
	}
	m.connectionManagerMu.RUnlock()
	for _, conn := range managed {
		if err := conn.Shutdown(); err != nil {
			nonFatalCloseConnectionErrors = common.AppendError(nonFatalCloseConnectionErrors, err)
		}
	}

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
	close(m.ShutdownC)
	m.setState(disconnectedState)
	m.Wg.Wait()

	// Flush any subscriptions from last connection across any managed connections, now no reader can still use them
	m.connectionManagerMu.Lock()
	for _, ws := range m.connectionManager {
		for _, conn := range ws.connections {
			conn.Subscriptions().Clear()
		}
		ws.connections = nil
		ws.subscriptions.Clear()
	}
	// Clean map of old connections
	clear(m.connections)
	clear(m.readersDone)
	m.connectionManagerMu.Unlock()
	// flush any subscriptions from last connection if needed
	m.subscriptions.Clear()
	m.ShutdownC = make(chan struct{})

	for _, conn := range []Connection{m.Conn, m.AuthConn} {
		if conn == nil {
			continue
		}
		conn, ok := conn.(*connection)
		if !ok {
			return nonFatalCloseConnectionErrors, fmt.Errorf("%s websocket: %w", m.exchangeName, common.GetTypeAssertError("*connection", conn))
		}
		conn.shutdown = m.ShutdownC
	}

	// Drain residual error in the single buffered channel, this mitigates
	// the cycle when `Connect` is called again and the connectionMonitor
	// starts but there is an old error in the channel.
	drain(m.ReadMessageErrors)

	return nonFatalCloseConnectionErrors, nil
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
func (m *Manager) SetWebsocketURL(u string, auth, reconnect bool) error {
	if m.useMultiConnectionManagement {
		// TODO: Add functionality for multi-connection management to change URL
		return fmt.Errorf("%s: %w", m.exchangeName, errCannotChangeConnectionURL)
	}
	defaultVals := u == "" || u == config.WebsocketURLNonDefaultMessage
	if auth {
		if defaultVals {
			u = m.defaultURLAuth
		}

		err := checkWebsocketURL(u)
		if err != nil {
			return err
		}
		m.runningURLAuth = u

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: setting authenticated websocket URL: %s\n", m.exchangeName, removeURLQueryString(u))
		}

		if m.AuthConn != nil {
			m.AuthConn.SetURL(u)
		}
	} else {
		if defaultVals {
			u = m.defaultURL
		}
		err := checkWebsocketURL(u)
		if err != nil {
			return err
		}
		m.runningURL = u

		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%s websocket: setting unauthenticated websocket URL: %s\n", m.exchangeName, removeURLQueryString(u))
		}

		if m.Conn != nil {
			m.Conn.SetURL(u)
		}
	}

	if m.IsConnected() && reconnect {
		log.Debugf(log.WebsocketMgr, "%s websocket: flushing websocket connection to %s\n", m.exchangeName, removeURLQueryString(u))
		return m.Shutdown()
	}
	return nil
}

// GetWebsocketURL returns the running websocket URL
func (m *Manager) GetWebsocketURL() string {
	return m.runningURL
}

// GetConfiguredWebsocketURLs returns known websocket connection URLs.
func (m *Manager) GetConfiguredWebsocketURLs() ([]string, error) {
	if err := common.NilGuard(m); err != nil {
		return nil, err
	}

	m.m.Lock()
	defer m.m.Unlock()

	if m.useMultiConnectionManagement {
		m.connectionManagerMu.RLock()
		defer m.connectionManagerMu.RUnlock()
		urls := make([]string, 0, len(m.connectionManager))
		seen := make(map[string]struct{}, len(m.connectionManager))
		for _, ws := range m.connectionManager {
			if ws == nil || ws.setup.URL == "" {
				continue
			}
			if _, ok := seen[ws.setup.URL]; ok {
				continue
			}
			seen[ws.setup.URL] = struct{}{}
			urls = append(urls, ws.setup.URL)
		}
		return urls, nil
	}

	if m.runningURL != "" {
		return []string{m.runningURL}, nil
	}
	if m.defaultURL != "" {
		return []string{m.defaultURL}, nil
	}
	return nil, nil
}

// SetProxyAddress sets websocket proxy address
func (m *Manager) SetProxyAddress(ctx context.Context, proxyAddr string) error {
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

	m.connectionManagerMu.RLock()
	for _, ws := range m.connectionManager {
		for _, conn := range ws.connections {
			conn.SetProxy(proxyAddr)
		}
	}
	m.connectionManagerMu.RUnlock()
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
	return m.connect(ctx)
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
		removeURLErrorQueryString(err)
		return err
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return fmt.Errorf("cannot set %w %s", errInvalidWebsocketURL, removeURLQueryString(s))
	}
	return nil
}

// Reader reads and handles data from a specific connection
func (m *Manager) Reader(ctx context.Context, conn Connection, handler func(ctx context.Context, conn Connection, message []byte) error) {
	defer m.Wg.Done()
	m.read(ctx, conn, handler)
}

// managedReader reads and handles data from a managed connection, then calls its setup's OnDisconnect before done is
// closed and the wait group released, so retiring the connection or shutting down waits for it
func (m *Manager) managedReader(ctx context.Context, conn Connection, setup *ConnectionSetup, done chan struct{}) {
	defer m.Wg.Done()
	defer close(done)
	if setup.OnDisconnect != nil {
		defer setup.OnDisconnect(ctx, conn)
	}
	m.read(ctx, conn, setup.Handler)
}

// retireConnections shuts a setup's connections down, then unregisters them; a setup with a disconnect hook first has its
// readers awaited, so the hook runs while its connection is still known. Setups without one are not awaited, as a reader
// whose handler takes the manager's lock would otherwise wait on its own retirement. The caller must hold m.m
func (m *Manager) retireConnections(ws *websocket, connections []Connection) error {
	var errs error
	for _, conn := range connections {
		if err := conn.Shutdown(); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	for _, conn := range connections {
		if ws.setup.OnDisconnect == nil {
			break
		}
		m.connectionManagerMu.RLock()
		done := m.readersDone[conn]
		m.connectionManagerMu.RUnlock()
		if done != nil {
			<-done
		}
	}
	m.connectionManagerMu.Lock()
	for _, conn := range connections {
		delete(m.connections, conn)
		delete(m.readersDone, conn)
		ws.connections = slices.DeleteFunc(ws.connections, func(c Connection) bool { return c == conn })
	}
	m.connectionManagerMu.Unlock()
	return errs
}

func (m *Manager) read(ctx context.Context, conn Connection, handler func(ctx context.Context, conn Connection, message []byte) error) {
	for {
		resp := conn.ReadMessage()
		if resp.Raw == nil {
			return // Connection has been closed
		}
		if err := handler(ctx, conn, resp.Raw); err != nil {
			err = fmt.Errorf("connection URL:[%v] error: %w", removeURLQueryString(conn.GetURL()), err)
			if errSend := m.DataHandler.Send(ctx, err); errSend != nil {
				log.Errorf(log.WebsocketMgr, "%s: %s %s", m.exchangeName, errSend, err)
			}
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
type ClosureFrame func(ctx context.Context) func() bool

// monitorFrame monitors a specific websocket component or critical system. It will exit if the observer returns true
// This is used for monitoring data throughput, connection status and other critical websocket components. The waitgroup
// is optional and is used to signal when the monitor has finished.
func (m *Manager) monitorFrame(ctx context.Context, wg *sync.WaitGroup, fn ClosureFrame) {
	if wg != nil {
		defer wg.Done()
	}
	observe := fn(ctx)
	for {
		if observe() {
			return
		}
	}
}

// monitorConnection monitors the connection and attempts to reconnect if the connection is lost
func (m *Manager) monitorConnection(ctx context.Context) func() bool {
	timer := time.NewTimer(m.connectionMonitorDelay)
	return func() bool { return m.observeConnection(ctx, timer) }
}

// observeConnection observes the connection and attempts to reconnect if the connection is lost
func (m *Manager) observeConnection(ctx context.Context, t *time.Timer) (exit bool) {
	select {
	case err := <-m.ReadMessageErrors:
		if errors.Is(err, errConnectionFault) && !m.isolateConnectionFault(err) {
			log.Warnf(log.WebsocketMgr, "%v websocket has been disconnected. Reason: %v", m.exchangeName, err)
			if m.IsConnected() {
				if shutdownErr := m.Shutdown(); shutdownErr != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket: connectionMonitor shutdown err: %s", m.exchangeName, shutdownErr)
				}
			} else {
				m.state.CompareAndSwap(connectingState, disconnectedState)
			}
		}
		// Speedier reconnection, instead of waiting for the next cycle.
		if m.IsEnabled() && (!m.IsConnected() && !m.IsConnecting()) {
			if connectErr := m.Connect(ctx); connectErr != nil {
				log.Errorln(log.WebsocketMgr, connectErr)
			}
		}
		if err := m.DataHandler.Send(ctx, err); err != nil {
			log.Errorf(log.WebsocketMgr, "%v websocket: connectionMonitor data handler err: %s", m.exchangeName, err)
		}
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
			err := m.Connect(ctx)
			if err != nil {
				log.Errorln(log.WebsocketMgr, err)
			}
		} else if m.useMultiConnectionManagement {
			m.retryOptionalSetups(ctx)
		}
		t.Reset(m.connectionMonitorDelay)
	}
	return false
}

// monitorTraffic monitors to see if there has been traffic within the trafficTimeout time window. If there is no traffic
// the connection is shutdown and will be reconnected by the connectionMonitor routine.
func (m *Manager) monitorTraffic(context.Context) func() bool {
	return func() bool {
		return m.observeTraffic(time.After(m.trafficTimeout), func() {
			go func() {
				if err := m.Shutdown(); err != nil {
					log.Errorf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown err: %s", m.exchangeName, err)
				}
			}()
		})
	}
}

func (m *Manager) observeTraffic(timeout <-chan time.Time, onTimeout func()) bool {
	select {
	case <-m.ShutdownC:
		if m.verbose {
			log.Debugf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown message received", m.exchangeName)
		}
	case <-timeout:
		if m.IsConnecting() || signalReceived(m.TrafficAlert) {
			return false
		}
		if m.verbose {
			log.Warnf(log.WebsocketMgr, "%v websocket: has not received a traffic alert in %v. Reconnecting", m.exchangeName, m.trafficTimeout)
		}
		if m.IsConnected() && onTimeout != nil {
			onTimeout()
		}
	}
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

// GetConnection returns the first available connection for a websocket by message filter (defined in exchange package _wrapper.go websocket connection)
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

	m.connectionManagerMu.RLock()
	defer m.connectionManagerMu.RUnlock()
	for _, ws := range m.connectionManager {
		if ws.setup.MessageFilter != messageFilter {
			continue
		}
		if len(ws.connections) == 0 {
			return nil, fmt.Errorf("%s: %s %w associated with message filter: '%v'", m.exchangeName, ws.setup.URL, ErrNotConnected, messageFilter)
		}
		return ws.connections[0], nil
	}

	return nil, fmt.Errorf("%s: %w associated with message filter: '%v'", m.exchangeName, ErrRequestRouteNotFound, messageFilter)
}
