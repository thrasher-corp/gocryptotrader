package datahistoryjobresult

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
	"github.com/volatiletech/null"
)

// Setup returns a DBService
func Setup(db database.IDatabase) (*DBService, error) {
	if db == nil {
		return nil, nil
	}
	if !db.IsConnected() {
		return nil, nil
	}
	cfg := db.GetConfig()
	dbCon, err := db.GetSQL()
	if err != nil {
		return nil, err
	}
	return &DBService{
		sql:    dbCon,
		driver: cfg.Driver,
	}, nil
}

// Upsert inserts or updates jobs into the database
func (db *DBService) Upsert(jobs ...*DataHistoryJobResult) error {
	if len(jobs) == 0 {
		return nil
	}
	ctx := context.TODO()

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
func (db *DBService) GetByJobID(jobID string) ([]DataHistoryJobResult, error) {
	var err error
	var job []DataHistoryJobResult
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

// GetJobResultsBetween will return all jobs between two dates
func (db *DBService) GetJobResultsBetween(jobID string, startDate, endDate time.Time) ([]DataHistoryJobResult, error) {
	var err error
	var jobs []DataHistoryJobResult
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		jobs, err = db.getJobResultsBetweenSQLite(jobID, startDate, endDate)
	case database.DBPostgreSQL:
		jobs, err = db.getJobResultsBetweenPostgres(jobID, startDate, endDate)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

func upsertSqlite(ctx context.Context, tx *sql.Tx, results ...*DataHistoryJobResult) error {
	for i := range results {
		if results[i].ID == "" {
			freshUUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			results[i].ID = freshUUID.String()
		}

		tempEvent := sqlite3.Datahistoryjobresult{
			ID:                results[i].ID,
			JobID:             results[i].JobID,
			Result:            null.NewString(results[i].Result, results[i].Result != ""),
			Status:            float64(results[i].Status),
			IntervalStartTime: results[i].IntervalStartDate.UTC().Format(time.RFC3339),
			IntervalEndTime:   results[i].IntervalEndDate.UTC().Format(time.RFC3339),
			RunTime:           results[i].Date.UTC().Format(time.RFC3339),
		}
		err := tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func upsertPostgres(ctx context.Context, tx *sql.Tx, results ...*DataHistoryJobResult) error {
	var err error
	for i := range results {
		if results[i].ID == "" {
			var freshUUID uuid.UUID
			freshUUID, err = uuid.NewV4()
			if err != nil {
				return err
			}
			results[i].ID = freshUUID.String()
		}
		tempEvent := postgres.Datahistoryjobresult{
			ID:                results[i].ID,
			JobID:             results[i].JobID,
			Result:            null.NewString(results[i].Result, results[i].Result != ""),
			Status:            float64(results[i].Status),
			IntervalStartTime: results[i].IntervalStartDate.UTC(),
			IntervalEndTime:   results[i].IntervalEndDate.UTC(),
			RunTime:           results[i].Date.UTC(),
		}
		err = tempEvent.Upsert(ctx, tx, false, nil, boil.Infer(), boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBService) getByJobIDSQLite(jobID string) ([]DataHistoryJobResult, error) {
	query := sqlite3.Datahistoryjobresults(qm.Where("job_id = ?", jobID))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	resp := make([]DataHistoryJobResult, len(results))
	for i := range results {
		var start, end, run time.Time
		start, err = time.Parse(time.RFC3339, results[i].IntervalStartTime)
		if err != nil {
			return nil, err
		}
		end, err = time.Parse(time.RFC3339, results[i].IntervalEndTime)
		if err != nil {
			return nil, err
		}
		run, err = time.Parse(time.RFC3339, results[i].RunTime)
		if err != nil {
			return nil, err
		}
		resp[i] = DataHistoryJobResult{
			ID:                results[i].ID,
			JobID:             results[i].JobID,
			IntervalStartDate: start,
			IntervalEndDate:   end,
			Status:            int64(results[i].Status),
			Result:            results[i].Result.String,
			Date:              run,
		}
	}

	return resp, nil
}

func (db *DBService) getByJobIDPostgres(jobID string) ([]DataHistoryJobResult, error) {
	query := postgres.Datahistoryjobresults(qm.Where("job_id = ?", jobID))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	resp := make([]DataHistoryJobResult, len(results))
	for i := range results {
		resp[i] = DataHistoryJobResult{
			ID:                results[i].ID,
			JobID:             results[i].JobID,
			IntervalStartDate: results[i].IntervalStartTime,
			IntervalEndDate:   results[i].IntervalEndTime,
			Status:            int64(results[i].Status),
			Result:            results[i].Result.String,
			Date:              results[i].RunTime,
		}
	}

	return resp, nil
}

func (db *DBService) getJobResultsBetweenSQLite(jobID string, startDate, endDate time.Time) ([]DataHistoryJobResult, error) {
	query := sqlite3.Datahistoryjobresults(qm.Where("job_id = ? AND run_time BETWEEN ? AND ? ", jobID, startDate.UTC().Format(time.RFC3339), endDate.UTC().Format(time.RFC3339)))
	resp, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	results := make([]DataHistoryJobResult, len(resp))
	for i := range resp {
		var start, end, run time.Time
		start, err = time.Parse(time.RFC3339, resp[i].IntervalStartTime)
		if err != nil {
			return nil, err
		}
		end, err = time.Parse(time.RFC3339, resp[i].IntervalEndTime)
		if err != nil {
			return nil, err
		}
		run, err = time.Parse(time.RFC3339, resp[i].RunTime)
		if err != nil {
			return nil, err
		}
		results[i] = DataHistoryJobResult{
			ID:                resp[i].ID,
			JobID:             resp[i].JobID,
			IntervalStartDate: start,
			IntervalEndDate:   end,
			Status:            int64(resp[i].Status),
			Result:            resp[i].Result.String,
			Date:              run,
		}
	}

	return results, nil
}

func (db *DBService) getJobResultsBetweenPostgres(jobID string, startDate, endDate time.Time) ([]DataHistoryJobResult, error) {
	query := postgres.Datahistoryjobresults(qm.Where("job_id = ? AND run_time BETWEEN ? AND  ? ", jobID, startDate, endDate))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	jobs := make([]DataHistoryJobResult, len(results))
	for i := range results {
		jobs[i] = DataHistoryJobResult{
			ID:                results[i].ID,
			JobID:             results[i].JobID,
			IntervalStartDate: results[i].IntervalStartTime,
			IntervalEndDate:   results[i].IntervalEndTime,
			Status:            int64(results[i].Status),
			Result:            results[i].Result.String,
			Date:              results[i].RunTime,
		}
	}

	return jobs, nil
}
