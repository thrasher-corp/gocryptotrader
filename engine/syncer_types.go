package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// const holds the sync item types
const (
	REST      = "REST"
	Websocket = "websocket"

	// default REST sync delays
	defaultSyncDelay                  = 10 * time.Second
	defaultExchangeTradeHistoryDelay  = time.Minute
	defaultExchangeSupportedPairDelay = 30 * time.Minute
	defaultDepositAddressDelay        = 5 * time.Minute
)

var (
	syncManagerUUID, _ = uuid.NewV4()
	// ErrInvalidItems alerts of no sync items enabled
	ErrInvalidItems = errors.New("no sync items enabled")
)

// Synchroniser wraps over individual synchronisation functionality
type Synchroniser interface {
	GetLastUpdated() time.Time
	GetNextUpdate() time.Time
	SetNewUpdate()
	IsProcessing() bool
	SetProcessing(bool)
	Execute()
	InitialSyncComplete()
	// Stream takes and matches the payload with a synchronisation agent
	// pre-processor requirement
	Stream(payload interface{}) Synchroniser
	IsRESTDisabled() bool
	DisableREST()
	EnableREST()
	GetExchangeName() string
	GetAgentName() string
	Lock()
	Unlock()
	Cancel()
	Clear()
	IsCancelled() bool
}

// SyncConfig stores the currency pair config
type SyncConfig struct {
	AccountBalance         bool
	AccountOrders          bool
	ExchangeTrades         bool
	ExchangeOrderbook      bool
	ExchangeSupportedPairs bool
	ExchangeTicker         bool
}

// SyncManager stores the exchange currency pair syncer object
type SyncManager struct {
	SyncConfig
	Agents   []Synchroniser
	started  int32
	stopped  int32
	shutdown chan struct{}
	pipe     chan SyncUpdate
	synchro  chan struct{}
	syncComm chan time.Time
	sync.RWMutex
	jobBuffer map[string]chan Synchroniser
	wg        sync.WaitGroup
}

// SyncUpdate wraps updates for concurrent processing
type SyncUpdate struct {
	Agent    Synchroniser
	Payload  interface{}
	Protocol string
	Err      error
}

// Agent defines core fields to implement the sychroniser interface.
// To add additional agents requires the new struct to imbed an agent and
// define an execution method and stream method
type Agent struct {
	Name            string
	Exchange        exchange.IBotExchange
	Processing      bool
	Cancelled       bool
	NextUpdate      time.Time
	LastUpdated     time.Time
	RestUpdateDelay time.Duration
	Pipe            chan SyncUpdate
	Wg              *sync.WaitGroup
	Disabled        bool
	mtx             sync.Mutex
}

// TickerAgent synchronises the exchange currency pair ticker
type TickerAgent struct {
	Agent
	AssetType asset.Item
	Pair      currency.Pair
}

// OrderbookAgent synchronises the exchange currency pair orderbook
type OrderbookAgent struct {
	Agent
	AssetType asset.Item
	Pair      currency.Pair
}

// TradeAgent synchronises the exchange currency pair trades
type TradeAgent struct {
	Agent
	AssetType asset.Item
	Pair      currency.Pair
}

// AccountBalanceAgent synchronises the exchange account balances
type AccountBalanceAgent struct {
	Agent
}

// OrderAgent synchronises the exchange account orders
type OrderAgent struct {
	Agent
	Pair  currency.Pair
	Asset asset.Item
}

// SupportedPairsAgent synchronises the exchange supported currency pairs
type SupportedPairsAgent struct {
	Agent
}
