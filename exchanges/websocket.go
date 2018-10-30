package exchange

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// WebsocketInit initialises the websocket struct
func (e *Base) WebsocketInit() {
	e.Websocket = &Websocket{
		defaultURL: "",
		enabled:    false,
		proxyAddr:  "",
		runningURL: "",
		init:       true,
	}
}

// WebsocketSetup sets main variables for websocket connection
func (e *Base) WebsocketSetup(connector func() error,
	subscriber func(channelToSubscribe WebsocketChannelSubscription) error,
	unsubscriber func(channelToUnsubscribe WebsocketChannelSubscription) error,
	exchangeName string,
	wsEnabled,
	verbose bool,
	defaultURL,
	runningURL string) error {

	e.Websocket.DataHandler = make(chan interface{}, 1)
	e.Websocket.Connected = make(chan struct{}, 1)
	e.Websocket.Disconnected = make(chan struct{}, 1)
	e.Websocket.TrafficAlert = make(chan struct{}, 1)
	e.Websocket.verbose = verbose

	e.Websocket.SetChannelSubscriber(subscriber)
	e.Websocket.SetChannelUnsubscriber(unsubscriber)
	err := e.Websocket.SetWsStatusAndConnection(wsEnabled)
	if err != nil {
		return err
	}
	e.Websocket.SetDefaultURL(defaultURL)
	e.Websocket.SetConnector(connector)
	e.Websocket.SetWebsocketURL(runningURL)
	e.Websocket.SetExchangeName(exchangeName)

	e.Websocket.init = false
	e.Websocket.noConnectionCheckLimit = 5
	e.Websocket.reconnectionLimit = 10

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
		go w.wsConnectionMonitor()
	}
	go w.manageSubscriptions()

	return nil
}

