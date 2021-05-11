package engine

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupDataHistoryManager creates a data history manager subsystem
func SetupDataHistoryManager(em iExchangeManager, dcm iDatabaseConnectionManager, processInterval time.Duration) (*DataHistoryManager, error) {
	if em == nil {
		return nil, errNilExchangeManager
	}
	if dcm == nil {
		return nil, errNilDatabaseConnectionManager
	}
	if processInterval <= 0 {
		processInterval = defaultTicker
	}

	dhj, err := datahistoryjob.Setup(dcm)
	dhjr, err := datahistoryjobresult.Setup(dcm)
	if err != nil {
		return nil, err
	}
	return &DataHistoryManager{
		exchangeManager:           em,
		databaseConnectionManager: dcm,
		shutdown:                  make(chan struct{}),
		interval:                  time.NewTicker(processInterval),
		jobDB:                     dhj,
		jobResultDB:               dhjr,
	}, nil
}

// Start runs the subsystem
func (m *DataHistoryManager) Start() error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return ErrSubSystemAlreadyStarted
	}
	m.shutdown = make(chan struct{})
	validJobs, err := m.PrepareJobs()
	if err != nil {
		return err
	}
	m.m.Lock()
	m.jobs = validJobs
	m.m.Unlock()

	m.wg.Add(1)
	m.run()

	return nil
}

// IsRunning checks whether the subsystem is running
func (m *DataHistoryManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

// Stop stops the subsystem
func (m *DataHistoryManager) Stop() error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !atomic.CompareAndSwapInt32(&m.started, 1, 0) {
		return ErrSubSystemNotStarted
	}
	close(m.shutdown)
	m.wg.Wait()
	return nil
}

// retrieveJobs will connect to the database and look for existing jobs
func (m *DataHistoryManager) retrieveJobs() ([]*DataHistoryJob, error) {
	if !m.databaseConnectionManager.IsConnected() {
		return nil, errDatabaseConnectionRequired
	}
	dbJobs, err := m.jobDB.GetAllIncompleteJobsAndResults()
	if err != nil {
		return nil, err
	}

	return convertDBModelToJob(dbJobs...)
}

// GetByNickname searches for jobs by name and returns it if found
// returns nil if not
func (m *DataHistoryManager) GetByNickname(nickname string) *DataHistoryJob {
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, nickname) {
			cpy := m.jobs[i]
			return cpy
		}
	}
	return nil
}

// UpsertJob allows for GRPC interaction to upsert a jobs to be processed
func (m *DataHistoryManager) UpsertJob(job *DataHistoryJob) error {
	m.m.Lock()
	defer m.m.Unlock()
	updated := false
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, job.Nickname) {
			updated = true
			m.jobs[i] = job
			break
		}
	}
	if !updated {
		m.jobs = append(m.jobs, job)
	}

	return m.jobDB.Upsert(&datahistoryjob.DataHistoryJob{
		ID:               job.ID.String(),
		Nickname:         job.Nickname,
		Exchange:         job.Exchange,
		Asset:            job.Asset.String(),
		Base:             job.Pair.Base.String(),
		Quote:            job.Pair.Quote.String(),
		StartDate:        job.StartDate,
		EndDate:          job.EndDate,
		Interval:         int64(job.Interval.Duration()),
		RequestSizeLimit: job.RequestSizeLimit,
		DataType:         job.DataType,
		MaxRetryAttempts: job.MaxRetryAttempts,
		BatchSize:        job.BatchSize,
		Status:           job.Status,
		CreatedDate:      job.CreatedDate,
	})
}

// RemoveJob allows for GRPC interaction to remove a job to be processed
// requires that the nickname field be set
func (m *DataHistoryManager) RemoveJob(nickname string) error {
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, nickname) {
			m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%v %w", nickname, errJobNotFound)
}

