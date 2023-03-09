package bithumb

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Exchange specific rate limit consts
const (
	bithumbRateInterval = time.Second
	bithumbAuthRate     = 95
	bithumbUnauthRate   = 95
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Auth   *rate.Limiter
	UnAuth *rate.Limiter
}

// Limit limits requests
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, int, error) {
	if f == request.Auth {
		return r.Auth, 1, nil
	}
	return r.UnAuth, 1, nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:   request.NewRateLimit(bithumbRateInterval, bithumbAuthRate),
		UnAuth: request.NewRateLimit(bithumbRateInterval, bithumbUnauthRate),
	}
}
