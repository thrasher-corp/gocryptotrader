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

// IDBService allows using data history job result database service
// without needing to care about implementation
type IDBService interface {
	Upsert(jobs ...*DataHistoryJobResult) error
	GetByJobID(jobID string) ([]DataHistoryJobResult, error)
	GetJobResultsBetween(jobID string, startDate, endDate time.Time) ([]DataHistoryJobResult, error)
}
