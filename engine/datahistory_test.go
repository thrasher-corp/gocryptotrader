package engine

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestSetupDataHistoryManager(t *testing.T) {
	_, err := SetupDataHistoryManager(nil, nil, 0)
	if !errors.Is(err, errNilExchangeManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilConfig)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), nil, 0)
	if !errors.Is(err, errNilDatabaseConnectionManager) {
		t.Errorf("error '%v', expected '%v'", err, errNilDatabaseConnectionManager)
	}

	_, err = SetupDataHistoryManager(SetupExchangeManager(), &database.Instance{}, 0)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, errDatabaseConnectionRequired)
	}

	m, err := SetupDataHistoryManager(SetupExchangeManager(), &database.Instance{}, time.Second)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
	}
}

func TestDataHistoryManagerIsRunning(t *testing.T) {
	m, err := SetupDataHistoryManager(SetupExchangeManager(), &database.Instance{}, time.Second)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
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
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, time.Second)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
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
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, time.Second)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if m == nil {
		t.Error("expected manager")
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
	m, err := SetupDataHistoryManager(engerino.ExchangeManager, engerino.DatabaseManager.dbConn, time.Second)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	atomic.StoreInt32(&m.started, 1)
	return m, engerino
}

func TestUpsertJob(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil {
		t.Error("expected manager")
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

	startDate := time.Date(1980, 1, 1, 1, 1, 1, 1, time.UTC)
	dhj.StartDate = startDate
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 1 {
		t.Fatal("unexpected jerrb")
	}
	if !m.jobs[0].StartDate.Equal(startDate) {
		t.Errorf("received '%v', expected '%v'", m.jobs[0].StartDate, startDate)
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
		StartDate:        time.Now().Add(-time.Hour),
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
}

func TestDeleteJob(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil {
		t.Error("expected manager")
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
	if jerb.Status != dataHistoryStatusRemoved {
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
	if m == nil {
		t.Error("expected manager")
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
	if m == nil {
		t.Error("expected manager")
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
	if m == nil {
		t.Error("expected manager")
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
	if m == nil {
		t.Error("expected manager")
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
	if m == nil {
		t.Error("expected manager")
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
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestPrepareJobs(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil {
		t.Error("expected manager")
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
	if m == nil {
		t.Error("expected manager")
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
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil {
		t.Error("expected manager")
	}
	defer CleanRPCTest(t, engerino)
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

	dhj := &DataHistoryJob{
		Nickname:  "TestProcessJobs",
		Exchange:  "Binance",
		Asset:     asset.Spot,
		Pair:      cp,
		StartDate: time.Now().Add(-time.Hour * 24 * 7),
		EndDate:   time.Now(),
		Interval:  kline.OneDay,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	_, err = m.runJob(dhj, exch)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj.Pair = currency.NewPair(currency.DOGE, currency.USDT)
	_, err = m.runJob(dhj, exch)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj.DataType = dataHistoryTradeDataType
	dhj.Interval = kline.OneMin
	dhj.StartDate = time.Now().Add(-time.Minute)
	dhj.EndDate = time.Now()
	dhj.Pair = currency.NewPair(currency.BTC, currency.USDT)
	err = m.compareJobsToData(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	_, err = m.runJob(dhj, exch)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestProcessJobs(t *testing.T) {
	m, engerino := setupDataHistoryManagerTest(t)
	if m == nil {
		t.Error("expected manager")
	}
	defer CleanRPCTest(t, engerino)
	dhj := &DataHistoryJob{
		Nickname:  "TestProcessJobs",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USD),
		StartDate: time.Now().Add(-time.Hour * 24),
		EndDate:   time.Now(),
		Interval:  kline.OneHour,
	}
	err := m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.processJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	dhj.Exchange = "Binance"
	err = m.processJobs()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestConverters(t *testing.T) {
	engerino := RPCTestSetup(t)
	defer CleanRPCTest(t, engerino)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), engerino.DatabaseManager.dbConn, time.Second)
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

	if dhj.ID.String() != dbJob[0].ID ||
		dhj.Nickname != dbJob[0].Nickname ||
		!dhj.StartDate.Equal(dbJob[0].StartDate) ||
		int64(dhj.Interval.Duration()) != dbJob[0].Interval ||
		dhj.Pair.Base.String() != dbJob[0].Base ||
		dhj.Pair.Quote.String() != dbJob[0].Quote {
		t.Error("expected matching job")
	}

	convertBack, err := m.convertDBModelToJob(*dbJob[0])
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if dhj.ID != convertBack[0].ID ||
		dhj.Nickname != convertBack[0].Nickname ||
		!dhj.StartDate.Equal(convertBack[0].StartDate) ||
		dhj.Interval != convertBack[0].Interval ||
		!dhj.Pair.Equal(convertBack[0].Pair) {
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
		jr.Status != result[0].Status {
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
