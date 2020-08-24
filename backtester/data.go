package backtest

import "time"

func (c Candle) Price() float64 {
	return c.Close
}

func (c Candle) Candle() Candle {
	return c
}

func (c Candle) Time() time.Time {
	return c.timestamp
}

func (c Candle) SetTime(t time.Time) {
	c.timestamp = t
}
