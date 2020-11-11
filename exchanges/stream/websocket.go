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

const (
	defaultJobBuffer = 1000
	// defaultTrafficPeriod defines a period of pause for the traffic monitor,
	// as there are periods with large incoming traffic alerts which requires a
	// timer reset, this limits work on this routine to a more effective rate
	// of check.
	defaultTrafficPeriod = time.Second
)

var errClosedConnection = errors.New("use of closed network connection")

// New initialises the websocket struct
func New() *Websocket {
	return &Websocket{
		Init:              true,
		DataHandler:       make(chan interface{}),
		ToRoutine:         make(chan interface{}, defaultJobBuffer),
		TrafficAlert:      make(chan struct{}),
		ReadMessageErrors: make(chan error),
		Match:             NewMatch(),
	}
}

// Setup sets main variables for websocket connection
func (w *Websocket) Setup(s *WebsocketSetup) error {
	if w == nil {
		return errors.New("websocket is nil")
	}

	if s == nil {
		return errors.New("websocket setup is nil")
	}

	if !w.Init {
		return fmt.Errorf("%s Websocket already initialised",
			s.ExchangeName)
	}

	w.verbose = s.Verbose
	if s.Features == nil {
		return errors.New("websocket features is unset")
	}

	w.features = s.Features

	enabled, err := w.features.Functionality()
	if err != nil {
		return err
	}

	if enabled.Subscribe && s.Subscriber == nil {
		return errors.New("features have been set yet channel subscriber is not set")
	}

	if enabled.Unsubscribe && s.Unsubscriber == nil {
		return errors.New("features have been set yet channel unsubscriber is not set")
	}

	w.enabled = s.Enabled
	if s.DefaultURL == "" {
		return errors.New("default url is empty")
	}
	w.defaultURL = s.DefaultURL
	if s.RunningURL == "" {
		return errors.New("running URL cannot be nil")
	}
	err = w.SetWebsocketURL(s.RunningURL, false, false)
	if err != nil {
		return err
	}

	if s.RunningURLAuth != "" {
		err = w.SetWebsocketURL(s.RunningURLAuth, true, false)
		if err != nil {
			return err
		}
	}

	if s.ExchangeName == "" {
		return errors.New("exchange name unset")
	}
	w.exchangeName = s.ExchangeName

	if s.ConnectionTimeout < time.Second {
		return fmt.Errorf("traffic timeout cannot be less than %s", time.Second)
	}
	w.trafficTimeout = s.ConnectionTimeout

	if s.GenerateConnection == nil {
		return errors.New("connection generation is not set")
	}

	w.ShutdownC = make(chan struct{})
	w.Wg = new(sync.WaitGroup)
	w.SetCanUseAuthenticatedEndpoints(s.AuthenticatedWebsocketAPISupport)

	w.Connections, err = NewConnectionManager(&ConnectionManagerConfig{
		Wg:                            w.Wg,
		ExchangeConnector:             s.Connector,
		ExchangeGenerateSubscriptions: s.GenerateSubscriptions,
		ExchangeSubscriber:            s.Subscriber,
		ExchangeUnsubscriber:          s.Unsubscriber,
		ExchangeGenerateConnection:    s.GenerateConnection,
		ExchangeReadConnection:        s.HandleStreamData,
		Features:                      w.features,
		Configurations:                s.ConnectionConfigurations,
		dataHandler:                   w.DataHandler,
	})
	if err != nil {
		return err
	}

	w.Orderbook.Setup(s.OrderbookBufferLimit,
		s.BufferEnabled,
		s.SortBuffer,
		s.SortBufferByUpdateIDs,
		s.UpdateEntriesByID,
		w.exchangeName,
		w.DataHandler)
	return nil
}

// // SetupNewConnection sets different types of potential connections
// func (w *Websocket) SetupNewConnection(c ConnectionSetup) error {
// 	if w == nil {
// 		return errors.New("setting up new connection error: websocket is nil")
// 	}
// 	// if c == (ConnectionSetup{}) {
// 	// 	return errors.New("setting up new connection error: websocket connection configuration empty")
// 	// }

