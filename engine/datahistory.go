package engine

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	exchangedb "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
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
	if err != nil {
		return nil, err
	}

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
		atomic.StoreInt32(&m.started, 0)
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
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, ErrSubSystemNotStarted
	}
	if !m.databaseConnectionManager.IsConnected() {
		return nil, errDatabaseConnectionRequired
	}
	dbJobs, err := m.jobDB.GetAllIncompleteJobsAndResults()
	if err != nil {
		return nil, err
	}

	var response []*DataHistoryJob
	for i := range dbJobs {
		dbJob, err := m.convertDBModelToJob(&dbJobs[i])
		if err != nil {
			return nil, err
		}
		response = append(response, dbJob)
	}

	return response, nil
}

// GetByID returns a job's details from its ID,
func (m *DataHistoryManager) GetByID(id string) (*DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, ErrSubSystemNotStarted
	}
	m.m.Lock()
	for i := range m.jobs {
		if m.jobs[i].ID.String() == id {
			cpy := m.jobs[i]
			m.m.Unlock()
			return cpy, nil
		}
	}
	m.m.Unlock()
	dbJ, err := m.jobDB.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("%w with id %s %s", errJobNotFound, id, err)
	}
	result, err := m.convertDBModelToJob(dbJ)
	if err != nil {
		return nil, fmt.Errorf("could not convert model with id %s %w", id, err)
	}
	return result, nil
}

// GetByNickname searches for jobs by name and returns it if found
// returns nil if not
// if fullDetails is enabled, it will retrieve all job history results from the database
func (m *DataHistoryManager) GetByNickname(nickname string, fullDetails bool) (*DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, ErrSubSystemNotStarted
	}
	if fullDetails {
		dbJ, err := m.jobDB.GetJobAndAllResults(nickname)
		if err != nil {
			return nil, err
		}
		result, err := m.convertDBModelToJob(dbJ)
		if err != nil {
			return nil, fmt.Errorf("could not convert model with nickname %s %w", nickname, err)
		}
		return result, nil
	}
	m.m.Lock()
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, nickname) {
			cpy := m.jobs[i]
			m.m.Unlock()
			return cpy, nil
		}
	}
	m.m.Unlock()
	// now try the database
	j, err := m.jobDB.GetByNickName(nickname)
	if err != nil {
		return nil, fmt.Errorf("%w, %s", errJobNotFound, err)
	}
	job, err := m.convertDBModelToJob(j)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// GetAllJobStatusBetween will return all jobs between two ferns
func (m *DataHistoryManager) GetAllJobStatusBetween(start, end time.Time) ([]*DataHistoryJob, error) {
	dbJobs, err := m.jobDB.GetJobsBetween(start, end)
	if err != nil {
		return nil, err
	}
	var results []*DataHistoryJob
	for i := range dbJobs {
		dbJob, err := m.convertDBModelToJob(&dbJobs[i])
		if err != nil {
			return nil, err
		}
		results = append(results, dbJob)
	}
	return results, nil
}

// DeleteJob helper function to assist in setting a job to deleted
func (m *DataHistoryManager) DeleteJob(nickname, id string) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}
	if nickname == "" && id == "" {
		return errNicknameIDUnset
	}
	if nickname != "" && id != "" {
		return errOnlyNicknameOrID
	}
	var dbJob *datahistoryjob.DataHistoryJob
	var err error
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, nickname) ||
			m.jobs[i].ID.String() == id {
			dbJob, err = m.convertJobToDBModel(m.jobs[i])
			if err != nil {
				return err
			}
			m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
		}
	}
	if dbJob == nil {
		if nickname != "" {
			dbJob, err = m.jobDB.GetByNickName(nickname)
			if err != nil {
				return err
			}
		} else {
			dbJob, err = m.jobDB.GetByID(id)
			if err != nil {
				return err
			}
		}
	}
	dbJob.Status = int64(dataHistoryStatusRemoved)
	err = m.jobDB.Upsert(dbJob)
	if err != nil {
		return err
	}

	return nil
}

func (m *DataHistoryManager) GetActiveJobs() ([]DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}

	m.m.Lock()
	defer m.m.Unlock()
	var results []DataHistoryJob
	for i := range m.jobs {
		if m.jobs[i].Status == dataHistoryStatusActive {
			results = append(results, *m.jobs[i])
		}
	}
	return results, nil
}

