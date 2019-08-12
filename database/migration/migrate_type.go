package migrations

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

type Migration struct {
	Sequence int
	Name     string
	UpSQL    string
	DownSQL  string
}

type Migrator struct {
	Conn       *database.Database
	Migrations []Migration
	Log        Logger
}

type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Errorf(format string, v ...interface{})
}
