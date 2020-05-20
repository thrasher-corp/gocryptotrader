package stream

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const defaultJobBuffer = 1000

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
	}
}

// Setup sets main variables for websocket connection
func (w *Websocket) Setup(setupData *WebsocketSetup) error {
	if w == nil {
		return errors.New("websocket is nil")
	}

	w.verbose = setupData.Verbose
	w.features = setupData.Features

	if w.features.Subscribe && setupData.Subscriber == nil {
		return errors.New("features have been set yet channel subscriber is not set")
	}
	w.channelSubscriber = setupData.Subscriber

	if w.features.Unsubscribe && setupData.UnSubscriber == nil {
		return errors.New("features have been set yet channel unsubscriber is not set")
	}
	w.channelUnsubscriber = setupData.UnSubscriber

	// if setupData.GenerateSubscriptions == nil {
	// 	return errors.New("channel GenerateSubscriptions is not set")
	// }
	w.channelGeneratesubs = setupData.GenerateSubscriptions

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
	w.trafficTimeout = setupData.WebsocketTimeout
	if setupData.Features == nil {
		return errors.New("feature set is nil")
	}

	w.SetWebsocketURL(setupData.RunningURL)
	w.SetCanUseAuthenticatedEndpoints(setupData.AuthenticatedWebsocketAPISupport)
	err := w.Initialise()
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
func (w *Websocket) SetupNewConnection(c ConnectionSetup, auth bool) error {
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
		ExchangeName:         w.exchangeName,
		URL:                  connectionURL,
		ProxyURL:             w.GetProxyAddress(),
		Verbose:              w.verbose,
		ResponseCheckTimeout: c.ResponseCheckTimeout,
		ResponseMaxLimit:     c.ResponseMaxLimit,
		Traffic:              w.TrafficAlert,
		readMessageErrors:    w.readMessageErrors,
		ShutdownC:            make(chan struct{}),
		idResponses:          make(map[int64]chan []byte),
	}

	if auth {
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
	w.ShutdownC = make(chan struct{})

	go w.dataMonitor()

	if w.features.Subscribe ||
		w.features.Unsubscribe ||
		w.features.FullPayloadSubscribe {
		fmt.Println("MANAGE SUBS!")
		w.Wg.Add(1)
		go w.manageSubscriptions()
	}

	var anotherWG sync.WaitGroup
	anotherWG.Add(1)
	go w.trafficMonitor(&anotherWG)

	err := w.connector()
	if err != nil {
		w.setConnectingStatus(false)
		return fmt.Errorf("%v Error connecting %s",
			w.exchangeName, err)
	}

	w.setConnectedStatus(true)
	w.setConnectingStatus(false)
	w.setInit(true)

	anotherWG.Wait()
	if !w.IsConnectionMonitorRunning() {
		go w.connectionMonitor()
	}

	return nil
}

// dataMonitor monitors job throughput and logs if there is a back log of data
func (w *Websocket) dataMonitor() {
	w.Wg.Add(1)
	defer func() {
		for {
			// Bleeds data from the websocket connection if needed
			select {
			case <-w.DataHandler:
			default:
				fmt.Println("DATA MONITOR DONE")
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
	fmt.Println("connection monitor started")
	if w.IsConnectionMonitorRunning() {
		return
	}
	w.setConnectionMonitorRunning(true)
	timer := time.NewTimer(connectionMonitorDelay)

	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		w.setConnectionMonitorRunning(false)
		if w.verbose {
			log.Debugf(log.WebsocketMgr,
				"%v websocket connection monitor exiting",
				w.exchangeName)
		}
	}()

	for {
		// if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v running connection monitor cycle",
			w.exchangeName)
		// }
		if !w.IsEnabled() {
			if w.verbose {
				log.Debugf(log.WebsocketMgr,
					"%v connectionMonitor: websocket disabled, shutting down",
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
					"%v websocket connection monitor exiting",
					w.exchangeName)
			}
			return
		}
		select {
		case err := <-w.readMessageErrors:
			// check if this error is a disconnection error
			if isDisconnectionError(err) {
				fmt.Println("disoconnection error")
				w.setConnectedStatus(false)
				w.setConnectingStatus(false)
				w.setInit(false)
				// if w.verbose {
				log.Debugf(log.WebsocketMgr,
					"%v websocket has been disconnected. Reason: %v",
					w.exchangeName, err)
				// }
				err = w.Connect()
				if err != nil {
					log.Error(log.WebsocketMgr, err)
				}
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(connectionMonitorDelay)
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
	defer func() {
		w.Orderbook.FlushCache()
		w.m.Unlock()
	}()
	if !w.IsConnected() {
		return fmt.Errorf("%v cannot shutdown a disconnected websocket",
			w.exchangeName)
	}
	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v shutting down websocket channels",
			w.exchangeName)
	}

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
	fmt.Println("SHUTTING DOWNNNNN")
	close(w.ShutdownC)
	fmt.Println("SHUTTING DOWNNNNN: WAITING")
	time.Sleep(time.Second)
	fmt.Println(&w.Wg)
	w.Wg.Wait()
	fmt.Println("shutdown called")
	w.setConnectedStatus(false)
	w.setConnectingStatus(false)
	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v completed websocket channel shutdown",
			w.exchangeName)
	}
	return nil
}

// RefreshConnection disconnects and reconnects websocket
func (w *Websocket) RefreshConnection() error {
	fmt.Println("refresh connection called")
	if w.features.Subscribe {
		fmt.Println("features subscribe enabled")
		newsubs, err := w.channelGeneratesubs()
		if err != nil {
			return err
		}

		subs, unsubs := w.GetChannelDifference(newsubs)
		if w.features.Unsubscribe {
			fmt.Println("features unsubscribe enabled")
			if len(unsubs) != 0 {
				err := w.RemoveSubscribedChannels(unsubs)
				if err != nil {
					return err
				}
			}

			if len(subs) != 0 {
				w.SubscribeToChannels(subs)
			}

			return nil
		} else if len(unsubs) == 0 {
			fmt.Println("features unsubscribe not enabled")
			if len(subs) != 0 {
				w.SubscribeToChannels(subs)
			}
			return nil
		}
	} else if w.features.FullPayloadSubscribe {
		newsubs, err := w.channelGeneratesubs()
		if err != nil {
			return err
		}

		if len(newsubs) != 0 {
			w.SubscribeToChannels(newsubs)
		}
		return nil
	}

	fmt.Println("feature subscribe and unsubscribe not enabled for exchange closing connection")

	err := w.Shutdown()
	if err != nil {
		return err
	}
	return w.Connect()
}

// trafficMonitor uses a timer of WebsocketTrafficLimitTime and once it expires
// Will reconnect if the TrafficAlert channel has not received any data
// The trafficTimer will reset on each traffic alert
func (w *Websocket) trafficMonitor(wg *sync.WaitGroup) {
	w.Wg.Add(1)
	wg.Done()
	trafficTimer := time.NewTimer(w.trafficTimeout)
	defer func() {
		if !trafficTimer.Stop() {
			select {
			case <-trafficTimer.C:
			default:
			}
		}
		w.setTrafficMonitorRunning(false)
		fmt.Println("traffic monitor done!")
		w.Wg.Done()
	}()
	if w.IsTrafficMonitorRunning() {
		fmt.Println("traffic monitor already started")
		return
	}
	w.setTrafficMonitorRunning(true)
	for {
		select {
		case <-w.ShutdownC:
			// if w.verbose {
			log.Debugf(log.WebsocketMgr,
				"%v trafficMonitor shutdown message received",
				w.exchangeName)
			// }
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
			// if w.verbose {
			log.Warnf(log.WebsocketMgr,
				"%v has not received a traffic alert in %v. Reconnecting",
				w.exchangeName,
				w.trafficTimeout)
			// }
			go w.Shutdown()
		}
	}
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

// SetWebsocketURL sets websocket URL and updates the underlying stream
// connection details
func (w *Websocket) SetWebsocketURL(websocketURL string, c ...Connection) {
	if websocketURL == "" || websocketURL == config.WebsocketURLNonDefaultMessage {
		w.runningURL = w.defaultURL
		return
	}
	w.runningURL = websocketURL

	for i := range c {
		c[i].SetURL(websocketURL)
	}
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
		return fmt.Errorf("%v Websocket already initialised", w.exchangeName)
	}
	w.setEnabled(w.enabled)
	return nil
}

// SetProxyAddress sets websocket proxy address
func (w *Websocket) SetProxyAddress(proxyAddr string) error {
	if w.proxyAddr == proxyAddr {
		return fmt.Errorf("%v Cannot set proxy address to the same address '%v'",
			w.exchangeName,
			w.proxyAddr)
	}

	w.proxyAddr = proxyAddr
	if !w.IsInit() && w.IsEnabled() {
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

// ManageSubscriptions ensures the subscriptions specified continue to be subscribed to
func (w *Websocket) manageSubscriptions() {
	defer func() {
		fmt.Println("Managed subscriptions done")
		w.Wg.Done()
	}()

	for {
		select {
		case <-w.ShutdownC:
			w.subscriptionMutex.Lock()
			w.subscriptions = nil
			w.subscriptionMutex.Unlock()
			if w.verbose {
				log.Debugf(log.WebsocketMgr,
					"%v shutdown manageSubscriptions",
					w.exchangeName)
			}
			return
		case sub := <-w.subscribe:
			if !w.IsConnected() {
				fmt.Println("not connected gee")
				fmt.Println("LOCK")
				w.subscriptionMutex.Lock()
				w.subscriptions = nil
				w.subscriptionMutex.Unlock()
				fmt.Println("UNLOCK")
				continue
			}
			if w.verbose {
				log.Debugf(log.WebsocketMgr,
					"%v checking subscriptions",
					w.exchangeName)
			}

			err := w.channelSubscriber(sub)
			if err != nil {
				w.DataHandler <- err
			}

		case unsub := <-w.unsubscribe:
			if !w.IsConnected() {
				w.subscriptionMutex.Lock()
				w.subscriptions = nil
				w.subscriptionMutex.Unlock()
				continue
			}

			err := w.channelUnsubscriber(unsub)
			if err != nil {
				w.DataHandler <- err
			}
		}
	}
}

// GetChannelDifference finds the difference between the subscribed channels
// and the new subscription list when pairs are disabled or enabled.
func (w *Websocket) GetChannelDifference(genSubs []ChannelSubscription) (sub []ChannelSubscription, unsub []ChannelSubscription) {
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

// RemoveSubscribedChannels removes supplied channels from channelsToSubscribe
func (w *Websocket) RemoveSubscribedChannels(channels []ChannelSubscription) error {
next:
	for x := range channels {
		for y := range w.subscriptions {
			if channels[x].Equal(&w.subscriptions[y]) {
				w.subscriptions[y] = w.subscriptions[len(w.subscriptions)-1]
				w.subscriptions = w.subscriptions[:len(w.subscriptions)-1]
				continue next
			}
		}
		return fmt.Errorf("subscription not found in list: %+v", channels[x])
	}
	w.unsubscribe <- channels
	return nil
}

// ResubscribeToChannel calls unsubscribe func and
// removes it from subscribedChannels to trigger a subscribe event
func (w *Websocket) ResubscribeToChannel(subscribedChannel *ChannelSubscription) error {
	w.subscriptionMutex.Lock()
	err := w.RemoveSubscribedChannels([]ChannelSubscription{*subscribedChannel})
	if err != nil {
		w.subscriptionMutex.Unlock()
		return err
	}

	w.SubscribeToChannels([]ChannelSubscription{*subscribedChannel})
	w.subscriptionMutex.Unlock()
	return nil
}

// SubscribeToChannels appends supplied channels to channelsToSubscribe
func (w *Websocket) SubscribeToChannels(channels []ChannelSubscription) {
	w.subscribe <- channels
	w.subscriptionMutex.Lock()
	w.subscriptions = append(w.subscriptions, channels...)
	w.subscriptionMutex.Unlock()
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

// MatchRequestResponse checks if a match is intended for the returned payload
// and returns true if match is possible, false if match cannot occur.
func (w *WebsocketConnection) MatchRequestResponse(id int64, data []byte) bool {
	w.Lock()
	defer w.Unlock()
	ch, ok := w.idResponses[id]
	if ok {
		select {
		case ch <- data:
		default:
			// this shouldn't occur but if it does continue to process as normal
			return false
		}
		return true
	}
	return false
}

// SendMessageReturnResponse will send a WS message to the connection and wait
// for response
func (w *WebsocketConnection) SendMessageReturnResponse(id int64, request interface{}) ([]byte, error) {
	ch := make(chan []byte, 1)

	w.Lock()
	w.idResponses[id] = ch
	w.Unlock()

	defer func() {
		w.Lock()
		close(ch)
		delete(w.idResponses, id)
		w.Unlock()
	}()

	err := w.SendJSONMessage(request)
	if err != nil {
		return nil, err
	}

	timer := time.NewTimer(w.ResponseMaxLimit)

	select {
	case payload := <-ch:
		return payload, nil
	case <-timer.C:
		timer.Stop()
		return nil, fmt.Errorf("timeout waiting for response with ID %v", id)
	}
}

// Dial sets proxy urls and then connects to the websocket
func (w *WebsocketConnection) Dial(dialer *websocket.Dialer, headers http.Header) error {
	if w.ProxyURL != "" {
		proxy, err := url.Parse(w.ProxyURL)
		if err != nil {
			return err
		}
		dialer.Proxy = http.ProxyURL(proxy)
	}
	var err error
	var conStatus *http.Response
	w.Connection, conStatus, err = dialer.Dial(w.URL, headers)
	if err != nil {
		fmt.Println("NOT CONNECTED")
		if conStatus != nil {
			return fmt.Errorf("%v %v %v Error: %v",
				w.URL,
				conStatus,
				conStatus.StatusCode,
				err)
		}
		return fmt.Errorf("%v Error: %v", w.URL, err)
	}
	if w.Verbose {
		log.Infof(log.WebsocketMgr,
			"%v Websocket connected to %s",
			w.ExchangeName,
			w.URL)
	}
	w.Traffic <- struct{}{}
	w.setConnectedStatus(true)
	return nil
}

// SendJSONMessage sends a JSON encoded message over the connection
func (w *WebsocketConnection) SendJSONMessage(data interface{}) error {
	w.Lock()
	defer w.Unlock()
	if !w.IsConnected() {
		return fmt.Errorf("%v cannot send message to a disconnected websocket",
			w.ExchangeName)
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr,
			"%v sending message to websocket %+v", w.ExchangeName, data)
	}
	if w.RateLimit > 0 {
		time.Sleep(time.Duration(w.RateLimit) * time.Millisecond)
	}
	return w.Connection.WriteJSON(data)
}

// SendRawMessage sends a message over the connection without JSON encoding it
func (w *WebsocketConnection) SendRawMessage(messageType int, message []byte) error {
	w.Lock()
	defer w.Unlock()
	if !w.IsConnected() {
		return fmt.Errorf("%v cannot send message to a disconnected websocket",
			w.ExchangeName)
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr,
			"%v sending message to websocket %s",
			w.ExchangeName,
			message)
	}
	if w.RateLimit > 0 {
		time.Sleep(time.Duration(w.RateLimit) * time.Millisecond)
	}
	return w.Connection.WriteMessage(messageType, message)
}

// SetupPingHandler will automatically send ping or pong messages based on
// WebsocketPingHandler configuration
func (w *WebsocketConnection) SetupPingHandler(handler PingHandler) {
	if handler.UseGorillaHandler {
		h := func(msg string) error {
			err := w.Connection.WriteControl(handler.MessageType,
				[]byte(msg),
				time.Now().Add(handler.Delay))
			if err == websocket.ErrCloseSent {
				return nil
			} else if e, ok := err.(net.Error); ok && e.Temporary() {
				return nil
			}
			return err
		}
		w.Connection.SetPingHandler(h)
		return
	}
	w.Wg.Add(1)
	defer w.Wg.Done()
	go func() {
		ticker := time.NewTicker(handler.Delay)
		for {
			select {
			case <-w.ShutdownC:
				ticker.Stop()
				return
			case <-ticker.C:
				err := w.SendRawMessage(handler.MessageType, handler.Message)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%v failed to send message to websocket %s",
						w.ExchangeName,
						handler.Message)
					return
				}
			}
		}
	}()
}

func (w *WebsocketConnection) setConnectedStatus(b bool) {
	if !b {
		fmt.Println("WOWOWOWOWOWOWOWO")
	}
	w.connectionMutex.Lock()
	w.connected = b
	w.connectionMutex.Unlock()
}

// IsConnected exposes websocket connection status
func (w *WebsocketConnection) IsConnected() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.connected
}

