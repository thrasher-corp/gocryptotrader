package stream

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	defaultJobBuffer = 1000
	// defaultTrafficPeriod defines a period of pause for the traffic monitor,
	// as there are periods with large incoming traffic alerts which requires a
	// timer reset, this limits work on this routine to a more effective rate
	// of check.
	defaultTrafficPeriod = time.Second
)

// New initialises the websocket struct
func New() *Websocket {
	return &Websocket{
		init:              true,
		DataHandler:       make(chan interface{}),
		ToRoutine:         make(chan interface{}, defaultJobBuffer),
		TrafficAlert:      make(chan struct{}),
		readMessageErrors: make(chan error),
		subscribe:         make(chan []ChannelSubscription),
		unsubscribe:       make(chan []ChannelSubscription),
		Match:             NewMatch(),
	}
}

// NewTestWebsocket returns a test websocket object
func NewTestWebsocket() *Websocket {
	return &Websocket{
		init:              true,
		DataHandler:       make(chan interface{}, 75),
		ToRoutine:         make(chan interface{}, defaultJobBuffer),
		TrafficAlert:      make(chan struct{}),
		readMessageErrors: make(chan error),
		subscribe:         make(chan []ChannelSubscription, 10),
		unsubscribe:       make(chan []ChannelSubscription, 10),
		Match:             NewMatch(),
	}
}

// Setup sets main variables for websocket connection
func (w *Websocket) Setup(setupData *WebsocketSetup) error {
	if w == nil {
		return errors.New("websocket is nil")
	}

	if !w.init {
		return fmt.Errorf("%s Websocket already initialised",
			setupData.ExchangeName)
	}

	w.verbose = setupData.Verbose

	if setupData.Features == nil {
		return errors.New("websocket features is unset")
	}

	w.features = setupData.Features

	if w.features.Subscribe && setupData.Subscriber == nil {
		return errors.New("features have been set yet channel subscriber is not set")
	}
	w.Subscriber = setupData.Subscriber

	if w.features.Unsubscribe && setupData.UnSubscriber == nil {
		return errors.New("features have been set yet channel unsubscriber is not set")
	}
	w.Unsubscriber = setupData.UnSubscriber

	w.GenerateSubs = setupData.GenerateSubscriptions

	w.enabled = setupData.Enabled
	if setupData.DefaultURL == "" {
		return errors.New("default url is empty")
	}
	w.defaultURL = setupData.DefaultURL
	w.connector = setupData.Connector
	if setupData.ExchangeName == "" {
		return errors.New("exchange name unset")
	}
	w.exchangeName = setupData.ExchangeName

	if setupData.WebsocketTimeout < time.Second {
		return fmt.Errorf("traffic timeout cannot be less than %s", time.Second)
	}

	w.trafficTimeout = setupData.WebsocketTimeout
	if setupData.Features == nil {
		return errors.New("feature set is nil")
	}

	if setupData.RunningURL == "" {
		return errors.New("running URL cannot be nil")
	}
	err := w.SetWebsocketURL(setupData.RunningURL, false)
	if err != nil {
		return err
	}

	w.ShutdownC = make(chan struct{})

	w.SetCanUseAuthenticatedEndpoints(setupData.AuthenticatedWebsocketAPISupport)
	err = w.Initialise()
	if err != nil {
		return err
	}

	w.Orderbook.Setup(setupData.OrderbookBufferLimit,
		setupData.BufferEnabled,
		setupData.SortBuffer,
		setupData.SortBufferByUpdateIDs,
		setupData.UpdateEntriesByID,
		w.exchangeName,
		w.DataHandler)
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

	if w.TrafficAlert == nil {
		return errors.New("setting up new connection error: traffic alert is nil, please call setup first")
	}

	if w.readMessageErrors == nil {
		return errors.New("setting up new connection error: read message errors is nil, please call setup first")
	}

	connectionURL := w.GetWebsocketURL()
	if c.URL != "" {
		connectionURL = c.URL
	}

	newConn := &WebsocketConnection{
		ExchangeName:      w.exchangeName,
		URL:               connectionURL,
		ProxyURL:          w.GetProxyAddress(),
		Verbose:           w.verbose,
		ResponseMaxLimit:  c.ResponseMaxLimit,
		Traffic:           w.TrafficAlert,
		readMessageErrors: w.readMessageErrors,
		ShutdownC:         make(chan struct{}),
		Wg:                &w.Wg,
		Match:             w.Match,
	}

	if c.Authenticated {
		w.AuthConn = newConn
	} else {
		w.Conn = newConn
	}

	return nil
}

