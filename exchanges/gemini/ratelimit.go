package gemini

import (
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
		Auth:   request.NewRateLimit(geminiRateInterval, geminiAuthRate),
		UnAuth: request.NewRateLimit(geminiRateInterval, geminiUnauthRate),
	}
}
