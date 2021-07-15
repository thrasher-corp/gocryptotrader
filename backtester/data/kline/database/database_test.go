package database

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	exchangeDB "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/repository/trade"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	verbose      = false
	testExchange = "binance"
)

func TestMain(m *testing.M) {
	if verbose {
		testhelpers.EnableVerboseTestOutput()
	}
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	t := m.Run()

	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestLoadDataCandles(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var err error
	bot := &engine.Engine{}
	dbConfg := database.Config{
		Enabled: true,
		Verbose: false,
		Driver:  "sqlite",
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test",
		},
	}
	bot.Config = &config.Config{
		Database: dbConfg,
	}

	err = bot.Config.CheckConfig()
	if err != nil && verbose {
		// this loads the database config to the global database
		// the errors are unrelated and likely prone to change for reasons that
		// this test does not need to care about

		// so we only log the error if verbose
		t.Log(err)
	}
	database.MigrationDir = filepath.Join("..", "..", "..", "..", "database", "migrations")
	testhelpers.MigrationDir = filepath.Join("..", "..", "..", "..", "database", "migrations")
	_, err = testhelpers.ConnectToDatabase(&dbConfg)
	if err != nil {
		t.Error(err)
	}

	bot.DatabaseManager, err = engine.SetupDatabaseConnectionManager(&bot.Config.Database)
	if err != nil {
		t.Error(err)
	}
	err = bot.DatabaseManager.Start(&bot.ServicesWG)
	if err != nil {
		t.Error(err)
	}

	err = exchangeDB.InsertMany([]exchangeDB.Details{{Name: testExchange}})
	if err != nil {
		t.Fatal(err)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	data := &gctkline.Item{
		Exchange: exch,
		Pair:     p,
		Asset:    a,
		Interval: gctkline.FifteenMin,
		Candles: []gctkline.Candle{
			{
				Time:   dInsert,
				Open:   1337,
				High:   1337,
				Low:    1337,
				Close:  1337,
				Volume: 1337,
			},
		},
	}
	_, err = gctkline.StoreInDatabase(data, true)
	if err != nil {
		t.Error(err)
	}

	_, err = LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, common.DataCandle, p, a)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataTrades(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var err error
	bot := &engine.Engine{}
	dbConfg := database.Config{
		Enabled: true,
		Verbose: false,
		Driver:  "sqlite",
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Database: "test",
		},
	}
	bot.Config = &config.Config{
		Database: dbConfg,
	}

	err = bot.Config.CheckConfig()
	if err != nil && verbose {
		// this loads the database config to the global database
		// the errors are unrelated and likely prone to change for reasons that
		// this test does not need to care about

		// so we only log the error if verbose
		t.Log(err)
	}
	database.MigrationDir = filepath.Join("..", "..", "..", "..", "database", "migrations")
	testhelpers.MigrationDir = filepath.Join("..", "..", "..", "..", "database", "migrations")
	_, err = testhelpers.ConnectToDatabase(&dbConfg)
	if err != nil {
		t.Error(err)
	}

	bot.DatabaseManager, err = engine.SetupDatabaseConnectionManager(&bot.Config.Database)
	if err != nil {
		t.Error(err)
	}
	err = bot.DatabaseManager.Start(&bot.ServicesWG)
	if err != nil {
		t.Error(err)
	}

	err = exchangeDB.InsertMany([]exchangeDB.Details{{Name: testExchange}})
	if err != nil {
		t.Fatal(err)
	}
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dInsert := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	err = trade.Insert(trade.Data{
		ID:        "123",
		TID:       "123",
		Exchange:  exch,
		Base:      p.Base.String(),
		Quote:     p.Quote.String(),
		AssetType: a.String(),
		Price:     1337,
		Amount:    1337,
		Side:      gctorder.Buy.String(),
		Timestamp: dInsert,
	})
	if err != nil {
		t.Error(err)
	}

	_, err = LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, common.DataTrade, p, a)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataInvalid(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, -1, p, a)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("expected '%v' received '%v'", err, common.ErrInvalidDataType)
	}
}
