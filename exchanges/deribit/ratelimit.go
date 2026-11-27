package deribit

import (
	"context"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchange/request"
)

const (
	// Request rates per interval
	minMatchingBurst   = 100
	nonMatchingRate    = 20
	portfoliMarginRate = 1
	// Deribit documents 5 a second for its shared ticker and order book limit, but accepts unauthenticated requests at
	// about one a second
	tickerOrderBookRate = 1
	// Weightings
	matchingWeight = 5
	standardWeight = 1
	// Rate limit keys
	nonMatchingEPL request.EndpointLimit = iota
	matchingEPL
	portfolioMarginEPL
	privatePortfolioMarginEPL
	blockTradeReadEPL
	blockTradeWriteEPL
	blockRFQReadEPL
	blockRFQWriteEPL
	tickerOrderBookEPL
)

// GetRateLimits returns the rate limit for the exchange
func GetRateLimits() request.RateLimitDefinitions {
	return request.RateLimitDefinitions{
		nonMatchingEPL:            request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, nonMatchingRate), standardWeight),
		matchingEPL:               request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, minMatchingBurst), matchingWeight),
		portfolioMarginEPL:        request.GetRateLimiterWithWeight(request.NewRateLimit(5*time.Second, portfoliMarginRate), standardWeight),
		privatePortfolioMarginEPL: request.GetRateLimiterWithWeight(request.NewRateLimit(5*time.Second, portfoliMarginRate), standardWeight),
		blockTradeReadEPL:         request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, nonMatchingRate), standardWeight),
		blockTradeWriteEPL:        request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, nonMatchingRate), standardWeight),
		blockRFQReadEPL:           request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, nonMatchingRate), standardWeight),
		blockRFQWriteEPL:          request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, nonMatchingRate), standardWeight),
		tickerOrderBookEPL:        request.GetRateLimiterWithWeight(request.NewRateLimit(time.Second, tickerOrderBookRate), standardWeight),
	}
}

// attemptRateLimits returns the endpoint limits each attempt of a request waits on, in turn. Deribit charges a ticker or
// order book request to the non-matching engine limit as well as their shared limit, which comes last so it paces the
// attempts Deribit receives
func attemptRateLimits(epl request.EndpointLimit) []request.EndpointLimit {
	if epl == tickerOrderBookEPL {
		return []request.EndpointLimit{nonMatchingEPL, tickerOrderBookEPL}
	}
	return []request.EndpointLimit{epl}
}

// initiateRateLimits waits on each endpoint limit in turn
func (e *Exchange) initiateRateLimits(ctx context.Context, limits []request.EndpointLimit) error {
	for _, epl := range limits {
		if err := e.Requester.InitiateRateLimit(ctx, epl); err != nil {
			return err
		}
	}
	return nil
}
