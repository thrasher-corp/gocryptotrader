package websocket

import (
	"sync"
	"sync/atomic"
	"time"

	underlying "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/internal/exchange/websocket/buffer"
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

// Manager provides connection and subscription management and routing
type Manager struct {
	enabled                       atomic.Bool
	state                         atomic.Uint32
	verbose                       bool
	canUseAuthenticatedEndpoints  atomic.Bool
	connectionMonitorRunning      atomic.Bool
	trafficTimeout                time.Duration
	connectionMonitorDelay        time.Duration
	proxyAddr                     string
	defaultURL                    string
	defaultURLAuth                string
	runningURL                    string
	runningURLAuth                string
	exchangeName                  string
	features                      *protocol.Features
	m                             sync.Mutex
	connections                   map[Connection]*ConnectionWrapper
	subscriptions                 *subscription.Store
	connector                     func() error
	rateLimitDefinitions          request.RateLimitDefinitions // rate limiters shared between Websocket and REST connections
	Subscriber                    func(subscription.List) error
	Unsubscriber                  func(subscription.List) error
	GenerateSubs                  func() (subscription.List, error)
	useMultiConnectionManagement  bool
	DataHandler                   chan any
	ToRoutine                     chan any
	Match                         *Match
	ShutdownC                     chan struct{}
	Wg                            sync.WaitGroup
	Orderbook                     buffer.Orderbook
	Trade                         trade.Trade // Trade is a notifier for trades
	Fills                         fill.Fills  // Fills is a notifier for fills
	TrafficAlert                  chan struct{}
	ReadMessageErrors             chan error
	Conn                          Connection // Public connection
	AuthConn                      Connection // Authenticated Private connection
	ExchangeLevelReporter         Reporter   // Latency reporter
	MaxSubscriptionsPerConnection int

	// connectionManager stores all *potential* connections for the exchange, organised within ConnectionWrapper structs.
	// Each ConnectionWrapper one connection (will be expanded soon) tailored for specific exchange functionalities or asset types. // TODO: Expand this to support multiple connections per ConnectionWrapper
	// For example, separate connections can be used for Spot, Margin, and Futures trading. This structure is especially useful
	// for exchanges that differentiate between trading pairs by using different connection endpoints or protocols for various asset classes.
	// If an exchange does not require such differentiation, all connections may be managed under a single ConnectionWrapper.
	connectionManager []*ConnectionWrapper
}

// ManagerSetup defines variables for setting up a websocket manager
type ManagerSetup struct {
	ExchangeConfig        *config.Exchange
	DefaultURL            string
	RunningURL            string
	RunningURLAuth        string
	Connector             func() error
	Subscriber            func(subscription.List) error
	Unsubscriber          func(subscription.List) error
	GenerateSubscriptions func() (subscription.List, error)
	Features              *protocol.Features
	OrderbookBufferConfig buffer.Config

	// UseMultiConnectionManagement allows the connections to be managed by the
	// connection manager. If false, this will default to the global fields
	// provided in this struct.
	UseMultiConnectionManagement bool

	TradeFeed bool
	FillsFeed bool

	MaxWebsocketSubscriptionsPerConnection int

	// RateLimitDefinitions contains the rate limiters shared between WebSocket and REST connections for all endpoints.
	// These rate limits take precedence over any rate limits specified in individual connection configurations.
	// If no connection-specific rate limit is provided and the endpoint does not match any of these definitions,
	// an error will be returned. However, if a connection configuration includes its own rate limit,
	// it will fall back to that configuration’s rate limit without raising an error.
	RateLimitDefinitions request.RateLimitDefinitions
}

// connection contains all the data needed to send a message to a websocket connection
type connection struct {
	Verbose                  bool
	connected                int32
	writeControl             sync.Mutex                     // Gorilla websocket does not allow more than one goroutine to utilise write methods
	RateLimit                *request.RateLimiterWithWeight // RateLimit is a rate limiter for the connection itself
	RateLimitDefinitions     request.RateLimitDefinitions   // RateLimitDefinitions contains the rate limiters shared between WebSocket and REST connections
	Reporter                 Reporter
	ExchangeName             string
	URL                      string
	ProxyURL                 string
	Wg                       *sync.WaitGroup
	Connection               *underlying.Conn
	shutdown                 chan struct{}
	Match                    *Match
	ResponseMaxLimit         time.Duration
	Traffic                  chan struct{}
	readMessageErrors        chan error
	bespokeGenerateMessageID func(highPrecision bool) int64
}
