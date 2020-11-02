package datahandler

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
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
}

// DataHandler interface used for loading and interacting with Data
type DataEventHandler interface {
	EventHandler
	DataType() DataType
	LatestPrice() float64
}

type DataType uint8
