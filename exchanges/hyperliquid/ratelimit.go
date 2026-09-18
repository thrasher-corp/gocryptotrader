package hyperliquid

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	infoStandardEPL request.EndpointLimit = iota + 1
	infoLightEPL
	infoRecentTradesEPL
	infoUserRoleEPL
	infoHistoricalOrdersEPL
	infoFundingHistoryEPL
	infoUserLedgerEPL
	candleEPLBase         request.EndpointLimit = 100
	exchangeActionEPLBase request.EndpointLimit = 200

	restRateLimitInterval   = time.Minute
	restRateLimitWeight     = 1200
	standardInfoWeight      = 20
	lightInfoWeight         = 2
	itemsPerExtraInfoWeight = 20
	// The endpoint currently returns 10 trades, consuming one additional weight bucket per the API's 20-item rule.
	recentTradesWeight     = standardInfoWeight + 1
	candleBaseWeight       = 20
	candlesPerExtraWeight  = 60
	maximumCandleBuckets   = (maximumCandleCount + candlesPerExtraWeight - 1) / candlesPerExtraWeight
	userRoleWeight         = 60
	historicalOrdersWeight = 120
	fundingHistoryWeight   = 45
	// The rate-limit table calls this endpoint nonUserFundingUpdates; the
	// corresponding Info request is userNonFundingLedgerUpdates.
	userLedgerHistoryWeight = standardInfoWeight +
		(maximumUserLedgerHistoryCount+itemsPerExtraInfoWeight-1)/itemsPerExtraInfoWeight
	maximumActionBatchSize = 1000
	actionBatchWeightSize  = 40
)

// GetRateLimits returns Hyperliquid's aggregate weighted REST rate limits.
func GetRateLimits() request.RateLimitDefinitions {
	limiter := request.NewRateLimit(restRateLimitInterval, restRateLimitWeight)
	limits := request.RateLimitDefinitions{
		infoStandardEPL:         request.GetRateLimiterWithWeight(limiter, standardInfoWeight),
		infoLightEPL:            request.GetRateLimiterWithWeight(limiter, lightInfoWeight),
		infoRecentTradesEPL:     request.GetRateLimiterWithWeight(limiter, recentTradesWeight),
		infoUserRoleEPL:         request.GetRateLimiterWithWeight(limiter, userRoleWeight),
		infoHistoricalOrdersEPL: request.GetRateLimiterWithWeight(limiter, historicalOrdersWeight),
		infoFundingHistoryEPL:   request.GetRateLimiterWithWeight(limiter, fundingHistoryWeight),
		infoUserLedgerEPL:       request.GetRateLimiterWithWeight(limiter, userLedgerHistoryWeight),
	}
	for extraWeight := request.Weight(1); extraWeight <= maximumCandleBuckets; extraWeight++ {
		limits[candleEPLBase+request.EndpointLimit(extraWeight)] = request.GetRateLimiterWithWeight(limiter, request.Weight(candleBaseWeight)+extraWeight)
	}
	for extraWeight := 0; extraWeight <= maximumActionBatchSize/actionBatchWeightSize; extraWeight++ {
		limits[exchangeActionEPLBase+request.EndpointLimit(extraWeight)] = request.GetRateLimiterWithWeight(limiter, request.Weight(1+extraWeight)) //nolint:gosec // The bounded maximum is 25.
	}
	return limits
}

func candleEndpointLimit(count uint64) request.EndpointLimit {
	if count == 0 {
		count = 1
	}
	if count > maximumCandleCount {
		count = maximumCandleCount
	}
	endpoint := candleEPLBase + 1
	for threshold := uint64(candlesPerExtraWeight); count > threshold; threshold += candlesPerExtraWeight {
		endpoint++
	}
	return endpoint
}

func exchangeActionEndpointLimit(batchLength int) request.EndpointLimit {
	if batchLength < 1 {
		batchLength = 1
	}
	if batchLength > maximumActionBatchSize {
		batchLength = maximumActionBatchSize
	}
	return exchangeActionEPLBase + request.EndpointLimit(batchLength/actionBatchWeightSize) //nolint:gosec // batchLength is clamped to 1,000.
}

func formatInterval(interval kline.Interval) (string, error) {
	switch interval {
	case kline.OneMin:
		return "1m", nil
	case kline.ThreeMin:
		return "3m", nil
	case kline.FiveMin:
		return "5m", nil
	case kline.FifteenMin:
		return "15m", nil
	case kline.ThirtyMin:
		return "30m", nil
	case kline.OneHour:
		return "1h", nil
	case kline.TwoHour:
		return "2h", nil
	case kline.FourHour:
		return "4h", nil
	case kline.EightHour:
		return "8h", nil
	case kline.TwelveHour:
		return "12h", nil
	case kline.OneDay:
		return "1d", nil
	case kline.ThreeDay:
		return "3d", nil
	case kline.OneWeek:
		return "1w", nil
	case kline.OneMonth:
		return "1M", nil
	default:
		return "", fmt.Errorf("%w: %s", kline.ErrUnsupportedInterval, interval)
	}
}
