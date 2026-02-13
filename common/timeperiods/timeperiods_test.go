package timeperiods

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindTimeRangesContainingData(t *testing.T) {
	// validation issues
	_, err := FindTimeRangesContainingData(
		time.Time{},
		time.Time{},
		0,
		nil,
	)
	require.EqualError(t, err, "invalid start time, invalid end time, invalid period", "FindTimeRangesContainingData must return correct validation error")
	// empty trade times
	searchStartTime := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)
	searchEndTime := time.Date(2020, 1, 1, 10, 0, 0, 0, time.UTC)
	tradeTimes := make([]time.Time, 0, 5)
	var ranges []TimeRange
	ranges, err = FindTimeRangesContainingData(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error with empty trade times")
	assert.Len(t, ranges, 1, "ranges should have 1 time range")
	// 1 trade with 3 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 2, 0, 0, 0, time.UTC))
	ranges, err = FindTimeRangesContainingData(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error")
	assert.Len(t, ranges, 3, "ranges should have 3 time ranges")
	// 2 trades with 3 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 3, 0, 0, 0, time.UTC))
	ranges, err = FindTimeRangesContainingData(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error")
	assert.Len(t, ranges, 3, "ranges should have 3 time ranges")
	// 3 trades with 5 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 5, 0, 0, 0, time.UTC))
	ranges, err = FindTimeRangesContainingData(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error")
	assert.Len(t, ranges, 5, "ranges should have 5 time ranges")
	// 4 trades with 5 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 6, 0, 0, 0, time.UTC))
	ranges, err = FindTimeRangesContainingData(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error")
	assert.Len(t, ranges, 5, "ranges should have 5 time ranges")
	// 5 trades with 6 periods
	tradeTimes = append(tradeTimes, time.Date(2020, 1, 1, 9, 0, 0, 0, time.UTC))
	ranges, err = FindTimeRangesContainingData(
		searchStartTime,
		searchEndTime,
		time.Hour,
		tradeTimes,
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error")
	assert.Len(t, ranges, 6, "ranges should have 6 time ranges")
}

func TestCalculateTimePeriodsInRange(t *testing.T) {
	// validation issues
	_, err := CalculateTimePeriodsInRange(time.Time{}, time.Time{}, 0)
	if err != nil && err.Error() != "invalid start time, invalid end time, invalid period" {
		t.Fatal(err)
	}
	// start after end
	var intervals []TimePeriod
	timeStart := time.Date(2020, 1, 1, 1, 0, 0, 0, time.UTC)
	timeEnd := time.Date(2020, 1, 2, 1, 0, 0, 0, time.UTC)
	intervals, err = CalculateTimePeriodsInRange(timeEnd, timeStart, time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 0 {
		t.Errorf("expected 0 interval(s), received %v", len(intervals))
	}
	// 1 interval
	intervals, err = CalculateTimePeriodsInRange(timeStart, timeStart.Add(time.Hour), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 1 {
		t.Errorf("expected 1 interval(s), received %v", len(intervals))
	}
	// multiple intervals
	intervals, err = CalculateTimePeriodsInRange(timeStart, timeEnd, time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 24 {
		t.Errorf("expected 24 interval(s), received %v", len(intervals))
	}
	// odd time
	intervals, err = CalculateTimePeriodsInRange(timeStart.Add(-time.Minute*30), timeEnd, time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 25 {
		t.Errorf("expected 25 interval(s), received %v", len(intervals))
	}
	// truncate always goes to zero, no mid rounding
	intervals, err = CalculateTimePeriodsInRange(timeStart, timeStart.Add(time.Minute), time.Hour)
	if err != nil {
		t.Error(err)
	}
	if len(intervals) != 0 {
		t.Errorf("expected 0 interval(s), received %v", len(intervals))
	}
}

func TestValidateCalculatePeriods(t *testing.T) {
	var tpc TimePeriodCalculator
	tpc.calculatePeriods()
	if len(tpc.TimePeriods) > 0 {
		t.Error("validation has been removed")
	}
}

func TestSort(t *testing.T) {
	var tpc TimePeriodCalculator
	date1 := time.Date(2020, 1, 1, 1, 1, 1, 1, time.UTC)
	date2 := time.Date(1901, 1, 1, 1, 1, 1, 1, time.UTC)
	tpc.TimePeriods = append(tpc.TimePeriods,
		TimePeriod{
			Time: date1,
		},
		TimePeriod{
			Time: date2,
		},
	)
	tpc.Sort(false)
	if !tpc.TimePeriods[0].Time.Equal(date2) {
		t.Errorf("expected %v, received  %v", date2, tpc.TimePeriods[0].Time)
	}

	tpc.Sort(true)
	if !tpc.TimePeriods[0].Time.Equal(date1) {
		t.Errorf("expected %v, received  %v", date1, tpc.TimePeriods[0].Time)
	}
}
