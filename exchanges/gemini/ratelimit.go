package gemini

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// gemini limit rates
	geminiRateInterval = time.Minute
	geminiAuthRate     = 600
	geminiUnauthRate   = 120
)

// RateLimit implements the request.Limiter interface
type RateLimit struct {
	Auth   *rate.Limiter
	UnAuth *rate.Limiter
}

// Limit limits the endpoint functionality
func (r *RateLimit) Limit(ctx context.Context, f request.EndpointLimit) (*rate.Limiter, request.Tokens, error) {
	if f == request.Auth {
		return r.Auth, 1, nil
	}
	return r.UnAuth, 1, nil
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimit {
	return &RateLimit{
		Auth:   request.NewRateLimit(geminiRateInterval, geminiAuthRate),
		UnAuth: request.NewRateLimit(geminiRateInterval, geminiUnauthRate),
	}
}
