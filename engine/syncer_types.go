package engine

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/openware/irix/asset"
)

// CurrencyPairSyncerConfig stores the currency pair config
type CurrencyPairSyncerConfig struct {
	SyncTicker           bool
	SyncOrderbook        bool
	SyncTrades           bool
	SyncContinuously     bool
	SyncTimeoutREST      time.Duration
	SyncTimeoutWebsocket time.Duration
	NumWorkers           int
	Verbose              bool
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
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    SyncBase
	Orderbook SyncBase
	Trade     SyncBase
}
