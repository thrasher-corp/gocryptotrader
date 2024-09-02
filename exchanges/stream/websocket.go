package stream

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"time"

	"github.com/gorilla/websocket"
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
	errExchangeConfigEmpty                  = errors.New("exchange config is empty")
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
)

var (
	globalReporter       Reporter
	trafficCheckInterval = 100 * time.Millisecond
)

// SetupGlobalReporter sets a reporter interface to be used
// for all exchange requests
func SetupGlobalReporter(r Reporter) {
	globalReporter = r
}

// NewWebsocket initialises the websocket struct
func NewWebsocket() *Websocket {
	return &Websocket{
		DataHandler:       make(chan interface{}, jobBuffer),
		ToRoutine:         make(chan interface{}, jobBuffer),
		ShutdownC:         make(chan struct{}),
		TrafficAlert:      make(chan struct{}, 1),
		ReadMessageErrors: make(chan error),
		Match:             NewMatch(),
		subscriptions:     subscription.NewStore(),
		features:          &protocol.Features{},
		Orderbook:         buffer.Orderbook{},
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

	if s.Connector == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketConnectorUnset)
	}
	w.connector = s.Connector

	if s.Subscriber == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketSubscriberUnset)
	}
	w.Subscriber = s.Subscriber

	if w.features.Unsubscribe && s.Unsubscriber == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketUnsubscriberUnset)
	}
	w.connectionMonitorDelay = s.ExchangeConfig.ConnectionMonitorDelay
	if w.connectionMonitorDelay <= 0 {
		w.connectionMonitorDelay = config.DefaultConnectionMonitorDelay
	}
	w.Unsubscriber = s.Unsubscriber

	if s.GenerateSubscriptions == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketSubscriptionsGeneratorUnset)
	}
	w.GenerateSubs = s.GenerateSubscriptions

	if s.DefaultURL == "" {
		return fmt.Errorf("%s websocket %w", w.exchangeName, errDefaultURLIsEmpty)
	}
	w.defaultURL = s.DefaultURL
	if s.RunningURL == "" {
		return fmt.Errorf("%s websocket %w", w.exchangeName, errRunningURLIsEmpty)
	}
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

	if s.ExchangeConfig.WebsocketTrafficTimeout < time.Second {
		return fmt.Errorf("%s %w cannot be less than %s",
			w.exchangeName,
			errInvalidTrafficTimeout,
			time.Second)
	}
	w.trafficTimeout = s.ExchangeConfig.WebsocketTrafficTimeout

	w.ShutdownC = make(chan struct{})
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

	return nil
}

// SetupNewConnection sets up an auth or unauth streaming connection
func (w *Websocket) SetupNewConnection(c ConnectionSetup) error {
	if w == nil {
		return fmt.Errorf("%w: %w", errConnSetup, errWebsocketIsNil)
	}

	if c.ResponseCheckTimeout == 0 &&
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

	connectionURL := w.GetWebsocketURL()
	if c.URL != "" {
		connectionURL = c.URL
	}

	if c.ConnectionLevelReporter == nil {
		c.ConnectionLevelReporter = w.ExchangeLevelReporter
	}

	if c.ConnectionLevelReporter == nil {
		c.ConnectionLevelReporter = globalReporter
	}

	newConn := &WebsocketConnection{
		_URL:                     connectionURL,
		responseMaxLimit:         c.ResponseMaxLimit,
		rateLimit:                c.RateLimit,
		reporter:                 c.ConnectionLevelReporter,
		parent:                   w,
		bespokeGenerateMessageID: c.BespokeGenerateMessageID,
	}

	if c.Authenticated {
		w.AuthConn = newConn
	} else {
		w.Conn = newConn
	}

	return nil
}

