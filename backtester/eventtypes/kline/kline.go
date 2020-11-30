package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

func (k *Kline) DataType() interfaces.DataType {
	return data.CandleType
}

func (k *Kline) Price() float64 {
	return k.Close
}
