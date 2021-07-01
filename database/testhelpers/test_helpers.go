package testhelpers

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	psqlConn "github.com/thrasher-corp/gocryptotrader/database/drivers/postgres"
	sqliteConn "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	"github.com/thrasher-corp/gocryptotrader/log"
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
	_, exists := os.LookupEnv("TRAVIS")
	if exists {
		return &database.Config{
			Enabled: true,
			Driver:  "postgres",
			ConnectionDetails: drivers.ConnectionDetails{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "",
				Database: "gct_dev_ci",
				SSLMode:  "",
			},
		}
	}

	_, exists = os.LookupEnv("APPVEYOR")
	if exists {
		return &database.Config{
			Enabled: true,
			Driver:  "postgres",
			ConnectionDetails: drivers.ConnectionDetails{
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "Password12!",
				Database: "gct_dev_ci",
				SSLMode:  "",
			},
		}
	}

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
	err = database.DB.SetConfig(conn)
	if err != nil {
		return nil, err
	}
	if conn.Driver == database.DBPostgreSQL {
		dbConn, err = psqlConn.Connect(conn)
		if err != nil {
			return nil, err
		}
	} else if conn.Driver == database.DBSQLite3 || conn.Driver == database.DBSQLite {
		database.DB.DataPath = TempDir
		dbConn, err = sqliteConn.Connect(conn.Database)
		if err != nil {
			return nil, err
		}
	}

	err = migrateDB(database.DB.SQL)
	if err != nil {
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
func EnableVerboseTestOutput() {
	c := log.GenDefaultSettings()
	log.RWM.Lock()
	log.GlobalLogConfig = &c
	log.RWM.Unlock()
	log.SetupGlobalLogger()

	DBLogger := database.Logger{}
	boil.DebugMode = true
	boil.DebugWriter = DBLogger
}
