package database

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		err := testhelpers.EnableVerboseTestOutput()
		if err != nil {
			fmt.Printf("failed to enable verbose test output: %v", err)
			os.Exit(1)
		}
	}
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = os.MkdirTemp("", "gct-temp")
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
	p := currency.NewBTCUSDT()
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
	conn, err := testhelpers.ConnectToDatabase(&dbConfg)
	require.NoError(t, err)

	err = exchangeDB.InsertMany([]exchangeDB.Details{{Name: testExchange}})
	require.NoError(t, err)
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
				Time:             dInsert,
				Open:             1337,
				High:             1337,
				Low:              1337,
				Close:            1337,
				Volume:           1337,
				ValidationIssues: "hello world",
			},
		},
	}
	_, err = gctkline.StoreInDatabase(data, true)
	assert.NoError(t, err)

	_, err = LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, common.DataCandle, p, a, false)
	assert.NoError(t, err)

	if err = conn.SQL.Close(); err != nil {
		t.Error(err)
	}
}

func TestLoadDataTrades(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
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
	conn, err := testhelpers.ConnectToDatabase(&dbConfg)
	require.NoError(t, err)

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
	require.NoError(t, err)

	_, err = LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, common.DataTrade, p, a, false)
	assert.NoError(t, err)

	if err = conn.SQL.Close(); err != nil {
		t.Error(err)
	}
}

func TestLoadDataInvalid(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	dStart := time.Date(2020, 1, 0, 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, -1, p, a, false)
	assert.ErrorIs(t, err, common.ErrInvalidDataType)

	_, err = LoadData(dStart, dEnd, gctkline.FifteenMin.Duration(), exch, -1, p, a, true)
	assert.ErrorIs(t, err, errNoUSDData)
}
