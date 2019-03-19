package base

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"reflect"
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
	OrderByFullfilledAsc = "fulfilled_on ASC"

	WarnTablesExist   = "Tables already exist in database, skipping insertion of new tables."
	WarnWrongPassword = "Incorrect password, please try again, %d attempts left."

	InfoInsertUser    = "Inserting new user into database..."
	InfoNoUsers       = "No users found in database, inserting new user..."
	InfoSingleUser    = "User found in database, checking password"
	InfoMultiUser     = "Mutiple users found in database, checking username and password"
	InfoUserNameFound = "User name %s found in database"

	DebugSchemaFileCreated = "Created schema file for database update and SQLBoiler model deployment at %s"
	DebugSchemaFileFound   = "Schema file found at %s"
	DebugDBConnecting      = "Opening connection to %s database using PATH: %s"
	DebugCreatedLog        = "Created helper file for SQLBoiler model deployment %s"
	DebugFoundLog          = "SQLBoiler file found at %s, verifying contents.."

	UsernameNotFound    = "user name %s not found in database"
	LoginFailure        = "failed to log into database for username %s"
	UsernameAlreadyUsed = "user name %s already in use"
	DBPathNotSet        = "path to %s database not set"

	// DefaultMemCache defaults to a 1 megabyte memcache for writing to db
	DefaultMemCache int64 = 1000000
)

var (
	// Ctx defines a base database context
	Ctx = context.Background()

	// SizeOfStatment is max size of assumed sql statement
	SizeOfStatment = reflect.TypeOf(sql.Stmt{}).Size()

	// SizeOfPointer is size of the pointer to an sql statement
	SizeOfPointer = reflect.TypeOf(new(sql.Stmt)).Size()

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

	// Write buffer to database based of size of memcache
	MaxSizeOfCache int64 // Max size in Bytes
	TxCounter      int64 // Number of transactions to infer memory size

	// transactionQueue has all the statements before committing to db
	transactionQueue *sql.Tx
	txMtx            sync.Mutex // Kind of redundant TODO: rethink this
	// design later

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
	_, mainErr := os.Stat(r.PathDBDir)
	if mainErr != nil {
		return mainErr
	}

	var sqlBoilerFile RelativeDbPaths
	fullPathToTomlFile := r.PathDBDir + SQLBoilerToml

	// Creates a configuration file that points to a database for generating new
	// database models, located in the database folder
	file, mainErr := common.ReadFile(fullPathToTomlFile)
	switch r.InstanceName {
	case SQLite:
		if mainErr != nil {
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

			err := toml.Unmarshal(file, &sqlBoilerFile)
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
		if mainErr != nil {
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

			err := toml.Unmarshal(file, &sqlBoilerFile)
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
	r.txMtx.Lock()
	defer func() { r.txMtx.Unlock(); r.Unlock() }()
	r.Connected = false
	err := r.CommitToDB()
	if err != nil {
		return err
	}
	return r.C.Close()
}

// NewTx returns a new pointer to a transaction for buffering database purposes
func (r *RelationalMap) NewTx() (*sql.Tx, error) {
	r.txMtx.Lock()
	if r.transactionQueue != nil {
		return r.transactionQueue, nil
	}
	var err error
	r.transactionQueue, err = r.C.BeginTx(Ctx, nil)
	if err != nil {
		r.txMtx.Unlock()
		return nil, err
	}
	return r.transactionQueue, nil
}

// CommitTx finishes the transaction and ends the input
func (r *RelationalMap) CommitTx(txLen int) error {
	defer r.txMtx.Unlock()

	// Check inferred size of memory allocation by transaction amounts
	r.TxCounter += int64(txLen)
	if r.TxCounter*int64(SizeOfPointer+SizeOfStatment) >= r.MaxSizeOfCache {
		return r.CommitToDB()
	}
	// Continue batch process and unlock()
	return nil
}

// CommitToDB commits transactional buffer to database
func (r *RelationalMap) CommitToDB() error {
	if r.transactionQueue == nil {
		return nil
	}
	if r.Verbose {
		log.Debugf("Insert %d records into %s database from transaction buffer",
			r.TxCounter,
			r.InstanceName)
	}
	err := r.transactionQueue.Commit()
	if err != nil {
		return err
	}
	r.TxCounter = 0          // reset counter
	r.transactionQueue = nil // force garbage collection
	return nil
}

// NewQuery intiates a new query thus writing transactional buffer to db for
// new query
func (r *RelationalMap) NewQuery() error {
	r.txMtx.Lock()
	return r.CommitToDB()
}

// FinishQuery unlocks transactional buffer
func (r *RelationalMap) FinishQuery() {
	r.txMtx.Unlock()
}
