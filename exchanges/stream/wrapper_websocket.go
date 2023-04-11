package stream

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	ErrNoAssetTypeConnection = errors.New("no websocket instance found for asset type")
)

// GetName ...
func (w *WrapperWebsocket) GetName() string {
	return w.exchangeName
}

// IsEnabled ...
func (w *WrapperWebsocket) IsEnabled() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.enabled
}

// Connect connects to all websocket connections
func (w *WrapperWebsocket) Connect() error {
	if len(w.AssetTypeWebsockets) == 0 {
		return ErrNoAssetTypeConnection
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
	var err error
	for x := range w.AssetTypeWebsockets {
		println(x.String())
		err = w.AssetTypeWebsockets[x].Connect()
		if err != nil {
			w.setConnectingStatus(false)
			return fmt.Errorf("%s Error connecting %v",
				w.exchangeName, err)
		}
	}
	w.setConnectedStatus(true)
	w.setConnectingStatus(false)
	w.setInit(true)
	return nil
}

// FlushChannels ...
func (w *WrapperWebsocket) FlushChannels() error {
	var err error
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].FlushChannels()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetSubscriptions calls
func (w *WrapperWebsocket) GetSubscriptions() []ChannelSubscription {
	w.subscriptionMutex.Lock()
	defer w.subscriptionMutex.Unlock()
	// 3- index slicing
	return append(w.subscriptions[:0:0], w.subscriptions...)
}

// SubscribeToChannels ...
func (w *WrapperWebsocket) SubscribeToChannels(channels []ChannelSubscription) error {
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

// GetAssetWebsocket ...
func (w *WrapperWebsocket) GetAssetWebsocket(assetType asset.Item) (*Websocket, error) {
	websocket, okay := w.AssetTypeWebsockets[assetType]
	if !okay {
		return nil, fmt.Errorf("%w, asset type: %v", ErrNoAssetTypeConnection, assetType)
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
		return errWebsocketIsNil
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
	if err := w.Orderbook.Setup(s.ExchangeConfig, &s.OrderbookBufferConfig, w.DataHandler); err != nil {
		return err
	}
	w.Trade.Setup(w.exchangeName, s.TradeFeed, w.DataHandler)
	w.Fills.Setup(s.FillsFeed, w.DataHandler)
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
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].SetProxyAddress(proxyAddr)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsConnected returns status of connection
func (w *WrapperWebsocket) IsConnected() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.connected
}

// Shutdown attempts to shut down a websocket connection and associated routines
// by using a package defined shutdown function
func (w *WrapperWebsocket) Shutdown() error {
	w.m.Lock()
	defer w.m.Unlock()
	var err error
	for x := range w.AssetTypeWebsockets {
		err = w.AssetTypeWebsockets[x].Shutdown()
		if err != nil && !errors.Is(err, errDisconnectedConnectionShutdown) {
			return err
		}
	}
	// flush any subscriptions from last connection if needed
	w.subscriptionMutex.Lock()
	w.subscriptions = nil
	w.subscriptionMutex.Unlock()

	w.setConnectedStatus(false)
	w.setConnectingStatus(false)
	if w.verbose {
		log.Debugf(log.WebsocketMgr,
			"%v websocket: completed websocket shutdown\n",
			w.exchangeName)
	}
	return nil
}

// IsConnecting returns status of connecting
func (w *WrapperWebsocket) IsConnecting() bool {
	w.connectionMutex.RLock()
	defer w.connectionMutex.RUnlock()
	return w.connecting
}

func (w *WrapperWebsocket) setConnectedStatus(b bool) {
	w.connectionMutex.Lock()
	w.connected = b
	w.connectionMutex.Unlock()
}

func (w *WrapperWebsocket) setConnectingStatus(b bool) {
	w.connectionMutex.Lock()
	w.connecting = b
	w.connectionMutex.Unlock()
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
		err = w.AssetTypeWebsockets[x].SetWebsocketURL(url, false, false)
		if err != nil {
			return err
		}
	}
	return nil
}

// AddWebsocket creates a websocket instance for a specified asset type
func (wr *WrapperWebsocket) AddWebsocket(s *WebsocketSetup) (*Websocket, error) {
	if wr == nil {
		return nil, errWebsocketIsNil
	}
	w, okay := wr.AssetTypeWebsockets[s.AssetType]
	if okay && w != nil {
		return w, fmt.Errorf("%s %w", wr.exchangeName, errWebsocketAlreadyInitialised)
	}
	if s == nil {
		return nil, errWebsocketSetupIsNil
	}
	if s.Connector == nil {
		return nil, fmt.Errorf("%s %w", wr.exchangeName, errWebsocketConnectorUnset)
	}
	if s.Subscriber == nil {
		return nil, fmt.Errorf("%s %w", wr.exchangeName, errWebsocketSubscriberUnset)
	}
	if /*w.features.Unsubscribe &&*/ s.Unsubscriber == nil {
		return nil, fmt.Errorf("%s %w", wr.exchangeName, errWebsocketUnsubscriberUnset)
	}
	connectionMonitorDelay := wr.connectionMonitorDelay
	if wr.connectionMonitorDelay <= 0 {
		connectionMonitorDelay = config.DefaultConnectionMonitorDelay
	}
	if s.GenerateSubscriptions == nil {
		return nil, fmt.Errorf("%s %w", wr.exchangeName, errWebsocketSubscriptionsGeneratorUnset)
	}
	if s.SubscriptionFilter == nil {
		return nil, fmt.Errorf("%s %v %w", wr.exchangeName, s.AssetType, errWebsocketSubscriptionFilterUnset)
	}
	if s.DefaultURL == "" {
		return nil, fmt.Errorf("%s websocket %w", wr.exchangeName, errDefaultURLIsEmpty)
	}
	if s.RunningURL == "" {
		return nil, fmt.Errorf("%s websocket %w", wr.exchangeName, errRunningURLIsEmpty)
	}
	var err error
	if s.RunningURLAuth != "" {
		err = w.SetWebsocketURL(s.RunningURLAuth, true, false)
		if err != nil {
			return nil, fmt.Errorf("%s %w", wr.exchangeName, err)
		}
	}
	wr.AssetTypeWebsockets[s.AssetType] = &Websocket{
		Init:                   true,
		DataHandler:            make(chan interface{}),
		Subscribe:              make(chan []ChannelSubscription),
		Unsubscribe:            make(chan []ChannelSubscription),
		ShutdownC:              make(chan struct{}),
		GenerateSubs:           s.GenerateSubscriptions,
		Subscriber:             s.Subscriber,
		Unsubscriber:           s.Unsubscriber,
		Wg:                     new(sync.WaitGroup),
		ToRoutine:              wr.ToRoutine,
		TrafficAlert:           wr.TrafficAlert,
		ReadMessageErrors:      wr.ReadMessageErrors,
		Match:                  wr.Match,
		trafficTimeout:         wr.trafficTimeout,
		connectionMonitorDelay: connectionMonitorDelay,
		SubscriptionFilter:     s.SubscriptionFilter,
		defaultURL:             s.DefaultURL,
		exchangeName:           wr.exchangeName,
		verbose:                wr.verbose,
		enabled:                wr.enabled,
		connector:              s.Connector,
		features:               wr.features,
	}
	err = wr.AssetTypeWebsockets[s.AssetType].SetWebsocketURL(s.RunningURL, false, false)
	if err != nil {
		return nil, fmt.Errorf("%s %w", wr.exchangeName, err)
	}
	return wr.AssetTypeWebsockets[s.AssetType], nil
}
