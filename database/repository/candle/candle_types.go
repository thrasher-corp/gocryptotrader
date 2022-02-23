package candle

import (
	"errors"
	"time"
)

var (
	errInvalidInput = errors.New("exchange, base, quote, asset, interval, start & end cannot be empty")
	errNoCandleData = errors.New("no candle data provided")
	// ErrNoCandleDataFound returns when no candle data is found
	ErrNoCandleDataFound = errors.New("no candle data found")
)

// Item generic candle holder for modelPSQL & modelSQLite
type Item struct {
	ID         string
	ExchangeID string
	Base       string
	Quote      string
	Interval   int64
	Asset      string
	Candles    []Candle
}

// Candle holds each interval
type Candle struct {
	Timestamp        time.Time
	Open             float64
	High             float64
	Low              float64
	Close            float64
	Volume           float64
	SourceJobID      string
	ValidationJobID  string
	ValidationIssues string
}
