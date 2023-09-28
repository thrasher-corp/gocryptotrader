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
	LastUpdated      time.Time
	HaveData         bool
	NumErrors        int
}

// currencyPairKey is the map key for the sync agents
type currencyPairKey struct {
	Exchange  string
	AssetType asset.Item
	Pair      currency.Pair
}

// currencyPairSyncAgent stores the sync agent info
type currencyPairSyncAgent struct {
	currencyPairKey
	Created  time.Time
	trackers []*syncBase
	locks    []sync.Mutex
}

// syncManager stores the exchange currency pair syncer object
type syncManager struct {
	initSyncCompleted              int32
	initSyncStarted                int32
	started                        int32
	shutdown                       chan bool
	format                         currency.PairFormat
	initSyncStartTime              time.Time
	fiatDisplayCurrency            currency.Code
	websocketRoutineManagerEnabled bool
	mux                            sync.Mutex
	initSyncWG                     sync.WaitGroup
	inService                      sync.WaitGroup

	currencyPairs            map[currencyPairKey]*currencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time

	remoteConfig    *config.RemoteControlConfig
	config          config.SyncManagerConfig
	exchangeManager iExchangeManager
}
