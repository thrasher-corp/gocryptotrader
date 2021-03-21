package kline

// ClosePrice returns the closing price of a kline
func (k *Kline) ClosePrice() float64 {
	return k.Close
}

// HighPrice returns the high price of a kline
func (k *Kline) HighPrice() float64 {
	return k.High
}

// LowPrice returns the low price of a kline
func (k *Kline) LowPrice() float64 {
	return k.Low
}

// OpenPrice returns the open price of a kline
func (k *Kline) OpenPrice() float64 {
	return k.Open
}
