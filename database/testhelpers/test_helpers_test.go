package testhelpers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
)

func TestMain(m *testing.M) {
	var err error
	PostgresTestDatabase = GetConnectionDetails()
	TempDir, err = ioutil.TempDir("", "gct-temp")
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
	testCases := []struct {
		name   string
		config *database.Config
		closer func(dbConn *database.Instance) error
		output interface{}
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
			output: nil,
		},
	}

	for x := range testCases {
		test := testCases[x]
		t.Run(test.name, func(t *testing.T) {
			if !CheckValidConfig(&test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := ConnectToDatabase(test.config)
			if err != nil {
				switch v := test.output.(type) {
				case error:
					if v.Error() != err.Error() {
						t.Fatal(err)
					}
					return
				default:
					break
				}
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
