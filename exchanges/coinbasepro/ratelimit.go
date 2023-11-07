package coinbasepro

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Coinbasepro rate limit constants
const (
	coinbaseV3Interval = time.Second
	coinbaseV3Rate     = 30

	coinbaseV2Interval = time.Hour
	coinbaseV2Rate     = 10000
)

const (
	V2Rate request.EndpointLimit = iota
	V3Rate
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	RateLimV3 *rate.Limiter
	RateLimV2 *rate.Limiter
}

// Limit limits outbound calls
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	if f == V3Rate {
		return r.RateLimV3.Wait(ctx)
	}
	return r.RateLimV2.Wait(ctx)
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		RateLimV3: request.NewRateLimit(coinbaseV3Interval, coinbaseV3Rate),
		RateLimV2: request.NewRateLimit(coinbaseV2Interval, coinbaseV2Rate),
	}
}
