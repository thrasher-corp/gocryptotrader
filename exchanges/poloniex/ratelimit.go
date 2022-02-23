package poloniex

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	poloniexRateInterval = time.Second
	poloniexAuthRate     = 6
	poloniexUnauthRate   = 6
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
// If your account's volume is over $5 million in 30 day volume,
// you may be eligible for an API rate limit increase.
// Please email poloniex@circle.com.
// As per https://docs.poloniex.com/#http-api
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:   request.NewRateLimit(poloniexRateInterval, poloniexAuthRate),
		UnAuth: request.NewRateLimit(poloniexRateInterval, poloniexUnauthRate),
	}
}
