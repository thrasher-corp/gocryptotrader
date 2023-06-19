package fundingrates

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// ErrFundingRateOutsideLimits is returned when a funding rate is outside the allowed date range
var ErrFundingRateOutsideLimits = errors.New("funding rate outside limits")

// RatesRequest is used to request funding rate details for a position
type RatesRequest struct {
	Asset                     asset.Item
	Pair                      currency.Pair
	StartDate                 time.Time
	EndDate                   time.Time
	IncludePayments           bool
	IncludePredictedRate      bool
	AdhereToFundingRateLimits bool
}

// Rates is used to return funding rate details for a position
type Rates struct {
	Exchange              string
	Asset                 asset.Item
	Pair                  currency.Pair
	StartDate             time.Time
	EndDate               time.Time
	LatestRate            Rate
	PredictedUpcomingRate Rate
	FundingRates          []Rate
	PaymentSum            decimal.Decimal
}

// Rate holds details for an individual funding rate
type Rate struct {
	Time    time.Time
	Rate    decimal.Decimal
	Payment decimal.Decimal
}
