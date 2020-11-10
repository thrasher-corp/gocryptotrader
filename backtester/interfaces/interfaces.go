package interfaces

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// DataHandler interface for Loading and Streaming data
type DataHandler interface {
	DataLoader
	DataStreamer
	Reset()
}

// DataLoader interface for Loading data into backtest supported format
type DataLoader interface {
	Load() error
}

// DataStreamer interface handles loading, parsing, distributing BackTest data
type DataStreamer interface {
	Next() (DataEventHandler, bool)
	GetStream() []DataEventHandler
	History() []DataEventHandler
	Latest() DataEventHandler
	List() []DataEventHandler
	Offset() int

	StreamOpen() []float64
	StreamHigh() []float64
	StreamLow() []float64
	StreamClose() []float64
	StreamVol() []float64
}

// EventHandler interface implements required GetTime() & Pair() return
type EventHandler interface {
	IsEvent() bool
	GetTime() time.Time
	Pair() currency.Pair
	GetExchange() string
	GetAssetType() asset.Item
}

// DataHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	DataType() DataType
	Price() float64
}

type DataType uint8

// Directioner dictates the side of an order
type Directioner interface {
	SetDirection(side order.Side)
	GetDirection() order.Side
}
