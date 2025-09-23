package bitget

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// Bitget rate limit constants
const (
	// There's a global rate limit of 6000/minute across all endpoints, but with this setup we'll be stopped by the individual limits first
	bitgetRateInterval         = time.Second
	bitgetRate20               = 20
	bitgetRate10               = 10
	bitgetRate5                = 5
	bitgetRate3                = 3
	bitgetRate2                = 2
	bitgetRate1                = 1
	bitgetSubscriptionInterval = time.Second
	bitgetSubscriptionRate     = 25
)

// Bitget rate limits
const (
	rate20 request.EndpointLimit = iota
	rate10
	rate5
	rate3
	rate2
	rate1
	rateSubscription
)

// GetRateLimits returns the rate limits for Bitget
func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		rate20:           request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate20, 1),
		rate10:           request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate10, 1),
		rate5:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate5, 1),
		rate3:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate3, 1),
		rate2:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate2, 1),
		rate1:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate1, 1),
		rateSubscription: request.NewRateLimitWithWeight(bitgetSubscriptionInterval, bitgetSubscriptionRate, 1),
	}
}
