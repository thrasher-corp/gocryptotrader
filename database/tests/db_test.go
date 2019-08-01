package tests

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/thrasher-/gocryptotrader/database"
	"github.com/thrasher-/gocryptotrader/database/drivers"
	dbpsql "github.com/thrasher-/gocryptotrader/database/drivers/postgres"
	dbsqlite "github.com/thrasher-/gocryptotrader/database/drivers/sqlite"
)

var (
	tempDir string
	trueptr = func(b bool) *bool { return &b }(true)

	postgresTestDatabase = database.Config{
		Enabled:           trueptr,
		Driver:            "postgres",
		ConnectionDetails: drivers.ConnectionDetails{
			//Host:     "",
			//Port:     5432,
			//Username: "",
			//Password: "",
			//Database: "",
		},
	}
)

func TestMain(m *testing.M) {
	var err error
	tempDir, err = ioutil.TempDir("", "gct-temp")

	if err != nil {
		fmt.Printf("failed to create temp file: %v", err)
		os.Exit(1)
	}

	t := m.Run()

	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Printf("Failed to remove temp db file: %v", err)
	}

	os.Exit(t)
}

func TestDatabaseConnect(t *testing.T) {
	testCases := []struct {
		name   string
		config database.Config
		output interface{}
	}{
		{
			"SQLite",
			database.Config{
				Driver:            "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{Database: path.Join(tempDir, "./testdb.db")},
			},
			nil,
		},
		{
			"SQliteNoDatabase",
			database.Config{
				ConnectionDetails: drivers.ConnectionDetails{Host: "local"},
			},
			database.ErrNoDatabaseProvided,
		},
		{
			name:   "Postgres",
			config: postgresTestDatabase,
			output: nil,
		},
	}

	for _, tests := range testCases {
		test := tests
		t.Run(test.name, func(t *testing.T) {

			if !checkValidConfig(t, &test.config.ConnectionDetails) {
				t.Skip("database not configured skipping test")
			}

			dbConn, err := connectToDatabase(t, &test.config)

			switch v := test.output.(type) {

			case error:
				if v != test.output.(error) {
					t.Fatal(err)
				}
				return
			default:
				break
			}

			if dbConn != nil {
				err := dbConn.SQL.Close()
				if err != nil {
					t.Error("Failed to close database")
				}
			}
		})
	}
}

func connectToDatabase(t *testing.T, conn *database.Config) (dbConn *database.Database, err error) {
	t.Helper()
	database.Conn.Config = conn

	if conn.Driver == "postgres" {
		dbConn, err = dbpsql.Connect()
		if err != nil {
			return
		}
	} else if conn.Driver == "sqlite" {
		dbConn, err = dbsqlite.Connect()
		if err != nil {
			return
		}
	}
	return
}

func closeDatabase(t *testing.T, conn *database.Database) (err error) {
	t.Helper()

	if conn != nil {
		err = conn.SQL.Close()
	}
	return
}

func checkValidConfig(t *testing.T, config *drivers.ConnectionDetails) bool {
	t.Helper()

	return !reflect.DeepEqual(drivers.ConnectionDetails{}, *config)
}
