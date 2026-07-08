package htx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// HTX rate limits per API Key
	htxSpotRateInterval = time.Second * 1
	htxSpotRequestRate  = 7

	htxFuturesRateInterval    = time.Second * 3
	htxFuturesAuthRequestRate = 30
	// Non market-request public interface rate
	htxFuturesUnAuthRequestRate    = 60
	htxFuturesTransferRateInterval = time.Second * 3
	htxFuturesTransferReqRate      = 10

	htxSwapRateInterval      = time.Second * 3
	htxSwapAuthRequestRate   = 30
	htxSwapUnauthRequestRate = 60

	htxFuturesAuth request.EndpointLimit = iota
	htxFuturesUnAuth
	htxFuturesTransfer
	htxSwapAuth
	htxSwapUnAuth
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Unset:      request.NewRateLimitWithWeight(htxSpotRateInterval, htxSpotRequestRate, 1),
		htxFuturesAuth:     request.NewRateLimitWithWeight(htxFuturesRateInterval, htxFuturesAuthRequestRate, 1),
		htxFuturesUnAuth:   request.NewRateLimitWithWeight(htxFuturesRateInterval, htxFuturesUnAuthRequestRate, 1),
		htxSwapAuth:        request.NewRateLimitWithWeight(htxSwapRateInterval, htxSwapAuthRequestRate, 1),
		htxSwapUnAuth:      request.NewRateLimitWithWeight(htxSwapRateInterval, htxSwapUnauthRequestRate, 1),
		htxFuturesTransfer: request.NewRateLimitWithWeight(htxFuturesTransferRateInterval, htxFuturesTransferReqRate, 1),
	}
}
