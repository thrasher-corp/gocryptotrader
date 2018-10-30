package exchange

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

const (
	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled = "exchange_websocket_not_enabled"
	// WebsocketTrafficLimitTime defines a standard time for no traffic from the
	// websocket connection
	WebsocketTrafficLimitTime = 5 * time.Second
	// WebsocketStateTimeout defines a const for when a websocket connection
	// times out, will be handled by the routine management system
	WebsocketStateTimeout = "TIMEOUT"

	websocketRestablishConnection = 1 * time.Second
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
	exchangeName string,
	wsEnabled bool,
	defaultURL,
	runningURL string) error {

	e.Websocket.DataHandler = make(chan interface{}, 1)
	e.Websocket.Connected = make(chan struct{}, 1)
	e.Websocket.Disconnected = make(chan struct{}, 1)
	e.Websocket.Intercomm = make(chan WebsocketResponse, 1)
	e.Websocket.TrafficAlert = make(chan struct{}, 1)

	err := e.Websocket.SetEnabled(wsEnabled)
	if err != nil {
		return err
	}

	e.Websocket.SetDefaultURL(defaultURL)
	e.Websocket.SetConnector(connector)
	e.Websocket.SetWebsocketURL(runningURL)
	e.Websocket.SetExchangeName(exchangeName)

	e.Websocket.init = false

	return nil
}

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing in routines.go
type Websocket struct {
	proxyAddr    string
	defaultURL   string
	runningURL   string
	exchangeName string
	enabled      bool
	init         bool
	connected    bool
	connector    func() error
	m            sync.Mutex

	// Connected denotes a channel switch for diversion of request flow
	Connected chan struct{}

	// Disconnected denotes a channel switch for diversion of request flow
	Disconnected chan struct{}

	// Intercomm denotes a channel from read data routine to handle data routine
	Intercomm chan WebsocketResponse

	// DataHandler pipes websocket data to an exchange websocket data handler
	DataHandler chan interface{}

	// ShutdownC is the main shutdown channel used within an exchange package
	// called by its own defined Shutdown function
	ShutdownC chan struct{}

	// Orderbook is a local cache of orderbooks
	Orderbook WebsocketOrderbookLocal

	// Wg defines a wait group for websocket routines for cleanly shutting down
	// routines
	Wg sync.WaitGroup

	// TrafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
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
			return

		case <-w.TrafficAlert: // Resets timer on traffic
			if !w.connected {
				w.Connected <- struct{}{}
				w.connected = true
			}

			trafficTimer.Reset(WebsocketTrafficLimitTime)

		case <-trafficTimer.C: // Falls through when timer runs out
			newtimer := time.NewTimer(10 * time.Second) // New secondary timer set
			if w.connected {
				// If connected divert traffic to rest
				w.Disconnected <- struct{}{}
				w.connected = false
			}

			select {
			case <-w.ShutdownC: // Returns on shutdown channel close
				return

			case <-newtimer.C: // If secondary timer runs state timeout is sent to the data handler
				w.DataHandler <- WebsocketStateTimeout
				return

			case <-w.TrafficAlert: // If in this time response traffic comes through
				trafficTimer.Reset(WebsocketTrafficLimitTime)
				if !w.connected {
					// If not connected divert traffic from REST to websocket
					w.Connected <- struct{}{}
					w.connected = true
				}
			}
		}
	}
}

// Connect intiates a websocket connection by using a package defined connection
// function
func (w *Websocket) Connect() error {
	w.m.Lock()
	defer w.m.Unlock()

	if !w.IsEnabled() {
		return fmt.Errorf("exchange_websocket.go %s error - websocket disabled",
			w.GetName())
	}

	if w.connected {
		return errors.New("exchange_websocket.go error - already connected, cannot connect again")
	}

	w.ShutdownC = make(chan struct{}, 1)

	var anotherWG sync.WaitGroup
	anotherWG.Add(1)
	go w.trafficMonitor(&anotherWG)
	anotherWG.Wait()

	err := w.connector()
	if err != nil {
		return fmt.Errorf("exchange_websocket.go connection error %s",
			err)
	}

	// Divert for incoming websocket traffic
	w.Connected <- struct{}{}
	w.connected = true

	return nil
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *Websocket) Shutdown() error {
	w.m.Lock()

	defer func() {
		w.Orderbook.FlushCache()
		w.m.Unlock()
	}()

	if !w.connected {
		return errors.New("exchange_websocket.go error - System not connected to shut down")
	}

	timer := time.NewTimer(5 * time.Second)
	c := make(chan struct{}, 1)

	go func(c chan struct{}) {
		close(w.ShutdownC)
		w.Wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-c:
		w.connected = false
		return nil
	case <-timer.C:
		return fmt.Errorf("%s - Websocket routines failed to shutdown",
			w.GetName())
	}
}

// SetWebsocketURL sets websocket URL
func (w *Websocket) SetWebsocketURL(URL string) {
	if URL == "" || URL == config.WebsocketURLNonDefaultMessage {
		w.runningURL = w.defaultURL
		return
	}
	w.runningURL = URL
}

