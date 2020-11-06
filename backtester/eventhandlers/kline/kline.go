package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

func (k *Kline) DataType() portfolio.DataType {
	return data.DataTypeCandle
}

func (k *Kline) Price() float64 {
	return k.Close
}
