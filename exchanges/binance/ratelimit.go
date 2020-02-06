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
	// 10 requests per second and max 100000 requests per day.
	binanceOrderInterval         = time.Second
	binanceOrderRequestRate      = 10
	binanceOrderDailyInterval    = time.Hour * 24
	binanceOrderDailyMaxRequests = 100000
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	GlobalRate *rate.Limiter
	Orders     *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	if f == request.Auth {
		time.Sleep(r.Orders.Reserve().Delay())
		return nil
	}
	time.Sleep(r.GlobalRate.Reserve().Delay())
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		GlobalRate: request.NewRateLimit(binanceGlobalInterval, binanceOrderDailyMaxRequests),
		Orders:     request.NewRateLimit(binanceOrderInterval, binanceOrderRequestRate),
	}
}
