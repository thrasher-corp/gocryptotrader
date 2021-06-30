package datahistoryjob

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
)

// DataHistoryJob is a DTO for database data
type DataHistoryJob struct {
	ID                      string
	Nickname                string
	ExchangeID              string
	ExchangeName            string
	Asset                   string
	Base                    string
	Quote                   string
	StartDate               time.Time
	EndDate                 time.Time
	Interval                int64
	RequestSizeLimit        int64
	DataType                int64
	MaxRetryAttempts        int64
	BatchSize               int64
	Status                  int64
	CreatedDate             time.Time
	Results                 []*datahistoryjobresult.DataHistoryJobResult
	PrerequisiteJobID       string
	PrerequisiteJobNickname string
	ConversionInterval      int64
	OverwriteData           bool
}

// DBService is a service which allows the interaction with
// the database without a direct reference to a global
type DBService struct {
	sql    database.ISQL
	driver string
}

// IDBService allows using data history job database service
// without needing to care about implementation
type IDBService interface {
	Upsert(...*DataHistoryJob) error
	GetByNickName(string) (*DataHistoryJob, error)
	GetByID(string) (*DataHistoryJob, error)
	GetJobsBetween(time.Time, time.Time) ([]DataHistoryJob, error)
	GetAllIncompleteJobsAndResults() ([]DataHistoryJob, error)
	GetJobAndAllResults(string) (*DataHistoryJob, error)
	GetRelatedUpcomingJobs(string) ([]*DataHistoryJob, error)
	SetRelationship(string, string, int64) error
}