// WsConnectionMonitor ensures that the WS keeps connecting
func (w *Websocket) wsConnectionMonitor() {
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
			w.DataHandler <- fmt.Errorf("%v WsConnectionMonitor: websocket disabled, shutting down", w.exchangeName)
			err := w.Shutdown()
			if err != nil {
				log.Error(err)
			}
			if w.verbose {
				log.Debugf("%v WsConnectionMonitor exiting", w.exchangeName)
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
func (w *Websocket) WebsocketReset() error {
	err := w.Shutdown()
	if err != nil {
		// does not return here to allow connection to be made if already shut down
		log.Errorf("%v shutdown error: %v", w.exchangeName, err)
	}
	log.Infof("%v reconnecting to websocket", w.exchangeName)
	w.m.Lock()
	w.init = true
	w.m.Unlock()
	err = w.Connect()
	if err != nil {
		log.Errorf("%v connection error: %v", w.exchangeName, err)
	}
	return err
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
			if w.verbose {
				log.Debugf("%v received a traffic alert", w.exchangeName)
			}
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

// Update updates a local cache using bid targets and ask targets then updates
// main cache in orderbook.go
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; amend volume of price target
func (w *WebsocketOrderbookLocal) Update(bidTargets, askTargets []orderbook.Item,
	p currency.Pair,
	updated time.Time,
	exchName string, assetType assets.AssetType) error {
	if bidTargets == nil && askTargets == nil {
		return errors.New("exchange.go websocket orderbook cache Update() error - cannot have bids and ask targets both nil")
	}

	if w.lastUpdated.After(updated) {
		return errors.New("exchange.go WebsocketOrderbookLocal Update() - update is before last update time")
	}

	w.m.Lock()
	defer w.m.Unlock()

	var orderbookAddress *orderbook.Base
	for i := range w.ob {
		if w.ob[i].Pair == p && w.ob[i].AssetType == assetType {
			orderbookAddress = w.ob[i]
		}
	}

	if orderbookAddress == nil {
		return fmt.Errorf("exchange.go WebsocketOrderbookLocal Update() - orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			exchName,
			p.String(),
			assetType)
	}

	if len(orderbookAddress.Asks) == 0 || len(orderbookAddress.Bids) == 0 {
		return errors.New("exchange.go websocket orderbook cache Update() error - snapshot incorrectly loaded")
	}

	if orderbookAddress.Pair == (currency.Pair{}) {
		return fmt.Errorf("exchange.go websocket orderbook cache Update() error - snapshot not found %v",
			p)
	}

	for x := range bidTargets {
		// bid targets
		func() {
			for y := range orderbookAddress.Bids {
				if orderbookAddress.Bids[y].Price == bidTargets[x].Price {
					if bidTargets[x].Amount == 0 {
						// Delete
						orderbookAddress.Bids = append(orderbookAddress.Bids[:y],
							orderbookAddress.Bids[y+1:]...)
						return
					}
					// Amend
					orderbookAddress.Bids[y].Amount = bidTargets[x].Amount
					return
				}
			}

			if bidTargets[x].Amount == 0 {
				// Makes sure we dont append things we missed
				return
			}

			// Append
			orderbookAddress.Bids = append(orderbookAddress.Bids, orderbook.Item{
				Price:  bidTargets[x].Price,
				Amount: bidTargets[x].Amount,
			})
		}()
		// bid targets
	}

	for x := range askTargets {
		func() {
			for y := range orderbookAddress.Asks {
				if orderbookAddress.Asks[y].Price == askTargets[x].Price {
					if askTargets[x].Amount == 0 {
						// Delete
						orderbookAddress.Asks = append(orderbookAddress.Asks[:y],
							orderbookAddress.Asks[y+1:]...)
						return
					}
					// Amend
					orderbookAddress.Asks[y].Amount = askTargets[x].Amount
					return
				}
			}

			if askTargets[x].Amount == 0 {
				// Makes sure we dont append things we missed
				return
			}

			// Append
			orderbookAddress.Asks = append(orderbookAddress.Asks, orderbook.Item{
				Price:  askTargets[x].Price,
				Amount: askTargets[x].Amount,
			})
		}()
	}

	return orderbookAddress.Process()

}

// LoadSnapshot loads initial snapshot of orderbook data, overite allows full
// orderbook to be completely rewritten because the exchange is a doing a full
// update not an incremental one
func (w *WebsocketOrderbookLocal) LoadSnapshot(newOrderbook *orderbook.Base, exchName string, overwrite bool) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return errors.New("exchange.go websocket orderbook cache LoadSnapshot() error - snapshot ask and bids are nil")
	}

	w.m.Lock()
	defer w.m.Unlock()

	for i := range w.ob {
		if w.ob[i].Pair.Equal(newOrderbook.Pair) && w.ob[i].AssetType == newOrderbook.AssetType {
			if overwrite {
				w.ob[i] = newOrderbook
				return newOrderbook.Process()
			}
			return errors.New("exchange.go websocket orderbook cache LoadSnapshot() error - Snapshot instance already found")
		}
	}

	w.ob = append(w.ob, newOrderbook)
	return newOrderbook.Process()
}

// UpdateUsingID updates orderbooks using specified ID
func (w *WebsocketOrderbookLocal) UpdateUsingID(bidTargets, askTargets []orderbook.Item,
	p currency.Pair,
	exchName string, assetType assets.AssetType, action string) error {
	w.m.Lock()
	defer w.m.Unlock()

	var orderbookAddress *orderbook.Base
	for i := range w.ob {
		if w.ob[i].Pair == p && w.ob[i].AssetType == assetType {
			orderbookAddress = w.ob[i]
		}
	}

	if orderbookAddress == nil {
		return fmt.Errorf("exchange.go WebsocketOrderbookLocal Update() - orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			exchName,
			assetType,
			p.String())
	}

	switch action {
	case "update":
		for _, target := range bidTargets {
			for i := range orderbookAddress.Bids {
				if orderbookAddress.Bids[i].ID == target.ID {
					orderbookAddress.Bids[i].Amount = target.Amount
					break
				}
			}
		}

		for _, target := range askTargets {
			for i := range orderbookAddress.Asks {
				if orderbookAddress.Asks[i].ID == target.ID {
					orderbookAddress.Asks[i].Amount = target.Amount
					break
				}
			}
		}

	case "delete":
		for _, target := range bidTargets {
			for i := range orderbookAddress.Bids {
				if orderbookAddress.Bids[i].ID == target.ID {
					orderbookAddress.Bids = append(orderbookAddress.Bids[:i],
						orderbookAddress.Bids[i+1:]...)
					break
				}
			}
		}

		for _, target := range askTargets {
			for i := range orderbookAddress.Asks {
				if orderbookAddress.Asks[i].ID == target.ID {
					orderbookAddress.Asks = append(orderbookAddress.Asks[:i],
						orderbookAddress.Asks[i+1:]...)
					break
				}
			}
		}

	case "insert":
		orderbookAddress.Bids = append(orderbookAddress.Bids, bidTargets...)
		orderbookAddress.Asks = append(orderbookAddress.Asks, askTargets...)
	}

	return orderbookAddress.Process()
}

// FlushCache flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *WebsocketOrderbookLocal) FlushCache() {
	w.m.Lock()
	w.ob = nil
	w.m.Unlock()
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
func (w *Websocket) manageSubscriptions() error {
	if !w.SupportsFunctionality(WebsocketSubscribeSupported) && !w.SupportsFunctionality(WebsocketUnsubscribeSupported) {
		return fmt.Errorf("%v does not support channel subscriptions, exiting ManageSubscriptions()", w.exchangeName)
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
			return nil
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
		w.channelsToSubscribe = append(w.channelsToSubscribe, channels[i])
	}
	w.noConnectionChecks = 0
}

// UnsubscribeToChannels removes supplied channels from channelsToSubscribe
func (w *Websocket) UnsubscribeToChannels(channels []WebsocketChannelSubscription) {
	for i := range channels {
		w.removeChannelToSubscribe(channels[i])
	}
}

// Equal two WebsocketChannelSubscription to determine equality
func (w *WebsocketChannelSubscription) Equal(subscribedChannel *WebsocketChannelSubscription) bool {
	return strings.EqualFold(w.Channel, subscribedChannel.Channel) &&
		strings.EqualFold(w.Currency.String(), subscribedChannel.Currency.String())
}
