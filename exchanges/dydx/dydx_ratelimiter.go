package dydx

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"golang.org/x/time/rate"
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

	// request rates per interval
	defaultV3Rate             = 175
	sendVerificationEmailRate = 2
	cancelOrdersRate          = 3
	cancelSingleOrderRate     = 250
	postOrdersRate            = 10
	postTestnetTokensRate     = 5
	cancelActiveOrdersRate    = 425
	getActiveOrderRate        = 175
	defaultRateRate           = 10
)

// RateLimiter limits dYdX requests
type RateLimiter struct {
	DefaultV3Limiter             *rate.Limiter
	SendVerificationEmailLimiter *rate.Limiter
	CancelOrdersLimiter          *rate.Limiter
	CancelSingleOrderLimiter     *rate.Limiter
	PostOrdersLimiter            *rate.Limiter
	PostTestnetTokensLimiter     *rate.Limiter
	CancelActiveOrdersLimiter    *rate.Limiter
	GetActiveOrderLimiter        *rate.Limiter
	DefaultRateLimiter           *rate.Limiter
}

// SetupRateLimiter returns the rate limit for the exchange
func SetupRateLimiter() *RateLimiter {
	return &RateLimiter{
		DefaultV3Limiter:             request.NewRateLimit(tenSecondInterval, defaultV3Rate),
		SendVerificationEmailLimiter: request.NewRateLimit(tenMinuteInterval, sendVerificationEmailRate),
		CancelOrdersLimiter:          request.NewRateLimit(tenSecondInterval, cancelOrdersRate),
		CancelSingleOrderLimiter:     request.NewRateLimit(tenSecondInterval, cancelSingleOrderRate),
		PostOrdersLimiter:            request.NewRateLimit(time.Second, postOrdersRate),
		PostTestnetTokensLimiter:     request.NewRateLimit(oneDayInterval, postTestnetTokensRate),
		CancelActiveOrdersLimiter:    request.NewRateLimit(tenSecondInterval, cancelActiveOrdersRate),
		GetActiveOrderLimiter:        request.NewRateLimit(tenSecondInterval, getActiveOrderRate),
		DefaultRateLimiter:           request.NewRateLimit(time.Minute, defaultRateRate),
	}
}

// Limit executes rate limiting functionality for dYdX exchange
func (r *RateLimiter) Limit(ctx context.Context, f request.EndpointLimit) error {
	var limiter *rate.Limiter
	var tokens int
	switch f {
	case defaultV3EPL:
		limiter, tokens = r.DefaultV3Limiter, 4 // there are 46 endpoints using this limiter
	case sendVerificationEmailEPL:
		return r.SendVerificationEmailLimiter.Wait(ctx)
	case cancelOrdersEPL:
		return r.CancelOrdersLimiter.Wait(ctx)
	case cancelSingleOrderEPL:
		return r.CancelSingleOrderLimiter.Wait(ctx)
	case postOrdersEPL:
		return r.PostOrdersLimiter.Wait(ctx)
	case postTestnetTokensEPL:
		return r.PostTestnetTokensLimiter.Wait(ctx)
	case cancelActiveOrdersEPL:
		return r.CancelActiveOrdersLimiter.Wait(ctx)
	case getActiveOrdersEPL:
		return r.GetActiveOrderLimiter.Wait(ctx)
	default: // in case non v3 endpoints are added.
		return r.DefaultRateLimiter.Wait(ctx)
	}
	var finalDelay time.Duration
	var reserves = make([]*rate.Reservation, tokens)
	for i := 0; i < tokens; i++ {
		reserves[i] = limiter.Reserve()
		finalDelay = reserves[i].Delay()
	}

	if dl, ok := ctx.Deadline(); ok && dl.Before(time.Now().Add(finalDelay)) {
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
