package lbank

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const rateInterval = 10 * time.Second

const (
	defaultEPL request.EndpointLimit = iota
	orderEPL
	contractEPL
)

// GetRateLimits returns LBank's rate limits. The spot API allows 500 order placements and cancellations and 200 other
// requests every 10 seconds. The contract API, on its own host, documents no limit, so it is given the spot API's
// limit for other requests
func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		defaultEPL:  request.NewRateLimitWithWeight(rateInterval, 200, 1),
		orderEPL:    request.NewRateLimitWithWeight(rateInterval, 500, 1),
		contractEPL: request.NewRateLimitWithWeight(rateInterval, 200, 1),
	}
}
