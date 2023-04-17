package binanceus

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
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

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	SpotRate       *rate.Limiter
	SpotOrdersRate *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, request.Tokens, error) {
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
		spotExchangeInfo,
		spotTradesQueryRate:
		return r.SpotRate, 10, nil
	case spotPriceChangeAllRate:
		return r.SpotRate, 40, nil
	case spotOrderbookDepth5000Rate:
		return r.SpotRate, 50, nil
	case spotOrderRate:
		return r.SpotOrdersRate, 1, nil
	case spotOrderQueryRate,
		spotSingleOCOOrderRate:
		return r.SpotOrdersRate, 2, nil
	case spotOpenOrdersSpecificRate:
		return r.SpotOrdersRate, 3, nil
	case spotAllOrdersRate,
		spotAllOCOOrdersRate:
		return r.SpotOrdersRate, 10, nil
	case spotOrderRateLimitRate:
		return r.SpotOrdersRate, 20, nil
	case spotOpenOrdersAllRate:
		return r.SpotOrdersRate, 40, nil
	default:
		return r.SpotRate, 1, nil
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		SpotRate:       request.NewRateLimit(spotInterval, spotRequestRate),
		SpotOrdersRate: request.NewRateLimit(spotOrderInterval, spotOrderRequestRate),
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
