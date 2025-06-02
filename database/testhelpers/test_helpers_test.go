package testhelpers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

func TestMain(m *testing.M) {
	var err error
	PostgresTestDatabase = GetConnectionDetails()
	TempDir, err = os.MkdirTemp("", "gct-temp")
	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	MigrationDir = filepath.Join("..", "migrations")
	t := m.Run()

	err = os.RemoveAll(TempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestDatabaseConnect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   *database.Config
		closer   func(dbConn *database.Instance) error
		expError error
	}{
		{
			"SQLite",
			&database.Config{
				Driver:            database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{Database: "./testdb.db"},
			},
			CloseDatabase,
			nil,
		},
		{
			"SQliteNoDatabase",
			&database.Config{
				Driver: database.DBSQLite3,
				ConnectionDetails: drivers.ConnectionDetails{
					Host: "localhost",
				},
			},
			nil,
			database.ErrNoDatabaseProvided,
		},
		{
			name:   "Postgres",
			config: PostgresTestDatabase,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !CheckValidConfig(&tc.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := ConnectToDatabase(tc.config)
			require.ErrorIs(t, err, tc.expError)

			if tc.closer != nil {
				assert.NoError(t, tc.closer(dbConn))
			}
		})
	}
}
