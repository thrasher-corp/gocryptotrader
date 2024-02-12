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
		marketRequests:  request.NewRateLimit(hitbtcRateInterval, hitbtcMarketDataReqRate, 1),
		tradingRequests: request.NewRateLimit(hitbtcRateInterval, hitbtcTradingReqRate, 1),
		otherRequests:   request.NewRateLimit(hitbtcRateInterval, hitbtcAllOthers, 1),
	}
}
