package backtest

import "time"

func (e Event) Time() time.Time {
	return e.time
}

func (e *Event) SetTime(t time.Time) {
	e.time = t
}