// Connect initiates a websocket connection by using a package defined connection
// function
func (w *Websocket) Connect() error {
	if w.connector == nil {
		return errNoConnectFunc
	}
	w.m.Lock()
	defer w.m.Unlock()

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

	w.dataMonitor()
	w.trafficMonitor()
	w.setState(connectingState)

	err := w.connector()
	if err != nil {
		w.setState(disconnectedState)
		return fmt.Errorf("%v Error connecting %w", w.exchangeName, err)
	}
	w.setState(connectedState)

	if !w.IsConnectionMonitorRunning() {
		err = w.connectionMonitor()
		if err != nil {
			log.Errorf(log.WebsocketMgr, "%s cannot start websocket connection monitor %v", w.GetName(), err)
		}
	}

	subs, err := w.GenerateSubs() // regenerate state on new connection
	if err != nil {
		return fmt.Errorf("%s websocket: %w", w.exchangeName, common.AppendError(ErrSubscriptionFailure, err))
	}
	if len(subs) != 0 {
		if err := w.SubscribeToChannels(subs); err != nil {
			return err
		}
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

// dataMonitor monitors job throughput and logs if there is a back log of data
func (w *Websocket) dataMonitor() {
	if w.IsDataMonitorRunning() {
		return
	}
	w.setDataMonitorRunning(true)
	w.Wg.Add(1)

	go func() {
		defer func() {
			w.setDataMonitorRunning(false)
			w.Wg.Done()
		}()
		dropped := 0
		for {
			select {
			case <-w.ShutdownC:
				return
			case d := <-w.DataHandler:
				select {
				case w.ToRoutine <- d:
					if dropped != 0 {
						log.Infof(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer recovered; %d messages were dropped", w.exchangeName, dropped)
						dropped = 0
					}
				default:
					if dropped == 0 {
						// If this becomes prone to flapping we could drain the buffer, but that's extreme and we'd like to avoid it if possible
						log.Warnf(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer full; dropping messages", w.exchangeName)
					}
					dropped++
				}
			}
		}
	}()
}

// connectionMonitor ensures that the WS keeps connecting
func (w *Websocket) connectionMonitor() error {
	if w.checkAndSetMonitorRunning() {
		return errAlreadyRunning
	}
	delay := w.connectionMonitorDelay

	go func() {
		timer := time.NewTimer(delay)
		for {
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
				timer.Stop()
				w.setConnectionMonitorRunning(false)
				return
			}
			select {
			case err := <-w.ReadMessageErrors:
				w.DataHandler <- err
				if IsDisconnectionError(err) {
					log.Warnf(log.WebsocketMgr, "%v websocket has been disconnected. Reason: %v", w.exchangeName, err)
					if w.IsConnected() {
						if shutdownErr := w.Shutdown(); shutdownErr != nil {
							log.Errorf(log.WebsocketMgr, "%v websocket: connectionMonitor shutdown err: %s", w.exchangeName, shutdownErr)
						}
					}
				}
			case <-timer.C:
				if !w.IsConnecting() && !w.IsConnected() {
					err := w.Connect()
					if err != nil {
						log.Errorln(log.WebsocketMgr, err)
					}
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(delay)
			}
		}
	}()
	return nil
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *Websocket) Shutdown() error {
	w.m.Lock()
	defer w.m.Unlock()

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

	if w.Conn != nil {
		if err := w.Conn.Shutdown(); err != nil {
			return err
		}
	}

	if w.AuthConn != nil {
		if err := w.AuthConn.Shutdown(); err != nil {
			return err
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

	if w.features.Subscribe {
		newsubs, err := w.GenerateSubs()
		if err != nil {
			return err
		}

		subs, unsubs := w.GetChannelDifference(newsubs)
		if w.features.Unsubscribe {
			if len(unsubs) != 0 {
				err := w.UnsubscribeChannels(unsubs)
				if err != nil {
					return err
				}
			}
		}

		if len(subs) < 1 {
			return nil
		}
		return w.SubscribeToChannels(subs)
	} else if w.features.FullPayloadSubscribe {
		// FullPayloadSubscribe means that the endpoint requires all
		// subscriptions to be sent via the websocket connection e.g. if you are
		// subscribed to ticker and orderbook but require trades as well, you
		// would need to send ticker, orderbook and trades channel subscription
		// messages.
		newsubs, err := w.GenerateSubs()
		if err != nil {
			return err
		}

		if len(newsubs) != 0 {
			// Purge subscription list as there will be conflicts
			w.subscriptions.Clear()
			return w.SubscribeToChannels(newsubs)
		}
		return nil
	}

	if err := w.Shutdown(); err != nil {
		return err
	}
	return w.Connect()
}

// trafficMonitor waits trafficCheckInterval before checking for a trafficAlert
// 1 slot buffer means that connection will only write to trafficAlert once per trafficCheckInterval to avoid read/write flood in high traffic
// Otherwise we Shutdown the connection after trafficTimeout, unless it's connecting. connectionMonitor is responsible for Connecting again
func (w *Websocket) trafficMonitor() {
	if w.IsTrafficMonitorRunning() {
		return
	}
	w.setTrafficMonitorRunning(true)
	w.Wg.Add(1)

	go func() {
		t := time.NewTimer(w.trafficTimeout)
		for {
			select {
			case <-w.ShutdownC:
				if w.verbose {
					log.Debugf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown message received", w.exchangeName)
				}
				t.Stop()
				w.setTrafficMonitorRunning(false)
				w.Wg.Done()
				return
			case <-time.After(trafficCheckInterval):
				select {
				case <-w.TrafficAlert:
					if !t.Stop() {
						<-t.C
					}
					t.Reset(w.trafficTimeout)
				default:
				}
			case <-t.C:
				checkAgain := w.IsConnecting()
				select {
				case <-w.TrafficAlert:
					checkAgain = true
				default:
				}
				if checkAgain {
					t.Reset(w.trafficTimeout)
					break
				}
				if w.verbose {
					log.Warnf(log.WebsocketMgr, "%v websocket: has not received a traffic alert in %v. Reconnecting", w.exchangeName, w.trafficTimeout)
				}
				w.setTrafficMonitorRunning(false) // Cannot defer lest Connect is called after Shutdown but before deferred call
				w.Wg.Done()                       // Without this the w.Shutdown() call below will deadlock
				if w.IsConnected() {
					err := w.Shutdown()
					if err != nil {
						log.Errorf(log.WebsocketMgr, "%v websocket: trafficMonitor shutdown err: %s", w.exchangeName, err)
					}
				}
				return
			}
		}
	}()
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

func (w *Websocket) setTrafficMonitorRunning(b bool) {
	w.trafficMonitorRunning.Store(b)
}

// IsTrafficMonitorRunning returns status of the traffic monitor
func (w *Websocket) IsTrafficMonitorRunning() bool {
	return w.trafficMonitorRunning.Load()
}

func (w *Websocket) checkAndSetMonitorRunning() (alreadyRunning bool) {
	return !w.connectionMonitorRunning.CompareAndSwap(false, true)
}

func (w *Websocket) setConnectionMonitorRunning(b bool) {
	w.connectionMonitorRunning.Store(b)
}

// IsConnectionMonitorRunning returns status of connection monitor
func (w *Websocket) IsConnectionMonitorRunning() bool {
	return w.connectionMonitorRunning.Load()
}

func (w *Websocket) setDataMonitorRunning(b bool) {
	w.dataMonitorRunning.Store(b)
}

// IsDataMonitorRunning returns status of data monitor
func (w *Websocket) IsDataMonitorRunning() bool {
	return w.dataMonitorRunning.Load()
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
			log.Debugf(log.WebsocketMgr,
				"%s websocket: setting authenticated websocket URL: %s\n",
				w.exchangeName,
				url)
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
			log.Debugf(log.WebsocketMgr,
				"%s websocket: setting unauthenticated websocket URL: %s\n",
				w.exchangeName,
				url)
		}

		if w.Conn != nil {
			w.Conn.SetURL(url)
		}
	}

	if w.IsConnected() && reconnect {
		log.Debugf(log.WebsocketMgr,
			"%s websocket: flushing websocket connection to %s\n",
			w.exchangeName,
			url)
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
func (w *Websocket) GetChannelDifference(newSubs subscription.List) (sub, unsub subscription.List) {
	if w.subscriptions == nil {
		w.subscriptions = subscription.NewStore()
	}
	return w.subscriptions.Diff(newSubs)
}

// UnsubscribeChannels unsubscribes from a list of websocket channel
func (w *Websocket) UnsubscribeChannels(channels subscription.List) error {
	if w.subscriptions == nil || len(channels) == 0 {
		return nil // No channels to unsubscribe from is not an error
	}
	for _, s := range channels {
		if w.subscriptions.Get(s) == nil {
			return fmt.Errorf("%w: %s", subscription.ErrNotFound, s)
		}
	}
	return w.Unsubscriber(channels)
}

// ResubscribeToChannel resubscribes to channel
// Sets state to Resubscribing, and exchanges which want to maintain a lock on it can respect this state and not RemoveSubscription
// Errors if subscription is already subscribing
func (w *Websocket) ResubscribeToChannel(s *subscription.Subscription) error {
	l := subscription.List{s}
	if err := s.SetState(subscription.ResubscribingState); err != nil {
		return fmt.Errorf("%w: %s", err, s)
	}
	if err := w.UnsubscribeChannels(l); err != nil {
		return err
	}
	return w.SubscribeToChannels(l)
}

// SubscribeToChannels subscribes to websocket channels using the exchange specific Subscriber method
// Errors are returned for duplicates or exceeding max Subscriptions
func (w *Websocket) SubscribeToChannels(subs subscription.List) error {
	if slices.Contains(subs, nil) {
		return fmt.Errorf("%w: List parameter contains an nil element", common.ErrNilPointer)
	}
	if err := w.checkSubscriptions(subs); err != nil {
		return err
	}
	if err := w.Subscriber(subs); err != nil {
		return fmt.Errorf("%w: %w", ErrSubscriptionFailure, err)
	}
	return nil
}

// AddSubscriptions adds subscriptions to the subscription store
// Sets state to Subscribing unless the state is already set
func (w *Websocket) AddSubscriptions(subs ...*subscription.Subscription) error {
	if w == nil {
		return fmt.Errorf("%w: AddSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	if w.subscriptions == nil {
		w.subscriptions = subscription.NewStore()
	}
	var errs error
	for _, s := range subs {
		if s.State() == subscription.InactiveState {
			if err := s.SetState(subscription.SubscribingState); err != nil {
				errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
			}
		}
		if err := w.subscriptions.Add(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// AddSuccessfulSubscriptions marks subscriptions as subscribed and adds them to the subscription store
func (w *Websocket) AddSuccessfulSubscriptions(subs ...*subscription.Subscription) error {
	if w == nil {
		return fmt.Errorf("%w: AddSuccessfulSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	if w.subscriptions == nil {
		w.subscriptions = subscription.NewStore()
	}
	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.SubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if err := w.subscriptions.Add(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// RemoveSubscriptions removes subscriptions from the subscription list and sets the status to Unsubscribed
func (w *Websocket) RemoveSubscriptions(subs ...*subscription.Subscription) error {
	if w == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on nil Websocket", common.ErrNilPointer)
	}
	if w.subscriptions == nil {
		return fmt.Errorf("%w: RemoveSubscriptions called on uninitialised Websocket", common.ErrNilPointer)
	}
	var errs error
	for _, s := range subs {
		if err := s.SetState(subscription.UnsubscribedState); err != nil {
			errs = common.AppendError(errs, fmt.Errorf("%w: %s", err, s))
		}
		if err := w.subscriptions.Remove(s); err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// GetSubscription returns a subscription at the key provided
// returns nil if no subscription is at that key or the key is nil
// Keys can implement subscription.MatchableKey in order to provide custom matching logic
func (w *Websocket) GetSubscription(key any) *subscription.Subscription {
	if w == nil || w.subscriptions == nil || key == nil {
		return nil
	}
	return w.subscriptions.Get(key)
}

// GetSubscriptions returns a new slice of the subscriptions
func (w *Websocket) GetSubscriptions() subscription.List {
	if w == nil || w.subscriptions == nil {
		return nil
	}
	return w.subscriptions.List()
}

// SetCanUseAuthenticatedEndpoints sets canUseAuthenticatedEndpoints val in a thread safe manner
func (w *Websocket) SetCanUseAuthenticatedEndpoints(b bool) {
	w.canUseAuthenticatedEndpoints.Store(b)
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in a thread safe manner
func (w *Websocket) CanUseAuthenticatedEndpoints() bool {
	return w.canUseAuthenticatedEndpoints.Load()
}

// IsDisconnectionError Determines if the error sent over chan ReadMessageErrors is a disconnection error
func IsDisconnectionError(err error) bool {
	if websocket.IsUnexpectedCloseError(err) {
		return true
	}
	if _, ok := err.(*net.OpError); ok {
		return !errors.Is(err, errClosedConnection)
	}
	return false
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
func (w *Websocket) checkSubscriptions(subs subscription.List) error {
	if w.subscriptions == nil {
		return fmt.Errorf("%w: Websocket.subscriptions", common.ErrNilPointer)
	}

	existing := w.subscriptions.Len()
	if w.MaxSubscriptionsPerConnection > 0 && existing+len(subs) > w.MaxSubscriptionsPerConnection {
		return fmt.Errorf("%w: current subscriptions: %v, incoming subscriptions: %v, max subscriptions per connection: %v - please reduce enabled pairs",
			errSubscriptionsExceedsLimit,
			existing,
			len(subs),
			w.MaxSubscriptionsPerConnection)
	}

	for _, s := range subs {
		if found := w.subscriptions.Get(s); found != nil {
			return fmt.Errorf("%w: %s", subscription.ErrDuplicate, s)
		}
	}

	return nil
}
