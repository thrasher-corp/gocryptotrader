package stream

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const defaultJobBuffer = 1000

var (
	// ErrSubscriptionFailure defines an error when a subscription fails
	ErrSubscriptionFailure = errors.New("subscription failure")
	// ErrNotConnected defines an error when websocket is not connected
	ErrNotConnected = errors.New("websocket is not connected")

	errExchangeConfigIsNil                  = errors.New("exchange config is nil")
	errWebsocketIsNil                       = errors.New("websocket is nil")
	errWebsocketSetupIsNil                  = errors.New("websocket setup is nil")
	errWebsocketFeaturesIsUnset             = errors.New("websocket features is unset")
	errConfigFeaturesIsNil                  = errors.New("exchange config features is nil")
	errDefaultURLIsEmpty                    = errors.New("default url is empty")
	errRunningURLIsEmpty                    = errors.New("running url cannot be empty")
	errInvalidWebsocketURL                  = errors.New("invalid websocket url")
	errExchangeConfigNameUnset              = errors.New("exchange config name unset")
	errInvalidTrafficTimeout                = errors.New("invalid traffic timeout")
	errWebsocketSubscriberUnset             = errors.New("websocket subscriber function needs to be set")
	errWebsocketUnsubscriberUnset           = errors.New("websocket unsubscriber functionality allowed but unsubscriber function not set")
	errWebsocketConnectorUnset              = errors.New("websocket connector function not set")
	errWebsocketSubscriptionsGeneratorUnset = errors.New("websocket subscriptions generator function needs to be set")
	errClosedConnection                     = errors.New("use of closed network connection")
)

var globalReporter Reporter

// SetupGlobalReporter sets a reporter interface to be used
// for all exchange requests
func SetupGlobalReporter(r Reporter) { globalReporter = r }

// New initialises the websocket struct
func New() *Websocket {
	return &Websocket{
		DataHandler:             make(chan interface{}),
		WebsocketRoutineManager: make(chan interface{}, defaultJobBuffer),
		ReadMessageErrors:       make(chan error),
		Subscribe:               make(chan []ChannelSubscription),
		Unsubscribe:             make(chan []ChannelSubscription),
		Match:                   NewMatch(),
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

	if s.ExchangeConfig == nil {
		return errExchangeConfigIsNil
	}

	if s.ExchangeConfig.Name == "" {
		return errExchangeConfigNameUnset
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

	err := w.ConnectionStatus.setEnabled(s.ExchangeConfig.Features.Enabled.Websocket)
	if err != nil && !errors.Is(err, ErrProtocolAlreadyDisabled) {
		return fmt.Errorf("%s %w", w.exchangeName, err)
	}

	if s.Connector == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketConnectorUnset)
	}
	w.ConnectionStatus.Connector = s.Connector

	if s.Subscriber == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketSubscriberUnset)
	}
	w.Subscriber = s.Subscriber

	if w.features.Unsubscribe && s.Unsubscriber == nil {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketUnsubscriberUnset)
	}
	w.connectionMonitorDelay = s.ConnectionMonitorDelay
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
	err = w.SetWebsocketURL(s.RunningURL, false, false)
	if err != nil {
		return fmt.Errorf("%s %w", w.exchangeName, err)
	}

	if s.RunningURLAuth != "" {
		err = w.SetWebsocketURL(s.RunningURLAuth, true, false)
		if err != nil {
			return fmt.Errorf("%s %w", w.exchangeName, err)
		}
	}

	if s.ExchangeConfig.WebsocketTrafficTimeout < time.Second { // RM
		return fmt.Errorf("%s %w cannot be less than %s",
			w.exchangeName,
			errInvalidTrafficTimeout,
			time.Second)
	}
	// w.trafficTimeout = s.ExchangeConfig.WebsocketTrafficTimeout

	w.ShutdownC = make(chan struct{})
	w.Wg = new(sync.WaitGroup)
	w.SetCanUseAuthenticatedEndpoints(s.ExchangeConfig.API.AuthenticatedWebsocketSupport)

	if err := w.Orderbook.Setup(s.ExchangeConfig, &s.OrderbookBufferConfig, w.DataHandler); err != nil {
		return err
	}

	w.Trade.Setup(w.exchangeName, s.TradeFeed, w.DataHandler)
	w.Fills.Setup(s.FillsFeed, w.DataHandler)
	return nil
}

