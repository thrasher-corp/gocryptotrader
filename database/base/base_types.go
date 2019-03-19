package base

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
}
