package bithumb

import (
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
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	if f == request.Auth {
		time.Sleep(r.Auth.Reserve().Delay())
		return nil
	}
	time.Sleep(r.UnAuth.Reserve().Delay())
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:   request.NewRateLimit(bithumbRateInterval, bithumbAuthRate),
		UnAuth: request.NewRateLimit(bithumbRateInterval, bithumbUnauthRate),
	}
}
