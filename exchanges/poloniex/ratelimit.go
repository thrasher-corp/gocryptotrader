package poloniex

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	rateInterval                 = time.Second
	unauthRate                   = 200
	authNonResourceIntensiveRate = 50
	authResourceIntensiveRate    = 10
	referenceDataRate            = 10
)

const (
	authNonResourceIntensiveEPL request.EndpointLimit = iota
	authResourceIntensiveEPL
	unauthEPL
	referenceDataEPL
)

// GetRateLimit returns the rate limit for the exchange
// If your account's volume is over $5 million in 30 day volume,
// you may be eligible for an API rate limit increase.
// Please email poloniex@circle.com.
// As per https://docs.poloniex.com/#http-api

func GetRateLimit() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		authNonResourceIntensiveEPL: request.NewRateLimitWithWeight(rateInterval, authNonResourceIntensiveRate, 1),
		authResourceIntensiveEPL:    request.NewRateLimitWithWeight(rateInterval, authResourceIntensiveRate, 1),
		unauthEPL:                   request.NewRateLimitWithWeight(rateInterval, unauthRate, 1),
		referenceDataEPL:            request.NewRateLimitWithWeight(rateInterval, referenceDataRate, 1),
	}
}
