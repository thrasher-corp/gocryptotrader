package wshandler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/idoall/gocryptotrader/common"
	"github.com/idoall/gocryptotrader/config"
	log "github.com/idoall/gocryptotrader/logger"
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
func (w *Websocket) Setup(connector func() error,
	subscriber func(channelToSubscribe WebsocketChannelSubscription) error,
	unsubscriber func(channelToUnsubscribe WebsocketChannelSubscription) error,
	exchangeName string,
	wsEnabled,
	verbose bool,
	defaultURL,
	runningURL string,
	authenticatedWebsocketAPISupport bool) error {

	w.DataHandler = make(chan interface{}, 1)
	w.Connected = make(chan struct{}, 1)
	w.Disconnected = make(chan struct{}, 1)
	w.TrafficAlert = make(chan struct{}, 1)
	w.verbose = verbose

	w.SetChannelSubscriber(subscriber)
	w.SetChannelUnsubscriber(unsubscriber)
	err := w.SetWsStatusAndConnection(wsEnabled)
	if err != nil {
		return err
	}
	w.SetDefaultURL(defaultURL)
	w.SetConnector(connector)
	w.SetWebsocketURL(runningURL)
	w.SetExchangeName(exchangeName)
	w.SetCanUseAuthenticatedEndpoints(authenticatedWebsocketAPISupport)

	w.init = false
	w.noConnectionCheckLimit = 5
	w.reconnectionLimit = 10

	return nil
}

// Connect intiates a websocket connection by using a package defined connection
// function
func (w *Websocket) Connect() error {
	w.m.Lock()
	defer w.m.Unlock()

	if !w.IsEnabled() {
		return errors.New(WebsocketNotEnabled)
	}

	if w.connected {
		w.connecting = false
		return errors.New("exchange_websocket.go error - already connected, cannot connect again")
	}

	w.connecting = true
	w.ShutdownC = make(chan struct{}, 1)
	err := w.connector()
	if err != nil {
		w.connecting = false
		return fmt.Errorf("exchange_websocket.go connection error %s",
			err)
	}

	if !w.connected {
		w.Connected <- struct{}{}
		w.connected = true
		w.connecting = false
	}

	var anotherWG sync.WaitGroup
	anotherWG.Add(1)
	go w.trafficMonitor(&anotherWG)
	anotherWG.Wait()
	if !w.connectionMonitorRunning {
		go w.connectionMonitor()
	}
	go w.manageSubscriptions()

	return nil
}

// connectionMonitor ensures that the WS keeps connecting
func (w *Websocket) connectionMonitor() {
	w.m.Lock()
	w.connectionMonitorRunning = true
	w.m.Unlock()
	defer func() {
		w.connectionMonitorRunning = false
	}()

	for {
		time.Sleep(connectionMonitorDelay)
		w.m.Lock()
		if !w.enabled {
			w.m.Unlock()
			w.DataHandler <- fmt.Errorf("%v connectionMonitor: websocket disabled, shutting down", w.exchangeName)
			err := w.Shutdown()
			if err != nil {
				log.Error(err)
			}
			if w.verbose {
				log.Debugf("%v connectionMonitor exiting", w.exchangeName)
			}
			return
		}
		w.m.Unlock()
		err := w.checkConnection()
		if err != nil {
			log.Error(err)
		}
	}
}

// checkConnection ensures the connection is maintained
// Will reconnect on disconnect
func (w *Websocket) checkConnection() error {
	if w.verbose {
		log.Debugf("%v checking connection", w.exchangeName)
	}
	switch {
	case !w.IsConnected() && !w.IsConnecting():
		w.m.Lock()
		defer w.m.Unlock()
		if w.verbose {
			log.Debugf("%v no connection. Attempt %v/%v", w.exchangeName, w.noConnectionChecks, w.noConnectionCheckLimit)
		}
		if w.noConnectionChecks >= w.noConnectionCheckLimit {
			if w.verbose {
				log.Debugf("%v resetting connection", w.exchangeName)
			}
			w.connecting = true
			go w.WebsocketReset()
			w.noConnectionChecks = 0
		}
		w.noConnectionChecks++
	case w.IsConnecting():
		if w.reconnectionChecks >= w.reconnectionLimit {
			return fmt.Errorf("%v websocket failed to reconnect after %v seconds",
				w.exchangeName,
				w.reconnectionLimit*int(connectionMonitorDelay.Seconds()))
		}
		if w.verbose {
			log.Debugf("%v Busy reconnecting", w.exchangeName)
		}
		w.reconnectionChecks++
	default:
		w.noConnectionChecks = 0
		w.reconnectionChecks = 0
	}
	return nil
}

