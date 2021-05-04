package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Data type descriptors
const (
	CandleDataType = iota
	TradeDataType
)

// Job status descriptors
const (
	StatusActive = iota
	StatusFailed
	StatusComplete
	StatusRemoved
)

var (
	errJobNotFound                = errors.New("job not found")
	errDatabaseConnectionRequired = errors.New("data history manager requires access to the database")
	defaultTicker                 = time.Minute
)

// DataHistoryManager is responsible for synchronising,
// retrieving and saving candle and trade data from loaded jobs
type DataHistoryManager struct {
	exchangeManager           iExchangeManager
	databaseConnectionManager iDatabaseConnectionManager
	started                   int32
	shutdown                  chan struct{}
	interval                  *time.Ticker
	jobs                      []*DataHistoryJob
	wg                        sync.WaitGroup
	m                         sync.RWMutex
	dataHistoryDB             *datahistoryjob.DBService
	dataHistoryJobDB          *datahistoryjobresult.DBService
}

// DataHistoryJob used to gather candle/trade history and save
// to the database
type DataHistoryJob struct {
	ID               uuid.UUID
	Nickname         string
	Exchange         string
	Asset            asset.Item
	Pair             currency.Pair
	StartDate        time.Time
	EndDate          time.Time
	IsRolling        bool
	Interval         kline.Interval
	RequestSizeLimit uint32
	DataType         int
	MaxRetryAttempts int
	Status           int
	CreatedDate      time.Time
	failures         []dataHistoryFailure
	continueFromData time.Time
	rangeHolder      kline.IntervalRangeHolder
	running          bool
}

type dataHistoryFailure struct {
	reason string
	time   time.Time
}
