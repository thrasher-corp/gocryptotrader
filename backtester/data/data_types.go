package data

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

const (
	DataTypeCandle interfaces.DataType = iota
	DataTypeTick
)

type Data struct {
	latest interfaces.DataEventHandler
	stream []interfaces.DataEventHandler

	offset int
}
