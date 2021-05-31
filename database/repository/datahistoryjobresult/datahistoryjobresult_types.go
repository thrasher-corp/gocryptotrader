package datahistoryjobresult

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
)

// DataHistoryJobResult is a DTO for database data
type DataHistoryJobResult struct {
	ID                string
	JobID             string
	IntervalStartDate time.Time
	IntervalEndDate   time.Time
	Status            int64
	Result            string
	Date              time.Time
}

// DBService is a service which allows the interaction with
// the database without a direct reference to a global
type DBService struct {
	sql    database.ISQL
	driver string
}
