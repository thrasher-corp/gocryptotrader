package currencypairsyncer

import (
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type iWebsocketDataReceiver interface {
	WebsocketDataReceiver(ws *stream.Websocket)
}

type iExchangeManager interface {
	GetExchanges() []exchange.IBotExchange
	GetExchangeByName(string) exchange.IBotExchange
}

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

// CurrencyPairSyncerConfig stores the currency pair config
type Config struct {
	SyncTicker       bool
	SyncOrderbook    bool
	SyncTrades       bool
	SyncContinuously bool
	SyncTimeout      time.Duration
	NumWorkers       int
	Verbose          bool
}

// ExchangeCurrencyPairSyncer stores the exchange currency pair syncer object
type ExchangeCurrencyPairSyncer struct {
	initSyncCompleted   int32
	initSyncStarted     int32
	shutdown            int32
	delimiter           string
	uppercase           bool
	initSyncStartTime   time.Time
	fiatDisplayCurrency currency.Code
	mux                 sync.Mutex
	initSyncWG          sync.WaitGroup

	currencyPairs            []currencyPairSyncAgent
	tickerBatchLastRequested map[string]time.Time

	remoteConfig          *config.RemoteControlConfig
	config                Config
	exchangeManager       iExchangeManager
	websocketDataReceiver iWebsocketDataReceiver
}
