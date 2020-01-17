package bitmex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Bitmex rate limits
const (
	bitmexRateInterval = time.Minute
	bitmexUnauthRate   = 30
	bitmexAuthRate     = 60
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Auth   *rate.Limiter
	UnAuth *rate.Limiter
}

// Limit limits outbound calls
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
		Auth:   request.NewRateLimit(bitmexRateInterval, bitmexAuthRate),
		UnAuth: request.NewRateLimit(bitmexRateInterval, bitmexUnauthRate),
	}
}
