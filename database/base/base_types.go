package base

import (
	"time"
)

// RelativeDbPaths defines a relative path structure for the SQlBoiler TOML file
type RelativeDbPaths struct {
	Postgress DatabaseFields `toml:"psql"`
	Sqlite    DatabaseFields `toml:"sqlite3"`
}

// DatabaseFields defines the minimum of fields of a database for SQLBoiler
// functionality
type DatabaseFields struct {
	DBName    string        `toml:"dbname"`
	Host      string        `toml:"host"`
	Port      string        `toml:"port"`
	User      string        `toml:"user"`
	Pass      string        `toml:"pass"`
	SSLMode   string        `toml:"sslmode"`
	Whitelist []interface{} `toml:"whitelist,omitempty"`
	Blacklist []interface{} `toml:"blacklist,omitempty"`
}

// ConnDetails define the connection details for connecting to a database
type ConnDetails struct {
	Verbose bool

	// Absolute path to the database directory
	DirectoryPath string

	// Absolute path for a SQLite3 database
	SQLPath string

	// PosgreSQL/Mysql etc connection fields
	DBName  string
	Host    string
	User    string
	Pass    string
	Port    string
	SSLMode string

	// MemCacheSize denotes a size of the maximum size of a transaction before
	// being written to database
	MemCacheSize int64
}

// PlatformTrades defines a paramater type for insertion of bulk trades
type PlatformTrades struct {
	OrderID      string
	ExchangeName string
	Pair         string
	AssetType    string
	OrderType    string
	Amount       float64
	Rate         float64
	FullfilledOn time.Time
}

// User defines user info that is from the database
type User struct {
	ID                int
	UserName          string
	Password          string
	Email             string
	OneTimePassword   string
	PasswordCreatedAt time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	LastLoggedIn      time.Time
	Enabled           bool
}

// Audit defines an audit trail for a high security change/modification/addition
// or deletion of data
type Audit struct {
	UserID       int64
	Change       string
	TimeOfChange time.Time
}
