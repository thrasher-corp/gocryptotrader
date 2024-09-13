package stream

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// Websocket functionality list and state consts
const (
	WebsocketNotAuthenticatedUsingRest = "%v - Websocket not authenticated, using REST\n"
	Ping                               = "ping"
	Pong                               = "pong"
	UnhandledMessage                   = " - Unhandled websocket message: "
)

const (
	uninitialisedState uint32 = iota
	disconnectedState
	connectingState
	connectedState
)

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing
type Websocket struct {
	canUseAuthenticatedEndpoints atomic.Bool
	enabled                      atomic.Bool
	state                        atomic.Uint32
	verbose                      bool
	connectionMonitorRunning     atomic.Bool
	trafficMonitorRunning        atomic.Bool
	dataMonitorRunning           atomic.Bool
	trafficTimeout               time.Duration
	connectionMonitorDelay       time.Duration
	proxyAddr                    string
	defaultURL                   string
	defaultURLAuth               string
	runningURL                   string
	runningURLAuth               string
	exchangeName                 string
	m                            sync.Mutex
	connector                    func() error

	subscriptions *subscription.Store

	// Subscriber function for exchange specific subscribe implementation
	Subscriber func(subscription.List) error
	// Subscriber function for exchange specific unsubscribe implementation
	Unsubscriber func(subscription.List) error
	// GenerateSubs function for exchange specific generating subscriptions from Features.Subscriptions, Pairs and Assets
	GenerateSubs func() (subscription.List, error)

	DataHandler chan interface{}
	ToRoutine   chan interface{}

	Match *Match

	// shutdown synchronises shutdown event across routines
	ShutdownC chan struct{}
	Wg        sync.WaitGroup

	// Orderbook is a local buffer of orderbooks
	Orderbook buffer.Orderbook

	// Trade is a notifier of occurring trades
	Trade trade.Trade

	// Fills is a notifier of occurring fills
	Fills fill.Fills

	// trafficAlert monitors if there is a halt in traffic throughput
	TrafficAlert chan struct{}
	// ReadMessageErrors will received all errors from ws.ReadMessage() and
	// verify if its a disconnection
	ReadMessageErrors chan error
	features          *protocol.Features

	// Standard stream connection
	Conn Connection
	// Authenticated stream connection
	AuthConn Connection

	// Latency reporter
	ExchangeLevelReporter Reporter

	// MaxSubScriptionsPerConnection defines the maximum number of
	// subscriptions per connection that is allowed by the exchange.
	MaxSubscriptionsPerConnection int
}

// WebsocketSetup defines variables for setting up a websocket connection
type WebsocketSetup struct {
	ExchangeConfig        *config.Exchange
	DefaultURL            string
	RunningURL            string
	RunningURLAuth        string
	Connector             func() error
	Subscriber            func(subscription.List) error
	Unsubscriber          func(subscription.List) error
	GenerateSubscriptions func() (subscription.List, error)
	Features              *protocol.Features

	// Local orderbook buffer config values
	OrderbookBufferConfig buffer.Config

	TradeFeed bool

	// Fill data config values
	FillsFeed bool

	// MaxWebsocketSubscriptionsPerConnection defines the maximum number of
	// subscriptions per connection that is allowed by the exchange.
	MaxWebsocketSubscriptionsPerConnection int
}

// WebsocketConnection contains all the data needed to send a message to a WS
// connection
type WebsocketConnection struct {
	Verbose   bool
	connected int32

	// Gorilla websocket does not allow more than one goroutine to utilise
	// writes methods
	writeControl sync.Mutex

	RateLimit    *request.RateLimiterWithWeight
	ExchangeName string
	URL          string
	ProxyURL     string
	Wg           *sync.WaitGroup
	Connection   *websocket.Conn
	ShutdownC    chan struct{}

	Match             *Match
	ResponseMaxLimit  time.Duration
	Traffic           chan struct{}
	readMessageErrors chan error

	// bespokeGenerateMessageID is a function that returns a unique message ID
	// defined externally. This is used for exchanges that require a unique
	// message ID for each message sent.
	bespokeGenerateMessageID func(highPrecision bool) int64

	Reporter Reporter
}
