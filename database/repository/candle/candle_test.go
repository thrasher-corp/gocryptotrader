package candle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

var (
	verbose       = false
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
		err := testhelpers.EnableVerboseTestOutput()
		if err != nil {
			fmt.Printf("failed to enable verbose test output: %v", err)
			os.Exit(1)
		}
	}

	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
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

func TestInsert(t *testing.T) {
	for _, tc := range []struct {
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB(false))
			}

			data, err := genOHCLVData()
			require.NoError(t, err)
			r, err := Insert(&data)
			require.NoError(t, err)

			assert.Equal(t, uint64(365), r)

			d, err := DeleteCandles(&data)
			require.NoError(t, err)
			assert.Equal(t, int64(365), d)
			assert.NoError(t, testhelpers.CloseDatabase(dbConn))
		})
	}
}

func TestInsertFromCSV(t *testing.T) {
	for _, tc := range []struct {
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB(false))
			}

			exchange.ResetExchangeCache()
			testFile := filepath.Join("..", "..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv")
			count, err := InsertFromCSV(testExchanges[0].Name, "BTC", "USDT", 86400, "spot", testFile)
			require.NoError(t, err)
			assert.Equal(t, uint64(365), count)

			assert.NoError(t, testhelpers.CloseDatabase(dbConn))
		})
	}
}

func TestSeries(t *testing.T) {
	for _, tc := range []struct {
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB(true))
			}

			start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
			ret, err := Series(testExchanges[0].Name, "BTC", "USDT", 86400, "spot", start, end)
			require.NoError(t, err)
			assert.Equal(t, 365, len(ret.Candles))

			_, err = Series("", "", "", 0, "", start, end)
			require.ErrorIs(t, err, errInvalidInput)

			_, err = Series(testExchanges[0].Name, "BTC", "MOON", 864000, "spot", start, end)
			assert.ErrorIs(t, err, ErrNoCandleDataFound)
			assert.NoError(t, testhelpers.CloseDatabase(dbConn))
		})
	}
}

func seedDB(includeOHLCVData bool) error {
	err := exchange.InsertMany(testExchanges)
	if err != nil {
		return err
	}

	if includeOHLCVData {
		exchange.ResetExchangeCache()
		data, err := genOHCLVData()
		if err != nil {
			return err
		}
		_, err = Insert(&data)
		return err
	}
	return nil
}

func genOHCLVData() (out Item, err error) {
	exchangeUUID, err := exchange.UUIDByName(testExchanges[0].Name)
	if err != nil {
		return
	}
	out.ExchangeID = exchangeUUID.String()
	out.Base = currency.BTC.String()
	out.Quote = currency.USDT.String()
	out.Interval = 86400
	out.Asset = "spot"

	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	for x := range 365 {
		out.Candles = append(out.Candles, Candle{
			Timestamp:        start.Add(time.Hour * 24 * time.Duration(x)),
			Open:             1000,
			High:             1000,
			Low:              1000,
			Close:            1000,
			Volume:           1000,
			ValidationIssues: "hello world!",
		})
	}

	return out, nil
}
