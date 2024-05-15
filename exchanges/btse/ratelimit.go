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

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		orderFunc: request.NewRateLimitWithWeight(btseRateInterval, btseOrdersLimit, 1),
		queryFunc: request.NewRateLimitWithWeight(btseRateInterval, btseQueryLimit, 1),
	}
}
