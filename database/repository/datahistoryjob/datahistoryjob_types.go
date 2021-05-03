package datahistoryjob

import (
	"context"
	"database/sql"
	"time"
)

type DataHistoryJob struct {
	ID               string
	NickName         string
	Exchange         string
	Asset            string
	Base             string
	Quote            string
	StartDate        time.Time
	EndDate          time.Time
	IsRolling        bool
	Interval         int64
	RequestSizeLimit int64
	DataType         int64
	MaxRetryAttempts int64
	Status           int64
}

type iDatabase interface {
	IsConnected() bool
	GetSQL() *sql.DB
}

type iSQL interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type DataHistoryDB struct {
	sql iSQL
}
