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
type Config struct {
	SyncTicker           bool
	SyncOrderbook        bool
	SyncTrades           bool
	SyncContinuously     bool
	SyncTimeoutREST      time.Duration
	SyncTimeoutWebsocket time.Duration
	NumWorkers           int
	Verbose              bool
}

// syncManager stores the exchange currency pair syncer object
type syncManager struct {
	initSyncCompleted   int32
	initSyncStarted     int32
	started             int32
	delimiter           string
	uppercase           bool
	initSyncStartTime   time.Time
	fiatDisplayCurrency currency.Code
	mux                 sync.Mutex
	initSyncWG          sync.WaitGroup
	inService           sync.WaitGroup

	currencyPairs            []currencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time

	remoteConfig          *config.RemoteControlConfig
	config                Config
	exchangeManager       iExchangeManager
	websocketDataReceiver iWebsocketDataReceiver
}
