package hyperliquid

import (
	"time"

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
	infoUserFillsEPL
	infoUserFundingEPL
	infoStakingHistoryEPL
	infoTWAPHistoryEPL
	candleEPLBase         request.EndpointLimit = 100
	exchangeActionEPLBase request.EndpointLimit = 200

	restRateLimitInterval   = time.Minute
	restRateLimitWeight     = 1200
	standardInfoWeight      = 20
	lightInfoWeight         = 2
	itemsPerExtraInfoWeight = 20
	// recentTrades returns 10 trades, which the per-item rule counts as one extra weight
	recentTradesWeight     = standardInfoWeight + 1
	candleBaseWeight       = 20
	candlesPerExtraWeight  = 60
	maximumCandleBuckets   = (maximumCandleCount + candlesPerExtraWeight - 1) / candlesPerExtraWeight
	userRoleWeight         = 60
	historicalOrdersWeight = standardInfoWeight + 2000/itemsPerExtraInfoWeight
	fundingHistoryWeight   = standardInfoWeight + maximumFundingHistoryCount/itemsPerExtraInfoWeight
	// The rate limit table names userNonFundingLedgerUpdates nonUserFundingUpdates
	userLedgerHistoryWeight = standardInfoWeight + maximumUserLedgerHistoryCount/itemsPerExtraInfoWeight
	userFillsWeight         = standardInfoWeight + maximumUserFillsCount/itemsPerExtraInfoWeight
	userFundingWeight       = standardInfoWeight + maximumUserFundingCount/itemsPerExtraInfoWeight
	// Staking history and rewards have no documented cap, so they reserve the weight of 2000 entries
	stakingHistoryWeight = standardInfoWeight + 2000/itemsPerExtraInfoWeight
	// TWAP history has no documented cap either; Hyperliquid has returned at most 2000 entries
	twapHistoryWeight = standardInfoWeight + 2000/itemsPerExtraInfoWeight
	// Hyperliquid documents no batch maximum; this local cap keeps every batch's weight within the rate limit table
	maximumActionBatchSize = 1000
	actionBatchWeightSize  = 40
)

// rateLimits holds Hyperliquid's aggregate IP-weighted REST rate limits. Hyperliquid applies them per IP address, not per
// Exchange instance, so every instance shares these limiters instead of building its own
var rateLimits = func() request.RateLimitDefinitions {
	limiter := request.NewRateLimit(restRateLimitInterval, restRateLimitWeight)
	limits := request.RateLimitDefinitions{
		infoStandardEPL:         request.GetRateLimiterWithWeight(limiter, standardInfoWeight),
		infoLightEPL:            request.GetRateLimiterWithWeight(limiter, lightInfoWeight),
		infoRecentTradesEPL:     request.GetRateLimiterWithWeight(limiter, recentTradesWeight),
		infoUserRoleEPL:         request.GetRateLimiterWithWeight(limiter, userRoleWeight),
		infoHistoricalOrdersEPL: request.GetRateLimiterWithWeight(limiter, historicalOrdersWeight),
		infoFundingHistoryEPL:   request.GetRateLimiterWithWeight(limiter, fundingHistoryWeight),
		infoUserLedgerEPL:       request.GetRateLimiterWithWeight(limiter, userLedgerHistoryWeight),
		infoUserFillsEPL:        request.GetRateLimiterWithWeight(limiter, userFillsWeight),
		infoUserFundingEPL:      request.GetRateLimiterWithWeight(limiter, userFundingWeight),
		infoStakingHistoryEPL:   request.GetRateLimiterWithWeight(limiter, stakingHistoryWeight),
		infoTWAPHistoryEPL:      request.GetRateLimiterWithWeight(limiter, twapHistoryWeight),
	}
	for extraWeight := request.Weight(1); extraWeight <= maximumCandleBuckets; extraWeight++ {
		limits[candleEPLBase+request.EndpointLimit(extraWeight)] = request.GetRateLimiterWithWeight(limiter, candleBaseWeight+extraWeight)
	}
	for extraWeight := range request.Weight(maximumActionBatchSize/actionBatchWeightSize + 1) {
		limits[exchangeActionEPLBase+request.EndpointLimit(extraWeight)] = request.GetRateLimiterWithWeight(limiter, 1+extraWeight)
	}
	return limits
}()

// candleEndpointLimit returns the rate limit of a candle snapshot, which weighs one extra per 60 candles returned
func candleEndpointLimit(count uint64) request.EndpointLimit {
	count = min(max(count, 1), maximumCandleCount)
	return candleEPLBase + request.EndpointLimit((count+candlesPerExtraWeight-1)/candlesPerExtraWeight)
}

// exchangeActionEndpointLimit returns the rate limit of an exchange action, which weighs 1 + floor(batch length / 40)
func exchangeActionEndpointLimit(batchLength uint64) request.EndpointLimit {
	batchLength = min(max(batchLength, 1), maximumActionBatchSize)
	return exchangeActionEPLBase + request.EndpointLimit(batchLength/actionBatchWeightSize)
}
