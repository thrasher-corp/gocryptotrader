package bithumb

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Exchange specific rate limit consts
const (
	bithumbRateInterval = time.Second
	bithumbAuthRate     = 95
	bithumbUnauthRate   = 95
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:  request.NewRateLimitWithWeight(bithumbRateInterval, bithumbAuthRate, 1),
		request.Unset: request.NewRateLimitWithWeight(bithumbRateInterval, bithumbUnauthRate, 1),
	}
}