// UpsertJob allows for GRPC interaction to upsert a jobs to be processed
func (m *DataHistoryManager) UpsertJob(job *DataHistoryJob, insertOnly bool) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
	if job == nil {
		return errNilJob
	}
	if job.Nickname == "" {
		return errNicknameUnset
	}
	if insertOnly {
		j, err := m.GetByNickname(job.Nickname, false)
		if err != nil && !errors.Is(err, errJobNotFound) {
			return err
		}
		if j != nil {
			return fmt.Errorf("%s %w", job.Nickname, errNicknameInUse)
		}
	}

	m.m.Lock()
	defer m.m.Unlock()

	err := m.validateJob(job)
	if err != nil {
		return err
	}
	toUpdate := false

	for i := range m.jobs {
		if !strings.EqualFold(m.jobs[i].Nickname, job.Nickname) {
			continue
		}
		toUpdate = true
		if job.Exchange != "" && m.jobs[i].Exchange != job.Exchange {
			m.jobs[i].Exchange = job.Exchange
		}
		if job.Asset != "" && m.jobs[i].Asset != job.Asset {
			m.jobs[i].Asset = job.Asset
		}
		if !job.Pair.IsEmpty() && !m.jobs[i].Pair.Equal(job.Pair) {
			m.jobs[i].Pair = job.Pair
		}
		if !job.StartDate.IsZero() && !m.jobs[i].StartDate.Equal(job.StartDate) {
			m.jobs[i].StartDate = job.StartDate
		}
		if !job.EndDate.IsZero() && !m.jobs[i].EndDate.Equal(job.EndDate) {
			m.jobs[i].EndDate = job.EndDate
		}
		if job.Interval != 0 && m.jobs[i].Interval != job.Interval {
			m.jobs[i].Interval = job.Interval
		}
		if job.RunBatchLimit != 0 && m.jobs[i].RunBatchLimit != job.RunBatchLimit {
			m.jobs[i].RunBatchLimit = job.RunBatchLimit
		}
		if job.RequestSizeLimit != 0 && m.jobs[i].RequestSizeLimit != job.RequestSizeLimit {
			m.jobs[i].RequestSizeLimit = job.RequestSizeLimit
		}
		if job.MaxRetryAttempts != 0 && m.jobs[i].MaxRetryAttempts != job.MaxRetryAttempts {
			m.jobs[i].MaxRetryAttempts = job.MaxRetryAttempts
		}

		m.jobs[i].DataType = job.DataType
		m.jobs[i].Status = job.Status
		m.jobs[i].rangeHolder, err = kline.CalculateCandleDateRanges(m.jobs[i].StartDate, m.jobs[i].EndDate, m.jobs[i].Interval, uint32(m.jobs[i].RequestSizeLimit))
		if err != nil {
			return err
		}

		break
	}
	if !toUpdate {
		job.rangeHolder, err = kline.CalculateCandleDateRanges(job.StartDate, job.EndDate, job.Interval, uint32(job.RequestSizeLimit))
		if err != nil {
			return err
		}
		m.jobs = append(m.jobs, job)
	}
	if job.ID == uuid.Nil {
		job.ID, err = uuid.NewV4()
		if err != nil {
			return err
		}
	}

	dbJob, err := m.convertJobToDBModel(job)
	if err != nil {
		return err
	}
	return m.jobDB.Upsert(dbJob)
}

func (m *DataHistoryManager) validateJob(job *DataHistoryJob) error {
	if job == nil {
		return errNilJob
	}
	if !job.Asset.IsValid() {
		return asset.ErrNotSupported
	}
	if job.Pair.IsEmpty() {
		return errCurrencyPairUnset
	}
	exch := m.exchangeManager.GetExchangeByName(job.Exchange)
	if exch == nil {
		return errExchangeNotLoaded
	}
	pairs, err := exch.GetAvailablePairs(job.Asset)
	if err != nil {
		return err
	}
	if !pairs.Contains(job.Pair, false) {
		return errCurrencyPairInvalid
	}
	if job.StartDate.After(job.EndDate) ||
		job.StartDate.IsZero() ||
		job.EndDate.IsZero() ||
		job.StartDate.After(time.Now()) {
		return errInvalidTimes
	}
	if job.Results == nil {
		job.Results = make(map[time.Time][]DataHistoryJobResult)
	}

	if job.RunBatchLimit <= 0 {
		log.Warnf(log.DataHistory, "job %s has unset batch limit, defaulting to %v", job.Nickname, defaultBatchLimit)
		job.RunBatchLimit = defaultBatchLimit
	}
	if job.MaxRetryAttempts <= 0 {
		log.Warnf(log.DataHistory, "job %s has unset max retry limit, defaulting to %v", job.Nickname, defaultRetryAttempts)
		job.MaxRetryAttempts = defaultRetryAttempts
	}

	job.StartDate = job.StartDate.Round(job.Interval.Duration())
	job.EndDate = job.EndDate.Round(job.Interval.Duration())

	return nil
}

