package withdraw

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/goose"
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

func TestWithdraw(t *testing.T) {
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
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			withdrawHelper,
			testhelpers.CloseDatabase,
			nil,
		},
		{
			"Postgres-Write",
			testhelpers.PostgresTestDatabase,
			withdrawHelper,
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

func withdrawHelper(t *testing.T) {
	t.Helper()
	var wg sync.WaitGroup
	for x := 0; x < 20; x++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			resp := &withdraw.Response{
				Exchange: &withdraw.ExchangeResponse{
					Name:   test,
					ID:     test,
					Status: test,
				},
				RequestDetails: &withdraw.Request{
					Exchange:    test,
					Description: test,
					Amount:      1.0,
				},
			}
			rnd := rand.Intn(2)
			if rnd == 0 {
				resp.RequestDetails.Currency = currency.AUD
				resp.RequestDetails.Type = 1
				resp.RequestDetails.Fiat = new(withdraw.FiatRequest)
				resp.RequestDetails.Fiat.Bank = new(banking.Account)
			} else {
				resp.RequestDetails.Currency = currency.BTC
				resp.RequestDetails.Type = 0
				resp.RequestDetails.Crypto = new(withdraw.CryptoRequest)
				resp.RequestDetails.Crypto.Address = test
				resp.RequestDetails.Crypto.FeeAmount = 0
				resp.RequestDetails.Crypto.AddressTag = test
			}
			Event(resp)
		}(x)
	}

	wg.Wait()
	_, err := GetEventByUUID(withdraw.DryRunID.String())
	if err != nil {
		if !errors.Is(err, ErrNoResults) {
			t.Fatal(err)
		}
	}

	v, err := GetEventsByExchange("test-1", 10)
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetEventByExchangeID("test-1", "test-1")
	if err != nil {
		t.Fatal(err)
	}

	if len(v) > 0 {
		_, err = GetEventByUUID(v[0].ID.String())
		if err != nil {
			if !errors.Is(err, ErrNoResults) {
				t.Fatal(err)
			}
		}
	}

	_, err = GetEventsByDate("test-1", time.Now().UTC().Add(-time.Minute), time.Now().UTC(), 5)
	if err != nil {
		t.Fatal(err)
	}
}
