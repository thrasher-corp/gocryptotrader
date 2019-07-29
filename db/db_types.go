package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/thrasher-/gocryptotrader/db/drivers"
)

type Database struct {
	Config *DatabaseConfig
	SQL    *sqlx.DB
}

type DatabaseConfig struct {
	Enabled                   *bool  `json:"enabled"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

var Conn = &Database{}
