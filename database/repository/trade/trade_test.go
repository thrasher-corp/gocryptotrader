package trade

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/goose"
	"github.com/thrasher-corp/sqlboiler/boil"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestMain(m *testing.M) {
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	boil.DebugMode = true

	t := m.Run()
	os.Exit(t)
}

func TestTrades(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
		output interface{}
	}{
		{
			"SQLite-Write",
			&database.Config{
				Driver: database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{
						Host: "localhost",
						Port: 5432,
						Username: "postgres",
						Password: "postgres",
						Database: "trades.db",
						SSLMode: "disable",
					},
			},
			tradeTester4000,
			testhelpers.CloseDatabase,
			nil,
		},
		{
			"Postgres-Write",
			testhelpers.PostgresTestDatabase,
			tradeTester4000,
			nil,
			nil,
		},
	}

		for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(test.config)
			if err != nil {
				t.Fatal(err)
			}

			path := filepath.Join("..", "..", "migrations")
			err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
			if err != nil {
				t.Fatalf("failed to run migrations %v", err)
			}

			if test.runner != nil {
				test.runner(t)
			}

			if test.closer != nil {
				err = test.closer(dbConn)
				if err != nil {
					t.Log(err)
				}
			}
		})
	}
}


func tradeTester4000(t *testing.T) {
	var trades []Data
	cp, _ := currency.NewPairFromString("BTC-USD")
	for i := 0; i < 20; i++ {
		uu, _ := uuid.NewV4()
		trades = append(trades, Data{
			ID:           uu.String(),
			Timestamp:    time.Now().Unix(),
			Exchange:     "Binance",
			CurrencyPair: cp.String(),
			AssetType:    asset.Spot.String(),
			Price:        float64(i * (i + 3)),
			Amount:        float64(i * (i + 2)),
			Side:         order.Buy.String(),
		})
	}
	err := Insert(trades...)
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetByUUID(trades[0].ID)
	if err != nil {
			t.Error(err)
	}

	v, err := GetByExchangeInRange("Binance", time.Now().Add(-time.Hour).Unix(), time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Error(err)
	}
	if len(v) == 0 {
		t.Error("Bad get!")
	}

	err = DeleteTrades(trades...)
	if err != nil {
		t.Error(err)
	}

	v, err = GetByExchangeInRange("Binance", time.Now().Add(-time.Hour).Unix(), time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Error(err)
	}
	if len(v) != 0 {
		t.Errorf("should all be ded %v", v)
	}
}
