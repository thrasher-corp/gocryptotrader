package coinbasepro

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Coinbasepro rate limit conts
const (
	coinbaseproRateInterval = time.Second
	coinbaseproAuthRate     = 5
	coinbaseproUnauthRate   = 2
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Auth   *rate.Limiter
	UnAuth *rate.Limiter
}

// Limit limits outbound calls
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	if f == request.Auth {
		return r.Auth.Wait(ctx)
	}
	return r.UnAuth.Wait(ctx)
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:   request.NewRateLimit(coinbaseproRateInterval, coinbaseproAuthRate),
		UnAuth: request.NewRateLimit(coinbaseproRateInterval, coinbaseproUnauthRate),
	}
}
