package config

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

var (
	errNoCurrencySettings               = errors.New("no currency settings set in the config")
	errBadInitialFunds                  = errors.New("initial funds set with invalid data, please check your config")
	errUnsetExchange                    = errors.New("exchange name unset for currency settings, please check your config")
	errUnsetCurrency                    = errors.New("currency unset for currency settings, please check your config")
	errBadSlippageRates                 = errors.New("invalid slippage rates in currency settings, please check your config")
	errSimultaneousProcessingRequired   = errors.New("exchange level funding requires simultaneous processing, please check your config and view funding readme for details")
	errExchangeLevelFundingRequired     = errors.New("invalid config, funding details set while exchange level funding is disabled")
	errExchangeLevelFundingDataRequired = errors.New("invalid config, exchange level funding enabled with no funding data set")
	errSizeLessThanZero                 = errors.New("size less than zero")
	errMaxSizeMinSizeMismatch           = errors.New("maximum size must be greater to minimum size")
	errMinMaxEqual                      = errors.New("minimum and maximum limits cannot be equal")
	errPerpetualsUnsupported            = errors.New("perpetual futures not yet supported")
	errFeatureIncompatible              = errors.New("feature is not compatible")
)

// Config defines what is in an individual strategy config
type Config struct {
	Nickname          string             `json:"nickname"`
	Goal              string             `json:"goal"`
	StrategySettings  StrategySettings   `json:"strategy-settings"`
	FundingSettings   FundingSettings    `json:"funding-settings"`
	CurrencySettings  []CurrencySettings `json:"currency-settings"`
	DataSettings      DataSettings       `json:"data-settings"`
	PortfolioSettings PortfolioSettings  `json:"portfolio-settings"`
	StatisticSettings StatisticSettings  `json:"statistic-settings"`
}

// DataSettings is a container for each type of data retrieval setting.
// Only ONE can be populated per config
type DataSettings struct {
	Interval                kline.Interval `json:"interval"`
	DataType                string         `json:"data-type"`
	VerboseExchangeRequests bool           `json:"verbose-exchange-requests"`
	APIData                 *APIData       `json:"api-data,omitempty"`
	DatabaseData            *DatabaseData  `json:"database-data,omitempty"`
	LiveData                *LiveData      `json:"live-data,omitempty"`
	CSVData                 *CSVData       `json:"csv-data,omitempty"`
}

// FundingSettings contains funding details for individual currencies
type FundingSettings struct {
	UseExchangeLevelFunding bool                   `json:"use-exchange-level-funding"`
	ExchangeLevelFunding    []ExchangeLevelFunding `json:"exchange-level-funding,omitempty"`
}

// StrategySettings contains what strategy to load, along with custom settings map
// (variables defined per strategy)
// along with defining whether the strategy will assess all currencies at once, or individually
type StrategySettings struct {
	Name                         string `json:"name"`
	SimultaneousSignalProcessing bool   `json:"use-simultaneous-signal-processing"`

	// If true, won't track USD values against currency pair
	// bool language is opposite to encourage use by default
	DisableUSDTracking bool           `json:"disable-usd-tracking"`
	CustomSettings     map[string]any `json:"custom-settings,omitempty"`
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
	Asset        asset.Item      `json:"asset"`
	Currency     currency.Code   `json:"currency"`
	InitialFunds decimal.Decimal `json:"initial-funds"`
	TransferFee  decimal.Decimal `json:"transfer-fee"`
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
	// MaximumOrderLeverageRate allows for orders to be placed with higher leverage rate. eg have $100 in collateral,
	// but place an order for $200 using 2x leverage
	MaximumOrderLeverageRate decimal.Decimal `json:"maximum-leverage-rate"`
	// MaximumCollateralLeverageRate allows for orders to be placed at `1x leverage, but utilise collateral as leverage to place more.
	// eg if this is 2x, and collateral is $100 I can place two long/shorts of $100
	MaximumCollateralLeverageRate decimal.Decimal `json:"maximum-collateral-leverage-rate"`
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
	ExchangeName string        `json:"exchange-name"`
	Asset        asset.Item    `json:"asset"`
	Base         currency.Code `json:"base"`
	Quote        currency.Code `json:"quote"`
	// USDTrackingPair is used for price tracking data only
	USDTrackingPair bool `json:"-"`

	SpotDetails    *SpotDetails    `json:"spot-details,omitempty"`
	FuturesDetails *FuturesDetails `json:"futures-details,omitempty"`

	BuySide  MinMax `json:"buy-side"`
	SellSide MinMax `json:"sell-side"`

	MinimumSlippagePercent decimal.Decimal `json:"min-slippage-percent"`
	MaximumSlippagePercent decimal.Decimal `json:"max-slippage-percent"`

	UsingExchangeMakerFee bool             `json:"-"`
	MakerFee              *decimal.Decimal `json:"maker-fee-override,omitempty"`
	UsingExchangeTakerFee bool             `json:"-"`
	TakerFee              *decimal.Decimal `json:"taker-fee-override,omitempty"`

	MaximumHoldingsRatio    decimal.Decimal `json:"maximum-holdings-ratio"`
	SkipCandleVolumeFitting bool            `json:"skip-candle-volume-fitting"`

	CanUseExchangeLimits          bool `json:"use-exchange-order-limits"`
	ShowExchangeOrderLimitWarning bool `json:"-"`
	UseExchangePNLCalculation     bool `json:"use-exchange-pnl-calculation"`
}

// SpotDetails contains funding information that cannot be shared with another
// pair during the backtesting run. Use exchange level funding to share funds
type SpotDetails struct {
	InitialBaseFunds  *decimal.Decimal `json:"initial-base-funds,omitempty"`
	InitialQuoteFunds *decimal.Decimal `json:"initial-quote-funds,omitempty"`
}

// FuturesDetails contains data relevant to futures currency pairs
type FuturesDetails struct {
	Leverage Leverage `json:"leverage"`
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
	NewEventTimeout           time.Duration `json:"new-event-timeout"`
	DataCheckTimer            time.Duration `json:"data-check-timer"`
	RealOrders                bool          `json:"real-orders"`
	ClosePositionsOnStop      bool          `json:"close-positions-on-stop"`
	DataRequestRetryTolerance int64         `json:"data-request-retry-tolerance"`
	DataRequestRetryWaitTime  time.Duration `json:"data-request-retry-wait-time"`
	ExchangeCredentials       []Credentials `json:"exchange-credentials"`
}

// Credentials holds each exchanges credentials
type Credentials struct {
	Exchange string               `json:"exchange"`
	Keys     accounts.Credentials `json:"credentials"`
}
