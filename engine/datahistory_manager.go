package engine

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database/repository/candle"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjobresult"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// SetupDataHistoryManager creates a data history manager subsystem
func SetupDataHistoryManager(em iExchangeManager, dcm iDatabaseConnectionManager, cfg *config.DataHistoryManager) (*DataHistoryManager, error) {
	if em == nil {
		return nil, errNilExchangeManager
	}
	if dcm == nil {
		return nil, errNilDatabaseConnectionManager
	}
	if cfg == nil {
		return nil, errNilConfig
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = defaultDataHistoryTicker
	}
	if cfg.MaxJobsPerCycle == 0 {
		cfg.MaxJobsPerCycle = defaultDataHistoryMaxJobsPerCycle
	}
	db := dcm.GetInstance()
	dhj, err := datahistoryjob.Setup(db)
	if err != nil {
		return nil, err
	}
	dhjr, err := datahistoryjobresult.Setup(db)
	if err != nil {
		return nil, err
	}

	return &DataHistoryManager{
		exchangeManager:            em,
		databaseConnectionInstance: db,
		shutdown:                   make(chan struct{}),
		interval:                   time.NewTicker(cfg.CheckInterval),
		jobDB:                      dhj,
		jobResultDB:                dhjr,
		maxJobsPerCycle:            cfg.MaxJobsPerCycle,
		verbose:                    cfg.Verbose,
		tradeChecker:               trade.HasTradesInRanges,
		tradeLoader:                trade.GetTradesInRange,
		tradeSaver:                 trade.SaveTradesToDatabase,
		candleLoader:               kline.LoadFromDatabase,
		candleSaver:                kline.StoreInDatabase,
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
	m.run()
	log.Debugf(log.DataHistory, "Data history manager %v", MsgSubSystemStarted)

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
	log.Debugf(log.DataHistory, "Data history manager %v", MsgSubSystemShutdown)
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
		err = m.validateJob(dbJob)
		if err != nil {
			log.Error(log.DataHistory, err)
			continue
		}
		response = append(response, dbJob)
	}

	return response, nil
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
		defer func() {
			err = m.Stop()
			if err != nil {
				log.Error(log.DataHistory, err)
			}
		}()
		return nil, fmt.Errorf("error retrieving jobs, has everything been setup? Data history manager will shut down. %w", err)
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
		case dataHistoryCandleDataType, dataHistoryCandleValidationDataType:
			candles, err = m.candleLoader(jobs[i].Exchange, jobs[i].Pair, jobs[i].Asset, jobs[i].Interval, jobs[i].StartDate, jobs[i].EndDate)
			if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
				return fmt.Errorf("%s could not load candle data: %w", jobs[i].Nickname, err)
			}
			jobs[i].rangeHolder.SetHasDataFromCandles(candles.Candles)
		case dataHistoryTradeDataType:
			err := m.tradeChecker(jobs[i].Exchange, jobs[i].Asset.String(), jobs[i].Pair.Base.String(), jobs[i].Pair.Quote.String(), jobs[i].rangeHolder)
			if err != nil && err != sql.ErrNoRows {
				return fmt.Errorf("%s could not load trade data: %w", jobs[i].Nickname, err)
			}
		case dataHistoryConvertCandlesDataType, dataHistoryConvertTradesDataType:
			candles, err = m.candleLoader(jobs[i].Exchange, jobs[i].Pair, jobs[i].Asset, jobs[i].ConversionInterval, jobs[i].StartDate, jobs[i].EndDate)
			if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
				return fmt.Errorf("%s could not load candle data: %w", jobs[i].Nickname, err)
			}
			jobs[i].rangeHolder.SetHasDataFromCandles(candles.Candles)
		default:
			return fmt.Errorf("%s %w %s", jobs[i].Nickname, errUnknownDataType, jobs[i].DataType)
		}
	}
	return nil
}

func (m *DataHistoryManager) run() {
	go func() {
		validJobs, err := m.PrepareJobs()
		if err != nil {
			log.Error(log.DataHistory, err)
		}
		m.m.Lock()
		m.jobs = validJobs
		m.m.Unlock()

		for {
			select {
			case <-m.shutdown:
				return
			case <-m.interval.C:
				if m.databaseConnectionInstance.IsConnected() {
					go func() {
						if err := m.runJobs(); err != nil {
							log.Error(log.DataHistory, err)
						}
					}()
				}
			}
		}
	}()
}