// SetupNewConnection sets up an auth or unauth streaming connection
func (w *Websocket) SetupNewConnection(c ConnectionSetup) error {
	if w == nil {
		return errors.New("setting up new connection error: websocket is nil")
	}
	if c == (ConnectionSetup{}) {
		return errors.New("setting up new connection error: websocket connection configuration empty")
	}

	if w.exchangeName == "" {
		return errors.New("setting up new connection error: exchange name not set, please call setup first")
	}

	if w.ReadMessageErrors == nil {
		return errors.New("setting up new connection error: read message errors is nil, please call setup first")
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
		ExchangeName:      w.exchangeName,
		URL:               connectionURL,
		ProxyURL:          w.GetProxyAddress(),
		Verbose:           w.verbose,
		ResponseMaxLimit:  c.ResponseMaxLimit,
		readMessageErrors: w.ReadMessageErrors,
		ShutdownC:         w.ShutdownC,
		Wg:                w.Wg,
		Match:             w.Match,
		RateLimit:         c.RateLimit,
		Reporter:          c.ConnectionLevelReporter,
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
	w.m.Lock()
	defer w.m.Unlock()

	err := w.ConnectionStatus.Connect()
	if err != nil {
		return err
	}

	w.Wg.Add(2)
	go w.connectionMonitor()
	go w.processingMonitor()

	subs, err := w.GenerateSubs() // regenerate state on new connection
	if err != nil {
		return fmt.Errorf("%v %w: %v", w.exchangeName, ErrSubscriptionFailure, err)
	}
	err = w.Subscriber(subs)
	if err != nil {
		return fmt.Errorf("%v %w: %v", w.exchangeName, ErrSubscriptionFailure, err)
	}
	return nil
}

// TODO: CHECK WHERE THIS IS BEING USED?
// Disable disables the exchange websocket protocol
func (w *Websocket) Disable() error {
	// TODO: Shutdown
	return w.ConnectionStatus.setEnabled(false)
}

// TODO: CHECK WHERE THIS IS BEING USED?
// Enable enables the exchange websocket protocol
func (w *Websocket) Enable() error {
	err := w.ConnectionStatus.setEnabled(true)
	if err != nil {
		return err
	}
	return w.Connect() // ATTEMPT TO CONNECT?
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *Websocket) Shutdown() error {
	w.m.Lock()
	defer w.m.Unlock()

	if w.features.Unsubscribe {
		// Unsubscribe from all channels if available. This exposes potential
		// implementation issues.
		subscriptions, err := w.GenerateSubs()
		if err != nil {
			return err
		}
		err = w.UnsubscribeChannels(subscriptions)
		if err != nil {
			return err
		}
	}

	// flush any subscriptions from last connection if needed
	w.subscriptionMutex.Lock()
	w.subscriptions = nil
	w.subscriptionMutex.Unlock()

	if w.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: shutting down websocket\n", w.exchangeName)
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

	close(w.ShutdownC)
	w.Wg.Wait()
	w.ShutdownC = make(chan struct{})

	err := w.ConnectionStatus.Shutdown()
	if err != nil {
		return err
	}

	if w.verbose {
		log.Debugf(log.WebsocketMgr, "%v websocket: completed websocket shutdown\n",
			w.exchangeName)
	}
	return nil
}

// FlushChannels flushes channel subscriptions when there is a pair/asset change
func (w *Websocket) FlushChannels() error {
	if !w.ConnectionStatus.IsEnabled() {
		return fmt.Errorf("%s websocket: service not enabled", w.exchangeName)
	}

	if !w.ConnectionStatus.IsConnected() {
		return fmt.Errorf("%s websocket: service not connected", w.exchangeName)
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
			w.subscriptionMutex.Lock()
			w.subscriptions = nil
			w.subscriptionMutex.Unlock()
			return w.SubscribeToChannels(newsubs)
		}
		return nil
	}

	if err := w.Shutdown(); err != nil {
		return err
	}
	return w.Connect()
}

