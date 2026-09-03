package datahistoryjobresult

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
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
		log.Fatal(err)
	}
	t := m.Run()
	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func seedDB() error {
	err := exchange.InsertMany(testExchanges)
	if err != nil {
		return err
	}

	for i := range testExchanges {
		lol, err := exchange.One(testExchanges[i].Name)
		if err != nil {
			return err
		}
		testExchanges[i].UUID = lol.UUID
	}

	return nil
}

func TestDataHistoryJob(t *testing.T) {
	t.Parallel()
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
				Driver:   database.DBSQLite3,
				Database: "./testdb",
			},
			seedDB: seedDB,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB())
			}

			db, err := Setup(dbConn)
			require.NoError(t, err)

			// postgres requires job for tests to function
			var id string
			if tc.name == "postgresql" {
				selectID, err := db.sql.Query("select id from datahistoryjob where nickname = 'testdatahistoryjob1'")
				require.NoError(t, err)
				defer func() {
					require.NoError(t, selectID.Close())
					require.NoError(t, selectID.Err())
				}()
				selectID.Next()
				err = selectID.Scan(&id)
				assert.NoError(t, err)
			}

			resulterinos := make([]*DataHistoryJobResult, 20)
			for i := range resulterinos {
				uu := uuid.NewV4()
				resulterinos[i] = &DataHistoryJobResult{
					ID:                uu.String(),
					JobID:             id,
					IntervalStartDate: time.Now(),
					IntervalEndDate:   time.Now().Add(time.Second),
					Status:            0,
					Result:            "Yay",
					Date:              time.Now(),
				}
			}
			err = db.Upsert(resulterinos...)
			require.NoError(t, err)
			// insert the same results to test conflict resolution

			resultaroos := make([]*DataHistoryJobResult, 20)
			for i := range resultaroos {
				uu := uuid.NewV4()
				j := &DataHistoryJobResult{
					ID:                uu.String(),
					JobID:             id,
					IntervalStartDate: time.Now(),
					IntervalEndDate:   time.Now().Add(time.Second),
					Status:            0,
					Result:            "Wow",
					Date:              time.Now(),
				}
				if i == 19 {
					j.Status = 1
					j.Date = time.Now().Add(time.Hour * 24)
				}
				resultaroos[i] = j
			}
			err = db.Upsert(resultaroos...)
			require.NoError(t, err)

			results, err := db.GetByJobID(id)
			require.NoError(t, err)
			assert.NotEmpty(t, results)

			results, err = db.GetJobResultsBetween(id, time.Now().Add(time.Hour*23), time.Now().Add(time.Hour*25))
			require.NoError(t, err)
			assert.NotEmpty(t, results)

			err = testhelpers.CloseDatabase(dbConn)
			assert.NoError(t, err)
		})
	}
}
