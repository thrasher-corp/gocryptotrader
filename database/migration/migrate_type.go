package migrations

import (
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/database"
)

var (
	// MigrationDir Default folder to look for migrations to apply
	MigrationDir = filepath.Join("./database", "migration", "migrations")
)

// Migration holds all information passes from a migration file
// Includes: Sequence(version), SQL queries to run on up & down
type Migration struct {
	Sequence int
	Name     string
	UpSQL    string
	DownSQL  string
}

// Migrator holds pointer to database struct slice of Migrations and logger
type Migrator struct {
	Conn       *database.Database
	Migrations []Migration
	Log        Logger
}

// Logger interface implementation
// Allows you to BYO Logging/Printing

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Errorf(format string, v ...interface{})
}
