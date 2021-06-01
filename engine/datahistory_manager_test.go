package engine

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestSetupDataHistoryManager(t *testing.T) {
	_, err := SetupDataHistoryManager(nil, nil, nil)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), nil, nil)
	if !errors.Is(err, errNilDatabaseConnectionManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilDatabaseConnectionManager)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), &database.Instance{}, nil)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), &database.Instance{}, &config.DataHistoryManager{})
	if !errors.Is(err, database.ErrDatabaseNotConnected) {
		t.Errorf("error '%v', expected '%v'", err, database.ErrDatabaseNotConnected)
	}

	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
}

func TestDataHistoryManagerIsRunning(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
	if m.IsRunning() {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
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
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}

	err = m.Start()
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
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
	err = m.Start()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.Stop()
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

func setupDataHistoryManagerTest(t *testing.T) (*DataHistoryManager, *Engine) {
	t.Helper()
	engerino := RPCTestSetup(t)
	exch := engerino.ExchangeManager.GetExchangeByName(testExchange)
	cp := currency.NewPair(currency.BTC, currency.USD)
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{Available: currency.Pairs{cp}}
	engerino.ExchangeManager.Add(exch)
	m, err := SetupDataHistoryManager(engerino.ExchangeManager, engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 1)
	return m, engerino
}

func TestUpsertJob(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

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

	dhj.Pair = currency.NewPair(currency.BTC, currency.USD)
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, errExchangeNotLoaded) {
		t.Errorf("error '%v', expected '%v'", err, errExchangeNotLoaded)
	}

	dhj.Exchange = strings.ToLower(testExchange)
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, errInvalidTimes) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidTimes)
	}

	dhj.StartDate = time.Now().Add(-time.Hour)
	dhj.EndDate = time.Now()

	err = m.UpsertJob(dhj, false)
	if err == nil {
		t.Error("expected error")
	}

	dhj.Interval = kline.OneHour
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 1 {
		t.Error("unexpected jerrb")
	}

	startDate := time.Date(1980, 1, 1, 1, 0, 0, 0, time.UTC)

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
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if !m.jobs[0].StartDate.Equal(startDate) {
		t.Error(err)
	}
}

