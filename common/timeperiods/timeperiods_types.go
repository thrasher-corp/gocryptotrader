package timeperiods

import (
	"sort"
	"time"
)

// TimePeriodCalculator is able analyse
// a time span and either break them down into
// chunks, or determine ranges that contain data or not
type TimePeriodCalculator struct {
	start           time.Time
	end             time.Time
	comparisonTimes []time.Time
	TimePeriods     []TimePeriod
	periodDuration  time.Duration
	TimeRanges      []TimeRange
}

// TimePeriod is a basic type which will know
// whether a period in time contains data
type TimePeriod struct {
	Time        time.Time
	dataInRange bool
}

// TimeRange holds a start and end dat range
// and whether that range contains data
type TimeRange struct {
	StartOfRange   time.Time
	EndOfRange     time.Time
	HasDataInRange bool
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
