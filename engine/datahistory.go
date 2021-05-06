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
func retrieveJobs(dcm iDatabaseConnectionManager) ([]*Job, error) {
	if !dcm.IsConnected() {
		return nil, errDatabaseConnectionRequired
	}
	var response []*Job

	return response, nil
}

// UpsertJob allows for GRPC interaction to upsert a jobs to be processed
func (m *DataHistoryManager) UpsertJob(cfg *Job) error {
	m.m.Lock()
	defer m.m.Unlock()
	updated := false
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, cfg.Nickname) {
			updated = true
			m.jobs[i] = cfg
			break
		}
	}
	if !updated {
		m.jobs = append(m.jobs, cfg)
	}

	return m.jobDB.Upsert(&datahistoryjob.DataHistoryJob{
		ID:               cfg.ID.String(),
		NickName:         cfg.Nickname,
		Exchange:         cfg.Exchange,
		Asset:            cfg.Asset.String(),
		Base:             cfg.Pair.Base.String(),
		Quote:            cfg.Pair.Quote.String(),
		StartDate:        cfg.StartDate,
		EndDate:          cfg.EndDate,
		Interval:         int64(cfg.Interval.Duration()),
		RequestSizeLimit: cfg.RequestSizeLimit,
		DataType:         cfg.DataType,
		MaxRetryAttempts: cfg.MaxRetryAttempts,
		Status:           cfg.Status,
		CreatedDate:      cfg.CreatedDate,
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
func (m *DataHistoryManager) PrepareJobs() ([]*Job, error) {
	var validJobs []*Job
	m.m.RLock()
	defer m.m.RUnlock()
	m.jobs = []*Job{}
	// get the db jobs
	dbJobs, err := m.jobDB.GetAllIncompleteJobsAndResults()
	if err != nil {
		return nil, err
	}
	for i := range dbJobs {
		//convert
		id, err := uuid.FromString(dbJobs[i].ID)
		if err != nil {
			return nil, err
		}
		cp, err := currency.NewPairFromString(fmt.Sprintf("%s-%s", dbJobs[i].Base, dbJobs[i].Quote))
		if err != nil {
			return nil, err
		}

		var jobResults []JobResults
		for j := range dbJobs[i].Results {
			jobResults = append(jobResults, JobResults{
				reason: dbJobs[i].Results[j].Result,
				time:   dbJobs[i].Results[j].Date,
			})
		}

		m.jobs = append(m.jobs, &Job{
			ID:               id,
			Nickname:         dbJobs[i].NickName,
			Exchange:         dbJobs[i].Exchange,
			Asset:            asset.Item(dbJobs[i].Asset),
			Pair:             cp,
			StartDate:        dbJobs[i].StartDate,
			EndDate:          dbJobs[i].EndDate,
			Interval:         kline.Interval(dbJobs[i].Interval),
			RequestSizeLimit: dbJobs[i].RequestSizeLimit,
			DataType:         dbJobs[i].DataType,
			MaxRetryAttempts: dbJobs[i].MaxRetryAttempts,
			Status:           dbJobs[i].Status,
			CreatedDate:      dbJobs[i].CreatedDate,
			Results:          jobResults,
		})

	}

	for i := range m.jobs {
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			log.Errorf(log.DataHistory, "exchange not loaded, cannot process jobs")
			continue
		}
		m.jobs[i].rangeHolder = kline.CalculateCandleDateRanges(m.jobs[i].StartDate, m.jobs[i].EndDate, m.jobs[i].Interval, uint32(m.jobs[i].RequestSizeLimit))

		// check the database to verify if you already have data in the range
		// if blarg then
		// m.jobs[i].rangeHolder[x].HasData = true
		validJobs = append(validJobs, m.jobs[i])
	}
	return validJobs, nil
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
					go m.processJobs()
				}
			}
		}
	}()
}

func (m *DataHistoryManager) processJobs() {
	var jobsToRemove []*Job
	m.m.RLock()
	defer m.m.RUnlock()
	for i := range m.jobs {
		if len(m.jobs[i].Results) > int(m.jobs[i].MaxRetryAttempts) {
			jobsToRemove = append(jobsToRemove, m.jobs[i])
			continue
		}
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			fail := JobResults{reason: "exchange not loaded, cannot process job"}
			// m.jobs[i].failures = append(m.jobs[i].failures, fail)
			log.Errorf(log.DataHistory, fail.reason)
			continue
		}
		m.runJob(m.jobs[i], exch)
	}
}

// runJob will process an individual job. It is either run as on a schedule
// or specifically via RPC command on demand
func (m *DataHistoryManager) runJob(job *Job, exch exchange.IBotExchange) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
	if job.Status == StatusComplete ||
		job.Status == StatusFailed ||
		job.Status == StatusRemoved {
		// job doesn't need to be run. Log it?
		return
	}
ranges:
	for j := range job.rangeHolder.Ranges {

		// what are you doing here?
		requiresProcessing := false
		// by nature of the job system, this is an invalid way of discovering if a job requires data
		// there needs to be a check for a jobResult for the time interval and whether it is completed or failed
		// if neither, then process the job ?
		for x := range job.rangeHolder.Ranges[j].Intervals {
			if !job.rangeHolder.Ranges[j].Intervals[x].HasData {
				requiresProcessing = true
			}
		}
		if !requiresProcessing {
			continue ranges
		}
		// processing the job
		switch job.DataType {
		case CandleDataType:
			niceCans, err := exch.GetHistoricCandles(job.Pair, job.Asset, job.rangeHolder.Ranges[j].Start.Time, job.rangeHolder.Ranges[j].End.Time, job.Interval)
			if err != nil {
				fail := JobResults{reason: "could not get candles: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
			err = job.rangeHolder.VerifyResultsHaveData(niceCans.Candles)
			if err != nil {
				fail := JobResults{reason: "could not verify results: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
			_, err = kline.StoreInDatabase(&niceCans, true)
			if err != nil {
				fail := JobResults{reason: "could not save results: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
		case TradeDataType:
			trades, err := exch.GetHistoricTrades(job.Pair, job.Asset, job.rangeHolder.Ranges[j].Start.Time, job.rangeHolder.Ranges[j].End.Time)
			if err != nil {
				fail := JobResults{reason: "could not get trades: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
			bigCans, err := trade.ConvertTradesToCandles(job.Interval, trades...)
			if err != nil {
				fail := JobResults{reason: "could not get convert candles to trades: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
			err = job.rangeHolder.VerifyResultsHaveData(bigCans.Candles)
			if err != nil {
				fail := JobResults{reason: "could not verify results: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
			err = trade.SaveTradesToDatabase(trades...)
			if err != nil {
				fail := JobResults{reason: "could not save results: " + err.Error()}
				job.Results = append(job.Results, fail)
				continue
			}
		}
		// insert the status of the job, is it a failure? etc etc
		err := m.jobDB.Upsert()
		if err != nil {
			// woah nelly
		}
	}
}
