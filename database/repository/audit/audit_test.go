package audit

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

func TestMain(m *testing.M) {
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

func TestAudit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config *database.Config
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
		output any
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}
			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)
			if tc.runner != nil {
				tc.runner(t)
			}
			if tc.closer != nil {
				assert.NoError(t, tc.closer(dbConn))
			}
		})
	}
}

func writeAudit(t *testing.T) {
	t.Helper()
	var wg sync.WaitGroup

	for x := range 20 {
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
