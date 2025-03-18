package engine

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const dataHistoryManagerName = "data_history_manager"

type (
	dataHistoryStatus   int64
	dataHistoryDataType int64
)

// Data type descriptors
const (
	dataHistoryCandleDataType dataHistoryDataType = iota
	dataHistoryTradeDataType
	dataHistoryConvertTradesDataType
	dataHistoryConvertCandlesDataType
	dataHistoryCandleValidationDataType
	dataHistoryCandleValidationSecondarySourceType
)

// DataHistoryJob status descriptors
const (
	dataHistoryStatusActive dataHistoryStatus = iota
	dataHistoryStatusFailed
	dataHistoryStatusComplete
	dataHistoryStatusRemoved
	dataHistoryIntervalIssuesFound
	dataHistoryStatusPaused
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
		return "issues found"
	case int64(d) == 5:
		return "paused"
	}
	return ""
}

// Valid ensures the value set is legitimate
func (d dataHistoryStatus) Valid() bool {
	return int64(d) >= 0 && int64(d) <= 5
}

// String stringifies iotas to readable
func (d dataHistoryDataType) String() string {
	n := int64(d)
	switch n {
	case 0:
		return "candles"
	case 1:
		return "trades"
	case 2:
		return "trade conversion"
	case 3:
		return "candle conversion"
	case 4:
		return "conversion validation"
	case 5:
		return "conversion validation secondary source"
	}
	return ""
}

// Valid ensures the value set is legitimate
func (d dataHistoryDataType) Valid() bool {
	return int64(d) >= 0 && int64(d) <= 5
}

var (
	errJobNotFound                = errors.New("job not found")
	errUnknownDataType            = errors.New("job has invalid datatype set and cannot be processed")
	errNilJob                     = errors.New("nil job received")
	errNicknameIDUnset            = errors.New("must set 'id' OR 'nickname'")
	errEmptyID                    = errors.New("id not set")
	errOnlyNicknameOrID           = errors.New("can only set 'id' OR 'nickname'")
	errBadStatus                  = errors.New("cannot set job status")
	errNicknameInUse              = errors.New("cannot continue as nickname already in use")
	errNicknameUnset              = errors.New("cannot continue as nickname unset")
	errJobInvalid                 = errors.New("job has not been setup properly and cannot be processed")
	errInvalidDataHistoryStatus   = errors.New("unsupported data history status received")
	errInvalidDataHistoryDataType = errors.New("unsupported data history data type received")
	errNilResult                  = errors.New("received nil job result")
	errJobMustBeActiveOrPaused    = errors.New("job must be active or paused to be set as a prerequisite")
	errNilCandles                 = errors.New("received nil candles")
)

const (
	// defaultDataHistoryTradeInterval is the default interval size used to verify whether there is any database data
	// for a trade job
	defaultDataHistoryTradeInterval           = kline.FifteenMin
	defaultDataHistoryMaxJobsPerCycle  int64  = 5
	defaultMaxResultInsertions         int64  = 10000
	defaultDataHistoryBatchLimit       uint64 = 3
	defaultDataHistoryRetryAttempts    uint64 = 3
	defaultDataHistoryRequestSizeLimit uint64 = 500
	defaultDataHistoryTicker                  = time.Minute
	defaultDataHistoryTradeRequestSize uint64 = 10
	defaultDecimalPlaceComparison      uint64 = 3
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
	jobDB                      datahistoryjob.IDBService
	jobResultDB                datahistoryjobresult.IDBService
	maxJobsPerCycle            int64
	maxResultInsertions        int64
	verbose                    bool
	candleLoader               func(string, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (*kline.Item, error)
	tradeLoader                func(string, string, string, string, time.Time, time.Time) ([]trade.Data, error)
	tradeSaver                 func(...trade.Data) error
	candleSaver                func(*kline.Item, bool) (uint64, error)
}

// DataHistoryJob used to gather candle/trade history and save
// to the database
type DataHistoryJob struct {
	ID                       uuid.UUID
	Nickname                 string
	Exchange                 string
	Asset                    asset.Item
	Pair                     currency.Pair
	StartDate                time.Time
	EndDate                  time.Time
	Interval                 kline.Interval
	RunBatchLimit            uint64
	RequestSizeLimit         uint64
	DataType                 dataHistoryDataType
	MaxRetryAttempts         uint64
	Status                   dataHistoryStatus
	CreatedDate              time.Time
	Results                  map[int64][]DataHistoryJobResult
	rangeHolder              *kline.IntervalRangeHolder
	OverwriteExistingData    bool
	ConversionInterval       kline.Interval
	DecimalPlaceComparison   uint64
	SecondaryExchangeSource  string
	IssueTolerancePercentage float64
	ReplaceOnIssue           bool
	// Prerequisites mean this job is paused until the prerequisite job is completed
	PrerequisiteJobID       uuid.UUID
	PrerequisiteJobNickname string
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
	Nickname                string
	Exchange                string
	Asset                   asset.Item
	Pair                    currency.Pair
	StartDate               time.Time
	EndDate                 time.Time
	Interval                kline.Interval
	Status                  dataHistoryStatus
	DataType                dataHistoryDataType
	ResultRanges            []string
	OverwriteExistingData   bool
	ConversionInterval      kline.Interval
	PrerequisiteJobNickname string
}
