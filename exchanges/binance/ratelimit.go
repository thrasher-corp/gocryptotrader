package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Binance limit rates
	// Global dictates the max rate limit for general request items which is
	// 1200 requests per minute
	spotInterval    = time.Minute
	spotRequestRate = 6000
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	spotOrderInterval        = 10 * time.Second
	spotOrderRequestRate     = 100
	cFuturesInterval         = time.Minute
	cFuturesRequestRate      = 2400
	cFuturesOrderInterval    = time.Minute
	cFuturesOrderRequestRate = 1200
	uFuturesInterval         = time.Minute
	uFuturesRequestRate      = 2400
	uFuturesOrderInterval    = time.Second * 10
	uFuturesOrderRequestRate = 300
)

// Binance Spot rate limits
const (
	spotDefaultRate request.EndpointLimit = iota
	spotExchangeInfo
	spotHistoricalTradesRate
	spotOrderbookDepth500Rate
	spotOrderbookDepth100Rate
	spotOrderbookDepth1000Rate
	spotOrderbookDepth5000Rate
	spotOrderbookTickerAllRate
	spotTicker1Rate
	spotTicker20Rate
	spotTicker100Rate
	spotTickerAllRate
	spotOpenOrdersAllRate
	spotOpenOrdersSpecificRate
	spotOrderRate
	spotOrderQueryRate
	spotAllOrdersRate
	spotAccountInformationRate
	uFuturesDefaultRate
	uFuturesHistoricalTradesRate
	uFuturesSymbolOrdersRate
	uFuturesPairOrdersRate
	uFuturesCurrencyForceOrdersRate
	uFuturesAllForceOrdersRate
	uFuturesIncomeHistoryRate
	uFuturesOrderbook50Rate
	uFuturesOrderbook100Rate
	uFuturesOrderbook500Rate
	uFuturesOrderbook1000Rate
	uFuturesKline100Rate
	uFuturesKline500Rate
	uFuturesKline1000Rate
	uFuturesKlineMaxRate
	uFuturesTickerPriceHistoryRate
	uFuturesOrdersDefaultRate
	uFuturesGetAllOrdersRate
	uFuturesAccountInformationRate
	uFuturesOrderbookTickerAllRate
	uFuturesCountdownCancelRate
	uFuturesBatchOrdersRate
	uFuturesGetAllOpenOrdersRate
	cFuturesDefaultRate
	cFuturesHistoricalTradesRate
	cFuturesTickerPriceHistoryRate
	cFuturesIncomeHistoryRate
	cFuturesOrderbook50Rate
	cFuturesOrderbook100Rate
	cFuturesOrderbook500Rate
	cFuturesOrderbook1000Rate
	cFuturesKline500Rate
	cFuturesKline1000Rate
	cFuturesKlineMaxRate
	cFuturesIndexMarkPriceRate
	cFuturesBatchOrdersRate
	cFuturesCancelAllOrdersRate
	cFuturesGetAllOpenOrdersRate
	cFuturesAllForceOrdersRate
	cFuturesCurrencyForceOrdersRate
	cFuturesPairOrdersRate
	cFuturesSymbolOrdersRate
	cFuturesAccountInformationRate
	cFuturesOrderbookTickerAllRate
	cFuturesOrdersDefaultRate
	uFuturesMultiAssetMarginRate
	uFuturesSetMultiAssetMarginRate
)