// IsConnected exposes websocket connection status
func (w *Websocket) IsConnected() bool {
	w.m.Lock()
	defer w.m.Unlock()
	return w.connected
}

// IsConnecting checks whether websocket is busy connecting
func (w *Websocket) IsConnecting() bool {
	w.m.Lock()
	defer w.m.Unlock()
	return w.connecting
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *Websocket) Shutdown() error {
	w.m.Lock()
	defer func() {
		w.Orderbook.FlushCache()
		w.m.Unlock()
	}()
	if !w.connected && w.ShutdownC == nil {
		return fmt.Errorf("%v cannot shutdown a disconnected websocket", w.exchangeName)
	}
	if w.verbose {
		log.Debugf("%v shutting down websocket channels", w.exchangeName)
	}
	timer := time.NewTimer(15 * time.Second)
	c := make(chan struct{}, 1)

	go func(c chan struct{}) {
		close(w.ShutdownC)
		w.Wg.Wait()
		if w.verbose {
			log.Debugf("%v completed websocket channel shutdown", w.exchangeName)
		}
		c <- struct{}{}
	}(c)

	select {
	case <-c:
		w.connected = false
		return nil
	case <-timer.C:
		return fmt.Errorf("%s websocket routines failed to shutdown after 15 seconds",
			w.GetName())
	}
}

// WebsocketReset sends the shutdown command, waits for channel/func closure and then reconnects
func (w *Websocket) WebsocketReset() {
	err := w.Shutdown()
	if err != nil {
		// does not return here to allow connection to be made if already shut down
		w.DataHandler <- fmt.Errorf("%v shutdown error: %v", w.exchangeName, err)
	}
	log.Infof("%v reconnecting to websocket", w.exchangeName)
	w.m.Lock()
	w.init = true
	w.m.Unlock()
	err = w.Connect()
	if err != nil {
		w.DataHandler <- fmt.Errorf("%v connection error: %v", w.exchangeName, err)
	}
}

// trafficMonitor monitors traffic and switches connection modes for websocket
func (w *Websocket) trafficMonitor(wg *sync.WaitGroup) {
	w.Wg.Add(1)
	wg.Done() // Makes sure we are unlocking after we add to waitgroup
	defer func() {
		if w.connected {
			w.Disconnected <- struct{}{}
		}
		w.Wg.Done()
	}()

	// Define an initial traffic timer which will be a delay then fall over to
	// WebsocketTrafficLimitTime after first response
	trafficTimer := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-w.ShutdownC: // Returns on shutdown channel close
			if w.verbose {
				log.Debugf("%v trafficMonitor shutdown message received", w.exchangeName)
			}
			return
		case <-w.TrafficAlert: // Resets timer on traffic
			w.m.Lock()
			if !w.connected {
				w.Connected <- struct{}{}
				w.connected = true
			}
			w.m.Unlock()
			trafficTimer.Reset(WebsocketTrafficLimitTime)
		case <-trafficTimer.C: // Falls through when timer runs out
			newtimer := time.NewTimer(10 * time.Second) // New secondary timer set
			if w.verbose {
				log.Debugf("%v has not received a traffic alert in 5 seconds.", w.exchangeName)
			}
			w.m.Lock()
			if w.connected {
				// If connected divert traffic to rest
				w.Disconnected <- struct{}{}
				w.connected = false
			}
			w.m.Unlock()

			select {
			case <-w.ShutdownC: // Returns on shutdown channel close
				w.m.Lock()
				w.connected = false
				w.m.Unlock()
				return

			case <-newtimer.C: // If secondary timer runs state timeout is sent to the data handler
				if w.verbose {
					log.Debugf("%v has not received a traffic alert in 15 seconds, exiting", w.exchangeName)
				}
				w.DataHandler <- fmt.Errorf("trafficMonitor %v", WebsocketStateTimeout)
				return

			case <-w.TrafficAlert: // If in this time response traffic comes through
				trafficTimer.Reset(WebsocketTrafficLimitTime)
				w.m.Lock()
				if !w.connected {
					// If not connected dive rt traffic from REST to websocket
					w.Connected <- struct{}{}
					if w.verbose {
						log.Debugf("%v has received a traffic alert. Setting status to connected", w.exchangeName)
					}
					w.connected = true
				}
				w.m.Unlock()
			}
		}
	}
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

