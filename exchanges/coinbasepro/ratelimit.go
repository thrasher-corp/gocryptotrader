package coinbase

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Coinbase pro rate limits
const (
	V2Rate request.EndpointLimit = iota
	V3Rate
	WSAuthRate
	WSUnauthRate
	PubRate
)

var rateLimits = request.RateLimitDefinitions{
	V2Rate:       request.NewRateLimitWithWeight(time.Hour, 10000, 1),
	V3Rate:       request.NewRateLimitWithWeight(time.Second, 27, 1),
	WSAuthRate:   request.NewRateLimitWithWeight(time.Second, 750, 1),
	WSUnauthRate: request.NewRateLimitWithWeight(time.Second, 8, 1),
	PubRate:      request.NewRateLimitWithWeight(time.Second, 10, 1),
}
