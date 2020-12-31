package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
)

func (k *Kline) DataType() common.DataType {
	return data.CandleType
}

func (k *Kline) Price() float64 {
	return k.Close
}