// PrepareJobs will validate the config jobs, verify their status with the database
// and return all valid jobs to be processed
// m.jobs will be overridden by this function
func (m *DataHistoryManager) PrepareJobs() ([]*DataHistoryJob, error) {
	m.m.Lock()
	defer m.m.Unlock()
	jobs, err := m.retrieveJobs()
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].DataType == TradeDataType &&
			jobs[i].Interval <= 0 {
			jobs[i].Interval = defaultTradeInterval
		}
		exch := m.exchangeManager.GetExchangeByName(jobs[i].Exchange)
		if exch == nil {
			log.Errorf(log.DataHistory, "exchange not loaded, cannot process jobs")
			continue
		}
	}
	err = m.compareJobsToData(jobs...)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (m *DataHistoryManager) compareJobsToData(jobs ...*DataHistoryJob) error {
	for i := range jobs {
		jobs[i].rangeHolder = kline.CalculateCandleDateRanges(m.jobs[i].StartDate, m.jobs[i].EndDate, m.jobs[i].Interval, uint32(m.jobs[i].RequestSizeLimit))

		switch jobs[i].DataType {
		case CandleDataType:
			candles, err := kline.LoadFromDatabase(jobs[i].Exchange, jobs[i].Pair, jobs[i].Asset, jobs[i].Interval, jobs[i].StartDate, jobs[i].EndDate)
			if err != nil {
				return err
			}
			err = jobs[i].rangeHolder.VerifyResultsHaveData(candles.Candles)
			if err != nil {
				return err
			}
		case TradeDataType:
			trades, err := trade.GetTradesInRange(jobs[i].Exchange, jobs[i].Asset.String(), jobs[i].Pair.Base.String(), jobs[i].Pair.Quote.String(), jobs[i].StartDate, jobs[i].EndDate)
			if err != nil {
				return err
			}
			candles, err := trade.ConvertTradesToCandles(jobs[i].Interval, trades...)
			if err != nil {
				return err
			}
			err = jobs[i].rangeHolder.VerifyResultsHaveData(candles.Candles)
			if err != nil {
				return err
			}
		default:
			return errUnknownDataType
		}

	}
	return nil
}

func (m *DataHistoryManager) run() {
	go func() {
		for {
			select {
			case <-m.shutdown:
				m.wg.Done()
				return
			case <-m.interval.C:
				if m.databaseConnectionManager.IsConnected() {
					go func() {
						if err := m.processJobs(); err != nil {
							log.Error(log.DataHistory, err)
						}
						validJobs, err := m.PrepareJobs()
						if err != nil {
							log.Error(log.DataHistory, err)
							return
						}
						m.m.Lock()
						m.jobs = validJobs
						m.m.Unlock()
					}()
				}
			}
		}
	}()
}

func (m *DataHistoryManager) processJobs() error {
	m.m.Lock()
	defer m.m.Unlock()
	var results []DataHistoryJobResult
	for i := range m.jobs {
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			id, err := uuid.NewV4()
			if err != nil {
				return err
			}
			fail := DataHistoryJobResult{
				ID:     id,
				JobID:  m.jobs[i].ID,
				Status: StatusFailed,
				Result: "exchange not loaded, cannot process job",
				Date:   time.Now(),
			}
			results = append(results, fail)
			log.Errorf(log.DataHistory, fail.Result)
			continue
		}
		result, err := m.runJob(m.jobs[i], exch)
		if err != nil {
			log.Error(log.DataHistory, err)
		}
		results = append(results, result...)
	}

	dbResults := convertJobResultToDBResult(results...)
	return m.jobResultDB.Upsert(dbResults...)
}

// runJob will process an individual job. It is either run as on a schedule
// or specifically via RPC command on demand
func (m *DataHistoryManager) runJob(job *DataHistoryJob, exch exchange.IBotExchange) ([]DataHistoryJobResult, error) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return nil, nil
	}
	if job.Status == StatusComplete ||
		job.Status == StatusFailed ||
		job.Status == StatusRemoved {
		// job doesn't need to be run. Log it?
		return nil, nil
	}
	var jobResults []DataHistoryJobResult
	var intervalsProcessed int64
processing:
	for i := range job.rangeHolder.Ranges {
		for j := range job.rangeHolder.Ranges[i].Intervals {
			if job.rangeHolder.Ranges[i].Intervals[j].HasData {
				continue
			}
			if intervalsProcessed >= job.BatchSize {
				job.Status = StatusFailed
				break processing
			}
			intervalsProcessed++
			status := StatusComplete
			id, err := uuid.NewV4()
			if err != nil {
				return nil, err
			}
			result := DataHistoryJobResult{
				ID:                id,
				JobID:             job.ID,
				IntervalStartDate: job.rangeHolder.Ranges[i].Intervals[j].Start.Time,
				IntervalEndDate:   job.rangeHolder.Ranges[i].Intervals[j].End.Time,
				Status:            int64(status),
			}
			// processing the job
			switch job.DataType {
			case CandleDataType:
				candles, err := exch.GetHistoricCandlesExtended(job.Pair, job.Asset, job.rangeHolder.Ranges[i].Intervals[j].Start.Time, job.rangeHolder.Ranges[i].Intervals[j].End.Time, job.Interval)
				if err != nil {
					result.Result = "could not get candles: " + err.Error()
					result.Status = StatusFailed
					break
				}
				_ = job.rangeHolder.VerifyResultsHaveData(candles.Candles)
				_, err = kline.StoreInDatabase(&candles, true)
				if err != nil {
					result.Result = "could not save results: " + err.Error()
					result.Status = StatusFailed
				}
			case TradeDataType:
				trades, err := exch.GetHistoricTrades(job.Pair, job.Asset, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time)
				if err != nil {
					result.Result = "could not get trades: " + err.Error()
					result.Status = StatusFailed
					break
				}
				candles, err := trade.ConvertTradesToCandles(job.Interval, trades...)
				if err != nil {
					result.Result = "could not convert candles to trades: " + err.Error()
					result.Status = StatusFailed
					break
				}
				_ = job.rangeHolder.VerifyResultsHaveData(candles.Candles)
				err = trade.SaveTradesToDatabase(trades...)
				if err != nil {
					result.Result = "could not save results: " + err.Error()
					result.Status = StatusFailed
				}
			default:
				return nil, errUnknownDataType
			}

			job.Results = append(job.Results, result)
			jobResults = append(jobResults, result)
		}
	}

	dbJob, err := convertJobToDBModel(job)
	if err != nil {
		return nil, err
	}

	err = m.jobDB.Upsert(&dbJob[0])
	if err != nil {
		return nil, err
	}

	// we return the jobs for when we process multiple jobs in sequence,
	// so that we only write to the database once for many job results
	return jobResults, nil
}