// GetWebsocketURL returns the running websocket URL
func (w *Websocket) GetWebsocketURL() string {
	return w.runningURL
}

// SetEnabled sets if websocket is enabled
func (w *Websocket) SetEnabled(enabled bool) error {
	if w.enabled == enabled {
		if w.init {
			return nil
		}
		return fmt.Errorf("exchange_websocket.go error - already set as %t",
			enabled)
	}

	w.enabled = enabled

	if !w.init {
		if enabled {
			if w.connected {
				return nil
			}
			return w.Connect()
		}

		if !w.connected {
			return nil
		}
		return w.Shutdown()
	}
	return nil
}

// IsEnabled returns bool
func (w *Websocket) IsEnabled() bool {
	return w.enabled
}

// SetProxyAddress sets websocket proxy address
func (w *Websocket) SetProxyAddress(URL string) error {
	if w.proxyAddr == URL {
		return errors.New("exchange_websocket.go error - Setting proxy address - same address")
	}

	w.proxyAddr = URL

	if !w.init && w.enabled {
		if w.connected {
			err := w.Shutdown()
			if err != nil {
				return err
			}
			return w.Connect()
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

// WebsocketOrderbookLocal defines a local cache of orderbooks for ammending,
// appending and deleting changes and updates the main store in orderbook.go
type WebsocketOrderbookLocal struct {
	ob          []orderbook.Base
	lastUpdated time.Time
	m           sync.Mutex
}

// Update updates a local cache using bid targets and ask targets then updates
// main cache in orderbook.go
// Volume == 0; deletion at price target
// Price target not found; append of price target
// Price target found; ammend volume of price target
func (w *WebsocketOrderbookLocal) Update(bidTargets, askTargets []orderbook.Item,
	p pair.CurrencyPair,
	updated time.Time,
	exchName, assetType string) error {
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
			orderbookAddress = &w.ob[i]
		}
	}

	if orderbookAddress == nil {
		return fmt.Errorf("exchange.go WebsocketOrderbookLocal Update() - orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			exchName,
			p.Pair().String(),
			assetType)
	}

	if len(orderbookAddress.Asks) == 0 || len(orderbookAddress.Bids) == 0 {
		return errors.New("exchange.go websocket orderbook cache Update() error - snapshot incorrectly loaded")
	}

	if orderbookAddress.Pair == (pair.CurrencyPair{}) {
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
						orderbookAddress.Asks = append(orderbookAddress.Bids[:y],
							orderbookAddress.Bids[y+1:]...)
						return
					}
					// Ammend
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
					// Ammend
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

	orderbook.ProcessOrderbook(exchName, p, *orderbookAddress, assetType)
	return nil
}

// LoadSnapshot loads initial snapshot of orderbook data
func (w *WebsocketOrderbookLocal) LoadSnapshot(newOrderbook orderbook.Base, exchName string) error {
	if len(newOrderbook.Asks) == 0 || len(newOrderbook.Bids) == 0 {
		return errors.New("exchange.go websocket orderbook cache LoadSnapshot() error - snapshot ask and bids are nil")
	}

	w.m.Lock()
	defer w.m.Unlock()

	for i := range w.ob {
		if w.ob[i].Pair == newOrderbook.Pair && w.ob[i].AssetType == newOrderbook.AssetType {
			return errors.New("exchange.go websocket orderbook cache LoadSnapshot() error - Snapshot instance already found")
		}
	}

	w.ob = append(w.ob, newOrderbook)
	w.lastUpdated = newOrderbook.LastUpdated

	orderbook.ProcessOrderbook(exchName,
		newOrderbook.Pair,
		newOrderbook,
		newOrderbook.AssetType)

	return nil
}

// UpdateUsingID updates orderbooks using specified ID
func (w *WebsocketOrderbookLocal) UpdateUsingID(bidTargets, askTargets []orderbook.Item,
	p pair.CurrencyPair,
	updated time.Time,
	exchName, assetType, action string) error {
	w.m.Lock()
	defer w.m.Unlock()

	var orderbookAddress *orderbook.Base
	for i := range w.ob {
		if w.ob[i].Pair == p && w.ob[i].AssetType == assetType {
			orderbookAddress = &w.ob[i]
		}
	}

	if orderbookAddress == nil {
		return fmt.Errorf("exchange.go WebsocketOrderbookLocal Update() - orderbook.Base could not be found for Exchange %s CurrencyPair: %s AssetType: %s",
			exchName,
			assetType,
			p.Pair().String())
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
		for _, target := range bidTargets {
			orderbookAddress.Bids = append(orderbookAddress.Bids, target)
		}

		for _, target := range askTargets {
			orderbookAddress.Asks = append(orderbookAddress.Asks, target)
		}
	}

	orderbook.ProcessOrderbook(exchName, p, *orderbookAddress, assetType)

	return nil
}

// FlushCache flushes w.ob data to be garbage collected and refreshed when a
// connection is lost and reconnected
func (w *WebsocketOrderbookLocal) FlushCache() {
	w.m.Lock()
	w.ob = nil
	w.m.Unlock()
}
