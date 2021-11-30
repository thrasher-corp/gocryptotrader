package kline

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
)

// Kline holds kline data and an event to be processed as
// a common.DataEventHandler type
type Kline struct {
	event.Base
	Open             decimal.Decimal
	Close            decimal.Decimal
	Low              decimal.Decimal
	High             decimal.Decimal
	Volume           decimal.Decimal
	ValidationIssues string
	FuturesData      *FuturesData
}

type FuturesData struct {
	Time          time.Time
	MarkPrice     decimal.Decimal
	PrevMarkPrice decimal.Decimal
}
