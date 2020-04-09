package wshandler

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

// New initialises the websocket struct
func New() *Websocket {
	return &Websocket{
		defaultURL: "",
		enabled:    false,
		proxyAddr:  "",
		runningURL: "",
		init:       true,
	}
}

// Setup sets main variables for websocket connection
func (w *Websocket) Setup(setupData *WebsocketSetup) error {
	w.DataHandler = make(chan interface{}, 1)
	w.TrafficAlert = make(chan struct{}, 1)
	w.verbose = setupData.Verbose
	w.SetChannelSubscriber(setupData.Subscriber)
	w.SetChannelUnsubscriber(setupData.UnSubscriber)
	w.enabled = setupData.Enabled
	w.SetDefaultURL(setupData.DefaultURL)
	w.SetConnector(setupData.Connector)
	w.SetWebsocketURL(setupData.RunningURL)
	w.SetExchangeName(setupData.ExchangeName)
	w.SetCanUseAuthenticatedEndpoints(setupData.AuthenticatedWebsocketAPISupport)
	w.trafficTimeout = setupData.WebsocketTimeout
	w.features = setupData.Features
	err := w.Initialise()
	if err != nil {
		return err
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
	w.ShutdownC = make(chan struct{}, 1)
	w.ReadMessageErrors = make(chan error, 1)
	err := w.connector()
	if err != nil {
		w.setConnectingStatus(false)
		return fmt.Errorf("%v Error connecting %s",
			w.exchangeName, err)
	}

	w.setConnectedStatus(true)
	w.setConnectingStatus(false)
	w.setInit(true)

	var anotherWG sync.WaitGroup
	anotherWG.Add(1)
	go w.trafficMonitor(&anotherWG)
	anotherWG.Wait()
	if !w.IsConnectionMonitorRunning() {
		go w.connectionMonitor()
	}
	if w.features.Subscribe || w.features.Unsubscribe {
		w.Wg.Add(1)
		go w.manageSubscriptions()
	}

	return nil
}

// connectionMonitor ensures that the WS keeps connecting
func (w *Websocket) connectionMonitor() {
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
			log.Debugf(log.WebsocketMgr, "%v websocket connection monitor exiting",
				w.exchangeName)
		}
	}()

	for {
		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%v running connection monitor cycle",
				w.exchangeName)
		}
		if !w.IsEnabled() {
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v connectionMonitor: websocket disabled, shutting down", w.exchangeName)
			}
			if w.IsConnected() {
				err := w.Shutdown()
				if err != nil {
					log.Error(log.WebsocketMgr, err)
				}
			}
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v websocket connection monitor exiting",
					w.exchangeName)
			}
			return
		}
		select {
		case err := <-w.ReadMessageErrors:
			// check if this error is a disconnection error
			if isDisconnectionError(err) {
				w.setConnectedStatus(false)
				w.setConnectingStatus(false)
				w.setInit(false)
				if w.verbose {
					log.Debugf(log.WebsocketMgr, "%v websocket has been disconnected. Reason: %v",
						w.exchangeName, err)
				}
				err = w.Connect()
				if err != nil {
					log.Error(log.WebsocketMgr, err)
				}
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
		return fmt.Errorf("%v cannot shutdown a disconnected websocket", w.exchangeName)
	}
	if w.verbose {
		log.Debugf(log.WebsocketMgr, "%v shutting down websocket channels", w.exchangeName)
	}
	close(w.ShutdownC)
	w.Wg.Wait()
	w.setConnectedStatus(false)
	w.setConnectingStatus(false)
	if w.verbose {
		log.Debugf(log.WebsocketMgr, "%v completed websocket channel shutdown", w.exchangeName)
	}
	return nil
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
		w.Wg.Done()
	}()
	if w.IsTrafficMonitorRunning() {
		return
	}
	w.setTrafficMonitorRunning(true)
	for {
		select {
		case <-w.ShutdownC:
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v trafficMonitor shutdown message received", w.exchangeName)
			}
			return
		case <-w.TrafficAlert:
			if !trafficTimer.Stop() {
				select {
				case <-trafficTimer.C:
				default:
				}
			}
			trafficTimer.Reset(w.trafficTimeout)
		case <-trafficTimer.C: // Falls through when timer runs out
			if w.verbose {
				log.Warnf(log.WebsocketMgr, "%v has not received a traffic alert in %v. Reconnecting", w.exchangeName, w.trafficTimeout)
			}
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
	isConnected := w.connected
	w.connectionMutex.RUnlock()
	return isConnected
}

