package stream

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	// ErrAssetWebsocketNotFound returned when the asset does not have a websocket instance
	ErrAssetWebsocketNotFound = errors.New("asset websocket not found")

	// ErrWebsocketWrapperNotInitialized returned when the websocket wrapper is not initialized
	ErrWebsocketWrapperNotInitialized = errors.New("websocket wrapper not initialized")

	// ErrNoAssetWebsocketInstanceFound returned when no instantiated asset websocket instance is added
	ErrNoAssetWebsocketInstanceFound = errors.New("no websocket instance found")

	errFeaturesNotSet = errors.New("websocket wrapper features not set")
)

// GetName returns exchange name
func (w *WrapperWebsocket) GetName() string {
	return w.exchangeName
}

// IsEnabled returns status of enable
func (w *WrapperWebsocket) IsEnabled() bool {
	return w.enabled.Load()
}

// Connect connects to all websocket connections
func (w *WrapperWebsocket) Connect() error {
	if len(w.AssetTypeWebsockets) == 0 {
		return ErrNoAssetWebsocketInstanceFound
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

	var errs error
	var err error

	if !w.IsDataMonitorRunning() {
		w.dataMonitor()
	}
	// connectWg is added to manage the waiting and closing of asset websockets separately.
	connectWg := sync.WaitGroup{}
	for x := range w.AssetTypeWebsockets {
		connectWg.Add(1)
		w.connectedAssetTypesLocker.Lock()
		w.connectedAssetTypesFlag |= x
		w.connectedAssetTypesLocker.Unlock()
		go func(assetType asset.Item) {
			defer connectWg.Done()
			err = w.AssetTypeWebsockets[assetType].Connect()
			if err != nil {
				errs = common.AppendError(errs, err)
				w.ShutdownC <- assetType
			}
		}(x)
	}
	connectWg.Wait()
	if errs != nil {
		return errs
	}
	return nil
}

func (w *WrapperWebsocket) setDataMonitorRunning(b bool) {
	w.dataMonitorRunning.Store(b)
}

// IsDataMonitorRunning returns status of data monitor
func (w *WrapperWebsocket) IsDataMonitorRunning() bool {
	return w.dataMonitorRunning.Load()
}

// dataMonitor monitors job throughput and logs if there is a back log of data
func (w *WrapperWebsocket) dataMonitor() {
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
		dropped := 0
		for {
			select {
			case a := <-w.ShutdownC:
				w.connectedAssetTypesLocker.Lock()
				if a == asset.Empty || a > w.connectedAssetTypesFlag {
					w.connectedAssetTypesLocker.Unlock()
					return
				}
				if w.connectedAssetTypesFlag == asset.Empty {
					w.setDataMonitorRunning(false)
					w.connectedAssetTypesLocker.Unlock()
					return
				}
				ws, ok := w.AssetTypeWebsockets[a]
				if !ok {
					ws.setState(disconnectedState)
					w.connectedAssetTypesLocker.Unlock()
					break
				}
				ws.setState(disconnectedState)

				if w.connectedAssetTypesFlag&a == a {
					w.connectedAssetTypesFlag ^= a
					if w.connectedAssetTypesFlag == asset.Empty {
						w.setDataMonitorRunning(false)
						w.connectedAssetTypesLocker.Unlock()
						return
					}
				}

				w.connectedAssetTypesLocker.Unlock()
			case d := <-w.DataHandler:
				select {
				case w.ToRoutine <- d:
					if dropped != 0 {
						log.Infof(log.WebsocketMgr, "%s exchange websocket ToRoutine channel buffer recovered; %d messages were dropped", w.exchangeName, dropped)
						dropped = 0
					}
				case a := <-w.ShutdownC:
					w.connectedAssetTypesLocker.Lock()
					if a == asset.Empty || a > w.connectedAssetTypesFlag {
						w.connectedAssetTypesLocker.Unlock()
						return
					}
					if w.connectedAssetTypesFlag == asset.Empty {
						w.setDataMonitorRunning(false)
						w.connectedAssetTypesLocker.Unlock()
						return
					}
					if ws, ok := w.AssetTypeWebsockets[a]; ok {
						ws.setState(disconnectedState)
						w.connectedAssetTypesLocker.Unlock()
						break
					}
					if a != asset.Empty && w.connectedAssetTypesFlag&a == a {
						w.connectedAssetTypesFlag ^= a
						if w.connectedAssetTypesFlag == asset.Empty {
							w.setDataMonitorRunning(false)
							w.connectedAssetTypesLocker.Unlock()
							return
						}
					}
					w.connectedAssetTypesLocker.Unlock()
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

// FlushChannels flushes channel subscriptions when there is a pair/asset change
func (w *WrapperWebsocket) FlushChannels() error {
	if len(w.AssetTypeWebsockets) == 0 {
		return ErrAssetWebsocketNotFound
	}
	var errs error
	for x := range w.AssetTypeWebsockets {
		err := w.AssetTypeWebsockets[x].FlushChannels()
		if err != nil {
			errs = common.AppendError(errs, err)
		}
	}
	return errs
}

// GetSubscriptions returns a copied list of subscriptions of all asset type websocket connections
// and is a private member that cannot be manipulated
func (w *WrapperWebsocket) GetSubscriptions() subscription.List {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	subscriptions := subscription.List{}
	for x := range w.AssetTypeWebsockets {
		subscriptions = append(subscriptions, w.AssetTypeWebsockets[x].GetSubscriptions()...)
	}
	return subscriptions
}

// SubscribeToChannels appends supplied channels to channelsToSubscribe
func (w *WrapperWebsocket) SubscribeToChannels(channels subscription.List) error {
	if len(channels) == 0 {
		return fmt.Errorf("%s websocket: cannot subscribe no channels supplied",
			w.exchangeName)
	}
	var err error
	var filteredChannels subscription.List
	for x := range w.AssetTypeWebsockets {
		filteredChannels, err = w.AssetTypeWebsockets[x].SubscriptionFilter(channels, x)
		if err != nil {
			return err
		}
		err = w.AssetTypeWebsockets[x].SubscribeToChannels(filteredChannels)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAssetWebsocket returns a websocket connection for an asset type
func (w *WrapperWebsocket) GetAssetWebsocket(assetType asset.Item) (*Websocket, error) {
	websocket, okay := w.AssetTypeWebsockets[assetType]
	if !okay {
		return nil, fmt.Errorf("%w asset type: '%v'", ErrAssetWebsocketNotFound, assetType)
	}
	return websocket, nil
}

// Enable enables the exchange websocket protocol
func (w *WrapperWebsocket) Enable() error {
	if w.IsConnected() || w.IsEnabled() {
		return fmt.Errorf("websocket is already enabled for exchange %s",
			w.exchangeName)
	}

	w.setEnabled(true)
	return w.Connect()
}

// UnsubscribeChannels unsubscribes from a websocket channel
func (w *WrapperWebsocket) UnsubscribeChannels(channels subscription.List) error {
	if len(channels) == 0 {
		return fmt.Errorf("%s websocket: channels not populated cannot remove",
			w.exchangeName)
	}
	var err error
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].UnsubscribeChannels(channels)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetProxyAddress returns the current websocket proxy
// MUST
func (w *WrapperWebsocket) GetProxyAddress() string {
	return w.proxyAddr
}

// Setup sets main variables for websocket connection
func (w *WrapperWebsocket) Setup(s *WebsocketWrapperSetup) error {
	if w == nil {
		return errWebsocketWrapperIsNil
	}
	if s == nil {
		return errWebsocketSetupIsNil
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
	w.enabled.Store(s.ExchangeConfig.Features.Enabled.Websocket)
	w.connectionMonitorDelay = s.ConnectionMonitorDelay
	if w.connectionMonitorDelay <= 0 {
		w.connectionMonitorDelay = config.DefaultConnectionMonitorDelay
	}
	if s.ExchangeConfig.WebsocketTrafficTimeout < time.Second {
		return fmt.Errorf("%s %w cannot be less than %s",
			w.exchangeName,
			errInvalidTrafficTimeout,
			time.Second)
	}
	w.trafficTimeout = s.ExchangeConfig.WebsocketTrafficTimeout
	w.Wg = new(sync.WaitGroup)
	w.SetCanUseAuthenticatedEndpoints(s.ExchangeConfig.API.AuthenticatedWebsocketSupport)
	if err := w.Orderbook.Setup(s.ExchangeConfig, &s.OrderbookBufferConfig, w.ToRoutine); err != nil {
		return err
	}
	w.Trade.Setup(w.exchangeName, s.TradeFeed, w.ToRoutine)
	w.Fills.Setup(s.FillsFeed, w.ToRoutine)
	return nil
}

// SetCanUseAuthenticatedEndpoints sets canUseAuthenticatedEndpoints val in
// a thread safe manner
// if asset types set the value will affect only the provided asset types
func (w *WrapperWebsocket) SetCanUseAuthenticatedEndpoints(val bool, assetTypes ...asset.Item) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	if len(assetTypes) > 0 {
		for a := range assetTypes {
			assetWs, okay := w.AssetTypeWebsockets[assetTypes[a]]
			if okay {
				assetWs.SetCanUseAuthenticatedEndpoints(val)
			}
		}
	}
	w.canUseAuthenticatedEndpoints = val
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *WrapperWebsocket) CanUseAuthenticatedEndpoints(assetTypes ...asset.Item) bool {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	if len(assetTypes) > 0 && assetTypes[0] != asset.Empty {
		assetWs, okay := w.AssetTypeWebsockets[assetTypes[0]]
		if okay {
			return assetWs.CanUseAuthenticatedEndpoints()
		}
	}
	return w.canUseAuthenticatedEndpoints
}

// GetWebsocketURL returns the running websocket URL
func (w *WrapperWebsocket) GetWebsocketURL() string {
	return w.runningURL
}

// SetProxyAddress sets websocket proxy address
func (w *WrapperWebsocket) SetProxyAddress(proxyAddr string) error {
	var err error
	if proxyAddr != "" {
		_, err = url.ParseRequestURI(proxyAddr)
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
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].SetProxyAddress(proxyAddr)
		if err != nil {
			return err
		}
	}
	w.proxyAddr = proxyAddr
	return nil
}

// IsConnected returns status of connection
func (w *WrapperWebsocket) IsConnected() bool {
	var connected bool
	for a := range w.AssetTypeWebsockets {
		connected = connected || w.AssetTypeWebsockets[a].IsConnected()
	}
	return connected
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *WrapperWebsocket) Shutdown() error {
	w.m.Lock()
	defer w.m.Unlock()
	if len(w.AssetTypeWebsockets) == 0 {
		return nil
	}
	var errs error
	var err error
	errorsChan := make(chan error, len(w.AssetTypeWebsockets))
	wg := &sync.WaitGroup{}
	for x := range w.AssetTypeWebsockets {
		// This go routined below is added to create an asynchroneous Shutdown() method call for each asset type websocket instance
		// so that we can lower the delay incured by the synchronized Shutdown() method call of closing.
		wg.Add(1)
		go func(errChan chan error, ws *Websocket) {
			defer wg.Done()
			err = ws.Shutdown()
			if err != nil && !errors.Is(err, errDisconnectedConnectionShutdown) {
				errChan <- err
			}
		}(errorsChan, w.AssetTypeWebsockets[x])
	}
	wg.Wait()
	close(errorsChan)
	for x := range errorsChan {
		if x != nil {
			errs = common.AppendError(errs, err)
		}
	}
	if errs != nil {
		log.Errorf(log.WebsocketMgr,
			"%v websocket: error while shutting down asset websocket connections %v\n",
			w.exchangeName, errs)
	}
	w.connectedAssetTypesLocker.Lock()
	if w.connectedAssetTypesFlag != asset.Empty {
		select {
		case w.ShutdownC <- w.connectedAssetTypesFlag:
		default:
			if w.verbose {
				log.Errorf(log.ExchangeSys, "%s sending message to a shutdown message to a closed channel", w.exchangeName)
			}
		}
	}
	w.connectedAssetTypesLocker.Unlock()
	close(w.ShutdownC)
	w.Wg.Wait()
	w.ShutdownC = make(chan asset.Item)
	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v websocket: completed websocket shutdown\n",
			w.exchangeName)
	}
	return errs
}

// IsConnecting returns status of connecting
func (w *WrapperWebsocket) IsConnecting() bool {
	if w.IsConnected() {
		return false
	}
	var connecting bool
	for a := range w.AssetTypeWebsockets {
		connecting = connecting || w.AssetTypeWebsockets[a].IsConnecting()
	}
	return connecting
}

func (w *WrapperWebsocket) setEnabled(b bool) {
	for _, ws := range w.AssetTypeWebsockets {
		ws.enabled.Store(b)
	}
}

// Disable disables the exchange websocket protocol
func (w *WrapperWebsocket) Disable() error {
	if !w.IsEnabled() {
		return fmt.Errorf("websocket is already disabled for exchange %s",
			w.exchangeName)
	}

	w.setEnabled(false)
	return nil
}

// SetWebsocketURL sets websocket URL and can refresh underlying connections
func (w *WrapperWebsocket) SetWebsocketURL(url string, auth, reconnect bool) error {
	var err error
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].SetWebsocketURL(url, auth, reconnect)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddWebsocket creates a websocket instance for a specified asset type
func (w *WrapperWebsocket) AddWebsocket(s *WebsocketSetup) (*Websocket, error) {
	if w == nil {
		return nil, errWebsocketWrapperIsNil
	}
	if s == nil {
		return nil, errWebsocketSetupIsNil
	}
	if s.AssetType == asset.Empty || !s.AssetType.IsValid() {
		return nil, asset.ErrNotSupported
	}
	ws, okay := w.AssetTypeWebsockets[s.AssetType]
	if okay && ws.IsInitialised() {
		return ws, fmt.Errorf("%s %w", w.exchangeName, errWebsocketAlreadyInitialised)
	}
	if s.Connector == nil {
		return nil, fmt.Errorf("%s %w", w.exchangeName, errWebsocketConnectorUnset)
	}
	if s.GenerateSubscriptions == nil {
		return nil, fmt.Errorf("%s %w", w.exchangeName, errWebsocketSubscriptionsGeneratorUnset)
	}
	if w.features == nil {
		return nil, errFeaturesNotSet
	}
	if w.features.Subscribe && s.Subscriber == nil {
		return nil, fmt.Errorf("%s %w", w.exchangeName, errWebsocketSubscriberUnset)
	}
	if w.features.Unsubscribe && s.Unsubscriber == nil {
		return nil, fmt.Errorf("%s %w", w.exchangeName, errWebsocketUnsubscriberUnset)
	}
	if s.DefaultURL == "" {
		return nil, fmt.Errorf("%s websocket %w", w.exchangeName, errDefaultURLIsEmpty)
	}
	if s.RunningURL == "" {
		return nil, fmt.Errorf("%s websocket %w", w.exchangeName, errRunningURLIsEmpty)
	}
	connectionMonitorDelay := w.connectionMonitorDelay
	if w.connectionMonitorDelay <= 0 {
		connectionMonitorDelay = config.DefaultConnectionMonitorDelay
	}
	assetWebsocket := &Websocket{
		Subscribe:              make(chan subscription.List),
		Unsubscribe:            make(chan subscription.List),
		GenerateSubs:           s.GenerateSubscriptions,
		Subscriber:             s.Subscriber,
		Unsubscriber:           s.Unsubscriber,
		Wg:                     sync.WaitGroup{},
		subscriptions:          subscription.NewStore(),
		DataHandler:            w.DataHandler,
		TrafficAlert:           w.TrafficAlert,
		ReadMessageErrors:      w.ReadMessageErrors,
		Match:                  w.Match,
		trafficTimeout:         w.trafficTimeout,
		connectionMonitorDelay: connectionMonitorDelay,
		defaultURL:             s.DefaultURL,
		exchangeName:           w.exchangeName,
		verbose:                w.verbose,
		connector:              s.Connector,
		features:               w.features,
		runningURLAuth:         s.RunningURLAuth,
		ShutdownC:              make(chan struct{}),
		AssetShutdownC:         w.ShutdownC,
		AssetType:              s.AssetType,
	}
	assetWebsocket.enabled.Store(w.enabled.Load())
	assetWebsocket.SetCanUseAuthenticatedEndpoints(w.canUseAuthenticatedEndpoints)
	err := assetWebsocket.SetWebsocketURL(s.RunningURL, false, false)
	if err != nil {
		return nil, fmt.Errorf("%s %w", w.exchangeName, err)
	}

	if s.RunningURLAuth != "" {
		err = assetWebsocket.SetWebsocketURL(s.RunningURLAuth, true, false)
		if err != nil {
			return nil, fmt.Errorf("%s %w", w.exchangeName, err)
		}
	}
	if s.MaxWebsocketSubscriptionsPerConnection < 0 {
		return nil, fmt.Errorf("%s %w", w.exchangeName, errInvalidMaxSubscriptions)
	}
	assetWebsocket.MaxSubscriptionsPerConnection = s.MaxWebsocketSubscriptionsPerConnection
	assetWebsocket.setState(disconnectedState)

	w.AssetTypeWebsockets[s.AssetType] = assetWebsocket
	return w.AssetTypeWebsockets[s.AssetType], nil
}

// TODO: ...
func (w *WrapperWebsocket) CanUseAuthenticatedWebsocketForWrapper() bool {
	return false
}
