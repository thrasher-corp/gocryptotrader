package datahistoryjob

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
)

// DataHistoryJob is a DTO for database data
type DataHistoryJob struct {
	ID               string
	Nickname         string
	ExchangeID       string
	ExchangeName     string
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
	Results          []*datahistoryjobresult.DataHistoryJobResult
}

// DBService is a service which allows the interaction with
// the database without a direct reference to a global
type DBService struct {
	sql    database.ISQL
	driver string
}