// SetupNewCustomConnection sets up an auth or unauth custom streaming
// connection
func (w *Websocket) SetupNewCustomConnection(c Connection, auth bool) error {
	if c == nil {
		return errors.New("connection is nil")
	}

	if auth {
		w.AuthConn = c
	} else {
		w.Conn = c
	}

	return nil
}

// Connect initiates a websocket connection by using a package defined connection
// function
func (w *Websocket) Connect() error {
	if w.connector == nil {
		return errors.New("websocket connect function not set, cannot continue")
	}
	w.m.Lock()
	defer w.m.Unlock()

	if !w.IsEnabled() {
		return errors.New(WebsocketNotEnabled)
	}
	if w.IsConnecting() {
		return fmt.Errorf("%v Websocket already attempting to connect",
			w.exchangeName)
	}
	if w.IsConnected() {
		return fmt.Errorf("%v Websocket already connected",
			w.exchangeName)
	}
	w.setConnectingStatus(true)

	go w.dataMonitor()

	err := w.trafficMonitor()
	if err != nil {
		return err
	}

	// flush any subscriptions from last connection if needed
	w.subscriptions = nil

	err = w.connector()
	if err != nil {
		w.setConnectingStatus(false)
		return fmt.Errorf("%v Error connecting %s",
			w.exchangeName, err)
	}

	w.setConnectedStatus(true)
	w.setConnectingStatus(false)
	w.setInit(true)

	if !w.IsConnectionMonitorRunning() {
		go w.connectionMonitor()
	}

	return nil
}

// Disable disables the exchange websocket protocol
func (w *Websocket) Disable() error {
	if !w.IsConnected() {
		return fmt.Errorf("websocket is already disabled for exchange %s",
			w.exchangeName)
	}

	w.setEnabled(false)
	return nil
}

// Enable enables the exchange websocket protocol
func (w *Websocket) Enable() error {
	if w.IsConnected() {
		return fmt.Errorf("websocket is already enabled for exchange %s",
			w.exchangeName)
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
	defer func() {
		for {
			// Bleeds data from the websocket connection if needed
			select {
			case <-w.DataHandler:
			default:
				w.setDataMonitorRunning(false)
				w.Wg.Done()
				return
			}
		}
	}()
	for {
		select {
		case <-w.ShutdownC:
			return
		case d := <-w.DataHandler:
			select {
			case w.ToRoutine <- d:
			case <-w.ShutdownC:
				return
			default:
				log.Errorf(log.WebsocketMgr,
					"%s exchange backlog in websocket processing detected",
					w.exchangeName)
				select {
				case w.ToRoutine <- d:
				case <-w.ShutdownC:
					return
				}
			}
		}
	}
}

// connectionMonitor ensures that the WS keeps connecting
func (w *Websocket) connectionMonitor() {
	if w.IsConnectionMonitorRunning() {
		return
	}
	w.setConnectionMonitorRunning(true)
	timer := time.NewTimer(connectionMonitorDelay)

	for {
		if w.verbose {
			log.Debugf(log.WebsocketMgr,
				"%v websocket: running connection monitor cycle\n",
				w.exchangeName)
		}
		if !w.IsEnabled() {
			if w.verbose {
				log.Debugf(log.WebsocketMgr,
					"%v websocket: connectionMonitor - websocket disabled, shutting down\n",
					w.exchangeName)
			}
			if w.IsConnected() {
				err := w.Shutdown()
				if err != nil {
					log.Error(log.WebsocketMgr, err)
				}
			}
			if w.verbose {
				log.Debugf(log.WebsocketMgr,
					"%v websocket: connection monitor exiting\n",
					w.exchangeName)
			}
			timer.Stop()
			w.setConnectionMonitorRunning(false)
			return
		}
		select {
		case err := <-w.readMessageErrors:
			// check if this error is a disconnection error
			if isDisconnectionError(err) {
				w.setInit(false)
				if w.verbose {
					log.Debugf(log.WebsocketMgr,
						"%v websocket has been disconnected. Reason: %v",
						w.exchangeName, err)
				}
				w.setConnectedStatus(false)
			} else {
				// pass off non disconnect errors to datahandler to manage
				w.DataHandler <- err
			}
		case <-timer.C:
			if !w.IsConnecting() && !w.IsConnected() {
				err := w.Connect()
				if err != nil {
					log.Error(log.WebsocketMgr, err)
				}
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(connectionMonitorDelay)
		}
	}
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *Websocket) Shutdown() error {
	w.m.Lock()
	defer w.m.Unlock()

	if !w.IsConnected() {
		return fmt.Errorf("%v websocket: cannot shutdown a disconnected websocket",
			w.exchangeName)
	}

	if w.IsConnecting() {
		return fmt.Errorf("%v websocket: cannot shutdown, in the process of reconnection",
			w.exchangeName)
	}

	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v websocket: shutting down websocket\n",
			w.exchangeName)
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
	w.setConnectedStatus(false)
	w.setConnectingStatus(false)
	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v websocket: completed websocket shutdown\n",
			w.exchangeName)
	}
	return nil
}

