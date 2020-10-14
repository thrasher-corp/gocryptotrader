package withdraw

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

var (
	verbose       = false
	testExchanges = []exchange.Details{
		{
			Name: "one",
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

			err = exchange.InsertMany(testExchanges)
			if err != nil {
				t.Fatal(err)
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

func seedWithdrawData() {
	for x := 0; x < 20; x++ {
		test := fmt.Sprintf("test-%v", x)
		resp := &withdraw.Response{
			Exchange: withdraw.ExchangeResponse{
				Name:   testExchanges[0].Name,
				ID:     test,
				Status: test,
			},
			RequestDetails: withdraw.Request{
				Exchange:    testExchanges[0].Name,
				Description: test,
				Amount:      1.0,
				Fiat: withdraw.FiatRequest{
					Bank: banking.Account{
						Enabled:             false,
						ID:                  fmt.Sprintf("test-%v", x),
						BankName:            fmt.Sprintf("test-%v-bank", x),
						AccountName:         "hello",
						AccountNumber:       fmt.Sprintf("test-%v", x),
						BSBNumber:           "123456",
						SupportedCurrencies: "BTC-AUD",
						SupportedExchanges:  testExchanges[0].Name,
					},
				},
			},
		}
		rnd := rand.Intn(2) // nolint:gosec // used for generating test data, no need to import crypo/rand
		if rnd == 0 {
			resp.RequestDetails.Currency = currency.AUD
			resp.RequestDetails.Type = 1
		} else {
			resp.RequestDetails.Currency = currency.BTC
			resp.RequestDetails.Type = 0
			resp.RequestDetails.Crypto.Address = test
			resp.RequestDetails.Crypto.FeeAmount = 0
			resp.RequestDetails.Crypto.AddressTag = test
		}
		exchange.ResetExchangeCache()
		Event(resp)
	}
}
func withdrawHelper(t *testing.T) {
	seedWithdrawData()

	_, err := GetEventByUUID(withdraw.DryRunID.String())
	if err != nil {
		if !errors.Is(err, ErrNoResults) {
			t.Fatal(err)
		}
	}

	v, err := GetEventsByExchange(testExchanges[0].Name, 10)
	if err != nil {
		t.Error(err)
	}

	if v[0].Exchange.Name != testExchanges[0].Name {
		t.Fatalf("expected name to be translated to valid string instead received: %v", v[0].Exchange.Name)
	}

	_, err = GetEventByExchangeID(testExchanges[0].Name, "test-1")
	if err != nil {
		t.Error(err)
	}

	if len(v) > 0 {
		_, err = GetEventByUUID(v[0].ID.String())
		if err != nil {
			if !errors.Is(err, ErrNoResults) {
				t.Error(err)
			}
		}
	}

	_, err = GetEventsByDate(testExchanges[0].Name, time.Now().UTC().Add(-time.Minute), time.Now().UTC(), 5)
	if err != nil {
		t.Error(err)
	}
}
