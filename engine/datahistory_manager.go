package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"uuid"

	"github.com/thrasher-corp/gocryptotrader/common"
	gctmath "github.com/thrasher-corp/gocryptotrader/common/math"
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
	if cfg.MaxJobsPerCycle <= 0 {
		cfg.MaxJobsPerCycle = defaultDataHistoryMaxJobsPerCycle
	}
	if cfg.MaxResultInsertions <= 0 {
		cfg.MaxResultInsertions = defaultMaxResultInsertions
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
		maxResultInsertions:        cfg.MaxResultInsertions,
		tradeLoader:                trade.GetTradesInRange,
		tradeSaver:                 trade.SaveTradesToDatabase,
		candleLoader:               kline.LoadFromDatabase,
		candleSaver:                kline.StoreInDatabase,
	}, nil
}

// Start runs the subsystem
func (m *DataHistoryManager) Start(ctx context.Context) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if m.databaseConnectionInstance == nil {
		return errNilDatabaseConnectionManager
	}
	if !m.started.CompareAndSwap(false, true) {
		return ErrSubSystemAlreadyStarted
	}
	m.shutdown = make(chan struct{})
	m.run(ctx)
	log.Debugf(log.DataHistory, "Data history manager %v", MsgSubSystemStarted)

	return nil
}

// IsRunning checks whether the subsystem is running
func (m *DataHistoryManager) IsRunning() bool {
	if m == nil {
		return false
	}
	return m.started.Load()
}

// Stop stops the subsystem
func (m *DataHistoryManager) Stop() error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.started.CompareAndSwap(true, false) {
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
	if !m.started.Load() {
		return nil, ErrSubSystemNotStarted
	}
	dbJobs, err := m.jobDB.GetAllIncompleteJobsAndResults()
	if err != nil {
		return nil, err
	}

	response := make([]*DataHistoryJob, 0, len(dbJobs))
	for i := range dbJobs {
		dbJob, err := m.convertDBModelToJob(&dbJobs[i])
		if err != nil {
			return nil, err
		}
		err = m.validateJob(dbJob)
		if err != nil {
			log.Errorln(log.DataHistory, err)
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
	if !m.started.Load() {
		return nil, ErrSubSystemNotStarted
	}
	jobs, err := m.retrieveJobs()
	if err != nil {
		defer func() {
			err = m.Stop()
			if err != nil {
				log.Errorln(log.DataHistory, err)
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

// rangeInterval returns the interval the job's date ranges are built with
func (j *DataHistoryJob) rangeInterval() kline.Interval {
	if j.DataType == dataHistoryConvertCandlesDataType {
		return j.ConversionInterval
	}
	return j.Interval
}

// dropStaleResults removes the results under each range's start that were not recorded for it, and its Complete results
// while an interval lacks data. Earlier conversion jobs recorded results for shorter batches, and Complete without saving
// converted candles.
func (j *DataHistoryJob) dropStaleResults() {
	for x := range j.rangeHolder.Ranges {
		start, end := j.rangeHolder.Ranges[x].Start.Time, j.rangeHolder.Ranges[x].End.Time
		key := start.Unix()
		results, ok := j.Results[key]
		if !ok {
			continue
		}
		missing := slices.ContainsFunc(j.rangeHolder.Ranges[x].Intervals, func(i kline.IntervalData) bool { return !i.HasData })
		results = slices.DeleteFunc(results, func(r DataHistoryJobResult) bool {
			return !r.coversRange(start, end) || missing && r.Status == dataHistoryStatusComplete
		})
		if len(results) == 0 {
			delete(j.Results, key)
			continue
		}
		j.Results[key] = results
	}
}

// candleRuns returns the first candle within each interval of h, in runs of consecutive intervals. An overwriting save
// deletes every stored candle between its first and last, so saving each run on its own leaves the candles of
// intervals without a converted candle, and outside h, untouched.
func candleRuns(h *kline.IntervalRangeHolder, candles []kline.Candle) [][]kline.Candle {
	sorted := slices.Clone(candles)
	slices.SortStableFunc(sorted, func(a, b kline.Candle) int { return a.Time.Compare(b.Time) })
	var runs [][]kline.Candle
	var run []kline.Candle
	var next int
	for x := range h.Ranges {
		for _, interval := range h.Ranges[x].Intervals {
			for next < len(sorted) && sorted[next].Time.Before(interval.Start.Time) {
				next++
			}
			if next < len(sorted) && sorted[next].Time.Before(interval.End.Time) {
				run = append(run, sorted[next])
				continue
			}
			if len(run) > 0 {
				runs = append(runs, run)
				run = nil
			}
		}
	}
	if len(run) > 0 {
		runs = append(runs, run)
	}
	return runs
}

// padSourceCandles returns one candle for each interval from start until end, holding the first of the candles given
// with data that starts within it, at the interval's start, or an empty candle, and which intervals had such a candle.
// A stored candle without data is the padding a candle job saves for an interval the exchange did not supply, so it
// leaves its interval missing. Conversion then groups the candles by the output interval they start in, rather than
// from the first candle it is given.
func padSourceCandles(candles []kline.Candle, start, end time.Time, interval kline.Interval) (padded []kline.Candle, present []bool) {
	if interval <= 0 || !start.Before(end) {
		return nil, nil
	}
	count := int(end.Sub(start) / interval.Duration())
	padded = make([]kline.Candle, count)
	present = make([]bool, count)
	for i := range padded {
		padded[i].Time = start.Add(time.Duration(i) * interval.Duration())
	}
	for i := range candles {
		if candles[i].Time.Before(start) || !candles[i].Time.Before(end) || !candles[i].HasData() {
			continue
		}
		slot := int(candles[i].Time.Sub(start) / interval.Duration())
		if slot >= count || present[slot] {
			continue
		}
		c := candles[i]
		c.Time = padded[slot].Time
		padded[slot] = c
		present[slot] = true
	}
	return padded, present
}

// intervalsWithoutData returns how many intervals of h have no data, and how many intervals h has
func intervalsWithoutData(h *kline.IntervalRangeHolder) (missing, total int) {
	for x := range h.Ranges {
		for _, interval := range h.Ranges[x].Intervals {
			total++
			if !interval.HasData {
				missing++
			}
		}
	}
	return missing, total
}

// rangeResult reports whether a range's results decide it and, if so, whether it completed rather than finishing with
// issues found, whatever order the results were loaded in
func rangeResult(results []DataHistoryJobResult) (decided, complete bool) {
	if slices.ContainsFunc(results, func(r DataHistoryJobResult) bool { return r.Status == dataHistoryStatusComplete }) {
		return true, true
	}
	if slices.ContainsFunc(results, func(r DataHistoryJobResult) bool { return r.Status == dataHistoryIntervalIssuesFound }) {
		return true, false
	}
	return false, false
}

// coversRange reports whether the result was recorded for the range from start until end
func (r *DataHistoryJobResult) coversRange(start, end time.Time) bool {
	return r.IntervalStartDate.Equal(start) && r.IntervalEndDate.Equal(end)
}

// rangeResults returns the results recorded for the range from start until end. An edit to a job's dates, request size
// or interval leaves its earlier results under the start seconds they share with its new ranges.
func (j *DataHistoryJob) rangeResults(start, end time.Time) []DataHistoryJobResult {
	return slices.DeleteFunc(slices.Clone(j.Results[start.Unix()]), func(r DataHistoryJobResult) bool { return !r.coversRange(start, end) })
}

// markRangeIssuesFound marks the results recorded for the range from start until end as having issues found
func (j *DataHistoryJob) markRangeIssuesFound(start, end time.Time) {
	results := j.Results[start.Unix()]
	for i := range results {
		if results[i].coversRange(start, end) {
			results[i].Status = dataHistoryIntervalIssuesFound
		}
	}
}

// setHasDataByCandleTime sets HasData for each interval of the ranges given that a candle starts, from whether a candle
// starting there has data, and leaves the intervals no candle starts as they were, as SetHasDataFromCandles, in kline,
// leaves those after its last candle. That function matches candles in order from a holder's first interval, so it
// rejects both a gap before later candles, which the candles a conversion job saves can have, and the candles fetched
// for any range after a job's first.
func setHasDataByCandleTime(ranges []kline.IntervalRange, candles []kline.Candle) {
	starts := make(map[int64]bool, len(candles))
	for i := range candles {
		start := candles[i].Time.UnixNano()
		starts[start] = starts[start] || candles[i].HasData()
	}
	for x := range ranges {
		for y := range ranges[x].Intervals {
			if hasData, ok := starts[ranges[x].Intervals[y].Start.Time.UnixNano()]; ok {
				ranges[x].Intervals[y].HasData = hasData
			}
		}
	}
}

func (m *DataHistoryManager) compareJobsToData(jobs ...*DataHistoryJob) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.started.Load() {
		return ErrSubSystemNotStarted
	}
	var err error
	for i := range jobs {
		jobs[i].rangeHolder, err = kline.CalculateCandleDateRanges(jobs[i].StartDate, jobs[i].EndDate, jobs[i].rangeInterval(), jobs[i].RequestSizeLimit)
		if err != nil {
			return err
		}
		var candles *kline.Item
		switch jobs[i].DataType {
		case dataHistoryCandleDataType,
			dataHistoryCandleValidationDataType,
			dataHistoryCandleValidationSecondarySourceType,
			dataHistoryConvertTradesDataType:
			candles, err = m.candleLoader(jobs[i].Exchange, jobs[i].Pair, jobs[i].Asset, jobs[i].Interval, jobs[i].StartDate, jobs[i].EndDate)
			if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
				return fmt.Errorf("%s could not load candle data: %w", jobs[i].Nickname, err)
			}
			if candles != nil {
				setHasDataByCandleTime(jobs[i].rangeHolder.Ranges, candles.Candles)
			}
		case dataHistoryTradeDataType:
			for x := range jobs[i].rangeHolder.Ranges {
				r := &jobs[i].rangeHolder.Ranges[x]
				if _, complete := rangeResult(jobs[i].rangeResults(r.Start.Time, r.End.Time)); !complete {
					continue
				}
				for z := range r.Intervals {
					r.Intervals[z].HasData = true
				}
			}
		case dataHistoryConvertCandlesDataType:
			candles, err = m.candleLoader(jobs[i].Exchange, jobs[i].Pair, jobs[i].Asset, jobs[i].ConversionInterval, jobs[i].rangeHolder.Start.Time, jobs[i].rangeHolder.End.Time)
			if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
				return fmt.Errorf("%s could not load candle data: %w", jobs[i].Nickname, err)
			}
			if candles != nil {
				setHasDataByCandleTime(jobs[i].rangeHolder.Ranges, candles.Candles)
			}
			jobs[i].dropStaleResults()
		default:
			return fmt.Errorf("%s %w %s", jobs[i].Nickname, errUnknownDataType, jobs[i].DataType)
		}
	}
	return nil
}

func (m *DataHistoryManager) run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-m.shutdown:
				return
			case <-m.interval.C:
				if m.databaseConnectionInstance != nil && m.databaseConnectionInstance.IsConnected() {
					go func() {
						if err := m.runJobs(ctx); err != nil {
							log.Errorln(log.DataHistory, err)
						}
					}()
				}
			}
		}
	}()
}

