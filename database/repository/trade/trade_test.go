package trade

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
		testhelpers.EnableVerboseTestOutput()
	}
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = ioutil.TempDir("", "gct-temp")
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestTrades(t *testing.T) {
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

			if test.seedDB != nil {
				err = test.seedDB()
				if err != nil {
					t.Error(err)
				}
			}

			tradeSQLTester(t)
			err = testhelpers.CloseDatabase(dbConn)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func tradeSQLTester(t *testing.T) {
	var trades, trades2 []Data
	firstTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		uu, _ := uuid.NewV4()
		trades = append(trades, Data{
			ID:        uu.String(),
			Timestamp: firstTime.Add(time.Minute * time.Duration(i+1)),
			Exchange:  testExchanges[0].Name,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
			AssetType: asset.Spot.String(),
			Price:     float64(i * (i + 3)),
			Amount:    float64(i * (i + 2)),
			Side:      order.Buy.String(),
			TID:       fmt.Sprintf("%v", i),
		})
	}
	err := Insert(trades...)
	if err != nil {
		t.Fatal(err)
	}
	// insert the same trades to test conflict resolution
	for i := 0; i < 20; i++ {
		uu, _ := uuid.NewV4()
		trades2 = append(trades2, Data{
			ID:        uu.String(),
			Timestamp: firstTime.Add(time.Minute * time.Duration(i+1)),
			Exchange:  testExchanges[0].Name,
			Base:      currency.BTC.String(),
			Quote:     currency.USD.String(),
			AssetType: asset.Spot.String(),
			Price:     float64(i * (i + 3)),
			Amount:    float64(i * (i + 2)),
			Side:      order.Buy.String(),
			TID:       fmt.Sprintf("%v", i),
		})
	}
	err = Insert(trades2...)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := GetInRange(
		testExchanges[0].Name,
		asset.Spot.String(),
		currency.BTC.String(),
		currency.USD.String(),
		firstTime.Add(-time.Hour),
		firstTime.Add(time.Hour),
	)
	if err != nil {
		t.Error(err)
	}
	if len(resp) != 20 {
		t.Fatalf("unique constraints failing, got %v", resp)
	}

	v, err := GetInRange(
		testExchanges[0].Name,
		asset.Spot.String(),
		currency.BTC.String(),
		currency.USD.String(),
		firstTime.Add(-time.Hour),
		firstTime.Add(time.Hour))
	if err != nil {
		t.Error(err)
	}
	if len(v) == 0 {
		t.Error("Bad get!")
	}

	ranges, err := kline.CalculateCandleDateRanges(firstTime, firstTime.Add(20*time.Minute), kline.OneMin, 100)
	if err != nil {
		t.Error(err)
	}
	err = VerifyTradeInIntervals(testExchanges[0].Name,
		asset.Spot.String(),
		currency.BTC.String(),
		currency.USD.String(),
		ranges)
	if err != nil {
		t.Error(err)
	}

	err = DeleteTrades(trades...)
	if err != nil {
		t.Error(err)
	}
	err = DeleteTrades(trades2...)
	if err != nil {
		t.Error(err)
	}

	v, err = GetInRange(
		testExchanges[0].Name,
		asset.Spot.String(),
		currency.BTC.String(),
		currency.USD.String(),
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour))
	if err != nil {
		t.Error(err)
	}
	if len(v) != 0 {
		t.Errorf("should all be ded %v", v)
	}
}

func seedDB() error {
	err := exchange.InsertMany(testExchanges)
	if err != nil {
		return err
	}

	return nil
}
