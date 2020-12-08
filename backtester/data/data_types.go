package data

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	CandleType interfaces.DataType = iota
)

type DataPerCurrency struct {
	Latest interfaces.DataEventHandler
	Stream []interfaces.DataEventHandler
}

type Data struct {
	latest interfaces.DataEventHandler
	stream []interfaces.DataEventHandler

	datas  map[string]map[asset.Item]map[currency.Pair]DataPerCurrency
	offset int
}

// Handler interface for Loading and Streaming data
type Handler interface {
	Loader
	Streamer
	Reset()
}

// Loader interface for Loading data into backtest supported format
type Loader interface {
	Load() error
}

// Streamer interface handles loading, parsing, distributing BackTest data
type Streamer interface {
	Next() (interfaces.DataEventHandler, bool)
	GetStream() []interfaces.DataEventHandler
	History() []interfaces.DataEventHandler
	Latest() interfaces.DataEventHandler
	List() []interfaces.DataEventHandler
	Offset() int

	StreamOpen() []float64
	StreamHigh() []float64
	StreamLow() []float64
	StreamClose() []float64
	StreamVol() []float64
}
