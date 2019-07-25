package db

import (
	"github.com/jmoiron/sqlx"
)

type DBStruct struct {
	SQL *sqlx.DB
}

var DBConn = &DBStruct{}
