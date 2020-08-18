package backtest

func (c Candle) Price() float64 {
	return c.Close
}

