package coinbasepro

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Coinbasepro rate limit conts
const (
	coinbaseproRateInterval = time.Second
	coinbaseproAuthRate     = 5
	coinbaseproUnauthRate   = 2
)

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithToken(coinbaseproRateInterval, coinbaseproAuthRate, 1),
		request.UnAuth: request.NewRateLimitWithToken(coinbaseproRateInterval, coinbaseproUnauthRate, 1),
	}
}
