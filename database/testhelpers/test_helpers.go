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
	"github.com/thrasher-corp/goose"
)

var (
	// TempDir temp folder for sqlite database
	TempDir string
	// PostgresTestDatabase postgresql database config details
	PostgresTestDatabase *database.Config

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
		Enabled: true,
		Driver:  "postgres",
		ConnectionDetails: drivers.ConnectionDetails{
			Host:     "localhost",
			Port:     5432,
			Username: "",
			Password: "",
			Database: "gct_test",
			SSLMode:  "disable",
		},
	}
}

// ConnectToDatabase opens connection to database and returns pointer to instance of database.DB
func ConnectToDatabase(conn *database.Config, runMigration bool) (dbConn *database.Instance, err error) {
	database.DB.Config = conn
	if conn.Driver == database.DBPostgreSQL {
		dbConn, err = psqlConn.Connect()
		if err != nil {
			return nil, err
		}
	} else if conn.Driver == database.DBSQLite3 || conn.Driver == database.DBSQLite {
		database.DB.DataPath = TempDir
		dbConn, err = sqliteConn.Connect()
		if err != nil {
			return nil, err
		}
	}

	if runMigration {
		err = migrateDB(database.DB.SQL)
		if err != nil {
			return nil, err
		}
	}

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
	err := ResetDB(db)
	if err != nil {
		return err
	}

	return MigrateDB(db)
}

func ResetDB(db *sql.DB) error {
	return goose.Run("reset", db, repository.GetSQLDialect(), MigrationDir, "")
}

func MigrateDB(db *sql.DB) error {
	return goose.Run("up", db, repository.GetSQLDialect(), MigrationDir, "")
}