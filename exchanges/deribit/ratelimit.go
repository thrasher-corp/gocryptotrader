package deribit

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
)

const (
	// request rates per interval
	minMatchingBurst   = 100
	nonMatchingRate    = 20
	minMatchingRate    = 5
	portfoliMarginRate = 1

	nonMatchingEPL request.EndpointLimit = iota
	matchingEPL
	portfolioMarginEPL
	privatePortfolioMarginEPL
)

// RateLimiter holds the rate limiter to endpoints
type RateLimiter struct {
	NonMatchingEngine      *rate.Limiter
	MatchingEngine         *rate.Limiter
	PortfolioMargin        *rate.Limiter
	PrivatePortfolioMargin *rate.Limiter
}

// SetRateLimit returns the rate limit for the exchange
func SetRateLimit() *RateLimiter {
	return &RateLimiter{
		NonMatchingEngine:      request.NewRateLimit(time.Second, nonMatchingRate),
		MatchingEngine:         request.NewRateLimit(time.Second, minMatchingBurst),
		PortfolioMargin:        request.NewRateLimit(5*time.Second, portfoliMarginRate),
		PrivatePortfolioMargin: request.NewRateLimit(5*time.Second, portfoliMarginRate),
	}
}

// Limit executes rate limiting functionality for Binance
func (r *RateLimiter) Limit(ctx context.Context, f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case nonMatchingEPL:
		limiter, tokens = r.NonMatchingEngine, 1
	case portfolioMarginEPL:
		limiter, tokens = r.PortfolioMargin, portfoliMarginRate
	case privatePortfolioMarginEPL:
		limiter, tokens = r.PrivatePortfolioMargin, portfoliMarginRate
	default:
		limiter, tokens = r.MatchingEngine, minMatchingRate
	}
	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		// Consume tokens 1 at a time as this avoids needing burst capacity in the limiter,
		// which would otherwise allow the rate limit to be exceeded over short periods
		reserves[i] = limiter.Reserve()
		finalDelay = reserves[i].Delay()
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
		// Cancel all potential reservations to free up rate limiter if deadline
		// is exceeded.
		for x := range reserves {
			reserves[x].Cancel()
		}
		return fmt.Errorf("rate limit delay of %s will exceed deadline: %w",
			finalDelay,
			context.DeadlineExceeded)
	}

	time.Sleep(finalDelay)
	return nil
}
