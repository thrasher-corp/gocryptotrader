package engine

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errNilCurrencyPairSyncer           = errors.New("nil currency pair syncer received")
	errNilCurrencyConfig               = errors.New("nil currency config received")
	errNilCurrencyPairFormat           = errors.New("nil currency pair format received")
	errNilWebsocketDataHandlerFunction = errors.New("websocket data handler function is nil")
	errNilWebsocket                    = errors.New("websocket is nil")
	errRoutineManagerNotStarted        = errors.New("websocket routine manager not started")
	errUseAPointer                     = errors.New("could not process, pass to websocket routine manager as a pointer")
)

const (
	stoppedState int32 = iota
	startingState
	readyState
)

// WebsocketRoutineManager is used to process websocket updates from a unified location
type WebsocketRoutineManager struct {
	state           int32
	verbose         bool
	exchangeManager iExchangeManager
	orderManager    iOrderManager
	syncer          iCurrencyPairSyncer
	currencyConfig  *currency.Config
	shutdown        chan struct{}
	dataHandlers    []WebsocketDataHandler
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// WebsocketDataHandler defines a function signature for a function that handles
// data coming from websocket connections.
type WebsocketDataHandler func(service string, incoming any) error
