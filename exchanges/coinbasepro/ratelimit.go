package coinbasepro

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Coinbasepro rate limit constants
const (
	coinbaseV3Interval = time.Second
	coinbaseV3Rate     = 27

	coinbaseV2Interval = time.Hour
	coinbaseV2Rate     = 10000

	coinbaseWSInterval = time.Second
	coinbaseWSRate     = 750

	coinbasePublicInterval = time.Second
	coinbasePublicRate     = 10
)

// Coinbase pro rate limits
const (
	V2Rate request.EndpointLimit = iota
	V3Rate
	WSRate
	PubRate
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	RateLimV3  *rate.Limiter
	RateLimV2  *rate.Limiter
	RateLimWS  *rate.Limiter
	RateLimPub *rate.Limiter
}

// Limit limits outbound calls
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case V3Rate:
		return r.RateLimV3.Wait(ctx)
	case V2Rate:
		return r.RateLimV2.Wait(ctx)
	case WSRate:
		return r.RateLimWS.Wait(ctx)
	case PubRate:
		return r.RateLimPub.Wait(ctx)
	default:
		return fmt.Errorf("%w %v", errUnknownEndpointLimit, f)
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		RateLimWS:  request.NewRateLimit(coinbaseWSInterval, coinbaseWSRate),
		RateLimV3:  request.NewRateLimit(coinbaseV3Interval, coinbaseV3Rate),
		RateLimV2:  request.NewRateLimit(coinbaseV2Interval, coinbaseV2Rate),
		RateLimPub: request.NewRateLimit(coinbasePublicInterval, coinbasePublicRate),
	}
}
