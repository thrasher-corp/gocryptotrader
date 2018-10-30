package engine

import (
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

// CurrencyPairSyncerConfig stores the currency pair config
type CurrencyPairSyncerConfig struct {
	SyncTicker       bool
	SyncOrderbook    bool
	SyncTrades       bool
	SyncContinuously bool

	NumWorkers int
	Verbose    bool
}

// ExchangeSyncerConfig stores the exchange syncer config
type ExchangeSyncerConfig struct {
	SyncDepositAddresses bool
	SyncOrders           bool
}

// ExchangeCurrencyPairSyncer stores the exchange currency pair syncer object
type ExchangeCurrencyPairSyncer struct {
	Cfg                      CurrencyPairSyncerConfig
	CurrencyPairs            []CurrencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time
	mux                      sync.Mutex
	initSyncWG               sync.WaitGroup

	initSyncCompleted int32
	initSyncStarted   int32
	initSyncStartTime time.Time
	shutdown          int32
}

// SyncBase stores information
type SyncBase struct {
	IsUsingWebsocket bool
	IsUsingREST      bool
	IsProcessing     bool
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
}

// CurrencyPairSyncAgent stores the sync agent info
type CurrencyPairSyncAgent struct {
	Created   time.Time
	Exchange  string
	AssetType assets.AssetType
	Pair      currency.Pair
	Ticker    SyncBase
	Orderbook SyncBase
	Trade     SyncBase
}
