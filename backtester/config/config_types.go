package config

import "time"

// Config defines what is in an individual strategy config
type Config struct {
	StrategyToLoad string              `json:"strategy"`
	ExchangePairs  map[string]Currency `json:"exchange-pairs"`
	StartDate      time.Time           `json:"start-date"`
	EndDate        time.Time           `json:"end-date"`
	DataSource     string              `json:"data-source"`
	CandleData     *CandleData         `json:"candle-data,omitempty"`
	Funding        Funding             `json:"funding"`
}

// Funding determines how the strategy will trade
type Funding struct {
	InitialFunds     float64 `json:"initial-funds"`
	MaximumOrderSize float64 `json:"maximum-order-size"`
}

// CandleData defines candle based variables
type CandleData struct {
	Interval string `json:"interval"`
}

// Currency stores pair based variables
type Currency struct {
	Base     string  `json:"base"`
	Quote    string  `json:"quote"`
	Asset    string  `json:"asset"`
	MakerFee float64 `json:"-"`
	TakerFee float64 `json:"-""`
}