// PrepareJobs will validate the config jobs, verify their status with the database
// and return all valid jobs to be processed
// m.jobs will be overridden by this function
func (m *DataHistoryManager) PrepareJobs() ([]*DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, ErrSubSystemNotStarted
	}
	m.m.Lock()
	defer m.m.Unlock()
	jobs, err := m.retrieveJobs()
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].DataType == dataHistoryTradeDataType &&
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
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}
	var err error
	for i := range jobs {
		jobs[i].rangeHolder, err = kline.CalculateCandleDateRanges(jobs[i].StartDate, jobs[i].EndDate, jobs[i].Interval, uint32(jobs[i].RequestSizeLimit))
		if err != nil {
			return err
		}
		var candles kline.Item
		switch jobs[i].DataType {
		case dataHistoryCandleDataType:
			candles, err = kline.LoadFromDatabase(jobs[i].Exchange, jobs[i].Pair, jobs[i].Asset, jobs[i].Interval, jobs[i].StartDate, jobs[i].EndDate)
			if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
				return err
			}
			err = jobs[i].rangeHolder.VerifyResultsHaveData(candles.Candles)
			if err != nil && !errors.Is(err, kline.ErrMissingCandleData) {
				return err
			}
		case dataHistoryTradeDataType:
			trades, err := trade.GetTradesInRange(jobs[i].Exchange, jobs[i].Asset.String(), jobs[i].Pair.Base.String(), jobs[i].Pair.Quote.String(), jobs[i].StartDate, jobs[i].EndDate)
			if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
				return err
			}
			candles, err = trade.ConvertTradesToCandles(jobs[i].Interval, trades...)
			if err != nil && !errors.Is(err, trade.ErrNoTradesSupplied) {
				return err
			}
			err = jobs[i].rangeHolder.VerifyResultsHaveData(candles.Candles)
			if err != nil && !errors.Is(err, kline.ErrMissingCandleData) {
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
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}

	if !atomic.CompareAndSwapInt32(&m.processing, 0, 1) {
		return fmt.Errorf("processJobs %w", errAlreadyRunning)
	}
	m.m.Lock()
	defer func() {
		m.m.Unlock()
		atomic.StoreInt32(&m.processing, 0)
	}()
	for i := range m.jobs {
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			log.Errorf(log.DataHistory, "exchange %s not loaded, cannot process job %s for %s %s",
				m.jobs[i].Exchange,
				m.jobs[i].Nickname,
				m.jobs[i].Asset,
				m.jobs[i].Pair)
			continue
		}
		err := m.runJob(m.jobs[i], exch)
		if err != nil {
			log.Error(log.DataHistory, err)
		}
	}
	return nil
}

// runJob will iterate
func (m *DataHistoryManager) runJob(job *DataHistoryJob, exch exchange.IBotExchange) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}
	if job.Status == dataHistoryStatusComplete ||
		job.Status == dataHistoryStatusFailed ||
		job.Status == dataHistoryStatusRemoved {
		// job doesn't need to be run. Log it?
		return nil
	}
	var intervalsProcessed int64
	if job.rangeHolder == nil || len(job.rangeHolder.Ranges) == 0 {
		return errJobInvalid
	}
