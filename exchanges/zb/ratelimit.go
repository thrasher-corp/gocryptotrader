package zb

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	zbRateInterval = time.Second
	zbAuthLimit    = 60
	zbUnauthLimit  = 60

	zbKlineDataInterval = time.Second * 2
	zbKlineDataLimit    = 1

	// Used to match endpints to rate limits
	klineFunc request.EndpointLimit = iota
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Auth      *rate.Limiter
	UnAuth    *rate.Limiter
	KlineData *rate.Limiter
}

// Limit limits the outbound requests
func (r *RateLimit) Limit(f request.EndpointLimit) error {
	switch f {
	case request.Auth:
		time.Sleep(r.Auth.Reserve().Delay())
	case klineFunc:
		time.Sleep(r.KlineData.Reserve().Delay())
	default:
		time.Sleep(r.UnAuth.Reserve().Delay())
	}
	return nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:      request.NewRateLimit(zbRateInterval, zbAuthLimit),
		UnAuth:    request.NewRateLimit(zbRateInterval, zbUnauthLimit),
		KlineData: request.NewRateLimit(zbKlineDataInterval, zbKlineDataLimit),
	}
}
