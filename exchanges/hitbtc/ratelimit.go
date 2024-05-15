package hitbtc

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	hitbtcRateInterval      = time.Second
	hitbtcMarketDataReqRate = 100
	hitbtcTradingReqRate    = 300
	hitbtcAllOthers         = 10

	marketRequests request.EndpointLimit = iota
	tradingRequests
	otherRequests
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		marketRequests:  request.NewRateLimitWithWeight(hitbtcRateInterval, hitbtcMarketDataReqRate, 1),
		tradingRequests: request.NewRateLimitWithWeight(hitbtcRateInterval, hitbtcTradingReqRate, 1),
		otherRequests:   request.NewRateLimitWithWeight(hitbtcRateInterval, hitbtcAllOthers, 1),
	}
}
