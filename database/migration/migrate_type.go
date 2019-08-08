package migrations

import (
	"io"

	"github.com/jmoiron/sqlx"
)

type Migration struct {
	Sequence int64
	Name     string
	UpSQL    io.Reader
	DownSQL  io.Reader
}

type Migrator struct {
	conn       *sqlx.Conn
	Migrations []Migration
}
