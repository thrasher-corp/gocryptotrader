package candle

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

var (
	verbose = false

	testExchanges = []exchange.Details{
		{
			Name: "one",
		},
		{
			Name: "two",
		},
	}
)

func TestMain(m *testing.M) {
	if verbose {
		testhelpers.EnableVerboseTestOutput()
	}

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

func TestInsert(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		seedDB func(includeOHLCVData bool) error
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

			dbConn, err := testhelpers.ConnectToDatabase(test.config, true)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				err = testhelpers.CloseDatabase(dbConn)
				if err != nil {
					t.Error(err)
				}
			})

			if test.seedDB != nil {
				err = test.seedDB(false)
				if err != nil {
					t.Fatal(err)
				}
			}

			data, err := genOHCLVData()
			if err != nil {
				t.Fatal(err)
			}

			err = Insert(&data)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSeries(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		seedDB func(includeOHLCVData bool) error
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

			dbConn, err := testhelpers.ConnectToDatabase(test.config, true)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				err = testhelpers.CloseDatabase(dbConn)
				if err != nil {
					t.Error(err)
				}
			})

			if test.seedDB != nil {
				err = test.seedDB(true)
				if err != nil {
					t.Fatal(err)
				}
			}

			start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			ret, err := Series(testExchanges[0].Name, "BTC", "USDT", "24h", start, end)
			if err != nil {
				t.Fatal(err)
			}
			if len(ret.Tick) != 365 {
				t.Errorf("unexpected number of results received:  %v", len(ret.Tick))
			}
		})
	}
}

func seedDB(includeOHLCVData bool) error {
	err := exchange.Seed(testExchanges)
	if err != nil {
		return err
	}

	if includeOHLCVData {
		exchange.ResetExchangeCache()
		data, err := genOHCLVData()
		if err != nil {
			return err
		}
		return Insert(&data)
	}
	return nil
}

func genOHCLVData() (out Candle, err error) {
	exchangeUUID, err := exchange.UUIDByName(testExchanges[0].Name)
	if err != nil {
		return
	}
	out.ExchangeID = exchangeUUID.String()
	out.Base = currency.BTC.String()
	out.Quote = currency.USDT.String()
	out.Interval = "24h"

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	for x := 0; x < 365; x++ {
		out.Tick = append(out.Tick, Tick{
			Timestamp: start.Add(time.Hour * 24 * time.Duration(x)),
			Open:      1000,
			High:      1000,
			Low:       1000,
			Close:     1000,
			Volume:    1000,
		})
	}

	return out, nil
}
