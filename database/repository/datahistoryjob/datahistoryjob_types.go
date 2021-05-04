package datahistoryjob

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
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
	Interval         int64
	RequestSizeLimit int64
	DataType         int64
	MaxRetryAttempts int64
	Status           int64
	CreatedDate      time.Time
}

type DBService struct {
	sql    database.ISQL
	driver string
}
