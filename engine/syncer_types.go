package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
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
	Cancel()
}

// SyncConfig stores the currency pair config
type SyncConfig struct {
	AccountBalance           bool
	AccountFees              bool
	AccountOrders            bool
	AccountFunding           bool
	AccountPosition          bool
	ExchangeTrades           bool
	ExchangeOrderbook        bool
	ExchangeDepositAddresses bool
	ExchangeTradeHistory     bool
	ExchangeSupportedPairs   bool
	ExchangeTicker           bool
	ExchangeKline            bool
	Verbose                  bool
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

// SyncUpdate wraps updates for concurrent processing
type SyncUpdate struct {
	Agent    Synchroniser
	Payload  interface{}
	Protocol string
	Err      error
}
