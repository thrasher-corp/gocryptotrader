package engine

import (
	"errors"
	"strings"
	"testing"
	"time"

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

func TestUpsertJob(t *testing.T) {
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
	err = m.UpsertJob(nil, false)
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

	dhj.Pair = currency.NewPair(currency.BTC, currency.USDT)
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

	dhj.Exchange = strings.ToLower(testExchange)
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, kline.ErrUnsetInterval) {
		t.Errorf("error '%v', expected '%v'", err, kline.ErrUnsetInterval)
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
		t.Error("unexpected jerrb")
	}
	if !m.jobs[0].StartDate.Equal(startDate) {
		t.Errorf("received '%v', expected '%v'", m.jobs[0].StartDate, startDate)
	}

	err = m.UpsertJob(dhj, true)
	if !errors.Is(err, errNicknameInUse) {
		t.Errorf("error '%v', expected '%v'", err, errNicknameInUse)
	}

	newJob := &DataHistoryJob{
		Nickname:         "test123",
		Exchange:         testExchange,
		Asset:            asset.Spot,
		Pair:             currency.NewPair(currency.BTC, currency.USDT),
		StartDate:        time.Now(),
		EndDate:          time.Now().Add(time.Second),
		Interval:         kline.OneMin,
		BatchSize:        1337,
		RequestSizeLimit: 100,
		DataType:         1,
		MaxRetryAttempts: 1,
	}
	err = m.UpsertJob(newJob, true)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestDeleteJob(t *testing.T) {
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
	dhj := &DataHistoryJob{
		Nickname:  "TestDeleteJob",
		Exchange:  testExchange,
		Asset:     asset.Spot,
		Pair:      currency.NewPair(currency.BTC, currency.USDT),
		StartDate: time.Now().Add(-time.Second),
		EndDate:   time.Now(),
		Interval:  kline.OneMin,
	}
	err = m.UpsertJob(dhj, false)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
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
}
