package btcmarkets

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// BTCMarkets Rate limit consts
const (
	btcmarketsRateInterval         = time.Second * 10
	btcmarketsAuthLimit            = 50
	btcmarketsUnauthLimit          = 50
	btcmarketsOrderLimit           = 30
	btcmarketsBatchOrderLimit      = 5
	btcmarketsWithdrawLimit        = 10
	btcmarketsCreateNewReportLimit = 1

	// Used to match endpoints to rate limits
	orderFunc request.EndpointLimit = iota
	batchFunc
	withdrawFunc
	newReportFunc
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithWeight(btcmarketsRateInterval, btcmarketsAuthLimit, 1),
		request.UnAuth: request.NewRateLimitWithWeight(btcmarketsRateInterval, btcmarketsUnauthLimit, 1),
		orderFunc:      request.NewRateLimitWithWeight(btcmarketsRateInterval, btcmarketsOrderLimit, 1),
		batchFunc:      request.NewRateLimitWithWeight(btcmarketsRateInterval, btcmarketsBatchOrderLimit, 1),
		withdrawFunc:   request.NewRateLimitWithWeight(btcmarketsRateInterval, btcmarketsWithdrawLimit, 1),
		newReportFunc:  request.NewRateLimitWithWeight(btcmarketsRateInterval, btcmarketsCreateNewReportLimit, 1),
	}
}
