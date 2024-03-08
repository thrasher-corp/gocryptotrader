package bitget

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

// Coinbasepro rate limit conts
const (
	bitgetRateInterval = time.Second
	bitgetRate         = 10
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Default *rate.Limiter
}

// Limit limits outbound calls
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	return r.Default.Wait(ctx)
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Default: request.NewRateLimit(bitgetRateInterval, bitgetRate),
	}
}
