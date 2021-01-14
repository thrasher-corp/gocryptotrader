package config

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
)

// Config defines what is in an individual strategy config
type Config struct {
	Nickname          string             `json:"nickname"` // this will override the strategy name in report output
	CurrencySettings  []CurrencySettings `json:"currency-settings"`
	StrategySettings  StrategySettings   `json:"strategy-settings"`
	PortfolioSettings PortfolioSettings  `json:"portfolio"`
	StatisticSettings StatisticSettings  `json:"statistic-settings"`
	// data source definitions:
	APIData      *APIData      `json:"api-data,omitempty"`
	DatabaseData *DatabaseData `json:"database-data,omitempty"`
	LiveData     *LiveData     `json:"live-data,omitempty"`
	CSVData      *CSVData      `json:"csv-data,omitempty"`
}

// StrategySettings contains what strategy to load, along with custom settings map
// (variables defined per strategy)
// along with defining whether the strategy will assess all currencies at once, or individually
type StrategySettings struct {
	Name            string                 `json:"name"`
	IsMultiCurrency bool                   `json:"is-multi-currency"`
	CustomSettings  map[string]interface{} `json:"custom-settings"`
}

// StatisticSettings holds configurable varialbes to adjust ratios where
// proper data is currently lacking
type StatisticSettings struct {
	RiskFreeRate float64 `json:"risk-free-rate"`
}

// PortfolioSettings act as a global protector for strategies
// these settings will override ExchangeSettings that go against it
// and assess the bigger picture
type PortfolioSettings struct {
	MaximumHoldingsRatio float64  `json:"diversification-ratio"`
	Leverage             Leverage `json:"leverage"`
	BuySide              MinMax   `json:"buy-side"`
	SellSide             MinMax   `json:"sell-side"`
}

type Leverage struct {
	CanUseLeverage  bool    `json:"can-use-leverage"`
	MaximumLeverage float64 `json:"maximum-leverage"`
}

type MinMax struct {
	MinimumSize  float64 `json:"minimum-size"` // will not place an order if under this amount
	MaximumSize  float64 `json:"maximum-size"` // can only place an order up to this amount
	MaximumTotal float64 `json:"maximum-total"`
}

// ExchangeSettings stores pair based variables
// It contains rules about the specific currency pair
// you wish to trade with
// Backtester will load the data of the currencies specified here
type CurrencySettings struct {
	ExchangeName string `json:"exchange-name"`
	Asset        string `json:"asset"`
	Base         string `json:"base"`
	Quote        string `json:"quote"`

	InitialFunds float64 `json:"initial-funds"`

	Leverage Leverage `json:"leverage"`
	BuySide  MinMax   `json:"buy-side"`
	SellSide MinMax   `json:"sell-side"`

	MinimumSlippagePercent float64 `json:"min-slippage-percent"`
	MaximumSlippagePercent float64 `json:"max-slippage-percent"`

	MakerFee float64 `json:"maker-fee-override"`
	TakerFee float64 `json:"taker-fee-override"`
}

// APIData defines all fields to configure API based data
type APIData struct {
	DataType  string        `json:"data-type"`
	Interval  time.Duration `json:"interval"`
	StartDate time.Time     `json:"start-date"`
	EndDate   time.Time     `json:"end-date"`
}

// CSVData defines all fields to configure CSV based data
type CSVData struct {
	DataType string        `json:"data-type"`
	Interval time.Duration `json:"interval"`
	FullPath string        `json:"full-path"`
}

// DatabaseData defines all fields to configure database based data
type DatabaseData struct {
	DataType       string           `json:"data-type"`
	Interval       time.Duration    `json:"interval"`
	StartDate      time.Time        `json:"start-date"`
	EndDate        time.Time        `json:"end-date"`
	ConfigOverride *database.Config `json:"config-override"`
}

// LiveData defines all fields to configure live data
type LiveData struct {
	Interval            time.Duration `json:"interval"`
	DataType            string        `json:"data-type"`
	APIKeyOverride      string        `json:"api-key-override"`
	APISecretOverride   string        `json:"api-secret-override"`
	APIClientIDOverride string        `json:"api-client-id-override"`
	API2FAOverride      string        `json:"api-2fa-override"`
	RealOrders          bool          `json:"fake-orders"`
}
