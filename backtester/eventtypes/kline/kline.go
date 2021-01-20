package kline

// Price returns the closing price of a kline
func (k *Kline) Price() float64 {
	return k.Close
}
