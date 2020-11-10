package config

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
)

// Config defines what is in an individual strategy config
type Config struct {
	StrategyToLoad   string           `json:"strategy"`
	ExchangeSettings ExchangeSettings `json:"exchange-settings"`
	CandleData       *CandleData      `json:"candle-data,omitempty"`
	DatabaseData     *DatabaseData    `json:"database-data,omitempty"`
	LiveData         *LiveData        `json:"live-data,omitempty"`
}

// ExchangeSettings stores pair based variables
type ExchangeSettings struct {
	Name             string  `json:"exchange-name"`
	Base             string  `json:"base"`
	Quote            string  `json:"quote"`
	Asset            string  `json:"asset"`
	InitialFunds     float64 `json:"initial-funds"`
	MinimumOrderSize float64 `json:"minimum-order-size"` // will not place an order if under this amount
	MaximumOrderSize float64 `json:"maximum-order-size"` // can only place an order up to this amount
	DefaultOrderSize float64 `json:"default-order-size"`
	MakerFee         float64 `json:"-"`
	TakerFee         float64 `json:"-"`
}

// CandleData defines candle based variables
type CandleData struct {
	StartDate time.Time     `json:"start-date"`
	EndDate   time.Time     `json:"end-date"`
	Interval  time.Duration `json:"interval"`
}

// DatabaseData defines the database settings to use for the strategy
type DatabaseData struct {
	DataType       string           `json:"data-type"`
	StartDate      time.Time        `json:"start-date"`
	EndDate        time.Time        `json:"end-date"`
	Interval       time.Duration    `json:"interval"`
	ConfigOverride *database.Config `json:"config-override"`
}

// LiveData defines the live settings to use for the strategy
type LiveData struct {
	DataType            string        `json:"data-type"`
	Interval            time.Duration `json:"interval"`
	APIKeyOverride      string        `json:"api-key-override"`
	APISecretOverride   string        `json:"api-secret-override"`
	APIClientIDOverride string        `json:"api-client-id-override"`
	API2FAOverride      string        `json:"api-2fa-override"`
	FakeOrders          bool          `json:"fake-orders"`
}
