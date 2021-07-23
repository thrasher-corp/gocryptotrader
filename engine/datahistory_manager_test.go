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
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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

func TestDeleteJob(t *testing.T) {
	t.Parallel()
	m := createDHM(t)
	dhj := &DataHistoryJob{
		Nickname:  "TestDeleteJob",
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

	err = m.DeleteJob("", "")
	if !errors.Is(err, errNicknameIDUnset) {
		t.Errorf("error '%v', expected '%v'", err, errNicknameIDUnset)
	}

	err = m.DeleteJob("1337", "1337")
	if !errors.Is(err, errOnlyNicknameOrID) {
		t.Errorf("error '%v', expected '%v'", err, errOnlyNicknameOrID)
	}

	err = m.DeleteJob(dhj.Nickname, "")
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 0 {
		t.Error("expected 0")
	}
	err = m.DeleteJob("", dhj.ID.String())
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	atomic.StoreInt32(&m.started, 0)
	err = m.DeleteJob("", dhj.ID.String())
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	err = m.DeleteJob("", dhj.ID.String())
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
		t.Fatalf("error '%v', expected '%v'", err, nil)
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
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	cp2 := currency.NewPair(currency.BTC, currency.USDT)
	exch2.SetDefaults()
	b = exch2.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:    currency.Pairs{cp2},
		Enabled:      currency.Pairs{cp2},
		AssetEnabled: convert.BoolPtr(true),
		ConfigFormat: &currency.PairFormat{Uppercase: true}}
	em.Add(exch2)
	m := &DataHistoryManager{
		jobDB:           dataHistoryJobService{},
		jobResultDB:     dataHistoryJobResultService{},
		started:         1,
		exchangeManager: em,
		tradeLoader:     dataHistoryTradeLoader,
		candleLoader:    dataHistoryCandleLoader,
		interval:        time.NewTicker(time.Minute),
	}
	return m
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

func dataHistoryTradeLoader(_, _, _, _ string, irh *kline.IntervalRangeHolder) error {
	for i := range irh.Ranges {
		for j := range irh.Ranges[i].Intervals {
			irh.Ranges[i].Intervals[j].HasData = true
		}
	}
	return nil
}

func dataHistoryCandleLoader(string, currency.Pair, asset.Item, kline.Interval, time.Time, time.Time) (kline.Item, error) {
	return kline.Item{}, nil
}
