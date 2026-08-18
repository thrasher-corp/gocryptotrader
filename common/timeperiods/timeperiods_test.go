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

func TestFindTimeRangesContainingDataOrderedRanges(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	ranges, err := FindTimeRangesContainingData(
		start,
		start.Add(5*time.Minute),
		time.Minute,
		[]time.Time{
			start.Add(3*time.Minute + 30*time.Second),
			start.Add(30 * time.Second),
		},
	)
	require.NoError(t, err, "FindTimeRangesContainingData must not error")
	expected := []TimeRange{
		{StartOfRange: start, EndOfRange: start.Add(time.Minute), HasDataInRange: true},
		{StartOfRange: start.Add(time.Minute), EndOfRange: start.Add(3 * time.Minute)},
		{StartOfRange: start.Add(3 * time.Minute), EndOfRange: start.Add(4 * time.Minute), HasDataInRange: true},
		{StartOfRange: start.Add(4 * time.Minute), EndOfRange: start.Add(5 * time.Minute)},
	}
	assert.Equal(t, expected, ranges, "FindTimeRangesContainingData should return correctly ordered ranges")
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

func TestSetTimePeriodExists(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	testCalculator := TimePeriodCalculator{
		start:          start,
		end:            start.Add(4 * time.Minute),
		periodDuration: time.Minute,
		comparisonTimes: []time.Time{
			start.Add(-time.Minute),
			start.Add(30 * time.Second),
			start.Add(45 * time.Second),
			start.Add(2*time.Minute + 30*time.Second).In(time.FixedZone("comparison", 10*60*60)),
			start.Add(4 * time.Minute),
			start.Add(5 * time.Minute),
		},
	}
	testCalculator.setTimePeriodExists()
	assert.Equal(t, []TimePeriod{
		{Time: start, dataInRange: true},
		{Time: start.Add(time.Minute)},
		{Time: start.Add(2 * time.Minute), dataInRange: true},
		{Time: start.Add(3 * time.Minute)},
	}, testCalculator.TimePeriods, "setTimePeriodExists should match comparison times to the correct periods")

	emptyCalculator := TimePeriodCalculator{
		start:          start,
		end:            start.Add(2 * time.Minute),
		periodDuration: time.Minute,
	}
	emptyCalculator.setTimePeriodExists()
	assert.Equal(t, []TimePeriod{{Time: start}, {Time: start.Add(time.Minute)}}, emptyCalculator.TimePeriods, "setTimePeriodExists should leave periods unmatched without comparison times")

	zeroRangeCalculator := TimePeriodCalculator{
		start:           start,
		end:             start,
		periodDuration:  time.Minute,
		comparisonTimes: []time.Time{start},
	}
	zeroRangeCalculator.setTimePeriodExists()
	assert.Empty(t, zeroRangeCalculator.TimePeriods, "setTimePeriodExists should leave a zero-length range empty")

	reversedRangeCalculator := TimePeriodCalculator{
		start:           start.Add(time.Minute),
		end:             start,
		periodDuration:  time.Minute,
		comparisonTimes: []time.Time{start},
	}
	reversedRangeCalculator.setTimePeriodExists()
	assert.Empty(t, reversedRangeCalculator.TimePeriods, "setTimePeriodExists should leave a reversed range empty")

	longRangeStart := time.Date(1700, time.January, 1, 0, 0, 0, 0, time.UTC)
	longRangeEnd := longRangeStart.AddDate(400, 0, 0)
	longRangeCalculator := TimePeriodCalculator{
		start:           longRangeStart,
		end:             longRangeEnd,
		periodDuration:  24 * time.Hour,
		comparisonTimes: []time.Time{longRangeEnd.Add(-time.Hour)},
	}
	longRangeCalculator.setTimePeriodExists()
	matchedPeriods := 0
	for i := range longRangeCalculator.TimePeriods {
		if longRangeCalculator.TimePeriods[i].dataInRange {
			matchedPeriods++
		}
	}
	assert.Equal(t, 1, matchedPeriods, "setTimePeriodExists should match one period across a multi-century range")
	require.NotEmpty(t, longRangeCalculator.TimePeriods, "setTimePeriodExists must calculate periods across a multi-century range")
	assert.True(t, longRangeCalculator.TimePeriods[len(longRangeCalculator.TimePeriods)-1].dataInRange, "setTimePeriodExists should match the final period across a multi-century range")

	zeroStartCalculator := TimePeriodCalculator{
		start:           time.Time{},
		end:             time.Time{}.Add(time.Minute),
		periodDuration:  time.Minute,
		comparisonTimes: []time.Time{time.Time{}.Add(30 * time.Second)},
	}
	zeroStartCalculator.setTimePeriodExists()
	assert.Empty(t, zeroStartCalculator.TimePeriods, "setTimePeriodExists should leave a zero-start range empty")

	truncatedZeroStart := time.Time{}.Add(30 * time.Second)
	ranges, err := FindTimeRangesContainingData(truncatedZeroStart, truncatedZeroStart.Add(time.Minute), time.Minute, []time.Time{truncatedZeroStart})
	require.NoError(t, err, "FindTimeRangesContainingData must not error when start truncates to zero")
	assert.Empty(t, ranges, "FindTimeRangesContainingData should leave a range empty when start truncates to zero")
}

func TestSetTimePeriodExistsSupportedPeriods(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	secondCalculator := TimePeriodCalculator{
		start:           start,
		end:             start.Add(2 * time.Second),
		periodDuration:  time.Second,
		comparisonTimes: []time.Time{start},
	}
	secondCalculator.setTimePeriodExists()
	assert.Equal(t, []TimePeriod{{Time: start, dataInRange: true}, {Time: start.Add(time.Second)}}, secondCalculator.TimePeriods, "setTimePeriodExists should match an exact start with a second period")

	hourCalculator := TimePeriodCalculator{
		start:           start,
		end:             start.Add(3 * time.Hour),
		periodDuration:  time.Hour,
		comparisonTimes: []time.Time{start.Add(90 * time.Minute)},
	}
	hourCalculator.setTimePeriodExists()
	assert.Equal(t, []TimePeriod{{Time: start}, {Time: start.Add(time.Hour), dataInRange: true}, {Time: start.Add(2 * time.Hour)}}, hourCalculator.TimePeriods, "setTimePeriodExists should match the correct hour period")
}

func TestSetTimePeriodExistsReuse(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	calculator := TimePeriodCalculator{
		start:           start,
		end:             start.Add(2 * time.Minute),
		periodDuration:  time.Minute,
		comparisonTimes: []time.Time{start},
	}
	calculator.setTimePeriodExists()
	calculator.comparisonTimes = []time.Time{start.Add(time.Minute)}
	calculator.setTimePeriodExists()
	assert.Equal(t, []TimePeriod{
		{Time: start, dataInRange: true},
		{Time: start.Add(time.Minute), dataInRange: true},
		{Time: start},
		{Time: start.Add(time.Minute), dataInRange: true},
	}, calculator.TimePeriods, "setTimePeriodExists should preserve matches when reused")
}

func TestSetTimePeriodExistsReuseWithoutNewPeriods(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	calculator := TimePeriodCalculator{
		start:          start,
		end:            start.Add(2 * time.Minute),
		periodDuration: time.Minute,
	}
	calculator.setTimePeriodExists()
	calculator.start = calculator.end
	calculator.comparisonTimes = []time.Time{start.Add(time.Minute)}
	calculator.setTimePeriodExists()
	expected := []TimePeriod{
		{Time: start},
		{Time: start.Add(time.Minute), dataInRange: true},
	}
	assert.Equal(t, expected, calculator.TimePeriods, "setTimePeriodExists should match existing periods without calculating new periods")
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
