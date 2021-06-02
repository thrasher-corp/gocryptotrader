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
	}
	return ""
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

var (
	errJobNotFound                = errors.New("job not found")
	errDatabaseConnectionRequired = errors.New("data history manager requires access to the database")
	errUnknownDataType            = errors.New("job has invalid datatype set and cannot be processed")
	errNilJob                     = errors.New("nil job received")
	errNicknameIDUnset            = errors.New("must set 'id' OR 'nickname'")
	errOnlyNicknameOrID           = errors.New("can only set 'id' OR 'nickname'")
	errNicknameInUse              = errors.New("cannot insert job as nickname already in use")
	errNicknameUnset              = errors.New("cannot insert job as nickname unset")
	errJobInvalid                 = errors.New("job has not been setup properly and cannot be processed")
	// defaultTradeInterval is the default interval size used to verify whether there is any database data
	// for a trade job
	defaultTradeInterval         = kline.FifteenMin
	defaultMaxJobsPerCycle int64 = 5
	defaultBatchLimit      int64 = 3
	defaultRetryAttempts   int64 = 3
	defaultTicker                = time.Minute
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
	wg                         sync.WaitGroup
	m                          sync.Mutex
	jobDB                      datahistoryjob.IDBService
	jobResultDB                datahistoryjobresult.IDBService
	maxJobsPerCycle            int64
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