// ReadMessage reads messages, can handle text, gzip and binary
func (w *WebsocketConnection) ReadMessage() (Response, error) {
	mType, resp, err := w.Connection.ReadMessage()
	if err != nil {
		if isDisconnectionError(err) {
			fmt.Println("connection false")
			w.setConnectedStatus(false)
			w.readMessageErrors <- err
		}
		return Response{}, err
	}

	select {
	case w.Traffic <- struct{}{}:
	default: // causes contention, just bypass if there is no receiver.
	}

	var standardMessage []byte
	switch mType {
	case websocket.TextMessage:
		standardMessage = resp
	case websocket.BinaryMessage:
		standardMessage, err = w.parseBinaryResponse(resp)
		if err != nil {
			return Response{}, err
		}
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr,
			"%v Websocket message received: %v",
			w.ExchangeName,
			string(standardMessage))
	}
	return Response{Raw: standardMessage, Type: mType}, nil
}

// parseBinaryResponse parses a websocket binary response into a usable byte array
func (w *WebsocketConnection) parseBinaryResponse(resp []byte) ([]byte, error) {
	var standardMessage []byte
	var err error
	// Detect GZIP
	if resp[0] == 31 && resp[1] == 139 {
		b := bytes.NewReader(resp)
		var gReader *gzip.Reader
		gReader, err = gzip.NewReader(b)
		if err != nil {
			return standardMessage, err
		}
		standardMessage, err = ioutil.ReadAll(gReader)
		if err != nil {
			return standardMessage, err
		}
		err = gReader.Close()
		if err != nil {
			return standardMessage, err
		}
	} else {
		reader := flate.NewReader(bytes.NewReader(resp))
		standardMessage, err = ioutil.ReadAll(reader)
		if err != nil {
			return standardMessage, err
		}
		err = reader.Close()
		if err != nil {
			return standardMessage, err
		}
	}
	return standardMessage, nil
}