// GetRateLimits returns the rate limit for the exchange
func GetRateLimits() request.RateLimitDefinitions {
	spotDefaultLimiter := request.NewRateLimit(spotInterval, spotRequestRate)
	spotOrderLimiter := request.NewRateLimit(spotOrderInterval, spotOrderRequestRate)
	usdMarginedFuturesLimiter := request.NewRateLimit(uFuturesInterval, uFuturesRequestRate)
	usdMarginedFuturesOrdersLimiter := request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate)
	coinMarginedFuturesLimiter := request.NewRateLimit(cFuturesInterval, cFuturesRequestRate)
	coinMarginedFuturesOrdersLimiter := request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate)

	return request.RateLimitDefinitions{
		spotDefaultRate:                 request.GetRateLimiterWithWeight(spotDefaultLimiter, 1),
		spotOrderbookTickerAllRate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 2),
		spotHistoricalTradesRate:        request.GetRateLimiterWithWeight(spotDefaultLimiter, 5),
		spotOrderbookDepth100Rate:       request.GetRateLimiterWithWeight(spotDefaultLimiter, 5),
		spotOrderbookDepth500Rate:       request.GetRateLimiterWithWeight(spotDefaultLimiter, 25),
		spotOrderbookDepth1000Rate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 50),
		spotOrderbookDepth5000Rate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 250),
		spotAccountInformationRate:      request.GetRateLimiterWithWeight(spotDefaultLimiter, 10),
		spotExchangeInfo:                request.GetRateLimiterWithWeight(spotDefaultLimiter, 10),
		spotTicker1Rate:                 request.GetRateLimiterWithWeight(spotDefaultLimiter, 2),
		spotTicker20Rate:                request.GetRateLimiterWithWeight(spotDefaultLimiter, 2),
		spotTicker100Rate:               request.GetRateLimiterWithWeight(spotDefaultLimiter, 40),
		spotTickerAllRate:               request.GetRateLimiterWithWeight(spotDefaultLimiter, 80),
		spotOrderRate:                   request.GetRateLimiterWithWeight(spotOrderLimiter, 1),
		spotOrderQueryRate:              request.GetRateLimiterWithWeight(spotOrderLimiter, 2),
		spotOpenOrdersSpecificRate:      request.GetRateLimiterWithWeight(spotOrderLimiter, 3),
		spotAllOrdersRate:               request.GetRateLimiterWithWeight(spotOrderLimiter, 10),
		spotOpenOrdersAllRate:           request.GetRateLimiterWithWeight(spotOrderLimiter, 40),
		uFuturesDefaultRate:             request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 1),
		uFuturesKline100Rate:            request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 1),
		uFuturesOrderbook50Rate:         request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 2),
		uFuturesOrderbook100Rate:        request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 5),
		uFuturesOrderbook500Rate:        request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 10),
		uFuturesOrderbook1000Rate:       request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 20),
		uFuturesKline500Rate:            request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 2),
		uFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 2),
		uFuturesKline1000Rate:           request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 5),
		uFuturesAccountInformationRate:  request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 5),
		uFuturesKlineMaxRate:            request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 10),
		uFuturesHistoricalTradesRate:    request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 20),
		uFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 40),
		uFuturesOrdersDefaultRate:       request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 1),
		uFuturesBatchOrdersRate:         request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 5),
		uFuturesGetAllOrdersRate:        request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 5),
		uFuturesCountdownCancelRate:     request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 10),
		uFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 20),
		uFuturesSymbolOrdersRate:        request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 20),
		uFuturesIncomeHistoryRate:       request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 30),
		uFuturesPairOrdersRate:          request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 40),
		uFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 40),
		uFuturesAllForceOrdersRate:      request.GetRateLimiterWithWeight(usdMarginedFuturesOrdersLimiter, 50),
		cFuturesDefaultRate:             request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 1),
		cFuturesKline500Rate:            request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 2),
		cFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 2),
		cFuturesKline1000Rate:           request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 5),
		cFuturesAccountInformationRate:  request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 5),
		cFuturesKlineMaxRate:            request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 10),
		cFuturesIndexMarkPriceRate:      request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 10),
		cFuturesHistoricalTradesRate:    request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 20),
		cFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 20),
		cFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithWeight(coinMarginedFuturesLimiter, 40),
		cFuturesAllForceOrdersRate:      request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 50),
		cFuturesOrdersDefaultRate:       request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 1),
		cFuturesBatchOrdersRate:         request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesCancelAllOrdersRate:     request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 10),
		cFuturesIncomeHistoryRate:       request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 20),
		cFuturesSymbolOrdersRate:        request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 20),
		cFuturesPairOrdersRate:          request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 40),
		cFuturesOrderbook50Rate:         request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 2),
		cFuturesOrderbook100Rate:        request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesOrderbook500Rate:        request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 10),
		cFuturesOrderbook1000Rate:       request.GetRateLimiterWithWeight(coinMarginedFuturesOrdersLimiter, 20),
		uFuturesMultiAssetMarginRate:    request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 30),
		uFuturesSetMultiAssetMarginRate: request.GetRateLimiterWithWeight(usdMarginedFuturesLimiter, 1),
	}
}

func openOrdersLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOpenOrdersAllRate
	}

	return spotOpenOrdersSpecificRate
}

func orderbookLimit(depth int) request.EndpointLimit {
	switch {
	case depth <= 100:
		return spotOrderbookDepth100Rate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	}

	return spotOrderbookDepth5000Rate
}
