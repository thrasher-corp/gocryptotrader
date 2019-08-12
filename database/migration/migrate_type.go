package migrations

import (
	"github.com/thrasher-corp/gocryptotrader/database"
)

type Migration struct {
	Sequence int
	Name     string
	UpSQL    []byte
	DownSQL  []byte
}

type Migrator struct {
	Conn       *database.Database
	Migrations []Migration
}