//-----------------------------------------------------------------------

func convertDBModelToJob(dbModels ...datahistoryjob.DataHistoryJob) ([]*DataHistoryJob, error) {
	var resp []*DataHistoryJob
	for i := range dbModels {
		id, err := uuid.FromString(dbModels[i].ID)
		if err != nil {
			return nil, err
		}
		cp, err := currency.NewPairFromString(fmt.Sprintf("%s-%s", dbModels[i].Base, dbModels[i].Quote))
		if err != nil {
			return nil, err
		}

		jobResults, err := convertDBResultToJobResult(dbModels[i].Results)
		if err != nil {
			return nil, err
		}

		resp = append(resp, &DataHistoryJob{
			ID:               id,
			Nickname:         dbModels[i].Nickname,
			Exchange:         dbModels[i].Exchange,
			Asset:            asset.Item(dbModels[i].Asset),
			Pair:             cp,
			StartDate:        dbModels[i].StartDate,
			EndDate:          dbModels[i].EndDate,
			Interval:         kline.Interval(dbModels[i].Interval),
			RequestSizeLimit: dbModels[i].RequestSizeLimit,
			DataType:         dbModels[i].DataType,
			MaxRetryAttempts: dbModels[i].MaxRetryAttempts,
			Status:           dbModels[i].Status,
			CreatedDate:      dbModels[i].CreatedDate,
			Results:          jobResults,
		})

	}
	return resp, nil
}

func convertDBResultToJobResult(dbModels []datahistoryjobresult.DataHistoryJobResult) ([]DataHistoryJobResult, error) {
	var result []DataHistoryJobResult
	for i := range dbModels {
		id, err := uuid.FromString(dbModels[i].ID)
		if err != nil {
			return nil, err
		}

		jobID, err := uuid.FromString(dbModels[i].JobID)
		if err != nil {
			return nil, err
		}
		result = append(result, DataHistoryJobResult{
			ID:                id,
			JobID:             jobID,
			IntervalStartDate: dbModels[i].IntervalStartDate,
			IntervalEndDate:   dbModels[i].IntervalEndDate,
			Status:            dbModels[i].Status,
			Result:            dbModels[i].Result,
			Date:              dbModels[i].Date,
		})
	}

	return result, nil
}

func convertJobResultToDBResult(results ...DataHistoryJobResult) []datahistoryjobresult.DataHistoryJobResult {
	var response []datahistoryjobresult.DataHistoryJobResult
	for i := range results {
		response = append(response, datahistoryjobresult.DataHistoryJobResult{
			ID:                results[i].ID.String(),
			JobID:             results[i].JobID.String(),
			IntervalStartDate: results[i].IntervalStartDate,
			IntervalEndDate:   results[i].IntervalEndDate,
			Status:            results[i].Status,
			Result:            results[i].Result,
			Date:              results[i].Date,
		})
	}
	return response
}

func convertJobToDBModel(models ...*DataHistoryJob) ([]datahistoryjob.DataHistoryJob, error) {
	var resp []datahistoryjob.DataHistoryJob
	for i := range models {
		resp = append(resp, datahistoryjob.DataHistoryJob{
			ID:               models[i].ID.String(),
			Nickname:         models[i].Nickname,
			Exchange:         models[i].Exchange,
			Asset:            models[i].Asset.String(),
			Base:             models[i].Pair.Base.String(),
			Quote:            models[i].Pair.Quote.String(),
			StartDate:        models[i].StartDate,
			EndDate:          models[i].EndDate,
			Interval:         int64(models[i].Interval.Duration()),
			RequestSizeLimit: models[i].RequestSizeLimit,
			DataType:         models[i].DataType,
			MaxRetryAttempts: models[i].MaxRetryAttempts,
			Status:           models[i].Status,
			CreatedDate:      models[i].CreatedDate,
			BatchSize:        models[i].BatchSize,
			Results:          convertJobResultToDBResult(models[i].Results...),
		})

	}
	return resp, nil
}
