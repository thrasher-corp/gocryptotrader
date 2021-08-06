package audit

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
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

func TestAudit(t *testing.T) {
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

			writeAudit,
			testhelpers.CloseDatabase,
			nil,
		},
		{
			"SQLite-Read",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},

			readHelper,
			testhelpers.CloseDatabase,
			nil,
		},
		{
			"Postgres-Write",
			testhelpers.PostgresTestDatabase,
			writeAudit,
			nil,
			nil,
		},
		{
			"Postgres-Read",
			testhelpers.PostgresTestDatabase,
			readHelper,
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

func writeAudit(t *testing.T) {
	t.Helper()
	var wg sync.WaitGroup

	for x := 0; x < 20; x++ {
		wg.Add(1)

		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			Event(test, test, test)
		}(x)
	}

	wg.Wait()
}

func readHelper(t *testing.T) {
	t.Helper()

	_, err := GetEvent(time.Now().Add(-time.Hour*60), time.Now(), "asc", 1)
	if err != nil {
		t.Error(err)
	}
}
