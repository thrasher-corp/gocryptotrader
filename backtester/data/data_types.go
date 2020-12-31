package data

import (
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

const (
	CandleType common.DataType = iota
)

type DataHolder struct {
	Data map[string]map[asset.Item]map[currency.Pair]Handler
}

func (d *DataHolder) Setup() {
	if d.Data == nil {
		d.Data = make(map[string]map[asset.Item]map[currency.Pair]Handler)
	}
}

func (d *DataHolder) AddDataForCurrency(e string, a asset.Item, p currency.Pair, k Handler) {
	e = strings.ToLower(e)
	if d.Data[e] == nil {
		d.Data[e] = make(map[asset.Item]map[currency.Pair]Handler)
	}
	if d.Data[e][a] == nil {
		d.Data[e][a] = make(map[currency.Pair]Handler)
	}
	d.Data[e][a][p] = k
}

func (d *DataHolder) GetAllData() map[string]map[asset.Item]map[currency.Pair]Handler {
	return d.Data
}

func (d *DataHolder) GetDataForCurrency(e string, a asset.Item, p currency.Pair) Handler {
	return d.Data[e][a][p]
}

func (d *DataHolder) Reset() {
	d.Data = nil
}

type Holder interface {
	Setup()
	AddDataForCurrency(string, asset.Item, currency.Pair, Handler)
	GetAllData() map[string]map[asset.Item]map[currency.Pair]Handler
	GetDataForCurrency(string, asset.Item, currency.Pair) Handler
	Reset()
}

type DataPerCurrency struct {
	Latest common.DataEventHandler
	Stream []common.DataEventHandler
}

type Data struct {
	latest common.DataEventHandler
	stream []common.DataEventHandler
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
	Next() (common.DataEventHandler, bool)
	GetStream() []common.DataEventHandler
	History() []common.DataEventHandler
	Latest() common.DataEventHandler
	List() []common.DataEventHandler
	Offset() int

	StreamOpen() []float64
	StreamHigh() []float64
	StreamLow() []float64
	StreamClose() []float64
	StreamVol() []float64

	HasDataAtTime(time.Time) bool
}
