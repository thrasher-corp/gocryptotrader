package engine

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/config"
)

// websocketRoutineManager is used to process websocket updates from a unified location
type websocketRoutineManager struct {
	started         int32
	verbose         bool
	exchangeManager iExchangeManager
	orderManager    iOrderManager
	syncer          iCurrencyPairSyncer
	currencyConfig  *config.CurrencyConfig
	shutdown        chan struct{}
	wg              sync.WaitGroup
}

var (
	errNilOrderManager       = errors.New("nil order manager received")
	errNilCurrencyPairSyncer = errors.New("nil currency pair syncer received")
	errNilCurrencyConfig     = errors.New("nil currency config received")
	errNilCurrencyPairFormat = errors.New("nil currency pair format received")
)
