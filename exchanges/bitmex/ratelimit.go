package bitmex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bitmex rate limits
const (
	bitmexRateInterval = time.Minute
	bitmexUnauthRate   = 30
	bitmexAuthRate     = 60
)

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithToken(bitmexRateInterval, bitmexAuthRate, 1),
		request.UnAuth: request.NewRateLimitWithToken(bitmexRateInterval, bitmexUnauthRate, 1),
	}
}
