package config

import (
	"errors"
	"time"

	"github.com/thrasher-corp/gocryptotrader/database"
)

// Errors for config validation
var (
	ErrBadDate            = errors.New("start date >= end date, please check your config")
	ErrNoCurrencySettings = errors.New("no currency settings set in the config")
	ErrBadInitialFunds    = errors.New("initial funds set with invalid data, please check your config")
	ErrUnsetExchange      = errors.New("exchange name unset for currency settings, please check your config")
	ErrUnsetAsset         = errors.New("asset unset for currency settings, please check your config")
	ErrUnsetCurrency      = errors.New("currency unset for currency settings, please check your config")
	ErrBadSlippageRates   = errors.New("invalid slippage rates in currency settings, please check your config")
	ErrStartEndUnset      = errors.New("data start and end dates are invalid, please check your config")
)

// Config defines what is in an individual strategy config
type Config struct {
	Nickname                 string             `json:"nickname"`
	Goal                     string             `json:"goal"`
	StrategySettings         StrategySettings   `json:"strategy-settings"`
	CurrencySettings         []CurrencySettings `json:"currency-settings"`
	DataSettings             DataSettings       `json:"data-settings"`
	PortfolioSettings        PortfolioSettings  `json:"portfolio-settings"`
	StatisticSettings        StatisticSettings  `json:"statistic-settings"`
	GoCryptoTraderConfigPath string             `json:"gocryptotrader-config-path"`
}

// DataSettings is a container for each type of data retrieval setting.
// Only ONE can be populated per config
type DataSettings struct {
	Interval     time.Duration `json:"interval"`
	DataType     string        `json:"data-type"`
	APIData      *APIData      `json:"api-data,omitempty"`
	DatabaseData *DatabaseData `json:"database-data,omitempty"`
	LiveData     *LiveData     `json:"live-data,omitempty"`
	CSVData      *CSVData      `json:"csv-data,omitempty"`
}

// StrategySettings contains what strategy to load, along with custom settings map
// (variables defined per strategy)
// along with defining whether the strategy will assess all currencies at once, or individually
type StrategySettings struct {
	Name                         string                 `json:"name"`
	SimultaneousSignalProcessing bool                   `json:"use-simultaneous-signal-processing"`
	CustomSettings               map[string]interface{} `json:"custom-settings"`
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
	Leverage Leverage `json:"leverage"`
	BuySide  MinMax   `json:"buy-side"`
	SellSide MinMax   `json:"sell-side"`
}

// Leverage rules are used to allow or limit the use of leverage in orders
// when supported
type Leverage struct {
	CanUseLeverage                 bool    `json:"can-use-leverage"`
	MaximumOrdersWithLeverageRatio float64 `json:"maximum-orders-with-leverage-ratio"`
	MaximumLeverageRate            float64 `json:"maximum-leverage-rate"`
}

// MinMax are the rules which limit the placement of orders.
type MinMax struct {
	MinimumSize  float64 `json:"minimum-size"` // will not place an order if under this amount
	MaximumSize  float64 `json:"maximum-size"` // can only place an order up to this amount
	MaximumTotal float64 `json:"maximum-total"`
}

// CurrencySettings stores pair based variables
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

	MaximumHoldingsRatio float64 `json:"maximum-holdings-ratio"`

	CanUseExchangeLimits          bool `json:"use-exchange-order-limits"`
	ShowExchangeOrderLimitWarning bool `json:"-"`
}

// APIData defines all fields to configure API based data
type APIData struct {
	StartDate        time.Time `json:"start-date"`
	EndDate          time.Time `json:"end-date"`
	InclusiveEndDate bool      `json:"inclusive-end-date"`
}

// CSVData defines all fields to configure CSV based data
type CSVData struct {
	FullPath string `json:"full-path"`
}

// DatabaseData defines all fields to configure database based data
type DatabaseData struct {
	StartDate        time.Time        `json:"start-date"`
	EndDate          time.Time        `json:"end-date"`
	ConfigOverride   *database.Config `json:"config-override"`
	InclusiveEndDate bool             `json:"inclusive-end-date"`
}

// LiveData defines all fields to configure live data
type LiveData struct {
	APIKeyOverride        string `json:"api-key-override"`
	APISecretOverride     string `json:"api-secret-override"`
	APIClientIDOverride   string `json:"api-client-id-override"`
	API2FAOverride        string `json:"api-2fa-override"`
	APISubaccountOverride string `json:"api-subaccount-override"`
	RealOrders            bool   `json:"real-orders"`
}