func (w *Websocket) setConnectingStatus(b bool) {
	w.connectionMutex.Lock()
	w.connecting = b
	w.connectionMutex.Unlock()
}

// IsConnecting returns status of connecting
func (w *Websocket) IsConnecting() bool {
	w.connectionMutex.RLock()
	isConnecting := w.connecting
	w.connectionMutex.RUnlock()
	return isConnecting
}

func (w *Websocket) setEnabled(b bool) {
	w.connectionMutex.Lock()
	w.enabled = b
	w.connectionMutex.Unlock()
}

// IsEnabled returns status of enabled
func (w *Websocket) IsEnabled() bool {
	w.connectionMutex.RLock()
	isEnabled := w.enabled
	w.connectionMutex.RUnlock()
	return isEnabled
}

func (w *Websocket) setInit(b bool) {
	w.connectionMutex.Lock()
	w.init = b
	w.connectionMutex.Unlock()
}

// IsInit returns status of init
func (w *Websocket) IsInit() bool {
	w.connectionMutex.RLock()
	isInit := w.init
	w.connectionMutex.RUnlock()
	return isInit
}

func (w *Websocket) setTrafficMonitorRunning(b bool) {
	w.connectionMutex.Lock()
	w.trafficMonitorRunning = b
	w.connectionMutex.Unlock()
}

// IsTrafficMonitorRunning returns status of the traffic monitor
func (w *Websocket) IsTrafficMonitorRunning() bool {
	w.connectionMutex.RLock()
	trafficMonRunning := w.trafficMonitorRunning
	w.connectionMutex.RUnlock()
	return trafficMonRunning
}

func (w *Websocket) setConnectionMonitorRunning(b bool) {
	w.connectionMutex.Lock()
	w.connectionMonitorRunning = b
	w.connectionMutex.Unlock()
}

// IsConnectionMonitorRunning returns status of connection monitor
func (w *Websocket) IsConnectionMonitorRunning() bool {
	w.connectionMutex.RLock()
	isConnMonRunning := w.connectionMonitorRunning
	w.connectionMutex.RUnlock()
	return isConnMonRunning
}

// CanUseAuthenticatedWebsocketForWrapper Handles a common check to
// verify whether a wrapper can use an authenticated websocket endpoint
func (w *Websocket) CanUseAuthenticatedWebsocketForWrapper() bool {
	if w.IsConnected() && w.CanUseAuthenticatedEndpoints() {
		return true
	} else if w.IsConnected() && !w.CanUseAuthenticatedEndpoints() {
		log.Infof(log.WebsocketMgr, WebsocketNotAuthenticatedUsingRest, w.exchangeName)
	}
	return false
}

// SetWebsocketURL sets websocket URL
func (w *Websocket) SetWebsocketURL(websocketURL string) {
	if websocketURL == "" || websocketURL == config.WebsocketURLNonDefaultMessage {
		w.runningURL = w.defaultURL
		return
	}
	w.runningURL = websocketURL
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
		return fmt.Errorf("%v Websocket already initialised",
			w.exchangeName)
	}
	w.setEnabled(w.enabled)
	return nil
}

