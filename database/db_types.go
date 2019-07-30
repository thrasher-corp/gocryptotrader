package database

import (
	"github.com/jmoiron/sqlx"
	"github.com/thrasher-/gocryptotrader/database/drivers"
)

type Database struct {
	Config *Config
	SQL    *sqlx.DB
}

type Config struct {
	Enabled                   *bool  `json:"enabled"`
	Driver                    string `json:"driver"`
	drivers.ConnectionDetails `json:"connectionDetails"`
}

var Conn = &Database{}
