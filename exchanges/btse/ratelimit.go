package btse

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	btseRateInterval = time.Second
	btseQueryLimit   = 15
	btseOrdersLimit  = 75

	queryFunc request.EndpointLimit = iota
	orderFunc
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Query  *rate.Limiter
	Orders *rate.Limiter
}

// Limit executes rate limiting functionality for exchange
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) error {
	switch f {
	case orderFunc:
		return r.Orders.Wait(ctx)
	default:
		return r.Query.Wait(ctx)
	}
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Orders: request.NewRateLimit(btseRateInterval, btseOrdersLimit),
		Query:  request.NewRateLimit(btseRateInterval, btseQueryLimit),
	}
}
