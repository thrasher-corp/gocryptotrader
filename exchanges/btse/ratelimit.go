package btse

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	btseRateInterval = time.Second
	btseQueryLimit   = 15
	btseOrdersLimit  = 75

	queryFunc request.EndpointLimit = iota
	orderFunc
)

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		orderFunc: request.NewRateLimitWithToken(btseRateInterval, btseOrdersLimit, 1),
		queryFunc: request.NewRateLimitWithToken(btseRateInterval, btseQueryLimit, 1),
	}
}