// 	if w.exchangeName == "" {
// 		return errors.New("setting up new connection error: exchange name not set, please call setup first")
// 	}

// 	if w.TrafficAlert == nil {
// 		return errors.New("setting up new connection error: traffic alert is nil, please call setup first")
// 	}

// 	if w.ReadMessageErrors == nil {
// 		return errors.New("setting up new connection error: read message errors is nil, please call setup first")
// 	}

// 	// connectionURL := w.GetWebsocketURL()
// 	// if c.URL != "" {
// 	// 	connectionURL = c.URL
// 	// }

// 	return w.Connections.LoadConfiguration(c)

// 	// return w.Connections.LoadNewConnection(&WebsocketConnection{
// 	// 	ExchangeName:      w.exchangeName,
// 	// 	URL:               connectionURL,
// 	// 	ProxyURL:          w.GetProxyAddress(),
// 	// 	Verbose:           w.verbose,
// 	// 	ResponseMaxLimit:  c.ResponseMaxLimit,
// 	// 	Traffic:           w.TrafficAlert,
// 	// 	readMessageErrors: w.ReadMessageErrors,
// 	// 	ShutdownC:         w.ShutdownC,
// 	// 	Wg:                w.Wg,
// 	// 	Match:             w.Match,
// 	// 	RateLimit:         c.RateLimit,
// 	// })

// 	// newConn :=

// 	// if c.Authenticated {
// 	// 	w.AuthConn = newConn
// 	// } else {
// 	// 	w.Conn = newConn
// 	// }

// 	// return nil
// }

// Connect initiates a websocket connection by using a package defined connection
// function
func (w *Websocket) Connect() error {
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

	// Flush any subscriptions from last connection if needed.
	w.Connections.FlushSubscriptions()

	w.dataMonitor()
	w.trafficMonitor()
	w.setConnectingStatus(true)

	err := w.Connections.FullConnect()
	if err != nil {
		w.setConnectingStatus(false)
		return fmt.Errorf("%v error connecting %s", w.exchangeName, err)
	}

	w.setConnectedStatus(true)
	w.setConnectingStatus(false)
	w.setInit(true)

	if !w.IsConnectionMonitorRunning() {
		w.connectionMonitor()
	}
	return nil
}

// Disable disables the exchange websocket protocol
func (w *Websocket) Disable() error {
	if !w.IsConnected() || !w.IsEnabled() {
		return fmt.Errorf("websocket is already disabled for exchange %s",
			w.exchangeName)
	}

	w.setEnabled(false)
	return nil
}

