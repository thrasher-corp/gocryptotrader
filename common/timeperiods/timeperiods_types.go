package timeperiods

import (
	"sort"
	"time"
)

type TimePeriodCalculator struct {
	start           time.Time
	end             time.Time
	comparisonTimes []time.Time
	TimePeriods     []TimePeriod
	periodDuration  time.Duration
	TimeRanges      []TimeRange
}

type TimePeriod struct {
	Time        time.Time
	dataInRange bool
}

type TimeRange struct {
	StartOfRange   time.Time
	EndOfRange     time.Time
	HasDataInRange bool
}

func (t *TimePeriodCalculator) Sort(desc bool) {
	sort.Slice(t.TimePeriods, func(i, j int) bool {
		if desc {
			return t.TimePeriods[i].Time.After(t.TimePeriods[j].Time)
		}
		return t.TimePeriods[i].Time.Before(t.TimePeriods[j].Time)
	})
}