// FlushChannels flushes channel subscriptions when there is a pair/asset change
func (w *Websocket) FlushChannels() error {
	if !w.IsEnabled() {
		return fmt.Errorf("%s websocket: service not enabled", w.exchangeName)
	}

	if !w.IsConnected() {
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

			if len(subs) != 0 {
				return w.SubscribeToChannels(subs)
			}

			return nil
		} else if len(unsubs) == 0 {
			if len(subs) == 0 {
				return nil
			}
			return w.SubscribeToChannels(subs)
		}
	} else if w.features.FullPayloadSubscribe {
		newsubs, err := w.GenerateSubs()
		if err != nil {
			return err
		}

		if len(newsubs) != 0 {
			return w.SubscribeToChannels(newsubs)
		}
		return nil
	}

	err := w.Shutdown()
	if err != nil {
		return err
	}
	return w.Connect()
}

// trafficMonitor uses a timer of WebsocketTrafficLimitTime and once it expires,
// it will reconnect if the TrafficAlert channel has not received any data. The
// trafficTimer will reset on each traffic alert
func (w *Websocket) trafficMonitor() error {
	if w.IsTrafficMonitorRunning() {
		return nil
	}

	w.Wg.Add(1)
	w.setTrafficMonitorRunning(true)

	go func() {
		var trafficTimer = time.NewTimer(w.trafficTimeout)
		pause := make(chan struct{})
		for {
			select {
			case <-w.ShutdownC:
				if w.verbose {
					log.Debugf(log.WebsocketMgr,
						"%v websocket: trafficMonitor shutdown message received\n",
						w.exchangeName)
				}
				trafficTimer.Stop()
				w.setTrafficMonitorRunning(false)
				w.Wg.Done()
				return
			case <-w.TrafficAlert:
				if !trafficTimer.Stop() {
					select {
					case <-trafficTimer.C:
					default:
					}
				}
				w.setConnectedStatus(true)
				trafficTimer.Reset(w.trafficTimeout)
			case <-trafficTimer.C: // Falls through when timer runs out
				if w.verbose {
					log.Warnf(log.WebsocketMgr,
						"%v websocket: has not received a traffic alert in %v. Reconnecting",
						w.exchangeName,
						w.trafficTimeout)
				}
				trafficTimer.Stop()
				w.Wg.Done()
				err := w.Shutdown()
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%v websocket: trafficMonitor shutdown err: %s",
						w.exchangeName, err)
				}
				w.setTrafficMonitorRunning(false)
				return
			}

			// Routine pausing mechanism
			go func(p chan struct{}) {
				time.Sleep(defaultTrafficPeriod)
				p <- struct{}{}
			}(pause)
			select {
			case <-w.ShutdownC:
				trafficTimer.Stop()
				w.setTrafficMonitorRunning(false)
				w.Wg.Done()
				return
			case <-pause:
			}
		}
	}()
	return nil
}

func (w *Websocket) setConnectedStatus(b bool) {
	w.connectionMutex.Lock()
	w.connected = b
	w.connectionMutex.Unlock()
}

// IsConnected returns status of connection
func (w *Websocket) IsConnected() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.connected
}

func (w *Websocket) setConnectingStatus(b bool) {
	w.connectionMutex.Lock()
	w.connecting = b
	w.connectionMutex.Unlock()
}

// IsConnecting returns status of connecting
func (w *Websocket) IsConnecting() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.connecting
}

func (w *Websocket) setEnabled(b bool) {
	w.connectionMutex.Lock()
	w.enabled = b
	w.connectionMutex.Unlock()
}

// IsEnabled returns status of enabled
func (w *Websocket) IsEnabled() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.enabled
}

func (w *Websocket) setInit(b bool) {
	w.connectionMutex.Lock()
	w.init = b
	w.connectionMutex.Unlock()
}

// IsInit returns status of init
func (w *Websocket) IsInit() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.init
}

func (w *Websocket) setTrafficMonitorRunning(b bool) {
	w.connectionMutex.Lock()
	w.trafficMonitorRunning = b
	w.connectionMutex.Unlock()
}

// IsTrafficMonitorRunning returns status of the traffic monitor
func (w *Websocket) IsTrafficMonitorRunning() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.trafficMonitorRunning
}

func (w *Websocket) setConnectionMonitorRunning(b bool) {
	w.connectionMutex.Lock()
	w.connectionMonitorRunning = b
	w.connectionMutex.Unlock()
}

