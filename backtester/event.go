package backtest

import "time"

func (e Event) Time() time.Time {
	return e.time
}

// SetTime returns the timestamp of an event
func (e *Event) SetTime(t time.Time) {
	e.time = t
}

