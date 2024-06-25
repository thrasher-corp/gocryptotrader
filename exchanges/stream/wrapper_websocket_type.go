package stream

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// DefaultTestSetup represents a default asset websocket connection instance setup request parameter.
var DefaultTestSetup = &WebsocketSetup{
	DefaultURL:   "ws://something.com",
	RunningURL:   "ws://something.com",
	Connector:    func() error { return nil },
	Subscriber:   func(_ subscription.List) error { return nil },
	Unsubscriber: func(_ subscription.List) error { return nil },
	GenerateSubscriptions: func() (subscription.List, error) {
		return subscription.List{
			{Channel: "TestSub"},
			{Channel: "TestSub2"},
			{Channel: "TestSub3"},
			{Channel: "TestSub4"},
		}, nil
	},
	AssetType: asset.Spot,
}

// DefaultWrapperSetup represents a default websockets wrapper setup.
var DefaultWrapperSetup = &WebsocketWrapperSetup{
	ExchangeConfig: &config.Exchange{
		Enabled:                 true,
		WebsocketTrafficTimeout: time.Second * 30,
		Name:                    "test",
		Features: &config.FeaturesConfig{
			Enabled: config.FeaturesEnabledConfig{
				Websocket: true,
			},
		},
		API: config.APIConfig{
			AuthenticatedWebsocketSupport: true,
		},
	},
	Features: &protocol.Features{
		Subscribe:   true,
		Unsubscribe: true,
	},
}

// WrapperWebsocket defines a return type for websocket connections via the interface
// wrapper for routine processing
type WrapperWebsocket struct {
	canUseAuthenticatedEndpoints bool
	enabled                      atomic.Bool
	verbose                      bool
	dataMonitorRunning           atomic.Bool
	trafficTimeout               time.Duration
	connectionMonitorDelay       time.Duration
	proxyAddr                    string
	runningURL                   string
	exchangeName                 string
	m                            sync.Mutex

	subscriptionMutex sync.Mutex
	DataHandler       chan interface{}
	ToRoutine         chan interface{}
	Match             *Match

	connectedAssetTypesLocker sync.Mutex
	// connectedAssetTypesFlag holds a list of asset type connections
	connectedAssetTypesFlag asset.Item
	// shutdown synchronises shutdown event across routines
	ShutdownC chan asset.Item

	Wg *sync.WaitGroup
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

	// Latency reporter
	ExchangeLevelReporter Reporter

	// AssetTypeWebsockets defines a map of asset type item to corresponding websocket class
	AssetTypeWebsockets map[asset.Item]*Websocket
	// GenerateSubs function for exchange specific generating subscriptions from Features.Subscriptions, Pairs and Assets
	GenerateSubs func() (subscription.List, error)
}

// NewWrapper creates a new websocket wrapper instance
func NewWrapper() *WrapperWebsocket {
	return &WrapperWebsocket{
		DataHandler:         make(chan interface{}, jobBuffer),
		ToRoutine:           make(chan interface{}, jobBuffer),
		TrafficAlert:        make(chan struct{}),
		ReadMessageErrors:   make(chan error),
		AssetTypeWebsockets: make(map[asset.Item]*Websocket),
		ShutdownC:           make(chan asset.Item),
		Match:               NewMatch(),
		Wg:                  &sync.WaitGroup{},
	}
}
