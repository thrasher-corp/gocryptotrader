package base

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/naoina/toml"
	"github.com/thrasher-/gocryptotrader/common"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// Exported strings for database packages
const (
	SQLBoilerToml  = "sqlboiler.toml"
	SQLite3Schema  = "sqlite3.schema"
	PostGresSchema = "postgres.schema"
	SQLite         = "sqlite3"
	Postgres       = "postgres"

	QueryExchangeName    = "exchange_name = ?"
	QueryCurrencyPair    = "currency_pair = ?"
	QueryAssetType       = "asset_type = ?"
	QueryUserName        = "user_name = ?"
	OrderByFulfilledDesc = "fulfilled_on DESC"

	WarnTablesExist   = "Tables already exist in database, skipping insertion of new tables."
	WarnWrongPassword = "Incorrect password, please try again, %d attempts left."

	InfoInsertClient  = "Inserting new client into database..."
	InfoNoClients     = "No clients found in database, inserting new client..."
	InfoSingleClient  = "Client found in database, checking password"
	InfoMultiClient   = "Mutiple clients found in database, checking username and password"
	InfoUserNameFound = "Username %s found in database"

	DebugSchemaFileCreated = "Created schema file for database update and SQLBoiler model deployment at %s"
	DebugSchemaFileFound   = "Schema file found at %s"
	DebugDBConnecting      = "Opening connection to %s database using PATH: %s"
	DebugCreatedLog        = "Created helper file for SQLBoiler model deployment %s"
	DebugFoundLog          = "SQLBoiler file found at %s, verifying contents.."

	UsernameNotFound    = "client username %s not found in database"
	LoginFailure        = "failed to log into database for username %s"
	UsernameAlreadyUsed = "client username %s already in use"
	DBPathNotSet        = "path to %s database not set"
)

var (
	// Ctx defines a base database context
	Ctx = context.Background()

	// ErrDatabaseConnection defines a database connection failure error
	ErrDatabaseConnection = errors.New("database connection not established")

	// ErrDirectoryNotSet defines a directory not set error
	ErrDirectoryNotSet = errors.New("directory path not set")
)

// RelationalMap defines a mapping of variables specific to an individual
// database
type RelationalMap struct {
	// Database connection
	C *sql.DB

	// Actual database name
	InstanceName string
	Enabled      bool
	Connected    bool
	Verbose      bool

	// Client reference interface{}
	Client interface{}

	// Exchange map reference interface{}
	Exchanges map[string]interface{}

	// Pathways to folders and instances
	PathToDB  string
	PathDBDir string

	// Connection fields
	DatabaseName string
	Host         string
	User         string
	Password     string
	Port         string
	SSLMode      string

	// Super duper locking mechanism
	sync.Mutex
}

// GetName returns name of database
func (r *RelationalMap) GetName() string {
	r.Lock()
	defer r.Unlock()
	return r.InstanceName
}

// IsEnabled returns if the database is enabled
func (r *RelationalMap) IsEnabled() bool {
	r.Lock()
	defer r.Unlock()
	return r.Enabled
}

// IsConnected returns if the database has established a connection
func (r *RelationalMap) IsConnected() bool {
	r.Lock()
	defer r.Unlock()
	return r.Connected
}

// SetupHelperFiles sets up helper files for SQLBoiler model generation
func (r *RelationalMap) SetupHelperFiles() error {
	// Checks to see if default directory is made
	err := common.CheckDir(r.PathDBDir, true)
	if err != nil {
		return err
	}

	var sqlBoilerFile RelativeDbPaths
	fullPathToTomlFile := r.PathDBDir + SQLBoilerToml

	// Creates a configuration file that points to a database for generating new
	// database models, located in the database folder
	file, err := common.ReadFile(fullPathToTomlFile)
	switch r.InstanceName {
	case SQLite:
		if err != nil {
			sqlBoilerFile.Sqlite.DBName = r.PathToDB

			e, err := toml.Marshal(sqlBoilerFile)
			if err != nil {
				return err
			}

			err = common.WriteFile(fullPathToTomlFile, e)
			if err != nil {
				return err
			}

			if r.Verbose {
				log.Debugf(DebugCreatedLog, fullPathToTomlFile)
			}
		} else {
			if r.Verbose {
				log.Debugf(DebugFoundLog, fullPathToTomlFile)
			}

			err = toml.Unmarshal(file, &sqlBoilerFile)
			if err != nil {
				return err
			}

			if sqlBoilerFile.Sqlite.DBName == "" {
				sqlBoilerFile.Sqlite.DBName = r.PathToDB

				e, err := toml.Marshal(sqlBoilerFile)
				if err != nil {
					return err
				}

				err = common.WriteFile(fullPathToTomlFile, e)
				if err != nil {
					return err
				}
			}
		}

	case Postgres:
		if err != nil {
			sqlBoilerFile.Postgress.DBName = r.DatabaseName
			sqlBoilerFile.Postgress.Host = r.Host
			sqlBoilerFile.Postgress.User = r.User
			sqlBoilerFile.Postgress.SSLMode = r.SSLMode
			sqlBoilerFile.Postgress.Port = r.Port

			e, err := toml.Marshal(sqlBoilerFile)
			if err != nil {
				return err
			}

			err = common.WriteFile(fullPathToTomlFile, e)
			if err != nil {
				return err
			}

			if r.Verbose {
				log.Debugf(DebugCreatedLog, fullPathToTomlFile)
			}
		} else {
			if r.Verbose {
				log.Debugf(DebugFoundLog, fullPathToTomlFile)
			}

			err = toml.Unmarshal(file, &sqlBoilerFile)
			if err != nil {
				return err
			}

			if sqlBoilerFile.Postgress.DBName == r.DatabaseName ||
				sqlBoilerFile.Postgress.Host == r.Host ||
				sqlBoilerFile.Postgress.User == r.User ||
				sqlBoilerFile.Postgress.SSLMode == r.SSLMode ||
				sqlBoilerFile.Postgress.Port == r.Port {
				return nil
			}

			sqlBoilerFile.Postgress.DBName = r.DatabaseName
			sqlBoilerFile.Postgress.Host = r.Host
			sqlBoilerFile.Postgress.User = r.User
			sqlBoilerFile.Postgress.SSLMode = r.SSLMode
			sqlBoilerFile.Postgress.Port = r.Port

			e, err := toml.Marshal(sqlBoilerFile)
			if err != nil {
				return err
			}

			err = common.WriteFile(fullPathToTomlFile, e)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Disconnect closes the database connection
func (r *RelationalMap) Disconnect() error {
	r.Lock()
	defer r.Unlock()
	r.Connected = false
	return r.C.Close()
}
