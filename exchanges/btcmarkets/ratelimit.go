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

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		request.Auth:   request.NewRateLimitWithToken(btcmarketsRateInterval, btcmarketsAuthLimit, 1),
		request.UnAuth: request.NewRateLimitWithToken(btcmarketsRateInterval, btcmarketsUnauthLimit, 1),
		orderFunc:      request.NewRateLimitWithToken(btcmarketsRateInterval, btcmarketsOrderLimit, 1),
		batchFunc:      request.NewRateLimitWithToken(btcmarketsRateInterval, btcmarketsBatchOrderLimit, 1),
		withdrawFunc:   request.NewRateLimitWithToken(btcmarketsRateInterval, btcmarketsWithdrawLimit, 1),
		newReportFunc:  request.NewRateLimitWithToken(btcmarketsRateInterval, btcmarketsCreateNewReportLimit, 1),
	}
}
