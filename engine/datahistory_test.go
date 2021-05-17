package engine

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	dbexchange "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/goose"
)

func TestSetupDataHistoryManager(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func setuptDataConnectionManagerTest(t *testing.T) (*DatabaseConnectionManager, error) {
	dcm, err := SetupDatabaseConnectionManager(&database.Config{
		Enabled: true,
		Driver:  database.DBSQLite3,
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: databaseName,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	err = dcm.Start(&wg)
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join("..", databaseFolder, migrationsFolder)
	err = goose.Run("up", database.DB.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Fatalf("failed to run migrations %v", err)
	}
	uuider, _ := uuid.NewV4()
	err = dbexchange.Insert(dbexchange.Details{Name: testExchange, UUID: uuider})
	if err != nil {
		t.Fatalf("failed to insert exchange %v", err)
	}
	return dcm, err
}

func cleanupDataHistoryTest(dcm *DatabaseConnectionManager, t *testing.T) {
	err := dcm.Stop()
	if err != nil {
		t.Error(err)
	}
	cfg := dcm.dbConn.GetConfig()
	err = os.Remove(cfg.Database)
	if err != nil {
		t.Error(err)
	}

}

func TestDataHistoryManagerStart(t *testing.T) {
	dcm, err := setuptDataConnectionManagerTest(t)
	defer cleanupDataHistoryTest(dcm, t)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), dcm.dbConn, time.Second)
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

	err = dcm.Stop()
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
}

func TestDataHistoryManagerStop(t *testing.T) {
	dcm, err := setuptDataConnectionManagerTest(t)
	defer cleanupDataHistoryTest(dcm, t)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), dcm.dbConn, time.Second)
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
	dcm, err := setuptDataConnectionManagerTest(t)
	defer cleanupDataHistoryTest(dcm, t)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), dcm.dbConn, time.Second)
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
	err = m.UpsertJob(nil)
	if !errors.Is(err, errNilJob) {
		t.Errorf("error '%v', expected '%v'", err, errNilJob)
	}
	dhj := &DataHistoryJob{}
	err = m.UpsertJob(dhj)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("error '%v', expected '%v'", err, asset.ErrNotSupported)
	}

	dhj.Asset = asset.Spot
	err = m.UpsertJob(dhj)
	if !errors.Is(err, errCurrencyPairUnset) {
		t.Errorf("error '%v', expected '%v'", err, errCurrencyPairUnset)
	}

	dhj.Pair = currency.NewPair(currency.BTC, currency.USDT)
	err = m.UpsertJob(dhj)
	if !errors.Is(err, errInvalidTimes) {
		t.Errorf("error '%v', expected '%v'", err, errInvalidTimes)
	}

	dhj.StartDate = time.Now().Add(-time.Hour)
	dhj.EndDate = time.Now()

	err = m.UpsertJob(dhj)
	if err == nil {
		t.Error("expected error")
	}

	dhj.Exchange = strings.ToLower(testExchange)
	err = m.UpsertJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 1 {
		t.Error("unexpected jerrb")
	}

	startDate := time.Date(1980, 1, 1, 1, 1, 1, 1, time.UTC)
	dhj.StartDate = startDate
	err = m.UpsertJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 1 {
		t.Error("unexpected jerrb")
	}
	if !m.jobs[0].StartDate.Equal(startDate) {
		t.Errorf("received '%v', expected '%v'", m.jobs[0].StartDate, startDate)
	}
}

func TestDeleteJob(t *testing.T) {
	dcm, err := setuptDataConnectionManagerTest(t)
	if !errors.Is(err, nil) {
		t.Fatalf("error '%v', expected '%v'", err, nil)
	}
	//defer cleanupDataHistoryTest(dcm, t)
	m, err := SetupDataHistoryManager(SetupExchangeManager(), dcm.dbConn, time.Second)
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
	}
	err = m.UpsertJob(dhj)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}

	err = m.RemoveJob(dhj.Nickname)
	if !errors.Is(err, nil) {
		t.Errorf("error '%v', expected '%v'", err, nil)
	}
	if len(m.jobs) != 0 {
		t.Error("expected 0")
	}
	if dhj.Status != dataHistoryStatusRemoved {
		t.Error("expected removed")
	}

	jerb, err := m.jobDB.GetJobAndAllResults(dhj.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if jerb.Status != dataHistoryStatusRemoved {
		t.Error("expected removed")
	}
}
