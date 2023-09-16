package poloniex

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	rateInterval                 = time.Second
	unauthRate                   = 200
	authNonResourceIntensiveRate = 50
	authResourceIntensiveRate    = 10
	referenceDataRate            = 10
)

const (
	authNonResourceIntensiveEPL request.EndpointLimit = iota
	authResourceIntensiveEPL
	unauthEPL
	referenceDataEPL
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	AuthNonResourceIntensive *rate.Limiter
	AuthResourceIntensive    *rate.Limiter
	Unauth                   *rate.Limiter
	ReferenceData            *rate.Limiter
}

// Limit limits outbound calls
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case authNonResourceIntensiveEPL:
		return r.AuthNonResourceIntensive.Wait(ctx)
	case authResourceIntensiveEPL:
		return r.AuthResourceIntensive.Wait(ctx)
	case referenceDataEPL:
		return r.ReferenceData.Wait(ctx)
	default:
		return r.Unauth.Wait(ctx)
	}
}

// SetRateLimit returns the rate limit for the exchange
// If your account's volume is over $5 million in 30 day volume,
// you may be eligible for an API rate limit increase.
// Please email poloniex@circle.com.
// As per https://docs.poloniex.com/#http-api
func SetRateLimit() *RateLimit {
	return &RateLimit{
		AuthNonResourceIntensive: request.NewRateLimit(rateInterval, authNonResourceIntensiveRate),
		AuthResourceIntensive:    request.NewRateLimit(rateInterval, authResourceIntensiveRate),
		Unauth:                   request.NewRateLimit(rateInterval, unauthRate),
		ReferenceData:            request.NewRateLimit(rateInterval, referenceDataRate),
	}
}
