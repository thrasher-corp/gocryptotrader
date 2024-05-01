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

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithWeight(bitmexRateInterval, bitmexAuthRate, 1),
		request.UnAuth: request.NewRateLimitWithWeight(bitmexRateInterval, bitmexUnauthRate, 1),
	}
}
