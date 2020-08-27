package backtest

type DataType uint8

const (
	DataTypeCandle DataType = iota
	DataTypeTick
)

type Candle struct {
	Event
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}
type Tick struct {
	Event
	DataEvent
	Bid float64
	Ask float64
}



type Data struct {
	latest DataEventHandler
	stream []DataEventHandler

	offset int
}