// GenerateMessageID Creates a messageID to checkout
func (w *WebsocketConnection) GenerateMessageID(useNano bool) int64 {
	if useNano {
		// force clock shift
		time.Sleep(time.Nanosecond)
		return time.Now().UnixNano()
	}
	return time.Now().Unix()
}

// GetShutdownChannel returns the underlying shutdown mechanism
func (w *WebsocketConnection) GetShutdownChannel() chan struct{} {
	return w.ShutdownC
}

// Shutdown shuts down and closes specific connection
func (w *WebsocketConnection) Shutdown() error {
	if w == nil || w.Connection == nil {
		return nil
	}
	w.Wg.Wait()
	return w.Connection.UnderlyingConn().Close()
}

// SetURL sets connection URL
func (w *WebsocketConnection) SetURL(url string) {
	w.URL = url
	return
}

// isDisconnectionError Determines if the error sent over chan ReadMessageErrors is a disconnection error
func isDisconnectionError(err error) bool {
	if websocket.IsUnexpectedCloseError(err) {
		fmt.Println("unexpected close")
		return true
	}
	switch e := err.(type) {
	case *websocket.CloseError:
		return true
	case *net.OpError:
		if e.Err.Error() == "use of closed network connection" {
			return false
		}
		fmt.Println("websocket close error", e.Err)
		return true
	}
	return false
}
