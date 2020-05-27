package stream

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/cache"
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

	subscriptionMutex sync.Mutex
	subscriptions     []ChannelSubscription
	subscribe         chan []ChannelSubscription
	unsubscribe       chan []ChannelSubscription

	channelSubscriber   func([]ChannelSubscription) error
	channelUnsubscriber func([]ChannelSubscription) error
	channelGeneratesubs func() ([]ChannelSubscription, error)

	DataHandler chan interface{}
	ToRoutine   chan interface{}

	Match *Match

	// shutdown synchronises shutdown event across routines
	ShutdownC chan struct{}
	Wg        sync.WaitGroup

	// Orderbook is a local cache of orderbooks
	Orderbook cache.Orderbook

	// trafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
	// ReadMessageErrors will received all errors from ws.ReadMessage() and
	// verify if its a disconnection
	readMessageErrors chan error
	features          *protocol.Features

	// Standard stream connection
	Conn Connection
	// Authenticated stream connection
	AuthConn Connection
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
	Subscriber                       func([]ChannelSubscription) error
	UnSubscriber                     func([]ChannelSubscription) error
	GenerateSubscriptions            func() ([]ChannelSubscription, error)
	Features                         *protocol.Features
	// Local orderbook cache config values
	OrderbookBufferLimit  int
	BufferEnabled         bool
	SortBuffer            bool
	SortBufferByUpdateIDs bool
	UpdateEntriesByID     bool
}

// UnhandledMessageWarning defines a container for unhandled message warnings
type UnhandledMessageWarning struct {
	Message string
}
