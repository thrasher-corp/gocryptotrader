package config

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/database"
)

// Errors for config validation
var (
	errBadDate                          = errors.New("start date >= end date, please check your config")
	errNoCurrencySettings               = errors.New("no currency settings set in the config")
	errBadInitialFunds                  = errors.New("initial funds set with invalid data, please check your config")
	errUnsetExchange                    = errors.New("exchange name unset for currency settings, please check your config")
	errUnsetAsset                       = errors.New("asset unset for currency settings, please check your config")
	errUnsetCurrency                    = errors.New("currency unset for currency settings, please check your config")
	errBadSlippageRates                 = errors.New("invalid slippage rates in currency settings, please check your config")
	errStartEndUnset                    = errors.New("data start and end dates are invalid, please check your config")
	errSimultaneousProcessingRequired   = errors.New("exchange level funding requires simultaneous processing, please check your config and view funding readme for details")
	errExchangeLevelFundingRequired     = errors.New("invalid config, funding details set while exchange level funding is disabled")
	errExchangeLevelFundingDataRequired = errors.New("invalid config, exchange level funding enabled with no funding data set")
	errSizeLessThanZero                 = errors.New("size less than zero")
	errMaxSizeMinSizeMismatch           = errors.New("maximum size must be greater to minimum size")
	errMinMaxEqual                      = errors.New("minimum and maximum limits cannot be equal")
)

// Config defines what is in an individual strategy config
type Config struct {
	Nickname          string             `json:"nickname"`
	Goal              string             `json:"goal"`
	StrategySettings  StrategySettings   `json:"strategy-settings"`
	CurrencySettings  []CurrencySettings `json:"currency-settings"`
	DataSettings      DataSettings       `json:"data-settings"`
	PortfolioSettings PortfolioSettings  `json:"portfolio-settings"`
	StatisticSettings StatisticSettings  `json:"statistic-settings"`
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
	UseExchangeLevelFunding      bool                   `json:"use-exchange-level-funding"`
	ExchangeLevelFunding         []ExchangeLevelFunding `json:"exchange-level-funding,omitempty"`
	// If true, won't track USD values against currency pair
	// bool language is opposite to encourage use by default
	DisableUSDTracking bool                   `json:"disable-usd-tracking"`
	CustomSettings     map[string]interface{} `json:"custom-settings,omitempty"`
}

// ExchangeLevelFunding allows the portfolio manager to access
// a shared pool. For example, The base currencies BTC and LTC can both
// access the same USDT funding to make purchasing decisions
// Similarly, when a BTC is sold, LTC can now utilise the increased funding
// Importantly, exchange level funding is all-inclusive, you cannot have it for only some uses
// It also is required to use SimultaneousSignalProcessing, otherwise the first currency processed
// will have dibs
type ExchangeLevelFunding struct {
	ExchangeName string          `json:"exchange-name"`
	Asset        string          `json:"asset"`
	Currency     string          `json:"currency"`
	InitialFunds decimal.Decimal `json:"initial-funds"`
	TransferFee  decimal.Decimal `json:"transfer-fee"`
	Collateral   bool            `json:"collateral"`
}

// StatisticSettings adjusts ratios where
// proper data is currently lacking
type StatisticSettings struct {
	RiskFreeRate decimal.Decimal `json:"risk-free-rate"`
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
	CanUseLeverage                 bool            `json:"can-use-leverage"`
	MaximumOrdersWithLeverageRatio decimal.Decimal `json:"maximum-orders-with-leverage-ratio"`
	MaximumLeverageRate            decimal.Decimal `json:"maximum-leverage-rate"`
}

// MinMax are the rules which limit the placement of orders.
type MinMax struct {
	MinimumSize  decimal.Decimal `json:"minimum-size"` // will not place an order if under this amount
	MaximumSize  decimal.Decimal `json:"maximum-size"` // can only place an order up to this amount
	MaximumTotal decimal.Decimal `json:"maximum-total"`
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
	// USDTrackingPair is used for price tracking data only
	USDTrackingPair bool `json:"-"`

	InitialBaseFunds   *decimal.Decimal `json:"initial-base-funds,omitempty"`
	InitialQuoteFunds  *decimal.Decimal `json:"initial-quote-funds,omitempty"`
	InitialLegacyFunds float64          `json:"initial-funds,omitempty"`

	Leverage Leverage `json:"leverage"`
	BuySide  MinMax   `json:"buy-side"`
	SellSide MinMax   `json:"sell-side"`

	MinimumSlippagePercent decimal.Decimal `json:"min-slippage-percent"`
	MaximumSlippagePercent decimal.Decimal `json:"max-slippage-percent"`

	MakerFee decimal.Decimal `json:"maker-fee-override"`
	TakerFee decimal.Decimal `json:"taker-fee-override"`

	MaximumHoldingsRatio decimal.Decimal `json:"maximum-holdings-ratio"`

	CanUseExchangeLimits          bool `json:"use-exchange-order-limits"`
	SkipCandleVolumeFitting       bool `json:"skip-candle-volume-fitting"`
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
	StartDate        time.Time       `json:"start-date"`
	EndDate          time.Time       `json:"end-date"`
	Config           database.Config `json:"config"`
	Path             string          `json:"path"`
	InclusiveEndDate bool            `json:"inclusive-end-date"`
}

// LiveData defines all fields to configure live data
type LiveData struct {
	APIKeyOverride        string `json:"api-key-override"`
	APISecretOverride     string `json:"api-secret-override"`
	APIClientIDOverride   string `json:"api-client-id-override"`
	API2FAOverride        string `json:"api-2fa-override"`
	APISubAccountOverride string `json:"api-sub-account-override"`
	RealOrders            bool   `json:"real-orders"`
}
