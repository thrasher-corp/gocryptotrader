package engine

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// syncBase stores information
type syncBase struct {
	IsUsingWebsocket bool
	IsUsingREST      bool
	IsProcessing     bool
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
}

// currencyPairSyncAgent stores the sync agent info
type currencyPairSyncAgent struct {
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    syncBase
	Orderbook syncBase
	Trade     syncBase
}

// SyncManagerConfig stores the currency pair synchronization manager config
type SyncManagerConfig struct {
	SynchronizeTicker       bool
	SynchronizeOrderbook    bool
	SynchronizeTrades       bool
	SynchronizeContinuously bool
	TimeoutREST             time.Duration
	TimeoutWebsocket        time.Duration
	NumWorkers              int
	FiatDisplayCurrency     currency.Code
	PairFormatDisplay       *currency.PairFormat
	Verbose                 bool
}

// syncManager stores the exchange currency pair syncer object
type syncManager struct {
	initSyncCompleted              int32
	initSyncStarted                int32
	started                        int32
	format                         currency.PairFormat
	initSyncStartTime              time.Time
	fiatDisplayCurrency            currency.Code
	websocketRoutineManagerEnabled bool
	mu                             sync.Mutex
	initSyncWG                     sync.WaitGroup
	inService                      sync.WaitGroup

	currencyPairs            map[string]map[*currency.Item]map[*currency.Item]map[asset.Item]*currencyPairSyncAgent
	tickerBatchLastRequested map[string]map[asset.Item]time.Time
	batchMtx                 sync.Mutex

	remoteConfig    *config.RemoteControlConfig
	config          SyncManagerConfig
	exchangeManager iExchangeManager

	createdCounter int64
	removedCounter int64

	jobs chan syncJob
}

// syncJob defines a potential REST synchronization job
type syncJob struct {
	exch  exchange.IBotExchange
	Pair  currency.Pair
	Asset asset.Item
	class int
}
