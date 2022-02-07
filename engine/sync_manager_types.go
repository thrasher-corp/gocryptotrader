package engine

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
	Created   time.Time
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
	Ticker    syncBase
	Orderbook syncBase
	Trade     syncBase
}

// Config stores the currency pair config
type SyncManagerConfig struct {
	Ticker              bool
	Orderbook           bool
	Trades              bool
	Continuously        bool
	TimeoutREST         time.Duration
	TimeoutWebsocket    time.Duration
	NumWorkers          int
	FiatDisplayCurrency currency.Code
	PairFormatDisplay   *currency.PairFormat
	Verbose             bool
}

// syncManager stores the exchange currency pair syncer object
type syncManager struct {
	initSyncCompleted              int32
	initSyncStarted                int32
	started                        int32
	delimiter                      string
	uppercase                      bool
	initSyncStartTime              time.Time
	fiatDisplayCurrency            currency.Code
	websocketRoutineManagerEnabled bool
	mux                            sync.Mutex
	initSyncWG                     sync.WaitGroup
	inService                      sync.WaitGroup

	currencyPairs            []currencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time

	remoteConfig    *config.RemoteControlConfig
	config          SyncManagerConfig
	exchangeManager iExchangeManager
}
