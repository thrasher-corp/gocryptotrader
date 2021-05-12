package datahistoryjob

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/models/postgres"
	"github.com/thrasher-corp/gocryptotrader/database/models/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	"github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/sqlboiler/boil"
	"github.com/thrasher-corp/sqlboiler/queries/qm"
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
func (db *DBService) GetJobAndAllResults(jobID string) (*DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getJobAndAllResultsSQLite(jobID)
	case database.DBPostgreSQL:
		return db.getJobAndAllResultsPostgres(jobID)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

func upsertSqlite(ctx context.Context, tx *sql.Tx, jobs ...*DataHistoryJob) error {
	for i := range jobs {
		if jobs[i].ID == "" {
			freshUUID, err := uuid.NewV4()
			if err != nil {
				return err
			}
			jobs[i].ID = freshUUID.String()
		}

		var tempEvent = sqlite3.Datahistoryjob{
			ID:             jobs[i].ID,
			Nickname:       jobs[i].Nickname,
			ExchangeNameID: jobs[i].ExchangeID,
			Asset:          strings.ToLower(jobs[i].Asset),
			Base:           strings.ToUpper(jobs[i].Base),
			Quote:          strings.ToUpper(jobs[i].Quote),
			StartTime:      jobs[i].StartDate.UTC().Format(time.RFC3339),
			EndTime:        jobs[i].EndDate.UTC().Format(time.RFC3339),
			Interval:       float64(jobs[i].Interval),
			DataType:       float64(jobs[i].DataType),
			RequestSize:    float64(jobs[i].RequestSizeLimit),
			MaxRetries:     float64(jobs[i].MaxRetryAttempts),
			Status:         float64(jobs[i].Status),
			Created:        time.Now().UTC().Format(time.RFC3339),
		}
		err := tempEvent.Insert(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func upsertPostgres(ctx context.Context, tx *sql.Tx, jobs ...*DataHistoryJob) error {
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
		var tempEvent = postgres.Datahistoryjob{
			ID:             jobs[i].ID,
			Nickname:       jobs[i].Nickname,
			ExchangeNameID: jobs[i].ExchangeID,
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

func (db *DBService) getByNicknameSQLite(nickname string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
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

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     exch.Name,
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

func (db *DBService) getByNicknamePostgres(nickname string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := postgres.Datahistoryjobs(qm.Where("nickname = ?", nickname))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exch, err := exchange.OneByUUIDString(result.ExchangeNameID)
	if err != nil {
		return nil, err
	}

	job = &DataHistoryJob{
		ID:               result.ID,
		Nickname:         result.Nickname,
		ExchangeID:       result.ExchangeNameID,
		ExchangeName:     exch.Name,
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

func (db *DBService) getJobsBetweenSQLite(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
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

		jobs = append(jobs, DataHistoryJob{
			ID:               results[i].ID,
			Nickname:         results[i].Nickname,
			ExchangeID:       results[i].ExchangeNameID,
			ExchangeName:     exch.Name,
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

func (db *DBService) getJobsBetweenPostgres(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
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

		jobs = append(jobs, DataHistoryJob{
			ID:               results[i].ID,
			Nickname:         results[i].Nickname,
			ExchangeID:       results[i].ExchangeNameID,
			ExchangeName:     exch.Name,
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

func (db *DBService) getJobAndAllResultsSQLite(jobID string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := sqlite3.Datahistoryjobs(qm.Load(sqlite3.DatahistoryjobRels.JobDatahistoryjobresults), qm.Where("job_id = ?", jobID))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return nil, err
	}

	exch, err := exchange.OneByUUIDString(result.ExchangeNameID)
	if err != nil {
		return nil, err
	}

	var jobResults []datahistoryjobresult.DataHistoryJobResult
	for i := range result.R.JobDatahistoryjobresults {

		start := convert.TimeFromUnixTimestampDecimal(result.R.JobDatahistoryjobresults[i].IntervalStartTime)
		end := convert.TimeFromUnixTimestampDecimal(result.R.JobDatahistoryjobresults[i].IntervalEndTime)
		ran := convert.TimeFromUnixTimestampDecimal(result.R.JobDatahistoryjobresults[i].RunTime)

		jobResults = append(jobResults, datahistoryjobresult.DataHistoryJobResult{
			ID:                result.R.JobDatahistoryjobresults[i].ID,
			JobID:             result.R.JobDatahistoryjobresults[i].JobID,
			IntervalStartDate: start,
			IntervalEndDate:   end,
			Status:            int64(result.R.JobDatahistoryjobresults[i].Status),
			Result:            result.R.JobDatahistoryjobresults[i].Result.String,
			Date:              ran,
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
		ExchangeName:     exch.Name,
		Asset:            result.Asset,
		Base:             result.Base,
		Quote:            result.Quote,
		StartDate:        start,
		EndDate:          end,
		Interval:         int64(result.Interval),
		RequestSizeLimit: int64(result.RequestSize),
		DataType:         int64(result.DataType),
		MaxRetryAttempts: int64(result.MaxRetries),
		Status:           int64(result.Status),
		CreatedDate:      created,
		Results:          jobResults,
	}

	return job, nil
}

func (db *DBService) getJobAndAllResultsPostgres(jobID string) (*DataHistoryJob, error) {
	var job *DataHistoryJob
	query := postgres.Datahistoryjobs(qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobresults), qm.Where("job_id = ?", jobID))
	result, err := query.One(context.Background(), db.sql)
	if err != nil {
		return job, err
	}

	exch, err := exchange.OneByUUIDString(result.ExchangeNameID)
	if err != nil {
		return nil, err
	}

	var jobResults []datahistoryjobresult.DataHistoryJobResult
	for i := range result.R.JobDatahistoryjobresults {
		jobResults = append(jobResults, datahistoryjobresult.DataHistoryJobResult{
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
		ExchangeName:     exch.Name,
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
		Results:          jobResults,
	}

	return job, nil
}

func (db *DBService) getAllIncompleteJobsAndResultsSQLite() ([]DataHistoryJob, error) {
	var jobs []DataHistoryJob
	query := sqlite3.Datahistoryjobs(qm.Load(sqlite3.DatahistoryjobRels.JobDatahistoryjobresults), qm.Where("status = ?", 0))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		exch, err := exchange.OneByUUIDString(results[i].ExchangeNameID)
		if err != nil {
			return nil, err
		}

		var jobResults []datahistoryjobresult.DataHistoryJobResult
		for j := range results[i].R.JobDatahistoryjobresults {
			start := convert.TimeFromUnixTimestampDecimal(results[i].R.JobDatahistoryjobresults[j].IntervalStartTime)
			end := convert.TimeFromUnixTimestampDecimal(results[i].R.JobDatahistoryjobresults[j].IntervalEndTime)
			ran := convert.TimeFromUnixTimestampDecimal(results[i].R.JobDatahistoryjobresults[j].RunTime)

			jobResults = append(jobResults, datahistoryjobresult.DataHistoryJobResult{
				ID:                results[i].R.JobDatahistoryjobresults[j].ID,
				JobID:             results[i].R.JobDatahistoryjobresults[j].JobID,
				IntervalStartDate: start,
				IntervalEndDate:   end,
				Status:            int64(results[i].R.JobDatahistoryjobresults[j].Status),
				Result:            results[i].R.JobDatahistoryjobresults[j].Result.String,
				Date:              ran,
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
			ExchangeName:     exch.Name,
			Asset:            results[i].Asset,
			Base:             results[i].Base,
			Quote:            results[i].Quote,
			StartDate:        start,
			EndDate:          end,
			Interval:         int64(results[i].Interval),
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
	query := postgres.Datahistoryjobs(qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobresults), qm.Where("status = ?", 0))
	results, err := query.All(context.Background(), db.sql)
	if err != nil {
		return jobs, err
	}

	for i := range results {
		exch, err := exchange.OneByUUIDString(results[i].ExchangeNameID)
		if err != nil {
			return nil, err
		}

		var jobResults []datahistoryjobresult.DataHistoryJobResult
		for j := range results[i].R.JobDatahistoryjobresults {
			jobResults = append(jobResults, datahistoryjobresult.DataHistoryJobResult{
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
			ExchangeName:     exch.Name,
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
			Results:          jobResults,
		})
	}

	return jobs, nil
}
