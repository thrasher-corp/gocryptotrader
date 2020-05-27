package binance

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// Binance limit rates
	// Global dictates the max rate limit for general request items which is
	// 1200 requests per minute
	binanceGlobalInterval    = time.Minute
	binanceGlobalRequestRate = 1200
	// Order related limits which are segregated from the global rate limits
	// 100 requests per 10 seconds and max 100000 requests per day.
	binanceOrderInterval         = 10 * time.Second
	binanceOrderRequestRate      = 100
	binanceOrderDailyInterval    = time.Hour * 24
	binanceOrderDailyMaxRequests = 100000
)

const (
	limitDefault request.EndpointLimit = iota
	limitHistoricalTrades
	limitOrderbookDepth500
	limitOrderbookDepth1000
	limitOrderbookDepth5000
	limitOrderbookTickerAll
	limitPriceChangeAll
	limitSymbolPriceAll
	limitOpenOrdersAll
	limitOrder
	limitOrdersAll
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	GlobalRate *rate.Limiter
	Orders     *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case limitHistoricalTrades:
		limiter, tokens = r.GlobalRate, 5
	case limitOrderbookDepth500:
		limiter, tokens = r.GlobalRate, 5
	case limitOrderbookDepth1000:
		limiter, tokens = r.GlobalRate, 10
	case limitOrderbookDepth5000:
		limiter, tokens = r.GlobalRate, 50
	case limitOrderbookTickerAll:
		limiter, tokens = r.GlobalRate, 2
	case limitPriceChangeAll:
		limiter, tokens = r.GlobalRate, 40
	case limitSymbolPriceAll:
		limiter, tokens = r.GlobalRate, 2
	case limitOpenOrdersAll:
		limiter, tokens = r.Orders, 40
	case limitOrder:
		limiter, tokens = r.Orders, 1
	case limitOrdersAll:
		limiter, tokens = r.Orders, 5
	default:
		limiter, tokens = r.GlobalRate, 1
	}

	var finalDelay time.Duration
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		finalDelay = limiter.Reserve().Delay()
	}
	time.Sleep(finalDelay)
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		GlobalRate: request.NewRateLimit(binanceGlobalInterval, binanceGlobalRequestRate),
		Orders:     request.NewRateLimit(binanceOrderInterval, binanceOrderRequestRate),
	}
}

func bestPriceLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return limitOrderbookTickerAll
	}

	return limitDefault
}

func openOrdersLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return limitOpenOrdersAll
	}

	return limitOrder
}

func orderbookLimit(depth int) request.EndpointLimit {
	switch {
	case depth <= 100:
		return limitDefault
	case depth <= 500:
		return limitOrderbookDepth500
	case depth <= 1000:
		return limitOrderbookDepth1000
	}

	return limitOrderbookDepth5000
}

func symbolPriceLimit(symbol string) request.EndpointLimit {
	if symbol == "" {
		return limitSymbolPriceAll
	}

	return limitDefault
}