// SetWsStatusAndConnection sets if websocket is enabled
// it will also connect/disconnect the websocket connection
func (w *Websocket) SetWsStatusAndConnection(enabled bool) error {
	w.m.Lock()
	if w.enabled == enabled {
		if w.init {
			w.m.Unlock()
			return nil
		}
		w.m.Unlock()
		return fmt.Errorf("exchange_websocket.go error - already set as %t",
			enabled)
	}
	w.enabled = enabled
	if !w.init {
		if enabled {
			if w.connected {
				w.m.Unlock()
				return nil
			}
			w.m.Unlock()
			return w.Connect()
		}

		if !w.connected {
			w.m.Unlock()
			return nil
		}
		w.m.Unlock()
		return w.Shutdown()
	}
	w.m.Unlock()
	return nil
}

// IsEnabled returns bool
func (w *Websocket) IsEnabled() bool {
	return w.enabled
}

// SetProxyAddress sets websocket proxy address
func (w *Websocket) SetProxyAddress(proxyAddr string) error {
	if w.proxyAddr == proxyAddr {
		return errors.New("exchange_websocket.go error - Setting proxy address - same address")
	}

	w.proxyAddr = proxyAddr
	if !w.init && w.enabled {
		if w.connected {
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
			log.Debugf("%v ManageSubscriptions exiting", w.exchangeName)
		}
		w.Wg.Done()
	}()
	for {
		select {
		case <-w.ShutdownC:
			w.subscribedChannels = []WebsocketChannelSubscription{}
			if w.verbose {
				log.Debugf("%v shutdown manageSubscriptions", w.exchangeName)
			}
			return
		default:
			time.Sleep(manageSubscriptionsDelay)
			if w.verbose {
				log.Debugf("%v checking subscriptions", w.exchangeName)
			}
			// Subscribe to channels Pending a subscription
			if w.SupportsFunctionality(WebsocketSubscribeSupported) {
				err := w.subscribeToChannels()
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

// subscribeToChannels compares channelsToSubscribe to subscribedChannels
// and subscribes to any channels not present in subscribedChannels
func (w *Websocket) subscribeToChannels() error {
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
				log.Debugf("%v Subscribing to %v %v", w.exchangeName, w.channelsToSubscribe[i].Channel, w.channelsToSubscribe[i].Currency.String())
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
	w.noConnectionChecks = 0
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
	return nil
}

// SendMessage the one true message request. Sends message to WS
func (w *WebsocketConnection) SendMessage(data interface{}) error {
	w.Lock()
	defer w.Unlock()
	json, err := common.JSONEncode(data)
	if err != nil {
		return err
	}
	if w.Verbose {
		log.Debugf("%v sending message to websocket %v", w.ExchangeName, string(json))
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

// ReadMessage reads messages, can handle text, gzip and binary
func (w *WebsocketConnection) ReadMessage() (WebsocketResponse, error) {
	mType, resp, err := w.Connection.ReadMessage()
	if err != nil {
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
		log.Debugf("%v Websocket message received: %v",
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
