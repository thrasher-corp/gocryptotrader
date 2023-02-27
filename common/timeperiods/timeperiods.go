package timeperiods

import (
	"errors"
	"sort"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// FindTimeRangesContainingData will break the start and end into time periods using the provided period
// it will then check whether any comparisonTimes are within those periods and concatenate them
// eg if no comparisonTimes match, you will receive 1 TimeRange of Start End with dataInRange = false
// eg2 if 1 comparisonTime matches in the middle of start and end, you will receive three ranges
func FindTimeRangesContainingData(start, end time.Time, period time.Duration, comparisonTimes []time.Time) ([]TimeRange, error) {
	var errs error
	if start.IsZero() {
		errs = common.AppendError(errs, errors.New("invalid start time"))
	}
	if end.IsZero() {
		errs = common.AppendError(errs, errors.New("invalid end time"))
	}
	if err := validatePeriod(period); err != nil {
		errs = common.AppendError(errs, err)
	}
	if errs != nil {
		return nil, errs
	}
	var t TimePeriodCalculator
	t.periodDuration = period
	t.start = start.Truncate(period)
	t.end = end.Truncate(period)
	t.comparisonTimes = comparisonTimes

	t.setTimePeriodExists()
	t.Sort(false)
	t.calculateRanges()

	return t.TimeRanges, nil
}

func validatePeriod(period time.Duration) error {
	if period != time.Hour &&
		period != time.Second &&
		period != time.Minute &&
		period != time.Hour*24 {
		return errors.New("invalid period")
	}
	return nil
}

// CalculateTimePeriodsInRange can break down start and end times into time periods
// eg 1 hourly intervals
func CalculateTimePeriodsInRange(start, end time.Time, period time.Duration) ([]TimePeriod, error) {
	var errs error
	if start.IsZero() {
		errs = common.AppendError(errs, errors.New("invalid start time"))
	}
	if end.IsZero() {
		errs = common.AppendError(errs, errors.New("invalid end time"))
	}
	if err := validatePeriod(period); err != nil {
		errs = common.AppendError(errs, err)
	}
	if errs != nil {
		return nil, errs
	}

	var t TimePeriodCalculator
	t.periodDuration = period
	t.start = start.Truncate(period)
	t.end = end.Truncate(period)

	t.calculatePeriods()

	return t.TimePeriods, nil
}

func (t *TimePeriodCalculator) calculateRanges() {
	var tr TimeRange
	for i := range t.TimePeriods {
		if i != 0 {
			if (t.TimePeriods[i].dataInRange && !t.TimePeriods[i-1].dataInRange) ||
				(!t.TimePeriods[i].dataInRange && t.TimePeriods[i-1].dataInRange) {
				// the status has changed and therefore a range has ended
				tr.HasDataInRange = t.TimePeriods[i-1].dataInRange
				tr.EndOfRange = t.TimePeriods[i].Time
				t.TimeRanges = append(t.TimeRanges, tr)
				tr = TimeRange{}
			}
		}
		if tr.StartOfRange.IsZero() {
			// start of new time range
			tr.StartOfRange = t.TimePeriods[i].Time
		}
	}
	if !tr.StartOfRange.IsZero() {
		if tr.EndOfRange.IsZero() {
			tr.EndOfRange = t.end
		}
		tr.HasDataInRange = t.TimePeriods[len(t.TimePeriods)-1].dataInRange
		t.TimeRanges = append(t.TimeRanges, tr)
	}
}

func (t *TimePeriodCalculator) calculatePeriods() {
	if t.start.IsZero() || t.end.IsZero() {
		return
	}
	if t.start.After(t.end) {
		return
	}
	iterateDateMate := t.start
	for !iterateDateMate.Equal(t.end) && iterateDateMate.Before(t.end) {
		tp := TimePeriod{
			Time:        iterateDateMate,
			dataInRange: false,
		}
		t.TimePeriods = append(t.TimePeriods, tp)
		iterateDateMate = iterateDateMate.Add(t.periodDuration)
	}
}

// setTimePeriodExists compares loaded comparisonTimes
// against calculated TimePeriods to determine whether
// there is existing data within the time period
func (t *TimePeriodCalculator) setTimePeriodExists() {
	t.calculatePeriods()
	for i := range t.TimePeriods {
		for j := range t.comparisonTimes {
			if t.comparisonTimes[j].Truncate(t.periodDuration).Equal(t.TimePeriods[i].Time) {
				t.TimePeriods[i].dataInRange = true
				break
			}
		}
	}
}

// Sort will sort the time period asc or desc
func (t *TimePeriodCalculator) Sort(desc bool) {
	sort.Slice(t.TimePeriods, func(i, j int) bool {
		if desc {
			return t.TimePeriods[i].Time.After(t.TimePeriods[j].Time)
		}
		return t.TimePeriods[i].Time.Before(t.TimePeriods[j].Time)
	})
}
