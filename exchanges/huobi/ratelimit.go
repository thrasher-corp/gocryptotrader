package huobi

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Huobi rate limits per API Key
	huobiSpotRateInterval = time.Second * 1
	huobiSpotRequestRate  = 7

	huobiFuturesRateInterval    = time.Second * 3
	huobiFuturesAuthRequestRate = 30
	// Non market-request public interface rate
	huobiFuturesUnAuthRequestRate    = 60
	huobiFuturesTransferRateInterval = time.Second * 3
	huobiFuturesTransferReqRate      = 10

	huobiSwapRateInterval      = time.Second * 3
	huobiSwapAuthRequestRate   = 30
	huobiSwapUnauthRequestRate = 60

	huobiFuturesAuth request.EndpointLimit = iota
	huobiFuturesUnAuth
	huobiFuturesTransfer
	huobiSwapAuth
	huobiSwapUnAuth
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Unset:        request.NewRateLimitWithWeight(huobiSpotRateInterval, huobiSpotRequestRate, 1),
		huobiFuturesAuth:     request.NewRateLimitWithWeight(huobiFuturesRateInterval, huobiFuturesAuthRequestRate, 1),
		huobiFuturesUnAuth:   request.NewRateLimitWithWeight(huobiFuturesRateInterval, huobiFuturesUnAuthRequestRate, 1),
		huobiSwapAuth:        request.NewRateLimitWithWeight(huobiSwapRateInterval, huobiSwapAuthRequestRate, 1),
		huobiSwapUnAuth:      request.NewRateLimitWithWeight(huobiSwapRateInterval, huobiSwapUnauthRequestRate, 1),
		huobiFuturesTransfer: request.NewRateLimitWithWeight(huobiFuturesTransferRateInterval, huobiFuturesTransferReqRate, 1),
	}
}