func TestDeleteJob(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	dhj := &DataHistoryJob{
		Nickname:  "TestDeleteJob",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
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
	jerb, err := m.jobDB.GetJobAndAllResults(dhj.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if jerb.Status != int64(dataHistoryStatusRemoved) {
		t.Error("expected removed")
	}

	err = m.DeleteJob("", dhj.ID.String())
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.DeleteJob("1337", "")
	if err == nil {
		t.Error("expected no results")
	}

	err = m.DeleteJob("", "1337")
	if err == nil {
		t.Error("expected no results")
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
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	dhj := &DataHistoryJob{
		Nickname:  "TestGetByNickname",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
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

	_, err = m.GetByNickname("test123", false)
	if !errors.Is(err, errJobNotFound) {
		t.Errorf("error '%v', expected '%v'", err, errJobNotFound)
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
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	dhj := &DataHistoryJob{
		Nickname:  "TestGetByID",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	_, err = m.GetByID(dhj.ID.String())
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	_, err = m.GetByID(dhj.Nickname)
	if !errors.Is(err, errJobNotFound) {
		t.Errorf("error '%v', expected '%v'", err, errJobNotFound)
	}

	m.jobs = []*DataHistoryJob{}
	_, err = m.GetByID(dhj.ID.String())
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	atomic.StoreInt32(&m.started, 0)
	_, err = m.GetByID(dhj.Nickname)
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	_, err = m.GetByID(dhj.Nickname)
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestRetrieveJobs(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	jobs, err := m.retrieveJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 0 {
		t.Error("expected no jobs")
	}

	dhj := &DataHistoryJob{
		Nickname:  "TestRetrieveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	jobs, err = m.retrieveJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Error("expected a job")
	}

	engerino.DatabaseManager.dbConn.SetConnected(false)
	_, err = m.retrieveJobs()
	if !errors.Is(err, errDatabaseConnectionRequired) {
		t.Errorf("error '%v', expected '%v'", err, errDatabaseConnectionRequired)
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
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

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
		StartDate: time.Now().Add(-time.Second),
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
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)
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
	if !errors.Is(err, errCurrencyPairInvalid) {
		t.Errorf("error '%v', expected '%v'", err, errCurrencyPairInvalid)
	}

	dhj.Pair = currency.NewPair(currency.BTC, currency.USD)
	err = m.validateJob(dhj)
	if !errors.Is(err, errInvalidTimes) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidTimes)
	}

	dhj.StartDate = time.Now().Add(time.Minute)
	dhj.EndDate = time.Now().Add(time.Hour)
	err = m.validateJob(dhj)
	if !errors.Is(err, errInvalidTimes) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidTimes)
	}

	dhj.StartDate = time.Now().Add(-time.Hour)
	dhj.EndDate = time.Now().Add(-time.Minute)
	err = m.validateJob(dhj)
	if !errors.Is(err, kline.ErrUnsupportedInterval) {
		t.Errorf("error '%v', expected '%v'", err, kline.ErrUnsupportedInterval)
	}

	dhj.Interval = kline.OneDay
	err = m.validateJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestPrepareJobs(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	dhj := &DataHistoryJob{
		Nickname:  "TestPrepareJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	jobs, err := m.PrepareJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Error("expected 1 job")
	}

	atomic.StoreInt32(&m.started, 0)
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
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	dhj := &DataHistoryJob{
		Nickname:  "TestCompareJobsToData",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.compareJobsToData(dhj)
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

	atomic.StoreInt32(&m.started, 0)
	err = m.compareJobsToData()
	if !errors.Is(err, ErrSubSystemNotStarted) {
		t.Errorf("error '%v', expected '%v'", err, ErrSubSystemNotStarted)
	}

	m = nil
	err = m.compareJobsToData()
	if !errors.Is(err, ErrNilSubsystem) {
		t.Errorf("error '%v', expected '%v'", err, ErrNilSubsystem)
	}
}

func TestRunJob(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(engerino.ExchangeManager, engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Fatal("expected manager")
	}
	exch := engerino.ExchangeManager.GetExchangeByName("Binance")
	cp := currency.NewPair(currency.BTC, currency.USDT)
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{Available: currency.Pairs{cp}, Enabled: currency.Pairs{cp}}
	b.CurrencyPairs.Pairs[asset.Spot].AssetEnabled = convert.BoolPtr(true)
	b.CurrencyPairs.Pairs[asset.Spot].ConfigFormat = &currency.PairFormat{
		Uppercase: true,
	}
	b.CurrencyPairs.Pairs[asset.Spot].RequestFormat = &currency.PairFormat{
		Uppercase: true,
	}
	engerino.ExchangeManager.Add(exch)
	atomic.StoreInt32(&m.started, 1)

	dhj := &DataHistoryJob{
		Nickname:  "TestProcessJobs",
		Exchange:  "Binance",
		Asset:     asset.Spot,
		Pair:      cp,
		StartDate: time.Now().Add(-time.Hour * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	err = m.UpsertJob(dhj, false)
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
		Pair:      cp,
		StartDate: time.Now().Add(-time.Minute * 5),
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
}

func TestRunJobs(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)
	dhj := &DataHistoryJob{
		Nickname:  "TestProcessJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Hour * 2),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.runJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestConverters(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, &config.DataHistoryManager{})
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}

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

	dbJob, err := m.convertJobToDBModel(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

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

func TestGenerateJobSummary(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)
	bn := "binance"
	exch := engerino.ExchangeManager.GetExchangeByName(bn)
	cp := currency.NewPair(currency.BTC, currency.USDT)
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{Available: currency.Pairs{cp}, Enabled: currency.Pairs{cp}}
	b.CurrencyPairs.Pairs[asset.Spot].AssetEnabled = convert.BoolPtr(true)
	b.CurrencyPairs.Pairs[asset.Spot].ConfigFormat = &currency.PairFormat{
		Uppercase: true,
	}
	b.CurrencyPairs.Pairs[asset.Spot].RequestFormat = &currency.PairFormat{
		Uppercase: true,
	}
	engerino.ExchangeManager.Add(exch)

	dhj := &DataHistoryJob{
		Nickname:  "TestGenerateJobSummary",
		Exchange:  bn,
		Asset:     asset.Spot,
		Pair:      cp,
		StartDate: time.Now().Add(-time.Minute),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	m.jobs, err = m.PrepareJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	err = m.runJob(dhj)
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
}

func TestGetAllJobStatusBetween(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil || engerino == nil {
		t.Fatal("expected non nil setup")
	}
	defer CleanRPCTest(t, engerino)

	dhj := &DataHistoryJob{
		Nickname:  "TestGetActiveJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Second),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	jobs, err := m.GetAllJobStatusBetween(time.Now().Add(-time.Minute), time.Now().Add(time.Minute))
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 1 {
		t.Error("expected 1 job")
	}

	jobs, err = m.GetAllJobStatusBetween(time.Now().Add(-time.Hour), time.Now().Add(-time.Minute*30))
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(jobs) != 0 {
		t.Error("expected 0 jobs")
	}
}
