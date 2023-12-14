package fundingrate

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrFundingRateOutsideLimits is returned when a funding rate is outside the allowed date range
	ErrFundingRateOutsideLimits = errors.New("funding rate outside limits")
	// ErrPaymentCurrencyCannotBeEmpty is returned when a payment currency is not set
	ErrPaymentCurrencyCannotBeEmpty = errors.New("payment currency cannot be empty")
	// ErrNoFundingRatesFound is returned when no funding rates are found
	ErrNoFundingRatesFound = errors.New("no funding rates found")
)

// HistoricalRatesRequest is used to request funding rate details for a position
type HistoricalRatesRequest struct {
	Asset asset.Item
	Pair  currency.Pair
	// PaymentCurrency is an optional parameter depending on exchange API
	// if you are paid in a currency that isn't easily inferred from the Pair,
	// eg BTCUSD-PERP use this field
	PaymentCurrency      currency.Code
	StartDate            time.Time
	EndDate              time.Time
	IncludePayments      bool
	IncludePredictedRate bool
	// RespectHistoryLimits if an exchange has a limit on rate history lookup
	// and your start date is beyond that time, this will set your start date
	// to the maximum allowed date rather than give you errors
	RespectHistoryLimits bool
}

// HistoricalRates is used to return funding rate details for a position
type HistoricalRates struct {
	Exchange              string
	Asset                 asset.Item
	Pair                  currency.Pair
	StartDate             time.Time
	EndDate               time.Time
	LatestRate            Rate
	PredictedUpcomingRate Rate
	FundingRates          []Rate
	PaymentSum            decimal.Decimal
	PaymentCurrency       currency.Code
	TimeOfNextRate        time.Time
}

// LatestRateRequest is used to request the latest funding rate
type LatestRateRequest struct {
	Asset                asset.Item
	Pair                 currency.Pair
	IncludePredictedRate bool
}

// LatestRateResponse for when you just want the latest rate
type LatestRateResponse struct {
	Exchange              string
	Asset                 asset.Item
	Pair                  currency.Pair
	LatestRate            Rate
	PredictedUpcomingRate Rate
	TimeOfNextRate        time.Time
	TimeChecked           time.Time
}

// Rate holds details for an individual funding rate
type Rate struct {
	Time    time.Time
	Rate    decimal.Decimal
	Payment decimal.Decimal
}
