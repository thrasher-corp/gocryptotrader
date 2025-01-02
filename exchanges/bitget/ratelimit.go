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
	Rate20 request.EndpointLimit = iota
	Rate10
	Rate5
	Rate3
	Rate2
	Rate1
	RateSubscription
)

func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		Rate20:           request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate20, 1),
		Rate10:           request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate10, 1),
		Rate5:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate5, 1),
		Rate3:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate3, 1),
		Rate2:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate2, 1),
		Rate1:            request.NewRateLimitWithWeight(bitgetRateInterval, bitgetRate1, 1),
		RateSubscription: request.NewRateLimitWithWeight(bitgetSubscriptionInterval, bitgetSubscriptionRate, 1),
	}
}
