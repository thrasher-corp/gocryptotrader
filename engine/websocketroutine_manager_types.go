package engine

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

var (
	errNilOrderManager       = errors.New("nil order manager received")
	errNilCurrencyPairSyncer = errors.New("nil currency pair syncer received")
	errNilCurrencyConfig     = errors.New("nil currency config received")
	errNilCurrencyPairFormat = errors.New("nil currency pair format received")
)

// websocketRoutineManager is used to process websocket updates from a unified location
type websocketRoutineManager struct {
	started         int32
	verbose         bool
	exchangeManager iExchangeManager
	orderManager    iOrderManager
	syncer          iCurrencyPairSyncer
	currencyConfig  *currency.Config
	shutdown        chan struct{}
	interceptor     Interceptor
	wg              sync.WaitGroup
	mu              sync.RWMutex
}

// Interceptor defines a function signature for an externally defined function
// that intercepts the data from the websocket routine manager.
type Interceptor func(service string, incoming interface{}) error