func (m *DataHistoryManager) runJobs() error {
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}

	if !atomic.CompareAndSwapInt32(&m.processing, 0, 1) {
		return fmt.Errorf("runJobs %w", errAlreadyRunning)
	}
	defer atomic.StoreInt32(&m.processing, 0)

	validJobs, err := m.PrepareJobs()
	if err != nil {
		return err
	}

	m.m.Lock()
	defer func() {
		m.m.Unlock()
	}()
	m.jobs = validJobs
	log.Infof(log.DataHistory, "processing data history jobs")
	for i := 0; (i < int(m.maxJobsPerCycle) || m.maxJobsPerCycle == -1) && i < len(m.jobs); i++ {
		err := m.runJob(m.jobs[i])
		if err != nil {
			log.Error(log.DataHistory, err)
		}
		if m.verbose {
			log.Debugf(log.DataHistory, "completed run of data history job %v", m.jobs[i].Nickname)
		}
	}
	log.Infof(log.DataHistory, "completed run of data history jobs")

	return nil
}

// runJob processes an active job, retrieves candle or trade data
// for a given date range and saves all results to the database
func (m *DataHistoryManager) runJob(job *DataHistoryJob) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}
	if job.Status != dataHistoryStatusActive {
		return nil
	}
	var intervalsProcessed int64
	if job.rangeHolder == nil || len(job.rangeHolder.Ranges) == 0 {
		return fmt.Errorf("%s %w invalid start/end range %s-%s",
			job.Nickname,
			errJobInvalid,
			job.StartDate.Format(common.SimpleTimeFormatWithTimezone),
			job.EndDate.Format(common.SimpleTimeFormatWithTimezone),
		)
	}

	exch := m.exchangeManager.GetExchangeByName(job.Exchange)
	if exch == nil {
		return fmt.Errorf("%s %w, cannot process job %s for %s %s",
			job.Exchange,
			errExchangeNotLoaded,
			job.Nickname,
			job.Asset,
			job.Pair)
	}
	if m.verbose {
		log.Debugf(log.DataHistory, "running data history job %v start: %s end: %s interval: %s datatype: %s",
			job.Nickname,
			job.StartDate,
			job.EndDate,
			job.Interval,
			job.DataType)
	}
	var result *DataHistoryJobResult
	var err error

	if job.DataType == dataHistoryCandleValidationDataType {
		// in order to verify that an area has been checked, we need to create a datahistoryresult
		var failures int64
		startDate := job.StartDate
	validations:
		for k, v := range job.Results {
			for i := range v {
				switch v[i].Status {
				case dataHistoryIntervalMissingData:
					continue validations
				case dataHistoryStatusFailed:
					failures++
				case dataHistoryStatusComplete:
					// this can occur in the scenario where data is missing
					// however no errors were encountered when data is missing
					// eg an exchange only returns an empty slice
					// or the exchange is simply missing the data and does not have an error
					startDate = k.Add(job.Interval.Duration())
					continue validations
				}
			}
			if failures >= job.MaxRetryAttempts {
				// failure threshold reached, we should not attempt
				// to check this interval again
				for x := range v {
					v[x].Status = dataHistoryIntervalMissingData
				}
				job.Results[k] = v
				startDate = k.Add(job.Interval.Duration())
				continue
			}
		}
		for i := int64(0); i < job.RunBatchLimit; i++ {
			requestEnd := startDate.Add(job.Interval.Duration() * time.Duration(job.RequestSizeLimit))
			if requestEnd.After(job.EndDate) {
				requestEnd = job.EndDate
			}
			result, err = m.validateCandles(job, exch, startDate, requestEnd)
			if err != nil {
				return err
			}

			lookup := job.Results[result.IntervalStartDate]
			lookup = append(lookup, *result)
			job.Results[result.IntervalStartDate] = lookup
			startDate = startDate.Add(job.Interval.Duration() * time.Duration(job.RequestSizeLimit))
		}
	} else {
	ranges:
		for i := range job.rangeHolder.Ranges {
			skipProcessing := true
			for j := range job.rangeHolder.Ranges[i].Intervals {
				if !job.rangeHolder.Ranges[i].Intervals[j].HasData {
					skipProcessing = false
					break
				}
			}
			if skipProcessing {
				_, ok := job.Results[job.rangeHolder.Ranges[i].Start.Time]
				if !ok && !job.OverwriteExistingData {
					// we have determined that data is there, however it is not reflected in
					// this specific job's results, which is required for a job to be complete
					id, err := uuid.NewV4()
					if err != nil {
						return err
					}
					job.Results[job.rangeHolder.Ranges[i].Start.Time] = []DataHistoryJobResult{
						{
							ID:                id,
							JobID:             job.ID,
							IntervalStartDate: job.rangeHolder.Ranges[i].Start.Time,
							IntervalEndDate:   job.rangeHolder.Ranges[i].End.Time,
							Status:            dataHistoryStatusComplete,
							Date:              time.Now(),
						},
					}
				}
				if !job.OverwriteExistingData {
					continue
				}
			}
			if intervalsProcessed >= job.RunBatchLimit {
				continue
			}

			var failures int64
			hasDataInRange := false
			resultLookup := job.Results[job.rangeHolder.Ranges[i].Start.Time]
			for x := range resultLookup {
				switch resultLookup[x].Status {
				case dataHistoryIntervalMissingData:
					continue ranges
				case dataHistoryStatusFailed:
					failures++
				case dataHistoryStatusComplete:
					// this can occur in the scenario where data is missing
					// however no errors were encountered when data is missing
					// eg an exchange only returns an empty slice
					// or the exchange is simply missing the data and does not have an error
					hasDataInRange = true
				}
			}
			if failures >= job.MaxRetryAttempts {
				// failure threshold reached, we should not attempt
				// to check this interval again
				for x := range resultLookup {
					resultLookup[x].Status = dataHistoryIntervalMissingData
				}
				job.Results[job.rangeHolder.Ranges[i].Start.Time] = resultLookup
				continue
			}
			if hasDataInRange {
				continue
			}
			if m.verbose {
				log.Debugf(log.DataHistory, "job %s processing range %v-%v", job.Nickname, job.rangeHolder.Ranges[i].Start, job.rangeHolder.Ranges[i].End)
			}
			intervalsProcessed++

			// processing the job
			switch job.DataType {
			case dataHistoryCandleDataType:
				result, err = m.processCandleData(job, exch, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time, int64(i))
			case dataHistoryTradeDataType:
				result, err = m.processTradeData(job, exch, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time, int64(i))
			case dataHistoryConvertTradesDataType:
				result, err = m.convertJobTradesToCandles(job, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time)
			case dataHistoryConvertCandlesDataType:
				result, err = m.upscaleJobCandleData(job, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time)
			default:
				return errUnknownDataType
			}
			if err != nil {
				return err
			}
			if result == nil {
				return errNilResult
			}

			lookup := job.Results[result.IntervalStartDate]
			lookup = append(lookup, *result)
			job.Results[result.IntervalStartDate] = lookup
		}
	}

	completed := true
	allResultsSuccessful := true
	allResultsFailed := true
