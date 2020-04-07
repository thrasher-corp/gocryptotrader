package wshandler

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/wsorderbook"
)

// Websocket functionality list and state consts
const (
	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled      = "exchange_websocket_not_enabled"
	manageSubscriptionsDelay = 5 * time.Second
	// connection monitor time delays and limits
	connectionMonitorDelay             = 2 * time.Second
	WebsocketNotAuthenticatedUsingRest = "%v - Websocket not authenticated, using REST"
	Ping                               = "ping"
	Pong                               = "pong"
	UnhandledMessage                   = " - Unhandled websocket message: "
)

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing in routines.go
type Websocket struct {
	canUseAuthenticatedEndpoints bool
	enabled                      bool
	init                         bool
	connected                    bool
	connecting                   bool
	trafficMonitorRunning        bool
	verbose                      bool
	connectionMonitorRunning     bool
	trafficTimeout               time.Duration
	proxyAddr                    string
	defaultURL                   string
	runningURL                   string
	exchangeName                 string
	m                            sync.Mutex
	connectionMutex              sync.RWMutex
	connector                    func() error

	subscriptionMutex   sync.Mutex
	subscribedChannels  []WebsocketChannelSubscription
	channelsToSubscribe []WebsocketChannelSubscription
	// subscription []WebsocketChannelSubscription

	channelSubscriber   func(channelToSubscribe *WebsocketChannelSubscription) error
	channelUnsubscriber func(channelToUnsubscribe *WebsocketChannelSubscription) error
	channelGeneratesubs func()
	DataHandler         chan interface{}
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
	features          *protocol.Features
}

// WebsocketSetup defines variables for setting up a websocket connection
type WebsocketSetup struct {
	Enabled                          bool
	Verbose                          bool
	AuthenticatedWebsocketAPISupport bool
	WebsocketTimeout                 time.Duration
	DefaultURL                       string
	ExchangeName                     string
	RunningURL                       string
	Connector                        func() error
	Subscriber                       func(channelToSubscribe *WebsocketChannelSubscription) error
	UnSubscriber                     func(channelToUnsubscribe *WebsocketChannelSubscription) error
	GenerateSubscriptions            func()
	Features                         *protocol.Features
}

// WebsocketChannelSubscription container for websocket subscriptions
// Currently only a one at a time thing to avoid complexity
type WebsocketChannelSubscription struct {
	Channel    string
	Currency   currency.Pair
	Params     map[string]interface{}
	subscribed bool
}

// WebsocketResponse defines generalised data from the websocket connection
type WebsocketResponse struct {
	Type int
	Raw  []byte
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
	trafic               chan struct{}
}

// WebsocketPingHandler container for ping handler settings
type WebsocketPingHandler struct {
	UseGorillaHandler bool
	MessageType       int
	Message           []byte
	Delay             time.Duration
}

// UnhandledMessageWarning defines a container for unhandled message warnings
type UnhandledMessageWarning struct {
	Message string
}
