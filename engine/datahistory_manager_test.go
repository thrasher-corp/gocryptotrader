package engine

import (
	"database/sql"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

func TestSetupDataHistoryManager(t *testing.T) {
	t.Parallel()
	_, err := SetupDataHistoryManager(nil, nil, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), nil, nil)
	if !errors.Is(err, errNilDatabaseConnectionManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilDatabaseConnectionManager)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), &DatabaseConnectionManager{}, nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), &DatabaseConnectionManager{}, &config.DataHistoryManager{})
	if !errors.Is(err, database.ErrNilInstance) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrNilInstance)
	}

	dbInst := &database.Instance{}
	err = dbInst.SetConfig(&database.Config{Enabled: true})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	dbInst.SetConnected(true)
	dbCM := &DatabaseConnectionManager{
		dbConn:  dbInst,
		started: 1,
	}
	err = dbInst.SetSQLiteConnection(&sql.DB{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m, err := SetupDataHistoryManager(SetupExchangeManager(), dbCM, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
}

func TestDataHistoryManagerIsRunning(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	m.started = 0
	if m.IsRunning() {
		t.Error("expected false")
	}
	m.started = 1
	if !m.IsRunning() {
		t.Error("expected true")
	}
	m = nil
	if m.IsRunning() {
		t.Error("expected false")
	}
}

func TestDataHistoryManagerStart(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	m.started = 0
	err := m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.Start()
	if !errors.Is(err, ErrSubSystemAlreadyStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemAlreadyStarted)
	}
	m = nil
	err = m.Start()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestDataHistoryManagerStop(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	m.shutdown = make(chan struct{})
	err := m.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m = nil
	err = m.Stop()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestUpsertJob(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	err := m.UpsertJob(nil, false)
	if !errors.Is(err, errNilJob) {
		t.Errorf("error '%v', expected '%v'", err, errNilJob)
	}
	dhj := &DataHistoryJob{}
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, errNicknameUnset) {
		t.Errorf("error '%v', expected '%v'", err, errNicknameUnset)
	}
	dhj.Nickname = "test1337"
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("error '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	dhj.Asset = asset.Spot
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("error '%v', expected '%v'", err, errCurrencyPairUnset)
	}

	dhj.Exchange = strings.ToLower(testExchange)
	dhj.Pair = currency.NewPair(currency.BTC, currency.USDT)
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, errCurrencyNotEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errCurrencyNotEnabled)
	}

	dhj.Pair = currency.NewPair(currency.BTC, currency.USD)
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, kline.ErrUnsupportedInterval) {
		t.Errorf("error '%v', expected '%v'", err, kline.ErrUnsupportedInterval)
	}

	dhj.Interval = kline.OneHour
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("error '%v', expected '%v'", err, common.ErrDateUnset)
	}

	dhj.StartDate = time.Now().Add(-time.Hour)
	dhj.EndDate = time.Now()
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 1 {
		t.Error("unexpected jerrb")
	}

	err = m.UpsertJob(dhj, true)
	if !errors.Is(err, errNicknameInUse) {
		t.Errorf("error '%v', expected '%v'", err, errNicknameInUse)
	}

	newJob := &DataHistoryJob{
		Nickname:         dhj.Nickname,
		Exchange:         testExchange,
		Asset:            asset.Spot,
		Pair:             currency.NewPair(currency.BTC, currency.USD),
		StartDate:        startDate,
		EndDate:          time.Now().Add(-time.Minute),
		Interval:         kline.FifteenMin,
		RunBatchLimit:    1338,
		RequestSizeLimit: 1337,
		DataType:         2,
		MaxRetryAttempts: 1337,
	}
	err = m.UpsertJob(newJob, false)
	if !errors.Is(err, errInvalidDataHistoryDataType) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidDataHistoryDataType)
	}

	newJob.DataType = dataHistoryTradeDataType
	err = m.UpsertJob(newJob, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if !m.jobs[0].StartDate.Equal(startDate) {
		t.Error(err)
	}
}