completionCheck:
	for i := range job.rangeHolder.Ranges {
		result, ok := job.Results[job.rangeHolder.Ranges[i].Start.Time]
		if !ok {
			completed = false
		}
	results:
		for j := range result {
			switch result[j].Status {
			case dataHistoryIntervalMissingData:
				allResultsSuccessful = false
				break results
			case dataHistoryStatusComplete:
				allResultsFailed = false
				break results
			default:
				completed = false
				break completionCheck
			}
		}
	}
	if completed {
		err := m.completionCheck(job, allResultsSuccessful, allResultsFailed)
		if err != nil {
			return err
		}
	}
	dbJob := m.convertJobToDBModel(job)
	err = m.jobDB.Upsert(dbJob)
	if err != nil {
		return fmt.Errorf("job %s failed to update database: %w", job.Nickname, err)
	}

	dbJobResults := m.convertJobResultToDBResult(job.Results)
	err = m.jobResultDB.Upsert(dbJobResults...)
	if err != nil {
		return fmt.Errorf("job %s failed to insert job results to database: %w", job.Nickname, err)
	}
	return nil
}

func (m *DataHistoryManager) completionCheck(job *DataHistoryJob, allResultsSuccessful, allResultsFailed bool) error {
	if job == nil {
		return errNilJob
	}
	if allResultsSuccessful && allResultsFailed {
		return errJobInvalid
	}
	switch {
	case allResultsSuccessful:
		job.Status = dataHistoryStatusComplete
	case allResultsFailed:
		job.Status = dataHistoryStatusFailed
	default:
		job.Status = dataHistoryIntervalMissingData
	}
	log.Infof(log.DataHistory, "job %s finished! Status: %s", job.Nickname, job.Status)
	if job.Status != dataHistoryStatusFailed {
		newJobs, err := m.jobDB.GetRelatedUpcomingJobs(job.Nickname)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var newJobNames []string
		for i := range newJobs {
			newJobs[i].Status = int64(dataHistoryStatusActive)
			newJobNames = append(newJobNames, newJobs[i].Nickname)
		}
		if len(newJobNames) > 0 {
			log.Infof(log.DataHistory, "setting the follow jobs to active: %s", strings.Join(newJobNames, ", "))
			err = m.jobDB.Upsert(newJobs...)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *DataHistoryManager) processCandleData(job *DataHistoryJob, exch exchange.IBotExchange, startRange, endRange time.Time, intervalIndex int64) (*DataHistoryJobResult, error) {
	if job == nil {
		return nil, errNilJob
	}
	if exch == nil {
		return nil, ErrExchangeNotFound
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}

	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	candles, err := exch.GetHistoricCandlesExtended(job.Pair, job.Asset, startRange, endRange, job.Interval)
	if err != nil {
		r.Result += "could not get candles: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	job.rangeHolder.SetHasDataFromCandles(candles.Candles)
	for i := range job.rangeHolder.Ranges[intervalIndex].Intervals {
		if !job.rangeHolder.Ranges[intervalIndex].Intervals[i].HasData {
			r.Status = dataHistoryStatusFailed
			r.Result += fmt.Sprintf("missing data from %v - %v. ",
				startRange.Format(common.SimpleTimeFormatWithTimezone),
				endRange.Format(common.SimpleTimeFormatWithTimezone))
		}
	}
	_, err = m.candleSaver(&candles, job.OverwriteExistingData)
	if err != nil {
		r.Result += "could not save results: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
	}
	return r, nil
}

func (m *DataHistoryManager) processTradeData(job *DataHistoryJob, exch exchange.IBotExchange, startRange, endRange time.Time, intervalIndex int64) (*DataHistoryJobResult, error) {
	if job == nil {
		return nil, errNilJob
	}
	if exch == nil {
		return nil, ErrExchangeNotFound
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	trades, err := exch.GetHistoricTrades(job.Pair, job.Asset, startRange, endRange)
	if err != nil {
		r.Result += "could not get trades: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	filteredTraders := trade.FilterTradesByTime(trades, startRange, endRange)
	candles, err := trade.ConvertTradesToCandles(job.Interval, filteredTraders...)
	if err != nil {
		r.Result += "could not convert candles to trades: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	job.rangeHolder.SetHasDataFromCandles(candles.Candles)
	for i := range job.rangeHolder.Ranges[intervalIndex].Intervals {
		if !job.rangeHolder.Ranges[intervalIndex].Intervals[i].HasData {
			r.Status = dataHistoryStatusFailed
			r.Result += fmt.Sprintf("missing data from %v - %v. ",
				job.rangeHolder.Ranges[intervalIndex].Intervals[i].Start.Time.Format(common.SimpleTimeFormatWithTimezone),
				job.rangeHolder.Ranges[intervalIndex].Intervals[i].End.Time.Format(common.SimpleTimeFormatWithTimezone))
		}
	}
	err = m.tradeSaver(trades...)
	if err != nil {
		r.Result += "could not save results: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
	}
	return r, nil
}

func (m *DataHistoryManager) convertJobTradesToCandles(job *DataHistoryJob, startRange, endRange time.Time) (*DataHistoryJobResult, error) {
	if job == nil {
		return nil, errNilJob
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	trades, err := m.tradeLoader(job.Exchange, job.Asset.String(), job.Pair.Base.String(), job.Pair.Quote.String(), startRange, endRange)
	if err != nil {
		r.Result = "could not get trades in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	candles, err := trade.ConvertTradesToCandles(job.ConversionInterval, trades...)
	if err != nil {
		r.Result = "could not convert trades in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	_, err = m.candleSaver(&candles, job.OverwriteExistingData)
	if err != nil {
		r.Result = "could not save candles in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
	}
	return r, nil
}

func (m *DataHistoryManager) upscaleJobCandleData(job *DataHistoryJob, startRange, endRange time.Time) (*DataHistoryJobResult, error) {
	if job == nil {
		return nil, errNilJob
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	candles, err := m.candleLoader(job.Exchange, job.Pair, job.Asset, job.Interval, startRange, endRange)
	if err != nil {
		r.Result = "could not get candles in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	newCandles, err := kline.ConvertToNewInterval(&candles, job.ConversionInterval)
	if err != nil {
		r.Result = "could not convert candles in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	_, err = m.candleSaver(newCandles, job.OverwriteExistingData)
	if err != nil {
		r.Result = "could not save candles in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
	}
	return r, nil
}

func (m *DataHistoryManager) validateCandles(job *DataHistoryJob, exch exchange.IBotExchange, startRange, endRange time.Time) (*DataHistoryJobResult, error) {
	if job == nil {
		return nil, errNilJob
	}
	if exch == nil {
		return nil, ErrExchangeNotFound
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}

	apiCandles, err := exch.GetHistoricCandlesExtended(job.Pair, job.Asset, startRange, endRange, job.Interval)
	if err != nil {
		r.Result = "could not get API candles: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}

	dbCandles, err := m.candleLoader(job.Exchange, job.Pair, job.Asset, job.Interval, startRange, endRange)
	if err != nil {
		r.Result = "could not get database candles: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	dbCandleMap := make(map[int64]kline.Candle)
	for i := range dbCandles.Candles {
		dbCandleMap[dbCandles.Candles[i].Time.Unix()] = dbCandles.Candles[i]
	}
	var validationIssues []string
	multiplier := float64(1)
	for i := int64(0); i < job.DecimalPlaceComparison; i++ {
		multiplier *= 10
	}
	for i := range apiCandles.Candles {
		can, ok := dbCandleMap[apiCandles.Candles[i].Time.Unix()]
		if !ok {
			validationIssues = append(validationIssues, fmt.Sprintf("issues found at %v missing candle data in database", apiCandles.Candles[i].Time.Format(common.SimpleTimeFormatWithTimezone)))
			r.Status = dataHistoryIntervalMissingData
			continue
		}
		var (
			candleIssues []string
			rAPIHigh, rAPILow, rAPIClose, rAPIOpen, rAPIVolume,
			rDBHigh, rDBLow, rDBClose, rDBOpen, rDBVolume float64
		)
		rAPIOpen = math.Round(apiCandles.Candles[i].Open*multiplier) / multiplier
		rAPIHigh = math.Round(apiCandles.Candles[i].High*multiplier) / multiplier
		rAPILow = math.Round(apiCandles.Candles[i].Low*multiplier) / multiplier
		rAPIClose = math.Round(apiCandles.Candles[i].Close*multiplier) / multiplier
		rAPIVolume = math.Round(apiCandles.Candles[i].Volume*multiplier) / multiplier
		rDBOpen = math.Round(can.Open*multiplier) / multiplier
		rDBHigh = math.Round(can.High*multiplier) / multiplier
		rDBLow = math.Round(can.Low*multiplier) / multiplier
		rDBClose = math.Round(can.Close*multiplier) / multiplier
		rDBVolume = math.Round(can.Volume*multiplier) / multiplier

		if rAPIHigh != rDBHigh {
			candleIssues = append(candleIssues, fmt.Sprintf("High api: %v db: %v", rAPIHigh, rDBHigh))
		}
		if rAPILow != rDBLow {
			candleIssues = append(candleIssues, fmt.Sprintf("Low api: %v db: %v", rAPILow, rDBLow))
		}
		if rAPIClose != rDBClose {
			candleIssues = append(candleIssues, fmt.Sprintf("Close api: %v db: %v", rAPIClose, rDBClose))
		}
		if rAPIOpen != rDBOpen {
			candleIssues = append(candleIssues, fmt.Sprintf("Open api: %v db: %v", rAPIOpen, rDBOpen))
		}
		if rAPIVolume != rDBVolume {
			candleIssues = append(candleIssues, fmt.Sprintf("Volume api: %v db: %v", rAPIVolume, rDBVolume))
		}
		if len(candleIssues) > 0 {
			candleIssues = append([]string{fmt.Sprintf("issues found at %v", can.Time.Format(common.SimpleTimeFormat))}, candleIssues...)
			validationIssues = append(validationIssues, candleIssues...)
			r.Status = dataHistoryStatusFailed
		}
	}
	if len(validationIssues) > 0 {
		r.Result = strings.Join(validationIssues, " -- ")
	}
	return r, nil
}

// SetJobRelationship will add/modify/delete a relationship with an existing job
// it will add the relationship and set the jobNickname job to paused
// if deleting, it will remove the relationship from the database and set the job to active
func (m *DataHistoryManager) SetJobRelationship(prerequisiteJobNickname, jobNickname string) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return ErrSubSystemNotStarted
	}
	status := dataHistoryStatusPaused
	if jobNickname == "" {
		return errNicknameUnset
	}
	if prerequisiteJobNickname == "" {
		status = dataHistoryStatusActive
	}
	return m.jobDB.SetRelationshipByNickname(prerequisiteJobNickname, jobNickname, int64(status))
}

// UpsertJob allows for GRPC interaction to upsert a job to be processed
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
		return fmt.Errorf("upsert job %w", errNicknameUnset)
	}

	if m == nil {
		return ErrNilSubsystem
	}
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
	j, err := m.GetByNickname(job.Nickname, false)
	if err != nil && !errors.Is(err, errJobNotFound) {
		return err
	}
	if insertOnly && j != nil ||
		(j != nil && j.Status != dataHistoryStatusActive) {
		return fmt.Errorf("upsert job %w nickname: %s - status: %s ", errNicknameInUse, j.Nickname, j.Status)
	}

	err = m.validateJob(job)
	if err != nil {
		return err
	}
	isUpdatingExistingJob := false
	m.m.Lock()
	defer m.m.Unlock()
	if !insertOnly {
		for i := range m.jobs {
			if !strings.EqualFold(m.jobs[i].Nickname, job.Nickname) {
				continue
			}
			isUpdatingExistingJob = true
			job.ID = m.jobs[i].ID
			m.jobs[i].OverwriteExistingData = job.OverwriteExistingData

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
			if job.ConversionInterval != 0 && m.jobs[i].ConversionInterval != job.ConversionInterval {
				m.jobs[i].ConversionInterval = job.ConversionInterval
			}
			if job.DecimalPlaceComparison != 0 && m.jobs[i].DecimalPlaceComparison != job.DecimalPlaceComparison {
				m.jobs[i].DecimalPlaceComparison = job.DecimalPlaceComparison
			}
			m.jobs[i].DataType = job.DataType
			m.jobs[i].Status = job.Status
			break
		}
	}
	if job.ID == uuid.Nil {
		job.ID, err = uuid.NewV4()
		if err != nil {
			return err
		}
	}
	interval := job.Interval
	if job.DataType == dataHistoryConvertTradesDataType ||
		job.DataType == dataHistoryConvertCandlesDataType {
		interval = job.ConversionInterval
	}
	job.rangeHolder, err = kline.CalculateCandleDateRanges(job.StartDate, job.EndDate, interval, uint32(job.RequestSizeLimit))
	if err != nil {
		return err
	}

	if !isUpdatingExistingJob {
		m.jobs = append(m.jobs, job)
	}

	dbJob := m.convertJobToDBModel(job)
	err = m.jobDB.Upsert(dbJob)
	if err != nil {
		return err
	}
	if job.PrerequisiteJobNickname == "" {
		return nil
	}
	job.Status = dataHistoryStatusPaused
	// only allow new jobs to create associations.
	// updating/removing existing associations is its own task
	return m.jobDB.SetRelationshipByNickname(job.PrerequisiteJobNickname, job.Nickname, int64(dataHistoryStatusPaused))
}

func (m *DataHistoryManager) validateJob(job *DataHistoryJob) error {
	if job == nil {
		return errNilJob
	}
	if !job.Asset.IsValid() {
		return fmt.Errorf("job %s %w %s", job.Nickname, asset.ErrNotSupported, job.Asset)
	}
	if job.Pair.IsEmpty() {
		return fmt.Errorf("job %s %w", job.Nickname, errCurrencyPairUnset)
	}
	if !job.Status.Valid() {
		return fmt.Errorf("job %s %w: %s", job.Nickname, errInvalidDataHistoryStatus, job.Status)
	}
	if !job.DataType.Valid() {
		return fmt.Errorf("job %s %w: %s", job.Nickname, errInvalidDataHistoryDataType, job.DataType)
	}
	exch := m.exchangeManager.GetExchangeByName(job.Exchange)
	if exch == nil {
		return fmt.Errorf("job %s cannot process job: %s %w",
			job.Nickname,
			job.Exchange,
			errExchangeNotLoaded)
	}
	pairs, err := exch.GetEnabledPairs(job.Asset)
	if err != nil {
		return fmt.Errorf("job %s exchange %s asset %s currency %s %w", job.Nickname, job.Exchange, job.Asset, job.Pair, err)
	}
	if !pairs.Contains(job.Pair, false) {
		return fmt.Errorf("job %s exchange %s asset %s currency %s %w", job.Nickname, job.Exchange, job.Asset, job.Pair, errCurrencyNotEnabled)
	}
	if job.Results == nil {
		job.Results = make(map[time.Time][]DataHistoryJobResult)
	}
	if job.RunBatchLimit <= 0 {
		log.Warnf(log.DataHistory, "job %s has unset batch limit, defaulting to %v", job.Nickname, defaultDataHistoryBatchLimit)
		job.RunBatchLimit = defaultDataHistoryBatchLimit
	}
	if job.MaxRetryAttempts <= 0 {
		log.Warnf(log.DataHistory, "job %s has unset max retry limit, defaulting to %v", job.Nickname, defaultDataHistoryRetryAttempts)
		job.MaxRetryAttempts = defaultDataHistoryRetryAttempts
	}
	if job.RequestSizeLimit <= 0 {
		job.RequestSizeLimit = defaultDataHistoryRequestSizeLimit
	}
	if job.DataType == dataHistoryTradeDataType {
		if job.Interval >= kline.FourHour || job.Interval <= kline.OneMin {
			log.Warnf(log.DataHistory, "job %s interval %v outside limit of 4h, defaulting to %v", job.Nickname, job.Interval.Word(), defaultDataHistoryTradeInterval)
			job.Interval = defaultDataHistoryTradeInterval
		}
		if job.RequestSizeLimit > 20 {
			log.Warnf(log.DataHistory, "job %s request size %v outside limit of 100, defaulting to %v", job.Nickname, job.RequestSizeLimit, defaultDataHistoryTradeRequestSize)
			job.RequestSizeLimit = defaultDataHistoryTradeRequestSize
		}
	}

	b := exch.GetBase()
	if !b.Features.Enabled.Kline.Intervals[job.Interval.Word()] && (job.DataType != dataHistoryTradeDataType && job.DataType != dataHistoryConvertTradesDataType) {
		return fmt.Errorf("job interval %s %s %w %s", job.Nickname, job.Interval.Word(), kline.ErrUnsupportedInterval, job.Exchange)
	}

	if (job.DataType == dataHistoryConvertTradesDataType || job.DataType == dataHistoryConvertCandlesDataType) &&
		job.ConversionInterval <= 0 {
		return fmt.Errorf("job conversion interval %s %s %w %s", job.Nickname, job.ConversionInterval.Word(), kline.ErrUnsupportedInterval, job.Exchange)
	}

	if job.DecimalPlaceComparison <= 0 && job.DataType == dataHistoryCandleValidationDataType {
		log.Warnf(log.DataHistory, "job %s decimal place comparison %v invalid. defaulting to %v", job.Nickname, job.DecimalPlaceComparison, defaultDecimalPlaceComparison)
		job.DecimalPlaceComparison = defaultDecimalPlaceComparison
	}

	job.StartDate = job.StartDate.Round(job.Interval.Duration())
	job.EndDate = job.EndDate.Round(job.Interval.Duration())
	if err := common.StartEndTimeCheck(job.StartDate, job.EndDate); err != nil {
		return fmt.Errorf("job %s %w start: %v end %v", job.Nickname, err, job.StartDate, job.EndDate)
	}

	return nil
}

// GetByID returns a job's details from its ID
func (m *DataHistoryManager) GetByID(id uuid.UUID) (*DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, ErrSubSystemNotStarted
	}
	if id == uuid.Nil {
		return nil, errEmptyID
	}
	m.m.Lock()
	for i := range m.jobs {
		if m.jobs[i].ID == id {
			cpy := *m.jobs[i]
			m.m.Unlock()
			return &cpy, nil
		}
	}
	m.m.Unlock()
	dbJ, err := m.jobDB.GetByID(id.String())
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
			return nil, fmt.Errorf("job %s could not load job from database: %w", nickname, err)
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
		if err == sql.ErrNoRows {
			// no need to display normal sql err to user
			return nil, errJobNotFound
		}
		return nil, fmt.Errorf("job %s %w, %s", nickname, errJobNotFound, err)
	}
	job, err := m.convertDBModelToJob(j)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// GetAllJobStatusBetween will return all jobs between two ferns
func (m *DataHistoryManager) GetAllJobStatusBetween(start, end time.Time) ([]*DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if atomic.LoadInt32(&m.started) == 0 {
		return nil, ErrSubSystemNotStarted
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
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

// SetJobStatus helper function to assist in setting a job to deleted
func (m *DataHistoryManager) SetJobStatus(nickname, id string, status dataHistoryStatus) error {
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
	if status != dataHistoryStatusPaused &&
		status != dataHistoryStatusRemoved &&
		status != dataHistoryStatusActive {
		return fmt.Errorf("%w %s", errCannotSetJobStatus, status.String())
	}
	var dbJob *datahistoryjob.DataHistoryJob
	var err error
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, nickname) ||
			m.jobs[i].ID.String() == id {
			dbJob = m.convertJobToDBModel(m.jobs[i])
			if status == dataHistoryStatusPaused ||
				status == dataHistoryStatusRemoved {
				m.jobs = append(m.jobs[:i], m.jobs[i+1:]...)
			}
			break
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
	dbJob.Status = int64(status)
	err = m.jobDB.Upsert(dbJob)
	if err != nil {
		return err
	}
	if status == dataHistoryStatusActive {
		job, err := m.convertDBModelToJob(dbJob)
		if err != nil {
			return err
		}
		m.jobs = append(m.jobs, job)
	}
	log.Infof(log.DataHistory, "set job %v status to %v", dbJob.Nickname, status.String())
	return nil
}

// GetActiveJobs returns all jobs with the status `dataHistoryStatusActive`
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

// GenerateJobSummary returns a human readable summary of a job's status
func (m *DataHistoryManager) GenerateJobSummary(nickname string) (*DataHistoryJobSummary, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	job, err := m.GetByNickname(nickname, false)
	if err != nil {
		return nil, fmt.Errorf("job: %v %w", nickname, err)
	}

	err = m.compareJobsToData(job)
	if err != nil {
		return nil, err
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
		ResultRanges: job.rangeHolder.DataSummary(true),
	}, nil
}

// ----------------------------Lovely-converters----------------------------
func (m *DataHistoryManager) convertDBModelToJob(dbModel *datahistoryjob.DataHistoryJob) (*DataHistoryJob, error) {
	id, err := uuid.FromString(dbModel.ID)
	if err != nil {
		return nil, err
	}
	cp, err := currency.NewPairFromString(fmt.Sprintf("%s-%s", dbModel.Base, dbModel.Quote))
	if err != nil {
		return nil, fmt.Errorf("job %s could not format pair %s-%s: %w", dbModel.Nickname, dbModel.Base, dbModel.Quote, err)
	}

	jobResults, err := m.convertDBResultToJobResult(dbModel.Results)
	if err != nil {
		return nil, fmt.Errorf("job %s could not convert database job: %w", dbModel.Nickname, err)
	}

	resp := &DataHistoryJob{
		ID:                      id,
		Nickname:                dbModel.Nickname,
		Exchange:                dbModel.ExchangeName,
		Asset:                   asset.Item(dbModel.Asset),
		Pair:                    cp,
		StartDate:               dbModel.StartDate,
		EndDate:                 dbModel.EndDate,
		Interval:                kline.Interval(dbModel.Interval),
		RunBatchLimit:           dbModel.BatchSize,
		RequestSizeLimit:        dbModel.RequestSizeLimit,
		DataType:                dataHistoryDataType(dbModel.DataType),
		MaxRetryAttempts:        dbModel.MaxRetryAttempts,
		Status:                  dataHistoryStatus(dbModel.Status),
		CreatedDate:             dbModel.CreatedDate,
		Results:                 jobResults,
		PrerequisiteJobNickname: dbModel.PrerequisiteJobNickname,
		OverwriteExistingData:   dbModel.OverwriteData,
		ConversionInterval:      kline.Interval(dbModel.ConversionInterval),
	}
	if resp.PrerequisiteJobNickname != "" {
		prereqID, err := uuid.FromString(dbModel.PrerequisiteJobID)
		if err != nil {
			return nil, err
		}
		resp.PrerequisiteJobID = prereqID
	}

	return resp, nil
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

func (m *DataHistoryManager) convertJobToDBModel(job *DataHistoryJob) *datahistoryjob.DataHistoryJob {
	model := &datahistoryjob.DataHistoryJob{
		Nickname:                job.Nickname,
		ExchangeName:            job.Exchange,
		Asset:                   job.Asset.String(),
		Base:                    job.Pair.Base.String(),
		Quote:                   job.Pair.Quote.String(),
		StartDate:               job.StartDate,
		EndDate:                 job.EndDate,
		Interval:                int64(job.Interval.Duration()),
		RequestSizeLimit:        job.RequestSizeLimit,
		DataType:                int64(job.DataType),
		MaxRetryAttempts:        job.MaxRetryAttempts,
		BatchSize:               job.RunBatchLimit,
		Status:                  int64(job.Status),
		CreatedDate:             job.CreatedDate,
		Results:                 m.convertJobResultToDBResult(job.Results),
		PrerequisiteJobNickname: job.PrerequisiteJobNickname,
		ConversionInterval:      int64(job.ConversionInterval),
		OverwriteData:           job.OverwriteExistingData,
	}
	if job.ID != uuid.Nil {
		model.ID = job.ID.String()
	}
	if job.PrerequisiteJobID != uuid.Nil {
		model.PrerequisiteJobID = job.PrerequisiteJobID.String()
	}

	return model
}
