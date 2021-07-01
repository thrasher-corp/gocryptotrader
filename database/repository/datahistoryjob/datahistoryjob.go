package datahistoryjob

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
)

// Setup returns a DBService
func Setup(db database.IDatabase) (*DBService, error) {
	if db == nil {
		return nil, database.ErrNilInstance
	}
	if !db.IsConnected() {
		return nil, database.ErrDatabaseNotConnected
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
func (db *DBService) Upsert(jobs ...*DataHistoryJob) error {
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

// GetByNickName returns a job by its nickname
func (db *DBService) GetByNickName(nickname string) (*DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getByNicknameSQLite(nickname)
	case database.DBPostgreSQL:
		return db.getByNicknamePostgres(nickname)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

// GetByID returns a job by its id
func (db *DBService) GetByID(id string) (*DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getByIDSQLite(id)
	case database.DBPostgreSQL:
		return db.getByIDPostgres(id)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

// GetJobsBetween will return all jobs between two dates
func (db *DBService) GetJobsBetween(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getJobsBetweenSQLite(startDate, endDate)
	case database.DBPostgreSQL:
		return db.getJobsBetweenPostgres(startDate, endDate)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

// GetAllIncompleteJobsAndResults returns all jobs that have the status "active"
func (db *DBService) GetAllIncompleteJobsAndResults() ([]DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getAllIncompleteJobsAndResultsSQLite()
	case database.DBPostgreSQL:
		return db.getAllIncompleteJobsAndResultsPostgres()
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

// GetJobAndAllResults returns a job and joins all job results
func (db *DBService) GetJobAndAllResults(nickname string) (*DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getJobAndAllResultsSQLite(nickname)
	case database.DBPostgreSQL:
		return db.getJobAndAllResultsPostgres(nickname)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

func upsertSqlite(ctx context.Context, tx *sql.Tx, jobs ...*DataHistoryJob) error {
	for i := range jobs {
		r, err := sqlite3.Exchanges(
			qm.Where("name = ?", strings.ToLower(jobs[i].ExchangeName))).One(ctx, tx)
		if err != nil {
			return err
		}
		var tempEvent = sqlite3.Datahistoryjob{
			ID:             jobs[i].ID,
			ExchangeNameID: r.ID,
			Nickname:       strings.ToLower(jobs[i].Nickname),
			Asset:          strings.ToLower(jobs[i].Asset),
			Base:           strings.ToUpper(jobs[i].Base),
			Quote:          strings.ToUpper(jobs[i].Quote),
			StartTime:      jobs[i].StartDate.UTC().Format(time.RFC3339),
			EndTime:        jobs[i].EndDate.UTC().Format(time.RFC3339),
			Interval:       float64(jobs[i].Interval),
			DataType:       float64(jobs[i].DataType),
			RequestSize:    float64(jobs[i].RequestSizeLimit),
			MaxRetries:     float64(jobs[i].MaxRetryAttempts),
			BatchCount:     float64(jobs[i].BatchSize),
			Status:         float64(jobs[i].Status),
			Created:        time.Now().UTC().Format(time.RFC3339),
		}

		err = tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func upsertPostgres(ctx context.Context, tx *sql.Tx, jobs ...*DataHistoryJob) error {
	for i := range jobs {
		r, err := postgres.Exchanges(
			qm.Where("name = ?", strings.ToLower(jobs[i].ExchangeName))).One(ctx, tx)
		if err != nil {
			return err
		}
		var tempEvent = postgres.Datahistoryjob{
			ID:             jobs[i].ID,
			Nickname:       strings.ToLower(jobs[i].Nickname),
			ExchangeNameID: r.ID,
			Asset:          strings.ToLower(jobs[i].Asset),
			Base:           strings.ToUpper(jobs[i].Base),
			Quote:          strings.ToUpper(jobs[i].Quote),
			StartTime:      jobs[i].StartDate.UTC(),
			EndTime:        jobs[i].EndDate.UTC(),
			Interval:       float64(jobs[i].Interval),
			DataType:       float64(jobs[i].DataType),
			BatchCount:     float64(jobs[i].BatchSize),
			RequestSize:    float64(jobs[i].RequestSizeLimit),
			MaxRetries:     float64(jobs[i].MaxRetryAttempts),
			Status:         float64(jobs[i].Status),
			Created:        time.Now().UTC(),
		}
		err = tempEvent.Upsert(ctx, tx, true, []string{"nickname"}, boil.Infer(), boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBService) getByNicknameSQLite(nickname string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	result, err := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", strings.ToLower(nickname))).One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exchangeResult, err := result.ExchangeName().One(context.Background(), db.sql)
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

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     exchangeResult.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        ts,
		EndDate:          te,
		Interval:         int64(result.Interval),
		BatchSize:        int64(result.BatchCount),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      c,
	}

	return job, nil
}

func (db *DBService) getByNicknamePostgres(nickname string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := postgres.Datahistoryjobs(qm.Where("nickname = ?", strings.ToLower(nickname)))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exchangeResult, err := result.ExchangeName().One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     exchangeResult.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        result.StartTime,
		EndDate:          result.EndTime,
		Interval:         int64(result.Interval),
		BatchSize:        int64(result.BatchCount),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      result.Created,
	}

	return job, nil
}

func (db *DBService) getByIDSQLite(id string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	result, err := sqlite3.Datahistoryjobs(qm.Where("id = ?", id)).One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exchangeResult, err := result.ExchangeName().One(context.Background(), db.sql)
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

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     exchangeResult.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        ts,
		EndDate:          te,
		Interval:         int64(result.Interval),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		BatchSize:        int64(result.BatchCount),
		Status:           int64(result.Status),
		CreatedDate:      c,
	}

	return job, nil
}

func (db *DBService) getByIDPostgres(id string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := postgres.Datahistoryjobs(qm.Where("id = ?", id))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exchangeResult, err := result.ExchangeName().One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     exchangeResult.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        result.StartTime,
		EndDate:          result.EndTime,
		Interval:         int64(result.Interval),
		BatchSize:        int64(result.BatchCount),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      result.Created,
	}

	return job, nil
}

func (db *DBService) getJobsBetweenSQLite(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
	query := sqlite3.Datahistoryjobs(qm.Where("created BETWEEN ? AND ? ", startDate.UTC().Format(time.RFC3339), endDate.UTC().Format(time.RFC3339)))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		exchangeResult, err := results[i].ExchangeName(qm.Where("id = ?", results[i].ExchangeNameID)).One(context.Background(), db.sql)
		if err != nil {
			return nil, err
		}
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

		jobs = append(jobs, DataHistoryJob{
			ID:               results[i].ID,
			Nickname:         results[i].Nickname,
			ExchangeID:       results[i].ExchangeNameID,
			ExchangeName:     exchangeResult.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        ts,
			EndDate:          te,
			Interval:         int64(results[i].Interval),
			RequestSizeLimit: int64(results[i].RequestSize),
			BatchSize:        int64(results[i].BatchCount),
			DataType:         int64(results[i].DataType),
			MaxRetryAttempts: int64(results[i].MaxRetries),
			Status:           int64(results[i].Status),
			CreatedDate:      c,
		})
	}

	return jobs, nil
}

func (db *DBService) getJobsBetweenPostgres(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
	query := postgres.Datahistoryjobs(qm.Where("created BETWEEN ? AND  ? ", startDate, endDate))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		exchangeResult, err := results[i].ExchangeName(qm.Where("id = ?", results[i].ExchangeNameID)).One(context.Background(), db.sql)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, DataHistoryJob{
			ID:               results[i].ID,
			Nickname:         results[i].Nickname,
			ExchangeID:       results[i].ExchangeNameID,
			ExchangeName:     exchangeResult.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        results[i].StartTime,
			EndDate:          results[i].EndTime,
			Interval:         int64(results[i].Interval),
			BatchSize:        int64(results[i].BatchCount),
			RequestSizeLimit: int64(results[i].RequestSize),
			DataType:         int64(results[i].DataType),
			MaxRetryAttempts: int64(results[i].MaxRetries),
			Status:           int64(results[i].Status),
			CreatedDate:      results[i].Created,
		})
	}

	return jobs, nil
}

func (db *DBService) getJobAndAllResultsSQLite(nickname string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := sqlite3.Datahistoryjobs(
		qm.Load(sqlite3.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Load(sqlite3.DatahistoryjobRels.ExchangeName),
		qm.Where("nickname = ?", strings.ToLower(nickname)))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return nil, err
	}

	var jobResults []*datahistoryjobresult.DataHistoryJobResult
	for i := range result.R.JobDatahistoryjobresults {
		var start, end, run time.Time
		start, err = time.Parse(time.RFC3339, result.R.JobDatahistoryjobresults[i].IntervalStartTime)
		if err != nil {
			return nil, err
		}
		end, err = time.Parse(time.RFC3339, result.R.JobDatahistoryjobresults[i].IntervalEndTime)
		if err != nil {
			return nil, err
		}
		run, err = time.Parse(time.RFC3339, result.R.JobDatahistoryjobresults[i].RunTime)
		if err != nil {
			return nil, err
		}

		jobResults = append(jobResults, &datahistoryjobresult.DataHistoryJobResult{
			ID:                result.R.JobDatahistoryjobresults[i].ID,
			JobID:             result.R.JobDatahistoryjobresults[i].JobID,
			IntervalStartDate: start,
			IntervalEndDate:   end,
			Status:            int64(result.R.JobDatahistoryjobresults[i].Status),
			Result:            result.R.JobDatahistoryjobresults[i].Result.String,
			Date:              run,
		})
	}

	start, err := time.Parse(time.RFC3339, result.StartTime)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, result.EndTime)
	if err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339, result.Created)
	if err != nil {
		return nil, err
	}

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     result.R.ExchangeName.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        start,
		EndDate:          end,
		Interval:         int64(result.Interval),
		BatchSize:        int64(result.BatchCount),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      created,
		Results:          jobResults,
	}

	return job, nil
}

func (db *DBService) getJobAndAllResultsPostgres(nickname string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := postgres.Datahistoryjobs(
		qm.Load(postgres.DatahistoryjobRels.ExchangeName),
		qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Where("nickname = ?", strings.ToLower(nickname)))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	var jobResults []*datahistoryjobresult.DataHistoryJobResult
	for i := range result.R.JobDatahistoryjobresults {
		jobResults = append(jobResults, &datahistoryjobresult.DataHistoryJobResult{
			ID:                result.R.JobDatahistoryjobresults[i].ID,
			JobID:             result.R.JobDatahistoryjobresults[i].JobID,
			IntervalStartDate: result.R.JobDatahistoryjobresults[i].IntervalStartTime,
			IntervalEndDate:   result.R.JobDatahistoryjobresults[i].IntervalEndTime,
			Status:            int64(result.R.JobDatahistoryjobresults[i].Status),
			Result:            result.R.JobDatahistoryjobresults[i].Result.String,
			Date:              result.R.JobDatahistoryjobresults[i].RunTime,
		})
	}

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     result.R.ExchangeName.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        result.StartTime,
		EndDate:          result.EndTime,
		Interval:         int64(result.Interval),
		BatchSize:        int64(result.BatchCount),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      result.Created,
		Results:          jobResults,
	}

	return job, nil
}

func (db *DBService) getAllIncompleteJobsAndResultsSQLite() ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
	query := sqlite3.Datahistoryjobs(
		qm.Load(sqlite3.DatahistoryjobRels.ExchangeName),
		qm.Load(sqlite3.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Where("status = ?", 0))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		var jobResults []*datahistoryjobresult.DataHistoryJobResult
		for j := range results[i].R.JobDatahistoryjobresults {
			var start, end, run time.Time
			start, err = time.Parse(time.RFC3339, results[i].R.JobDatahistoryjobresults[j].IntervalStartTime)
			if err != nil {
				return nil, err
			}
			end, err = time.Parse(time.RFC3339, results[i].R.JobDatahistoryjobresults[j].IntervalEndTime)
			if err != nil {
				return nil, err
			}
			run, err = time.Parse(time.RFC3339, results[i].R.JobDatahistoryjobresults[j].RunTime)
			if err != nil {
				return nil, err
			}

			jobResults = append(jobResults, &datahistoryjobresult.DataHistoryJobResult{
				ID:                results[i].R.JobDatahistoryjobresults[j].ID,
				JobID:             results[i].R.JobDatahistoryjobresults[j].JobID,
				IntervalStartDate: start,
				IntervalEndDate:   end,
				Status:            int64(results[i].R.JobDatahistoryjobresults[j].Status),
				Result:            results[i].R.JobDatahistoryjobresults[j].Result.String,
				Date:              run,
			})
		}

		start, err := time.Parse(time.RFC3339, results[i].StartTime)
		if err != nil {
			return nil, err
		}
		end, err := time.Parse(time.RFC3339, results[i].EndTime)
		if err != nil {
			return nil, err
		}
		created, err := time.Parse(time.RFC3339, results[i].Created)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, DataHistoryJob{
			ID:               results[i].ID,
			Nickname:         results[i].Nickname,
			ExchangeID:       results[i].ExchangeNameID,
			ExchangeName:     results[i].R.ExchangeName.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        start,
			EndDate:          end,
			Interval:         int64(results[i].Interval),
			BatchSize:        int64(results[i].BatchCount),
			RequestSizeLimit: int64(results[i].RequestSize),
			DataType:         int64(results[i].DataType),
			MaxRetryAttempts: int64(results[i].MaxRetries),
			Status:           int64(results[i].Status),
			CreatedDate:      created,
			Results:          jobResults,
		})
	}

	return jobs, nil
}

