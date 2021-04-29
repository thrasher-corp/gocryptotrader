package engine

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database/repository/datahistoryjob"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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

	jobs, err := retrieveJobs(dcm)
	if err != nil {
		return nil, err
	}

	dcm.GetSQL()
	dhj, err := datahistoryjob.Setup(dcm)

	return &DataHistoryManager{
		exchangeManager:           em,
		databaseConnectionManager: dcm,
		shutdown:                  make(chan struct{}),
		interval:                  time.NewTicker(processInterval),
		jobs:                      jobs,
		dataHistoryDB:             dhj,
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

	validJobs := m.PrepareJobs()
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
func retrieveJobs(dcm iDatabaseConnectionManager) ([]*DataHistoryJob, error) {
	if !dcm.IsConnected() {
		return nil, errDatabaseConnectionRequired
	}
	var response []*DataHistoryJob

	return response, nil
}

// UpsertJob allows for GRPC interaction to upsert a jobs to be processed
func (m *DataHistoryManager) UpsertJob(cfg *DataHistoryJob) error {
	m.m.Lock()
	defer m.m.Unlock()
	for i := range m.jobs {
		if strings.EqualFold(m.jobs[i].Nickname, cfg.Nickname) {
			m.jobs[i] = cfg
		}
	}
	m.jobs = append(m.jobs, cfg)
	return nil
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
func (m *DataHistoryManager) PrepareJobs() []*DataHistoryJob {
	var validJobs []*DataHistoryJob
	m.m.RLock()
	defer m.m.RUnlock()
	for i := range m.jobs {
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			log.Errorf(log.DataHistory, "exchange not loaded, cannot process jobs")
			continue
		}
		m.jobs[i].ranges = kline.CalculateCandleDateRanges(m.jobs[i].StartDate, m.jobs[i].EndDate, m.jobs[i].Interval, m.jobs[i].RequestSizeLimit)

		// check the database to verify if you already have data in the range
		// if blarg then
		// m.jobs[i].ranges[x].HasData = true
		validJobs = append(validJobs, m.jobs[i])
	}
	return validJobs
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
	var jobsToRemove []*DataHistoryJob
	m.m.RLock()
	defer m.m.RUnlock()
	for i := range m.jobs {
		if len(m.jobs[i].failures) > m.jobs[i].MaxRetryAttempts {
			jobsToRemove = append(jobsToRemove, m.jobs[i])
			continue
		}
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			fail := dataHistoryFailure{reason: "exchange not loaded, cannot process job"}
			m.jobs[i].failures = append(m.jobs[i].failures, fail)
			log.Errorf(log.DataHistory, fail.reason)
			continue
		}
		m.runJob(m.jobs[i], exch)
	}
}

// runJob will process an individual job. It is either run as on a schedule
// or specifically via RPC command on demand
func (m *DataHistoryManager) runJob(job *DataHistoryJob, exch exchange.IBotExchange) {
	if m == nil || atomic.LoadInt32(&m.started) == 0 {
		return
	}
ranges:
	for j := range job.ranges.Ranges {
		// what are you doing here?
		for x := range job.ranges.Ranges[j].Intervals {
			if job.ranges.Ranges[j].Intervals[x].HasData {
				continue ranges
			}
		}
		// processing the job
		switch job.DataType {
		case CandleDataType:
			niceCans, err := exch.GetHistoricCandles(job.Pair, job.Asset, job.ranges.Ranges[j].Start.Time, job.ranges.Ranges[j].End.Time, job.Interval)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not get candles: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
			err = job.ranges.VerifyResultsHaveData(niceCans.Candles)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not verify results: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
			_, err = kline.StoreInDatabase(&niceCans, true)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not save results: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
		case TradeDataType:
			trades, err := exch.GetHistoricTrades(job.Pair, job.Asset, job.ranges.Ranges[j].Start.Time, job.ranges.Ranges[j].End.Time)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not get trades: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
			bigCans, err := trade.ConvertTradesToCandles(job.Interval, trades...)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not get convert candles to trades: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
			err = job.ranges.VerifyResultsHaveData(bigCans.Candles)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not verify results: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
			err = trade.SaveTradesToDatabase(trades...)
			if err != nil {
				fail := dataHistoryFailure{reason: "could not save results: " + err.Error()}
				job.failures = append(job.failures, fail)
				continue
			}
		}
		// insert the status of the job, is it a failure? etc etc
		err := m.dataHistoryDB.Upsert()
		if err != nil {
			// woah nelly
		}
	}
}
