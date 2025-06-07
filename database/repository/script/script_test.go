package script

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
	"github.com/volatiletech/null"
)

var verbose = false

func TestMain(m *testing.M) {
	var err error
	testhelpers.PostgresTestDatabase = testhelpers.GetConnectionDetails()
	testhelpers.TempDir, err = os.MkdirTemp("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	if verbose {
		err = testhelpers.EnableVerboseTestOutput()
		if err != nil {
			fmt.Printf("failed to enable verbose test output: %v", err)
			os.Exit(1)
		}
	}

	t := m.Run()

	err = os.RemoveAll(testhelpers.TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestScript(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		config *database.Config
		runner func()
		closer func(dbConn *database.Instance) error
		output any
	}{
		{
			"SQLite-Write",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			writeScript,
			testhelpers.CloseDatabase,
			nil,
		},
		{
			"Postgres-Write",
			testhelpers.PostgresTestDatabase,
			writeScript,
			nil,
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.runner != nil {
				tc.runner()
			}

			if tc.closer != nil {
				assert.NoError(t, tc.closer(dbConn))
			}
		})
	}
}

func writeScript() {
	var wg sync.WaitGroup
	for x := range 20 {
		wg.Add(1)

		go func(x int) {
			defer wg.Done()
			test := fmt.Sprintf("test-%v", x)
			var data null.Bytes
			Event(test, test, test, data, test, test, time.Now())
		}(x)
	}
	wg.Wait()
}