// SetProxyAddress sets websocket proxy address
func (w *Websocket) SetProxyAddress(proxyAddr string) error {
	if w.proxyAddr == proxyAddr {
		return fmt.Errorf("%v Cannot set proxy address to the same address '%v'", w.exchangeName, w.proxyAddr)
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

// SetDefaultURL sets default websocket URL
func (w *Websocket) SetDefaultURL(defaultURL string) {
	w.defaultURL = defaultURL
}

// GetDefaultURL returns the default websocket URL
func (w *Websocket) GetDefaultURL() string {
	return w.defaultURL
}

// SetConnector sets connection function
func (w *Websocket) SetConnector(connector func() error) {
	w.connector = connector
}

// SetExchangeName sets exchange name
func (w *Websocket) SetExchangeName(exchName string) {
	w.exchangeName = exchName
}

// GetName returns exchange name
func (w *Websocket) GetName() string {
	return w.exchangeName
}

// SetChannelSubscriber sets the function to use the base subscribe func
func (w *Websocket) SetChannelSubscriber(subscriber func(channelToSubscribe WebsocketChannelSubscription) error) {
	w.channelSubscriber = subscriber
}

// SetChannelUnsubscriber sets the function to use the base unsubscribe func
func (w *Websocket) SetChannelUnsubscriber(unsubscriber func(channelToUnsubscribe WebsocketChannelSubscription) error) {
	w.channelUnsubscriber = unsubscriber
}

// ManageSubscriptions ensures the subscriptions specified continue to be subscribed to
func (w *Websocket) manageSubscriptions() {
	if !w.features.Subscribe && !w.features.Unsubscribe {
		w.DataHandler <- fmt.Errorf("%v does not support channel subscriptions, exiting ManageSubscriptions()", w.exchangeName)
		return
	}
	defer func() {
		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%v ManageSubscriptions exiting", w.exchangeName)
		}
		w.Wg.Done()
	}()
	for {
		select {
		case <-w.ShutdownC:
			w.subscriptionMutex.Lock()
			w.subscribedChannels = []WebsocketChannelSubscription{}
			w.subscriptionMutex.Unlock()
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v shutdown manageSubscriptions", w.exchangeName)
			}
			return
		default:
			time.Sleep(manageSubscriptionsDelay)
			if !w.IsConnected() {
				w.subscriptionMutex.Lock()
				w.subscribedChannels = []WebsocketChannelSubscription{}
				w.subscriptionMutex.Unlock()

				continue
			}
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v checking subscriptions", w.exchangeName)
			}
			// Subscribe to channels Pending a subscription
			if w.features.Subscribe {
				err := w.appendSubscribedChannels()
				if err != nil {
					w.DataHandler <- err
				}
			}
			if w.features.Unsubscribe {
				err := w.unsubscribeToChannels()
				if err != nil {
					w.DataHandler <- err
				}
			}
		}
	}
}

// appendSubscribedChannels compares channelsToSubscribe to subscribedChannels
// and subscribes to any channels not present in subscribedChannels
func (w *Websocket) appendSubscribedChannels() error {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	for i := range w.channelsToSubscribe {
		channelIsSubscribed := false
		for j := 0; j < len(w.subscribedChannels); j++ {
			if w.subscribedChannels[j].Equal(&w.channelsToSubscribe[i]) {
				channelIsSubscribed = true
				break
			}
		}
		if !channelIsSubscribed {
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v Subscribing to %v %v", w.exchangeName, w.channelsToSubscribe[i].Channel, w.channelsToSubscribe[i].Currency.String())
			}
			err := w.channelSubscriber(w.channelsToSubscribe[i])
			if err != nil {
				return err
			}
			w.subscribedChannels = append(w.subscribedChannels, w.channelsToSubscribe[i])
		}
	}
	return nil
}

// unsubscribeToChannels compares subscribedChannels to channelsToSubscribe
// and unsubscribes to any channels not present in  channelsToSubscribe
func (w *Websocket) unsubscribeToChannels() error {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	for i := range w.subscribedChannels {
		subscriptionFound := false
		for j := 0; j < len(w.channelsToSubscribe); j++ {
			if w.channelsToSubscribe[j].Equal(&w.subscribedChannels[i]) {
				subscriptionFound = true
				break
			}
		}
		if !subscriptionFound {
			err := w.channelUnsubscriber(w.subscribedChannels[i])
			if err != nil {
				return err
			}
		}
	}
	// Now that the slices should match, assign rather than looping and appending the differences
	w.subscribedChannels = append(w.channelsToSubscribe[:0:0], w.channelsToSubscribe...) //nolint:gocritic

	return nil
}

// RemoveSubscribedChannels removes supplied channels from channelsToSubscribe
func (w *Websocket) RemoveSubscribedChannels(channels []WebsocketChannelSubscription) {
	for i := range channels {
		w.removeChannelToSubscribe(channels[i])
	}
}

