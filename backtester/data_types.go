package backtest

import "time"

type Candle struct {
	timestamp time.Time

	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

func (c Candle) Time() time.Time {
	return time.Now()
}

func (c Candle) SetTime(t time.Time) {

}
