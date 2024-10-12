package deribit

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// Request rates per interval
	minMatchingBurst   = 100
	nonMatchingRate    = 20
	portfoliMarginRate = 1
	// Weightings
	matchingWeight = 5
	standardWeight = 1
	// Rate limit keys
	nonMatchingEPL request.EndpointLimit = iota
	matchingEPL
	portfolioMarginEPL
	privatePortfolioMarginEPL
)

// GetRateLimits returns the rate limit for the exchange
func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		nonMatchingEPL:            request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, nonMatchingRate), standardWeight),
		matchingEPL:               request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, minMatchingBurst), matchingWeight),
		portfolioMarginEPL:        request.GetRateLimiterWithWeight(request.NewRateLimit(5*time.Second, portfoliMarginRate), standardWeight),
		privatePortfolioMarginEPL: request.GetRateLimiterWithWeight(request.NewRateLimit(5*time.Second, portfoliMarginRate), standardWeight),
	}
}
