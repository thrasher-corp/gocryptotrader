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

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Unset:        request.NewRateLimit(huobiSpotRateInterval, huobiSpotRequestRate, 1),
		huobiFuturesAuth:     request.NewRateLimit(huobiFuturesRateInterval, huobiFuturesAuthRequestRate, 1),
		huobiFuturesUnAuth:   request.NewRateLimit(huobiFuturesRateInterval, huobiFuturesUnAuthRequestRate, 1),
		huobiSwapAuth:        request.NewRateLimit(huobiSwapRateInterval, huobiSwapAuthRequestRate, 1),
		huobiSwapUnAuth:      request.NewRateLimit(huobiSwapRateInterval, huobiSwapUnauthRequestRate, 1),
		huobiFuturesTransfer: request.NewRateLimit(huobiFuturesTransferRateInterval, huobiFuturesTransferReqRate, 1),
	}
}
