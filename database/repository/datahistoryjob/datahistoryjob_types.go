package datahistoryjob

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
)

type DataHistoryJob struct {
	ID               string
	Nickname         string
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
	BatchSize        int64
	Status           int64
	CreatedDate      time.Time
	Results          []datahistoryjobresult.DataHistoryJobResult
}

type DBService struct {
	sql    database.ISQL
	driver string
}