// IsConnectionMonitorRunning returns status of connection monitor
func (w *Websocket) IsConnectionMonitorRunning() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.connectionMonitorRunning
}

func (w *Websocket) setDataMonitorRunning(b bool) {
	w.connectionMutex.Lock()
	w.dataMonitorRunning = b
	w.connectionMutex.Unlock()
}

// IsDataMonitorRunning returns status of data monitor
func (w *Websocket) IsDataMonitorRunning() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.dataMonitorRunning
}

// CanUseAuthenticatedWebsocketForWrapper Handles a common check to
// verify whether a wrapper can use an authenticated websocket endpoint
func (w *Websocket) CanUseAuthenticatedWebsocketForWrapper() bool {
	if w.IsConnected() && w.CanUseAuthenticatedEndpoints() {
		return true
	} else if w.IsConnected() && !w.CanUseAuthenticatedEndpoints() {
		log.Infof(log.WebsocketMgr,
			WebsocketNotAuthenticatedUsingRest,
			w.exchangeName)
	}
	return false
}

// SetWebsocketURL sets websocket URL and can refresh underlying connections
func (w *Websocket) SetWebsocketURL(websocketURL string, reconnect bool) error {
	if websocketURL == "" || websocketURL == config.WebsocketURLNonDefaultMessage {
		w.runningURL = w.defaultURL
	} else {
		w.runningURL = websocketURL
	}

	if w.verbose {
		log.Debugf(log.ExchangeSys,
			"%s websocket: setting websocket URL: %s\n",
			w.exchangeName,
			websocketURL)
	}

	if w.Conn != nil {
		w.Conn.SetURL(w.runningURL)
	}
	if w.AuthConn != nil {
		w.AuthConn.SetURL(w.runningURL)
	}

	if w.IsConnected() && reconnect {
		log.Debugf(log.ExchangeSys,
			"%s websocket: flushing websocket connection to %s\n",
			w.exchangeName,
			websocketURL)
		return w.Shutdown()
	}
	return nil
}

// GetWebsocketURL returns the running websocket URL
func (w *Websocket) GetWebsocketURL() string {
	return w.runningURL
}

// Initialise verifies status and connects
func (w *Websocket) Initialise() error {
	if w.IsEnabled() {
		if w.IsInit() {
			return nil
		}
		return fmt.Errorf("%v websocket: already initialised", w.exchangeName)
	}
	w.setEnabled(w.enabled)
	return nil
}

// SetProxyAddress sets websocket proxy address
func (w *Websocket) SetProxyAddress(proxyAddr string) error {
	if w.proxyAddr == proxyAddr {
		return fmt.Errorf("%v websocket: cannot set proxy address to the same address '%v'",
			w.exchangeName,
			w.proxyAddr)
	}

	log.Debugf(log.ExchangeSys,
		"%s websocket: setting websocket proxy: %s\n",
		w.exchangeName,
		proxyAddr)

	if w.Conn != nil {
		w.Conn.SetProxy(proxyAddr)
	}
	if w.AuthConn != nil {
		w.AuthConn.SetProxy(proxyAddr)
	}

	w.proxyAddr = proxyAddr
	if w.IsInit() && w.IsEnabled() {
		if w.IsConnected() {
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
	defer w.subscriptionMutex.Unlock()

channels:
	for x := range channels {
		for y := range w.subscriptions {
			if channels[x].Equal(&w.subscriptions[y]) {
				continue channels
			}
		}
		return fmt.Errorf("%s websocket: subscription not found in list: %+v",
			w.exchangeName,
			channels[x])
	}

	err := w.Unsubscriber(channels)
	if err != nil {
		return err
	}

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
	return nil
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
	defer w.subscriptionMutex.Unlock()
	for x := range channels {
		for y := range w.subscriptions {
			if channels[x].Equal(&w.subscriptions[y]) {
				return fmt.Errorf("%s websocket: %v already subscribed",
					w.exchangeName,
					channels[x])
			}
		}
	}

	err := w.Subscriber(channels)
	if err != nil {
		return err
	}
	w.subscriptions = append(w.subscriptions, channels...)
	return nil
}

// Equal two WebsocketChannelSubscription to determine equality
func (w *ChannelSubscription) Equal(s *ChannelSubscription) bool {
	return strings.EqualFold(w.Channel, s.Channel) &&
		w.Currency.Equal(s.Currency)
}

// GetSubscriptions returns a copied list of subscriptions
// subscriptions is a private member and cannot be manipulated
func (w *Websocket) GetSubscriptions() []ChannelSubscription {
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
	switch e := err.(type) {
	case *websocket.CloseError:
		return true
	case *net.OpError:
		if e.Err.Error() == "use of closed network connection" {
			return false
		}
		return true
	}
	return false
}
