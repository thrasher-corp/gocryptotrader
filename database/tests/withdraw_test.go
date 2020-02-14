package tests

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	withdraw_store "github.com/thrasher-corp/gocryptotrader/database/repository/withdraw"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/goose"
)

func TestWithdraw(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		runner func()
		closer func(t *testing.T, dbConn *database.Db) error
		output interface{}
	}{
		{
			"SQLite-Write",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			writeWithdraw,
			closeDatabase,
			nil,
		},
		{
			"SQLite-Read",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},

			readWithdrawHelper,
			closeDatabase,
			nil,
		},
		{
			"Postgres-Write",
			postgresTestDatabase,
			writeWithdraw,
			nil,
			nil,
		},
		{
			"Postgres-Read",
			postgresTestDatabase,
			readWithdrawHelper,
			nil,
			nil,
		},
	}

	for _, tests := range testCases {
		test := tests

		t.Run(test.name, func(t *testing.T) {
			if !checkValidConfig(t, &test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := connectToDatabase(t, test.config)

			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join("..", "migrations")
			err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
			if err != nil {
				t.Fatalf("failed to run migrations %v", err)
			}

			if test.runner != nil {
				test.runner()
			}

			if test.closer != nil {
				err = test.closer(t, dbConn)
				if err != nil {
					t.Log(err)
				}
			}
		})
	}
}

func writeWithdraw() {
	var wg sync.WaitGroup

	for x := 0; x < 20; x++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			resp := &withdraw.Response{
				ID: withdraw.DryRunID,
				Exchange: &withdraw.ExchangeResponse{
					Name:   test,
					ID:     test,
					Status: test,
				},
				RequestDetails: &withdraw.Request{
					Exchange:    test,
					Currency:    currency.AUD,
					Description: test,
					Amount:      1.0,
					Type:        1,
					Fiat: &withdraw.FiatRequest{
						Bank: &banking.Account{
							BankName:       test,
							BankAddress:    test,
							BankPostalCode: test,
							BankPostalCity: test,
							BankCountry:    test,
							AccountName:    test,
							AccountNumber:  test,
							SWIFTCode:      test,
							IBAN:           test,
							BSBNumber:      test,
						},
					},
				},
			}

			withdraw_store.Event(resp)
		}(x)
	}

	wg.Wait()
}

func readWithdrawHelper() {
	// TODO: implement read to read first result and confirm data was written
}
