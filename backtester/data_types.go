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

type DataHandler interface {
	Next() (DataEvent, bool)
	Stream() []DataEvent
	History() []DataEvent
	Latest() DataEvent
	Reset()

	StreamOpen() []float64
	StreamHigh() []float64
	StreamLow() []float64
	StreamClose() []float64
	StreamVolume() []float64
}
