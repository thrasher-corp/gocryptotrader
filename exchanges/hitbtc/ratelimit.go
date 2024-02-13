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

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		marketRequests:  request.NewRateLimitWithToken(hitbtcRateInterval, hitbtcMarketDataReqRate, 1),
		tradingRequests: request.NewRateLimitWithToken(hitbtcRateInterval, hitbtcTradingReqRate, 1),
		otherRequests:   request.NewRateLimitWithToken(hitbtcRateInterval, hitbtcAllOthers, 1),
	}
}
