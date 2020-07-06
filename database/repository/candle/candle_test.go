package candle

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/seed"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/sqlboiler/boil"
)

func TestMain(m *testing.M) {
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
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

func TestSeries(t *testing.T) {
	boil.DebugMode = true
	boil.DebugWriter = os.Stdout

	testCases := []struct {
		name   string
		config *database.Config
		seedDB func() error
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
	}{
		{
			name:   "postgresql",
			config: testhelpers.PostgresTestDatabase,
			seedDB: seedDB,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			seedDB: seedDB,
		},
	}

	for x := range testCases {
		test := testCases[x]

		t.Run(test.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(test.config)
			if err != nil {
				t.Fatal(err)
			}

			err = test.seedDB()
			if err != nil {
				t.Error(err)
			}

			start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			ret, err := Series("Binance", "BTC", "USDT", "24h", start, end)
			if err != nil {
				t.Fatal(err)
			}
			if len(ret.Tick) != 365 {
				t.Fatalf("unexpected number of results received:  %v", len(ret.Tick))
			}

			err = testhelpers.CloseDatabase(dbConn)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func seedDB() error {
	err := seed.Exchange()
	if err != nil {
		return err
	}

	return genOHCLVData()
}

func genOHCLVData() error {
	exchangeUUID, err := exchange.UUIDByName("Binance")
	if err != nil {
		return err
	}

	tempCandles := &Candle{
		ExchangeID: exchangeUUID.String(),
		Base:       "BTC",
		Quote:      "USDT",
		Interval:   "24h",
	}

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	for x := 0; x < 365; x++ {
		tempCandles.Tick = append(tempCandles.Tick, Tick{
			Timestamp: start.Add(time.Hour * 24 * time.Duration(x)),
			Open:      1000,
			High:      1000,
			Low:       1000,
			Close:     1000,
			Volume:    1000,
		})
	}

	return Insert(tempCandles)
}
