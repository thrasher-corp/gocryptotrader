package backtest

func (c Candle) Price() float64 {
	return c.Close
}

func (c Candle) Candle() Candle {
	return c
}
