package localbitcoins

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const orderBookLimiter request.EndpointLimit = 1
const tickerLimiter request.EndpointLimit = 2

// RateLimit define s custom rate limiter scoped for orderbook requests
type RateLimit struct {
	Orderbook *rate.Limiter
	Ticker    *rate.Limiter
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	if f == orderBookLimiter {
		time.Sleep(r.Orderbook.Reserve().Delay())
	} else if f == tickerLimiter {
		time.Sleep(r.Ticker.Reserve().Delay())
	}
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		// 4 seconds per book fetching is the best time frame to actually
		// receive without retying. There is undocumentated rate limit.
		Orderbook: request.NewRateLimit(4*time.Second, 1),
		Ticker:    request.NewRateLimit(time.Second, 1),
	}
}
