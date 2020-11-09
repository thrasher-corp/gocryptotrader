package config

import "time"

// Config defines what is in an individual strategy config
type Config struct {
	StrategyToLoad   string           `json:"strategy"`
	ExchangeSettings ExchangeSettings `json:"exchange-settings"`
	DataSource       string           `json:"data-source"`
	CandleData       *CandleData      `json:"candle-data,omitempty"`
}

// CandleData defines candle based variables
type CandleData struct {
	StartDate time.Time     `json:"start-date"`
	EndDate   time.Time     `json:"end-date"`
	Interval  time.Duration `json:"interval"`
}

// ExchangeSettings stores pair based variables
type ExchangeSettings struct {
	Name             string  `json:"exchange-name"`
	Base             string  `json:"base"`
	Quote            string  `json:"quote"`
	Asset            string  `json:"asset"`
	InitialFunds     float64 `json:"initial-funds"`
	MaximumOrderSize float64 `json:"maximum-order-size"`
	MakerFee         float64 `json:"-"`
	TakerFee         float64 `json:"-"`
}
