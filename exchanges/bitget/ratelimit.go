package bitget

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Coinbasepro rate limit constants
const (
	bitgetRateInterval = time.Second
	bitgetRate20       = 20
	bitgetRate10       = 10
	bitgetRate5        = 5
	bitgetRate2        = 2
	bitgetRate1        = 1
)

// Bitget rate limits
const (
	Rate20 request.EndpointLimit = iota
	Rate10
	Rate5
	Rate2
	Rate1
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	RateLim20 *rate.Limiter
	RateLim10 *rate.Limiter
	RateLim5  *rate.Limiter
	RateLim2  *rate.Limiter
	RateLim1  *rate.Limiter
}

// Limit limits outbound calls
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case Rate20:
		return r.RateLim20.Wait(ctx)
	case Rate10:
		return r.RateLim10.Wait(ctx)
	case Rate5:
		return r.RateLim5.Wait(ctx)
	case Rate2:
		return r.RateLim2.Wait(ctx)
	case Rate1:
		return r.RateLim1.Wait(ctx)
	default:
		return errors.Errorf(errUnknownEndpointLimit, f)
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		RateLim20: request.NewRateLimit(bitgetRateInterval, bitgetRate20),
		RateLim10: request.NewRateLimit(bitgetRateInterval, bitgetRate10),
		RateLim5:  request.NewRateLimit(bitgetRateInterval, bitgetRate5),
		RateLim2:  request.NewRateLimit(bitgetRateInterval, bitgetRate2),
		RateLim1:  request.NewRateLimit(bitgetRateInterval, bitgetRate1),
	}
}