// Enable enables the exchange websocket protocol
func (w *Websocket) Enable() error {
	if w.IsConnected() || w.IsEnabled() {
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

	go func() {
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
					log.Warnf(log.WebsocketMgr,
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
	}()
}

// connectionMonitor ensures that the WS keeps connecting
func (w *Websocket) connectionMonitor() {
	if w.IsConnectionMonitorRunning() {
		return
	}
	w.setConnectionMonitorRunning(true)
	go func() {
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
			case err := <-w.ReadMessageErrors:
				if isDisconnectionError(err) {
					w.setInit(false)
					log.Warnf(log.WebsocketMgr,
						"%v websocket has been disconnected. Reason: %v",
						w.exchangeName, err)
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
	}()
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

	err := w.Connections.Shutdown()
	if err != nil {
		return fmt.Errorf("%v websocket: could not close down connection: %w ",
			w.exchangeName,
			err)
	}

	// flush any subscriptions from last connection if needed
	w.Connections.FlushSubscriptions()

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

	enabled, err := w.features.Functionality()
	if err != nil {
		return fmt.Errorf("%s websocket: cannot flush channels %s", w.exchangeName, err)
	}

	if enabled.Subscribe {
		newsubs, err := w.Connections.GenerateSubscriptions()
		if err != nil {
			return err
		}

		subs, unsubs, err := w.Connections.GetChannelDifference(newsubs)
		if err != nil {
			return fmt.Errorf("%s websocket: error %v", w.exchangeName, err)
		}

		if enabled.Unsubscribe && len(unsubs) != 0 {
			err := w.Connections.Unsubscribe(unsubs)
			if err != nil {
				return err
			}
		}

		if len(subs) != 0 {
			return w.Connections.Subscribe(subs)
		}
		return nil
	} else if enabled.FullPayloadSubscribe {
		// FullPayloadSubscribe means that the endpoint requires all
		// subscriptions to be sent via the websocket connection e.g. if you are
		// subscribed to ticker and orderbook but require trades as well, you
		// would need to send ticker, orderbook and trades channel subscription
		// messages.
		newsubs, err := w.Connections.GenerateSubscriptions()
		if err != nil {
			return err
		}

		if len(newsubs) != 0 {
			// Purge subscription list as there will be conflicts.
			w.Connections.FlushSubscriptions()
			return w.Connections.AssociateAndSubscribe(newsubs)
		}
		return nil
	}

	err = w.Shutdown()
	if err != nil {
		return err
	}
	return w.Connect()
}

// trafficMonitor uses a timer of WebsocketTrafficLimitTime and once it expires,
// it will reconnect if the TrafficAlert channel has not received any data. The
// trafficTimer will reset on each traffic alert
func (w *Websocket) trafficMonitor() {
	if w.IsTrafficMonitorRunning() {
		return
	}
	w.setTrafficMonitorRunning(true)
	w.Wg.Add(1)

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
				if !w.IsConnecting() && w.IsConnected() {
					err := w.Shutdown()
					if err != nil {
						log.Errorf(log.WebsocketMgr,
							"%v websocket: trafficMonitor shutdown err: %s",
							w.exchangeName, err)
					}
				}
				w.setTrafficMonitorRunning(false)
				return
			}

			if w.IsConnected() {
				// Routine pausing mechanism
				go func(p chan<- struct{}) {
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
		}
	}()
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
	w.Init = b
	w.connectionMutex.Unlock()
}

// IsInit returns status of init
func (w *Websocket) IsInit() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.Init
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

// ResubscribeToChannel resubscribes to channel
func (w *Websocket) ResubscribeToChannel(subscribedChannel *ChannelSubscription) error {
	err := w.Connections.AssociateAndUnsubscribe([]ChannelSubscription{*subscribedChannel})
	if err != nil {
		return err
	}
	return w.Connections.AssociateAndSubscribe([]ChannelSubscription{*subscribedChannel})
}

// Equal two WebsocketChannelSubscription to determine equality
func (w *ChannelSubscription) Equal(s *ChannelSubscription) bool {
	return strings.EqualFold(w.Channel, s.Channel) &&
		w.Currency.Equal(s.Currency) &&
		w.Asset == w.Asset &&
		w.SubscriptionType == s.SubscriptionType
}

// GetSubscriptions returns a copied list of subscriptions
// subscriptions is a private member and cannot be manipulated
func (w *Websocket) GetSubscriptions() []ChannelSubscription {
	// w.subscriptionMutex.Lock()
	// defer w.subscriptionMutex.Unlock()
	// return append(w.subscriptions[:0:0], w.subscriptions...)
	return nil
}

// SetCanUseAuthenticatedEndpoints sets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *Websocket) SetCanUseAuthenticatedEndpoints(val bool) {
	// w.subscriptionMutex.Lock()
	// defer w.subscriptionMutex.Unlock()
	w.canUseAuthenticatedEndpoints = val
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *Websocket) CanUseAuthenticatedEndpoints() bool {
	// w.subscriptionMutex.Lock()
	// defer w.subscriptionMutex.Unlock()
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
		return fmt.Errorf("cannot set invalid websocket URL %s", s)
	}
	return nil
}
