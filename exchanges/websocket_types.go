package exchange

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
)

// Websocket functionality list and state consts
const (
	NoWebsocketSupport       uint32 = 0
	WebsocketTickerSupported uint32 = 1 << (iota - 1)
	WebsocketOrderbookSupported
	WebsocketKlineSupported
	WebsocketTradeDataSupported
	WebsocketAccountSupported
	WebsocketAllowsRequests
	WebsocketSubscribeSupported
	WebsocketUnsubscribeSupported

	WebsocketTickerSupportedText      = "TICKER STREAMING SUPPORTED"
	WebsocketOrderbookSupportedText   = "ORDERBOOK STREAMING SUPPORTED"
	WebsocketKlineSupportedText       = "KLINE STREAMING SUPPORTED"
	WebsocketTradeDataSupportedText   = "TRADE STREAMING SUPPORTED"
	WebsocketAccountSupportedText     = "ACCOUNT STREAMING SUPPORTED"
	WebsocketAllowsRequestsText       = "WEBSOCKET REQUESTS SUPPORTED"
	NoWebsocketSupportText            = "WEBSOCKET NOT SUPPORTED"
	UnknownWebsocketFunctionality     = "UNKNOWN FUNCTIONALITY BITMASK"
	WebsocketSubscribeSupportedText   = "WEBSOCKET SUBSCRIBE SUPPORTED"
	WebsocketUnsubscribeSupportedText = "WEBSOCKET UNSUBSCRIBE SUPPORTED"

	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled = "exchange_websocket_not_enabled"
	// WebsocketTrafficLimitTime defines a standard time for no traffic from the
	// websocket connection
	WebsocketTrafficLimitTime     = 5 * time.Second
	websocketRestablishConnection = time.Second
	manageSubscriptionsDelay      = 5 * time.Second
	// connection monitor time delays and limits
	connectionMonitorDelay = 2 * time.Second
	// WebsocketStateTimeout defines a const for when a websocket connection
	// times out, will be handled by the routine management system
	WebsocketStateTimeout = "TIMEOUT"
)

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing in routines.go
type Websocket struct {
	proxyAddr                string
	defaultURL               string
	runningURL               string
	exchangeName             string
	enabled                  bool
	init                     bool
	connected                bool
	connecting               bool
	verbose                  bool
	connector                func() error
	m                        sync.Mutex
	subscriptionLock         sync.Mutex
	connectionMonitorRunning bool
	reconnectionLimit        int
	noConnectionChecks       int
	reconnectionChecks       int
	noConnectionCheckLimit   int
	// Subscriptions stuff
	subscribedChannels  []WebsocketChannelSubscription
	channelsToSubscribe []WebsocketChannelSubscription
	channelSubscriber   func(channelToSubscribe WebsocketChannelSubscription) error
	channelUnsubscriber func(channelToUnsubscribe WebsocketChannelSubscription) error
	// Connected denotes a channel switch for diversion of request flow
	Connected chan struct{}
	// Disconnected denotes a channel switch for diversion of request flow
	Disconnected chan struct{}
	// DataHandler pipes websocket data to an exchange websocket data handler
	DataHandler chan interface{}
	// ShutdownC is the main shutdown channel which controls all websocket go funcs
	ShutdownC                 chan struct{}
	ShutdownConnectionMonitor chan struct{}
	// Orderbook is a local cache of orderbooks
	Orderbook WebsocketOrderbookLocal

	// Wg defines a wait group for websocket routines for cleanly shutting down
	// routines
	Wg sync.WaitGroup
	// TrafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
	// Functionality defines websocket stream capabilities
	Functionality uint32
}

// WebsocketChannelSubscription container for websocket subscriptions
// Currently only a one at a time thing to avoid complexity
type WebsocketChannelSubscription struct {
	Channel  string
	Currency currency.Pair
	Params   map[string]interface{}
}

// WebsocketOrderbookLocal defines a local cache of orderbooks for amending,
// appending and deleting changes and updates the main store in orderbook.go
type WebsocketOrderbookLocal struct {
	ob          []*orderbook.Base
	lastUpdated time.Time
	m           sync.Mutex
}

// WebsocketResponse defines generalised data from the websocket connection
type WebsocketResponse struct {
	Type int
	Raw  []byte
}

// WebsocketOrderbookUpdate defines a websocket event in which the orderbook
// has been updated in the orderbook package
type WebsocketOrderbookUpdate struct {
	Pair     currency.Pair
	Asset    assets.AssetType
	Exchange string
}

// TradeData defines trade data
type TradeData struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    assets.AssetType
	Exchange     string
	EventType    string
	EventTime    int64
	Price        float64
	Amount       float64
	Side         string
}

// TickerData defines ticker feed
type TickerData struct {
	Timestamp  time.Time
	Pair       currency.Pair
	AssetType  assets.AssetType
	Exchange   string
	ClosePrice float64
	Quantity   float64
	OpenPrice  float64
	HighPrice  float64
	LowPrice   float64
}

// KlineData defines kline feed
type KlineData struct {
	Timestamp  time.Time
	Pair       currency.Pair
	AssetType  assets.AssetType
	Exchange   string
	StartTime  time.Time
	CloseTime  time.Time
	Interval   string
	OpenPrice  float64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
}

// WebsocketPositionUpdated reflects a change in orders/contracts on an exchange
type WebsocketPositionUpdated struct {
	Timestamp time.Time
	Pair      currency.Pair
	AssetType assets.AssetType
	Exchange  string
}
