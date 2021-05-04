package datahistoryjobresult

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
)

type DataHistoryJobResult struct {
	ID                string
	JobID             string
	IntervalStartDate time.Time
	IntervalEndDate   time.Time
	Status            int64
	Result            string
	Date              time.Time
}

type DBService struct {
	sql    database.ISQL
	driver string
}
