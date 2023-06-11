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
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.enabled
}

// Connect connects to all websocket connections
func (w *WrapperWebsocket) Connect() error {
	if len(w.AssetTypeWebsockets) == 0 {
		return ErrNoAssetWebsocketInstanceFound
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
	var errs error
	var err error

	if !w.IsDataMonitorRunning() {
		w.dataMonitor()
	}
	for x := range w.AssetTypeWebsockets {
		w.connectedAssetTypesFlag |= x
		w.Wg.Add(1)
		go func(assetType asset.Item) {
			defer w.Wg.Done()
			err = w.AssetTypeWebsockets[assetType].Connect()
			if err != nil {
				log.Errorf(log.WebsocketMgr, "%v", err)
			}
		}(x)
	}
	if errs != nil {
		return errs
	}
	w.setInit(true)
	return nil
}

func (w *WrapperWebsocket) setDataMonitorRunning(b bool) {
	w.connectionMutex.Lock()
	w.dataMonitorRunning = b
	w.connectionMutex.Unlock()
}

// IsDataMonitorRunning returns status of data monitor
func (w *WrapperWebsocket) IsDataMonitorRunning() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.dataMonitorRunning
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
		for {
			select {
			case a := <-w.ShutdownC:
				if ws, ok := w.AssetTypeWebsockets[a]; ok {
					ws.setConnectedStatus(false)
				}
				if a != asset.Empty && w.connectedAssetTypesFlag&a == a {
					w.connectedAssetTypesFlag ^= a
					if w.connectedAssetTypesFlag == asset.Empty {
						w.setDataMonitorRunning(false)
						return
					}
				}
			case d := <-w.DataHandler:
				select {
				case w.ToRoutine <- d:
				case a := <-w.ShutdownC:
					if ws, ok := w.AssetTypeWebsockets[a]; ok {
						ws.setConnectedStatus(false)
					}
					if a != asset.Empty && w.connectedAssetTypesFlag&a == a {
						w.connectedAssetTypesFlag ^= a
						if w.connectedAssetTypesFlag == asset.Empty {
							w.setDataMonitorRunning(false)
							return
						}
					}
				default:
					log.Warnf(log.WebsocketMgr,
						"%s exchange backlog in websocket processing detected",
						w.exchangeName)
					w.ToRoutine <- d
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
func (w *WrapperWebsocket) GetSubscriptions() []ChannelSubscription {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	subscriptions := []ChannelSubscription{}
	for x := range w.AssetTypeWebsockets {
		subscriptions = append(subscriptions, w.AssetTypeWebsockets[x].GetSubscriptions()...)
	}
	return subscriptions
}

// SubscribeToChannels appends supplied channels to channelsToSubscribe
func (w *WrapperWebsocket) SubscribeToChannels(channels []ChannelSubscription) error {
	if len(channels) == 0 {
		return fmt.Errorf("%s websocket: cannot subscribe no channels supplied",
			w.exchangeName)
	}
	var err error
	var filteredChannels []ChannelSubscription
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
func (w *WrapperWebsocket) UnsubscribeChannels(channels []ChannelSubscription) error {
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
	if !w.Init {
		return fmt.Errorf("%s %w", w.exchangeName, errWebsocketAlreadyInitialised)
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
	w.enabled = s.ExchangeConfig.Features.Enabled.Websocket
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
func (w *WrapperWebsocket) SetCanUseAuthenticatedEndpoints(val bool) {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	w.canUseAuthenticatedEndpoints = val
}

// CanUseAuthenticatedEndpoints gets canUseAuthenticatedEndpoints val in
// a thread safe manner
func (w *WrapperWebsocket) CanUseAuthenticatedEndpoints() bool {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
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
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
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
	if w.connectedAssetTypesFlag != asset.Empty {
		w.ShutdownC <- w.connectedAssetTypesFlag
	}
	var errs error
	var err error
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].Shutdown()
		if err != nil && !errors.Is(err, errDisconnectedConnectionShutdown) {
			errs = common.AppendError(errs, err)
		}
	}
	if errs != nil {
		return errs
	}
	close(w.ShutdownC)
	w.Wg.Wait()
	w.ShutdownC = make(chan asset.Item)
	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v websocket: completed websocket shutdown\n",
			w.exchangeName)
	}
	return nil
}

// IsConnecting returns status of connecting
func (w *WrapperWebsocket) IsConnecting() bool {
	if w.IsConnected() {
		return false
	}
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	var connecting bool
	for a := range w.AssetTypeWebsockets {
		connecting = connecting || w.AssetTypeWebsockets[a].IsConnecting()
	}
	return connecting
}

// IsInit returns status of init
func (w *WrapperWebsocket) IsInit() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.Init
}

func (w *WrapperWebsocket) setInit(b bool) {
	w.connectionMutex.Lock()
	w.Init = b
	w.connectionMutex.Unlock()
}

func (w *WrapperWebsocket) setEnabled(b bool) {
	w.connectionMutex.Lock()
	w.enabled = b
	w.connectionMutex.Unlock()
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
	if !w.Init {
		return nil, ErrWebsocketWrapperNotInitialized
	}
	if s.AssetType == asset.Empty || !s.AssetType.IsValid() {
		return nil, asset.ErrNotSupported
	}
	ws, okay := w.AssetTypeWebsockets[s.AssetType]
	if okay && ws != nil {
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
		Init:                   true,
		Subscribe:              make(chan []ChannelSubscription),
		Unsubscribe:            make(chan []ChannelSubscription),
		GenerateSubs:           s.GenerateSubscriptions,
		Subscriber:             s.Subscriber,
		Unsubscriber:           s.Unsubscriber,
		Wg:                     w.Wg,
		DataHandler:            w.DataHandler,
		ToRoutine:              w.ToRoutine,
		TrafficAlert:           w.TrafficAlert,
		ReadMessageErrors:      w.ReadMessageErrors,
		Match:                  w.Match,
		trafficTimeout:         w.trafficTimeout,
		connectionMonitorDelay: connectionMonitorDelay,
		defaultURL:             s.DefaultURL,
		exchangeName:           w.exchangeName,
		verbose:                w.verbose,
		enabled:                w.enabled,
		connector:              s.Connector,
		features:               w.features,
		runningURLAuth:         s.RunningURLAuth,
		ShutdownC:              make(chan struct{}),
		AssetShutdownC:         w.ShutdownC,
		AssetType:              s.AssetType,
	}
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
	w.AssetTypeWebsockets[s.AssetType] = assetWebsocket
	return w.AssetTypeWebsockets[s.AssetType], nil
}