func (m *DataHistoryManager) runJobs(ctx context.Context) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.started.Load() {
		return ErrSubSystemNotStarted
	}

	if !m.processing.CompareAndSwap(false, true) {
		return fmt.Errorf("cannot process jobs, %w", ErrSubSystemAlreadyStarted)
	}
	defer m.processing.Store(false)

	validJobs, err := m.PrepareJobs()
	if err != nil {
		return err
	}
	if len(validJobs) == 0 {
		if m.verbose {
			log.Infof(log.DataHistory, "no data history jobs to process")
		}
		return nil
	}

	log.Infof(log.DataHistory, "processing data history jobs")
	for i := 0; (i < int(m.maxJobsPerCycle) || m.maxJobsPerCycle == -1) && i < len(validJobs); i++ {
		err := m.runJob(ctx, validJobs[i])
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
				if m.verbose {
					log.Debugf(log.DataHistory, "stopping data history run due to cancelled context")
				}
				return nil
			}
			log.Errorln(log.DataHistory, err)
		}
		if m.verbose {
			log.Debugf(log.DataHistory, "completed run of data history job %v", validJobs[i].Nickname)
		}
	}
	log.Infof(log.DataHistory, "completed run of data history jobs")

	return nil
}

// runJob processes an active job, retrieves candle or trade data
// for a given date range and saves all results to the database
func (m *DataHistoryManager) runJob(ctx context.Context, job *DataHistoryJob) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.started.Load() {
		return ErrSubSystemNotStarted
	}
	if job == nil {
		return errNilJob
	}
	if job.Status != dataHistoryStatusActive {
		return fmt.Errorf("job %s %w", job.Nickname, errJobInvalid)
	}
	if job.rangeHolder == nil || len(job.rangeHolder.Ranges) == 0 {
		return fmt.Errorf("%s %w invalid start/end range %s-%s",
			job.Nickname,
			errJobInvalid,
			job.StartDate.Format(common.SimpleTimeFormatWithTimezone),
			job.EndDate.Format(common.SimpleTimeFormatWithTimezone),
		)
	}
	exchangeName := job.Exchange
	if job.DataType == dataHistoryCandleValidationSecondarySourceType {
		exchangeName = job.SecondaryExchangeSource
	}
	exch, err := m.exchangeManager.GetExchangeByName(exchangeName)
	if err != nil {
		return fmt.Errorf("%w, cannot process job %s for %s %s",
			err,
			job.Nickname,
			job.Asset,
			job.Pair)
	}

	if job.DataType == dataHistoryCandleValidationDataType ||
		job.DataType == dataHistoryCandleValidationSecondarySourceType {
		err = m.runValidationJob(ctx, job, exch)
		if err != nil {
			return err
		}
	} else {
		err = m.runDataJob(ctx, job, exch)
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

// runDataJob will fetch data from an API endpoint or convert existing database data
// into a new candle type
func (m *DataHistoryManager) runDataJob(ctx context.Context, job *DataHistoryJob, exch exchange.IBotExchange) error {
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
	var intervalsProcessed uint64
	var err error
	var result *DataHistoryJobResult
ranges:
	for i := range job.rangeHolder.Ranges {
		rangeStart, rangeEnd := job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time
		skipProcessing := true
		for j := range job.rangeHolder.Ranges[i].Intervals {
			if !job.rangeHolder.Ranges[i].Intervals[j].HasData {
				skipProcessing = false
				break
			}
		}
		if skipProcessing {
			key := rangeStart.Unix()
			if _, complete := rangeResult(job.rangeResults(rangeStart, rangeEnd)); !complete && !job.OverwriteExistingData {
				// we have determined that data is there, however it is not reflected in
				// this specific job's results, which is required for a job to be complete
				job.Results[key] = append(job.Results[key], DataHistoryJobResult{
					ID:                uuid.NewV4(),
					JobID:             job.ID,
					IntervalStartDate: job.rangeHolder.Ranges[i].Start.Time,
					IntervalEndDate:   job.rangeHolder.Ranges[i].End.Time,
					Status:            dataHistoryStatusComplete,
					Date:              time.Now(),
				})
			}
			if !job.OverwriteExistingData {
				continue
			}
		}
		if intervalsProcessed >= job.RunBatchLimit {
			continue
		}

		if m.verbose {
			log.Debugf(log.DataHistory, "running data history job %v start: %s end: %s interval: %s datatype: %s",
				job.Nickname,
				job.rangeHolder.Ranges[i].Start.Time,
				job.rangeHolder.Ranges[i].End.Time,
				job.Interval,
				job.DataType)
		}

		var failures uint64
		hasDataInRange := false
		resultLookup := job.rangeResults(rangeStart, rangeEnd)
		if len(resultLookup) > 0 {
			for x := range resultLookup {
				switch resultLookup[x].Status {
				case dataHistoryIntervalIssuesFound:
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
				job.markRangeIssuesFound(rangeStart, rangeEnd)
				continue
			}
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
			result, err = m.processCandleData(ctx, job, exch, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time, int64(i))
		case dataHistoryTradeDataType:
			result, err = m.processTradeData(ctx, job, exch, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time, int64(i))
		case dataHistoryConvertTradesDataType:
			result, err = m.convertTradesToCandles(job, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time)
		case dataHistoryConvertCandlesDataType:
			result, err = m.convertCandleData(job, job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time)
		default:
			return errUnknownDataType
		}
		if err != nil {
			return err
		}
		if result == nil {
			return errNilResult
		}

		lookup := job.Results[result.IntervalStartDate.Unix()]
		lookup = append(lookup, *result)
		job.Results[result.IntervalStartDate.Unix()] = lookup
	}
	completed := true
	allResultsSuccessful := true
	allResultsFailed := true
	for i := range job.rangeHolder.Ranges {
		decided, complete := rangeResult(job.rangeResults(job.rangeHolder.Ranges[i].Start.Time, job.rangeHolder.Ranges[i].End.Time))
		if !decided {
			completed = false
			break
		}
		if complete {
			allResultsFailed = false
		} else {
			allResultsSuccessful = false
		}
	}
	if completed {
		err := m.completeJob(job, allResultsSuccessful, allResultsFailed)
		if err != nil {
			return err
		}
	}
	return nil
}

// validationIntervalLength returns the time one validation request spans, rejecting a length a time.Duration cannot hold
func validationIntervalLength(interval kline.Interval, requestSizeLimit uint64) (time.Duration, error) {
	if interval <= 0 || requestSizeLimit == 0 || requestSizeLimit > math.MaxInt64/uint64(interval) {
		return 0, fmt.Errorf("%w: interval %s with request size limit %d", kline.ErrInvalidInterval, interval, requestSizeLimit)
	}
	return interval.Duration() * time.Duration(requestSizeLimit), nil //nolint:gosec // the limit is checked to fit above
}

// runValidationJob verifies existing database candle data against
// the original API's data, or a secondary exchange source
func (m *DataHistoryManager) runValidationJob(ctx context.Context, job *DataHistoryJob, exch exchange.IBotExchange) error {
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
	var intervalsProcessed uint64
	var jobIntervals, intervalsToCheck []time.Time
	intervalLength, err := validationIntervalLength(job.Interval, job.RequestSizeLimit)
	if err != nil {
		return fmt.Errorf("job %s %w", job.Nickname, err)
	}
	rangeEnd := func(start time.Time) time.Time {
		if end := start.Add(intervalLength); end.Before(job.EndDate) {
			return end
		}
		return job.EndDate
	}
	for i := job.StartDate; i.Before(job.EndDate); i = i.Add(intervalLength) {
		jobIntervals = append(jobIntervals, i)
	}
	for _, interval := range jobIntervals {
		results := job.rangeResults(interval, rangeEnd(interval))
		if slices.ContainsFunc(results, func(r DataHistoryJobResult) bool { return r.Status == dataHistoryStatusComplete }) {
			continue
		}
		if uint64(len(results)) >= job.MaxRetryAttempts {
			job.markRangeIssuesFound(interval, rangeEnd(interval))
			continue
		}
		intervalsToCheck = append(intervalsToCheck, interval)
	}

	for i := range intervalsToCheck {
		if intervalsProcessed >= job.RunBatchLimit {
			break
		}
		if err := common.StartEndTimeCheck(intervalsToCheck[i], job.EndDate); err != nil {
			break
		}
		requestEnd := rangeEnd(intervalsToCheck[i])
		if m.verbose {
			log.Debugf(log.DataHistory, "running data history job %v start: %s end: %s interval: %s datatype: %s",
				job.Nickname,
				intervalsToCheck[i],
				requestEnd,
				job.Interval,
				job.DataType)
		}
		intervalsProcessed++
		result, err := m.validateCandles(ctx, job, exch, intervalsToCheck[i], requestEnd)
		if err != nil {
			return err
		}
		lookup := job.Results[result.IntervalStartDate.Unix()]
		lookup = append(lookup, *result)
		job.Results[result.IntervalStartDate.Unix()] = lookup
	}

	completed := true
	allResultsSuccessful := true
	allResultsFailed := true
	for i := range jobIntervals {
		decided, complete := rangeResult(job.rangeResults(jobIntervals[i], rangeEnd(jobIntervals[i])))
		if !decided {
			completed = false
			break
		}
		if complete {
			allResultsFailed = false
		} else {
			allResultsSuccessful = false
		}
	}
	if completed {
		err := m.completeJob(job, allResultsSuccessful, allResultsFailed)
		if err != nil {
			return err
		}
	}

	return nil
}

// completeJob will set the job's overall status and
// set any jobs' status where the current job is a prerequisite to 'active'
func (m *DataHistoryManager) completeJob(job *DataHistoryJob, allResultsSuccessful, allResultsFailed bool) error {
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
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
		job.Status = dataHistoryIntervalIssuesFound
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
			log.Infof(log.DataHistory, "setting the following jobs to active: %s", strings.Join(newJobNames, ", "))
			err = m.jobDB.Upsert(newJobs...)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *DataHistoryManager) saveCandlesInBatches(job *DataHistoryJob, candles *kline.Item, r *DataHistoryJobResult) error {
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
	if job == nil {
		return errNilJob
	}
	if candles == nil {
		return errNilCandles
	}
	if r == nil {
		return errNilResult
	}
	if m.maxResultInsertions <= 0 {
		m.maxResultInsertions = defaultMaxResultInsertions
	}
	var err error
	for i := 0; i < len(candles.Candles); i += int(m.maxResultInsertions) {
		newCandle := *candles
		if i+int(m.maxResultInsertions) > len(newCandle.Candles) {
			if m.verbose {
				log.Debugf(log.DataHistory, "Saving %v candles. Range %v-%v/%v", len(newCandle.Candles[i:]), i, len(candles.Candles), len(candles.Candles))
			}
			newCandle.Candles = newCandle.Candles[i:]
			_, err = m.candleSaver(&newCandle, job.OverwriteExistingData)
			if err != nil {
				r.Result += "could not save results: " + err.Error() + ". "
				r.Status = dataHistoryStatusFailed
			}
			break
		}
		if m.verbose {
			log.Debugf(log.DataHistory, "Saving %v candles. Range %v-%v/%v", m.maxResultInsertions, i, i+int(m.maxResultInsertions), len(candles.Candles))
		}
		newCandle.Candles = newCandle.Candles[i : i+int(m.maxResultInsertions)]
		_, err = m.candleSaver(&newCandle, job.OverwriteExistingData)
		if err != nil {
			r.Result += "could not save results: " + err.Error() + ". "
			r.Status = dataHistoryStatusFailed
		}
	}
	return nil
}

func (m *DataHistoryManager) processCandleData(ctx context.Context, job *DataHistoryJob, exch exchange.IBotExchange, startRange, endRange time.Time, intervalIndex int64) (*DataHistoryJobResult, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	if job == nil {
		return nil, errNilJob
	}
	if exch == nil {
		return nil, ErrExchangeNotFound
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}

	id := uuid.NewV4()
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	candles, err := exch.GetHistoricCandlesExtended(ctx,
		job.Pair,
		job.Asset,
		job.Interval,
		startRange,
		endRange)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			return nil, err
		}
		r.Result += "could not get candles: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	setHasDataByCandleTime(job.rangeHolder.Ranges[intervalIndex:intervalIndex+1], candles.Candles)
	for i := range job.rangeHolder.Ranges[intervalIndex].Intervals {
		if !job.rangeHolder.Ranges[intervalIndex].Intervals[i].HasData {
			r.Status = dataHistoryStatusFailed
			r.Result += fmt.Sprintf("missing data from %v - %v. ",
				job.rangeHolder.Ranges[intervalIndex].Intervals[i].Start.Time.Format(common.SimpleTimeFormatWithTimezone),
				job.rangeHolder.Ranges[intervalIndex].Intervals[i].End.Time.Format(common.SimpleTimeFormatWithTimezone))
		}
	}
	candles.SourceJobID = job.ID
	err = m.saveCandlesInBatches(job, candles, r)
	return r, err
}

func (m *DataHistoryManager) processTradeData(ctx context.Context, job *DataHistoryJob, exch exchange.IBotExchange, startRange, endRange time.Time, intervalIndex int64) (*DataHistoryJobResult, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	if job == nil {
		return nil, errNilJob
	}
	if exch == nil {
		return nil, ErrExchangeNotFound
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id := uuid.NewV4()
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	trades, err := exch.GetHistoricTrades(ctx,
		job.Pair,
		job.Asset,
		startRange,
		endRange)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			return nil, err
		}
		r.Result += "could not get trades: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	// A wrapper can return trades outside the dates it is given, which belong to other ranges or to no range of the job
	trades = slices.DeleteFunc(trades, func(t trade.Data) bool {
		return t.Timestamp.Before(startRange) || !t.Timestamp.Before(endRange)
	})
	candles, err := trade.ConvertTradesToCandles(job.Interval, trades...)
	if err != nil {
		r.Result += "could not convert candles to trades: " + err.Error() + ". "
		r.Status = dataHistoryStatusFailed
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	setHasDataByCandleTime(job.rangeHolder.Ranges[intervalIndex:intervalIndex+1], candles.Candles)
	for i := range job.rangeHolder.Ranges[intervalIndex].Intervals {
		if !job.rangeHolder.Ranges[intervalIndex].Intervals[i].HasData {
			r.Status = dataHistoryStatusFailed
			r.Result += fmt.Sprintf("missing data from %v - %v. ",
				job.rangeHolder.Ranges[intervalIndex].Intervals[i].Start.Time.Format(common.SimpleTimeFormatWithTimezone),
				job.rangeHolder.Ranges[intervalIndex].Intervals[i].End.Time.Format(common.SimpleTimeFormatWithTimezone))
		}
	}
	for i := 0; i < len(trades); i += int(m.maxResultInsertions) {
		if i+int(m.maxResultInsertions) > len(trades) {
			if m.verbose {
				log.Debugf(log.DataHistory, "Saving %v trades. Range %v-%v/%v", len(trades[i:]), i, len(trades), len(trades))
			}
			err = m.tradeSaver(trades[i:]...)
			if err != nil {
				r.Result += "could not save results: " + err.Error() + ". "
				r.Status = dataHistoryStatusFailed
			}
			break
		}
		if m.verbose {
			log.Debugf(log.DataHistory, "Saving %v trades. Range %v-%v/%v", m.maxResultInsertions, i, i+int(m.maxResultInsertions), len(trades))
		}
		err = m.tradeSaver(trades[i : i+int(m.maxResultInsertions)]...)
		if err != nil {
			r.Result += "could not save results: " + err.Error() + ". "
			r.Status = dataHistoryStatusFailed
		}
	}
	return r, nil
}

func (m *DataHistoryManager) convertTradesToCandles(job *DataHistoryJob, startRange, endRange time.Time) (*DataHistoryJobResult, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	if job == nil {
		return nil, errNilJob
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id := uuid.NewV4()
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
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	candles, err := trade.ConvertTradesToCandles(job.Interval, trades...)
	if err != nil {
		r.Result = "could not convert trades in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	candles.SourceJobID = job.ID
	err = m.saveCandlesInBatches(job, candles, r)
	return r, err
}

func (m *DataHistoryManager) convertCandleData(job *DataHistoryJob, startRange, endRange time.Time) (*DataHistoryJobResult, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	if job == nil {
		return nil, errNilJob
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id := uuid.NewV4()
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}
	intervals, err := kline.CalculateCandleDateRanges(startRange, endRange, job.ConversionInterval, 0)
	if err != nil {
		r.Result = "could not calculate converted candle intervals in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	candles, err := m.candleLoader(job.Exchange, job.Pair, job.Asset, job.Interval, intervals.Start.Time, intervals.End.Time)
	if err != nil {
		r.Result = "could not get candles in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	var present []bool
	if candles != nil {
		candles.Candles, present = padSourceCandles(candles.Candles, intervals.Start.Time, intervals.End.Time, job.Interval)
	}
	newCandles, err := candles.ConvertToNewInterval(job.ConversionInterval)
	if err != nil {
		r.Result = "could not convert candles in range: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	// A converted candle needs every source candle of its interval, so that one built from part of them is not kept
	perCandle := int(job.ConversionInterval / job.Interval)
	for i := range newCandles.Candles {
		if slices.Contains(present[i*perCandle:min((i+1)*perCandle, len(present))], false) {
			newCandles.Candles[i] = kline.Candle{}
		}
	}
	newCandles.SourceJobID = job.ID
	for _, run := range candleRuns(intervals, newCandles.Candles) {
		converted := *newCandles
		converted.Candles = run
		if err := m.saveCandlesInBatches(job, &converted, r); err != nil {
			return nil, err
		}
	}
	if r.Status != dataHistoryStatusComplete {
		return r, nil
	}
	// The batch is judged on what the store holds, as the next preparation will judge it
	saved, err := m.candleLoader(job.Exchange, job.Pair, job.Asset, job.ConversionInterval, intervals.Start.Time, intervals.End.Time)
	if err != nil && !errors.Is(err, candle.ErrNoCandleDataFound) {
		return nil, fmt.Errorf("%s could not load saved candles: %w", job.Nickname, err)
	}
	if saved != nil {
		setHasDataByCandleTime(intervals.Ranges, saved.Candles)
	}
	if missing, total := intervalsWithoutData(intervals); missing > 0 {
		r.Result = fmt.Sprintf("missing %d of %d converted candles from %v - %v. ", missing, total,
			startRange.Format(common.SimpleTimeFormatWithTimezone), endRange.Format(common.SimpleTimeFormatWithTimezone))
		r.Status = dataHistoryStatusFailed
	}
	return r, nil
}

func (m *DataHistoryManager) validateCandles(ctx context.Context, job *DataHistoryJob, exch exchange.IBotExchange, startRange, endRange time.Time) (*DataHistoryJobResult, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	if job == nil {
		return nil, errNilJob
	}
	if exch == nil {
		return nil, ErrExchangeNotFound
	}
	if err := common.StartEndTimeCheck(startRange, endRange); err != nil {
		return nil, err
	}
	id := uuid.NewV4()
	r := &DataHistoryJobResult{
		ID:                id,
		JobID:             job.ID,
		IntervalStartDate: startRange,
		IntervalEndDate:   endRange,
		Status:            dataHistoryStatusComplete,
		Date:              time.Now(),
	}

	apiCandles, err := exch.GetHistoricCandlesExtended(ctx,
		job.Pair,
		job.Asset,
		job.Interval,
		startRange,
		endRange)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			return nil, err
		}
		r.Result = "could not get API candles: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	apiCandles.ValidationJobID = job.ID
	dbCandles, err := m.candleLoader(job.Exchange, job.Pair, job.Asset, job.Interval, startRange, endRange)
	if err != nil {
		r.Result = "could not get database candles: " + err.Error()
		r.Status = dataHistoryStatusFailed
		return r, nil //nolint:nilerr // error is returned in the job result
	}
	if len(dbCandles.Candles) == 0 {
		r.Result = fmt.Sprintf("missing database candles for period %v-%v", startRange, endRange)
		r.Status = dataHistoryIntervalIssuesFound
		return r, nil
	}

	if len(dbCandles.Candles) > 0 && len(apiCandles.Candles) == 0 {
		r.Result = fmt.Sprintf("no matching API data for database candles for period %v-%v", startRange, endRange)
		r.Status = dataHistoryStatusFailed
		return r, nil
	}
	if len(dbCandles.Candles) != len(apiCandles.Candles) && m.verbose {
		log.Warnf(log.DataHistory, "mismatched candle length for period %v-%v. DB: %v, API: %v", startRange, endRange, len(dbCandles.Candles), len(apiCandles.Candles))
	}

	dbCandleMap := make(map[int64]kline.Candle)
	for i := range dbCandles.Candles {
		dbCandleMap[dbCandles.Candles[i].Time.Unix()] = dbCandles.Candles[i]
	}
	var validationIssues []string
	multiplier := int64(1)
	for range job.DecimalPlaceComparison {
		multiplier *= 10
	}
	for i := range apiCandles.Candles {
		can, ok := dbCandleMap[apiCandles.Candles[i].Time.Unix()]
		if !ok {
			validationIssues = append(validationIssues, fmt.Sprintf("issues found at %v missing candle data in database", apiCandles.Candles[i].Time.Format(common.SimpleTimeFormatWithTimezone)))
			r.Status = dataHistoryIntervalIssuesFound
			continue
		}
		var candleIssues []string
		var candleModified bool

		issue, modified := m.CheckCandleIssue(job, multiplier, apiCandles.Candles[i].Open, can.Open, "Open")
		if issue != "" {
			candleIssues = append(candleIssues, issue)
		}
		if modified {
			candleModified = true
		}
		issue, modified = m.CheckCandleIssue(job, multiplier, apiCandles.Candles[i].High, can.High, "High")
		if issue != "" {
			candleIssues = append(candleIssues, issue)
		}
		if modified {
			candleModified = true
		}
		issue, modified = m.CheckCandleIssue(job, multiplier, apiCandles.Candles[i].Low, can.Low, "Low")
		if issue != "" {
			candleIssues = append(candleIssues, issue)
		}
		if modified {
			candleModified = true
		}
		issue, modified = m.CheckCandleIssue(job, multiplier, apiCandles.Candles[i].Close, can.Close, "Close")
		if issue != "" {
			candleIssues = append(candleIssues, issue)
		}
		if modified {
			candleModified = true
		}
		if job.SecondaryExchangeSource == "" {
			issue, modified = m.CheckCandleIssue(job, multiplier, apiCandles.Candles[i].Volume, can.Volume, "Volume")
			if issue != "" {
				candleIssues = append(candleIssues, issue)
			}
			if modified {
				candleModified = true
			}
		}
		if candleModified {
			candleIssues = append(candleIssues, "replacing mismatched database candle data with API data")
		}
		// we update candles regardless to link candle to validation job
		apiCandles.Candles[i] = can

		if len(candleIssues) > 0 {
			candleIssues = append([]string{fmt.Sprintf("issues found at %v", can.Time.Format(time.DateTime))}, candleIssues...)
			validationIssues = append(validationIssues, candleIssues...)
			r.Status = dataHistoryStatusFailed
			apiCandles.Candles[i].ValidationIssues = strings.Join(candleIssues, ", ")
		}
	}
	if len(validationIssues) > 0 {
		r.Result = strings.Join(validationIssues, " -- ")
	}
	err = m.saveCandlesInBatches(job, apiCandles, r)
	return r, err
}

// CheckCandleIssue verifies that stored data matches API data
// a job can specify a level of rounding along with a tolerance percentage
// a job can also replace data with API data if the database data exceeds the tolerance
func (m *DataHistoryManager) CheckCandleIssue(job *DataHistoryJob, multiplier int64, apiData, dbData float64, candleField string) (issue string, replace bool) {
	if m == nil {
		return ErrNilSubsystem.Error(), false
	}
	if !m.started.Load() {
		return ErrSubSystemNotStarted.Error(), false
	}
	if job == nil {
		return errNilJob.Error(), false
	}

	floatiplier := float64(multiplier)
	if floatiplier > 0 {
		apiData = math.Round(apiData*floatiplier) / floatiplier
		dbData = math.Round(dbData*floatiplier) / floatiplier
	}
	if apiData != dbData {
		var diff float64
		if apiData < dbData {
			diff = gctmath.PercentageChange(apiData, dbData)
		} else {
			diff = gctmath.PercentageChange(dbData, apiData)
		}
		if diff > job.IssueTolerancePercentage {
			issue = fmt.Sprintf("%s api: %v db: %v diff: %v %%", candleField, apiData, dbData, diff)
		}
		if job.ReplaceOnIssue &&
			job.IssueTolerancePercentage != 0 &&
			diff > job.IssueTolerancePercentage &&
			job.SecondaryExchangeSource == "" {
			replace = true
		}
	}
	return issue, replace
}

// SetJobRelationship will add/modify/delete a relationship with an existing job
// it will add the relationship and set the jobNickname job to paused
// if deleting, it will remove the relationship from the database and set the job to active
func (m *DataHistoryManager) SetJobRelationship(prerequisiteJobNickname, jobNickname string) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.started.Load() {
		return ErrSubSystemNotStarted
	}
	if jobNickname == "" {
		return errNicknameUnset
	}
	status := dataHistoryStatusPaused
	if prerequisiteJobNickname == "" {
		j, err := m.GetByNickname(jobNickname, false)
		if err != nil {
			return err
		}
		status = j.Status
		if j.Status == dataHistoryStatusPaused {
			status = dataHistoryStatusActive
		}
	} else {
		j, err := m.GetByNickname(prerequisiteJobNickname, false)
		if err != nil {
			return err
		}
		if j.Status != dataHistoryStatusActive && j.Status != dataHistoryStatusPaused {
			return fmt.Errorf("cannot set prerequisite %v to job %v, %w", prerequisiteJobNickname, jobNickname, errJobMustBeActiveOrPaused)
		}
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
	j, err := m.GetByNickname(job.Nickname, false)
	if err != nil && !errors.Is(err, errJobNotFound) {
		return err
	}
	if insertOnly && j != nil ||
		(j != nil && j.Status != dataHistoryStatusActive) {
		return fmt.Errorf("upsert job %w nickname: %s - status: %s ", errNicknameInUse, j.Nickname, j.Status)
	}
	if job.PrerequisiteJobNickname != "" {
		var p *DataHistoryJob
		p, err = m.GetByNickname(job.PrerequisiteJobNickname, false)
		if err != nil {
			return fmt.Errorf("upsert job %s could not find prerequisite job nickname %v %w", job.Nickname, job.PrerequisiteJobNickname, err)
		}
		if p.Status != dataHistoryStatusActive && p.Status != dataHistoryStatusPaused {
			return fmt.Errorf("upsert job %s prerequisite job nickname %v already completed %w", job.Nickname, p.Nickname, errJobInvalid)
		}
	}

	err = m.validateJob(job)
	if err != nil {
		return err
	}
	existingJob, err := m.GetByNickname(job.Nickname, false)
	if err != nil && err != errJobNotFound {
		return err
	}
	if existingJob != nil {
		job.ID = existingJob.ID
	}
	if job.ID == uuid.Nil() {
		job.ID = uuid.NewV4()
	}
	job.rangeHolder, err = kline.CalculateCandleDateRanges(job.StartDate, job.EndDate, job.rangeInterval(), job.RequestSizeLimit)
	if err != nil {
		return err
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
	return m.jobDB.SetRelationshipByNickname(job.PrerequisiteJobNickname, job.Nickname, int64(dataHistoryStatusPaused))
}

func (m *DataHistoryManager) validateJob(job *DataHistoryJob) error {
	if !m.IsRunning() {
		return ErrSubSystemNotStarted
	}
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
	exchangeName := job.Exchange
	if job.DataType == dataHistoryCandleValidationSecondarySourceType {
		if job.SecondaryExchangeSource == "" {
			return fmt.Errorf("job %s %w, secondary exchange name required to lookup existing results", job.Nickname, common.ErrExchangeNameNotSet)
		}
		exchangeName = job.SecondaryExchangeSource
		if job.Exchange == "" {
			return fmt.Errorf("job %s %w, exchange name required to lookup existing results", job.Nickname, common.ErrExchangeNameNotSet)
		}
	}
	exch, err := m.exchangeManager.GetExchangeByName(exchangeName)
	if err != nil {
		return fmt.Errorf("job %s cannot process job: %v", job.Nickname, err)
	}
	pairs, err := exch.GetEnabledPairs(job.Asset)
	if err != nil {
		return fmt.Errorf("job %s exchange %s asset %s currency %s %w", job.Nickname, job.Exchange, job.Asset, job.Pair, err)
	}
	if !pairs.Contains(job.Pair, false) {
		return fmt.Errorf("job %s exchange %s asset %s currency %s %w", job.Nickname, job.Exchange, job.Asset, job.Pair, errCurrencyNotEnabled)
	}
	if job.Results == nil {
		job.Results = make(map[int64][]DataHistoryJobResult)
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
		if job.Interval > kline.FourHour {
			log.Warnf(log.DataHistory, "job %s interval %v above the limit of 4h, defaulting to %v interval size worth of trades to fetch", job.Nickname, job.Interval.Word(), defaultDataHistoryTradeInterval)
			job.Interval = defaultDataHistoryTradeInterval
		} else if job.Interval < kline.OneMin {
			log.Warnf(log.DataHistory, "job %s interval %v below the limit of 1m, defaulting to %v interval size worth of trades to fetch", job.Nickname, job.Interval.Word(), defaultDataHistoryTradeInterval)
			job.Interval = defaultDataHistoryTradeInterval
		}
		if job.RequestSizeLimit > defaultDataHistoryTradeRequestSize {
			log.Warnf(log.DataHistory, "job %s interval request size %v outside limit of %v, defaulting to %v intervals worth of trades per request", job.Nickname, job.RequestSizeLimit, defaultDataHistoryTradeRequestSize, defaultDataHistoryTradeRequestSize)
			job.RequestSizeLimit = defaultDataHistoryTradeRequestSize
		}
	}

	// A job's ranges need whole-second intervals: results are matched to their ranges by start and end, which SQLite keeps
	// to the second, and validation matches candles by the second. Candle conversion checks both of its intervals below
	if job.DataType != dataHistoryConvertCandlesDataType && job.Interval.Duration()%time.Second != 0 {
		return fmt.Errorf("job %s %s %w: %s", job.Nickname, job.DataType, errIntervalNotWholeSeconds, job.Interval)
	}

	b := exch.GetBase()
	// TODO: In future allow custom candles.
	if !b.Features.Enabled.Kline.Intervals.ExchangeSupported(job.Interval) &&
		(job.DataType == dataHistoryCandleDataType || job.DataType == dataHistoryCandleValidationDataType) {
		return fmt.Errorf("job interval %s %s %w %s", job.Nickname, job.Interval.Word(), kline.ErrUnsupportedInterval, job.Exchange)
	}
	if job.DataType == dataHistoryConvertTradesDataType && job.Interval <= 0 {
		return fmt.Errorf("job conversion interval %s %s %w %s", job.Nickname, job.Interval.Word(), kline.ErrUnsupportedInterval, job.Exchange)
	}

	if job.DataType == dataHistoryConvertCandlesDataType {
		if job.ConversionInterval <= 0 {
			return fmt.Errorf("job conversion interval %s %s %w %s", job.Nickname, job.ConversionInterval.Word(), kline.ErrUnsupportedInterval, job.Exchange)
		}
		if job.Interval <= 0 {
			return fmt.Errorf("job interval %s %s %w %s", job.Nickname, job.Interval.Word(), kline.ErrUnsupportedInterval, job.Exchange)
		}
		// The candle store keeps intervals in whole seconds, and conversion counts fixed spans rather than calendar months
		if job.Interval.IsCalendar() || job.ConversionInterval.IsCalendar() || job.Interval.Duration()%time.Second != 0 || job.ConversionInterval.Duration()%time.Second != 0 {
			return fmt.Errorf("job conversion %s from %s to %s %w %s", job.Nickname, job.Interval, job.ConversionInterval, kline.ErrUnsupportedInterval, job.Exchange)
		}
	}

	if job.DataType == dataHistoryCandleValidationDataType || job.DataType == dataHistoryCandleValidationSecondarySourceType {
		if job.DecimalPlaceComparison > maxDecimalPlaceComparison {
			return fmt.Errorf("job %s %w: %d exceeds %d", job.Nickname, errInvalidDecimalPlaces, job.DecimalPlaceComparison, maxDecimalPlaceComparison)
		}
		// Zero is unset for the primary source, while a secondary source job compares whole units
		if job.DecimalPlaceComparison == 0 && job.DataType == dataHistoryCandleValidationDataType {
			log.Warnf(log.DataHistory, "job %s decimal place comparison %v invalid. defaulting to %v decimal places when comparing data for validation", job.Nickname, job.DecimalPlaceComparison, defaultDecimalPlaceComparison)
			job.DecimalPlaceComparison = defaultDecimalPlaceComparison
		}
		if job.RequestSizeLimit > defaultDataHistoryRequestSizeLimit {
			log.Warnf(log.DataHistory, "job %s validation batch %v above limit of %v. defaulting to %v intervals to process per request", job.Nickname, job.RequestSizeLimit, defaultDataHistoryRequestSizeLimit, defaultDataHistoryRequestSizeLimit)
			job.RequestSizeLimit = defaultDataHistoryRequestSizeLimit
		}
	}

	job.StartDate = job.StartDate.Round(job.Interval.Duration())
	job.EndDate = job.EndDate.Round(job.Interval.Duration())
	if job.DataType == dataHistoryConvertCandlesDataType {
		// Conversion keeps to the whole conversion intervals within the dates, once rounded to the source interval, so
		// that it never needs source candles from beyond them
		conversion := job.ConversionInterval.Duration()
		if start := job.StartDate.Truncate(conversion); start.Before(job.StartDate) {
			job.StartDate = start.Add(conversion)
		}
		job.EndDate = job.EndDate.Truncate(conversion)
	}
	if err := common.StartEndTimeCheck(job.StartDate, job.EndDate); err != nil {
		return fmt.Errorf("job %s %w start: %v end %v", job.Nickname, err, job.StartDate, job.EndDate)
	}
	if err := checkIntervalCount(job.StartDate, job.EndDate, job.Interval.Duration(), maxDataHistoryJobIntervals); err != nil {
		return fmt.Errorf("job %s %w", job.Nickname, err)
	}
	if job.DataType == dataHistoryConvertCandlesDataType {
		// A convert candles job's date ranges are built with its conversion interval
		if err := checkIntervalCount(job.StartDate, job.EndDate, job.ConversionInterval.Duration(), maxDataHistoryJobIntervals); err != nil {
			return fmt.Errorf("job %s conversion interval %w", job.Nickname, err)
		}
	}

	return nil
}

// GetByID returns a job's details from its ID
func (m *DataHistoryManager) GetByID(id uuid.UUID) (*DataHistoryJob, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if !m.started.Load() {
		return nil, ErrSubSystemNotStarted
	}
	if id == uuid.Nil() {
		return nil, errEmptyID
	}
	dbJ, err := m.jobDB.GetByID(id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w with id %s", errJobNotFound, id)
		}
		return nil, fmt.Errorf("could not load job with id %s: %w", id, err)
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
	if !m.started.Load() {
		return nil, ErrSubSystemNotStarted
	}
	if fullDetails {
		dbJ, err := m.jobDB.GetJobAndAllResults(nickname)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errJobNotFound
			}
			return nil, fmt.Errorf("job %s could not load job from database: %w", nickname, err)
		}
		result, err := m.convertDBModelToJob(dbJ)
		if err != nil {
			return nil, fmt.Errorf("could not convert model with nickname %s %w", nickname, err)
		}
		return result, nil
	}
	j, err := m.jobDB.GetByNickName(nickname)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no need to display normal sql err to user
			return nil, errJobNotFound
		}
		return nil, fmt.Errorf("job %s could not load job from database: %w", nickname, err)
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
	if !m.started.Load() {
		return nil, ErrSubSystemNotStarted
	}
	if err := common.StartEndTimeCheck(start, end); err != nil {
		return nil, err
	}
	dbJobs, err := m.jobDB.GetJobsBetween(start, end)
	if err != nil {
		return nil, err
	}
	results := make([]*DataHistoryJob, len(dbJobs))
	for i := range dbJobs {
		dbJob, err := m.convertDBModelToJob(&dbJobs[i])
		if err != nil {
			return nil, err
		}
		results[i] = dbJob
	}
	return results, nil
}

// SetJobStatus helper function to assist in setting a job to deleted
func (m *DataHistoryManager) SetJobStatus(nickname, id string, status dataHistoryStatus) error {
	if m == nil {
		return ErrNilSubsystem
	}
	if !m.started.Load() {
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
		return fmt.Errorf("%w received: %s, can only pause, unpause or delete jobs", errBadStatus, status.String())
	}
	var dbJob *datahistoryjob.DataHistoryJob
	var err error
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
	if dbJob.Status == int64(status) {
		return fmt.Errorf("%w job %v, status already set to %v", errBadStatus, dbJob.Nickname, dataHistoryStatus(dbJob.Status))
	}
	switch dataHistoryStatus(dbJob.Status) {
	case dataHistoryStatusActive:
		if status != dataHistoryStatusRemoved && status != dataHistoryStatusPaused {
			return fmt.Errorf("%w job %v", errBadStatus, dataHistoryStatus(dbJob.Status))
		}
	case dataHistoryStatusPaused:
		if status != dataHistoryStatusRemoved && status != dataHistoryStatusActive {
			return fmt.Errorf("%w job %v", errBadStatus, dataHistoryStatus(dbJob.Status))
		}
	default:
		return fmt.Errorf("%w job %v, invalid status", errBadStatus, dataHistoryStatus(dbJob.Status))
	}

	dbJob.Status = int64(status)
	err = m.jobDB.Upsert(dbJob)
	if err != nil {
		return err
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

	var results []DataHistoryJob
	jobs, err := m.jobDB.GetAllIncompleteJobsAndResults()
	if err != nil {
		return nil, err
	}
	for i := range jobs {
		if jobs[i].Status == int64(dataHistoryStatusActive) {
			var job *DataHistoryJob
			job, err = m.convertDBModelToJob(&jobs[i])
			if err != nil {
				return nil, err
			}
			if job != nil {
				results = append(results, *job)
			}
		}
	}
	return results, nil
}

// GenerateJobSummary returns a human readable summary of a job's status
func (m *DataHistoryManager) GenerateJobSummary(nickname string) (*DataHistoryJobSummary, error) {
	if m == nil {
		return nil, ErrNilSubsystem
	}
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
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

// convertDBModelToJob converts a database model into a data history job
func (m *DataHistoryManager) convertDBModelToJob(dbModel *datahistoryjob.DataHistoryJob) (*DataHistoryJob, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	id, err := uuid.Parse(dbModel.ID)
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

	ai, err := asset.New(dbModel.Asset)
	if err != nil {
		return nil, fmt.Errorf("job %s could not derive asset: %w",
			dbModel.Nickname, err)
	}

	resp := &DataHistoryJob{
		ID:                       id,
		Nickname:                 dbModel.Nickname,
		Exchange:                 dbModel.ExchangeName,
		Asset:                    ai,
		Pair:                     cp,
		StartDate:                dbModel.StartDate,
		EndDate:                  dbModel.EndDate,
		Interval:                 kline.Interval(dbModel.Interval),
		RunBatchLimit:            dbModel.BatchSize,
		RequestSizeLimit:         dbModel.RequestSizeLimit,
		DataType:                 dataHistoryDataType(dbModel.DataType),
		MaxRetryAttempts:         dbModel.MaxRetryAttempts,
		Status:                   dataHistoryStatus(dbModel.Status),
		CreatedDate:              dbModel.CreatedDate,
		Results:                  jobResults,
		OverwriteExistingData:    dbModel.OverwriteData,
		ConversionInterval:       kline.Interval(dbModel.ConversionInterval),
		DecimalPlaceComparison:   dbModel.DecimalPlaceComparison,
		SecondaryExchangeSource:  dbModel.SecondarySourceExchangeName,
		IssueTolerancePercentage: dbModel.IssueTolerancePercentage,
		ReplaceOnIssue:           dbModel.ReplaceOnIssue,
		PrerequisiteJobNickname:  dbModel.PrerequisiteJobNickname,
	}
	if resp.PrerequisiteJobNickname != "" {
		prereqID, err := uuid.Parse(dbModel.PrerequisiteJobID)
		if err != nil {
			return nil, err
		}
		resp.PrerequisiteJobID = prereqID
	}

	return resp, nil
}

func (m *DataHistoryManager) convertDBResultToJobResult(dbModels []*datahistoryjobresult.DataHistoryJobResult) (map[int64][]DataHistoryJobResult, error) {
	if !m.IsRunning() {
		return nil, ErrSubSystemNotStarted
	}
	result := make(map[int64][]DataHistoryJobResult)
	for i := range dbModels {
		id, err := uuid.Parse(dbModels[i].ID)
		if err != nil {
			return nil, err
		}

		jobID, err := uuid.Parse(dbModels[i].JobID)
		if err != nil {
			return nil, err
		}
		lookup := result[dbModels[i].IntervalStartDate.Unix()]
		lookup = append(lookup, DataHistoryJobResult{
			ID:                id,
			JobID:             jobID,
			IntervalStartDate: dbModels[i].IntervalStartDate,
			IntervalEndDate:   dbModels[i].IntervalEndDate,
			Status:            dataHistoryStatus(dbModels[i].Status),
			Result:            dbModels[i].Result,
			Date:              dbModels[i].Date,
		})
		result[dbModels[i].IntervalStartDate.Unix()] = lookup
	}

	return result, nil
}

func (m *DataHistoryManager) convertJobResultToDBResult(results map[int64][]DataHistoryJobResult) []*datahistoryjobresult.DataHistoryJobResult {
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
		Nickname:                    job.Nickname,
		ExchangeName:                job.Exchange,
		Asset:                       job.Asset.String(),
		Base:                        job.Pair.Base.String(),
		Quote:                       job.Pair.Quote.String(),
		StartDate:                   job.StartDate,
		EndDate:                     job.EndDate,
		Interval:                    int64(job.Interval.Duration()),
		RequestSizeLimit:            job.RequestSizeLimit,
		DataType:                    int64(job.DataType),
		MaxRetryAttempts:            job.MaxRetryAttempts,
		BatchSize:                   job.RunBatchLimit,
		Status:                      int64(job.Status),
		CreatedDate:                 job.CreatedDate,
		Results:                     m.convertJobResultToDBResult(job.Results),
		PrerequisiteJobNickname:     job.PrerequisiteJobNickname,
		ConversionInterval:          int64(job.ConversionInterval),
		OverwriteData:               job.OverwriteExistingData,
		DecimalPlaceComparison:      job.DecimalPlaceComparison,
		SecondarySourceExchangeName: job.SecondaryExchangeSource,
		IssueTolerancePercentage:    job.IssueTolerancePercentage,
		ReplaceOnIssue:              job.ReplaceOnIssue,
	}
	if job.ID != uuid.Nil() {
		model.ID = job.ID.String()
	}
	if job.PrerequisiteJobID != uuid.Nil() {
		model.PrerequisiteJobID = job.PrerequisiteJobID.String()
	}

	return model
}
