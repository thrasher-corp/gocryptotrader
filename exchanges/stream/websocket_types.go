package stream

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
)

// Websocket functionality list and state consts
const (
	// WebsocketNotEnabled alerts of a disabled websocket
	WebsocketNotEnabled = "exchange_websocket_not_enabled"
	// connection monitor time delays and limits
	connectionMonitorDelay             = 2 * time.Second
	WebsocketNotAuthenticatedUsingRest = "%v - Websocket not authenticated, using REST\n"
	Ping                               = "ping"
	Pong                               = "pong"
	UnhandledMessage                   = " - Unhandled websocket message: "
)

// Websocket defines a return type for websocket connections via the interface
// wrapper for routine processing
type Websocket struct {
	canUseAuthenticatedEndpoints bool
	enabled                      bool
	Init                         bool
	connected                    bool
	connecting                   bool
	verbose                      bool
	connectionMonitorRunning     bool
	trafficMonitorRunning        bool
	dataMonitorRunning           bool
	trafficTimeout               time.Duration
	proxyAddr                    string
	defaultURL                   string
	defaultURLAuth               string
	runningURL                   string
	runningURLAuth               string
	exchangeName                 string
	m                            sync.Mutex
	connectionMutex              sync.RWMutex
	connector                    func() error

	subscriptionMutex sync.Mutex
	subscriptions     []ChannelSubscription
	Subscribe         chan []ChannelSubscription
	Unsubscribe       chan []ChannelSubscription

	// Subscriber function for package defined websocket subscriber
	// functionality
	Subscriber func([]ChannelSubscription) error
	// Unsubscriber function for packaged defined websocket unsubscriber
	// functionality
	Unsubscriber func([]ChannelSubscription) error
	// GenerateSubs function for package defined websocket generate
	// subscriptions functionality
	GenerateSubs func() ([]ChannelSubscription, error)

	DataHandler chan interface{}
	ToRoutine   chan interface{}

	Match *Match

	// shutdown synchronises shutdown event across routines
	ShutdownC chan struct{}
	Wg        *sync.WaitGroup

	// Orderbook is a local buffer of orderbooks
	Orderbook buffer.Orderbook

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
	RunningURLAuth                   string
	Connector                        func() error
	Subscriber                       func([]ChannelSubscription) error
	UnSubscriber                     func([]ChannelSubscription) error
	GenerateSubscriptions            func() ([]ChannelSubscription, error)
	Features                         *protocol.Features
	// Local orderbook buffer config values
	OrderbookBufferLimit  int
	BufferEnabled         bool
	SortBuffer            bool
	SortBufferByUpdateIDs bool
	UpdateEntriesByID     bool
}

// WebsocketConnection contains all the data needed to send a message to a WS
// connection
type WebsocketConnection struct {
	Verbose   bool
	connected int32

	// Gorilla websocket does not allow more than one goroutine to utilise
	// writes methods
	writeControl sync.Mutex

	RateLimit    int64
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
}
