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

// DataHistoryJob status descriptors
const (
	StatusActive = iota
	StatusFailed
	StatusComplete
	StatusRemoved
)

var (
	errJobNotFound                = errors.New("job not found")
	errDatabaseConnectionRequired = errors.New("data history manager requires access to the database")
	errUnknownDataType            = errors.New("job has invalid datatype set and cannot be processed")
	defaultTicker                 = time.Minute
	// defaultTradeInterval is the default interval size used to verify whether there is any database data
	// for a trade job
	defaultTradeInterval = kline.FifteenMin
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
	m                         sync.Mutex
	jobDB                     *datahistoryjob.DBService
	jobResultDB               *datahistoryjobresult.DBService
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
	Interval         kline.Interval
	BatchSize        int64
	RequestSizeLimit int64
	DataType         int64
	MaxRetryAttempts int64
	Status           int64
	CreatedDate      time.Time
	Results          []DataHistoryJobResult
	continueFromData time.Time
	rangeHolder      kline.IntervalRangeHolder
	running          bool
}

// DataHistoryJobResult contains details on
// the result of a history request
type DataHistoryJobResult struct {
	ID                uuid.UUID
	JobID             uuid.UUID
	IntervalStartDate time.Time
	IntervalEndDate   time.Time
	Status            int64
	Result            string
	Date              time.Time
}
