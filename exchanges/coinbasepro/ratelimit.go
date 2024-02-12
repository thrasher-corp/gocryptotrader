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
		request.Auth:   request.NewRateLimit(coinbaseproRateInterval, coinbaseproAuthRate, 1),
		request.UnAuth: request.NewRateLimit(coinbaseproRateInterval, coinbaseproUnauthRate, 1),
	}
}
