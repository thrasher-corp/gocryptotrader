package candle

import "time"

// Candle generic candle holder for modelPSQL & modelSQLite
type Candle struct {
	ID         string
	ExchangeID string
	Base       string
	Quote      string
	Interval   string
	Asset      string
	Tick       []Tick
}

// Tick holds each interval
type Tick struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}
