package binanceus

import (
	"context"
	"fmt"
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
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case spotDefaultRate:
		limiter, tokens = r.SpotRate, 1
	case spotOrderbookTickerAllRate,
		spotSymbolPriceAllRate:
		limiter, tokens = r.SpotRate, 2
	case spotHistoricalTradesRate,
		spotOrderbookDepth500Rate:
		limiter, tokens = r.SpotRate, 5
	case spotOrderbookDepth1000Rate,
		spotAccountInformationRate,
		spotExchangeInfo,
		spotTradesQueryRate:
		limiter, tokens = r.SpotRate, 10
	case spotPriceChangeAllRate:
		limiter, tokens = r.SpotRate, 40
	case spotOrderbookDepth5000Rate:
		limiter, tokens = r.SpotRate, 50
	case spotOrderRate:
		limiter, tokens = r.SpotOrdersRate, 1
	case spotOrderQueryRate,
		spotSingleOCOOrderRate:
		limiter, tokens = r.SpotOrdersRate, 2
	case spotOpenOrdersSpecificRate:
		limiter, tokens = r.SpotOrdersRate, 3
	case spotAllOrdersRate,
		spotAllOCOOrdersRate:
		limiter, tokens = r.SpotOrdersRate, 10
	case spotOrderRateLimitRate:
		limiter, tokens = r.SpotOrdersRate, 20
	case spotOpenOrdersAllRate:
		limiter, tokens = r.SpotOrdersRate, 40
	default:
		limiter, tokens = r.SpotRate, 1
	}
	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reserves[i] = limiter.Reserve()
		finalDelay = reserves[i].Delay()
	}
	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		// Cancel all potential reservations to free up rate limiter if deadline
		// is exceeded.
		for x := range reserves {
			reserves[x].Cancel()
		}
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}
	time.Sleep(finalDelay)
	return nil
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
