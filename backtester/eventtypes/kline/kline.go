package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
)

func (k *Kline) DataType() portfolio.DataType {
	return data.DataTypeCandle
}

func (k *Kline) Price() float64 {
	return k.Close
}
