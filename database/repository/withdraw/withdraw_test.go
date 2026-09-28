package withdraw

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
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

//nolint:forbidigo // TestMain reports setup and teardown failures before or after a *testing.T exists
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

func TestWithdraw(t *testing.T) {
	testCases := []struct {
		name   string
		config *database.Config
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
		output any
	}{
		{
			"SQLite-Write",
			&database.Config{
				Driver:   database.DBSQLite3,
				Database: "./testdb",
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

	for _, test := range testCases {
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

// A missing table fails an insert while leaving the transaction live, the only combination that
// reproduced the swallowed error. Creating withdrawal_history alone pushes the failure onto the
// relationship inserts, so every formerly broken block is covered. Both helpers run on SQLite;
// only propagation is under test
func TestAddEventReturnsInsertFailure(t *testing.T) {
	t.Parallel()
	// addPSQLEvent leaves the ID to the database, so the column needs a default here just as
	// gen_random_uuid() provides on postgres
	const historyTable = `CREATE TABLE withdrawal_history (id text PRIMARY KEY NOT NULL DEFAULT (lower(hex(randomblob(16)))), exchange_name_id text NOT NULL,
		exchange_id text NOT NULL, status text NOT NULL, currency text NOT NULL, amount real NOT NULL,
		description text NULL, withdraw_type integer NOT NULL, created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP)`

	for name, addEvent := range map[string]func(context.Context, *sql.Tx, *withdraw.Response) error{
		"addSQLiteEvent": addSQLiteEvent,
		"addPSQLEvent":   addPSQLEvent,
	} {
		for _, tc := range []struct {
			failing string
			schema  string
			reqType withdraw.RequestType
		}{
			{failing: "history", reqType: withdraw.Crypto},
			{failing: "crypto", schema: historyTable, reqType: withdraw.Crypto},
			{failing: "fiat", schema: historyTable, reqType: withdraw.Fiat},
		} {
			t.Run(name+"/"+tc.failing, func(t *testing.T) {
				t.Parallel()
				db, err := sql.Open(database.DBSQLite3, ":memory:")
				require.NoError(t, err, "Open must not error")
				t.Cleanup(func() { assert.NoError(t, db.Close(), "Close should not error") })

				if tc.schema != "" {
					_, err = db.ExecContext(t.Context(), tc.schema)
					require.NoError(t, err, "creating withdrawal_history must not error")
				}

				tx, err := db.BeginTx(t.Context(), nil)
				require.NoError(t, err, "BeginTx must not error")

				res := &withdraw.Response{
					Exchange:       withdraw.ExchangeResponse{Name: "one", ID: "1", Status: "ok"},
					RequestDetails: withdraw.Request{Currency: currency.BTC, Amount: 1, Type: tc.reqType},
				}
				res.RequestDetails.Crypto.Address = "addr"

				assert.Error(t, addEvent(t.Context(), tx, res), "a failed insert should be returned, not swallowed")
				assert.NoError(t, tx.Rollback(), "the transaction should still be live for the caller to roll back")
			})
		}
	}
}

func seedWithdrawData() {
	for x := range 20 {
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
		rnd := rand.IntN(2) //nolint:gosec // used for generating test data, no need to import crypto/rand
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
	t.Helper()
	seedWithdrawData()

	_, err := GetEventByUUID(withdraw.DryRunID.String())
	require.ErrorIs(t, err, common.ErrNoResults)

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
			assert.ErrorIs(t, err, common.ErrNoResults)
		}
	}

	_, err = GetEventsByDate(testExchanges[0].Name, time.Now().UTC().Add(-time.Minute), time.Now().UTC(), 5)
	if err != nil {
		t.Error(err)
	}
}
