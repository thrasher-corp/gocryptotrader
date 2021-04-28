package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const (
	CandleDataType = iota
	TradeDataType
)

var (
	errJobNotFound                = errors.New("job not found")
	errDatabaseConnectionRequired = errors.New("data history manager requires access to the database")
)

type DataHistoryManager struct {
	exchangeManager           iExchangeManager
	databaseConnectionManager iDatabaseConnectionManager
	started                   int32
	shutdown                  chan struct{}
	interval                  *time.Ticker
	jobs                      []*DataHistoryJob
	wg                        sync.WaitGroup
	m                         sync.RWMutex
}

type DataHistoryJob struct {
	Nickname         string         `json:"nickname"`
	Exchange         string         `json:"exchange"`
	Asset            asset.Item     `json:"asset"`
	Pair             currency.Pair  `json:"pair"`
	StartDate        time.Time      `json:"start-date"`
	EndDate          time.Time      `json:"end-date"`
	IsRolling        bool           `json:"is-rolling"`
	Interval         kline.Interval `json:"interval"`
	RequestSizeLimit uint32         `json:"request-size-limit"`
	DataType         int            `json:"data-type"`
	MaxRetryAttempts int            `json:"retry-attempts"`
	failures         []dataHistoryFailure
	continueFromData time.Time
	ranges           kline.IntervalRangeHolder
	running          bool
}

type dataHistoryFailure struct {
	reason string
}