func (db *DBService) getAllIncompleteJobsAndResultsPostgres() ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
	query := postgres.Datahistoryjobs(
		qm.Load(postgres.DatahistoryjobRels.ExchangeName),
		qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Where("status = ?", 0))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		var jobResults []*datahistoryjobresult.DataHistoryJobResult
		for j := range results[i].R.JobDatahistoryjobresults {
			jobResults = append(jobResults, &datahistoryjobresult.DataHistoryJobResult{
				ID:                results[i].R.JobDatahistoryjobresults[j].ID,
				JobID:             results[i].R.JobDatahistoryjobresults[j].JobID,
				IntervalStartDate: results[i].R.JobDatahistoryjobresults[j].IntervalStartTime,
				IntervalEndDate:   results[i].R.JobDatahistoryjobresults[j].IntervalEndTime,
				Status:            int64(results[i].R.JobDatahistoryjobresults[j].Status),
				Result:            results[i].R.JobDatahistoryjobresults[j].Result.String,
				Date:              results[i].R.JobDatahistoryjobresults[j].RunTime,
			})
		}

		jobs = append(jobs, DataHistoryJob{
			ID:               results[i].ID,
			Nickname:         results[i].Nickname,
			ExchangeID:       results[i].ExchangeNameID,
			ExchangeName:     results[i].R.ExchangeName.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        results[i].StartTime,
			EndDate:          results[i].EndTime,
			Interval:         int64(results[i].Interval),
			BatchSize:        int64(results[i].BatchCount),
			RequestSizeLimit: int64(results[i].RequestSize),
			DataType:         int64(results[i].DataType),
			MaxRetryAttempts: int64(results[i].MaxRetries),
			Status:           int64(results[i].Status),
			CreatedDate:      results[i].Created,
			Results:          jobResults,
		})
	}

	return jobs, nil
}
