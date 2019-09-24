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
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	log "github.com/thrasher-corp/gocryptotrader/logger"
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
	w.enabled = setupData.WsEnabled
	err := w.Initialise()
	if err != nil {
		return err
	}
	w.SetDefaultURL(setupData.DefaultURL)
	w.SetConnector(setupData.Connector)
	w.SetWebsocketURL(setupData.RunningURL)
	w.SetExchangeName(setupData.ExchangeName)
	w.SetCanUseAuthenticatedEndpoints(setupData.AuthenticatedWebsocketAPISupport)
	w.trafficTimeout = setupData.WebsocketTimeout

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
	if w.SupportsFunctionality(WebsocketSubscribeSupported) || w.SupportsFunctionality(WebsocketUnsubscribeSupported) {
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
		timer.Stop()
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
				<-timer.C
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
	if !w.IsConnected() && w.ShutdownC == nil {
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
		trafficTimer.Stop()
		w.Wg.Done()
	}()

	for {
		select {
		case <-w.ShutdownC:
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v trafficMonitor shutdown message received", w.exchangeName)
			}
			return
		case <-w.TrafficAlert:
			if !trafficTimer.Stop() {
				<-trafficTimer.C
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
	if !w.IsInit() {
		if w.IsConnected() {
			return nil
		}
		return w.Connect()
	}
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

// GetFunctionality returns a functionality bitmask for the websocket
// connection
func (w *Websocket) GetFunctionality() uint32 {
	return w.Functionality
}

// SupportsFunctionality returns if the functionality is supported as a boolean
func (w *Websocket) SupportsFunctionality(f uint32) bool {
	return w.GetFunctionality()&f == f
}

// FormatFunctionality will return each of the websocket connection compatible
// stream methods as a string
func (w *Websocket) FormatFunctionality() string {
	var functionality []string
	for i := 0; i < 32; i++ {
		var check uint32 = 1 << uint32(i)
		if w.GetFunctionality()&check != 0 {
			switch check {
			case WebsocketTickerSupported:
				functionality = append(functionality, WebsocketTickerSupportedText)

			case WebsocketOrderbookSupported:
				functionality = append(functionality, WebsocketOrderbookSupportedText)

			case WebsocketKlineSupported:
				functionality = append(functionality, WebsocketKlineSupportedText)

			case WebsocketTradeDataSupported:
				functionality = append(functionality, WebsocketTradeDataSupportedText)

			case WebsocketAccountSupported:
				functionality = append(functionality, WebsocketAccountSupportedText)

			case WebsocketAllowsRequests:
				functionality = append(functionality, WebsocketAllowsRequestsText)

			case WebsocketSubscribeSupported:
				functionality = append(functionality, WebsocketSubscribeSupportedText)

			case WebsocketUnsubscribeSupported:
				functionality = append(functionality, WebsocketUnsubscribeSupportedText)

			case WebsocketAuthenticatedEndpointsSupported:
				functionality = append(functionality, WebsocketAuthenticatedEndpointsSupportedText)

			case WebsocketAccountDataSupported:
				functionality = append(functionality, WebsocketAccountDataSupportedText)

			case WebsocketSubmitOrderSupported:
				functionality = append(functionality, WebsocketSubmitOrderSupportedText)

			case WebsocketCancelOrderSupported:
				functionality = append(functionality, WebsocketCancelOrderSupportedText)

			case WebsocketWithdrawSupported:
				functionality = append(functionality, WebsocketWithdrawSupportedText)

			case WebsocketMessageCorrelationSupported:
				functionality = append(functionality, WebsocketMessageCorrelationSupportedText)

			case WebsocketSequenceNumberSupported:
				functionality = append(functionality, WebsocketSequenceNumberSupportedText)

			case WebsocketDeadMansSwitchSupported:
				functionality = append(functionality, WebsocketDeadMansSwitchSupportedText)

			default:
				functionality = append(functionality,
					fmt.Sprintf("%s[1<<%v]", UnknownWebsocketFunctionality, i))
			}
		}
	}

	if len(functionality) > 0 {
		return strings.Join(functionality, " & ")
	}

	return NoWebsocketSupportText
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
	if !w.SupportsFunctionality(WebsocketSubscribeSupported) && !w.SupportsFunctionality(WebsocketUnsubscribeSupported) {
		w.DataHandler <- fmt.Errorf("%v does not support channel subscriptions, exiting ManageSubscriptions()", w.exchangeName)
		return
	}
	w.Wg.Add(1)
	defer func() {
		if w.verbose {
			log.Debugf(log.WebsocketMgr, "%v ManageSubscriptions exiting", w.exchangeName)
		}
		w.Wg.Done()
	}()
	for {
		select {
		case <-w.ShutdownC:
			w.subscriptionLock.Lock()
			w.subscribedChannels = []WebsocketChannelSubscription{}
			w.subscriptionLock.Unlock()
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v shutdown manageSubscriptions", w.exchangeName)
			}
			return
		default:
			time.Sleep(manageSubscriptionsDelay)
			if !w.IsConnected() {
				w.subscriptionLock.Lock()
				w.subscribedChannels = []WebsocketChannelSubscription{}
				w.subscriptionLock.Unlock()

				continue
			}
			if w.verbose {
				log.Debugf(log.WebsocketMgr, "%v checking subscriptions", w.exchangeName)
			}
			// Subscribe to channels Pending a subscription
			if w.SupportsFunctionality(WebsocketSubscribeSupported) {
				err := w.appendSubscribedChannels()
				if err != nil {
					w.DataHandler <- err
				}
			}
			if w.SupportsFunctionality(WebsocketUnsubscribeSupported) {
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
	w.subscriptionLock.Lock()
	defer w.subscriptionLock.Unlock()
	for i := 0; i < len(w.channelsToSubscribe); i++ {
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
	w.subscriptionLock.Lock()
	defer w.subscriptionLock.Unlock()
	for i := 0; i < len(w.subscribedChannels); i++ {
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
	w.subscriptionLock.Lock()
	defer w.subscriptionLock.Unlock()
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
	w.subscriptionLock.Lock()
	defer w.subscriptionLock.Unlock()
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
	w.subscriptionLock.Lock()
	defer w.subscriptionLock.Unlock()
	w.canUseAuthenticatedEndpoints = val
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *Websocket) CanUseAuthenticatedEndpoints() bool {
	w.subscriptionLock.Lock()
	defer w.subscriptionLock.Unlock()
	return w.canUseAuthenticatedEndpoints
}

// AddResponseWithID adds data to IDResponses with locks and a nil check
func (w *WebsocketConnection) AddResponseWithID(id int64, data []byte) {
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
	if err != nil {
		if conStatus != nil {
			return fmt.Errorf("%v %v %v Error: %v", w.URL, conStatus, conStatus.StatusCode, err)
		}
		return fmt.Errorf("%v Error: %v", w.URL, err)
	}
	if w.Verbose {
		log.Infof(log.WebsocketMgr, "%v Websocket connected", w.ExchangeName)
	}
	w.setConnectedStatus(true)
	return nil
}

// SendMessage the one true message request. Sends message to WS
func (w *WebsocketConnection) SendMessage(data interface{}) error {
	w.Lock()
	defer w.Unlock()
	if !w.IsConnected() {
		return fmt.Errorf("%v cannot send message to a disconnected websocket", w.ExchangeName)
	}
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	if w.Verbose {
		log.Debugf(log.WebsocketMgr,
			"%v sending message to websocket %v", w.ExchangeName, string(json))
	}
	if w.RateLimit > 0 {
		time.Sleep(time.Duration(w.RateLimit) * time.Millisecond)
	}
	return w.Connection.WriteMessage(websocket.TextMessage, json)
}

// SendMessageReturnResponse will send a WS message to the connection
// It will then run a goroutine to await a JSON response
// If there is no response it will return an error
func (w *WebsocketConnection) SendMessageReturnResponse(id int64, request interface{}) ([]byte, error) {
	err := w.SendMessage(request)
	if err != nil {
		return nil, err
	}
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
				if k == id {
					w.Unlock()
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
	defer w.connectionMutex.RUnlock()
	return w.connected
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
	switch err.(type) {
	case *websocket.CloseError, *net.OpError:
		return true
	}
	return false
}
