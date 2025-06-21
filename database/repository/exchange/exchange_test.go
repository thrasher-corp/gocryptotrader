package exchange

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/testhelpers"
)

var (
	verbose = false

	testExchanges = []Details{
		{
			Name: "one",
		},
		{
			Name: "two",
		},
		{
			Name: "three",
		},
	}
)

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

func TestInsertMany(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		config *database.Config
		seedDB func() error
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
	}{
		{
			name:   "postgresql",
			config: testhelpers.PostgresTestDatabase,
			seedDB: seed,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			seedDB: seed,
		},
	} {
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

			require.NoError(t, InsertMany(testExchanges))

			err = testhelpers.CloseDatabase(dbConn)
			assert.NoError(t, err)
		})
	}
}

func TestOneAndOneByUUID(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config *database.Config
		seedDB func() error
		runner func(t *testing.T)
		closer func(dbConn *database.Instance) error
	}{
		{
			name:   "postgresql",
			config: testhelpers.PostgresTestDatabase,
			seedDB: seed,
		},
		{
			name: "SQLite",
			config: &database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb"},
			},
			seedDB: seed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !testhelpers.CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := testhelpers.ConnectToDatabase(tc.config)
			require.NoError(t, err)

			if tc.seedDB != nil {
				require.NoError(t, tc.seedDB())
			}

			ret, err := One("one")
			require.NoError(t, err)

			ret2, err := OneByUUID(ret.UUID)
			require.NoError(t, err)

			assert.Equal(t, ret.Name, ret2.Name)
			assert.NoError(t, testhelpers.CloseDatabase(dbConn))
		})
	}
}

func seed() error {
	return InsertMany(testExchanges)
}

func TestLoadCSV(t *testing.T) {
	testData := filepath.Join("..", "..", "..", "testdata", "exchangelist.csv")
	if _, err := LoadCSV(testData); err != nil {
		t.Fatal(err)
	}
}