processing:
	for i := range job.rangeHolder.Ranges {
		for j := range job.rangeHolder.Ranges[i].Intervals {
			if job.rangeHolder.Ranges[i].Intervals[j].HasData {
				continue
			}
			if intervalsProcessed >= job.RunBatchLimit {
				break processing
			}
			var failures int64
			resultLookup := job.Results[job.rangeHolder.Ranges[i].Intervals[j].Start.Time]
			for x := range resultLookup {
				if resultLookup[x].Status == dataHistoryStatusFailed {
					failures++
				}
			}
			if failures >= job.MaxRetryAttempts {
				job.Status = dataHistoryStatusFailed
				break processing
			}
			intervalsProcessed++
			status := dataHistoryStatusComplete
			id, err := uuid.NewV4()
			if err != nil {
				return err
			}
			result := DataHistoryJobResult{
				ID:                id,
				JobID:             job.ID,
				IntervalStartDate: job.rangeHolder.Ranges[i].Intervals[j].Start.Time,
				IntervalEndDate:   job.rangeHolder.Ranges[i].Intervals[j].End.Time,
				Status:            status,
			}
			// processing the job
			switch job.DataType {
			case dataHistoryCandleDataType:
				candles, err := exch.GetHistoricCandlesExtended(job.Pair, job.Asset, job.rangeHolder.Ranges[i].Intervals[j].Start.Time, job.rangeHolder.Ranges[i].Intervals[j].End.Time, job.Interval)
				if err != nil {
					result.Result = "could not get candles: " + err.Error()
					result.Status = dataHistoryStatusFailed
					break
				}
				_ = job.rangeHolder.VerifyResultsHaveData(candles.Candles)
				_, err = kline.StoreInDatabase(&candles, true)
				if err != nil {
					result.Result = "could not save results: " + err.Error()
					result.Status = dataHistoryStatusFailed
				}
			case dataHistoryTradeDataType:
				trades, err := exch.GetHistoricTrades(job.Pair, job.Asset, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time)
				if err != nil {
					result.Result = "could not get trades: " + err.Error()
					result.Status = dataHistoryStatusFailed
					break
				}
				candles, err := trade.ConvertTradesToCandles(job.Interval, trades...)
				if err != nil {
					result.Result = "could not convert candles to trades: " + err.Error()
					result.Status = dataHistoryStatusFailed
					break
				}
				_ = job.rangeHolder.VerifyResultsHaveData(candles.Candles)
				err = trade.SaveTradesToDatabase(trades...)
				if err != nil {
					result.Result = "could not save results: " + err.Error()
					result.Status = dataHistoryStatusFailed
				}
			default:
				return errUnknownDataType
			}

			lookup := job.Results[result.IntervalStartDate]
			lookup = append(lookup, result)
			job.Results[result.IntervalStartDate] = lookup
		}
	}

	completed := true
	for i := range job.rangeHolder.Ranges {
		for j := range job.rangeHolder.Ranges[i].Intervals {
			if !job.rangeHolder.Ranges[i].Intervals[j].HasData {
				completed = false
			}
		}
	}
	if completed {
		job.Status = dataHistoryStatusComplete
	}

	dbJob, err := m.convertJobToDBModel(job)
	if err != nil {
		return err
	}

	err = m.jobDB.Upsert(dbJob)
	if err != nil {
		return err
	}

	dbJobResults := m.convertJobResultToDBResult(job.Results)
	return m.jobResultDB.Upsert(dbJobResults...)
}

// GenerateJobSummary returns a human readable summary of a job's status
func (m *DataHistoryManager) GenerateJobSummary(nickname string) (*DataHistoryJobSummary, error) {
	job, err := m.GetByNickname(nickname, true)
	if err != nil {
		return nil, err
	}

	err = m.compareJobsToData(job)
	if err != nil {
		return nil, err
	}

	var (
		rangeStart, rangeEnd, prevStart, prevEnd time.Time
		rangeHasData                             bool
		rangeTexts                               []string
	)
	rangeStart = job.StartDate
	for i := range job.rangeHolder.Ranges {
		for j := range job.rangeHolder.Ranges[i].Intervals {
			if job.rangeHolder.Ranges[i].Intervals[j].HasData {
				if !rangeHasData && !rangeEnd.IsZero() {
					rangeTexts = append(rangeTexts, m.createDateSummaryRange(rangeStart, rangeEnd, rangeHasData))
					prevStart = rangeStart
					prevEnd = rangeEnd
					rangeStart = job.rangeHolder.Ranges[i].Intervals[j].Start.Time
				}
				rangeHasData = true
			} else {
				if rangeHasData && !rangeEnd.IsZero() {
					rangeTexts = append(rangeTexts, m.createDateSummaryRange(rangeStart, rangeEnd, rangeHasData))
					prevStart = rangeStart
					prevEnd = rangeEnd
					rangeStart = job.rangeHolder.Ranges[i].Intervals[j].Start.Time
				}
				rangeHasData = false
			}
			rangeEnd = job.rangeHolder.Ranges[i].Intervals[j].End.Time
		}
	}
	if !rangeStart.Equal(prevStart) || !rangeEnd.Equal(prevEnd) {
		rangeTexts = append(rangeTexts, m.createDateSummaryRange(rangeStart, rangeEnd, rangeHasData))
	}

	return &DataHistoryJobSummary{
		Nickname:     job.Nickname,
		Exchange:     job.Exchange,
		Asset:        job.Asset,
		Pair:         job.Pair,
		StartDate:    job.StartDate,
		EndDate:      job.EndDate,
		Interval:     job.Interval,
		Status:       job.Status,
		DataType:     job.DataType,
		ResultRanges: rangeTexts,
	}, nil
}

