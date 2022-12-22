package synchronize

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine/subsystem"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const holds the sync item types
const ManagerName = "exchange_syncer"

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
	initSyncStarted   int32
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