func TestSetJobStatus(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestSetJobStatus",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.SetJobStatus("", "", 0)
	if !errors.Is(err, errNicknameIDUnset) {
		t.Errorf("error '%v', expected '%v'", err, errNicknameIDUnset)
	}

	err = m.SetJobStatus("1337", "1337", 0)
	if !errors.Is(err, errOnlyNicknameOrID) {
		t.Errorf("error '%v', expected '%v'", err, errOnlyNicknameOrID)
	}

	err = m.SetJobStatus(dhj.Nickname, "", dataHistoryStatusRemoved)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 0 {
		t.Error("expected 0")
	}
	err = m.SetJobStatus("", dhj.ID.String(), dataHistoryStatusActive)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 1 {
		t.Error("expected 1")
	}

	err = m.SetJobStatus("", dhj.ID.String(), dataHistoryStatusPaused)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 0 {
		t.Error("expected 0")
	}

	atomic.StoreInt32(&m.started, 0)
	err = m.SetJobStatus("", dhj.ID.String(), 0)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	err = m.SetJobStatus("", dhj.ID.String(), 0)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestGetByNickname(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGetByNickname",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.GetByNickname(dhj.Nickname, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.GetByNickname(dhj.Nickname, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.jobs = []*DataHistoryJob{}
	_, err = m.GetByNickname(dhj.Nickname, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetByNickname("test123", false)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.GetByNickname("test123", false)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestGetByID(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGetByID",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.GetByID(dhj.ID)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	_, err = m.GetByID(uuid.UUID{})
	if !errors.Is(err, errEmptyID) {
		t.Errorf("error '%v', expected '%v'", err, errEmptyID)
	}

	m.jobs = []*DataHistoryJob{}
	_, err = m.GetByID(dhj.ID)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetByID(dhj.ID)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.GetByID(dhj.ID)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestRetrieveJobs(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestRetrieveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	jobs, err := m.retrieveJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Error("expected job")
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.retrieveJobs()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.retrieveJobs()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestGetActiveJobs(t *testing.T) {
	t.Parallel()
	m := createDHM(t)

	jobs, err := m.GetActiveJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}

	dhj := &DataHistoryJob{
		Nickname:  "TestGetActiveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	jobs, err = m.GetActiveJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Error("expected 1 job")
	}

	dhj.Status = dataHistoryStatusFailed
	jobs, err = m.GetActiveJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetActiveJobs()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.GetActiveJobs()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestValidateJob(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	err := m.validateJob(nil)
	if !errors.Is(err, errNilJob) {
		t.Errorf("error '%v', expected '%v'", err, errNilJob)
	}
	dhj := &DataHistoryJob{}
	err = m.validateJob(dhj)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("error '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	dhj.Asset = asset.Spot
	err = m.validateJob(dhj)
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("error '%v', expected '%v'", err, errCurrencyPairUnset)
	}

	dhj.Exchange = testExchange
	dhj.Pair = currency.NewPair(currency.BTC, currency.USDT)
	err = m.validateJob(dhj)
	if !errors.Is(err, errCurrencyNotEnabled) {
		t.Errorf("error '%v', expected '%v'", err, errCurrencyNotEnabled)
	}

	dhj.Pair = currency.NewPair(currency.BTC, currency.USD)
	err = m.validateJob(dhj)
	if !errors.Is(err, kline.ErrUnsupportedInterval) {
		t.Errorf("error '%v', expected '%v'", err, kline.ErrUnsupportedInterval)
	}

	dhj.Interval = kline.OneMin
	err = m.validateJob(dhj)
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("error '%v', expected '%v'", err, common.ErrDateUnset)
	}

	dhj.StartDate = time.Now().Add(time.Minute)
	dhj.EndDate = time.Now().Add(time.Hour)
	err = m.validateJob(dhj)
	if !errors.Is(err, common.ErrStartAfterTimeNow) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidTimes)
	}

	dhj.StartDate = time.Now().Add(-time.Hour)
	dhj.EndDate = time.Now().Add(-time.Minute)
	err = m.validateJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestGetAllJobStatusBetween(t *testing.T) {
	t.Parallel()
	m := createDHM(t)

	dhj := &DataHistoryJob{
		Nickname:  "TestGetActiveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	jobs, err := m.GetAllJobStatusBetween(time.Now().Add(-time.Minute*5), time.Now().Add(time.Minute))
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Error("expected 1 job")
	}

	_, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	m.started = 0
	_, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestPrepareJobs(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	jobs, err := m.PrepareJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, received %v", len(jobs))
	}
	m.started = 0
	_, err = m.PrepareJobs()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m = nil
	_, err = m.PrepareJobs()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestCompareJobsToData(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGenerateJobSummary",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.compareJobsToData(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj.DataType = dataHistoryTradeDataType
	err = m.compareJobsToData(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj.DataType = 1337
	err = m.compareJobsToData(dhj)
	if !errors.Is(err, errUnknownDataType) {
		t.Errorf("error '%v', expected '%v'", err, errUnknownDataType)
	}
	m.started = 0
	err = m.compareJobsToData(dhj)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}
	m = nil
	err = m.compareJobsToData(dhj)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestRunJob(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestProcessJobs",
		Exchange:  "Binance",
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-time.Hour * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.tradeSaver = dataHistoryTradeSaver
	m.candleSaver = dataHistoryCandleSaver
	m.tradeLoader = dataHistoryTraderLoader
	m.tradeChecker = dataHistoryHasDataChecker

	err = m.runJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj.Pair = currency.NewPair(currency.DOGE, currency.USDT)
	err = m.runJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhjt := &DataHistoryJob{
		Nickname:  "TestProcessJobs2",
		Exchange:  "Binance",
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-time.Hour * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
		DataType:  dataHistoryTradeDataType,
	}
	err = m.UpsertJob(dhjt, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.compareJobsToData(dhjt)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.runJob(dhjt)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhjt.DataType = dataHistoryConvertCandlesDataType
	err = m.runJob(dhjt)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhjt.DataType = dataHistoryConvertTradesDataType
	err = m.runJob(dhjt)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhjt.DataType = dataHistoryCandleValidationDataType
	err = m.runJob(dhjt)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	atomic.StoreInt32(&m.started, 0)
	err = m.runJob(dhjt)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	err = m.runJob(dhjt)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestGenerateJobSummaryTest(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestGenerateJobSummary",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Minute * 5),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	summary, err := m.GenerateJobSummary("TestGenerateJobSummary")
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(summary.ResultRanges) == 0 {
		t.Error("expected result ranges")
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GenerateJobSummary("TestGenerateJobSummary")
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.GenerateJobSummary("TestGenerateJobSummary")
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestRunJobs(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	err := m.runJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 0)
	err = m.runJobs()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	err = m.runJobs()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestConverters(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	id, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	id2, err := uuid.NewV4()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj := &DataHistoryJob{
		ID:        id,
		Nickname:  "TestProcessJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-time.Hour * 24),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}

	dbJob := m.convertJobToDBModel(dhj)
	if dhj.ID.String() != dbJob.ID ||
		dhj.Nickname != dbJob.Nickname ||
		!dhj.StartDate.Equal(dbJob.StartDate) ||
		int64(dhj.Interval.Duration()) != dbJob.Interval ||
		dhj.Pair.Base.String() != dbJob.Base ||
		dhj.Pair.Quote.String() != dbJob.Quote {
		t.Error("expected matching job")
	}

	convertBack, err := m.convertDBModelToJob(dbJob)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if dhj.ID != convertBack.ID ||
		dhj.Nickname != convertBack.Nickname ||
		!dhj.StartDate.Equal(convertBack.StartDate) ||
		dhj.Interval != convertBack.Interval ||
		!dhj.Pair.Equal(convertBack.Pair) {
		t.Error("expected matching job")
	}

	jr := DataHistoryJobResult{
		ID:                id,
		JobID:             id2,
		IntervalStartDate: dhj.StartDate,
		IntervalEndDate:   dhj.EndDate,
		Status:            0,
		Result:            "test123",
		Date:              time.Now(),
	}
	mapperino := make(map[time.Time][]DataHistoryJobResult)
	mapperino[dhj.StartDate] = append(mapperino[dhj.StartDate], jr)
	result := m.convertJobResultToDBResult(mapperino)
	if jr.ID.String() != result[0].ID ||
		jr.JobID.String() != result[0].JobID ||
		jr.Result != result[0].Result ||
		!jr.Date.Equal(result[0].Date) ||
		!jr.IntervalStartDate.Equal(result[0].IntervalStartDate) ||
		!jr.IntervalEndDate.Equal(result[0].IntervalEndDate) ||
		jr.Status != dataHistoryStatus(result[0].Status) {
		t.Error("expected matching job")
	}

	andBackAgain, err := m.convertDBResultToJobResult(result)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if jr.ID != andBackAgain[dhj.StartDate][0].ID ||
		jr.JobID != andBackAgain[dhj.StartDate][0].JobID ||
		jr.Result != andBackAgain[dhj.StartDate][0].Result ||
		!jr.Date.Equal(andBackAgain[dhj.StartDate][0].Date) ||
		!jr.IntervalStartDate.Equal(andBackAgain[dhj.StartDate][0].IntervalStartDate) ||
		!jr.IntervalEndDate.Equal(andBackAgain[dhj.StartDate][0].IntervalEndDate) ||
		jr.Status != andBackAgain[dhj.StartDate][0].Status {
		t.Error("expected matching job")
	}
}

// test helper functions
func createDHM(t *testing.T) *DataHistoryManager {
	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	cp := currency.NewPair(currency.BTC, currency.USD)
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:    currency.Pairs{cp},
		Enabled:      currency.Pairs{cp},
		AssetEnabled: convert.BoolPtr(true)}
	em.Add(exch)

	exch2, err := em.NewExchangeByName("Binance")
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	cp2 := currency.NewPair(currency.BTC, currency.USDT)
	exch2.SetDefaults()
	b = exch2.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp2},
		Enabled:       currency.Pairs{cp2},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}

	em.Add(exch2)
	m := &DataHistoryManager{
		jobDB:           dataHistoryJobService{},
		jobResultDB:     dataHistoryJobResultService{},
		started:         1,
		exchangeManager: em,
		tradeChecker:    dataHistoryHasDataChecker,
		candleLoader:    dataHistoryCandleLoader,
		interval:        time.NewTicker(time.Minute),
	}
	return m
}

func TestProcessCandleData(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	_, err := m.processCandleData(nil, nil, time.Time{}, time.Time{})
	if !errors.Is(err, errNilJob) {
		t.Errorf("received %v expected %v", err, errNilJob)
	}
	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	_, err = m.processCandleData(j, nil, time.Time{}, time.Time{})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received %v expected %v", err, ErrExchangeNotFound)
	}

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.processCandleData(j, exch, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, common.ErrDateUnset)
	}

	m.candleSaver = dataHistoryCandleSaver
	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	r, err := m.processCandleData(j, fakeExchange, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
	r, err = m.processCandleData(j, exch, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
}

func TestProcessTradeData(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	_, err := m.processTradeData(nil, nil, time.Time{}, time.Time{})
	if !errors.Is(err, errNilJob) {
		t.Errorf("received %v expected %v", err, errNilJob)
	}
	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	_, err = m.processTradeData(j, nil, time.Time{}, time.Time{})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received %v expected %v", err, ErrExchangeNotFound)
	}

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.processTradeData(j, exch, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, common.ErrDateUnset)
	}
	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	m.tradeSaver = dataHistoryTradeSaver
	r, err := m.processTradeData(j, fakeExchange, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
	r, err = m.processTradeData(j, exch, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
}

func TestConvertJobTradesToCandles(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	_, err := m.convertJobTradesToCandles(nil, time.Time{}, time.Time{})
	if !errors.Is(err, errNilJob) {
		t.Errorf("received %v expected %v", err, errNilJob)
	}
	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	_, err = m.convertJobTradesToCandles(j, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, common.ErrDateUnset)
	}
	m.tradeLoader = dataHistoryTraderLoader
	m.candleSaver = dataHistoryCandleSaver
	r, err := m.convertJobTradesToCandles(j, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
}

func TestUpscaleJobCandleData(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	m.candleSaver = dataHistoryCandleSaver
	_, err := m.upscaleJobCandleData(nil, time.Time{}, time.Time{})
	if !errors.Is(err, errNilJob) {
		t.Errorf("received %v expected %v", err, errNilJob)
	}
	j := &DataHistoryJob{
		Nickname:           "",
		Exchange:           testExchange,
		Asset:              asset.Spot,
		Pair:               currency.NewPair(currency.BTC, currency.USDT),
		StartDate:          time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:            time.Now(),
		Interval:           kline.OneHour,
		ConversionInterval: kline.OneDay,
	}
	_, err = m.upscaleJobCandleData(j, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, common.ErrDateUnset)
	}

	r, err := m.upscaleJobCandleData(j, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusComplete)
	}
}

func TestValidateCandles(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	_, err := m.validateCandles(nil, nil, time.Time{}, time.Time{})
	if !errors.Is(err, errNilJob) {
		t.Errorf("received %v expected %v", err, errNilJob)
	}
	j := &DataHistoryJob{
		Nickname:  "",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-kline.OneHour.Duration() * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	_, err = m.validateCandles(j, nil, time.Time{}, time.Time{})
	if !errors.Is(err, ErrExchangeNotFound) {
		t.Errorf("received %v expected %v", err, ErrExchangeNotFound)
	}

	em := SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	exch.SetDefaults()
	fakeExchange := dhmExchange{
		IBotExchange: exch,
	}
	_, err = m.validateCandles(j, exch, time.Time{}, time.Time{})
	if !errors.Is(err, common.ErrDateUnset) {
		t.Errorf("received %v expected %v", err, common.ErrDateUnset)
	}
	j.rangeHolder, err = kline.CalculateCandleDateRanges(j.StartDate, j.EndDate, j.Interval, 1337)
	if err != nil {
		t.Error(err)
	}
	r, err := m.validateCandles(j, fakeExchange, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
	r, err = m.validateCandles(j, exch, j.StartDate, j.EndDate)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if r.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", r.Status, dataHistoryStatusFailed)
	}
}

func TestSetJobRelationship(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	err := m.SetJobRelationship("test", "123")
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}

	err = m.SetJobRelationship("", "123")
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}

	err = m.SetJobRelationship("", "")
	if !errors.Is(err, errNicknameUnset) {
		t.Errorf("received %v expected %v", err, errNicknameUnset)
	}
	m.started = 0
	err = m.SetJobRelationship("", "")
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("received %v expected %v", err, ErrSubSystemNotStarted)
	}

	m = nil
	err = m.SetJobRelationship("", "")
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("received %v expected %v", err, ErrNilSubsystem)
	}
}

func TestCompletionCheck(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	err := m.completionCheck(nil, false, false)
	if !errors.Is(err, errNilJob) {
		t.Errorf("received %v expected %v", err, errNilJob)
	}
	j := &DataHistoryJob{
		Status: dataHistoryStatusActive,
	}
	err = m.completionCheck(j, false, false)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if j.Status != dataHistoryIntervalMissingData {
		t.Errorf("received %v expected %v", j.Status, dataHistoryIntervalMissingData)
	}

	err = m.completionCheck(j, true, false)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if j.Status != dataHistoryStatusComplete {
		t.Errorf("received %v expected %v", j.Status, dataHistoryStatusComplete)
	}

	err = m.completionCheck(j, false, true)
	if !errors.Is(err, nil) {
		t.Errorf("received %v expected %v", err, nil)
	}
	if j.Status != dataHistoryStatusFailed {
		t.Errorf("received %v expected %v", j.Status, dataHistoryStatusFailed)
	}

	err = m.completionCheck(j, true, true)
	if !errors.Is(err, errJobInvalid) {
		t.Errorf("received %v expected %v", err, errJobInvalid)
	}
}

// these structs and function implementations are used
// to override database implementations as we are not testing those
// results here. see tests in the database folder
type dataHistoryJobService struct {
	datahistoryjob.IDBService
}

type dataHistoryJobResultService struct {
	datahistoryjobresult.IDBService
}

var (
	jobID     = "00a434e2-8502-4d6b-865f-e4243fd8b5a7"
	startDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	endDate   = time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
)

func (d dataHistoryJobService) Upsert(_ ...*datahistoryjob.DataHistoryJob) error {
	return nil
}

func (d dataHistoryJobService) SetRelationship(string, string, int64) error {
	return nil
}

func (d dataHistoryJobService) GetByNickName(nickname string) (*datahistoryjob.DataHistoryJob, error) {
	jc := j
	jc.Nickname = nickname
	return &jc, nil
}

func (d dataHistoryJobService) GetJobsBetween(_, _ time.Time) ([]datahistoryjob.DataHistoryJob, error) {
	jc := j
	return []datahistoryjob.DataHistoryJob{jc}, nil
}

func (d dataHistoryJobService) GetByID(id string) (*datahistoryjob.DataHistoryJob, error) {
	jc := j
	jc.ID = id
	return &jc, nil
}

func (d dataHistoryJobService) GetAllIncompleteJobsAndResults() ([]datahistoryjob.DataHistoryJob, error) {
	jc := j
	return []datahistoryjob.DataHistoryJob{jc}, nil
}

func (d dataHistoryJobService) GetJobAndAllResults(nickname string) (*datahistoryjob.DataHistoryJob, error) {
	jc := j
	jc.Nickname = nickname
	return &jc, nil
}

func (d dataHistoryJobService) GetRelatedUpcomingJobs(_ string) ([]*datahistoryjob.DataHistoryJob, error) {
	return []*datahistoryjob.DataHistoryJob{
		{
			Nickname: "test123",
			Status:   int64(dataHistoryStatusPaused),
		},
	}, nil
}

func (d dataHistoryJobResultService) Upsert(_ ...*datahistoryjobresult.DataHistoryJobResult) error {
	return nil
}

func (d dataHistoryJobResultService) GetByJobID(_ string) ([]datahistoryjobresult.DataHistoryJobResult, error) {
	return nil, nil
}

func (d dataHistoryJobResultService) GetJobResultsBetween(_ string, _, _ time.Time) ([]datahistoryjobresult.DataHistoryJobResult, error) {
	return nil, nil
}

var j = datahistoryjob.DataHistoryJob{
	ID:               jobID,
	Nickname:         "datahistoryjob",
	ExchangeName:     testExchange,
	Asset:            "spot",
	Base:             "btc",
	Quote:            "usd",
	StartDate:        startDate,
	EndDate:          endDate,
	Interval:         int64(kline.OneHour.Duration()),
	RequestSizeLimit: 3,
	MaxRetryAttempts: 3,
	BatchSize:        3,
	CreatedDate:      endDate,
	Status:           0,
	Results: []*datahistoryjobresult.DataHistoryJobResult{
		{
			ID:    jobID,
			JobID: jobID,
		},
	},
}

func dataHistoryHasDataChecker(_, _, _, _ string, irh *kline.IntervalRangeHolder) error {
	for i := range irh.Ranges {
		for j := range irh.Ranges[i].Intervals {
			irh.Ranges[i].Intervals[j].HasData = true
		}
	}
	return nil
}

func dataHistoryTraderLoader(exch, a, base, quote string, start, _ time.Time) ([]trade.Data, error) {
	cp, err := currency.NewPairFromStrings(base, quote)
	if err != nil {
		return nil, err
	}
	return []trade.Data{
		{
			Exchange:     exch,
			CurrencyPair: cp,
			AssetType:    asset.Item(a),
			Side:         order.Buy,
			Price:        1337,
			Amount:       1337,
			Timestamp:    start,
		},
	}, nil
}

func dataHistoryCandleLoader(exch string, cp currency.Pair, a asset.Item, i kline.Interval, start, _ time.Time) (kline.Item, error) {
	return kline.Item{
		Exchange: exch,
		Pair:     cp,
		Asset:    a,
		Interval: i,
		Candles: []kline.Candle{
			{
				Time:   start,
				Open:   1,
				High:   10,
				Low:    1,
				Close:  4,
				Volume: 8,
			},
		},
	}, nil
}

func dataHistoryTradeSaver(...trade.Data) error {
	return nil
}

func dataHistoryCandleSaver(_ *kline.Item, _ bool) (uint64, error) {
	return 0, nil
}

// dhmExchange aka datahistorymanager fake exchange overrides exchange functions
// we're not testing an actual exchange's implemented functions
type dhmExchange struct {
	exchange.IBotExchange
}

func (f dhmExchange) GetHistoricCandlesExtended(p currency.Pair, a asset.Item, timeStart, _ time.Time, interval kline.Interval) (kline.Item, error) {
	return kline.Item{
		Exchange: testExchange,
		Pair:     p,
		Asset:    a,
		Interval: interval,
		Candles: []kline.Candle{
			{
				Time:   timeStart,
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
			{
				Time:   timeStart.Add(interval.Duration()),
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
			{
				Time:   timeStart.Add(interval.Duration() * 2),
				Open:   1,
				High:   2,
				Low:    3,
				Close:  4,
				Volume: 5,
			},
		},
	}, nil
}

func (f dhmExchange) GetHistoricTrades(p currency.Pair, a asset.Item, startTime, endTime time.Time) ([]trade.Data, error) {
	return []trade.Data{
		{
			Exchange:     testExchange,
			CurrencyPair: p,
			AssetType:    a,
			Side:         order.Buy,
			Price:        1337,
			Amount:       4,
			Timestamp:    startTime.Add(time.Minute),
		},
		{
			Exchange:     testExchange,
			CurrencyPair: p,
			AssetType:    a,
			Side:         order.Buy,
			Price:        1338,
			Amount:       2,
			Timestamp:    startTime.Add(time.Minute * 2),
		},
	}, nil
}