func (m *DataHistoryManager) createDateSummaryRange(start, end time.Time, hasData bool) string {
	dataString := "missing"
	if hasData {
		dataString = "has"
	}

	return fmt.Sprintf("%s data between %s and %s",
		dataString,
		start.Format(common.SimpleTimeFormat),
		end.Format(common.SimpleTimeFormat))
}

// ----------------------------Lovely-converters----------------------------
func (m *DataHistoryManager) convertDBModelToJob(dbModel *datahistoryjob.DataHistoryJob) (*DataHistoryJob, error) {
	id, err := uuid.FromString(dbModel.ID)
	if err != nil {
		return nil, err
	}
	cp, err := currency.NewPairFromString(fmt.Sprintf("%s-%s", dbModel.Base, dbModel.Quote))
	if err != nil {
		return nil, err
	}

	jobResults, err := m.convertDBResultToJobResult(dbModel.Results)
	if err != nil {
		return nil, err
	}

	return &DataHistoryJob{
		ID:               id,
		Nickname:         dbModel.Nickname,
		Exchange:         dbModel.ExchangeName,
		Asset:            asset.Item(dbModel.Asset),
		Pair:             cp,
		StartDate:        dbModel.StartDate,
		EndDate:          dbModel.EndDate,
		Interval:         kline.Interval(dbModel.Interval),
		RequestSizeLimit: dbModel.RequestSizeLimit,
		DataType:         dataHistoryDataType(dbModel.DataType),
		MaxRetryAttempts: dbModel.MaxRetryAttempts,
		Status:           dataHistoryStatus(dbModel.Status),
		CreatedDate:      dbModel.CreatedDate,
		Results:          jobResults,
	}, nil
}

func (m *DataHistoryManager) convertDBResultToJobResult(dbModels []*datahistoryjobresult.DataHistoryJobResult) (map[time.Time][]DataHistoryJobResult, error) {
	result := make(map[time.Time][]DataHistoryJobResult)
	for i := range dbModels {
		id, err := uuid.FromString(dbModels[i].ID)
		if err != nil {
			return nil, err
		}

		jobID, err := uuid.FromString(dbModels[i].JobID)
		if err != nil {
			return nil, err
		}
		lookup := result[dbModels[i].IntervalStartDate]
		lookup = append(lookup, DataHistoryJobResult{
			ID:                id,
			JobID:             jobID,
			IntervalStartDate: dbModels[i].IntervalStartDate,
			IntervalEndDate:   dbModels[i].IntervalEndDate,
			Status:            dataHistoryStatus(dbModels[i].Status),
			Result:            dbModels[i].Result,
			Date:              dbModels[i].Date,
		})
		// double check
		result[dbModels[i].IntervalStartDate] = lookup
	}

	return result, nil
}

func (m *DataHistoryManager) convertJobResultToDBResult(results map[time.Time][]DataHistoryJobResult) []*datahistoryjobresult.DataHistoryJobResult {
	var response []*datahistoryjobresult.DataHistoryJobResult
	for _, v := range results {
		for i := range v {
			response = append(response, &datahistoryjobresult.DataHistoryJobResult{
				ID:                v[i].ID.String(),
				JobID:             v[i].JobID.String(),
				IntervalStartDate: v[i].IntervalStartDate,
				IntervalEndDate:   v[i].IntervalEndDate,
				Status:            int64(v[i].Status),
				Result:            v[i].Result,
				Date:              v[i].Date,
			})
		}
	}
	return response
}

func (m *DataHistoryManager) convertJobToDBModel(models *DataHistoryJob) (*datahistoryjob.DataHistoryJob, error) {
	exchangeID, err := exchangedb.One(strings.ToLower(models.Exchange))
	if err != nil {
		return nil, fmt.Errorf("%s %w. %s", models.Exchange, err, "please ensure exchange table setup")
	}
	return &datahistoryjob.DataHistoryJob{
		ID:               models.ID.String(),
		Nickname:         models.Nickname,
		ExchangeName:     models.Exchange,
		ExchangeID:       exchangeID.UUID.String(),
		Asset:            models.Asset.String(),
		Base:             models.Pair.Base.String(),
		Quote:            models.Pair.Quote.String(),
		StartDate:        models.StartDate,
		EndDate:          models.EndDate,
		Interval:         int64(models.Interval.Duration()),
		RequestSizeLimit: models.RequestSizeLimit,
		DataType:         int64(models.DataType),
		MaxRetryAttempts: models.MaxRetryAttempts,
		Status:           int64(models.Status),
		CreatedDate:      models.CreatedDate,
		BatchSize:        models.RunBatchLimit,
		Results:          m.convertJobResultToDBResult(models.Results),
	}, nil
}
