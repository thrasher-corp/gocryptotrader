package data

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ErrHandlerNotFound returned when a handler is not found for specified exchange, asset, pair
var ErrHandlerNotFound = errors.New("handler not found")

// HandlerPerCurrency stores an event handler per exchange asset pair
type HandlerPerCurrency struct {
	data map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler
}

// Holder interface dictates what a data holder is expected to do
type Holder interface {
	Setup()
	SetDataForCurrency(string, asset.Item, currency.Pair, Handler)
	GetAllData() map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]Handler
	GetDataForCurrency(ev common.Event) (Handler, error)
	Reset()
}

// Base is the base implementation of some interface functions
// where further specific functions are implemented in DataFromKline
type Base struct {
	latest     Event
	stream     []Event
	offset     int64
	isLiveData bool
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
	AppendStream(s ...Event)
}

// Streamer interface handles loading, parsing, distributing BackTest data
type Streamer interface {
	Next() Event
	GetStream() []Event
	History() []Event
	Latest() Event
	List() []Event
	IsLastEvent() bool
	Offset() int64

	StreamOpen() []decimal.Decimal
	StreamHigh() []decimal.Decimal
	StreamLow() []decimal.Decimal
	StreamClose() []decimal.Decimal
	StreamVol() []decimal.Decimal

	HasDataAtTime(time.Time) bool
}

// Event interface used for loading and interacting with Data
type Event interface {
	common.Event
	GetUnderlyingPair() currency.Pair
	GetClosePrice() decimal.Decimal
	GetHighPrice() decimal.Decimal
	GetLowPrice() decimal.Decimal
	GetOpenPrice() decimal.Decimal
	GetVolume() decimal.Decimal
}
