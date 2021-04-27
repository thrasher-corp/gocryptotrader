package engine

import (
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

const (
	CandleDataType = iota
	TradeDataType
)

type DataHistoryManager struct {
	exchangeManager           iExchangeManager
	DatabaseConnectionManager *DatabaseConnectionManager
	started                   int32
	shutdown                  chan struct{}
	interval                  *time.Ticker
	jobs                      []DataHistoryJobConfig
}

type DataHistoryJobConfig struct {
	Nickname         string         `json:"nickname"`
	Exchange         string         `json:"exchange"`
	Asset            asset.Item     `json:"asset"`
	Pair             currency.Pair  `json:"pair"`
	StartDate        time.Time      `json:"start-date"`
	EndDate          time.Time      `json:"end-date"`
	IsRolling        bool           `json:"is-rolling"`
	Interval         kline.Interval `json:"interval"`
	RequestSizeLimit uint32         `json:"request-size-limit"`
	DataType         int            `json:"data-type"`
	MaxRetryAttempts int            `json:"retry-attempts"`
	failures         []dataHistoryFailure
	continueFromData time.Time
	ranges           kline.IntervalRangeHolder
	running          bool
}

type dataHistoryFailure struct {
	reason string
}

func SetupDataHistoryManager(em iExchangeManager, dcm *DatabaseConnectionManager, processInterval time.Duration, jobs []DataHistoryJobConfig) (*DataHistoryManager, error) {
	if em == nil {

	}
	if dcm == nil {

	}
	return &DataHistoryManager{
		exchangeManager:           em,
		DatabaseConnectionManager: dcm,
		shutdown:                  make(chan struct{}),
		interval:                  time.NewTicker(processInterval),
		jobs:                      jobs,
	}, nil
}

func (m *DataHistoryManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return atomic.LoadInt32(&m.started) == 1
}

func (m *DataHistoryManager) Start() error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !atomic.CompareAndSwapInt32(&m.started, 0, 1) {
		return ErrSubSystemAlreadyStarted
	}
	m.shutdown = make(chan struct{})

	go func() {
		err := m.run()
		if err != nil {
			log.Error(log.DataHistory, err)
		}
	}()

	return nil
}

func (m *DataHistoryManager) PrepareJobs() []DataHistoryJobConfig {
	var validJobs []DataHistoryJobConfig
	for i := range m.jobs {
		exch := m.exchangeManager.GetExchangeByName(m.jobs[i].Exchange)
		if exch == nil {
			log.Errorf(log.DataHistory, "exchange not loaded, cannot process job")
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

func (m *DataHistoryManager) run() error {
	for {
		select {
		case <-m.shutdown:
			return nil
		case <-m.interval.C:
			var jobsToRemove []DataHistoryJobConfig
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
			ranges:
				for j := range m.jobs[i].ranges.Ranges {
					for x := range m.jobs[i].ranges.Ranges[j].Intervals {
						if m.jobs[i].ranges.Ranges[j].Intervals[x].HasData {
							continue ranges
						}
						switch m.jobs[i].DataType {
						case CandleDataType:
							niceCans, err := exch.GetHistoricCandles(m.jobs[i].Pair, m.jobs[i].Asset, m.jobs[i].ranges.Ranges[j].Start.Time, m.jobs[i].ranges.Ranges[j].End.Time, m.jobs[i].Interval)
							if err != nil {
								fail := dataHistoryFailure{reason: "could not get candles: " + err.Error()}
								m.jobs[i].failures = append(m.jobs[i].failures, fail)
								continue
							}
							err = m.jobs[i].ranges.VerifyResultsHaveData(niceCans.Candles)
							if err != nil {
								fail := dataHistoryFailure{reason: "could not verify results: " + err.Error()}
								m.jobs[i].failures = append(m.jobs[i].failures, fail)
								continue
							}
							// save the data
						case TradeDataType:
							trades, err := exch.GetHistoricTrades(m.jobs[i].Pair, m.jobs[i].Asset, m.jobs[i].ranges.Ranges[j].Start.Time, m.jobs[i].ranges.Ranges[j].End.Time)
							if err != nil {
								fail := dataHistoryFailure{reason: "could not get trades: " + err.Error()}
								m.jobs[i].failures = append(m.jobs[i].failures, fail)
								continue
							}
							bigCans, err := trade.ConvertTradesToCandles(m.jobs[i].Interval, trades...)
							if err != nil {
								fail := dataHistoryFailure{reason: "could not get convert candles to trades: " + err.Error()}
								m.jobs[i].failures = append(m.jobs[i].failures, fail)
								continue
							}
							err = m.jobs[i].ranges.VerifyResultsHaveData(bigCans.Candles)
							if err != nil {
								fail := dataHistoryFailure{reason: "could not verify results: " + err.Error()}
								m.jobs[i].failures = append(m.jobs[i].failures, fail)
								continue
							}
							// save the data
						}
					}
				}

				// if it doesn't have the data, increase failure rate and say why (eg exchange doesn't provide data)
			}

		}
	}
}