// CanUseAuthenticatedWebsocketForWrapper Handles a common check to
// verify whether a wrapper can use an authenticated websocket endpoint
func (w *Websocket) CanUseAuthenticatedWebsocketForWrapper() bool {
	connected := w.ConnectionStatus.IsConnected()
	if connected && w.CanUseAuthenticatedEndpoints() {
		return true
	} else if connected && !w.CanUseAuthenticatedEndpoints() {
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

	if w.ConnectionStatus.IsConnected() && reconnect {
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
	if proxyAddr != "" {
		_, err := url.ParseRequestURI(proxyAddr)
		if err != nil {
			return fmt.Errorf("%v websocket: cannot set proxy address error '%v'",
				w.exchangeName,
				err)
		}

		if w.proxyAddr == proxyAddr {
			return fmt.Errorf("%v websocket: cannot set proxy address to the same address '%v'",
				w.exchangeName,
				w.proxyAddr)
		}

		log.Debugf(log.ExchangeSys,
			"%s websocket: setting websocket proxy: %s\n",
			w.exchangeName,
			proxyAddr)
	} else {
		log.Debugf(log.ExchangeSys,
			"%s websocket: removing websocket proxy\n",
			w.exchangeName)
	}

	if w.Conn != nil {
		w.Conn.SetProxy(proxyAddr)
	}
	if w.AuthConn != nil {
		w.AuthConn.SetProxy(proxyAddr)
	}

	w.proxyAddr = proxyAddr
	if w.ConnectionStatus.IsEnabled() { // INIT TAKEN OFF.
		if w.ConnectionStatus.IsConnected() {
			err := w.Shutdown()
			if err != nil {
				return err
			}
		}
		return w.Connect()
	}
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
func (w *Websocket) GetChannelDifference(genSubs []ChannelSubscription) (sub, unsub []ChannelSubscription) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()

oldsubs:
	for x := range w.subscriptions {
		for y := range genSubs {
			if w.subscriptions[x].Equal(&genSubs[y]) {
				continue oldsubs
			}
		}
		unsub = append(unsub, w.subscriptions[x])
	}

newsubs:
	for x := range genSubs {
		for y := range w.subscriptions {
			if genSubs[x].Equal(&w.subscriptions[y]) {
				continue newsubs
			}
		}
		sub = append(sub, genSubs[x])
	}
	return
}

// UnsubscribeChannels unsubscribes from a websocket channel
func (w *Websocket) UnsubscribeChannels(channels []ChannelSubscription) error {
	if len(channels) == 0 {
		return fmt.Errorf("%s websocket: channels not populated cannot remove",
			w.exchangeName)
	}
	w.subscriptionMutex.Lock()

channels:
	for x := range channels {
		for y := range w.subscriptions {
			if channels[x].Equal(&w.subscriptions[y]) {
				continue channels
			}
		}
		w.subscriptionMutex.Unlock()
		return fmt.Errorf("%s websocket: subscription not found in list: %+v",
			w.exchangeName,
			channels[x])
	}
	w.subscriptionMutex.Unlock()
	return w.Unsubscriber(channels)
}

// ResubscribeToChannel resubscribes to channel
func (w *Websocket) ResubscribeToChannel(subscribedChannel *ChannelSubscription) error {
	err := w.UnsubscribeChannels([]ChannelSubscription{*subscribedChannel})
	if err != nil {
		return err
	}
	return w.SubscribeToChannels([]ChannelSubscription{*subscribedChannel})
}

// SubscribeToChannels appends supplied channels to channelsToSubscribe
func (w *Websocket) SubscribeToChannels(channels []ChannelSubscription) error {
	if len(channels) == 0 {
		return fmt.Errorf("%s websocket: cannot subscribe no channels supplied",
			w.exchangeName)
	}
	w.subscriptionMutex.Lock()
	for x := range channels {
		for y := range w.subscriptions {
			if channels[x].Equal(&w.subscriptions[y]) {
				w.subscriptionMutex.Unlock()
				return fmt.Errorf("%s websocket: %v already subscribed",
					w.exchangeName,
					channels[x])
			}
		}
	}
	w.subscriptionMutex.Unlock()
	if err := w.Subscriber(channels); err != nil {
		return fmt.Errorf("%v %w: %v", w.exchangeName, ErrSubscriptionFailure, err)
	}
	return nil
}

// AddSuccessfulSubscriptions adds subscriptions to the subscription lists that
// has been successfully subscribed
func (w *Websocket) AddSuccessfulSubscriptions(channels ...ChannelSubscription) {
	w.subscriptionMutex.Lock()
	w.subscriptions = append(w.subscriptions, channels...)
	w.subscriptionMutex.Unlock()
}

// RemoveSuccessfulUnsubscriptions removes subscriptions from the subscription
// list that has been successfulling unsubscribed
func (w *Websocket) RemoveSuccessfulUnsubscriptions(channels ...ChannelSubscription) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	for x := range channels {
		for y := range w.subscriptions {
			if channels[x].Equal(&w.subscriptions[y]) {
				w.subscriptions[y] = w.subscriptions[len(w.subscriptions)-1]
				w.subscriptions[len(w.subscriptions)-1] = ChannelSubscription{}
				w.subscriptions = w.subscriptions[:len(w.subscriptions)-1]
				break
			}
		}
	}
}

// Equal two WebsocketChannelSubscription to determine equality
func (w *ChannelSubscription) Equal(s *ChannelSubscription) bool {
	return strings.EqualFold(w.Channel, s.Channel) &&
		w.Currency.Equal(s.Currency)
}

// GetSubscriptions returns a copied list of subscriptions
// and is a private member that cannot be manipulated
func (w *Websocket) GetSubscriptions() []ChannelSubscription {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	return append(w.subscriptions[:0:0], w.subscriptions...)
}

// SetCanUseAuthenticatedEndpoints sets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *Websocket) SetCanUseAuthenticatedEndpoints(val bool) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	w.canUseAuthenticatedEndpoints = val
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *Websocket) CanUseAuthenticatedEndpoints() bool {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	return w.canUseAuthenticatedEndpoints
}

