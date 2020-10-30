package timeperiods

import (
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
