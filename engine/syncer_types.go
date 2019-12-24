package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
)

// const holds the sync item types
const (
	SyncItemTicker = iota
	SyncItemOrderbook
	SyncItemTrade

	DefaultSyncerWorkers = 15
	DefaultSyncerTimeout = time.Second * 15

	syncProtocolREST      = "REST     "
	syncProtocolWebsocket = "websocket"
)

var (
	createdCounter     = 0
	removedCounter     = 0
	syncManagerUUID, _ = uuid.NewV4()
	// ErrInvalidItems  and such
	ErrInvalidItems = errors.New("no sync items enabled")
)

// Synchroniser wraps over individual synchronisation functionality
type Synchroniser interface {
	GetLastUpdated() time.Time
	GetNextUpdate() time.Time
	SetLastUpdated(time.Time)
	SetNextUpdate(time.Time)
	IsUsingProtocol(string) bool
	SetUsingProtocol(string)
	IsProcessing() bool
	SetProcessing(bool)
	Execute()
	InitialSyncComplete()
	// Stream takes and matches the payload with a synchronisation agent
	// pre-processor requirement
	Stream(payload interface{}) Synchroniser
	Cancel()
}

// SyncConfig stores the currency pair config
type SyncConfig struct {
	Ticker      bool
	Orderbook   bool
	Trades      bool
	Continuous  bool
	SyncTimeout time.Duration
	NumWorkers  int
	Verbose     bool
}

// ExchangeSyncerConfig stores the exchange syncer config
type ExchangeSyncerConfig struct {
	SyncDepositAddresses bool
	SyncOrders           bool
}

// SyncManager stores the exchange currency pair syncer object
type SyncManager struct {
	SyncConfig
	Agents   []Synchroniser
	shutdown chan struct{}
	pipe     chan SyncUpdate
	synchro  chan struct{}
	syncComm chan time.Time
	sync.Mutex
}