// removeChannelToSubscribe removes an entry from w.channelsToSubscribe
// so an unsubscribe event can be triggered
func (w *Websocket) removeChannelToSubscribe(subscribedChannel WebsocketChannelSubscription) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	channelLength := len(w.channelsToSubscribe)
	i := 0
	for j := 0; j < len(w.channelsToSubscribe); j++ {
		if !w.channelsToSubscribe[j].Equal(&subscribedChannel) {
			w.channelsToSubscribe[i] = w.channelsToSubscribe[j]
			i++
		}
	}
	w.channelsToSubscribe = w.channelsToSubscribe[:i]
	if channelLength == len(w.channelsToSubscribe) {
		w.DataHandler <- fmt.Errorf("%v removeChannelToSubscribe() Channel %v Currency %v could not be removed because it was not found",
			w.exchangeName,
			subscribedChannel.Channel,
			subscribedChannel.Currency)
	}
}

// ResubscribeToChannel calls unsubscribe func and
// removes it from subscribedChannels to trigger a subscribe event
func (w *Websocket) ResubscribeToChannel(subscribedChannel WebsocketChannelSubscription) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	err := w.channelUnsubscriber(subscribedChannel)
	if err != nil {
		w.DataHandler <- err
	}
	// Remove the channel from the list of subscribed channels
	// ManageSubscriptions will automatically resubscribe
	i := 0
	for j := 0; j < len(w.subscribedChannels); j++ {
		if !w.subscribedChannels[j].Equal(&subscribedChannel) {
			w.subscribedChannels[i] = w.subscribedChannels[j]
			i++
		}
	}
	w.subscribedChannels = w.subscribedChannels[:i]
}

// SubscribeToChannels appends supplied channels to channelsToSubscribe
func (w *Websocket) SubscribeToChannels(channels []WebsocketChannelSubscription) {
	for i := range channels {
		channelFound := false
		for j := range w.channelsToSubscribe {
			if w.channelsToSubscribe[j].Equal(&channels[i]) {
				channelFound = true
			}
		}
		if !channelFound {
			w.channelsToSubscribe = append(w.channelsToSubscribe, channels[i])
		}
	}
}

// Equal two WebsocketChannelSubscription to determine equality
func (w *WebsocketChannelSubscription) Equal(subscribedChannel *WebsocketChannelSubscription) bool {
	return strings.EqualFold(w.Channel, subscribedChannel.Channel) &&
		strings.EqualFold(w.Currency.String(), subscribedChannel.Currency.String())
}

// GetSubscriptions returns a copied list of subscriptions
// subscriptions is a private member and cannot be manipulated
func (w *Websocket) GetSubscriptions() []WebsocketChannelSubscription {
	return append(w.subscribedChannels[:0:0], w.subscribedChannels...)
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
	canUseAuthEndpoints := w.canUseAuthenticatedEndpoints
	w.subscriptionMutex.Unlock()
	return canUseAuthEndpoints
}

// SetResponseIDAndData adds data to IDResponses with locks and a nil check
func (w *WebsocketConnection) SetResponseIDAndData(id int64, data []byte) {
	w.Lock()
	defer w.Unlock()
	if w.IDResponses == nil {
		w.IDResponses = make(map[int64][]byte)
	}
	w.IDResponses[id] = data
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
	if conStatus != nil {
		conStatus.Body.Close()
	}
	if err != nil {
		if conStatus != nil {
			return fmt.Errorf("%v %v %v Error: %v", w.URL, conStatus, conStatus.StatusCode, err)
		}
		return fmt.Errorf("%v Error: %v", w.URL, err)
	}
	if w.Verbose {
		log.Infof(log.WebsocketMgr, "%v Websocket connected to %s", w.ExchangeName, w.URL)
	}
	w.setConnectedStatus(true)
	return nil
}

// SendJSONMessage sends a JSON encoded message over the connection
func (w *WebsocketConnection) SendJSONMessage(data interface{}) error {
	w.Lock()
	defer w.Unlock()
	if !w.IsConnected() {
		return fmt.Errorf("%v cannot send message to a disconnected websocket", w.ExchangeName)
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
		return fmt.Errorf("%v cannot send message to a disconnected websocket", w.ExchangeName)
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr,
			"%v sending message to websocket %s", w.ExchangeName, message)
	}
	if w.RateLimit > 0 {
		time.Sleep(time.Duration(w.RateLimit) * time.Millisecond)
	}
	return w.Connection.WriteMessage(messageType, message)
}

