package dydx

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	//  endpoint limits
	defaultV3EPL request.EndpointLimit = iota
	sendVerificationEmailEPL
	cancelOrdersEPL
	cancelSingleOrderEPL
	postOrdersEPL
	postTestnetTokensEPL
	cancelActiveOrdersEPL
	getActiveOrdersEPL
	defaultRateEPL

	// interval durations
	seventeenSecondInterval = time.Second * 17
	tenMinuteInterval       = time.Minute * 10
	tenSecondInterval       = time.Second * 10
	oneDayInterval          = time.Hour * 24
)

// GetRateLimit returns the rate limit for the exchange
func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		defaultV3EPL:             request.NewRateLimitWithWeight(tenSecondInterval, 175, 1),
		sendVerificationEmailEPL: request.NewRateLimitWithWeight(tenMinuteInterval, 2, 1),
		cancelOrdersEPL:          request.NewRateLimitWithWeight(tenSecondInterval, 3, 1),
		cancelSingleOrderEPL:     request.NewRateLimitWithWeight(tenSecondInterval, 250, 1),
		postOrdersEPL:            request.NewRateLimitWithWeight(time.Second, 10, 1),
		postTestnetTokensEPL:     request.NewRateLimitWithWeight(oneDayInterval, 5, 1),
		cancelActiveOrdersEPL:    request.NewRateLimitWithWeight(tenSecondInterval, 425, 1),
		getActiveOrdersEPL:       request.NewRateLimitWithWeight(tenSecondInterval, 175, 1),
		// DefaultRateLimiter:           request.NewRateLimitWithWeight(time.Minute, 10,1),
	}
}
