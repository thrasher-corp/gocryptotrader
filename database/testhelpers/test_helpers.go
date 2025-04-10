package testhelpers

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	psqlConn "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	sqliteConn "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/goose"
	"github.com/thrasher-corp/sqlboiler/boil"
)

var (
	// TempDir temp folder for sqlite database
	TempDir string
	// PostgresTestDatabase postgresql database config details
	PostgresTestDatabase *database.Config
	// MigrationDir default folder for migration's
	MigrationDir = filepath.Join("..", "..", "migrations")
)

// GetConnectionDetails returns connection details for CI or test db instances
func GetConnectionDetails() *database.Config {
	return &database.Config{
		Enabled:           true,
		Driver:            "postgres",
		ConnectionDetails: drivers.ConnectionDetails{
			// Host:     "",
			// Port:     5432,
			// Username: "",
			// Password: "",
			// Database: "",
			// SSLMode:  "",
		},
	}
}

// ConnectToDatabase opens connection to database and returns pointer to instance of database.DB
func ConnectToDatabase(conn *database.Config) (dbConn *database.Instance, err error) {
	if err := database.DB.SetConfig(conn); err != nil {
		return nil, err
	}

	switch conn.Driver {
	case database.DBPostgreSQL:
		dbConn, err = psqlConn.Connect(conn)
	case database.DBSQLite3, database.DBSQLite:
		database.DB.DataPath = TempDir
		dbConn, err = sqliteConn.Connect(conn.Database)
	default:
		return nil, fmt.Errorf("unsupported database driver: %q", conn.Driver)
	}

	if err != nil {
		return nil, err
	}

	if err := migrateDB(database.DB.SQL); err != nil {
		return nil, err
	}

	database.DB.SetConnected(true)
	return
}

// CloseDatabase closes database connection
func CloseDatabase(conn *database.Instance) (err error) {
	if conn != nil {
		return conn.SQL.Close()
	}
	return nil
}

// CheckValidConfig checks if database connection details are empty
func CheckValidConfig(config *drivers.ConnectionDetails) bool {
	return !reflect.DeepEqual(drivers.ConnectionDetails{}, *config)
}

func migrateDB(db *sql.DB) error {
	return goose.Run("up", db, repository.GetSQLDialect(), MigrationDir, "")
}

// EnableVerboseTestOutput enables debug output for SQL queries
func EnableVerboseTestOutput() error {
	boil.DebugMode = true
	boil.DebugWriter = database.Logger{}
	return nil
}