// isDisconnectionError Determines if the error sent over chan ReadMessageErrors is a disconnection error
func isDisconnectionError(err error) bool {
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

// connectionMonitor ensures that stream protocol connection is maintained
func (w *Websocket) connectionMonitor() {
	defer w.Wg.Done()
	timer := time.NewTimer(0) // Fire immediately
	for {
		select {
		case <-w.ShutdownC:
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v websocket: connection monitor exiting\n", w.exchangeName)
			}
			if !timer.Stop() {
				<-timer.C
			}
			return
		case err := <-w.ReadMessageErrors:
			if !isDisconnectionError(err) {
				// pass off non disconnect errors to datahandler to manage
				w.DataHandler <- err
				break
			}
			log.Warnf(log.WebsocketMgr, "%v websocket has been disconnected. Reason: %v", w.exchangeName, err)
			// TODO: Differentiate between connections that need to be cycled
			// as this will shutdown both connections and then reconnect both.
			if err = w.Shutdown(); err != nil {
				log.Error(log.WebsocketMgr, err)
			}
		case <-timer.C:
			if !w.ConnectionStatus.IsConnected() {
				err := w.Connect()
				if err != nil {
					log.Error(log.WebsocketMgr, err)
				}
			}
			timer.Reset(w.connectionMonitorDelay)
		}
	}
}

// processingMonitor is responsible for monitoring the job throughput of the
// protocol stream and logs a message if there is a backlog of data that is not
// being processed in a timely manner. This warns that the system is not
// performing at an optimal level usually as a result of improper handling of
// data in the websocket routine manager. See engine/websocketroutine_manager.go.
func (w *Websocket) processingMonitor() {
	defer w.Wg.Done()
	for {
		select {
		case <-w.ShutdownC:
			for range w.DataHandler {
				// Bleeds data from the websocket connection if needed
				continue
			}
			return
		case dataFromWebsocketConnection := <-w.DataHandler:
			select {
			case w.WebsocketRoutineManager <- dataFromWebsocketConnection:
			default:
				log.Warnf(log.WebsocketMgr, "%s exchange backlog in websocket processing detected", w.exchangeName)
				w.WebsocketRoutineManager <- dataFromWebsocketConnection
			}
		}
	}
}
