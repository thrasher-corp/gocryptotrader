package kline

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Candle struct {
	event.Event
	Open   float64
	Close  float64
	Low    float64
	High   float64
	Volume float64
}

type DataFromKline struct {
	Item kline.Item
	data.Data
}
