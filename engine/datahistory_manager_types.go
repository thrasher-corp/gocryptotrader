package engine

import (
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const dataHistoryManagerName = "data_history_manager"

type dataHistoryStatus int64
type dataHistoryDataType int64

// Data type descriptors
const (
	dataHistoryCandleDataType dataHistoryDataType = iota
	dataHistoryTradeDataType
)

// DataHistoryJob status descriptors
const (
	dataHistoryStatusActive dataHistoryStatus = iota
	dataHistoryStatusFailed
	dataHistoryStatusComplete
	dataHistoryStatusRemoved
	dataHistoryIntervalMissingData
)

// String stringifies iotas to readable
func (d dataHistoryStatus) String() string {
	switch {
	case int64(d) == 0:
		return "active"
	case int64(d) == 1:
		return "failed"
	case int64(d) == 2:
		return "complete"
	case int64(d) == 3:
		return "removed"
	case int64(d) == 4:
		return "missing data"
	}
	return ""
}

// Valid ensures the value set is legitimate
func (d dataHistoryStatus) Valid() bool {
	return int64(d) >= 0 && int64(d) <= 4
}

// String stringifies iotas to readable
func (d dataHistoryDataType) String() string {
	switch {
	case int64(d) == 0:
		return "candles"
	case int64(d) == 1:
		return "trades"
	}
	return ""
}

// Valid ensures the value set is legitimate
func (d dataHistoryDataType) Valid() bool {
	return int64(d) == 0 || int64(d) == 1
}

var (
	errJobNotFound                = errors.New("job not found")
	errUnknownDataType            = errors.New("job has invalid datatype set and cannot be processed")
	errNilJob                     = errors.New("nil job received")
	errNicknameIDUnset            = errors.New("must set 'id' OR 'nickname'")
	errEmptyID                    = errors.New("id not set")
	errOnlyNicknameOrID           = errors.New("can only set 'id' OR 'nickname'")
	errNicknameInUse              = errors.New("cannot continue as nickname already in use")
	errNicknameUnset              = errors.New("cannot continue as nickname unset")
	errJobInvalid                 = errors.New("job has not been setup properly and cannot be processed")
	errInvalidDataHistoryStatus   = errors.New("unsupported data history status received")
	errInvalidDataHistoryDataType = errors.New("unsupported data history data type received")
	errCanOnlyDeleteActiveJobs    = errors.New("can only delete active jobs")
	// defaultDataHistoryTradeInterval is the default interval size used to verify whether there is any database data
	// for a trade job
	defaultDataHistoryTradeInterval          = kline.FifteenMin
	defaultDataHistoryMaxJobsPerCycle  int64 = 5
	defaultDataHistoryBatchLimit       int64 = 3
	defaultDataHistoryRetryAttempts    int64 = 3
	defaultDataHistoryRequestSizeLimit int64 = 10
	defaultDataHistoryTicker                 = time.Minute
)

// DataHistoryManager is responsible for synchronising,
// retrieving and saving candle and trade data from loaded jobs
type DataHistoryManager struct {
	exchangeManager            iExchangeManager
	databaseConnectionInstance database.IDatabase
	started                    int32
	processing                 int32
	shutdown                   chan struct{}
	interval                   *time.Ticker
	jobs                       []*DataHistoryJob
	m                          sync.Mutex
	jobDB                      datahistoryjob.IDBService
	jobResultDB                datahistoryjobresult.IDBService
	maxJobsPerCycle            int64
	verbose                    bool
	tradeLoader                func(string, string, string, string, *kline.IntervalRangeHolder) error
	candleLoader               func(string, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (kline.Item, error)
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
	RunBatchLimit    int64
	RequestSizeLimit int64
	DataType         dataHistoryDataType
	MaxRetryAttempts int64
	Status           dataHistoryStatus
	CreatedDate      time.Time
	Results          map[time.Time][]DataHistoryJobResult
	rangeHolder      *kline.IntervalRangeHolder
}

// DataHistoryJobResult contains details on
// the result of a history request
type DataHistoryJobResult struct {
	ID                uuid.UUID
	JobID             uuid.UUID
	IntervalStartDate time.Time
	IntervalEndDate   time.Time
	Status            dataHistoryStatus
	Result            string
	Date              time.Time
}

// DataHistoryJobSummary is a human readable summary of the job
// for quickly understanding the status of a given job
type DataHistoryJobSummary struct {
	Nickname     string
	Exchange     string
	Asset        asset.Item
	Pair         currency.Pair
	StartDate    time.Time
	EndDate      time.Time
	Interval     kline.Interval
	Status       dataHistoryStatus
	DataType     dataHistoryDataType
	ResultRanges []string
}
