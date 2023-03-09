package binance

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
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
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate           *rate.Limiter
	SpotOrdersRate     *rate.Limiter
	UFuturesRate       *rate.Limiter
	UFuturesOrdersRate *rate.Limiter
	CFuturesRate       *rate.Limiter
	CFuturesOrdersRate *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, int, error) {
	switch f {
	case spotDefaultRate:
		return r.SpotRate, 1, nil
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate:
		return r.SpotRate, 2, nil
	case spotHistoricalTradesRate,
		spotOrderbookDepth500Rate:
		return r.SpotRate, 5, nil
	case spotOrderbookDepth1000Rate,
		spotAccountInformationRate,
		spotExchangeInfo:
		return r.SpotRate, 10, nil
	case spotPriceChangeAllRate:
		return r.SpotRate, 40, nil
	case spotOrderbookDepth5000Rate:
		return r.SpotRate, 50, nil
	case spotOrderRate:
		return r.SpotOrdersRate, 1, nil
	case spotOrderQueryRate:
		return r.SpotOrdersRate, 2, nil
	case spotOpenOrdersSpecificRate:
		return r.SpotOrdersRate, 3, nil
	case spotAllOrdersRate:
		return r.SpotOrdersRate, 10, nil
	case spotOpenOrdersAllRate:
		return r.SpotOrdersRate, 40, nil
	case uFuturesDefaultRate,
		uFuturesKline100Rate:
		return r.UFuturesRate, 1, nil
	case uFuturesOrderbook50Rate,
		uFuturesKline500Rate,
		uFuturesOrderbookTickerAllRate:
		return r.UFuturesRate, 2, nil
	case uFuturesOrderbook100Rate,
		uFuturesKline1000Rate,
		uFuturesAccountInformationRate:
		return r.UFuturesRate, 5, nil
	case uFuturesOrderbook500Rate,
		uFuturesKlineMaxRate:
		return r.UFuturesRate, 10, nil
	case uFuturesOrderbook1000Rate,
		uFuturesHistoricalTradesRate:
		return r.UFuturesRate, 20, nil
	case uFuturesTickerPriceHistoryRate:
		return r.UFuturesRate, 40, nil
	case uFuturesOrdersDefaultRate:
		return r.UFuturesOrdersRate, 1, nil
	case uFuturesBatchOrdersRate,
		uFuturesGetAllOrdersRate:
		return r.UFuturesOrdersRate, 5, nil
	case uFuturesCountdownCancelRate:
		return r.UFuturesOrdersRate, 10, nil
	case uFuturesCurrencyForceOrdersRate,
		uFuturesSymbolOrdersRate:
		return r.UFuturesOrdersRate, 20, nil
	case uFuturesIncomeHistoryRate:
		return r.UFuturesOrdersRate, 30, nil
	case uFuturesPairOrdersRate,
		uFuturesGetAllOpenOrdersRate:
		return r.UFuturesOrdersRate, 40, nil
	case uFuturesAllForceOrdersRate:
		return r.UFuturesOrdersRate, 50, nil
	case cFuturesKline100Rate:
		return r.CFuturesRate, 1, nil
	case cFuturesKline500Rate,
		cFuturesOrderbookTickerAllRate:
		return r.CFuturesRate, 2, nil
	case cFuturesKline1000Rate,
		cFuturesAccountInformationRate:
		return r.CFuturesRate, 5, nil
	case cFuturesKlineMaxRate,
		cFuturesIndexMarkPriceRate:
		return r.CFuturesRate, 10, nil
	case cFuturesHistoricalTradesRate,
		cFuturesCurrencyForceOrdersRate:
		return r.CFuturesRate, 20, nil
	case cFuturesTickerPriceHistoryRate:
		return r.CFuturesRate, 40, nil
	case cFuturesAllForceOrdersRate:
		return r.CFuturesRate, 50, nil
	case cFuturesOrdersDefaultRate:
		return r.CFuturesOrdersRate, 1, nil
	case cFuturesBatchOrdersRate,
		cFuturesGetAllOpenOrdersRate:
		return r.CFuturesOrdersRate, 5, nil
	case cFuturesCancelAllOrdersRate:
		return r.CFuturesOrdersRate, 10, nil
	case cFuturesIncomeHistoryRate,
		cFuturesSymbolOrdersRate:
		return r.CFuturesOrdersRate, 20, nil
	case cFuturesPairOrdersRate:
		return r.CFuturesOrdersRate, 40, nil
	case cFuturesOrderbook50Rate:
		return r.CFuturesRate, 2, nil
	case cFuturesOrderbook100Rate:
		return r.CFuturesRate, 5, nil
	case cFuturesOrderbook500Rate:
		return r.CFuturesRate, 10, nil
	case cFuturesOrderbook1000Rate:
		return r.CFuturesRate, 20, nil
	case cFuturesDefaultRate:
		return r.CFuturesRate, 1, nil
	default:
		return r.SpotRate, 1, nil
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		SpotRate:           request.NewRateLimit(spotInterval, spotRequestRate),
		SpotOrdersRate:     request.NewRateLimit(spotOrderInterval, spotOrderRequestRate),
		UFuturesRate:       request.NewRateLimit(uFuturesInterval, uFuturesRequestRate),
		UFuturesOrdersRate: request.NewRateLimit(uFuturesOrderInterval, uFuturesOrderRequestRate),
		CFuturesRate:       request.NewRateLimit(cFuturesInterval, cFuturesRequestRate),
		CFuturesOrdersRate: request.NewRateLimit(cFuturesOrderInterval, cFuturesOrderRequestRate),
	}
}

func bestPriceLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return spotOrderbookTickerAllRate
	}

	return spotDefaultRate
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
