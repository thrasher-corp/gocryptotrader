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

// rateLimits holds LBank's rate limits, which every Exchange instance shares so that together they stay within them.
// The spot API allows 500 order placements and cancellations and 200 other requests every 10 seconds. The contract API,
// on its own host, documents no limit, so it is given the spot API's limit for other requests
var rateLimits = request.RateLimitDefinitions{
	defaultEPL:  request.NewRateLimitWithWeight(rateInterval, 200, 1),
	orderEPL:    request.NewRateLimitWithWeight(rateInterval, 500, 1),
	contractEPL: request.NewRateLimitWithWeight(rateInterval, 200, 1),
}
