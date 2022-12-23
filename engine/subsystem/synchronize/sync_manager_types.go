package synchronize

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	// DefaultWorkers limits the number of sync workers
	DefaultWorkers = 15
	// DefaultTimeoutREST the default time to switch from REST to websocket
	// protocols without a response.
	DefaultTimeoutREST = time.Second * 15
	// DefaultTimeoutWebsocket the default time to switch from websocket to REST
	// protocols without a response.
	DefaultTimeoutWebsocket = time.Minute
	// ManagerName defines a string identifier for the subsystem
	ManagerName = "exchange_syncer"

	defaultChannelBuffer = 10000
	book                 = "%s %s %s %s ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s"
)

var (
	// ErrNoItemsEnabled is for when there is not atleast one sync item enabled
	// e.g. an orderbook or ticker item.
	ErrNoItemsEnabled = errors.New("no sync items enabled")

	errUnknownSyncType   = errors.New("unknown sync type")
	errAgentNotFound     = errors.New("sync agent not found")
	errExchangeNameUnset = errors.New("exchange name unset")
	errProtocolUnset     = errors.New("protocol unset")
)

// Base stores independent sync information e.g a specific orderbook.
type Base struct {
	IsUsingWebsocket bool
	IsUsingREST      bool
	IsProcessing     bool
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
	mu               sync.Mutex
}

// Agent stores the sync agent information on exchange, asset type and pair
// and holds the individual item bases.
type Agent struct {
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    Base
	Orderbook Base
	Trade     Base
}

// ManagerConfig stores the currency pair synchronization manager config
type ManagerConfig struct {
	// TODO: Bitmask booleans to reduce switch cases.
	SynchronizeTicker       bool
	SynchronizeOrderbook    bool
	SynchronizeTrades       bool
	SynchronizeContinuously bool
	TimeoutREST             time.Duration
	TimeoutWebsocket        time.Duration
	NumWorkers              int
	FiatDisplayCurrency     currency.Code
	PairFormatDisplay       currency.PairFormat
	Verbose                 bool
	ExchangeManager         subsystem.ExchangeManager
	RemoteConfig            *config.RemoteControlConfig
	APIServerManager        subsystem.APIServer
}

// Manager defines the main total currency pair synchronization subsystem that
// fetches and maintains up to date market data.
type Manager struct {
	initSyncCompleted int32
	started           int32
	initSyncStartTime time.Time
	mu                sync.Mutex
	initSyncWG        sync.WaitGroup

	currencyPairs            map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*Agent
	tickerBatchLastRequested map[string]map[asset.Item]time.Time
	batchMtx                 sync.Mutex

	ManagerConfig

	createdCounter int64
	removedCounter int64

	orderbookJobs chan RESTJob
	tickerJobs    chan RESTJob
	tradeJobs     chan RESTJob
}

// RESTJob defines a potential REST synchronization job
type RESTJob struct {
	exch  exchange.IBotExchange
	Pair  currency.Pair
	Asset asset.Item
	Item  subsystem.SynchronizationType
}
