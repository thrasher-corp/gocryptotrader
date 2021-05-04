package datahistoryjobresult

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/boil/qm"
)

// Setup returns a usable DBService service
// so you don't need to interact with globals in any fashion
func Setup(db database.IDatabase) (*DBService, error) {
	if db == nil {
		return nil, nil
	}
	if !db.IsConnected() {
		return nil, nil
	}
	cfg := db.GetConfig()
	return &DBService{
		sql:    db.GetSQL(),
		driver: cfg.Driver,
	}, nil
}

// Upsert inserts or updates jobs into the database
func (db *DBService) Upsert(jobs ...DataHistoryJobResult) error {
	ctx := context.Background()

	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginTx %w", err)
	}
	defer func() {
		if err != nil {
			errRB := tx.Rollback()
			if errRB != nil {
				log.Errorf(log.DatabaseMgr, "Insert tx.Rollback %v", errRB)
			}
		}
	}()

	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		err = upsertSqlite(ctx, tx, jobs...)
	case database.DBPostgreSQL:
		err = upsertPostgres(ctx, tx, jobs...)
	default:
		return database.ErrNoDatabaseProvided
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetByJobID returns a job by its related JobID
func (db *DBService) GetByJobID(jobID string) (*DataHistoryJobResult, error) {
	var err error
	var job *DataHistoryJobResult
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		job, err = db.getByJobIDSQLite(jobID)
	case database.DBPostgreSQL:
		job, err = db.getByJobIDPostgres(jobID)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

// GetJobsBetween will return all jobs between two dates
func (db *DBService) GetJobsBetween(startDate, endDate time.Time) ([]DataHistoryJobResult, error) {
	var err error
	var jobs []DataHistoryJobResult
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		jobs, err = db.getJobsBetweenSQLite(startDate, endDate)
	case database.DBPostgreSQL:
		jobs, err = db.getJobsBetweenPostgres(startDate, endDate)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
	if err != nil {
		return nil, err
	}
	return jobs, nil
}
func upsertSqlite(ctx context.Context, tx *sql.Tx, jobs ...DataHistoryJobResult) error {
	for i := range jobs {
		if jobs[i].ID == "" {
			freshUUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			jobs[i].ID = freshUUID.String()
		}
		var tempEvent = sqlite3.Datahistoryjobresult{
			ID:                jobs[i].ID,
			JobID:             jobs[i].JobID,
			Result:            null.String{},
			Status:            jobs[i].Status,
			IntervalStartTime: jobs[i].IntervalStartDate,
			IntervalEndTime:   jobs[i].IntervalEndDate,
			RunTime:           jobs[i].Date,
		}
		err := tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func upsertPostgres(ctx context.Context, tx *sql.Tx, jobs ...DataHistoryJobResult) error {
	var err error
	for i := range jobs {
		if jobs[i].ID == "" {
			var freshUUID uuid.UUID
			freshUUID, err = uuid.NewV4()
			if err != nil {
				return err
			}
			jobs[i].ID = freshUUID.String()
		}
		exchangeUUID, err := exchange.UUIDByName(jobs[i].Exchange)
		if err != nil {
			return err
		}
		var tempEvent = postgres.Datahistoryjobresult{
			ID:             jobs[i].ID,
			Nickname:       jobs[i].NickName,
			ExchangeNameID: exchangeUUID.String(),
			Asset:          strings.ToLower(jobs[i].Asset),
			Base:           strings.ToUpper(jobs[i].Base),
			Quote:          strings.ToUpper(jobs[i].Quote),
			StartTime:      jobs[i].StartDate.UTC(),
			EndTime:        jobs[i].EndDate.UTC(),
			Interval:       float64(jobs[i].Interval),
			DataType:       float64(jobs[i].DataType),
			RequestSize:    float64(jobs[i].RequestSizeLimit),
			MaxRetries:     float64(jobs[i].MaxRetryAttempts),
			Status:         float64(jobs[i].Status),
			Created:        time.Now().UTC(),
		}
		err = tempEvent.Upsert(ctx, tx, true, nil, boil.Infer(), boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBService) getByJobIDSQLite(nickname string) (*DataHistoryJobResult, error) {
	var job *DataHistoryJobResult
	query := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", nickname))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}
	ts, err := time.Parse(time.RFC3339, result.StartTime)
	if err != nil {
		return nil, err
	}

	te, err := time.Parse(time.RFC3339, result.EndTime)
	if err != nil {
		return nil, err
	}

	c, err := time.Parse(time.RFC3339, result.Created)
	if err != nil {
		return nil, err
	}

	exch, err := exchange.OneByUUIDString(result.ExchangeNameID)
	if err != nil {
		return nil, err
	}

	job = &DataHistoryJobResult{
		ID:               result.ID,
		NickName:         result.Nickname,
		Exchange:         exch.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        ts,
		EndDate:          te,
		Interval:         int64(result.Interval),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      c,
	}

	return job, nil
}

func (db *DBService) getByJobIDPostgres(nickname string) (*DataHistoryJobResult, error) {
	var job *DataHistoryJobResult
	query := postgres.Datahistoryjobs(qm.Where("nickname = ?", nickname))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exch, err := exchange.OneByUUIDString(result.ExchangeNameID)
	if err != nil {
		return nil, err
	}

	job = &DataHistoryJobResult{
		ID:               result.ID,
		NickName:         result.Nickname,
		Exchange:         exch.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        result.StartTime,
		EndDate:          result.EndTime,
		Interval:         int64(result.Interval),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      result.Created,
	}

	return job, nil
}

func (db *DBService) getJobsBetweenSQLite(startDate, endDate time.Time) ([]DataHistoryJobResult, error) {
	var jobs []DataHistoryJobResult
	query := sqlite3.Datahistoryjobs(qm.Where("created BETWEEN ? AND  ? ", startDate, endDate))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		ts, err := time.Parse(time.RFC3339, results[i].StartTime)
		if err != nil {
			return nil, err
		}

		te, err := time.Parse(time.RFC3339, results[i].EndTime)
		if err != nil {
			return nil, err
		}

		c, err := time.Parse(time.RFC3339, results[i].Created)
		if err != nil {
			return nil, err
		}

		exch, err := exchange.OneByUUIDString(results[i].ExchangeNameID)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, DataHistoryJobResult{
			ID:               results[i].ID,
			NickName:         results[i].Nickname,
			Exchange:         exch.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        ts,
			EndDate:          te,
			Interval:         int64(results[i].Interval),
			RequestSizeLimit: int64(results[i].RequestSize),
			DataType:         int64(results[i].DataType),
			MaxRetryAttempts: int64(results[i].MaxRetries),
			Status:           int64(results[i].Status),
			CreatedDate:      c,
		})
	}

	return jobs, nil
}

func (db *DBService) getJobsBetweenPostgres(startDate, endDate time.Time) ([]DataHistoryJobResult, error) {
	var jobs []DataHistoryJobResult
	query := postgres.Datahistoryjobs(qm.Where("created BETWEEN ? AND  ? ", startDate, endDate))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		exch, err := exchange.OneByUUIDString(results[i].ExchangeNameID)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, DataHistoryJobResult{
			ID:               results[i].ID,
			NickName:         results[i].Nickname,
			Exchange:         exch.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        results[i].StartTime,
			EndDate:          results[i].EndTime,
			Interval:         int64(results[i].Interval),
			RequestSizeLimit: int64(results[i].RequestSize),
			DataType:         int64(results[i].DataType),
			MaxRetryAttempts: int64(results[i].MaxRetries),
			Status:           int64(results[i].Status),
			CreatedDate:      results[i].Created,
		})
	}

	return jobs, nil
}
