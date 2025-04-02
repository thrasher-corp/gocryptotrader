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
	"github.com/volatiletech/null"
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

// GetRelatedUpcomingJobs will return related jobs
func (db *DBService) GetRelatedUpcomingJobs(nickname string) ([]*DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getRelatedUpcomingJobsSQLite(nickname)
	case database.DBPostgreSQL:
		return db.getRelatedUpcomingJobsPostgres(nickname)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

// GetPrerequisiteJob will return the job that must complete before the
// referenced job
func (db *DBService) GetPrerequisiteJob(nickname string) (*DataHistoryJob, error) {
	switch db.driver {
	case database.DBSQLite3, database.DBSQLite:
		return db.getPrerequisiteJobSQLite(nickname)
	case database.DBPostgreSQL:
		return db.getPrerequisiteJobPostgres(nickname)
	default:
		return nil, database.ErrNoDatabaseProvided
	}
}

// SetRelationshipByID removes a relationship in the event of a changed
// relationship during upsertion
func (db *DBService) SetRelationshipByID(prerequisiteJobID, followingJobID string, status int64) error {
	ctx := context.TODO()
	if strings.EqualFold(prerequisiteJobID, followingJobID) {
		return errCannotSetSamePrerequisite
	}
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
		err = setRelationshipByIDSQLite(ctx, tx, prerequisiteJobID, followingJobID, status)
	case database.DBPostgreSQL:
		err = setRelationshipByIDPostgres(ctx, tx, prerequisiteJobID, followingJobID, status)
	default:
		return database.ErrNoDatabaseProvided
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SetRelationshipByNickname removes a relationship in the event of a changed
// relationship during upsertion
func (db *DBService) SetRelationshipByNickname(prerequisiteNickname, followingNickname string, status int64) error {
	ctx := context.TODO()
	if strings.EqualFold(prerequisiteNickname, followingNickname) {
		return errCannotSetSamePrerequisite
	}
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
		err = setRelationshipByNicknameSQLite(ctx, tx, prerequisiteNickname, followingNickname, status)
	case database.DBPostgreSQL:
		err = setRelationshipByNicknamePostgres(ctx, tx, prerequisiteNickname, followingNickname, status)
	default:
		return database.ErrNoDatabaseProvided
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func upsertSqlite(ctx context.Context, tx *sql.Tx, jobs ...*DataHistoryJob) error {
	for i := range jobs {
		exch, err := sqlite3.Exchanges(
			qm.Where("name = ?", strings.ToLower(jobs[i].ExchangeName))).One(ctx, tx)
		if err != nil {
			return fmt.Errorf("could not retrieve exchange '%v', %w", jobs[i].ExchangeName, err)
		}
		var secondaryExch *sqlite3.Exchange
		if jobs[i].SecondarySourceExchangeName != "" {
			secondaryExch, err = sqlite3.Exchanges(
				qm.Where("name = ?", strings.ToLower(jobs[i].SecondarySourceExchangeName))).One(ctx, tx)
			if err != nil {
				return fmt.Errorf("could not retrieve secondary exchange '%v', %w", jobs[i].SecondarySourceExchangeName, err)
			}
		}

		var overwrite, replaceOnIssue int64
		if jobs[i].OverwriteData {
			overwrite = 1
		}
		if jobs[i].ReplaceOnIssue {
			replaceOnIssue = 1
		}
		tempEvent := sqlite3.Datahistoryjob{
			ID:                       jobs[i].ID,
			ExchangeNameID:           exch.ID,
			Nickname:                 strings.ToLower(jobs[i].Nickname),
			Asset:                    strings.ToLower(jobs[i].Asset),
			Base:                     strings.ToUpper(jobs[i].Base),
			Quote:                    strings.ToUpper(jobs[i].Quote),
			StartTime:                jobs[i].StartDate.UTC().Format(time.RFC3339),
			EndTime:                  jobs[i].EndDate.UTC().Format(time.RFC3339),
			Interval:                 float64(jobs[i].Interval),
			DataType:                 float64(jobs[i].DataType),
			RequestSize:              float64(jobs[i].RequestSizeLimit),
			MaxRetries:               float64(jobs[i].MaxRetryAttempts),
			BatchCount:               float64(jobs[i].BatchSize),
			Status:                   float64(jobs[i].Status),
			Created:                  time.Now().UTC().Format(time.RFC3339),
			ConversionInterval:       null.Float64{Float64: float64(jobs[i].ConversionInterval), Valid: jobs[i].ConversionInterval > 0},
			OverwriteData:            null.Int64{Int64: overwrite, Valid: overwrite == 1},
			DecimalPlaceComparison:   null.Int64{Int64: int64(jobs[i].DecimalPlaceComparison), Valid: jobs[i].DecimalPlaceComparison > 0}, //nolint:gosec // TODO: Make uint64
			ReplaceOnIssue:           null.Int64{Int64: replaceOnIssue, Valid: replaceOnIssue == 1},
			IssueTolerancePercentage: null.Float64{Float64: jobs[i].IssueTolerancePercentage, Valid: jobs[i].IssueTolerancePercentage > 0},
		}
		if secondaryExch != nil {
			tempEvent.SecondaryExchangeID = null.String{String: secondaryExch.ID, Valid: true}
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
		exch, err := postgres.Exchanges(
			qm.Where("name = ?", strings.ToLower(jobs[i].ExchangeName))).One(ctx, tx)
		if err != nil {
			return fmt.Errorf("could not retrieve exchange '%v', %w", jobs[i].ExchangeName, err)
		}
		var secondaryExch *postgres.Exchange
		if jobs[i].SecondarySourceExchangeName != "" {
			secondaryExch, err = postgres.Exchanges(
				qm.Where("name = ?", strings.ToLower(jobs[i].SecondarySourceExchangeName))).One(ctx, tx)
			if err != nil {
				return fmt.Errorf("could not retrieve secondary exchange '%v', %w", jobs[i].SecondarySourceExchangeName, err)
			}
		}

		tempEvent := postgres.Datahistoryjob{
			ID:                       jobs[i].ID,
			Nickname:                 strings.ToLower(jobs[i].Nickname),
			ExchangeNameID:           exch.ID,
			Asset:                    strings.ToLower(jobs[i].Asset),
			Base:                     strings.ToUpper(jobs[i].Base),
			Quote:                    strings.ToUpper(jobs[i].Quote),
			StartTime:                jobs[i].StartDate.UTC(),
			EndTime:                  jobs[i].EndDate.UTC(),
			Interval:                 float64(jobs[i].Interval),
			DataType:                 float64(jobs[i].DataType),
			BatchCount:               float64(jobs[i].BatchSize),
			RequestSize:              float64(jobs[i].RequestSizeLimit),
			MaxRetries:               float64(jobs[i].MaxRetryAttempts),
			Status:                   float64(jobs[i].Status),
			Created:                  time.Now().UTC(),
			ConversionInterval:       null.Float64{Float64: float64(jobs[i].ConversionInterval), Valid: jobs[i].ConversionInterval > 0},
			OverwriteData:            null.Bool{Bool: jobs[i].OverwriteData, Valid: jobs[i].OverwriteData},
			DecimalPlaceComparison:   null.Int{Int: int(jobs[i].DecimalPlaceComparison), Valid: jobs[i].DecimalPlaceComparison > 0}, //nolint:gosec // TODO: Make uint64
			ReplaceOnIssue:           null.Bool{Bool: jobs[i].ReplaceOnIssue, Valid: jobs[i].ReplaceOnIssue},
			IssueTolerancePercentage: null.Float64{Float64: jobs[i].IssueTolerancePercentage, Valid: jobs[i].IssueTolerancePercentage > 0},
		}
		if secondaryExch != nil {
			tempEvent.SecondaryExchangeID = null.String{String: secondaryExch.ID, Valid: true}
		}
		err = tempEvent.Upsert(ctx, tx, true, []string{"nickname"}, boil.Infer(), boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DBService) getByNicknameSQLite(nickname string) (*DataHistoryJob, error) {
	result, err := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", strings.ToLower(nickname))).One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createSQLiteDataHistoryJobResponse(result)
}

func (db *DBService) getByNicknamePostgres(nickname string) (*DataHistoryJob, error) {
	query := postgres.Datahistoryjobs(qm.Where("nickname = ?", strings.ToLower(nickname)))
	result, err := query.One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	return db.createPostgresDataHistoryJobResponse(result)
}

func (db *DBService) getByIDSQLite(id string) (*DataHistoryJob, error) {
	result, err := sqlite3.Datahistoryjobs(qm.Where("id = ?", id)).One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createSQLiteDataHistoryJobResponse(result)
}

func (db *DBService) getByIDPostgres(id string) (*DataHistoryJob, error) {
	query := postgres.Datahistoryjobs(qm.Where("id = ?", id))
	result, err := query.One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createPostgresDataHistoryJobResponse(result)
}

func (db *DBService) getJobsBetweenSQLite(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	query := sqlite3.Datahistoryjobs(qm.Where("created BETWEEN ? AND ? ", startDate.UTC().Format(time.RFC3339), endDate.UTC().Format(time.RFC3339)))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	jobs := make([]DataHistoryJob, 0, len(results))
	for i := range results {
		job, err := db.createSQLiteDataHistoryJobResponse(results[i])
		if err != nil {
			return nil, fmt.Errorf("could not return job %v: %w", results[i].Nickname, err)
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (db *DBService) getJobsBetweenPostgres(startDate, endDate time.Time) ([]DataHistoryJob, error) {
	query := postgres.Datahistoryjobs(qm.Where("created BETWEEN ? AND  ? ", startDate, endDate))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	jobs := make([]DataHistoryJob, 0, len(results))
	for i := range results {
		job, err := db.createPostgresDataHistoryJobResponse(results[i])
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (db *DBService) getJobAndAllResultsSQLite(nickname string) (*DataHistoryJob, error) {
	query := sqlite3.Datahistoryjobs(
		qm.Load(sqlite3.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Load(sqlite3.DatahistoryjobRels.ExchangeName),
		qm.Where("nickname = ?", strings.ToLower(nickname)))
	result, err := query.One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createSQLiteDataHistoryJobResponse(result)
}

func (db *DBService) getJobAndAllResultsPostgres(nickname string) (*DataHistoryJob, error) {
	query := postgres.Datahistoryjobs(
		qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Where("nickname = ?", strings.ToLower(nickname)))
	result, err := query.One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createPostgresDataHistoryJobResponse(result)
}

func (db *DBService) getAllIncompleteJobsAndResultsSQLite() ([]DataHistoryJob, error) {
	query := sqlite3.Datahistoryjobs(
		qm.Load(sqlite3.DatahistoryjobRels.ExchangeName),
		qm.Load(sqlite3.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Where("status = ?", 0))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	jobs := make([]DataHistoryJob, 0, len(results))
	for i := range results {
		job, err := db.createSQLiteDataHistoryJobResponse(results[i])
		if err != nil {
			return nil, fmt.Errorf("could not return job %v: %w", results[i].Nickname, err)
		}

		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (db *DBService) getAllIncompleteJobsAndResultsPostgres() ([]DataHistoryJob, error) {
	query := postgres.Datahistoryjobs(
		qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobresults),
		qm.Where("status = ?", 0))
	results, err := query.All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	jobs := make([]DataHistoryJob, 0, len(results))
	for i := range results {
		job, err := db.createPostgresDataHistoryJobResponse(results[i])
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}

	return jobs, nil
}

func (db *DBService) getRelatedUpcomingJobsSQLite(nickname string) ([]*DataHistoryJob, error) {
	job, err := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", nickname)).One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	results, err := job.JobDatahistoryjobs().All(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	resp := make([]*DataHistoryJob, 0, len(results))
	for i := range results {
		job, err := db.createSQLiteDataHistoryJobResponse(results[i])
		if err != nil {
			return nil, fmt.Errorf("could not return job %v: %w", results[i].Nickname, err)
		}
		resp = append(resp, job)
	}
	return resp, nil
}

func (db *DBService) getRelatedUpcomingJobsPostgres(nickname string) ([]*DataHistoryJob, error) {
	q := postgres.Datahistoryjobs(qm.Load(postgres.DatahistoryjobRels.JobDatahistoryjobs), qm.Where("nickname = ?", nickname))
	jobWithRelations, err := q.One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	response := make([]*DataHistoryJob, 0, len(jobWithRelations.R.JobDatahistoryjobs))
	for i := range jobWithRelations.R.JobDatahistoryjobs {
		job, err := db.getByIDPostgres(jobWithRelations.R.JobDatahistoryjobs[i].ID)
		if err != nil {
			return nil, err
		}
		response = append(response, job)
	}
	return response, nil
}

func setRelationshipByIDSQLite(ctx context.Context, tx *sql.Tx, prerequisiteJobID, followingJobID string, status int64) error {
	job, err := sqlite3.Datahistoryjobs(qm.Where("id = ?", followingJobID)).One(ctx, tx)
	if err != nil {
		return err
	}
	job.Status = float64(status)
	_, err = job.Update(ctx, tx, boil.Infer())
	if err != nil {
		return err
	}

	if prerequisiteJobID == "" {
		return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, true)
	}
	result, err := sqlite3.Datahistoryjobs(qm.Where("id = ?", prerequisiteJobID)).One(ctx, tx)
	if err != nil {
		return err
	}

	return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, true, result)
}

func setRelationshipByIDPostgres(ctx context.Context, tx *sql.Tx, prerequisiteJobID, followingJobID string, status int64) error {
	job, err := postgres.Datahistoryjobs(qm.Where("id = ?", followingJobID)).One(ctx, tx)
	if err != nil {
		return err
	}
	job.Status = float64(status)
	_, err = job.Update(ctx, tx, boil.Infer())
	if err != nil {
		return err
	}

	if prerequisiteJobID == "" {
		return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, false)
	}
	result, err := postgres.Datahistoryjobs(qm.Where("id = ?", prerequisiteJobID)).One(ctx, tx)
	if err != nil {
		return err
	}

	return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, false, result)
}

func (db *DBService) getPrerequisiteJobSQLite(nickname string) (*DataHistoryJob, error) {
	result, err := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", nickname)).One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	job, err := result.PrerequisiteJobDatahistoryjobs().One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createSQLiteDataHistoryJobResponse(job)
}

func (db *DBService) getPrerequisiteJobPostgres(nickname string) (*DataHistoryJob, error) {
	job, err := postgres.Datahistoryjobs(qm.Where("nickname = ?", nickname)).One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}
	result, err := job.PrerequisiteJobDatahistoryjobs().One(context.TODO(), db.sql)
	if err != nil {
		return nil, err
	}

	return db.createPostgresDataHistoryJobResponse(result)
}

func setRelationshipByNicknameSQLite(ctx context.Context, tx *sql.Tx, prerequisiteJobNickname, followingJobNickname string, status int64) error {
	job, err := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", followingJobNickname)).One(ctx, tx)
	if err != nil {
		return err
	}
	job.Status = float64(status)
	_, err = job.Update(ctx, tx, boil.Infer())
	if err != nil {
		return err
	}

	if prerequisiteJobNickname == "" {
		return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, true)
	}
	result, err := sqlite3.Datahistoryjobs(qm.Where("nickname = ?", prerequisiteJobNickname)).One(ctx, tx)
	if err != nil {
		return err
	}
	return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, true, result)
}

func setRelationshipByNicknamePostgres(ctx context.Context, tx *sql.Tx, prerequisiteJobNickname, followingJobNickname string, status int64) error {
	job, err := postgres.Datahistoryjobs(qm.Where("nickname = ?", followingJobNickname)).One(ctx, tx)
	if err != nil {
		return err
	}
	job.Status = float64(status)
	_, err = job.Update(ctx, tx, boil.Infer())
	if err != nil {
		return err
	}

	if prerequisiteJobNickname == "" {
		return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, false)
	}
	result, err := postgres.Datahistoryjobs(qm.Where("nickname = ?", prerequisiteJobNickname)).One(ctx, tx)
	if err != nil {
		return err
	}
	return job.SetPrerequisiteJobDatahistoryjobs(ctx, tx, false, result)
}

// helpers

func (db *DBService) createSQLiteDataHistoryJobResponse(result *sqlite3.Datahistoryjob) (*DataHistoryJob, error) {
	var exchange *sqlite3.Exchange
	var err error
	if result.R != nil && result.R.ExchangeName != nil {
		exchange = result.R.ExchangeName
	} else {
		exchange, err = result.ExchangeName().One(context.TODO(), db.sql)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve exchange '%v' %w", result.ExchangeNameID, err)
		}
	}
	var secondaryExchangeName string
	if result.SecondaryExchangeID.String != "" {
		var secondaryExchangeResult *sqlite3.Exchange
		secondaryExchangeResult, err = result.SecondaryExchange().One(context.TODO(), db.sql)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve secondary exchange '%v' %w", result.SecondaryExchangeID, err)
		}
		if secondaryExchangeResult != nil {
			secondaryExchangeName = secondaryExchangeResult.Name
		}
	}

	ts, err := time.Parse(time.RFC3339, result.StartTime)
	if err != nil {
		return nil, err
	}
	tEnd, err := time.Parse(time.RFC3339, result.EndTime)
	if err != nil {
		return nil, err
	}
	c, err := time.Parse(time.RFC3339, result.Created)
	if err != nil {
		return nil, err
	}

	prereqJob, err := result.PrerequisiteJobDatahistoryjobs().One(context.TODO(), db.sql)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	var prereqNickname, prereqID string
	if prereqJob != nil {
		prereqID = prereqJob.ID
		prereqNickname = prereqJob.Nickname
	}

	var jobResults []*datahistoryjobresult.DataHistoryJobResult
	if result.R != nil {
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
	}

	return &DataHistoryJob{
		ID:                          result.ID,
		Nickname:                    result.Nickname,
		ExchangeID:                  exchange.ID,
		ExchangeName:                exchange.Name,
		Asset:                       result.Asset,
		Base:                        result.Base,
		Quote:                       result.Quote,
		StartDate:                   ts,
		EndDate:                     tEnd,
		Interval:                    int64(result.Interval),
		RequestSizeLimit:            uint64(result.RequestSize),
		DataType:                    int64(result.DataType),
		MaxRetryAttempts:            uint64(result.MaxRetries),
		BatchSize:                   uint64(result.BatchCount),
		Status:                      int64(result.Status),
		CreatedDate:                 c,
		PrerequisiteJobID:           prereqID,
		PrerequisiteJobNickname:     prereqNickname,
		ConversionInterval:          int64(result.ConversionInterval.Float64),
		OverwriteData:               result.OverwriteData.Int64 == 1,
		DecimalPlaceComparison:      uint64(result.DecimalPlaceComparison.Int64), //nolint:gosec // TODO: Make uint64
		SecondarySourceExchangeName: secondaryExchangeName,
		IssueTolerancePercentage:    result.IssueTolerancePercentage.Float64,
		ReplaceOnIssue:              result.ReplaceOnIssue.Int64 == 1,
		Results:                     jobResults,
	}, nil
}

func (db *DBService) createPostgresDataHistoryJobResponse(result *postgres.Datahistoryjob) (*DataHistoryJob, error) {
	var exchange *postgres.Exchange
	var err error
	if result.R != nil && result.R.ExchangeName != nil {
		exchange = result.R.ExchangeName
	} else {
		exchange, err = result.ExchangeName().One(context.TODO(), db.sql)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve exchange '%v' %w", result.ExchangeNameID, err)
		}
	}

	var secondaryExchangeName string
	if result.SecondaryExchangeID.String != "" {
		var secondaryExchangeResult *postgres.Exchange
		secondaryExchangeResult, err = result.SecondaryExchange().One(context.TODO(), db.sql)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve secondary exchange '%v' %w", result.SecondaryExchangeID, err)
		}
		if secondaryExchangeResult != nil {
			secondaryExchangeName = secondaryExchangeResult.Name
		}
	}

	prereqJob, err := result.PrerequisiteJobDatahistoryjobs().One(context.TODO(), db.sql)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	var prereqNickname, prereqID string
	if prereqJob != nil {
		prereqID = prereqJob.ID
		prereqNickname = prereqJob.Nickname
	}

	var jobResults []*datahistoryjobresult.DataHistoryJobResult
	if result.R != nil {
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
	}

	return &DataHistoryJob{
		ID:                          result.ID,
		Nickname:                    result.Nickname,
		ExchangeID:                  exchange.ID,
		ExchangeName:                exchange.Name,
		Asset:                       result.Asset,
		Base:                        result.Base,
		Quote:                       result.Quote,
		StartDate:                   result.StartTime,
		EndDate:                     result.EndTime,
		Interval:                    int64(result.Interval),
		RequestSizeLimit:            uint64(result.RequestSize),
		DataType:                    int64(result.DataType),
		MaxRetryAttempts:            uint64(result.MaxRetries),
		BatchSize:                   uint64(result.BatchCount),
		Status:                      int64(result.Status),
		CreatedDate:                 result.Created,
		Results:                     jobResults,
		PrerequisiteJobID:           prereqID,
		PrerequisiteJobNickname:     prereqNickname,
		ConversionInterval:          int64(result.ConversionInterval.Float64),
		OverwriteData:               result.OverwriteData.Bool,
		DecimalPlaceComparison:      uint64(result.DecimalPlaceComparison.Int), //nolint:gosec // TODO: Make uint64
		SecondarySourceExchangeName: secondaryExchangeName,
		IssueTolerancePercentage:    result.IssueTolerancePercentage.Float64,
		ReplaceOnIssue:              result.ReplaceOnIssue.Bool,
	}, nil
}
