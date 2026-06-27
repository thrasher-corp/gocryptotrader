package apexpro

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Rate limit definitions. See: https://api-docs.pro.apex.exchange/#general-rate-limits
const (
	publicEPL request.EndpointLimit = iota
	privateGetEPL
	privatePostEPL
	createOrderEPL
)

var rateLimits = request.RateLimitDefinitions{
	publicEPL:      request.NewRateLimitWithWeight(time.Minute, 600, 1),
	privateGetEPL:  request.NewRateLimitWithWeight(time.Minute, 600, 1),
	privatePostEPL: request.NewRateLimitWithWeight(time.Minute, 300, 1),
	createOrderEPL: request.NewRateLimitWithWeight(time.Minute, 300, 1),
}
