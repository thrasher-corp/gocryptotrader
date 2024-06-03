package binanceus

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	spotInterval    = time.Minute
	spotRequestRate = 1200
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	spotOrderInterval    = 10 * time.Second
	spotOrderRequestRate = 100
)

// Binance Spot rate limits
const (
	spotDefaultRate request.EndpointLimit = iota
	spotExchangeInfo
	spotHistoricalTradesRate
	spotOrderbookDepth500Rate
	spotOrderbookDepth1000Rate
	spotOrderbookDepth5000Rate
	spotOrderbookTickerAllRate
	spotPriceChangeAllRate
	spotSymbolPriceAllRate
	spotSingleOCOOrderRate
	spotOpenOrdersAllRate
	spotOpenOrdersSpecificRate
	spotOrderRate
	spotOrderQueryRate
	spotTradesQueryRate
	spotAllOrdersRate
	spotAllOCOOrdersRate
	spotOrderRateLimitRate
	spotAccountInformationRate
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	spotRate := request.NewRateLimit(spotInterval, spotRequestRate)
	spotOrdersRate := request.NewRateLimit(spotOrderInterval, spotOrderRequestRate)
	return request.RateLimitDefinitions{
		spotDefaultRate:            request.GetRateLimiterWithWeight(spotRate, 1),
		spotOrderbookTickerAllRate: request.GetRateLimiterWithWeight(spotRate, 2),
		spotSymbolPriceAllRate:     request.GetRateLimiterWithWeight(spotRate, 2),
		spotHistoricalTradesRate:   request.GetRateLimiterWithWeight(spotRate, 5),
		spotOrderbookDepth500Rate:  request.GetRateLimiterWithWeight(spotRate, 5),
		spotOrderbookDepth1000Rate: request.GetRateLimiterWithWeight(spotRate, 10),
		spotAccountInformationRate: request.GetRateLimiterWithWeight(spotRate, 10),
		spotExchangeInfo:           request.GetRateLimiterWithWeight(spotRate, 10),
		spotTradesQueryRate:        request.GetRateLimiterWithWeight(spotRate, 10),
		spotPriceChangeAllRate:     request.GetRateLimiterWithWeight(spotRate, 40),
		spotOrderbookDepth5000Rate: request.GetRateLimiterWithWeight(spotRate, 50),
		spotOrderRate:              request.GetRateLimiterWithWeight(spotOrdersRate, 1),
		spotOrderQueryRate:         request.GetRateLimiterWithWeight(spotOrdersRate, 2),
		spotSingleOCOOrderRate:     request.GetRateLimiterWithWeight(spotOrdersRate, 2),
		spotOpenOrdersSpecificRate: request.GetRateLimiterWithWeight(spotOrdersRate, 3),
		spotAllOrdersRate:          request.GetRateLimiterWithWeight(spotOrdersRate, 10),
		spotAllOCOOrdersRate:       request.GetRateLimiterWithWeight(spotOrdersRate, 10),
		spotOrderRateLimitRate:     request.GetRateLimiterWithWeight(spotOrdersRate, 20),
		spotOpenOrdersAllRate:      request.GetRateLimiterWithWeight(spotOrdersRate, 40),
	}
}

// orderbookLimit returns the endpoint rate limit representing enum given order depth
func orderbookLimit(depth int64) request.EndpointLimit {
	switch {
	case depth <= 100:
		return spotDefaultRate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	}
	return spotOrderbookDepth5000Rate
}
