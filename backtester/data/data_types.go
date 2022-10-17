package data

import (
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrHandlerNotFound returned when a handler is not found for specified exchange, asset, pair
	ErrHandlerNotFound = errors.New("handler not found")

	errNothingToAdd         = errors.New("cannot append empty event to stream")
	errInvalidEventSupplied = errors.New("invalid event supplied")
	errInvalidOffset        = errors.New("event base set to invalid offset")
	errMisMatchedEvent      = errors.New("cannot add event to stream, does not match")
)

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
	Reset() error
}

// Base is the base implementation of some interface functions
// where further specific functions are implemented in DataFromKline
type Base struct {
	m          sync.Mutex
	latest     Event
	stream     []Event
	offset     int64
	isLiveData bool
}

// Handler interface for Loading and Streaming data
type Handler interface {
	Loader
	Streamer
	Reset() error
}

// Loader interface for Loading data into backtest supported format
type Loader interface {
	Load() error
	AppendStream(s ...Event) error
}

// Streamer interface handles loading, parsing, distributing BackTest data
type Streamer interface {
	Next() (Event, error)
	GetStream() ([]Event, error)
	History() ([]Event, error)
	Latest() (Event, error)
	List() ([]Event, error)
	IsLastEvent() (bool, error)
	Offset() (int64, error)

	StreamOpen() ([]decimal.Decimal, error)
	StreamHigh() ([]decimal.Decimal, error)
	StreamLow() ([]decimal.Decimal, error)
	StreamClose() ([]decimal.Decimal, error)
	StreamVol() ([]decimal.Decimal, error)

	HasDataAtTime(time.Time) (bool, error)
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
