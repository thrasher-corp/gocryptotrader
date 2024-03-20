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
	spotRequestRate = 1200
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	spotOrderInterval        = 10 * time.Second
	spotOrderRequestRate     = 100
	cFuturesInterval         = time.Minute
	cFuturesRequestRate      = 6000
	cFuturesOrderInterval    = time.Minute
	cFuturesOrderRequestRate = 1200
	uFuturesInterval         = time.Minute
	uFuturesRequestRate      = 2400
	uFuturesOrderInterval    = time.Minute
	uFuturesOrderRequestRate = 1200
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
	cFuturesKline100Rate
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

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	spotDefaultLimiter := request.NewRateLimit(spotInterval, spotRequestRate)
	spotOrderLimiter := request.NewRateLimit(spotOrderInterval, spotOrderRequestRate)
	usdMarginedFuturesLimiter := request.NewRateLimit(uFuturesInterval, uFuturesRequestRate)
	usdMarginedFuturesOrdersLimiter := request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate)
	coinMarginedFuturesLimiter := request.NewRateLimit(cFuturesInterval, cFuturesRequestRate)
	coinMarginedFuturesOrdersLimiter := request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate)

	return request.RateLimitDefinitions{
		spotDefaultRate:                 request.GetRateLimiterWithToken(spotDefaultLimiter, 1),
		spotOrderbookTickerAllRate:      request.GetRateLimiterWithToken(spotDefaultLimiter, 2),
		spotSymbolPriceAllRate:          request.GetRateLimiterWithToken(spotDefaultLimiter, 2),
		spotHistoricalTradesRate:        request.GetRateLimiterWithToken(spotDefaultLimiter, 5),
		spotOrderbookDepth500Rate:       request.GetRateLimiterWithToken(spotDefaultLimiter, 5),
		spotOrderbookDepth1000Rate:      request.GetRateLimiterWithToken(spotDefaultLimiter, 10),
		spotAccountInformationRate:      request.GetRateLimiterWithToken(spotDefaultLimiter, 10),
		spotExchangeInfo:                request.GetRateLimiterWithToken(spotDefaultLimiter, 10),
		spotPriceChangeAllRate:          request.GetRateLimiterWithToken(spotDefaultLimiter, 40),
		spotOrderbookDepth5000Rate:      request.GetRateLimiterWithToken(spotDefaultLimiter, 50),
		spotOrderRate:                   request.GetRateLimiterWithToken(spotOrderLimiter, 1),
		spotOrderQueryRate:              request.GetRateLimiterWithToken(spotOrderLimiter, 2),
		spotOpenOrdersSpecificRate:      request.GetRateLimiterWithToken(spotOrderLimiter, 3),
		spotAllOrdersRate:               request.GetRateLimiterWithToken(spotOrderLimiter, 10),
		spotOpenOrdersAllRate:           request.GetRateLimiterWithToken(spotOrderLimiter, 40),
		uFuturesDefaultRate:             request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 1),
		uFuturesKline100Rate:            request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 1),
		uFuturesOrderbook50Rate:         request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 2),
		uFuturesKline500Rate:            request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 2),
		uFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 2),
		uFuturesOrderbook100Rate:        request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 5),
		uFuturesKline1000Rate:           request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 5),
		uFuturesAccountInformationRate:  request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 5),
		uFuturesOrderbook500Rate:        request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 10),
		uFuturesKlineMaxRate:            request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 10),
		uFuturesOrderbook1000Rate:       request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 20),
		uFuturesHistoricalTradesRate:    request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 20),
		uFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 40),
		uFuturesOrdersDefaultRate:       request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 1),
		uFuturesBatchOrdersRate:         request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 5),
		uFuturesGetAllOrdersRate:        request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 5),
		uFuturesCountdownCancelRate:     request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 10),
		uFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 20),
		uFuturesSymbolOrdersRate:        request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 20),
		uFuturesIncomeHistoryRate:       request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 30),
		uFuturesPairOrdersRate:          request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 40),
		uFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 40),
		uFuturesAllForceOrdersRate:      request.GetRateLimiterWithToken(usdMarginedFuturesOrdersLimiter, 50),
		cFuturesDefaultRate:             request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 1),
		cFuturesKline500Rate:            request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 2),
		cFuturesOrderbookTickerAllRate:  request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 2),
		cFuturesKline1000Rate:           request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 5),
		cFuturesAccountInformationRate:  request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 5),
		cFuturesKlineMaxRate:            request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 10),
		cFuturesIndexMarkPriceRate:      request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 10),
		cFuturesHistoricalTradesRate:    request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 20),
		cFuturesCurrencyForceOrdersRate: request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 20),
		cFuturesTickerPriceHistoryRate:  request.GetRateLimiterWithToken(coinMarginedFuturesLimiter, 40),
		cFuturesAllForceOrdersRate:      request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 50),
		cFuturesOrdersDefaultRate:       request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 1),
		cFuturesBatchOrdersRate:         request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesGetAllOpenOrdersRate:    request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesCancelAllOrdersRate:     request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 10),
		cFuturesIncomeHistoryRate:       request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 20),
		cFuturesSymbolOrdersRate:        request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 20),
		cFuturesPairOrdersRate:          request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 40),
		cFuturesOrderbook50Rate:         request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 2),
		cFuturesOrderbook100Rate:        request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 5),
		cFuturesOrderbook500Rate:        request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 10),
		cFuturesOrderbook1000Rate:       request.GetRateLimiterWithToken(coinMarginedFuturesOrdersLimiter, 20),
		uFuturesMultiAssetMarginRate:    request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 30),
		uFuturesSetMultiAssetMarginRate: request.GetRateLimiterWithToken(usdMarginedFuturesLimiter, 1),
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
		return spotDefaultRate
	case depth <= 500:
		return spotOrderbookDepth500Rate
	case depth <= 1000:
		return spotOrderbookDepth1000Rate
	}

	return spotOrderbookDepth5000Rate
}
