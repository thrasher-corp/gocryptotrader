package poloniex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	rateInterval         = time.Second
	threeSecondsInterval = time.Second * 3
	tenSecondsInterval   = time.Second * 10

	unauthRate                   = 200
	authNonResourceIntensiveRate = 50
	authResourceIntensiveRate    = 10
	referenceDataRate            = 10

	// used with futures account endpoint calls.
	accountOverviewRate     = 3
	fTransactionHistoryRate = 9
	fOrderRate              = 30
	fCancelOrderRate        = 40
)

const (
	authNonResourceIntensiveEPL request.EndpointLimit = iota
	authResourceIntensiveEPL
	unauthEPL
	referenceDataEPL

	accountOverviewEPL
	fTransactionHistoryEPL
	fOrderEPL
	fCancelOrderEPL
	fCancelAllLimitOrdersEPL
	fCancelMultipleLimitOrdersEPL
	fCancelAllStopOrdersEPL
	fGetOrdersEPL
	fGetUntriggeredStopOrderEPL
	fGetCompleted24HrEPL
	fGetSingleOrderDetailEPL
	fGetFuturesOrdersV2EPL
	fGetFuturesFillsEPL
	fGetActiveOrderValueCalculationEPL
	fGetFillsV2EPL
	fGetFuturesPositionDetailsEPL
	fGetPositionListEPL
	fGetFundingRateEPL
)

// rateLimits returns the rate limit for the exchange
// As per https://docs.poloniex.com/#http-api
var rateLimits = request.RateLimitDefinitions{
	authNonResourceIntensiveEPL:        request.NewRateLimitWithWeight(rateInterval, authNonResourceIntensiveRate, 1),
	authResourceIntensiveEPL:           request.NewRateLimitWithWeight(rateInterval, authResourceIntensiveRate, 1),
	unauthEPL:                          request.NewRateLimitWithWeight(rateInterval, unauthRate, 1),
	referenceDataEPL:                   request.NewRateLimitWithWeight(rateInterval, referenceDataRate, 1),
	accountOverviewEPL:                 request.NewRateLimitWithWeight(rateInterval, accountOverviewRate, 1),
	fTransactionHistoryEPL:             request.NewRateLimitWithWeight(threeSecondsInterval, fTransactionHistoryRate, 1),
	fOrderEPL:                          request.NewRateLimitWithWeight(threeSecondsInterval, fOrderRate, 1),
	fCancelOrderEPL:                    request.NewRateLimitWithWeight(threeSecondsInterval, fCancelOrderRate, 1),
	fCancelAllLimitOrdersEPL:           request.NewRateLimitWithWeight(threeSecondsInterval, 1, 1),
	fCancelMultipleLimitOrdersEPL:      request.NewRateLimitWithWeight(threeSecondsInterval, 3, 1),
	fCancelAllStopOrdersEPL:            request.NewRateLimitWithWeight(tenSecondsInterval, 2, 1),
	fGetOrdersEPL:                      request.NewRateLimitWithWeight(threeSecondsInterval, 3, 1),
	fGetUntriggeredStopOrderEPL:        request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
	fGetCompleted24HrEPL:               request.NewRateLimitWithWeight(threeSecondsInterval, 3, 1),
	fGetSingleOrderDetailEPL:           request.NewRateLimitWithWeight(threeSecondsInterval, 40, 1),
	fGetFuturesOrdersV2EPL:             request.NewRateLimitWithWeight(threeSecondsInterval, 30, 1),
	fGetFuturesFillsEPL:                request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
	fGetActiveOrderValueCalculationEPL: request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
	fGetFillsV2EPL:                     request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
	fGetFuturesPositionDetailsEPL:      request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
	fGetPositionListEPL:                request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
	fGetFundingRateEPL:                 request.NewRateLimitWithWeight(threeSecondsInterval, 9, 1),
}
