package backtest

type Portfolio struct {
	InitialFunds float64
	Funds        float64
	holdings     Position
	orderBook    []OrderEvent
}