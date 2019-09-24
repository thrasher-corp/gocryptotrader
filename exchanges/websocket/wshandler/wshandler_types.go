package wshandler

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wsorderbook"
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
	WebsocketAuthenticatedEndpointsSupported
	WebsocketAccountDataSupported
	WebsocketSubmitOrderSupported
	WebsocketCancelOrderSupported
	WebsocketWithdrawSupported
	WebsocketMessageCorrelationSupported
	WebsocketSequenceNumberSupported
	WebsocketDeadMansSwitchSupported

	WebsocketTickerSupportedText                 = "TICKER STREAMING SUPPORTED"
	WebsocketOrderbookSupportedText              = "ORDERBOOK STREAMING SUPPORTED"
	WebsocketKlineSupportedText                  = "KLINE STREAMING SUPPORTED"
	WebsocketTradeDataSupportedText              = "TRADE STREAMING SUPPORTED"
	WebsocketAccountSupportedText                = "ACCOUNT STREAMING SUPPORTED"
	WebsocketAllowsRequestsText                  = "WEBSOCKET REQUESTS SUPPORTED"
	NoWebsocketSupportText                       = "WEBSOCKET NOT SUPPORTED"
	UnknownWebsocketFunctionality                = "UNKNOWN FUNCTIONALITY BITMASK"
	WebsocketSubscribeSupportedText              = "WEBSOCKET SUBSCRIBE SUPPORTED"
	WebsocketUnsubscribeSupportedText            = "WEBSOCKET UNSUBSCRIBE SUPPORTED"
	WebsocketAuthenticatedEndpointsSupportedText = "WEBSOCKET AUTHENTICATED ENDPOINTS SUPPORTED"
	WebsocketAccountDataSupportedText            = "WEBSOCKET ACCOUNT DATA SUPPORTED"
	WebsocketSubmitOrderSupportedText            = "WEBSOCKET SUBMIT ORDER SUPPORTED"
	WebsocketCancelOrderSupportedText            = "WEBSOCKET CANCEL ORDER SUPPORTED"
	WebsocketWithdrawSupportedText               = "WEBSOCKET WITHDRAW SUPPORTED"
	WebsocketMessageCorrelationSupportedText     = "WEBSOCKET MESSAGE CORRELATION SUPPORTED"
	WebsocketSequenceNumberSupportedText         = "WEBSOCKET SEQUENCE NUMBER SUPPORTED"
	WebsocketDeadMansSwitchSupportedText         = "WEBSOCKET DEAD MANS SWITCH SUPPORTED"
	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled      = "exchange_websocket_not_enabled"
	manageSubscriptionsDelay = 5 * time.Second
	// connection monitor time delays and limits
	connectionMonitorDelay = 2 * time.Second
)

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing in routines.go
type Websocket struct {
	// Functionality defines websocket stream capabilities
	Functionality                uint32
	canUseAuthenticatedEndpoints bool
	enabled                      bool
	init                         bool
	connected                    bool
	connecting                   bool
	verbose                      bool
	connectionMonitorRunning     bool
	trafficTimeout               time.Duration
	proxyAddr                    string
	defaultURL                   string
	runningURL                   string
	exchangeName                 string
	m                            sync.Mutex
	subscriptionLock             sync.Mutex
	connectionMutex              sync.RWMutex
	connector                    func() error
	subscribedChannels           []WebsocketChannelSubscription
	channelsToSubscribe          []WebsocketChannelSubscription
	channelSubscriber            func(channelToSubscribe WebsocketChannelSubscription) error
	channelUnsubscriber          func(channelToUnsubscribe WebsocketChannelSubscription) error
	DataHandler                  chan interface{}
	// ShutdownC is the main shutdown channel which controls all websocket go funcs
	ShutdownC chan struct{}
	// Orderbook is a local cache of orderbooks
	Orderbook wsorderbook.WebsocketOrderbookLocal
	// Wg defines a wait group for websocket routines for cleanly shutting down
	// routines
	Wg sync.WaitGroup
	// TrafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
	// ReadMessageErrors will received all errors from ws.ReadMessage() and verify if its a disconnection
	ReadMessageErrors chan error
}

type WebsocketSetup struct {
	WsEnabled                        bool
	Verbose                          bool
	AuthenticatedWebsocketAPISupport bool
	WebsocketTimeout                 time.Duration
	DefaultURL                       string
	ExchangeName                     string
	RunningURL                       string
	Connector                        func() error
	Subscriber                       func(channelToSubscribe WebsocketChannelSubscription) error
	UnSubscriber                     func(channelToUnsubscribe WebsocketChannelSubscription) error
}

// WebsocketChannelSubscription container for websocket subscriptions
// Currently only a one at a time thing to avoid complexity
type WebsocketChannelSubscription struct {
	Channel  string
	Currency currency.Pair
	Params   map[string]interface{}
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
	Asset    asset.Item
	Exchange string
}

// TradeData defines trade data
type TradeData struct {
	Timestamp    time.Time
	CurrencyPair currency.Pair
	AssetType    asset.Item
	Exchange     string
	EventType    string
	EventTime    int64
	Price        float64
	Amount       float64
	Side         string
}

// TickerData defines ticker feed
type TickerData struct {
	Exchange    string
	Open        float64
	Close       float64
	Volume      float64
	QuoteVolume float64
	High        float64
	Low         float64
	Bid         float64
	Ask         float64
	Last        float64
	PriceATH    float64
	Timestamp   time.Time
	AssetType   asset.Item
	Pair        currency.Pair
}

// KlineData defines kline feed
type KlineData struct {
	Timestamp  time.Time
	Pair       currency.Pair
	AssetType  asset.Item
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
	AssetType asset.Item
	Exchange  string
}

// WebsocketConnection contains all the data needed to send a message to a WS
type WebsocketConnection struct {
	sync.Mutex
	Verbose         bool
	connected       bool
	connectionMutex sync.RWMutex
	RateLimit       float64
	ExchangeName    string
	URL             string
	ProxyURL        string
	Wg              sync.WaitGroup
	Connection      *websocket.Conn
	Shutdown        chan struct{}
	// These are the request IDs and the corresponding response JSON
	IDResponses          map[int64][]byte
	ResponseCheckTimeout time.Duration
	ResponseMaxLimit     time.Duration
	TrafficTimeout       time.Duration
}