// SetupPingHandler will automatically send ping or pong messages based on
// WebsocketPingHandler configuration
func (w *WebsocketConnection) SetupPingHandler(handler WebsocketPingHandler) {
	if handler.UseGorillaHandler {
		h := func(msg string) error {
			err := w.Connection.WriteControl(handler.MessageType, []byte(msg), time.Now().Add(handler.Delay))
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
			case <-w.Shutdown:
				ticker.Stop()
				return
			case <-ticker.C:
				err := w.SendRawMessage(handler.MessageType, handler.Message)
				if err != nil {
					log.Errorf(log.WebsocketMgr,
						"%v failed to send message to websocket %s", w.ExchangeName, handler.Message)
					return
				}
			}
		}
	}()
}

// SendMessageReturnResponse will send a WS message to the connection
// It will then run a goroutine to await a JSON response
// If there is no response it will return an error
func (w *WebsocketConnection) SendMessageReturnResponse(id int64, request interface{}) ([]byte, error) {
	err := w.SendJSONMessage(request)
	if err != nil {
		return nil, err
	}
	w.SetResponseIDAndData(id, nil)
	var wg sync.WaitGroup
	wg.Add(1)
	go w.WaitForResult(id, &wg)
	defer func() {
		delete(w.IDResponses, id)
	}()
	wg.Wait()
	if _, ok := w.IDResponses[id]; !ok {
		return nil, fmt.Errorf("timeout waiting for response with ID %v", id)
	}

	return w.IDResponses[id], nil
}

// IsIDWaitingForResponse will verify whether the websocket is awaiting
// a response with a correlating ID. If true, the datahandler won't process
// the data, and instead will be processed by the wrapper function
func (w *WebsocketConnection) IsIDWaitingForResponse(id int64) bool {
	w.Lock()
	defer w.Unlock()
	for k := range w.IDResponses {
		if k == id && w.IDResponses[k] == nil {
			return true
		}
	}
	return false
}

// WaitForResult will keep checking w.IDResponses for a response ID
// If the timer expires, it will return without
func (w *WebsocketConnection) WaitForResult(id int64, wg *sync.WaitGroup) {
	defer wg.Done()
	timer := time.NewTimer(w.ResponseMaxLimit)
	for {
		select {
		case <-timer.C:
			return
		default:
			w.Lock()
			for k := range w.IDResponses {
				if k == id && w.IDResponses[k] != nil {
					w.Unlock()
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					return
				}
			}
			w.Unlock()
			time.Sleep(w.ResponseCheckTimeout)
		}
	}
}

func (w *WebsocketConnection) setConnectedStatus(b bool) {
	w.connectionMutex.Lock()
	w.connected = b
	w.connectionMutex.Unlock()
}

// IsConnected exposes websocket connection status
func (w *WebsocketConnection) IsConnected() bool {
	w.connectionMutex.RLock()
	isConnected := w.connected
	w.connectionMutex.RUnlock()
	return isConnected
}

// ReadMessage reads messages, can handle text, gzip and binary
func (w *WebsocketConnection) ReadMessage() (WebsocketResponse, error) {
	mType, resp, err := w.Connection.ReadMessage()
	if err != nil {
		if isDisconnectionError(err) {
			w.setConnectedStatus(false)
		}
		return WebsocketResponse{}, err
	}
	var standardMessage []byte
	switch mType {
	case websocket.TextMessage:
		standardMessage = resp
	case websocket.BinaryMessage:
		standardMessage, err = w.parseBinaryResponse(resp)
		if err != nil {
			return WebsocketResponse{}, err
		}
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr, "%v Websocket message received: %v",
			w.ExchangeName,
			string(standardMessage))
	}
	return WebsocketResponse{Raw: standardMessage, Type: mType}, nil
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
		return time.Now().UnixNano()
	}
	return time.Now().Unix()
}

// isDisconnectionError Determines if the error sent over chan ReadMessageErrors is a disconnection error
func isDisconnectionError(err error) bool {
	if websocket.IsUnexpectedCloseError(err) {
		return true
	}
	switch err.(type) {
	case *websocket.CloseError, *net.OpError:
		return true
	}
	return false
}
